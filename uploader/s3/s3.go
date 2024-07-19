package s3

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v13/events"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/log"
	"github.com/essentialkaos/ek/v13/passthru"
	"github.com/essentialkaos/ek/v13/path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/essentialkaos/atlassian-cloud-backuper/uploader"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Config is configuration for S3 uploader
type Config struct {
	Host        string
	Region      string
	AccessKeyID string
	SecretKey   string
	Bucket      string
	Path        string
}

// S3Uploader is S3 uploader instance
type S3Uploader struct {
	config     *Config
	dispatcher *events.Dispatcher
}

// ////////////////////////////////////////////////////////////////////////////////// //

// validate backuper interface
var _ uploader.Uploader = (*S3Uploader)(nil)

// ////////////////////////////////////////////////////////////////////////////////// //

// NewUploader creates new S3 uploader instance
func NewUploader(config *Config) (*S3Uploader, error) {
	err := config.Validate()

	if err != nil {
		return nil, err
	}

	return &S3Uploader{config, nil}, nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// SetDispatcher sets events dispatcher
func (u *S3Uploader) SetDispatcher(d *events.Dispatcher) {
	if u != nil {
		u.dispatcher = d
	}
}

// Upload uploads given file to S3 storage
func (u *S3Uploader) Upload(file, fileName string) error {
	u.dispatcher.DispatchAndWait(uploader.EVENT_UPLOAD_STARTED, "S3")

	lastUpdate := time.Now()
	fileSize := fsutil.GetSize(file)
	outputFile := path.Join(u.config.Path, fileName)

	log.Info(
		"Uploading backup file to %s:%s (%s/%s)",
		u.config.Bucket, u.config.Path, u.config.Host, u.config.Region,
	)

	client := s3.New(s3.Options{
		Region:       "ru-central1",
		BaseEndpoint: aws.String("https://storage.yandexcloud.net"),
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			u.config.AccessKeyID, u.config.SecretKey, "",
		)),
	})

	inputFD, err := os.OpenFile(file, os.O_RDONLY, 0)

	if err != nil {
		return fmt.Errorf("Can't open backup file for reading: %v", err)
	}

	defer inputFD.Close()

	r := passthru.NewReader(inputFD, fileSize)

	r.Update = func(n int) {
		if time.Since(lastUpdate) < 3*time.Second {
			return
		}

		u.dispatcher.Dispatch(
			uploader.EVENT_UPLOAD_PROGRESS,
			&uploader.ProgressInfo{Progress: r.Progress(), Current: r.Current(), Total: r.Total()},
		)

		lastUpdate = time.Now()
	}

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(u.config.Bucket),
		Key:    aws.String(outputFile),
		Body:   r,
	})

	if err != nil {
		return fmt.Errorf("Can't upload file to S3: %v", err)
	}

	log.Info("File successfully uploaded to S3!")
	u.dispatcher.DispatchAndWait(uploader.EVENT_UPLOAD_DONE, "S3")

	return nil
}

// Write writes data from given reader to given file
func (u *S3Uploader) Write(r io.ReadCloser, fileName string) error {
	u.dispatcher.DispatchAndWait(uploader.EVENT_UPLOAD_STARTED, "S3")

	outputFile := path.Join(u.config.Path, fileName)

	log.Info(
		"Uploading backup file to %s:%s (%s/%s)",
		u.config.Bucket, u.config.Path, u.config.Host, u.config.Region,
	)

	client := s3.New(s3.Options{
		Region:       "ru-central1",
		BaseEndpoint: aws.String("https://storage.yandexcloud.net"),
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			u.config.AccessKeyID, u.config.SecretKey, "",
		)),
	})

	_, err := client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(u.config.Bucket),
		Key:    aws.String(outputFile),
		Body:   r,
	})

	if err != nil {
		return fmt.Errorf("Can't upload file to S3: %v", err)
	}

	log.Info("File successfully uploaded to S3!")
	u.dispatcher.DispatchAndWait(uploader.EVENT_UPLOAD_DONE, "S3")

	return nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Validate validates configuration
func (c *Config) Validate() error {
	switch {
	case c == nil:
		return fmt.Errorf("Configuration validation error: config is nil")

	case c.Host == "":
		return fmt.Errorf("Configuration validation error: host is empty")

	case c.Region == "":
		return fmt.Errorf("Configuration validation error: region is empty")

	case c.AccessKeyID == "":
		return fmt.Errorf("Configuration validation error: access key is empty")

	case c.SecretKey == "":
		return fmt.Errorf("Configuration validation error: secret key is empty")

	case c.Bucket == "":
		return fmt.Errorf("Configuration validation error: bucket is empty")

	case c.Path == "":
		return fmt.Errorf("Configuration validation error: path is empty")

	case strings.HasPrefix(c.Host, "https://"),
		strings.HasPrefix(c.Host, "http://"):
		return fmt.Errorf("Configuration validation error: host must not contain scheme")
	}

	return nil
}
