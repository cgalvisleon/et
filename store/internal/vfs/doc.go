// Package vfs es la frontera entre el motor y el sistema de archivos: todo acceso a disco de smgr, control, wal y
// xact pasa por estas interfaces, para que las pruebas de modelo cambien el disco real por crashfs, que pierde lo que
// no tuvo fsync.
//
// Qué hace: define FS y File. Qué no hace: no sabe de páginas, registros ni formatos. No depende de ningún otro
// paquete del motor.
//
// Implementaciones (ola 1, agente de crashfs): la real, sobre el sistema operativo —con Sync durable de verdad:
// F_FULLFSYNC en macOS, fsync en Linux— y la simulada, en crashfs.
package vfs
