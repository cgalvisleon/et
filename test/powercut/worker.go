package main

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/store"
)

// keysPerStore es cuántas claves distintas usa cada escritor (datos chicos: el perfil "big" usa claves nuevas).
const keysPerStore = 400

// worker es un escritor: tiene su store y su secuencia de operaciones, así el orden de sus operaciones en el WAL es
// el orden en que las hace.
type worker struct {
	g       int
	seed    uint64
	profile string
	e       *store.Engine
	s       *store.Store
	r       *rand.Rand
	j       *journal
	seq     int64
	pending int64 // última operación diferida sin WaitDurable; 0 si ninguna
	next    int   // próxima clave nueva del perfil "big"
	err     error // el error que lo detuvo, si alguno
	ops     int64 // operaciones con efecto
}

/**
* newWorker crea el escritor g y abre (o crea) su store.
* @param e *store.Engine
* @param j *journal
* @param seed uint64
* @param profile string
* @param g int
* @return *worker, error
**/
func newWorker(e *store.Engine, j *journal, seed uint64, profile string, g int) (*worker, error) {
	w := &worker{g: g, seed: seed, profile: profile, e: e, j: j, r: rand.New(rand.NewPCG(seed, uint64(g)+1))}
	return w, w.reopen(e)
}

/**
* reopen toma el store del escritor en el motor e (tras reabrir el motor).
* @param e *store.Engine
* @return error
**/
func (w *worker) reopen(e *store.Engine) error {
	s, err := e.Store(storeName(w.g))
	if err != nil {
		return err
	}
	w.e, w.s = e, s
	return nil
}

/**
* noEffect dice si el error es de una operación rechazada sin tocar el motor (se puede reintentar).
* @param err error
* @return bool
**/
func noEffect(err error) bool {
	if errors.Is(err, store.ErrFailed) {
		return false
	}
	return errors.Is(err, store.ErrBusy) || errors.Is(err, store.ErrThrottled) || errors.Is(err, store.ErrConflict) ||
		errors.Is(err, store.ErrDiskFull) || errors.Is(err, store.ErrStorePending)
}

/**
* key elige una clave: del conjunto fijo, o una nueva en el perfil "big" para que los datos crezcan.
* @return string
**/
func (w *worker) key() string {
	if w.profile == "big" && w.r.IntN(10) != 0 {
		w.next++
		return fmt.Sprintf("n%07d", w.next)
	}
	return fmt.Sprintf("k%04d", w.r.IntN(keysPerStore))
}

/**
* run hace operaciones hasta que stop se cierre, haga max (0 = sin tope) o una operación falle con efecto incierto.
* @param stop <-chan struct{}
* @param max int64
* @return error: el que lo detuvo (nil si paró por stop o max)
**/
func (w *worker) run(stop <-chan struct{}, max int64) error {
	for done := int64(0); max == 0 || done < max; done++ {
		select {
		case <-stop:
			return nil
		default:
		}
		if err := w.step(); err != nil {
			w.err = err
			return err
		}
	}
	return nil
}

/**
* step elige y hace una operación al azar.
* @return error: solo si la operación falló con efecto incierto (el escritor debe parar)
**/
func (w *worker) step() error {
	n := w.r.IntN(100)
	switch {
	case n < 33:
		return w.single('P', w.key())
	case n < 42:
		return w.single('D', fmt.Sprintf("k%04d", w.r.IntN(keysPerStore)))
	case n < 60:
		return w.batch(false)
	case n < 74:
		return w.single('p', w.key())
	case n < 79:
		return w.single('d', fmt.Sprintf("k%04d", w.r.IntN(keysPerStore)))
	case n < 87:
		return w.batch(true)
	case n < 97:
		return w.waitDurable()
	case n < 98 && w.g == 0 && w.profile != "big":
		return w.ddl('E')
	case n < 99 && w.g == 0 && w.profile != "big":
		return w.ddl('X')
	}
	return w.single('P', w.key())
}

/**
* begin anota el intento de una operación y le da su número.
* @param kind byte
* @param args []string
* @return int64
**/
func (w *worker) begin(kind byte, args []string) int64 {
	w.seq++
	if len(args) == 0 {
		w.j.line("I %d %d %c", w.g, w.seq, kind)
	} else {
		w.j.line("I %d %d %c %s", w.g, w.seq, kind, strings.Join(args, " "))
	}
	return w.seq
}

/**
* finish anota el resultado: confirmada (durable), diferida (queda pendiente de WaitDurable) o sin efecto. Una
* operación que no escribió (Delete de un id que no existe, lote sin escrituras) no escribe en el WAL ni espera el
* fsync: se anota sin efecto, porque su vuelta no hace durable lo diferido anterior.
* @param seq int64
* @param deferred bool
* @param wrote bool: la operación escribió algo
* @param err error
* @return error: err si el efecto es incierto
**/
func (w *worker) finish(seq int64, deferred, wrote bool, err error) error {
	switch {
	case err == nil && !wrote:
		w.j.line("F %d %d", w.g, seq)
	case err == nil && deferred:
		w.pending = seq
		w.ops++
	case err == nil:
		w.j.line("C %d %d", w.g, seq)
		w.pending = 0
		w.ops++
	case noEffect(err):
		w.j.line("F %d %d", w.g, seq)
		return nil
	default:
		return fmt.Errorf("escritor %d, op %d: %w", w.g, seq, err)
	}
	return nil
}

/**
* single hace un Put o Delete suelto, durable (P, D) o diferido (p, d).
* @param kind byte
* @param k string
* @return error
**/
func (w *worker) single(kind byte, k string) error {
	seq := w.begin(kind, []string{k})
	var err error
	wrote := true
	switch kind {
	case 'P':
		err = w.s.Put(k, value(w.seed, w.profile, w.g, seq, k))
	case 'p':
		err = w.s.Deferred().Put(k, value(w.seed, w.profile, w.g, seq, k))
	case 'D':
		wrote, err = w.s.Delete(k)
	case 'd':
		wrote, err = w.s.Deferred().Delete(k)
	}
	return w.finish(seq, kind == 'p' || kind == 'd', wrote || err != nil, err)
}

/**
* batch hace un lote de 2 a 8 claves distintas (PutIn o DeleteIn) y lo confirma con Commit o CommitDeferred.
* @param deferred bool
* @return error
**/
func (w *worker) batch(deferred bool) error {
	n := 2 + w.r.IntN(7)
	seen := map[string]bool{}
	var args []string
	for len(args) < n {
		k := w.key()
		if seen[k] {
			continue
		}
		seen[k] = true
		if w.r.IntN(4) == 0 {
			args = append(args, "-"+k)
		} else {
			args = append(args, "+"+k)
		}
	}
	kind := byte('B')
	if deferred {
		kind = 'b'
	}
	seq := w.begin(kind, args)
	tx := w.e.NewBatch()
	wrote := false
	for _, a := range args {
		var err error
		found := true
		if a[0] == '+' {
			err = w.s.PutIn(tx, a[1:], value(w.seed, w.profile, w.g, seq, a[1:]))
		} else {
			found, err = w.s.DeleteIn(tx, a[1:])
		}
		wrote = wrote || found
		if err != nil {
			tx.Rollback()
			return w.finish(seq, deferred, true, err)
		}
	}
	var err error
	if deferred {
		err = tx.CommitDeferred()
	} else {
		err = tx.Commit()
	}
	return w.finish(seq, deferred, wrote || err != nil, err)
}

/**
* waitDurable espera el fsync de lo diferido y, si vuelve bien, confirma la última operación diferida.
* @return error
**/
func (w *worker) waitDurable() error {
	if w.pending == 0 {
		return nil
	}
	if err := w.e.WaitDurable(); err != nil {
		if noEffect(err) {
			return nil
		}
		return fmt.Errorf("escritor %d, WaitDurable: %w", w.g, err)
	}
	w.j.line("C %d %d", w.g, w.pending)
	w.pending = 0
	return nil
}

/**
* ddl vacía (E) o borra y vuelve a crear (X) el store del escritor; reintenta mientras haya lotes abiertos (ErrBusy).
* @param kind byte
* @return error
**/
func (w *worker) ddl(kind byte) error {
	seq := w.begin(kind, nil)
	var err error
	for try := 0; try < 50; try++ {
		if kind == 'E' {
			err = w.s.Empty()
		} else {
			err = w.e.DropStore(storeName(w.g))
		}
		if !errors.Is(err, store.ErrBusy) && !errors.Is(err, store.ErrStorePending) {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if err == nil && kind == 'X' {
		if rerr := w.reopen(w.e); rerr != nil {
			w.j.line("C %d %d", w.g, seq)
			return fmt.Errorf("escritor %d, Store tras DropStore: %w", w.g, rerr)
		}
	}
	return w.finish(seq, false, true, err)
}

/**
* runWorkers corre los escritores en paralelo hasta stop o max operaciones cada uno.
* @param ws []*worker
* @param stop <-chan struct{}
* @param max int64
* @param onErr func(*worker, error): se llama (desde la goroutine del escritor) cuando uno para por un error
**/
func runWorkers(ws []*worker, stop <-chan struct{}, max int64, onErr func(*worker, error)) {
	var wg sync.WaitGroup
	for _, w := range ws {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := w.run(stop, max); err != nil && onErr != nil {
				onErr(w, err)
			}
		}()
	}
	wg.Wait()
}

/**
* maintenance hace checkpoints y limpiezas a mano mientras corre la carga, para que los cortes caigan también ahí.
* @param e *store.Engine
* @param stop <-chan struct{}
* @param errs *atomic.Int64: cuenta los errores (no detienen nada: el estado lo juzgan los escritores)
**/
func maintenance(e *store.Engine, stop <-chan struct{}, errs *atomic.Int64) {
	ck := time.NewTicker(700 * time.Millisecond)
	vc := time.NewTicker(1300 * time.Millisecond)
	defer ck.Stop()
	defer vc.Stop()
	for {
		var err error
		select {
		case <-stop:
			return
		case <-ck.C:
			err = e.Checkpoint()
		case <-vc.C:
			err = e.Vacuum()
		}
		if err != nil {
			errs.Add(1)
			if errors.Is(err, store.ErrFailed) || errors.Is(err, store.ErrClosed) {
				return
			}
		}
	}
}
