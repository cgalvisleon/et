package create

import "github.com/cgalvisleon/elvis/file"

func MakeDeployments(name string) error {
	_, err := file.MakeFolder("deployments", "dev")
	if err != nil {
		return err
	}

	_, err = file.MakeFolder("deployments", "local")
	if err != nil {
		return err
	}

	_, err = file.MakeFolder("deployments", "prd")
	if err != nil {
		return err
	}

	return nil
}
