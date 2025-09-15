package file

import (
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
	if result.Exist && result.IsDir {
		logs.Log("file", "exist path folder:", result.Path)
	} else if result.Exist {
		logs.Log("file", "exist path file:", result.Path)
	}

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
* @param folder, name, model string, args ...any
* @return string, error
**/
func MakeFile(folder, name, model string, args ...any) (string, error) {
	path := fmt.Sprintf(`%s/%s`, folder, name)
	info := ExistPath(path)
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
* ReadFile
* @param path string
* @return string, error
**/
func ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	logs.Log("file", "read file:", path)
	return string(content), nil
}
