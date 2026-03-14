package resilience

import "github.com/cgalvisleon/et/logs"

type LoadInstanceFn func(id string) (*Instance, error)
type SaveInstanceFn func(*Instance) error

var loadInstance LoadInstanceFn
var saveInstance SaveInstanceFn

func SetLoadInstance(fn LoadInstanceFn) {
	if fn == nil {
		return
	}

	logs.Log("workflow", "SetLoadInstance")
	loadInstance = fn
}

func SetSaveInstance(fn SaveInstanceFn) {
	if fn == nil {
		return
	}

	logs.Log("workflow", "SetSaveInstance")
	saveInstance = fn
}
