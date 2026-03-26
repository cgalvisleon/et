package vm

import (
	"os"
	"path/filepath"
)

type Loader struct {
	baseDir string
}

func newLoader(baseDir string) *Loader {
	return &Loader{baseDir: baseDir}
}

/**
* LoadFile - Carga un archivo
* @param string path
* @return (string, error)
**/
func (l *Loader) LoadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

/**
* Load - Carga un archivo
* @param string path
* @return (string, error)
**/
func (l *Loader) Load(path string) (string, error) {
	fullPath := filepath.Join(l.baseDir, path)
	return l.LoadFile(fullPath)
}
