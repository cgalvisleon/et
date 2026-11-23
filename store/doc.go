// Package store es un motor de almacenamiento de pares (id string, data []byte) para construir bases de datos sobre
// él (F-23 de la Mesa de trabajo), con la arquitectura de almacenamiento de PostgreSQL como referencia: árbol B+ en
// disco por páginas de 8 KB, caché de páginas de tamaño fijo, WAL (registro previo de escritura) por página con
// páginas enteras tras cada checkpoint, MVCC (control de concurrencia multiversión) con transacciones de xid de 64
// bits y un archivo de estados, checkpoints repartidos y recuperación desde el último checkpoint.
//
// Este documento es la referencia del diseño (propuesta features/propuestas/et-store-arquitectura-postgres.md, con las
// decisiones y preguntas de F-23 en features/data/et-store-motor.json; los números de «pregunta n» y «decisión n» son
// los de ahí). Las piezas viven en paquetes internos (store/internal/...), cada uno con su doc.go, que detalla sus
// formatos con una tabla comentada: si una tabla de un paquete y este documento difieren, es un error de uno de los
// dos. Si este documento y el código difieren, manda el código. El historial (qué cambió, cuándo y cómo se probó) está
// en doc/CHANGELOG.md, no acá. Cómo operarlo (dimensionar, respaldar, replicar, verificar, qué hacer ante cada
// error) está en store/MANUAL.md, el manual de operación.
//
// # Estado del código
//
// Hechas: la fase 1A (motor, operaciones sueltas, checkpoints, recuperación), la 1B (lotes, snapshots, MVCC, esperas,
// limpieza), la 2 (recorridos, Exists, Empty, escritura diferida, versiones, congelado y truncado del archivo de
// estados, la prueba del diseño recursivo) y, de la 3, Verify y Repair con cmd/storecheck, Stats y OnStats, Pressure y
// DirUsage, el disco lleno o lento, el DDL dentro de un lote, el respaldo en caliente con su restauración y el archivo
// continuo del WAL con la recuperación a un punto en el tiempo, y la replicación (tarea 46: réplica física, ranuras de
// retención y flujo lógico de cambios) y el borrado de páginas del árbol (tarea 70: las hojas vacías y los padres que
// se quedan sin hijos vuelven a la lista de libres, ver «Borrado de páginas del árbol»).
//
// El código del diseño anterior (un log append-only con todas las claves en un índice en RAM) se borró el 2026-11-19;
// lo que servía de él (lsn contiguos, reserva bajo writeMu, escritura en paralelo, published, group commit, CRC-32C,
// recuperación con corte en el primer registro inválido) vive en store/internal/wal.
//
// Disco: vfs.OS() es el sistema de archivos real (fsync con F_FULLFSYNC en macOS, fdatasync en Linux; Lock devuelve
// vfs.ErrLocked si otro tiene el LOCK, y el error público del motor lo envuelve); crashfs.New() es el disco simulado
// de las pruebas con cortes (pierde, reordena o rompe por sectores lo que no tuvo fsync; un fsync fallido olvida lo
// pendiente, como Linux).
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
// La caché de páginas vive fuera del heap de Go (mmap anónimo en darwin, linux y los BSD; heap en windows), como la
// memoria compartida de PostgreSQL: el GC no la cuenta, así el RSS de un motor ronda su tamaño más unos MB, no el
// doble (decisión 62; ver store/internal/buffer/doc.go). Close la devuelve al SO; un motor sin cerrar la retiene.
//
// # Directorio del motor
//
//	<dir>/
//	  LOCK                    bloqueo exclusivo del directorio (flock): un motor por directorio
//	  control                 archivo de control: dos copias de 512 B (store/internal/control)
//	  catalog                 catálogo de stores: nombre → número de archivo, página meta, contador (control)
//	  data/<fileNo>.0001…     páginas de 8 KB del árbol de cada store, archivos de 1 GB (smgr; %010d.%04d)
//	  data/<fileNo>_vm.0001…  mapa de visibilidad de cada store: lo pone la limpieza, lo borra toda escritura
//	  xact/0001…              archivo de estados de las transacciones (xact); el checkpoint borra los de xids congelados
//	  wal/0000000001.seg…     segmentos del WAL de WAL_SEGMENT_SIZE (wal)
//	  backup_label            solo en una copia de Backup hasta restaurarla: el inicio y el fin del respaldo
//	  archive_status          con WAL_ARCHIVE=on: el último segmento archivado y la última pérdida de continuidad
//	  slots                   las ranuras de retención del WAL (si se creó alguna)
//	  replslot/*.spill        derrames del búfer de reordenamiento de un ChangeReader abierto (se borran solos)
//
// Un motor es un directorio, con una caché y un WAL para todos sus stores; un proceso puede abrir varios motores
// (pregunta 32). Cada store es un árbol propio con sus archivos, como el relfilenode de PostgreSQL (pregunta 43):
// vaciarlo o borrarlo es cambiar el catálogo y borrar archivos. Crear, vaciar y borrar un store pasan por el WAL
// (pregunta 56): el registro va primero, después el catálogo y los archivos, así una réplica (WalSince) los
// ve y la recuperación los rehace en orden con los demás cambios. Los números de archivo no se reutilizan.
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
//	btree     árbol B+ Lehman-Yao con versiones y borrado de páginas     page, smgr, buffer, fsm, overflow, xact, wal
//	metric    histograma de latencias log2 de tamaño fijo (Stats)        —
//	store     motor: checkpoints, recuperación, transacciones, API       todos
//
// El grafo es acíclico, y cada paquete usa a los demás por sus interfaces (smgr.Manager, wal.Log, wal.Flusher,
// buffer.Pool, fsm.FreeList, overflow.Chains, xact.StatusFile, xact.Manager, btree.Tree, control.ControlFile,
// control.CatalogFile, vfs.FS): cada uno se prueba con dobles de los otros.
//
// # Archivos del motor
//
//	archivo        qué tiene
//	catalog.go     la API pública: envolturas delgadas, por secciones
//	engine.go      Engine, open, close, enter/leave, enterRead, fail/failed, logRecord
//	config.go      variables de entorno y constantes (ver «Variables de entorno»)
//	msg.go         errores públicos y mensajes
//	store.go       Store y las operaciones sueltas: write, writeRound, writeLeaf, commit, update
//	counts.go      contadores vivos de Count y walProxy (el Flusher de la recuperación)
//	records.go     formato de KindCheckpoint, KindCreateStore(In), KindDropStore, KindEmptyStore y KindCommit
//	ddl.go         Store (abre o crea), DropStore, Stores, openTree
//	ddlbatch.go    storeIn, createIn, dropStoreIn, markDrop, finishDDL, applyDDL, undoDDL, pendingFiles
//	batch.go       Batch: newBatch, writeIn/writeInRound/batchLeaf, waitOther, readIn, putIn…updateIn, commit, rollback, aborto por tiempo
//	snapshot.go    Snapshot: snapshotFor, release, readAt
//	timers.go      Engine.timers: los temporizadores de TRANSACTION_TIMEOUT de lotes y snapshots, que Close detiene
//	scan.go        scanRange (rango [from, to), sentido, offset, limit), scanner, scanAt, keys, keysAfter, countAt,
//	               exists, existsIn, existsAt
//	empty.go       Store.empty y Engine.emptyFiles
//	durable.go     Deferred, syncCommit, noteCommit, waitDurable, walWriter
//	version.go     getVersion, getVersionIn, getVersionAt
//	checkpoint.go  checkpoint (pasos 1–7), pacer (reparto en el 90 %), checkpointer de fondo
//	recovery.go    recover: control, catálogo, piezas, rehacer cada Kind con su dueño, fin de recuperación
//	vacuum.go      Engine.vacuum, Store.vacuum (serializado por store), freeDead, la limpieza de fondo (vacuumer)
//	frozen.go      frozenXIDs (frozenXID de cada store y del control), engineFrozen, vacuumOptions, freezeStore
//	space.go       syncGuard, cleanNoSpace, writeError, admit, throttle, deferredBrake
//	pressure.go    PressureCause, pressure
//	usage.go       DiskUsage, dirUsage, Store.dirUsage
//	stats.go       Stats, Latency, engineMetrics, stats, onStats
//	report.go      Report, Finding, FindingKind (las clases de hallazgo), MaxFindings
//	verify.go      check (Verify y Repair), el control, el catálogo, los archivos de data/, checkStore, checkHandle
//	storecheck.go  storeCheck: el inventario de páginas de un store, el conteo, el mapa, el congelado, fixCount
//	backup.go      BackupInfo, backups (retenciones y borrados que esperan), unlinkData, backupRun (los pasos)
//	restore.go     el marcador backup_label, checkBackupWAL, removeBackupLabel, checkTarget (el objetivo de OpenAt)
//	format.go      checkFormat: la versión del formato de los archivos de la raíz antes de abrir (ErrFormatVersion)
//	compress.go    compression (STORE_COMPRESSION), encode, decompress, Engine.compressed: el formato 2 (a)
//	archive.go     archiveState, startArchive, PendingArchive, SetArchived, SwitchWAL, el tope, archive_status
//	walsince.go    WalReader (WalSince, Next): leer el WAL durable desde un lsn, del disco
//	replica.go     WalEntry, replicaState, startReplica, applyWal (conflictos con snapshots), restartpoint
//	slots.go       slotState, CreateSlot, DropSlot, AdvanceSlot, checkpointSlots (el tope), el archivo slots
//	decode.go      ChangeReader: búfer de reordenamiento por xid con derrame a disco, Next, Confirm
//
// # Formatos en disco (little-endian; versión del formato 2, congelado el 2026-11-22: ver «Versiones del formato» y «Formato 2»)
//
// Todos los enteros son little-endian: el orden nativo de arm64 y amd64, que permite poner las marcas de visibilidad
// de una entrada con un OR atómico sobre su uint32. Las sumas son CRC-32C (Castagnoli).
//
// Control, cada copia de 512 B (copia A en el byte 0, B en el 512; se escribe la que no tiene la última versión; al
// leer gana la válida con mayor seq): magic(4) version(4) seq(8) state(4) pageSize(4) blocksPerFile(4)
// reservado(4) checkpointTime(8) redoSegment(8) redoOffset(8) redoLSN(8) checkpointLSN(8) nextXID(8)
// oldestActiveXID(8) frozenXID(8) slots(8 × [nombre(24) restartLSN(8)], reservado y en cero: las ranuras viven en
// el archivo slots, que admite dos lsn y marcas por ranura) reservado(156) crc(4). frozenXID es el del motor: el menor de los stores al checkpoint; el archivo de estados se
// trunca por debajo.
//
// Catálogo (se reescribe entero: catalog.tmp, fsync, rename, fsync del directorio): encabezado magic(4) version(4)
// nextFileNo(4) entryCount(4) countLSN(8) reservado(8); por store fileNo(4) metaBlock(4) count(8) flags(4, bit 0
// FlagPendingCreate) nameLen(2) reservado(2) name(nameLen, 1 a 255 B); al final crc(4) de todo lo anterior.
//
// Página (8.192 B): encabezado de 32 B pd_lsn(8) pd_checksum(2) pd_flags(2) pd_lower(2) pd_upper(2) pd_special(2)
// pd_pagesize_version(2) pd_prune_xid(8) reservado(4); punteros a ítems de 4 B (lp_off 15 bits, lp_flags 2, lp_len
// 15) desde el byte 32; ítems alineados a 8 desde el final hacia arriba; espacio especial de 16 B al final:
// prev(4) next(4) level(4) flags(2) cycleID(2). Suma: CRC-32C de la página con pd_checksum en cero y el número de
// bloque, plegado a 16 bits. Una página en ceros es nueva y válida.
//
// Entrada de hoja de la forma 1 (una versión; hojas con LayoutVersion 1, las del formato 1): flags(4) idLen(2)
// reservado(2) lsn(8) xid(8) valueLen(4) = 28 B, más el id, más el valor en línea o la referencia a desborde (primer
// bloque, 4). Entrada de la forma 2 (hojas con LayoutVersion 2, las que escribe el formato 2; ver «Formato 2: hojas de
// la forma 2 (b y c)»): word0(4) = marcas (bits 0–7) | largo del sufijo (8–19) | largo en línea (20–31), lsn(8) y
// xid(8) salvo en una congelada sin desborde = 20 B (12 congelada), más el id sin el prefijo de la hoja, más el valor
// en línea o first(4) len(4). Clave: (id ascendente, lsn descendente). Va en línea si 28 + id + valor ≤ 2.032 B (la
// regla de la forma 1 en las dos); si no, a desborde. Con el puntero y la alineación, una versión de la forma 1 ocupa
// ~32 B + id + valor; una de la 2, ~24 B (16 congelada) + el resto del id + valor. flags: EntryTombstone, EntryOverflow, las marcas de visibilidad (EntryXidCommitted, EntryXidAborted),
// EntryFrozen (congelada: confirmada y visible para todo snapshot, sin consultar el archivo de estados; la pone la
// limpieza con WAL), EntryFreeing (versión muerta cuya cadena de desborde se está liberando) y, desde el formato 2,
// EntryCompressed (el valor guardado, en línea o en desborde, es method(1) rawLen(4) y los datos comprimidos;
// valueLen es lo guardado; ver «Formato 2»). La página de una hoja lleva page.FlagAllVisible mientras su bit del mapa
// de visibilidad puede estar puesto.
//
// Pivote (páginas internas y claves altas): keyLen(2) flags(2) downlink(4) lsn(8) = 16 B, más el prefijo truncado
// del id.
//
// Página meta (bloque 0 de cada store, desde el byte 32): magic(4) version(4) root(4) level(4) fastRoot(4)
// fastLevel(4) freeHead(4) deletedHead(4) frozenXID(8) flags(4) reservado(4) = 48 B; flags (formato 2; en una meta
// del 1, en cero) bit 0 MetaLeavesForm2: todas las hojas del store son de la forma 2. frozenXID es el del store: ninguna versión
// del store con un xid menor queda sin congelar (o sin quitar). fastRoot/fastLevel es la raíz rápida (btm_fastroot de
// nbtree): la página de más a la izquierda del nivel más bajo con una sola página, donde empiezan las bajadas.
// deletedHead es la cabeza de la pila de páginas del árbol borradas que esperan reciclarse (0 = vacía; era un campo
// reservado en cero, así que una meta anterior al borrado de páginas se lee con la pila vacía).
//
// Página borrada del árbol (tarea 70): espacio especial con flags = OpaqueDeleted (más OpaqueLeaf si era una hoja),
// prev y next los de sus hermanas al desengancharla (los lectores que todavía llegan siguen a la derecha) y su nivel;
// un solo ítem de 16 B: safeXID(8) nextDeleted(4) reservado(4). safeXID es el próximo xid al desengancharla;
// nextDeleted, la siguiente de la pila (0 = fin). Hoja medio muerta: la hoja vacía con su clave alta y la marca
// OpaqueHalfDead; el enlace (downlink) de su clave alta, que en una hoja viva es 0, anota la cima del subárbol que se
// borra con ella (0 = la hoja misma). Ver «Borrado de páginas del árbol».
//
// Página de desborde: encabezado, el trozo del valor desde el byte 32 hasta pd_lower (hasta 8.144 B), espacio
// especial con flags = desborde, next = la siguiente de la cadena y level = la posición de la página en la cadena
// (detecta orden cambiado, bucles y saltos a otra cadena); pd_prune_xid = xid de la versión dueña. Cada cadena es de
// una sola versión.
//
// Archivo de estados: páginas de 8.192 B sin encabezado, 2 bits por xid (0 en curso, 1 confirmada, 2 abortada),
// 4.096 páginas por archivo. El lsn de confirmación por grupo de 32 xids vive en memoria con cada página en caché. Los
// archivos enteros por debajo del frozenXID del motor se borran; un xid de esos se lee confirmado (y congelado).
//
// Registros del motor (1–15; datos principales, sin bloques; formatos en records.go, backup.go):
//
//	kind                  n  datos principales
//	KindCheckpoint        1  redoSegment(8) redoOffset(8) redoLSN(8) nextXID(8) oldestActiveXID(8) time(8)
//	KindCommit            2  time(8) storeCount(4), storeCount × [fileNo(4) delta(8, con signo)] y, solo si el lote
//	                         hizo DDL, la cola createCount(4) dropCount(4), createCount × fileNo(4), dropCount ×
//	                         fileNo(4) (sin cola = sin DDL: los registros viejos se leen igual); con wal.FlagOrigin
//	                         en las marcas del registro, origin(2) al final, después de la cola (Batch.SetOrigin,
//	                         tarea 83; nunca 0: sin origen no se escribe)
//	KindAbort             3  nada
//	KindPageImage         4  reservado: no se escribe (las marcas de visibilidad van como KindSetHints de btree)
//	KindCreateStore       5  fileNo(4) nameLen(2) name
//	KindDropStore         6  fileNo(4) nameLen(2) name: el número de archivo esperado
//	KindEmptyStore        7  oldFileNo(4) newFileNo(4) nameLen(2) name
//	KindCreateStoreIn     8  como KindCreateStore, con el xid del lote en el encabezado (pendiente hasta su KindCommit)
//	KindBackupEnd         9  redoLSN(8) checkpointLSN(8) time(8): el fin de un respaldo; no cambia nada al rehacerlo
//
// Registros de las piezas (formatos de sus deltas en sus doc.go): btree 16 KindLeafWrite (versión nueva en una hoja,
// que es también el cambio lógico: id, xid y valor; con FlagCommit es una operación suelta que nace confirmada), 17
// KindSplit, 18 KindInsertPivot, 19 KindNewRoot, 20 KindInitStore, 21 KindLeafVacuum (limpieza de hoja, que también
// congela), 22 KindSetHints (la primera marca de visibilidad de una hoja tras un checkpoint, con su imagen), 23
// KindSetFrozen (frozenXID de la meta de un store), 24 KindMarkHalfDead (hoja medio muerta y el enlace quitado del
// padre del subárbol), 25 KindUnlinkPage (página desenganchada de sus hermanas, borrada y apilada en la meta), 26
// KindRecyclePage (de la pila de borradas a la lista de libres), 27 KindLeafRewrite (formato 2: hoja reescrita en la
// forma 2, por la migración desde el 1 o un prefijo que se acorta); overflow 32 KindOverflowWrite; fsm 40 KindFreePush,
// 41 KindFreePop, 42 KindVMSet, 43 KindVMClear.
//
// Registro del WAL: crc(4) lsn(8) totalLen(4) xid(8) kind(1) flags(1) blockCount(1) reservado(1) mainLen(4) = 32 B
// (blockCount ≤ 5: el desenganche de una página toca cinco);
// por bloque fileNo(4) fork(1) flags(1) reservado(2) block(4) imageLen(2) deltaLen(2) = 16 B, la imagen y el delta;
// al final los datos principales. Flags del registro: FlagCommit, FlagCountAdd, FlagCountSub; de un bloque:
// BlockHasImage, BlockImageCompressed, BlockWillInit. La imagen de página entera es la página antes del cambio, sin el
// hueco entre pd_lower y pd_upper, comprimida con S2 si achica.
//
// Marcador backup_label (64 B, en la raíz, solo en una copia de Backup hasta restaurarla; backup_label.tmp, fsync,
// rename): magic(4) "ETBK" version(4) redoSegment(8) redoOffset(8) redoLSN(8) checkpointLSN(8) endLSN(8) time(8)
// reservado(4) crc(4); endLSN = 0 es «en curso».
//
// Estado del archivo continuo archive_status (32 B en la raíz, solo con WAL_ARCHIVE=on; archive_status.tmp, fsync,
// rename, fsync del directorio): magic(4) "ETAR" version(4) archived(8) gap(8) reservado(4) crc(4).
//
// Ranuras slots (en la raíz; slots.tmp, fsync, rename, fsync del directorio; se reescribe entero al crear, borrar o
// invalidar una ranura y en cada checkpoint con avances): magic(4) "ETSL" version(4) count(4) reservado(4); por ranura
// nameLen(1) flags(1: bit 0 lógica, bit 1 invalidada) reservado(6) restartLSN(8) confirmedLSN(8) name(nameLen, 1 a
// 64 B); al final crc(4) de todo lo anterior. Hasta 16 ranuras.
//
// Derrame del búfer de reordenamiento replslot/<nombre en hex>-<n>.spill (no es durable: se borra al entregar o
// descartar la transacción, al cerrar el lector y al abrir otro de la misma ranura): una tras otra, largo(4) + kind(1)
// file(4) a(4) b(4) idLen(2) dataLen(4) id data, con kind 1 trozo de desborde (a = bloque, b = siguiente), 2 versión
// (a = 1 lápida, b = primer bloque de desborde y data su largo, o data el valor), 3 store creado en el lote (id).
//
// Los magic se guardan como uint32 little-endian del texto leído de izquierda a derecha (0x4554424B = "ETBK"), como
// los de control, catálogo y página meta.
//
// # Versiones del formato
//
// Tarea 54 de F-23. Cada archivo que se lee sin depender de otro lleva su versión; los demás los gobierna la del
// control, porque nunca se leen sin él (o sin el backup_label de una copia, que también la lleva):
//
//	archivo                       dónde está la versión                       si no coincide
//	control (cada copia)          version(4), byte 4; FormatVersion = 2       ErrFormatVersion (una copia con su crc
//	                                                                          válido basta, aunque la otra valga)
//	catalog                       version(4), byte 4                          ErrFormatVersion
//	backup_label                  version(4), byte 4                          ErrFormatVersion
//	archive_status                version(4), byte 4                          ErrFormatVersion
//	slots                         version(4), byte 4                          ErrFormatVersion
//	página (todas las clases)     pd_pagesize_version = 8.192 | 1, y 8.192 |  daño de la página (ErrCorrupt al leerla,
//	                              2 en una hoja de la forma 2 (la versión de  hallazgo de Verify); una hoja de la
//	                              página no es la del formato: una hoja del   forma 1 en un store cuya meta dice
//	                              formato 1 sigue en 1 hasta reescribirse)    MetaLeavesForm2 también es daño
//	página meta de un store       version(4) = 1, tras su magic               daño de la página
//	segmento del WAL              encabezado (32 B): version(4), byte 8       ErrFormatVersion (checkFormat); roto:
//	                              (pregunta 76; wal/doc.go)                   ErrCorrupt al recorrerlo
//	marca de sellado del segmento sin versión: la del encabezado del segmento —
//	archivo de estados (xact/)    sin versión propia: la del control          —
//	derrame replslot/             sin versión: no es durable, se borra        —
//
// Open (checkFormat, antes de leer backup_label, el control para recuperar, el catálogo o el WAL, y de cambiar nada)
// compara la versión del control, del catálogo, del encabezado de cada segmento del WAL desde el punto de rehacer del
// control y hacia atrás hasta el lsn más viejo que retiene una ranura (wal.OtherVersion: lo que Open lee; los que
// retiene solo el archivo continuo atrasado los valida quien los lee), de
// backup_label, de archive_status y de slots con la que lee el binario: una mayor es de un binario más nuevo (MSG_FORMAT_NEWER: actualizar et) y una menor, de un formato anterior
// sin migración (MSG_FORMAT_OLDER); las dos dan ErrFormatVersion y el directorio queda como estaba (TestOpenFormatVersion).
// Desde el formato 2 «la que lee» es un rango (wal.Readable: de wal.MinFormatVersion = 1 a FormatVersion = 2): un
// archivo del 1 se lee y lo que se escribe sale en 2 (ver «Formato 2: migración desde el formato 1 (e)»).
// Un archivo sin la magia de su formato no es «otra versión» sino daño, y lo informa quien lo lee. La versión de las
// páginas y de la meta no se mira al abrir: el control las cubre, y una página de otra versión con un control de esta
// es daño. La réplica física copia el WAL byte a byte: primario y réplica tienen que leer la misma versión, y un
// segmento que llega por el flujo, del archivo continuo o en una copia dice la suya en su encabezado. Un encabezado
// de segmento roto (crc, número, lsn inicial) es daño: ErrCorrupt (wal.ErrCorrupted traducido), en Open, OpenAt,
// OpenReplica, WalSince y el archivo continuo (SegmentInfo). Una sola versión para todo: control.FormatVersion =
// wal.FormatVersion.
//
// Prueba de ida y vuelta (format_test.go): TestFormatRoundTrip arma un directorio de referencia (hojas, páginas
// internas, meta, desborde y una cadena liberada, páginas borradas y libres, versiones congeladas y el mapa de
// visibilidad, un store vaciado y uno borrado, un lote con origen, una ranura física y una lógica, un segmento
// sellado y archivado, y una copia de Backup con backup_label), comprueba que cada archivo de la raíz se decodifica
// y se vuelve a escribir igual byte a byte (control.DecodeCopy/EncodeCopy, DecodeCatalog/EncodeCatalog,
// decodeSlots/encodeSlots, decodeArchiveStatus, decodeLabel), la suma de cada página de data/, y que el motor y la
// copia abren con el contenido esperado, Verify limpio, también tras reabrir. TestFormatV1Reference hace lo mismo
// con el directorio guardado en store/testdata/format-v1 (primary/, backup/ y want.json), que se genera una sola vez
// con go test ./store -run TestFormatV1Reference -update-format-v1 y que la prueba exige (sin él falla).
//
// Estado (2026-11-22): **formato 2 congelado el 2026-11-22** (tareas 132 y 134, «Formato 2»): este binario escribe el
// 2 y lee el 1 y el 2. TestFormatV2Reference exige store/testdata/format-v2 (primary/, backup/ y want-bytes.json, con
// los valores en base64 porque la referencia del 2 tiene valores binarios), generado una sola vez con
// go test ./store -run TestFormatV2Reference -update-format-v2 y que nunca se regenera; format-v1 sigue igual.
//
// Estado (2026-11-21): **el formato 1 está congelado** (tarea 54). Antes de congelar entró el encabezado de los
// segmentos del WAL (pregunta 76; el formato en store/internal/wal/doc.go, «Encabezado del segmento»), y se midió el
// encabezado de 28 B de cada versión con la forma en que josefina guarda sus índices (pregunta 75,
// TestIndexHeaderShare con STORE_BENCH=1): es el 17,8 % del disco de los índices y un encabezado de 16 B ahorraría
// ~14 %, menos que el lugar libre de sus hojas (al 57 % tras actualizar), que no es formato (tarea 108). Se congeló
// con 28 B (pregunta 77, decidida el 2026-11-21). Un directorio escrito antes del encabezado de los segmentos no
// abre: sus segmentos se leen como versión 0 (wal.OtherVersion: un registro válido en el byte 0) y Open da
// ErrFormatVersion sin tocar nada.
//
// Regla, desde el 2026-11-21: **todo cambio en disco sube FormatVersion** (wal.FormatVersion, que el control toma) y
// lleva migración (o lectura de la versión anterior), un directorio de referencia nuevo (testdata/format-v<n>) junto
// a los anteriores, que se siguen abriendo o migrando, y su prueba de ida y vuelta (checkCodecs compara lo que
// escribe este binario con el archivo de una versión anterior salvo el byte de la versión: asWritten). Nunca se regenera format-v1: si
// TestFormatV1Reference falla, el cambio rompió el formato 1. Lo que cuenta como formato: todo lo de «Formatos en
// disco», los registros del WAL (también los Kind y deltas de las piezas), el encabezado y la marca de los
// segmentos, y los archivos de la raíz. Un cambio de comportamiento que no cambia bytes (dónde se parte una página,
// cuándo se limpia) no es de formato.
//
// # Formato 2
//
// Tarea 132 de F-23, pregunta 78 (decidida el 2026-11-22): un solo formato 2 con (a) el motor comprime los valores
// grandes, (b) prefijo de clave compartido en la hoja y (c) cabecera de versión más corta; sin (d) (empaquetar el
// desborde). Medido con la base real (tarea 131, muestra 1/20 de josephine): store 2,84 × Pebble; los documentos 3,28 ×
// porque no se comprimen y el desborde ocupa al menos una página por valor; los índices 2,54 × con la cabecera de 28 B
// = 32 % de sus hojas. Estimado con prototipos: (a) −23 % del total, (b) −8 %, (c) −7 % (−12 % con 8 B en las
// congeladas); (a)+(b)+(c) ≈ 1,74 × Pebble.
//
// Estado: **formato 2 congelado el 2026-11-22** (tarea 134). FormatVersion = 2 (wal.FormatVersion, que el control
// toma) y MinFormatVersion = 1: este binario lee los dos y escribe el 2. Hecho: (a), (b)+(c) (hojas de la forma 2) y la
// migración desde el 1 (los archivos al abrir; las hojas de a poco). La referencia es store/testdata/format-v2
// (TestFormatV2Reference la exige); format-v1 se conserva y nunca se regenera: TestFormatV1Reference sigue abriéndolo
// y TestFormatV1Migrates comprueba que se migra (también las hojas, con Vacuum). Desde ahora un cambio en disco es un
// formato 3. La forma 1 de la hoja (con EntryCompressed) sigue siendo parte del 2: un directorio del 1 a medio migrar
// es un directorio del 2.
//
// # Formato 2: compresión de valores (a)
//
// Cuándo: en cada escritura con valor (Put, Insert, Update, sus formas In y Deferred), una sola vez, antes de las
// vueltas de espera y del desborde (Engine.compressed en compress.go: write y writeIn), si STORE_COMPRESSION no es off
// y el valor mide al menos STORE_COMPRESSION_MIN (256 B). El valor comprimido, con su encabezado, se guarda solo si
// ahorra al menos un octavo; si no, va tal cual (leerlo no cuesta CPU). La decisión en línea o desborde se toma sobre lo
// guardado (28 + id + guardado ≤ 2.032): un documento JSON de 2 a 5 KB vuelve a la hoja. Una lápida nunca se comprime.
// Los valores ya escritos no se recomprimen: quedan como están hasta que se reescriben (pregunta 80).
//
// Marca en la entrada: EntryCompressed (bit 6 de flags, btree). Con ella, valueLen es el largo guardado (comprimido y
// con su encabezado) y el valor en línea, o la cadena de desborde (EntryOverflow | EntryCompressed), guarda esos bytes;
// Ref.Len es ese largo. Una lápida con EntryCompressed es daño (parseEntry). La marca nunca cambia después de escribir
// la versión: las marcas de visibilidad, el congelado, «liberándose», la poda (tarea 108), la partición y la
// recuperación la copian con la entrada sin mirarla.
//
// Valor comprimido (lo guardado, en línea o juntando la cadena de desborde; little-endian):
//
//	off tam campo
//	  0   1 method   1 = zstd (un cuadro, nivel SpeedFastest, sin suma de contenido), 2 = S2 (un bloque); otro = daño
//	  1   4 rawLen   largo sin comprimir, ≤ MaxValueLen
//	  5   … datos    del método; descomprimidos tienen que medir rawLen
//
// El método va en el valor, no en la entrada ni en el control: un motor lee cualquiera, y STORE_COMPRESSION solo
// elige con cuál escribe (cambiarla no pide migración; s2 por omisión, zstd, auto = S2 bajo 2 KB y zstd desde 2 KB,
// u off). Sin
// suma propia: la página (en línea) y cada página de desborde tienen la suya; un byte cambiado dentro de los datos que
// pasara la suma de la página puede no notarse al descomprimir (zstd sin suma), uno del encabezado o un cuadro roto sí.
// Sin diccionario (a confirmar: un diccionario por store entrenado con sus valores ayudaría a los JSON de < 512 B, que
// sin él casi no achican; sería otro method con el id del diccionario).
//
// WAL: sin Kind nuevo ni cambio en los deltas. KindLeafWrite lleva la entrada con el valor comprimido (el delta es la
// entrada) y KindOverflowWrite los trozos de lo comprimido; el WAL también achica. La réplica física aplica los bytes
// tal cual y descomprime al leer, como el primario. El flujo lógico (Changes) entrega el valor sin comprimir:
// leafWrite anota la marca itemCompressed en el búfer de reordenamiento (no es durable: su formato cambia libremente)
// y replay descomprime después de armar el valor de la cadena. Respaldo, archivo continuo y OpenAt copian bytes.
//
// Lectura: valueOf (Read, ReadIn, ReadAt, GetVersion…, Update, que compara con el valor descomprimido) y emit de los
// recorridos (ForEach… con una copia nueva; ForEachBorrowed descomprime en un búfer del recorrido que se reutiliza,
// válido solo dentro de fn como siempre). Un valor que no se descomprime es ErrCorrupt (MSG_COMPRESSED).
//
// Verify: descomprime cada valor comprimido en línea (btree.VerifyWith entrega su valor) y también los que viven en
// una cadena de desborde (tarea 137: si la cadena está sana la lee entera con overflow.Manager.Read y la descomprime,
// storeCheck.chainValue; en memoria, a lo sumo un valor leído y uno descomprimido a la vez), y anota FindValue (no
// reparable) si no se descomprime o no da el largo anotado; una versión «liberándose» no se lee. Repair no toca los
// valores.
//
// Memoria y CPU: un codificador y un decodificador de zstd por proceso (EncodeAll y DecodeAll concurrentes,
// WithLowerEncoderMem, WithDecoderLowmem); S2 no guarda estado. Nada crece con la base.
//
// Medido (Apple M5, 2026-11-22; compress_test.go): BenchmarkValueCodec (CPU por valor, JSON tipo josefina con ids al
// azar; la razón incluye los 5 B del encabezado):
//
//	tamaño   zstd razón  comprimir  descomprimir   S2 razón  comprimir  descomprimir
//	1 KB       0,64       5,3 µs     ~3,6 µs         0,83     0,96 µs     0,12 µs
//	2 KB       0,51       8,1 µs      3,2 µs         0,67     1,5 µs      0,23 µs
//	4 KB       0,42      12,0 µs      5,4 µs         0,56     3,1 µs      0,56 µs
//	8 KB       0,36      23,3 µs     10,1 µs         0,50     6,0 µs      1,2 µs
//	16 KB      0,32      42,6 µs     18,4 µs         0,46    11,8 µs      2,6 µs
//
// Por debajo de 512 B ninguno ahorra un octavo con estos documentos (quedan sin comprimir). Las opciones de zstd
// (memoria baja, ventana de 64 KB, nivel por omisión) no cambian la CPU más de un 30 % y la razón menos de un 2 %.
// TestCompressDisk (STORE_BENCH=1; 40.000 documentos con la mezcla de tamaños de la base josephine, doc-sizes.md:
// 59 MB de JSON; disco del store tras Vacuum y Checkpoint; Put diferido y Read de cada documento, ForEach completo):
//
//	STORE_COMPRESSION  disco ÷ JSON   WAL      Put diferido   Read        ForEach (por documento)
//	off                1,64           90 MB    20–26 µs       3,6–4,1 µs  0,77 µs
//	s2                 1,10           69 MB    16–17 µs       4,0–4,4 µs  0,85 µs
//	zstd               0,94           56 MB    17–21 µs       6,2–6,4 µs  3,95 µs
//	auto               1,06           64 MB    17,8 µs        4,5 µs      1,60 µs
//
// off da lo mismo que la muestra real sin comprimir (1,67 ×, tarea 128) y zstd lo mismo que su prototipo (0,90 ×).
// Comprimir no encarece la escritura (escribe menos páginas y WAL; los rangos son de dos corridas). La lectura paga
// la descompresión: con S2 casi nada (Read +8 %, ForEach +10 %); con zstd Read +60 % y un recorrido completo 5 veces
// más lento, a cambio de un 15 % menos de disco que S2. Por omisión S2 (pregunta 81): con ella el disco de los
// documentos baja un tercio y la CPU de leer queda igual; zstd queda para quien prefiere disco a CPU.
//
// # Formato 2: hojas de la forma 2 (b y c)
//
// Tarea 134 de F-23 (preguntas 79 y 82, decididas el 2026-11-22), hecha el 2026-11-22. Una hoja nueva lleva
// LayoutVersion 2 en pd_pagesize_version (8.192 | 2); las demás clases de página (meta, internas, desborde, libres,
// borradas, vm) siguen en 1, y una hoja en 1 es la forma 1 (más EntryCompressed). btree lee las dos formas detrás de
// node.entryAt/keyAt (parseEntry y parseEntry2) y escribe solo la 2: al iniciar una hoja (KindInitStore, la derecha
// de KindSplit), al rearmar la izquierda de una partición, y al reescribir una hoja de la forma 1 antes de cambiarla
// (KindLeafRewrite; ver la migración, abajo). El formato byte a byte está en store/internal/btree/doc.go, «Hojas de la
// forma 2»; el resumen:
//
//	ítems de una hoja 2   qué guarda
//	1                     clave alta (pivote, como en la 1; no está en la de más a la derecha)
//	2 (o 1)               ítem de prefijo: pivote sin enlace ni lsn con el prefijo de la hoja (0 a 255 B)
//	resto                 entradas de la forma 2 en orden (id, lsn descendente), cada una sin el prefijo
//
//	entrada 2   word0(4) = marcas (bits 0–7) | largo del sufijo (8–19) | largo en línea (20–31), lsn(8), xid(8) salvo
//	            en una congelada sin desborde, sufijo del id, valor en línea o first(4) len(4): 20 B viva, 12 B
//	            congelada (pregunta 82)
//
// (b) Prefijo compartido. Invariante: el id de toda entrada de la hoja empieza con su prefijo. El prefijo se elige al
// armar la hoja (choosePrefix): el común de su primera y su última entrada, hasta 255 B. Una escritura cuyo id no lo
// tiene reescribe antes la hoja con el común de los dos (KindLeafRewrite) y, si así no cabe, la parte; una que no
// cabe intenta, después de podar (tarea 108) y antes de partir, recortar las entradas con un prefijo más largo si ya lo
// comparten. Cada mitad de una partición elige el suyo (igual o más largo que el de la hoja: sus entradas solo se
// achican; con 255 B de tope la partición cuenta el ítem de prefijo de cada mitad con su largo máximo y siempre
// encuentra un corte). La búsqueda recorta la clave una vez por hoja (un id sin el prefijo cae antes o después de toda
// la hoja) y compara solo sufijos; keyAt da la clave con el prefijo aparte (key.pre) y Version el id entero.
//
// Diseño corregido al implementar (lo de arriba reemplaza lo diseñado el 2026-11-22): el prefijo iba a salir de las
// cotas (clave baja como segundo ítem y clave alta), con la izquierda de una partición conservando el suyo. No sirve:
// (1) el borrado de una hoja (tarea 70) pasa su espacio de claves a la hermana derecha, que desde ahí recibe ids
// menores que su clave baja y sin su prefijo; (2) con ids crecientes (la hoja de más a la derecha, sin cota alta, la
// cola de cada grupo de un índice) el prefijo de las cotas es vacío y la izquierda lo conservaba: casi ninguna hoja
// tenía prefijo. Por eso el prefijo sale de las entradas, una escritura que no lo comparte lo acorta, cada mitad de una
// partición elige el suyo y la clave baja no se guarda; el largo va en el ítem de prefijo y no en pd_flags.
//
// (c) Cabecera corta. Las marcas de visibilidad siguen con un OR atómico sobre el uint32 del byte 0 (los bits altos de
// word0 no cambian después de escribir la entrada). Congelar achica la entrada: KindLeafVacuum (sin cambio de forma)
// rehace en una hoja 2 su lista de congeladas reemplazando cada entrada por la misma sin xid (si no tiene desborde) y
// compacta con page.Repair. Una congelada en línea da Version.XID = 0.
//
// WAL: KindLeafWrite lleva en una hoja 2 la entrada de la forma 2 con el id entero (bit 15 del número de ítem): el flujo
// lógico (DecodeLeafWrite) la lee sin la página, y al aplicarla se le quita el prefijo de la hoja. KindSplit marca con
// el bit 15 de count una derecha de la forma 2; la izquierda de una hoja 2 se rearma con su prefijo nuevo en el mismo
// rehacer (determinista: sale de la página y del delta). KindInitStore lleva layout(2) en la hoja raíz y flags(4) en
// la meta (MetaLeavesForm2). KindLeafRewrite (27) = prefixLen(2): rearma la hoja en la forma 2 con ese prefijo.
// KindSetFrozen admite flags(4) al final (se suman a las de la meta). Registros del formato 1 (entradas de la forma 1,
// KindInitStore sin layout) se rehacen como antes, sobre hojas de la forma 1.
//
// Lo que no cambió: la poda antes de partir (tarea 108) y el borrado de páginas (tarea 70) trabajan igual sobre hojas
// 2 (una hoja medio muerta conserva su ítem de prefijo; la borrada vuelve a la forma 1: es otra clase de página); las
// marcas de visibilidad; Repair (sin reparación nueva). Verify comprueba el ítem de prefijo, cada entrada y que el
// prefijo deje lugar a algún id dentro de las cotas de la hoja (btree.prefixInBounds, tarea 137: los ids con ese
// prefijo tienen que cruzarse con [cota baja, clave alta); así se ve también en una hoja sin entradas), cuenta las
// hojas de la forma 1 (Report.LeavesForm1) y da daño si la meta dice MetaLeavesForm2 y queda alguna. Un prefijo
// dañado sale como FindTree (no reparable, como cualquier otro daño del árbol). Límite: un prefijo cambiado que sigue
// cruzándose con las cotas y deja las entradas en orden y dentro de ellas no se ve (la hoja de más a la derecha, sin
// clave alta, con un prefijo mayor: TestReviewDamagedPrefix); sus ids cambian de nombre. Verlo pide una copia del
// prefijo en otro lado (formato) o la suma de la página, que se comprueba al leerla del disco y no en la caché. No se
// exige más (que prefijo y cota baja coincidan hasta el más corto): no es un invariante, el borrado de una hoja baja
// la cota de la hermana derecha y una partición posterior da a su izquierda el prefijo de sus entradas
// (TestPrefixAfterLeafDelete).
//
// Medido (Apple M5, 2026-11-22; antes = el binario con (a) y la hoja de la forma 1). Disco:
//
//	prueba (STORE_BENCH=1)                       antes      después    cambio
//	TestIndexHeaderShare índices, tras cargar     154,7 MB   115,5 MB   −25 % (disco ÷ datos vivos 2,02 → 1,51)
//	  tras actualizar y limpiar                   161,2 MB   119,7 MB   −26 % (2,10 → 1,56)
//	  principal (documentos de ~400 B)             49,1 MB    46,5 MB    −5 %
//	  todo el modelo, tras actualizar             210,3 MB   166,1 MB   −21 % (1,83 → 1,44)
//	TestDiskSteadyState (disco ÷ datos vivos)     1,38       1,27        −8 %
//	TestCompressDisk s2 (documentos JSON)         64,7 MB    64,1 MB     −1 % (pesan los valores)
//
// El prototipo de la tarea 131 (muestra real de josephine) daba en los índices −16 % con (b) y −29 a −36 % con
// (b)+(c); el −25 % de TestIndexHeaderShare cae en ese rango por abajo: sus índices no tienen versiones congeladas (la
// cabecera de 12 B no entra en juego) y las claves son más cortas que las reales (50 B contra 51 + 35 de sobrecarga).
// Sobre el total del modelo, −21 %, contra −15 a −20 % de (b)+(c) en el prototipo. Las columnas «28 B %» y «llenado»
// de TestIndexHeaderShare suponen la entrada de la forma 1: con la 2 pasan del 100 %.
//
// CPU (go test -bench, mediana de 2–3 corridas): el árbol solo (store/internal/btree, bench_test.go: ids de índice de
// 36 B) BenchmarkTreeLookup 1,87 → 1,08 µs, BenchmarkTreeWrite 2,36 → 1,70 µs, BenchmarkTreeScan (100.000 ids)
// 5,2 → 6,0 ms (+15 %: armar el id entero de cada versión); el motor BenchmarkPutPlain diferido 4,97 → 4,35 µs,
// BenchmarkPutIndexed diferido 8,5 → 6,7 µs (1 índice) y 10,3 → 8,4 µs (2), con commit igual (manda el fsync),
// BenchmarkQueryByIndex 476 → 431 µs, BenchmarkScanFilter 9,6 → 9,8 ms. Asignaciones por operación iguales o menos
// (PutPlain 42/43 → 41/43, QueryByIndex 4.090 → 3.916); los bytes por escritura suben ~20–30 % (las páginas que arman
// las particiones y las reescrituras, repartidas). Lo que más ganó no es de formato: page.CheckItems se hace una vez
// por carga de la página en la caché (buffer.ItemsChecker) y no en cada bloqueo; con hojas de ~1,5 veces más
// entradas repetirlo hacía a Lookup un 30 % más lento.
//
// # Formato 2: migración desde el formato 1 (e)
//
// Hecha para (a): Open de un directorio del 1 (sin herramienta aparte): checkFormat acepta 1 y 2 en cada archivo; la
// recuperación rehace el WAL del 1 (sus registros son del 2: el 2 solo agrega la marca); antes de escribir nada
// nuevo, wal.Open (Options.Upgrade, salvo en una réplica) deja atrás el segmento activo del 1 (lo sella y sigue en uno
// nuevo con encabezado del 2, o le reescribe el encabezado si no tiene registros; esto pasa antes de markInProduction,
// pero no escribe registros) y el control (markInProduction tras un cierre limpio, y siempre el checkpoint del final
// de la recuperación) y el catálogo pasan al 2. Así ningún registro con EntryCompressed que escriba un primario cae en
// un segmento que dice 1, y un binario del 1 que abre el directorio migrado da ErrFormatVersion (control 2) sin
// tocarlo. Una réplica abierta de una copia del 1 (OpenReplica) no hace Upgrade: su control y su catálogo pasan al 2
// con el primer restartpoint (al abrir), y su segmento activo sigue en 1 hasta que el flujo del primario cambia de
// segmento (la réplica rota con FormatVersion: frente a un primario todavía del 1, sus segmentos nuevos dicen 2 y el
// de él 1; los registros son los mismos). Si se la promueve (Promote) antes de eso, Promote rota el WAL antes de
// escribir nada (tarea 138, wal.StartSegment: sella el del 1 y sigue en uno del 2, o le reescribe el encabezado si no
// tiene registros), así que lo que escribe como primario —también valores comprimidos— nunca cae en un segmento que
// dice 1; un binario del 1 no abre el directorio (control 2). Los segmentos sellados del 1, slots y archive_status siguen en 1 hasta reciclarse o reescribirse: se leen
// igual. Las páginas siguen en LayoutVersion 1 (la (a) no cambia su forma). Réplicas: el binario nuevo primero en
// las réplicas (leen 1 y 2), después en el primario; una réplica del 1 frente a un primario migrado da
// ErrFormatVersion en el primer segmento del 2. No hay vuelta atrás: un directorio migrado no lo abre un binario del 1
// (copiarlo antes con Backup).
//
// Para (b)+(c), hecha (pregunta 79: migración perezosa, como la versión por página de PostgreSQL): las páginas de un
// directorio del 1 no se tocan al abrir. Una hoja de la forma 1 se reescribe en la 2 (KindLeafRewrite) cuando una
// escritura va a cambiarla; la limpieza de un store cuya meta no tiene MetaLeavesForm2 recorre todas sus hojas sin
// saltar por el mapa de visibilidad, reescribe las de la forma 1 y, si la pasada termina sin error, anota la marca
// (KindSetFrozen con flags): desde ahí salta hojas como siempre. Las marcas de visibilidad y las lecturas no
// reescriben nada. El registro de una reescritura es chico (prefixLen); la primera vez que se toca la hoja tras un
// checkpoint lleva su imagen, como cualquier cambio: mientras la limpieza recorre un store del 1 escribe un WAL del
// tamaño de sus hojas. Cuánto falta: Stats.LeafForm1Stores (stores abiertos con hojas de la forma 1; el motor lo lee de
// cada meta al abrir el store y lo pone al día cada limpieza) y Report.LeavesForm1 de Verify (las hojas exactas);
// Stats.LeafRewrites cuenta las que reescribió la limpieza. La limpieza de fondo corre también sin escrituras mientras
// quede algo (leafMigration.due: el catálogo era del 1 al abrir y todavía no hubo una pasada entera, o un store abierto
// sin migrar); Engine.Vacuum la fuerza. No hay herramienta aparte (a confirmar: Engine.Upgrade o storecheck -upgrade).
// Una réplica no limpia: sus hojas se reescriben con los registros del primario, y su Stats.LeafForm1Stores baja al
// rehacer el KindSetFrozen con MetaLeavesForm2 de la limpieza del primario (btree.RecordMetaFlags en applyEntry; un
// store vaciado o borrado deja de contar en syncHandles; tarea 137).
//
// # Límites
//
//   - id: de 1 a 2.000 B (pregunta 37). Nombre de un store: texto UTF-8 válido sin el carácter nulo, de 1 a 255 B
//     (pregunta 57).
//   - valor: hasta overflow.MaxValueLen = 64 MiB. Un valor grande se guarda en páginas de desborde y cada una es un
//     registro del WAL de ~8 KB, así que el tamaño del valor no cambia el de los registros: ningún registro pasa de 64
//     KiB y ninguno cruza segmentos del WAL (si no cabe en el activo, se rota). El tope existe porque el valor se
//     recibe y se devuelve entero en memoria y pasa entero por el WAL: 64 MiB es un octavo del volumen que dispara un
//     checkpoint, así que un solo Put no se acerca a la cota de Open.
//   - transacciones y snapshots: 10 minutos (TRANSACTION_TIMEOUT); un snapshot puede pedir el suyo o ninguno.
//
// # API pública
//
// store/catalog.go tiene toda la API pública, por secciones; la implementación es privada en los demás archivos.
//
//	// Motor
//	func Open(dir string) (*Engine, error)                     // control + WAL desde el último checkpoint; restaura una copia
//	func OpenWith(dir string, o Options) (*Engine, error)      // Open con opciones propias (tarea 84; cero = la variable)
//	func OpenAt(dir string, lsn uint64) (*Engine, error)       // restaura una copia de Backup hasta lsn (PITR)
//	func OpenAtTime(dir string, t time.Time) (*Engine, error)  // hasta el último registro antes del primero con hora > t
//	func (e *Engine) Close() error                             // limpieza si hace falta (closeVacuum) + checkpoint final
//	func (e *Engine) Store(name string) (*Store, error)        // abre el store, o lo crea si no existe
//	func (e *Engine) DropStore(name string) error
//	func (e *Engine) Stores() []string
//	func (e *Engine) StoreIn(tx *Batch, name string) (*Store, error) // abre, o crea dentro del lote
//	func (e *Engine) DropStoreIn(tx *Batch, name string) error      // borra al confirmar el lote
//	func (e *Engine) Checkpoint() error
//	func (e *Engine) Vacuum() error
//	func (e *Engine) WaitDurable() error                       // lo confirmado hasta ahora, en disco
//
//	// Transacciones
//	func (e *Engine) NewBatch() *Batch                         // una transacción (snapshot isolation)
//	func (b *Batch) Commit() error
//	func (b *Batch) CommitDeferred() error                     // Commit sin esperar el fsync
//	func (b *Batch) Rollback() error
//	func (b *Batch) SetOrigin(origin uint16) error             // va en su KindCommit y sale en TxInfo.Origin (tarea 83)
//	func (b *Batch) CommitLSN() uint64                         // el lsn de su KindCommit, tras confirmar
//	func (e *Engine) Snapshot() *Snapshot                      // tiempo máximo TRANSACTION_TIMEOUT
//	func (e *Engine) SnapshotFor(timeout time.Duration) *Snapshot // tiempo propio; 0 = sin tiempo máximo
//	func (v *Snapshot) Close() error
//
//	// Store: operaciones sueltas (cada una es su propia transacción y confirma al volver)
//	func (s *Store) Name() string
//	func (s *Store) Put(id string, data []byte) error
//	func (s *Store) Remove(id string) error
//	func (s *Store) Insert(id string, data []byte) (bool, error)
//	func (s *Store) Update(id string, data []byte) (bool, error)
//	func (s *Store) Delete(id string) (bool, error)
//	func (s *Store) Read(id string) ([]byte, bool, error)
//	func (s *Store) GetVersion(id string) ([]byte, uint64, bool, error) // Read con la versión
//	func (s *Store) Exists(id string) (bool, error)
//	func (s *Store) Count() int64                             // el contador, O(1)
//	func (s *Store) Empty() error                             // O(1); ErrBusy con lotes o snapshots abiertos
//	func (s *Store) Deferred() Deferred                       // Put, Remove, Insert, Update, Delete sin esperar el fsync
//
//	// Store: dentro de un lote
//	func (s *Store) PutIn(tx *Batch, id string, data []byte) error
//	func (s *Store) RemoveIn(tx *Batch, id string) error
//	func (s *Store) InsertIn(tx *Batch, id string, data []byte) (bool, error)
//	func (s *Store) UpdateIn(tx *Batch, id string, data []byte) (bool, error)
//	func (s *Store) DeleteIn(tx *Batch, id string) (bool, error)
//	func (s *Store) ReadIn(tx *Batch, id string) ([]byte, bool, error)
//	func (s *Store) GetVersionIn(tx *Batch, id string) ([]byte, uint64, bool, error)
//	func (s *Store) ExistsIn(tx *Batch, id string) (bool, error)
//
//	// Store: sobre un snapshot
//	func (s *Store) ReadAt(v *Snapshot, id string) ([]byte, bool, error)
//	func (s *Store) GetVersionAt(v *Snapshot, id string) ([]byte, uint64, bool, error)
//	func (s *Store) ExistsAt(v *Snapshot, id string) (bool, error)
//	func (s *Store) ForEachAt(v *Snapshot, fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error
//	func (s *Store) ForEachPrefixAt(v, prefix, fn, asc, offset, limit) error
//	func (s *Store) ForEachRangeAt(v, from, to, fn, asc, offset, limit) error  // [from, to)
//	func (s *Store) ForEachBorrowedAt(v *Snapshot, fn) error                   // ascendente, data prestado
//	func (s *Store) CountAt(v *Snapshot) (int64, error)                        // recorre, O(n)
//
//	// Store: recorridos sueltos (cada uno con un snapshot propio)
//	func (s *Store) Keys(asc bool, offset, limit int) ([]string, error)
//	func (s *Store) KeysAfter(after string, limit int) ([]string, error)
//	func (s *Store) KeysPrefix(prefix string, asc bool, offset, limit int) ([]string, error)
//	func (s *Store) ForEach(fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error
//	func (s *Store) ForEachPrefix(prefix string, fn, asc bool, offset, limit int) error
//	func (s *Store) ForEachRange(from, to string, fn, asc bool, offset, limit int) error
//	func (s *Store) ForEachBorrowed(fn func(id string, data []byte) (bool, error)) error
//
//	// Revisión, estadísticas y disco
//	func (e *Engine) Verify() (Report, error)                  // revisa sin cambiar nada
//	func (e *Engine) Repair() (Report, error)                  // arregla lo reparable, con WAL
//	func (e *Engine) Stats() Stats                             // foto barata del motor
//	func (e *Engine) OnStats(interval time.Duration, fn func(Stats)) func() // fn cada interval; devuelve stop
//	func (e *Engine) Pressure() float64                        // 0 = sin presión, ≥ 1 = ya frena
//	func (e *Engine) DirUsage() (DiskUsage, error)
//	func (s *Store) DirUsage() (int64, error)
//
//	// Respaldo y archivo continuo
//	func (e *Engine) Backup(dir string) (BackupInfo, error)    // copia en caliente; Open(dir) restaura
//	func (e *Engine) BackupTo(sink BackupSink) (BackupInfo, error) // lo mismo hacia un flujo (un tar), archivo por archivo
//	func (e *Engine) PendingArchive() ([]WALSegment, error)    // sellados sin archivar, hasta 256
//	func (e *Engine) SetArchived(seg uint64) error
//	func (e *Engine) SwitchWAL() (uint64, error)               // sella el segmento activo
//
//	// Replicación (ver «Replicación»)
//	func OpenReplica(dir string) (*Engine, error)              // una copia de Backup como réplica de solo lectura
//	func OpenReplicaWith(dir string, o Options) (*Engine, error)
//	func (e *Engine) Promote() error                           // la réplica pasa a primario sin cerrarla (tarea 85)
//	func (e *Engine) WalSince(lsn uint64) (*WalReader, error)  // el WAL durable desde lsn + 1, del disco
//	func (r *WalReader) Next(maxBytes int) ([]WalEntry, error)
//	func (e *Engine) ApplyWal(entries []WalEntry) error        // en la réplica; ApplyWalEntry, uno solo
//	func (e *Engine) ReplayLSN() uint64                        // lo aplicado por la réplica
//	func (e *Engine) CreateSlot(name string) (SlotInfo, error) // física; CreateLogicalSlot, lógica
//	func (e *Engine) DropSlot(name string) error
//	func (e *Engine) Slots() []SlotInfo
//	func (e *Engine) AdvanceSlot(name string, lsn uint64) error
//	func (e *Engine) Changes(slot string) (*ChangeReader, error)
//	func (e *Engine) ChangesWith(slot string, f ChangeFilter) (*ChangeReader, error) // con filtro (store, id, origen)
//	func (e *Engine) ChangesFrom(pos ChangePos, f ChangeFilter) (*ChangeReader, error) // sin ranura, desde pos
//	func (e *Engine) ChangeHead() (ChangePos, error)           // la posición actual, sin esperar a los lotes abiertos
//	func (e *Engine) AdvanceLogicalSlot(name string, pos ChangePos) error
//	func (r *ChangeReader) Next(maxTx int, fn func(tx TxInfo, c Change) error) (int, error)
//	func (r *ChangeReader) Pos() ChangePos                     // tras lo entregado: ChangesFrom(Pos(), …) sigue
//	func (r *ChangeReader) Confirm() error
//	func (r *ChangeReader) Close() error
//
// Tipos: Engine, Store, Batch, Snapshot, Deferred, Report y Finding (report.go), Stats y Latency (stats.go),
// PressureCause (pressure.go), DiskUsage (usage.go), BackupInfo (backup.go), WALSegment (archive.go), WalReader
// (walsince.go), WalEntry (replica.go), SlotInfo (slots.go), ChangeReader, Change, ChangeOp y TxInfo (decode.go); constantes
// MaxIDLen, MaxValueLen y MaxNameLen. Errores públicos (msg.go): ErrLocked, ErrClosed, ErrFailed, ErrBadID,
// ErrValueTooLarge, ErrBadName, ErrStoreNotFound, ErrStoreDropped, ErrConflict, ErrSerialization, ErrTimeout,
// ErrTxDone, ErrOtherEngine, ErrBusy, ErrStorePending, ErrDiskFull, ErrThrottled, ErrArchiveOff, ErrNotSealed,
// ErrRecoveryTarget, ErrBackupDir, ErrBackupIncomplete, ErrReadOnly, ErrNotReplica, ErrWalGap, ErrWalRemoved,
// ErrSlotNotFound, ErrSlotExists, ErrSlotLimit, ErrRecoveryConflict, ErrCorrupt (también el daño que encuentra el
// paquete wal: wal.ErrCorrupted se traduce, tarea 101), ErrFormatVersion; se comparan con errors.Is.
//
// Sync de josefina (un gancho por cambio aplicado) queda cubierto por Changes (el flujo lógico). Decidido no hacer: un recorrido dentro de un lote (ForEachIn, pregunta 64: se recorre con un snapshot
// y dentro del lote se lee por id con ReadIn).
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
//     TRANSACTION_TIMEOUT aborta el lote (ErrTimeout). Tras un error de esos el lote queda abortado enseguida (no
//     retiene sus ids): sus operaciones devuelven ese error, Commit lo devuelve y Rollback lo acepta. Commit es todo
//     o nada, también tras una caída, con un fsync para todo el lote. Usar un lote terminado devuelve ErrTxDone, y un
//     lote o snapshot de otro motor, ErrOtherEngine.
//   - Un snapshot es una foto fija de todos los stores del motor para leer y recorrer, sin bloquear a los escritores.
//     Mientras siga abierto (igual que un lote), la limpieza no quita las versiones que podría ver.
//   - Lo devuelto por Read es del llamador, y data en ForEach… también; en ForEachBorrowed… data vale solo dentro de
//     fn. Un callback de recorrido corre sin nada tomado: puede leer y escribir en el motor (el recorrido no ve lo
//     escrito después de su snapshot); Empty desde él devuelve ErrBusy (el recorrido tiene su snapshot abierto).
//   - Exists, ExistsIn y ExistsAt ven lo mismo que Read, ReadIn y ReadAt, sin leer ni copiar el valor.
//   - Los recorridos entregan la versión visible más nueva de cada id (las lápidas no se ven; con un snapshot, las
//     versiones viejas que todavía ve aunque haya otras más nuevas sin limpiar) en orden de id por bytes; offset y
//     limit cuentan ids visibles (limit ≤ 0 = sin límite). Count es el contador (O(1)); CountAt recorre (O(n)).
//   - Empty vacía el store en O(1) como TRUNCATE; con lotes o snapshots abiertos en el motor devuelve ErrBusy.
//   - Store.Deferred() y Batch.CommitDeferred confirman sin esperar el fsync: lo confirmado se ve enseguida; un corte
//     puede perderlo, siempre entero y en el orden del WAL, hasta que Engine.WaitDurable vuelva (o una confirmación
//     durable posterior, o el escritor de fondo del WAL). GetVersion, GetVersionIn y GetVersionAt son Read, ReadIn y
//     ReadAt con la versión: el lsn de la versión devuelta (0 si no existe).
//   - Si el motor quedó inutilizable (ver «Fallos de escritura»), todo devuelve ErrFailed hasta reabrirlo; si fue por
//     disco lleno, las lecturas siguen y las escrituras devuelven además ErrDiskFull. Un ErrDiskFull sin ErrFailed es
//     una operación rechazada sin efecto, y ErrThrottled una frenada por el WAL sin reciclar (ver «Disco lleno o
//     lento»): las dos se reintentan.
//
// # Variables de entorno
//
// Se leen al abrir (Open, OpenAt), con envar:
//
//	variable                 por omisión  qué fija
//	STORE_CACHE_SIZE         128          tamaño de la caché de páginas, en MB, mínimo 1 (pregunta 21)
//	WAL_SEGMENT_SIZE         128          tamaño de cada segmento del WAL, en MB, mínimo 1 (pregunta 27)
//	SYNC_ON_WRITE            true         una confirmación espera el fsync del WAL (false = todas diferidas, salvo
//	                                      la de un lote con DDL)
//	CHECKPOINT_TIMEOUT       300          segundos entre checkpoints, mínimo 1 (pregunta 28)
//	MAX_WAL_SIZE             1024         MB de WAL que acotan la recuperación; dispara a MAX_WAL_SIZE / 1,9; con
//	                                      más de 2 × MAX_WAL_SIZE sin reciclar los escritores esperan un checkpoint;
//	                                      fija también los repuestos (MAX_WAL_SIZE / 2 / WAL_SEGMENT_SIZE)
//	LOCK_TIMEOUT             5            segundos que una escritura espera a otra transacción (pregunta 33)
//	TRANSACTION_TIMEOUT      600          segundos que puede durar un lote o un snapshot; 0 = sin tope (pregunta 35)
//	VACUUM_INTERVAL          60           segundos entre vueltas de la limpieza de fondo; 0 = apagada (a confirmar)
//	VACUUM_THRESHOLD         1000         escrituras desde la última limpieza para correr otra (a confirmar)
//	WAL_WRITER_DELAY         200          ms entre vueltas del escritor de fondo del WAL; 0 = apagado (pregunta 65)
//	WAL_ARCHIVE              off          on (o true, 1, yes) = archivo continuo: retiene los segmentos sellados
//	                                      hasta SetArchived (tarea 66)
//	WAL_ARCHIVE_MAX_SIZE     0            MB que puede retener el archivo; 0 = sin tope (frena a los escritores)
//	MAX_SLOT_WAL_KEEP_SIZE   4096         MB de WAL que puede retener una ranura antes de invalidarse (pregunta 48);
//	                                      0 = sin tope (la ranura frena a los escritores, como el archivo)
//	WAL_BUFFERS              1024         KB de registros sellados que esperan en memoria antes de escribirse
//	                                      (wal_buffers; tarea 100); 0 = una escritura por registro
//	STORE_COMPRESSION        s2           cómo se comprimen los valores grandes (formato 2, tarea 132): s2, zstd,
//	                                      auto (S2 bajo 2 KB, zstd desde 2 KB) u off; solo elige al escribir
//	                                      (pregunta 81)
//	STORE_COMPRESSION_MIN    256          bytes: un valor más corto no se comprime
//
// OpenWith y OpenReplicaWith (tarea 84) toman las mismas de un Options (STORE_COMPRESSION en Compression, "" = la
// variable; STORE_COMPRESSION_MIN en CompressionMin): un campo en cero deja la variable, uno
// negativo apaga lo que 0 apaga (WALWriterDelay, VacuumInterval, TransactionTimeout, ArchiveMaxMB, SlotKeepMB,
// WALBuffersKB). Así un proceso abre varios motores con distinta caché, WAL y tiempos.
//
// Constantes, sin variable: archivos de datos de 1 GB, CHECKPOINT_COMPLETION_TARGET = 0,9, FREEZE_MIN_AGE = 0,
// FREEZE_TABLE_AGE = un archivo de estados (134.217.728 xids; ver «Congelado y truncado del archivo de estados»), el
// freno del WAL en 2 × MAX_WAL_SIZE (walBacklogFactor), el de las diferidas en 8.192 registros (maxDeferredLag), la
// espera de la réplica a sus snapshots antes de cancelarlos en 30 s (replicaConflictDelay, como
// max_standby_streaming_delay) y la memoria del búfer de reordenamiento en 16 MB (logicalWorkMem, como
// logical_decoding_work_mem). RELSEG_SIZE (del diseño anterior) dejó de usarse.
//
// # El orden de un cambio
//
// Una escritura (Put, Insert… sueltas o en un lote) hace, como PostgreSQL:
//
//  0. Sin nada tomado, el freno del WAL (throttle); con el motor tomado, la entrada de lugar (admit: el WAL tiene
//     medio segmento preasignado). Si falta lugar, ErrDiskFull sin escribir nada (ver «Disco lleno o lento»).
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
//  9. Escribir el registro en su lugar del WAL (en paralelo con otros) y publicarlo. Toda reserva se escribe, también
//     si la operación falla después de reservar: una reserva sin escribir frena published y deja Close esperando.
//  10. Confirmar: una operación suelta ya lleva FlagCommit; un lote escribe su KindCommit (con la suma de Count por
//     número de archivo de store, no por nombre, sumada bajo writeMu al reservar). Anotar el lsn como la última
//     confirmación (WaitDurable). Con SYNC_ON_WRITE, y si no es diferida (Deferred, CommitDeferred), esperar a que el
//     WAL sea durable hasta ese lsn (group commit). Después anotar confirmada en el archivo de estados con ese lsn y
//     quitar la transacción de las en curso (despierta a quien la espera).
//
// La página va al disco después, en un checkpoint, al expulsarla de la caché o por el escritor de fondo, y nunca
// antes de que el WAL sea durable hasta su pd_lsn («WAL antes que la página»): las escrituras del usuario solo
// esperan al WAL, que es secuencial.
//
// Update compara el valor antes de bloquear la hoja (leerlo de desborde con la hoja bloqueada frenaría a todos) y,
// con la hoja bloqueada, comprueba que la versión que comparó sigue siendo la última (por su lsn); si no, repite.
//
// # Operaciones sueltas
//
// Una operación suelta es una transacción con su xid (xact.Transactions.Begin): si el valor no cabe en línea escribe
// antes su cadena de desborde con ese xid; toma commitMu en lectura; btree.Write con Commit (FlagCommit), la
// condición y un UnderLock que suma su delta al contador bajo writeMu; si btree devuelve WaitError suelta commitMu,
// libera el desborde, sale del store y del motor (Store.mu y closeMu), espera a esa transacción hasta LOCK_TIMEOUT
// (si vence, ErrConflict) y repite desde la entrada con el mismo xid; si escribió, espera el WAL durable
// (SYNC_ON_WRITE) y anota confirmada en el archivo de estados; suelta commitMu y termina la transacción (End
// despierta a quien espera). Las lecturas y las escrituras de otros no ven la versión hasta que su xid figura
// confirmada (Read Committed real, no solo «escrita en la hoja»). Si la condición no se cumple y el valor había ido a
// desborde, la cadena se libera; el xid de una operación que no escribió queda «en curso» en el archivo de estados y
// se lee abortado tras reabrir (no hay versiones con él).
//
// A confirmar (ola 4): cualquier error de escritura inutiliza el motor (salvo el disco lleno limpio, ver «Disco lleno
// o lento»); Count incluye escrituras que todavía se están confirmando; el xid de una operación sin efecto queda en
// curso hasta reabrir; DropStore espera a las operaciones en curso del store (una que espera a otra transacción no
// cuenta: espera fuera del store y al volver encuentra ErrStoreDropped).
//
// # Lotes y snapshots
//
// Un lote toma su xid (xact Begin) y su snapshot (xact Snapshot(xid), cuyo Xmin cuenta en OldestActive) al crearlo.
// Cada escritura es la suelta con Commit = false, el xid y el snapshot del lote: btree devuelve la propia versión o
// la última confirmada (ErrSerialization si no es visible para el snapshot), y WaitError si otra en curso tiene el
// id; se espera sin páginas, Store.mu, closeMu ni Batch.mu (Batch.waitOther; el temporizador, Rollback y las otras
// operaciones del lote no quedan detrás) hasta LOCK_TIMEOUT sin pasar el tiempo del lote (ErrConflict o ErrTimeout),
// y se repite si el lote sigue activo.
// Las versiones van al árbol desde la primera escritura; en RAM el lote solo guarda la suma de Count por número de
// archivo. Commit: con commitMu en lectura, reserva KindCommit (time y las sumas, que se suman a los contadores bajo
// writeMu en el underLock de Reserve), lo escribe, espera el WAL durable (un solo fsync, SYNC_ON_WRITE), anota
// confirmada con ese lsn, suelta commitMu y End. Rollback igual con KindAbort, sin fsync (una transacción sin
// registro de confirmación cuenta como abortada tras una caída). Un lote sin escrituras no escribe nada en el WAL.
// Un error de serialización, conflicto o tiempo aborta el lote enseguida (KindAbort); un temporizador lo aborta al
// pasar TRANSACTION_TIMEOUT aunque nadie lo use. Las operaciones de un lote corren de a una (Batch.mu).
//
// Un snapshot es una foto de solo lectura sin xid (xact TakeSnapshot, registrada por número para que su Xmin cuente
// en OldestActive hasta ReleaseSnapshot); ReadAt es Lookup con esa foto. Close o su tiempo lo sueltan; una lectura
// en curso termina antes (Snapshot.mu).
//
// Orden de locks: Batch.mu → closeMu (lectura) → Store.mu → páginas…; Batch.mu → closeMu (lectura) → commitMu
// (lectura) → writeMu; Snapshot.mu (lectura) → closeMu (lectura) → Store.mu. Ninguno de los del motor toma Batch.mu
// ni Snapshot.mu.
//
// Por qué commitMu solo al confirmar: el lote toma commitMu solo desde reservar su KindCommit o KindAbort hasta anotar
// el estado, no desde su primer registro: las versiones de un lote no confirman nada, así que un registro de hoja
// anterior al punto de rehacer no necesita estar en el archivo de estados; lo que importa es su confirmación. Tomarlo
// desde el primer registro frenaría todos los checkpoints (y, con el RWMutex, a todos los escritores) mientras dure un
// lote.
//
// A confirmar (ola 5A): Close no espera a los lotes abiertos ni los deshace (quedan inservibles con ErrClosed y se leen
// abortados al reabrir); sí detiene los temporizadores de TRANSACTION_TIMEOUT de los lotes y snapshots abiertos
// (Engine.timers, timers.go): uno armado retendría el motor entero, caché incluida, hasta vencer; el xid de un lote sin
// escrituras queda «en curso» en el archivo de estados, como el de una operación suelta sin efecto; InsertIn no mira
// antes si el id existe con un valor grande (esa lectura con el snapshot ocultaría un cambio confirmado después, que
// debe dar ErrSerialization), así que puede escribir y liberar el desborde en vano; no hay detección de interbloqueos
// entre lotes: dos que se esperan mutuamente salen por LOCK_TIMEOUT con ErrConflict. Las cadenas de desborde de un lote
// deshecho (o de versiones propias que otra del mismo lote reemplazó) no se liberan al deshacer: las libera la limpieza
// (ola 5B) al quitar esas versiones.
//
// # DDL dentro de un lote
//
// Tarea 67 de F-23 (fase 3), decisión 66: crear y borrar stores dentro de un lote, para que un catálogo de josefina y
// sus stores nazcan o mueran juntos. Engine.StoreIn(tx, name) abre el store para el lote o lo crea dentro de él;
// Engine.DropStoreIn(tx, name) lo borra al confirmar. Store y DropStore sueltos siguen igual, salvo que un
// DropStore suelto también espera (con el motor suelto, hasta LOCK_TIMEOUT; 0 = sin límite; luego ErrBusy) a los lotes
// abiertos que escribieron en el store: sus escrituras se perderían en silencio (N4, tarea 111).
//
// Crear (createIn, con ddlMu): número de archivo nuevo, KindCreateStoreIn con el xid del lote (durable, como
// KindCreateStore: ningún archivo del store existe en disco antes que su registro), la entrada del catálogo con
// control.FlagPendingCreate (bit 0 de flags, que se guarda con el catálogo), el catálogo guardado y el árbol nuevo
// (KindInitStore); el *Store queda con creator = xid del lote. Las versiones que el lote escribe en él son las de
// cualquier lote.
//
// Borrar (markDrop, con ddlMu y Store.mu en escritura): no escribe nada en el WAL ni en el catálogo; solo anota
// dropper = xid del lote. Si otro lote abierto escribió en el store (Store.batchWriters, que suma cada lote la
// primera vez que escribe en un store y descuenta al terminar) devuelve ErrBusy sin abortar el lote, como Empty: sus
// versiones se perderían en silencio al borrar. Un store creado por el mismo lote se borra enseguida (KindDropStore).
//
// Commit (finishDDL, con ddlMu de punta a punta): el KindCommit lleva la cola de DDL (los archivos creados y los
// borrados; los borrados no van en las sumas de Count) y siempre espera el fsync (syncAlways), también con
// CommitDeferred y con SYNC_ON_WRITE=false: los archivos de un store borrado se borran y el catálogo se guarda solo
// con el commit durable, como en PostgreSQL (si no, un corte dejaría el lote abortado con su DDL hecho). Después, los creados
// pierden FlagPendingCreate (creator = 0, createdBy = xid) y los borrados se marcan inservibles (ErrStoreDropped),
// salen del catálogo, se guarda y se borran sus páginas de la caché y sus archivos. Como ddlMu está tomado desde
// antes de reservar el KindCommit hasta guardar el catálogo, el paso 3 del checkpoint (que toma ddlMu) guarda el
// catálogo antes o después del DDL, nunca en medio: si el punto de rehacer queda después del KindCommit, el catálogo
// guardado ya lo tiene aplicado y los archivos ya están borrados.
//
// Rollback, aborto (serialización, conflicto, TRANSACTION_TIMEOUT, disco lleno limpio): KindAbort; los borrados
// vuelven a estar como antes (dropper = 0, nada que deshacer en disco) y los creados se borran con un KindDropStore
// durable. Ese registro hace falta: sin él, al rehacer, el KindCreateStoreIn del store abortado ocuparía el nombre y
// taparía el KindCreateStore de un store suelto del mismo nombre creado después (Create no hace nada si el nombre
// existe). Si el motor está cerrado o inutilizable, el lote termina sin resolver su DDL y lo resuelve la
// recuperación.
//
// Recuperación: KindCreateStoreIn se rehace como KindCreateStore y, si agregó la entrada, le pone FlagPendingCreate;
// el KindCommit con cola quita la marca a sus creados y borra sus borrados por número de archivo (los saca del
// catálogo si todavía tienen ese número y borra sus archivos, también los que dejó un corte entre el commit y el
// borrado). Al terminar de rehacer, toda entrada que sigue con FlagPendingCreate es de un lote que no confirmó
// (ninguna transacción sobrevive a reabrir): se borra con sus archivos (redoState.dropPending) sin escribir en el WAL,
// y el checkpoint de fin de recuperación guarda el catálogo sin ella; un corte antes lo repite igual. Un borrado sin
// confirmar no dejó rastro en disco: el store sigue entero. Así no quedan archivos huérfanos que Verify reporte.
//
// Visibilidad y concurrencia (lo más simple y seguro):
//
//   - Un store que crea un lote abierto no está en Stores. Store(name) desde fuera espera hasta LOCK_TIMEOUT a que
//     ese lote termine: si confirmó, lo devuelve; si deshizo, lo crea suelto; si no terminó, ErrStorePending. Desde
//     el mismo lote se usa StoreIn (Store se esperaría a sí mismo). Las escrituras sueltas o de otros lotes en él
//     (con un *Store filtrado) dan ErrStorePending enseguida; las lecturas no ven nada (sus versiones son de una
//     transacción en curso) y DropStore da ErrStorePending.
//   - Dos lotes que crean el mismo nombre: el segundo StoreIn espera al primero (hasta LOCK_TIMEOUT sin pasar su
//     tiempo: si vence, ErrConflict o ErrTimeout y aborta). Si el primero confirmó, el segundo recibe
//     ErrSerialization (su snapshot no veía el store, como un id confirmado después); si deshizo, lo crea él. Un lote
//     que empezó después del commit lo abre normalmente.
//   - Un snapshot (o lote) tomado antes del commit de un store creado no ve sus datos: el store le aparece vacío, sin
//     lecturas a medias, porque sus versiones son del xid del lote. StoreIn y DropStoreIn de un lote así sí lo notan
//     (createdBy no visible): ErrSerialization.
//   - Mientras un lote borra un store, el store sigue entero: lecturas, recorridos, Count y escrituras sueltas siguen
//     (una suelta confirma antes que el borrado, así que perderla con él es el orden serial); las escrituras de otros
//     lotes dan ErrStorePending (confirmarían después del borrado y se perderían); otro DropStoreIn o StoreIn espera
//     al lote y, si confirmó, recibe ErrSerialization; DropStore suelto da ErrStorePending. Tras el commit, el
//     *Store da ErrStoreDropped, también a un snapshot viejo (como DropStore suelto: los archivos no se guardan).
//   - StoreIn o DropStoreIn de un store que el mismo lote borra dan ErrStoreDropped.
//   - Empty no puede cruzarse con nada de esto: exige que no haya lotes abiertos.
//
// Pruebas (ddlbatch_test.go): TestStoreInCommit, TestStoreInRollback (Rollback, tiempo máximo, aborto por
// serialización), TestDropStoreIn, TestStoreInDropOwn, TestStoreInSameName, TestDropStoreInBusy,
// TestDDLBatchOpenAtClose (Close o corte con el lote abierto y un checkpoint en medio), TestDDLBatchCrash (un corte en
// cada operación de disco de un lote que crea, escribe, borra y confirma, en los tres modos: estado de antes o de
// después entero, Count, sin huérfanos y Verify limpio) y TestDDLBatchConcurrent (con -race); el modelo de
// TestModelCrash abre los stores de sus lotes con StoreIn (crea dentro del lote) y a veces borra con DropStoreIn.
//
// A confirmar (tarea 67): borrar y volver a crear el mismo nombre dentro de un lote no se admite (ErrStoreDropped);
// un snapshot viejo ve vacío un store creado después en vez de «no existe»; DropStoreIn rechaza (ErrBusy) en vez de
// esperar a los lotes que escribieron en el store; los stores borrados por un lote no se recuerdan, así que un
// StoreIn posterior de un lote viejo que no esperó a ese borrado crea uno nuevo.
//
// # Recorridos, Exists y Empty
//
// Un recorrido es un btree.Cursor (btree/doc.go): lee una hoja por llamada a Next y entre una y otra no tiene nada
// fijado ni bloqueado. Por cada hoja, el motor toma Snapshot.mu (lectura; un Close o el tiempo del snapshot esperan a
// esa hoja), closeMu y Store.mu (lectura), lee la hoja y suelta todo; el valor de desborde de cada id se lee igual,
// de a uno; recién entonces llama a fn. Así fn corre sin páginas fijadas ni locks del motor («nunca esperar a una
// transacción ni a un fsync con una página bloqueada» no depende de lo que haga fn) y puede usar el motor, incluso
// ReadAt con el mismo snapshot o cerrarlo (el recorrido termina con ErrTxDone en la hoja siguiente). Lo que ocupa
// en RAM es una hoja de versiones (y un valor de desborde a la vez), no crece con el store; solo Keys acumula lo
// que devuelve. Las hojas siguientes a la primera se fijan con buffer.Ring: un recorrido grande no vacía la caché.
// Los recorridos sueltos (ForEach, Keys…) toman un snapshot propio sin tiempo máximo y lo cierran con defer, también
// si fn entra en pánico; con un snapshot, la limpieza no quita nada de lo que el recorrido puede ver.
//
// ForEachBorrowed (prestado): la hoja se copia entera a un búfer del cursor que se reutiliza de hoja en hoja y se
// suelta antes de llamar a fn; data apunta a ese búfer (sin una copia por valor) y vale solo dentro de fn. Se eligió
// esto en vez de llamar a fn con la página fijada: con la página fijada fn no podría esperar a nada del motor sin
// arriesgar un bloqueo, y una regla así es fácil de romper desde josefina. Es ascendente porque, hacia atrás, el
// primer id de cada hoja sobrevive a la hoja siguiente y su valor no puede vivir en el búfer reutilizado.
//
// Prefijo y rango: ForEachRange recorre [from, to) ("" = sin límite); ForEachPrefix y KeysPrefix usan [prefix,
// prefixEnd(prefix)), donde prefixEnd quita los 0xFF del final y suma uno al último byte (sin límite si no queda
// ninguno). Hacia atrás el cursor baja hasta to (incluido en btree) y saltea ese id.
//
// Empty (como TRUNCATE): reserva un número de archivo nuevo, escribe KindEmptyStore (durable), cambia el catálogo
// (control.Stores.Empty: archivo nuevo, contador 0) y lo guarda, quita las páginas viejas de la caché, borra los
// archivos viejos y crea el árbol nuevo (KindInitStore), todo con ddlMu; el *Store sigue sirviendo con el archivo
// nuevo (file y tree cambian con Store.mu en escritura). La recuperación lo rehace como DropStore: si el store
// todavía tiene el archivo viejo pasa al nuevo con contador 0, y borra los archivos del viejo si ya no son de nadie;
// los bloques del viejo anteriores al registro se saltan si el catálogo guardado ya tiene el nuevo. Con lotes y
// snapshots: PostgreSQL toma un lock exclusivo, espera a las transacciones que usan la tabla y los snapshots viejos
// ven la tabla vacía (TRUNCATE no es MVCC). Acá un lote o snapshot es de todo el motor y no se sabe qué stores va a
// leer, así que Empty exige que no haya ninguno abierto (Engine.openTx, que suman newBatch y snapshotFor con txGate
// en lectura): si hay, devuelve ErrBusy enseguida; mientras vacía tiene txGate en escritura y no empieza ninguno.
// Rechazar en vez de esperar evita que quien llama se bloquee contra su propio snapshot (un Empty dentro de un
// recorrido, o con un lote suyo abierto) y es MVCC-seguro: nadie puede ver el store antes y después. Las operaciones
// sueltas en curso del store, la limpieza y Count se esperan con Store.mu.
//
// A confirmar (fase 2): Empty es de todo el motor respecto de lotes y snapshots (un lote que nunca toca el store
// también lo frena); el borrado de los archivos viejos no espera al checkpoint (como DropStore); un corte que pierde el
// borrado deja archivos huérfanos de un número que ya nadie usa (Verify los encuentra y Repair los borra); KeysAfter y
// offset recorren lo anterior dentro de la página (offset es O(offset)); un recorrido dentro de un lote no se hace
// (ForEachIn, pregunta 64).
//
// # Escritura diferida y versiones
//
// Escritura diferida (tareas 38 y 39 de F-23; como synchronous_commit = off de PostgreSQL): Store.Deferred() da
// Put, Remove, Insert, Update y Delete que confirman sin esperar el fsync, Batch.CommitDeferred es Commit sin
// esperarlo y Engine.WaitDurable espera a que todo lo confirmado antes de llamar (cualquier store: el WAL es uno) esté
// en disco. El orden de un cambio es el mismo salvo el paso 10: se reserva, escribe y publica el registro, se anota
// su lsn en Engine.lastCommit (el mayor, con CAS) y se marca confirmada en el archivo de estados sin esperar el
// fsync. lastCommit sube antes de marcar confirmada: todo lo visible al llamar a WaitDurable está por debajo, y
// WaitDurable es wal.WaitDurable(lastCommit), que espera la publicación de lo anterior y hace un fsync compartido
// (group commit). Las confirmaciones durables y las diferidas usan el mismo camino; lo único que cambia es la espera.
//
// Garantías y por qué valen:
//
//   - Visibilidad: lo diferido se ve enseguida (lecturas sueltas, lotes y snapshots nuevos), aunque no sea durable,
//     como en PostgreSQL. Una lectura o una condición que no escribe (un Insert que encuentra el id, un Update sin
//     cambios) no espera el fsync de lo que vio: si un corte pierde la versión diferida, lo que contestó deja de valer.
//   - Atomicidad: una escritura suelta es un registro KindLeafWrite con FlagCommit (más, antes, los de su desborde) y
//     un lote confirma con un único KindCommit. La recuperación rehace un prefijo contiguo del WAL (lsn = anterior + 1,
//     CRC; el primer registro inválido corta), así que un cambio diferido queda entero o no queda: sin su registro de
//     hoja no hay versión (un desborde ya escrito queda huérfano, como ya pasaba), y sin el KindCommit las versiones
//     del lote son de un xid que se lee abortado.
//   - Orden: lo durable es siempre un prefijo del WAL (durable sube solo sobre lo publicado contiguo), así que si una
//     confirmación durable sigue a una diferida en el WAL, la diferida también queda en disco.
//   - «WAL antes que la página»: la caché, el escritor de fondo y el checkpoint esperan el WAL durable hasta el pd_lsn
//     de cada página antes de escribirla, y el archivo de estados hasta el mayor lsn de confirmación de sus grupos: una
//     página o un estado de una confirmación diferida nunca llegan al disco antes que su registro. El checkpoint
//     espera además el WAL durable hasta el punto de rehacer (waitChanges) antes de guardar el catálogo con los
//     contadores, así Count tras un corte es el de lo que sobrevivió.
//   - Regla 49 (ninguna marca de confirmada antes de que el lsn de confirmación sea durable): btree solo pone
//     EntryXidCommitted si xact.HintAllowed(estado, lsn de grupo, wal.Durable()); con una confirmación diferida el lsn
//     de su grupo (el mayor de los 32 xids) está por encima de Durable y la marca no se pone hasta que el fsync llega
//     (TestDeferredHintWaitsDurable). La limpieza decide por la marca o, sin ella, por el estado, y lo que quita va en
//     el WAL detrás de la confirmación que hizo muerta a la versión vieja: no puede llegar al disco sin ella.
//
// El escritor de fondo del WAL (walWriter, como el walwriter de PostgreSQL) hace cada WAL_WRITER_DELAY (200 ms; 0 lo
// apaga) un WaitDurable si
// lastCommit pasó a Durable: lo que puede perder un corte queda acotado a esa pausa más un fsync, también con
// SYNC_ON_WRITE=false. SYNC_ON_WRITE=false vuelve diferidas todas las confirmaciones del motor (Put, Commit…);
// Deferred y CommitDeferred lo son siempre, también con SYNC_ON_WRITE=true, que es la forma de diferir solo a quien
// agrupa sus propias confirmaciones y mantener durable al resto. Los registros de crear, vaciar y borrar un store y
// el de checkpoint esperan siempre su fsync (y con él lo diferido anterior).
//
// Forma frente a josefina (internal/pebble): allí KV.Deferred() da un Writer (Insert, Update, Delete) y WaitDurable
// está en cada KV y en cada Index pero sincroniza el WAL único del nodo; Commit(b) de un lote nunca espera. Acá
// Deferred es un valor del store con las mismas operaciones más Put y Remove (sirve como ese Writer), WaitDurable
// está en el Engine porque el WAL es del motor (un método por store prometería algo que no es por store), y el lote
// tiene Commit durable y CommitDeferred, en vez de un modo del lote fijado al crearlo, porque se decide al confirmar.
//
// Versiones (concurrencia optimista): GetVersion, GetVersionIn y GetVersionAt son Read, ReadIn y ReadAt con el lsn de
// la versión devuelta, que ya está en la clave de la entrada (id ascendente, lsn descendente): no cuesta nada más.
// Cambia con cada escritura que aplica (un Update sin cambios o un Insert que no escribe no la cambian), nunca baja,
// sobrevive a checkpoints, limpieza, particiones y reabrir, y es 0 si el id no existe; la de un cambio propio de un
// lote es la que verán los demás si confirma. Como el lsn sigue al WAL, tras un corte que pierde una escritura
// diferida su número puede volver a usarse para otra: una versión leída de algo diferido solo vale mientras no haya
// un corte o tras WaitDurable. No hay UpdateIfVersion/DeleteIfVersion: el patrón es un lote (GetVersionIn y la
// escritura en el mismo lote; si otra transacción cambió el id después de empezar el lote, la escritura da
// ErrSerialization, TestOptimisticBatchPattern), y josefina compara sus versiones bajo su propio lock de modelo y
// guarda la suya dentro del valor (decisión 46), así que una escritura condicional por versión no tendría quien la use.
//
// A confirmar (fase 2): que una lectura no espere el fsync de lo diferido que ve (como PostgreSQL); que GetVersion
// devuelva el lsn y no un contador por id.
//
// # Orden de los bloqueos
//
//   - Entre páginas: de izquierda a derecha, de hija a padre; nunca al revés. Páginas del árbol → página meta → página
//     libre (fsm); página de datos → su página del mapa de visibilidad.
//   - Pedir una página (buffer.Pool.Pin) puede esperar al WAL (reemplazar una víctima sucia), pero nunca los bloqueos
//     de la víctima: si otro la fijó después de que el clock-sweep la eligiera y tiene su contenido o su escritura (io)
//     tomados, el reemplazo la suelta y elige otra (TryLock, como LWLockConditionalAcquire en BufferAlloc). Al bajar o
//     moverse a la derecha se suelta la página antes de pedir la siguiente (Lehman-Yao). Pedirla con otra bloqueada
//     solo se hace como nbtree, en el orden de arriba (al partir: la página nueva de fsm y la antigua hermana derecha;
//     al terminar una partición: el padre y la meta; en la raíz nueva: la meta; en Latest: la hermana derecha), y
//     nunca entre Reserve y Write de un registro: así la única espera de Pin es al WAL, que no toma páginas, y no puede
//     cerrar un ciclo, solo alargar el bloqueo.
//   - Consultar el estado de una transacción (xact) con una hoja bloqueada en compartido puede leer el disco o esperar
//     al WAL (como clog en PostgreSQL); con un bloqueo exclusivo, igual.
//   - Dentro de un búfer: io (escritura al disco) → contenido → encabezado.
//   - Página (bloqueo de contenido) → writeMu → drainMu → pubMu; writeMu → syncMu; syncMu → pubMu; leadMu (quién hace
//     el fsync del group commit) es una hoja. Nunca writeMu → página. Partir una hoja pide antes su página nueva con
//     la hoja suelta (fsm.Grow, tarea 100): la extensión del archivo no va con páginas del árbol bloqueadas.
//   - Réplica (tareas 85 y 104), un solo orden: closeMu (lectura) → ckptMu → applyMu → replayMu → ddlMu → Store.mu.
//     Lo siguen el restartpoint (ckptMu → applyMu), Promote (ckptMu → applyMu → replayMu → ddlMu), el aplicador
//     (closeMu → applyMu → … → Store.mu) y Verify (quietReplica: applyMu después de ckptMu en checkFiles y después de
//     closeMu en cada store, nunca alrededor de todo: tomarlo antes de ckptMu cerraba un ciclo con el restartpoint,
//     TestReplicaVerifyLockOrder).
//   - Nunca se espera a otra transacción (xact.Manager.Wait) ni a un fsync con una página bloqueada. Un pin sí se puede
//     tener mientras tanto. El fsync del checkpoint (smgr Sync) va con el candado del fork suelto, así una extensión
//     (fsm.Allocate → PinNew → smgr Extend, con páginas del árbol bloqueadas) no lo espera. La excepción documentada:
//     cuando una extensión hace nacer un archivo de 1 GB, smgr Extend hace fsync del anterior bajo el fork en
//     exclusivo (una vez por GB de crecimiento), para que un corte no deje un archivo corto en medio del fork.
//   - Los locks del mapa de la caché, de la caché del archivo de estados y del catálogo en memoria son hojas: no se
//     toma otro con ellos tomados. El catálogo nunca se toca con una página bloqueada.
//   - DDL en lote: Batch.mu → closeMu (lectura) → ddlMu → Store.mu (StoreIn, DropStoreIn); al terminar un lote con DDL,
//     Batch.mu → closeMu (lectura) → ddlMu → commitMu (lectura) → writeMu, y después (sin commitMu) ddlMu → Store.mu.
//     Las esperas a otro lote (xact Wait) se hacen sin Batch.mu, closeMu, ddlMu, Store.mu ni páginas.
//   - Locks del motor: closeMu (lectura) → ddlMu → txGate → Store.mu (Empty); closeMu (lectura) → txGate (lectura)
//     (newBatch, Snapshot); closeMu (lectura) → Store.mu (lectura) → commitMu (lectura) → página → writeMu → contadores
//     (hoja); ckptMu → commitMu (escritura); ckptMu → ddlMu; ckptMu → HintGate (escritura) → writeMu. Una escritura,
//     suelta o de un lote, que choca con otra transacción en curso suelta todo (commitMu, Store.mu, closeMu y
//     Batch.mu; su desborde ya escrito se libera) antes de esperarla, como XactLockTableWait de PostgreSQL, y repite
//     desde la entrada (el store pudo borrarse o vaciarse; el lote, terminar o abortar). WaitDurable y el escritor del WAL: closeMu (lectura)
//     → syncMu (y pubMu al esperar la publicación), sin páginas. Verify y Repair: closeMu (lectura) → ckptMu → ddlMu
//     (control, catálogo y data/); Store.vacuumMu → closeMu (lectura) → Store.mu (escritura) → commitMu (escritura,
//     solo para repetir el conteo) → páginas. Backup: ckptMu → writeMu (wal.ForceImages, wal.Hold) y el checkpoint →
//     ddlMu (catálogo y lista de data/); sin closeMu (cuenta en Engine.bg); backups.mu es una hoja. También cuentan en
//     Engine.bg sin closeMu ChangeReader.Next y la espera de CreateLogicalSlot (tarea 112). Archivo continuo:
//     closeMu (lectura) → archive.mu → writeMu (SetArchived, wal.Hold); ckptMu → archive.mu → writeMu (el tope, en el
//     paso 6); recover → archive.mu → writeMu (wal.Switch). Ranuras: closeMu (lectura) → slots.mu → writeMu (Hold);
//     ckptMu → slots.mu (paso 6). Réplica: applyMu → txGate (escritura, para un registro que choca con los
//     snapshots) → replayMu (escritura) → ddlMu → Store.mu; las lecturas de páginas closeMu (lectura) → replayMu
//     (lectura) → Store.mu (Store.begin); Verify y el restartpoint toman applyMu después de ckptMu (ver Réplica).
//   - Marcas de visibilidad: con bloqueo compartido, por OR atómico (y leídas con atomic.LoadUint32); la escritura de
//     la página al disco calcula la suma sobre una copia. La marca de confirmada solo se pone si el WAL es durable
//     hasta el lsn de confirmación del grupo de su xid (pregunta 49). La primera marca en una página tras un checkpoint
//     se anota como KindSetHints (con la imagen de la hoja), con la hoja bloqueada en exclusivo después de soltar el
//     bloqueo compartido de la lectura; la lectura toma HintGate en compartido para decidir y marcar, y el checkpoint
//     la toma en exclusivo alrededor de BeginCheckpoint (como DELAY_CHKPT_START): página → HintGate; HintGate →
//     writeMu.
//   - Limpieza: Store.vacuumMu → closeMu (lectura) → Store.mu (lectura) → páginas (una hoja a la vez; antes de cada
//     una mira si un DDL espera Store.mu con ddlMu tomado —Store.lockDDL sube Store.ddlWaiting— o si Close cerró
//     stop, y se corta: btree.VacuumOptions.Stop, V2 y V3, tarea 113; con ella, en
//     compartido, su hermana derecha para ver si comparte un id, y la página del mapa para poner su bit) → writeMu; la
//     cadena de desborde se libera con la hoja ya soltada; la meta (KindSetFrozen) al final, sin hojas. Una escritura o
//     una partición que encuentra la hoja con page.FlagAllVisible borra su bit (página del mapa) con la hoja bloqueada,
//     antes de reservar su registro. Engine.frozen.mu es una hoja.
//   - Borrado de páginas (como nbtree, de izquierda a derecha y de abajo hacia arriba): marcar medio muerta bloquea la
//     hoja (antes, sin ella, mira en compartido su hermana izquierda), con ella en compartido su hermana derecha, la
//     página del mapa (si hay que borrar su bit) y, bajando desde la raíz con la hoja bloqueada, el padre del subárbol
//     (como la búsqueda del padre al terminar una partición: de hija a padre); desenganchar bloquea la hoja (si se
//     borra un padre), la hermana izquierda de la que se borra (siguiendo a la derecha), la que se borra, su hermana
//     derecha y la meta; reciclar, la meta y después la página borrada. Antes de reciclar, la limpieza espera sin nada
//     bloqueado a las operaciones del árbol en vuelo (btree.opGuard); ninguna operación espera a la limpieza con una
//     página bloqueada.
//
// # Checkpoint
//
// Se dispara cada CHECKPOINT_TIMEOUT (5 min) o cuando el WAL desde el punto de rehacer del último checkpoint terminado
// pasa MAX_WAL_SIZE / (1 + 0,9) (con 1 GB, ~539 MiB): así lo que hay que reproducir tras una caída ronda MAX_WAL_SIZE
// (pregunta 28). Un fallo en cualquier paso deja el motor inutilizable, salvo el disco lleno limpio en los pasos 2
// (archivo de estados) y 3 (catalog.tmp): el checkpoint no cuenta y se reintenta (ver «Disco lleno o lento»). Al
// terminar despierta a los escritores frenados por el WAL sin reciclar. Pasos:
//
//  1. wal.Log.BeginCheckpoint, bajo writeMu: el punto de rehacer pasa a la posición del próximo registro; ahí se toman
//     nextXID, el xid vivo más viejo y los contadores de Count (consistentes con ese lsn, porque se suman bajo
//     writeMu). Desde ese momento la primera escritura de cada página lleva su imagen entera.
//  2. Escribir las páginas sucias, repartido en el 90 % del intervalo (CHECKPOINT_COMPLETION_TARGET = 0,9), cada una
//     tras el WAL durable hasta su pd_lsn; las páginas del archivo de estados; smgr.SyncAll y fsync de xact. Una página
//     que otro (reemplazo, escritor de fondo) está escribiendo sigue sucia hasta que su escritura termina, y FlushAll
//     la espera: el fsync nunca se adelanta a una escritura en curso (buffer/doc.go).
//  3. Guardar el catálogo con los contadores y countLSN = el lsn del punto de rehacer (la copia del catálogo y de los
//     contadores se toma en el paso 1, en el mismo instante, bajo writeMu; si no, la recuperación contaría dos veces).
//  4. Escribir el registro KindCheckpoint y esperar su fsync.
//  5. Escribir el control (seq + 1) con el punto de rehacer, checkpointLSN, nextXID, el xid vivo más viejo y el
//     frozenXID del motor (tomado al empezar el paso 1, antes del punto de rehacer).
//  6. Con el archivo continuo y WAL_ARCHIVE_MAX_SIZE, aplicar su tope (enforceArchiveCap). Reciclar los segmentos del
//     WAL anteriores al punto de rehacer (respetando los segmentos que retienen un respaldo en curso, el archivo
//     continuo y las ranuras, wal.Hold; antes, checkpointSlots invalida las que pasan MAX_SLOT_WAL_KEEP_SIZE, guarda
//     el archivo slots con los avances y recién entonces mueve sus retenciones; ninguna pasa nunca lo guardado): se
//     renombran como repuestos hasta
//     wal.Options.Spares (MAX_WAL_SIZE / 2 / WAL_SEGMENT_SIZE, 4 por omisión) y el resto se borra.
//  7. Truncar el archivo de estados por debajo de ese frozenXID (xact Truncate: los archivos enteros, de menor a
//     mayor), o del frozenXID del control de un respaldo en curso si es menor.
//
// Un corte entre 3 y 5 deja el control con el punto de rehacer anterior y el catálogo con el nuevo: la recuperación
// suma a los contadores solo las confirmaciones con lsn ≥ countLSN, así que no cuenta dos veces. Close hace un
// checkpoint final con state = cerrado bien (antes, si hace falta, la limpieza al cerrar: ver «Limpieza»).
//
// Entre los pasos 1 y 2 el checkpoint espera a las operaciones que reservaron su registro antes del punto de rehacer
// y todavía no anotaron su estado en el archivo de estados (commitMu, como delayChkpt de PostgreSQL): si no, su
// confirmación quedaría antes del punto de rehacer y fuera del archivo de estados guardado, y tras un corte se leería
// como abortada. El catálogo que guardan crear y borrar un store (fuera del checkpoint) lleva los contadores del
// último checkpoint con su countLSN, nunca los vivos: así sigue siendo coherente. El checkpointer de fondo mira cada
// segundo si toca uno; el de Checkpoint(), el de Close y el de fin de recuperación no se reparten. Close (antes de
// pedir closeMu) y un Checkpoint() que espera ckptMu apuran al repartido en curso (Engine.ckptHurry: pacer deja de
// dormir), así Close no queda detrás de todo el reparto con el motor detenido (N3, tarea 110).
//
// Cambios ya reservados: en el paso 1, tras fijar el punto de rehacer, el checkpoint espera el WAL durable hasta el lsn
// anterior (Engine.waitChanges). Cada cambio de página reserva su registro, aplica, marca sucia y recién después
// escribe el registro; sin esa espera, un cambio reservado antes del punto de rehacer y marcado sucio después de que
// FlushAll juntó las sucias quedaba fuera del checkpoint y no se rehacía (PostgreSQL lo evita marcando sucia antes de
// insertar el registro).
//
// Marcas de visibilidad: todos los árboles reciben el mismo HintGate (Engine.hintGate, un sync.RWMutex) y el paso 1
// del checkpoint lo toma en exclusivo alrededor de wal.BeginCheckpoint.
//
// # Recuperación (Open)
//
// En el orden de engine.go (open) y recovery.go (recover):
//
//  1. Tomar LOCK (vfs.ErrLocked → ErrLocked). Antes de leer o cambiar nada más, checkFormat: un archivo de la raíz
//     de otra versión del formato es ErrFormatVersion (ver «Versiones del formato»).
//  2. Leer backup_label si lo hay (una copia de Backup; dañado o sin fin: ErrBackupIncomplete) y el control (la copia
//     válida más nueva). Un motor es nuevo si no tiene archivo de control, o si su control no tiene copias válidas y el
//     catálogo no tiene stores (un Create cortado); uno nuevo con marcador es ErrBackupIncomplete. Con marcador,
//     comprobar sin escribir nada que el WAL llega hasta su KindBackupEnd (ver «Respaldo en caliente y
//     restauración»); con un objetivo (OpenAt), comprobar también sin escribir que se puede alcanzar
//     (ErrRecoveryTarget).
//  3. Leer el catálogo (vacío en un motor nuevo) y cargar sus contadores. Abrir la caché y el archivo de estados antes
//     que el WAL (los usa el rehacer), con un Flusher (walProxy) que cuenta todo como durable hasta que wal.Open
//     termina. xact Truncate(frozenXID del control): fija el límite aunque falten archivos y termina un truncado que
//     un corte dejó a medias.
//  4. wal.Open(fs, dir, opciones, redo, rehacer) recorre desde el punto de rehacer (wal.Start = segmento 1,
//     desplazamiento 32 —tras el encabezado del segmento—, lsn 1 en un motor nuevo); en uno que no es nuevo debe aparecer el registro del checkpoint (si
//     no, ErrCorrupt). Por cada registro: crear, vaciar y borrar un store se rehacen sobre el catálogo (idempotentes:
//     borrar y vaciar llevan el número de archivo esperado, y solo actúan si el store todavía lo tiene; el borrado
//     borra los archivos del número si ya no es de ningún store, también los que dejó un corte); los bloques de un
//     store borrado se saltan; con imagen, se restaura sin condiciones y se aplica el delta (los bloques con imagen o
//     BlockWillInit se fijan con buffer.Pool.PinZero, sin leer la página del disco, que puede estar rota por el
//     corte); sin imagen, se aplica solo si el pd_lsn de la página es menor que el lsn del registro (idempotente),
//     con la función de rehacer del dueño del Kind (btree.Redo recibe todo el rango 16–31, overflow 32–39, fsm 40–47);
//     confirmaciones y abortos van al archivo de estados (xact SetStatus; la confirmación de un xid ya truncado no
//     anota nada, un aborto así es corrupción), y una confirmación con lsn ≥ countLSN suma a los contadores. Las
//     escrituras sueltas (KindLeafWrite con FlagCommit) anotan confirmada su xid al rehacerlas.
//  5. wal.Open corta el WAL en su primer registro inválido (o tras el registro del objetivo de OpenAt,
//     wal.Options.StopLSN) y deja en largo 0 los segmentos siguientes (un registro viejo nunca confirmado no puede
//     pasar por válido después). Un registro inválido en un segmento sellado es corrupción: Open falla (la marca de
//     sellado de cada segmento dice hasta dónde llegan sus registros; ver wal/doc.go, «Marca de sellado»).
//     Tras un cierre limpio no corta (tarea 109): con el control en StateShutdown, sin marcador de respaldo, sin
//     objetivo y fuera de una réplica, el motor pasa wal.Options.CleanEnd = CheckpointLSN + 1 (el WAL termina en el
//     checkpoint de Close) y, si el recorrido termina justo ahí, wal.Open no trunca el activo ni los repuestos, que
//     siguen preasignados (ahorra un truncado + fsync por segmento al abrir y volver a llenar de ceros el resto del
//     activo y cada repuesto: hasta 128 MB + fsync cada uno). Es seguro porque tras un Close no hay en ningún segmento
//     un registro con lsn ≥ el final (Close espera toda reserva; lo que queda después son ceros o registros viejos
//     de lsn menor, que el recorrido descarta como siempre, y los repuestos tienen el encabezado de otro número). Para
//     que siga siéndolo, recover anota el control en StateInProduction (markInProduction, con fsync) antes de escribir
//     nada en el WAL: si un corte deja registros nuevos, el control ya no dice cerrado bien y el próximo Open corta.
//     Si el final no coincide (un corte entre el WAL y el control) o un segmento mide más que WAL_SEGMENT_SIZE (otro
//     tamaño), se corta como siempre.
//  6. loadSlots lee el archivo slots y pone sus retenciones (una cuyo lsn ya no está en el WAL queda invalidada). Con
//     WAL_ARCHIVE=on (no en una réplica: Promote lo arranca al promoverla), startArchive lee archive_status, retiene desde lo no archivado y sella el segmento en que el
//     WAL quedó (ver «Archivo continuo del WAL y recuperación a un punto en el tiempo»).
//  7. Borrar los stores que siguen con FlagPendingCreate (los creó un lote cuyo KindCommit no llegó a ser durable),
//     con sus archivos, sin escribir en el WAL (redoState.dropPending; ver «DDL dentro de un lote»).
//  8. nextXID = el mayor entre el del control, el mayor xid visto + 1 y el mayor frozenXID que ponen los KindInitStore
//     y KindSetFrozen rehechos (btree.RecordFrozenXID: un corte puede perder los xid asignados después de esos
//     registros, y un xid reusado menor que el frozenXID de un store rompería su promesa). Los xid menores en estado
//     «en curso» se leen como abortados (eran transacciones en curso al caerse), sin marcarlos uno por uno: xact
//     Recover(nextXID) fija ese horizonte y pone en cero los estados desde nextXID (como TrimCLOG de PostgreSQL).
//  9. Checkpoint de fin de recuperación (sin repartir) y state = en producción; en un motor nuevo guarda el catálogo y
//     crea el control (control.Create). Con marcador, borrarlo: la copia ya es un motor más.
//
// Costo de Open tras un Close limpio (tarea 69, TestOpenBench): no toca data/ más que con un ReadDir (los archivos de
// datos se abren al usarlos); lee el control, el catálogo, ~1 MB del segmento activo del WAL, los encabezados de los
// repuestos y una página del archivo de estados, y hace 7 fsync (control ×2, catalog.tmp, la raíz, el registro de
// checkpoint, el archivo de estados y xact/). Con F_FULLFSYNC de macOS (~4 ms cada uno) esos fsync son casi todo:
// ~25 ms con 1, 10 o 100 millones de claves, igual E/S en los tres. Lo único que crece con los datos es el ReadDir de
// data/ (un archivo por GB).
//
// Las particiones incompletas las termina la próxima inserción que las encuentre; los borrados de páginas a medias (una
// hoja medio muerta), la próxima limpieza (o Repair). Un store del catálogo cuya página
// meta no existe o está en ceros (su KindInitStore no llegó a ser durable) se crea de nuevo al abrirlo con Store:
// ningún registro durable apunta a sus páginas.
//
// A confirmar (ola 4): el Flusher de la recuperación cuenta como durable lo leído del WAL (como PostgreSQL).
//
// # Limpieza en el motor
//
// Vacuum toma el horizonte (xact OldestActive: el xid en curso más viejo o el Xmin más viejo de un lote o snapshot
// abierto) una vez al empezar y limpia los stores del catálogo de a uno con btree.Vacuum(Horizon, FreezeLimit,
// Aggressive, XID 0, Free = overflow.Manager.FreeTail de la versión muerta), que además congela y salta hojas por el
// mapa de visibilidad (ver «Congelado y truncado del archivo de estados»). Un lote o snapshot que empiece después tiene
// Xmin ≥ ese horizonte, así que nada de lo que se quita le hace falta. Cada store se limpia con Store.vacuumMu (btree
// no admite dos limpiezas del mismo árbol) y con el store tomado en lectura como cualquier operación: un DropStore que
// llega espera a que termine (y frena mientras tanto las operaciones nuevas del store, por el RWMutex), y un store ya
// borrado se salta; un store que se borró entre la lista y su turno no se vuelve a crear (Engine.existing). Si un store
// falla (una página dañada) se sigue con los demás y se devuelve el primer error; un fallo de escritura del WAL o de la
// caché deja el motor inutilizable como cualquier escritura, y con el motor cerrado o inutilizable se detiene.
//
// La limpieza de fondo (como autovacuum, con un criterio simple, a confirmar) mira cada VACUUM_INTERVAL si hubo al
// menos VACUUM_THRESHOLD escrituras aplicadas (sueltas o de lotes, también las deshechas: dejan versiones muertas)
// desde su última vuelta, o si se asignaron medio archivo de estados de xids (para que el congelado y el truncado sigan
// también con transacciones que no escriben), y si es así corre Vacuum. Close marca el motor cerrado (espera a la
// limpieza del store en curso; la siguiente ve ErrClosed), detiene la de fondo y recién después hace el checkpoint
// final; si el motor quedó inutilizable, la de fondo termina sola.
//
// Limpieza al cerrar (closeVacuum, tarea 124, 2026-11-22): el intervalo cuenta desde Open, así que un proceso que se
// reinicia más seguido que VACUUM_INTERVAL nunca limpiaba y data/ crecía (test/soak: 1,3 → 3,3 × los vivos en 2 min).
// Ahora Close, antes de marcar el motor cerrado (con el checkpointer y el escritor del WAL vivos, así un escritor
// frenado se suelta), corre Vacuum si la limpieza de fondo está encendida (VACUUM_INTERVAL > 0) y hubo al menos
// VACUUM_THRESHOLD escrituras desde la última limpieza: el mismo contador del motor (Engine.writes, que empieza en 0
// en cada Open) que mira la de fondo; el criterio por xids no se mira al cerrar. Una réplica no limpia; un motor
// cerrado, inutilizable (ErrFailed) o un disco lleno limpio (ErrDiskFull) no limpian y el cierre sigue igual; otro
// error de la limpieza (una página dañada) se suma al de Close. No toca el formato en disco.
//
// A confirmar (ola 5C): los valores de VACUUM_INTERVAL y VACUUM_THRESHOLD y el criterio por cantidad de escrituras
// (PostgreSQL usa la fracción de tuplas muertas por tabla); una cadena de desborde escrita antes de un corte sin que
// llegara su hoja queda huérfana (ninguna versión la nombra y la limpieza no la ve; la prueba de modelo las cuenta):
// Repair las devuelve a la lista (fase 3, «Verificación y reparación»); DropStore espera a la limpieza del store en
// curso en vez de cortarla.
//
// # Borrado de páginas del árbol
//
// Tarea 70 de F-23 (bloque A), como _bt_pagedel de nbtree (detalle y formatos en btree/doc.go, «Borrado de páginas»).
// La limpieza devuelve al espacio libre las hojas que deja vacías y los padres que se quedan sin hijos, en tres pasos,
// cada uno un registro del WAL con su Redo y cada estado intermedio un árbol válido:
//
//  1. Medio muerta (KindMarkHalfDead, como _bt_mark_page_halfdead): una hoja vacía que no es la de más a la derecha de
//     su nivel ni la raíz, sin partición incompleta (ni ella ni su hermana izquierda hacia ella) y cuya hermana derecha
//     no está medio muerta. Se sube por los padres de un solo hijo hasta la cima del subárbol que se borra (un padre que
//     se puede borrar: ni el de más a la derecha de su nivel, ni la raíz, ni partido a medias); en el padre de la cima,
//     el enlace a la cima pasa a ser el de su hermana derecha y se quita el de la hermana: el espacio de claves del
//     subárbol pasa a la hermana derecha. Si la cima es el último enlace de un padre con más hijos no se borra (su
//     hermana es de otro padre); la próxima pasada lo intenta otra vez.
//  2. Desenganche (KindUnlinkPage, como _bt_unlink_halfdead_page), de arriba hacia abajo: la cima, cada padre de un
//     solo hijo y por último la hoja. La izquierda y la derecha se enlazan entre sí; la página queda borrada con sus
//     enlaces (un lector que todavía llega a ella sigue a la derecha), con safeXID = el próximo xid (el safexid de
//     nbtree) y apilada en la meta. Si la hermana derecha queda sola en su nivel, pasa a ser la raíz rápida.
//  3. Reciclado (KindRecyclePage), al empezar una pasada posterior: mientras la de arriba de la pila tenga safeXID <
//     horizonte, la saca de la pila y la deja en la cabeza de la lista de libres (formato de fsm).
//
// Un lector nunca cae en una página reciclada: (a) toda operación suelta del árbol (Lookup, Exists, Write, cada Next
// de un Cursor, Verify) se cuenta en btree.opGuard, de tamaño fijo, y antes de reciclar la limpieza espera, sin nada
// bloqueado y hasta 2 s, a las que estaban en vuelo (si no terminan, recicla en otra pasada): ninguna puede tener un
// enlace leído antes del desenganche; (b) un recorrido entre dos Next no se cuenta, pero retiene el horizonte con su
// snapshot (todo recorrido del motor tiene uno): safeXID ≥ su Xmin, así que la página no se recicla mientras siga
// abierto. Las páginas que se están borrando (medio muertas o borradas, P_IGNORE de nbtree) las salta toda bajada o
// recorrido siguiendo a la derecha; un recorrido hacia atrás va a la izquierda como _bt_walk_left (si la página de
// partida se borró, sigue a su derecha hasta una no borrada y busca la izquierda de esa); uno hacia adelante salta las
// versiones de ids que ya pasó (el espacio de claves de una hoja borrada pasa a la derecha). Sin claves saltadas ni
// repetidas (TestPageDeleteConcurrent en btree, TestPageDeleteReplica).
//
// Cortes: un corte entre los pasos deja una hoja medio muerta (y quizá parte de su cadena todavía sin desenganchar);
// la próxima limpieza la termina —también una que quedó a la izquierda de todo, sin enlace en el padre, que busca por
// los enlaces izquierdos de la primera hoja—, y Verify la informa como FindHalfDead (Repair la termina). Réplica: los
// tres registros se rehacen como los demás; el reciclado choca con los snapshots abiertos de la réplica
// (snapshotConflict: espera o cancela, como el safexid de XLOG_BTREE_REUSE_PAGE), y una lectura suelta de la réplica
// tiene replayMu durante toda la operación. El flujo lógico los ignora (no son cambios de datos).
//
// Raíz rápida (btm_fastroot): la página de más a la izquierda de su nivel por la que empiezan las bajadas a niveles
// iguales o menores; la raíz nueva la pone en sí misma, al partirse la página de más a la izquierda de su nivel sube
// al padre (en el KindInsertPivot) y en el desenganche baja a la hermana derecha si queda sola (o vuelve a la raíz si
// la borrada era la raíz rápida). Una bajada que la encuentra borrada, de otro nivel o con hermana izquierda baja
// desde la raíz. Un árbol al que se le borra todo queda con una página por nivel (la de más a la derecha) y la raíz
// rápida en la hoja, como nbtree (la raíz nunca se borra ni baja de nivel).
//
// Medido (TestDiskSteadyState): ventana deslizante de 20.000 ids (13 B) con valores de 256 B, uno de cada cuatro
// reemplazado, limpieza de fondo cada 5 ms: el disco del store (sus dos forks) queda en 1,38 × los datos vivos (ids +
// valores) desde la segunda vuelta de la ventana (meta ≤ 1,5): hojas al 87 % (la partición al final deja la izquierda
// al 90 %) con 32 B fijos por versión (28 de encabezado y 4 del puntero), más internas, libres y borradas en espera.
// Con valores de 100 B da 1,65 × (pesan los 32 B por versión) y con 1 KB, 1,40 ×. Para lograrlo hizo falta además que
// la limpieza quite la lápida de referencia de un id que nombra la clave alta de su hoja cuando a la derecha ya no
// quedan versiones de ese id (antes quedaba para siempre y la hoja nunca se vaciaba: 13 × tras diez vueltas).
// Desde la tarea 108 (2026-11-21; 1,43 antes) una escritura que no cabe poda primero las versiones muertas de la hoja
// y una hoja con inserción ascendente local se parte en el punto de inserción («Llenado de las hojas» en
// btree/doc.go): los índices de josefina pasan del 57–66 % al 79–82 % de llenado y el store de documentos ya no baja
// del 85 % al 48 % tras actualizar (TestIndexHeaderShare).
//
// A confirmar (tarea 70): la pila de borradas es LIFO y se recicla mientras la de arriba sea segura (con un snapshot
// largo abierto, las borradas antes de él esperan a las más nuevas); el reciclado solo al empezar cada pasada (la
// página vuelve una pasada después); la espera de hasta 2 s a las operaciones en vuelo; no se borran páginas en la
// misma pasada en que la hermana derecha quedó vacía (nbtree lo hace con rightsib_empty), ni se encoge el archivo.
//
// # Congelado y truncado del archivo de estados
//
// Tarea 62 de F-23, decisión 44 («congelado con mapa de visibilidad», como relfrozenxid/datfrozenxid de PostgreSQL):
// el archivo de estados crece con cada transacción (2 bits por xid, en disco); el congelado lo acota. No hay vuelta de
// xid (son de 64 bits): el congelado solo sirve para borrar archivos de estados viejos.
//
//   - Congelar: la limpieza marca EntryFrozen en toda versión confirmada con xid < FreezeLimit = horizonte −
//     FREEZE_MIN_AGE, en el mismo KindLeafVacuum que quita las muertas de la hoja (con imagen tras un checkpoint, como
//     todo cambio: «WAL antes que la página»). El registro va en el WAL detrás de la confirmación de esas versiones
//     (que terminó antes de tomar el horizonte), así que ningún prefijo del WAL tiene la versión congelada sin su
//     confirmación: la regla 49 no hace falta (es para las marcas sin WAL). Una abortada nunca se congela: está muerta
//     y se quita en esa misma pasada (en línea) o se marca «liberándose» (desborde), y una «liberándose» se lee
//     abortada por su marca sin consultar el archivo de estados.
//   - Mapa de visibilidad: una hoja que queda con todo visible para todos (confirmado con xid < horizonte o congelado,
//     sin «liberándose» ni muertas, sin partición incompleta y sin compartir un id con una vecina) lleva
//     page.FlagAllVisible (en el mismo KindLeafVacuum) y su bit VMAllVisible, o VMAllFrozen si además todo quedó
//     congelado (KindVMSet, después, con la hoja todavía bloqueada). Toda escritura o partición de una hoja con
//     page.FlagAllVisible borra antes su bit (KindVMClear, con la hoja bloqueada, antes del registro de la hoja, que
//     apaga la marca de la página): en todo prefijo del WAL, bit puesto ⇒ marca en la página ⇒ sin cambios desde que la
//     limpieza la vio. Como PD_ALL_VISIBLE, la marca de la página ahorra mirar el mapa en cada escritura.
//   - Saltar: con dos niveles o más, la limpieza recorre las hojas por los enlaces del nivel 1 (copiados con la página
//     bloqueada en compartido) y salta sin leerla cada hoja enlazada cuyo bit es VMAllFrozen (o VMAllVisible si no es
//     agresiva); desde cada hoja que lee sigue a la derecha hasta el enlace siguiente, así ve las hojas nuevas de
//     particiones posteriores a la copia. Saltar es seguro porque una hoja con bit no se partió desde que se puso (la
//     partición lo borra) y lo que una partición posterior se lleve está congelado o es de un xid ≥ horizonte.
//   - frozenXID del store (página meta, KindSetFrozen): una pasada que terminó sin saltar hojas solo todo-visibles
//     (btree VacuumStats.Complete, como el bit all-frozen de PostgreSQL) lo sube al menor xid que dejó sin congelar,
//     acotado por el horizonte. Un store nuevo (Create, Empty) nace con el xid vivo más viejo: ninguna transacción que
//     pueda escribir en él tiene uno menor.
//   - frozenXID del motor (control): el menor de los stores del catálogo, tomado al empezar el paso 1 del checkpoint
//     (lo que lo justifica —los congelados de las hojas y la meta— tiene lsn anterior al punto de rehacer y queda en
//     disco con el checkpoint); un store que el motor todavía no leyó cuenta con el frozenXID del control al abrir
//     (cota baja de todos). DropStore y Empty olvidan el archivo viejo: dejan de retenerlo.
//   - Truncar: el paso 7 del checkpoint, después de guardar el control, llama a xact Truncate(frozenXID): borra los
//     archivos enteros por debajo. Status de un xid truncado devuelve confirmada sin leer el disco: por debajo del
//     frozenXID toda versión está congelada (que no consulta), «liberándose» (que no consulta) o ya no existe. Un corte
//     a mitad deja un prefijo de archivos borrado; Open vuelve a llamar a Truncate con el frozenXID del control.
//
// Decisiones (a confirmar): FREEZE_MIN_AGE = 0, se congela toda versión confirmada visible para todos la primera vez
// que la limpieza pasa por su hoja (el registro casi siempre ya hace falta para quitar las muertas, y no hay vuelta de
// xid que justifique esperar); así una hoja todo-visible queda también todo-congelada y cada pasada normal ya es
// completa. FREEZE_TABLE_AGE = un archivo de estados (134.217.728 xids): si el frozenXID de un store quedó así de
// detrás del horizonte, su pasada es agresiva (no salta las hojas solo todo-visibles, que con FREEZE_MIN_AGE 0 solo
// quedan de una configuración anterior). Con eso el disco del archivo de estados queda acotado por lo que separa al
// horizonte del xid vivo más viejo más dos archivos (el de la frontera y el actual), en vez de crecer para siempre. Un
// árbol de una sola hoja (sin nivel 1) o sin mapa se recorre entero. La hoja de un id cuyas versiones siguen en la
// vecina no lleva bits hasta que la limpieza quita esas versiones (se mira la hermana derecha). Un store que nunca
// tuvo una pasada completa (formato anterior, frozenXID 0 en su meta) cuenta con el frozenXID del control.
//
// # Fallos de escritura
//
// Si falla una escritura o un fsync del WAL, de una página, del archivo de estados o del control, el motor queda
// inutilizable (como el PANIC de PostgreSQL): toda operación devuelve el error, no se escriben más páginas sucias y
// el fsync no se reintenta (tras un fsync fallido el sistema operativo puede haber descartado las páginas: reintentar
// daría un éxito falso). Se recupera cerrando y abriendo: la recuperación reconstruye desde el WAL durable.
//
// Excepción (tarea 47): un disco lleno donde no se escribió nada irreversible devuelve ErrDiskFull y el motor
// sigue; uno donde ya no se puede volver atrás lo deja inutilizable igual, pero de solo lectura (ver «Disco lleno o
// lento»). Una operación que encuentra la caché o el WAL fallados también deja el motor inutilizable (de solo
// lectura si la causa fue de disco lleno); Close no escribe nada y devuelve el fallo. Un error de lectura por una
// página dañada se devuelve sin inutilizar el motor.
//
// # Disco lleno o lento
//
// Tarea 47 de F-23 (fase 3). Disco lleno. vfs.IsNoSpace reconoce ENOSPC, EDQUOT y el disco lleno simulado
// (vfs.ErrNoSpace, el de crashfs). Como PostgreSQL, el motor distingue dónde falta el lugar:
//
//	dónde                                  qué pasa
//	entrada de una escritura (admit)       ErrDiskFull, el motor sigue: no se escribió nada (ni se asignó xid)
//	extender un archivo de datos           ErrDiskFull, el motor sigue: la página nueva no existe ni se referencia
//	página del archivo de estados          queda sucia en su caché; el checkpoint aborta con ErrDiskFull y se reintenta
//	catalog.tmp (checkpoint o DDL)         el catálogo anterior sigue entero: el checkpoint aborta; el DDL sigue
//	WAL en medio de una operación          solo lectura (wal.ErrNoRoom fuera de admit)
//	registro del WAL ya reservado          solo lectura (no pasa con los segmentos preasignados, salvo copia en escritura)
//	página sucia (checkpoint, reemplazo)   solo lectura (sobrescribe un bloque ya extendido y escrito: idem)
//	control                                solo lectura (dos copias fijas que se sobrescriben: idem)
//	cualquier fsync                        solo lectura (volumen fino, NFS, asignación diferida)
//
// Entrada de lugar: toda escritura (suelta, de un lote, crear un store, la limpieza) llama antes de escribir nada a
// admit → wal.Room, que exige al menos medio segmento preasignado entre el resto del segmento activo y el siguiente.
// Con wal.Options.Preallocate (el motor lo usa siempre, como wal_init_zero) cada segmento se escribe entero con
// ceros y fsync antes de recibir registros: Room rellena el resto del activo (el que dejó Open al cortar la cola) y,
// pasada la mitad, prepara el siguiente (un repuesto reciclado, o uno nuevo). Lo hace bajo writeMu, frenando a los
// demás lo que tarde en escribir un segmento (como el XLogFileInit de PostgreSQL). Si no hay lugar, la operación
// devuelve ErrDiskFull y el motor sigue; la próxima entrada vuelve a intentar desde donde quedó, así que al liberar
// espacio las escrituras vuelven solas, sin reabrir. Lo que ya está preasignado alcanza para las operaciones en curso.
//
// Archivos de datos: buffer.PinNew escribe la página nueva en ceros (smgr.Extend) antes de usarla, como smgrextend,
// así el disco lleno sale ahí y no al escribir la página sucia después. Lo mismo al extender de a varias páginas
// (buffer.ExtendBy, fsm.Grow): smgr.Extend escribe en ceros cada bloque intermedio, nunca deja un hueco (Truncate o
// escribir más allá del final), porque ext4 y XFS no reservan el lugar de un hueco y el ENOSPC saldría después, al
// desalojar la página, con el motor ya sin vuelta atrás (tarea 107). Quien la pidió vuelve sin dejar nada a
// medias: una partición no empieza (o queda incompleta, que es un estado válido que termina el próximo que pase),
// una cadena de desborde libera los trozos ya escritos; la escritura devuelve ErrDiskFull, libera el desborde que
// había preparado y un lote aborta (txError). Crear un store cuya página meta no entra devuelve ErrDiskFull: el store
// queda en el catálogo sin páginas válidas y el próximo Store lo crea de nuevo (como tras un corte antes de su
// KindInitStore). La limpieza cortada por el disco lleno devuelve ErrDiskFull: cada paso suyo es un cambio completo.
//
// Archivo de estados y catálogo: una página de estados que no se puede escribir queda sucia en su caché (xact no
// queda fallado; el reemplazo prefiere ranuras limpias, y un archivo de estados que no se puede crear se lee en
// ceros); catalog.tmp sin lugar deja el catálogo anterior entero y CatalogFile no queda fallado. En el checkpoint
// (pasos 2 y 3) eso aborta el checkpoint con ErrDiskFull: el control sigue en el anterior, no se recicla nada y el
// checkpointer lo reintenta en su próxima vuelta (el punto de rehacer que movió el paso 1 solo hace pedir más
// imágenes de página). Tras un cambio de stores ya durable en el WAL (crear, borrar, vaciar) el catálogo se guarda en
// el próximo checkpoint (Engine.catalogBehind; Verify no compara el guardado mientras tanto): si hay un corte antes,
// la recuperación rehace el cambio desde el WAL.
//
// «Limpio» exige además que ningún fsync haya fallado (syncGuard envuelve el disco del motor y anota cada Sync y
// SyncDir fallidos), que el WAL y la caché sigan sanos y que no sea la falta de lugar del WAL en medio de una
// operación (Engine.cleanNoSpace). Una reserva del WAL que tiene que preasignar en medio de una operación y no puede
// devuelve wal.ErrNoRoom sin dejar el WAL inutilizable, pero el motor sí queda de solo lectura: la operación pudo
// haber pedido una página (de la lista de libres o extendiendo) que quedaría sin dueño.
//
// Solo lectura: donde ya no se puede volver atrás rige «Fallos de escritura» (no se reintenta ningún fsync), pero si
// la causa es de disco lleno el motor queda de solo lectura (Engine.readOnly, Stats.ReadOnly): las lecturas siguen
// (Read, Exists, GetVersion, los recorridos, Snapshot y Store de uno que existe, por enterRead); las escrituras,
// NewBatch, Checkpoint, Vacuum, WaitDurable y crear un store devuelven ErrDiskFull junto con ErrFailed. La caché de
// solo lectura (buffer.Pool.Failed con causa de disco lleno) sigue dando páginas, pero no escribe ninguna: para hacer
// lugar reusa solo búferes limpios y las páginas sucias se quedan en ella con su cambio (si todas están sucias, la
// lectura falla). Close no escribe nada (sin checkpoint final) y no se cuelga; Open después, con lugar, recupera todo
// lo confirmado desde el WAL durable. Recuperar sin reabrir no se intenta (decisión conservadora: tras un fsync
// fallido nada de lo escrito sin fsync es seguro).
//
// Qué se pierde en solo lectura: nada confirmado y durable. La operación que encontró el fallo no se confirmó (no se
// anotó su estado, no se ve), salvo que su registro haya llegado al disco: entonces aparece al reabrir, como un corte
// entre el registro y la respuesta. Las confirmaciones diferidas sin fsync se ven mientras dura el motor de solo
// lectura y al reabrir pueden faltar (siempre un prefijo en el orden del WAL, como tras un corte). Count puede incluir
// la operación que falló (se suma bajo el writeMu del WAL al reservar). Open con el disco todavía lleno recupera o
// falla con ErrDiskFull (el checkpoint que cierra la recuperación necesita lugar), nunca corrompe. Close con el disco
// lleno y el motor sano devuelve ErrDiskFull si su checkpoint final no tiene lugar: lo confirmado ya está en el WAL y
// el próximo Open lo recupera.
//
// Disco lento. Lo que puede crecer si el fsync o la escritura tardan, y cómo se acota:
//
//   - Group commit: cada escritor espera su fsync en su goroutine; un fsync confirma todo lo publicado. La única cola
//     del WAL (finished, las escrituras terminadas por encima de published) la acotan las escrituras en curso.
//   - Páginas sucias: la caché es fija; si las reemplazables están sucias, quien necesita un búfer escribe una (y
//     espera al WAL) antes de seguir.
//   - WAL sin reciclar: antes de cada escritura, sin nada tomado (ni closeMu), throttle frena al escritor mientras el
//     WAL desde el punto de rehacer del último checkpoint terminado (log.Written() − doneWAL), o el que espera al
//     archivo continuo sin tope (archiveBacklog), pase de 2 × MAX_WAL_SIZE (walBacklogFactor, constante): espera a
//     que termine un checkpoint o avance el archivador (SetArchived), hasta LOCK_TIMEOUT, y al vencer devuelve
//     ErrThrottled sin escribir nada (un lote aborta). El checkpointer dispara por ese mismo volumen (uno que no
//     terminó no cuenta) y su reparto ya se apura cuando el WAL va adelantado. Close despierta a los frenados.
//   - Escrituras diferidas (Deferred, CommitDeferred, SYNC_ON_WRITE=false): no esperan el fsync, salvo que haya más
//     de maxDeferredLag = 8.192 registros sin fsync: entonces la confirmación espera el fsync hasta su lsn − 8.192
//     (deferredBrake). Lo que un corte puede perder queda acotado aunque el escritor de fondo del WAL no dé abasto.
//
// Nada de esto crece en el heap de Go con lo escrito (TestSlowDisk: 16 escritores sobre un disco con fsync lento).
// Stats.DiskFull cuenta las operaciones rechazadas con ErrDiskFull limpio y Stats.Throttled las escrituras frenadas.
//
// Pruebas: crashfs.SetCapacity da ErrNoSpace en escrituras y Truncate que crecen, al crear un archivo con el disco
// lleno y, con la capacidad por debajo de lo usado, en el Sync con escrituras pendientes; subirla vuelve a dejar
// escribir. crashfs.SetDelay demora escrituras y Sync. diskfull_test.go (TestDiskFull: el disco se llena a mitad de
// una carga al azar con cuatro cupos, las lecturas siguen, Verify limpio, Count coincide, al liberar espacio se sigue
// sin reabrir; TestDiskFullFsync y TestDiskFullDeferred: solo lectura y WaitDurable; TestDiskFullCrash: un corte tras
// un ErrDiskFull no pierde nada confirmado), slowdisk_test.go (TestSlowDisk, TestDeferredBrake,
// TestWALBacklogThrottle), wal/space_test.go, TestReadOnlyAfterDiskFull (buffer), TestDiskFull (xact) y
// TestCatalogSaveDiskFull (control).
//
// En Linux, con un loop ext4 lleno de verdad (test/linuxvm/diskfull.sh, tarea 50), el disco lleno siempre salió
// limpio (en la entrada o al extender) y el modo de solo lectura no se alcanzó: haría falta un fsync que falle por
// lugar (volumen fino, NFS).
//
// A confirmar: en APFS, con copia en escritura, sobrescribir un bloque preasignado podría pedir lugar nuevo y dejar
// el motor de solo lectura; preasignar en segundo plano en vez de bajo writeMu.
//
// # Presión y uso de disco
//
// Engine.Pressure (tarea 65 de F-23, fase 3, como pebble.Pressure de josefina) dice qué tan cerca está el motor de
// frenar a los escritores: un float64, 0 = sin presión, ≥ 1 = ya frena. Es el máximo de cinco razones, o 1 si el motor
// quedó de solo lectura o inutilizable; la mayor es la causa (Stats.Pressure y Stats.PressureCause; Pressure da solo el
// valor):
//
//	causa      razón                                                  qué frena al llegar a 1
//	wal        WAL sin reciclar (walBacklog) / 2 × MAX_WAL_SIZE        throttle: espera un checkpoint, ErrThrottled
//	archive    WAL sin archivar (archiveBacklog, sin tope) / el mismo  throttle: espera al archivador, ErrThrottled
//	slot       WAL que retiene la ranura más atrasada / MAX_SLOT_WAL_  con tope, la ranura se invalida (paso 6);
//	           KEEP_SIZE (sin tope: / 2 × MAX_WAL_SIZE)               sin tope, throttle: ErrThrottled
//	deferred   registros sin fsync (Published − Durable) / 8.192       deferredBrake: la diferida espera un fsync
//	dirty      páginas sucias / búferes de la caché                    nadie en el motor: un escritor que necesita
//	                                                                  un búfer escribe antes una página sucia
//	read_only  el motor quedó de solo lectura por un disco lleno (= 1) toda escritura: ErrDiskFull con ErrFailed
//	failed     el motor quedó inutilizable por otro fallo (= 1)         toda escritura: ErrFailed
//
// Solo lee atómicos (y failMu, una hoja): no toma candados largos ni recorre el árbol, así se puede preguntar antes
// de cada escritura masiva, como hace la admisión de josefina. A diferencia de josefina no guarda el valor 50 ms:
// calcularlo cuesta lo mismo que leerlo.
//
// Engine.DirUsage (como pebble.DirUsage) mide lo que ocupa el motor, en bytes (DiskUsage): Control (los archivos de
// la raíz: control, catalog, LOCK y un catalog.tmp a medias), Data (data/), WAL (wal/: los segmentos con lo
// preasignado y los repuestos, que ocupan el disco aunque no tengan registros), Xact (xact/), Total (la suma) y
// Stores (por nombre). Los cuatro primeros salen del largo de cada archivo (ReadDir + OpenFile(O_RDONLY) + Size por
// el vfs, así funciona con crashfs; un archivo que se borra en el medio no cuenta); Stores, de smgr.NBlocks de los
// dos forks de cada store × 8 KB, con ddlMu tomado para que ningún store se cree, borre o vacíe a la vez. No lee
// páginas: el costo es un open y un stat por archivo (los segmentos de 1 GB y del WAL son pocos). No es una foto
// atómica: Data puede diferir de la suma de Stores si en el medio una escritura extendió un archivo, o si hay
// huérfanos (los que Repair borraría). Store.DirUsage es la parte de un store (lo mismo que DiskUsage.Stores[name]),
// con el store tomado en lectura: ErrStoreDropped si se borró. DropStore y Empty borran los archivos enseguida, así
// el store borrado o el contenido viejo dejan de contar al volver.
//
// Pruebas (pressure_test.go): TestPressureWAL, TestPressureDeferred, TestPressureDirty, TestPressureReadOnly
// (crashfs.SetCapacity), TestDirUsage (Total = la suma de las partes = crashfs.Used, Data = la suma de Stores, tras
// cargar, DropStore y Empty), TestDirUsageOS (Total = lo que mide el directorio real) y TestPressureUsageConcurrent
// (con -race).
//
// # Estadísticas
//
// Engine.Stats (tarea 43 de F-23, fase 3) es una foto barata del motor: lee contadores atómicos uno a uno, sin recorrer
// el árbol ni detener a nadie (dos campos pueden no ser del mismo instante). Los contadores son desde Open, como los de
// pg_stat_*: para tasas se restan dos fotos. Campos:
//
//	grupo          campos                                         de dónde sale
//	caché          CacheSize, CachePages, CacheDirty, CacheHits,  buffer.Pool.Stats (Used = búferes con página;
//	               CacheMisses, CacheWritten                      sube al dar un búfer vacío, baja en DropFile)
//	WAL            WALSinceCheckpoint, LSN, DurableLSN            walBacklog (desde el último checkpoint terminado,
//	                                                              tarea 82), Published, Durable
//	checkpoints    Checkpoints, LastCheckpoint,                   contados al terminar el paso 7 (también el que
//	               LastCheckpointTook                             cierra la recuperación; uno de fondo se reparte)
//	limpieza       WritesSinceVacuum, Vacuums, PagesDeleted,      Engine.writes menos su valor al empezar la última
//	               PagesRecycled                                  limpieza (lo que mira VACUUM_THRESHOLD); páginas
//	                                                              del árbol borradas y recicladas (btree.VacuumStats)
//	transacciones  OpenBatches, OpenSnapshots, NextXID,           openTxn/closeTxn por tipo; xact OldestActive;
//	               OldestXID, FrozenXID, FrozenAge                engineFrozen; FrozenAge = OldestXID − FrozenXID
//	operaciones    Reads, Writes, Commits                         lecturas puntuales (Read, Exists, GetVersion y sus
//	                                                              …In/…At; no los recorridos); escrituras aplicadas;
//	                                                              confirmaciones (suelta aplicada o Commit con
//	                                                              escrituras)
//	disco          ReadOnly, DiskFull, Throttled                  readOnly; rechazos con ErrDiskFull limpio; frenadas
//	                                                              por el WAL sin reciclar (ver «Disco lleno o lento»)
//	archivo        Archive, ArchivedSegment, ArchivePending,      archiveState (atómicos) y wal.ActiveSegment: lo
//	               ArchiveRetained, ArchiveGap, ArchiveLost       archivado, los sellados pendientes, sus bytes, la
//	                                                              última pérdida de continuidad y cuántas hubo
//	presión        Pressure, PressureCause                        pressure (ver «Presión y uso de disco»)
//	latencias      CommitLatency, ReadLatency, FsyncLatency       metric.Histogram: confirmar (registro del lote,
//	                                                              espera del fsync y archivo de estados), una lectura
//	                                                              puntual, cada fsync del group commit del WAL
//
// Las latencias son histogramas de 64 cubetas log2 de nanosegundos con contadores atómicos (store/internal/metric,
// 536 B cada uno): nada crece con las claves ni con el tiempo (regla 1). Latency da Count, Mean, P50, P99 y Max; P50
// y P99 son el límite superior de su cubeta (error de hasta ×2, para ver órdenes de magnitud).
//
// Costo: en el camino caliente solo hay sumas atómicas y un time.Now por lectura, por confirmación y por fsync;
// ningún candado nuevo. Stats toma tres candados cortos que ya toman el checkpointer y la limpieza (el writeMu del
// WAL en SizeSince, el de las transacciones en curso en OldestActive y el del catálogo en engineFrozen), uno a la
// vez. Sirve también con el motor cerrado (devuelve los últimos valores).
//
// OnStats(interval, fn) arranca una goroutine (en Engine.bg) que cada interval (0 o menos = 1 s) llama a fn con
// Stats sin ningún candado tomado; antes de cada llamada mira si la detuvieron, así no empieza otra tras stop o
// Close. Devuelve stop, que se puede llamar varias veces y desde fn (no espera a la llamada en curso). Close cierra
// Engine.stop y espera a bg, así que espera a la llamada en curso y después no hay más: fn no debe llamar a Close
// (se esperaría a sí misma). Con el motor cerrado, OnStats no arranca nada. El Add a bg va con closeMu en lectura y
// Close marca closed con closeMu en escritura antes de bg.Wait: ningún Add llega tarde.
//
// A confirmar (fase 3): Metrics (el et.Json del diseño anterior); la edad de los snapshots en tiempo (hoy solo su
// cantidad y el horizonte en xids).
//
// # Verificación y reparación
//
// Tarea 45 de F-23 (fase 3, pregunta 29). Verify revisa con el motor abierto y no cambia nada (ni marcas de
// visibilidad: su recorrido no las pone). Repair hace el mismo recorrido y arregla cada cosa reparable al encontrarla.
// Los dos devuelven un Report: los totales (stores, páginas del árbol, borradas en espera, de cadenas, libres y sin
// dueño, versiones, ids vivos, archivos huérfanos) y los hallazgos (Kind, Store, File, Block y Blocks, Desc, Repairable, Repaired), con
// String() para imprimir; guarda con detalle los primeros MaxFindings = 1.000 y cuenta todos (Total), así un motor muy
// dañado no da un informe que crezca con las páginas. El error de Verify y Repair es solo ErrClosed o ErrFailed (la
// caché o el WAL fallaron); el daño va en el informe. cmd/storecheck es el comando: go run ./cmd/storecheck -dir <dir>
// [-repair] (sale con 0 sin pendientes, 1 con hallazgos sin arreglar, 2 si no pudo revisar).
//
// Qué se revisa, en este orden:
//
//  1. Con ckptMu y ddlMu (ningún checkpoint ni DDL mientras tanto, unos milisegundos): el control leído del disco
//     (válido, su punto de rehacer es el del último checkpoint terminado, Engine.ctlRedo, y el vigente del WAL no es
//     anterior a él: el paso 1 de un checkpoint mueve el vigente y, si un disco lleno limpio corta los pasos 2 o 3, el
//     vigente queda adelante del control, como RedoRecPtr de PostgreSQL, sin daño: las imágenes de página se deciden
//     contra un punto posterior al de la recuperación, así que sobran, nunca faltan; checkpointLSN ≥ punto de rehacer,
//     frozenXID ≤ nextXID); el
//     catálogo leído sin tocar catalog.tmp (control.CatalogFile.Read: los mismos stores con los mismos números que el
//     de memoria, nextFileNo no mayor); y data/: los números de archivo que ningún store tiene son huérfanos (de un
//     DropStore o Empty cortados entre el catálogo y el borrado) y los nombres que no son de archivos de datos se
//     informan sin tocarlos.
//  2. Cada store del catálogo, de a uno, con Store.vacuumMu → closeMu (lectura) → Store.mu en escritura: espera a sus
//     operaciones en curso y frena las nuevas de ese store mientras dura (como DropStore); los otros stores, los lotes
//     que no lo tocan, los checkpoints y el escritor de fondo siguen. Un store sin abrir se abre solo si su página meta
//     existe: uno sin meta (Store lo crearía vacío) o con la meta ilegible se informa sin abrirlo.
//  3. El árbol con btree.VerifyWith (la estructura de btree.Verify, con buffer.Ring para no vaciar la caché): cada
//     página del árbol se anota en un bitset de dueños; por cada hoja ya soltada, el conteo (por id, la versión más
//     nueva confirmada decide, como Count; el estado sale de las marcas o del archivo de estados, sin marcar), las
//     versiones sin congelar con xid menor que el frozenXID del store (y ese frozenXID no mayor que nextXID), las
//     cadenas de desborde (overflow.Manager.Pages: cada página de desborde, del xid de la versión, en su posición y
//     con su trozo; de una versión «liberándose», solo lo que todavía es suyo), los valores comprimidos en línea o en
//     desborde (formato 2: se descomprimen; FindValue, no reparable, si no) y su bit del mapa de visibilidad (un
//     bit puesto exige page.FlagAllVisible y todas las versiones congeladas o confirmadas con xid menor que el
//     horizonte; VMAllFrozen, todas congeladas). Una hoja medio muerta (un borrado de páginas cortado a medias) es un
//     hallazgo reparable, FindHalfDead: btree.Verify reconoce el borrado a medias (la hoja y su cadena de padres de un
//     solo hijo) y comprueba sus enlaces, y esas páginas son del árbol, no «sin dueño»; también recorre la pila de
//     páginas borradas de la meta (cada una borrada, sin ciclos; son del árbol) y comprueba la raíz rápida. Si el árbol
//     no pasa (una hoja ilegible, un enlace roto), lo demás del store no es confiable y no se mira ni se repara.
//  4. La lista de libres (fsm.Manager.WalkFree): cada página es libre, existe y no tiene otro dueño (un bucle o una
//     página del árbol en la lista se ven ahí).
//  5. Los bits del mapa en bloques que no son hojas o no existen (fsm.Manager.ForEachVM).
//  6. Las páginas sin dueño: las que no son meta, árbol, cadena viva ni lista de libres, agrupadas en tramos seguidos
//     de la misma clase (en ceros —un PinNew sin registro—, vacías —sacadas de la lista sin usar—, libres fuera de la
//     lista, ilegibles, páginas del árbol inalcanzables y de desborde, que son cadenas huérfanas: escritas antes de
//     un corte sin que llegara su hoja, o de un Free cortado).
//  7. Count: el conteo contra el contador. Las confirmaciones de lotes no toman el store, así que si el conteo no
//     coincide con el contador de antes ni con el de después del recorrido se repite solo el conteo con commitMu en
//     escritura (ninguna confirmación en curso en todo el motor, el tiempo de recorrer las hojas de ese store).
//
// Qué repara Repair (cada arreglo con su registro del WAL, así un corte en medio deja un motor que abre y cuyo Verify
// solo informa lo que faltó, TestRepairCrash):
//
//   - Archivos huérfanos: los borra (smgr.Unlink y los que queden por nombre, fsync de data/); sin WAL, como el
//     borrado de DropStore, porque los números no se reutilizan y borrarlos es idempotente.
//   - Páginas sin dueño y cadenas huérfanas: las devuelve a la lista de libres con fsm.Manager.Reclaim (un KindFreePush
//     con BlockWillInit sobre la página fijada sin leerla, así también una ilegible), solo si en ese store no hay nada
//     que haga dudar de quién es cada página (ninguna cadena viva rota, página repetida ni lista de libres dañada).
//   - Bits del mapa de visibilidad mal puestos: los borra (KindVMClear); borrar un bit siempre es seguro (lo pierde
//     solo la limpieza, que vuelve a leer la hoja).
//   - Count: lo corrige con una confirmación propia (KindCommit de un xid nuevo sin versiones, con lo que suma al
//     store), hecha con commitMu en escritura y esperando su fsync; la recuperación la rehace como cualquier otra.
//   - Borrados de páginas a medias: los termina (btree.FinishDeletes: los KindUnlinkPage que faltaban; las páginas
//     quedan en la pila de borradas con el próximo xid como marca y la limpieza las recicla después).
//
// Qué no repara (lo informa sin tocarlo, y el store sigue legible en lo que no depende de eso): una página del
// árbol ilegible o una estructura que no pasa btree.Verify, una cadena viva rota (ese valor se perdió: Read del id da
// error), una página con dos dueños, la lista de libres apuntando a una página que no es libre o con un bucle (haría
// falta reescribir la cabeza en la meta con un registro nuevo), una versión sin congelar por debajo del frozenXID (el
// frozenXID no baja), un store sin meta, el control y el catálogo (Open ya falla si no se leen, y uno que no coincide
// con el motor no tiene arreglo seguro), un bit del mapa en una página del mapa ilegible, y el WAL (wal.Open ya lo
// valida al abrir: corta en el primer registro inválido y falla si está en un segmento sellado).
//
// Memoria: dos bitsets por el store que se revisa (dueños y hojas: un bit por bloque, 16 KB cada uno por archivo de
// datos de 1 GB), las entradas de una hoja, los bits de una página del mapa (hasta 32.640) y hasta MaxFindings
// hallazgos; nada crece con las claves. Medido en un M5 sobre disco real: 2 MB de datos en 1,9 ms, 19 MB en 19 ms y
// 97 MB (1.000.000 ids y 1.000 valores de 20 KB) en 119 ms, con unos 68 B asignados por versión (los ids que copia
// cada hoja, basura enseguida) y el heap en uso subiendo unos 3 MB.
//
// A confirmar (fase 3): que Repair deje sin tocar un store dañado salvo sus bits del mapa; que la lista de libres
// dañada no se reconstruya; los MaxFindings; que una hoja con bits y una versión de otra transacción en curso cuente
// como bit mal puesto (lo es: la limpieza nunca la deja así).
//
// # Respaldo en caliente y restauración
//
// Tarea 44 de F-23 (fase 3), como pg_basebackup: Engine.Backup(dir) copia el motor a dir (vacío o inexistente) sin
// detener las escrituras, los lotes ni el DDL, y Open(dir) restaura la copia. Se eligió un directorio y no un
// io.Writer: la copia queda lista para abrir, se prueba sobre vfs (backup recibe el vfs.FS del destino, crashfs en las
// pruebas) y el archivo continuo del WAL (tarea 66) parte de una copia así más el WAL siguiente, y la réplica (tarea
// 46) partirá de lo mismo.
//
// Pasos (backupRun.run):
//
//  1. prepare: el destino tiene que estar vacío (si no, ErrBackupDir); crea data/, xact/ y wal/ y escribe el marcador
//     «en curso» (sin fin): una copia a medias no se puede abrir.
//  2. start, con ckptMu: el modo respaldo (wal.ForceImages, con un contador: mientras dura, la imagen de página entera
//     se decide contra max(punto de rehacer, inicio del respaldo), como forcePageWrites de PostgreSQL), un checkpoint
//     sin repartir (su punto de rehacer R0 es el inicio), la retención del WAL desde el segmento de R0 (wal.Hold:
//     Recycle no toca esos segmentos) y la del archivo de estados desde el frozenXID de ese checkpoint (el paso 7 de
//     los checkpoints siguientes no trunca por debajo). Después, con ddlMu: el catálogo en memoria (es el guardado en
//     el paso 3, más el DDL ya durable de después: contadores a countLSN = R0) y la lista de archivos de data/ de sus
//     stores; desde ahí los borrados de archivos esperan (ver DDL).
//  3. copyData: el catálogo (CatalogFile.Save en el destino) y los archivos de datos de la lista, de a uno, leídos por
//     el vfs en tramos de 32 páginas con un búfer fijo (nada entero en memoria, ninguna página fijada en la caché),
//     cortados al último bloque entero; un archivo que ya no está se salta. Al terminar, los borrados que esperaban.
//  4. copyXact: los archivos de estados que haya.
//  5. writeEnd: KindBackupEnd (durable) en el WAL del motor; se apaga el modo respaldo.
//  6. copyWAL: los segmentos desde el de R0 hasta el de KindBackupEnd, este solo hasta el final de ese registro (lo
//     que sigue lo escribieron otros después del fin); después se suelta la retención.
//  7. finishDest: el control del checkpoint del inicio (control.Create en el destino: punto de rehacer R0, su
//     checkpointLSN, nextXID, xid vivo más viejo y frozenXID) y el marcador final (backup_label.tmp, fsync, rename).
//
// Todo lo que el respaldo toma (modo respaldo, retenciones, la cuenta de copia de datos) lo suelta finish con defer,
// también con error; Engine.bg cuenta el respaldo, así Close lo interrumpe (ErrClosed en el próximo tramo) y lo espera
// antes de cerrar las piezas. Un error del destino (disco lleno: ErrDiskFull) no toca al motor: el destino no pasa
// por el syncGuard del motor. Un corte del motor durante el respaldo no deja nada «pegado»: el modo respaldo y las
// retenciones viven en memoria.
//
// Por qué la copia es consistente: el checkpoint del inicio deja en disco toda página cambiada antes de R0 y el
// archivo de estados con toda confirmación anterior; desde R0 la primera modificación de cada página lleva su imagen
// entera (el punto de rehacer vigente nunca baja de R0, y el modo respaldo lo asegura en Reserve aunque no lo fuera;
// la primera marca de visibilidad tras R0 va como KindSetHints con su imagen, decidida contra el punto de rehacer,
// que desde el checkpoint del inicio ya es ≥ R0), así que una
// página leída a medio escribir (rota) o cambiada durante la copia se rehace entera desde el WAL. Todo lo que la copia
// leyó tiene lsn menor que KindBackupEnd («WAL antes que la página»: una página o un estado llegan al disco después de
// su registro, y la copia terminó antes de escribir el fin), así que rehacer de R0 hasta el fin la deja como estaba el
// motor en ese lsn: lo confirmado antes del fin (también lo diferido, porque el fin se espera durable) y nada de
// después. Los bloques extendidos después de R0 nacen con imagen o BlockWillInit, así que un archivo copiado más corto
// (o uno que no se copió: el de un store creado después de la lista) se completa al rehacer, como tras un corte. Los
// contadores de Count: el catálogo copiado tiene los de R0 con countLSN = R0, y la recuperación suma lo confirmado
// desde ahí.
//
// DDL durante el respaldo (crear, borrar y vaciar stores, Empty, DropStore, DDL en lote): nada se bloquea. La lista
// de archivos y el catálogo se toman juntos con ddlMu; un store creado después no está en ninguno de los dos y el WAL
// lo crea al restaurar. Un borrado (DropStore, Empty, el commit o el deshacer de un lote con DDL) escribe su registro y
// cambia el catálogo como siempre, pero mientras un respaldo copia archivos de datos (backups.copying, con ddlMu) sus
// archivos no se borran: Engine.unlinkData los anota en backups.deferred y se borran al terminar esa copia. Así la
// copia encuentra entero todo archivo de su catálogo, y al restaurar el registro de borrado (que está entre R0 y el
// fin) los borra. Verify no los informa como huérfanos mientras esperan. Un corte del motor antes de borrarlos los deja
// huérfanos (si un checkpoint posterior al borrado ya pasó su registro): Verify los informa y Repair los borra, como
// tras un DropStore cortado. Store que recrea un store sin página meta (tras un corte) sí borra enseguida: ese archivo
// la copia lo salta si ya no está, y su KindInitStore lo rehace.
//
// Truncado del archivo de estados: el control de la copia tiene el frozenXID del checkpoint del inicio; mientras haya
// un respaldo en curso, el paso 7 trunca solo por debajo del menor de esos frozenXID (backups.truncateLimit), así la
// copia tiene los archivos de estados que su control promete.
//
// Retención y reciclaje: mientras dura el respaldo los checkpoints siguen (el freno de los escritores mide el WAL desde
// el último checkpoint terminado, no lo retenido) pero no reciclan desde el segmento de R0: el disco del WAL crece con
// lo escrito durante el respaldo, como en PostgreSQL. Al terminar, el próximo checkpoint recicla lo retenido.
//
// Restaurar: Open sobre la copia (recover, paso 0) lee el marcador: dañado o sin fin, ErrBackupIncomplete. Antes de
// cambiar nada recorre el WAL desde el punto de rehacer del control (wal.NewReader, solo lectura) y exige que llegue al
// registro del lsn de fin del marcador y que sea el KindBackupEnd de este respaldo (su redoLSN); si el WAL termina
// antes (cortado, falta un segmento), ErrBackupIncomplete sin abrir a medias. Después recupera como tras un corte
// (KindBackupEnd no hace nada al rehacerse), hace el checkpoint de fin de recuperación y borra el marcador: desde ahí
// la copia es un motor más. Si un corte se lleva ese borrado, el próximo Open ve el marcador con el punto de rehacer ya
// pasado el fin y no comprueba nada. Para verificar una copia: abrirla y llamar a Verify.
//
// El formato de backup_label está en «Formatos en disco».
//
// Pruebas (backup_test.go): TestBackupConcurrent (escritores sueltos, diferidos y en lotes, y ráfagas y checkpoints
// entre archivo y archivo; la copia restaurada es igual a lo confirmado hasta el fin, Verify limpio, Count),
// TestBackupDDL (crear, borrar, vaciar y DDL en lote durante la copia; sin huérfanos en el original ni en la copia),
// TestBackupRecycle (WAL chico: los checkpoints del medio reciclarían el segmento del inicio), TestBackupIncomplete
// (WAL cortado antes del fin y marcador sin fin: ErrBackupIncomplete sin tocar la copia; destino no vacío:
// ErrBackupDir), TestBackupDestFull (crashfs.SetCapacity en el destino: ErrDiskFull, el original sano y sin nada
// retenido), TestBackupCrash (corte del original en medio, en los tres modos), TestBackupClose y TestBackupOS (disco
// real); en wal, TestForceImages y TestHold.
//
// A confirmar (tarea 44): copiar los segmentos sellados enteros (con lo preasignado: hasta WAL_SEGMENT_SIZE de más);
// los archivos que espera un respaldo cortado quedan para Repair; no hay Restore aparte ni VerifyBackup (Open restaura
// y Verify revisa); la copia no se comprime ni se envía a un io.Writer; mientras dura no hay tope al WAL retenido (las
// ranuras, con MAX_SLOT_WAL_KEEP_SIZE, son aparte: ver «Replicación»).
//
// # Archivo continuo del WAL y recuperación a un punto en el tiempo
//
// Tarea 66 de F-23 (fase 3; archive.go, la retención y los segmentos en wal/archive.go), como archive_command de
// PostgreSQL y SetArchived de josefina. Se activa con WAL_ARCHIVE=on al abrir (una variable, como josefina activa su
// archivo por configuración; no cambia con el motor abierto). Mientras está activo ningún segmento sellado se recicla
// ni se borra hasta que el archivador lo marque copiado. La unidad es el segmento entero (como archive_command), no el
// registro: SetArchived recibe el número de segmento, no un lsn como en josefina, porque lo que el archivador copia y
// lo que Recycle borra son archivos.
//
// Retención: reutiliza wal.Hold, la del respaldo. archiveState.next es el primer segmento sin archivar; el motor tiene
// una retención desde él (Recycle no toca los segmentos ≥ next). SetArchived(seg) guarda archive_status con seg
// (durable: tmp, fsync, rename, fsync del directorio) y recién después mueve la retención a seg + 1: un corte entre los
// dos solo hace volver a archivar (el archivador tiene que ser idempotente, como el de PostgreSQL). Los segmentos ya
// archivados se reciclan en el próximo checkpoint como siempre. En memoria solo hay dos números y una retención: la
// lista de pendientes sale del WAL (los números entre next y el activo; el final de cada uno de lo anotado al sellarlo
// o, para los de antes del punto de rehacer de Open, de recorrerlo una vez).
//
// Archivador (la API):
//
//	PendingArchive() ([]WALSegment, error)   los sellados sin archivar, en orden, hasta 256: número, nombre, ruta,
//	                                          StartLSN, EndLSN y Size (bytes válidos desde el inicio)
//	SetArchived(seg uint64) error            ya copié hasta seg inclusive (ErrNotSealed si no está sellado)
//	SwitchWAL() (uint64, error)              sella el activo aunque no esté lleno (pg_switch_wal), para archivar ya
//	OpenAt(dir string, lsn uint64)           restaura una copia de Backup hasta lsn (recovery_target_lsn)
//
// Qué se copia de un segmento: alcanza con sus Size bytes. El archivo mide WAL_SEGMENT_SIZE (la preasignación, tarea
// 47, lo llena de ceros, y un repuesto reciclado trae registros viejos), pero lo que sigue al último registro válido
// son ceros o registros de lsn menor, que Scan descarta: copiar el archivo entero también sirve, solo ocupa más. Un
// segmento sellado no cambia más (seal espera sus escrituras y hace fsync antes de pasar al siguiente).
//
// Sellar al abrir: tras un corte, wal.Open sigue escribiendo en el último segmento con registros válidos; si ese
// segmento ya estaba sellado (con SwitchWAL, o lleno con el siguiente todavía vacío) el archivador pudo copiarlo, y
// agregarle registros rompería esa copia. Por eso, con el archivo activo, Open sella el segmento en que quedó el WAL
// si tiene registros (wal.Switch): un segmento por apertura, que el archivador copia con su Size.
//
// Archivador atrasado (pregunta 48, decisión 36: ranuras con tope, como max_slot_wal_keep_size):
//
//   - Sin tope (WAL_ARCHIVE_MAX_SIZE = 0, por omisión): lo retenido (segmentos sellados sin archivar ×
//     WAL_SEGMENT_SIZE) suma al freno de los escritores: con más de 2 × MAX_WAL_SIZE, throttle espera a que el
//     archivador avance (SetArchived despierta a los frenados) hasta LOCK_TIMEOUT y después da ErrThrottled. Nunca se
//     pierde nada ni se rompe la continuidad del archivo; el costo es que un archivador caído detiene las escrituras
//     (con el WAL acotado a ~2 × MAX_WAL_SIZE más lo que retenga un respaldo). Stats.PressureCause = archive.
//   - Con tope (WAL_ARCHIVE_MAX_SIZE > 0): no frena. En el paso 6 de cada checkpoint, si lo retenido pasa del tope, el
//     motor deja de retener lo anterior al punto de rehacer: guarda archive_status con gap = ese segmento (antes de
//     soltar nada) y mueve la retención ahí; esos segmentos se reciclan sin archivar. Stats.ArchiveGap dice desde
//     qué segmento sigue el archivo y Stats.ArchiveLost cuántas veces pasó: un respaldo anterior a la pérdida ya no
//     llega a un punto posterior, hace falta uno nuevo. PendingArchive sigue desde el gap.
//
// Persistencia (archive_status, formato en «Formatos en disco»). Al abrir con WAL_ARCHIVE=on: sin archivo (primera vez)
// se archiva desde el segmento más viejo que haya; con archivo, desde archived + 1; si con el archivo apagado se
// reciclaron segmentos sin archivar (el más viejo que hay es mayor que archived + 1), se anota la pérdida (gap = el más
// viejo). Con WAL_ARCHIVE=off el motor ni lo lee ni lo cambia, y todo es como antes de la tarea 66. Uno dañado es
// ErrCorrupt. Backup no lo copia: la copia restaurada empieza su propio archivo (no hay líneas de tiempo como en
// PostgreSQL: el archivo de una copia restaurada va a otro destino).
//
// Recuperación a un punto en el tiempo (PITR): la base es una copia de Backup (tarea 44) y los segmentos archivados
// desde antes de su fin. Para restaurar: copiar a wal/ de la copia los segmentos archivados desde el primero que tiene
// la copia (el último de la copia lo cortó el respaldo en su KindBackupEnd: la versión archivada lo reemplaza, es la
// misma más lo que sigue), con su nombre, y abrir. Open rehace como siempre (paso 0: el WAL llega al fin del respaldo;
// paso 2: Scan sigue de un segmento al siguiente mientras el lsn sea contiguo), así llega hasta el final de lo
// archivado. OpenAt(dir, lsn) para antes: comprueba sin escribir nada que hay backup_label, que lsn ≥ el fin del
// respaldo y ≥ el checkpoint del control (antes la copia no es consistente: sus páginas pueden tener cambios hasta el
// fin) y que el registro de lsn está en el WAL (si no, ErrRecoveryTarget y la copia queda como estaba); después
// wal.Open recorre con Options.StopLSN = lsn: rehace hasta ese registro inclusive y corta lo que sigue como una cola
// inválida. Lo confirmado después de lsn no está; lo que estaba a medias en lsn queda deshecho, como tras un corte
// (también un DDL en lote sin su KindCommit). El objetivo es un lsn de registro (Stats.LSN, BackupInfo.EndLSN,
// WALSegment.EndLSN); por xid o por hora queda pendiente (KindCommit trae la hora: se podría parar en la primera
// confirmación posterior).
//
// Pruebas (archive_test.go): TestArchiveRetains (sin SetArchived nada se recicla tras cuatro checkpoints; después sí;
// Stats; ErrNotSealed; SwitchWAL), TestArchiveOff (como antes: recicla, sin archive_status, ErrArchiveOff),
// TestArchivePersist (Close/Open y un corte en los tres modos de crashfs tras SetArchived y SwitchWAL: lo archivado
// sigue, nada sin archivar se recicla, el segmento copiado no cambia al reabrir), TestArchiveThrottle (sin tope:
// ErrThrottled, Pressure por el archivo, nada perdido, sigue al archivar), TestArchiveCap (con tope: no frena,
// ArchiveGap persistente, recicla), TestArchivePITR (respaldo + escrituras + segmentos archivados: hasta el final, y
// con OpenAt hasta un lsn intermedio, Verify limpio; objetivos inalcanzables y sin backup_label: ErrRecoveryTarget sin
// cambiar nada); en wal, TestSwitchSegmentInfo y TestStopLSN.
//
// A confirmar (tarea 66): objetivo por xid o por hora; líneas de tiempo (el archivo de una copia restaurada empieza
// de nuevo); un comando que haga de archivador (copiar a un directorio u objeto) y un Restore que traiga los
// segmentos; comprimir los segmentos archivados; el tope con un aviso además de Stats (el motor no escribe registros
// de log).
//
// # Replicación
//
// Tarea 46 de F-23 (fase 3; walsince.go, replica.go, slots.go, decode.go; en wal, stream.go). Tres piezas, como en
// PostgreSQL: la réplica física (streaming), las ranuras de retención (replication slots con max_slot_wal_keep_size,
// pregunta 48) y el flujo lógico de cambios (decodificado lógico, preguntas 31 y 50). El transporte no es del motor:
// josefina lleva los WalEntry y los Change por su hub TCP; las pruebas conectan primario y réplica en el mismo proceso.
//
// Leer el WAL (WalSince). WalReader guarda solo la posición del próximo registro; Next lo ubica la primera vez
// (wal.Locate: el segmento cuyo primer registro es ≤ lsn, recorrido hasta él) y después lee con wal.Read del disco,
// no de la memoria, hasta Durable (una réplica nunca tiene algo que el primario puede perder en un corte) y hasta
// maxBytes por tanda. Cada WalEntry es el registro tal cual (encabezado, bloques y datos, con su crc) con su lsn,
// segmento y desplazamiento. Si el segmento ya no está (se recicló), ErrWalRemoved: la réplica necesita un Backup
// nuevo. WalSince no retiene nada; para eso, una ranura. Los recorridos reusan sus búferes (un sync.Pool en wal).
//
// Réplica física. Se arma así: CreateSlot en el primario, Backup del primario (la copia trae el WAL hasta su fin),
// OpenReplica(copia) y un bucle WalSince(ReplayLSN) → Next → ApplyWal → WaitDurable → AdvanceSlot(ReplayLSN) en el
// primario. Decisiones:
//
//   - WAL idéntico byte a byte: ApplyWal escribe cada registro en el WAL local en la misma posición (wal.Append: el
//     final del segmento activo, o el principio del siguiente si el primario selló antes; otra posición o un lsn que
//     no sigue es ErrWalGap sin escribir nada) y recién después lo rehace con el mismo redoState de la recuperación
//     (cada Kind a su dueño). Así los puntos de rehacer de los KindCheckpoint del primario sirven tal cual en la réplica.
//     Los ya aplicados (lsn ≤ ReplayLSN) se saltan: reenviar es seguro.
//   - La réplica no escribe registros propios. Sus checkpoints son restartpoints: toman el último KindCheckpoint
//     rehecho, escriben las páginas sucias y el archivo de estados (todo lo aplicado, que incluye lo anterior a su
//     punto de rehacer), guardan el catálogo con los contadores a lo aplicado (countLSN = aplicado + 1), el control con
//     ese punto de rehacer y su lsn (wal.SetRedo), reciclan el WAL anterior y truncan el archivo de estados por debajo
//     del frozenXID tomado al rehacer el KindCheckpoint anterior (lo que lo justifica queda antes del punto de rehacer
//     del siguiente; los KindSetFrozen y KindInitStore rehechos ponen el frozenXID de su store). Sin un checkpoint
//     nuevo no hacen nada. Las lecturas no ponen marcas de visibilidad (btree.Config.NoHints): una marca anotaría un
//     KindSetHints en el WAL, y sin WAL rompería la suma de una página a medio escribir tras un restartpoint.
//   - OpenReplica recupera como Open, salvo: un directorio sin motor es ErrNotReplica (se empieza de un Backup); no
//     borra los stores de lotes sin confirmar ni fija el horizonte del archivo de estados (esas transacciones pueden
//     confirmar después en el primario); no arranca el archivo continuo ni la limpieza de fondo; termina con un
//     restartpoint. Un corte de la réplica se recupera igual que uno del primario: reabrir sigue desde lo durable.
//     Promover: Open sobre el directorio de la réplica (deshace lo que seguía en curso y es un motor más; no hay líneas
//     de tiempo, así que el primario viejo no puede seguirla).
//   - Solo lectura: enter devuelve ErrReadOnly (escrituras, NewBatch, Vacuum, Repair, Backup, SetArchived); Store de
//     un store que no existe es ErrStoreNotFound, y uno creado por un lote sin confirmar o cuyo KindInitStore todavía
//     no llegó, ErrStorePending. Read, Exists, GetVersion, Count, los recorridos, Snapshot, Verify y Checkpoint (un
//     restartpoint) andan.
//   - Concurrencia con las lecturas: aplicar toma replayMu en escritura por tanda; cada lectura de páginas (Store.begin,
//     también cada hoja de un recorrido) lo toma en lectura. Así una lectura nunca ve un registro a medias (una
//     partición toca tres páginas) y un DDL rehecho cambia el catálogo y los *Store abiertos (syncHandles: uno borrado
//     da ErrStoreDropped, uno vaciado espera su KindInitStore) sin lecturas en curso. Un DDL espera antes el fsync del
//     WAL local: el catálogo y los archivos nunca se adelantan a él.
//   - Snapshots estables: un snapshot de la réplica ve lo confirmado al tomarlo: sin transacciones en curso
//     (InProgress vacío), con Xmax = el siguiente xid visto en el WAL aplicado, así un xid confirmado lo ve si es
//     menor que su Xmax. Si una tanda trae un registro que puede cambiar lo que ve un snapshot (snapshotConflict:
//     una confirmación —KindCommit o una escritura suelta—, una limpieza de hoja, una liberación o un reciclado de
//     páginas, un DDL) el aplicador cierra la entrada de snapshots nuevos (txGate; un snapshot se toma y se anota con
//     txGate en lectura, así ninguno queda a medio tomar) y, antes de cada registro así, mira a quiénes afecta
//     (conflictOf, tarea 125): una confirmación, a los que verían su xid (xid < Xmax: uno tomado antes de que llegara
//     el primer registro del xid no lo ve ni antes ni después); un KindLeafVacuum (la limpieza o la poda antes de
//     partir de la tarea 108), leyendo la hoja antes de rehacerlo (btree.LeafVacuumConflict, como el latestRemovedXid
//     que PostgreSQL calcula en el standby), a los que ven una versión que quita o marca «liberándose» sin ver una más
//     nueva del mismo id en la hoja, o no veían una que congela (una congelada la ven todos); una liberación o un
//     reciclado de páginas y un DDL, a todos (también si la hoja no se puede leer). Espera a que se cierren los
//     afectados, hasta replicaConflictDelay (30 s, desde la primera espera de la tanda), y cancela los que queden con
//     ErrRecoveryConflict (Stats.RecoveryConflicts); los demás siguen abiertos. Se eligió esperar y no llevar la
//     lista de transacciones en curso del primario (las KnownAssignedXids de PostgreSQL): los xid sin escrituras no
//     aparecen en el WAL y esa lista crecería sin tope. El costo: con un snapshot largo que ve lo que cambia el
//     aplicador se atrasa (lo acota el plazo); la limpieza de fondo congela (FREEZE_MIN_AGE = 0) versiones recientes
//     que un snapshot largo no ve, así que lo sigue afectando.
//   - Durabilidad: ApplyWal no hace fsync; WaitDurable en la réplica hace durable lo aplicado (también el escritor de
//     fondo del WAL). Lo que se confirma en la ranura del primario es ReplayLSN después de WaitDurable.
//
// Ranuras (slots.go). Cada ranura tiene RestartLSN (lo más viejo que retiene) y ConfirmedLSN (hasta dónde confirmó su
// consumidor), y retiene con wal.Hold el segmento de RestartLSN (wal.SegmentOf: solo lee el primer registro de cada
// segmento). CreateSlot la crea con ConfirmedLSN = lo escrito y la guarda enseguida (antes del Backup de la réplica,
// así la copia y lo que sigue quedan retenidos); DropSlot la borra; Slots las describe (SlotInfo: Retained = segmentos
// desde el retenido, Lag = registros sin confirmar). AdvanceSlot anota el avance en memoria y se guarda en el paso 6
// del siguiente checkpoint o en Close, como en PostgreSQL; la retención (wal.Hold) solo pasa a otro segmento después
// de guardar el archivo que la promete (moveHold: AdvanceSlot guarda antes de moverla, una vez por segmento, y
// despierta a los escritores frenados; el paso 6 guarda y después mueve las de todas). Así ninguna retención pasa lo
// guardado, aunque un AdvanceSlot llegue entre el guardado y el reciclaje, y con un disco lleno al guardar las
// retenciones quedan donde estaban: lo que se recicla siempre sale de lo guardado y nunca falta WAL que la ranura
// guardada promete. Tras un corte del primario la ranura vuelve al valor guardado. DropSlot guarda antes de soltar la
// retención (si no puede guardar, la ranura sigue). El consumidor
// recibe de nuevo lo posterior: la réplica lo salta por lsn y el lógico por TxInfo.CommitLSN. Nunca retrocede.
// Tope: en el paso 6, una ranura que retiene más de MAX_SLOT_WAL_KEEP_SIZE se invalida (suelta su retención, queda
// marcada en el archivo, Stats.SlotsInvalidated y SlotsInvalid); AdvanceSlot, WalSince y Changes de ella dan
// ErrWalRemoved y su consumidor necesita empezar de nuevo (Backup o una ranura lógica nueva); el motor sigue sano.
// Sin tope (0), la ranura más atrasada suma al freno de los escritores (ErrThrottled), como el archivo continuo sin
// tope. Pressure considera la ranura (PressureSlot: con tope, lo retenido sobre el tope; sin tope, sobre el del
// freno). Las ranuras andan también en una réplica (para encadenar réplicas), salvo las lógicas (ErrReadOnly).
//
// Flujo lógico (decode.go). El registro de inserción en la hoja (KindLeafWrite) ya es el cambio lógico: id, xid,
// lápida y el valor en línea o la referencia a desborde (decisión 38: no se repite el valor). Un valor en desborde se
// arma con los KindOverflowWrite de su cadena, que van antes en el WAL con el mismo xid (cada uno trae su bloque, el
// siguiente y el trozo); por eso no hizo falta ningún registro lógico adicional: el costo es leer los trozos del WAL
// en vez de un solo registro. Los nombres de los stores salen del catálogo al abrir el lector y de los registros de
// DDL que lee (un store borrado antes de abrirlo da Store "").
//
//   - CreateLogicalSlot espera (hasta TRANSACTION_TIMEOUT, si no ErrBusy) a que terminen las transacciones en curso
//     al crearla: sus primeros cambios pueden estar antes de lo que la ranura retiene. Espera sin closeMu, contada en
//     Engine.bg: Close la corta (ErrClosed, la ranura se borra; R4, tarea 112). Su ConfirmedLSN queda en el
//     último registro reservado después (no en Published: con WAL_BUFFERS un commit diferido puede seguir en el
//     búfer, tarea 102), así nunca entrega una transacción a medias.
//   - Changes(slot) lee desde RestartLSN; ChangeReader.Next corre sin closeMu, contado en Engine.bg (Close lo espera
//     y lo corta entre registros con ErrClosed), así fn puede leer del motor aunque Close esté en curso (R3, tarea
//     112); pasa cada registro por un búfer de reordenamiento por xid
//     y, en cada confirmación (KindCommit, o una escritura suelta con FlagCommit), entrega a fn sus cambios en el orden
//     del WAL (ChangePut con el valor nuevo, ChangeDelete, ChangeCreate de Store o StoreIn, ChangeDrop de DropStore o
//     de la cola de DDL, ChangeEmpty de Empty), con TxInfo{XID, CommitLSN}.
//     Las confirmaciones salen en el orden del WAL, que es el de confirmación. Lo confirmado hasta ConfirmedLSN se
//     salta. KindAbort descarta; una transacción que nunca confirmó (murió con un corte del primario, o no escribió) se
//     descarta en el primer KindCheckpoint cuyo xid vivo más viejo la supera. Los DDL sueltos (Store, DropStore, Empty)
//     salen como una transacción propia de xid 0. El KindDropStore de xid 0 que deshace un store creado dentro de un
//     lote no es un DDL suelto: si el lote sigue abierto (dropOwnCreate, o un disco lleno en createIn) el store queda
//     cancelado dentro de ese lote y al confirmar se saltan su create y sus cambios; si el lote ya abortó (undoDDL),
//     se descarta (el lector anota qué archivos crea cada lote sin confirmar: crece con los stores, no con las claves).
//   - Memoria: cada cambio se guarda codificado; si lo que el lector tiene en memoria pasa de logicalWorkMem (16 MB),
//     la transacción más grande pasa a disco (replslot/, formato en «Formatos en disco») y sus cambios siguientes se
//     agregan detrás. Al entregarla se recorre su archivo con un búfer fijo y después lo de memoria; el archivo se
//     borra al entregarla o descartarla. Lo que vive en RAM: hasta logicalWorkMem, una entrada por transacción en curso
//     y los trozos de la cadena del valor que se está armando (un valor, como Read).
//   - Confirm deja la ranura en ConfirmedLSN = el último registro leído y RestartLSN = el primer registro de la
//     transacción más vieja todavía en el búfer: un lector nuevo de la ranura (otro proceso, otro Open) relee desde ahí
//     y salta lo ya entregado, sin repetidos ni huecos. Si fn devuelve un error, el lector queda inservible (la
//     transacción salió a medias): se cierra y se abre otro.
//
// Pruebas (replica_test.go, slots_test.go, decode_test.go; en wal, TestReadAppendCopy): TestReplicaCatchUp (escritores
// sueltos, diferidos y en lotes, DDL suelto y en lote, Empty y checkpoints que reciclan en el primario; lectores con
// snapshots estables en la réplica; al final mismo contenido, Count y Verify limpio; reabrir la réplica sigue;
// promoverla con Open), TestReplicaCrash (corte de la réplica en un punto al azar de una tanda, en los tres modos de
// crashfs, 5 semillas: reabre, alcanza y coincide), TestReplicaNotFromScratch, TestSlotRetains (retiene tras varios
// checkpoints, recicla al avanzar, errores, Stats y Pressure), TestSlotCap (invalidación por el tope con el motor sano;
// sin tope frena), TestSlotPersist (Close/Open y corte en los tres modos), TestChangesOrder (orden de confirmación,
// nunca un lote deshecho, modelo = motor), TestChangesSpill (una transacción de 1.500 cambios con desborde pasa a disco
// con 64 KB de memoria), TestChangesRestart (un consumidor que confirma y reabre, también con Close/Open del motor y un
// lote abierto que cruza la confirmación, recibe lo mismo que uno que nunca paró).
//
// A confirmar (tarea 46): esperar a los snapshots de la réplica en vez de llevar las transacciones en curso del
// primario; replicaConflictDelay y logicalWorkMem como constantes; los avances de una ranura se guardan en el
// checkpoint (entrega al menos una vez tras un corte del primario); 16 ranuras de hasta 64 bytes; la réplica no
// archiva ni admite ranuras lógicas; no hay réplica síncrona (el primario no espera a la réplica al confirmar); Sync
// de josefina (gancho en proceso) se arma sobre Changes.
//
// Flujo por posición y filtro (tarea 75, changes.go): ChangePos{Restart, Confirmed} es una posición del flujo (se lee
// desde Restart y se salta lo confirmado hasta Confirmed). ChangesWith(ranura, f) y ChangesFrom(pos, f) aplican un
// ChangeFilter: Origin por transacción (TxInfo.Origin), Store por cambio, ID por cambio antes de armar un valor en
// desborde. ChangesFrom no usa ranura: no retiene WAL (ErrWalRemoved si ya se recicló), anda en una réplica, sus
// archivos de derrame se llaman «~n-…» y Confirm da ErrSlotNotFound; quien lee guarda Pos() (Restart = el primer
// registro de la transacción más vieja en su búfer, o lo leído + 1; Confirmed = lo leído) y AdvanceLogicalSlot(nombre,
// pos) lleva una ranura lógica a una posición sin abrir su lector. ChangeHead no espera a los lotes abiertos: cada lote
// anota en su primera escritura (writeIn, storeIn, dropStoreIn: noteFirst) una cota de su primer registro (el último
// lsn reservado + 1, wal.LastLSN) en headState, un mapa acotado por las transacciones abiertas, y la saca al terminar,
// después de su registro final; una escritura suelta con desborde hace lo mismo antes de escribir su cadena (prepare)
// y la saca después de su hoja (tarea 103: sin eso, un lector que partía entre la cadena y la hoja daba ErrCorrupt,
// TestChangeHeadOverflowWrite); ChangeHead da Restart = el menor de esas cotas (o el último reservado + 1) y Confirmed = el
// último reservado, todo bajo headState.mu. Batch.CommitLSN es el TxInfo.CommitLSN de su transacción.
//
// Origen (tarea 83): Batch.SetOrigin(n) guarda n en el lote; su KindCommit lleva wal.FlagOrigin y origin(2) al final
// (formato en «Formatos en disco»), y el flujo lo da en TxInfo.Origin. Un nodo multi de josefina marca lo que aplica
// por replicación con el nodo de donde vino y filtra por origen lo que publica.
//
// Ranuras y topología (tarea 76): alcanzan las 16 de siempre, con una lógica por nodo origen (leída por todos los
// vecinos con ChangesFrom desde su posición, y avanzada al mínimo de ellas con AdvanceLogicalSlot) más una física por
// réplica del modo single (TestSlotTopology).
//
// Promote (tarea 85, promote.go): con ckptMu → applyMu, antes de tocar nada, rota el WAL (startPrimaryWAL →
// wal.StartSegment, tarea 138: con registros en el activo lo sella como SwitchWAL, con el encabezado del siguiente
// con fsync antes de la marca de sellado; sin registros y con encabezado de una versión anterior lo reescribe) y, con
// WAL_ARCHIVE=on, arranca el archivo como Open (startArchive: archive_status, retención desde el segmento más viejo,
// el sellado pendiente de archivar); así el primer registro como primario va a un segmento nuevo del formato actual.
// Con el disco lleno ahí da ErrDiskFull y sigue siendo réplica. Un corte después de rotar deja un directorio que se
// abre con Open (termina la promoción); con OpenReplica, el próximo ApplyWal del primario viejo da ErrOutOfOrder (su
// WAL ya no sigue las posiciones del primario: lo mismo que tras dropPending). Después, con replayMu, termina la
// recuperación como Open (dropPending,
// syncHandles, horizonte del archivo de estados con status.Recover y el próximo xid), marca replicaState.promoted —el
// puntero e.replica no cambia nunca, así nadie lo lee a medio escribir; todo pregunta isReplica()—, arranca la limpieza
// de fondo y, con ckptMu todavía tomado, hace el checkpoint de fin de recuperación (checkpointLocked, con su
// KindCheckpoint). ApplyWal da ErrNotReplica desde ahí; los snapshots abiertos siguen; los stores abiertos antes
// siguen sin marcas de visibilidad (NoHints) hasta reabrir el motor.
//
// # Muchos stores y DDL en lote (tareas 86 y 98)
//
// Alta suelta (Store): bajo ddlMu se escribe KindCreateStore sin esperar su fsync, se da de alta en el catálogo de
// memoria y se crea el árbol (KindInitStore); ddlLSN queda en el último lsn reservado; ya sin ddlMu, handle espera el
// fsync hasta ahí (las altas concurrentes comparten uno; mientras tanto otros ya ven el store, y lo que confirmen de
// forma durable cubre el alta, que va antes en el WAL). Un alta en un lote (StoreIn) no espera: la cubre el commit.
// El catálogo ya no se guarda en cada DDL (catalogChanged → catalogBehind): lo guarda el paso 3 del checkpoint, que
// antes espera el fsync de ddlLSN, así el catálogo guardado nunca tiene un alta que no es durable; la réplica sigue
// guardándolo tras cada DDL que rehace (storeCatalog). smgr.Create y smgr.Unlink no hacen fsync de data/: lo hace
// el checkpoint antes de que el control pase su registro, SyncAll en el paso 2 y otra vez smgr.SyncDir en el paso 3
// con ddlMu (una baja con registro antes del punto de rehacer pudo hacer su unlink después del paso 2, pero antes de
// soltar ddlMu: D1, TestDropUnlinkAfterSyncAll). Al recuperar: los stores sueltos dados
// de alta en lo rehecho sin página meta (su KindInitStore no llegó a ser durable: su Store no volvió bien) se borran
// sin WAL (dropUninitialized, como dropPending), y los archivos de datos con número ≥ NextFileNo (de un alta cuyo
// registro se perdió) también (removeUnallocated). La caché de descriptores de smgr cierra uno con escrituras sin
// fsync con vfs.FlushFile (fsync simple en macOS) en lugar de F_FULLFSYNC. Medido: 1.590 altas, de 17,5 a 4,4 ms
// cada una; el checkpoint que sigue, de 10 a 1,2 s (TestManyStoresBench, STORE_BENCH=1).
//
// DropStoreIn sobre un store en el que escribieron otros lotes abiertos (pregunta 73) los espera con el lote suelto,
// mirando cada vez más espaciado (hasta 10 ms), hasta LOCK_TIMEOUT sin pasar el tiempo del lote; después ErrBusy, sin
// abortarlo. Los stores que borró un lote confirmado se recuerdan con su xid (droppedNames, con ddlMu) mientras quede
// un snapshot que no ve el borrado (pregunta 74; se olvidan cuando el xid es menor que OldestActive): StoreIn de un
// lote más viejo con ese nombre da ErrSerialization en lugar de crear otro.
//
// # Respaldo hacia un flujo y por hora (tarea 77)
//
// BackupTo(sink) hace los mismos pasos que Backup, pero cada archivo va a sink.File(nombre, largo) una vez y entero
// (lo que pide un tar), con «/» y relativo a la copia; el catálogo, el control y backup_label se arman en un disco en
// memoria (crashfs.New) con el mismo código y se entregan después. El marcador sale «en curso» al principio y el final
// al terminar: al extraer un tar el segundo pisa al primero, y una copia cortada no abre (ErrBackupIncomplete).
// OpenAtTime(dir, t) traduce la hora a un lsn sin escribir nada (timeTarget): recorre el WAL desde el punto de rehacer
// del control y el objetivo es el registro anterior al primero con hora posterior a t (KindCommit y KindCheckpoint en
// sus primeros 8 bytes, KindBackupEnd en el último campo; recordTime); sin registros posteriores, el último del WAL;
// antes del fin del respaldo, ErrRecoveryTarget.
//
// # Escritor de fondo y escrituras concurrentes (tareas 97 y 100)
//
// El escritor de fondo de la caché limpia también, en una segunda pasada, las sucias que pasan la mitad de la caché
// aunque estén en uso (buffer, «BgWriterDirtyTarget»). Las confirmaciones durables concurrentes comparten el fsync y
// los registros salen del búfer del WAL por tandas (wal, «Búfer del WAL y group commit»); partir una hoja pide antes su
// página (fsm.Grow); page.CheckItems no ordena. Medido en josefina (test/bench, 100 workers): de 3.283 a 5.448
// escrituras/s con p50 de 13,8 a 8,0 ms (Pebble, el mismo día: 5.270–5.504 y 8,3–8,5 ms); test/startup de 35.216 a
// ~75.000 docs/s (Pebble 84.495). El límite que queda es el fsync (F_FULLFSYNC ~4 ms en macOS).
//
// # Diseño recursivo
//
// Tarea 41 de F-23, solo en pruebas (store/recursive_test.go, package store_test, solo API pública): las piezas de
// una base de datos se arman encima del store.
//
//   - Índice secundario: la colección users (id → JSON) tiene un store por índice, users_by_city, con claves
//     valor\x00id y valor vacío (el orden del árbol agrupa por valor y luego por id; el valor no puede tener NUL). El
//     documento y sus entradas van en el mismo lote: ReadIn del documento viejo, RemoveIn de la entrada vieja solo si
//     el campo cambió, PutIn de la nueva y del documento. Dos lotes sobre el mismo documento chocan en su id
//     (ErrSerialization o ErrConflict) y el perdedor se deshace y reintenta. Consulta: ForEachPrefixAt(valor\x00) sobre
//     un snapshot y ReadAt del documento con ese mismo snapshot.
//   - Catálogo: un store catalog (nombre → definición con sus índices) que se escribe en el mismo lote que los primeros
//     documentos; se lee con un snapshot al abrir. La prueba usa Engine.Store y DropStore sueltos (es anterior al DDL
//     en lote), así que el catálogo es la verdad: un store que no describe es huérfano y se borra al abrir. Con StoreIn
//     y DropStoreIn (ver «DDL dentro de un lote») el catálogo y sus stores nacen y mueren juntos.
//   - Se comprueba: 8 escritores concurrentes (mitad con CommitDeferred) sobre 48 ids, con lectores que en cada
//     snapshot comparan índice y recorrido completo, y al final índice vs. documentos; y tres cortes de proceso
//     (SIGKILL al hijo que escribe) tras los que índice y datos coinciden y cada lote confirmado está. El corte de
//     proceso no pierde lo no sincronizado (eso lo cubre TestModelCrash con crashfs, que no es público).
//
// Costo por nivel (Apple M5, macOS, un documento por lote, -benchtime 2s; Commit espera el fsync con F_FULLFSYNC):
//
//	benchmark                          Commit                CommitDeferred
//	PutPlain (sin índice)              4,05 ms, 39 allocs    7,8 µs, 40 allocs
//	PutIndexed idx=1                   4,01 ms, 54 allocs    15,5 µs, 58 allocs
//	PutIndexed idx=2                   4,01 ms, 67 allocs    18,5 µs, 71 allocs
//
//	consulta de 200 de 20.000 documentos    ns/op       allocs/op
//	QueryByIndex (prefijo + ReadAt)         586 µs      4.049
//	ScanFilter (recorrer y decodificar)     9,1 ms      280.270
//
// Con fsync el índice no cuesta nada visible (el fsync es todo); sin él, el primer índice duplica la CPU de la
// escritura y cada uno más suma unos 3 µs. La consulta por índice es 15 veces más rápida que el recorrido con 1 %
// de selectividad.
//
// Índice único (pregunta 67): sin API nueva. El índice único usa como clave el valor solo (sin \x00id) y se escribe
// con InsertIn en el mismo lote que el documento: si el valor ya existe, InsertIn devuelve false (duplicado), y dos
// lotes concurrentes que escriben la misma clave chocan con ErrSerialization (no hay write skew porque escriben la
// misma clave). Al cambiar el valor, el lote borra la clave vieja con DeleteIn.
//
// Decidido no hacer por ahora: un recorrido dentro de un lote (ForEachIn, pregunta 64: se recorre con un snapshot,
// ForEachAt, y dentro del lote se lee por id con ReadIn).
// El DDL dentro del lote (pregunta 66) se hizo después: tarea 67.
package store
