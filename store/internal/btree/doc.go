// Package btree es el árbol B+ de cada store, al estilo del nbtree de PostgreSQL (Lehman-Yao): hojas e internas
// enlazadas con sus hermanas (izquierda y derecha), clave alta en cada página que no es la de más a la derecha de su
// nivel, búsqueda sin bloquear el árbol (una página bloqueada a la vez; si la clave pasa la clave alta, se sigue a la
// derecha), partición en dos registros del WAL, página meta en el bloque 0 y los valores en las hojas (pregunta 17).
//
// Qué hace: buscar la versión que ve una lectura, escribir una versión nueva con su condición comprobada con la hoja
// bloqueada, partir páginas, crear la raíz, terminar particiones incompletas, la inserción rápida al final (una pista
// por store: la hoja más a la derecha), recorrer (Cursor, fase 2), poner las marcas de visibilidad, quitar versiones
// muertas (fase 1B), congelar y llevar el mapa de visibilidad y el frozenXID del store (fase 2), borrar del árbol las
// hojas vacías y los padres que se quedan sin hijos y reciclar sus páginas (tarea 70), y rehacer sus Kind del WAL. Qué no hace: no espera a otras transacciones (devuelve WaitError y quien llama espera sin páginas bloqueadas),
// no guarda valores grandes (recibe la overflow.Ref ya escrita), no confirma transacciones. Depende de page, smgr,
// buffer, fsm (FreeList, VisibilityMap), overflow (Ref), xact (Status, Snapshot) y wal por sus interfaces.
//
// # Clave
//
// (id ascendente por bytes, lsn de la escritura descendente): las versiones de un id quedan juntas y la más nueva
// primero (preguntas 24, 25 y 39). Pueden repartirse en dos hojas vecinas: una lectura sigue a la derecha mientras la
// clave siga siendo el mismo id.
//
// # Entrada de hoja de la forma 1 (EntryHeaderSize = 28 + id + valor o referencia; little-endian)
//
// La de las hojas con LayoutVersion 1: las del formato 1, que este binario lee y reescribe en la forma 2 (ver «Hojas
// de la forma 2»); ya no escribe ninguna.
//
//	off tam campo
//	  0   4 flags     EntryTombstone, EntryOverflow, las marcas de visibilidad EntryXidCommitted, EntryXidAborted
//	                  (se ponen con OR atómico, ver buffer), EntryFrozen (congelada: confirmada y visible para todo
//	                  snapshot sin consultar el archivo de estados; la pone la limpieza con KindLeafVacuum) y
//	                  EntryFreeing (versión muerta cuya cadena de desborde se está liberando: solo con EntryOverflow) y
//	                  EntryCompressed (formato 2, tarea 132: el valor guardado, en línea o en desborde, lo comprimió
//	                  el motor; nunca en una lápida; btree no lo mira: lo pone Write.Compressed)
//	  4   2 idLen     1 a MaxIDLen
//	  6   2 reservado
//	  8   8 lsn       lsn del registro que escribió la versión (parte de la clave)
//	 16   8 xid       transacción que la escribió
//	 24   4 valueLen  largo del valor guardado (en línea, o total si está en desborde; con EntryCompressed, el
//	                  comprimido con su encabezado; 0 en una lápida)
//	 28   … id, después el valor en línea, o la referencia a desborde: primer bloque (4)
//
// Una versión va en línea si EntryHeaderSize + idLen + valueLen ≤ MaxEntrySize, en las dos formas (la regla no depende
// de la hoja donde cae); si no, a desborde (preguntas 22 y 37). MaxEntrySize = 2.032 garantiza que tres entradas máximas y una clave alta máxima entren en una hoja, como el
// tercio de página de PostgreSQL. Con EntryCompressed, valueLen es el del valor comprimido: el motor comprime antes
// de decidir (store/doc.go, «Formato 2»). VerifyWith entrega con su valor las versiones comprimidas en línea (el
// motor comprueba que se descomprimen; las comprimidas en desborde las lee él de su cadena); las demás, sin él. Un borrado es una lápida (EntryTombstone, sin valor); el reemplazo no toca la
// versión anterior (pregunta 38). Remove de un id sin versión viva no escribe nada.
//
// # Hojas de la forma 2 (formato 2, tarea 134 de F-23; leaf2.go)
//
// Una hoja de la forma 2 lleva LayoutVersion 2 en pd_pagesize_version (las demás clases de página siguen en 1, y solo
// una hoja viva puede ser de la 2: pageNode lo rechaza en cualquier otra). Sus ítems:
//
//	ítem           qué guarda
//	1              clave alta (pivote, como en la forma 1; no está en la de más a la derecha)
//	2 (o 1)        ítem de prefijo: un pivote sin enlace ni lsn (flags 0, downlink 0) cuya clave es el prefijo de la
//	               hoja, de 0 a MaxPrefixLen = 255 B
//	resto          entradas de la forma 2, en orden (id, lsn descendente): firstData es uno más que en la forma 1
//
// Entrada de la forma 2 (little-endian):
//
//	off tam campo
//	  0   4 word0     marcas (bits 0–7: las de la forma 1; el OR atómico de las marcas de visibilidad sigue sobre este
//	                  uint32) | largo del sufijo (bits 8–19) | largo en línea (bits 20–31; 0 en lápidas y desborde).
//	                  Los bits altos no cambian nunca después de escribir la entrada
//	  4   8 lsn       parte de la clave
//	 12   8 xid       solo sin (EntryFrozen y no EntryOverflow): una congelada en línea no lo guarda (es visible para
//	                  todos; Version.XID da 0); una con desborde lo conserva (FreeTail lo compara con pd_prune_xid)
//	  …   … sufijo    el id sin el prefijo de la hoja (puede ser vacío: todas las versiones de un mismo id)
//	  …   … valor     en línea, o first(4) len(4) en desborde
//
// 20 B por versión viva (28 en la forma 1) y 12 B congelada en línea. Invariante: el id de toda entrada empieza con el
// prefijo de su hoja; nada más (el prefijo no tiene que ser común a las cotas). El prefijo se elige al armar la hoja
// (choosePrefix): el común de su primera y su última entrada, acotado a 255 B; vacío sin entradas. Así una hoja que
// solo recibe ids crecientes (la de más a la derecha, la cola de un grupo de un índice) ya está recortada cuando se
// parte. Una escritura cuyo id no tiene el prefijo reescribe la hoja antes con un prefijo más corto (needsRewrite,
// rewriteLeaf: el común con ese id); si así no cabe, se parte y vuelve a bajar. Una escritura que no cabe intenta,
// antes de partir y después de podar, recortar las entradas con un prefijo más largo si las entradas lo comparten
// (longerPrefix). Acotar el prefijo a 255 B deja que la partición cuente el ítem de prefijo de cada mitad con su largo
// máximo y las entradas con su largo de antes (al recortarlas solo se achican: el prefijo de cada mitad es igual o
// más largo que el de la hoja), y siempre encuentre un corte.
//
// Lectura: entryAt/keyAt leen las dos formas (parseEntry, parseEntry2); la clave de una entrada de la forma 2 trae el
// prefijo aparte (key.pre) y compareKeys compara los ids enteros sin armarlos. La búsqueda en una hoja recorta la
// clave de búsqueda una vez (node.trim): si su id no tiene el prefijo cae antes o después de toda la hoja; si lo tiene,
// la búsqueda binaria compara solo sufijos. Version.ID es el id entero (una sola asignación).
//
// Diseño corregido al implementar (2026-11-22; store/doc.go «Formato 2: hojas de la forma 2 (b y c)»): el diseño
// sacaba el prefijo de las cotas, con la clave baja como segundo ítem. No sirve tal cual: (1) el borrado de una hoja
// (tarea 70) pasa su espacio de claves a la hermana derecha, que recibe ids menores que su clave baja y sin su
// prefijo, y (2) la izquierda de una partición conservaba su prefijo, así que con ids crecientes (la hoja de más a la
// derecha, sin cota alta, prefijo vacío) ninguna hoja tenía prefijo. Por eso el prefijo sale de las entradas, la
// hoja se reescribe cuando llega un id sin él, cada mitad de una partición elige el suyo, y la clave baja no se guarda.
// El largo va en el ítem de prefijo, no en pd_flags.
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
// derecha es su clave alta; los datos empiezan en el 2 (en la de más a la derecha, en el 1). El downlink de una clave
// alta es 0, salvo en una hoja medio muerta: ahí anota la cima del subárbol que se borra con ella (0 = la hoja misma),
// como el «top parent link» del t_tid de la clave alta en nbtree.
//
// # Página meta (bloque 0; contenido desde page.HeaderSize, MetaSize = 48)
//
//	 0   4 magic      MetaMagic ("ETBT")
//	 4   4 version    1
//	 8   4 root       bloque de la raíz
//	12   4 level      nivel de la raíz
//	16   4 fastRoot   raíz rápida, como btm_fastroot: la página de más a la izquierda del nivel más bajo con una
//	                  sola página; las bajadas a ese nivel o más abajo empiezan ahí (ver «Borrado de páginas»)
//	20   4 fastLevel
//	24   4 freeHead   primera página de la lista de libres (fsm); 0 = vacía
//	28   4 deleted    cabeza de la pila de páginas borradas que esperan reciclarse; 0 = vacía (era reservado, en
//	                  cero: una meta anterior se lee con la pila vacía)
//	32   8 frozenXID  ninguna versión con xid menor queda sin congelar (fase 2, pregunta 44): nace con
//	                  Config.FrozenXID y solo sube (KindSetFrozen)
//	40   4 flags      formato 2 (en una meta del 1, en cero): bit 0 MetaLeavesForm2 = todas las hojas del store son de
//	                  la forma 2 (la pone Create y la limpieza al terminar de reescribir las del formato 1)
//	44   4 reservado
//
// Su espacio especial lleva flags = page.OpaqueMeta. Un store nuevo nace con KindInitStore: la meta y una hoja raíz
// vacía en el bloque 1. Una hoja lleva page.FlagAllVisible en su encabezado mientras su bit del mapa de visibilidad
// puede estar puesto (ver «Congelado y mapa de visibilidad»).
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
//	KindInitStore    0: meta (bloque 0), BlockWillInit         root(4) = 1 + frozenXID(8) + flags(4) (formato 2;
//	                                                           sin flags, formato 1; sin frozenXID, anterior: 0)
//	                 1: hoja raíz (bloque 1), BlockWillInit     layout(2) = 2: hoja vacía de la forma 2 con su ítem
//	                                                           de prefijo vacío (formato 2); vacío: de la forma 1
//	KindLeafWrite    0: la hoja                                off(2) + la entrada con lsn 0 (el lsn del registro se
//	                                                           pone al aplicar); FlagCommit, FlagCountAdd/Sub; apaga
//	                                                           page.FlagAllVisible. Bit 15 de off (formato 2): la
//	                                                           entrada es de la forma 2 con el id entero como sufijo
//	                                                           (el flujo lógico la lee sin la página) y solo marcas
//	                                                           de escritura; va en una hoja de la forma 2 sin su
//	                                                           prefijo. Sin el bit, de la forma 1 en una hoja de la 1
//	KindSplit        0: la izquierda                           right(4) keep(2) + clave alta nueva (pivote sin enlace):
//	                                                           conserva sus primeros keep ítems de datos, enlaza a right
//	                                                           y se marca OpaqueIncompleteSplit; una hoja de la forma 2
//	                                                           se arma con choosePrefix de las que conserva (igual o
//	                                                           más largo: sus entradas se recortan), sin marcas de página
//	                 1: la derecha nueva, BlockWillInit         prev(4) next(4) level(4) flags(2) count(2) + count ×
//	                                                           [len(2) ítem]: su clave alta (la vieja de la izquierda, si
//	                                                           tenía) y sus ítems de datos; bit 15 de count (formato 2):
//	                                                           hoja de la forma 2 (sus ítems traen el de prefijo)
//	                 2: la antigua hermana derecha (si había)   prev(4) = la derecha nueva
//	KindInsertPivot  0: el padre                               off(2) + pivote (clave alta de la hija → su hermana nueva)
//	                 1: la hija (la izquierda de la partición)  vacío: apaga OpaqueIncompleteSplit
//	                 2: meta (opcional)                        fastRoot(4) fastLevel(4) = el padre: la hija era la de
//	                                                           más a la izquierda de su nivel y la raíz rápida estaba
//	                                                           en su nivel o más abajo (sube, como split_only_page)
//	KindNewRoot      0: la raíz nueva, BlockWillInit            level(4) left(4) + pivote de la derecha; la raíz tiene
//	                                                           (menos infinito → left) y ese pivote, con OpaqueRoot
//	                 1: meta                                   root(4) level(4) (la raíz rápida es la misma)
//	                 2: la raíz vieja                          vacío: apaga OpaqueIncompleteSplit y OpaqueRoot
//	KindLeafVacuum   0: la hoja                                nMark(2) nRemove(2) + nMark × off(2) + nRemove ×
//	                                                           off(2) [+ nFreeze(2) pageFlags(2) + nFreeze × off(2)]:
//	                                                           marca EntryFreeing en las versiones con desborde de la
//	                                                           primera lista, EntryFrozen en las de la tercera (en una
//	                                                           hoja de la forma 2 las achica: sin xid si no tienen
//	                                                           desborde, reemplazadas en su lugar) y, con
//	                                                           pageFlags bit 0, page.FlagAllVisible; quita las de la
//	                                                           segunda (los números son de antes de quitar; cada lista
//	                                                           ascendente, sin números comunes) y compacta con
//	                                                           page.Repair; xid = el de la limpieza. Sin la parte del
//	                                                           congelado es el formato de la fase 1B
//	KindSetHints     0: la hoja (con su imagen: va solo como   count(2) + count × [off(2) flags(4)]: OR de
//	                    primera marca tras un checkpoint)      EntryXidCommitted / EntryXidAborted; xid 0
//	KindSetFrozen    0: meta                                   frozenXID(8): solo sube [+ flags(4), formato 2: se suman
//	                                                           a las de la meta]; xid 0
//	KindLeafRewrite  0: la hoja                                prefixLen(2) ≤ 255 (formato 2): arma de nuevo la hoja, de
//	                                                           cualquier forma, en la forma 2 con un prefijo de
//	                                                           prefixLen bytes (el comienzo del id de su primera
//	                                                           entrada; todas lo tienen), con su clave alta, enlaces,
//	                                                           marcas de página (también FlagAllVisible) y pd_prune_xid.
//	                                                           xid 0
//	KindMarkHalfDead 0: la hoja                                top(4): la cima del subárbol (0 = la hoja); la hoja
//	                                                           tiene que estar vacía (solo su clave alta), sin ser la
//	                                                           de más a la derecha ni la raíz, ni partida a medias;
//	                                                           pone OpaqueHalfDead, apaga page.FlagAllVisible y anota
//	                                                           top en el downlink de la clave alta
//	                 1: el padre del subárbol                  off(2) child(4) right(4): el enlace off apunta a child
//	                                                           (la cima) y off+1 a right (su hermana derecha); off pasa
//	                                                           a apuntar a right y se quita off+1. xid 0
//	KindUnlinkPage   0: la que se borra, BlockWillInit          prev(4) next(4) level(4) safeXID(8) nextDeleted(4)
//	                                                           flags(2): la deja borrada (abajo) con esos enlaces y
//	                                                           nivel; flags dice qué bloques opcionales vienen (bit 0
//	                                                           la izquierda, bit 1 la hoja, bit 2 la raíz rápida en
//	                                                           la meta); prev ≠ 0 si y solo si viene la izquierda
//	                 1: la hermana derecha                     prev(4) = prev de la borrada
//	                 2: meta                                   deleted(4) = la borrada [+ fastRoot(4) fastLevel(4)]
//	                 3: la hermana izquierda (si prev ≠ 0)     next(4) = next de la borrada
//	                 4: la hoja medio muerta (si la borrada    top(4): la cima siguiente (el único hijo de la borrada;
//	                    es un padre)                           0 si ese hijo es la hoja). Los bloques 3 y 4 que no
//	                                                           vienen no ocupan lugar: el 4 puede ser el tercero
//	                                                           (blockRole). Hasta cinco bloques (wal.MaxBlocks = 5,
//	                                                           como XLOG_BTREE_UNLINK_PAGE_META). xid 0
//	KindRecyclePage  0: meta                                   deleted(4) freeHead(4): saca la página de la pila y la
//	                                                           pone en la cabeza de la lista de libres
//	                 1: la página, BlockWillInit                next(4) = la cabeza anterior de la lista: queda libre
//	                                                           con el formato de fsm (page.OpaqueFree). xid 0
//
// Página borrada: page.Init con el espacio especial prev, next (los de sus hermanas al borrarla), level y flags =
// OpaqueDeleted (más OpaqueLeaf si era una hoja), y un solo ítem de 16 B: safeXID(8) nextDeleted(4) reservado(4).
// readNode la acepta (checkDeleted: el ítem y next ≠ 0) y una medio muerta (hoja vacía con su clave alta).
//
// Cada estado intermedio es un árbol válido: entre KindSplit y su KindInsertPivot (o KindNewRoot) la derecha nueva
// se alcanza por el enlace de la izquierda; entre KindMarkHalfDead y los KindUnlinkPage de su subárbol, la hoja medio
// muerta y los padres que faltan desenganchar siguen enganchados en sus niveles y se saltan siguiendo a la derecha. KindNewRoot toca dos niveles (la raíz nueva y la vieja), como el
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
// la hoja sigue siendo la de más a la derecha, sin partición incompleta, con lugar y la clave va después de su
// primera).
//
//   - Bajada (descend): lee la raíz de la meta (o la raíz rápida, si el nivel buscado no está por encima de ella; si
//     al bloquearla no es una página viva de su nivel y la de más a la izquierda, baja desde la raíz) y baja con una
//     página bloqueada a la vez, en compartido salvo la del nivel buscado; en cada página, si la clave es ≥ que su
//     clave alta o la página se está borrando (medio muerta o borrada), sigue a la derecha (moveRight). Una bajada
//     para escribir que encuentra una página con OpaqueIncompleteSplit la suelta, termina esa partición y repite (como
//     _bt_moveright con forupdate): así el padre de una partición siempre existe.
//   - Escritura (Write): busca la posición de (id, MaxLSN) —antes que todas las versiones del id, porque su lsn será el
//     mayor—, mira las versiones del id (siguiendo a la derecha, con la hoja bloqueada y la hermana en compartido) para
//     armar Latest, aplica la condición; si la entrada no cabe, parte la hoja, termina la partición y vuelve a bajar.
//   - Partición: el punto de corte equilibra bytes (90 % a la izquierda en la hoja de más a la derecha cuando la
//     inserción va al final, 70 % en una interna; 50 % si no). En una hoja con inserción ascendente local —las 4
//     versiones antes del punto de inserción tienen lsn creciente, como al final de un grupo de duplicados con sufijo
//     creciente (índice valor + jid de josefina)— se parte en el punto de inserción, con la izquierda entre el 50 % y
//     el 90 % (ver «Llenado de las hojas»). La clave alta nueva de una hoja es el
//     separador truncado; la de una interna, la clave de su primer enlace a la derecha, que ahí pasa a ser menos
//     infinito. Después busca el padre bajando otra vez hasta la clave alta (sin pila) y siguiendo a la derecha hasta
//     el enlace a la hija; si el padre no tiene lugar, lo parte y repite.
//   - Lectura (Lookup): la primera versión visible del id (sin snapshot: la más nueva confirmada; sin Config.Status,
//     toda versión cuenta como confirmada; el motor siempre pasa el archivo de estados), siguiendo a la derecha.
//   - Recorridos (Cursor, fase 2; Scan y ScanBackward lo usan): NewCursor(ScanOptions) baja una vez y cada Next lee una
//     hoja entera con bloqueo compartido, la suelta y devuelve sus versiones visibles: entre dos Next no queda nada
//     fijado ni bloqueado, así quien llama suelta sus propios locks antes de usar lo leído. Hacia atrás, el primer id
//     de cada hoja queda pendiente hasta leer la anterior (sus versiones más nuevas pueden estar ahí), y para ir a la
//     izquierda se sigue a la derecha desde la hermana anotada hasta la página no borrada cuyo enlace derecho es la
//     recién leída (como _bt_walk_left: hasta 4 pasos; si no aparece y la recién leída se borró, se sigue a su derecha
//     hasta una no borrada y se busca la izquierda de esa). Hacia adelante, las versiones de ids que ya pasaron se
//     saltan (el espacio de claves de una hoja borrada pasa a su derecha). Opciones: Ring fija las hojas siguientes a la primera con
//     buffer.Ring (un recorrido grande no vacía la caché); KeysOnly no copia los valores; Borrow copia los valores de
//     la hoja a un búfer del cursor que se reutiliza (valen hasta el próximo Next; solo hacia adelante).
//   - Exists: Lookup sin copiar el valor.
//   - Verify comprueba la estructura entera (pruebas). VerifyWith (fase 3, Verify del motor) es lo mismo con
//     VerifyOptions: la estrategia de la caché (buffer.Ring), Page (cada página del árbol, ya soltada) y Leaf (cada
//     hoja ya soltada con sus entradas sin el valor en línea); no pone marcas ni cambia nada, y guarda una hoja de
//     entradas. RecordFrozenXID da el frozenXID que pone un KindInitStore o KindSetFrozen (la recuperación no deja el
//     siguiente xid por debajo).
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
// Vacuum(VacuumOptions{Horizon, FreezeLimit, Aggressive, XID, Free, NextXID, Stop}) recorre las hojas de izquierda a
// derecha (Stop, mirado antes de cada hoja, la corta con ErrVacuumStopped y Complete = false), una
// bloqueada en exclusivo a la vez (por los enlaces del nivel 1 si hay mapa y dos niveles, saltando hojas: ver
// «Congelado y mapa de visibilidad»; si no, por los enlaces de las hojas). Por id, de la versión más nueva a la más
// vieja: la de referencia es la primera confirmada visible para todos (EntryFrozen, o xid < Horizon: toda transacción
// anterior terminó y todo snapshot la ve); están muertas las de transacciones abortadas y todas las más viejas que la
// de referencia (las versiones de un id que siguen en la hoja de la derecha heredan la de referencia de la izquierda);
// la de referencia, si es una lápida y no le queda nada detrás, también se quita, salvo que sus versiones viejas sigan
// a la derecha: que la clave alta de la hoja sea del mismo id y que la primera hoja con entradas a su derecha empiece
// con él (rightStartsWith); así la lápida se va en la pasada siguiente a la que quitó esas versiones. Las muertas en
// línea se quitan; las de desborde se marcan EntryFreeing en el mismo KindLeafVacuum (pregunta 58). Con la hoja ya soltada, Free
// libera cada cadena marcada (overflow.Manager.FreeTail: desde el final y repetible) y un segundo KindLeafVacuum quita
// esas versiones, buscándolas por su clave (siguiendo a la derecha si una partición las movió). Una versión
// «liberándose» es invisible para toda lectura y no cuenta para Latest; si un corte (o un error de Free) la deja
// marcada, la próxima pasada sigue liberando lo que falta de su cadena y la quita. Dentro de una hoja el espacio se
// reutiliza porque page.DeleteItems cierra los huecos y page.Repair compacta; una hoja que queda vacía se borra del
// árbol (ver «Borrado de páginas») y no lleva bits del mapa de visibilidad.
// Quien llama no corre dos limpiezas del mismo store a la vez (el motor las serializa con Store.vacuumMu).
//
// # Congelado y mapa de visibilidad (fase 2)
//
// En la misma pasada, con la hoja bloqueada, la limpieza congela (EntryFrozen) toda versión que queda, confirmada y con
// xid < FreezeLimit, en el mismo KindLeafVacuum. Si todo lo que queda es visible para todos (congelado o confirmado
// con xid < Horizon; ni muertas, ni «liberándose», ni en curso), la hoja no tiene partición incompleta y no comparte un
// id con una vecina (su primer id venía de la hoja anterior, o su último sigue en la hermana derecha, que se mira
// bloqueándola en compartido: si la limpieza ya quitó esas versiones, la clave alta sigue nombrando el id pero ya no
// comparte), el registro pone page.FlagAllVisible y después, todavía con la hoja bloqueada, Config.VM.Set pone su bit:
// VMAllFrozen si todo quedó congelado, si no VMAllVisible. Una abortada nunca se congela: está muerta.
//
// Toda escritura (writeLeaf) o partición (split) que encuentra la hoja con page.FlagAllVisible borra antes su bit
// (Config.VM.Clear, un KindVMClear con la hoja bloqueada y antes de reservar su registro); KindLeafWrite y KindSplit
// dejan la página sin la marca. Así, en todo prefijo del WAL, un bit puesto implica la marca en la página y ningún
// cambio desde que la limpieza la vio.
//
// Recorrido: con Config.VM y la raíz en el nivel 1 o más, Vacuum copia los enlaces de cada página del nivel 1 (de
// izquierda a derecha por sus enlaces, con bloqueo compartido; también los de la siguiente para saber dónde termina la
// última cadena) y, por cada enlace, salta la hoja sin leerla si su bit es VMAllFrozen (o VMAllVisible sin Aggressive),
// o la limpia y sigue a la derecha hasta el enlace siguiente (así ve las hojas de particiones posteriores a la copia o
// sin enlace en el padre). Saltar es seguro: el bit solo se pone en una hoja sin partición incompleta y cualquier
// partición posterior lo borra, así que las hojas a su derecha hasta el enlace siguiente están en la copia.
// VacuumStats: Frozen, Skipped, SkippedVisible, Complete (sin error y sin saltar hojas solo todo-visibles) y FrozenXID
// (el menor xid que quedó sin congelar en lo recorrido, acotado por Horizon: el frozenXID que el store puede guardar
// con SetFrozenXID si Complete). FrozenXID y SetFrozenXID leen y suben el de la meta.
//
// # Llenado de las hojas (tarea 108)
//
// Dos cosas, sin cambio de formato ni estado en memoria, para que las hojas no queden a medio llenar:
//
//   - Partir en el punto de inserción (splitTarget y ascendingAt en split.go, como _bt_afternewitemoff de nbtree): si
//     las ascendingRun (4) versiones justo antes del punto de inserción se escribieron en ese orden (lsn creciente),
//     la inserción sigue una racha ascendente local y lo de la izquierda no vuelve a recibir inserciones: se busca
//     dejar a la izquierda exactamente lo anterior al punto, acotado al 90 % (una inserción al final de una hoja que
//     no es la de más a la derecha deja un 10 % a la derecha, como el usemult de _bt_findsplitloc) y solo si eso es
//     al menos la mitad (si no, mitad y mitad). Es la heurística más simple que encontramos que sirve con varios
//     grupos terminando en la misma hoja; «la anterior es la más nueva de la hoja» fallaba ahí (las inserciones de los
//     grupos se alternan). Con inserciones al azar la racha sale una vez cada 24 y además tiene que caer pasada la
//     mitad. Cada grupo deja su hoja de cola a medio llenar (recibe las inserciones) y a veces un resto del comienzo
//     del grupo siguiente (lo que quedó a la derecha del primer corte): pesa poco cuando cada grupo ocupa muchas hojas.
//   - Podar antes de partir (pruneLeaf en write.go, como la limpieza simple / bottom-up deletion de nbtree): una
//     escritura que no cabe, una vez por Write y antes de extender el archivo, quita de la hoja las versiones en línea
//     que ningún snapshot puede ver con el criterio de la limpieza (planLeaf con Config.Horizon = xact OldestActive,
//     sin congelar, sin marcar «liberándose» ni liberar desbordes, sin bits del mapa, sin heredar la versión de
//     referencia de la hoja izquierda ni mirar la hermana derecha: poda menos que la limpieza, nunca más) y, si quitó
//     algo, vuelve a buscar la posición y la última versión con la hoja todavía bloqueada. El registro es el mismo
//     KindLeafVacuum de la limpieza con solo la lista de quitar (xid 0): el WAL va antes que la página, la réplica
//     lo rehace byte a byte con applyLeafVacuum y no hace falta un Kind nuevo. Una hoja con page.FlagAllVisible no se
//     mira (no tiene nada muerto). No pone marcas de visibilidad (statusOf sin hints: la regla 49 no se toca) ni
//     recicla páginas (opGuard no cambia). Pruned cuenta lo quitado.
//     El horizonte se lee con la hoja suelta (writeLeaf la suelta, lee Config.Horizon y vuelve a bajar: una bajada de
//     más por hoja llena, nada por escritura): OldestActive toma el mutex de xact y recorre transacciones y snapshots,
//     y con la hoja más caliente bloqueada en exclusivo frenaba a todos los que la esperan. Un horizonte ya viejo al
//     podar solo poda menos (TestPruneHorizonUnlocked; tarea 129).
//   - Lado de la entrada nueva al partir en el punto de inserción: con grupos (v01|… y v02|…) el separador del corte
//     sale truncado («v02») y la entrada nueva, menor, cae a la izquierda y queda la última ahí, como «partir después
//     del ítem nuevo» de _bt_afternewitemoff; las siguientes del grupo también van a la izquierda hasta que se llena y
//     se parte al final al 90 %. No es una partición de más: exigir que la entrada nueva caiga a la derecha (si no,
//     mitad y mitad) bajó los índices de TestIndexHeaderShare del 83 al 65 % y TestSplitAscendingGroups del 86 al 55 %
//     con 5 grupos (tarea 129).
//
// # Borrado de páginas (tarea 70)
//
// Como _bt_pagedel de nbtree (delete.go). Vacuum, antes de recorrer, recicla (recycle) y termina los borrados a
// medias que quedaron a la izquierda de la primera hoja con enlace (finishLeftmost); durante el recorrido, una hoja
// medio muerta se termina (unlinkHalfDead) y una que queda vacía tras limpiarla (y liberar sus cadenas) se borra
// (deleteLeaf). VacuumStats cuenta HalfDead, Deleted y Recycled.
//
//  1. markHalfDead: con la hoja en compartido mira que se pueda borrar (deletableLeaf: hoja vacía, no la de más a la
//     derecha, no la raíz, sin partición incompleta, no medio muerta), la suelta y mira su hermana izquierda: si su
//     enlace derecho es la hoja y está partida a medias, la hoja todavía no tiene enlace en el padre
//     (_bt_leftsib_splitflag). Bloquea la hoja en exclusivo, lo comprueba otra vez, borra su bit del mapa si la página
//     tiene page.FlagAllVisible, mira en compartido que su hermana derecha no esté medio muerta
//     (_bt_rightsib_halfdeadflag) y busca el padre del subárbol (subtreeParent, como _bt_lock_subtree_parent):
//     findParent con la clave alta de la cima y before (llega a la página cuyo rango termina ahí, como pivotsearch);
//     si la cima es el único enlace de un padre que se puede borrar, el padre pasa a ser la cima y se sube; si es el
//     último de un padre con más hijos, no se borra; si no, el enlace siguiente tiene que ser su hermana derecha.
//     Una partición sin terminar en el camino (finishError) hace que no se borre en esta pasada.
//  2. unlinkHalfDead: repite unlinkStep hasta que la hoja queda borrada. Cada paso desengancha la cima actual (la que
//     anota la hoja, o la hoja): bloquea la hoja si la cima es otra, la hermana izquierda (desde la anotada, a la
//     derecha hasta la no borrada que enlaza a la cima), la cima, su hermana derecha y la meta, comprueba los
//     enlaces y el estado (un padre con un solo enlace; la hoja medio muerta y vacía) y escribe el KindUnlinkPage con
//     safeXID = VacuumOptions.NextXID() (o Horizon sin él).
//  3. recycle (al empezar cada pasada): si la página de arriba de la pila tiene safeXID < Horizon, espera a las
//     operaciones en vuelo (opGuard.drain, hasta 2 s; si no, recicla en otra pasada) y recicla de arriba hacia abajo
//     mientras se pueda (recycleOne: meta y página en exclusivo, KindRecyclePage).
//
// Por qué un lector nunca cae en una página reciclada: una página se recicla solo si (a) no hay operación suelta en
// vuelo que haya empezado antes (opGuard: dos contadores por época; Lookup, Exists, Write, cada Cursor.Next y
// VerifyWith entran y salen; drain cambia de época y espera a la anterior) y (b) todo snapshot abierto es posterior a
// su borrado (safeXID < Horizon): un Cursor entre dos Next no está en opGuard, pero en el motor siempre tiene un
// snapshot que retiene el horizonte. Un Cursor sin snapshot usado fuera del motor tiene que retener el horizonte de
// otra forma (las pruebas lo emulan). Mientras no se recicla, una página borrada conserva sus enlaces: quien llega a
// ella (por un enlace viejo del padre, de una hermana o de la raíz rápida) sigue a la derecha, donde está su espacio de
// claves.
//
// Raíz rápida: newRoot la pone en la raíz nueva; fastRootUp la sube al padre en el KindInsertPivot cuando la hija es
// la de más a la izquierda de su nivel y la raíz rápida está en su nivel o más abajo; el desenganche la baja a la
// hermana derecha si quedó sola en su nivel, o la vuelve a la raíz si la borrada era la raíz rápida. Verify comprueba
// que sea la de más a la izquierda de su nivel. La raíz nunca se borra ni baja de nivel: un árbol vaciado queda con la
// página de más a la derecha de cada nivel y la raíz rápida en la hoja.
//
// Verify (VerifyWith) reconoce un borrado a medias: la hoja medio muerta y cada interna con un solo enlace cuya cadena
// llega a ella con la cima al nivel de la interna o más arriba no cuentan para las cotas de claves de las siguientes
// ni para la clave de los enlaces (se busca la hermana izquierda que coincide saltándolas), y cada nivel tiene tantas
// páginas como enlaces arriba más particiones incompletas más cimas de borrados a medias; empieza cada nivel por su
// página de más a la izquierda (las cimas quedan a la izquierda del primer enlace); recorre la pila de borradas
// (VerifyStats.Deleted; cada una a Page) y cuenta las medio muertas (VerifyStats.HalfDead, LeafInfo.HalfDead).
// FinishDeletes termina los borrados a medias sin limpiar (Repair del motor).
//
// Pruebas (delete_test.go): TestPageDeleteRange (borrar un tramo, reciclar, cada bloque con dueño, volver a escribir
// sin crecer), TestPageDeleteAll (todo: una página por nivel y la raíz rápida en la hoja; volver a llenar la sube),
// TestPageDeleteRightmostChild, TestPageDeleteCut (corte antes de cada KindUnlinkPage en los tres modos de crashfs),
// TestPageDeleteCrashRandom (ventana deslizante con limpiezas, corte al azar), TestPageDeleteConcurrent (lectores y
// recorridos en los dos sentidos con -race mientras se borra y recicla), TestPageDeleteRedoRejects,
// TestReadDeletedPages; semillas nuevas en FuzzApplyDelta y FuzzReadNode.
//
// Toda página del árbol que se bloquea pasa por page.CheckItems la primera vez desde que la caché la cargó (la caché
// solo verifica la suma; buffer.ItemsChecker recuerda que ya pasó, y se borra cuando el búfer recibe otra página).
// Antes (hasta la tarea 134) se repetía en cada acceso: con las hojas de la forma 2, que tienen más entradas, era un
// tercio de un Lookup. Los cambios en memoria los hacen funciones que validan lo que tocan.
//
// Bloqueos: como nbtree, la partición, terminar una partición y la raíz nueva fijan páginas con otra bloqueada (la
// página nueva de fsm, la antigua hermana derecha, la meta, el padre); el borrado también (la hermana derecha y el
// padre del subárbol con la hoja; la hoja, la izquierda, la borrada, la derecha y la meta al desenganchar; la meta y
// la borrada al reciclar); siempre de izquierda a derecha, de hija a padre, árbol → meta, y nunca se espera a nada
// entre Reserve y Write, así que expulsar una página sucia (que espera al WAL) no puede cerrar un ciclo (es el orden
// de store/doc.go, «Orden de los bloqueos»). La espera del reciclado a las operaciones en vuelo se hace sin nada
// bloqueado.
//
// Réplica (tarea 46): Config.NoHints apaga las marcas de visibilidad de las lecturas (la réplica no escribe registros
// propios); DecodeLeafWrite lee de un KindLeafWrite de las dos formas la versión que escribe con el id entero (el
// cambio lógico que entrega el flujo de cambios del motor).
//
// # Migración perezosa desde el formato 1 (tarea 134)
//
// Un store del formato 1 tiene hojas de la forma 1 y su meta sin MetaLeavesForm2. Ninguna hoja de la forma 1 se
// cambia: una escritura que llega a una la reescribe antes en la forma 2 (KindLeafRewrite, con el prefijo de
// rewritePrefix; con prefijo vacío siempre cabe: cada entrada se achica 8 B y el ítem de prefijo mide 16), y una
// partición solo encuentra hojas de la forma 2. La limpieza de un store sin la marca no salta hojas por el mapa,
// reescribe cada hoja de la forma 1 que visita (VacuumStats.Rewritten) antes de limpiarla y, si la pasada termina sin
// error, anota MetaLeavesForm2 con un KindSetFrozen con flags (VacuumStats.LeavesForm2); desde ahí vuelve a saltar
// hojas. Las hojas medio muertas o borradas no se reescriben (terminan borradas). Las marcas de visibilidad y las
// lecturas no reescriben nada: una hoja de la forma 1 se lee igual. Verify cuenta las de la forma 1
// (VerifyStats.LeavesForm1, LeafInfo.Form1) y da daño si la meta dice MetaLeavesForm2 y queda alguna; LeavesForm2 lee
// la marca y RecordMetaFlags da las marcas de un KindSetFrozen (la réplica pone al día con eso su cuenta). En una hoja
// de la forma 2 Verify comprueba además que el prefijo deja lugar a algún id dentro de sus cotas (prefixInBounds; el
// límite de lo que ve, en store/doc.go «Formato 2: hojas de la forma 2 (b y c)»).
// Un KindLeafWrite de la forma 1 (de un WAL del formato 1) solo se rehace sobre una hoja de la forma 1; uno de la 2,
// sobre una de la 2.
package btree
