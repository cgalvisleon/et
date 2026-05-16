package jsql

/**
* initCore: Initializes the core schema tables required by the DB instance.
* @return error
**/
func (s *DB) initCore() error {
	err := defineCatalog(s)
	if err != nil {
		return err
	}
	return nil
}

var (
	catalog *Model
)

/**
* defineCatalog: Defines the catalog table.
* @param db *DB
* @return error
**/
func defineCatalog(db *DB) error {
	if catalog != nil {
		return nil
	}

	var err error
	catalog, err = db.Define(Def{
		Schema:  "core",
		Name:    "catalog",
		Version: 1,
		Columns: []Column{
			{Name: ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "name", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
			{Name: "version", TypeColumn: COLUMN, TypeData: INT, Default: 0},
			{Name: "kind", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "definition", TypeColumn: COLUMN, TypeData: BYTES, Default: []byte{}},
		},
		IdxField: IDX,
		PrimaryKeys: []DefIndex{
			{Name: ID},
		},
		Indexes: []DefIndex{
			{Name: "name"},
			{Name: "kind"},
			{Name: "version"},
		},
		IsDebug: true,
		IsTest:  true,
	})
	if err != nil {
		return err
	}
	err = catalog.Init()
	if err != nil {
		return err
	}

	return nil
}
