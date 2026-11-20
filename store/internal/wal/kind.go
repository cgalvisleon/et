package wal

// Kind es el tipo de un registro del WAL.
type Kind uint8

// Del motor (1–15): los formatos están en doc.go.
const (
	KindCheckpoint  Kind = 1
	KindCommit      Kind = 2
	KindAbort       Kind = 3
	KindPageImage   Kind = 4 // reservado: las marcas de visibilidad van como KindSetHints
	KindCreateStore Kind = 5 // alta de un store en el catálogo: nombre y número de archivo (pregunta 56)
	KindDropStore   Kind = 6 // baja de un store: nombre y número de archivo esperado; se saca del catálogo y se borran sus archivos
	KindEmptyStore  Kind = 7 // vaciado: nombre, número de archivo viejo y nuevo; los archivos viejos se borran
)

// De btree (16–31): formato de sus deltas en store/internal/btree.
const (
	KindLeafWrite   Kind = 16 // versión nueva en una hoja (también el cambio lógico)
	KindSplit       Kind = 17 // partición: izquierda, derecha nueva y la antigua hermana derecha (su prev)
	KindInsertPivot Kind = 18 // enlace a la derecha nueva en el padre; quita OpaqueIncompleteSplit a la hija
	KindNewRoot     Kind = 19 // raíz nueva y página meta
	KindInitStore   Kind = 20 // página meta y hoja raíz vacía de un store nuevo (BlockWillInit)
	KindLeafVacuum  Kind = 21 // limpieza: quita versiones muertas de una hoja (fase 1B) y congela las visibles (fase 2)
	KindSetHints    Kind = 22 // marcas de visibilidad: la primera marca tras un checkpoint, con la imagen de la hoja
	KindSetFrozen   Kind = 23 // frozenXID nuevo en la página meta de un store (fase 2, congelado)
)

// De overflow (32–39).
const (
	KindOverflowWrite Kind = 32 // página de desborde nueva con su trozo
)

// De fsm (40–47).
const (
	KindFreePush Kind = 40 // una página pasa a la lista de libres (meta + la página)
	KindFreePop  Kind = 41 // una página sale de la lista de libres (meta + la página)
	KindVMSet    Kind = 42 // marca bits del mapa de visibilidad (fase 1B/2)
	KindVMClear  Kind = 43 // borra bits del mapa de visibilidad
)

// Flags de un registro.
type Flags uint8

const (
	FlagCommit   Flags = 1 << 0 // el registro confirma su transacción (operación suelta)
	FlagCountAdd Flags = 1 << 1 // la escritura agregó un id vivo (había lápida o nada)
	FlagCountSub Flags = 1 << 2 // la escritura quitó un id vivo (lápida sobre una versión viva)
)

// BlockFlags son las marcas de una referencia a bloque.
type BlockFlags uint8

const (
	BlockHasImage        BlockFlags = 1 << 0 // lleva la imagen de la página antes del cambio
	BlockImageCompressed BlockFlags = 1 << 1 // la imagen está comprimida con S2
	BlockWillInit        BlockFlags = 1 << 2 // el delta inicia la página entera: al rehacer no se lee del disco
)
