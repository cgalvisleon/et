package page

// Lo que comparten btree (dueño del formato de la página meta, btree/doc.go) y fsm (que lee y cambia la cabeza de la
// lista de páginas libres): un solo lugar para que no diverjan. fsm no puede importar btree (btree depende de fsm).
const (
	// MetaBlock es el bloque de la página meta de un store.
	MetaBlock BlockNumber = 0
	// MetaMagic identifica la página meta de un store ("ETBT").
	MetaMagic = 0x45544254
	// MetaOffMagic es la posición de la magia en la página meta.
	MetaOffMagic = HeaderSize + 0
	// MetaOffFreeHead es la posición de la cabeza de la lista de páginas libres en la página meta.
	MetaOffFreeHead = HeaderSize + 24
)
