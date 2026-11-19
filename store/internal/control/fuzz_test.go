package control

import (
	"bytes"
	"hash/crc32"
	"testing"
)

// crcOf es el CRC-32C de b, para armar entradas válidas a mano.
func crcOf(b []byte) uint32 { return crc32.Checksum(b, castagnoli) }

// FuzzDecodeControl: decodeControl nunca entra en pánico con bytes cualquiera, y lo que acepta se vuelve a
// codificar igual byte a byte (no hay dos formas de escribir el mismo control).
// go test ./store/internal/control -run '^$' -fuzz FuzzDecodeControl -fuzztime 1m
func FuzzDecodeControl(f *testing.F) {
	for _, seq := range []uint64{1, 2, 1 << 40} {
		b, err := encodeControl(withSeq(sampleControl(int64(seq)), seq))
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Add(make([]byte, CopySize))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) {
		checkControlBytes(t, b)
		checkControlBytes(t, withCRC(b)) // con el CRC bien, llega a la validación de los campos
	})
}

// checkControlBytes decodifica b y, si lo acepta, exige que se vuelva a codificar igual.
func checkControlBytes(t *testing.T, b []byte) {
	c, err := decodeControl(b)
	if err != nil {
		return
	}
	again, err := encodeControl(c)
	if err != nil {
		t.Fatalf("accepted control does not encode: %v", err)
	}
	if !bytes.Equal(again, b) {
		t.Fatal("re-encoded control differs")
	}
}

// FuzzDecodeCatalog: decodeCatalog nunca entra en pánico ni reserva por un entryCount mentiroso, lo que acepta se
// vuelve a codificar igual y LoadStores lo acepta.
// go test ./store/internal/control -run '^$' -fuzz FuzzDecodeCatalog -fuzztime 1m
func FuzzDecodeCatalog(f *testing.F) {
	for _, c := range []Catalog{sampleCatalog(), {NextFileNo: 1}} {
		b, err := encodeCatalog(c)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) {
		checkCatalogBytes(t, b)
		checkCatalogBytes(t, withCRC(b))
	})
}

// checkCatalogBytes decodifica b y, si lo acepta, exige que se vuelva a codificar igual y que LoadStores lo acepte.
func checkCatalogBytes(t *testing.T, b []byte) {
	c, err := decodeCatalog(b)
	if err != nil {
		return
	}
	again, err := encodeCatalog(c)
	if err != nil {
		t.Fatalf("accepted catalog does not encode: %v", err)
	}
	if !bytes.Equal(again, b) {
		t.Fatal("re-encoded catalog differs")
	}
	if _, err := LoadStores(c); err != nil {
		t.Fatalf("LoadStores rejects an accepted catalog: %v", err)
	}
}

// withCRC devuelve una copia de b con sus últimos 4 bytes reemplazados por el CRC-32C de lo anterior.
func withCRC(b []byte) []byte {
	if len(b) < crcSize {
		return b
	}
	c := append([]byte(nil), b...)
	n := len(c) - crcSize
	le.PutUint32(c[n:], crcOf(c[:n]))
	return c
}
