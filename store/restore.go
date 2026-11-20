package store

import (
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"slices"

	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Restauración de un respaldo en caliente (ver «Respaldo en caliente y restauración» en doc.go): Open sobre la copia
// encuentra el marcador backup_label y, antes de cambiar nada, comprueba que el WAL copiado llega hasta el registro
// KindBackupEnd; después recupera como siempre y, terminada la recuperación, borra el marcador.

const (
	// backupLabelName es el marcador de una copia (como backup_label de PostgreSQL); backupLabelTemp, su versión a
	// medio escribir.
	backupLabelName = "backup_label"
	backupLabelTemp = "backup_label.tmp"

	// backupLabelMagic es "ETBK"; backupLabelSize, el largo del marcador.
	backupLabelMagic = 0x4554424b
	backupLabelSize  = 64
)

// crcTable es la tabla CRC-32C del marcador, como la del resto de los formatos.
var crcTable = crc32.MakeTable(crc32.Castagnoli)

// backupLabel es el contenido del marcador. endLSN = 0: la copia no terminó.
type backupLabel struct {
	redo          wal.Position // el punto de rehacer del checkpoint del inicio
	checkpointLSN page.LSN
	endLSN        page.LSN // el registro KindBackupEnd
	time          int64
}

/**
* encodeLabel arma el marcador: magic(4) version(4) redoSegment(8) redoOffset(8) redoLSN(8) checkpointLSN(8)
* endLSN(8) time(8) reservado(4) crc(4) de todo lo anterior.
* @param l backupLabel
* @return []byte
**/
func encodeLabel(l backupLabel) []byte {
	b := make([]byte, backupLabelSize)
	le.PutUint32(b[0:], backupLabelMagic)
	le.PutUint32(b[4:], control.FormatVersion)
	le.PutUint64(b[8:], l.redo.Segment)
	le.PutUint64(b[16:], uint64(l.redo.Offset))
	le.PutUint64(b[24:], uint64(l.redo.LSN))
	le.PutUint64(b[32:], uint64(l.checkpointLSN))
	le.PutUint64(b[40:], uint64(l.endLSN))
	le.PutUint64(b[48:], uint64(l.time))
	le.PutUint32(b[60:], crc32.Checksum(b[:60], crcTable))
	return b
}

/**
* decodeLabel lee el marcador.
* @param b []byte
* @return backupLabel, bool: false si el largo, el magic, la versión o el crc no son los del formato
**/
func decodeLabel(b []byte) (backupLabel, bool) {
	if len(b) != backupLabelSize || le.Uint32(b[0:]) != backupLabelMagic || le.Uint32(b[4:]) != control.FormatVersion ||
		le.Uint32(b[60:]) != crc32.Checksum(b[:60], crcTable) {
		return backupLabel{}, false
	}
	return backupLabel{
		redo: wal.Position{
			Segment: le.Uint64(b[8:]),
			Offset:  int64(le.Uint64(b[16:])),
			LSN:     page.LSN(le.Uint64(b[24:])),
		},
		checkpointLSN: page.LSN(le.Uint64(b[32:])),
		endLSN:        page.LSN(le.Uint64(b[40:])),
		time:          int64(le.Uint64(b[48:])),
	}, true
}

/**
* writeLabel escribe el marcador en el destino de forma atómica: backup_label.tmp, fsync, rename y fsync del
* directorio.
* @param l backupLabel
* @return error
**/
func (b *backupRun) writeLabel(l backupLabel) error {
	tmp, path := filepath.Join(b.dir, backupLabelTemp), filepath.Join(b.dir, backupLabelName)
	f, err := b.dst.OpenFile(tmp, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return destError(err)
	}
	_, err = f.WriteAt(encodeLabel(l), 0)
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = b.dst.Rename(tmp, path)
	}
	if err == nil {
		err = b.dst.SyncDir(b.dir)
	}
	if err == nil && l.endLSN != 0 {
		b.count(backupLabelSize)
	}
	return destError(err)
}

/**
* readBackupLabel lee el marcador del motor, si lo hay.
* @return backupLabel, bool: si hay marcador, error: ErrBackupIncomplete si está a medias o dañado
**/
func (e *Engine) readBackupLabel() (backupLabel, bool, error) {
	names, err := e.fs.ReadDir(e.dir)
	if err != nil {
		return backupLabel{}, false, err
	}
	if !slices.Contains(names, backupLabelName) {
		return backupLabel{}, false, nil
	}
	f, err := e.fs.OpenFile(filepath.Join(e.dir, backupLabelName), os.O_RDONLY)
	if err != nil {
		return backupLabel{}, true, err
	}
	defer f.Close()
	b := make([]byte, backupLabelSize+1)
	n, _ := f.ReadAt(b, 0)
	l, ok := decodeLabel(b[:n])
	switch {
	case !ok:
		return l, true, fmt.Errorf(MSG_BACKUP_LABEL, ErrBackupIncomplete)
	case l.endLSN == 0:
		return l, true, fmt.Errorf(MSG_BACKUP_UNFINISHED, ErrBackupIncomplete)
	}
	return l, true, nil
}

/**
* checkBackupWAL comprueba, sin escribir nada, que el WAL desde redo (el punto de rehacer del control) llega hasta el
* registro KindBackupEnd del marcador y que ese registro es el de este respaldo. Si el punto de rehacer ya pasó ese
* registro (la copia ya se restauró y un corte se llevó el borrado del marcador), no hay nada que comprobar.
* @param redo wal.Position, l backupLabel
* @return error: ErrBackupIncomplete
**/
func (e *Engine) checkBackupWAL(redo wal.Position, l backupLabel) error {
	if redo.LSN > l.endLSN {
		return nil
	}
	errFound := errors.New("backup end found")
	end, err := wal.NewReader(e.fs, e.dir).Scan(redo, func(rec *wal.Record) error {
		if rec.LSN != l.endLSN {
			return nil
		}
		if start, ok := decodeBackupEnd(rec.Main); rec.Kind != wal.KindBackupEnd || !ok || start != l.redo.LSN {
			return fmt.Errorf(MSG_BACKUP_END_RECORD, ErrBackupIncomplete, rec.LSN, rec.Kind)
		}
		return errFound
	})
	switch {
	case errors.Is(err, errFound):
		return nil
	case err != nil:
		return fmt.Errorf(MSG_BACKUP_WAL, ErrBackupIncomplete, l.endLSN, err)
	}
	return fmt.Errorf(MSG_BACKUP_SHORT, ErrBackupIncomplete, end.LSN-1, l.endLSN)
}

/**
* removeBackupLabel borra el marcador al terminar la recuperación de una copia (con fsync del directorio): desde ahí
* el motor es uno más.
* @return error
**/
func (e *Engine) removeBackupLabel() error {
	if err := e.fs.Remove(filepath.Join(e.dir, backupLabelName)); err != nil {
		return err
	}
	return e.fs.SyncDir(e.dir)
}

/**
* checkTarget comprueba, sin escribir nada, el objetivo de OpenAt (recovery_target_lsn): solo sobre una copia de
* Backup (sus páginas no tienen cambios posteriores al fin del respaldo), no antes del fin del respaldo ni del
* checkpoint del control (la copia no sería consistente) y con el registro de ese lsn en el WAL que hay (el del
* respaldo más los segmentos archivados agregados a wal/).
* @param ctl control.Control, l backupLabel, restoring bool
* @return error: ErrRecoveryTarget
**/
func (e *Engine) checkTarget(ctl control.Control, l backupLabel, restoring bool) error {
	stop := e.cfg.stopLSN
	if !restoring {
		return fmt.Errorf(MSG_TARGET_NO_BACKUP, ErrRecoveryTarget)
	}
	if low := max(l.endLSN, ctl.CheckpointLSN); stop < low {
		return fmt.Errorf(MSG_TARGET_EARLY, ErrRecoveryTarget, stop, low)
	}
	errFound := errors.New("target found")
	end, err := wal.NewReader(e.fs, e.dir).Scan(ctl.Redo, func(rec *wal.Record) error {
		if rec.LSN == stop {
			return errFound
		}
		return nil
	})
	switch {
	case errors.Is(err, errFound):
		return nil
	case err != nil:
		return fmt.Errorf(MSG_TARGET_WAL, ErrRecoveryTarget, stop, err)
	}
	return fmt.Errorf(MSG_TARGET_LATE, ErrRecoveryTarget, end.LSN-1, stop)
}
