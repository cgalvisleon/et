package create

import (
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/strs"
)

func MakePkg(projectName, name, schema string) error {
	pathPkg, err := file.MakeFolder("pkg", name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(pathPkg, "event.go", modelEvent, name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(pathPkg, "config.go", modelConfig, name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(pathPkg, "rpc.go", modelRpc, name)
	if err != nil {
		return err
	}

	if len(schema) > 0 {
		_, err = file.MakeFile(pathPkg, "controller.go", modelDbController, name)
		if err != nil {
			return err
		}

		modelo := strs.Titlecase(name)
		_, err = file.MakeFile(pathPkg, "model.go", modelModel, name, schema, projectName, modelo)
		if err != nil {
			return err
		}

		fileName := strs.Format(`router-%s.go`, strs.Lowcase(name))
		_, err = file.MakeFile(pathPkg, fileName, modelDbHandler, name, modelo, projectName, schema)
		if err != nil {
			return err
		}

		title := strs.Titlecase(name)
		_, err = file.MakeFile(pathPkg, "router.go", modelDbRouter, name, title)
		if err != nil {
			return err
		}
	} else {
		_, err = file.MakeFile(pathPkg, "controller.go", modelController, name)
		if err != nil {
			return err
		}

		modelo := strs.Titlecase(name)
		fileName := strs.Format(`h%s.go`, modelo)
		_, err = file.MakeFile(pathPkg, fileName, modelHandler, name, toCamelCase(modelo))
		if err != nil {
			return err
		}

		_, err = file.MakeFile(pathPkg, "router.go", modelRouter, name)
		if err != nil {
			return err
		}
	}

	return nil
}

func MakeModel(projectName, packageName, modelo, schema string) error {
	pathPkg := strs.Format(`./pkg/%s`, packageName)

	if len(schema) > 0 {
		modelo := strs.Titlecase(modelo)
		_, _ = file.MakeFile(pathPkg, "model.go", modelModel, packageName, modelo, projectName)

		fileName := strs.Format(`router-%s.go`, strs.Lowcase(modelo))
		_, err := file.MakeFile(pathPkg, fileName, modelDbHandler, packageName, toCamelCase(modelo), projectName, schema)
		if err != nil {
			return err
		}
	} else {
		modelo = strs.Titlecase(modelo)
		fileName := strs.Format(`h%s.go`, modelo)
		_, err := file.MakeFile(pathPkg, fileName, modelHandler, packageName, toCamelCase(modelo), projectName)
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
