package create

import (
	"fmt"

	"github.com/cgalvisleon/et/create/template"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/strs"
)

func MakeDeployments(name string) error {
	path, err := file.MakeFolder("deployments", name)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("`/%s`", strs.Lowcase(name))
	net := "proxy"
	_, err = file.MakeFile(path, "local.yml", template.ModelDeploy, name, url, net)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(path, "statefulset_tempalte.yml", template.StatefulsetTemplate, name)
	if err != nil {
		return err
	}

	_, err = file.MakeFile(path, "deployment_template.yml", template.DeploymentTemplate, name)
	if err != nil {
		return err
	}

	return nil
}
