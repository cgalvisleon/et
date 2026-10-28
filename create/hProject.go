package create

import (
	"github.com/cgalvisleon/et/create/template"
	"github.com/cgalvisleon/et/file"
)

/**
* MakeProject: creates the project folder.
* @param name string
* @return error
**/
func MakeProject(name string) error {
	_, err := file.MakeFolder(name)
	if err != nil {
		return err
	}

	return nil
}

/**
* MkProject: scaffolds a project: the microservice plus README.md and the .env files, reporting progress.
* @param packageName, name, author, schema string
* @return error
**/
func MkProject(packageName, name, author, schema string) error {
	ProgressAdd(3)
	err := MkMicroservice(packageName, name, schema)
	if err != nil {
		return err
	}

	ProgressNext()
	err = MakeReadme(packageName, name)
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

/**
* MakeReadme: scaffolds README.md in the current folder.
* @param moduleName, name string
* @return error
**/
func MakeReadme(moduleName, name string) error {
	_, _ = file.MakeFile(".", "README.md", template.ModelReadme, name, moduleName, "```")

	return nil
}

/**
* MakeEnv: scaffolds .env, .env.prd and .env.qa in the current folder.
* @param packageName string
* @return error
**/
func MakeEnv(packageName string) error {
	_, _ = file.MakeFile(".", ".env", template.ModelEnvar, packageName)

	_, _ = file.MakeFile(".", ".env.prd", template.ModelEnvar, packageName)

	_, _ = file.MakeFile(".", ".env.qa", template.ModelEnvar, packageName)

	return nil
}

/**
* MakeVersion: scaffolds version.sh in the current folder.
* @param name string
* @return error
**/
func MakeVersion(name string) error {
	_, _ = file.MakeFile(".", "version.sh", template.Version, name)

	return nil
}

/**
* MakeDeploy: scaffolds deploy.sh in the current folder.
* @param name string
* @return error
**/
func MakeDeploy(name string) error {
	_, _ = file.MakeFile(".", "deploy.sh", template.Deploy, name)

	return nil
}
