package main

import (
	"github.com/cgalvisleon/et/create"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{Use: "go"}
	rootCmd.AddCommand(create.Create)
	rootCmd.Execute()
}
