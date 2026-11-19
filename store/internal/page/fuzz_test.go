package page

import "testing"

// FuzzVerify valida bytes cualesquiera como página leída del disco y los recorre con todos los métodos: nada entra
// en pánico, y si Verify acepta una página no nueva su suma coincide.
func FuzzVerify(f *testing.F) {
	p := make(Page, Size)
	f.Add([]byte(p), uint32(0))
	_ = p.Init(OpaqueSize)
	for _, s := range []string{"uno", "dos", "tres"} {
		_, _ = p.AddItem([]byte(s))
	}
	p.SetChecksum(5)
	f.Add([]byte(p), uint32(5))
	f.Add([]byte("corta"), uint32(1))
	f.Fuzz(func(t *testing.T, b []byte, blk uint32) {
		err := Verify(b, BlockNumber(blk))
		if len(b) != Size {
			if err == nil {
				t.Fatal("Verify accepted a page of the wrong size")
			}
			return
		}
		p := Page(b)
		if err == nil && !p.IsNew() && p.Checksum() != p.ComputeChecksum(BlockNumber(blk)) {
			t.Fatal("Verify accepted a bad checksum")
		}
		exercise(p)
	})
}

// FuzzOps aplica operaciones al azar (agregar, insertar, reemplazar, quitar, marcar, liberar, compactar) y compara
// la página con un modelo después de cada una.
func FuzzOps(f *testing.F) {
	f.Add([]byte{0, 0, 0, 10, 0, 0, 0, 20, 1, 1, 0, 5, 2, 1, 0, 40, 3, 1, 0, 0})
	f.Add([]byte{0, 0, 3, 0, 0, 0, 3, 0, 6, 1, 0, 0, 7, 0, 0, 0, 0, 0, 0, 9, 5, 2, 0, 0, 8, 0, 0, 0})
	f.Add([]byte{0, 0, 7, 255, 0, 0, 7, 255, 0, 0, 7, 255, 2, 1, 7, 255, 4, 2, 0, 0})
	f.Fuzz(func(t *testing.T, ops []byte) {
		runOps(t, ops)
	})
}
