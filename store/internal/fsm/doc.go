// Package fsm lleva el espacio libre de cada store y su mapa de visibilidad.
//
// Qué hace:
//
//   - Lista de páginas libres del fork principal, encadenada desde la página meta (campo freeHead de la meta, ver
//     btree) por el campo next del espacio especial de cada página libre (marca page.OpaqueFree). Allocate saca la
//     primera (KindFreePop) o, si la lista está vacía, extiende el archivo; Free la devuelve (KindFreePush). En la
//     fase 1 solo se liberan páginas de desborde (las hojas vacías esperan al borrado de páginas del árbol, fase 2).
//     Como cada store tiene sus archivos, no hay una lista global ni un punto caliente entre stores.
//   - Mapa de visibilidad (fork smgr.ForkVM, fase 1B/2, pregunta 44): VMBitsPerBlock = 2 bits por bloque del fork
//     principal —VMAllVisible (todas sus versiones visibles para todos) y VMAllFrozen (todas congeladas)—. Una
//     página del mapa es un encabezado de página (pd_special = page.Size) seguido del mapa desde el byte
//     page.HeaderSize: VMBlocksPerPage = 32.640 bloques por página. Cambiar una hoja borra sus bits (KindVMClear,
//     en el mismo registro de la hoja cuando se puede); la limpieza los pone (KindVMSet).
//
// Qué no hace: no busca espacio dentro de una hoja (eso es del árbol: pd_lower/pd_upper). Depende de page, smgr,
// buffer (Pool, Handle) y wal (Log) por sus interfaces.
//
// Implementación: ola 2.
package fsm
