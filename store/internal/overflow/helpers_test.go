package overflow

import (
	"math/rand"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/fsm"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// testDir es el directorio del motor de prueba.
const testDir = "/db"

// testFile es el store de las pruebas.
const testFile smgr.FileNo = 1

// testOptions: segmentos de 1 MB, el mínimo.
var testOptions = wal.Options{SegmentSize: wal.MinSegmentSize, Spares: 1}

// Campos de la página meta que usan las pruebas (formato de btree; los mismos que repite fsm).
const (
	metaMagic   = 0x45544254 // "ETBT"
	offMagic    = page.HeaderSize + 0
	offFreeHead = page.HeaderSize + 24
)

// env es un motor de prueba abierto con las piezas reales: archivos de datos, WAL, caché, fsm y overflow.
type env struct {
	fs   crashfs.FS
	sm   smgr.Manager
	log  *wal.WAL
	pool buffer.Pool
	free *fsm.Manager
	ov   *Manager
}

// lazyFlusher es el WAL que ve la caché: mientras se rehace dentro de wal.Open todavía no existe (lo rehecho ya
// está en el WAL del disco), después es el WAL abierto.
type lazyFlusher struct {
	log atomic.Pointer[wal.WAL]
}

func (f *lazyFlusher) Durable() page.LSN {
	if log := f.log.Load(); log != nil {
		return log.Durable()
	}
	return ^page.LSN(0)
}

func (f *lazyFlusher) WaitDurable(lsn page.LSN) error {
	if log := f.log.Load(); log != nil {
		return log.WaitDurable(lsn)
	}
	return nil
}

/**
* redoAll reparte un registro entre las funciones de rehacer de overflow y fsm, como lo hará el motor.
* @param pool buffer.Pool, sm smgr.Manager, rec *wal.Record
* @return error
**/
func redoAll(pool buffer.Pool, sm smgr.Manager, rec *wal.Record) error {
	switch rec.Kind {
	case wal.KindOverflowWrite:
		return Redo(pool, sm, rec)
	case wal.KindFreePush, wal.KindFreePop, wal.KindVMSet, wal.KindVMClear:
		return fsm.Redo(pool, sm, rec)
	}
	return nil
}

/**
* openEnv abre el motor de prueba sobre fsys con una caché de buffers páginas, rehaciendo el WAL desde wal.Start.
* @param t testing.TB, fsys crashfs.FS, buffers int
* @return *env
**/
func openEnv(t testing.TB, fsys crashfs.FS, buffers int) *env {
	t.Helper()
	sm, err := smgr.Open(fsys, testDir, 16)
	if err != nil {
		t.Fatal(err)
	}
	fl := &lazyFlusher{}
	pool, err := buffer.Open(buffer.Config{Smgr: sm, Flusher: fl, Buffers: buffers})
	if err != nil {
		t.Fatal(err)
	}
	log, _, err := wal.Open(fsys, testDir, testOptions, wal.Start, func(rec *wal.Record) error {
		return redoAll(pool, sm, rec)
	})
	if err != nil {
		t.Fatal(err)
	}
	fl.log.Store(log)
	t.Cleanup(func() {
		pool.Close()
		log.Close()
		sm.Close()
	})
	free := fsm.New(pool, sm, log)
	return &env{fs: fsys, sm: sm, log: log, pool: pool, free: free, ov: New(pool, free, log)}
}

// crash detiene la caché (su escritor de fondo no escribe más) y corta el disco.
func (e *env) crash(mode crashfs.Mode, seed int64) {
	e.pool.Close()
	e.fs.Crash(mode, seed)
}

/**
* createStore crea el store file con su página meta (lista de libres vacía) ya durable, como la dejaría btree.
* @param t testing.TB, sm smgr.Manager, file smgr.FileNo
**/
func createStore(t testing.TB, sm smgr.Manager, file smgr.FileNo) {
	t.Helper()
	meta := make(page.Page, page.Size)
	if err := meta.Init(page.OpaqueSize); err != nil {
		t.Fatal(err)
	}
	if err := meta.SetOpaque(page.Opaque{Flags: page.OpaqueMeta}); err != nil {
		t.Fatal(err)
	}
	le.PutUint32(meta[offMagic:], metaMagic)
	le.PutUint32(meta[offMagic+4:], 1)
	if err := meta.SetLower(page.HeaderSize + 48); err != nil {
		t.Fatal(err)
	}
	meta.SetChecksum(0)
	if err := sm.Create(file, smgr.ForkMain); err != nil {
		t.Fatal(err)
	}
	if err := sm.Extend(file, smgr.ForkMain, 0, meta); err != nil {
		t.Fatal(err)
	}
	if err := sm.SyncAll(); err != nil {
		t.Fatal(err)
	}
}

/**
* newEnv abre un motor nuevo en memoria con el store de prueba creado.
* @param t testing.TB, buffers int
* @return *env
**/
func newEnv(t testing.TB, buffers int) *env {
	t.Helper()
	e := openEnv(t, crashfs.New(), buffers)
	createStore(t, e.sm, testFile)
	return e
}

/**
* readPage devuelve una copia de la página block del store de prueba.
* @param t testing.TB, pool buffer.Pool, block page.BlockNumber
* @return page.Page
**/
func readPage(t testing.TB, pool buffer.Pool, block page.BlockNumber) page.Page {
	t.Helper()
	h, err := pool.Pin(buffer.Tag{File: testFile, Fork: smgr.ForkMain, Block: block}, buffer.Normal)
	if err != nil {
		t.Fatalf("Pin(%d): %v", block, err)
	}
	h.Lock(buffer.Share)
	defer release(h)
	return append(page.Page(nil), h.Page()...)
}

/**
* writePage reemplaza la página block del store de prueba por p (para dañar cadenas a mano).
* @param t testing.TB, pool buffer.Pool, block page.BlockNumber, p page.Page
**/
func writePage(t testing.TB, pool buffer.Pool, block page.BlockNumber, p page.Page) {
	t.Helper()
	h, err := pool.Pin(buffer.Tag{File: testFile, Fork: smgr.ForkMain, Block: block}, buffer.Normal)
	if err != nil {
		t.Fatalf("Pin(%d): %v", block, err)
	}
	h.Lock(buffer.Exclusive)
	defer release(h)
	copy(h.Page(), p)
	h.MarkDirty(p.LSN())
}

/**
* freeList recorre la lista de libres del store de prueba desde la meta y devuelve sus bloques en orden. Falla si
* tiene un ciclo o una página que no es libre.
* @param t testing.TB, pool buffer.Pool
* @return []page.BlockNumber
**/
func freeList(t testing.TB, pool buffer.Pool) []page.BlockNumber {
	t.Helper()
	var list []page.BlockNumber
	for b := page.BlockNumber(le.Uint32(readPage(t, pool, 0)[offFreeHead:])); b != 0; {
		if slices.Contains(list, b) {
			t.Fatalf("ciclo en la lista de libres: %v y otra vez %d", list, b)
		}
		list = append(list, b)
		o, err := readPage(t, pool, b).Opaque()
		if err != nil || o.Flags != page.OpaqueFree {
			t.Fatalf("bloque %d de la lista no es libre: %+v, %v", b, o, err)
		}
		b = o.Next
	}
	return list
}

/**
* chainBlocks devuelve los bloques de la cadena ref en orden, siguiendo next sin validar.
* @param t testing.TB, pool buffer.Pool, ref Ref
* @return []page.BlockNumber
**/
func chainBlocks(t testing.TB, pool buffer.Pool, ref Ref) []page.BlockNumber {
	t.Helper()
	blocks := []page.BlockNumber{}
	for b, i := ref.First, 0; b != 0 && i < pagesOf(ref.Len); i++ {
		blocks = append(blocks, b)
		o, err := readPage(t, pool, b).Opaque()
		if err != nil {
			t.Fatalf("bloque %d: %v", b, err)
		}
		b = o.Next
	}
	return blocks
}

// randomValue arma un valor de n bytes al azar.
func randomValue(rng *rand.Rand, n int) []byte {
	v := make([]byte, n)
	rng.Read(v)
	return v
}
