package jsql

import (
	"fmt"

	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrex"
)

const (
	EVENT_JREX_SET = "jrex:set"
)

/**
* wrapper: Wraps the jrex with the model
* @param rex *jrex.Jrex, model *Model
* @return void
**/
func wrapper(rex *jrex.Jrex, model *Model) {
	rex.OnSave(func(rex *jrex.Jrex) error {
		channel := fmt.Sprintf("%s:%s", EVENT_JREX_SET, model.TenantId)
		event.Publish(channel, rex.ToJson())
		return nil
	})
	rex.Set("db", model.db)
	rex.Set("getDb", GetDb)
	rex.Set("newTx", NewTx)
}
