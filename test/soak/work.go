package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/store"
)

/**
* run es el estado compartido de la prueba: el motor, los stores, el modelo, la compuerta y los contadores.
* gate: cada escritor la toma en lectura por unidad de trabajo (que termina con sus lotes y snapshots cerrados); el
* verificador la toma en escritura para mirar el motor quieto, cerrarlo y abrirlo.
**/
type run struct {
	cfg   config
	gate  sync.RWMutex
	e     *store.Engine
	data  []*store.Store
	bank  *store.Store
	model *model

	stop     chan struct{}
	stopOnce sync.Once
	failMu   sync.Mutex
	failure  error

	ops       atomic.Int64 // operaciones de los escritores de datos y del banco
	commits   atomic.Int64 // lotes confirmados (datos + banco)
	rollbacks atomic.Int64 // lotes deshechos a propósito
	conflicts atomic.Int64 // lotes del banco que chocaron (ErrSerialization / ErrConflict) y se deshicieron
	busy      atomic.Int64 // Empty con ErrBusy (había lotes o snapshots abiertos)
	throttled atomic.Int64 // escrituras rechazadas con ErrThrottled (sin efecto)
	ddl       atomic.Int64 // ciclos de DDL hechos
	audits    atomic.Int64 // auditorías del banco
	ddlSeq    int
}

const bankName = "banco"

/**
* dataName da el nombre del store de datos s.
* @param s int
* @return string
**/
func dataName(s int) string {
	return fmt.Sprintf("datos-%d", s)
}

/**
* fail anota el primer error inesperado y detiene la prueba.
* @param format string, args ...any
**/
func (r *run) fail(format string, args ...any) {
	r.failMu.Lock()
	defer r.failMu.Unlock()
	if r.failure != nil {
		return
	}
	r.failure = fmt.Errorf(format, args...)
	logf("FALLA %v", r.failure)
	r.halt()
}

/**
* halt detiene la carga (una sola vez).
**/
func (r *run) halt() {
	r.stopOnce.Do(func() { close(r.stop) })
}

/**
* failed devuelve el error que detuvo la prueba (nil si ninguno).
* @return error
**/
func (r *run) failed() error {
	r.failMu.Lock()
	defer r.failMu.Unlock()
	return r.failure
}

/**
* stopped dice si la prueba terminó (por tiempo, señal o falla).
* @return bool
**/
func (r *run) stopped() bool {
	select {
	case <-r.stop:
		return true
	default:
		return false
	}
}

/**
* pacer reparte la tasa: cada llamada a wait duerme lo que falta para no pasar de interval por operación.
**/
type pacer struct {
	interval time.Duration
	next     time.Time
}

/**
* wait duerme hasta el turno de la próxima operación (sin dormir si interval es 0).
**/
func (p *pacer) wait() {
	if p.interval <= 0 {
		return
	}
	now := time.Now()
	if p.next.Before(now.Add(-time.Second)) {
		p.next = now // tras una pausa no recupera el atraso de golpe
	}
	p.next = p.next.Add(p.interval)
	if d := time.Until(p.next); d > 0 {
		time.Sleep(d)
	}
}

/**
* writer es un escritor de datos: dueño de los índices i con i % escritores == id en todos los stores de datos.
**/
type writer struct {
	r   *run
	id  int
	n   int // escritores
	rnd *rand.Rand
}

/**
* pick elige un store y un índice propio al azar.
* @return int, int
**/
func (w *writer) pick() (int, int) {
	s := w.rnd.IntN(len(w.r.data))
	per := (w.r.model.keys - w.id + w.n - 1) / w.n
	return s, w.rnd.IntN(per)*w.n + w.id
}

/**
* loop corre unidades de trabajo hasta el fin de la prueba.
* @param p *pacer
**/
func (w *writer) loop(p *pacer) {
	for !w.r.stopped() {
		p.wait()
		w.r.gate.RLock()
		if w.r.stopped() {
			w.r.gate.RUnlock()
			return
		}
		err := w.unit()
		w.r.gate.RUnlock()
		if err != nil {
			w.r.fail("escritor %d: %v", w.id, err)
			return
		}
		w.r.ops.Add(1)
	}
}

/**
* unit hace una operación elegida por peso.
* @return error: inesperado
**/
func (w *writer) unit() error {
	x := w.rnd.IntN(100)
	switch {
	case x < 22:
		return w.read()
	case x < 37:
		return w.put(false)
	case x < 42:
		return w.put(true)
	case x < 50:
		return w.insert()
	case x < 60:
		return w.update()
	case x < 66:
		return w.delete()
	case x < 69:
		return w.remove()
	case x < 72:
		return w.version()
	case x < 77:
		return w.scanRange()
	case x < 80:
		return w.scanPrefix()
	case x < 92:
		return w.batch(false)
	case x < 95:
		return w.batch(true)
	default:
		return w.snapshot()
	}
}

/**
* throttled dice si err es un rechazo sin efecto por freno del WAL (se cuenta y se sigue).
* @param err error
* @return bool
**/
func (w *writer) throttled(err error) bool {
	if errors.Is(err, store.ErrThrottled) {
		w.r.throttled.Add(1)
		return true
	}
	return false
}

func (w *writer) read() error {
	s, i := w.pick()
	data, found, err := w.r.data[s].Read(keyOf(i))
	if err != nil {
		return fmt.Errorf("Read: %w", err)
	}
	return w.r.model.expect(s, i, data, found)
}

func (w *writer) put(deferred bool) error {
	m := w.r.model
	s, i := w.pick()
	g := m.gen[s][i] + 1
	var err error
	if deferred {
		err = w.r.data[s].Deferred().Put(keyOf(i), m.value(s, i, g))
	} else {
		err = w.r.data[s].Put(keyOf(i), m.value(s, i, g))
	}
	if w.throttled(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Put: %w", err)
	}
	m.gen[s][i], m.live[s][i] = g, true
	return nil
}

func (w *writer) insert() error {
	m := w.r.model
	s, i := w.pick()
	g := m.gen[s][i] + 1
	wrote, err := w.r.data[s].Insert(keyOf(i), m.value(s, i, g))
	if w.throttled(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Insert: %w", err)
	}
	if wrote == m.live[s][i] {
		return fmt.Errorf("Insert %d/%s escribió=%v con la clave viva=%v", s, keyOf(i), wrote, m.live[s][i])
	}
	if wrote {
		m.gen[s][i], m.live[s][i] = g, true
	}
	return nil
}

func (w *writer) update() error {
	m := w.r.model
	s, i := w.pick()
	g := m.gen[s][i] + 1
	wrote, err := w.r.data[s].Update(keyOf(i), m.value(s, i, g))
	if w.throttled(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Update: %w", err)
	}
	if wrote != m.live[s][i] {
		return fmt.Errorf("Update %d/%s escribió=%v con la clave viva=%v", s, keyOf(i), wrote, m.live[s][i])
	}
	if wrote {
		m.gen[s][i] = g
	}
	return nil
}

func (w *writer) delete() error {
	m := w.r.model
	s, i := w.pick()
	wrote, err := w.r.data[s].Delete(keyOf(i))
	if w.throttled(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Delete: %w", err)
	}
	if wrote != m.live[s][i] {
		return fmt.Errorf("Delete %d/%s borró=%v con la clave viva=%v", s, keyOf(i), wrote, m.live[s][i])
	}
	m.live[s][i] = false
	return nil
}

func (w *writer) remove() error {
	s, i := w.pick()
	err := w.r.data[s].Remove(keyOf(i))
	if w.throttled(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Remove: %w", err)
	}
	w.r.model.live[s][i] = false
	return nil
}

func (w *writer) version() error {
	s, i := w.pick()
	data, _, found, err := w.r.data[s].GetVersion(keyOf(i))
	if err != nil {
		return fmt.Errorf("GetVersion: %w", err)
	}
	if err := w.r.model.expect(s, i, data, found); err != nil {
		return err
	}
	ok, err := w.r.data[s].Exists(keyOf(i))
	if err != nil {
		return fmt.Errorf("Exists: %w", err)
	}
	if ok != found {
		return fmt.Errorf("Exists %d/%s = %v, GetVersion = %v", s, keyOf(i), ok, found)
	}
	return nil
}

/**
* checkScan compara lo recorrido en [lo, hi) con el modelo: solo los índices propios (los demás cambian a la vez).
* @param s, lo, hi int, seen map[int]bool
* @return error
**/
func (w *writer) checkScan(s, lo, hi int, seen map[int]bool) error {
	for i := lo; i < hi && i < w.r.model.keys; i++ {
		if i%w.n == w.id && seen[i] != w.r.model.live[s][i] {
			return fmt.Errorf("recorrido %d/[%s,%s): %s visto=%v, el modelo dice %v", s, keyOf(lo), keyOf(hi), keyOf(i), seen[i], w.r.model.live[s][i])
		}
	}
	return nil
}

func (w *writer) scanRange() error {
	s, lo := w.pick()
	hi := lo + 64
	seen := map[int]bool{}
	prev := -1
	err := w.r.data[s].ForEachRange(keyOf(lo), keyOf(hi), func(id string, data []byte) (bool, error) {
		i, ok := indexOf(id)
		if !ok || i < lo || i >= hi || i <= prev {
			return false, fmt.Errorf("ForEachRange [%s,%s) devolvió %q tras %d", keyOf(lo), keyOf(hi), id, prev)
		}
		prev = i
		if i%w.n == w.id {
			seen[i] = true
			if err := w.r.model.expect(s, i, data, true); err != nil {
				return false, err
			}
		}
		return true, nil
	}, true, 0, 0)
	if err != nil {
		return fmt.Errorf("ForEachRange: %w", err)
	}
	return w.checkScan(s, lo, hi, seen)
}

func (w *writer) scanPrefix() error {
	s, i := w.pick()
	prefix := keyOf(i)[:7] // 10 índices
	lo := i / 10 * 10
	ids, err := w.r.data[s].KeysPrefix(prefix, w.rnd.IntN(2) == 0, 0, 0)
	if err != nil {
		return fmt.Errorf("KeysPrefix: %w", err)
	}
	seen := map[int]bool{}
	for _, id := range ids {
		j, ok := indexOf(id)
		if !ok || j < lo || j >= lo+10 {
			return fmt.Errorf("KeysPrefix %q devolvió %q", prefix, id)
		}
		seen[j] = true
	}
	return w.checkScan(s, lo, lo+10, seen)
}

/**
* change es una escritura pendiente de un lote.
**/
type change struct {
	s, i int
	gen  uint32
	live bool
}

/**
* batch escribe de 2 a 8 claves propias en un lote y lo confirma (a veces diferido) o, si rollback, lo deshace y
* comprueba que no quedó nada.
* @param rollback bool
* @return error
**/
func (w *writer) batch(rollback bool) error {
	m := w.r.model
	tx := w.r.e.NewBatch()
	done := false
	defer func() {
		if !done {
			tx.Rollback()
		}
	}()
	pend := map[[2]int]change{}
	state := func(s, i int) change {
		if c, ok := pend[[2]int{s, i}]; ok {
			return c
		}
		return change{s, i, m.gen[s][i], m.live[s][i]}
	}
	n := 2 + w.rnd.IntN(7)
	for k := 0; k < n; k++ {
		s, i := w.pick()
		st := w.r.data[s]
		c := state(s, i)
		g := c.gen + 1
		var err error
		switch w.rnd.IntN(5) {
		case 0, 1:
			err = st.PutIn(tx, keyOf(i), m.value(s, i, g))
			c.gen, c.live = g, true
		case 2:
			var wrote bool
			wrote, err = st.InsertIn(tx, keyOf(i), m.value(s, i, g))
			if err == nil && wrote == c.live {
				return fmt.Errorf("InsertIn %d/%s escribió=%v con viva=%v", s, keyOf(i), wrote, c.live)
			}
			if wrote {
				c.gen, c.live = g, true
			}
		case 3:
			var wrote bool
			wrote, err = st.DeleteIn(tx, keyOf(i))
			if err == nil && wrote != c.live {
				return fmt.Errorf("DeleteIn %d/%s borró=%v con viva=%v", s, keyOf(i), wrote, c.live)
			}
			c.live = false
		default:
			var data []byte
			var found bool
			data, found, err = st.ReadIn(tx, keyOf(i))
			if err == nil {
				if found != c.live || (found && string(data) != string(m.value(s, i, c.gen))) {
					return fmt.Errorf("ReadIn %d/%s: existe=%v, el lote espera %v (gen %d)", s, keyOf(i), found, c.live, c.gen)
				}
			}
		}
		if w.throttled(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("lote: %w", err)
		}
		pend[[2]int{s, i}] = c
	}
	done = true
	if rollback {
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("Rollback: %w", err)
		}
		w.r.rollbacks.Add(1)
		for k := range pend { // nada del lote quedó
			data, found, err := w.r.data[k[0]].Read(keyOf(k[1]))
			if err != nil {
				return fmt.Errorf("Read tras Rollback: %w", err)
			}
			if err := m.expect(k[0], k[1], data, found); err != nil {
				return fmt.Errorf("tras Rollback: %w", err)
			}
		}
		return nil
	}
	var err error
	if w.rnd.IntN(4) == 0 {
		err = tx.CommitDeferred()
	} else {
		err = tx.Commit()
	}
	if w.throttled(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Commit: %w", err)
	}
	w.r.commits.Add(1)
	for _, c := range pend {
		m.gen[c.s][c.i], m.live[c.s][c.i] = c.gen, c.live
	}
	return nil
}

/**
* snapshot toma un snapshot, cambia una clave propia y exige que el snapshot siga viendo el valor anterior.
* @return error
**/
func (w *writer) snapshot() error {
	m := w.r.model
	s, i := w.pick()
	v := w.r.e.Snapshot()
	defer v.Close()
	oldGen, oldLive := m.gen[s][i], m.live[s][i]
	g := oldGen + 1
	err := w.r.data[s].Put(keyOf(i), m.value(s, i, g))
	if w.throttled(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Put con snapshot: %w", err)
	}
	m.gen[s][i], m.live[s][i] = g, true
	data, found, err := w.r.data[s].ReadAt(v, keyOf(i))
	if err != nil {
		return fmt.Errorf("ReadAt: %w", err)
	}
	if found != oldLive || (found && string(data) != string(m.value(s, i, oldGen))) {
		return fmt.Errorf("ReadAt %d/%s vio el cambio posterior al snapshot (existe=%v, antes %v)", s, keyOf(i), found, oldLive)
	}
	return v.Close()
}

// ---------------------------------------------------------------------------------------------------------------
// Banco: cuentas en un store aparte, transferencias en lotes que chocan entre sí; la suma no cambia nunca.
// ---------------------------------------------------------------------------------------------------------------

const initialBalance = 1000

/**
* accountOf da el id de la cuenta a.
* @param a int
* @return string
**/
func accountOf(a int) string {
	return fmt.Sprintf("cuenta-%04d", a)
}

/**
* balance codifica un saldo.
* @param v int64
* @return []byte
**/
func balance(v int64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(v))
	return b
}

/**
* parseBalance decodifica un saldo.
* @param b []byte
* @return int64, error
**/
func parseBalance(b []byte) (int64, error) {
	if len(b) != 8 {
		return 0, fmt.Errorf("saldo de %d bytes", len(b))
	}
	return int64(binary.LittleEndian.Uint64(b)), nil
}

/**
* bankLoop transfiere entre dos cuentas al azar hasta el fin de la prueba; un choque se deshace y se cuenta.
* @param r *run, id int, p *pacer
**/
func bankLoop(r *run, id int, p *pacer) {
	rnd := rand.New(rand.NewPCG(r.cfg.seed, uint64(1000+id)))
	for !r.stopped() {
		p.wait()
		r.gate.RLock()
		if r.stopped() {
			r.gate.RUnlock()
			return
		}
		err := transfer(r, rnd)
		r.gate.RUnlock()
		if err != nil {
			r.fail("banco %d: %v", id, err)
			return
		}
		r.ops.Add(1)
	}
}

/**
* transfer mueve un monto entre dos cuentas en un lote.
* @param r *run, rnd *rand.Rand
* @return error: inesperado (un choque no lo es)
**/
func transfer(r *run, rnd *rand.Rand) error {
	n := r.cfg.accounts
	a, b := rnd.IntN(n), rnd.IntN(n-1)
	if b >= a {
		b++
	}
	amt := int64(1 + rnd.IntN(10))
	tx := r.e.NewBatch()
	err := func() error {
		var sa, sb int64
		for k, acc := range []int{a, b} {
			data, found, err := r.bank.ReadIn(tx, accountOf(acc))
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("falta la cuenta %s", accountOf(acc))
			}
			v, err := parseBalance(data)
			if err != nil {
				return err
			}
			if k == 0 {
				sa = v
			} else {
				sb = v
			}
		}
		if err := r.bank.PutIn(tx, accountOf(a), balance(sa-amt)); err != nil {
			return err
		}
		return r.bank.PutIn(tx, accountOf(b), balance(sb+amt))
	}()
	if err == nil && rnd.IntN(20) == 0 {
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("Rollback: %w", err)
		}
		r.rollbacks.Add(1)
		return nil
	}
	if err == nil {
		err = tx.Commit()
		if err == nil {
			r.commits.Add(1)
			return nil
		}
	}
	tx.Rollback()
	switch {
	case errors.Is(err, store.ErrSerialization), errors.Is(err, store.ErrConflict):
		r.conflicts.Add(1)
		return nil
	case errors.Is(err, store.ErrThrottled):
		r.throttled.Add(1)
		return nil
	}
	return fmt.Errorf("transferencia: %w", err)
}

/**
* auditBank suma las cuentas en un snapshot (debe dar cuentas × saldo inicial); si hold > 0 mantiene el snapshot
* abierto ese tiempo (retiene el horizonte de la limpieza) y vuelve a sumar: debe dar lo mismo.
* @param r *run, hold time.Duration
* @return error
**/
func auditBank(r *run, hold time.Duration) error {
	v := r.e.Snapshot()
	defer v.Close()
	sum := func() (int64, int, error) {
		var total int64
		n := 0
		err := r.bank.ForEachAt(v, func(id string, data []byte) (bool, error) {
			b, err := parseBalance(data)
			total += b
			n++
			return err == nil, err
		}, true, 0, 0)
		return total, n, err
	}
	total, n, err := sum()
	if err != nil {
		return fmt.Errorf("ForEachAt: %w", err)
	}
	want := int64(r.cfg.accounts) * initialBalance
	if total != want || n != r.cfg.accounts {
		return fmt.Errorf("auditoría: %d cuentas suman %d, se esperaban %d que suman %d", n, total, r.cfg.accounts, want)
	}
	if c, err := r.bank.CountAt(v); err != nil || c != int64(n) {
		return fmt.Errorf("CountAt = %d (%v), el recorrido %d", c, err, n)
	}
	if hold > 0 {
		select {
		case <-time.After(hold):
		case <-r.stop:
			return nil
		}
		if again, _, err := sum(); err != nil || again != total {
			return fmt.Errorf("el snapshot cambió: %d tras %v, antes %d (%v)", again, hold, total, err)
		}
	}
	r.audits.Add(1)
	return v.Close()
}

/**
* auditLoop audita el banco cada 2 s y, cada longEvery, con un snapshot largo de longHold.
* @param r *run
**/
func auditLoop(r *run) {
	last := time.Now()
	for !r.stopped() {
		select {
		case <-time.After(2 * time.Second):
		case <-r.stop:
			return
		}
		hold := time.Duration(0)
		if time.Since(last) >= r.cfg.longEvery {
			hold, last = r.cfg.longHold, time.Now()
		}
		r.gate.RLock()
		err := error(nil)
		if !r.stopped() {
			err = auditBank(r, hold)
		}
		r.gate.RUnlock()
		if err != nil {
			r.fail("auditor: %v", err)
			return
		}
	}
}

// ---------------------------------------------------------------------------------------------------------------
// DDL ocasional: stores que se crean, llenan, vacían y borran (sueltos y en lotes, también deshechos).
// ---------------------------------------------------------------------------------------------------------------

/**
* ddlLoop hace un ciclo de DDL cada cfg.ddlEvery.
* @param r *run
**/
func ddlLoop(r *run) {
	for !r.stopped() {
		select {
		case <-time.After(r.cfg.ddlEvery):
		case <-r.stop:
			return
		}
		r.gate.RLock()
		err := error(nil)
		if !r.stopped() {
			err = ddlCycle(r)
		}
		r.gate.RUnlock()
		if err != nil {
			r.fail("ddl: %v", err)
			return
		}
		r.ddl.Add(1)
	}
}

/**
* hasStore dice si Stores() lista name.
* @param e *store.Engine, name string
* @return bool
**/
func hasStore(e *store.Engine, name string) bool {
	for _, n := range e.Stores() {
		if n == name {
			return true
		}
	}
	return false
}

/**
* ddlCycle rota entre tres variantes: suelto (Store, Put diferido, Empty, DropStore), en lote confirmado (StoreIn +
* DropStoreIn) y en lote deshecho. Ninguno deja stores: el verificador exige el conjunto fijo.
* @param r *run
* @return error
**/
func ddlCycle(r *run) error {
	r.ddlSeq++
	name := fmt.Sprintf("ddl-%d", r.ddlSeq)
	e := r.e
	switch r.ddlSeq % 3 {
	case 0:
		s, err := e.Store(name)
		if err != nil {
			return fmt.Errorf("Store: %w", err)
		}
		for i := 0; i < 200; i++ {
			if err := s.Deferred().Put(keyOf(i), balance(int64(i))); err != nil {
				return fmt.Errorf("Put: %w", err)
			}
		}
		if c := s.Count(); c != 200 {
			return fmt.Errorf("Count %s = %d, se esperaba 200", name, c)
		}
		if ids, err := s.Keys(false, 10, 5); err != nil || len(ids) != 5 || ids[0] != keyOf(189) {
			return fmt.Errorf("Keys %s = %v (%v)", name, ids, err)
		}
		switch err := s.Empty(); {
		case errors.Is(err, store.ErrBusy):
			r.busy.Add(1)
		case err != nil:
			return fmt.Errorf("Empty: %w", err)
		case s.Count() != 0:
			return fmt.Errorf("Count %s = %d tras Empty", name, s.Count())
		}
		if err := e.DropStore(name); err != nil {
			return fmt.Errorf("DropStore: %w", err)
		}
	case 1:
		tx := e.NewBatch()
		s, err := e.StoreIn(tx, name)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("StoreIn: %w", err)
		}
		for i := 0; i < 50; i++ {
			if err := s.PutIn(tx, keyOf(i), balance(int64(i))); err != nil {
				tx.Rollback()
				return fmt.Errorf("PutIn: %w", err)
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("Commit StoreIn: %w", err)
		}
		if !hasStore(e, name) || s.Count() != 50 {
			return fmt.Errorf("%s tras Commit: listado=%v Count=%d", name, hasStore(e, name), s.Count())
		}
		tx = e.NewBatch()
		if err := e.DropStoreIn(tx, name); err != nil {
			tx.Rollback()
			return fmt.Errorf("DropStoreIn: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("Commit DropStoreIn: %w", err)
		}
	default:
		tx := e.NewBatch()
		s, err := e.StoreIn(tx, name)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("StoreIn: %w", err)
		}
		if err := s.PutIn(tx, keyOf(0), balance(1)); err != nil {
			tx.Rollback()
			return fmt.Errorf("PutIn: %w", err)
		}
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("Rollback StoreIn: %w", err)
		}
	}
	if hasStore(e, name) {
		return fmt.Errorf("%s sigue listado tras borrarlo o deshacerlo", name)
	}
	return nil
}
