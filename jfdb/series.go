package jfdb

import "time"

type Series struct {
	Schema       *Schema
	Created_date time.Time
	Update_date  time.Time
	Tag          string
	Description  string
	Index        Number
}
