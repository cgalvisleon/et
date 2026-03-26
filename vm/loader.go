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
* ReadFile - Lee un archivo
* @param string path
* @return (string, error)
**/
func (l *Loader) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

/**
* Read - Lee un archivo
* @param string path
* @return (string, error)
**/
func (l *Loader) Read(path string) (string, error) {
	fullPath := filepath.Join(l.baseDir, path)
	return l.ReadFile(fullPath)
}
