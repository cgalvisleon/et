package file

import (
	"encoding/json"
	"errors"
	"fmt"
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
* GetExtencion
* @param filename string
* @return string
**/
func GetExtencion(filename string) string {
	lst := strings.Split(filename, ".")
	n := len(lst)
	if n > 1 {
		return lst[n-1]
	}

	return ""
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
* Remove
* @param path string
* @return bool, error
**/
func Remove(path string) (bool, error) {
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
* Save
* @param path string, obj any
* @return error
**/
func Save(path string, obj any) error {
	// Crear los directorios si no existen.
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}

	// WriteFile crea el archivo si no existe y lo sobrescribe si existe.
	return os.WriteFile(path, data, 0644)
}

/**
* Load
* @param path string, obj *T
* @return bool, error
**/
func Load[T any](path string, obj *T) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	if err := json.Unmarshal(data, obj); err != nil {
		return false, err
	}

	return true, nil
}

/**
* LoadOrSave
* @param path string, obj *T
* @return bool, error
**/
func LoadOrSave[T any](path string, obj *T) (bool, error) {
	loaded, err := Load(path, obj)
	if err != nil {
		return false, err
	}

	if loaded {
		return true, nil
	}

	if err := Save(path, obj); err != nil {
		return false, err
	}

	return false, nil
}
