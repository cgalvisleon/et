package create

import (
	"github.com/cgalvisleon/et/create/template"
	"github.com/cgalvisleon/et/file"
)

func MakeProject(name string) error {
	_, err := file.MakeFolder(name)
	if err != nil {
		return err
	}

	return nil
}

func MkProject(packageName, name, author, schema string) error {
	ProgressAdd(3)
	err := MkMicroservice(packageName, name, schema)
	if err != nil {
		return err
	}

	ProgressNext()
	err = MakeReadme(name)
	if err != nil {
		return err
	}

	ProgressNext()
	err = MakeEnv(name)
	if err != nil {
		return err
	}

	ProgressNext()

	return nil
}

func MakeReadme(packageName string) error {
	_, _ = file.MakeFile(".", "README.md", template.ModelReadme, packageName, "```")

	return nil
}

func MakeEnv(packageName string) error {
	_, _ = file.MakeFile(".", ".env", template.ModelEnvar, packageName)

	_, _ = file.MakeFile(".", ".env.prd", template.ModelEnvar, packageName)

	_, _ = file.MakeFile(".", ".env.qa", template.ModelEnvar, packageName)

	return nil
}

func MakeVersion(name string) error {
	_, _ = file.MakeFile(".", "version.sh", template.Version, name)

	return nil
}

func MakeDeploy(name string) error {
	_, _ = file.MakeFile(".", "deploy.sh", template.Deploy, name)

	return nil
}
