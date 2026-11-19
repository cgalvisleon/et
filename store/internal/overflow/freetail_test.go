package overflow

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// errCut simula el corte a mitad de FreeTail.
var errCut = errors.New("corte a mitad de la liberación")

/**
* writeChain escribe un valor al azar de pages páginas para owner y devuelve su Ref y el valor.
* @param t testing.TB, e *env, rng *rand.Rand, owner page.XID, pages int
* @return Ref, []byte
**/
func writeChain(t testing.TB, e *env, rng *rand.Rand, owner page.XID, pages int) (Ref, []byte) {
	t.Helper()
	value := randomValue(rng, (pages-1)*ChunkSize+1+rng.Intn(ChunkSize-1))
	ref, err := e.ov.Write(testFile, owner, value)
	if err != nil {
		t.Fatal(err)
	}
	return ref, value
}

/**
* checkAccounting comprueba que cada bloque del store (salvo la meta) está exactamente una vez en la lista de libres
* o en una de las cadenas vivas: ni huérfanas ni páginas dobles en la lista.
* @param t testing.TB, e *env, live ...[]page.BlockNumber
**/
func checkAccounting(t testing.TB, e *env, live ...[]page.BlockNumber) {
	t.Helper()
	seen := map[page.BlockNumber]string{}
	note := func(b page.BlockNumber, what string) {
		if prev, ok := seen[b]; ok {
			t.Fatalf("el bloque %d está dos veces: %s y %s", b, prev, what)
		}
		seen[b] = what
	}
	for _, b := range freeList(t, e.pool) {
		note(b, "libre")
	}
	for i, chain := range live {
		for _, b := range chain {
			note(b, fmt.Sprintf("cadena viva %d", i))
		}
	}
	n, err := e.sm.NBlocks(testFile, smgr.ForkMain)
	if err != nil {
		t.Fatal(err)
	}
	for b := page.BlockNumber(1); b < page.BlockNumber(n); b++ {
		if _, ok := seen[b]; !ok {
			t.Fatalf("el bloque %d es huérfano (ni libre ni de una cadena viva)", b)
		}
	}
}

// TestFreeTailResume: una liberación desde el final cortada antes de empezar, a mitad o al terminar, con un corte del
// disco en los tres modos de crashfs y a veces con páginas ya reutilizadas por otra versión, se termina al repetirla:
// la cadena queda liberada entera, sin huérfanas, sin páginas dobles en la lista y sin tocar las cadenas vivas.
func TestFreeTailResume(t *testing.T) {
	modes := []struct {
		name string
		mode crashfs.Mode
	}{{"LoseUnsynced", crashfs.LoseUnsynced}, {"ReorderUnsynced", crashfs.ReorderUnsynced}, {"TornWrites", crashfs.TornWrites}}
	const pages = 6
	for _, md := range modes {
		for _, cut := range []int{0, 1, pages / 2, pages - 1, pages} {
			for seed := range int64(4) {
				t.Run(fmt.Sprintf("%s/cut%d/seed%d", md.name, cut, seed), func(t *testing.T) {
					fs := crashfs.New()
					e := openEnv(t, fs, 24)
					createStore(t, e.sm, testFile)
					rng := rand.New(rand.NewSource(seed))

					dead, _ := writeChain(t, e, rng, 7, pages)
					keep, keepValue := writeChain(t, e, rng, 8, 3)
					other, _ := writeChain(t, e, rng, 9, 2)
					if err := e.ov.Free(testFile, other, 9); err != nil {
						t.Fatal(err)
					}
					keepBlocks := chainBlocks(t, e.pool, keep)
					if err := e.log.WaitDurable(e.log.Published()); err != nil {
						t.Fatal(err)
					}
					if err := e.pool.FlushAll(nil); err != nil {
						t.Fatal(err)
					}
					if err := e.sm.SyncAll(); err != nil {
						t.Fatal(err)
					}

					e.ov.afterFree = func(freed int) error {
						if freed == cut && cut < pages {
							return errCut
						}
						return nil
					}
					if cut > 0 {
						freed, err := e.ov.FreeTail(testFile, dead, 7, 20)
						if cut < pages && (!errors.Is(err, errCut) || freed != cut) {
							t.Fatalf("FreeTail cortado: %d, %v", freed, err)
						}
						if cut == pages && (err != nil || freed != pages) {
							t.Fatalf("FreeTail entero: %d, %v", freed, err)
						}
					}
					if seed%2 == 0 {
						if err := e.log.WaitDurable(e.log.Published()); err != nil {
							t.Fatal(err)
						}
					}

					e.crash(md.mode, seed)
					e2 := openEnv(t, fs, 24)
					var reused []page.BlockNumber
					if seed >= 2 { // otra versión toma páginas liberadas antes de repetir
						ref, value := writeChain(t, e2, rng, 30, 2)
						reused = chainBlocks(t, e2.pool, ref)
						if got, err := e2.ov.Read(testFile, ref); err != nil || !bytes.Equal(got, value) {
							t.Fatalf("la versión nueva: %v", err)
						}
					}
					if _, err := e2.ov.FreeTail(testFile, dead, 7, 21); err != nil {
						t.Fatalf("repetir FreeTail: %v", err)
					}
					if n, err := e2.ov.FreeTail(testFile, dead, 7, 22); err != nil || n != 0 {
						t.Fatalf("FreeTail sin nada que liberar: %d, %v", n, err)
					}
					if got, err := e2.ov.Read(testFile, keep); err != nil || !bytes.Equal(got, keepValue) {
						t.Fatalf("la cadena viva cambió: %v", err)
					}
					checkAccounting(t, e2, keepBlocks, reused)
				})
			}
		}
	}
}

// TestFreeTailChecks: owner 0 y referencias imposibles se rechazan; una página de la versión dañada da ErrBadChain y
// no libera nada; otro dueño no libera nada.
func TestFreeTailChecks(t *testing.T) {
	e := newEnv(t, 16)
	rng := rand.New(rand.NewSource(1))
	ref, _ := writeChain(t, e, rng, 5, 3)
	if _, err := e.ov.FreeTail(testFile, ref, 0, 1); !errors.Is(err, ErrBadRef) {
		t.Fatalf("owner 0: %v", err)
	}
	if _, err := e.ov.FreeTail(testFile, Ref{}, 5, 1); !errors.Is(err, ErrBadRef) {
		t.Fatalf("referencia vacía: %v", err)
	}
	if n, err := e.ov.FreeTail(testFile, ref, 6, 1); err != nil || n != 0 {
		t.Fatalf("otro dueño: %d, %v", n, err)
	}
	blocks := chainBlocks(t, e.pool, ref)
	p := readPage(t, e.pool, blocks[1])
	if err := p.SetLower(page.HeaderSize + 10); err != nil {
		t.Fatal(err)
	}
	writePage(t, e.pool, blocks[1], p)
	if n, err := e.ov.FreeTail(testFile, ref, 5, 1); !errors.Is(err, ErrBadChain) || n != 0 {
		t.Fatalf("cadena dañada: %d, %v", n, err)
	}
	if free := freeList(t, e.pool); len(free) != 0 || slices.Contains(free, blocks[0]) {
		t.Fatalf("una cadena dañada liberó páginas: %v", free)
	}
}
