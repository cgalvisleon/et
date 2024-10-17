package linq

import (
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
)

type TypeShape int

const (
	ShapePoint TypeShape = iota
	ShapeLine
	ShapePolygon
	ShapeMultiPart
)

func (t TypeShape) String() string {
	switch t {
	case ShapePoint:
		return "Point"
	case ShapeLine:
		return "Line"
	case ShapePolygon:
		return "Polygon"
	case ShapeMultiPart:
		return "MultiPart"
	}
	return "Unknown"
}

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
	TpRollup   // Rollup (Enrollar) with other model
	TpCreatedTime
	TpCreatedBy
	TpLastEditedTime
	TpLastEditedBy
	TpProject
	TpSource
	TpJson
	TpArray
	TpSerie
	TpCode
	TpShape
	TpUUIndex
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
	case TpSource:
		return "Source"
	case TpJson:
		return "Json"
	case TpArray:
		return "Array"
	case TpSerie:
		return "Serie"
	case TpCode:
		return "Code"
	case TpShape:
		return "Shape"
	case TpUUIndex:
		return "Unix time index"
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
	case TpRollup:
		return ""
	case TpCreatedTime:
		return timezone.NowTime()
	case TpCreatedBy:
		return et.Json{
			"_id":  "-1",
			"name": "",
		}
	case TpLastEditedTime:
		return timezone.NowTime()
	case TpLastEditedBy:
		return et.Json{
			"_id":  "-1",
			"name": "",
		}
	case TpProject:
		return ""
	case TpSource:
		return et.Json{}
	case TpJson:
		return et.Json{}
	case TpArray:
		return []et.Json{}
	case TpSerie:
		return 0
	case TpCode:
		return "000000"
	case TpShape:
		return et.Json{
			"tp":      ShapePoint,
			"lat":     0,
			"lng":     0,
			"located": false,
		}
	case TpUUIndex:
		return 0
	default:
		return ""
	}
}

func (t TypeData) Indexed() bool {
	switch t {
	case TpKey, TpSelect, TpMultiSelect, TpStatus, TpDate, TpPerson, TpCheckbox, TpURL, TpEmail, TpPhone, TpRollup, TpCreatedTime, TpCreatedBy, TpLastEditedTime, TpLastEditedBy, TpProject, TpSerie, TpCode, TpShape, TpUUIndex:
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

func (t TypeData) Describe() *et.Json {
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
	case TpInteger:
		return &et.Json{
			"default": t.Default(),
			"min":     0,
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
	case TpSource:
		return &et.Json{
			"default": et.Json{},
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
	case TpCode:
		return &et.Json{
			"default": "000000",
			"format":  "%06d",
		}
	case TpShape:
		return &et.Json{
			"default": et.Json{
				"tp":      ShapePoint,
				"lat":     0,
				"lng":     0,
				"located": false,
			},
		}
	case TpUUIndex:
		return &et.Json{
			"default": 0,
		}
	default:
		return &et.Json{
			"default": "",
		}
	}
}
