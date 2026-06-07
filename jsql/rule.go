package jsql

type Rule struct {
	model *Model
}

func defineRule(db *DB, schema string) (*Rule, error) {
	return &Rule{}, nil
}

func (s *Rule) SetModule(module string, source any) error {
	return nil
}

func (s *Rule) GetModule(module string, source any) (bool, error) {
	return false, nil
}

func (s *Rule) DeleteModule(module string) error {
	return nil
}
