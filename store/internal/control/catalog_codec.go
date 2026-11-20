package control

import (
	"fmt"
	"hash/crc32"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// Desplazamientos del encabezado y de cada entrada del catálogo (tabla en doc.go).
const (
	catOffMagic      = 0
	catOffVersion    = 4
	catOffNextFileNo = 8
	catOffCount      = 12
	catOffCountLSN   = 16
	catOffReserved   = 24

	entOffFileNo    = 0
	entOffMetaBlock = 4
	entOffCount     = 8
	entOffFlags     = 16
	entOffNameLen   = 20
	entOffReserved  = 22

	crcSize = 4

	// minEntrySize es lo menos que ocupa una entrada: sin nombre más un byte de nombre.
	minEntrySize = CatalogEntrySize + 1
)

// checkName valida el nombre de un store: texto UTF-8 válido, sin el carácter nulo, de 1 a MaxNameLen bytes
// (pregunta 57: así siempre se puede mostrar).
// name: el nombre. Devuelve ErrBadName o nil.
func checkName(name string) error {
	if len(name) == 0 || len(name) > MaxNameLen {
		return fmt.Errorf("%w: %d bytes", ErrBadName, len(name))
	}
	if !utf8.ValidString(name) {
		return fmt.Errorf("%w: no es UTF-8 válido", ErrBadName)
	}
	if strings.IndexByte(name, 0) >= 0 {
		return fmt.Errorf("%w: contiene el carácter nulo", ErrBadName)
	}
	return nil
}

// checkCatalog valida las reglas del catálogo: nombres válidos y únicos, números de archivo distintos de 0, únicos y
// menores que NextFileNo, contadores no negativos y menos de 2^32 entradas.
// c: el catálogo. Devuelve ErrBadCatalog con el detalle, o nil.
func checkCatalog(c Catalog) error {
	if c.NextFileNo == 0 {
		return fmt.Errorf("%w: next file number is 0", ErrBadCatalog)
	}
	if uint64(len(c.Entries)) > math.MaxUint32 {
		return fmt.Errorf("%w: too many entries", ErrBadCatalog)
	}
	names := make(map[string]struct{}, len(c.Entries))
	files := make(map[smgr.FileNo]struct{}, len(c.Entries))
	for i, e := range c.Entries {
		if err := checkName(e.Name); err != nil {
			return fmt.Errorf("%w: entry %d: %v", ErrBadCatalog, i, err)
		}
		if e.File == 0 || e.File >= c.NextFileNo {
			return fmt.Errorf("%w: entry %q: file number %d outside 1..%d", ErrBadCatalog, e.Name, e.File, c.NextFileNo-1)
		}
		if e.Count < 0 {
			return fmt.Errorf("%w: entry %q: negative count %d", ErrBadCatalog, e.Name, e.Count)
		}
		if _, dup := names[e.Name]; dup {
			return fmt.Errorf("%w: duplicated name %q", ErrBadCatalog, e.Name)
		}
		if _, dup := files[e.File]; dup {
			return fmt.Errorf("%w: duplicated file number %d", ErrBadCatalog, e.File)
		}
		names[e.Name] = struct{}{}
		files[e.File] = struct{}{}
	}
	return nil
}

// encodeCatalog arma el archivo del catálogo con su CRC, con las entradas en el orden dado.
// c: el catálogo. Devuelve los bytes o ErrBadCatalog si no cumple las reglas.
func encodeCatalog(c Catalog) ([]byte, error) {
	if err := checkCatalog(c); err != nil {
		return nil, err
	}
	size := CatalogHeaderSize + crcSize
	for _, e := range c.Entries {
		size += CatalogEntrySize + len(e.Name)
	}
	b := make([]byte, size)
	le.PutUint32(b[catOffMagic:], CatalogMagic)
	le.PutUint32(b[catOffVersion:], FormatVersion)
	le.PutUint32(b[catOffNextFileNo:], uint32(c.NextFileNo))
	le.PutUint32(b[catOffCount:], uint32(len(c.Entries)))
	le.PutUint64(b[catOffCountLSN:], uint64(c.CountLSN))
	off := CatalogHeaderSize
	for _, e := range c.Entries {
		le.PutUint32(b[off+entOffFileNo:], uint32(e.File))
		le.PutUint32(b[off+entOffMetaBlock:], uint32(e.MetaBlock))
		le.PutUint64(b[off+entOffCount:], uint64(e.Count))
		le.PutUint32(b[off+entOffFlags:], e.Flags)
		le.PutUint16(b[off+entOffNameLen:], uint16(len(e.Name)))
		off += CatalogEntrySize
		off += copy(b[off:], e.Name)
	}
	le.PutUint32(b[off:], crc32.Checksum(b[:off], castagnoli))
	return b, nil
}

// decodeCatalog lee el catálogo sin confiar en sus bytes: comprueba CRC, magia, versión, ceros reservados, que cada
// entrada quepa antes de leerla, que no sobren bytes y las reglas de checkCatalog. Lo que acepta, encodeCatalog lo
// vuelve a escribir igual byte a byte.
// b: el archivo entero. Devuelve el catálogo o ErrBadCatalog con el detalle.
func decodeCatalog(b []byte) (Catalog, error) {
	if len(b) < CatalogHeaderSize+crcSize {
		return Catalog{}, fmt.Errorf("%w: length %d is less than the header", ErrBadCatalog, len(b))
	}
	body := b[:len(b)-crcSize]
	if got, want := le.Uint32(b[len(body):]), crc32.Checksum(body, castagnoli); got != want {
		return Catalog{}, fmt.Errorf("%w: crc %08x, want %08x", ErrBadCatalog, got, want)
	}
	if m := le.Uint32(b[catOffMagic:]); m != CatalogMagic {
		return Catalog{}, fmt.Errorf("%w: magic %08x", ErrBadCatalog, m)
	}
	if v := le.Uint32(b[catOffVersion:]); v != FormatVersion {
		return Catalog{}, fmt.Errorf("%w: %w: %d, want %d", ErrBadCatalog, ErrFormatVersion, v, FormatVersion)
	}
	if !allZero(b[catOffReserved:CatalogHeaderSize]) {
		return Catalog{}, fmt.Errorf("%w: reserved header bytes are not zero", ErrBadCatalog)
	}

	// Antes de reservar memoria: entryCount no puede pasar de lo que entra en el archivo.
	count := int(le.Uint32(b[catOffCount:]))
	if count > (len(body)-CatalogHeaderSize)/minEntrySize {
		return Catalog{}, fmt.Errorf("%w: %d entries do not fit in %d bytes", ErrBadCatalog, count, len(b))
	}
	c := Catalog{
		NextFileNo: smgr.FileNo(le.Uint32(b[catOffNextFileNo:])),
		CountLSN:   page.LSN(le.Uint64(b[catOffCountLSN:])),
		Entries:    make([]Entry, 0, count),
	}
	off := CatalogHeaderSize
	for i := 0; i < count; i++ {
		e, next, err := decodeEntry(body, off)
		if err != nil {
			return Catalog{}, fmt.Errorf("%w: entry %d: %v", ErrBadCatalog, i, err)
		}
		c.Entries = append(c.Entries, e)
		off = next
	}
	if off != len(body) {
		return Catalog{}, fmt.Errorf("%w: %d bytes left after the entries", ErrBadCatalog, len(body)-off)
	}
	if err := checkCatalog(c); err != nil {
		return Catalog{}, err
	}
	return c, nil
}

// decodeEntry lee una entrada que empieza en off, comprobando que quepa en body.
// body: el catálogo sin el crc; off: dónde empieza. Devuelve la entrada, dónde empieza la siguiente o un error.
func decodeEntry(body []byte, off int) (Entry, int, error) {
	if len(body)-off < CatalogEntrySize {
		return Entry{}, 0, fmt.Errorf("truncated at byte %d", off)
	}
	h := body[off : off+CatalogEntrySize]
	if !allZero(h[entOffReserved:CatalogEntrySize]) {
		return Entry{}, 0, fmt.Errorf("reserved bytes are not zero")
	}
	n := int(le.Uint16(h[entOffNameLen:]))
	if n == 0 || n > MaxNameLen {
		return Entry{}, 0, fmt.Errorf("name length %d outside 1..%d", n, MaxNameLen)
	}
	start := off + CatalogEntrySize
	if len(body)-start < n {
		return Entry{}, 0, fmt.Errorf("name truncated at byte %d", start)
	}
	e := Entry{
		Name:      string(body[start : start+n]),
		File:      smgr.FileNo(le.Uint32(h[entOffFileNo:])),
		MetaBlock: page.BlockNumber(le.Uint32(h[entOffMetaBlock:])),
		Count:     int64(le.Uint64(h[entOffCount:])),
		Flags:     le.Uint32(h[entOffFlags:]),
	}
	return e, start + n, nil
}

// EncodeCatalog es encodeCatalog para las pruebas de ida y vuelta del formato (store/format_test.go).
// c: el catálogo. Devuelve el archivo entero o ErrBadCatalog.
func EncodeCatalog(c Catalog) ([]byte, error) {
	return encodeCatalog(c)
}

// DecodeCatalog es decodeCatalog para las pruebas de ida y vuelta del formato: lo que acepta, EncodeCatalog lo
// vuelve a escribir igual byte a byte.
// b: el archivo. Devuelve el catálogo, o ErrBadCatalog (ErrFormatVersion si es de otra versión).
func DecodeCatalog(b []byte) (Catalog, error) {
	return decodeCatalog(b)
}
