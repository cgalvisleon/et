package reg

import (
	"sync"
	"testing"
)

/**
* TestIdsUniqueAndOrdered: UUID (v7) y ULID no se repiten pedidos desde muchas
* goroutines a la vez; el UUID además crece en una misma goroutine (el ULID solo es
* ordenable por milisegundo).
**/
func TestIdsUniqueAndOrdered(t *testing.T) {
	for name, gen := range map[string]func() string{"UUID": UUID, "ULID": ULID} {
		const workers, each = 16, 2000
		var mu sync.Mutex
		seen := map[string]bool{}
		var wg sync.WaitGroup
		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ids := make([]string, each)
				for i := range ids {
					ids[i] = gen()
					if name == "UUID" && i > 0 && ids[i] <= ids[i-1] {
						t.Errorf("%s not ordered: %s after %s", name, ids[i], ids[i-1])
						return
					}
				}
				mu.Lock()
				defer mu.Unlock()
				for _, id := range ids {
					if seen[id] {
						t.Errorf("%s repeated: %s", name, id)
					}
					seen[id] = true
				}
			}()
		}
		wg.Wait()
	}
	if v := UUID(); len(v) != 36 || v[14] != '7' {
		t.Fatalf("UUID must be a version 7 UUID: %s", v)
	}
}
