package xact

import (
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/store/internal/page"
)

// running es una transacción en curso.
type running struct {
	done chan struct{} // se cierra en End: despierta a los que esperan
	xmin page.XID      // el Xmin de su snapshot, si tomó uno (0 = ninguno)
}

// Transactions lleva las transacciones en curso: asigna xid, toma snapshots y hace esperar. La memoria crece solo
// con las transacciones en curso (la concurrencia), no con las que ya terminaron.
type Transactions struct {
	mu      sync.Mutex
	next    page.XID
	running map[page.XID]*running

	// readers son los snapshots de solo lectura abiertos (TakeSnapshot), por número: su Xmin cuenta en OldestActive
	// hasta ReleaseSnapshot. Crece con los snapshots abiertos, no con las transacciones terminadas.
	readers    map[uint64]page.XID
	nextReader uint64
}

var _ Manager = (*Transactions)(nil)

// NewManager crea el registro de transacciones en curso, vacío.
// @param next: el siguiente xid a asignar (el nextXID de la recuperación); 0 empieza en 1.
func NewManager(next page.XID) *Transactions {
	return &Transactions{next: max(next, 1), running: map[page.XID]*running{}, readers: map[uint64]page.XID{}}
}

// Begin asigna el xid siguiente y anota la transacción en curso.
func (m *Transactions) Begin() page.XID {
	m.mu.Lock()
	defer m.mu.Unlock()
	xid := m.next
	m.next++
	m.running[xid] = &running{done: make(chan struct{})}
	return xid
}

// End quita xid de las en curso y despierta a quien la espera. Se llama después de SetStatus.
func (m *Transactions) End(xid page.XID) {
	m.mu.Lock()
	r := m.running[xid]
	delete(m.running, xid)
	m.mu.Unlock()
	if r != nil {
		close(r.done)
	}
}

// Snapshot toma la foto actual para la transacción own (0 = solo lectura). El Xmin de la primera foto de own cuenta
// en OldestActive hasta su End.
func (m *Transactions) Snapshot(own page.XID) Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshotLocked(own)
}

// snapshotLocked toma la foto con m.mu tomado.
func (m *Transactions) snapshotLocked(own page.XID) Snapshot {
	inProgress := make([]page.XID, 0, len(m.running))
	for xid := range m.running {
		inProgress = append(inProgress, xid)
	}
	slices.Sort(inProgress)

	xmin := m.next
	if len(inProgress) > 0 {
		xmin = inProgress[0]
	}
	if r := m.running[own]; r != nil && r.xmin == 0 {
		r.xmin = xmin
	}
	return Snapshot{Own: own, Xmin: xmin, Xmax: m.next, InProgress: inProgress}
}

// TakeSnapshot toma una foto de solo lectura (Own = 0) y la registra: su Xmin cuenta en OldestActive hasta
// ReleaseSnapshot con el número devuelto.
func (m *Transactions) TakeSnapshot() (Snapshot, uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	snap := m.snapshotLocked(0)
	m.nextReader++
	m.readers[m.nextReader] = snap.Xmin
	return snap, m.nextReader
}

// ReleaseSnapshot quita del horizonte el snapshot de solo lectura id. Soltarlo dos veces no hace nada.
func (m *Transactions) ReleaseSnapshot(id uint64) {
	m.mu.Lock()
	delete(m.readers, id)
	m.mu.Unlock()
}

// Wait espera a que xid termine, hasta timeout (ErrWaitTimeout); timeout ≤ 0 espera sin límite. Una xid que no
// está en curso vuelve enseguida.
func (m *Transactions) Wait(xid page.XID, timeout time.Duration) error {
	m.mu.Lock()
	r := m.running[xid]
	m.mu.Unlock()
	if r == nil {
		return nil
	}
	if timeout <= 0 {
		<-r.done
		return nil
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-r.done:
		return nil
	case <-timer.C:
		return ErrWaitTimeout
	}
}

// NextXID es el siguiente xid a asignar.
func (m *Transactions) NextXID() page.XID {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.next
}

// OldestActive es el menor xid en curso, Xmin de un snapshot de una transacción en curso o Xmin de un snapshot de
// solo lectura abierto; NextXID si no hay.
func (m *Transactions) OldestActive() page.XID {
	m.mu.Lock()
	defer m.mu.Unlock()
	oldest := m.next
	for xid, r := range m.running {
		oldest = min(oldest, xid)
		if r.xmin != 0 {
			oldest = min(oldest, r.xmin)
		}
	}
	for _, xmin := range m.readers {
		oldest = min(oldest, xmin)
	}
	return oldest
}
