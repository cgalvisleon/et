package store

// La API pública del paquete. La implementación es privada, en los demás archivos (ver doc.go).

// ---- Apertura y cierre ----

/**
* Open: Abre el store del directorio dir, o lo crea si no existe. Lee RELSEG_SIZE (MB por segmento, 128 por omisión)
* y SYNC_ON_WRITE (true por omisión: cada escritura espera a estar en disco). Falla si otro proceso lo tiene abierto.
* @param dir string
* @return *Store, error
**/
func Open(dir string) (*Store, error) {
	return open(dir, envOptions())
}

/**
* Close: Espera las escrituras en curso, las deja en disco y cierra el store. Llamarlo de nuevo no hace nada.
* @return error
**/
func (s *Store) Close() error {
	return s.close()
}

// ---- Escritura ----

/**
* Put: Guarda data para id, exista o no: no lee ni compara lo que había (escritura ciega, el camino rápido). Con
* SYNC_ON_WRITE vuelve cuando la escritura está en disco. Es seguro llamarlo desde varias goroutines a la vez.
* @param id string (1 a 65.535 bytes), data []byte (hasta 1 GiB; nil o vacío se guarda como vacío)
* @return error
**/
func (s *Store) Put(id string, data []byte) error {
	return s.put(id, data)
}

/**
* Remove: Borra id, exista o no: no lo busca (borrado ciego, el camino rápido); repetirlo da lo mismo. Con
* SYNC_ON_WRITE vuelve cuando el borrado está en disco.
* @param id string
* @return error
**/
func (s *Store) Remove(id string) error {
	return s.remove(id)
}

/**
* Insert: Guarda data solo si id no existe. Entre varios Insert del mismo id a la vez, exactamente uno lo guarda.
* @param id string, data []byte
* @return bool (true si lo guardó), error
**/
func (s *Store) Insert(id string, data []byte) (bool, error) {
	return s.insert(id, data)
}

/**
* Update: Guarda data solo si id existe y su valor es distinto del actual.
* @param id string, data []byte
* @return bool (true si lo cambió), error
**/
func (s *Store) Update(id string, data []byte) (bool, error) {
	return s.update(id, data)
}

/**
* Delete: Borra id solo si existe.
* @param id string
* @return bool (true si lo borró), error
**/
func (s *Store) Delete(id string) (bool, error) {
	return s.delete(id)
}

// ---- Lectura ----

/**
* Read: El valor de id. No espera a los escritores (no toma su lock): ve lo último que terminó de escribirse. Verifica
* el crc del registro. El slice devuelto es del llamador: puede modificarlo.
* @param id string
* @return []byte, bool (false si id no existe), error
**/
func (s *Store) Read(id string) ([]byte, bool, error) {
	return s.read(id)
}
