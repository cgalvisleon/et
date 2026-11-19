// Package store guarda pares (id string, data []byte) en disco, como base para construir persistencia de datos de
// alto rendimiento (F-23 de la Mesa de trabajo).
//
// # Diseño
//
// Un store es un directorio con varios archivos de segmento («0000000001.seg», «0000000002.seg»…). Cada escritura se
// anexa al final del segmento activo; cuando el activo llega a RELSEG_SIZE MB (128 por omisión) se sella —espera a que
// terminen sus escrituras en curso y hace fsync— y se crea el siguiente. Los segmentos sellados no cambian.
//
// En memoria hay un índice ordenado por id (un árbol B) que dice, para cada id, en qué segmento y posición está su
// último registro. Al abrir, el índice se rehace recorriendo los segmentos.
//
// # Formato de un registro
//
//	[crc:4][lsn:8][kind:1][idLen:2][dataLen:4][id][data]
//
// Enteros en big-endian. crc es el CRC-32C (código de verificación de errores, polinomio Castagnoli) de todo lo que
// le sigue, así que un registro a medio escribir o una zona en ceros no pasa la verificación. lsn es el número de
// secuencia del log: crece con cada escritura y decide cuál de dos registros del mismo id es el más nuevo. kind es
// 1 (guardar: data es el valor) o 2 (borrar: dataLen 0). Dentro del log, el orden de los registros es el de su lsn:
// el lsn y la posición se reservan juntos bajo writeMu, y un segmento se sella antes de crear el siguiente.
//
// # Operaciones
//
//   - Put y Remove son ciegos: no miran lo que había; Remove anexa un registro de borrado aunque el id no exista.
//   - Insert (solo si no existe), Update (solo si existe y el valor cambió) y Delete (solo si existe) son
//     condicionales y devuelven si escribieron.
//   - Read va por el índice al registro (pread, sin tomar writeMu), verifica crc, lsn e id, y devuelve el valor con
//     una sola asignación: el registro leído, del que el valor es una vista. Ve lo último que terminó de escribirse
//     (el índice recibe la ref después del WriteAt), aunque todavía no esté publicado ni en disco.
//
// # Escritura concurrente
//
// Varios escritores trabajan a la vez (propuesta P-4):
//
//  1. Reservar: bajo writeMu, cada escritura toma su lsn y su lugar (segmento y posición) en el log. Es lo único
//     serializado, y es corto.
//  2. Escribir: fuera del lock, cada una escribe su registro en su lugar (WriteAt); las escrituras van en paralelo y
//     pueden terminar en cualquier orden.
//  3. Publicar: al terminar, la escritura actualiza el índice y avisa su lsn. published es el mayor lsn tal que todos
//     los anteriores ya están escritos: avanza solo de forma contigua.
//  4. Confirmar: con SYNC_ON_WRITE (true por omisión), Put espera a que su lsn esté en disco. Un solo fsync confirma a
//     todas las escrituras publicadas hasta ese momento (group commit: varias escrituras comparten un fsync).
//
// # Escrituras condicionales
//
// El índice se actualiza después del WriteAt, fuera de writeMu: mirar solo el índice no basta, porque un Put del
// mismo id puede estar reservado y sin terminar. Por eso cada escritura, ciega o no, se anota al reservar en un
// registro de pendientes por id (pending.go): cuántas escrituras de ese id faltan por terminar y qué hace la última.
// El estado lógico de un id es el de su última escritura reservada: la pendiente si la hay, si no lo que dice el
// índice. Ese estado solo cambia al reservar, y reservar es bajo writeMu; así, comprobar la condición y reservar
// en la misma sección de writeMu es atómico frente a cualquier otro escritor del mismo id (de varios Insert del
// mismo id a la vez, exactamente uno gana). Cuando una escritura termina, pasa su ref al índice y deja de estar
// pendiente en un solo paso, bajo el lock de pending.
//
// Update necesita comparar con el valor actual, y leer del disco con writeMu tomado pararía a todos los escritores.
// Lo lee fuera del lock (si el id tiene escrituras pendientes, espera a que se publiquen) y luego reserva con la
// condición «el id sigue en el lsn que leí»; si otro lo escribió en medio, vuelve a leer (concurrencia optimista).
//
// # Borrados
//
// Un borrado deja en el índice una lápida (la ref con deleted y el lsn del borrado) mientras queden escrituras
// pendientes del mismo id: así, un Put más viejo (menor lsn) que termina después del borrado no revive el id, porque
// en el índice gana el mayor lsn. Cuando termina la última escritura pendiente del id, la lápida ya no hace falta
// (ninguna escritura vieja puede llegar) y se quita: en reposo el índice no tiene lápidas. Al recuperar, los
// segmentos se recorren en orden de lsn y un borrado simplemente quita el id del índice.
//
// # Invariantes
//
//   - Orden de los locks: writeMu → pending.mu → índice; writeMu → pubMu, writeMu → syncMu y syncMu → pubMu; nunca
//     al revés. Al terminar una escritura se toma pending.mu → índice sin writeMu. syncMu no toma writeMu (lee el
//     segmento activo de forma atómica). El lock del índice es una hoja: no se toma otro mientras se tiene.
//     pending.mu nunca se tiene durante una rotación: sellar espera a escrituras que necesitan pending.mu para
//     terminar.
//   - Read no toma writeMu: la lista de segmentos, el activo y el cierre se leen de forma atómica (rotate publica
//     una lista nueva en vez de cambiar la que hay).
//   - durable ≤ published: nunca se confirma un lsn con escrituras anteriores sin terminar. Por eso, tras una caída,
//     lo que queda después del primer hueco del segmento activo nunca se confirmó y se puede descartar.
//   - Un segmento se sella (todas sus escrituras terminadas y fsync hecho) antes de crear el siguiente, así que al
//     confirmar basta con hacer fsync del segmento activo.
//   - En el índice gana el registro de mayor lsn: dos escritores del mismo id pueden publicar en cualquier orden.
//   - Si una escritura falla, su lugar queda como un hueco que published no puede saltar: el store queda en error y
//     no acepta más escrituras (cerrarlo y abrirlo de nuevo lo recupera).
//
// # Recuperación
//
// Al abrir se recorre cada segmento verificando el crc de cada registro. En el último segmento, el primer registro
// inválido marca el final: el archivo se corta ahí (lo que seguía no se había confirmado). En un segmento sellado, un
// registro inválido es corrupción real y Open falla. Los registros de borrado se aplican igual que los de guardar.
//
// Un archivo LOCK dentro del directorio, con un bloqueo exclusivo (flock), impide que dos procesos abran el mismo
// store para escribir.
//
// Open crea el directorio si no existe, con los intermedios que falten, y hace fsync del padre de cada uno que crea:
// el nombre de un directorio nuevo, como el de un archivo nuevo, solo es durable tras el fsync de su padre.
package store
