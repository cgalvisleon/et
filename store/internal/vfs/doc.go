// Package vfs es la frontera entre el motor y el sistema de archivos: todo acceso a disco de smgr, control, wal y
// xact pasa por estas interfaces, para que las pruebas de modelo cambien el disco real por crashfs, que pierde lo que
// no tuvo fsync.
//
// Qué hace: define FS y File. Qué no hace: no sabe de páginas, registros ni formatos. No depende de ningún otro
// paquete del motor.
//
// Implementaciones: la real es OS(), sobre el sistema operativo, con Sync durable de verdad (F_FULLFSYNC en macOS,
// fdatasync en Linux), fsync de directorios y flock para Lock; fuera de unix no hay fsync de directorios ni flock
// (SyncDir no hace nada y Lock solo crea el archivo). La simulada, que pierde lo que no tuvo fsync, es crashfs.New().
// Flusher (opcional; FlushFile la usa o cae a Sync) lleva lo escrito al dispositivo y trae sus errores sin vaciar la
// caché del disco: el fsync simple en macOS (~30 µs contra ~4 ms de F_FULLFSYNC), fdatasync en Linux. No hace nada
// durable: smgr lo usa al cerrar un descriptor con escrituras sin fsync (tarea 98), y el checkpoint sincroniza de
// verdad.
//
// Disco lleno: IsNoSpace reconoce el disco lleno del sistema operativo (ENOSPC, EDQUOT) y el simulado (ErrNoSpace);
// el motor lo usa para decidir si un fallo es «disco lleno» (ver store/doc.go, «Disco lleno o lento»).
//
// Lo que no hace: no usa F_NOCACHE en macOS (la caché del sistema se suma a la del motor; a confirmar si hace falta,
// pediría una marca nueva en OpenFile) ni mmap.
package vfs
