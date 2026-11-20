// Package fsm lleva el espacio libre de cada store y su mapa de visibilidad.
//
// Qué hace:
//
//   - Lista de páginas libres del fork principal, encadenada desde la página meta (campo freeHead de la meta, ver
//     btree) por el campo next del espacio especial de cada página libre (marca page.OpaqueFree). Allocate saca la
//     primera (KindFreePop) o, si la lista está vacía, extiende el archivo; Free la devuelve (KindFreePush). En la
//     fase 1 solo se liberan páginas de desborde (las hojas vacías esperan al borrado de páginas del árbol, fase 2).
//     Como cada store tiene sus archivos, no hay una lista global ni un punto caliente entre stores.
//   - Mapa de visibilidad (fork smgr.ForkVM, fase 1B/2, pregunta 44): VMBitsPerBlock = 2 bits por bloque del fork
//     principal —VMAllVisible (todas sus versiones visibles para todos) y VMAllFrozen (todas congeladas)—. Una
//     página del mapa es un encabezado de página (pd_special = page.Size) seguido del mapa desde el byte
//     page.HeaderSize: VMBlocksPerPage = 32.640 bloques por página. Cambiar una hoja borra sus bits (KindVMClear, un
//     registro propio con la hoja bloqueada y antes del de la hoja, solo si la hoja tiene page.FlagAllVisible); la
//     limpieza los pone (KindVMSet, después de su KindLeafVacuum, con la hoja bloqueada). Así en todo prefijo del WAL
//     un bit puesto implica una hoja sin cambios desde que la limpieza la vio (btree/doc.go, «Congelado y mapa de
//     visibilidad»). Solo las hojas llevan bits.
//
// Qué no hace: no busca espacio dentro de una hoja (eso es del árbol: pd_lower/pd_upper). Depende de page, smgr,
// buffer (Pool, Handle) y wal (Log) por sus interfaces. Manager (New) implementa FreeList y VisibilityMap; Redo
// rehace sus registros en la recuperación. Ninguna estructura en RAM crece con el número de páginas: todo vive en
// las páginas de la caché.
//
// # Lista de páginas libres
//
// La meta guarda freeHead en el byte page.HeaderSize + 24 (formato completo en btree/doc.go; fsm repite la posición,
// la magia y el bloque 0 porque no puede importar btree). Una página libre es una página de store vacía
// (page.Init con page.OpaqueSize) con espacio especial flags = page.OpaqueFree y next = la siguiente libre (0 = fin).
// La lista es una pila: Free apila y Allocate saca la cabeza. Una página sacada queda vacía con el espacio especial
// en cero, bloqueada en exclusivo para quien la pidió, que la inicia en su propio registro con BlockWillInit. Una
// página nueva (lista vacía) sale de buffer.Pool.PinNew en ceros, sin registro: si el corte llega antes del registro
// de quien la pidió, queda una página en ceros sin dueño (como en PostgreSQL), que no se pierde del archivo pero
// tampoco vuelve a la lista hasta que Repair del motor la devuelve (ver «Verificación y reparación»).
//
// # Mapa de visibilidad
//
// Los bits del bloque b están en la página b / VMBlocksPerPage del fork del mapa, byte page.HeaderSize +
// (b mod VMBlocksPerPage) / 4, desplazamiento 2 × (b mod 4): VMAllVisible el bit bajo, VMAllFrozen el alto. Una
// página del mapa se inicia con page.Init(0) y pd_lower = page.Size (no tiene hueco: si pd_lower quedara en
// page.HeaderSize, la imagen de página entera del WAL dejaría fuera el mapa). Una página en ceros (nueva) no tiene
// bits; la primera escritura en ella la inicia en el mismo registro. Set extiende el fork si no cubre el bloque
// (bajo un mutex propio, hoja); Get de un bloque fuera del mapa devuelve 0. Como en PostgreSQL, VMAllFrozen implica
// VMAllVisible (Set pone los dos) y poner bits que ya están o borrar los que no están no escribe registro.
//
// # Verificación y reparación (fase 3)
//
// Para Verify y Repair del motor (store/doc.go): WalkFree recorre la lista desde la meta sin cambiar nada (cada página
// libre y menor que el largo del fork, nunca más pasos que bloques; con buffer.Ring); ForEachVM entrega cada bloque
// con bits en el mapa (copiados de una página del mapa a la vez, ya soltada); Reclaim devuelve a la lista una página
// que nadie usa sea cual sea su contenido: la fija con PinZero (sin leerla, así también una ilegible) y la inicia con
// el mismo KindFreePush que Free (BlockWillInit), sin la comprobación de doble liberación (una página libre fuera de la
// lista ya tiene page.OpaqueFree). Free sigue siendo la de uso normal.
//
// # Registros del WAL (deltas, little-endian)
//
//	Kind          bloques                                     delta
//	KindFreePush  0: meta (ForkMain, bloque 0)                newHead(4) = la página liberada
//	              1: la página liberada, BlockWillInit         next(4) = la cabeza anterior
//	KindFreePop   0: meta                                     newHead(4) = next de la página sacada
//	              1: la página sacada, BlockWillInit           vacío (queda vacía, espacio especial en cero)
//	KindVMSet     0: página del mapa (ForkVM)                 block(4) del fork principal + bits(1) que se ponen
//	KindVMClear   0: página del mapa (ForkVM)                 block(4) + bits(1) que se borran (Clear: los dos)
//
// El xid del registro es el de quien pide o libera la página; los del mapa no llevan xid. La meta y la página del
// mapa llevan su imagen entera si su pd_lsn es anterior al punto de rehacer; la página libre o sacada nunca, porque
// el delta la inicia entera.
//
// # Orden de un cambio y bloqueos
//
// Cada cambio sigue store/doc.go: bloquear en exclusivo → armar el registro (con las imágenes) → wal.Log.Reserve
// (si devuelve ErrNeedImage, agregar la imagen y repetir) → aplicar → pd_lsn y MarkDirty → soltar → Write. Allocate
// escribe el registro con la página sacada todavía bloqueada (se la entrega así a quien la pidió) y la meta ya
// suelta.
//
// Orden entre páginas (a confirmar en store/doc.go, que todavía no habla de la meta): páginas del árbol → meta →
// página libre; página de datos → página del mapa. Por eso quien llama a Allocate puede tener hojas o internas
// bloqueadas pero nunca la meta, y quien llama a Free no puede tener bloqueadas ni la meta ni la página que libera.
// Allocate bloquea la meta durante todo el cambio: las altas y bajas de un store se serializan en su meta.
//
// # Rehacer
//
// Redo(pool, sm, rec) rehace un registro de fsm: valida su forma, extiende el fork si el archivo quedó más corto que
// el WAL y, por bloque, restaura la imagen sin condiciones y aplica el delta; con BlockWillInit inicia la página
// entera; sin ninguno de los dos, aplica solo si pd_lsn < lsn del registro. Es idempotente: rehacer dos veces deja
// las mismas páginas. Usa buffer.Pool.Pin, que lee y verifica la página: con una página rota por sectores (que el
// registro iba a pisar con su imagen o su BlockWillInit) Pin falla por la suma; hace falta en buffer un pin que no
// lea, como RBM_ZERO_AND_LOCK de PostgreSQL (a confirmar en buffer).
package fsm
