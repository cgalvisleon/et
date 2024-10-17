package create

import (
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/strs"
)

func MakeDeployments(name string) error {
	path, err := file.MakeFolder("deployments", strs.Lowcase(name))
	if err != nil {
		return err
	}

	url := strs.Format("`/api/%s`", strs.Lowcase(name))
	net := "proxy"
	_, err = file.MakeFile(path, "local.yml", modelDeploy, name, url, net)
	if err != nil {
		return err
	}

	url = strs.Format("`/qa/api/%s`", strs.Lowcase(name))
	net = "qa"
	_, err = file.MakeFile(path, "qa.yml", modelDeploy, name, url, net)
	if err != nil {
		return err
	}

	url = strs.Format("`/api/%s`", strs.Lowcase(name))
	net = "prd"
	_, err = file.MakeFile(path, "prd.yml", modelDeploy, name, url, net)
	if err != nil {
		return err
	}

	return nil
}
