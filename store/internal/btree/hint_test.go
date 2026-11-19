package btree

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// xidState es el estado de una transacción en lsnStatus, con el lsn de confirmación de su grupo.
type xidState struct {
	st  xact.Status
	lsn page.LSN
}

// lsnStatus es un doble del archivo de estados que también devuelve el lsn de confirmación (sin anotar = en curso).
type lsnStatus struct {
	mu sync.Mutex
	m  map[page.XID]xidState
}

func newLSNStatus() *lsnStatus { return &lsnStatus{m: map[page.XID]xidState{}} }

func (s *lsnStatus) Status(xid page.XID) (xact.Status, page.LSN, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	x := s.m[xid]
	return x.st, x.lsn, nil
}

func (s *lsnStatus) set(xid page.XID, st xact.Status, lsn page.LSN) {
	s.mu.Lock()
	s.m[xid] = xidState{st: st, lsn: lsn}
	s.mu.Unlock()
}

func (s *lsnStatus) get(xid page.XID) xact.Status {
	st, _, _ := s.Status(xid)
	return st
}

/**
* treeWith crea el store de prueba con el archivo de estados status y la compuerta gate (pueden ser nil).
* @param t testing.TB, e *env, status StatusReader, gate HintGate
* @return *BTree
**/
func treeWith(t testing.TB, e *env, status StatusReader, gate HintGate) *BTree {
	t.Helper()
	cfg := e.config()
	cfg.Status, cfg.HintGate = status, gate
	if _, err := Create(cfg); err != nil {
		t.Fatal(err)
	}
	tr, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

/**
* reopenWith abre el store de prueba ya creado con el archivo de estados status.
* @param t testing.TB, e *env, status StatusReader
* @return *BTree
**/
func reopenWith(t testing.TB, e *env, status StatusReader) *BTree {
	t.Helper()
	cfg := e.config()
	cfg.Status = status
	tr, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

/**
* leafEntries devuelve todas las entradas de las hojas, en orden, con sus marcas (recorriendo los enlaces a la
* derecha desde la hoja de más a la izquierda).
* @param t testing.TB, tr *BTree
* @return []Version
**/
func leafEntries(t testing.TB, tr *BTree) []Version {
	t.Helper()
	h, n, _, err := tr.descend(minusInf, 0, buffer.Share, false)
	if err != nil {
		t.Fatal(err)
	}
	var out []Version
	for {
		for off := n.firstData(); off <= n.maxOff(); off++ {
			e, err := n.entryAt(off)
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, e.version())
		}
		if n.rightmost() {
			h.Release()
			return out
		}
		if h, n, err = tr.stepRight(h, n); err != nil {
			t.Fatal(err)
		}
	}
}

/**
* entryOf busca en leafEntries la versión (id, lsn).
* @param t testing.TB, tr *BTree, id string, lsn page.LSN
* @return Version
**/
func entryOf(t testing.TB, tr *BTree, id string, lsn page.LSN) Version {
	t.Helper()
	for _, v := range leafEntries(t, tr) {
		if v.ID == id && v.LSN == lsn {
			return v
		}
	}
	t.Fatalf("no está la versión (%q, %d)", id, lsn)
	return Version{}
}

/**
* walRecords devuelve los registros de kind del WAL de prueba desde wal.Start (copias de sus campos fijos).
* @param t testing.TB, e *env, kind wal.Kind
* @return []wal.Record: sin imágenes ni deltas
**/
func walRecords(t testing.TB, e *env, kind wal.Kind) []wal.Record {
	t.Helper()
	if err := e.log.WaitDurable(e.log.Published()); err != nil {
		t.Fatal(err)
	}
	var out []wal.Record
	if _, err := wal.NewReader(e.fs, testDir).Scan(wal.Start, func(rec *wal.Record) error {
		if rec.Kind == kind {
			r := wal.Record{LSN: rec.LSN, Kind: rec.Kind, XID: rec.XID}
			for _, b := range rec.Blocks {
				r.Blocks = append(r.Blocks, wal.BlockRef{Block: b.Block, Flags: b.Flags})
			}
			out = append(out, r)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

/**
* writeAs escribe una versión de la transacción xid (sin confirmar).
* @param t testing.TB, tr *BTree, xid page.XID, id, value string
* @return page.LSN
**/
func writeAs(t testing.TB, tr *BTree, xid page.XID, id, value string) page.LSN {
	t.Helper()
	res, err := tr.Write(id, Write{XID: xid, Value: []byte(value)})
	if err != nil || !res.Applied {
		t.Fatalf("Write(%q) de %d: %+v %v", id, xid, res, err)
	}
	return res.LSN
}

// TestHintsRule: una lectura marca confirmada una versión solo cuando el WAL es durable hasta el lsn de confirmación
// de su grupo, marca abortada la abortada y no marca la que sigue en curso; con la página anotada desde el punto de
// rehacer, sin WAL; la primera marca tras un checkpoint va al WAL (KindSetHints con la imagen de la página), y las
// siguientes de esa página ya no. Con y sin HintGate.
func TestHintsRule(t *testing.T) {
	for _, gated := range []bool{false, true} {
		t.Run(fmt.Sprintf("gate=%v", gated), func(t *testing.T) {
			e := openEnv(t, crashfs.New(), wal.Start, 16)
			status := newLSNStatus()
			var gate HintGate
			if gated {
				gate = &sync.RWMutex{}
			}
			tr := treeWith(t, e, status, gate)
			lsnA := writeAs(t, tr, 10, "a", "va")
			lsnB := writeAs(t, tr, 11, "b", "vb")
			lsnC := writeAs(t, tr, 12, "c", "vc")
			lsnD := writeAs(t, tr, 13, "d", "vd")
			future := e.log.Published() + 1000
			status.set(10, xact.StatusCommitted, future) // confirmada, pero su confirmación no es durable
			status.set(11, xact.StatusAborted, 0)
			status.set(13, xact.StatusCommitted, 0)
			for _, id := range []string{"a", "b", "c", "d"} {
				if _, _, err := tr.Lookup(id, nil); err != nil {
					t.Fatal(err)
				}
			}
			flags := func(id string, lsn page.LSN) EntryFlags { return entryOf(t, tr, id, lsn).Flags & hintFlags }
			if f := flags("a", lsnA); f != 0 {
				t.Fatalf("marca antes de que la confirmación sea durable: %#x", f)
			}
			if f := flags("b", lsnB); f != EntryXidAborted {
				t.Fatalf("abortada: %#x", f)
			}
			if f := flags("c", lsnC); f != 0 {
				t.Fatalf("en curso: %#x", f)
			}
			if f := flags("d", lsnD); f != EntryXidCommitted {
				t.Fatalf("confirmada y durable: %#x", f)
			}
			if err := e.log.WaitDurable(e.log.Published()); err != nil {
				t.Fatal(err)
			}
			status.set(10, xact.StatusCommitted, e.log.Durable())
			if _, _, err := tr.Lookup("a", nil); err != nil {
				t.Fatal(err)
			}
			if f := flags("a", lsnA); f != EntryXidCommitted {
				t.Fatalf("confirmada con el WAL ya durable: %#x", f)
			}
			if n := len(walRecords(t, e, wal.KindSetHints)); n != 0 {
				t.Fatalf("%d KindSetHints con la página ya anotada desde el punto de rehacer", n)
			}

			// Tras un checkpoint, la primera marca en la hoja va al WAL con la imagen.
			checkpoint(t, e)
			status.set(12, xact.StatusAborted, 0)
			if _, _, err := tr.Lookup("c", nil); err != nil {
				t.Fatal(err)
			}
			recs := walRecords(t, e, wal.KindSetHints)
			if len(recs) != 1 || recs[0].Blocks[0].Flags&wal.BlockHasImage == 0 || recs[0].Blocks[0].Block != rootBlock {
				t.Fatalf("KindSetHints tras el checkpoint: %+v", recs)
			}
			if f := flags("c", lsnC); f != EntryXidAborted {
				t.Fatalf("marca anotada en el WAL: %#x", f)
			}
			lsnE := writeAs(t, tr, 14, "e", "ve")
			status.set(14, xact.StatusCommitted, 0)
			if err := tr.Scan("", nil, func(Version) (bool, error) { return true, nil }); err != nil {
				t.Fatal(err)
			}
			if f := flags("e", lsnE); f != EntryXidCommitted {
				t.Fatalf("marca puesta por Scan: %#x", f)
			}
			if n := len(walRecords(t, e, wal.KindSetHints)); n != 1 {
				t.Fatalf("%d KindSetHints: la segunda marca de la página no tenía que ir al WAL", n)
			}
			if _, err := tr.Verify(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestHintsConcurrent: lecturas (Lookup, Scan, ScanBackward) que ponen marcas a la vez que escrituras, transacciones
// que terminan y checkpoints que cierran HintGate: sin carreras (con -race), y al final cada marca coincide con el
// estado de su transacción.
func TestHintsConcurrent(t *testing.T) {
	e := openEnvWith(t, crashfs.New(), wal.Start, 32, 0)
	status := newLSNStatus()
	gate := &sync.RWMutex{}
	tr := treeWith(t, e, status, gate)
	const ids = 300
	var xidMu sync.Mutex
	nextXID := page.XID(100)
	newXID := func() page.XID {
		xidMu.Lock()
		defer xidMu.Unlock()
		nextXID++
		return nextXID
	}
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for w := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(w)))
			for range 400 {
				xid := newXID()
				id := fmt.Sprintf("id%04d", rng.Intn(ids))
				_, err := tr.Write(id, Write{XID: xid, Value: randomValue(rng, 120)})
				var we *WaitError
				if errors.As(err, &we) {
					status.set(xid, xact.StatusAborted, 0)
					continue
				}
				if err != nil {
					errs <- err
					return
				}
				switch rng.Intn(4) {
				case 0:
					status.set(xid, xact.StatusAborted, 0)
				default:
					status.set(xid, xact.StatusCommitted, e.log.Published())
				}
			}
		}()
	}
	for r := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(100 + r)))
			for i := range 300 {
				var err error
				switch i % 10 {
				case 0:
					err = tr.Scan("", nil, func(Version) (bool, error) { return true, nil })
				case 1:
					err = tr.ScanBackward("", nil, func(Version) (bool, error) { return true, nil })
				default:
					_, _, err = tr.Lookup(fmt.Sprintf("id%04d", rng.Intn(ids)), nil)
				}
				if err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 10 {
			gate.Lock()
			_, err := e.log.BeginCheckpoint(nil)
			gate.Unlock()
			if err == nil {
				err = e.pool.FlushAll(nil)
			}
			if err == nil {
				err = e.log.WaitDurable(e.log.Published())
			}
			if err != nil {
				errs <- err
				return
			}
		}
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if _, err := tr.Verify(); err != nil {
		t.Fatal(err)
	}
	for _, v := range leafEntries(t, tr) {
		st := status.get(v.XID)
		switch {
		case v.Flags&EntryXidCommitted != 0 && st != xact.StatusCommitted:
			t.Fatalf("versión (%q, %d) marcada confirmada con estado %d", v.ID, v.LSN, st)
		case v.Flags&EntryXidAborted != 0 && st != xact.StatusAborted:
			t.Fatalf("versión (%q, %d) marcada abortada con estado %d", v.ID, v.LSN, st)
		}
	}
}

// hintOp es una versión escrita en TestHintsCrash: su id, transacción y valor.
type hintOp struct {
	id    string
	xid   page.XID
	value string
}

/**
* commitRecord escribe el KindCommit de xid (sin datos: en las pruebas basta con el xid) y devuelve su lsn.
* @param t testing.TB, e *env, xid page.XID
* @return page.LSN
**/
func commitRecord(t testing.TB, e *env, xid page.XID) page.LSN {
	t.Helper()
	res, err := e.log.Reserve(&wal.Record{XID: xid, Kind: wal.KindCommit}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := res.Write(); err != nil {
		t.Fatal(err)
	}
	return res.LSN()
}

// TestHintsCrash: con lecturas que ponen marcas, confirmaciones que a veces no llegan a ser durables (o nunca se
// escriben), checkpoints y
// páginas que van al disco, un corte en los tres modos de crashfs nunca deja visible una versión cuya confirmación se
// perdió: tras rehacer, cada id muestra la versión más nueva de una transacción confirmada en el WAL recuperado, y
// ninguna marca de confirmada queda en una versión de otra transacción.
func TestHintsCrash(t *testing.T) {
	modes := []struct {
		name string
		mode crashfs.Mode
	}{{"LoseUnsynced", crashfs.LoseUnsynced}, {"ReorderUnsynced", crashfs.ReorderUnsynced}, {"TornWrites", crashfs.TornWrites}}
	for _, md := range modes {
		for seed := range int64(6) {
			t.Run(fmt.Sprintf("%s/seed%d", md.name, seed), func(t *testing.T) {
				fs := crashfs.New()
				e := openEnv(t, fs, wal.Start, 20)
				status := newLSNStatus()
				tr := treeWith(t, e, status, nil)
				redo := checkpoint(t, e)
				rng := rand.New(rand.NewSource(seed))
				var ops []hintOp
				const n = 400
				for i := range n {
					xid := page.XID(100 + i)
					id := fmt.Sprintf("k%03d", rng.Intn(60))
					value := fmt.Sprintf("v%d-%x", i, rng.Int63())
					_, err := tr.Write(id, Write{XID: xid, Value: []byte(value)})
					var we *WaitError
					if errors.As(err, &we) {
						continue
					}
					if err != nil {
						t.Fatal(err)
					}
					ops = append(ops, hintOp{id: id, xid: xid, value: value})
					switch r := rng.Intn(10); {
					case r < 6:
						status.set(xid, xact.StatusCommitted, commitRecord(t, e, xid))
					case r < 7:
						// Confirmada en memoria con una confirmación que nunca llega al WAL (como Deferred): no
						// puede quedar marcada.
						status.set(xid, xact.StatusCommitted, e.log.Published()+1<<40)
					case r < 9:
						status.set(xid, xact.StatusAborted, 0)
					}
					for range 6 {
						if _, _, err := tr.Lookup(fmt.Sprintf("k%03d", rng.Intn(60)), nil); err != nil {
							t.Fatal(err)
						}
					}
					switch {
					case i == n/2 || rng.Intn(30) == 0:
						redo = checkpoint(t, e)
					case rng.Intn(25) == 0:
						if err := e.log.WaitDurable(e.log.Published()); err != nil {
							t.Fatal(err)
						}
					case rng.Intn(40) == 0:
						if err := e.pool.FlushAll(nil); err != nil {
							t.Fatal(err)
						}
					}
				}

				hinted := 0
				for _, v := range leafEntries(t, tr) {
					if v.Flags&hintFlags != 0 {
						hinted++
					}
				}
				t.Logf("%d versiones, %d con marca antes del corte, %d KindSetHints", len(ops), hinted,
					len(walRecords(t, e, wal.KindSetHints)))
				fs.Crash(md.mode, seed)
				committed := map[page.XID]bool{}
				e2 := openEnvHook(t, fs, redo, 20, -1, func(rec *wal.Record) {
					if rec.Kind == wal.KindCommit {
						committed[rec.XID] = true
					}
				})
				// Las confirmaciones anteriores al punto de rehacer no se vuelven a leer: las da el WAL entero.
				if _, err := wal.NewReader(fs, testDir).Scan(wal.Start, func(rec *wal.Record) error {
					if rec.Kind == wal.KindCommit {
						committed[rec.XID] = true
					}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
				recovered := newLSNStatus()
				for xid := range committed {
					recovered.set(xid, xact.StatusCommitted, 0)
				}
				tr2 := reopenWith(t, e2, recovered)
				if _, err := tr2.Verify(); err != nil {
					t.Fatal(err)
				}
				want := map[string]string{}
				for _, o := range ops {
					if committed[o.xid] {
						want[o.id] = o.value
					}
				}
				for i := range 60 {
					id := fmt.Sprintf("k%03d", i)
					v, ok, err := tr2.Lookup(id, nil)
					if err != nil {
						t.Fatal(err)
					}
					if w, has := want[id]; ok != has || (ok && string(v.Value) != w) {
						t.Fatalf("%s tras el corte: %q %v, se esperaba %q %v", id, v.Value, ok, w, has)
					}
				}
				for _, v := range leafEntries(t, tr2) {
					if v.Flags&EntryXidCommitted != 0 && !committed[v.XID] {
						t.Fatalf("versión (%q, %d) marcada confirmada sin su KindCommit", v.ID, v.LSN)
					}
				}
			})
		}
	}
}
