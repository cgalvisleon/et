// Package crashfs es un sistema de archivos en memoria que simula cortes de energía, para las pruebas de modelo del
// motor (fase 1C: «0 escrituras confirmadas perdidas en 1.000 cortes simulados»).
//
// Qué hace: New() implementa vfs.FS guardando, por archivo y por directorio, lo escrito y lo durable por separado.
// Crash descarta lo que no tuvo fsync y, según el Mode, deja sobrevivir parte, entera o rota por sectores.
// FailAfter, FailSyncAfter y SetCapacity inyectan fallas de E/S y de disco lleno; SetDelay simula un disco lento.
// Qué no hace: no es rápido ni sirve
// fuera de pruebas. Depende solo de vfs.
//
// # Modelo
//
// Un archivo es un nodo (como un inode) con dos contenidos: el actual, que ven las lecturas, y el durable, el que
// tenía en su último Sync, más la lista ordenada de escrituras y truncados hechos desde entonces. Sync vuelve durable
// esa lista. En un corte el nodo queda en su contenido durable más las operaciones de la lista que el Mode deje
// sobrevivir, aplicadas en su orden: LoseUnsynced ninguna; ReorderUnsynced cada una entera o no; TornWrites cada
// sector de cada escritura por separado (los sectores se alinean a múltiplos del tamaño del sector en el archivo; un
// truncado queda entero o no). La probabilidad de sobrevivir se elige al azar en cada corte, de 0 a 1.
//
// Un directorio tiene sus entradas actuales (nombre → nodo), las durables y la lista ordenada de cambios desde el
// último SyncDir (altas, bajas y renombres; un renombre dentro de un directorio es un solo cambio, atómico). En un
// corte el directorio queda en sus entradas durables más un prefijo al azar de esa lista (ninguno con LoseUnsynced),
// como un sistema de archivos con diario, que guarda sus cambios de nombres en orden. El fsync de un archivo no hace
// durable su nombre: un archivo creado sin SyncDir puede desaparecer entero, y un renombre sin SyncDir, revertirse.
// Un renombre entre dos directorios son dos cambios, uno en cada uno, durables por separado.
//
// Los directorios se crean solo con MkdirAll, que hace fsync del padre: son durables enseguida (y el fsync del padre
// vuelve durables también los otros cambios pendientes de ese padre). No se borran ni se renombran.
//
// Un Sync que falla (FailAfter, FailSyncAfter) olvida las escrituras pendientes del archivo sin volverlas durables,
// aunque las lecturas las sigan viendo: como Linux, que tras un fsync fallido marca las páginas limpias y un fsync
// posterior devuelve éxito sin escribirlas. Por eso el motor no debe reintentar un fsync. Una escritura que falla no
// escribe nada.
//
// # Disco lleno y disco lento
//
// SetCapacity acota la suma de los largos de los archivos con nombre. Lo que la pasaría falla con ErrNoSpace (que es
// vfs.ErrNoSpace) sin escribir nada: una escritura o un Truncate que agrandan un archivo y la creación de un archivo
// con el disco ya lleno; sobrescribir lo que ya existe no pide lugar. Si la capacidad se baja por debajo de lo usado
// (un volumen fino o la asignación diferida, que prometieron lugar y no lo tenían), también falla con ErrNoSpace el
// Sync de un archivo con escrituras pendientes (las olvida, como todo Sync fallido) y el SyncDir de un directorio con
// cambios pendientes. Subir la capacidad (o quitarla con 0) vuelve a dejar escribir. Used da lo usado, para fijar un
// cupo relativo.
//
// SetDelay agrega una demora a cada escritura y a cada Sync, antes de tomar el mutex del disco: el resto de las
// operaciones sigue sin esperar.
//
// # Lo que no simula
//
// Escrituras a medias dentro de un sector; bits cambiados o sectores con basura (eso lo cubre el fuzzing de páginas y
// del WAL); una escritura que falla habiendo escrito una parte; que el largo de un archivo crezca sin sus datos
// (bloques en ceros tras el final viejo, salvo los huecos que dejen sectores perdidos entre otros que sí quedan);
// orden entre directorios distintos ni entre los datos y los nombres; permisos, enlaces, fechas.
package crashfs
