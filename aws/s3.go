package aws

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
)

/**
* UploaderS3
* @param bucket, filename, contentType string, contentFile []byte
* @return *s3manager.UploadOutput, error
**/
func UploaderS3(bucket, filename, contentType string, contentFile []byte) (*s3manager.UploadOutput, error) {
	sess, err := newSession()
	if err != nil {
		return nil, err
	}

	uploader := s3manager.NewUploader(sess)

	input := &s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(filename),
		Body:        bytes.NewReader(contentFile),
		ContentType: aws.String(contentType),
		ACL:         aws.String("public-read"),
	}

	result, err := uploader.UploadWithContext(context.Background(), input)
	if err != nil {
		return nil, err
	}

	return result, err
}

/**
* UploaderFile
* @param r *http.Request, folder, name string
* @return et.Item, error
**/
func UploaderFile(r *http.Request, folder, name string) (et.Item, error) {
	r.ParseMultipartForm(2000)
	fileparts, fileInfo, err := r.FormFile("myFile")
	if err != nil {
		return et.Item{}, err
	}
	defer fileparts.Close()

	contentType := fileInfo.Header.Get("Content-Type")
	ext := file.ExtencionFile(fileInfo.Filename)
	filename := fileInfo.Filename
	if len(name) > 0 {
		filename = fmt.Sprintf(`%s.%s`, name, ext)
	}
	if len(folder) > 0 {
		filename = fmt.Sprintf(`%s/%s`, folder, filename)
	}

	err = config.Validate([]string{
		"BUCKET",
		"STORAGE_TYPE",
		"HOSTNAME",
	})
	if err != nil {
		return et.Item{}, err
	}

	bucket := config.String("BUCKET", "")
	storageType := config.String("STORAGE_TYPE", "")
	if storageType == "S3" {
		contentFile, err := io.ReadAll(fileparts)
		if err != nil {
			return et.Item{}, err
		}

		output, err := UploaderS3(bucket, filename, contentType, contentFile)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{
			Ok: true,
			Result: et.Json{
				"bucket": bucket,
				"url":    output.Location,
			},
		}, nil
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

	hostname := config.String("HOSTNAME", "")
	url := fmt.Sprintf(`%s/%s`, hostname, outputFile)

	return et.Item{
		Ok: true,
		Result: et.Json{
			"bucket": bucket,
			"url":    url,
		},
	}, nil
}

/**
* UploaderB64
* @param b64, filename, contentType string
* @return et.Json, error
**/
func UploaderB64(b64, filename, contentType string) (et.Item, error) {
	if !utility.ValidStr(b64, 0, []string{""}) {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "b64")
	}

	if !utility.ValidStr(filename, 0, []string{""}) {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "filename")
	}

	if !utility.ValidStr(contentType, 0, []string{""}) {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "content-type")
	}

	err := config.Validate([]string{
		"STORAGE_TYPE",
		"BUCKET",
		"HOSTNAME",
	})
	if err != nil {
		return et.Item{}, err
	}

	storageType := config.String("STORAGE_TYPE", "")
	bucket := config.String("BUCKET", "")
	if storageType == "S3" {
		contentFile, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return et.Item{}, err
		}

		output, err := UploaderS3(bucket, filename, contentType, contentFile)
		if err != nil {
			return et.Item{}, err
		}

		return et.Item{
			Ok: true,
			Result: et.Json{
				"bucket": bucket,
				"url":    output.Location,
			},
		}, nil
	}

	file.MakeFolder(bucket)
	outputFile := fmt.Sprintf(`%s/%s`, bucket, filename)
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

	hostname := config.String("HOSTNAME", "")
	url := fmt.Sprintf(`%s/%s`, hostname, outputFile)

	return et.Item{
		Ok:     true,
		Result: et.Json{"url": url},
	}, nil
}

/**
* DeleteS3
* @param bucket, key string
* @return *s3.DeleteObjectOutput, error
**/
func DeleteS3(bucket, key string) (*s3.DeleteObjectOutput, error) {
	sess, err := newSession()
	if err != nil {
		return nil, err
	}

	s3client := s3.New(sess)

	request := &s3.DeleteObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}

	result, err := s3client.DeleteObject(request)
	if err != nil {
		return nil, err
	}

	return result, err
}

/**
* DownloadS3
* @param bucket, key string
* @return *s3.GetObjectOutput, error
**/
func DownloadS3(bucket, key string) (*s3.GetObjectOutput, error) {
	sess, err := newSession()
	if err != nil {
		return nil, err
	}

	s3client := s3.New(sess)

	request := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}

	result, err := s3client.GetObject(request)
	if err != nil {
		return nil, err
	}

	return result, err
}

/**
* DeleteFile
* @param url string
* @return bool, error
**/
func DeleteFile(url string) (bool, error) {
	err := config.Validate([]string{
		"STORAGE_TYPE",
		"BUCKET",
	})
	if err != nil {
		return false, err
	}

	storageType := config.String("STORAGE_TYPE", "")
	bucket := config.String("BUCKET", "")
	if storageType == "S3" {
		key := strings.ReplaceAll(url, fmt.Sprintf(`https://%s.s3.amazonaws.com/`, bucket), ``)
		_, err := DeleteS3(bucket, key)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	outdel, err := file.RemoveFile(url)
	if err != nil {
		return false, err
	}

	return outdel, nil
}

/**
* DownloaderFile
* @param url string
* @return string, error
**/
func DownloaderFile(url string) (string, error) {
	err := config.Validate([]string{
		"STORAGE_TYPE",
		"BUCKET",
	})
	if err != nil {
		return "", err
	}

	storageType := config.String("STORAGE_TYPE", "")
	bucket := config.String("BUCKET", "")
	if storageType == "S3" {
		key := strings.ReplaceAll(url, fmt.Sprintf(`https://%s.s3.amazonaws.com/`, bucket), ``)
		_, err := DownloadS3(bucket, key)
		if err != nil {
			return "", err
		}

		return url, nil
	}

	return url, nil
}
