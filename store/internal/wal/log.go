package wal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/store/internal/metric"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// WAL es el log abierto para escribir; implementa Log y Reader. Cada bloque de campos lo protege el lock de su
// comentario. Orden de los locks: writeMu → drainMu → pubMu, writeMu → syncMu, syncMu → pubMu; nunca al revés.
type WAL struct {
	fs   vfs.FS
	dir  string // <motor>/wal
	opts Options

	// writeMu: reservas, rotación, punto de rehacer, reciclado y cierre.
	writeMu sync.Mutex
	active  *segment            // se cambia con writeMu y syncMu tomados: WaitDurable lo lee solo con syncMu
	sizes   map[uint64]Position // fin de lo escrito en cada segmento anterior al activo (SizeSince, SegmentInfo)
	maxNum  uint64              // el mayor segmento que existe: los mayores que el activo son repuestos reciclados
	// nextReady: el segmento siguiente al activo existe y está preasignado entero (Options.Preallocate).
	nextReady bool
	written   atomic.Int64  // bytes reservados desde Open (Written); se escribe con writeMu
	activeNum atomic.Uint64 // el número del segmento activo (ActiveSegment); se escribe con writeMu
	lastLSN   atomic.Uint64 // último lsn reservado; se escribe con writeMu
	redo      atomic.Pointer[Position]
	closed    atomic.Bool // se pone con writeMu y syncMu tomados

	// forcing y forceFrom: el modo respaldo (ForceImages). Con forcing > 0, la imagen de página entera se decide
	// contra max(punto de rehacer, forceFrom). holds: los segmentos que Recycle no debe tocar (Hold), por número.
	forcing   int
	forceFrom page.LSN
	holds     map[uint64]int

	// pubMu: qué registros esperan escribirse y cuáles ya están en el archivo.
	pubMu   sync.Mutex
	pubCond *sync.Cond
	// pending: registros ya sellados (Write) que todavía no están en el archivo, por lsn; pendingBytes su largo.
	// Se escriben en tandas contiguas (drain): un pwrite por tanda en lugar de uno por registro (tarea 100).
	pending      map[page.LSN]*reservation
	pendingBytes int64
	wanted       page.LSN      // el mayor lsn que alguien espera ver escrito (waitPublished): mientras pase a published, Write escribe enseguida
	failure      error         // primer error de escritura o fsync: el WAL queda inutilizable
	published    atomic.Uint64 // todos los lsn ≤ published están escritos en el archivo; se escribe con pubMu

	// drainMu: una tanda de escritura a la vez (drain); scratch es su búfer, de tamaño fijo (maxRun).
	drainMu sync.Mutex
	scratch []byte

	// syncMu: qué está en disco; lo toma solo el líder del fsync (WaitDurable), rotate y Close.
	syncMu  sync.Mutex
	durable atomic.Uint64 // todos los lsn ≤ durable tienen fsync; se escribe con syncMu

	// leadMu: quién hace el fsync del group commit. syncing es el aviso del fsync en curso (se cierra al terminar),
	// nil si no hay ninguno. Se toma corto y sin otro lock.
	leadMu  sync.Mutex
	syncing chan struct{}

	// syncs mide cada fsync de WaitDurable (el del group commit), de tamaño fijo y solo con atómicos.
	syncs metric.Histogram
}

// Comprobación en compilación de que WAL cumple las interfaces.
var (
	_ Log    = (*WAL)(nil)
	_ Reader = (*WAL)(nil)
)

// Open abre el WAL del motor de dir y lo recupera: recorre los registros desde redo (el punto de rehacer del último
// checkpoint, o Start en un motor nuevo) llamando a fn con cada uno (nil: ninguno), corta la cola inválida (el resto
// del último segmento y los segmentos siguientes, que nunca se confirmaron) y deja listo el WAL para seguir
// escribiendo. Si la cola inválida empieza dentro de un segmento sellado (checkSealed), falla con ErrCorrupted sin
// cortar nada. Devuelve el WAL y el final de lo válido: la posición del próximo registro.
func Open(fsys vfs.FS, dir string, opts Options, redo Position, fn func(rec *Record) error) (*WAL, Position, error) {
	if opts.SegmentSize < minSegmentSize {
		return nil, Position{}, fmt.Errorf(MSG_SEGMENT_SIZE, opts.SegmentSize, minSegmentSize)
	}
	if fn == nil {
		fn = func(*Record) error { return nil }
	}
	walDir := filepath.Join(dir, Dir)
	if err := fsys.MkdirAll(walDir); err != nil {
		return nil, Position{}, err
	}
	nums, err := listSegments(fsys, walDir)
	if err != nil {
		return nil, Position{}, err
	}
	if len(nums) == 0 {
		if redo != Start {
			return nil, Position{}, fmt.Errorf(MSG_WAL_MISSING, redo)
		}
		file, err := createSegment(fsys, walDir, 1)
		if err != nil {
			return nil, Position{}, err
		}
		file.Close() // el encabezado lo escribe ensureHeader, igual que tras un corte antes de escribirlo
		nums = []uint64{1}
	}

	sizes := map[uint64]Position{}
	end, err := scan(fsys, walDir, redo, opts.StopLSN, recordsOnly(fn), func(end Position) { sizes[end.Segment] = end })
	if err != nil {
		return nil, end, err
	}
	if opts.StopLSN == 0 || end.LSN <= opts.StopLSN { // el objetivo de una recuperación corta a propósito
		if err := checkSealed(fsys, walDir, end, nums); err != nil {
			return nil, end, err
		}
	}
	if err := cutTail(fsys, walDir, end, nums); err != nil {
		return nil, end, err
	}
	file, err := fsys.OpenFile(segmentPath(walDir, end.Segment), os.O_RDWR)
	if err != nil {
		return nil, end, err
	}
	if err := ensureHeader(file, end); err != nil {
		file.Close()
		return nil, end, err
	}
	delete(sizes, end.Segment)

	allocated := opts.SegmentSize // sin preasignación nunca se rellena con ceros
	if opts.Preallocate {
		allocated = end.Offset // el resto lo preasigna Room (o la primera reserva que lo necesite)
	}
	w := &WAL{
		fs:      fsys,
		dir:     walDir,
		opts:    opts,
		active:  &segment{num: end.Segment, file: file, size: end.Offset, allocated: allocated},
		sizes:   sizes,
		maxNum:  nums[len(nums)-1],
		pending: map[page.LSN]*reservation{},
	}
	w.pubCond = sync.NewCond(&w.pubMu)
	w.activeNum.Store(end.Segment)
	last := uint64(end.LSN - 1)
	w.lastLSN.Store(last)
	w.published.Store(last)
	w.durable.Store(last)
	w.redo.Store(&redo)
	return w, end, nil
}

/**
* ensureHeader escribe (con fsync) el encabezado del segmento activo de Open si todavía no tiene registros y el
* encabezado falta: un segmento recién creado (el primero de un motor nuevo) o uno cortado antes de que su
* encabezado fuera durable. Uno con registros ya pasó por checkStartHeader o por el paso de un segmento al siguiente.
* @param file vfs.File, end Position: el final válido, en este segmento
* @return error
**/
func ensureHeader(file vfs.File, end Position) error {
	if end.Offset != SegmentHeaderSize {
		return nil
	}
	h, state, err := readHeader(file)
	if err != nil || state == headerValid && h.Num == end.Segment && h.First == end.LSN {
		return err
	}
	if _, err := file.WriteAt(encodeHeader(end.Segment, end.LSN), 0); err != nil {
		return err
	}
	return file.Sync()
}

// cutTail corta lo que sigue al final válido: el segmento de end en su desplazamiento y los siguientes enteros.
// Hay que cortarlos, no solo ignorarlos: un registro viejo nunca confirmado podría quedar justo donde termina uno
// nuevo y pasar por el siguiente al recuperar otra vez.
func cutTail(fsys vfs.FS, dir string, end Position, nums []uint64) error {
	for _, num := range nums {
		switch {
		case num == end.Segment:
			if err := truncateSegment(fsys, dir, num, end.Offset); err != nil {
				return err
			}
		case num > end.Segment:
			if err := truncateSegment(fsys, dir, num, 0); err != nil {
				return err
			}
		}
	}
	return nil
}

// Scan implementa Reader sobre los segmentos de este WAL.
func (w *WAL) Scan(from Position, fn func(rec *Record) error) (Position, error) {
	return scan(w.fs, w.dir, from, 0, recordsOnly(fn), nil)
}

// reservation es el lugar reservado de un registro ya codificado (sin lsn ni crc).
type reservation struct {
	w       *WAL
	seg     *segment
	offset  int64
	buf     []byte
	size    int64
	lsn     page.LSN
	written bool
}

// LSN es el lsn asignado.
func (r *reservation) LSN() page.LSN {
	return r.lsn
}

// End es la posición siguiente al registro.
func (r *reservation) End() Position {
	return Position{Segment: r.seg.num, Offset: r.offset + r.size, LSN: r.lsn + 1}
}

// Write completa el lsn y el crc y deja el registro en espera de escribirse (pending). Lo escribe enseguida, junto
// con los que esperan detrás de él, salvo que el WAL tenga búfer (Options.Buffer), lo pendiente no llegue a él y
// nadie espere el registro: entonces queda para la próxima tanda (la de WaitDurable, la del sellado o la de otro
// Write). Nunca espera a otra reserva: si falta un registro anterior, la tanda la escribe su dueño. Se llama una vez.
func (r *reservation) Write() error {
	if r.written {
		return fmt.Errorf(MSG_WRITTEN_TWICE, r.lsn)
	}
	r.written = true
	sealRecord(r.buf, r.lsn)
	w := r.w
	w.pubMu.Lock()
	if w.failure != nil {
		err := w.failure
		w.pubMu.Unlock()
		return failedError(err)
	}
	w.pending[r.lsn] = r
	w.pendingBytes += r.size
	lazy := w.opts.Buffer > 0 && w.pendingBytes < w.opts.Buffer && w.wanted <= page.LSN(w.published.Load())
	w.pubMu.Unlock()
	if lazy {
		return nil
	}
	return w.drain()
}

// maxRun es el largo máximo de una tanda de drain (un pwrite).
const maxRun = 1 << 20

// drain escribe en el archivo los registros pendientes que siguen a published, en tandas contiguas del mismo
// segmento de hasta maxRun bytes, y avanza published tras cada una. Para en el primer hueco (una reserva que su dueño
// todavía no escribió): no espera a nadie. Un error de escritura deja el WAL inutilizable.
func (w *WAL) drain() error {
	w.drainMu.Lock()
	defer w.drainMu.Unlock()
	for {
		run := w.nextRun()
		if len(run) == 0 {
			return nil
		}
		if err := w.writeRun(run); err != nil {
			w.fail(err)
			return failedError(err)
		}
		w.pubMu.Lock()
		w.published.Store(uint64(run[len(run)-1].lsn))
		w.pubCond.Broadcast()
		w.pubMu.Unlock()
	}
}

// nextRun saca de pending la tanda contigua que sigue a published (vacía si hay un hueco o el WAL falló). Con
// drainMu tomado.
func (w *WAL) nextRun() []*reservation {
	w.pubMu.Lock()
	defer w.pubMu.Unlock()
	if w.failure != nil {
		return nil
	}
	var run []*reservation
	size := int64(0)
	for next := page.LSN(w.published.Load()) + 1; ; next++ {
		r := w.pending[next]
		if r == nil || len(run) > 0 && (size+r.size > maxRun || r.seg != run[0].seg ||
			r.offset != run[len(run)-1].offset+run[len(run)-1].size) {
			break
		}
		delete(w.pending, next)
		w.pendingBytes -= r.size
		run = append(run, r)
		size += r.size
	}
	return run
}

// writeRun escribe una tanda contigua con un solo pwrite. Con drainMu tomado.
func (w *WAL) writeRun(run []*reservation) error {
	first := run[0]
	buf := first.buf
	if len(run) > 1 {
		if w.scratch == nil {
			w.scratch = make([]byte, maxRun)
		}
		buf = w.scratch[:0]
		for _, r := range run {
			buf = append(buf, r.buf...)
		}
	}
	_, err := first.seg.file.WriteAt(buf, first.offset)
	for _, r := range run {
		r.buf = nil
	}
	return err
}

// Reserve codifica rec, y bajo writeMu comprueba las imágenes contra el punto de rehacer vigente (ErrNeedImage),
// asigna el lsn y el lugar (rotando si no cabe), pone rec.LSN y llama a underLock. Ver Log.Reserve.
func (w *WAL) Reserve(rec *Record, underLock func(lsn page.LSN)) (Reservation, error) {
	size, err := recordSize(rec)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, size)
	encodeRecord(rec, buf)

	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	if err := w.writable(); err != nil {
		return nil, err
	}
	if needsImage(rec, w.imageLSN()) {
		return nil, ErrNeedImage
	}
	seg, offset, err := w.allocate(int64(size))
	if err != nil {
		return nil, err
	}
	lsn := page.LSN(w.lastLSN.Add(1))
	rec.LSN = lsn
	if underLock != nil {
		underLock(lsn)
	}
	return &reservation{w: w, seg: seg, offset: offset, buf: buf, size: int64(size), lsn: lsn}, nil
}

// writable devuelve por qué no se puede escribir (cerrado o fallado), o nil. Con writeMu tomado.
func (w *WAL) writable() error {
	if w.closed.Load() {
		return ErrClosed
	}
	if err := w.Failed(); err != nil {
		return failedError(err)
	}
	return nil
}

// allocate toma size bytes del segmento activo; si no caben, lo sella y sigue en el siguiente. Con preasignación, si
// el lugar todavía no está preasignado lo preasigna ahora: con el disco lleno devuelve ErrNoRoom sin dejar el WAL
// inutilizable (no se asignó lsn). Con writeMu tomado.
func (w *WAL) allocate(size int64) (*segment, int64, error) {
	if w.active.size+size > w.capacity() {
		if err := w.rotate(); err != nil {
			return nil, 0, w.reserveError(err)
		}
	}
	if w.active.size+size > w.active.allocated {
		if err := w.fillActive(); err != nil {
			return nil, 0, w.reserveError(err)
		}
	}
	seg := w.active
	offset := seg.size
	seg.size += size
	w.written.Add(size)
	return seg, offset, nil
}

// rotate sella el segmento activo (sus escrituras terminadas y fsync, y después su marca de sellado) y pasa al siguiente: un repuesto reciclado si
// lo hay, o uno nuevo. Con preasignación, antes de sellar deja el siguiente preasignado: si el disco está lleno
// devuelve ese error sin tocar nada. Un fsync o una apertura fallidos dejan el WAL inutilizable. Con writeMu tomado.
func (w *WAL) rotate() error {
	if w.opts.Preallocate {
		if err := w.prepareNext(); err != nil {
			return err
		}
	}
	old := w.active
	num := old.num + 1
	file, err := w.openNext(num)
	if err != nil {
		return err
	}
	// Orden: los registros del activo durables, después el encabezado del siguiente (con fsync) y recién entonces la
	// marca de sellado. Así un encabezado durable nunca promete un lsn que el activo no tiene en disco (si no, un corte
	// entre los dos fsync dejaba un hueco falso) y un segmento sellado siempre tiene el siguiente con su encabezado
	// (checkSealed lo exige). El primer registro del siguiente es el próximo lsn (writeMu tomado).
	if err := w.seal(old); err != nil {
		file.Close()
		w.fail(err)
		return failedError(err)
	}
	if err := w.writeHeader(file, num, page.LSN(w.lastLSN.Load())+1); err != nil {
		file.Close()
		return err
	}
	if err := w.markSealed(old); err != nil {
		file.Close()
		return err
	}

	// El cambio de activo va con syncMu: un WaitDurable que leyó el viejo termina su fsync antes de que se cierre.
	w.syncMu.Lock()
	w.active = &segment{num: num, file: file, size: SegmentHeaderSize, allocated: w.opts.SegmentSize}
	w.nextReady = false
	w.activeNum.Store(num)
	w.syncMu.Unlock()
	w.sizes[old.num] = Position{Segment: old.num, Offset: old.size, LSN: page.LSN(w.lastLSN.Load()) + 1}
	return old.file.Close()
}

// Published es el mayor lsn tal que todos los anteriores están escritos.
func (w *WAL) Published() page.LSN {
	return page.LSN(w.published.Load())
}

// LastLSN es el último lsn reservado (puede no estar escrito todavía).
func (w *WAL) LastLSN() page.LSN {
	return page.LSN(w.lastLSN.Load())
}

// Durable es el mayor lsn tal que todos los anteriores tienen fsync.
func (w *WAL) Durable() page.LSN {
	return page.LSN(w.durable.Load())
}

// waitPublished espera a que todos los registros hasta lsn estén escritos en el archivo: anota lsn como esperado
// (así cada Write escribe enseguida) y escribe lo pendiente; si falta una reserva que su dueño no escribió, espera
// a que lo haga. Falla si el WAL falló.
func (w *WAL) waitPublished(lsn page.LSN) error {
	if w.Published() >= lsn {
		return nil
	}
	w.pubMu.Lock()
	w.wanted = max(w.wanted, lsn)
	w.pubMu.Unlock()
	if err := w.drain(); err != nil {
		return err
	}
	w.pubMu.Lock()
	defer w.pubMu.Unlock()
	for w.Published() < lsn && w.failure == nil {
		w.pubCond.Wait()
	}
	if w.Published() < lsn {
		return failedError(w.failure)
	}
	return nil
}

// WaitDurable vuelve cuando todo hasta lsn tiene fsync (group commit). Un solo líder a la vez hace el fsync, que
// confirma todo lo publicado al empezar; los demás esperan su aviso (un canal que se cierra al terminar, así se
// despiertan todos a la vez y no uno tras otro en una cola de syncMu, tarea 100 de F-23) y, si su lsn quedó fuera,
// el primero en llegar es el líder del fsync siguiente. Un fsync fallido deja el WAL inutilizable y no se reintenta.
func (w *WAL) WaitDurable(lsn page.LSN) error {
	if w.Durable() >= lsn {
		return nil
	}
	if last := page.LSN(w.lastLSN.Load()); lsn > last {
		return fmt.Errorf(MSG_LSN_AHEAD, lsn, last)
	}
	if err := w.waitPublished(lsn); err != nil {
		return err
	}
	for {
		if w.Durable() >= lsn {
			return nil
		}
		w.leadMu.Lock()
		if w.syncing != nil {
			done := w.syncing
			w.leadMu.Unlock()
			<-done
			continue
		}
		done := make(chan struct{})
		w.syncing = done
		w.leadMu.Unlock()
		err := w.syncPublished()
		w.leadMu.Lock()
		w.syncing = nil
		w.leadMu.Unlock()
		close(done)
		if err != nil {
			return err
		}
	}
}

// syncPublished hace el fsync del segmento activo, que confirma todo lo publicado hasta ese momento, y sube durable.
// Lo llama solo el líder de WaitDurable; syncMu lo separa de rotate y Close, que cambian o cierran el activo.
func (w *WAL) syncPublished() error {
	w.syncMu.Lock()
	defer w.syncMu.Unlock()
	if w.closed.Load() {
		return ErrClosed
	}
	if err := w.Failed(); err != nil {
		return failedError(err)
	}
	// target se lee con syncMu tomado: el fsync confirma todo lo publicado hasta ahora. Los segmentos anteriores al
	// activo ya se sellaron con su fsync (ver rotate).
	target := w.published.Load()
	if target <= w.durable.Load() {
		return nil
	}
	start := time.Now()
	err := w.active.file.Sync()
	w.syncs.Since(start)
	if err != nil {
		w.fail(err)
		return failedError(err)
	}
	w.durable.Store(max(w.durable.Load(), target))
	return nil
}

// SyncLatency es la foto de las latencias de los fsync de WaitDurable (el de rotate, al sellar un segmento, no
// cuenta).
func (w *WAL) SyncLatency() metric.Summary {
	return w.syncs.Summary()
}

// RedoPoint es el punto de rehacer vigente.
func (w *WAL) RedoPoint() Position {
	return *w.redo.Load()
}

// BeginCheckpoint fija, bajo writeMu, el punto de rehacer en la posición del próximo registro, llama a underLock con
// ella y la devuelve. Si el próximo registro no cabe en el segmento activo empezará el siguiente: Scan lo sigue.
func (w *WAL) BeginCheckpoint(underLock func(redo Position)) (Position, error) {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	if err := w.writable(); err != nil {
		return Position{}, err
	}
	redo := Position{Segment: w.active.num, Offset: w.active.size, LSN: page.LSN(w.lastLSN.Load()) + 1}
	w.redo.Store(&redo)
	if underLock != nil {
		underLock(redo)
	}
	return redo, nil
}

// Recycle quita los segmentos anteriores al de keep: renombra a números futuros hasta Options.Spares y borra el
// resto. Un repuesto conserva registros viejos con crc válido, pero sus lsn son menores que los nuevos: Scan los
// toma por el final del WAL. Quien recorra el WAL a la vez (WalSince) no debe pedir segmentos anteriores a keep.
func (w *WAL) Recycle(keep Position) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	if err := w.writable(); err != nil {
		return err
	}
	if keep.Segment > w.active.num {
		return fmt.Errorf(MSG_KEEP_AHEAD, keep.Segment, w.active.num)
	}
	nums, err := listSegments(w.fs, w.dir)
	if err != nil {
		return err
	}
	limit := keep.Segment
	for num := range w.holds {
		limit = min(limit, num)
	}
	changed := false
	for _, num := range nums {
		if num >= limit {
			break
		}
		if err := w.recycleOne(num); err != nil {
			return err
		}
		changed = true
	}
	for num := range w.sizes { // también las anotaciones de SegmentInfo de segmentos ya reciclados
		if num < limit {
			delete(w.sizes, num)
		}
	}
	if !changed {
		return nil
	}
	if err := w.fs.SyncDir(w.dir); err != nil {
		w.fail(err)
		return failedError(err)
	}
	return nil
}

// imageLSN es el lsn contra el que se decide la imagen de página entera: una página con pd_lsn menor la necesita en
// su próximo cambio. Es el punto de rehacer vigente o, en modo respaldo, el inicio del respaldo si es mayor. Con
// writeMu tomado.
func (w *WAL) imageLSN() page.LSN {
	lsn := w.RedoPoint().LSN
	if w.forcing > 0 {
		lsn = max(lsn, w.forceFrom)
	}
	return lsn
}

// ForceImages empieza el modo respaldo (como forcePageWrites de PostgreSQL): hasta llamar a release, la primera
// modificación de cada página desde el lsn devuelto lleva su imagen entera, aunque ya la haya tenido desde el último
// checkpoint. Se cuenta: varios respaldos a la vez. release se llama una vez.
func (w *WAL) ForceImages() (page.LSN, func()) {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	from := page.LSN(w.lastLSN.Load()) + 1
	w.forcing++
	w.forceFrom = max(w.forceFrom, from)
	var once sync.Once
	return from, func() {
		once.Do(func() {
			w.writeMu.Lock()
			defer w.writeMu.Unlock()
			w.forcing--
		})
	}
}

// Hold retiene los segmentos desde seg: Recycle no los renombra ni los borra hasta llamar a release (el respaldo los
// copia al final). Se cuenta: varias retenciones a la vez. release se llama una vez.
func (w *WAL) Hold(seg uint64) func() {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	if w.holds == nil {
		w.holds = map[uint64]int{}
	}
	w.holds[seg]++
	var once sync.Once
	return func() {
		once.Do(func() {
			w.writeMu.Lock()
			defer w.writeMu.Unlock()
			if w.holds[seg]--; w.holds[seg] <= 0 {
				delete(w.holds, seg)
			}
		})
	}
}

// Holds es cuántas retenciones (Hold) y modos respaldo (ForceImages) siguen abiertos (pruebas: el respaldo los suelta
// siempre).
func (w *WAL) Holds() int {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	n := 0
	for _, c := range w.holds {
		n += c
	}
	return n + w.forcing
}

// recycleOne renombra el segmento num al número siguiente al mayor, si faltan repuestos, o lo borra. Con writeMu
// tomado. Como se recorren de menor a mayor, los números siguen contiguos.
func (w *WAL) recycleOne(num uint64) error {
	if int(w.maxNum-w.active.num) < w.opts.Spares {
		if err := w.fs.Rename(segmentPath(w.dir, num), segmentPath(w.dir, w.maxNum+1)); err != nil {
			return err
		}
		w.maxNum++
		return nil
	}
	return w.fs.Remove(segmentPath(w.dir, num))
}

// Written es cuántos bytes de registros se reservaron desde Open, sin candados: la diferencia entre dos lecturas es
// el WAL escrito entre ellas (lo mismo que mide SizeSince, sin recorrer los segmentos).
func (w *WAL) Written() int64 {
	return w.written.Load()
}

// SizeSince es cuántos bytes de WAL hay escritos o reservados desde from hasta el final.
func (w *WAL) SizeSince(from Position) int64 {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	if from.Segment > w.active.num {
		return 0
	}
	var total int64
	for num := from.Segment; num < w.active.num; num++ {
		total += w.sizes[num].Offset
	}
	return max(total+w.active.size-from.Offset, 0)
}

// fail deja el WAL inutilizable con err (si no lo estaba) y despierta a quien espera publicación.
func (w *WAL) fail(err error) {
	w.pubMu.Lock()
	defer w.pubMu.Unlock()
	if w.failure == nil {
		w.failure = err
	}
	w.pubCond.Broadcast()
}

// Failed es el primer error de escritura o fsync, o nil.
func (w *WAL) Failed() error {
	w.pubMu.Lock()
	defer w.pubMu.Unlock()
	return w.failure
}

// failedError envuelve el error que dejó al WAL inutilizable.
func failedError(err error) error {
	return fmt.Errorf(MSG_FAILED, err)
}

// Close deja de aceptar reservas, espera las escrituras en curso, hace fsync del activo y lo cierra. Toda reserva
// hecha antes debe escribirse (Write) para que Close vuelva. Cerrar dos veces no hace nada.
func (w *WAL) Close() error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	if w.closed.Load() {
		return nil
	}
	err := w.seal(w.active)
	if err != nil {
		w.fail(err)
	}

	w.syncMu.Lock()
	defer w.syncMu.Unlock()
	w.closed.Store(true)
	if err == nil && w.Failed() == nil {
		w.durable.Store(w.published.Load())
	}
	return errors.Join(err, w.active.file.Close())
}
