package btree

import (
	"errors"
	"sync/atomic"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/fsm"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// rootBlock es la hoja raíz de un store nuevo.
const rootBlock page.BlockNumber = 1

// StatusReader es lo que el árbol lee del archivo de estados (xact.StatusFile lo cumple).
type StatusReader interface {
	Status(xid page.XID) (xact.Status, page.LSN, error)
}

// Config es lo que necesita el árbol de un store.
type Config struct {
	File smgr.FileNo  // el store
	Pool buffer.Pool  // la caché del motor
	Smgr smgr.Manager // los archivos de datos (Create)
	Free fsm.FreeList // de donde salen las páginas nuevas
	Log  wal.Log      // el WAL del motor

	// Status es el archivo de estados; nil = toda versión cuenta como confirmada (fase 1A, solo operaciones
	// sueltas que nacen confirmadas; a confirmar con el motor de la ola 4).
	Status StatusReader
}

// BTree es el árbol de un store: implementa Tree. No guarda nada del árbol en memoria salvo la pista de la hoja de más
// a la derecha (regla 1). Es seguro usarlo desde varias goroutines.
type BTree struct {
	file   smgr.FileNo
	pool   buffer.Pool
	sm     smgr.Manager
	free   fsm.FreeList
	log    wal.Log
	status StatusReader

	// fastLeaf es la pista de la inserción rápida: la última hoja de más a la derecha donde se escribió (0 = ninguna).
	// Solo es una pista: se comprueba con la hoja bloqueada antes de usarla.
	fastLeaf atomic.Uint32

	// afterSplit, si no es nil, corre entre los dos pasos de una partición; si devuelve error, la partición queda
	// incompleta (solo para las pruebas: simula un corte entre KindSplit y KindInsertPivot).
	afterSplit func(left page.BlockNumber) error
}

// Comprobación en compilación de que BTree cumple Tree.
var _ Tree = (*BTree)(nil)

/**
* Open abre el árbol de un store ya creado. No lee nada: la meta se lee en cada bajada.
* @param cfg Config
* @return *BTree, error: ErrConfig si falta Pool, Smgr, Free o Log, o File es 0
**/
func Open(cfg Config) (*BTree, error) {
	if cfg.File == 0 || cfg.Pool == nil || cfg.Smgr == nil || cfg.Free == nil || cfg.Log == nil {
		return nil, ErrConfig
	}
	return &BTree{file: cfg.File, pool: cfg.Pool, sm: cfg.Smgr, free: cfg.Free, log: cfg.Log, status: cfg.Status}, nil
}

/**
* Create crea el store: su fork principal (si falta) y, en un KindInitStore, la página meta (bloque 0) y una hoja raíz
* vacía (bloque 1), las dos con BlockWillInit. Quien llama garantiza que nadie más usa el store mientras tanto.
* @param cfg Config
* @return *BTree, error: ErrExists si el store ya tiene páginas
**/
func Create(cfg Config) (*BTree, error) {
	t, err := Open(cfg)
	if err != nil {
		return nil, err
	}
	n, err := t.sm.NBlocks(t.file, smgr.ForkMain)
	if errors.Is(err, smgr.ErrNoFork) {
		n, err = 0, t.sm.Create(t.file, smgr.ForkMain)
	}
	if err != nil {
		return nil, err
	}
	if n != 0 {
		return nil, ErrExists
	}
	metaH, err := t.pool.PinNew(t.file, smgr.ForkMain)
	if err != nil {
		return nil, err
	}
	rootH, err := t.pool.PinNew(t.file, smgr.ForkMain)
	if err != nil {
		metaH.Release()
		return nil, err
	}
	if metaH.Tag().Block != MetaBlock || rootH.Tag().Block != rootBlock {
		metaH.Release()
		rootH.Release()
		return nil, corrupt("new store got blocks %d and %d", metaH.Tag().Block, rootH.Tag().Block)
	}
	refs := []wal.BlockRef{
		{Flags: wal.BlockWillInit, Delta: blockDelta(rootBlock)},
		{Flags: wal.BlockWillInit},
	}
	res, err := t.logChange(wal.KindInitStore, 0, 0, refs, []buffer.Handle{metaH, rootH}, nil)
	metaH.Release()
	rootH.Release()
	if err != nil {
		return nil, err
	}
	if err := res.Write(); err != nil {
		return nil, err
	}
	return t, nil
}

/**
* tag es el tag de un bloque del fork principal del store.
* @param blk page.BlockNumber
* @return buffer.Tag
**/
func (t *BTree) tag(blk page.BlockNumber) buffer.Tag {
	return buffer.Tag{File: t.file, Fork: smgr.ForkMain, Block: blk}
}

/**
* lockNode fija una página del árbol, la bloquea y la valida.
* @param blk page.BlockNumber, mode buffer.LockMode
* @return buffer.Handle, node, error: con error no queda nada fijado
**/
func (t *BTree) lockNode(blk page.BlockNumber, mode buffer.LockMode) (buffer.Handle, node, error) {
	if blk == MetaBlock {
		return nil, node{}, corruptAt(blk, "link to the meta page")
	}
	h, err := t.pool.Pin(t.tag(blk), buffer.Normal)
	if err != nil {
		return nil, node{}, err
	}
	h.Lock(mode)
	n, err := readNode(blk, h.Page())
	if err != nil {
		h.Release()
		return nil, node{}, err
	}
	return h, n, nil
}

/**
* readMeta lee la raíz de la página meta con bloqueo compartido y la suelta.
* @return meta, error
**/
func (t *BTree) readMeta() (meta, error) {
	h, err := t.pool.Pin(t.tag(MetaBlock), buffer.Normal)
	if err != nil {
		return meta{}, err
	}
	defer h.Release()
	h.Lock(buffer.Share)
	return readMetaPage(h.Page())
}

/**
* releaseAll suelta varias páginas (las nil se saltan).
* @param hs ...buffer.Handle
**/
func releaseAll(hs ...buffer.Handle) {
	for _, h := range hs {
		if h != nil {
			h.Release()
		}
	}
}
