//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package buffer

import (
	"fmt"
	"syscall"

	"github.com/cgalvisleon/et/store/internal/page"
)

/**
* mapArena reserva las n páginas de la caché fuera del heap de Go, con un mmap anónimo y privado (como la memoria
* compartida de PostgreSQL): el GC no la recorre ni la cuenta para su meta, así el proceso no llega al doble de la
* caché. El mmap está alineado a la página del SO, que cumple la alineación a 8 de los búferes. Solo guarda bytes de
* páginas, nunca punteros de Go.
* @param n int: cantidad de páginas, > 0
* @return page.Page: n×page.Size bytes en ceros, error
**/
func mapArena(n int) (page.Page, error) {
	mem, err := syscall.Mmap(-1, 0, n*page.Size, syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		return nil, fmt.Errorf("buffer: reservar %d páginas: %w", n, err)
	}
	return mem, nil
}

/**
* unmapArena devuelve la memoria al SO. Tocarla después es un SIGSEGV: solo se llama sin búferes fijados y con la
* caché cerrada (ver pool.freeArena).
* @param mem page.Page: lo que devolvió mapArena
* @return error
**/
func unmapArena(mem page.Page) error {
	if err := syscall.Munmap(mem); err != nil {
		return fmt.Errorf("buffer: liberar la caché: %w", err)
	}
	return nil
}
