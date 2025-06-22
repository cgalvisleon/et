package create

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

func MkMolue(projectName, packageName, modelo, schema string) error {
	ProgressAdd(2)
	err := MakeInternalModel(modelo, schema)
	if err != nil {
		return err
	}

	ProgressNext()
	err = MakeModel(projectName, packageName, modelo, schema)
	if err != nil {
		return err
	}

	ProgressNext()

	return nil
}

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
