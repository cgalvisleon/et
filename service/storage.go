package service

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
)

type Storage interface {
	Uploader(bucket, filename, contentType string, contentFile []byte) (et.Item, error)
	UploaderFile(r *http.Request, folder, name string) (et.Item, error)
}

type LocalStorage struct {
	Path string
}

func NewLocalStorage(path string) *LocalStorage {
	return &LocalStorage{
		Path: path,
	}
}

/**
* UploaderFile
* @param r *http.Request, folder, name string
* @return et.Item, error
**/
func (s *LocalStorage) UploaderFile(r *http.Request, bucket, folder, fileName string) (et.Item, error) {
	r.ParseMultipartForm(2000)
	fileparts, fileInfo, err := r.FormFile("myFile")
	if err != nil {
		return et.Item{}, err
	}
	defer fileparts.Close()

	ext := file.GetExtencion(fileInfo.Filename)
	filename := fileInfo.Filename
	if len(fileName) > 0 {
		filename = fmt.Sprintf(`%s.%s`, fileName, ext)
	}
	if len(folder) > 0 {
		filename = fmt.Sprintf(`%s/%s`, folder, filename)
	}

	file.MakeFolder(bucket)
	outputFile := fmt.Sprintf(`%s/%s`, bucket, filename)

	output, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return et.Item{}, err
	}
	defer output.Close()

	_, err = io.Copy(output, fileparts)
	if err != nil {
		return et.Item{}, err
	}

	hostname := envar.GetStr("HOSTNAME", "")
	url := fmt.Sprintf(`%s/%s`, hostname, outputFile)

	return et.Item{
		Ok: true,
		Result: et.Json{
			"provider": "Local Storage",
			"bucket":   bucket,
			"url":      url,
		},
	}, nil
}

/**
* UploaderB64
* @param bucket, b64, fileName, contentType string
* @return et.Json, error
**/
func (s *LocalStorage) UploaderB64(bucket, b64, fileName, contentType string) (et.Item, error) {
	if !utility.ValidStr(b64, 0, []string{""}) {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "b64")
	}

	if !utility.ValidStr(fileName, 0, []string{""}) {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "filename")
	}

	if !utility.ValidStr(contentType, 0, []string{""}) {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "content-type")
	}

	file.MakeFolder(bucket)
	outputFile := fmt.Sprintf(`%s/%s`, bucket, fileName)
	dec, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return et.Item{}, err
	}

	output, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return et.Item{}, err
	}
	defer output.Close()

	if _, err := output.Write(dec); err != nil {
		return et.Item{}, err
	}

	if err := output.Sync(); err != nil {
		return et.Item{}, err
	}

	return et.Item{
		Ok: true,
		Result: et.Json{
			"provider": "Local Storage",
			"bucket":   bucket,
			"url":      outputFile,
		},
	}, nil
}

/**
* DeleteS3
* @param bucket, key string
* @return *s3.DeleteObjectOutput, error
**/
func (s *LocalStorage) Delete(bucket, filePath string) (et.Item, error) {
	url := path.Join(s.Path, bucket, filePath)
	outdel, err := file.Remove(url)
	if err != nil {
		return et.Item{}, err
	}

	return et.Item{
		Ok: outdel,
		Result: et.Json{
			"provider": "Local Storage",
			"bucket":   bucket,
			"url":      filePath,
		},
	}, nil
}
