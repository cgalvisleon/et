package create

/**
* MkMicroservice: scaffolds a whole microservice (cmd, deployments, internal, pkg, scripts, test)
* reporting progress on the bar.
* @param projectName, name, schema string
* @return error
**/
func MkMicroservice(projectName, name, schema string) error {
	ProgressAdd(6)
	err := MakeCmd(projectName, name)
	if err != nil {
		return err
	}

	ProgressNext()
	err = MakeDeployments(name)
	if err != nil {
		return err
	}

	ProgressNext()
	err = MakeInternal(projectName, name, schema)
	if err != nil {
		return err
	}

	ProgressNext()
	err = MakePkg(projectName, name, schema)
	if err != nil {
		return err
	}

	ProgressNext()
	err = MakeScripts(name)
	if err != nil {
		return err
	}

	ProgressNext()
	err = MakeTest(name)
	if err != nil {
		return err
	}

	ProgressNext()

	return nil
}

/**
* MkModel: scaffolds a model in an existing microservice: its internal model (with schema) and its pkg handler.
* @param projectName, packageName, modelo, schema string
* @return error
**/
func MkModel(projectName, packageName, modelo, schema string) error {
	ProgressAdd(2)
	if len(schema) > 0 {
		if err := MakeInternalModel(modelo, schema); err != nil {
			return err
		}
	}

	ProgressNext()
	err := MakeModel(projectName, packageName, modelo, schema)
	if err != nil {
		return err
	}

	ProgressNext()

	return nil
}

/**
* MkRpc: scaffolds the rpc handler of a package reporting progress on the bar.
* @param name string
* @return error
**/
func MkRpc(name string) error {
	ProgressAdd(1)
	err := MakeRpc(name)
	if err != nil {
		return err
	}

	ProgressNext()

	return nil
}

/**
*
**/
func DeleteMicroservice(packageName string) error {
	ProgressAdd(1)
	err := DeleteCmd(packageName)
	if err != nil {
		return err
	}

	ProgressNext()

	return nil
}
