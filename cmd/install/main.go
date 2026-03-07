package main

import (
	"fmt"
	"os/exec"
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
