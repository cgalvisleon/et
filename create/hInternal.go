package create

import (
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/strs"
)

func MakeInternal(packageName, name, schema string) error {
	if len(schema) > 0 {
		path, err := file.MakeFolder("internal", "data", name)
		if err != nil {
			return err
		}

		schemaVar := strs.Append("schema", strs.Titlecase(schema), "")
		_, err = file.MakeFile(path, "schema.go", modelSchema, name, schemaVar, schema)
		if err != nil {
			return err
		}

		modelo := strs.Titlecase(name)
		fileName := strs.Format(`%s.go`, modelo)
		tableName := strs.Uppcase(name)
		_, err = file.MakeFile(path, fileName, modelDbHandler, name, modelo, tableName, schemaVar)
		if err != nil {
			return err
		}

		_, err = file.MakeFile(path, "msg.go", modelMsg, name)
		if err != nil {
			return err
		}
	}

	path, err := file.MakeFolder("internal", "service", name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(path, "service.go", modelService, packageName, name)
	if err != nil {
		return err
	}

	path, err = file.MakeFolder("internal", "service", name, "v1")
	if err != nil {
		return err
	}

	if len(schema) > 0 {
		_, err = file.MakeFile(path, "api.go", modelDbApi, packageName, name)
		if err != nil {
			return err
		}
	} else {
		_, err = file.MakeFile(path, "api.go", modelApi, packageName, name)
		if err != nil {
			return err
		}
	}

	return nil
}
