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

	err = defineSeries(s)
	if err != nil {
		return err
	}

	return nil
}
