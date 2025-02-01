package create

import (
	"strings"

	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/strs"
)

func toCamelCase(input string) string {
	var result string
	names := strings.Split(input, "-")
	for i, name := range names {
		names[i] = strs.Titlecase(name)
		result = strs.Append(result, names[i], "")
	}

	return result
}

func tableName(name string) string {
	result := strs.Lowcase(name)
	result = strs.ReplaceAll(result, []string{"-"}, "_")

	return result
}

func MakeInternalModel(name, schema string) error {
	path, _ := file.MakeFolder("internal", "models", schema)

	_, _ = file.MakeFile(path, "schema.go", modelSchema, schema)

	_, _ = file.MakeFile(path, "msg.go", modelMsg, schema)

	modelo := strs.Titlecase(name)
	fileName := strs.Format(`%s.go`, name)
	tableName := tableName(name)
	_, err := file.MakeFile(path, fileName, modelData, name, toCamelCase(modelo), tableName, schema)
	if err != nil {
		return err
	}

	return nil
}

func MakeInternal(projectName, name, schema string) error {
	if len(schema) > 0 {
		err := MakeInternalModel(name, schema)
		if err != nil {
			return err
		}
	}

	path, _ := file.MakeFolder("internal", "service", name)

	_, _ = file.MakeFile(path, "service.go", modelService, projectName, name)

	path, _ = file.MakeFolder("internal", "service", name, "v1")

	if len(schema) > 0 {
		_, _ = file.MakeFile(path, "api.go", modelDbApi, projectName, name)
	} else {
		_, _ = file.MakeFile(path, "api.go", modelApi, projectName, name)
	}

	return nil
}
