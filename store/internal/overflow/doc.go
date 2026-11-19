// Package overflow guarda los valores que no caben en línea en una hoja en cadenas de páginas de desborde, en la
// línea de TOAST (pregunta 22), en el fork principal del mismo store.
//
// # Página de desborde
//
// Encabezado de página normal; el trozo del valor va desde page.HeaderSize hasta pd_lower (pd_lower = 32 + largo
// del trozo; pd_upper = pd_special = page.Size - page.OpaqueSize); sin punteros a ítems. Espacio especial común con
// flags = page.OpaqueOverflow, next = la siguiente página de la cadena (0 en la última), prev y level en 0. Cada
// página guarda hasta ChunkSize = 8.144 bytes. La entrada de hoja guarda el largo total y el primer bloque (Ref):
// leer es seguir la cadena hasta juntar el largo.
//
// # Ciclo de vida
//
// Cada cadena pertenece a exactamente una versión: se escribe antes de la entrada de hoja que la apunta, con la
// misma transacción (un KindOverflowWrite con BlockWillInit por página: el delta es el trozo, y es también el cambio
// lógico que junta un decodificador), y la limpieza la libera (fsm.Free por página) al quitar esa versión, también
// si su transacción abortó. No hay reutilización entre versiones: Update con el mismo valor no escribe. Sin
// compresión (fase posterior).
//
// Qué no hace: no decide cuándo un valor va a desborde (eso lo dice btree.MaxEntrySize), ni la visibilidad. Depende
// de page, smgr, buffer, fsm y wal por sus interfaces.
//
// Implementación: ola 3.
package overflow
