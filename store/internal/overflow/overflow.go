package overflow

import (
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// ChunkSize es lo que guarda de un valor cada página de desborde.
const ChunkSize = page.Size - page.HeaderSize - page.OpaqueSize

// Ref apunta a una cadena de desborde: su primer bloque y el largo total del valor.
type Ref struct {
	First page.BlockNumber
	Len   uint32
}

// Chains escribe, lee y libera cadenas de desborde.
type Chains interface {
	// Write guarda data en una cadena nueva del store file para la transacción xid y devuelve su Ref. Las páginas
	// quedan en la caché y en el WAL; no espera el fsync.
	Write(file smgr.FileNo, xid page.XID, data []byte) (Ref, error)

	// Read devuelve una copia del valor completo, verificando que la cadena tenga ref.Len bytes.
	Read(file smgr.FileNo, ref Ref) ([]byte, error)

	// Free devuelve todas las páginas de la cadena a la lista de libres (limpieza).
	Free(file smgr.FileNo, ref Ref, xid page.XID) error
}
