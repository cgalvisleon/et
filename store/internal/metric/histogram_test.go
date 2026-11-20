package metric

import (
	"sync"
	"testing"
	"time"
)

// TestHistogramQuantiles: p50 y p99 caen en la cubeta log2 correcta (error ≤ ×2) y nunca pasan del máximo.
func TestHistogramQuantiles(t *testing.T) {
	var h Histogram
	if s := h.Summary(); s != (Summary{}) {
		t.Fatalf("vacío: %+v", s)
	}
	for range 98 {
		h.Observe(100 * time.Microsecond)
	}
	h.Observe(10 * time.Millisecond)
	h.Observe(20 * time.Millisecond)
	s := h.Summary()
	if s.Count != 100 || s.Max != 20*time.Millisecond {
		t.Fatalf("cuenta o máximo: %+v", s)
	}
	if s.P50 < 100*time.Microsecond || s.P50 >= 200*time.Microsecond {
		t.Fatalf("p50 = %v, se esperaba en [100 µs, 200 µs)", s.P50)
	}
	if s.P99 < 10*time.Millisecond || s.P99 > 20*time.Millisecond {
		t.Fatalf("p99 = %v, se esperaba en [10 ms, 20 ms]", s.P99)
	}
	if s.Mean <= 100*time.Microsecond {
		t.Fatalf("promedio = %v", s.Mean)
	}

	var z Histogram
	z.Observe(0)
	z.Observe(-time.Second)
	if s := z.Summary(); s.Count != 2 || s.P99 != 0 || s.Max != 0 {
		t.Fatalf("ceros: %+v", s)
	}
}

// TestHistogramConcurrent: observar desde varias goroutines mientras otra toma fotos (con -race).
func TestHistogramConcurrent(t *testing.T) {
	var h Histogram
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 1000 {
				h.Observe(time.Duration(g*1000 + i))
			}
		}()
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 100 {
			_ = h.Summary()
		}
	}()
	wg.Wait()
	<-done
	if s := h.Summary(); s.Count != 8000 || s.Max != 7999 {
		t.Fatalf("%+v", s)
	}
}
