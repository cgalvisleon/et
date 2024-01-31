package jfdb

import (
	"time"

	"github.com/cgalvisleon/et/et"
)

type Record struct {
	Collection   *Collection
	Created_date time.Time
	Update_date  time.Time
	Id           string
	Data         et.Json
	Index        Number
}
