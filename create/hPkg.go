package create

import (
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/strs"
)

func MakePkg(name, schema string) error {
	path, err := file.MakeFolder("pkg", name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(path, "event.go", modelEvent, name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(path, "msg.go", modelMsg, name)
	if err != nil {
		return err
	}

	if len(schema) > 0 {
		_, err = file.MakeFile(path, "controller.go", modelDbController, name)
		if err != nil {
			return err
		}

		schemaVar := strs.Append("schema", strs.Titlecase(schema), "")
		_, err = file.MakeFile(path, "schema.go", modelSchema, name, schemaVar, schema)
		if err != nil {
			return err
		}

		modelo := strs.Titlecase(name)
		_, err = file.MakeFile(path, "model.go", modelModel, name, modelo)
		if err != nil {
			return err
		}

		path, err := file.MakeFolder("pkg", name)
		if err != nil {
			return err
		}

		fileName := strs.Format(`h%s.go`, modelo)
		_, err = file.MakeFile(path, fileName, modelDbHandler, name, modelo, schemaVar, strs.Uppcase(modelo), strs.Lowcase(modelo))
		if err != nil {
			return err
		}

		title := strs.Titlecase(name)
		_, err = file.MakeFile(path, "router.go", modelDbRouter, name, title)
		if err != nil {
			return err
		}
	} else {
		_, err = file.MakeFile(path, "controller.go", modelController, name)
		if err != nil {
			return err
		}

		modelo := strs.Titlecase(name)
		fileName := strs.Format(`h%s.go`, modelo)
		_, err = file.MakeFile(path, fileName, modelHandler, name, modelo, strs.Lowcase(modelo))
		if err != nil {
			return err
		}

		_, err = file.MakeFile(path, "router.go", modelRouter, name, strs.Lowcase(name))
		if err != nil {
			return err
		}
	}

	return nil
}

func MakeModel(packageName, modelo, schema string) error {
	path := strs.Format(`./pkg/%s`, packageName)

	if len(schema) > 0 {
		schemaVar := strs.Append("schema", strs.Titlecase(schema), "")
		_, _ = file.MakeFile(path, "schema.go", modelSchema, packageName, schemaVar, schema)

		modelo := strs.Titlecase(modelo)
		_, _ = file.MakeFile(path, "model.go", modelModel, packageName, modelo)

		modelo = strs.Titlecase(modelo)
		fileName := strs.Format(`h%s.go`, modelo)
		_, err := file.MakeFile(path, fileName, modelDbHandler, packageName, modelo, schemaVar, strs.Uppcase(modelo), strs.Lowcase(modelo))
		if err != nil {
			return err
		}
	} else {
		modelo = strs.Titlecase(modelo)
		fileName := strs.Format(`h%s.go`, modelo)
		_, err := file.MakeFile(path, fileName, modelHandler, packageName, modelo, strs.Lowcase(modelo))
		if err != nil {
			return err
		}
	}

	return nil
}

func MakeRpc(name string) error {
	path, err := file.MakeFolder("pkg", name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(path, "hRpc.go", modelhRpc, name)
	if err != nil {
		return err
	}

	return nil
}
