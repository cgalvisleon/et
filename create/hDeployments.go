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

	url := strs.Format("`/%s`", strs.Lowcase(name))
	net := "proxy"
	_, err = file.MakeFile(path, "local.yml", modelDeploy, name, url, net)
	if err != nil {
		return err
	}

	return nil
}
