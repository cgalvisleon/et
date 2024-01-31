package file

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func params(str string, args ...any) string {
	var result string = str
	for i, v := range args {
		p := fmt.Sprintf(`$%d`, i+1)
		rp := fmt.Sprintf(`%v`, v)
		result = strings.ReplaceAll(result, p, rp)
	}

	return result
}

func append(str1, str2, sp string) string {
	if len(str1) == 0 {
		return str2
	}
	if len(str2) == 0 {
		return str1
	}

	return fmt.Sprintf(`%s%s%s`, str1, sp, str2)
}

func ExistPath(name string) bool {
	_, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}

	return true
}

func MakeFile(folder, name, model string, args ...any) (string, error) {
	path := fmt.Sprintf(`%s/%s`, folder, name)

	if ExistPath(path) {
		return "", errors.New("file found")
	}

	_content := params(model, args...)
	content := []byte(_content)
	err := os.WriteFile(path, content, 0666)
	if err != nil {
		return "", err
	}

	return path, nil
}

func MakeFolder(names ...string) (string, error) {
	var path string
	for _, name := range names {
		path = append(path, name, "/")

		if !ExistPath(path) {
			err := os.MkdirAll(path, os.ModePerm)
			if err != nil {
				return path, err
			}
		}
	}

	return path, nil
}

func RemoveFile(path string) (bool, error) {
	file := path
	if _, err := os.Stat(file); os.IsNotExist(err) {
		if err != nil {
			return false, err
		}

		return true, nil
	} else {
		os.Remove(file)
		return true, nil
	}
}

func ExtencionFile(filename string) string {
	lst := strings.Split(filename, ".")
	n := len(lst)
	if n > 1 {
		return lst[n-1]
	}

	return ""
}
