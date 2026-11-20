package store

/**
* getVersion devuelve la versión más nueva confirmada de id (como Read) con su número de versión: el lsn del registro
* que la escribió.
* @param id string
* @return []byte: copia del valor, uint64: la versión (0 si no existe), bool: si existe, error
**/
func (s *Store) getVersion(id string) ([]byte, uint64, bool, error) {
	cur, found, err := s.readVersion(id, nil)
	return cur.value, uint64(cur.lsn), found, err
}

/**
* getVersionIn devuelve id como lo ve el lote b (como ReadIn) con su versión; la de un cambio propio es el lsn de su
* registro, la misma que verán los demás si el lote confirma.
* @param b *Batch, id string
* @return []byte, uint64, bool, error
**/
func (s *Store) getVersionIn(b *Batch, id string) ([]byte, uint64, bool, error) {
	cur, found, err := s.readIn(b, id)
	return cur.value, uint64(cur.lsn), found, err
}

/**
* getVersionAt devuelve id como lo ve el snapshot v (como ReadAt) con su versión.
* @param v *Snapshot, id string
* @return []byte, uint64, bool, error
**/
func (s *Store) getVersionAt(v *Snapshot, id string) ([]byte, uint64, bool, error) {
	cur, found, err := s.readVersionAt(v, id)
	return cur.value, uint64(cur.lsn), found, err
}
