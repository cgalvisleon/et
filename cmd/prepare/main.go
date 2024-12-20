package main

import (
	"fmt"
	"os/exec"
)

var dependencies = []string{
	"github.com/fsnotify/fsnotify",
	"github.com/bwmarrin/snowflake",
	"github.com/joho/godotenv/autoload",
	"github.com/google/uuid",
	"github.com/matoous/go-nanoid/v2",
	"github.com/oklog/ulid",
	"golang.org/x/crypto/bcrypt",
	"golang.org/x/exp/slices",
	"github.com/manifoldco/promptui",
	"github.com/schollz/progressbar/v3",
	"github.com/spf13/cobra",
}

func main() {
	total := len(dependencies)
	for i, dep := range dependencies {
		fmt.Printf("Installing %s... %s\r", dep, progressBar(i, total, 20))
		err := installLibrary(dep)
		if err != nil {
			fmt.Printf("Error installing %s: %s\n", dep, err)
			return
		}
	}

	fmt.Println("\n¡Completado!")
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
