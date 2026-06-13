package file

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

type FileInfo struct {
	Path  string
	Info  os.FileInfo
	Error error
	IsDir bool
	Exist bool
}

func (s *FileInfo) Json() et.Json {
	return et.Json{
		"path":  s.Path,
		"info":  s.Info,
		"error": s.Error,
		"isDir": s.IsDir,
		"exist": s.Exist,
	}
}

/**
* Params
* @param str string, args ...any
* @return string
**/
func params(str string, args ...any) string {
	var result string = str
	for i, v := range args {
		p := fmt.Sprintf(`$%d`, i+1)
		rp := fmt.Sprintf(`%v`, v)
		result = strings.ReplaceAll(result, p, rp)
	}

	return result
}

/**
* Append
* @param str1, str2, sp string
* @return string
**/
func append(str1, str2, sp string) string {
	if len(str1) == 0 {
		return str2
	}
	if len(str2) == 0 {
		return str1
	}

	return fmt.Sprintf(`%s%s%s`, str1, sp, str2)
}

/**
* ExistPath
* @param path string
* @return bool
**/
func ExistPath(path string) FileInfo {
	result := FileInfo{
		Path:  path,
		Info:  nil,
		Error: nil,
		IsDir: false,
	}

	result.Path, result.Error = filepath.Abs(path)
	if result.Error != nil {
		return result
	}

	result.Info, result.Error = os.Stat(path)
	if os.IsNotExist(result.Error) {
		result.Exist = false
		result.Error = nil
		return result
	} else if result.Error != nil {
		return result
	}

	result.Exist = true
	result.IsDir = result.Info != nil && result.Info.IsDir()

	return result
}

/**
* MakeFolder
* @param names ...string
* @return string, error
**/
func MakeFolder(names ...string) (string, error) {
	var path string
	for _, name := range names {
		path = append(path, name, "/")
		absPath, err := filepath.Abs(path)
		if err != nil {
			return path, err
		}

		info := ExistPath(absPath)
		if info.Error != nil {
			return info.Path, info.Error
		} else if info.Exist {
			continue
		} else {
			err := os.MkdirAll(absPath, 0755)
			if err != nil {
				return path, err
			}
		}
	}

	logs.Log("file", "make folder:", path)
	return path, nil
}

/**
* MakeFile
* @param path, name, model string, args ...any
* @return string, error
**/
func MakeFile(path, name, model string, args ...any) (string, error) {
	pathFile := fmt.Sprintf(`%s/%s`, path, name)
	info := ExistPath(pathFile)
	if info.Error != nil {
		return info.Path, info.Error
	} else if info.IsDir {
		return info.Path, nil
	} else if info.Exist {
		return info.Path, nil
	}

	file, err := os.Create(info.Path)
	if err != nil {
		return "", err
	}

	content := params(model, args...)
	bt := []byte(content)
	_, err = file.Write(bt)
	if err != nil {
		return "", err
	}

	logs.Log("file", "make file:", path)
	return path, nil
}

/**
* RemoveFile
* @param path string
* @return bool, error
**/
func RemoveFile(path string) (bool, error) {
	file := path
	if _, err := os.Stat(file); os.IsNotExist(err) {
		if err != nil {
			return false, err
		}

		logs.Log("file", "remove file:", file)
		return true, nil
	} else {
		os.Remove(file)
		return true, nil
	}
}

/**
* ExtencionFile
* @param filename string
* @return string
**/
func ExtencionFile(filename string) string {
	lst := strings.Split(filename, ".")
	n := len(lst)
	if n > 1 {
		return lst[n-1]
	}

	return ""
}

/**
* openOrCreate
* @param path string
* @return *os.File, bool, error
**/
func openOrCreate(path string) (*os.File, bool, error) {
	created := false

	if _, err := os.Stat(path); os.IsNotExist(err) {
		created = true
	}

	f, err := os.OpenFile(
		path,
		os.O_RDWR|os.O_CREATE,
		0o644,
	)
	if err != nil {
		return nil, false, err
	}

	return f, created, nil
}

/**
* LoadOrCreateJSON
* @param path string, defaultValue T
* @return T, error
**/
func LoadOrCreateJSON[T any](path string, defaultValue T) (T, error) {
	var result T

	f, created, err := openOrCreate(path)
	if err != nil {
		return result, err
	}
	defer f.Close()

	if created {
		// Crear el archivo con los valores por defecto
		b, err := json.MarshalIndent(defaultValue, "", "  ")
		if err != nil {
			return result, err
		}

		if _, err := f.Write(b); err != nil {
			return result, err
		}

		return defaultValue, nil
	}

	// Leer contenido existente
	b, err := io.ReadAll(f)
	if err != nil {
		return result, err
	}

	if len(b) == 0 {
		return defaultValue, nil
	}

	if err := json.Unmarshal(b, &result); err != nil {
		return result, err
	}

	return result, nil
}

/**
* WriteJSON
* @param path string, value T
* @return error
**/
func WriteJSON[T any](path string, value T) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	f, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o644,
	)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(b)
	return err
}

/**
* LoadOrCreateString
* @param path string, defaultValue string
* @return string, error
**/
func LoadString(path string, defaultValue string) (string, error) {
	f, created, err := openOrCreate(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if created {
		if _, err := f.WriteString(defaultValue); err != nil {
			return "", err
		}

		return defaultValue, nil
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

/**
* WriteString
* @param path string, value string
* @return error
**/
func WriteString(path string, value string) error {
	return os.WriteFile(path, []byte(value), 0o644)
}
