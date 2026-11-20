package control

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Desplazamientos de los campos de una copia del control (tabla en doc.go).
const (
	offMagic          = 0
	offVersion        = 4
	offSeq            = 8
	offState          = 16
	offPageSize       = 20
	offBlocksPerFile  = 24
	offReserved1      = 28
	offCheckpointTime = 32
	offRedoSegment    = 40
	offRedoOffset     = 48
	offRedoLSN        = 56
	offCheckpointLSN  = 64
	offNextXID        = 72
	offOldestXID      = 80
	offFrozenXID      = 88
	offSlots          = 96
	offReserved2      = offSlots + SlotCount*SlotSize // 352
	offCRC            = CopySize - 4                  // 508
)

// castagnoli es la tabla de CRC-32C que usan todos los formatos del motor.
var castagnoli = crc32.MakeTable(crc32.Castagnoli)

var le = binary.LittleEndian

// validState dice si s es uno de los estados conocidos.
func validState(s State) bool {
	return s == StateShutdown || s == StateInProduction || s == StateInRecovery
}

// checkControl valida los campos de c que el formato restringe.
// c: el control. Devuelve ErrBadControl con el detalle, o nil.
func checkControl(c Control) error {
	switch {
	case c.Seq == 0:
		return fmt.Errorf("%w: seq 0", ErrBadControl)
	case !validState(c.State):
		return fmt.Errorf("%w: unknown state %d", ErrBadControl, c.State)
	case c.Redo.Offset < 0:
		return fmt.Errorf("%w: negative redo offset %d", ErrBadControl, c.Redo.Offset)
	case c.CheckpointLSN < c.Redo.LSN:
		return fmt.Errorf("%w: checkpoint lsn %d before redo lsn %d", ErrBadControl, c.CheckpointLSN, c.Redo.LSN)
	}
	for i, s := range c.Slots {
		if len(s.Name) > SlotNameLen || bytes.IndexByte([]byte(s.Name), 0) >= 0 {
			return fmt.Errorf("%w: slot %d name must have up to %d bytes and no NUL", ErrBadControl, i, SlotNameLen)
		}
	}
	return nil
}

// encodeControl arma una copia de CopySize bytes con su CRC.
// c: el control, con su Seq. Devuelve la copia o ErrBadControl si un campo no cabe en el formato.
func encodeControl(c Control) ([]byte, error) {
	if err := checkControl(c); err != nil {
		return nil, err
	}
	b := make([]byte, CopySize)
	le.PutUint32(b[offMagic:], ControlMagic)
	le.PutUint32(b[offVersion:], FormatVersion)
	le.PutUint64(b[offSeq:], c.Seq)
	le.PutUint32(b[offState:], uint32(c.State))
	le.PutUint32(b[offPageSize:], page.Size)
	le.PutUint32(b[offBlocksPerFile:], smgr.BlocksPerFile)
	le.PutUint64(b[offCheckpointTime:], uint64(c.CheckpointTime))
	le.PutUint64(b[offRedoSegment:], c.Redo.Segment)
	le.PutUint64(b[offRedoOffset:], uint64(c.Redo.Offset))
	le.PutUint64(b[offRedoLSN:], uint64(c.Redo.LSN))
	le.PutUint64(b[offCheckpointLSN:], uint64(c.CheckpointLSN))
	le.PutUint64(b[offNextXID:], uint64(c.NextXID))
	le.PutUint64(b[offOldestXID:], uint64(c.OldestActiveXID))
	le.PutUint64(b[offFrozenXID:], uint64(c.FrozenXID))
	for i, s := range c.Slots {
		off := offSlots + i*SlotSize
		copy(b[off:off+SlotNameLen], s.Name)
		le.PutUint64(b[off+SlotNameLen:], uint64(s.RestartLSN))
	}
	le.PutUint32(b[offCRC:], crc32.Checksum(b[:offCRC], castagnoli))
	return b, nil
}

// decodeControl lee una copia sin confiar en sus bytes: comprueba largo, CRC, magia, versión, tamaños, ceros
// reservados y rangos. Lo que acepta, encodeControl lo vuelve a escribir igual byte a byte.
// b: la copia. Devuelve el control o ErrBadControl con el detalle.
func decodeControl(b []byte) (Control, error) {
	if len(b) != CopySize {
		return Control{}, fmt.Errorf("%w: length %d, want %d", ErrBadControl, len(b), CopySize)
	}
	if got, want := le.Uint32(b[offCRC:]), crc32.Checksum(b[:offCRC], castagnoli); got != want {
		return Control{}, fmt.Errorf("%w: crc %08x, want %08x", ErrBadControl, got, want)
	}
	if m := le.Uint32(b[offMagic:]); m != ControlMagic {
		return Control{}, fmt.Errorf("%w: magic %08x", ErrBadControl, m)
	}
	if v := le.Uint32(b[offVersion:]); v != FormatVersion {
		return Control{}, fmt.Errorf("%w: %w: %d, want %d", ErrBadControl, ErrFormatVersion, v, FormatVersion)
	}
	if n := le.Uint32(b[offPageSize:]); n != page.Size {
		return Control{}, fmt.Errorf("%w: page size %d, want %d", ErrBadControl, n, page.Size)
	}
	if n := le.Uint32(b[offBlocksPerFile:]); n != smgr.BlocksPerFile {
		return Control{}, fmt.Errorf("%w: blocks per file %d, want %d", ErrBadControl, n, smgr.BlocksPerFile)
	}
	if !allZero(b[offReserved1:offCheckpointTime]) || !allZero(b[offReserved2:offCRC]) {
		return Control{}, fmt.Errorf("%w: reserved bytes are not zero", ErrBadControl)
	}

	c := Control{
		Seq:            le.Uint64(b[offSeq:]),
		State:          State(le.Uint32(b[offState:])),
		CheckpointTime: int64(le.Uint64(b[offCheckpointTime:])),
		Redo: wal.Position{
			Segment: le.Uint64(b[offRedoSegment:]),
			Offset:  int64(le.Uint64(b[offRedoOffset:])),
			LSN:     page.LSN(le.Uint64(b[offRedoLSN:])),
		},
		CheckpointLSN:   page.LSN(le.Uint64(b[offCheckpointLSN:])),
		NextXID:         page.XID(le.Uint64(b[offNextXID:])),
		OldestActiveXID: page.XID(le.Uint64(b[offOldestXID:])),
		FrozenXID:       page.XID(le.Uint64(b[offFrozenXID:])),
	}
	for i := range c.Slots {
		off := offSlots + i*SlotSize
		name, err := decodeSlotName(b[off : off+SlotNameLen])
		if err != nil {
			return Control{}, fmt.Errorf("%w: slot %d: %v", ErrBadControl, i, err)
		}
		c.Slots[i] = Slot{Name: name, RestartLSN: page.LSN(le.Uint64(b[off+SlotNameLen:]))}
	}
	if err := checkControl(c); err != nil {
		return Control{}, err
	}
	return c, nil
}

// decodeSlotName lee el nombre de una ranura: los bytes hasta el primer cero, y el resto en cero.
// b: los SlotNameLen bytes del nombre. Devuelve el nombre o un error si hay bytes después del relleno.
func decodeSlotName(b []byte) (string, error) {
	n := bytes.IndexByte(b, 0)
	if n < 0 {
		return string(b), nil
	}
	if !allZero(b[n:]) {
		return "", fmt.Errorf("name padding is not zero")
	}
	return string(b[:n]), nil
}

// allZero dice si todos los bytes de b son cero.
func allZero(b []byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}

// EncodeCopy es encodeControl para las pruebas de ida y vuelta del formato (store/format_test.go).
// c: el control, con su Seq. Devuelve la copia de CopySize bytes o ErrBadControl.
func EncodeCopy(c Control) ([]byte, error) {
	return encodeControl(c)
}

// DecodeCopy es decodeControl para las pruebas de ida y vuelta del formato: lo que acepta, EncodeCopy lo vuelve a
// escribir igual byte a byte.
// b: una copia. Devuelve el control, o ErrBadControl (ErrFormatVersion si es de otra versión).
func DecodeCopy(b []byte) (Control, error) {
	return decodeControl(b)
}
