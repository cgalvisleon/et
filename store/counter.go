package store

import (
	"encoding/json"
	"sync/atomic"
)

/**
* counter: Valor numérico atómico (sin lock); se usa solo a través de sus métodos.
* Guarda los bits en un uint64: con T con signo, la suma en complemento a dos da el mismo
* resultado que en T.
**/
type counter[T ~int | ~int64 | ~uint64] struct {
	value atomic.Uint64
}

/**
* inc: Suma uno y retorna el nuevo valor.
* @return T
**/
func (s *counter[T]) inc() T {
	return T(s.value.Add(1))
}

/**
* dec: Resta uno y retorna el nuevo valor.
* @return T
**/
func (s *counter[T]) dec() T {
	return T(s.value.Add(^uint64(0)))
}

/**
* add: Suma n y retorna el nuevo valor.
* @param n T
* @return T
**/
func (s *counter[T]) add(n T) T {
	return T(s.value.Add(uint64(n)))
}

/**
* count: Retorna el valor actual.
* @return T
**/
func (s *counter[T]) count() T {
	return T(s.value.Load())
}

/**
* set: Reemplaza el valor.
* @param value T
**/
func (s *counter[T]) set(value T) {
	s.value.Store(uint64(value))
}

/**
* setMax: Reemplaza el valor solo si el nuevo es mayor.
* @param value T
* @return T
**/
func (s *counter[T]) setMax(value T) T {
	for {
		old := s.value.Load()
		if value <= T(old) {
			return T(old)
		}
		if s.value.CompareAndSwap(old, uint64(value)) {
			return value
		}
	}
}

/**
* MarshalJSON: Serializa el contador como su valor.
* @return []byte, error
**/
func (s *counter[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.count())
}
