package linq

import (
	"github.com/cgalvisleon/et/strs"
)

/**
 * Define column define a column in the model
 * @param name string
 * @param description string
 * @param typeData TypeData
 * @param _default interface{}
 * @return *Column
**/
func (m *Model) DefineColumn(name, description string, typeData TypeData, _default interface{}) *Column {
	return newColumn(m, name, description, TpColumn, typeData, _default)
}

/**
 * Define atrib define a atrib in the model
 * @param name string
 * @param description string
 * @param typeData TypeData
 * @param _default interface{}
 * @return *Column
**/
func (m *Model) DefineAtrib(name, description string, typeData TypeData, _default interface{}) *Column {
	source := Col(m, SourceField.Low())
	if source == nil {
		_ = m.DefineColumn(SourceField.Low(), "Source field", TpSource, TpJson.Default())
	}

	result := newColumn(m, name, description, TpAtrib, typeData, _default)

	return result
}

/**
 * Define detail define a detail in the model
 * @param name string
 * @param description string
 * @param _default interface{}
 * @return *Column
**/
func (m *Model) DefineDetail(name, description string, _default interface{}, funcDetail FuncDetail) *Column {
	result := newColumn(m, name, description, TpDetail, TpFunction, _default)
	result.FuncDetail = funcDetail

	m.Details = append(m.Details, result)

	return result
}

/**
 * Define rollup in the model
 * @param name string
 * @param foreignKey []string
 * @param parentModel *Model
 * @param parentKey []string
 * @param _select string
 * @return *Column
**/
func (m *Model) DefineRollup(name string, foreignKey []string, parentModel *Model, parentKey []string, _select []string) *Column {
	result := newColumn(m, name, "", TpOther, TpRollup, TpRollup.Default())
	if result == nil {
		return nil
	}

	result.RelationTo = &Relation{
		ForeignKey: foreignKey,
		Parent:     parentModel,
		ParentKey:  parentKey,
		Select:     _select,
		Calculate:  TpUniqueValue,
		Limit:      1,
	}

	m.DefineForeignKey(foreignKey, parentModel, parentKey)
	m.RelationTo = append(m.RelationTo, result)

	return result
}

/**
 * Define index in the model
 * @param name []string
 * @param asc bool
 * @return *Model
**/
func (m *Model) DefineIndex(name []string, asc bool) *Model {
	for _, v := range name {
		m.AddIndex(v, asc)
	}

	return m
}

/**
 * Define unique in the model
 * @param name string
 * @param asc bool
 * @return *Model
**/
func (m *Model) DefineUnique(name string, asc bool) *Model {
	m.AddUnique(name, asc)

	return m
}

/**
 * Define hidden columns in the model
 * @param cols []string
 * @return *Model
**/
func (m *Model) DefineHidden(cols []string) *Model {
	for _, v := range cols {
		col := Col(m, v)
		if col != nil {
			col.SetHidden(true)
		}
	}

	return m
}

/**
 * Define required columns in the model
 * @param col string
 * @param msg string
 * @return *Model
**/
func (m *Model) DefineRequired(cols []ColRequired) *Model {
	for _, def := range cols {
		column := Col(m, def.Name)
		if column != nil {
			column.SetRequired(true, def.Message)
		}
	}

	return m
}

/**
 * Define primary key in the model
 * @param cols []string
 * @return *Model
**/
func (m *Model) DefinePrimaryKey(cols []string) *Model {
	for _, v := range cols {
		m.AddPrimaryKey(v)
	}

	return m
}

/**
 * Define foreign key in the model
 * @param foreignKey []string
 * @param parentModel *Model
 * @param parentKey []string
 * @return *Model
**/
func (m *Model) DefineForeignKey(foreignKey []string, parentModel *Model, parentKey []string) *Model {
	for i, key := range foreignKey {
		foreignKey[i] = strs.Uppcase(key)
	}
	for i, key := range parentKey {
		parentKey[i] = strs.Uppcase(key)
	}
	m.AddForeignKey(foreignKey, parentModel, parentKey)

	return m
}

/**
 * Define formula in the model
 * @param name string
 * @param formula string
 * @return *Column
**/
func (m *Model) DefineFormula(name, formula string) *Column {
	result := newColumn(m, name, "", TpDetail, TpFormula, *TpFormula.Describe())
	result.Formula = formula

	return result
}

/**
 * Define trigger in the model
 * @param event TypeTrigger
 * @param trigger Trigger
**/
func (m *Model) DefineTrigger(event TypeTrigger, trigger Trigger) {
	switch event {
	case BeforeInsert:
		m.BeforeInsert = append(m.BeforeInsert, trigger)
	case AfterInsert:
		m.AfterInsert = append(m.AfterInsert, trigger)
	case BeforeUpdate:
		m.BeforeUpdate = append(m.BeforeUpdate, trigger)
	case AfterUpdate:
		m.AfterUpdate = append(m.AfterUpdate, trigger)
	case BeforeDelete:
		m.BeforeDelete = append(m.BeforeDelete, trigger)
	case AfterDelete:
		m.AfterDelete = append(m.AfterDelete, trigger)
	}
}

/**
 * Define integrity in the model
 * @param integrity bool
**/
func (m *Model) DefineIntegrity(integrity bool) {
	m.Integrity = integrity
}
