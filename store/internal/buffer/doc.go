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
// sobre el búfer.
package buffer
