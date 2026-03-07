package crontab

import "github.com/cgalvisleon/et/logs"

type GetInstanceFn func(id string, dest any) (bool, error)
type SetInstanceFn func(id, tag string, obj any) error

var getInstance GetInstanceFn
var setInstance SetInstanceFn

func SetGetInstance(fn GetInstanceFn) {
	if fn == nil {
		return
	}

	logs.Log(packageName, "SetLoadInstance")
	getInstance = fn
}

func SetSetInstance(fn SetInstanceFn) {
	if fn == nil {
		return
	}

	logs.Log(packageName, "SetSaveInstance")
	setInstance = fn
}
