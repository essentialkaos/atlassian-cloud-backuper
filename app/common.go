package app

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/path"
	"github.com/essentialkaos/ek/v13/timeutil"

	"github.com/essentialkaos/katana"

	knfu "github.com/essentialkaos/ek/v13/knf/united"

	"github.com/essentialkaos/atlassian-cloud-backuper/backuper"
	"github.com/essentialkaos/atlassian-cloud-backuper/backuper/confluence"
	"github.com/essentialkaos/atlassian-cloud-backuper/backuper/jira"
	"github.com/essentialkaos/atlassian-cloud-backuper/uploader"
	"github.com/essentialkaos/atlassian-cloud-backuper/uploader/fs"
	"github.com/essentialkaos/atlassian-cloud-backuper/uploader/s3"
	"github.com/essentialkaos/atlassian-cloud-backuper/uploader/sftp"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// getBackuper returns backuper instances
func getBackuper(target string) (backuper.Backuper, error) {
	var err error
	var bkpr backuper.Backuper

	bkpConfig, err := getBackuperConfig(target)

	if err != nil {
		return nil, err
	}

	switch target {
	case TARGET_JIRA:
		bkpr, err = jira.NewBackuper(bkpConfig)
	case TARGET_CONFLUENCE:
		bkpr, err = confluence.NewBackuper(bkpConfig)
	}

	return bkpr, nil
}

// getOutputFileName returns name for backup output file
func getOutputFileName(target string) string {
	var template string

	switch target {
	case TARGET_JIRA:
		template = knfu.GetS(JIRA_OUTPUT_FILE, `jira-backup-%Y-%m-%d`) + ".zip"
	case TARGET_CONFLUENCE:
		template = knfu.GetS(JIRA_OUTPUT_FILE, `confluence-backup-%Y-%m-%d`) + ".zip"
	}

	return timeutil.Format(time.Now(), template)
}

// getBackuperConfig returns configuration for backuper
func getBackuperConfig(target string) (*backuper.Config, error) {
	switch target {
	case TARGET_JIRA:
		return &backuper.Config{
			Account:         knfu.GetS(ACCESS_ACCOUNT),
			Email:           knfu.GetS(ACCESS_EMAIL),
			APIKey:          knfu.GetS(ACCESS_API_KEY),
			WithAttachments: knfu.GetB(JIRA_INCLUDE_ATTACHMENTS),
			ForCloud:        knfu.GetB(JIRA_CLOUD_FORMAT),
		}, nil

	case TARGET_CONFLUENCE:
		return &backuper.Config{
			Account:         knfu.GetS(ACCESS_ACCOUNT),
			Email:           knfu.GetS(ACCESS_EMAIL),
			APIKey:          knfu.GetS(ACCESS_API_KEY),
			WithAttachments: knfu.GetB(CONFLUENCE_INCLUDE_ATTACHMENTS),
			ForCloud:        knfu.GetB(CONFLUENCE_CLOUD_FORMAT),
		}, nil
	}

	return nil, fmt.Errorf("Unknown target %q", target)
}

// getUploader returns uploader instance
func getUploader(target string) (uploader.Uploader, error) {
	var secret *katana.Secret

	if knfu.GetS(STORAGE_ENCRYPTION_KEY) != "" {
		secret = katana.NewSecret(knfu.GetS(STORAGE_ENCRYPTION_KEY))
	}

	switch strings.ToLower(knfu.GetS(STORAGE_TYPE)) {
	case STORAGE_FS:
		return fs.NewUploader(&fs.Config{
			Secret: secret,
			Path:   path.Join(knfu.GetS(STORAGE_FS_PATH), target),
			Mode:   knfu.GetM(STORAGE_FS_MODE, 0600),
		})

	case STORAGE_SFTP:
		keyData, err := readPrivateKeyData()

		if err != nil {
			return nil, err
		}

		return sftp.NewUploader(&sftp.Config{
			Secret: secret,
			Host:   knfu.GetS(STORAGE_SFTP_HOST),
			User:   knfu.GetS(STORAGE_SFTP_USER),
			Key:    keyData,
			Path:   path.Join(knfu.GetS(STORAGE_SFTP_PATH), target),
			Mode:   knfu.GetM(STORAGE_SFTP_MODE, 0600),
		})

	case STORAGE_S3:
		return s3.NewUploader(&s3.Config{
			Secret:      secret,
			Host:        knfu.GetS(STORAGE_S3_HOST),
			Region:      knfu.GetS(STORAGE_S3_REGION),
			AccessKeyID: knfu.GetS(STORAGE_S3_ACCESS_KEY),
			SecretKey:   knfu.GetS(STORAGE_S3_SECRET_KEY),
			Bucket:      knfu.GetS(STORAGE_S3_BUCKET),
			Path:        path.Join(knfu.GetS(STORAGE_S3_PATH), target),
			PartSize:    knfu.GetSZ(STORAGE_S3_PART_SIZE, 5*1024*1024),
		})
	}

	return nil, fmt.Errorf("Unknown storage type %q", knfu.GetS(STORAGE_TYPE))
}

// readPrivateKeyData reads private key data
func readPrivateKeyData() ([]byte, error) {
	if fsutil.IsExist(knfu.GetS(STORAGE_SFTP_KEY)) {
		return os.ReadFile(knfu.GetS(STORAGE_SFTP_KEY))
	}

	return base64.StdEncoding.DecodeString(knfu.GetS(STORAGE_SFTP_KEY))
}
