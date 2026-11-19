package buffer

import (
	"cmp"
	"fmt"
	"slices"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

/**
* flush escribe el búfer al disco si está sucio (FlushBuffer): bajo d.io, borra justDirtied, copia la página bajo
* bloqueo compartido, espera al WAL durable hasta el pd_lsn de la copia, calcula la suma sobre la copia, la escribe
* y recién entonces lo marca limpio, salvo que alguien lo haya vuelto a ensuciar mientras tanto (justDirtied: un
* cambio o una marca de visibilidad posterior a la copia). Mientras dura la escritura el búfer sigue sucio: así
* FlushAll lo anota y espera a que termine (d.io) en vez de saltearlo. Un fallo deja la caché inutilizable.
* @param d *desc: fijado por quien llama
* @return error
**/
func (p *pool) flush(d *desc) error {
	d.io.Lock()
	defer d.io.Unlock()
	if err := p.failed.Load(); err != nil {
		return *err // no se escribe más nada tras un fallo
	}

	d.mu.Lock()
	if d.state != stateValid || !d.dirty {
		d.mu.Unlock()
		return nil
	}
	d.justDirtied = false
	tag := d.tag
	d.mu.Unlock()

	buf := p.scratch.Get().(page.Page)
	defer p.scratch.Put(buf)
	d.content.RLock()
	copyPage(buf, d.data)
	d.content.RUnlock()

	if err := p.write(tag, buf); err != nil {
		return p.fail(err)
	}
	p.written.Add(1)

	d.mu.Lock()
	if !d.justDirtied {
		d.dirty = false
		p.dirty.Add(-1)
	}
	d.mu.Unlock()
	return nil
}

/**
* write escribe la copia de una página cumpliendo «WAL antes que la página»: espera a que el WAL sea durable hasta su
* pd_lsn, pone la suma (salvo a una página en ceros, que se guarda en ceros) y la pasa a smgr.
* @param tag Tag, buf page.Page: la copia
* @return error
**/
func (p *pool) write(tag Tag, buf page.Page) error {
	if lsn := buf.LSN(); lsn > p.flusher.Durable() {
		if err := p.flusher.WaitDurable(lsn); err != nil {
			return fmt.Errorf("buffer: esperar el WAL hasta %d para %s: %w", lsn, tag, err)
		}
	}
	if !buf.IsNew() {
		buf.SetChecksum(tag.Block)
	}
	if err := p.smgr.Write(tag.File, tag.Fork, tag.Block, buf); err != nil {
		return fmt.Errorf("buffer: escribir %s: %w", tag, err)
	}
	return nil
}

/**
* copyPage copia src en dst con cargas atómicas de 32 bits: un lector con bloqueo compartido puede estar poniendo
* marcas de visibilidad con un OR atómico sobre la página mientras tanto.
* @param dst, src page.Page: page.Size bytes, src alineada a 4
**/
func copyPage(dst, src page.Page) {
	from := unsafe.Slice((*uint32)(unsafe.Pointer(&src[0])), page.Size/4)
	to := unsafe.Slice((*uint32)(unsafe.Pointer(&dst[0])), page.Size/4)
	for i := range from {
		to[i] = atomic.LoadUint32(&from[i])
	}
}

// pending es un búfer sucio anotado por FlushAll, con el tag que tenía.
type pending struct {
	id  int32
	tag Tag
}

/**
* FlushAll escribe todas las páginas que estaban sucias al llamarlo (BufferSync), en orden de archivo y bloque, y
* llama a pace tras cada una. Una que ya cambió de página o se limpió mientras tanto ya llegó al disco; una que otro
* está escribiendo sigue sucia hasta que la escritura termina, y flush la espera (d.io). Al volver, todas están en el
* disco (falta el fsync de smgr).
* @param pace func(written, total int): reparte el vaciado en el tiempo; nil = sin pausa
* @return error: el fallo de escritura (la caché queda inutilizable)
**/
func (p *pool) FlushAll(pace func(written, total int)) error {
	if err := p.failed.Load(); err != nil {
		return *err
	}
	list := p.dirtyList()
	for i, item := range list {
		d := &p.descs[item.id]
		if p.pinIfTag(d, item.tag) {
			err := p.flush(d)
			p.unpin(d)
			if err != nil {
				return err
			}
		}
		if pace != nil {
			pace(i+1, len(list))
		}
	}
	return nil
}

/**
* dirtyList anota los búferes sucios, ordenados por tag (escrituras secuenciales por archivo).
* @return []pending
**/
func (p *pool) dirtyList() []pending {
	var list []pending
	for i := range p.descs {
		d := &p.descs[i]
		d.mu.Lock()
		if d.state == stateValid && d.dirty {
			list = append(list, pending{id: d.id, tag: d.tag})
		}
		d.mu.Unlock()
	}
	slices.SortFunc(list, func(a, b pending) int {
		return cmp.Or(cmp.Compare(a.tag.File, b.tag.File), cmp.Compare(a.tag.Fork, b.tag.Fork),
			cmp.Compare(a.tag.Block, b.tag.Block))
	})
	return list
}

/**
* pinIfTag fija el búfer si todavía tiene la página tag.
* @param d *desc, tag Tag
* @return bool
**/
func (p *pool) pinIfTag(d *desc, tag Tag) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state != stateValid || d.tag != tag {
		return false
	}
	d.pins++
	return true
}

/**
* DropFile quita de la caché las páginas del store, sin escribirlas (DropRelationBuffers): recorre todos los
* búferes. Quien llama garantiza que nadie más usa el store; una escritura en curso de una de sus páginas (escritor
* de fondo, checkpoint) se espera, y las que vengan después la verán limpia.
* @param file smgr.FileNo
**/
func (p *pool) DropFile(file smgr.FileNo) {
	for i := range p.descs {
		d := &p.descs[i]
		d.mu.Lock()
		match := d.state == stateValid && d.tag.File == file
		d.mu.Unlock()
		if !match {
			continue
		}

		d.io.Lock()
		d.mu.Lock()
		tag, drop := d.tag, d.state == stateValid && d.tag.File == file
		if drop {
			if d.dirty {
				p.dirty.Add(-1)
			}
			d.tag, d.state, d.usage, d.dirty, d.justDirtied = Tag{}, stateEmpty, 0, false, false
		}
		d.mu.Unlock()
		d.io.Unlock()
		if drop {
			p.remove(tag, d.id)
		}
	}
}

/**
* bgWriter es el escritor de fondo: cada delay hace una ronda hasta Close o hasta que la caché falla.
* @param delay time.Duration
**/
func (p *pool) bgWriter(delay time.Duration) {
	defer p.wg.Done()
	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
			if p.bgWriteRound() != nil {
				return
			}
		}
	}
}

/**
* bgWriteRound recorre una vuelta desde la manecilla y escribe, hasta BgWriterMaxPages, los búferes sucios que el
* clock-sweep reemplazaría ya (sin pins y con uso 0, como SyncOneBuffer con skip_recently_used): así quien necesita
* un búfer casi nunca espera al WAL ni al disco.
* @return error: el fallo de escritura
**/
func (p *pool) bgWriteRound() error {
	n := uint64(len(p.descs))
	start := p.hand.Load()
	written := 0
	for i := uint64(0); i < n && written < BgWriterMaxPages; i++ {
		d := &p.descs[(start+i)%n]
		if !p.pinIfReusableDirty(d) {
			continue
		}
		err := p.flush(d)
		p.unpin(d)
		if err != nil {
			return err
		}
		written++
	}
	return nil
}

/**
* pinIfReusableDirty fija el búfer si está sucio, sin pins y con uso 0.
* @param d *desc
* @return bool
**/
func (p *pool) pinIfReusableDirty(d *desc) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state != stateValid || !d.dirty || d.pins != 0 || d.usage != 0 {
		return false
	}
	d.pins++
	return true
}
