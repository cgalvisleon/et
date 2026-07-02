package queue_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/celsiainternet/elvis/logs"
	"github.com/celsiainternet/elvis/queue"
)

/**
* Example_queue: Muestra el patrón "batching queue" clásico de Queue:
* cada item enviado con Push se agrupa en un arreglo y ese arreglo se
* despacha al handler cuando ocurre la primera de dos condiciones de
* disparo: se alcanza el tamaño máximo del lote (maxEvents) o vence el
* tiempo máximo de espera (period).
**/
func Example_queue() {
	var mu sync.Mutex
	var batches [][]int

	var wg sync.WaitGroup
	wg.Add(2) // un lote disparado por tamaño y otro disparado por timeout

	handler := func(ctx context.Context, batch []int) error {
		mu.Lock()
		logs.Debugf("batch: %v", batch)
		batches = append(batches, batch)
		mu.Unlock()
		wg.Done()
		return nil
	}

	q := queue.New(100, 3, 200*time.Millisecond, handler)

	// Condición 1 (batch size): 3 items alcanzan maxEvents=3 y
	// disparan el lote de inmediato, sin esperar el timeout.
	q.Push(1)
	q.Push(2)
	q.Push(3)

	// Condición 2 (timeout): solo 2 items, no alcanzan maxEvents, así
	// que el lote se despacha cuando vence period.
	time.Sleep(50 * time.Millisecond)
	q.Push(4)
	q.Push(5)

	wg.Wait()
	q.Close()

	mu.Lock()
	defer mu.Unlock()
	fmt.Println("lotes despachados:", len(batches))
	fmt.Println("lote por tamaño:", batches[0])
	fmt.Println("lote por timeout:", batches[1])
	// Output:
	// lotes despachados: 2
	// lote por tamaño: [1 2 3]
	// lote por timeout: [4 5]
}
