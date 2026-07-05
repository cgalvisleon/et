package aws

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
)

type S3AWS struct {
	Params Params
	sess   *session.Session
}

/**
* NewS3AWS
* @param params Params
* @return *S3AWS, error
**/
func NewS3AWS(params Params) (*S3AWS, error) {
	result := &S3AWS{
		Params: params,
	}

	sess, err := newSession(params)
	if err != nil {
		return nil, err
	}

	result.sess = sess
	return result, nil
}

/**
* Uploader
* @param bucket, filename, contentType string, contentFile []byte
* @return et.Item, error
**/
func (s *S3AWS) Uploader(bucket, filename, contentType string, contentFile []byte) (et.Item, error) {
	uploader := s3manager.NewUploader(s.sess)

	input := &s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(filename),
		Body:        bytes.NewReader(contentFile),
		ContentType: aws.String(contentType),
		ACL:         aws.String("public-read"),
	}

	result, err := uploader.UploadWithContext(context.Background(), input)
	if err != nil {
		return et.Item{}, err
	}

	return et.Item{
		Ok: true,
		Result: et.Json{
			"provider": "AWS S3",
			"bucket":   bucket,
			"url":      result.Location,
			"result":   result,
		},
	}, nil
}

/**
* UploaderFile
* @param r *http.Request, bucket, folder, fileName string
* @return et.Item, error
**/
func (s *S3AWS) UploaderFile(r *http.Request, bucket, folder, fileName string) (et.Item, error) {
	r.ParseMultipartForm(2000)
	fileparts, fileInfo, err := r.FormFile("myFile")
	if err != nil {
		return et.Item{}, err
	}
	defer fileparts.Close()

	contentType := fileInfo.Header.Get("Content-Type")
	ext := file.GetExtencion(fileInfo.Filename)
	filename := fileInfo.Filename
	if len(fileName) > 0 {
		filename = fmt.Sprintf(`%s.%s`, fileName, ext)
	}

	if len(folder) > 0 {
		filename = fmt.Sprintf(`%s/%s`, folder, filename)
	}

	contentFile, err := io.ReadAll(fileparts)
	if err != nil {
		return et.Item{}, err
	}

	result, err := s.Uploader(bucket, filename, contentType, contentFile)
	if err != nil {
		return et.Item{}, err
	}

	return result, nil
}

/**
* UploaderB64
* @param bucket, b64, fileName, contentType string
* @return et.Json, error
**/
func (s *S3AWS) UploaderB64(bucket, b64, fileName, contentType string) (et.Item, error) {
	if !utility.ValidStr(b64, 0, []string{""}) {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "b64")
	}

	if !utility.ValidStr(fileName, 0, []string{""}) {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "filename")
	}

	if !utility.ValidStr(contentType, 0, []string{""}) {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "content-type")
	}

	contentFile, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return et.Item{}, err
	}

	result, err := s.Uploader(bucket, fileName, contentType, contentFile)
	if err != nil {
		return et.Item{}, err
	}

	return result, nil
}

/**
* DeleteS3
* @param bucket, key string
* @return *s3.DeleteObjectOutput, error
**/
func (s *S3AWS) Delete(bucket, filePath string) (et.Item, error) {
	s3client := s3.New(s.sess)

	request := &s3.DeleteObjectInput{
		Bucket: &bucket,
		Key:    &filePath,
	}

	result, err := s3client.DeleteObject(request)
	if err != nil {
		return et.Item{}, err
	}

	return et.Item{
		Ok: true,
		Result: et.Json{
			"provider": "AWS S3",
			"bucket":   bucket,
			"url":      filePath,
			"result":   result,
		},
	}, nil
}

/**
* DownloadS3
* @param bucket, key string
* @return *s3.GetObjectOutput, error
**/
func (s *S3AWS) Download(bucket, filePath string) (et.Item, error) {
	s3client := s3.New(s.sess)

	request := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &filePath,
	}

	result, err := s3client.GetObject(request)
	if err != nil {
		return et.Item{}, err
	}

	return et.Item{
		Ok: true,
		Result: et.Json{
			"provider": "AWS S3",
			"bucket":   bucket,
			"url":      filePath,
			"result":   result,
		},
	}, nil
}
