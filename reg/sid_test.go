package reg

import (
	"math/rand"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

/**
* TestSIDOrderAndUnique: Un millón de SID seguidos: todos distintos y cada texto mayor
* que el anterior.
**/
func TestSIDOrderAndUnique(t *testing.T) {
	g, _ := NewSIDGenerator(7)
	prev := ""
	seen := make(map[string]bool, 1_000_000)
	for range 1_000_000 {
		s := g.NewString()
		if s <= prev {
			t.Fatalf("not ordered: %s after %s", s, prev)
		}
		if seen[s] {
			t.Fatalf("repeated %s", s)
		}
		seen[s] = true
		prev = s
	}
}

/**
* TestSIDConcurrent: 16 goroutines sobre el mismo generador no repiten.
**/
func TestSIDConcurrent(t *testing.T) {
	g, _ := NewSIDGenerator(1)
	var mu sync.Mutex
	seen := map[int64]bool{}
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			ids := make([]int64, 20000)
			for i := range ids {
				ids[i] = g.Next()
			}
			mu.Lock()
			defer mu.Unlock()
			for _, id := range ids {
				if seen[id] {
					t.Errorf("repeated %d", id)
				}
				seen[id] = true
			}
		})
	}
	wg.Wait()
}

/**
* TestSIDTwoNodes: Dos nodos con el reloj congelado en el mismo ms nunca chocan,
* aunque la secuencia se agote y tomen ms prestados.
**/
func TestSIDTwoNodes(t *testing.T) {
	a, _ := NewSIDGenerator(1)
	b, _ := NewSIDGenerator(2)
	fixed := func() int64 { return SIDEpoch + 1000 }
	a.now, b.now = fixed, fixed
	seen := map[int64]bool{}
	for range 10000 {
		for _, id := range []int64{a.Next(), b.Next()} {
			if seen[id] {
				t.Fatalf("collision %d", id)
			}
			seen[id] = true
		}
	}
}

/**
* TestSIDClockBackwards: Si el reloj retrocede, el SID siguiente sigue siendo mayor.
**/
func TestSIDClockBackwards(t *testing.T) {
	g, _ := NewSIDGenerator(0)
	now := SIDEpoch + 5000
	g.now = func() int64 { return now }
	a := g.Next()
	now -= 2000
	if b := g.Next(); b <= a {
		t.Fatalf("went backwards: %d <= %d", b, a)
	}
}

/**
* TestSIDTextSortEqualsNumericSort: Codificar y decodificar da el mismo número, y
* ordenar los textos da el mismo orden que ordenar los números.
**/
func TestSIDTextSortEqualsNumericSort(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	ids := make([]int64, 100000)
	strs := make([]string, len(ids))
	for i := range ids {
		ids[i] = r.Int63()
		strs[i] = EncodeSID(ids[i])
		back, err := DecodeSID(strs[i])
		if err != nil || back != ids[i] {
			t.Fatalf("roundtrip %d → %s → %d (%v)", ids[i], strs[i], back, err)
		}
	}
	slices.Sort(ids)
	slices.Sort(strs)
	for i := range ids {
		if EncodeSID(ids[i]) != strs[i] {
			t.Fatalf("order mismatch at %d", i)
		}
	}
	if s := EncodeSID(1<<63 - 1); s != "AzL8n0Y58m7" {
		t.Fatalf("max = %s", s)
	}
}

/**
* TestSIDDecodeRejects: Rechaza largo incorrecto, caracteres inválidos y valores fuera de rango.
**/
func TestSIDDecodeRejects(t *testing.T) {
	for _, s := range []string{"", "abc", "zzzzzzzzzzz", "AzL8n0Y58m8", "0000000000-"} {
		if _, err := DecodeSID(s); err == nil {
			t.Fatalf("accepted %q", s)
		}
	}
}

/**
* TestSIDParts: Parts devuelve el nodo, la secuencia y el instante de creación.
**/
func TestSIDParts(t *testing.T) {
	g, _ := NewSIDGenerator(42)
	before := time.Now().UTC().Truncate(time.Millisecond)
	ts, node, seq := SIDParts(g.Next())
	if node != 42 || seq != 0 || ts.Before(before) || ts.After(time.Now()) {
		t.Fatalf("parts: %v %d %d", ts, node, seq)
	}
	if _, err := NewSIDGenerator(1024); err == nil {
		t.Fatal("accepted node 1024")
	}
}

/**
* TestSIDDefault: El generador del proceso: SetSIDNode fija el nodo sin retroceder, y
* GenSID/GetSID/TagSID siguen la convención de UUID/ULID/XID.
**/
func TestSIDDefault(t *testing.T) {
	a := SIDInt()
	if err := SetSIDNode(9); err != nil {
		t.Fatal(err)
	}
	b := SIDInt()
	if b <= a {
		t.Fatalf("SetSIDNode went backwards: %d <= %d", b, a)
	}
	if _, node, _ := SIDParts(b); node != 9 || SIDNode() != 9 {
		t.Fatalf("node = %d", node)
	}
	if err := SetSIDNode(-1); err == nil {
		t.Fatal("accepted node -1")
	}
	if len(SID()) != 11 || !strings.HasPrefix(GenSID("orders"), "orders:") {
		t.Fatal("SID/GenSID shape")
	}
	if GetSID("x1") != "x1" || len(GetSID("new")) != 11 || TagSID("t", "x1") != "x1" || !strings.HasPrefix(TagSID("t", "*"), "t:") {
		t.Fatal("GetSID/TagSID")
	}
}

/**
* BenchmarkSID: Costo de un SID de texto.
**/
func BenchmarkSID(b *testing.B) {
	g, _ := NewSIDGenerator(1)
	for b.Loop() {
		_ = g.NewString()
	}
}

/**
* TestGenSnowflake: 200.000 seguidos (la versión con rand(1000) repetía el 87,7%) y 8 goroutines a la vez: sin
* repetidos, 19 dígitos, el texto en orden y el mismo número que su SID.
**/
func TestGenSnowflake(t *testing.T) {
	prev := ""
	seen := make(map[string]bool, 200_000)
	for range 200_000 {
		s := GenSnowflake()
		if len(s) != 19 || s <= prev || seen[s] {
			t.Fatalf("snowflake %q después de %q (repetido: %v)", s, prev, seen[s])
		}
		seen[s] = true
		prev = s
	}
	n, err := strconv.ParseInt(prev, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	if _, node, _ := SIDParts(n); node != SIDNode() {
		t.Fatalf("nodo %d, el del proceso es %d", node, SIDNode())
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			ids := make([]string, 20_000)
			for i := range ids {
				ids[i] = GenSnowflake()
			}
			mu.Lock()
			defer mu.Unlock()
			for _, id := range ids {
				if seen[id] {
					t.Errorf("repetido %s", id)
				}
				seen[id] = true
			}
		})
	}
	wg.Wait()
}
