package create

import (
	"fmt"

	"github.com/cgalvisleon/et/create/template"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/strs"
)

/**
* MakePkg: scaffolds pkg/<name>/ (event, config, rpc, controller, handler and router files); with a schema
* it uses the db templates and adds model.go.
* @param projectName, name, schema string
* @return error
**/
func MakePkg(projectName, name, schema string) error {
	pathPkg, err := file.MakeFolder("pkg", name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(pathPkg, "event.go", template.ModelEvent, name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(pathPkg, "config.go", template.ModelConfig, name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(pathPkg, "rpc.go", template.ModelRpc, name)
	if err != nil {
		return err
	}

	if len(schema) > 0 {
		_, err = file.MakeFile(pathPkg, "controller.go", template.ModelDbController, name)
		if err != nil {
			return err
		}

		modelo := strs.Titlecase(name)
		_, err = file.MakeFile(pathPkg, "model.go", template.ModelModel, name, schema, projectName, modelo)
		if err != nil {
			return err
		}

		fileName := fmt.Sprintf(`router-%s.go`, strs.Lowcase(name))
		_, err = file.MakeFile(pathPkg, fileName, template.ModelDbHandler, name, modelo, projectName, schema)
		if err != nil {
			return err
		}

		title := strs.Titlecase(name)
		_, err = file.MakeFile(pathPkg, "router.go", template.ModelDbRouter, name, title)
		if err != nil {
			return err
		}
	} else {
		_, err = file.MakeFile(pathPkg, "controller.go", template.ModelController, name)
		if err != nil {
			return err
		}

		modelo := strs.Titlecase(name)
		fileName := fmt.Sprintf(`h%s.go`, modelo)
		_, err = file.MakeFile(pathPkg, fileName, template.ModelHandler, name, toCamelCase(modelo))
		if err != nil {
			return err
		}

		_, err = file.MakeFile(pathPkg, "router.go", template.ModelRouter, name, toCamelCase(modelo))
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* MakeModel: scaffolds the handler file of a model in pkg/<packageName>/ (router-<modelo>.go with schema,
* h<Modelo>.go without); never overwrites an existing file.
* @param projectName, packageName, modelo, schema string
* @return error
**/
func MakeModel(projectName, packageName, modelo, schema string) error {
	pathPkg := fmt.Sprintf(`./pkg/%s`, packageName)

	if len(schema) > 0 {
		modelo := strs.Titlecase(modelo)
		// model.go already exists from the initial scaffold and file.MakeFile never
		// overwrites an existing file — wiring Define<Modelo> into initModels is a
		// manual step the caller is reminded about (see CmdModelo's Run in hCommand.go).

		fileName := fmt.Sprintf(`router-%s.go`, strs.Lowcase(modelo))
		_, err := file.MakeFile(pathPkg, fileName, template.ModelDbHandler, packageName, toCamelCase(modelo), projectName, schema)
		if err != nil {
			return err
		}
	} else {
		modelo = strs.Titlecase(modelo)
		fileName := fmt.Sprintf(`h%s.go`, modelo)
		_, err := file.MakeFile(pathPkg, fileName, template.ModelHandler, packageName, toCamelCase(modelo), projectName)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* MakeRpc: scaffolds pkg/<name>/hRpc.go.
* @param name string
* @return error
**/
func MakeRpc(name string) error {
	path, err := file.MakeFolder("pkg", name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(path, "hRpc.go", template.ModelhRpc, name)
	if err != nil {
		return err
	}

	return nil
}
