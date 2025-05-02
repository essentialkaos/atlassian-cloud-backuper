package confluence

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
	"strconv"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v13/events"
	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/log"
	"github.com/essentialkaos/ek/v13/req"

	"github.com/essentialkaos/atlassian-cloud-backuper/backuper"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type ConfluenceBackuper struct {
	config     *backuper.Config
	dispatcher *events.Dispatcher
}

type BackupPrefs struct {
	WithAttachments bool `json:"cbAttachments"`
	ForCloud        bool `json:"exportToCloud"`
}

type BackupProgressInfo struct {
	CurrentStatus              string `json:"currentStatus"`
	AlternativePercentage      string `json:"alternativePercentage"`
	Filename                   string `json:"fileName"`
	Size                       int    `json:"size"`
	Time                       int    `json:"time"`
	ConcurrentBackupInProgress bool   `json:"concurrentBackupInProgress"`
	IsOutdated                 bool   `json:"isOutdated"`
}

// ////////////////////////////////////////////////////////////////////////////////// //

// validate backuper interface
var _ backuper.Backuper = (*ConfluenceBackuper)(nil)

// ////////////////////////////////////////////////////////////////////////////////// //

func NewBackuper(config *backuper.Config) (*ConfluenceBackuper, error) {
	err := config.Validate()

	if err != nil {
		return nil, err
	}

	return &ConfluenceBackuper{config, nil}, nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// SetDispatcher sets events dispatcher
func (b *ConfluenceBackuper) SetDispatcher(d *events.Dispatcher) {
	if b != nil {
		b.dispatcher = d
	}
}

// Backup starts backup process
func (b *ConfluenceBackuper) Backup(outputFile string, force bool) error {
	_, err := b.Start(force)

	if err != nil {
		return err
	}

	backupFileURL, err := b.Progress("")

	if err != nil {
		return err
	}

	return b.Download(backupFileURL, outputFile)
}

// Start creates task for backuping data
func (b *ConfluenceBackuper) Start(force bool) (string, error) {
	log.Info("Starting Confluence backup process for account %s…", b.config.Account)
	log.Info("Checking for existing backup task…")

	info, _ := b.getBackupProgress()

	if !force && info != nil && !info.IsOutdated {
		log.Info(
			"Found previously created backup task",
			log.F{"backup-status", info.CurrentStatus},
			log.F{"backup-perc", info.AlternativePercentage},
			log.F{"backup-size", info.Size},
			log.F{"backup-file", info.Filename},
			log.F{"backup-outdated", info.IsOutdated},
		)
	} else {
		if force {
			log.Info("Starting new backup (force: %t)…", force)
		} else {
			log.Info("No previously created backup task or task is outdated, starting new backup…")
		}

		err := b.startBackup()

		if err != nil {
			return "", fmt.Errorf("Can't start backup: %w", err)
		}
	}

	b.dispatcher.DispatchAndWait(backuper.EVENT_BACKUP_STARTED, nil)

	return "", nil
}

// Progress monitors backup creation progress
func (b *ConfluenceBackuper) Progress(taskID string) (string, error) {
	var backupFileURL string

	errNum := 0
	lastProgress := ""
	start := time.Now()

	for range time.NewTicker(15 * time.Second).C {
		progressInfo, err := b.getBackupProgress()

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

		b.dispatcher.Dispatch(backuper.EVENT_BACKUP_PROGRESS, b.convertProgressInfo(progressInfo))

		if progressInfo.Size == 0 && progressInfo.AlternativePercentage >= lastProgress {
			log.Info(
				"(%s%%) Backup in progress: %s",
				progressInfo.AlternativePercentage,
				progressInfo.CurrentStatus,
			)
			lastProgress = progressInfo.AlternativePercentage
		}

		if progressInfo.Filename != "" {
			backupFileURL = progressInfo.Filename
			break
		}
	}

	return backupFileURL, nil
}

// IsBackupCreated returns true if backup created and ready for download
func (b *ConfluenceBackuper) IsBackupCreated() (bool, error) {
	progressInfo, err := b.getBackupProgress()

	if err != nil {
		return false, err
	}

	return progressInfo.Size != 0 && progressInfo.Filename != "", nil
}

// GetBackupFile returns name of created backup file
func (b *ConfluenceBackuper) GetBackupFile() (string, error) {
	progressInfo, err := b.getBackupProgress()

	if err != nil {
		return "", err
	}

	return progressInfo.Filename, nil
}

// Download downloads backup file
func (b *ConfluenceBackuper) Download(backupFile, outputFile string) error {
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
func (b *ConfluenceBackuper) GetReader(backupFile string) (io.ReadCloser, error) {
	backupFileURL := b.config.AccountURL() + "/wiki/download/" + backupFile

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
func (b *ConfluenceBackuper) startBackup() error {
	resp, err := req.Request{
		URL:         b.config.AccountURL() + "/wiki/rest/obm/1.0/runbackup",
		Auth:        req.AuthBasic{b.config.Email, b.config.APIKey},
		Accept:      req.CONTENT_TYPE_JSON,
		ContentType: req.CONTENT_TYPE_JSON,
		Body: &BackupPrefs{
			WithAttachments: b.config.WithAttachments,
			ForCloud:        b.config.ForCloud,
		},
	}.Post()

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("API returned non-ok status code (%d)", resp.StatusCode)
	}

	return nil
}

// getBackupProgress returns backup progress info
func (b *ConfluenceBackuper) getBackupProgress() (*BackupProgressInfo, error) {
	resp, err := req.Request{
		URL:         b.config.AccountURL() + "/wiki/rest/obm/1.0/getprogress",
		Auth:        req.AuthBasic{b.config.Email, b.config.APIKey},
		Accept:      req.CONTENT_TYPE_JSON,
		ContentType: req.CONTENT_TYPE_JSON,
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

	// Remove useless dot from the end of current status message
	progressInfo.CurrentStatus = strings.TrimRight(progressInfo.CurrentStatus, ".")

	return progressInfo, nil
}

// convertProgressInfo converts progress info from internal format to general backuper format
func (b *ConfluenceBackuper) convertProgressInfo(i *BackupProgressInfo) *backuper.ProgressInfo {
	perc, err := strconv.Atoi(strings.TrimRight(i.AlternativePercentage, "%"))

	if err != nil {
		return &backuper.ProgressInfo{Message: "Unknown status", Progress: 0}
	}

	return &backuper.ProgressInfo{
		Message:  i.CurrentStatus,
		Progress: perc,
	}
}

// downloadBackup downloads backup and saves it as a file
func (b *ConfluenceBackuper) downloadBackup(backupFile, outputFile string) error {
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
