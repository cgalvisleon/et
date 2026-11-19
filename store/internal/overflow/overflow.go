package overflow

import (
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

const (
	// ChunkSize es lo que guarda de un valor cada página de desborde.
	ChunkSize = page.Size - page.HeaderSize - page.OpaqueSize

	// MaxValueLen es el largo máximo de un valor (64 MiB, ver «Límites» en store/doc.go).
	MaxValueLen = 64 << 20
)

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

	// Free devuelve todas las páginas de la cadena a la lista de libres (una cadena que nadie apunta).
	Free(file smgr.FileNo, ref Ref, xid page.XID) error

	// FreeTail libera lo que queda de la cadena de una versión de owner, de la última página a la primera; se puede
	// repetir tras un corte (limpieza, pregunta 58). Devuelve cuántas páginas liberó.
	FreeTail(file smgr.FileNo, ref Ref, owner, xid page.XID) (int, error)
}

/**
* pagesOf es cuántas páginas ocupa un valor de n bytes.
* @param n uint32: largo del valor, mayor que 0
* @return int
**/
func pagesOf(n uint32) int {
	return int((uint64(n) + ChunkSize - 1) / ChunkSize)
}

/**
* chunkLen es el largo del trozo que guarda la página seq de la cadena de un valor de total bytes: ChunkSize en
* todas salvo la última, que guarda el resto.
* @param seq int: posición de la página en la cadena, desde 0
* @param total uint32: largo del valor
* @return int
**/
func chunkLen(seq int, total uint32) int {
	return min(int(total)-seq*ChunkSize, ChunkSize)
}
