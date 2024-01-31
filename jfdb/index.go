package jfdb

import (
	"time"

	"github.com/cgalvisleon/et/et"
)

type Index struct {
	Collection   *Collection
	Created_date time.Time
	Update_date  time.Time
	Id           string
	Name         string
	Sorted       bool
	Atrib        string
	Filename     string
	Data         et.Json
	Index        Number
}
