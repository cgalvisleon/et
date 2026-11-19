// Package btree es el árbol B+ de cada store, al estilo del nbtree de PostgreSQL (Lehman-Yao): hojas e internas
// enlazadas con sus hermanas (izquierda y derecha), clave alta en cada página que no es la de más a la derecha de su
// nivel, búsqueda sin bloquear el árbol (una página bloqueada a la vez; si la clave pasa la clave alta, se sigue a la
// derecha), partición en dos registros del WAL, página meta en el bloque 0 y los valores en las hojas (pregunta 17).
//
// Qué hace: buscar la versión que ve una lectura, escribir una versión nueva con su condición comprobada con la hoja
// bloqueada, partir páginas, crear la raíz, terminar particiones incompletas, la inserción rápida al final (una pista
// por store: la hoja más a la derecha), recorrer (fase 2), quitar versiones muertas (fase 1B) y rehacer sus Kind del
// WAL. Qué no hace: no espera a otras transacciones (devuelve WaitError y quien llama espera sin páginas
// bloqueadas), no guarda valores grandes (recibe la overflow.Ref ya escrita), no confirma transacciones. Depende de
// page, smgr, buffer, fsm, overflow (Ref), xact (Status, Snapshot) y wal por sus interfaces.
//
// # Clave
//
// (id ascendente por bytes, lsn de la escritura descendente): las versiones de un id quedan juntas y la más nueva
// primero (preguntas 24, 25 y 39). Pueden repartirse en dos hojas vecinas: una lectura sigue a la derecha mientras la
// clave siga siendo el mismo id.
//
// # Entrada de hoja (EntryHeaderSize = 28 + id + valor o referencia; little-endian)
//
//	off tam campo
//	  0   4 flags     EntryTombstone, EntryOverflow y las marcas de visibilidad EntryXidCommitted, EntryXidAborted,
//	                  EntryFrozen; las marcas se ponen con OR atómico (ver buffer)
//	  4   2 idLen     1 a MaxIDLen
//	  6   2 reservado
//	  8   8 lsn       lsn del registro que escribió la versión (parte de la clave)
//	 16   8 xid       transacción que la escribió
//	 24   4 valueLen  largo del valor (en línea, o total si está en desborde; 0 en una lápida)
//	 28   … id, después el valor en línea, o la referencia a desborde: primer bloque (4)
//
// Una versión va en línea si EntryHeaderSize + idLen + valueLen ≤ MaxEntrySize; si no, a desborde (preguntas 22 y
// 37). MaxEntrySize = 2.032 garantiza que tres entradas máximas y una clave alta máxima entren en una hoja, como el
// tercio de página de PostgreSQL. Un borrado es una lápida (EntryTombstone, sin valor); el reemplazo no toca la
// versión anterior (pregunta 38). Remove de un id sin versión viva no escribe nada.
//
// # Pivote (páginas internas y claves altas; PivotHeaderSize = 16 + clave)
//
//	off tam campo
//	  0   2 keyLen    largo de la clave (0 = menos infinito: el primer enlace de una página interna)
//	  2   2 flags     PivotHasLSN: el separador lleva también el lsn (el corte cayó entre versiones de un id)
//	  4   4 downlink  bloque de la hija (0 en una clave alta)
//	  8   8 lsn       con PivotHasLSN; si no, 0
//	 16   … clave     prefijo del id: el más corto que separa la última clave de la izquierda de la primera de la
//	                  derecha (truncado de sufijos de nbtree)
//
// Un pivote (k, sin lsn) manda a la derecha toda versión con id ≥ k; uno (k, lsn) manda a la derecha las versiones
// de id k con lsn ≤ lsn y todas las de id > k. Como en nbtree, el ítem 1 de una página que no es la de más a la
// derecha es su clave alta; los datos empiezan en el 2 (en la de más a la derecha, en el 1).
//
// # Página meta (bloque 0; contenido desde page.HeaderSize, MetaSize = 48)
//
//	 0   4 magic      MetaMagic ("ETBT")
//	 4   4 version    1
//	 8   4 root       bloque de la raíz
//	12   4 level      nivel de la raíz
//	16   4 fastRoot   raíz rápida (la página más baja con un solo hijo por nivel), como btm_fastroot
//	20   4 fastLevel
//	24   4 freeHead   primera página de la lista de libres (fsm); 0 = vacía
//	28   4 reservado
//	32   8 frozenXID  horizonte de congelado del store (fase 2, pregunta 44)
//	40   8 reservado
//
// Su espacio especial lleva flags = page.OpaqueMeta. Un store nuevo nace con KindInitStore: la meta y una hoja raíz
// vacía en el bloque 1.
//
// # Registros del WAL (deltas)
//
//   - KindLeafWrite, un bloque (la hoja): delta = offset (2) donde va el ítem + la entrada completa; flags del
//     registro FlagCountAdd/FlagCountSub según la versión anterior viva o no, y FlagCommit si es una operación
//     suelta.
//   - KindSplit, KindInsertPivot, KindNewRoot, KindInitStore, KindLeafVacuum: formato propio de este paquete (ola
//     3), con la regla de que cada estado intermedio sea un árbol válido (OpaqueIncompleteSplit entre la partición y
//     el enlace en el padre) y ningún registro toque más de un nivel, salvo la partición con su enlace a la izquierda
//     (tres páginas del mismo nivel).
package btree
