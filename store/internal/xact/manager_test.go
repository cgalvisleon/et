package xact

import (
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/page"
)

// Begin asigna xid únicos y consecutivos desde varios goroutines; End los quita.
func TestBeginEnd(t *testing.T) {
	m := NewManager(0)
	const workers, per = 16, 200
	var mu sync.Mutex
	seen := map[page.XID]bool{}
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range per {
				xid := m.Begin()
				mu.Lock()
				if seen[xid] {
					t.Errorf("xid %d twice", xid)
				}
				seen[xid] = true
				mu.Unlock()
				m.Snapshot(xid)
				m.End(xid)
			}
		}()
	}
	wg.Wait()
	if m.NextXID() != workers*per+1 || len(seen) != workers*per {
		t.Fatalf("next = %d, seen = %d", m.NextXID(), len(seen))
	}
	if m.OldestActive() != m.NextXID() || len(m.running) != 0 {
		t.Fatal("transactions left running")
	}
}

// Snapshot y visibilidad.
func TestSnapshot(t *testing.T) {
	m := NewManager(10)
	a, b, c := m.Begin(), m.Begin(), m.Begin() // 10, 11, 12
	m.End(b)
	v := m.Snapshot(c)
	if v.Xmin != a || v.Xmax != 13 || !slices.Equal(v.InProgress, []page.XID{a, c}) {
		t.Fatalf("snapshot = %+v", v)
	}
	cases := []struct {
		xid  page.XID
		st   Status
		want bool
	}{
		{c, StatusInProgress, true}, // la propia
		{a, StatusCommitted, false}, // en curso al tomarla
		{b, StatusCommitted, true},
		{b, StatusAborted, false},
		{5, StatusCommitted, true},
		{13, StatusCommitted, false}, // posterior
	}
	for _, tc := range cases {
		if got := v.Visible(tc.xid, tc.st); got != tc.want {
			t.Errorf("Visible(%d, %d) = %v", tc.xid, tc.st, got)
		}
	}

	// El Xmin del snapshot de c retiene el horizonte después de que a termina.
	m.End(a)
	if m.OldestActive() != a {
		t.Fatalf("OldestActive = %d, want %d", m.OldestActive(), a)
	}
	m.End(c)
	if m.OldestActive() != 13 {
		t.Fatalf("OldestActive = %d, want 13", m.OldestActive())
	}
	if empty := m.Snapshot(0); empty.Xmin != 13 || len(empty.InProgress) != 0 {
		t.Fatalf("empty snapshot = %+v", empty)
	}
}

// Wait vuelve al terminar la otra, enseguida si no está en curso y con ErrWaitTimeout si se pasa.
func TestWait(t *testing.T) {
	m := NewManager(1)
	x := m.Begin()
	if err := m.Wait(x, 20*time.Millisecond); !errors.Is(err, ErrWaitTimeout) {
		t.Fatalf("timeout: %v", err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := m.Wait(x, time.Minute); err != nil {
				t.Error(err)
			}
		}()
	}
	time.Sleep(10 * time.Millisecond)
	m.End(x)
	wg.Wait()
	if err := m.Wait(x, 0); err != nil {
		t.Fatalf("ended xid: %v", err)
	}
	if err := m.Wait(999, time.Millisecond); err != nil {
		t.Fatalf("unknown xid: %v", err)
	}
}
