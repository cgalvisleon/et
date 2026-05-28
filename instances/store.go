package instances

import (
	"github.com/cgalvisleon/et/et"
)

type Store interface {
	Set(id, tag, ownerId string, obj any) error
	Get(id string, dest any) (bool, error)
	Delete(id string) error
	Query(query et.Json) (et.Items, error)
}
