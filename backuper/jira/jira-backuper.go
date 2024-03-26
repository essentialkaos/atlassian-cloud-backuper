package jira

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/essentialkaos/ek/v12/events"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/log"
	"github.com/essentialkaos/ek/v12/req"

	"github.com/essentialkaos/atlassian-cloud-backuper/backuper"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type JiraBackuper struct {
	config     *backuper.Config
	dispatcher *events.Dispatcher
}

type BackupPrefs struct {
	WithAttachments bool `json:"cbAttachments"`
	ForCloud        bool `json:"exportToCloud"`
}

type BackupTaskInfo struct {
	TaskID string `json:"taskId"`
}

type BackupProgressInfo struct {
	Status     string `json:"status"`
	Desc       string `json:"description"`
	Message    string `json:"message"`
	Result     string `json:"result"`
	ExportType string `json:"exportType"`
	Progress   int    `json:"progress"`
}

// ////////////////////////////////////////////////////////////////////////////////// //

func NewBackuper(config *backuper.Config) (*JiraBackuper, error) {
	err := config.Validate()

	if err != nil {
		return nil, err
	}

	return &JiraBackuper{config, nil}, nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// SetDispatcher sets events dispatcher
func (b *JiraBackuper) SetDispatcher(d *events.Dispatcher) {
	if b != nil {
		b.dispatcher = d
	}
}

// Backup starts backup process
func (b *JiraBackuper) Backup() error {
	var err error
	var backupTaskID, backupFile string

	log.Info("Starting Jira backup process for account %s…", b.config.Account)
	log.Info("Checking for existing backup task…")

	start := time.Now()
	backupTaskID, _ = b.getLastTaskID()

	if backupTaskID != "" {
		log.Info("Found previously created backup task with ID %s", backupTaskID)
	} else {
		log.Info("No previously created task found, run backup…")

		backupTaskID, err = b.startBackup()

		if err != nil {
			return fmt.Errorf("Can't start backup: %w", err)
		}
	}

	b.dispatcher.DispatchAndWait(backuper.EVENT_BACKUP_STARTED, nil)

	errNum := 0
	lastProgress := -1

	for range time.NewTicker(15 * time.Second).C {
		progressInfo, err := b.getTaskProgress(backupTaskID)

		if err != nil {
			log.Error("Got error while checking progress: %w", err)
			errNum++

			if errNum > 10 {
				return fmt.Errorf("Can't download backup: too much errors")
			}
		} else {
			errNum = 0
		}

		if time.Since(start) > 6*time.Hour {
			return fmt.Errorf("Can't download backup: backup task took too much time")
		}

		b.dispatcher.Dispatch(
			backuper.EVENT_BACKUP_PROGRESS,
			&backuper.ProgressInfo{Message: progressInfo.Message, Progress: progressInfo.Progress},
		)

		if progressInfo.Progress < 100 && progressInfo.Progress >= lastProgress {
			log.Info("(%d%%) Backup in progress: %s", progressInfo.Progress, progressInfo.Message)
			lastProgress = progressInfo.Progress
		}

		if progressInfo.Progress >= 100 && progressInfo.Result != "" {
			backupFile = progressInfo.Result
			break
		}
	}

	log.Info("Backup is ready for download, fetching file…")
	log.Info("Writting backup file into %s", b.config.OutputFile)

	b.dispatcher.DispatchAndWait(backuper.EVENT_BACKUP_SAVING, nil)

	err = b.downloadBackup(backupFile)

	if err != nil {
		return err
	}

	b.dispatcher.DispatchAndWait(backuper.EVENT_BACKUP_DONE, nil)

	log.Info(
		"Backup successfully saved (size: %s)",
		fmtutil.PrettySize(fsutil.GetSize(b.config.OutputFile)),
	)

	return nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// startBackup starts backup process
func (b *JiraBackuper) startBackup() (string, error) {
	resp, err := req.Request{
		URL:               b.config.AccountURL() + "/rest/backup/1/export/runbackup",
		BasicAuthUsername: b.config.Email,
		BasicAuthPassword: b.config.APIKey,
		Accept:            req.CONTENT_TYPE_JSON,
		ContentType:       req.CONTENT_TYPE_JSON,
		Body: &BackupPrefs{
			WithAttachments: b.config.WithAttachments,
			ForCloud:        b.config.ForCloud,
		},
	}.Post()

	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned non-ok status code (%d)", resp.StatusCode)
	}

	backupInfo := &BackupTaskInfo{}
	err = resp.JSON(backupInfo)

	if err != nil {
		return "", fmt.Errorf("Can't decode API response: %v", err)
	}

	return backupInfo.TaskID, nil
}

// getLastTaskID returns ID of the last task for backup
func (b *JiraBackuper) getLastTaskID() (string, error) {
	resp, err := req.Request{
		URL:               b.config.AccountURL() + "/rest/backup/1/export/lastTaskId",
		BasicAuthUsername: b.config.Email,
		BasicAuthPassword: b.config.APIKey,
		Accept:            req.CONTENT_TYPE_JSON,
		ContentType:       req.CONTENT_TYPE_JSON,
		AutoDiscard:       true,
	}.Get()

	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned non-ok status code (%d)", resp.StatusCode)
	}

	return resp.String(), nil
}

// getTaskProgress returns progress for task
func (b *JiraBackuper) getTaskProgress(taskID string) (*BackupProgressInfo, error) {
	resp, err := req.Request{
		URL:               b.config.AccountURL() + "/rest/backup/1/export/getProgress",
		BasicAuthUsername: b.config.Email,
		BasicAuthPassword: b.config.APIKey,
		Accept:            req.CONTENT_TYPE_JSON,
		ContentType:       req.CONTENT_TYPE_JSON,
		Query:             req.Query{"taskId": taskID},
		AutoDiscard:       true,
	}.Get()

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned non-ok status code (%d)", resp.StatusCode)
	}

	progressInfo := &BackupProgressInfo{}
	err = resp.JSON(progressInfo)

	if err != nil {
		return nil, fmt.Errorf("Can't decode API response: %v", err)
	}

	return progressInfo, nil
}

// downloadBackup downloads backup and saves it as a file
func (b *JiraBackuper) downloadBackup(backupFile string) error {
	backupFileURL := b.config.AccountURL() + "/plugins/servlet/" + backupFile

	log.Debug("Downloading file from %s", backupFileURL)

	resp, err := req.Request{
		URL:               backupFileURL,
		BasicAuthUsername: b.config.Email,
		BasicAuthPassword: b.config.APIKey,
		AutoDiscard:       true,
	}.Get()

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("API returned non-ok status code (%d)", resp.StatusCode)
	}

	fd, err := os.OpenFile(b.config.OutputFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)

	if err != nil {
		return fmt.Errorf("Can't open file for saving data: %w", err)
	}

	defer fd.Close()

	w := bufio.NewWriter(fd)
	_, err = io.Copy(w, resp.Body)

	if err != nil {
		return fmt.Errorf("File writting error: %w", err)
	}

	w.Flush()

	return nil
}
