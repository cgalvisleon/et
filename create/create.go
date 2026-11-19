package create

import "github.com/spf13/cobra"

var Create = &cobra.Command{
	Use:   "go",
	Short: "Create a project, microservice, model or rpc.",
	Long:  "Scaffolds a project, a microservice, a model or an rpc handler on et: asks what to create and its data.",
	Run: func(cmd *cobra.Command, args []string) {
		PromptCreate()
	},
}
