package linq

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
)

/**
* beforeInsert set the values before insert
* @param model *Model
* @param value *Values
* @return error
**/
func beforeInsert(model *Model, value *Values) error {
	now := utility.Now()
	if model.ColumnCreatedTime != nil {
		value.Set(model.ColumnCreatedTime, now)
	}
	if model.ColumnLastEditedTime != nil {
		value.Set(model.ColumnLastEditedTime, now)
	}
	if model.ColumnSerie != nil {
		index, err := model.DB.NextSerie(model.Table)
		if err != nil {
			return logs.Alert(err)
		}

		value.Set(model.ColumnSerie, index)
	}

	return nil
}

/**
* beforeUpdate set the values before update
* @param model *Model
* @param value *Values
* @return error
**/
func beforeUpdate(model *Model, value *Values) error {
	now := utility.Now()
	if model.ColumnLastEditedTime != nil {
		value.Set(model.ColumnLastEditedTime, now)
	}

	return nil
}
