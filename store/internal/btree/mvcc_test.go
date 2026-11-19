package btree

import (
	"errors"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// statusTable es un doble del archivo de estados: xid → estado (sin anotar = en curso).
type statusTable struct {
	mu sync.Mutex
	m  map[page.XID]xact.Status
}

func (s *statusTable) Status(xid page.XID) (xact.Status, page.LSN, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.m[xid], 0, nil
}

func (s *statusTable) set(xid page.XID, st xact.Status) {
	s.mu.Lock()
	s.m[xid] = st
	s.mu.Unlock()
}

// TestVisibility: con el archivo de estados, Lookup sin snapshot ve la última confirmada, con snapshot la que ve su
// foto; una escritura sobre una versión de otra transacción en curso devuelve WaitError, sobre una confirmada después
// de la foto ErrSerialization, y salta las abortadas.
func TestVisibility(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 16)
	status := &statusTable{m: map[page.XID]xact.Status{10: xact.StatusCommitted}}
	cfg := e.config()
	cfg.Status = status
	if _, err := Create(cfg); err != nil {
		t.Fatal(err)
	}
	tr, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	write := func(xid page.XID, v string, snap *xact.Snapshot) (Result, error) {
		return tr.Write("k", Write{XID: xid, Value: []byte(v), Snapshot: snap})
	}
	lookup := func(snap *xact.Snapshot) string {
		v, ok, err := tr.Lookup("k", snap)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			return "<none>"
		}
		return string(v.Value)
	}

	if _, err := write(10, "v10", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := write(20, "v20", nil); err != nil { // 20 sigue en curso
		t.Fatal(err)
	}
	if got := lookup(nil); got != "v10" {
		t.Fatalf("Read suelto con 20 en curso: %s", got)
	}
	var we *WaitError
	if _, err := write(30, "v30", nil); !errors.As(err, &we) || we.XID != 20 {
		t.Fatalf("escribir sobre 20 en curso: %v", err)
	}
	if _, err := write(20, "v20b", nil); err != nil { // la propia transacción sí puede
		t.Fatal(err)
	}

	before := &xact.Snapshot{Own: 40, Xmin: 20, Xmax: 40, InProgress: []page.XID{20}}
	status.set(20, xact.StatusCommitted)
	if got := lookup(nil); got != "v20b" {
		t.Fatalf("Read suelto con 20 confirmada: %s", got)
	}
	if got := lookup(before); got != "v10" {
		t.Fatalf("snapshot de antes de confirmar 20: %s", got)
	}
	if _, err := write(40, "v40", before); !errors.Is(err, ErrSerialization) {
		t.Fatalf("escribir con un snapshot que no ve la última confirmada: %v", err)
	}

	if _, err := write(50, "v50", nil); err != nil {
		t.Fatal(err)
	}
	status.set(50, xact.StatusAborted)
	if got := lookup(nil); got != "v20b" {
		t.Fatalf("con 50 abortada: %s", got)
	}
	res, err := write(60, "v60", nil)
	if err != nil || !res.Applied {
		t.Fatalf("escribir sobre una abortada: %+v %v", res, err)
	}
	own := &xact.Snapshot{Own: 60, Xmin: 60, Xmax: 61}
	if got := lookup(own); got != "v60" {
		t.Fatalf("la propia transacción ve su versión: %s", got)
	}
}
