package store

import (
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/fsm"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/xact"
)

/**
* smallXact es testConfig con archivos de estados de una página (32.768 xids) y el primer xid de un motor nuevo
* first: así unas pocas transacciones cruzan un archivo y el truncado se ve sin millones de xids.
* @param first page.XID
* @return config
**/
func smallXact(first page.XID) config {
	cfg := testConfig()
	cfg.xactFilePages = 1
	cfg.firstXID = first
	cfg.freezeTableAge = xact.XidsPerPage
	cfg.vacuumXids = xact.XidsPerPage / 2
	return cfg
}

/**
* xactFiles lista los archivos de estados del motor de prueba.
* @param t testing.TB, fsys vfs.FS
* @return []string
**/
func xactFiles(t testing.TB, fsys vfs.FS) []string {
	t.Helper()
	names, err := fsys.ReadDir(filepath.Join(testDir, xact.Dir))
	mustOK(t, err)
	return names
}

/**
* xactFileOf es el nombre del archivo de estados (de una página) que tiene xid.
* @param xid page.XID
* @return string
**/
func xactFileOf(xid page.XID) string {
	return fmt.Sprintf("%04d", uint64(xid)/xact.XidsPerPage+1)
}

/**
* checkLeaves lee todas las hojas del store y comprueba lo que el congelado promete: toda versión con xid < below está
* congelada o «liberándose» (no consulta su estado), una hoja con bits del mapa tiene page.FlagAllVisible y una
* todo-congelada tiene todas sus versiones congeladas.
* @param t testing.TB, s *Store, below page.XID
* @return int: versiones congeladas
**/
func checkLeaves(t testing.TB, s *Store, below page.XID) int {
	t.Helper()
	n, err := s.e.sm.NBlocks(s.file, smgr.ForkMain)
	mustOK(t, err)
	frozen := 0
	for blk := page.BlockNumber(1); blk < page.BlockNumber(n); blk++ {
		p := readPage(t, s, blk)
		o, err := p.Opaque()
		if p.IsNew() || err != nil || o.Flags&page.OpaqueLeaf == 0 {
			continue
		}
		bits, err := s.e.free.Get(s.file, blk)
		mustOK(t, err)
		if bits != 0 && p.Flags()&page.FlagAllVisible == 0 {
			t.Fatalf("store %q leaf %d: bits %d without FlagAllVisible", s.name, blk, bits)
		}
		first := page.FirstOffset
		if o.Next != 0 {
			first++
		}
		for off := first; off <= p.MaxOffset(); off++ {
			item, err := p.Item(off)
			mustOK(t, err)
			flags := btree.EntryFlags(binary.LittleEndian.Uint32(item[btree.OffEntryFlags:]))
			xid := page.XID(binary.LittleEndian.Uint64(item[btree.OffEntryXID:]))
			if flags&btree.EntryFrozen != 0 {
				frozen++
			}
			if bits&fsm.VMAllFrozen != 0 && flags&btree.EntryFrozen == 0 {
				t.Fatalf("store %q leaf %d all-frozen with xid %d unfrozen", s.name, blk, xid)
			}
			if xid < below && flags&(btree.EntryFrozen|btree.EntryFreeing) == 0 {
				t.Fatalf("store %q leaf %d: xid %d < %d not frozen (flags %#x)", s.name, blk, xid, below, flags)
			}
		}
	}
	return frozen
}

/**
* writeMix escribe en el store n Put sueltos sobre 40 ids (uno de cada siete a desborde) y lotes deshechos que
* escriben ids «fantasma» (abortados: nunca deben verse) y los anota en model.
* @param t testing.TB, e *Engine, s *Store, n int, model map[string]string
**/
func writeMix(t testing.TB, e *Engine, s *Store, n int, model map[string]string) {
	t.Helper()
	for i := range n {
		id := fmt.Sprintf("id%02d", i%40)
		value := fmt.Sprintf("v%d-%s", i, strings.Repeat("x", i%7*500))
		mustOK(t, s.Put(id, []byte(value)))
		model[id] = value
		if i%25 == 0 {
			tx := e.NewBatch()
			mustOK(t, s.PutIn(tx, fmt.Sprintf("ghost%02d", i%9), []byte(strings.Repeat("g", 3000))))
			mustOK(t, s.PutIn(tx, id, []byte("rolled back")))
			mustOK(t, tx.Rollback())
		}
	}
}

/**
* checkModel compara el store con model (Read de cada id, ningún fantasma, Count).
* @param t testing.TB, s *Store, model map[string]string
**/
func checkModel(t testing.TB, s *Store, model map[string]string) {
	t.Helper()
	for id, want := range model {
		got, ok, err := s.Read(id)
		if err != nil || !ok || string(got) != want {
			t.Fatalf("Read(%q) = %d B, %v, %v; want %d B", id, len(got), ok, err, len(want))
		}
	}
	for i := range 9 {
		if _, ok, err := s.Read(fmt.Sprintf("ghost%02d", i)); err != nil || ok {
			t.Fatalf("an aborted ghost is visible: %v %v", ok, err)
		}
	}
	if s.Count() != int64(len(model)) {
		t.Fatalf("Count = %d, want %d", s.Count(), len(model))
	}
}

// TestFreezeTruncatesStatusFiles: con transacciones que cruzan un archivo de estados, Vacuum congela todo (también
// lo escrito antes de lotes deshechos, cuyas versiones se quitan), sube el frozenXID del store y el checkpoint lo
// guarda en el control y borra el archivo de estados viejo. Las lecturas siguen iguales antes y después de reabrir
// (los xid truncados se leen confirmados, y por debajo solo quedan versiones congeladas).
func TestFreezeTruncatesStatusFiles(t *testing.T) {
	base := crashfs.New()
	first := page.XID(3*xact.XidsPerPage - 100)
	cfg := smallXact(first)
	e := openTest(t, newProc(base), cfg)
	s := mustStore(t, e, "a")
	model := map[string]string{}
	writeMix(t, e, s, 300, model)
	if files := xactFiles(t, base); !slices.Contains(files, xactFileOf(first)) {
		t.Fatalf("status files %v: want %s before the checkpoint", files, xactFileOf(first))
	}
	mustOK(t, e.Vacuum())
	frozen, err := s.tree.FrozenXID()
	mustOK(t, err)
	if frozen <= page.XID(3*xact.XidsPerPage) {
		t.Fatalf("store frozenXID = %d, want > %d", frozen, 3*xact.XidsPerPage)
	}
	mustOK(t, e.Checkpoint())
	ctl, err := e.ctl.Read()
	mustOK(t, err)
	if ctl.FrozenXID != frozen {
		t.Fatalf("control frozenXID = %d, want %d", ctl.FrozenXID, frozen)
	}
	if files := xactFiles(t, base); slices.Contains(files, xactFileOf(first)) {
		t.Fatalf("status files %v: %s not truncated", files, xactFileOf(first))
	}
	if e.status.Oldest() != page.XID(3*xact.XidsPerPage) {
		t.Fatalf("status Oldest = %d", e.status.Oldest())
	}
	t.Logf("frozen %d versions below %d", checkLeaves(t, s, frozen), frozen)
	checkModel(t, s, model)
	mustOK(t, e.Close())

	e = openTest(t, newProc(base), cfg)
	defer e.Close()
	s = mustStore(t, e, "a")
	checkModel(t, s, model)
	checkLeaves(t, s, frozen)
	writeMix(t, e, s, 50, model)
	checkModel(t, s, model)
}

// TestFrozenXIDNotRetained: el frozenXID del motor es el menor de los stores; un store viejo sin limpiar lo frena,
// Empty le da un frozenXID nuevo (el xid vivo más viejo al vaciarlo) y DropStore lo saca: ninguno de los dos retiene
// el truncado. Un store nuevo nace con el xid vivo más viejo, no con 0.
func TestFrozenXIDNotRetained(t *testing.T) {
	e := openTest(t, crashfs.New(), smallXact(1))
	defer e.Close()
	x := mustStore(t, e, "x")
	mustOK(t, x.Put("k", []byte("v")))
	xFrozen, err := x.tree.FrozenXID()
	mustOK(t, err)
	if xFrozen == 0 {
		t.Fatal("a new store was born with frozenXID 0")
	}
	y := mustStore(t, e, "y")
	for i := range 200 {
		mustOK(t, y.Put(fmt.Sprintf("k%03d", i), []byte("v")))
	}
	vacuumOnly := func(s *Store) page.XID {
		t.Helper()
		horizon := e.txs.OldestActive()
		mustOK(t, s.vacuum(horizon))
		return horizon
	}
	h1 := vacuumOnly(y)
	if got := e.engineFrozen(); got != xFrozen {
		t.Fatalf("engine frozenXID = %d, want the old store's %d", got, xFrozen)
	}
	mustOK(t, x.Empty())
	if got := e.engineFrozen(); got != h1 {
		t.Fatalf("after Empty the engine frozenXID = %d, want %d", got, h1)
	}
	z := mustStore(t, e, "z")
	for i := range 100 {
		mustOK(t, y.Put(fmt.Sprintf("k%03d", i), []byte("w")))
	}
	vacuumOnly(y)
	retained := e.engineFrozen()
	mustOK(t, e.DropStore("x"))
	mustOK(t, e.DropStore("z"))
	h3 := vacuumOnly(y)
	if got := e.engineFrozen(); got <= retained || got != h3 {
		t.Fatalf("after DropStore the engine frozenXID = %d, want %d (> %d)", got, h3, retained)
	}
	_ = z
	mustOK(t, e.Checkpoint())
	ctl, err := e.ctl.Read()
	mustOK(t, err)
	if ctl.FrozenXID != h3 {
		t.Fatalf("control frozenXID = %d, want %d", ctl.FrozenXID, h3)
	}
}

// TestStatusFilesBounded: con muchas transacciones (más de tres archivos de estados de prueba) y limpieza y
// checkpoint de vez en cuando, los archivos de estados no se acumulan: quedan a lo sumo tres.
func TestStatusFilesBounded(t *testing.T) {
	base := crashfs.New()
	e := openTest(t, base, smallXact(1))
	defer e.Close()
	s := mustStore(t, e, "a")
	d := s.Deferred()
	most := 0
	for i := range 4 * xact.XidsPerPage {
		mustOK(t, d.Put(fmt.Sprintf("k%03d", i%300), []byte("value")))
		if i%(xact.XidsPerPage/4) == 0 {
			mustOK(t, e.Vacuum())
			mustOK(t, e.Checkpoint())
			most = max(most, len(xactFiles(t, base)))
		}
	}
	if most > 3 {
		t.Fatalf("up to %d status files", most)
	}
	if files := xactFiles(t, base); slices.Contains(files, "0001") || slices.Contains(files, "0002") {
		t.Fatalf("old status files kept: %v", files)
	}
}

// hookFS es un disco de un proceso que avisa antes de cada Remove y de cada WriteAt de un archivo (para cortar en un
// punto exacto).
type hookFS struct {
	*procFS
	onRemove    func(name string)
	afterRemove func(name string)
	onWrite     func(name string)
	hookMu      sync.Mutex
}

func (h *hookFS) Remove(name string) error {
	if h.onRemove != nil {
		h.onRemove(name)
	}
	err := h.procFS.Remove(name)
	if h.afterRemove != nil {
		h.afterRemove(name)
	}
	return err
}

func (h *hookFS) OpenFile(name string, flag int) (vfs.File, error) {
	f, err := h.procFS.OpenFile(name, flag)
	if err != nil {
		return nil, err
	}
	return &hookFile{File: f, name: name, h: h}, nil
}

// hookFile es un archivo de hookFS.
type hookFile struct {
	vfs.File
	name string
	h    *hookFS
}

func (f *hookFile) WriteAt(b []byte, off int64) (int, error) {
	if hook := f.h.writeHook(); hook != nil {
		hook(f.name)
	}
	return f.File.WriteAt(b, off)
}

// writeHook devuelve onWrite (lo fija la goroutine de la prueba antes de la limpieza; las escrituras de fondo lo
// leen después).
func (h *hookFS) writeHook() func(string) {
	h.hookMu.Lock()
	defer h.hookMu.Unlock()
	return h.onWrite
}

// setWriteHook fija onWrite.
func (h *hookFS) setWriteHook(fn func(string)) {
	h.hookMu.Lock()
	defer h.hookMu.Unlock()
	h.onWrite = fn
}

var _ io.WriterAt = (*hookFile)(nil)

// TestFreezeCrashCuts corta el disco en medio de una limpieza que congela (en la escritura n del WAL) y en medio del
// truncado (antes y después de borrar el archivo de estados, tras el paso 6 y tras el 7 del checkpoint), en los tres
// modos de crashfs. Al reabrir, el contenido es el del modelo, las hojas cumplen el congelado y otra limpieza con su
// checkpoint termina el truncado.
func TestFreezeCrashCuts(t *testing.T) {
	modes := map[string]crashfs.Mode{
		"lose": crashfs.LoseUnsynced, "reorder": crashfs.ReorderUnsynced, "torn": crashfs.TornWrites,
	}
	points := []string{"vacuum3", "vacuum20", "vacuum45", "beforeRemove", "afterRemove", "step6", "step7"}
	for name, mode := range modes {
		for _, point := range points {
			t.Run(name+"/"+point, func(t *testing.T) {
				t.Parallel()
				freezeCut(t, mode, point)
			})
		}
	}
}

/**
* freezeCut es un caso de TestFreezeCrashCuts.
* @param t *testing.T, mode crashfs.Mode, point string
**/
func freezeCut(t *testing.T, mode crashfs.Mode, point string) {
	base := crashfs.New()
	first := page.XID(3*xact.XidsPerPage - 100)
	cfg := smallXact(first)
	cfg.wal.Buffer = 0 // cada registro a su escritura: los cortes cuentan las escrituras del WAL
	proc := newProc(base)
	fsys := &hookFS{procFS: proc}
	e := openTest(t, fsys, cfg)
	s := mustStore(t, e, "a")
	model := map[string]string{}
	writeMix(t, e, s, 300, model)
	mustOK(t, e.Checkpoint())

	var killed atomic.Bool
	kill := func() {
		if killed.CompareAndSwap(false, true) {
			proc.kill(mode, int64(len(point)))
		}
	}
	switch point {
	case "vacuum3", "vacuum20", "vacuum45":
		var n int
		fmt.Sscanf(point, "vacuum%d", &n)
		var mu sync.Mutex
		fsys.setWriteHook(func(name string) {
			mu.Lock()
			defer mu.Unlock()
			if strings.Contains(name, "/wal/") {
				if n--; n == 0 {
					kill()
				}
			}
		})
		e.Vacuum()
		if !killed.Load() {
			t.Fatalf("the vacuum wrote fewer WAL blocks than %s", point)
		}
	default:
		mustOK(t, e.Vacuum())
		isXact := func(name string) bool { return strings.Contains(name, "/"+xact.Dir+"/") }
		fsys.onRemove = func(name string) {
			if isXact(name) && point == "beforeRemove" {
				kill()
			}
		}
		fsys.afterRemove = func(name string) { // borrado hecho, sin el fsync del directorio
			if isXact(name) && point == "afterRemove" {
				kill()
			}
		}
		e.afterStep = func(step int) {
			if point == "step6" && step == 6 || point == "step7" && step == 7 {
				kill()
			}
		}
		e.Checkpoint()
		if !killed.Load() {
			t.Fatalf("the checkpoint never reached %s", point)
		}
	}
	closeWithin(t, e, 10*time.Second)

	e = openTest(t, newProc(base), cfg)
	defer e.Close()
	s = mustStore(t, e, "a")
	checkModel(t, s, model)
	verifyTrees(t, e)
	checkLeaves(t, s, e.status.Oldest())
	mustOK(t, e.Vacuum())
	mustOK(t, e.Checkpoint())
	frozen, err := s.tree.FrozenXID()
	mustOK(t, err)
	checkLeaves(t, s, frozen)
	if files := xactFiles(t, base); slices.Contains(files, xactFileOf(first)) {
		t.Fatalf("status files %v: %s not truncated after the next checkpoint", files, xactFileOf(first))
	}
	checkModel(t, s, model)
}
