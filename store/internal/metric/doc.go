// Package metric tiene las piezas de medición del motor: un histograma de latencias de tamaño fijo para el camino
// caliente (solo atómicos, sin candados, sin memoria nueva por observación).
//
// Qué hace: cuenta duraciones en cubetas log2 de nanosegundos y da la cantidad, la suma, el máximo y percentiles
// aproximados. Qué no hace: no guarda muestras ni ventanas de tiempo: los contadores son desde que se creó el
// histograma (como pg_stat_*: quien quiera tasas resta dos fotos). No depende de nada del motor.
//
// # Histograma (Histogram)
//
//	campo        tamaño    qué guarda
//	buckets      64 × 8 B  cubeta i: duraciones d con bits.Len64(d) = i, es decir 2^(i−1) ≤ d < 2^i ns (la 0: d = 0)
//	count        8 B       observaciones
//	sum          8 B       suma de las duraciones, en ns
//	max          8 B       la mayor duración vista, en ns
//
// 536 B por histograma, siempre: nada crece con el número de claves ni con el tiempo. Observe hace tres sumas
// atómicas y, si hace falta, un CompareAndSwap del máximo. Quantile recorre las 64 cubetas y devuelve el límite
// superior de la cubeta donde cae el percentil (acotado por el máximo): el error es como mucho ×2, suficiente para
// ver órdenes de magnitud (un fsync de 100 µs frente a uno de 10 ms). Una foto tomada mientras otros observan es
// aproximada: cada contador se lee por separado.
package metric
