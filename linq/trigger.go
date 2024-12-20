package linq

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

/**
* beforeInsert set the values before insert
* @param model *Model
* @param value *Values
* @return error
**/
func beforeInsert(model *Model, value *Values) error {
	now := timezone.NowTime()
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
	if model.ColumnUUIndex != nil {
		uuindex := utility.UUIndex(model.Table)
		value.Set(model.ColumnUUIndex, uuindex)
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
	now := timezone.NowTime()
	if model.ColumnLastEditedTime != nil {
		value.Set(model.ColumnLastEditedTime, now)
	}

	return nil
}

/**
* beforeDelete set the values before update
* @param model *Model
* @param value *Values
* @return error
**/
func beforeDelete(model *Model, value *Values) error {

	return nil
}

/**
* afterInsert set the values after insert
* @param model *Model
* @param value *Values
* @return error
**/
func afterInsert(model *Model, value *Values) error {

	return nil
}

/**
* afterUpdate set the values after insert
* @param model *Model
* @param value *Values
* @return error
**/
func afterUpdate(model *Model, value *Values) error {

	return nil
}

/**
* afterDelete set the values after delete
* @param model *Model
* @param value *Values
* @return error
**/
func afterDelete(model *Model, value *Values) error {

	return nil
}
