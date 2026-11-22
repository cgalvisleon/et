package btree

import (
	"testing"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// TestVacuumConflictAffects: una versión quitada afecta solo a los snapshots que la ven sin ver una más nueva del mismo
// id; una sin xid (o congelada) y sin más nuevas, a todos; un congelado, a los que no veían su xid (tarea 125).
func TestVacuumConflictAffects(t *testing.T) {
	snap := func(xmax page.XID) *xact.Snapshot { return &xact.Snapshot{Xmin: xmax, Xmax: xmax} }
	cases := []struct {
		name string
		c    VacuumConflict
		xmax page.XID
		want bool
	}{
		{"no ve la quitada", VacuumConflict{removed: []removedVersion{{xid: 10, newer: 20}}}, 5, false},
		{"ve la quitada y no la nueva", VacuumConflict{removed: []removedVersion{{xid: 10, newer: 20}}}, 15, true},
		{"ve la nueva", VacuumConflict{removed: []removedVersion{{xid: 10, newer: 20}}}, 25, false},
		{"sin nueva en la hoja", VacuumConflict{removed: []removedVersion{{xid: 10}}}, 25, true},
		{"la ven todos", VacuumConflict{removed: []removedVersion{{}}}, 1, true},
		{"congela lo que no veía", VacuumConflict{frozen: 30}, 25, true},
		{"congela lo que ya veía", VacuumConflict{frozen: 30}, 31, false},
		{"nada", VacuumConflict{}, 1, false},
	}
	for _, c := range cases {
		if got := c.c.Affects(snap(c.xmax)); got != c.want {
			t.Errorf("%s (Xmax %d): %v, se esperaba %v", c.name, c.xmax, got, c.want)
		}
	}
}
