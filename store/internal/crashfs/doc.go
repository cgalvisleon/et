// Package crashfs es un sistema de archivos en memoria que simula cortes de energía, para las pruebas de modelo del
// motor (fase 1C: «0 escrituras confirmadas perdidas en 1.000 cortes simulados»).
//
// Qué hace: implementa vfs.FS guardando, por archivo y por directorio, lo escrito y lo durable por separado. Crash
// descarta lo que no tuvo fsync (Sync del archivo para los datos, SyncDir para altas, bajas y renombres) y puede
// romper escrituras a medias por sectores. Qué no hace: no es rápido ni sirve fuera de pruebas. Depende solo de vfs.
//
// Implementación: ola 1 (el mismo agente escribe la implementación real de vfs sobre el sistema operativo).
package crashfs
