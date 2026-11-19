package page

import (
	"bytes"
	"errors"
	"testing"
)

// modelItem es lo que el modelo espera de un puntero: data nil = libre.
type modelItem struct {
	data  []byte
	flags LineFlags
}

// model es la página esperada, en estructuras de Go simples.
type model struct {
	items   []modelItem
	leaked  int // bytes de ítems liberados con SetUnused que esperan a Repair
	hasFree bool
	seq     byte
}

// free es el hueco que el modelo espera entre pd_lower y pd_upper.
func (m *model) free(special int) int {
	used := HeaderSize + ItemIDSize*len(m.items) + m.leaked
	for _, it := range m.items {
		used += Align(len(it.data))
	}
	return special - used
}

// firstUnused es el primer número libre del modelo, 0 si no hay.
func (m *model) firstUnused() int {
	for i, it := range m.items {
		if it.data == nil {
			return i + 1
		}
	}
	return 0
}

// valid dice si off es un número de ítem que existe.
func (m *model) valid(off int) bool { return off >= 1 && off <= len(m.items) }

// newItem arma un ítem de n bytes reconocible por su número de secuencia.
func (m *model) newItem(n int) []byte {
	m.seq++
	b := bytes.Repeat([]byte{m.seq}, n)
	b[0] = byte(n)
	return b
}

// checkErr compara si la operación debía andar con lo que pasó.
func checkErr(t testing.TB, op string, want bool, err error) {
	t.Helper()
	if want && err != nil {
		t.Fatalf("%s: unexpected error %v", op, err)
	}
	if !want && err == nil {
		t.Fatalf("%s: expected an error", op)
	}
}

/**
* runOps aplica a una página nueva las operaciones codificadas en ops (de a 4 bytes: operación, número y largo) y
* después de cada una la compara con el modelo.
* @param t testing.TB
* @param ops []byte
**/
func runOps(t testing.TB, ops []byte) {
	p := make(Page, Size)
	if err := p.Init(OpaqueSize); err != nil {
		t.Fatal(err)
	}
	special := p.Special()
	m := &model{}
	for len(ops) >= 4 {
		op, arg, n := ops[0]%9, int(ops[1]), 1+(int(ops[2])<<8|int(ops[3]))%2100
		ops = ops[4:]
		off := arg % (len(m.items) + 2) // incluye 0 y MaxOffset + 1, que no son ítems
		free := m.free(special)
		switch op {
		case 0: // AddItem
			item := m.newItem(n)
			reuse := 0
			if m.hasFree {
				if reuse = m.firstUnused(); reuse == 0 {
					m.hasFree = false
				}
			}
			need := Align(n)
			if reuse == 0 {
				need += ItemIDSize
			}
			got, err := p.AddItem(item)
			checkErr(t, "AddItem", need <= free, err)
			if err == nil {
				if reuse == 0 {
					m.items = append(m.items, modelItem{item, LineNormal})
					reuse = len(m.items)
				} else {
					m.items[reuse-1] = modelItem{item, LineNormal}
				}
				if int(got) != reuse {
					t.Fatalf("AddItem: got offset %d, want %d", got, reuse)
				}
			}
		case 1: // InsertItem
			item := m.newItem(n)
			ok := off >= 1 && off <= len(m.items)+1 && Align(n)+ItemIDSize <= free
			err := p.InsertItem(item, OffsetNumber(off))
			checkErr(t, "InsertItem", ok, err)
			if err == nil {
				m.items = append(m.items[:off-1], append([]modelItem{{item, LineNormal}}, m.items[off-1:]...)...)
			}
		case 2: // ReplaceItem
			item := m.newItem(n)
			ok := m.valid(off) && m.items[off-1].data != nil && Align(n)-Align(len(m.items[off-1].data)) <= free
			err := p.ReplaceItem(OffsetNumber(off), item)
			checkErr(t, "ReplaceItem", ok, err)
			if err == nil {
				m.items[off-1].data = item
			}
		case 3, 4: // DeleteItem, más seguido para que la página no se llene
			err := p.DeleteItem(OffsetNumber(off))
			checkErr(t, "DeleteItem", m.valid(off), err)
			if err == nil {
				m.items = append(m.items[:off-1], m.items[off:]...)
			}
		case 5: // MarkDead
			err := p.MarkDead(OffsetNumber(off))
			checkErr(t, "MarkDead", m.valid(off) && m.items[off-1].data != nil, err)
			if err == nil {
				m.items[off-1].flags = LineDead
			}
		case 6: // SetUnused
			err := p.SetUnused(OffsetNumber(off))
			checkErr(t, "SetUnused", m.valid(off), err)
			if err == nil {
				m.leaked += Align(len(m.items[off-1].data))
				m.items[off-1] = modelItem{}
				m.hasFree = true
			}
		case 7: // Repair
			checkErr(t, "Repair", true, p.Repair())
			m.leaked = 0
			m.hasFree = m.firstUnused() != 0
		case 8: // DeleteItems: dos números, o uno repetido (desordenado)
			if len(m.items) == 0 {
				continue
			}
			a := 1 + arg%len(m.items)
			b := len(m.items)
			err := p.DeleteItems([]OffsetNumber{OffsetNumber(a), OffsetNumber(b)})
			if a == b {
				if !errors.Is(err, ErrUnsorted) {
					t.Fatalf("DeleteItems: want ErrUnsorted, got %v", err)
				}
				continue
			}
			checkErr(t, "DeleteItems", true, err)
			m.items = append(m.items[:b-1], m.items[b:]...)
			m.items = append(m.items[:a-1], m.items[a:]...)
		}
		comparePage(t, p, m, special)
	}
	// La página final pasa Verify con su suma y no pasa en otro bloque.
	p.SetChecksum(7)
	if err := Verify(p, 7); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if err := Verify(p, 8); !errors.Is(err, ErrChecksum) {
		t.Fatalf("Verify on another block: %v", err)
	}
}

/**
* comparePage comprueba que la página y el modelo coinciden: punteros, contenido, espacio libre y marcas.
* @param t testing.TB, p Page, m *model, special int
**/
func comparePage(t testing.TB, p Page, m *model, special int) {
	t.Helper()
	if err := p.CheckItems(); err != nil {
		t.Fatalf("CheckItems: %v", err)
	}
	if int(p.MaxOffset()) != len(m.items) {
		t.Fatalf("MaxOffset %d, want %d", p.MaxOffset(), len(m.items))
	}
	if got, want := p.ExactFreeSpace(), m.free(special); got != want {
		t.Fatalf("ExactFreeSpace %d, want %d", got, want)
	}
	if got := p.Flags()&FlagHasFreeLines != 0; got != m.hasFree {
		t.Fatalf("FlagHasFreeLines %v, want %v", got, m.hasFree)
	}
	for i, it := range m.items {
		off := OffsetNumber(i + 1)
		id, _ := p.ItemID(off)
		data, err := p.Item(off)
		if it.data == nil {
			if id != 0 || !errors.Is(err, ErrNoItem) {
				t.Fatalf("item %d: want unused, got %#x %v", off, id, err)
			}
			continue
		}
		if err != nil || id.Flags() != it.flags || !bytes.Equal(data, it.data) {
			t.Fatalf("item %d: flags %d err %v, want flags %d (len %d vs %d)", off, id.Flags(), err, it.flags, len(data), len(it.data))
		}
	}
}
