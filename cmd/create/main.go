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
}

func installLibrary(library string) error {
	cmd := exec.Command("go", "get", library)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	return nil
}
