package store

import (
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// Respaldo en caliente (fase 3, tarea 44 de F-23; ver «Respaldo en caliente y restauración» en doc.go): como
// pg_basebackup, una copia de los archivos del motor mientras sigue escribiendo, más el WAL desde el punto de rehacer
// del checkpoint del inicio hasta el registro KindBackupEnd. Open sobre la copia la restaura: rehace ese WAL.

const (
	// backupChunk es el búfer fijo de la copia: 32 páginas. Nada se lee entero en memoria.
	backupChunk = 32 * page.Size

	// backupEndMainSize: redoLSN(8) checkpointLSN(8) time(8), los datos principales de KindBackupEnd.
	backupEndMainSize = 24
)

// BackupInfo describe un respaldo terminado.
type BackupInfo struct {
	StartLSN      uint64        // punto de rehacer del checkpoint del inicio: la restauración rehace desde ahí
	CheckpointLSN uint64        // registro KindCheckpoint de ese checkpoint
	EndLSN        uint64        // registro KindBackupEnd: la restauración tiene que llegar hasta él
	Files         int           // archivos copiados (control, catálogo, datos, estados, segmentos del WAL, marcador)
	Bytes         int64         // bytes copiados (sin contar el catálogo)
	Took          time.Duration // duración
}

// BackupSink recibe los archivos de un respaldo hacia un flujo (BackupTo), de a uno y con su largo, como pide un
// tar: name es relativo a la copia, con «/» («catalog», «data/0000000001.0001», «wal/0000000003.seg»). Restaurar es
// dejar los archivos en un directorio y abrirlo (Open, OpenAt, OpenAtTime u OpenReplica).
type BackupSink interface {
	File(name string, size int64) (io.WriteCloser, error)
}

// backups es el estado de los respaldos en curso de un motor (puede haber varios a la vez).
type backups struct {
	// mu es una hoja: frozen es el frozenXID del control de cada respaldo, por número; el paso 7 del checkpoint no
	// trunca el archivo de estados por debajo del menor.
	mu     sync.Mutex
	frozen map[int]page.XID
	next   int

	// Con ddlMu: copying cuenta los respaldos que copian archivos de datos; mientras haya alguno, los archivos que
	// borran DropStore, Empty y el DDL en lote se anotan en deferred y se borran al terminar esa copia.
	copying  int
	deferred []smgr.FileNo
}

/**
* holdFrozen anota el frozenXID del control de un respaldo: el archivo de estados no se trunca por debajo.
* @param frozen page.XID
* @return int: el número del respaldo, para releaseFrozen
**/
func (b *backups) holdFrozen(frozen page.XID) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.frozen == nil {
		b.frozen = map[int]page.XID{}
	}
	b.next++
	b.frozen[b.next] = frozen
	return b.next
}

/**
* releaseFrozen suelta lo que anotó holdFrozen.
* @param id int
**/
func (b *backups) releaseFrozen(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.frozen, id)
}

/**
* truncateLimit es hasta dónde puede truncar el paso 7 del checkpoint: su frozenXID, o el menor de los respaldos en
* curso si es menor (sus copias tienen ese frozenXID en el control y necesitan los archivos de estados de arriba).
* @param frozen page.XID
* @return page.XID
**/
func (b *backups) truncateLimit(frozen page.XID) page.XID {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, f := range b.frozen {
		frozen = min(frozen, f)
	}
	return frozen
}

/**
* deferredFile dice si los archivos de file esperan el fin de la copia de un respaldo para borrarse. Con ddlMu.
* @param file smgr.FileNo
* @return bool
**/
func (b *backups) deferredFile(file smgr.FileNo) bool {
	return slices.Contains(b.deferred, file)
}

/**
* unlinkData borra los archivos de un store que ya no es de nadie (DropStore, Empty, DDL en lote), o lo deja para el
* fin de la copia si un respaldo está copiando archivos de datos: su catálogo todavía lo tiene y el WAL que rehace el
* borrado necesita sus páginas. Con ddlMu tomado.
* @param file smgr.FileNo
* @return error
**/
func (e *Engine) unlinkData(file smgr.FileNo) error {
	if e.backups.copying > 0 {
		e.backups.deferred = append(e.backups.deferred, file)
		return nil
	}
	return e.sm.Unlink(file)
}

// backupRun es un respaldo en curso: lo que tomó al empezar y lo que tiene que soltar al terminar.
type backupRun struct {
	e    *Engine
	dst  vfs.FS
	dir  string
	sink BackupSink // BackupTo: los archivos van a sink, no a dst (nil en Backup)
	buf  []byte
	info BackupInfo

	ckpt    checkpointInfo  // el checkpoint del inicio
	ckptLSN page.LSN        // su registro
	cat     control.Catalog // el catálogo tomado con ddlMu al empezar
	data    []string        // los archivos de data/ de los stores de cat
	end     wal.Position    // la posición siguiente a KindBackupEnd
	endLSN  page.LSN        // el lsn de KindBackupEnd
	unforce func()          // apaga el modo respaldo (wal.ForceImages)
	unhold  func()          // suelta la retención del WAL (wal.Hold)
	frozen  int             // número de holdFrozen (0 = ninguno)
	copying bool            // cuenta en backups.copying
}

/**
* backup copia el motor en caliente al directorio dir de dst (vacío o inexistente) sin detener escrituras, lotes ni
* DDL. Pasos: modo respaldo y checkpoint (el inicio), catálogo y archivos de datos, archivos de estados, registro
* KindBackupEnd, WAL desde el punto de rehacer hasta ese registro, control y marcador. Todo lo que toma lo suelta al
* volver, también con error (finish). Close lo interrumpe (ErrClosed).
* @param dst vfs.FS, dir string
* @return BackupInfo, error: ErrClosed, ErrFailed, ErrDiskFull (en el destino o en el WAL), ErrBackupDir
**/
func (e *Engine) backup(dst vfs.FS, dir string) (BackupInfo, error) {
	started := time.Now()
	if err := e.enter(); err != nil {
		return BackupInfo{}, err
	}
	e.bg.Add(1) // Close espera al respaldo (lo interrumpe con stop) antes de cerrar las piezas
	e.leave()
	defer e.bg.Done()

	b := &backupRun{e: e, dst: dst, dir: dir, buf: make([]byte, backupChunk)}
	defer b.finish()
	if err := b.run(); err != nil {
		return BackupInfo{}, fmt.Errorf(MSG_BACKUP, dir, err)
	}
	b.info.Took = time.Since(started)
	return b.info, nil
}

/**
* backupTo es backup hacia un flujo (tarea 77): los mismos pasos, cada archivo entregado a sink de a uno con su largo
* (lo que pide un tar), sin directorio intermedio. Los archivos chicos (catálogo, control, marcador) se arman en un
* disco en memoria con el mismo código que en Backup. El marcador sale dos veces: «en curso» primero y el final al
* terminar (como en un tar, el segundo pisa al primero; una copia cortada no se puede abrir).
* @param sink BackupSink
* @return BackupInfo, error: los de Backup y los de sink
**/
func (e *Engine) backupTo(sink BackupSink) (BackupInfo, error) {
	started := time.Now()
	if err := e.enter(); err != nil {
		return BackupInfo{}, err
	}
	e.bg.Add(1)
	e.leave()
	defer e.bg.Done()

	b := &backupRun{e: e, dst: crashfs.New(), dir: "/backup", sink: sink, buf: make([]byte, backupChunk)}
	defer b.finish()
	if err := b.run(); err != nil {
		return BackupInfo{}, fmt.Errorf(MSG_BACKUP, "sink", err)
	}
	b.info.Took = time.Since(started)
	return b.info, nil
}

/**
* run hace los pasos del respaldo en orden.
* @return error
**/
func (b *backupRun) run() error {
	steps := []func() error{b.prepare, b.start, b.copyData, b.copyXact, b.writeEnd, b.copyWAL, b.finishDest}
	for _, step := range steps {
		if err := b.alive(); err != nil {
			return err
		}
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

/**
* alive devuelve ErrClosed si Close está esperando al respaldo.
* @return error
**/
func (b *backupRun) alive() error {
	select {
	case <-b.e.stop:
		return ErrClosed
	default:
		return nil
	}
}

/**
* prepare crea el directorio de destino, exige que esté vacío y deja el marcador «en curso» (sin fin): una copia a
* medias no se puede abrir.
* @return error
**/
func (b *backupRun) prepare() error {
	if b.sink != nil {
		if err := b.dst.MkdirAll(b.dir); err != nil { // el disco en memoria de los archivos chicos
			return err
		}
		return b.writeLabel(backupLabel{})
	}
	if err := b.dst.MkdirAll(b.dir); err != nil {
		return destError(err)
	}
	names, err := b.dst.ReadDir(b.dir)
	if err != nil {
		return destError(err)
	}
	if len(names) > 0 {
		return fmt.Errorf(MSG_BACKUP_DIR, ErrBackupDir, b.dir)
	}
	for _, sub := range []string{smgr.DataDir, xact.Dir, wal.Dir} {
		if err := b.dst.MkdirAll(filepath.Join(b.dir, sub)); err != nil {
			return destError(err)
		}
	}
	return b.writeLabel(backupLabel{})
}

/**
* start es el inicio: con ckptMu, el modo respaldo (imágenes de página enteras forzadas), un checkpoint sin repartir
* (su punto de rehacer es el inicio), la retención del WAL desde su segmento y del archivo de estados desde su
* frozenXID; después, con ddlMu, el catálogo en memoria y la lista de archivos de data/ de sus stores, y desde ahí
* los borrados de archivos esperan al fin de la copia de datos.
* @return error
**/
func (b *backupRun) start() error {
	e := b.e
	if err := e.writable(); err != nil {
		return err
	}
	e.ckptMu.Lock()
	defer e.ckptMu.Unlock()
	_, b.unforce = e.log.ForceImages()
	info, lsn, err := e.checkpointLocked(control.StateInProduction, false)
	if err != nil {
		return err
	}
	b.ckpt, b.ckptLSN = info, lsn
	b.unhold = e.log.Hold(info.redo.Segment)
	b.frozen = e.backups.holdFrozen(info.frozen)
	b.info.StartLSN, b.info.CheckpointLSN = uint64(info.redo.LSN), uint64(lsn)

	e.ddlMu.Lock()
	defer e.ddlMu.Unlock()
	b.cat = e.cat.Catalog()
	names, err := e.fs.ReadDir(filepath.Join(e.dir, smgr.DataDir))
	if err != nil {
		return err
	}
	for _, name := range names {
		if file, ok := parseDataName(name); ok && hasFile(b.cat, file) {
			b.data = append(b.data, name)
		}
	}
	e.backups.copying++
	b.copying = true
	return nil
}

/**
* hasFile dice si algún store del catálogo tiene el número de archivo file.
* @param cat control.Catalog, file smgr.FileNo
* @return bool
**/
func hasFile(cat control.Catalog, file smgr.FileNo) bool {
	return slices.ContainsFunc(cat.Entries, func(entry control.Entry) bool { return entry.File == file })
}

/**
* copyData guarda el catálogo del inicio en el destino y copia los archivos de datos de sus stores, de a uno y por
* tramos de backupChunk; al terminar, los borrados que esperaban se hacen. Un archivo que ya no está (lo recreó Store
* tras un corte) se salta: el WAL lo vuelve a crear.
* @return error
**/
func (b *backupRun) copyData() error {
	b.e.backupStep("start")
	if err := control.NewCatalogFile(b.dst, b.dir).Save(b.cat); err != nil {
		return destError(err)
	}
	if err := b.sendSmall(control.CatalogFileName); err != nil {
		return err
	}
	b.count(0)
	for _, name := range b.data {
		path := filepath.Join(smgr.DataDir, name)
		if err := b.copyFile(path, -1, true); err != nil {
			return err
		}
		b.e.backupStep(path)
	}
	if err := b.syncDest(smgr.DataDir); err != nil {
		return err
	}
	return b.releaseFiles()
}

/**
* copyXact copia los archivos de estados que haya ahora (ninguno se trunca por debajo del frozenXID del inicio
* mientras dure el respaldo).
* @return error
**/
func (b *backupRun) copyXact() error {
	names, err := b.e.fs.ReadDir(filepath.Join(b.e.dir, xact.Dir))
	if err != nil {
		return err
	}
	for _, name := range names {
		path := filepath.Join(xact.Dir, name)
		if err := b.copyFile(path, -1, false); err != nil {
			return err
		}
		b.e.backupStep(path)
	}
	return b.syncDest(xact.Dir)
}

/**
* writeEnd escribe KindBackupEnd en el WAL del motor (durable) y apaga el modo respaldo: todo lo que la copia leyó
* tiene un lsn menor, así que rehacer hasta este registro deja la copia consistente.
* @return error
**/
func (b *backupRun) writeEnd() error {
	e := b.e
	e.backupStep("end")
	if err := e.writable(); err != nil {
		return err
	}
	main := encodeBackupEnd(b.ckpt.redo.LSN, b.ckptLSN, time.Now().UnixNano())
	res, err := e.log.Reserve(&wal.Record{Kind: wal.KindBackupEnd, Main: main}, nil)
	if errors.Is(err, wal.ErrNoRoom) {
		return e.diskFull(err) // no se reservó nada: el motor sigue
	}
	if err != nil {
		return e.fail(err)
	}
	b.end, b.endLSN = res.End(), res.LSN()
	if err := res.Write(); err != nil {
		return e.fail(err)
	}
	if err := e.log.WaitDurable(b.endLSN); err != nil {
		return e.fail(err)
	}
	b.info.EndLSN = uint64(b.endLSN)
	b.unforce() // ya no hace falta forzar imágenes
	return nil
}

/**
* copyWAL copia los segmentos retenidos desde el del punto de rehacer del inicio hasta el de KindBackupEnd, este
* último solo hasta el final de ese registro (lo que sigue lo escribieron otros después).
* @return error
**/
func (b *backupRun) copyWAL() error {
	for seg := b.ckpt.redo.Segment; seg <= b.end.Segment; seg++ {
		limit := int64(-1)
		if seg == b.end.Segment {
			limit = b.end.Offset
		}
		path := filepath.Join(wal.Dir, fmt.Sprintf("%010d.seg", seg))
		if err := b.copyFile(path, limit, false); err != nil {
			return err
		}
		b.e.backupStep(path)
	}
	return b.syncDest(wal.Dir)
}

/**
* finishDest escribe en el destino el control del checkpoint del inicio y el marcador con el lsn de KindBackupEnd:
* recién entonces la copia se puede abrir.
* @return error
**/
func (b *backupRun) finishDest() error {
	c := control.Control{
		State:           control.StateInProduction,
		CheckpointTime:  b.ckpt.time,
		Redo:            b.ckpt.redo,
		CheckpointLSN:   b.ckptLSN,
		NextXID:         b.ckpt.nextXID,
		OldestActiveXID: b.ckpt.oldest,
		FrozenXID:       b.ckpt.frozen,
	}
	if err := control.NewControlFile(b.dst, b.dir).Create(c); err != nil {
		return destError(err)
	}
	if err := b.sendSmall(control.FileName); err != nil {
		return err
	}
	b.count(2 * control.CopySize)
	return b.writeLabel(backupLabel{redo: b.ckpt.redo, checkpointLSN: b.ckptLSN, endLSN: b.endLSN,
		time: b.ckpt.time})
}

/**
* releaseFiles termina la cuenta de la copia de datos de este respaldo y, si era la última, borra los archivos que
* esperaban (sus registros ya son durables). Un fallo deja el motor inutilizable, como cualquier borrado.
* @return error
**/
func (b *backupRun) releaseFiles() error {
	if !b.copying {
		return nil
	}
	b.copying = false
	e := b.e
	e.ddlMu.Lock()
	defer e.ddlMu.Unlock()
	e.backups.copying--
	if e.backups.copying > 0 {
		return nil
	}
	files := e.backups.deferred
	e.backups.deferred = nil
	for _, file := range files {
		if err := e.sm.Unlink(file); err != nil {
			return e.fail(err)
		}
	}
	return nil
}

/**
* finish suelta todo lo que tomó el respaldo, haya terminado bien o no: los borrados pendientes, el modo respaldo, la
* retención del WAL y la del archivo de estados.
**/
func (b *backupRun) finish() {
	_ = b.releaseFiles() // un fallo ya dejó el motor inutilizable
	for _, release := range []func(){b.unforce, b.unhold} {
		if release != nil {
			release()
		}
	}
	if b.frozen != 0 {
		b.e.backups.releaseFrozen(b.frozen)
	}
}

/**
* copyFile copia rel (relativo al directorio del motor) al mismo lugar del destino, por tramos de backupChunk, y hace
* su fsync. Sin limit (−1) copia el largo actual; con pages corta al último bloque entero (un bloque a medio extender
* lo rehace el WAL). Un origen que no existe se salta.
* @param rel string, limit int64, pages bool
* @return error
**/
func (b *backupRun) copyFile(rel string, limit int64, pages bool) error {
	src, err := b.e.fs.OpenFile(filepath.Join(b.e.dir, rel), os.O_RDONLY)
	if errors.Is(err, iofs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer src.Close()
	size, err := src.Size()
	if err != nil {
		return err
	}
	if limit >= 0 {
		size = min(size, limit)
	}
	if pages {
		size -= size % page.Size
	}
	if b.sink != nil {
		return b.sendFile(rel, src, size)
	}
	out, err := b.dst.OpenFile(filepath.Join(b.dir, rel), os.O_RDWR|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return destError(err)
	}
	n, err := b.copyRange(src, out, size)
	if err == nil {
		err = destError(out.Sync())
	}
	if cerr := out.Close(); err == nil {
		err = destError(cerr)
	}
	b.count(n)
	return err
}

/**
* copyRange copia los primeros size bytes de src a out con el búfer fijo. Si src termina antes (no pasa: los
* archivos no se acortan), copia lo que hay.
* @param src vfs.File, out io.WriterAt, size int64
* @return int64: bytes copiados, error
**/
func (b *backupRun) copyRange(src vfs.File, out io.WriterAt, size int64) (int64, error) {
	var off int64
	for off < size {
		if err := b.alive(); err != nil {
			return off, err
		}
		chunk := b.buf[:min(int64(len(b.buf)), size-off)]
		n, err := src.ReadAt(chunk, off)
		if n > 0 {
			if _, werr := out.WriteAt(chunk[:n], off); werr != nil {
				return off, destError(werr)
			}
			off += int64(n)
		}
		if err == io.EOF {
			return off, nil
		}
		if err != nil {
			return off, err
		}
	}
	return off, nil
}

/**
* syncDest hace fsync del subdirectorio sub del destino (las altas de sus archivos).
* @param sub string
* @return error
**/
func (b *backupRun) syncDest(sub string) error {
	if b.sink != nil {
		return nil // el flujo es del llamador
	}
	return destError(b.dst.SyncDir(filepath.Join(b.dir, sub)))
}

/**
* sendFile entrega a sink los primeros size bytes de src como rel (con «/», relativo a la copia), por tramos de
* backupChunk.
* @param rel string, src vfs.File, size int64
* @return error
**/
func (b *backupRun) sendFile(rel string, src vfs.File, size int64) error {
	out, err := b.sink.File(filepath.ToSlash(rel), size)
	if err != nil {
		return destError(err)
	}
	n, err := b.copyRange(src, writerAt{out}, size)
	if err == nil && n != size {
		err = fmt.Errorf(MSG_BACKUP_SHORT_FILE, rel, n, size)
	}
	if cerr := out.Close(); err == nil {
		err = destError(cerr)
	}
	b.count(n)
	return err
}

/**
* sendSmall entrega a sink el archivo name que se armó en el disco en memoria (catálogo, control, marcador). Sin
* sink no hace nada.
* @param name string
* @return error
**/
func (b *backupRun) sendSmall(name string) error {
	if b.sink == nil {
		return nil
	}
	src, err := b.dst.OpenFile(filepath.Join(b.dir, name), os.O_RDONLY)
	if err != nil {
		return err
	}
	defer src.Close()
	size, err := src.Size()
	if err != nil {
		return err
	}
	out, err := b.sink.File(name, size)
	if err != nil {
		return destError(err)
	}
	_, err = b.copyRange(src, writerAt{out}, size)
	if cerr := out.Close(); err == nil {
		err = destError(cerr)
	}
	return err
}

// writerAt adapta un io.Writer a la escritura secuencial de copyRange (los desplazamientos llegan en orden).
type writerAt struct{ w io.Writer }

// WriteAt escribe p a continuación (copyRange da los tramos en orden).
func (w writerAt) WriteAt(p []byte, _ int64) (int, error) { return w.w.Write(p) }

/**
* count suma un archivo copiado a BackupInfo.
* @param n int64: sus bytes
**/
func (b *backupRun) count(n int64) {
	b.info.Files++
	b.info.Bytes += n
}

/**
* backupStep avisa a las pruebas en cada paso del respaldo (afterBackupStep), sin nada tomado: "start" (tomado el
* inicio), la ruta de cada archivo copiado y "end" antes de escribir KindBackupEnd.
* @param step string
**/
func (e *Engine) backupStep(step string) {
	if e.afterBackupStep != nil {
		e.afterBackupStep(step)
	}
}

/**
* destError marca como ErrDiskFull un disco lleno del destino (el motor sigue sano).
* @param err error
* @return error
**/
func destError(err error) error {
	if err != nil && vfs.IsNoSpace(err) && !errors.Is(err, ErrDiskFull) {
		return fmt.Errorf(MSG_DISK_FULL, ErrDiskFull, err)
	}
	return err
}

/**
* encodeBackupEnd arma los datos principales de KindBackupEnd.
* @param redo, checkpoint page.LSN, now int64
* @return []byte
**/
func encodeBackupEnd(redo, checkpoint page.LSN, now int64) []byte {
	b := make([]byte, backupEndMainSize)
	le.PutUint64(b[0:], uint64(redo))
	le.PutUint64(b[8:], uint64(checkpoint))
	le.PutUint64(b[16:], uint64(now))
	return b
}

/**
* decodeBackupEnd lee el punto de rehacer de un KindBackupEnd.
* @param b []byte
* @return page.LSN: el redoLSN del respaldo, bool: false si el largo no es el del formato
**/
func decodeBackupEnd(b []byte) (page.LSN, bool) {
	if len(b) != backupEndMainSize {
		return 0, false
	}
	return page.LSN(le.Uint64(b[0:])), true
}
