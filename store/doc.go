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
// secuencia del log: crece con cada escritura y decide cuál de dos registros del mismo id es el más nuevo.
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
// # Invariantes
//
//   - Orden de los locks: writeMu → pubMu, writeMu → syncMu y syncMu → pubMu; nunca al revés. syncMu no toma writeMu
//     (lee el segmento activo de forma atómica). El lock del índice es una hoja: no se toma otro mientras se tiene.
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
// registro inválido es corrupción real y Open falla.
//
// Un archivo LOCK dentro del directorio, con un bloqueo exclusivo (flock), impide que dos procesos abran el mismo
// store para escribir.
//
// Open crea el directorio si no existe, con los intermedios que falten, y hace fsync del padre de cada uno que crea:
// el nombre de un directorio nuevo, como el de un archivo nuevo, solo es durable tras el fsync de su padre.
package store
