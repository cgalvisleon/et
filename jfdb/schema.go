package jfdb

import (
	"time"

	"github.com/cgalvisleon/et/et"
)

type Schema struct {
	Database     *Database
	Created_date time.Time
	Update_date  time.Time
	Id           string
	Name         string
	Description  string
	Data         et.Json
	Collections  []*Collection
}
