package store

import (
	"fmt"
	"strings"
	"sync"

	"github.com/klauspost/compress/s2"
	"github.com/klauspost/compress/zstd"

	"github.com/cgalvisleon/et/envar"
)

// Compresión de los valores grandes (formato 2, tarea 132 de F-23; ver «Formato 2» en doc.go). El motor comprime el
// valor antes de decidir si va en línea o a desborde, marca la versión con btree.EntryCompressed y guarda el valor
// comprimido con su encabezado: method(1) rawLen(4) y los datos del método. Quien lee recibe siempre el valor sin
// comprimir.

const (
	// compressNone: sin compresión (STORE_COMPRESSION=off).
	compressNone byte = 0

	// compressZstd: un cuadro zstd sin suma propia (las páginas tienen la suya), nivel más rápido.
	compressZstd byte = 1

	// compressS2: un bloque S2 (formato de bloque de Snappy extendido).
	compressS2 byte = 2

	// compressAuto: política de escritura (no es un método en disco): S2 para los valores de menos de
	// autoZstdMin bytes y zstd para los demás.
	compressAuto byte = 255

	// autoZstdMin: con STORE_COMPRESSION=auto, desde este largo se usa zstd (lo que sin comprimir iría a desborde).
	autoZstdMin = 2048

	// compressHeaderSize es el encabezado del valor comprimido: method(1) rawLen(4).
	compressHeaderSize = 5

	// defaultCompressMin (STORE_COMPRESSION_MIN): un valor más corto no se comprime.
	defaultCompressMin = 256
)

// compression es la política de compresión de un motor: el método con que escribe y desde qué largo.
type compression struct {
	method byte // compressNone, compressZstd o compressS2
	min    int  // largo mínimo del valor para intentar comprimirlo
}

/**
* envCompression lee STORE_COMPRESSION (s2 por omisión, zstd, auto u off; otro valor = s2) y STORE_COMPRESSION_MIN
* (256 B).
* @return compression
**/
func envCompression() compression {
	return compression{
		method: compressMethodOf(envar.GetStr("STORE_COMPRESSION", "s2")),
		min:    max(envar.GetInt("STORE_COMPRESSION_MIN", defaultCompressMin), 1),
	}
}

/**
* compressMethodOf traduce el nombre de un método: off, none, false, no o 0 la apagan; zstd, auto (S2 bajo
* autoZstdMin y zstd desde ahí) y cualquier otro, S2 (pregunta 81).
* @param name string
* @return byte
**/
func compressMethodOf(name string) byte {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "off", "none", "false", "0", "no":
		return compressNone
	case "zstd":
		return compressZstd
	case "auto":
		return compressAuto
	}
	return compressS2
}

var (
	zstdOnce sync.Once
	zstdEnc  *zstd.Encoder
	zstdDec  *zstd.Decoder
)

/**
* zstdCodec da el codificador y el decodificador de zstd del proceso (EncodeAll y DecodeAll admiten llamadas
* concurrentes), creados la primera vez que se usan.
* @return *zstd.Encoder, *zstd.Decoder
**/
func zstdCodec() (*zstd.Encoder, *zstd.Decoder) {
	zstdOnce.Do(func() {
		var err error
		zstdEnc, err = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest), zstd.WithEncoderCRC(false),
			zstd.WithLowerEncoderMem(true), zstd.WithZeroFrames(true))
		if err != nil {
			panic(err) // solo con opciones inválidas
		}
		zstdDec, err = zstd.NewReader(nil, zstd.WithDecoderLowmem(true), zstd.WithDecoderMaxMemory(uint64(MaxValueLen)),
			zstd.WithDecoderConcurrency(0), zstd.IgnoreChecksum(true))
		if err != nil {
			panic(err)
		}
	})
	return zstdEnc, zstdDec
}

/**
* encode comprime data si la política lo pide y vale la pena: el valor comprimido con su encabezado tiene que
* ahorrar al menos un octavo; si no, se guarda tal cual (leerlo no cuesta CPU).
* @param data []byte: el valor sin comprimir (no se modifica)
* @return []byte: lo que se guarda, bool: si está comprimido
**/
func (c compression) encode(data []byte) ([]byte, bool) {
	if c.method == compressNone || len(data) < c.min || len(data) > MaxValueLen {
		return data, false
	}
	return c.encodeWith(data, len(data)-len(data)/8)
}

/**
* encodeWith comprime data y se queda con el resultado si mide (con el encabezado) a lo sumo limit bytes.
* @param data []byte, limit int
* @return []byte: lo que se guarda, bool: si está comprimido
**/
func (c compression) encodeWith(data []byte, limit int) ([]byte, bool) {
	method := c.method
	if method == compressAuto {
		method = compressS2
		if len(data) >= autoZstdMin {
			method = compressZstd
		}
	}
	var out []byte
	switch method {
	case compressS2:
		buf := make([]byte, compressHeaderSize+s2.MaxEncodedLen(len(data)))
		out = buf[:compressHeaderSize+len(s2.Encode(buf[compressHeaderSize:], data))]
	default:
		enc, _ := zstdCodec()
		out = enc.EncodeAll(data, make([]byte, compressHeaderSize, compressHeaderSize+len(data)/2))
	}
	out[0] = method
	le.PutUint32(out[1:], uint32(len(data)))
	if len(out) > limit {
		return data, false
	}
	return out, true
}

/**
* compressedLen lee el encabezado de un valor comprimido sin descomprimirlo: método conocido y largo sin comprimir
* dentro de MaxValueLen.
* @param stored []byte: el valor guardado (con EntryCompressed)
* @return byte: el método, int: el largo sin comprimir, error: ErrCorrupt
**/
func compressedLen(stored []byte) (byte, int, error) {
	if len(stored) <= compressHeaderSize {
		return 0, 0, fmt.Errorf(MSG_COMPRESSED, ErrCorrupt, "short value")
	}
	method, raw := stored[0], int(le.Uint32(stored[1:]))
	if method != compressZstd && method != compressS2 {
		return 0, 0, fmt.Errorf(MSG_COMPRESSED, ErrCorrupt, fmt.Sprintf("unknown method %d", method))
	}
	if raw > MaxValueLen {
		return 0, 0, fmt.Errorf(MSG_COMPRESSED, ErrCorrupt, fmt.Sprintf("length %d", raw))
	}
	return method, raw, nil
}

/**
* decompress devuelve el valor sin comprimir de un valor guardado con EntryCompressed, en dst si le alcanza (dst[:0]
* se reutiliza; nil = uno nuevo). Valida el encabezado y que el largo resultante sea el anotado, sin confiar en los
* bytes.
* @param dst []byte, stored []byte
* @return []byte, error: ErrCorrupt
**/
func decompress(dst, stored []byte) ([]byte, error) {
	method, raw, err := compressedLen(stored)
	if err != nil {
		return nil, err
	}
	if cap(dst) < raw {
		dst = make([]byte, 0, raw)
	}
	var out []byte
	switch method {
	case compressS2:
		n, derr := s2.DecodedLen(stored[compressHeaderSize:])
		if derr != nil || n != raw {
			return nil, fmt.Errorf(MSG_COMPRESSED, ErrCorrupt, "bad s2 length")
		}
		out, err = s2.Decode(dst[:raw], stored[compressHeaderSize:])
	default:
		_, dec := zstdCodec()
		out, err = dec.DecodeAll(stored[compressHeaderSize:], dst[:0])
	}
	if err != nil {
		return nil, fmt.Errorf(MSG_COMPRESSED, ErrCorrupt, err.Error())
	}
	if len(out) != raw {
		return nil, fmt.Errorf(MSG_COMPRESSED, ErrCorrupt, fmt.Sprintf("length %d, want %d", len(out), raw))
	}
	return out, nil
}

/**
* compressed devuelve el cambio con su valor comprimido según la política del motor (una lápida o un valor que no se
* comprime quedan igual). Se llama una vez por escritura, antes de las vueltas de espera.
* @param c change: con el valor sin comprimir
* @return change
**/
func (e *Engine) compressed(c change) change {
	if c.tombstone || c.compressed {
		return c
	}
	c.data, c.compressed = e.cfg.compress.encode(c.data)
	return c
}
