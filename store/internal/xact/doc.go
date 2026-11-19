// Package xact lleva las transacciones del motor: asigna xid, guarda su estado (en curso, confirmada, abortada) en
// el archivo de estados —como pg_xact—, sabe cuáles están en curso, toma snapshots y hace esperar a una transacción
// que choca con otra.
//
// Qué hace: el archivo de estados por páginas con una caché de tamaño fijo (como el SLRU de PostgreSQL); el lsn de
// confirmación por grupo de xids; la lista de transacciones en curso; la tabla de esperas por xid (acotada por las
// transacciones en curso); la regla de visibilidad de un snapshot. Qué no hace: no escribe en el WAL (el motor anota
// KindCommit/KindAbort y después llama a SetStatus), no sabe de páginas del árbol. Depende de vfs, page (XID, LSN) y
// wal (Flusher: una página del archivo de estados no se escribe al disco antes de que el WAL sea durable hasta el
// mayor lsn de confirmación de sus grupos).
//
// # Archivo de estados (xact/<n:%04d>, n desde 1)
//
// Páginas de PageSize = 8.192 bytes sin encabezado ni suma (como pg_xact): 2 bits por xid, XidsPerPage = 32.768 xids
// por página, PagesPerFile = 4.096 páginas por archivo (32 MB, unos 134 millones de xids). El xid x está en el
// archivo 1 + x / XidsPerFile, página (x % XidsPerFile) / XidsPerPage, byte (x % XidsPerPage) / 4, bits
// 2 × (x % 4) y siguiente. Los bits solo pasan de StatusInProgress a confirmada o abortada, así que una página rota a
// medias por un corte sigue siendo correcta tras reproducir el WAL: no necesita imagen entera.
//
// El lsn de confirmación por grupo (GroupSize = 32 xids, 1.024 grupos por página) vive solo en memoria, junto a cada
// página de la caché, como en PostgreSQL: dice hasta qué lsn tiene que ser durable el WAL antes de poner la marca de
// confirmada en una versión (pregunta 49) y antes de escribir la página al disco. Tras abrir, todo lo que dice el
// archivo ya es durable y el lsn de grupo empieza en 0.
//
// # Implementación (Statuses y Transactions)
//
// Statuses (Open) es el archivo con su caché: CachePages ranuras fijas (DefaultCachePages = 128, 1 MB), cada una con
// su página y el lsn de sus 1.024 grupos. Se reemplaza la menos usada últimamente que no esté en E/S; si está sucia,
// se escribe antes, después de wal.Flusher.WaitDurable(el mayor lsn de sus grupos). Un solo mutex hoja guarda las
// ranuras y se suelta durante toda E/S y toda espera al WAL (la ranura queda marcada como cargándose o
// escribiéndose). Leer una página más allá del final del archivo da ceros (en curso); escribirla extiende el
// archivo. Flush escribe las sucias y hace fsync de los archivos escritos y del directorio. Truncate borra los
// archivos enteros por debajo del horizonte y después Status devuelve ErrTruncated; al abrir, el primer archivo que
// queda marca ese límite. Recover(next), al terminar la recuperación, fija el horizonte y pone en cero los estados
// desde next (resto de su página, páginas y archivos siguientes), como TrimCLOG: así un estado que llegó al disco sin
// registro en el WAL no se pega a un xid que se vuelve a asignar. Un fallo de escritura o fsync deja todo en error.
//
// Transactions (NewManager) guarda solo las transacciones en curso: cada una con un canal que End cierra (Wait) y el
// Xmin de su primer snapshot (OldestActive).
//
// # Visibilidad
//
// Un Snapshot es (Xmin, Xmax, InProgress): el xid más viejo en curso al tomarlo, el siguiente xid a asignar y la
// lista ordenada de los que estaban en curso entre ambos. Una versión con xid x es visible para el snapshot si x es
// el de la propia transacción, o si x < Xmax, x no está en InProgress y su estado es confirmada. Los xid menores que
// el horizonte de recuperación (el nextXID al terminar Open) que siguen en StatusInProgress se leen como abortados:
// eran transacciones en curso al caerse, y no se marcan una por una.
//
// La visibilidad «última confirmada» (Read suelto sin snapshot, pregunta 41) es: la versión más nueva de su id cuyo
// estado es confirmada.
package xact
