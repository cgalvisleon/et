package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var dependencies = []string{
	"github.com/fsnotify/fsnotify@v1.9.0",
	"github.com/bwmarrin/snowflake@v0.3.0",
	"github.com/joho/godotenv/autoload@v1.5.1",
	"github.com/google/uuid@v1.6.0",
	"github.com/matoous/go-nanoid/v2@latest",
	"golang.org/x/crypto/bcrypt@v0.37.0",
	"golang.org/x/exp/slices@v0.0.0-20250408133849-7e4ce0ab07d0",
	"github.com/manifoldco/promptui@v0.9.0",
	"github.com/schollz/progressbar/v3@v3.18.0",
	"github.com/spf13/cobra@v1.9.1",
	"github.com/cgalvisleon/jdb/jdb@latest",
	"github.com/mattn/go-colorable@v0.1.14",
	"github.com/dimiro1/banner@v1.1.0",
	"github.com/chzyer/readline@v0.0.0-20180603132655-2972be24d48e",
}

func main() {
	total := 100
	for i, dep := range dependencies {
		p := (i + 1) * 100 / len(dependencies)
		fmt.Printf("\r[%-50s] %d%% Installing %s", progressBar(p, total, 50), p, dep)
		err := installLibrary(dep)
		if err != nil {
			return
		}
	}

	fmt.Printf("\r[%-50s] %d%% ¡Completado!", progressBar(total, total, 50), total)
	fmt.Println()

	if err := installContext(); err != nil {
		fmt.Printf("No se pudo instalar LIBRARY_CONTEXT.md: %v\n", err)
		return
	}

	fmt.Println("LIBRARY_CONTEXT.md instalado y referenciado en CLAUDE.md")
}

/**
* installContext: Copies LIBRARY_CONTEXT.md from the et module into the
* current project and references it from CLAUDE.md so AI assistants like
* Claude pick it up as persistent context.
* @return error
**/
func installContext() error {
	modDir, err := etModuleDir()
	if err != nil {
		return err
	}

	src := filepath.Join(modDir, "LIBRARY_CONTEXT.md")
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	dst := "LIBRARY_CONTEXT.md"
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return err
	}

	return referenceInClaudeMd(dst)
}

/**
* etModuleDir: Resolves the local directory of the github.com/cgalvisleon/et module.
* @return string
* @return error
**/
func etModuleDir() (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/cgalvisleon/et")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

/**
* referenceInClaudeMd: Ensures CLAUDE.md imports the given context file via
* the "@file" syntax, creating CLAUDE.md if it does not exist.
* @param file string
* @return error
**/
func referenceInClaudeMd(file string) error {
	importLine := "@" + file
	claudeFile := "CLAUDE.md"

	data, err := os.ReadFile(claudeFile)
	if os.IsNotExist(err) {
		content := "# CLAUDE.md\n\n" + importLine + "\n"
		return os.WriteFile(claudeFile, []byte(content), 0644)
	}
	if err != nil {
		return err
	}

	if strings.Contains(string(data), importLine) {
		return nil
	}

	content := strings.TrimRight(string(data), "\n") + "\n\n" + importLine + "\n"
	return os.WriteFile(claudeFile, []byte(content), 0644)
}

func installLibrary(library string) error {
	cmd := exec.Command("go", "get", library)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	return nil
}

func progressBar(current, total, width int) string {
	progress := int(float64(current) / float64(total) * float64(width))
	return fmt.Sprintf("%s%s", string(repeatRune('=', progress)), string(repeatRune(' ', width-progress)))
}

func repeatRune(char rune, count int) []rune {
	r := make([]rune, count)
	for i := 0; i < count; i++ {
		r[i] = char
	}
	return r
}
