package queue

import (
	"context"
	"sync"
	"time"

	"github.com/celsiainternet/elvis/logs"
)

/**
* Queue: Cola genérica que implementa el patrón clásico de "batching
* queue": agrupa cada item recibido por Push en un arreglo y despacha
* ese arreglo al handler cuando ocurre la primera de dos condiciones
* de disparo: se alcanza maxEvents elementos (batch size) o transcurre
* period sin completar el lote (timeout).
**/
type Queue[T any] struct {
	queue     chan T
	handler   func(context.Context, []T) error
	maxEvents int
	period    time.Duration
	mu        sync.RWMutex
	closed    bool
	done      chan struct{}
}

/**
* New: Crea una cola genérica que agrupa items en lotes y los despacha
* a handler cuando se reúnen maxEvents elementos (batch size) o
* transcurre period (timeout), lo que ocurra primero.
* @param queueSize int, tamaño máximo de la cola
* @param maxEvents int, cantidad máxima de eventos por lote
* @param period time.Duration, tiempo máximo de espera por lote
* @param handler func(context.Context, []T) error
* @return *Queue[T]
**/
func New[T any](
	queueSize int,
	maxEvents int,
	period time.Duration,
	handler func(context.Context, []T) error,
) *Queue[T] {
	if maxEvents <= 0 {
		maxEvents = 1
	}

	if period <= 0 {
		period = time.Second
	}

	q := &Queue[T]{
		queue:     make(chan T, queueSize),
		handler:   handler,
		maxEvents: maxEvents,
		period:    period,
		done:      make(chan struct{}),
	}

	go q.worker()

	return q
}

/**
* worker: Acumula items en un arreglo (buffer) y despacha el lote al
* handler tan pronto se cumple alguna de las dos condiciones de
* disparo: el arreglo alcanza maxEvents elementos, o vence el timer de
* period sin haberse llenado. El timer se reinicia después de cada
* despacho, de modo que siempre mide el tiempo desde el último lote.
**/
func (q *Queue[T]) worker() {
	defer close(q.done)

	ctx := context.Background()

	buffer := make([]T, 0, q.maxEvents)

	timer := time.NewTimer(q.period)
	defer timer.Stop()

	dispatch := func(batch []T) {
		if err := q.handler(ctx, batch); err != nil {
			logs.Alert(err)
		}
	}

	resetTimer := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(q.period)
	}

	for {
		select {
		case item, ok := <-q.queue:
			if !ok {
				if len(buffer) > 0 {
					dispatch(buffer)
				}
				return
			}

			buffer = append(buffer, item)

			// Condición de disparo 1: tamaño de lote alcanzado.
			if len(buffer) >= q.maxEvents {
				batch := buffer
				buffer = make([]T, 0, q.maxEvents)
				resetTimer()
				dispatch(batch)
			}
		case <-timer.C:
			// Condición de disparo 2: venció el tiempo de espera.
			if len(buffer) > 0 {
				batch := buffer
				buffer = make([]T, 0, q.maxEvents)
				dispatch(batch)
			}
			timer.Reset(q.period)
		}
	}
}

/**
* Push: Encola un item para ser agrupado y procesado. No hace nada si
* la cola ya fue cerrada.
* @param item T
**/
func (q *Queue[T]) Push(item T) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		return
	}

	q.queue <- item
}

/**
* Len: Retorna la cantidad de items pendientes en la cola.
* @return int
**/
func (q *Queue[T]) Len() int {
	return len(q.queue)
}

/**
* Close: Cierra la cola y espera a que el worker despache el lote
* pendiente antes de retornar. Los llamados a Push posteriores a Close
* no tienen efecto.
**/
func (q *Queue[T]) Close() {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return
	}

	q.closed = true
	close(q.queue)
	q.mu.Unlock()

	<-q.done
}
