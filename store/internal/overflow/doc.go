// Package overflow guarda los valores que no caben en línea en una hoja en cadenas de páginas de desborde, en la
// línea de TOAST (pregunta 22), en el fork principal del mismo store.
//
// # Página de desborde
//
// Encabezado de página normal; el trozo del valor va desde page.HeaderSize hasta pd_lower (pd_lower = 32 + largo del
// trozo; pd_upper = pd_special = page.Size - page.OpaqueSize); sin punteros a ítems. Espacio especial común con flags =
// page.OpaqueOverflow (solo esa marca), next = la siguiente página de la cadena (0 en la última), level = la posición
// de la página en la cadena (desde 0) y prev en 0; pd_prune_xid lleva el xid de la versión dueña (el del registro): con
// él FreeTail sabe qué páginas siguen siendo de la versión. Cada página guarda ChunkSize = 8.144 bytes salvo la última,
// que guarda el resto. La entrada de hoja guarda el largo total y el primer bloque (Ref): leer es seguir la cadena
// hasta juntar el largo.
//
// Como el largo total fija cuántas páginas tiene la cadena y cuánto guarda cada una, Read valida cada página antes
// de copiarla: que sea de desborde, que su level sea su posición, que su trozo mida lo que corresponde y que solo la
// última tenga next = 0. Un corte, un bucle, un salto a otra cadena o un largo que miente dan ErrBadChain sin
// visitar más páginas de las que tiene el valor y sin pánico.
//
// # Registro del WAL (little-endian)
//
//	Kind               bloques                                    delta              datos principales
//	KindOverflowWrite  0: la página de desborde (ForkMain),       el trozo (1 a      next(4) seq(4)
//	                      BlockWillInit                           ChunkSize bytes)
//
// Nunca lleva imagen: el registro inicia la página entera. Mide 32 + 16 + 8.144 + 8 = 8.200 B como mucho. El xid es
// el de la transacción de la versión. Un decodificador junta el valor por los registros: cada uno trae su trozo, su
// posición y la siguiente página.
//
// # Escribir, leer y liberar
//
// Write escribe la cadena de la última página a la primera: así cada página ya conoce a su siguiente cuando se
// escribe y nunca hay dos páginas bloqueadas a la vez (no se pide una página con otra bloqueada, store/doc.go). Por
// cada trozo: fsm.Allocate (la página llega bloqueada en exclusivo) → wal.Log.Reserve → aplicar, pd_lsn, MarkDirty
// → soltar → Write de la reserva. Si algo falla a medias, devuelve a la lista lo ya escrito. Read fija una página a
// la vez con bloqueo compartido. Free recorre la cadena entera validándola y solo entonces libera (fsm.Free) de la
// primera a la última: una cadena dañada no libera nada, así una página ajena nunca vuelve a la lista. Free es para
// una cadena que nadie apunta (un Write cuya entrada de hoja no se escribió).
//
// FreeTail(file, ref, owner, xid) es la liberación de la limpieza (pregunta 58): junta desde la primera las páginas
// que siguen siendo de la versión —de desborde, en su posición, con owner en pd_prune_xid y con su trozo— y las libera
// de la última a la primera, cada una con su KindFreePush. Un corte a mitad deja liberada una cola y entero el
// prefijo: repetirla libera solo el prefijo, y sin nada que liberar no hace nada. Una página liberada que ya tomó otra
// versión tiene otro xid (los xid no se repiten, y una versión muerta es de una transacción terminada), así que nunca
// se libera dos veces. owner 0 se rechaza (ErrBadRef). Mientras tanto la versión queda marcada «liberándose» en su
// hoja (btree, KindLeafVacuum), y se quita al terminar.
//
// Rehacer (Redo): valida la forma, extiende el fork si el archivo quedó corto y pisa la página con PinZero (no se
// lee del disco, que puede estar rota por un corte). Es idempotente.
//
// # Ciclo de vida
//
// Cada cadena pertenece a exactamente una versión: se escribe antes de la entrada de hoja que la apunta, con la
// misma transacción (un KindOverflowWrite con BlockWillInit por página: el delta es el trozo, y es también el cambio
// lógico que junta un decodificador), y la limpieza la libera (FreeTail, desde el final) al quitar esa versión, también
// si su transacción abortó. No hay reutilización entre versiones: Update con el mismo valor no escribe. Sin
// compresión (fase posterior).
//
// A confirmar: un corte a mitad de Free (no de FreeTail) deja liberadas las primeras páginas y la cadena rota; volver
// a llamar a Free devuelve ErrBadChain y las páginas restantes quedan huérfanas, como las de un Write cortado a medias
// (páginas sacadas de la lista sin entrada que las apunte). Verify del motor las encuentra (páginas de desborde sin
// versión) y Repair las devuelve a la lista, como las huérfanas de fsm. La limpieza usa FreeTail, que no tiene ese
// problema.
//
// Pages (fase 3, Verify del motor) recorre las páginas de la cadena de una versión sin copiar el valor: cada una de
// desborde, del xid de la versión (pd_prune_xid), en su posición y con su trozo, con buffer.Ring; con partial (una
// versión «liberándose») termina sin error en la primera que ya no es de la versión.
//
// Qué no hace: no decide cuándo un valor va a desborde (eso lo dice btree.MaxEntrySize), ni la visibilidad. Depende
// de page, smgr, buffer, fsm y wal por sus interfaces. Manager (New) implementa Chains; ninguna estructura en RAM
// crece con el tamaño de la base (un valor sí ocupa su largo al leerlo: se devuelve entero).
//
// DecodeWrite lee de un KindOverflowWrite su bloque, el siguiente, la posición y el trozo: con ellos el flujo lógico
// del motor (tarea 46) arma el valor de una versión en desborde sin leer páginas.
package overflow
