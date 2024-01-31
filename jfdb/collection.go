package jfdb

import "time"

type Collection struct {
	Schema       *Schema
	Created_date time.Time
	Update_date  time.Time
	Id           string
	Name         string
	Description  string
	Filename     string
	Indices      []*Index
	Data         []*Record
	Index        Number
}
