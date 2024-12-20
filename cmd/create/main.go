package main

import (
	"os/exec"

	"github.com/cgalvisleon/et/create"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{Use: "go"}
	rootCmd.AddCommand(create.Create)
	rootCmd.Execute()
	modTidy()
}

func modTidy() error {
	cmd := exec.Command("go", "mod", "tidy")
	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	return nil
}
