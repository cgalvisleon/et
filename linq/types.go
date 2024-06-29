package linq

import (
	"time"

	"github.com/cgalvisleon/et/et"
)

type TypeData int

const (
	TpKey TypeData = iota
	TpText
	TpMemo
	TpNumber
	TpSelect
	TpMultiSelect
	TpStatus
	TpDate
	TpPerson
	TpFile // files & media
	TpCheckbox
	TpURL   // URL
	TpEmail // Email
	TpPhone
	TpFormula  // Formula
	TpFunction // Function
	TpRelation // Relation with other model
	TpRollup   // Rollup (Enrollar) with other model
	TpCreatedTime
	TpCreatedBy
	TpLastEditedTime
	TpLastEditedBy
	TpProject
	TpJson
	TpArray
	TpSerie
)

func (t TypeData) String() string {
	switch t {
	case TpKey:
		return "Key"
	case TpText:
		return "Text"
	case TpMemo:
		return "Memo"
	case TpNumber:
		return "Number"
	case TpSelect:
		return "Select"
	case TpMultiSelect:
		return "Multi select"
	case TpStatus:
		return "Status"
	case TpDate:
		return "Date"
	case TpPerson:
		return "Person"
	case TpFile:
		return "File"
	case TpCheckbox:
		return "Checkbox"
	case TpURL:
		return "URL"
	case TpEmail:
		return "Email"
	case TpPhone:
		return "Phone"
	case TpFormula:
		return "Formula"
	case TpFunction:
		return "Function"
	case TpRelation:
		return "Relation"
	case TpRollup:
		return "Rollup"
	case TpCreatedTime:
		return "Created time"
	case TpCreatedBy:
		return "Created by"
	case TpLastEditedTime:
		return "Last edited time"
	case TpLastEditedBy:
		return "Last edited by"
	case TpProject:
		return "Project"
	case TpJson:
		return "Json"
	case TpArray:
		return "Array"
	case TpSerie:
		return "Serie"
	default:
		return "Unknown"
	}
}

func (t TypeData) Default() interface{} {
	switch t {
	case TpKey:
		return "-1"
	case TpText:
		return ""
	case TpMemo:
		return ""
	case TpNumber:
		return 0
	case TpSelect:
		return ""
	case TpMultiSelect:
		return ""
	case TpStatus:
		return "0"
	case TpDate:
		return ""
	case TpPerson:
		return et.Json{
			"_id":  "-1",
			"name": "",
		}
	case TpFile:
		return et.Json{}
	case TpCheckbox:
		return false
	case TpURL:
		return ""
	case TpEmail:
		return ""
	case TpPhone:
		return ""
	case TpFormula:
		return ""
	case TpFunction:
		return ""
	case TpRelation:
		return ""
	case TpRollup:
		return ""
	case TpCreatedTime:
		return time.Now()
	case TpCreatedBy:
		return et.Json{
			"_id":  "-1",
			"name": "",
		}
	case TpLastEditedTime:
		return time.Now()
	case TpLastEditedBy:
		return et.Json{
			"_id":  "-1",
			"name": "",
		}
	case TpProject:
		return ""
	case TpJson:
		return et.Json{}
	case TpArray:
		return []et.Json{}
	case TpSerie:
		return 0
	default:
		return ""
	}
}

func (t TypeData) Indexed() bool {
	switch t {
	case TpKey, TpSelect, TpMultiSelect, TpStatus, TpDate, TpPerson, TpCheckbox, TpURL, TpEmail, TpPhone, TpRelation, TpRollup, TpCreatedTime, TpCreatedBy, TpLastEditedTime, TpLastEditedBy, TpProject, TpSerie:
		return true
	default:
		return false
	}
}

func (t TypeData) Mutate(val interface{}) {
	switch val.(type) {
	case int, int8, int16, int32, int64, float32, float64:
		t = TpNumber
	case bool:
		t = TpCheckbox
	case et.Json:
		t = TpJson
	case *et.Json:
		t = TpJson
	case []et.Json:
		t = TpArray
	case []*et.Json:
		t = TpArray
	case time.Time:
		t = TpDate
	default:
		t = TpText
	}
}

func (t TypeData) Definition() *et.Json {
	switch t {
	case TpKey:
		return &et.Json{
			"default": t.Default(),
		}
	case TpText:
		return &et.Json{
			"default": t.Default(),
			"max":     250,
		}
	case TpMemo:
		return &et.Json{
			"default": t.Default(),
			"max":     0,
		}
	case TpNumber:
		return &et.Json{
			"default": t.Default(),
			"format":  "number",
			"min":     0,
			"max":     0,
		}
	case TpSelect: //check
		return &et.Json{
			"default": t.Default(),
			"options": []et.Json{},
			"sort":    false,
		}
	case TpMultiSelect: //Check
		return &et.Json{
			"default": t.Default(),
			"options": []et.Json{},
			"sort":    false,
		}
	case TpStatus: //Type
		return &et.Json{
			"default": t.Default(),
			"options": []et.Json{},
			"sort":    false,
		}
	case TpDate:
		return &et.Json{
			"default":     t.Default(),
			"format_data": "date time",
			"time_zone":   "12_hour",
		}
	case TpPerson:
		return &et.Json{
			"default": t.Default(),
		}
	case TpFile:
		return &et.Json{
			"default": t.Default(),
		}
	case TpCheckbox:
		return &et.Json{
			"default": t.Default(),
		}
	case TpURL:
		return &et.Json{
			"default":       t.Default(),
			"show_full_url": false,
		}
	case TpEmail:
		return &et.Json{
			"default": t.Default(),
		}
	case TpPhone:
		return &et.Json{
			"default": t.Default(),
		}
	case TpFormula:
		return &et.Json{
			"default":       t.Default(),
			"formula":       "",
			"number_format": "number",
			"show_as":       "number",
		}
	case TpFunction:
		return &et.Json{
			"default":  t.Default(),
			"function": "",
		}
	case TpRelation:
		return &et.Json{
			"default":             t.Default(),
			"related_to":          "",
			"limit":               0,
			"show_on_actividades": false,
		}
	case TpRollup:
		return &et.Json{
			"default":    "",
			"related_to": "",
			"property":   "",
			"calculate":  "",
		}
	case TpCreatedTime:
		return &et.Json{
			"default":     "",
			"format_data": "date",
			"time_zone":   "12_hour",
		}
	case TpCreatedBy:
		return &et.Json{
			"default": "",
		}
	case TpLastEditedTime:
		return &et.Json{
			"default":     "",
			"format_data": "date",
			"time_zone":   "12_hour",
		}
	case TpLastEditedBy:
		return &et.Json{
			"default": "",
		}
	case TpProject:
		return &et.Json{
			"default": "",
		}
	case TpJson:
		return &et.Json{
			"default": et.Json{},
		}
	case TpArray:
		return &et.Json{
			"default": []et.Json{},
			"limit":   0,
		}
	case TpSerie:
		return &et.Json{
			"default": 0,
		}
	}
	return &et.Json{
		"default": "",
	}
}
