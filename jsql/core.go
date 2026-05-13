package jsql

/**
* initCore: Initializes the core schema tables required by the DB instance.
* @return error
**/
func (s *DB) initCore() error {
	return nil
}

var (
	catalog *Model
)

func (s *DB) defineCatalog() error {
	if catalog != nil {
		return nil
	}

	var err error
	catalog, err = s.Define(Define{
		Schema:  "core",
		Name:    "catalog",
		Version: 1,
		Columns: []Column{
			{Name: ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "name", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
			{Name: "version", TypeColumn: COLUMN, TypeData: INT, Default: 0},
			{Name: "kind", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
			{Name: "definition", TypeColumn: COLUMN, TypeData: JSON, Default: ""},
		},
		IdxField: IDX,
		PrimaryKeys: []Index{
			{Name: ID, Sorted: true},
		},
		Indexes: []Index{
			{Name: "name", Sorted: true},
			{Name: "kind", Sorted: true},
			{Name: "version", Sorted: true},
		},
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
