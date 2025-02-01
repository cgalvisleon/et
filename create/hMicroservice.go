package create

func MkProject(packageName, name, author, schema string) error {
	ProgressNext(20)

	ProgressNext(20)
	err := MkMicroservice(packageName, name, schema)
	if err != nil {
		return err
	}

	ProgressNext(20)
	err = MakeReadme(name)
	if err != nil {
		return err
	}

	ProgressNext(20)
	err = MakeEnv(name)
	if err != nil {
		return err
	}

	ProgressNext(10)

	return nil
}

func MkMicroservice(projectName, name, schema string) error {
	ProgressNext(10)
	err := MakeCmd(projectName, name)
	if err != nil {
		return err
	}

	ProgressNext(10)
	err = MakeDeployments(name)
	if err != nil {
		return err
	}

	ProgressNext(10)
	err = MakeInternal(projectName, name, schema)
	if err != nil {
		return err
	}

	ProgressNext(10)
	err = MakePkg(projectName, name, schema)
	if err != nil {
		return err
	}

	ProgressNext(10)
	err = MakeScripts(name)
	if err != nil {
		return err
	}

	ProgressNext(40)
	err = MakeTest(name)
	if err != nil {
		return err
	}

	ProgressNext(10)

	return nil
}

func MkMolue(projectName, packageName, modelo, schema string) error {
	ProgressNext(10)
	err := MakeInternalModel(modelo, schema)
	if err != nil {
		return err
	}

	ProgressNext(10)
	err = MakeModel(projectName, packageName, modelo, schema)
	if err != nil {
		return err
	}

	ProgressNext(80)

	return nil
}

func MkRpc(name string) error {
	ProgressNext(10)
	err := MakeRpc(name)
	if err != nil {
		return err
	}

	ProgressNext(90)

	return nil
}

/**
*
**/
func DeleteMicroservice(packageName string) error {
	ProgressNext(10)
	err := DeleteCmd(packageName)
	if err != nil {
		return err
	}

	ProgressNext(80)

	ProgressNext(10)

	return nil
}
