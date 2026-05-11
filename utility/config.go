package utility

import "github.com/cgalvisleon/et/et"

type Config interface {
	GetParams(key string) (et.Json, error)
	Set(key string, params et.Json) error
}
