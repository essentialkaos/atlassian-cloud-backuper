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
	"time"

	"github.com/essentialkaos/ek/v13/events"
	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/log"
	"github.com/essentialkaos/ek/v13/options"
	"github.com/essentialkaos/ek/v13/path"
	"github.com/essentialkaos/ek/v13/spinner"
	"github.com/essentialkaos/ek/v13/timeutil"

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

// startApp starts app in basic mode
func startApp(args options.Arguments) error {
	var dispatcher *events.Dispatcher

	if options.GetB(OPT_INTERACTIVE) {
		dispatcher = events.NewDispatcher()
		addEventsHandlers(dispatcher)
	}

	defer temp.Clean()

	target := args.Get(0).String()
	bkpr, err := getBackuper(target)

	if err != nil {
		return fmt.Errorf("Can't start backuping process: %v", err)
	}

	bkpr.SetDispatcher(dispatcher)

	outputFileName := getOutputFileName(target)
	tmpFile := path.Join(temp.MkName(".zip"), outputFileName)

	err = bkpr.Backup(tmpFile)

	if err != nil {
		spinner.Done(false)
		return fmt.Errorf("Error while backuping process: %v", err)
	}

	log.Info("Backup process successfully finished!")

	updr, err := getUploader(target)

	if err != nil {
		return fmt.Errorf("Can't start uploading process: %v", err)
	}

	updr.SetDispatcher(dispatcher)

	err = updr.Upload(tmpFile, outputFileName)

	if err != nil {
		spinner.Done(false)
		return fmt.Errorf("Error while uploading process: %v", err)
	}

	return nil
}

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
	var err error
	var updr uploader.Uploader

	switch knfu.GetS(STORAGE_TYPE) {
	case STORAGE_FS:
		updr, err = fs.NewUploader(&fs.Config{
			Path: path.Join(knfu.GetS(STORAGE_FS_PATH), target),
			Mode: knfu.GetM(STORAGE_FS_MODE, 0600),
		})

	case STORAGE_SFTP:
		keyData, err := readPrivateKeyData()

		if err != nil {
			return nil, err
		}

		updr, err = sftp.NewUploader(&sftp.Config{
			Host: knfu.GetS(STORAGE_SFTP_HOST),
			User: knfu.GetS(STORAGE_SFTP_USER),
			Key:  keyData,
			Path: path.Join(knfu.GetS(STORAGE_SFTP_PATH), target),
			Mode: knfu.GetM(STORAGE_SFTP_MODE, 0600),
		})

	case STORAGE_S3:
		updr, err = s3.NewUploader(&s3.Config{
			Host:        knfu.GetS(STORAGE_S3_HOST),
			Region:      knfu.GetS(STORAGE_S3_REGION),
			AccessKeyID: knfu.GetS(STORAGE_S3_ACCESS_KEY),
			SecretKey:   knfu.GetS(STORAGE_S3_SECRET_KEY),
			Bucket:      knfu.GetS(STORAGE_S3_BUCKET),
			Path:        path.Join(knfu.GetS(STORAGE_S3_PATH), target),
		})
	}

	return updr, err
}

// readPrivateKeyData reads private key data
func readPrivateKeyData() ([]byte, error) {
	if fsutil.IsExist(knfu.GetS(STORAGE_SFTP_KEY)) {
		return os.ReadFile(knfu.GetS(STORAGE_SFTP_KEY))
	}

	return base64.StdEncoding.DecodeString(knfu.GetS(STORAGE_SFTP_KEY))
}

// addEventsHandlers registers events handlers
func addEventsHandlers(dispatcher *events.Dispatcher) {
	dispatcher.AddHandler(backuper.EVENT_BACKUP_STARTED, func(payload any) {
		fmtc.NewLine()
		spinner.Show("Starting downloading process")
	})

	dispatcher.AddHandler(backuper.EVENT_BACKUP_PROGRESS, func(payload any) {
		p := payload.(*backuper.ProgressInfo)
		spinner.Update("[%d%%] %s", p.Progress, p.Message)
	})

	dispatcher.AddHandler(backuper.EVENT_BACKUP_SAVING, func(payload any) {
		spinner.Done(true)
		spinner.Show("Fetching backup file")
	})

	dispatcher.AddHandler(backuper.EVENT_BACKUP_DONE, func(payload any) {
		spinner.Done(true)
	})

	dispatcher.AddHandler(uploader.EVENT_UPLOAD_STARTED, func(payload any) {
		spinner.Show("Uploading backup file to %s storage", payload)
	})

	dispatcher.AddHandler(uploader.EVENT_UPLOAD_PROGRESS, func(payload any) {
		p := payload.(*uploader.ProgressInfo)
		spinner.Update(
			"[%s] Uploading file (%s/%s)",
			fmtutil.PrettyPerc(p.Progress),
			fmtutil.PrettySize(p.Current),
			fmtutil.PrettySize(p.Total),
		)
	})

	dispatcher.AddHandler(uploader.EVENT_UPLOAD_DONE, func(payload any) {
		spinner.Update("Uploading file")
		spinner.Done(true)
		fmtc.NewLine()
	})
}
