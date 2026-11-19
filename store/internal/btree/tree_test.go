package btree

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/overflow"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// TestCreate: un store nuevo está vacío y es válido; crearlo otra vez falla; ids fuera de rango se rechazan.
func TestCreate(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 16)
	tr := createTree(t, e)
	st := checkTree(t, tr, model{})
	if st.Levels != 1 || st.Pages != 1 || st.Entries != 0 {
		t.Fatalf("store nuevo: %+v", st)
	}
	if _, err := Create(e.config()); !errors.Is(err, ErrExists) {
		t.Fatalf("Create otra vez: %v, se esperaba ErrExists", err)
	}
	if _, err := tr.Write("", Write{Value: []byte("x")}); !errors.Is(err, ErrIDLength) {
		t.Fatalf("id vacío: %v", err)
	}
	if _, err := tr.Write(strings.Repeat("a", MaxIDLen+1), Write{}); !errors.Is(err, ErrIDLength) {
		t.Fatalf("id de %d B: %v", MaxIDLen+1, err)
	}
	if _, err := tr.Write("a", Write{Value: make([]byte, MaxEntrySize)}); !errors.Is(err, ErrEntryTooLarge) {
		t.Fatalf("valor que no cabe en línea: %v", err)
	}
	if _, err := Open(Config{}); !errors.Is(err, ErrConfig) {
		t.Fatalf("Open sin configuración: %v", err)
	}
}

// TestRandomInserts: miles de escrituras, reemplazos y borrados al azar con una caché chica (expulsiones), contra un
// modelo en memoria; el árbol crece a varios niveles.
func TestRandomInserts(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 24)
	tr := createTree(t, e)
	m := model{}
	rng := rand.New(rand.NewSource(1))
	for i := range 4000 {
		id := randomID(rng, 40)
		if rng.Intn(8) == 0 {
			if _, live := m[id]; live || rng.Intn(2) == 0 {
				remove(t, tr, m, id)
			}
			continue
		}
		put(t, tr, m, id, randomValue(rng, 120))
		if i%1000 == 999 {
			checkTree(t, tr, m)
		}
	}
	st := checkTree(t, tr, m)
	if st.Levels < 2 || st.Incomplete != 0 {
		t.Fatalf("estructura: %+v", st)
	}
	if s := e.pool.Stats(); s.Misses <= int64(s.Buffers) {
		t.Fatalf("la caché no expulsó nada: %+v", s)
	}
}

// TestAscendingFastPath: ids crecientes (como SID, ULID o UUID v7) van por la inserción rápida, y la partición de la
// hoja de más a la derecha deja la izquierda casi llena (90 %); con ids largos el árbol llega a tres niveles.
func TestAscendingFastPath(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 24)
	tr := createTree(t, e)
	m := model{}
	prefix := strings.Repeat("p", 300)
	value := bytes.Repeat([]byte{7}, 40)
	fastHits := 0
	for i := range 3000 {
		id := fmt.Sprintf("%s%08d", prefix, i)
		hint := tr.fastLeaf.Load()
		put(t, tr, m, id, value)
		if hint != 0 && hint == tr.fastLeaf.Load() {
			fastHits++
		}
	}
	st := checkTree(t, tr, m)
	if st.Levels < 3 {
		t.Fatalf("se esperaban 3 niveles o más: %+v", st)
	}
	if fastHits < 2000 {
		t.Fatalf("la inserción rápida se usó %d veces de 3000", fastHits)
	}
	perLeaf := capacity / page_SpaceFor(EntryHeaderSize+len(prefix)+8+len(value))
	leaves := st.Pages - (st.Pages / 10) // aproximado: casi todas son hojas
	if minLeaves := 3000 / perLeaf; leaves > minLeaves*12/10+2 {
		t.Fatalf("%d hojas para %d entradas (mínimo %d): las particiones al final no llenan la izquierda", leaves, 3000, minLeaves)
	}
}

// TestDescendingInserts: ids decrecientes parten siempre la hoja de más a la izquierda.
func TestDescendingInserts(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 24)
	tr := createTree(t, e)
	m := model{}
	for i := 2000; i > 0; i-- {
		put(t, tr, m, fmt.Sprintf("key-%06d", i), []byte(fmt.Sprintf("value of %d with some padding", i)))
	}
	if st := checkTree(t, tr, m); st.Levels < 2 {
		t.Fatalf("estructura: %+v", st)
	}
}

// TestIDLengthLimits: ids de 1 B y de 2.000 B (los de 2.000 con un prefijo común largo, así los pivotes no se
// pueden truncar y las internas guardan 3 o 4 enlaces: muchas particiones de internas y raíces nuevas); valores en
// línea al máximo y referencias a desborde.
func TestIDLengthLimits(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 32)
	tr := createTree(t, e)
	m := model{}
	for c := range 256 {
		put(t, tr, m, string([]byte{byte(c)}), []byte{byte(c), 1, 2})
	}
	prefix := strings.Repeat("L", MaxIDLen-8)
	maxInline := MaxEntrySize - EntryHeaderSize - MaxIDLen
	rng := rand.New(rand.NewSource(2))
	for _, i := range rng.Perm(300) {
		id := fmt.Sprintf("%s%08d", prefix, i)
		put(t, tr, m, id, bytes.Repeat([]byte{byte(i)}, maxInline))
	}
	st := checkTree(t, tr, m)
	if st.Levels < 4 {
		t.Fatalf("se esperaban 4 niveles o más con pivotes de 2.000 B: %+v", st)
	}

	// Una versión en desborde: el árbol guarda la Ref y Lookup la devuelve sin leer la cadena.
	big := prefix + "overflow"
	ref := &overflow.Ref{First: 12345, Len: 10 << 20}
	if _, err := tr.Write(big, Write{XID: 1, Overflow: ref, Commit: true}); err != nil {
		t.Fatal(err)
	}
	v, ok, err := tr.Lookup(big, nil)
	if err != nil || !ok || v.Flags&EntryOverflow == 0 || v.Overflow != *ref || v.Value != nil {
		t.Fatalf("Lookup de desborde: %+v, %v, %v", v, ok, err)
	}
	if _, err := tr.Verify(); err != nil {
		t.Fatal(err)
	}
}

// TestManyVersionsOfOneID: sin limpieza (fase 1B) las versiones de un id se acumulan y se reparten en muchas hojas;
// los separadores llevan el lsn (PivotHasLSN). Lookup y Scan ven la más nueva; el borrado esconde el id.
func TestManyVersionsOfOneID(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 24)
	tr := createTree(t, e)
	m := model{}
	put(t, tr, m, "a", []byte("before"))
	put(t, tr, m, "z", []byte("after"))
	for i := range 1500 {
		put(t, tr, m, "hot", []byte(fmt.Sprintf("version %04d of the hot id, padded to make it longer", i)))
	}
	st := checkTree(t, tr, m)
	if st.Entries != 1502 || st.Levels < 2 {
		t.Fatalf("estructura: %+v", st)
	}
	remove(t, tr, m, "hot")
	checkTree(t, tr, m)
	if res := remove(t, tr, m, "hot"); res.Applied {
		t.Fatal("Remove de un id ya borrado escribió una lápida")
	}
	put(t, tr, m, "hot", []byte("back"))
	checkTree(t, tr, m)
}

// TestWriteConditions: Check recibe la última versión y decide; CountDelta y las marcas de Count siguen a la versión
// viva anterior.
func TestWriteConditions(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 16)
	tr := createTree(t, e)
	insert := func(id, v string) Result {
		res, err := tr.Write(id, Write{XID: 1, Value: []byte(v), Commit: true,
			Check: func(l Latest) Action {
				if l.Live {
					return Skip
				}
				return Apply
			}})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	if res := insert("k", "1"); !res.Applied || res.CountDelta != 1 {
		t.Fatalf("Insert nuevo: %+v", res)
	}
	if res := insert("k", "2"); res.Applied {
		t.Fatalf("Insert de un id vivo escribió: %+v", res)
	}
	var seen Latest
	res, err := tr.Write("k", Write{XID: 1, Value: []byte("3"), Commit: true, Check: func(l Latest) Action {
		seen = l
		return Apply
	}})
	if err != nil || res.CountDelta != 0 || !seen.Live || string(seen.Version.Value) != "1" {
		t.Fatalf("Update: %+v %v, vio %+v", res, err, seen)
	}
	if res, _ := tr.Write("k", Write{XID: 1, Tombstone: true, Commit: true}); res.CountDelta != -1 {
		t.Fatalf("Remove: %+v", res)
	}
	if res := insert("k", "4"); !res.Applied || res.CountDelta != 1 {
		t.Fatalf("Insert tras borrar: %+v", res)
	}
	var lsns []uint64
	for i := range 3 {
		res := put(t, tr, nil, fmt.Sprint("x", i), []byte("v"))
		lsns = append(lsns, uint64(res.LSN))
	}
	if !slices.IsSorted(lsns) || lsns[0] == 0 {
		t.Fatalf("lsn de las escrituras: %v", lsns)
	}
}

// page_SpaceFor es lo que ocupa un ítem de n bytes con su puntero (para las cuentas de las pruebas).
func page_SpaceFor(n int) int {
	return (n+7)&^7 + 4
}
