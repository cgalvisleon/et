package jfdb

import (
	"errors"
	"os"
	"strings"

	"github.com/cgalvisleon/et/strs"
)

func parcear(str string, args ...any) string {
	var result string = str
	for i, v := range args {
		p := strs.Format(`$%d`, i+1)
		rp := strs.Format(`%v`, v)
		result = strings.ReplaceAll(result, p, rp)
	}

	return result
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

func MakeFile(folder, name string, content []byte) (string, error) {
	path := strs.Format(`%s/%s`, folder, name)

	if ExistPath(path) {
		return "", errors.New("file found")
	}

	err := os.WriteFile(path, content, 0666)
	if err != nil {
		return "", err
	}

	return path, nil
}

func MakeFolder(names ...string) (string, error) {
	var path string
	for _, name := range names {
		path = strs.Append(path, name, "/")

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
