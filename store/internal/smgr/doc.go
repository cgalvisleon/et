// Package smgr (storage manager) guarda las páginas de cada store en sus archivos de datos, como el md.c de
// PostgreSQL: cada store tiene un número de archivo (FileNo, como relfilenode) y uno o más forks; cada fork es una
// secuencia de bloques de page.Size repartida en archivos de BlocksPerFile bloques (1 GB).
//
// Nombres, relativos al directorio del motor:
//
//	data/<fileNo:%010d>.<n:%04d>      fork principal (árbol, desborde, lista de libres); n desde 0001
//	data/<fileNo:%010d>_vm.<n:%04d>   fork del mapa de visibilidad
//
// El archivo n guarda los bloques [(n-1)×BlocksPerFile, n×BlocksPerFile). Todos menos el último tienen exactamente
// 1 GB.
//
// Qué hace: crear, extender, leer, escribir, hacer fsync y borrar los archivos de un store; una caché de descriptores
// abiertos de tamaño fijo (con miles de stores no se abren todos); fsync del directorio data/ al crear un archivo.
// Qué no hace: no verifica sumas ni interpreta páginas (eso es de buffer y page), no sabe del WAL ni de la regla «WAL
// antes que la página» (la cumple buffer antes de llamar a Write). Depende de vfs y page.
//
// Implementación (ola 1): Open(fs, dir, maxFiles) devuelve el Manager. Por fork guarda en memoria si existe y cuántos
// bloques tiene (se lee de los archivos la primera vez; todos menos el último deben medir 1 GB, si no ErrShortFile).
// Los descriptores van en una caché LRU de maxFiles (DefaultMaxFiles = 256): expulsar un archivo con escrituras sin
// fsync le hace fsync antes de cerrarlo, y sigue marcado para el próximo Sync. Sync quita las marcas y después hace
// fsync; un fsync fallido deja el gestor inutilizable (todo devuelve ese error). Unlink borra del último archivo al
// primero. Un Extend que hace nacer un archivo hace fsync de los anteriores (ya llenos) antes de crearlo y de data/
// después: así un corte nunca deja un archivo corto en el medio del fork (lo encontró TestExtendNewFileCrash). Bloqueos: el fork (RWMutex: exclusivo para Create, Extend y Unlink) → el mutex del gestor y el de la caché,
// que son hojas.
package smgr
