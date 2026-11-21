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
// fsync le hace fsync antes de cerrarlo, y sigue marcado para el próximo Sync. Sync quita las marcas y toma los
// descriptores con el fork en compartido, y hace los fsync con el fork suelto (así Extend, Read y Write del fork no
// esperan el fsync del checkpoint; un Unlink en medio no cierra un descriptor en uso: lo cierra su último release); un
// fsync fallido deja el gestor inutilizable (todo devuelve ese error). Unlink borra del último archivo al
// primero. Un Extend que hace nacer un archivo hace fsync de los anteriores (ya llenos) antes de crearlo y de data/
// después: así un corte nunca deja un archivo corto en el medio del fork (lo encontró TestExtendNewFileCrash). Ese
// fsync va con el fork en exclusivo, una vez por GB de crecimiento: es la única espera a un fsync bajo el candado del
// fork. Create y Unlink no hacen fsync de data/ (tarea 98 de F-23): lo anotan y lo hace el próximo SyncAll (paso 2
// del checkpoint) o SyncDir (paso 3, con ddlMu: el unlink de una baja que llegó después del paso 2, hallazgo D1),
// antes de que el control pase su registro; un corte antes lo arregla el WAL (el rehacer crea el archivo que falte,
// ensureBlock de btree, y vuelve a borrar el que reaparezca). Un Extend más allá de NBlocks escribe en ceros los
// bloques intermedios (writeZeros, de a 128 KB), nunca un hueco con Truncate ni escribiendo más allá del final: ext4 y
// XFS no reservan el lugar de un hueco, y la página sucia que se escribiera después ahí daría ENOSPC en el
// desalojo o el checkpoint, donde ya no hay vuelta atrás (tarea 107 de F-23; lo encontró diskfull.sh en la VM).
// Bloqueos: el fork (RWMutex: exclusivo para Create, Extend y Unlink) → el mutex del gestor y el de la caché, que son
// hojas.
package smgr
