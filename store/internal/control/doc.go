// Package control lee y escribe los dos archivos pequeños que dicen cómo abrir el motor: el archivo de control
// (dónde empieza la recuperación) y el catálogo de stores (nombre → archivo de datos y contador).
//
// Qué hace: el formato y la escritura segura de ambos. Qué no hace: no decide cuándo hay checkpoint ni qué valores
// van (eso es del motor), no lee el WAL. Depende de vfs, page (LSN, XID), smgr (FileNo) y wal (Position, solo el tipo).
//
// # Archivo de control («control», 1.024 bytes)
//
// Dos copias de CopySize = 512 bytes: la A en el byte 0 y la B en el 512. Cada escritura va a la copia seq % 2 (la
// que no tiene la última versión) y termina con Sync: si un corte rompe la que se escribía, queda la otra entera.
// Al leer gana la copia válida (magia, versión y CRC correctos) con mayor seq; si ninguna es válida, Open falla.
// Cada copia, little-endian:
//
//	off tam campo
//	  0   4 magic            ControlMagic ("ETCT")
//	  4   4 version          FormatVersion
//	  8   8 seq              número de escritura del control; crece de a uno
//	 16   4 state            StateShutdown, StateInProduction, StateInRecovery
//	 20   4 pageSize         page.Size; otro valor = Open falla
//	 24   4 blocksPerFile    smgr.BlocksPerFile; otro valor = Open falla
//	 28   4 reservado        en cero
//	 32   8 checkpointTime   hora del último checkpoint (Unix en nanosegundos)
//	 40   8 redoSegment      punto de rehacer: número del segmento del WAL…
//	 48   8 redoOffset       …desplazamiento dentro de él…
//	 56   8 redoLSN          …y lsn del primer registro a reproducir
//	 64   8 checkpointLSN    lsn del registro de checkpoint (≥ redoLSN); la recuperación debe encontrarlo
//	 72   8 nextXID          el siguiente xid a asignar al tomar el punto de rehacer
//	 80   8 oldestActiveXID  el xid vivo más viejo en ese momento (0 si ninguno)
//	 88   8 frozenXID        reservado (fase 2, congelado): los xid menores se tratan como confirmados y el archivo de estados se trunca por debajo
//	 96 256 slots            reservado (fase 3): SlotCount ranuras de retención del WAL de SlotSize bytes: nombre (24, relleno con ceros) + restartLSN (8)
//	352 156 reservado        en cero
//	508   4 crc              CRC-32C de los bytes 0–507
//
// # Catálogo («catalog»)
//
// Se reescribe entero a «catalog.tmp», Sync, rename a «catalog» y SyncDir del motor: al crear o borrar un store y en
// cada checkpoint. Crece con la cantidad de stores, no con las claves. Little-endian:
//
//	encabezado (CatalogHeaderSize = 32)
//	  0   4 magic       CatalogMagic ("ETCG")
//	  4   4 version     FormatVersion
//	  8   4 nextFileNo  el próximo FileNo a asignar (nunca se reutiliza uno)
//	 12   4 entryCount
//	 16   8 countLSN    los contadores cuentan todas las confirmaciones con lsn < countLSN
//	 24   8 reservado
//	entradas, entryCount veces (CatalogEntrySize = 24 + nombre)
//	  0   4 fileNo
//	  4   4 metaBlock   bloque de la página meta del store (siempre 0 en la fase 1)
//	  8   8 count       ids vivos confirmados (Count) a countLSN
//	 16   4 flags       reservado (fase 2: store con mapa de visibilidad, etc.)
//	 20   2 nameLen     1 a MaxNameLen
//	 22   2 reservado
//	 24   n name
//	cola
//	  0   4 crc         CRC-32C de todo lo anterior
package control
