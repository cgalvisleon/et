package store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
)

/**
* read: El valor de id, sin tomar writeMu: índice → pread del registro → verificar crc, lsn e id. Una sola asignación:
* el registro completo, del que el valor es una vista (no se copia otra vez).
* @param id string
* @return []byte, bool (si existe), error
**/
func (s *Store) read(id string) ([]byte, bool, error) {
	if err := checkID(id); err != nil {
		return nil, false, err
	}
	if s.closed.Load() {
		return nil, false, errors.New(MSG_CLOSED)
	}
	r, ok := s.index.get(id)
	if !ok {
		return nil, false, nil
	}
	data, err := s.readRef(id, r)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

/**
* readRef: Lee y verifica el registro al que apunta r. El registro ya está escrito: el índice solo recibe una ref
* después de su WriteAt, y los segmentos solo crecen (nada sobrescribe un registro).
* @param id string, r ref
* @return []byte (el valor), error
**/
func (s *Store) readRef(id string, r ref) ([]byte, error) {
	// closed se mira con mapsMu tomado: close lo marca antes de esperar a los lectores (waitReaders), así que o esta
	// lectura ve closed, o close espera a que termine antes de deshacer los mapas.
	defer s.mapsMu.runlock(s.mapsMu.rlock())
	if s.closed.Load() {
		return nil, errors.New(MSG_CLOSED)
	}

	seg := s.segmentAt(r.segment)
	if seg == nil {
		return nil, fmt.Errorf(MSG_BAD_REF, id, r.segment)
	}
	rec, mapped, err := readRecord(seg, r)
	if err != nil {
		return nil, err
	}
	h := decodeHeader(rec)
	if h.size() != len(rec) || !validRecord(h, rec) || h.lsn != r.lsn || !sameID(h, rec, id) {
		return nil, fmt.Errorf(MSG_CORRUPTED_READ, id, seg.path, r.offset)
	}
	data := recordData(h, rec)
	if mapped {
		// La vista es del mapa, que se deshace al cerrar: el llamador recibe su propia copia.
		data = append(make([]byte, 0, len(data)), data...)
	}
	return data, nil
}

/**
* readRecord: Los bytes del registro de r: una vista del mapa del segmento si cabe en él (sin llamada al sistema), o
* un pread si no (fuera de unix, o un registro más grande que el tamaño de segmento).
* @param seg *segment, r ref
* @return []byte, bool (si es una vista del mapa), error
**/
func readRecord(seg *segment, r ref) ([]byte, bool, error) {
	if rec, ok := seg.view(r.offset, int(r.size)); ok {
		return rec, true, nil
	}
	buf := make([]byte, r.size)
	if err := seg.readAt(buf, r.offset); err != nil {
		if errors.Is(err, os.ErrClosed) {
			return nil, false, errors.New(MSG_CLOSED)
		}
		return nil, false, err
	}
	return buf, false, nil
}

/**
* sameID: El registro es de id. Comparar string(bytes) con un string no asigna memoria en Go.
* @param h header, buf []byte, id string
* @return bool
**/
func sameID(h header, buf []byte, id string) bool {
	return string(buf[headerSize:headerSize+h.idLen]) == id
}

/**
* put: Guarda data para id sin mirar lo que había (escritura ciega).
* @param id string, data []byte
* @return error
**/
func (s *Store) put(id string, data []byte) error {
	if err := checkPut(id, data); err != nil {
		return err
	}
	_, err := s.write(id, kindPut, data, nil)
	return err
}

/**
* remove: Borra id sin mirar si existe (escritura ciega): siempre anexa un registro de borrado. Repetirlo da lo mismo.
* @param id string
* @return error
**/
func (s *Store) remove(id string) error {
	if err := checkID(id); err != nil {
		return err
	}
	_, err := s.write(id, kindDelete, nil, nil)
	return err
}

/**
* insert: Guarda data solo si id no existe. La comprobación y la reserva van juntas bajo writeMu (ver reserve).
* @param id string, data []byte
* @return bool (si lo guardó), error
**/
func (s *Store) insert(id string, data []byte) (bool, error) {
	if err := checkPut(id, data); err != nil {
		return false, err
	}
	return s.write(id, kindPut, data, func(cur current) bool { return !cur.exists })
}

/**
* delete: Borra id solo si existe.
* @param id string
* @return bool (si lo borró), error
**/
func (s *Store) delete(id string) (bool, error) {
	if err := checkID(id); err != nil {
		return false, err
	}
	return s.write(id, kindDelete, nil, func(cur current) bool { return cur.exists })
}

/**
* update: Guarda data solo si id existe y su valor es distinto. Comparar exige leer el valor actual, y no se lee del
* disco con writeMu tomado (pararía a todos los escritores): se lee fuera, y la reserva solo se hace si el id sigue en
* la misma versión (el mismo lsn) que se leyó. Si otro lo escribió en medio, se vuelve a leer (concurrencia
* optimista).
* @param id string, data []byte
* @return bool (si lo cambió), error
**/
func (s *Store) update(id string, data []byte) (bool, error) {
	if err := checkPut(id, data); err != nil {
		return false, err
	}
	for {
		cur, value, err := s.readCurrent(id)
		if err != nil || !cur.exists || bytes.Equal(value, data) {
			return false, err
		}
		sameVersion := func(now current) bool { return now == cur }
		done, err := s.write(id, kindPut, data, sameVersion)
		if err != nil || done {
			return done, err
		}
	}
}

/**
* readCurrent: El estado lógico de id y, si existe, su valor. Si el id tiene escrituras pendientes, su valor todavía
* no se puede leer: se espera a que estén publicadas y se vuelve a mirar.
* @param id string
* @return current, []byte, error
**/
func (s *Store) readCurrent(id string) (current, []byte, error) {
	for {
		if s.closed.Load() {
			return current{}, nil, errors.New(MSG_CLOSED)
		}
		cur, busy := s.pending.state(id, s.index)
		if busy {
			if err := s.waitPublished(cur.lsn); err != nil {
				return current{}, nil, err
			}
			continue
		}
		if !cur.exists {
			return cur, nil, nil
		}
		r, ok := s.index.get(id)
		if !ok || r.lsn != cur.lsn {
			continue // otro escritor de id entre las dos miradas
		}
		value, err := s.readRef(id, r)
		return cur, value, err
	}
}
