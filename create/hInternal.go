package create

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/create/template"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/strs"
)

func MakeInternal(projectName, name, schema string) error {
	if len(schema) > 0 {
		err := MakeInternalModel(name, schema)
		if err != nil {
			return err
		}
	}

	path, _ := file.MakeFolder("internal", "services", name)
	_, _ = file.MakeFile(path, "service.go", template.ModelService, projectName, name)

	path, _ = file.MakeFolder("internal", "services", name, "v1")

	if len(schema) > 0 {
		_, _ = file.MakeFile(path, "api.go", template.ModelDbApi, projectName, name)
	} else {
		_, _ = file.MakeFile(path, "api.go", template.ModelApi, projectName, name)
	}

	return nil
}

func MakeInternalModel(name, schema string) error {
	path, _ := file.MakeFolder("internal", "models", schema)

	_, _ = file.MakeFile(path, "schema.go", template.ModelSchema, schema)

	_, _ = file.MakeFile(path, "msg.go", template.ModelMsg, schema)

	modelo := strs.Titlecase(name)
	fileName := fmt.Sprintf(`%s.go`, name)
	tableName := tableName(name)
	_, err := file.MakeFile(path, fileName, template.ModelData, name, toCamelCase(modelo), tableName, schema)
	if err != nil {
		return err
	}

	return nil
}

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
