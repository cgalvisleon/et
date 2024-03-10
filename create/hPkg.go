package create

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/file"
)

func MakePkg(name, schema, schemaVar string) error {
	path, err := file.MakeFolder("pkg", name)
	if err != nil {
		return err
	}

	_, err = file.Make(path, "event.go", modelEvent, name)
	if err != nil {
		return err
	}

	modelo := et.Titlecase(name)
	_, err = file.Make(path, "model.go", modelModel, name, modelo)
	if err != nil {
		return err
	}

	_, err = file.Make(path, "msg.go", modelMsg, name)
	if err != nil {
		return err
	}

	_, err = file.Make(path, "controller.go", modelController, name)
	if err != nil {
		return err
	}

	title := et.Titlecase(name)
	_, err = file.Make(path, "router.go", modelRouter, name, title)
	if err != nil {
		return err
	}

	if len(schema) > 0 {
		_, err = file.Make(path, "schema.go", modelSchema, name, schemaVar, schema)
		if err != nil {
			return err
		}
	}

	return MakeModel(name, name, schemaVar)
}

func MakeModel(name, modelo, schemaVar string) error {
	path, err := file.MakeFolder("pkg", name)
	if err != nil {
		return err
	}

	modelo = et.Titlecase(modelo)
	fileName := et.Format(`h%s.go`, modelo)
	_, err = file.Make(path, fileName, modelHandler, name, modelo, schemaVar, et.Uppcase(modelo), et.Lowcase(modelo))
	if err != nil {
		return err
	}

	return nil
}

func MakeRpc(name string) error {
	path, err := file.MakeFolder("pkg", name)
	if err != nil {
		return err
	}

	_, err = file.Make(path, "hRpc.go", modelhRpc, name)
	if err != nil {
		return err
	}

	return nil
}
