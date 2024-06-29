package linq

type COl struct {
	Name        string
	Description string
	TypeData    TypeData
	Default     interface{}
}

type ColFkey struct {
	ForeignKey  []string
	ParentModel *Model
	ParentKey   []string
}

type ColRol struct {
	Name       string
	ForeignKey []string
	Parent     *Model
	ParentKey  []string
	Select     []string
	Calculate  TpCaculate
}

type ColDetail struct {
	Name        string
	Description string
	Default     any
	FuncDetail  FuncDetail
}

type ColRequired struct {
	Name    string
	Message string
}

type ColFormula struct {
	Name    string
	Formula string
}

type TRigger struct {
	TypeTrigger TypeTrigger
	Trigger     Trigger
}

type Definition struct {
	Schema      string
	Name        string
	Description string
	Version     int
	Columns     []COl
	Atribs      []COl
	Indexes     []string
	Uniques     []string
	Hidden      []string
	Required    []ColRequired
	PrimaryKey  []string
	ForeignKey  []ColFkey
	Relation    []ColRol
	Rollup      []ColRol
	Details     []ColDetail
	Formulas    []ColFormula
	Trigger     []TRigger
}

func MOdel(def *Definition) *Model {
	schema := NewSchema(def.Schema, "")
	result := NewModel(schema, def.Name, def.Description, def.Version)
	for _, col := range def.Columns {
		result.DefineColum(col.Name, col.Description, col.TypeData, col.Default)
	}
	for _, col := range def.Atribs {
		result.DefineAtrib(col.Name, col.Description, col.TypeData, col.Default)
	}
	for _, idx := range def.Indexes {
		result.DefineIndex(idx, true)
	}
	for _, uni := range def.Uniques {
		result.DefineUnique(uni, true)
	}
	result.DefineHidden(def.Hidden)
	for _, req := range def.Required {
		result.DefineRequired(req.Name, req.Message)
	}
	result.DefinePrimaryKey(def.PrimaryKey)
	for _, fk := range def.ForeignKey {
		result.DefineForeignKey(fk.ForeignKey, fk.ParentModel, fk.ParentKey)
	}
	for _, ref := range def.Rollup {
		result.DefineRollup(ref.Name, ref.ForeignKey, ref.Parent, ref.ParentKey, ref.Select[0])
	}
	for _, ref := range def.Relation {
		result.DefineRelation(ref.Name, ref.ForeignKey, ref.Parent, ref.ParentKey, ref.Select, ref.Calculate)
	}
	for _, det := range def.Details {
		result.DefineDetail(det.Name, det.Description, det.Default, det.FuncDetail)
	}
	for _, frm := range def.Formulas {
		result.DefineFormula(frm.Name, frm.Formula)
	}
	for _, trig := range def.Trigger {
		result.DefineTrigger(trig.TypeTrigger, trig.Trigger)
	}

	return result
}
