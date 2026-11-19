package create

import (
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/cgalvisleon/et/file"
)

/**
* makeFile: file.MakeFile (never overwrites) plus gofmt on a newly created .go file, so the generated imports are
* ordered whatever the module name is (the templates cannot know where it sorts against github.com/cgalvisleon/et).
* @param path, name, model string, args ...any
* @return string, error
**/
func makeFile(path, name, model string, args ...any) (string, error) {
	existed := file.ExistPath(filepath.Join(path, name)).Exist
	result, err := file.MakeFile(path, name, model, args...)
	if err != nil || existed || !strings.HasSuffix(name, ".go") {
		return result, err
	}

	return result, formatFile(result)
}

/**
* formatFile: Rewrites a Go file in gofmt style; a file that does not parse is left as is.
* @param path string
* @return error
**/
func formatFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	out, err := format.Source(src)
	if err != nil {
		return nil
	}

	return os.WriteFile(path, out, 0644)
}
