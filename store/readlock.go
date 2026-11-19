package store

import (
	"math/rand/v2"
	"runtime"
	"sync"
)

// cacheLine: el largo de una línea de caché en Apple M (128 bytes; en x86 son 64, así que también alcanza). Cada
// franja de readLock ocupa una propia para que dos núcleos que leen no se disputen la misma.
const cacheLine = 128

/**
* stripe: Una franja de readLock: un RWMutex con relleno hasta ocupar su propia línea de caché.
**/
type stripe struct {
	mu sync.RWMutex
	_  [cacheLine - 24]byte // 24 = el tamaño de sync.RWMutex
}

/**
* readLock: Un lock de lectores y escritores repartido en franjas. Un lector toma una sola franja, elegida al azar;
* un escritor las toma todas. Con un solo sync.RWMutex, cada lector suma y resta en el mismo contador: con muchos
* núcleos esa línea de caché salta de uno a otro y la lectura deja de escalar (en BenchmarkRead con 10 núcleos, el
* contador del RWMutex del índice era el 18% del tiempo). A cambio, escribir cuesta un Lock por franja.
**/
type readLock struct {
	stripes []stripe
}

/**
* newReadLock: Un readLock con una franja por procesador lógico (GOMAXPROCS al crearlo).
* @return *readLock
**/
func newReadLock() *readLock {
	return &readLock{stripes: make([]stripe, runtime.GOMAXPROCS(0))}
}

/**
* rlock: Toma una franja para leer.
* @return int (la franja: se devuelve en runlock)
**/
func (l *readLock) rlock() int {
	i := rand.IntN(len(l.stripes))
	l.stripes[i].mu.RLock()
	return i
}

/**
* runlock: Suelta la franja que dio rlock.
* @param i int
**/
func (l *readLock) runlock(i int) {
	l.stripes[i].mu.RUnlock()
}

/**
* lock: Toma todas las franjas para escribir, en orden (así dos escritores no se cruzan).
**/
func (l *readLock) lock() {
	for i := range l.stripes {
		l.stripes[i].mu.Lock()
	}
}

/**
* unlock: Suelta todas las franjas.
**/
func (l *readLock) unlock() {
	for i := range l.stripes {
		l.stripes[i].mu.Unlock()
	}
}
