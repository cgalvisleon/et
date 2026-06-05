package stores

import "github.com/cgalvisleon/et/et"

type Store interface {
	Set(id, tag, tenantId, ownerId string, obj any, userId string) error
	Get(id string, dest any) (bool, error)
	Delete(id string) error
	Query(query et.Json) (et.Items, error)
}
