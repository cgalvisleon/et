package create

import (
	"fmt"

	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/spf13/cobra"
)

var CmdProject = &cobra.Command{
	Use:   "micro [name author schema]",
	Short: "Create project base type microservice.",
	Long:  "Template project to microservice include folder cmd, deployments, pkg, rest, test and web, with files .go required for making a microservice.",
	Run: func(cmd *cobra.Command, args []string) {
		packageName, err := utility.GetGoMod("module")
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		name, err := PrompStr("Name", true)
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		author, err := PrompStr("Author", true)
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		schema, err := PrompStr("Schema", false)
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		err = MkProject(packageName, name, author, schema)
		if err != nil {
			fmt.Printf("Command failed %v\n", err)
			return
		}
	},
}

var CmdMicro = &cobra.Command{
	Use:   "micro [name schema]",
	Short: "Create project base type microservice.",
	Long:  "Template project to microservice include folder cmd, deployments, pkg, rest, test and web, with files .go required for making a microservice.",
	Run: func(cmd *cobra.Command, args []string) {
		projectName, err := utility.GetGoMod("module")
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		name, err := PrompStr("Name", true)
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		schema, err := PrompStr("Schema", false)
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		err = MkMicroservice(projectName, name, schema)
		if err != nil {
			fmt.Printf("Command failed %v\n", err)
			return
		}
	},
}

var CmdModelo = &cobra.Command{
	Use:   "modelo [name modelo schema]",
	Short: "Create model to microservice.",
	Long:  "Template model to microservice include function handler model.",
	Run: func(cmd *cobra.Command, args []string) {
		projectName, err := utility.GetGoMod("module")
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		packageName, err := PrompStr("Package", true)
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		modelo, err := PrompStr("Model", true)
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		schema, err := PrompStr("Schema", false)
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		err = MkMolue(projectName, packageName, modelo, schema)
		if err != nil {
			fmt.Printf("Command failed %v\n", err)
			return
		}

		title := strs.Titlecase(modelo)
		var message string
		if len(schema) > 0 {
			message = fmt.Sprintf(`Remember to wire up the new model manually:
  1. In pkg/%s/model.go, add "%s.Define%s(db)" inside initModels.
  2. Copy the route registrations from the bottom of pkg/%s/router-%s.go into pkg/%s/router.go.`,
				packageName, schema, title, packageName, strs.Lowcase(modelo), packageName)
		} else {
			message = fmt.Sprintf(`Remember to copy the route registration from the bottom of pkg/%s/h%s.go into pkg/%s/router.go.`,
				packageName, title, packageName)
		}
		fmt.Println(message)
	},
}

var CmdRpc = &cobra.Command{
	Use:   "rpc [name]",
	Short: "Create rpc model to microservice.",
	Long:  "Template rpc model to microservice include function handler model.",
	Run: func(cmd *cobra.Command, args []string) {
		name, err := PrompStr("Package", true)
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		err = MkRpc(name)
		if err != nil {
			fmt.Printf("Command failed %v\n", err)
			return
		}
	},
}
