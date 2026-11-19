package fsm

import (
	"slices"
	"testing"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// testDir es el directorio del motor de prueba.
const testDir = "/db"

// testOptions: segmentos de 1 MB, el mínimo.
var testOptions = wal.Options{SegmentSize: wal.MinSegmentSize, Spares: 1}

// env es un motor de prueba abierto: archivos de datos, WAL, caché y fsm.
type env struct {
	fs   vfs.FS
	sm   smgr.Manager
	log  *wal.WAL
	pool *testPool
	fsm  *Manager
}

/**
* openEnv abre el motor de prueba sobre fsys: el WAL desde wal.Start, rehaciendo con Redo los registros de fsm que
* encuentre (ninguno en un disco nuevo).
* @param t testing.TB, fsys vfs.FS
* @return *env
**/
func openEnv(t testing.TB, fsys vfs.FS) *env {
	t.Helper()
	sm, err := smgr.Open(fsys, testDir, 16)
	if err != nil {
		t.Fatal(err)
	}
	pool := newTestPool(sm, nil)
	log, _, err := wal.Open(fsys, testDir, testOptions, wal.Start, func(rec *wal.Record) error {
		return Redo(pool, sm, rec)
	})
	if err != nil {
		t.Fatal(err)
	}
	pool.fl = log
	t.Cleanup(func() {
		log.Close()
		sm.Close()
	})
	return &env{fs: fsys, sm: sm, log: log, pool: pool, fsm: New(pool, sm, log)}
}

/**
* createStore crea el store file con su página meta (lista vacía) ya durable, como lo dejaría KindInitStore de btree
* tras un checkpoint.
* @param t testing.TB, sm smgr.Manager, file smgr.FileNo
**/
func createStore(t testing.TB, sm smgr.Manager, file smgr.FileNo) {
	t.Helper()
	meta := make(page.Page, page.Size)
	if err := initPage(meta, page.Opaque{Flags: page.OpaqueMeta}); err != nil {
		t.Fatal(err)
	}
	le.PutUint32(meta[offMagic:], metaMagic)
	le.PutUint32(meta[offMagic+4:], 1)
	if err := meta.SetLower(page.HeaderSize + 48); err != nil {
		t.Fatal(err)
	}
	meta.SetChecksum(metaBlock)
	if err := sm.Create(file, smgr.ForkMain); err != nil {
		t.Fatal(err)
	}
	if err := sm.Extend(file, smgr.ForkMain, metaBlock, meta); err != nil {
		t.Fatal(err)
	}
	if err := sm.SyncAll(); err != nil {
		t.Fatal(err)
	}
}

/**
* freeList recorre la lista de libres de file desde la meta y devuelve sus bloques en orden. Falla si una página de
* la lista no tiene page.OpaqueFree o si la lista tiene un ciclo.
* @param t testing.TB, pool buffer.Pool, file smgr.FileNo
* @return []page.BlockNumber
**/
func freeList(t testing.TB, pool buffer.Pool, file smgr.FileNo) []page.BlockNumber {
	t.Helper()
	read := func(block page.BlockNumber) page.Page {
		h, err := pool.Pin(buffer.Tag{File: file, Fork: smgr.ForkMain, Block: block}, buffer.Normal)
		if err != nil {
			t.Fatalf("Pin(%d): %v", block, err)
		}
		h.Lock(buffer.Share)
		defer h.Unlock()
		defer h.Release()
		return append(page.Page(nil), h.Page()...)
	}
	var list []page.BlockNumber
	for b := freeHead(read(metaBlock)); b != 0; {
		if slices.Contains(list, b) {
			t.Fatalf("ciclo en la lista de libres: %v y otra vez %d", list, b)
		}
		list = append(list, b)
		o, err := read(b).Opaque()
		if err != nil || o.Flags&page.OpaqueFree == 0 {
			t.Fatalf("bloque %d de la lista no es libre: %+v, %v", b, o, err)
		}
		b = o.Next
	}
	return list
}

// state es lo que fsm guarda de un store: la lista de libres y los bits del mapa de unos bloques.
type state struct {
	list []page.BlockNumber
	vm   []VMBits
}

/**
* stateOf lee el estado de file: la lista y los bits de probes.
* @param t testing.TB, e *env, file smgr.FileNo, probes []page.BlockNumber
* @return state
**/
func stateOf(t testing.TB, e *env, file smgr.FileNo, probes []page.BlockNumber) state {
	t.Helper()
	s := state{list: freeList(t, e.pool, file)}
	for _, b := range probes {
		bits, err := e.fsm.Get(file, b)
		if err != nil {
			t.Fatalf("Get(%d): %v", b, err)
		}
		s.vm = append(s.vm, bits)
	}
	return s
}

// equal compara dos estados.
func (s state) equal(o state) bool {
	return slices.Equal(s.list, o.list) && slices.Equal(s.vm, o.vm)
}

// allocate pide una página y la suelta enseguida (quien la pide la iniciaría con su propio registro).
func allocate(t testing.TB, f *Manager, file smgr.FileNo) page.BlockNumber {
	t.Helper()
	h, err := f.Allocate(file, 7)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	block := h.Tag().Block
	release(h)
	return block
}

// free devuelve block a la lista.
func free(t testing.TB, f *Manager, file smgr.FileNo, block page.BlockNumber) {
	t.Helper()
	if err := f.Free(file, block, 7); err != nil {
		t.Fatalf("Free(%d): %v", block, err)
	}
}
