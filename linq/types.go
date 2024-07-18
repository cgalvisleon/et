package linq

import (
	"time"

	"github.com/cgalvisleon/et/js"
)

type TypeData int

const (
	TpKey TypeData = iota
	TpText
	TpMemo
	TpInteger
	TpNumber
	TpSelect
	TpMultiSelect
	TpStatus
	TpDate // Date & Time
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
	TpData
	TpJson
	TpArray
	TpSerie
	TpCode
)

func (t TypeData) String() string {
	switch t {
	case TpKey:
		return "Key"
	case TpText:
		return "Text"
	case TpMemo:
		return "Memo"
	case TpInteger:
		return "Integer"
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
	case TpData:
		return "Data"
	case TpJson:
		return "Json"
	case TpArray:
		return "Array"
	case TpSerie:
		return "Serie"
	case TpCode:
		return "Code"
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
	case TpInteger:
		return 0
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
		return js.Json{
			"_id":  "-1",
			"name": "",
		}
	case TpFile:
		return js.Json{}
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
		return js.Json{
			"_id":  "-1",
			"name": "",
		}
	case TpLastEditedTime:
		return time.Now()
	case TpLastEditedBy:
		return js.Json{
			"_id":  "-1",
			"name": "",
		}
	case TpProject:
		return ""
	case TpData:
		return js.Json{}
	case TpJson:
		return js.Json{}
	case TpArray:
		return []js.Json{}
	case TpSerie:
		return 0
	case TpCode:
		return "000000"
	default:
		return ""
	}
}

func (t TypeData) Indexed() bool {
	switch t {
	case TpKey, TpSelect, TpMultiSelect, TpStatus, TpDate, TpPerson, TpCheckbox, TpURL, TpEmail, TpPhone, TpRelation, TpRollup, TpCreatedTime, TpCreatedBy, TpLastEditedTime, TpLastEditedBy, TpProject, TpSerie, TpCode:
		return true
	default:
		return false
	}
}

func (t TypeData) Mutate(val interface{}) {
	switch val.(type) {
	case int, int8, int16, int32, int64:
		t = TpInteger
	case float32, float64:
		t = TpNumber
	case bool:
		t = TpCheckbox
	case js.Json:
		t = TpJson
	case *js.Json:
		t = TpJson
	case []js.Json:
		t = TpArray
	case []*js.Json:
		t = TpArray
	case time.Time:
		t = TpDate
	default:
		t = TpText
	}
}

func (t TypeData) Describe() *js.Json {
	switch t {
	case TpKey:
		return &js.Json{
			"default": t.Default(),
		}
	case TpText:
		return &js.Json{
			"default": t.Default(),
			"max":     250,
		}
	case TpMemo:
		return &js.Json{
			"default": t.Default(),
			"max":     0,
		}
	case TpInteger:
		return &js.Json{
			"default": t.Default(),
			"min":     0,
			"max":     0,
		}
	case TpNumber:
		return &js.Json{
			"default": t.Default(),
			"format":  "number",
			"min":     0,
			"max":     0,
		}
	case TpSelect: //check
		return &js.Json{
			"default": t.Default(),
			"options": []js.Json{},
			"sort":    false,
		}
	case TpMultiSelect: //Check
		return &js.Json{
			"default": t.Default(),
			"options": []js.Json{},
			"sort":    false,
		}
	case TpStatus: //Type
		return &js.Json{
			"default": t.Default(),
			"options": []js.Json{},
			"sort":    false,
		}
	case TpDate:
		return &js.Json{
			"default":     t.Default(),
			"format_data": "date time",
			"time_zone":   "12_hour",
		}
	case TpPerson:
		return &js.Json{
			"default": t.Default(),
		}
	case TpFile:
		return &js.Json{
			"default": t.Default(),
		}
	case TpCheckbox:
		return &js.Json{
			"default": t.Default(),
		}
	case TpURL:
		return &js.Json{
			"default":       t.Default(),
			"show_full_url": false,
		}
	case TpEmail:
		return &js.Json{
			"default": t.Default(),
		}
	case TpPhone:
		return &js.Json{
			"default": t.Default(),
		}
	case TpFormula:
		return &js.Json{
			"default":       t.Default(),
			"formula":       "",
			"number_format": "number",
			"show_as":       "number",
		}
	case TpFunction:
		return &js.Json{
			"default":  t.Default(),
			"function": "",
		}
	case TpRelation:
		return &js.Json{
			"default":             t.Default(),
			"related_to":          "",
			"limit":               0,
			"show_on_actividades": false,
		}
	case TpRollup:
		return &js.Json{
			"default":    "",
			"related_to": "",
			"property":   "",
			"calculate":  "",
		}
	case TpCreatedTime:
		return &js.Json{
			"default":     "",
			"format_data": "date",
			"time_zone":   "12_hour",
		}
	case TpCreatedBy:
		return &js.Json{
			"default": "",
		}
	case TpLastEditedTime:
		return &js.Json{
			"default":     "",
			"format_data": "date",
			"time_zone":   "12_hour",
		}
	case TpLastEditedBy:
		return &js.Json{
			"default": "",
		}
	case TpProject:
		return &js.Json{
			"default": "",
		}
	case TpData:
		return &js.Json{
			"default": js.Json{},
		}
	case TpJson:
		return &js.Json{
			"default": js.Json{},
		}
	case TpArray:
		return &js.Json{
			"default": []js.Json{},
			"limit":   0,
		}
	case TpSerie:
		return &js.Json{
			"default": 0,
		}
	case TpCode:
		return &js.Json{
			"default": "000000",
			"format":  "%06d",
		}
	}
	return &js.Json{
		"default": "",
	}
}
