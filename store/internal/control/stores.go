package control

import (
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// FirstFileNo es el primer número de archivo de un motor nuevo; el 0 no es válido.
const FirstFileNo smgr.FileNo = 1

// Stores es el catálogo en memoria: nombre → store, con el contador de Count de cada uno. Es seguro con varias
// goroutines; su lock es una hoja (ver store/doc.go, «Orden de los bloqueos»).
//
// Create, Drop y Empty son idempotentes para rehacerlos desde el WAL en orden: cada uno nombra el número de archivo
// que espera, así un registro viejo no toca un store que después se borró, vació o volvió a crear con otro número.
// Ninguno escribe al disco: el motor guarda Catalog() con CatalogFile.Save cuando corresponde.
type Stores struct {
	mu         sync.RWMutex
	nextFileNo smgr.FileNo
	countLSN   page.LSN
	byName     map[string]*Entry
	byFile     map[smgr.FileNo]*Entry
}

// NewStores devuelve un catálogo vacío de un motor nuevo.
func NewStores() *Stores {
	return &Stores{
		nextFileNo: FirstFileNo,
		byName:     map[string]*Entry{},
		byFile:     map[smgr.FileNo]*Entry{},
	}
}

// LoadStores arma el catálogo en memoria a partir del leído del disco.
// c: el catálogo. Devuelve los Stores o ErrBadCatalog si c no cumple las reglas del formato.
func LoadStores(c Catalog) (*Stores, error) {
	if err := checkCatalog(c); err != nil {
		return nil, err
	}
	s := NewStores()
	s.nextFileNo = c.NextFileNo
	s.countLSN = c.CountLSN
	for _, e := range c.Entries {
		s.add(e)
	}
	return s, nil
}

// add agrega e a los dos mapas. Requiere s.mu tomado (o s sin publicar).
func (s *Stores) add(e Entry) {
	p := &e
	s.byName[e.Name] = p
	s.byFile[e.File] = p
}

// Catalog devuelve una copia del catálogo, con las entradas ordenadas por nombre, para guardarla con Save.
// Los contadores son los de este momento: el motor la toma bajo writeMu y le pone CountLSN = el lsn del próximo
// registro (o usa SetCountLSN antes), para que contadores y countLSN sean del mismo instante.
func (s *Stores) Catalog() Catalog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c := Catalog{NextFileNo: s.nextFileNo, CountLSN: s.countLSN, Entries: make([]Entry, 0, len(s.byName))}
	for _, e := range s.byName {
		c.Entries = append(c.Entries, *e)
	}
	sort.Slice(c.Entries, func(i, j int) bool { return c.Entries[i].Name < c.Entries[j].Name })
	return c
}

// Get devuelve el store llamado name y si existe.
func (s *Stores) Get(name string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if e, ok := s.byName[name]; ok {
		return *e, true
	}
	return Entry{}, false
}

// ByFile devuelve el store cuyo número de archivo es file y si existe (la recuperación salta los bloques de un
// archivo que no está).
func (s *Stores) ByFile(file smgr.FileNo) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if e, ok := s.byFile[file]; ok {
		return *e, true
	}
	return Entry{}, false
}

// Names devuelve los nombres de los stores en orden.
func (s *Stores) Names() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.byName))
	for name := range s.byName {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Len es la cantidad de stores.
func (s *Stores) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byName)
}

// NextFileNo es el próximo número de archivo que daría AllocFileNo.
func (s *Stores) NextFileNo() smgr.FileNo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextFileNo
}

// CountLSN es el lsn hasta el que cuentan los contadores (las confirmaciones con lsn menor ya están sumadas).
func (s *Stores) CountLSN() page.LSN {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.countLSN
}

// SetCountLSN fija CountLSN (el motor lo hace en el checkpoint, con el lsn del punto de rehacer).
func (s *Stores) SetCountLSN(lsn page.LSN) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.countLSN = lsn
}

// AllocFileNo reserva un número de archivo nuevo para crear o vaciar un store. Los números nunca se reutilizan: uno
// reservado y no usado (el registro del WAL no llegó a escribirse) queda como hueco.
// Devuelve el número o ErrBadFileNo si se agotaron.
func (s *Stores) AllocFileNo() (smgr.FileNo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.nextFileNo == math.MaxUint32 {
		return 0, fmt.Errorf("%w: file numbers exhausted", ErrBadFileNo)
	}
	file := s.nextFileNo
	s.nextFileNo++
	return file, nil
}

// reserveFileNo deja nextFileNo por encima de file (un registro rehecho trae números que el catálogo leído puede no
// tener reservados). Requiere s.mu tomado.
func (s *Stores) reserveFileNo(file smgr.FileNo) error {
	if file == 0 || file == math.MaxUint32 {
		return fmt.Errorf("%w: %d", ErrBadFileNo, file)
	}
	if file >= s.nextFileNo {
		s.nextFileNo = file + 1
	}
	return nil
}

// Create da de alta el store name con el archivo file (KindCreateStore). Si name ya existe no cambia nada.
// name: 1 a MaxNameLen bytes; file: de AllocFileNo o del registro del WAL.
// Devuelve si lo agregó; ErrBadName, o ErrBadFileNo si file es 0 o ya es de otro store (corrupción).
func (s *Stores) Create(name string, file smgr.FileNo) (bool, error) {
	if err := checkName(name); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byName[name]; ok {
		return false, nil
	}
	if other, ok := s.byFile[file]; ok {
		return false, fmt.Errorf("%w: %d already belongs to store %q", ErrBadFileNo, file, other.Name)
	}
	if err := s.reserveFileNo(file); err != nil {
		return false, err
	}
	s.add(Entry{Name: name, File: file})
	return true, nil
}

// Drop da de baja el store name si su archivo es file (KindDropStore); si no está o tiene otro número no cambia
// nada. Devuelve la entrada quitada (el motor borra sus archivos) y si la quitó.
func (s *Stores) Drop(name string, file smgr.FileNo) (Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byName[name]
	if !ok || e.File != file {
		return Entry{}, false
	}
	delete(s.byName, name)
	delete(s.byFile, file)
	return *e, true
}

// Empty vacía el store name (KindEmptyStore): si su archivo es oldFile pasa a newFile, con la página meta 0 y el
// contador en 0; si no está o tiene otro número no cambia nada. El motor borra después los archivos de oldFile.
// Devuelve si lo vació; ErrBadFileNo si newFile es 0, igual a oldFile o ya es de otro store.
func (s *Stores) Empty(name string, oldFile, newFile smgr.FileNo) (bool, error) {
	if newFile == oldFile {
		return false, fmt.Errorf("%w: empty keeps file number %d", ErrBadFileNo, newFile)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byName[name]
	if !ok || e.File != oldFile {
		return false, nil
	}
	if other, ok := s.byFile[newFile]; ok {
		return false, fmt.Errorf("%w: %d already belongs to store %q", ErrBadFileNo, newFile, other.Name)
	}
	if err := s.reserveFileNo(newFile); err != nil {
		return false, err
	}
	delete(s.byFile, oldFile)
	e.File, e.MetaBlock, e.Count = newFile, 0, 0
	s.byFile[newFile] = e
	return true, nil
}

// AddCount suma delta al contador del store con el archivo file (una confirmación). Si el archivo no es de ningún
// store (se borró o vació después) no hace nada.
// Devuelve si sumó; ErrNegativeCount (sin cambiar nada) si el contador quedaría negativo o desbordaría.
func (s *Stores) AddCount(file smgr.FileNo, delta int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byFile[file]
	if !ok {
		return false, nil
	}
	if delta < 0 && (delta == math.MinInt64 || e.Count < -delta) || delta > 0 && e.Count > math.MaxInt64-delta {
		return false, fmt.Errorf("%w: store %q has %d, delta %d", ErrNegativeCount, e.Name, e.Count, delta)
	}
	e.Count += delta
	return true, nil
}

// SetMeta fija el bloque de la página meta del store con el archivo file (siempre 0 en la fase 1).
// Devuelve si el store existe.
func (s *Stores) SetMeta(file smgr.FileNo, block page.BlockNumber) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byFile[file]
	if ok {
		e.MetaBlock = block
	}
	return ok
}

// SetFlags fija las marcas del store con el archivo file (reservadas, fase 2). Devuelve si el store existe.
func (s *Stores) SetFlags(file smgr.FileNo, flags uint32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byFile[file]
	if ok {
		e.Flags = flags
	}
	return ok
}
