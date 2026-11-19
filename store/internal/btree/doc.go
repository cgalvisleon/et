// Package btree es el árbol B+ de cada store, al estilo del nbtree de PostgreSQL (Lehman-Yao): hojas e internas
// enlazadas con sus hermanas (izquierda y derecha), clave alta en cada página que no es la de más a la derecha de su
// nivel, búsqueda sin bloquear el árbol (una página bloqueada a la vez; si la clave pasa la clave alta, se sigue a la
// derecha), partición en dos registros del WAL, página meta en el bloque 0 y los valores en las hojas (pregunta 17).
//
// Qué hace: buscar la versión que ve una lectura, escribir una versión nueva con su condición comprobada con la hoja
// bloqueada, partir páginas, crear la raíz, terminar particiones incompletas, la inserción rápida al final (una pista
// por store: la hoja más a la derecha), recorrer (fase 2), poner las marcas de visibilidad, quitar versiones muertas
// (fase 1B) y rehacer sus Kind del WAL. Qué no hace: no espera a otras transacciones (devuelve WaitError y quien llama espera sin páginas
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
//	  0   4 flags     EntryTombstone, EntryOverflow, las marcas de visibilidad EntryXidCommitted, EntryXidAborted,
//	                  EntryFrozen (se ponen con OR atómico, ver buffer) y EntryFreeing (versión muerta cuya cadena
//	                  de desborde se está liberando: solo con EntryOverflow)
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
// # Registros del WAL (deltas, little-endian)
//
// Write.UnderLock recibe el lsn y el delta de Count de la escritura (+1, -1 o 0): el motor suma sus contadores bajo
// el writeMu del WAL.
//
// Cada registro sigue el orden de un cambio (store/doc.go): páginas bloqueadas en exclusivo → registro con las
// imágenes que hagan falta → wal.Log.Reserve (ErrNeedImage: agregar la imagen y repetir) → aplicar con la misma función
// que rehace (applyBlock) → pd_lsn y MarkDirty → soltar → Write. Todos los bloques son del fork principal de un store.
//
//	Kind             bloques                                   delta
//	KindInitStore    0: meta (bloque 0), BlockWillInit         root(4) = 1
//	                 1: hoja raíz (bloque 1), BlockWillInit     vacío (hoja vacía con OpaqueLeaf|OpaqueRoot)
//	KindLeafWrite    0: la hoja                                off(2) + la entrada con lsn 0 (el lsn del registro se
//	                                                           pone al aplicar); FlagCommit, FlagCountAdd/Sub
//	KindSplit        0: la izquierda                           right(4) keep(2) + clave alta nueva (pivote sin enlace):
//	                                                           conserva sus primeros keep ítems de datos, enlaza a right
//	                                                           y se marca OpaqueIncompleteSplit
//	                 1: la derecha nueva, BlockWillInit         prev(4) next(4) level(4) flags(2) count(2) + count ×
//	                                                           [len(2) ítem]: su clave alta (la vieja de la izquierda, si
//	                                                           tenía) y sus ítems de datos
//	                 2: la antigua hermana derecha (si había)   prev(4) = la derecha nueva
//	KindInsertPivot  0: el padre                               off(2) + pivote (clave alta de la hija → su hermana nueva)
//	                 1: la hija (la izquierda de la partición)  vacío: apaga OpaqueIncompleteSplit
//	KindNewRoot      0: la raíz nueva, BlockWillInit            level(4) left(4) + pivote de la derecha; la raíz tiene
//	                                                           (menos infinito → left) y ese pivote, con OpaqueRoot
//	                 1: meta                                   root(4) level(4) (la raíz rápida es la misma)
//	                 2: la raíz vieja                          vacío: apaga OpaqueIncompleteSplit y OpaqueRoot
//	KindLeafVacuum   0: la hoja                                nMark(2) nRemove(2) + nMark × off(2) + nRemove ×
//	                                                           off(2): marca EntryFreeing en las versiones con
//	                                                           desborde de la primera lista, quita las de la segunda
//	                                                           (cada lista ascendente, sin números comunes) y compacta
//	                                                           con page.Repair; xid = el de la limpieza
//	KindSetHints     0: la hoja (con su imagen: va solo como   count(2) + count × [off(2) flags(4)]: OR de
//	                    primera marca tras un checkpoint)      EntryXidCommitted / EntryXidAborted; xid 0
//
// Cada estado intermedio es un árbol válido: entre KindSplit y su KindInsertPivot (o KindNewRoot) la derecha nueva
// se alcanza por el enlace de la izquierda. KindNewRoot toca dos niveles (la raíz nueva y la vieja), como el
// XLOG_BTREE_NEWROOT de nbtree (a confirmar: btree/doc.go decía que ningún registro tocaba más de un nivel).
// Los dos de la ola 5B se rehacen como los demás (imagen sin condiciones, o el delta si pd_lsn < lsn): rehacerlos dos
// veces deja lo mismo. Las marcas van como KindSetHints en el rango de btree, no como KindPageImage del motor: así el
// motor no tiene que saber de hojas (como XLOG_FPI_FOR_HINT, pero con el delta de las marcas además de la imagen).
//
// # Implementación (ola 3, fase 1A)
//
// BTree (Open, Create) implementa Tree; Redo rehace sus Kind (valida la forma y todos los deltas antes de tocar
// páginas: un bloque con BlockWillInit se pone en ceros al fijarlo). Lo único en memoria es la pista de la inserción
// rápida (la última hoja de más a la derecha escrita, un uint32 por store; se usa si la raíz está en el nivel 1 o más,
// la hoja sigue siendo la de más a la derecha, sin partición incompleta, con lugar y la clave va después de su primera).
//
//   - Bajada (descend): lee la raíz de la meta y baja con una página bloqueada a la vez, en compartido salvo la del
//     nivel buscado; en cada página, si la clave es ≥ que su clave alta, sigue a la derecha (moveRight). Una bajada
//     para escribir que encuentra una página con OpaqueIncompleteSplit la suelta, termina esa partición y repite
//     (como _bt_moveright con forupdate): así el padre de una partición siempre existe.
//   - Escritura (Write): busca la posición de (id, MaxLSN) —antes que todas las versiones del id, porque su lsn será el
//     mayor—, mira las versiones del id (siguiendo a la derecha, con la hoja bloqueada y la hermana en compartido) para
//     armar Latest, aplica la condición; si la entrada no cabe, parte la hoja, termina la partición y vuelve a bajar.
//   - Partición: el punto de corte equilibra bytes (90 % a la izquierda en la hoja de más a la derecha cuando la
//     inserción va al final, 70 % en una interna; 50 % si no; a confirmar). La clave alta nueva de una hoja es el
//     separador truncado; la de una interna, la clave de su primer enlace a la derecha, que ahí pasa a ser menos
//     infinito. Después busca el padre bajando otra vez hasta la clave alta (sin pila) y siguiendo a la derecha hasta
//     el enlace a la hija; si el padre no tiene lugar, lo parte y repite.
//   - Lectura (Lookup): la primera versión visible del id (sin snapshot: la más nueva confirmada; sin Config.Status,
//     toda versión cuenta como confirmada; el motor siempre pasa el archivo de estados), siguiendo a la derecha.
//   - Recorridos (Scan, ScanBackward): bajan una vez y avanzan por las hojas; leen cada hoja entera, la sueltan y
//     recién entonces llaman al callback. Hacia atrás, el primer id de cada hoja queda pendiente hasta leer la anterior
//     (sus versiones más nuevas pueden estar ahí), y para ir a la izquierda se sigue a la derecha desde la hermana
//     anotada hasta la página cuyo enlace derecho es la recién leída (como _bt_walk_left, sin borrado de páginas).
//   - Verify comprueba la estructura entera (pruebas y el Verify del motor, fase 3).
//
// # Marcas de visibilidad (ola 5B)
//
// statusOf mira primero las marcas de la entrada (una «liberándose» cuenta como abortada) y, si no hay, el archivo de
// estados; si el estado es definitivo —abortada, o confirmada con xact.HintAllowed: el WAL durable hasta el lsn de
// confirmación del grupo de su xid (pregunta 49)— lo anota en la entrada (hint). Lo hacen Lookup, Scan, ScanBackward y
// la búsqueda de Latest de Write; la limpieza no marca.
//
//   - Sin WAL: si el pd_lsn de la página es ≥ el punto de rehacer vigente (la página ya tiene su imagen en el WAL desde
//     el último checkpoint), OR atómico sobre el uint32 de marcas (alineado: los ítems van a 8) y MarkDirtyHint, con
//     el bloqueo que haya (compartido o exclusivo). Config.HintGate, si está, se toma en compartido para decidir y
//     marcar, y el checkpoint la toma en exclusivo alrededor de BeginCheckpoint (como DELAY_CHKPT_START); sin ella, se
//     vuelve a mirar el punto de rehacer después de marcar y, si un checkpoint empezó entre medio, la marca se anota
//     además en el WAL (queda una ventana corta en la que la página podría ir al disco rota sin imagen: por eso el
//     motor debería pasar la compuerta).
//   - Con WAL: si la página es anterior al punto de rehacer, la marca queda diferida (hasta 256 por operación, con su
//     clave) y, ya sin páginas bloqueadas, flushHints bloquea la hoja en exclusivo, busca cada versión por su clave
//     (si una partición la movió, se salta) y escribe un KindSetHints, que lleva la imagen de la página antes de las
//     marcas (reserve la agrega porque pd_lsn < punto de rehacer). Después la página tiene pd_lsn ≥ punto de rehacer y
//     las marcas siguientes van sin WAL. Los errores de flushHints se ignoran (una marca es una pista).
//
// # Limpieza (ola 5B, fase 1)
//
// Vacuum(VacuumOptions{Horizon, XID, Free}) recorre las hojas de izquierda a derecha por sus enlaces, una bloqueada en
// exclusivo a la vez. Por id, de la versión más nueva a la más vieja: la de referencia es la primera confirmada
// visible para todos (EntryFrozen, o xid < Horizon: toda transacción anterior terminó y todo snapshot la ve); están
// muertas las de transacciones abortadas y todas las más viejas que la de referencia (las versiones de un id que
// siguen en la hoja de la derecha heredan la de referencia de la izquierda); la de referencia, si es una lápida y no
// le queda nada detrás, también se quita, salvo que la clave alta de la hoja sea del mismo id (sus versiones viejas
// pueden seguir a la derecha: a confirmar, queda hasta la fase 2). Las muertas en línea se quitan; las de desborde se
// marcan EntryFreeing en el mismo KindLeafVacuum (pregunta 58). Con la hoja ya soltada, Free libera cada cadena
// marcada (overflow.Manager.FreeTail: desde el final y repetible) y un segundo KindLeafVacuum quita esas versiones,
// buscándolas por su clave (siguiendo a la derecha si una partición las movió). Una versión «liberándose» es
// invisible para toda lectura y no cuenta para Latest; si un corte (o un error de Free) la deja marcada, la próxima
// pasada sigue liberando lo que falta de su cadena y la quita. Las hojas no se borran (fase 2); el espacio se
// reutiliza porque page.DeleteItems cierra los huecos y page.Repair compacta. Quien llama no corre dos limpiezas del
// mismo store a la vez (a confirmar: el motor las serializa).
//
// Toda página del árbol que se bloquea pasa por page.CheckItems (la caché solo verifica la suma; a confirmar: un
// gancho de la caché al cargar del disco evitaría repetirlo en cada acceso).
//
// Bloqueos: como nbtree, la partición, terminar una partición y la raíz nueva fijan páginas con otra bloqueada (la
// página nueva de fsm, la antigua hermana derecha, la meta, el padre); siempre de izquierda a derecha, de hija a
// padre, árbol → meta, y nunca se espera a nada entre Reserve y Write, así que expulsar una página sucia (que espera
// al WAL) no puede cerrar un ciclo (a confirmar en store/doc.go, que hoy lo prohíbe en general).
package btree
