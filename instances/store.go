package instances

import "github.com/cgalvisleon/et/et"

type Store interface {
	Set(id, tag string, obj any) error
	Get(id string, dest any) (bool, error)
	Delete(id string) error
	Query(query et.Json) (et.Items, error)
}

type GetInstanceFn func(id string, dest any) (bool, error)
type SetInstanceFn func(id, tag string, obj any) error
type DeleteInstanceFn func(id string) error
type QueryInstanceFn func(query et.Json) (et.Items, error)
