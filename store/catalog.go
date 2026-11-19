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
