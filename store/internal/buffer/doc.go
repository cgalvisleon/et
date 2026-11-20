// Package buffer es la caché de páginas del motor (shared_buffers de PostgreSQL): una sola por motor, de tamaño fijo
// (STORE_CACHE_SIZE, 128 MB por omisión), para todos sus stores. Es la única memoria del motor que crece con el uso,
// y no pasa de su tamaño.
//
// Qué hace: fijar páginas (pin: mientras está fijada no se expulsa), bloquear su contenido (compartido para leer,
// exclusivo para cambiar), reemplazo clock-sweep con contador de uso (tope 5, como PostgreSQL), anillo de 256 KB para
// recorridos grandes, verificar la suma al leer y calcularla al escribir (sobre una copia), escribir páginas sucias
// cumpliendo «WAL antes que la página» (WaitDurable hasta su pd_lsn), un escritor de fondo que mantiene páginas
// limpias (como bgwriter) y el vaciado repartido de un checkpoint. Qué no hace: no arma registros del WAL ni sabe qué
// hay en una página; no decide páginas enteras. Depende de page, smgr (Manager, FileNo, Fork) y wal (Flusher), por
// sus interfaces: se prueba con dobles.
//
// # Dónde vive la memoria
//
// Los búferes están en un solo arena reservado fuera del heap de Go, como la memoria compartida de PostgreSQL
// (decisión 62 de F-23): en darwin, linux y los BSD es un mmap anónimo y privado (arena_mmap.go); en el resto
// (windows) es un []uint64 del heap (arena_heap.go). Fuera del heap el GC no lo recorre ni lo cuenta para su meta, así
// el proceso no crece al doble de la caché: con 128 MB de caché, un motor ocupa unos 150 MB de RSS en vez de 280. El
// arena solo guarda bytes de páginas, nunca punteros de Go. Está alineado a la página del SO (los búferes, a 8). Las
// copias para escribir (scratch) siguen en el heap.
//
// Tocar el arena liberado (Munmap) es un SIGSEGV, no un error, así que se libera solo cuando nada puede tocarlo:
// toda lectura o escritura de d.data se hace con el búfer fijado (handle, flush, read, PinZero); toda fijada pasa por
// pinLocked, que con d.mu tomado rechaza la caché cerrada; Close marca closed, espera al escritor de fondo y llama a
// freeArena, que mira los pins de todos los búferes bajo d.mu y libera solo si están todos en 0 (tras closed los pins
// solo bajan). Si alguno sigue fijado (un Handle sin Release), lo libera el último unpin. Al liberar deja d.data en
// nil: un acceso tardío es un pánico de índice y el GC no ve punteros a un rango que el runtime podría reutilizar.
// Orden: arenaMu → d.mu. Un motor que no se cierra no devuelve su arena (no lo libera el GC).
//
// # Bloqueos
//
// El bloqueo de contenido es un RWMutex por búfer, corto. El mapa (Tag → búfer) se reparte en particiones, cada una
// con su lock, que es una hoja (no se toma otro con él tomado). Ver el orden completo en store/doc.go.
//
// # Marcas de visibilidad
//
// Un lector con bloqueo compartido puede poner marcas de visibilidad en una entrada de hoja con un OR atómico sobre
// su uint32 de marcas (los ítems están alineados a page.ItemAlign y los búferes a 8) y llamar a MarkDirtyHint. Por
// eso escribir una página al disco copia la página (bajo bloqueo compartido) y calcula la suma sobre la copia, nunca
// sobre el búfer. La copia se hace con cargas atómicas de 32 bits; quien lea esas marcas con bloqueo compartido
// también tiene que leerlas con atomic.LoadUint32, o -race verá una carrera con el OR de otro lector.
//
// # Implementación (ola 2)
//
// Open(Config) reserva los búferes en un solo arena fuera del heap (Buffers, o STORE_CACHE_SIZE con EnvBuffers) y
// arranca el escritor de fondo (BgWriterDelay; negativo = sin él); Close lo detiene y libera el arena. Cada búfer
// tiene un descriptor con tres locks, en este orden: io (serializa sus escrituras al disco: dos escritores nunca
// escriben el mismo búfer a la vez) → content (el bloqueo de contenido) → mu (encabezado: tag, estado, pins, uso,
// sucia; corto). Las NumPartitions particiones del mapa se toman solas: buscar en el mapa y fijar el búfer son dos
// pasos, y si el búfer cambió de página entre medio se repite.
//
// Cargar una página sigue BufferAlloc: elegir una víctima (fijada y limpia; si estaba sucia se escribe antes),
// publicar el tag nuevo en el mapa (si otro ya lo publicó, se usa el suyo), cambiar el tag bajo mu (si alguien fijó
// la víctima mientras tanto, se busca otra), quitar el tag viejo del mapa y leer fuera de todo lock; quien fija una
// página que se está cargando espera a que termine. Una lectura fallida (suma mala, bloque inexistente) devuelve el
// error con el tag y deja el búfer libre.
//
// Escribir un búfer (reemplazo, escritor de fondo, FlushAll) con io tomado: borrar justDirtied, copiarlo bajo
// bloqueo compartido, WaitDurable hasta el pd_lsn de la copia, suma sobre la copia (una página en ceros se escribe en
// ceros), smgr.Write y recién entonces marcarlo limpio, salvo que justDirtied se haya vuelto a poner (como
// BM_JUST_DIRTIED de PostgreSQL: MarkDirty o MarkDirtyHint durante o después de la copia, que la copia puede no
// tener). El búfer sigue sucio mientras dura la escritura: FlushAll lo anota y lo espera (io) en vez de saltearlo, así
// al volver toda página que estaba sucia está en el disco, también la que otro estaba escribiendo. Si se marcara
// limpio al empezar, un checkpoint que llega con una escritura en curso (esperando al WAL, por ejemplo) no la vería,
// haría el fsync antes de que llegue al disco y movería el punto de rehacer más allá de su cambio: tras un corte la
// página volvería a su versión vieja sin nada que rehacer. Un fallo de la escritura o de WaitDurable deja la caché
// inutilizable (ErrFailed con la causa) y no se escribe nada más.
//
// El escritor de fondo, cada BgWriterDelay, recorre una vuelta desde la manecilla y escribe hasta BgWriterMaxPages
// búferes sucios sin pins y con uso 0. FlushAll anota los sucios, los ordena por tag y escribe los que siguen con la
// misma página. El anillo es uno por caché, compartido por los recorridos con Ring (a confirmar). PinNew serializa
// las extensiones con un solo lock por caché (a confirmar). Reemplazar una página sucia espera al WAL: con el
// escritor de fondo es raro, pero quien llama a Pin no debería tener otra página bloqueada (ver «Orden de los
// bloqueos» en store/doc.go). No se llama a FlushAll ni a DropFile con un bloqueo de contenido tomado.
package buffer
