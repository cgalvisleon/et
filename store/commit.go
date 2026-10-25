package store

/**
* commit: Ejecuta fn bajo writeMu y, si escribió, espera a que su registro esté en disco.
* El fsync se hace fuera de writeMu, así que las escrituras siguientes entran al log mientras
* tanto y un solo fsync las cubre a todas (group commit). Lo escrito es visible (índice, Sync)
* desde que se suelta writeMu; quien escribe retorna cuando ya es durable. Con wait false
* (escritura diferida) retorna sin esperar y quien llama debe usar waitAll antes de confirmar.
* @param wait bool, fn func() (bool, error) (true si escribió en el log)
* @return bool, error
**/
func (s *FileStore) commit(wait bool, fn func() (bool, error)) (bool, error) {
	ok, seq, err := s.commitLocked(fn)
	if err != nil || !ok || !wait || !s.SyncOnWrite {
		return ok, err
	}

	return ok, s.waitDurable(seq)
}

/**
* commitLocked: Ejecuta fn con writeMu tomado y retorna el número de registro escrito.
* El unlock va en defer: un hook de Sync que entra en pánico no deja writeMu tomado.
* @param fn func() (bool, error)
* @return bool, uint64, error
**/
func (s *FileStore) commitLocked(fn func() (bool, error)) (bool, uint64, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	ok, err := fn()
	return ok, s.appended.count(), err
}

/**
* waitAll: Espera a que todo lo escrito hasta ahora (también lo diferido) esté en disco.
* Sin SyncOnWrite no espera. Retorna el primer error de fsync, si lo hubo.
* @return error
**/
func (s *FileStore) waitAll() error {
	if !s.SyncOnWrite {
		return nil
	}

	return s.waitDurable(s.appended.count())
}

/**
* waitDurable: Espera a que el registro seq esté en disco. Si no hay un fsync en curso, el que
* llega hace uno (líder) que cubre todo lo escrito hasta ese momento y despierta a los demás.
* @param seq uint64
* @return error
**/
func (s *FileStore) waitDurable(seq uint64) error {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()

	for s.synced < seq && s.syncErr == nil {
		if s.syncing {
			s.syncCond.Wait()
			continue
		}

		s.syncing = true
		s.syncMu.Unlock()
		target, err := s.syncActive()
		s.syncMu.Lock()
		s.syncing = false
		if err != nil {
			s.syncErr = err
		} else if target > s.synced {
			s.synced = target
		}
		s.syncCond.Broadcast()
	}

	return s.syncErr
}

/**
* syncActive: Hace fsync del segmento activo y retorna hasta qué registro quedó en disco.
* Bajo indexMu el activo no cambia: todo lo escrito hasta target está en él o en un segmento
* ya sellado (la rotación y la compactación hacen fsync antes de cambiar el activo). El segmento
* se fija con acquire para que compactación o Close no cierren el archivo durante el fsync.
* @return uint64, error
**/
func (s *FileStore) syncActive() (uint64, error) {
	s.indexMu.RLock()
	seg := s.active
	target := s.appended.count()
	seg.acquire()
	s.indexMu.RUnlock()
	defer seg.release()

	// Retirado = Close lo selló (fsync) antes de retirarlo; el archivo puede estar cerrado.
	if seg.retired.Load() {
		return target, nil
	}

	// file.Sync y no flush: flush lee readOnly, que la rotación cambia bajo writeMu.
	if err := seg.file.Sync(); err != nil {
		return 0, err
	}

	return target, nil
}
