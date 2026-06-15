package jsql

import "github.com/cgalvisleon/et/jrex"

func wrapper(jrex *jrex.Jrex) {
	jrex.Set("newTx", NewTx)
}
