// Package wal es el registro previo de escritura (write-ahead log) del motor: el log de segmentos del diseño anterior
// (lsn contiguos, reserva bajo writeMu, escritura en paralelo, published, group commit, CRC-32C, recuperación con
// corte) con registros nuevos: cambios por página, páginas enteras comprimidas, confirmaciones, abortos y
// checkpoints.
//
// Qué hace: formato de los registros; reservar lsn y lugar; escribir en paralelo y publicar en orden; group commit;
// el punto de rehacer y la decisión de página entera; leer el WAL desde una posición; cortar la cola inválida;
// reciclar segmentos; medir la latencia de cada fsync del group commit (SyncLatency, un metric.Histogram de tamaño
// fijo). Qué no hace: no aplica los registros a las páginas (cada dueño de un Kind tiene su función de rehacer; el
// motor las reparte), no sabe de transacciones más allá del xid de cada registro. Depende de vfs, page, metric y
// smgr (FileNo, Fork).
//
// # Segmentos
//
// wal/<n:%010d>.seg, n desde 1, de hasta WAL_SEGMENT_SIZE bytes (128 MB por omisión). Un registro nunca cruza
// segmentos: si no cabe en lo que queda del activo, se sella el activo (sus escrituras terminadas y fsync) y se
// sigue en el siguiente. Un segmento reciclado (renombrado a un número futuro tras un checkpoint) conserva registros
// viejos con CRC válido: el chequeo de lsn contiguo los descarta.
//
// # Registro (little-endian)
//
//	encabezado (HeaderSize = 32)
//	  0   4 crc        CRC-32C de los bytes [4, totalLen)
//	  4   8 lsn        exactamente el lsn del registro anterior + 1 (el primero del WAL, 1)
//	 12   4 totalLen   largo total, ≤ MaxRecordSize
//	 16   8 xid        transacción del registro; 0 si no tiene (checkpoint, página entera por marcas)
//	 24   1 kind       Kind
//	 25   1 flags      FlagCommit, FlagCountAdd, FlagCountSub
//	 26   1 blockCount referencias a bloques que siguen, ≤ MaxBlocks
//	 27   1 reservado
//	 28   4 mainLen    largo de los datos principales
//	referencias a bloques, blockCount veces (BlockHeaderSize = 16 + imagen + delta)
//	  0   4 fileNo     store (smgr.FileNo)
//	  4   1 fork       smgr.Fork
//	  5   1 flags      BlockHasImage, BlockImageCompressed, BlockWillInit
//	  6   2 reservado
//	  8   4 block      page.BlockNumber
//	 12   2 imageLen   largo de la imagen guardada (0 sin imagen)
//	 14   2 deltaLen   largo del cambio para este bloque
//	 16   … imagen, después delta
//	datos principales (mainLen bytes)
//
// Imagen de página entera: la página ANTES del cambio del registro, sin el hueco [pd_lower, pd_upper) (el
// encabezado de 32 bytes siempre queda antes del hueco, así que al restaurar se leen pd_lower y pd_upper de la propia
// imagen) y, con BlockImageCompressed, comprimida en bloque S2 (github.com/klauspost/compress/s2; si no achica, va sin
// comprimir). Al rehacer, la imagen se restaura sin condiciones y después se aplica el delta, si lo hay.
//
// # Tipos de registro
//
// Los Kind se reparten por dueño, que define el formato de su delta y de sus datos principales y su función de
// rehacer: 1–15 el motor (este paquete fija sus formatos), 16–31 btree, 32–39 overflow, 40–47 fsm.
//
//	KindCheckpoint  sin bloques; principal: redoSegment(8) redoOffset(8) redoLSN(8) nextXID(8) oldestActiveXID(8) time(8)
//	KindCommit      sin bloques; principal: time(8) storeCount(4) y storeCount × [fileNo(4) delta(8, con signo)]: lo
//	                que la transacción suma a Count de cada store
//	KindAbort       sin bloques ni principal
//	KindPageImage   solo imágenes (la primera marca de visibilidad de una página tras un checkpoint, como
//	                wal_log_hints)
//	KindLeafWrite   (btree) una versión nueva en una hoja; es también el cambio lógico (id, xid y valor en línea o
//	                referencia a desborde) que leen Sync y WalSince. Con FlagCommit es una operación suelta que nace
//	                confirmada: no hay KindCommit aparte y su delta de Count va en FlagCountAdd/FlagCountSub
//	KindOverflowWrite (overflow) una página de desborde nueva (BlockWillInit) con su trozo del valor
//
// Los demás Kind de btree, overflow y fsm están en kind.go (KindSetFrozen, 23, es el frozenXID de la meta de un store;
// KindLeafVacuum también congela).
//
// # Abrir y recuperar
//
// Open(fs, dir, opts, redo, fn) recorre el WAL desde redo (el punto de rehacer del control, o Start = segmento 1,
// desplazamiento 0, lsn 1 en un motor nuevo) llamando a fn con cada registro, corta la cola (el resto del último
// segmento válido y los segmentos siguientes enteros: un registro viejo nunca confirmado podría quedar justo donde
// termina uno nuevo) y deja el WAL listo para escribir a continuación. NewReader da un Reader para recorrer sin
// escribir. Options.Spares es cuántos segmentos reciclados se guardan; Recycle borra el resto. MakeImage y
// RestoreImage (o BlockRef.SetImage y BlockRef.RestoreImage) arman y restauran las imágenes de página entera.
//
// # Preasignación
//
// Con Options.Preallocate (el motor la usa siempre) cada segmento se escribe entero con ceros y fsync antes de
// recibir registros, como wal_init_zero de PostgreSQL: así escribir un registro reservado sobrescribe bloques que ya
// existen y no pide lugar nuevo, y el disco lleno sale antes de empezar una operación. segment.allocated es hasta
// dónde está escrito el activo (Open lo deja en el final válido: el resto se rellena después) y nextReady dice si el
// siguiente ya existe entero. Room, que el motor llama al entrar cada escritura, exige medio segmento preasignado
// entre el resto del activo y el siguiente: si falta, rellena el activo y prepara el siguiente (un repuesto reciclado
// o uno nuevo, creado con O_EXCL y fsync del directorio) bajo writeMu. Si no hay lugar devuelve ErrNoRoom (con el
// error de disco lleno dentro) sin dejar el WAL inutilizable; el próximo intento sigue desde el largo del archivo. Una
// reserva que necesita rotar o rellenar y no puede devuelve también ErrNoRoom, sin asignar lsn. Un segmento a medio
// rellenar empieza en ceros (o con registros viejos de lsn menor): Scan lo toma por el final del WAL. Lo que deja el
// WAL inutilizable sigue igual: un fsync fallido (también el de la preasignación), una escritura de un registro
// reservado que falla, o un fallo de escritura que no es de disco lleno. Written da los bytes reservados desde Open,
// sin candados (el motor mide con él el WAL sin reciclar).
//
// # Escribir un registro
//
// Ver store/doc.go («El orden de un cambio»). En resumen: quien cambia una página la tiene bloqueada en exclusivo,
// arma el Record (con la imagen previa si page.LSN < RedoPoint().LSN), llama a Reserve —que bajo writeMu vuelve a
// decidir la página entera contra el punto de rehacer vigente y devuelve ErrNeedImage si falta una, toma el lsn y el
// lugar, y llama a underLock—, aplica el cambio con ese lsn, suelta la página y llama a Write.
package wal
