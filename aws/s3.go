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
	if err := r.ParseMultipartForm(2000); err != nil {
		return et.Item{}, err
	}

	fileparts, fileInfo, err := r.FormFile("myFile")
	if err != nil {
		return et.Item{}, err
	}
	defer fileparts.Close()

	contentType := fileInfo.Header.Get("Content-Type")
	ext := file.GetExtension(fileInfo.Filename)
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
* Delete: Deletes the object filePath of bucket.
* @param bucket, filePath string
* @return et.Item, error
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
* Download: Gets the object filePath of bucket; the S3 output goes in result.
* @param bucket, filePath string
* @return et.Item, error
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

/**
* UploadStream: Uploads the object key of bucket from r in parts, without holding it in memory (large files, e.g.
* backups); private (no ACL). An error reading r aborts the upload and nothing is left in the bucket.
* @param bucket, key, contentType string, r io.Reader
* @return et.Item, error
**/
func (s *S3AWS) UploadStream(bucket, key, contentType string, r io.Reader) (et.Item, error) {
	uploader := s3manager.NewUploader(s.sess)
	input := &s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   r,
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
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
			"key":      key,
			"url":      result.Location,
		},
	}, nil
}

/**
* List: The keys of bucket that start with prefix, in order.
* @param bucket, prefix string
* @return []string, error
**/
func (s *S3AWS) List(bucket, prefix string) ([]string, error) {
	s3client := s3.New(s.sess)
	result := []string{}
	err := s3client.ListObjectsV2Pages(&s3.ListObjectsV2Input{Bucket: aws.String(bucket), Prefix: aws.String(prefix)},
		func(page *s3.ListObjectsV2Output, last bool) bool {
			for _, obj := range page.Contents {
				result = append(result, aws.StringValue(obj.Key))
			}
			return true
		})
	return result, err
}

/**
* Open: Reads the object key of bucket as a stream; the caller closes it.
* @param bucket, key string
* @return io.ReadCloser, error
**/
func (s *S3AWS) Open(bucket, key string) (io.ReadCloser, error) {
	s3client := s3.New(s.sess)
	result, err := s3client.GetObject(&s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}
