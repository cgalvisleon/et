package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cgalvisleon/et/logs"
)

type Loader struct {
	baseDir string
}

func newLoader(baseDir string) *Loader {
	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		panic(err)
	}
	return &Loader{baseDir: absPath}
}

func (l *Loader) Resolve(modulePath string, currentDir string) (string, error) {
	// 1. Relativo ./ o ../
	if strings.HasPrefix(modulePath, "./") || strings.HasPrefix(modulePath, "../") {
		full := filepath.Join(currentDir, modulePath)
		result, err := l.resolveAsFileOrDir(full)
		if err != nil {
			return "", err
		}
		logs.Log("Resolve", result)
		return result, nil
	}

	// 2. node_modules
	nm := filepath.Join(l.baseDir, "node_modules", modulePath)
	result, err := l.resolveAsFileOrDir(nm)
	if err != nil {
		return "", err
	}
	logs.Log("Resolve", result)
	return result, nil
}

func (l *Loader) resolveAsFileOrDir(base string) (string, error) {
	// archivo directo
	if exists(base) {
		return base, nil
	}

	// archivo .js
	if exists(base + ".js") {
		return base + ".js", nil
	}

	// carpeta con package.json
	pkgFile := filepath.Join(base, "package.json")
	if exists(pkgFile) {
		data, _ := os.ReadFile(pkgFile)

		var pkg struct {
			Main string `json:"main"`
		}
		json.Unmarshal(data, &pkg)

		if pkg.Main != "" {
			return filepath.Join(base, pkg.Main), nil
		}
	}

	// fallback index.js
	index := filepath.Join(base, "index.js")
	if exists(index) {
		return index, nil
	}

	return "", fmt.Errorf("module not found: %s", base)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
