package store

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
)

const (
	packageName     = "store"
	maxIdLen        = 65535
	fixedHeaderSize = 19 // LSN(8) + DataLen(4) + CRC(4) + IDLen(2) + Status(1)
)

/**
* normalize: Limpia un nombre para usarlo como nombre de archivo.
* @param input string
* @return string
**/
func normalize(input string) string {
	// 1. Quitar espacios al inicio y final
	s := strings.TrimSpace(input)

	// 2. Reemplazar uno o más espacios por _
	s = reSpaces.ReplaceAllString(s, "_")

	// 3. Eliminar todo lo que no sea letra, número, _ o .
	s = reInvalid.ReplaceAllString(s, "")

	// 4. Garantizar que no empiece con número
	s = reLeadingDigits.ReplaceAllString(s, "")

	return s
}

// Expresiones de normalize, compiladas una vez (quien usa el store puede normalizar nombres en cada petición).
var (
	reSpaces        = regexp.MustCompile(`\s+`)
	reInvalid       = regexp.MustCompile(`[^a-zA-Z0-9_.]`)
	reLeadingDigits = regexp.MustCompile(`^[0-9]+`)
)

/**
* newRecordHeaderAt: Codifica el encabezado de un registro con el LSN dado.
* @param lsn uint64, id string, data []byte, status byte
* @return recordHeader, []byte, error
**/
func newRecordHeaderAt(lsn uint64, id string, data []byte, status byte) (recordHeader, []byte, error) {
	idBytes := []byte(id)
	idLen := len(idBytes)

	if idLen == 0 || idLen > maxIdLen {
		return recordHeader{}, nil, errors.New(msg.MSG_STORE_INVALID_ID_LENGTH)
	}

	dataLen := len(data)
	if dataLen > math.MaxUint32 {
		return recordHeader{}, nil, errors.New(msg.MSG_DATA_TOO_LARGE)
	}

	headerLen := fixedHeaderSize + idLen
	result := recordHeader{
		LSN:     lsn,
		DataLen: uint32(dataLen),
		CRC:     checksum(data),
		IDLen:   uint16(idLen),
		Status:  status,
	}

	// Layout: [LSN:8][DataLen:4][CRC:4][IDLen:2][ID:IDLen][Status:1]
	header := make([]byte, headerLen)
	binary.BigEndian.PutUint64(header[0:8], result.LSN)
	putUint32(header[8:12], result.DataLen)
	putUint32(header[12:16], result.CRC)
	putUint16(header[16:18], result.IDLen)
	copy(header[18:18+idLen], idBytes)
	header[18+idLen] = status

	return result, header, nil
}

type Mode int

const (
	ReadOnly Mode = iota
	ReadWrite
)

type FileStore struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	WAL                 counter[uint64] `json:"wal"`         // último LSN escrito
	TombStones          counter[int]    `json:"tomb_stones"` // registros obsoletos pendientes de compactar
	Path                string          `json:"path"`
	PathSnapshot        string          `json:"path_snapshot"`
	PathCompact         string          `json:"path_compact"`
	PathRecover         string          `json:"path_recover"` // cuarentena de bytes corruptos (Recover)
	MaxSegment          int64           `json:"max_segment"`
	SyncOnWrite         bool            `json:"sync_on_write"`
	Size                counter[int64]  `json:"size"`
	MinThresholdCompact int             `json:"min_threshold_compact"`
	writeMu             sync.Mutex      `json:"-"` // serializa log + índice: append, rotación, compaction, close
	indexMu             sync.RWMutex    `json:"-"` // compartido por operaciones de un id; exclusivo para cambiar index, segments o active
	segments            []*segment      `json:"-"` // segmentos de datos
	active              *segment        `json:"-"` // segmento activo para escritura
	index               *shardedIndex   `json:"-"` // índice en memoria, por fragmentos (index.go)
	mode                Mode            `json:"-"` // modo de operación
	compacting          int32           `json:"-"` // 0 = idle, 1 = running
	compactWg           sync.WaitGroup  `json:"-"` // espera que termine la goroutine de compaction
	compactMu           sync.Mutex      `json:"-"` // una sola compaction a la vez (automática o Prune)
	debug               bool            `json:"-"`
	reading             counter[int]    `json:"-"` // Get/ForEach en ejecución
	inserting           counter[int]    `json:"-"` // Insert en ejecución (incluye los que esperan writeMu)
	updating            counter[int]    `json:"-"` // Update en ejecución (incluye los que esperan writeMu)
	deleting            counter[int]    `json:"-"` // Delete en ejecución (incluye los que esperan writeMu)
	syncFns             []func(Change)  `json:"-"` // funciones ancladas con Sync (bajo writeMu)
	statsFns            []func(et.Json) `json:"-"` // funciones ancladas con OnStats
	statsMu             sync.Mutex      `json:"-"` // protege statsFns
	statsCh             chan struct{}   `json:"-"` // aviso de cambio de Stats (buffer 1)
	statsDone           chan struct{}   `json:"-"` // detiene statsLoop
	statsStart          sync.Once       `json:"-"`
	statsStop           sync.Once       `json:"-"`
	appended            counter[uint64] `json:"-"` // registros escritos en el log, en orden (bajo writeMu)
	syncMu              sync.Mutex      `json:"-"` // protege syncing, synced y syncErr (group commit)
	syncCond            *sync.Cond      `json:"-"` // despierta a los que esperan un fsync (sobre syncMu)
	syncing             bool            `json:"-"` // hay un líder haciendo fsync
	synced              uint64          `json:"-"` // último appended que ya está en disco
	syncErr             error           `json:"-"` // primer error de fsync (pegajoso)
}

/**
* toJson: Retorna el estado del store como JSON.
* @return et.Json
**/
func (s *FileStore) toJson() et.Json {
	return et.Json{
		"id":                    s.ID,
		"name":                  s.Name,
		"wal":                   s.WAL.count(),
		"tomb_stones":           s.TombStones.count(),
		"path":                  s.Path,
		"path_snapshot":         s.PathSnapshot,
		"path_compact":          s.PathCompact,
		"path_recover":          s.PathRecover,
		"max_segment":           s.MaxSegment,
		"sync_on_write":         s.SyncOnWrite,
		"size":                  s.Size.count(),
		"min_threshold_compact": s.MinThresholdCompact,
	}
}

/**
* toString: Retorna el estado del store como texto.
* @return string
**/
func (s *FileStore) toString() string {
	return s.toJson().ToString()
}

/**
* isDebug: Activa los logs de depuración.
* @return *FileStore
**/
func (s *FileStore) isDebug() *FileStore {
	s.debug = true
	return s
}

/**
* count: Retorna el número de claves vivas.
* @return int
**/
func (s *FileStore) count() int {
	return s.countIndex()
}

/**
* loadSegments: Abre los archivos de segmento existentes.
* @return error
**/
func (s *FileStore) loadSegments() error {
	files, err := os.ReadDir(s.Path)
	if err != nil {
		return err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, f := range files {
		name := f.Name()
		path := filepath.Join(s.Path, name)
		st, _ := os.Stat(path)

		flag := os.O_CREATE | os.O_RDWR
		if s.mode == ReadOnly {
			flag = os.O_RDONLY
		}
		fd, err := os.OpenFile(path, flag, 0644)
		if err != nil {
			return err
		}

		size := st.Size()
		var seg *segment
		if s.mode == ReadOnly {
			seg = newReadOnlySegment(fd, size, name)
		} else {
			if _, err := fd.Seek(size, io.SeekStart); err != nil {
				return err
			}
			seg = newSegment(fd, size, name)
		}
		s.segments = append(s.segments, seg)
		s.Size.add(size)
		if s.debug {
			logs.Log(packageName, "load:segments:", s.Path, ":", seg.toString())
		}
	}

	// Todos menos el último ya no reciben escrituras (en ReadOnly, ninguno): se mapean. El último,
	// el activo, se mapea con capacidad MaxSegment.
	for i, seg := range s.segments {
		if s.mode == ReadOnly || i < len(s.segments)-1 {
			seg.mapReadOnly()
		} else {
			seg.mapActive(s.MaxSegment)
		}
	}

	if len(s.segments) == 0 {
		if s.mode == ReadOnly {
			return nil
		}
		return s.newSegment()
	}

	s.active = s.segments[len(s.segments)-1]
	return nil
}

/**
* newSegment: Sella el segmento activo y crea uno nuevo (requiere writeMu).
* @return error
**/
func (s *FileStore) newSegment() error {
	if s.active != nil {
		if err := s.active.seal(); err != nil {
			return err
		}
	}

	name := fmt.Sprintf("segment-%06d.dat", len(s.segments)+1)
	path := filepath.Join(s.Path, name)

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	// Get/Read/ForEach leen segments bajo indexMu: el cambio lo toma aunque ya se tenga writeMu.
	seg := newSegment(fd, 0, name)
	seg.mapActive(s.MaxSegment)
	s.indexMu.Lock()
	s.segments = append(s.segments, seg)
	s.active = seg
	s.indexMu.Unlock()
	if s.debug {
		logs.Log(packageName, "new:segment:", s.Path, ":", seg.toString())
	}

	return nil
}

/**
* appendRecordLocked: Agrega un registro al segmento activo, rotándolo si está lleno (requiere writeMu).
* No hace fsync: la durabilidad la espera commit.
* lsn 0 asigna el siguiente LSN local; lsn > 0 conserva el LSN recibido (replicación).
* @param lsn uint64, id string, data []byte, status byte
* @return *recordRef, uint64, error
**/
func (s *FileStore) appendRecordLocked(lsn uint64, id string, data []byte, status byte) (*recordRef, uint64, error) {
	recordSize := int64(fixedHeaderSize) + int64(len(id)) + int64(len(data))
	if s.active.size+recordSize > s.MaxSegment {
		if err := s.newSegment(); err != nil {
			return nil, 0, err
		}

		if err := s.createSnapshot(); err != nil {
			return nil, 0, err
		}
	}

	if lsn == 0 {
		lsn = s.WAL.inc()
	} else {
		s.WAL.setMax(lsn)
	}

	ref, err := s.active.writeRecord(lsn, id, data, status)
	if err != nil {
		return nil, 0, err
	}
	ref.segment = len(s.segments) - 1
	s.Size.add(recordSize)
	// El fsync lo hace commit fuera de writeMu, uno para todos los que esperan (group commit).
	s.appended.inc()

	return ref, lsn, nil
}

/**
* compactIfNeeded: Lanza una compactación en segundo plano si los tombstones superan el 10% del
* índice (o MinThresholdCompact).
**/
func (s *FileStore) compactIfNeeded() {
	threshold := max(int(float64(s.countIndex())*0.1), s.MinThresholdCompact)
	if s.TombStones.count() <= threshold {
		return
	}

	if atomic.CompareAndSwapInt32(&s.compacting, 0, 1) {
		s.compactWg.Add(1)
		go func() {
			defer s.compactWg.Done()
			defer atomic.StoreInt32(&s.compacting, 0)
			if err := s.compact(); err != nil && s.debug {
				logs.Debug("compact error:", err)
			}
		}()
	}
}

/**
* getIndex: Retorna la referencia de id.
* @param id string
* @return *recordRef, bool
**/
func (s *FileStore) getIndex(id string) (*recordRef, bool) {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()
	return s.index.get(id)
}

/**
* countIndex: Retorna el número de claves del índice.
* @return int
**/
func (s *FileStore) countIndex() int {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()
	return s.index.len()
}

/**
* setIndex: Guarda la referencia de id; retorna true si ya existía.
* indexMu compartido: las escrituras ya se serializan con writeMu, y el fragmento de id
* basta para separarla de los lectores.
* @param id string, ref *recordRef
* @return bool (true if id already existed)
**/
func (s *FileStore) setIndex(id string, ref *recordRef) bool {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()
	return s.setIndexLocked(id, ref)
}

/**
* setIndexLocked: Igual que setIndex, pero requiere indexMu tomado.
* @param id string, ref *recordRef
* @return bool (true if id already existed)
**/
func (s *FileStore) setIndexLocked(id string, ref *recordRef) bool {
	exists := s.index.set(id, ref)
	if s.debug {
		logs.Debug("put:", s.Path, ":lsn:", s.WAL.count(), ":ID:", id, ":ref:", ref.toString())
	}
	return exists
}

/**
* deleteIndex: Quita id del índice; retorna true si existía.
* @param id string
* @return bool (true if id existed)
**/
func (s *FileStore) deleteIndex(id string) bool {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()
	return s.deleteIndexLocked(id)
}

/**
* deleteIndexLocked: Igual que deleteIndex, pero requiere indexMu tomado.
* @param id string
* @return bool (true if id existed)
**/
func (s *FileStore) deleteIndexLocked(id string) bool {
	return s.index.remove(id)
}

/**
* rebuildIndex: Aplica un segmento al índice (requiere indexMu).
* @param segIndex int
* @return error
**/
func (s *FileStore) rebuildIndex(segIndex int) (int64, error) {
	if s.index == nil {
		s.index = newShardedIndex(nil)
	}

	validEnd := int64(0)
	err := s.segments[segIndex].scan(0, func(offset int64, h recordHeader, data []byte) error {
		validEnd = offset + h.recordSize()
		if h.Status == Active {
			s.setIndexLocked(h.ID, &recordRef{segment: segIndex, offset: offset, length: h.DataLen, crc: h.CRC, hasCRC: true})
		} else if h.Status == Deleted {
			s.deleteIndexLocked(h.ID)
		}

		// Restaurar WAL al LSN más alto visto en disco
		s.WAL.setMax(h.LSN)
		return nil
	})
	return validEnd, err
}

/**
* buildIndex: Aplica el segmento activo sobre el índice cargado del snapshot.
* @return int64 (fin del último registro válido del activo), error
**/
func (s *FileStore) buildIndex() (int64, error) {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	idx := len(s.segments) - 1
	return s.rebuildIndex(idx)
}

/**
* sortedKeysLocked: Retorna las claves ordenadas asc/desc con offset y limit (limit <= 0 sin límite);
* requiere indexMu.
* @param asc bool, offset int, limit int
* @return []string
**/
func (s *FileStore) sortedKeysLocked(asc bool, offset, limit int) []string {
	keys, _ := s.index.sorted(asc, offset, limit)
	return keys
}

/**
* pinLocked: Retorna el segmento de ref marcado como en lectura; requiere indexMu y llamar release().
* @param ref *recordRef
* @return *segment, error
**/
func (s *FileStore) pinLocked(ref *recordRef) (*segment, error) {
	if ref == nil || ref.segment < 0 || ref.segment >= len(s.segments) {
		return nil, errors.New(msg.MSG_STORE_CORRUPTED_RECORD)
	}

	seg := s.segments[ref.segment]
	seg.acquire()
	return seg, nil
}

/**
* rebuildIndexes: Reconstruye el índice recorriendo todos los segmentos.
* @return int64 (fin del último registro válido del último segmento), error
**/
func (s *FileStore) rebuildIndexes() (int64, error) {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	s.index = newShardedIndex(nil)
	validEnd := int64(0)
	for i := range s.segments {
		end, err := s.rebuildIndex(i)
		if err != nil {
			return 0, err
		}
		validEnd = end
	}

	return validEnd, nil
}

/**
* repairTail: Tras una caída el segmento activo puede terminar en un registro cortado (los
* sellados no: la rotación hace fsync antes de crear el siguiente). Si hay bytes después del
* último registro válido, los copia a PathRecover y trunca el activo en validEnd, para que
* lo escrito desde ahora quede a continuación de lo válido y no detrás de la basura.
* @param validEnd int64
* @return error
**/
func (s *FileStore) repairTail(validEnd int64) error {
	seg := s.active
	if seg == nil || validEnd >= seg.size {
		return nil
	}

	quarantine, err := s.quarantineTail(seg.file, seg.name, validEnd, seg.size, time.Now().Format("20060102150405"))
	if err != nil {
		return err
	}
	if _, err := seg.file.Seek(validEnd, io.SeekStart); err != nil {
		return err
	}

	logs.Log(packageName, "open:repair:", s.Path, ":", seg.name, ":truncated:", seg.size-validEnd, ":quarantine:", quarantine)
	s.Size.add(validEnd - seg.size)
	seg.size = validEnd
	seg.visible.Store(validEnd)
	return nil
}

/**
* close: Espera la compactación, hace fsync y retira todos los segmentos.
* @return error
**/
func (s *FileStore) close() error {
	s.stopStats()
	s.compactWg.Wait()

	// compactMu también espera una Compact() lanzada por Prune.
	s.compactMu.Lock()
	defer s.compactMu.Unlock()

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	// fsync y retiro de cada segmento: se cierra ahora o cuando termine su último lector.
	var closeErr error
	for _, seg := range s.segments {
		if err := seg.seal(); err != nil && closeErr == nil {
			closeErr = err
		}
		seg.retire()
	}

	return closeErr
}

/**
* empty: Cierra el store y borra sus datos.
* @return error
**/
func (s *FileStore) empty() error {
	err := s.close()
	if err != nil {
		return err
	}

	s.indexMu.Lock()
	s.index = newShardedIndex(nil)
	s.indexMu.Unlock()
	s.WAL.set(0)
	s.TombStones.set(0)
	s.Size.set(0)

	// Without its snapshot, the next Open with this name starts empty (the snapshot
	// would bring back the old index and the old WAL counter)
	if err := s.removeSnapshot(); err != nil {
		return err
	}
	defer os.RemoveAll(s.Path)
	return nil
}

/**
* checkWrite: Valida que el store acepte escrituras y que id no esté vacío.
* @param id string
* @return error
**/
func (s *FileStore) checkWrite(id string) error {
	if s.mode == ReadOnly {
		return errors.New(msg.MSG_STORE_IS_READ_ONLY)
	}

	if id == "" {
		return errors.New(msg.MSG_STORE_ID_IS_REQUIRED)
	}

	return nil
}

/**
* put: Escribe data en el log y actualiza el índice (requiere writeMu); retorna true si id ya existía.
* @param id string, data []byte
* @return bool (true if id already existed), uint64 (LSN), error
**/
func (s *FileStore) put(id string, data []byte) (bool, uint64, error) {
	ref, lsn, err := s.appendRecordLocked(0, id, data, Active)
	if err != nil {
		return false, 0, err
	}

	exists := s.setIndex(id, ref)
	if exists {
		s.TombStones.inc()
	}

	return exists, lsn, nil
}

/**
* insert: Guarda data solo si id no existe; retorna true si insertó.
* @param id string, data []byte, wait bool (false: no espera el fsync)
* @return bool (true if inserted), error
**/
func (s *FileStore) insert(id string, data []byte, wait bool) (bool, error) {
	if err := s.checkWrite(id); err != nil {
		return false, err
	}

	defer s.track(&s.inserting)()

	return s.commit(wait, func() (bool, error) {
		if _, exists := s.getIndex(id); exists {
			return false, nil
		}

		_, lsn, err := s.put(id, data)
		if err != nil {
			return false, err
		}

		s.emitLocked(Change{Op: OpInsert, ID: id, Data: data, LSN: lsn})

		return true, nil
	})
}

/**
* update: Reemplaza data solo si id existe y el valor cambió; retorna true si actualizó.
* @param id string, data []byte, wait bool (false: no espera el fsync)
* @return bool (true if updated), error
**/
func (s *FileStore) update(id string, data []byte, wait bool) (bool, error) {
	if err := s.checkWrite(id); err != nil {
		return false, err
	}

	defer s.track(&s.updating)()

	return s.commit(wait, func() (bool, error) {
		ref, exists := s.getIndex(id)
		if !exists {
			return false, nil
		}

		// Con el CRC en el índice, un dato distinto se reconoce sin leer el disco bajo writeMu.
		if !ref.differs(data) {
			old, err := s.readPinned(ref, id)
			if err != nil {
				return false, err
			}
			if bytes.Equal(old, data) {
				return false, nil
			}
		}

		_, lsn, err := s.put(id, data)
		if err != nil {
			return false, err
		}

		s.emitLocked(Change{Op: OpUpdate, ID: id, Data: data, LSN: lsn})
		s.compactIfNeeded()

		return true, nil
	})
}

/**
* delete: Elimina id solo si existe; retorna true si eliminó.
* @param id string, wait bool (false: no espera el fsync)
* @return bool (true if deleted), error
**/
func (s *FileStore) delete(id string, wait bool) (bool, error) {
	if err := s.checkWrite(id); err != nil {
		return false, err
	}

	defer s.track(&s.deleting)()

	// Verificación, tombstone e índice bajo writeMu: nada se cuela entre ellos.
	return s.commit(wait, func() (bool, error) {
		if _, exists := s.getIndex(id); !exists {
			return false, nil
		}

		_, lsn, err := s.appendRecordLocked(0, id, nil, Deleted)
		if err != nil {
			return false, err
		}

		s.deleteIndex(id)
		s.TombStones.inc()

		if s.debug {
			logs.Debug("deleted:", s.Path, ":total:", s.countIndex(), ":ID:", id)
		}

		s.emitLocked(Change{Op: OpDelete, ID: id, LSN: lsn})
		s.compactIfNeeded()

		return true, nil
	})
}

/**
* isExist: Indica si id existe.
* @param id string
* @return bool
**/
func (s *FileStore) isExist(id string) bool {
	if id == "" {
		return false
	}

	_, existed := s.getIndex(id)
	return existed
}

/**
* readPinned: Lee el registro de ref sin sostener indexMu durante la lectura; con id vacío lo lee del disco.
* @param ref *recordRef, id string
* @return []byte, error
**/
func (s *FileStore) readPinned(ref *recordRef, id string) ([]byte, error) {
	s.indexMu.RLock()
	seg, err := s.pinLocked(ref)
	s.indexMu.RUnlock()
	if err != nil {
		return nil, err
	}
	defer seg.release()

	if id == "" {
		return seg.read(ref)
	}
	return seg.readRecord(ref, id)
}

/**
* get: Retorna los datos de id, si existe.
* @param id string
* @return []byte, bool (true if id exists), error
**/
func (s *FileStore) get(id string) ([]byte, bool, error) {
	defer s.track(&s.reading)()

	// Búsqueda y fijado bajo un mismo RLock; la lectura a disco va después.
	s.indexMu.RLock()
	ref, exists := s.index.get(id)
	if !exists {
		s.indexMu.RUnlock()
		return nil, false, nil
	}
	seg, err := s.pinLocked(ref)
	s.indexMu.RUnlock()
	if err != nil {
		return nil, false, err
	}
	defer seg.release()

	data, err := seg.readRecord(ref, id)
	if err != nil {
		return nil, false, err
	}

	return data, true, nil
}

/**
* getVersion: get más la versión de id: el LSN de su último registro (0 si no existe).
* Cambia con cada Insert/Update/Delete que escribe y se conserva al compactar y al
* replicar (ApplyWalEntry mantiene el LSN).
* @param id string
* @return []byte, uint64, bool, error
**/
func (s *FileStore) getVersion(id string) ([]byte, uint64, bool, error) {
	defer s.track(&s.reading)()

	s.indexMu.RLock()
	ref, exists := s.index.get(id)
	if !exists {
		s.indexMu.RUnlock()
		return nil, 0, false, nil
	}
	seg, err := s.pinLocked(ref)
	s.indexMu.RUnlock()
	if err != nil {
		return nil, 0, false, err
	}
	defer seg.release()

	data, lsn, err := seg.readVersion(ref, id)
	if err != nil {
		return nil, 0, false, err
	}
	return data, lsn, true, nil
}

/**
* forEachItem: Registro seleccionado por ForEach, con su segmento fijado.
**/
type forEachItem struct {
	id  string
	ref *recordRef
	seg *segment
}

/**
* forEachResult: Resultado de leer un forEachItem.
**/
type forEachResult struct {
	data []byte
	err  error
}

/**
* forEach: Llama a fn por cada registro en orden de clave (asc/desc) con offset y limit
* (limit <= 0 sin límite). Lee en paralelo con hasta NumCPU goroutines, pero llama
* a fn de a uno y en orden; si fn retorna false se detiene. No sostiene locks,
* así que fn puede usar el store.
* @param fn func(id string, data []byte) (bool, error), asc bool, offset, limit int
* @return error
**/
func (s *FileStore) forEach(fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error {
	defer s.track(&s.reading)()

	// ---- Selección: claves y segmentos fijados bajo un solo RLock ----
	s.indexMu.RLock()
	keys, refs := s.index.sorted(asc, offset, limit)
	items := make([]forEachItem, len(keys))
	pinned := make(map[*segment]bool)
	for i, id := range keys {
		ref := refs[i]
		seg, err := s.pinLocked(ref)
		if err != nil {
			s.indexMu.RUnlock()
			for seg := range pinned {
				seg.release()
			}
			return err
		}
		if pinned[seg] {
			seg.release() // basta un pin por segmento
		}
		pinned[seg] = true
		items[i] = forEachItem{id: id, ref: ref, seg: seg}
	}
	s.indexMu.RUnlock()
	defer func() {
		for seg := range pinned {
			seg.release()
		}
	}()

	n := len(items)
	if n == 0 {
		return nil
	}

	// ---- Lectura concurrente con entrega ordenada ----
	// Los workers leen en paralelo y dejan cada resultado en slots[i%window]; esta
	// goroutine los toma en orden y llama a fn. sem limita lo pendiente a window.
	workers := min(runtime.NumCPU(), n)
	window := workers * 4
	slots := make([]chan forEachResult, window)
	for i := range slots {
		slots[i] = make(chan forEachResult, 1)
	}
	jobs := make(chan int, window)
	sem := make(chan struct{}, window)
	done := make(chan struct{})

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for i := range jobs {
				select {
				case <-done:
					slots[i%window] <- forEachResult{}
					continue
				default:
				}
				data, err := items[i].seg.readRecord(items[i].ref, items[i].id)
				slots[i%window] <- forEachResult{data: data, err: err}
			}
		})
	}

	wg.Go(func() {
		defer close(jobs)
		for i := range n {
			select {
			case sem <- struct{}{}:
			case <-done:
				return
			}
			jobs <- i
		}
	})

	// Detener el productor y esperar a todos antes de liberar los segmentos (incluso si fn entra en pánico).
	defer func() {
		close(done)
		wg.Wait()
	}()

	for i := range n {
		r := <-slots[i%window]
		<-sem
		if r.err != nil {
			return r.err
		}

		cont, err := fn(items[i].id, r.data)
		if err != nil {
			return err
		}
		if !cont {
			return nil
		}
	}

	return nil
}

/**
* keys: Retorna las claves ordenadas asc/desc con offset y limit, sin leer los registros.
* @param asc bool, offset int, limit int
* @return []string
**/
func (s *FileStore) keys(asc bool, offset, limit int) []string {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	return s.sortedKeysLocked(asc, offset, limit)
}

/**
* newFileStore: Arma un FileStore con sus rutas y configuración, sin tocar el disco.
* @param pathData, pathWald, name string, mode Mode
* @return *FileStore
**/
func newFileStore(pathData, pathWald, name string, mode Mode) *FileStore {
	name = normalize(name)
	result := &FileStore{
		Name:                name,
		Path:                filepath.Join(pathData, "segments", name),
		PathSnapshot:        filepath.Join(pathWald, "snapshot", name),
		PathCompact:         filepath.Join(pathWald, "compact", name),
		PathRecover:         filepath.Join(pathWald, "recover", name),
		MaxSegment:          envar.GetInt64("RELSEG_SIZE", 128) * 1024 * 1024,
		MinThresholdCompact: envar.GetInt("MIN_THRESHOLD_COMPACT", 1000),
		SyncOnWrite:         envar.GetBool("SYNC_ON_WRITE", true),
		index:               newShardedIndex(nil),
		mode:                mode,
		statsCh:             make(chan struct{}, 1),
		statsDone:           make(chan struct{}),
	}
	result.syncCond = sync.NewCond(&result.syncMu)
	return result
}

/**
* open: Abre (o crea) el store y carga su índice.
* @param pathData, pathWald, name string, mode Mode
* @return *FileStore, error
**/
func open(pathData, pathWald, name string, mode Mode) (*FileStore, error) {
	fs := newFileStore(pathData, pathWald, name, mode)

	if mode == ReadOnly {
		if _, err := os.Stat(fs.Path); os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: store not found at %s", errors.New(msg.MSG_STORE_NOT_FOUND), fs.Path)
		}
	} else {
		// Una compactación interrumpida deja los segmentos a medio renombrar: se restauran antes
		// de crear el directorio, o Open arrancaría vacío.
		restored, removedOld, _, err := fs.repairCompaction()
		if err != nil {
			return nil, fmt.Errorf("repairCompaction: %w", err)
		}
		if restored || removedOld {
			logs.Log(packageName, "open:repair:", fs.Path, ":restored:", restored, ":removed_old:", removedOld)
		}
		if restored {
			if err := fs.removeSnapshot(); err != nil {
				return nil, err
			}
		}

		if err := os.MkdirAll(fs.Path, 0755); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(fs.PathSnapshot, 0755); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(fs.PathCompact, 0755); err != nil {
			return nil, err
		}
	}

	if err := fs.loadSegments(); err != nil {
		return nil, fmt.Errorf("loadSegments: %w", err)
	}

	// Con snapshot solo se aplica el último segmento; sin snapshot se recorren todos.
	snapshotLoaded, snapErr := fs.tryLoadSnapshot()
	if snapErr != nil && fs.debug {
		logs.Debug("snapshot unavailable, full rebuild:", snapErr)
	}
	var validEnd int64
	var err error
	if snapshotLoaded {
		if validEnd, err = fs.buildIndex(); err != nil {
			return nil, fmt.Errorf("buildIndex: %w", err)
		}
	} else {
		if validEnd, err = fs.rebuildIndexes(); err != nil {
			return nil, fmt.Errorf("rebuildIndexes: %w", err)
		}
	}

	if mode == ReadWrite {
		if err := fs.repairTail(validEnd); err != nil {
			return nil, fmt.Errorf("repairTail: %w", err)
		}
	}

	return fs, nil
}
