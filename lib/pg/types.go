package lib

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/strs"
)

/**
* Return default ddl value
* @param col *linq.Column
* @return string
**/
func ddlDefault(col *linq.Column) string {
	str, ok := col.Default.(string)
	if ok {
		switch str {
		case "NOW()":
			return "DEFAULT NOW()"
		case "FALSE":
			return "DEFAULT FALSE"
		case "TRUE":
			return "DEFAULT TRUE"
		case "NULL":
			return "DEFAULT NULL"
		default:
			return strs.Format(`DEFAULT '%v'`, str)
		}
	}
	var result string
	switch col.TypeData {
	case linq.TpKey:
		result = `'-1'`
	case linq.TpText:
		result = `''`
	case linq.TpMemo:
		result = `''`
	case linq.TpInteger:
		result = `0`
	case linq.TpNumber:
		result = `0`
	case linq.TpDate:
		result = `NOW()`
	case linq.TpCheckbox:
		result = `FALSE`
	case linq.TpRollup:
		result = `''`
	case linq.TpCreatedTime:
		result = `NOW()`
	case linq.TpCreatedBy:
		result = `'{ "_id": "", "name": "" }'`
	case linq.TpLastEditedTime:
		result = `NOW()`
	case linq.TpLastEditedBy:
		result = `'{ "_id": "", "name": "" }'`
	case linq.TpStatus:
		result = `'{ "_id": "0", "main": "State", "name": "Activo" }'`
	case linq.TpPerson:
		result = `'{ "_id": "", "name": "" }'`
	case linq.TpFile:
		result = `''`
	case linq.TpURL:
		result = `''`
	case linq.TpEmail:
		result = `''`
	case linq.TpPhone:
		result = `''`
	case linq.TpFormula:
		result = `''`
	case linq.TpSelect:
		result = `''`
	case linq.TpMultiSelect:
		result = `''`
	case linq.TpSource:
		result = `'{}'`
	case linq.TpJson:
		result = `'{}'`
	case linq.TpArray:
		result = `'[]'`
	case linq.TpSerie:
		result = `0`
	case linq.TpCode:
		result = `'000000'`
	case linq.TpShape:
		result = `'{ "tp": 0, "lat": 0, "lng": 0, "located": false }'`
	default:
		val := col.Default
		result = strs.Format(`%v`, et.Quote(val))
	}

	return strs.Append("DEFAULT", result, " ")
}

/**
* Return ddl type
* @param col *linq.Column
* @return string
**/
func ddlType(col *linq.Column) string {
	switch col.TypeData {
	case linq.TpKey, linq.TpRollup, linq.TpStatus, linq.TpPhone, linq.TpSelect, linq.TpMultiSelect, linq.TpCode:
		return "VARCHAR(80)"
	case linq.TpMemo:
		return "TEXT"
	case linq.TpInteger:
		return "BIGINT"
	case linq.TpNumber:
		return "DECIMAL(18, 2)"
	case linq.TpDate:
		return "TIMESTAMP"
	case linq.TpCheckbox:
		return "BOOLEAN"
	case linq.TpCreatedTime:
		return "TIMESTAMP"
	case linq.TpCreatedBy:
		return "JSONB"
	case linq.TpLastEditedTime:
		return "TIMESTAMP"
	case linq.TpLastEditedBy:
		return "JSONB"
	case linq.TpPerson:
		return "JSONB"
	case linq.TpFile:
		return "JSONB"
	case linq.TpURL:
		return "TEXT"
	case linq.TpFormula:
		return "JSONB"
	case linq.TpSource:
		return "JSONB"
	case linq.TpJson:
		return "JSONB"
	case linq.TpArray:
		return "JSONB"
	case linq.TpSerie:
		return "BIGINT"
	case linq.TpShape:
		return "JSONB"
	default:
		return "VARCHAR(250)"
	}
}
