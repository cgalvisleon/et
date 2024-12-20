package main

import (
	"os/exec"

	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/create"
	"github.com/spf13/cobra"
)

func main() {
	installLibrary("github.com/fsnotify/fsnotify")
	installLibrary("github.com/bwmarrin/snowflake")
	installLibrary("github.com/joho/godotenv/autoload")
	installLibrary("github.com/google/uuid")
	installLibrary("github.com/matoous/go-nanoid/v2")
	installLibrary("github.com/oklog/ulid")
	installLibrary("golang.org/x/crypto/bcrypt")
	installLibrary("golang.org/x/exp/slices")
	installLibrary("github.com/manifoldco/promptui")
	installLibrary("github.com/schollz/progressbar/v3")
	installLibrary("github.com/spf13/cobra")

	console.Log("Et", "All is ok.")

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
