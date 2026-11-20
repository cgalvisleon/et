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
// # Marca de sellado (tarea 90)
//
// Los registros de un segmento terminan antes de WAL_SEGMENT_SIZE − TrailerSize (TrailerSize = 32; Open exige
// WAL_SEGMENT_SIZE ≥ MaxRecordSize + TrailerSize). Al sellarlo (rotate: Switch, o un registro que no cabe), después
// del fsync de sus registros se escribe en sus últimos 32 bytes la marca de sellado y se hace otro fsync, así la marca
// nunca es durable antes que los registros que promete (un fsync fallido no deja un WAL que no abre):
//
//	0   4 crc     CRC-32C de [4, 32)
//	4   4 magic   0x4c414553 («SEAL»)
//	8   8 cero    donde un registro tendría totalLen: la marca nunca pasa por un registro
//	16  8 num     número del segmento (uno reciclado conserva la de su número viejo, que no vale para el nuevo)
//	24  8 end     desplazamiento siguiente a su último registro
//
// Al abrir, solo el último segmento sin sellar puede terminar en un registro roto (una cola de escrituras sin fsync).
// Si el final válido cae antes del end de la marca de su segmento, o queda un segmento sellado después del final, es
// corrupción (bit rot, una escritura perdida por el disco): Open devuelve ErrCorrupted sin cortar nada (checkSealed),
// salvo que el corte sea el objetivo de Options.StopLSN. Un segmento sin marca (escrito por una versión anterior, o con
// registros en esa zona) se trata como antes. Un segmento sellado que Open reabre como activo (el siguiente quedó
// vacío) sigue recibiendo registros después de su end: la marca vieja solo protege lo que ya promete.
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
// rehacer: 1–15 el motor (este paquete fija los formatos de 1–4; los de 5–9 están en store/doc.go, «Formatos en
// disco»), 16–31 btree, 32–39 overflow, 40–47 fsm.
//
//	KindCheckpoint  sin bloques; principal: redoSegment(8) redoOffset(8) redoLSN(8) nextXID(8) oldestActiveXID(8) time(8)
//	KindCommit      sin bloques; principal: time(8) storeCount(4) y storeCount × [fileNo(4) delta(8, con signo)]: lo
//	                que la transacción suma a Count de cada store; si el lote hizo DDL, una cola con los fileNo
//	                creados y borrados (store/doc.go)
//	KindAbort       sin bloques ni principal
//	KindPageImage   reservado: no se escribe (la primera marca de visibilidad de una hoja tras un checkpoint va
//	                como KindSetHints de btree, con la imagen de la hoja)
//	KindLeafWrite   (btree) una versión nueva en una hoja; es también el cambio lógico (id, xid y valor en línea o
//	                referencia a desborde) que leen Sync y WalSince. Con FlagCommit es una operación suelta que nace
//	                confirmada: no hay KindCommit aparte y su delta de Count va en FlagCountAdd/FlagCountSub
//	KindOverflowWrite (overflow) una página de desborde nueva (BlockWillInit) con su trozo del valor
//
// Los demás Kind de btree, overflow y fsm están en kind.go (KindSetFrozen, 23, es el frozenXID de la meta de un store;
// KindLeafVacuum también congela; KindMarkHalfDead, KindUnlinkPage y KindRecyclePage, 24 a 26, son el borrado de
// páginas del árbol).
//
// # Abrir y recuperar
//
// Open(fs, dir, opts, redo, fn) recorre el WAL desde redo (el punto de rehacer del control, o Start = segmento 1,
// desplazamiento 0, lsn 1 en un motor nuevo) llamando a fn con cada registro, comprueba que el final no corte un
// segmento sellado (ver «Marca de sellado»), corta la cola (el resto del último segmento válido y los segmentos
// siguientes enteros: un registro viejo nunca confirmado podría quedar justo donde termina uno nuevo) y deja el WAL
// listo para escribir a continuación. NewReader da un Reader para recorrer sin
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
// # Modo respaldo y retención
//
// ForceImages (el modo respaldo del motor, tarea 44, como forcePageWrites) cuenta los respaldos en curso: mientras
// haya alguno, Reserve decide la imagen de página entera contra el mayor entre el punto de rehacer y el inicio del
// respaldo más nuevo. Hold(seg) retiene los segmentos desde seg: Recycle no los renombra ni los borra hasta soltarla.
// Reservation.End da la posición siguiente al registro (hasta dónde copiar el WAL). KindBackupEnd (9) es el fin de un
// respaldo: sin bloques, principal redoLSN(8) checkpointLSN(8) time(8).
//
// # Archivo continuo
//
// Para el archivo continuo del motor (tarea 66; la retención es Hold): ActiveSegment (atómico) es el segmento activo,
// y todo segmento menor que existe está sellado y no cambia. Switch sella el activo aunque no esté lleno (como
// pg_switch_wal; con el activo vacío no hace nada): es una rotación, así que con preasignación puede dar ErrNoRoom
// sin tocar nada. Scan sigue de un segmento sellado así al siguiente igual que de uno lleno (el primer registro del
// siguiente tiene el lsn contiguo). SegmentInfo(num) da de un segmento sellado el lsn de su primer registro y la
// posición siguiente al último (End.Offset = bytes válidos): el final sale de lo anotado al sellarlo o al recorrerlo en
// Open (el mapa sizes, que ahora guarda una Position por segmento) y, para un segmento anterior al punto de rehacer de
// Open, de recorrerlo una vez sin writeMu. Recycle borra las anotaciones de los segmentos reciclados. Segments lista
// los números del directorio y SegmentName da el nombre de un archivo.
//
// Options.StopLSN (el objetivo de una recuperación a un punto en el tiempo, recovery_target_lsn): Open recorre hasta el
// registro de ese lsn inclusive y lo que sigue se corta como una cola inválida (cutTail).
//
// # Escribir un registro
//
// Ver store/doc.go («El orden de un cambio»). En resumen: quien cambia una página la tiene bloqueada en exclusivo,
// arma el Record (con la imagen previa si page.LSN < RedoPoint().LSN), llama a Reserve —que bajo writeMu vuelve a
// decidir la página entera contra el punto de rehacer vigente y devuelve ErrNeedImage si falta una, toma el lsn y el
// lugar, y llama a underLock—, aplica el cambio con ese lsn, suelta la página y llama a Write.
//
// # Réplica
//
// Para la replicación del motor (tarea 46; stream.go): Read recorre desde una posición exacta hasta un lsn y un tope
// de bytes y entrega cada registro con su posición y sus bytes tal cual; Locate ubica un lsn (el segmento cuyo primer
// registro es ≤ lsn, recorrido hasta él) y SegmentOf solo su segmento (lee el primer registro de cada uno); los dos dan
// ErrRemoved si ya se recicló, igual que Scan desde un segmento que no está. En la réplica, Append escribe un registro
// ya armado en la misma posición que en el primario (el final del activo, o el principio del siguiente: rota), así el
// WAL es una copia byte a byte; un lsn o una posición que no siguen dan ErrOutOfOrder sin escribir nada. SetRedo fija
// el punto de rehacer sin escribir (los restartpoints) y Decode lee un registro entero. Los recorridos reusan sus
// búferes (sync.Pool).
package wal
