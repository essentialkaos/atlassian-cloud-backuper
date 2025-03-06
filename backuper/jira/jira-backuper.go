package jira

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2025 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/essentialkaos/ek/v13/events"
	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/log"
	"github.com/essentialkaos/ek/v13/req"

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

// validate backuper interface
var _ backuper.Backuper = (*JiraBackuper)(nil)

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
func (b *JiraBackuper) Backup(outputFile string, force bool) error {
	backupTaskID, err := b.Start(force)

	if err != nil {
		return err
	}

	backupFileURL, err := b.Progress(backupTaskID)

	if err != nil {
		return err
	}

	return b.Download(backupFileURL, outputFile)
}

// Start creates task for backuping data
func (b *JiraBackuper) Start(force bool) (string, error) {
	var err error
	var backupTaskID string

	log.Info("Starting Jira backup process for account %s…", b.config.Account)

	if !force {
		log.Info("Checking for existing backup task…")

		backupTaskID, _ = b.getLastTaskID()

		if backupTaskID == "" {
			log.Info("No previously created task found, starting new backup…")
		}
	} else {
		log.Info("Starting new backup…")
	}

	if backupTaskID == "" {
		backupTaskID, err = b.startBackup()

		if err != nil {
			return "", fmt.Errorf("Can't start backup: %w", err)
		}
	}

	b.dispatcher.DispatchAndWait(backuper.EVENT_BACKUP_STARTED, nil)

	return backupTaskID, nil
}

// Progress monitors backup creation progress
func (b *JiraBackuper) Progress(taskID string) (string, error) {
	var backupFileURL string

	errNum := 0
	lastProgress := -1
	start := time.Now()

	for range time.NewTicker(15 * time.Second).C {
		progressInfo, err := b.getTaskProgress(taskID)

		if err != nil {
			log.Error("Got error while checking progress: %w", err)
			errNum++

			if errNum > 10 {
				return "", fmt.Errorf("Can't download backup: too much errors")
			}
		} else {
			errNum = 0
		}

		if time.Since(start) > 6*time.Hour {
			return "", fmt.Errorf("Can't download backup: backup task took too much time")
		}

		if progressInfo == nil {
			continue
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
			backupFileURL = progressInfo.Result
			break
		}
	}

	return backupFileURL, nil
}

// IsBackupCreated returns true if backup created and ready for download
func (b *JiraBackuper) IsBackupCreated() (bool, error) {
	backupTaskID, _ := b.getLastTaskID()

	if backupTaskID == "" {
		return false, nil
	}

	progressInfo, err := b.getTaskProgress(backupTaskID)

	if err != nil {
		return false, err
	}

	return progressInfo.Progress >= 100 && progressInfo.Result != "", nil
}

// GetBackupFile returns name of created backup file
func (b *JiraBackuper) GetBackupFile() (string, error) {
	backupTaskID, _ := b.getLastTaskID()

	if backupTaskID == "" {
		return "", fmt.Errorf("Can't find backup task ID")
	}

	progressInfo, err := b.getTaskProgress(backupTaskID)

	if err != nil {
		return "", err
	}

	return progressInfo.Result, nil
}

// Download downloads backup file
func (b *JiraBackuper) Download(backupFile, outputFile string) error {
	log.Info("Backup is ready for download, fetching file…")
	log.Info("Writing backup file into %s", outputFile)

	b.dispatcher.DispatchAndWait(backuper.EVENT_BACKUP_SAVING, nil)

	err := b.downloadBackup(backupFile, outputFile)

	if err != nil {
		return err
	}

	b.dispatcher.DispatchAndWait(backuper.EVENT_BACKUP_DONE, nil)

	log.Info(
		"Backup successfully saved (size: %s)",
		fmtutil.PrettySize(fsutil.GetSize(outputFile)),
	)

	return nil
}

// GetReader returns reader for given backup file
func (b *JiraBackuper) GetReader(backupFile string) (io.ReadCloser, error) {
	backupFileURL := b.config.AccountURL() + "/plugins/servlet/" + backupFile

	log.Debug("Downloading file from %s", backupFileURL)

	resp, err := req.Request{
		URL:         backupFileURL,
		Auth:        req.AuthBasic{b.config.Email, b.config.APIKey},
		AutoDiscard: true,
	}.Get()

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned non-ok status code (%d)", resp.StatusCode)
	}

	return resp.Body, nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// startBackup starts backup process
func (b *JiraBackuper) startBackup() (string, error) {
	resp, err := req.Request{
		URL:         b.config.AccountURL() + "/rest/backup/1/export/runbackup",
		Auth:        req.AuthBasic{b.config.Email, b.config.APIKey},
		Accept:      req.CONTENT_TYPE_JSON,
		ContentType: req.CONTENT_TYPE_JSON,
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
		URL:         b.config.AccountURL() + "/rest/backup/1/export/lastTaskId",
		Auth:        req.AuthBasic{b.config.Email, b.config.APIKey},
		Accept:      req.CONTENT_TYPE_JSON,
		ContentType: req.CONTENT_TYPE_JSON,
		AutoDiscard: true,
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
		URL:         b.config.AccountURL() + "/rest/backup/1/export/getProgress",
		Auth:        req.AuthBasic{b.config.Email, b.config.APIKey},
		Accept:      req.CONTENT_TYPE_JSON,
		ContentType: req.CONTENT_TYPE_JSON,
		Query:       req.Query{"taskId": taskID},
		AutoDiscard: true,
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
func (b *JiraBackuper) downloadBackup(backupFile, outputFile string) error {
	r, err := b.GetReader(backupFile)

	if err != nil {
		return err
	}

	defer r.Close()

	fd, err := os.OpenFile(outputFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)

	if err != nil {
		return fmt.Errorf("Can't open file for saving data: %w", err)
	}

	defer fd.Close()

	w := bufio.NewWriter(fd)
	_, err = io.Copy(w, r)

	if err != nil {
		return fmt.Errorf("File writing error: %w", err)
	}

	w.Flush()

	return nil
}
