// Package page es el formato de la página de 8 KB, tomado de bufpage.h de PostgreSQL: encabezado, punteros a ítems
// que crecen hacia abajo, ítems que crecen hacia arriba desde el final y un espacio especial al final de la página.
//
// Qué hace: define los tipos base que viajan en las páginas (LSN, XID, BlockNumber), las constantes del formato, la
// suma de verificación y (ola 1) las funciones que leen y cambian una página en un []byte: iniciarla, agregar,
// quitar y compactar ítems, leer el espacio especial, calcular y verificar la suma. Qué no hace: no sabe qué hay
// dentro de un ítem (eso es de btree y overflow), ni de archivos, caché o WAL. No depende de ningún otro paquete.
//
// # Encabezado (HeaderSize = 32 bytes, little-endian)
//
//	off tam campo
//	  0   8 pd_lsn              lsn del último registro del WAL que cambió la página
//	  8   2 pd_checksum         suma de verificación (ver abajo)
//	 10   2 pd_flags            FlagHasFreeLines, FlagPageFull, FlagAllVisible
//	 12   2 pd_lower            fin del arreglo de punteros = inicio del espacio libre
//	 14   2 pd_upper            fin del espacio libre = inicio de los ítems
//	 16   2 pd_special          inicio del espacio especial (Size si la página no tiene)
//	 18   2 pd_pagesize_version Size | LayoutVersion
//	 20   8 pd_prune_xid        el xid más viejo que una limpieza podría quitar de la página, 0 si ninguno
//	 28   4 reservado           en cero
//
// pd_prune_xid es de 64 bits (en PostgreSQL 32), como todos los xid del motor.
//
// # Punteros a ítems (ItemIDSize = 4 bytes)
//
// Desde el byte 32 hasta pd_lower, uno por ítem, numerados desde 1 (OffsetNumber). Cada uno es un uint32
// little-endian con lp_off en los bits 0–14 (posición del ítem desde el inicio de la página), lp_flags en los bits
// 15–16 (LineUnused, LineNormal, LineRedirect, LineDead) y lp_len en los bits 17–31 (largo del ítem). Un ítem se puede
// mover dentro de la página (compactar) sin cambiar su número. Los ítems empiezan en posiciones múltiplo de ItemAlign:
// así las marcas de visibilidad de una entrada de hoja (un uint32 en su byte 0) se pueden poner con un OR atómico.
//
// # Espacio especial (OpaqueSize = 16 bytes)
//
// Todas las páginas de un store (meta, árbol, desborde, libre) llevan el mismo espacio especial al final, al estilo
// de BTPageOpaqueData:
//
//	off tam campo
//	  0   4 prev     hermana de la izquierda (árbol); 0 = ninguna (el bloque 0 es la página meta)
//	  4   4 next     hermana de la derecha (árbol), siguiente de la cadena (desborde) o de la lista de libres; 0 = ninguna
//	  8   4 level    nivel en el árbol, 0 = hoja
//	 12   2 flags    OpaqueLeaf, OpaqueRoot, OpaqueMeta, OpaqueOverflow, OpaqueFree, OpaqueIncompleteSplit…
//	 14   2 cycleID  ciclo de limpieza de la última partición (como btpo_cycleid)
//
// Las páginas del mapa de visibilidad (fork vm) no tienen espacio especial: pd_special = Size.
//
// # Suma de verificación
//
// CRC-32C (Castagnoli) de la página con pd_checksum en cero, seguido del número de bloque en 4 bytes little-endian
// (una página escrita en otro lugar no pasa), plegado a 16 bits: uint16(c) ^ uint16(c>>16). Se calcula al escribir la
// página al disco, sobre una copia, y se verifica al leerla. Una página toda en ceros es válida y nueva (como
// PageIsNew): el archivo pudo extenderse sin que la página llegara a escribirse.
package page
