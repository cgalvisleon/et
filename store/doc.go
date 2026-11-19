// Package store es un motor de almacenamiento de pares (id string, data []byte) para construir bases de datos sobre
// él (F-23 de la Mesa de trabajo), con la arquitectura de almacenamiento de PostgreSQL como referencia: árbol B+ en
// disco por páginas de 8 KB, caché de páginas de tamaño fijo, WAL (registro previo de escritura) por página con
// páginas enteras tras cada checkpoint, MVCC (control de concurrencia multiversión) con transacciones de xid de 64
// bits y un archivo de estados, checkpoints repartidos y recuperación desde el último checkpoint.
//
// Este documento es la referencia del diseño nuevo (propuesta features/propuestas/et-store-arquitectura-postgres.md,
// con las 52 decisiones de F-23). Las piezas viven en paquetes internos (store/internal/...), cada uno con su doc.go,
// que detalla sus formatos con una tabla comentada: si una tabla de un paquete y la lista de este documento difieren,
// es un error de uno de los dos.
//
// # Estado del código
//
// IMPORTANTE: la implementación que hoy tiene este paquete (store.go, ops.go, segment.go, record.go, pending.go,
// index.go, readlock.go, dir.go, fs_*.go, catalog.go) es del DISEÑO ANTERIOR: un log append-only de segmentos con
// todas las claves en un índice ordenado en RAM, que se rehace al abrir recorriendo el log. No cumple las dos reglas
// de abajo y se reemplaza en la ola 4 por el motor descrito aquí. Hasta entonces sigue compilando y sus pruebas
// sirven de especificación del comportamiento de la API; su diseño está escrito en legacy.go. Lo que pasa de ella al motor nuevo es el log como base del
// WAL (store/internal/wal): lsn contiguos, reserva de lsn y lugar bajo writeMu, escritura en paralelo, published
// (avanza solo sobre lsn contiguos), group commit, CRC-32C, recuperación con corte en el primer registro inválido o
// fuera de orden, fsync del directorio y el LOCK.
//
// Los paquetes internos de la ola 0 solo tienen formatos, tipos e interfaces; sus implementaciones llegan por olas:
// 1 page, smgr, control, wal, xact, crashfs (y vfs); 2 buffer, fsm; 3 btree, overflow; 4 checkpoints, recuperación,
// el motor y la API de la fase 1A; 5 MVCC, transacciones, esperas, limpieza.
//
// # Las dos reglas
//
//  1. Nada del árbol en RAM más allá de una caché de páginas de tamaño fijo: ninguna estructura en memoria crece con
//     el número de claves. Lo que crece con otra cosa está acotado: la caché del archivo de estados (fija), las
//     transacciones en curso y su tabla de esperas (por concurrencia), el catálogo, los contadores de Count y los
//     descriptores de archivos (por stores abiertos; los descriptores en una caché fija).
//  2. Open no crece con el tamaño de los datos: lee el control y el catálogo y reproduce solo el WAL escrito desde el
//     punto de rehacer del último checkpoint, cuyo volumen acota MAX_WAL_SIZE. La caché empieza vacía.
//
// # Directorio del motor
//
//	<dir>/
//	  LOCK                    bloqueo exclusivo del directorio (flock): un motor por directorio
//	  control                 archivo de control: dos copias de 512 B (store/internal/control)
//	  catalog                 catálogo de stores: nombre → número de archivo, página meta, contador (control)
//	  data/<fileNo>.0001…     páginas de 8 KB del árbol de cada store, archivos de 1 GB (smgr; %010d.%04d)
//	  data/<fileNo>_vm.0001…  mapa de visibilidad de cada store (fase 1B/2)
//	  xact/0001…              archivo de estados de las transacciones (xact)
//	  wal/0000000001.seg…     segmentos del WAL de WAL_SEGMENT_SIZE (wal)
//
// Un motor es un directorio, con una caché y un WAL para todos sus stores; un proceso puede abrir varios motores
// (pregunta 32). Cada store es un árbol propio con sus archivos, como el relfilenode de PostgreSQL (pregunta 43):
// vaciarlo o borrarlo es cambiar el catálogo y borrar archivos. Crear, vaciar y borrar un store pasan por el WAL
// (pregunta 56): el registro va primero, después el catálogo y los archivos, así una réplica (WalSince) los ve y la
// recuperación los rehace en orden con los demás cambios. Los números de archivo no se reutilizan.
//
// # Paquetes internos
//
//	paquete   qué hace                                                   depende de
//	vfs       interfaz del sistema de archivos (real y simulado)         —
//	crashfs   disco en memoria que pierde lo que no tuvo fsync           vfs
//	page      formato de la página de 8 KB; LSN, XID, BlockNumber        —
//	smgr      archivos de datos por store y fork, fsync, descriptores    page, vfs
//	wal       registros, reserva, group commit, punto de rehacer, Scan   page, smgr, vfs
//	control   archivo de control y catálogo                              page, smgr, wal (Position), vfs
//	xact      xid, archivo de estados, en curso, snapshots, esperas      page, wal (Flusher), vfs
//	buffer    caché de páginas: pins, bloqueos, clock-sweep, bgwriter    page, smgr, wal (Flusher)
//	fsm       lista de páginas libres y mapa de visibilidad              page, smgr, buffer, wal
//	overflow  cadenas de páginas de desborde                             page, smgr, buffer, fsm, wal
//	btree     árbol B+ Lehman-Yao con versiones                          page, smgr, buffer, fsm, overflow, xact, wal
//	store     motor: checkpoints, recuperación, transacciones, API       todos
//
// El grafo es acíclico, y cada paquete usa a los demás por sus interfaces (smgr.Manager, wal.Log, wal.Flusher,
// buffer.Pool, fsm.FreeList, overflow.Chains, xact.StatusFile, xact.Manager, btree.Tree, control.ControlFile,
// control.CatalogFile, vfs.FS): cada uno se prueba con dobles de los otros.
//
// # Formatos en disco (little-endian; versión del formato 1)
//
// Todos los enteros son little-endian: el orden nativo de arm64 y amd64, que permite poner las marcas de visibilidad
// de una entrada con un OR atómico sobre su uint32. Las sumas son CRC-32C (Castagnoli).
//
// Control, cada copia de 512 B (copia A en el byte 0, B en el 512; se escribe la que no tiene la última versión; al
// leer gana la válida con mayor seq): magic(4) version(4) seq(8) state(4) pageSize(4) blocksPerFile(4)
// reservado(4) checkpointTime(8) redoSegment(8) redoOffset(8) redoLSN(8) checkpointLSN(8) nextXID(8)
// oldestActiveXID(8) frozenXID(8, fase 2) slots(8 × [nombre(24) restartLSN(8)], fase 3) reservado(156) crc(4).
//
// Catálogo (se reescribe entero: catalog.tmp, fsync, rename, fsync del directorio): encabezado magic(4) version(4)
// nextFileNo(4) entryCount(4) countLSN(8) reservado(8); por store fileNo(4) metaBlock(4) count(8) flags(4)
// nameLen(2) reservado(2) name(nameLen, 1 a 255 B); al final crc(4) de todo lo anterior.
//
// Página (8.192 B): encabezado de 32 B pd_lsn(8) pd_checksum(2) pd_flags(2) pd_lower(2) pd_upper(2) pd_special(2)
// pd_pagesize_version(2) pd_prune_xid(8) reservado(4); punteros a ítems de 4 B (lp_off 15 bits, lp_flags 2, lp_len
// 15) desde el byte 32; ítems alineados a 8 desde el final hacia arriba; espacio especial de 16 B al final:
// prev(4) next(4) level(4) flags(2) cycleID(2). Suma: CRC-32C de la página con pd_checksum en cero y el número de
// bloque, plegado a 16 bits. Una página en ceros es nueva y válida.
//
// Entrada de hoja (una versión): flags(4) idLen(2) reservado(2) lsn(8) xid(8) valueLen(4) = 28 B, más el id, más el
// valor en línea o la referencia a desborde (primer bloque, 4). Clave: (id ascendente, lsn descendente). Va en línea
// si 28 + id + valor ≤ 2.032 B; si no, a desborde. Con el puntero y la alineación, una versión ocupa ~32 B + id +
// valor.
//
// Pivote (páginas internas y claves altas): keyLen(2) flags(2) downlink(4) lsn(8) = 16 B, más el prefijo truncado
// del id.
//
// Página meta (bloque 0 de cada store, desde el byte 32): magic(4) version(4) root(4) level(4) fastRoot(4)
// fastLevel(4) freeHead(4) reservado(4) frozenXID(8) reservado(8) = 48 B.
//
// Página de desborde: encabezado, el trozo del valor desde el byte 32 hasta pd_lower (hasta 8.144 B), espacio
// especial con flags = desborde y next = la siguiente de la cadena.
//
// Archivo de estados: páginas de 8.192 B sin encabezado, 2 bits por xid (0 en curso, 1 confirmada, 2 abortada),
// 4.096 páginas por archivo. El lsn de confirmación por grupo de 32 xids vive en memoria con cada página en caché.
//
// Registro del WAL: crc(4) lsn(8) totalLen(4) xid(8) kind(1) flags(1) blockCount(1) reservado(1) mainLen(4) = 32 B;
// por bloque fileNo(4) fork(1) flags(1) reservado(2) block(4) imageLen(2) deltaLen(2) = 16 B, la imagen y el delta;
// al final los datos principales. Tipos (wal/kind.go): checkpoint, confirmación (con lo que suma a Count de cada
// store), aborto, página entera (marcas de visibilidad), versión nueva en hoja (que es también el cambio lógico:
// id, xid y valor; con FlagCommit es una operación suelta que nace confirmada), partición, enlace en el padre, raíz
// nueva, store nuevo, limpieza de hoja, página de desborde, lista de libres y mapa de visibilidad. La imagen de página
// entera es la página antes del cambio, sin el hueco entre pd_lower y pd_upper, comprimida con S2 si achica.
//
// # Límites
//
//   - id: de 1 a 2.000 B (pregunta 37). Nombre de un store: de 1 a 255 B.
//   - valor: hasta MaxValueLen = 64 MiB. Un valor grande se guarda en páginas de desborde y cada una es un registro
//     del WAL de ~8 KB, así que el tamaño del valor no cambia el de los registros: ningún registro pasa de 64 KiB y
//     ninguno cruza segmentos del WAL (si no cabe en el activo, se rota). El tope existe porque el valor se recibe y
//     se devuelve entero en memoria y pasa entero por el WAL: 64 MiB es un octavo del volumen que dispara un
//     checkpoint, así que un solo Put no se acerca a la cota de Open.
//   - transacciones y snapshots: 10 minutos (TRANSACTION_TIMEOUT); un snapshot puede pedir el suyo o ninguno.
//
// # El orden de un cambio
//
// Una escritura (Put, Insert… sueltas o en un lote) hace, como PostgreSQL:
//
//  1. Si el valor no cabe en línea, escribir antes su cadena de desborde (con el xid de la transacción).
//  2. Bajar por el árbol con Lehman-Yao (una página bloqueada en compartido a la vez; si la clave pasa la clave alta,
//     seguir a la derecha) y bloquear la hoja en exclusivo. Si la hoja tiene OpaqueIncompleteSplit, terminar antes
//     esa partición.
//  3. Mirar las versiones del id en la hoja (siguiendo a la derecha si siguen en la vecina). Si la más nueva es de
//     otra transacción en curso: soltar la hoja, esperar a ese xid (tabla de esperas, hasta LOCK_TIMEOUT; si vence,
//     error de conflicto) y volver al paso 2. Nunca se espera con una página bloqueada.
//  4. Comprobar la condición (Insert, Update, Delete) contra la última versión que cuenta: en una operación suelta,
//     la última confirmada (Read Committed, pregunta 40); en un lote, la propia o la que ve su snapshot, y si la última
//     confirmada no es visible para el snapshot, error de serialización (pregunta 34), también para Put y Remove.
//  5. Si no hay espacio, partir la hoja (dos pasos del WAL) y volver al paso 2.
//  6. Armar el registro: el delta (la entrada sin lsn todavía) y, si el pd_lsn de la hoja es menor que el punto de
//     rehacer vigente, la imagen de la hoja ANTES del cambio, comprimida fuera de writeMu.
//  7. wal.Log.Reserve, bajo writeMu: vuelve a decidir la página entera contra el punto de rehacer vigente (si un
//     checkpoint empezó y falta la imagen, devuelve ErrNeedImage: se agrega con la hoja todavía bloqueada y se
//     repite), asigna el lsn y el lugar, y en una operación suelta suma su delta al contador de Count del store.
//  8. Aplicar el cambio en la página con la clave (id, lsn), poner pd_lsn = lsn, marcarla sucia, soltar la hoja.
//  9. Escribir el registro en su lugar del WAL (en paralelo con otros) y publicarlo.
//  10. Confirmar: una operación suelta ya lleva FlagCommit; un lote escribe su KindCommit (con la suma de Count por
//     store, sumada bajo writeMu al reservar). Con SYNC_ON_WRITE esperar a que el WAL sea durable hasta ese lsn
//     (group commit). Después anotar confirmada en el archivo de estados con ese lsn y quitar la transacción de las en
//     curso (despierta a quien la espera).
//
// La página va al disco después, en un checkpoint, al expulsarla de la caché o por el escritor de fondo, y nunca
// antes de que el WAL sea durable hasta su pd_lsn («WAL antes que la página»): las escrituras del usuario solo
// esperan al WAL, que es secuencial.
//
// Update compara el valor antes de bloquear la hoja (leerlo de desborde con la hoja bloqueada frenaría a todos) y,
// con la hoja bloqueada, comprueba que la versión que comparó sigue siendo la última (por su lsn); si no, repite.
//
// # Orden de los bloqueos
//
//   - Entre páginas: de izquierda a derecha, de hija a padre; nunca al revés.
//   - Página (bloqueo de contenido) → writeMu → pubMu; writeMu → syncMu; syncMu → pubMu. Nunca writeMu → página.
//   - Nunca se espera a otra transacción (xact.Manager.Wait) ni a un fsync con una página bloqueada. Un pin sí se
//     puede tener mientras tanto.
//   - Los locks del mapa de la caché, de la caché del archivo de estados y del catálogo en memoria son hojas: no se
//     toma otro con ellos tomados. El catálogo nunca se toca con una página bloqueada.
//   - Marcas de visibilidad: con bloqueo compartido, por OR atómico; la escritura de la página al disco calcula la
//     suma sobre una copia. La marca de confirmada solo se pone si el WAL es durable hasta el lsn de confirmación del
//     grupo de su xid (pregunta 49). La primera marca en una página tras un checkpoint se anota como KindPageImage.
//
// # Checkpoint
//
// Se dispara cada CHECKPOINT_TIMEOUT (5 min) o cuando el WAL desde el punto de rehacer pasa MAX_WAL_SIZE / (1 + 0,9)
// (con 1 GB, ~539 MiB): así lo que hay que reproducir tras una caída ronda MAX_WAL_SIZE (pregunta 28). Pasos:
//
//  1. wal.Log.BeginCheckpoint, bajo writeMu: el punto de rehacer pasa a la posición del próximo registro; ahí se toman
//     nextXID, el xid vivo más viejo y los contadores de Count (consistentes con ese lsn, porque se suman bajo
//     writeMu). Desde ese momento la primera escritura de cada página lleva su imagen entera.
//  2. Escribir las páginas sucias, repartido en el 90 % del intervalo (CHECKPOINT_COMPLETION_TARGET = 0,9), cada una
//     tras el WAL durable hasta su pd_lsn; las páginas del archivo de estados; smgr.SyncAll y fsync de xact.
//  3. Guardar el catálogo con los contadores y countLSN = el lsn del punto de rehacer.
//  4. Escribir el registro KindCheckpoint y esperar su fsync.
//  5. Escribir el control (seq + 1) con el punto de rehacer, checkpointLSN, nextXID y el xid vivo más viejo.
//  6. Reciclar los segmentos del WAL anteriores al punto de rehacer (respetando las ranuras, fase 3).
//
// Un corte entre 3 y 5 deja el control con el punto de rehacer anterior y el catálogo con el nuevo: la recuperación
// suma a los contadores solo las confirmaciones con lsn ≥ countLSN, así que no cuenta dos veces. Close hace un
// checkpoint final con state = cerrado bien.
//
// # Recuperación (Open)
//
//  1. Tomar LOCK; leer el control (la copia válida más nueva) y el catálogo.
//  2. Recorrer el WAL desde el punto de rehacer (wal.Reader.Scan) y debe aparecer el registro del checkpoint. Por cada
//     registro: crear, vaciar y borrar un store se rehacen sobre el catálogo (idempotentes: un store que ya existe o
//     ya no está se deja como está); los bloques de un store borrado se saltan; con imagen, se restaura sin
//     condiciones y se aplica el delta; sin imagen, se aplica solo si el pd_lsn de la página es menor que el lsn del
//     registro (idempotente), con la función de rehacer del dueño del Kind; confirmaciones y abortos van al archivo de
//     estados, y una confirmación con lsn ≥ countLSN suma a los contadores.
//  3. Cortar el WAL en su primer registro inválido. Un hueco seguido de registros confirmados es corrupción: Open
//     falla.
//  4. nextXID = el mayor entre el del control y el mayor xid visto + 1. Los xid menores en estado «en curso» se leen
//     como abortados (eran transacciones en curso al caerse), sin marcarlos uno por uno.
//  5. Checkpoint de fin de recuperación y state = en producción. Las particiones incompletas las termina la próxima
//     inserción que las encuentre.
//
// # Fallos de escritura
//
// Si falla una escritura o un fsync del WAL, de una página, del archivo de estados o del control, el motor queda
// inutilizable (como el PANIC de PostgreSQL): toda operación devuelve el error, no se escriben más páginas sucias y
// el fsync no se reintenta (tras un fsync fallido el sistema operativo puede haber descartado las páginas: reintentar
// daría un éxito falso). Se recupera cerrando y abriendo: la recuperación reconstruye desde el WAL durable.
//
// # Variables de entorno
//
//	variable                 por omisión  qué fija
//	STORE_CACHE_SIZE         128          tamaño de la caché de páginas, en MB (pregunta 21)
//	WAL_SEGMENT_SIZE         128          tamaño de cada segmento del WAL, en MB, mínimo 1 (pregunta 27)
//	SYNC_ON_WRITE            true         una confirmación espera el fsync del WAL
//	CHECKPOINT_TIMEOUT       300          segundos entre checkpoints (pregunta 28)
//	MAX_WAL_SIZE             1024         MB de WAL que acotan la recuperación; dispara a MAX_WAL_SIZE / 1,9
//	TRANSACTION_TIMEOUT      600          segundos que puede durar un lote o un snapshot (pregunta 35)
//	LOCK_TIMEOUT             5            segundos que una escritura espera a otra transacción (pregunta 33)
//	MAX_SLOT_WAL_KEEP_SIZE   4096         MB de WAL que puede retener una ranura (fase 3, pregunta 48)
//
// Los archivos de datos son de 1 GB fijos y CHECKPOINT_COMPLETION_TARGET es la constante 0,9, sin variable.
// RELSEG_SIZE (del diseño anterior) deja de usarse.
//
// # API pública (ola 4 en adelante)
//
// Reemplaza a la actual (Open(dir) (*Store, error) del diseño anterior). Las fases dicen cuándo llega cada método.
//
//	// Motor
//	func Open(dir string) (*Engine, error)                    // 1A: control + WAL desde el último checkpoint
//	func (e *Engine) Close() error                            // 1A: checkpoint final
//	func (e *Engine) Store(name string) (*Store, error)       // 1A: abre el store, o lo crea si no existe
//	func (e *Engine) DropStore(name string) error             // 1A
//	func (e *Engine) Stores() []string                        // 1A
//	func (e *Engine) NewBatch() *Batch                        // 1B: una transacción
//	func (b *Batch) Commit() error                            // 1B
//	func (b *Batch) Rollback() error                          // 1B
//	func (e *Engine) Snapshot() *Snapshot                     // 1B: tiempo máximo TRANSACTION_TIMEOUT
//	func (e *Engine) SnapshotFor(timeout time.Duration) *Snapshot // 1B: tiempo propio; 0 = sin tiempo máximo
//	func (v *Snapshot) Close() error                          // 1B
//	func (e *Engine) Checkpoint() error                       // 1A
//	func (e *Engine) Vacuum() error                           // 1B
//	func (e *Engine) Verify() error                           // 3
//	func (e *Engine) Repair() error                           // 3
//	func (e *Engine) Stats() et.Json                          // 3 (también OnStats)
//
//	// Store: operaciones sueltas (cada una es su propia transacción y confirma al volver)
//	func (s *Store) Put(id string, data []byte) error
//	func (s *Store) Remove(id string) error
//	func (s *Store) Insert(id string, data []byte) (bool, error)
//	func (s *Store) Update(id string, data []byte) (bool, error)
//	func (s *Store) Delete(id string) (bool, error)
//	func (s *Store) Read(id string) ([]byte, bool, error)
//	func (s *Store) Exists(id string) (bool, error)          // 2
//	func (s *Store) Count() int64                            // 2
//	func (s *Store) Empty() error                            // 2
//
//	// Store: dentro de un lote
//	func (s *Store) PutIn(tx *Batch, id string, data []byte) error
//	func (s *Store) RemoveIn(tx *Batch, id string) error
//	func (s *Store) InsertIn(tx *Batch, id string, data []byte) (bool, error)
//	func (s *Store) UpdateIn(tx *Batch, id string, data []byte) (bool, error)
//	func (s *Store) DeleteIn(tx *Batch, id string) (bool, error)
//	func (s *Store) ReadIn(tx *Batch, id string) ([]byte, bool, error)
//	func (s *Store) ExistsIn(tx *Batch, id string) (bool, error)   // 2
//
//	// Store: sobre un snapshot
//	func (s *Store) ReadAt(v *Snapshot, id string) ([]byte, bool, error)
//	func (s *Store) ExistsAt(v *Snapshot, id string) (bool, error)  // 2
//	func (s *Store) ForEachAt(v *Snapshot, fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error // 2
//	func (s *Store) CountAt(v *Snapshot) (int64, error)              // 2
//
// Después, como API secundaria: Keys, ForEach, GetVersion, Deferred/WaitDurable (fase 2) y Sync, WalSince,
// ApplyWalEntry (fase 3, por transacción confirmada y en orden de confirmación).
//
// Semántica:
//
//   - Put guarda siempre (crea o reemplaza) y Remove borra sin preguntar (si no hay versión viva no escribe nada);
//     Insert (solo si no existe), Update (solo si existe y el valor cambió) y Delete (solo si existe) devuelven si
//     escribieron (pregunta 1).
//   - Una operación suelta es Read Committed: si otra transacción escribe el mismo id, espera a que termine (hasta
//     LOCK_TIMEOUT) y aplica su condición sobre lo último confirmado (pregunta 40). Read suelto no toma snapshot:
//     devuelve la versión más nueva confirmada (pregunta 41). Count devuelve el contador del store: los ids vivos
//     según lo último confirmado (pregunta 45).
//   - Un lote es una transacción con aislamiento por snapshot (como Repeatable Read, pregunta 34): ve una foto fija
//     tomada al crearlo más sus propios cambios. Escribir un id que otra transacción confirmó después de esa foto
//     devuelve ErrSerialization (también Put y Remove); esperar a otra más de LOCK_TIMEOUT devuelve ErrConflict; pasar
//     TRANSACTION_TIMEOUT aborta el lote (ErrTimeout). Tras un error de esos el lote solo admite Rollback. Commit es
//     todo o nada, también tras una caída, con un fsync para todo el lote.
//   - Un snapshot es una foto fija de todos los stores del motor para leer y recorrer, sin bloquear a los escritores.
//     Mientras siga abierto (igual que un lote), la limpieza no quita las versiones que podría ver.
//   - Lo devuelto por Read es del llamador. Los callbacks de recorridos no pueden escribir en el store.
//   - Si el motor quedó inutilizable (ver «Fallos de escritura»), todo devuelve ErrFailed hasta reabrirlo.
package store
