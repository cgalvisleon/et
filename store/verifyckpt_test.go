package store

import (
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// catalogFullFS hace fallar con disco lleno la creación de catalog.tmp mientras full está puesto: el paso 3 del
// checkpoint no entra (disco lleno limpio) y todo lo demás sí.
type catalogFullFS struct {
	vfs.FS
	full atomic.Bool
}

func (f *catalogFullFS) OpenFile(name string, flag int) (vfs.File, error) {
	if f.full.Load() && filepath.Base(name) == control.CatalogTempName {
		return nil, vfs.ErrNoSpace
	}
	return f.FS.OpenFile(name, flag)
}

// TestVerifyAfterCutCheckpoint (tarea 95): un checkpoint que el disco lleno corta en el paso 3 no cuenta (el control
// y el catálogo guardados siguen en el anterior), aunque el WAL ya movió su punto de rehacer vigente en el paso 1. Eso
// no es una inconsistencia —el WAL decide las imágenes contra un punto posterior al del control: más imágenes, nunca
// menos— y Verify queda limpio; después de un checkpoint que termina, también, y al reabrir está todo.
func TestVerifyAfterCutCheckpoint(t *testing.T) {
	base := crashfs.New()
	fsys := &catalogFullFS{FS: base}
	e := openTest(t, fsys, testConfig())
	s := mustStore(t, e, "a")
	for i := range 50 {
		mustOK(t, s.Put(string(rune('a'+i%26))+string(rune('0'+i/26)), []byte("v")))
	}
	mustOK(t, e.Checkpoint())
	mustOK(t, s.Put("after", []byte("1")))
	fsys.full.Store(true)
	if err := e.Checkpoint(); !errors.Is(err, ErrDiskFull) {
		t.Fatalf("checkpoint with catalog.tmp full: %v, want ErrDiskFull", err)
	}
	mustVerifyClean(t, e)
	mustOK(t, s.Put("later", []byte("2")))
	fsys.full.Store(false)
	mustVerifyClean(t, e)
	mustOK(t, e.Checkpoint())
	mustVerifyClean(t, e)
	mustOK(t, e.Close())
	e = openTest(t, base, testConfig())
	defer e.Close()
	if n := mustStore(t, e, "a").Count(); n != 52 {
		t.Fatalf("after reopening, Count = %d; want 52", n)
	}
	mustVerifyClean(t, e)
}
