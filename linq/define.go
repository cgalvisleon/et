package linq

import (
	"github.com/cgalvisleon/et/strs"
)

// DefineColumn define a column in the model
func (m *Model) DefineColum(name, description string, typeData TypeData, _default interface{}) *Column {
	return newColumn(m, name, description, TpColumn, typeData, _default)
}

// DefineAtrib define a atrib in the model
func (m *Model) DefineAtrib(name, description string, typeData TypeData, _default interface{}) *Column {
	source := COlumn(m, SourceField.Low())
	if source == nil {
		source = m.DefineColum(SourceField.Low(), "Source field", TpJson, TpJson.Default())
	}

	result := newColumn(m, name, description, TpAtrib, typeData, _default)

	return result
}

// Define a detail collumn to the model
func (m *Model) DefineDetail(name, description string, _default interface{}, funcDetail FuncDetail) *Column {
	result := newColumn(m, name, description, TpDetail, TpFunction, _default)
	result.FuncDetail = funcDetail

	m.Details = append(m.Details, result)

	return result
}

// Define reference to object in the model
func (m *Model) DefineRelation(name string, foreignKey []string, parentModel *Model, parentKey []string, _select []string, tpCalculate TpCaculate) *Column {
	result := newColumn(m, name, "", TpDetail, TpRelation, TpRelation.Default())
	if result == nil {
		return nil
	}

	result.RelationTo = &Relation{
		ForeignKey: foreignKey,
		Parent:     parentModel,
		ParentKey:  parentKey,
		Select:     _select,
		Calculate:  tpCalculate,
		Limit:      tpCalculate.Limit(),
	}

	m.DefineForeignKey(foreignKey, parentModel, parentKey)
	m.RelationTo = append(m.RelationTo, result)

	return result
}

// Define reference to object in the model
func (m *Model) DefineRollup(name string, foreignKey []string, parentModel *Model, parentKey []string, _select string) *Column {
	result := newColumn(m, name, "", TpDetail, TpRollup, TpRollup.Default())
	if result == nil {
		return nil
	}

	result.RelationTo = &Relation{
		ForeignKey: foreignKey,
		Parent:     parentModel,
		ParentKey:  parentKey,
		Select:     []string{_select},
		Calculate:  TpUniqueValue,
		Limit:      1,
	}

	m.DefineForeignKey(foreignKey, parentModel, parentKey)
	m.RelationTo = append(m.RelationTo, result)

	return result
}

// Define index in the model
func (m *Model) DefineIndex(name string, asc bool) *Model {
	m.AddIndex(name, asc)

	return m
}

// Define unique index in the model
func (m *Model) DefineUnique(name string, asc bool) *Model {
	m.AddUnique(name, asc)

	return m
}

// Define hidden columns in the model
func (m *Model) DefineHidden(cols []string) *Model {
	for _, v := range cols {
		col := COlumn(m, v)
		if col != nil {
			col.SetHidden(true)
		}
	}

	return m
}

// Define required columns in the model
func (m *Model) DefineRequired(col string, msg string) *Model {
	column := COlumn(m, col)
	if column != nil {
		column.SetRequired(true, msg)
	}

	return m
}

// Define primary key in the model
func (m *Model) DefinePrimaryKey(cols []string) *Model {
	for _, v := range cols {
		m.AddPrimaryKey(v)
	}

	return m
}

// Define foreign key in the model
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

// Define a sql column to the model
func (m *Model) DefineFormula(name, formula string) *Column {
	result := newColumn(m, name, "", TpDetail, TpFormula, *TpFormula.Definition())
	result.Formula = formula

	return result
}

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
