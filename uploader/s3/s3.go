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
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/essentialkaos/katana"

	"github.com/essentialkaos/atlassian-cloud-backuper/uploader"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Config is configuration for S3 uploader
type Config struct {
	Secret *katana.Secret

	Host        string
	Region      string
	AccessKeyID string
	SecretKey   string
	Bucket      string
	Path        string
	PartSize    int64
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
	fd, err := os.Open(file)

	if err != nil {
		return fmt.Errorf("Can't open backup file for reading: %v", err)
	}

	defer fd.Close()

	err = u.Write(fd, fileName, fsutil.GetSize(file))

	if err != nil {
		return fmt.Errorf("Can't save backup: %w", err)
	}

	return nil
}

// Write writes data from given reader to given file
func (u *S3Uploader) Write(r io.ReadCloser, fileName string, fileSize int64) error {
	u.dispatcher.DispatchAndWait(uploader.EVENT_UPLOAD_STARTED, "S3")

	var rr io.Reader
	var err error

	lastUpdate := time.Now()
	outputFile := fileName

	if u.config.Path != "" {
		outputFile = path.Join(u.config.Path, fileName)
	}

	log.Info(
		"Uploading backup file to %s:%s (%s/%s)",
		u.config.Bucket, u.config.Path, u.config.Host, u.config.Region,
	)

	rr = r

	if u.config.Secret != nil {
		sr, err := u.config.Secret.NewReader(r, katana.MODE_ENCRYPT)

		if err != nil {
			return fmt.Errorf("Can't create encrypted reader: %w", err)
		}

		rr = sr
	}

	if fileSize > 0 {
		pr := passthru.NewReader(rr, fileSize)

		pr.Update = func(n int) {
			if time.Since(lastUpdate) < 3*time.Second {
				return
			}

			u.dispatcher.Dispatch(
				uploader.EVENT_UPLOAD_PROGRESS,
				&uploader.ProgressInfo{
					Progress: pr.Progress(),
					Current:  pr.Current(),
					Total:    pr.Total(),
				},
			)
		}

		rr = pr
	}

	client := s3.New(s3.Options{
		Region:       u.config.Region,
		BaseEndpoint: aws.String("https://" + u.config.Host),
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			u.config.AccessKeyID, u.config.SecretKey, "",
		)),
	})

	manager := manager.NewUploader(client, func(c *manager.Uploader) {
		c.PartSize = u.config.PartSize * 1024 * 1024
	})

	_, err = manager.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(u.config.Bucket),
		Key:    aws.String(outputFile),
		Body:   rr,
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
