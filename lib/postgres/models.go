package lib

/**
* ModelExist get a model from the database
* @param schema string
* @param name string
* @return js.Json
**/
func (d *Postgres) ModelExist(schema, name string) (bool, error) {
	return ExistTable(d.DB, schema, name)
}
