package metric

import (
	"math/bits"
	"sync/atomic"
	"time"
)

// numBuckets son las cubetas log2: una por largo en bits de una duración en ns (0 a 63; un int64 positivo cabe).
const numBuckets = 64

// Histogram cuenta duraciones en cubetas log2 (ver doc.go). El valor cero sirve; no se copia después de usarlo.
type Histogram struct {
	buckets [numBuckets]atomic.Int64
	count   atomic.Int64
	sum     atomic.Int64
	max     atomic.Int64
}

// Summary es una foto de un histograma.
type Summary struct {
	Count int64         // observaciones
	Mean  time.Duration // promedio (0 sin observaciones)
	P50   time.Duration // mediana, aproximada (límite superior de su cubeta log2)
	P99   time.Duration // percentil 99, aproximado igual
	Max   time.Duration // la mayor
}

/**
* Observe anota una duración (las negativas cuentan como 0). Solo atómicos.
* @param d time.Duration
**/
func (h *Histogram) Observe(d time.Duration) {
	ns := max(int64(d), 0)
	h.buckets[bits.Len64(uint64(ns))].Add(1)
	h.count.Add(1)
	h.sum.Add(ns)
	for {
		cur := h.max.Load()
		if ns <= cur || h.max.CompareAndSwap(cur, ns) {
			return
		}
	}
}

/**
* Since anota el tiempo pasado desde start.
* @param start time.Time
**/
func (h *Histogram) Since(start time.Time) {
	h.Observe(time.Since(start))
}

/**
* Summary da la cantidad, el promedio, p50, p99 y el máximo.
* @return Summary
**/
func (h *Histogram) Summary() Summary {
	var counts [numBuckets]int64
	var total int64
	for i := range h.buckets {
		counts[i] = h.buckets[i].Load()
		total += counts[i]
	}
	s := Summary{Count: total, Max: time.Duration(h.max.Load())}
	if total == 0 {
		return s
	}
	if n := h.count.Load(); n > 0 {
		s.Mean = time.Duration(h.sum.Load() / n)
	}
	s.P50 = quantile(&counts, total, 0.50, s.Max)
	s.P99 = quantile(&counts, total, 0.99, s.Max)
	return s
}

/**
* quantile devuelve el límite superior de la cubeta donde cae el percentil q, sin pasar del máximo visto.
* @param counts *[numBuckets]int64, total int64: la suma de counts, q float64: de 0 a 1, limit time.Duration
* @return time.Duration
**/
func quantile(counts *[numBuckets]int64, total int64, q float64, limit time.Duration) time.Duration {
	rank := int64(q*float64(total-1)) + 1 // la observación número rank, en orden (1 = la menor)
	var seen int64
	for i, c := range counts {
		seen += c
		if seen >= rank {
			return min(upper(i), limit)
		}
	}
	return limit
}

/**
* upper es el límite superior de la cubeta i: 2^i − 1 ns (la 0 es d = 0).
* @param i int
* @return time.Duration
**/
func upper(i int) time.Duration {
	if i >= 63 {
		return time.Duration(1<<63 - 1)
	}
	return time.Duration(uint64(1)<<uint(i) - 1)
}
