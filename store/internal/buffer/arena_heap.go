//go:build !(darwin || linux || freebsd || netbsd || openbsd || dragonfly)

package buffer

import "github.com/cgalvisleon/et/store/internal/page"

/**
* mapArena reserva las n páginas de la caché en el heap de Go (windows y los sistemas sin mmap anónimo): el GC la
* libera cuando nadie la apunta.
* @param n int: cantidad de páginas, > 0
* @return page.Page, error: siempre nil
**/
func mapArena(n int) (page.Page, error) {
	return heapPages(n), nil
}

/**
* unmapArena no hace nada: la memoria es del heap.
* @param mem page.Page
* @return error: siempre nil
**/
func unmapArena(mem page.Page) error {
	return nil
}
