package router

import "github.com/cgalvisleon/et/et"

type SetFn func(id string, data et.Json) error
type GetFn func(id string) (et.Json, error)
type DeleteFn func(id string) error

var (
	setFn    SetFn
	getFn    GetFn
	deleteFn DeleteFn
)

/**
* OnSet
* @params f SetFn
**/
func OnSet(f SetFn) {
	setFn = f
}

/**
* OnGet
* @params f GetFn
**/
func OnGet(f GetFn) {
	getFn = f
}

/**
* OnDelete
* @params f DeleteFn
**/
func OnDelete(f DeleteFn) {
	deleteFn = f
}
