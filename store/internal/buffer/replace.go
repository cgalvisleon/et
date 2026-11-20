package buffer

import (
	"errors"
	"runtime"
	"sync"
)

// errVictimBusy: la víctima elegida tiene su contenido o su escritura tomados por otro; el reemplazo elige otra.
var errVictimBusy = errors.New("buffer: victim is locked by another")

// ring es el anillo de los recorridos grandes (Strategy Ring): ringLen búferes que esos recorridos reutilizan en
// vez de pasar por el clock-sweep, para no vaciar la caché de todos. Es uno por caché, compartido por todos los
// recorridos (a confirmar: PostgreSQL da uno a cada recorrido, pero Pin no recibe un objeto de estrategia).
type ring struct {
	mu    sync.Mutex
	slots []int32 // búfer de cada lugar; -1 = todavía ninguno
	next  int
}

/**
* init deja el anillo con n lugares vacíos.
* @param n int
**/
func (r *ring) init(n int) {
	r.slots = make([]int32, n)
	for i := range r.slots {
		r.slots[i] = -1
	}
}

/**
* advance toma el próximo lugar del anillo.
* @return int: el lugar, int32: su búfer (-1 = ninguno)
**/
func (r *ring) advance() (int, int32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	slot := r.next
	r.next = (r.next + 1) % len(r.slots)
	return slot, r.slots[slot]
}

/**
* set anota el búfer de un lugar.
* @param slot int, id int32
**/
func (r *ring) set(slot int, id int32) {
	r.mu.Lock()
	r.slots[slot] = id
	r.mu.Unlock()
}

/**
* victim devuelve un búfer fijado (pins = 1) y limpio para cargar otra página (ver candidate). Si estaba sucio lo
* escribe antes, cumpliendo «WAL antes que la página». Quien pide la página puede tener otras bloqueadas, así que la
* escritura de la víctima no espera sus bloqueos: si otro la fijó después de elegirla y tiene su contenido (o su
* escritura) tomado, la suelta y elige otra (errVictimBusy), como BufferAlloc de PostgreSQL. Con la caché de solo
* lectura (un fallo de disco lleno) no escribe nada: saltea los sucios, que se quedan en la caché con su cambio, y si
* en una vuelta entera no encuentra uno limpio devuelve el fallo.
* @param strategy Strategy
* @return *desc, error: ErrNoBuffers, o el fallo de escritura
**/
func (p *pool) victim(strategy Strategy) (*desc, error) {
	for skipped, busy := 0, 0; ; {
		d, err := p.candidate(strategy)
		if err != nil {
			return nil, err
		}
		err = p.flush(d, true)
		if err == nil {
			return d, nil
		}
		p.unpin(d)
		if errors.Is(err, errVictimBusy) {
			if busy++; busy%len(p.descs) == 0 {
				runtime.Gosched() // una vuelta entera ocupada: dejar correr a quien tiene los bloqueos
			}
			continue
		}
		if skipped++; !p.readOnly() || skipped > len(p.descs) {
			return nil, err
		}
	}
}

/**
* candidate elige un búfer fijado (pins = 1) para reemplazar, sin escribirlo: del anillo si la estrategia es Ring y
* el búfer de su lugar está libre y poco usado (uso ≤ 1, como GetBufferFromRing), si no del clock-sweep.
* @param strategy Strategy
* @return *desc, error: ErrNoBuffers, ErrClosed
**/
func (p *pool) candidate(strategy Strategy) (*desc, error) {
	slot, id := -1, int32(-1)
	if strategy == Ring {
		slot, id = p.ring.advance()
	}

	var d *desc
	if id >= 0 && p.tryPin(&p.descs[id], 1) {
		d = &p.descs[id]
	} else {
		var err error
		if d, err = p.clockSweep(); err != nil {
			return nil, err
		}
		if slot >= 0 {
			p.ring.set(slot, d.id)
		}
	}
	return d, nil
}

/**
* tryPin fija el búfer si nadie lo tiene fijado, su uso no pasa de maxUsage y la caché sigue abierta.
* @param d *desc, maxUsage uint8
* @return bool
**/
func (p *pool) tryPin(d *desc, maxUsage uint8) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.pins != 0 || d.usage > maxUsage {
		return false
	}
	return p.pinLocked(d)
}

/**
* clockSweep recorre los búferes desde la manecilla: baja el contador de uso de los que no están fijados y fija el
* primero que encuentra libre con uso 0 (StrategyGetBuffer). Si da una vuelta entera sin bajar ningún contador,
* todos están fijados.
* @return *desc: fijado (pins = 1), error: ErrNoBuffers, ErrClosed
**/
func (p *pool) clockSweep() (*desc, error) {
	n := uint64(len(p.descs))
	tries := n
	for {
		d := &p.descs[(p.hand.Add(1)-1)%n]
		d.mu.Lock()
		switch {
		case d.pins == 0 && d.usage == 0:
			ok := p.pinLocked(d)
			d.mu.Unlock()
			if !ok {
				return nil, ErrClosed
			}
			return d, nil
		case d.pins == 0:
			d.usage--
			tries = n
		default:
			tries--
		}
		d.mu.Unlock()
		if tries == 0 {
			return nil, ErrNoBuffers
		}
	}
}
