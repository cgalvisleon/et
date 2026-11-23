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
// wal/<n:%010d>.seg, n desde 1, de hasta WAL_SEGMENT_SIZE bytes (128 MB por omisión): encabezado (32 B), registros
// desde el byte 32 y, si está sellado, la marca de sellado en sus últimos 32 B. Un registro nunca cruza segmentos:
// si no cabe en lo que queda del activo, se sella el activo (sus escrituras terminadas y fsync) y se sigue en el
// siguiente. Un segmento reciclado (renombrado a un número futuro tras un checkpoint) conserva su encabezado viejo
// (otro número) y registros viejos con CRC válido: el encabezado y el chequeo de lsn contiguo los descartan.
//
// # Encabezado del segmento (pregunta 76, tarea 54)
//
// Los primeros SegmentHeaderSize = 32 bytes de cada segmento, como el long page header de PostgreSQL: un segmento
// suelto (archivado, en un respaldo, recibido por una réplica) dice su versión sin el control.
//
//	0   4 crc      CRC-32C de [4, 32)
//	4   4 magic    0x47455357 («WSEG»)
//	8   4 version  FormatVersion (= control.FormatVersion, que la toma de aquí: un solo número); 2 desde la tarea
//	               132 (formato 2, en desarrollo); se lee de MinFormatVersion (1) a FormatVersion (Readable)
//	12  4 cero     reservado
//	16  8 num      número del segmento (uno reciclado conserva el de su número viejo, que no vale para el nuevo)
//	24  8 first    lsn de su primer registro (el que tiene o el que va a tener)
//
// Lo mínimo útil: magia y versión (fijas para siempre en los bytes 4 y 8: un binario de otra versión las reconoce
// antes del crc), el número (descarta un repuesto reciclado y un archivo renombrado) y el lsn inicial (Locate,
// SegmentOf y SegmentInfo lo leen sin recorrer registros; el paso al siguiente segmento exige first = el lsn que
// sigue, o es un hueco). No lleva el tamaño de segmento (un segmento sellado mide eso, y la réplica ya exige las
// mismas posiciones) ni un identificador del sistema (el control no tiene uno; agregarlo sería una versión nueva).
//
// Quién lo escribe: rotate, en el siguiente, con fsync, entre el fsync de los registros del activo y su marca de
// sellado (+1 fsync por rotación). Ese orden da las dos reglas: un encabezado durable nunca promete un lsn que el
// activo no tiene en disco (al revés, un corte entre los dos fsync deja un hueco falso: TestReplicaCrash en la VM lo
// encontró; TestRotateSyncOrder lo fija), y un segmento sellado siempre tiene el siguiente con su encabezado durable.
// Y Open, en el activo sin registros cuyo
// encabezado falta (el primero de un motor nuevo, o uno cortado antes de que fuera durable: ensureHeader). La
// réplica rota igual al recibir el primer registro de un segmento, con el mismo número y el mismo lsn: su encabezado
// es idéntico al del primario (salvo la versión si los binarios son de formatos distintos: ver «Formato 2» abajo) y
// las posiciones (el primer registro en el byte 32) coinciden byte a byte.
//
// Formato 2 (tarea 132): los registros no cambian de forma (el 2 solo agrega EntryCompressed a la entrada de hoja,
// que viaja en el delta de KindLeafWrite), así que un segmento del 1 se recorre igual. Options.Upgrade (el motor lo
// pone salvo en una réplica) hace que Open deje atrás un segmento activo del 1 antes de devolver el WAL (upgrade: lo
// sella y rota si tiene registros; si no, le reescribe el encabezado con fsync): ningún registro que escriba después
// un primario cae en un segmento que dice 1. Una réplica no lo hace: sigue en el segmento del 1 de su copia hasta que
// el flujo rota (rotate escribe siempre FormatVersion, así que frente a un primario del 1 su encabezado difiere del
// de él en la versión). Al promoverla, el motor llama a StartSegment antes de escribir nada (tarea 138): con registros
// sella el activo y rota (como Switch); sin registros, si es de una versión anterior, le reescribe el encabezado. Así
// una réplica promovida tampoco escribe en un segmento que dice 1.
//
// Quién lo valida: el recorrido (scan: Open, la recuperación, OpenAt, Read de WalSince y de la réplica) en el
// segmento donde empieza (checkStartHeader: de este número, de esta versión, first ≤ from; ausente solo si from es
// el byte 32) y en cada paso al siguiente (sin encabezado, roto o reciclado = el final del WAL; first mayor = hueco,
// ErrCorrupted; otra versión = ErrFormatVersion); checkSealed (un final justo en el fin de un segmento sellado cuyo
// siguiente existe es ErrCorrupted: su encabezado era durable); Locate, SegmentOf y SegmentInfo (el archivo
// continuo); y OtherVersion, que checkFormat del motor llama antes de tocar nada (ErrFormatVersion del motor) con los
// segmentos que Open va a leer: desde el del punto de rehacer del control (los que lee la recuperación, más los
// repuestos) y hacia atrás hasta el lsn más viejo que retiene una ranura (lo que lee SegmentOf); los que retiene solo
// el archivo continuo atrasado los valida quien los lee.
// ParseSegmentHeader lo lee desde fuera del paquete.
//
// # Marca de sellado (tarea 90)
//
// Los registros de un segmento terminan antes de WAL_SEGMENT_SIZE − TrailerSize (TrailerSize = 32; Open exige
// WAL_SEGMENT_SIZE ≥ SegmentHeaderSize + MaxRecordSize + TrailerSize). Al sellarlo (rotate: Switch, o un registro que no cabe), después
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
// salvo que el corte sea el objetivo de Options.StopLSN. También lo es un final justo en el end de un segmento
// sellado cuyo siguiente existe: el encabezado del siguiente se hizo durable antes de sellar (ver «Encabezado del
// segmento»), así que el recorrido tenía que seguir en él. Un segmento sin marca (con registros en esa zona) se trata
// como uno sin sellar.
//
// # Registro (little-endian)
//
//	encabezado (HeaderSize = 32)
//	  0   4 crc        CRC-32C de los bytes [4, totalLen)
//	  4   8 lsn        exactamente el lsn del registro anterior + 1 (el primero del WAL, 1)
//	 12   4 totalLen   largo total, ≤ MaxRecordSize
//	 16   8 xid        transacción del registro; 0 si no tiene (checkpoint, página entera por marcas)
//	 24   1 kind       Kind
//	 25   1 flags      FlagCommit (bit 0), FlagCountAdd (1), FlagCountSub (2), FlagOrigin (3: un KindCommit cuyos
//	                   datos terminan con origin(2), Batch.SetOrigin; tarea 83)
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
// desplazamiento 32 —tras el encabezado—, lsn 1 en un motor nuevo) llamando a fn con cada registro, comprueba que el final no corte un
// segmento sellado (ver «Marca de sellado»), corta la cola (el resto del último segmento válido y los segmentos
// siguientes enteros: un registro viejo nunca confirmado podría quedar justo donde termina uno nuevo) y deja el WAL
// listo para escribir a continuación. NewReader da un Reader para recorrer sin
// escribir.
//
// Tras un cierre limpio no corta (Options.CleanEnd, tarea 109): el motor pasa el lsn siguiente a su checkpoint de
// Close y, si el recorrido termina justo ahí (sin StopLSN) y ningún segmento desde el final mide más que SegmentSize,
// Open no trunca el activo ni los siguientes y toma allocated del largo del activo. Así el activo y los repuestos
// siguen preasignados: sin un truncado + fsync por segmento al abrir ni volver a llenarlos de ceros. Lo que queda
// después del final son ceros o registros viejos de lsn menor (Close esperó toda reserva; un corte anterior ya se
// cortó al abrir), que el recorrido descarta igual que en un repuesto reciclado, y un repuesto tiene el encabezado y la
// marca de sellado de otro número. Quien lo pasa garantiza que nada se escribió después del Close: el motor anota su
// control en StateInProduction antes de escribir (store/doc.go, «Recuperación»). Si el final no coincide, se corta como
// siempre. Options.Spares es cuántos segmentos reciclados se guardan; Recycle borra el resto. MakeImage y
// RestoreImage (o BlockRef.SetImage y BlockRef.RestoreImage) arman y restauran las imágenes de página entera.
//
// # Preasignación
//
// Con Options.Preallocate (el motor la usa siempre) cada segmento se escribe entero con ceros y fsync antes de
// recibir registros, como wal_init_zero de PostgreSQL: así escribir un registro reservado sobrescribe bloques que ya
// existen y no pide lugar nuevo, y el disco lleno sale antes de empezar una operación. segment.allocated es hasta
// dónde está escrito el activo (Open lo deja en el final válido, o en el largo del archivo tras un cierre limpio con
// CleanEnd: el resto se rellena después) y nextReady dice si el
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
// pg_switch_wal; con el activo sin registros, solo su encabezado, no hace nada): es una rotación, así que con preasignación puede dar ErrNoRoom
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
// # Búfer del WAL y group commit (tarea 100)
//
// Write sella el registro (lsn y crc) y lo deja en pending (por lsn, bajo pubMu). Los pendientes se escriben por
// tandas (drain, bajo drainMu, una a la vez): la tanda contigua que sigue a published, del mismo segmento y de hasta
// maxRun = 1 MB, sale con un solo pwrite y published avanza hasta su último lsn. Con Options.Buffer = 0 cada Write
// hace drain enseguida; con Buffer (WAL_BUFFERS, 1 MB en el motor) Write deja el registro pendiente salvo que lo
// pendiente llegue a Buffer o que alguien espere un lsn mayor que published (wanted). drain para en el primer hueco
// (una reserva que su dueño no escribió todavía): nunca espera a otra reserva; el dueño del hueco, al escribirlo, ve
// wanted y sigue la tanda. waitPublished(lsn) anota wanted, hace drain y espera en pubCond. published es entonces
// «escrito en el archivo» (Published, Stats.LSN): Read, Locate y los lectores del motor nunca pasan Durable.
// WaitDurable(lsn): waitPublished y después el fsync del activo, que hace un solo líder a la vez (leadMu, syncing): el
// fsync cubre todo lo publicado al empezar, y los que esperaban (un canal que se cierra al terminar, así se despiertan
// todos a la vez) vuelven si su lsn quedó cubierto o el primero en llegar es el líder del siguiente. El sellado de un
// segmento (rotate, Close) espera con waitPublished a que todo lo reservado esté escrito antes de su fsync. Orden:
// writeMu → drainMu → pubMu; leadMu es una hoja; syncMu lo toman el líder, rotate y Close.
//
// # Réplica
//
// Para la replicación del motor (tarea 46; stream.go): Read recorre desde una posición exacta hasta un lsn y un tope
// de bytes y entrega cada registro con su posición y sus bytes tal cual; Locate ubica un lsn (el segmento cuyo primer
// registro es ≤ lsn, recorrido hasta él) y SegmentOf solo su segmento (lee el primer registro de cada uno); los dos dan
// ErrRemoved si ya se recicló, igual que Scan desde un segmento que no está. En la réplica, Append escribe un registro
// ya armado en la misma posición que en el primario (el final del activo, o el byte 32 del siguiente: rota y escribe
// el mismo encabezado), así el WAL es una copia byte a byte; un lsn o una posición que no siguen dan ErrOutOfOrder sin escribir nada. SetRedo fija
// el punto de rehacer sin escribir (los restartpoints) y Decode lee un registro entero. Los recorridos reusan sus
// búferes (sync.Pool).
package wal
