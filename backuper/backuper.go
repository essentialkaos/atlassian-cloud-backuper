package backuper

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"io"

	"github.com/essentialkaos/ek/v13/events"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	EVENT_BACKUP_STARTED  = "backup-started"
	EVENT_BACKUP_PROGRESS = "backup-progress"
	EVENT_BACKUP_SAVING   = "backup-saving"
	EVENT_BACKUP_DONE     = "backup-done"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Backuper is generic backuper interface
type Backuper interface {
	// Backup runs backup process
	Backup(outputFile string) error

	// SetDispatcher sets events dispatcher
	SetDispatcher(d *events.Dispatcher)

	// Start creates task for backuping data
	Start() (string, error)

	// Progress monitors backup creation progress
	Progress(taskID string) (string, error)

	// Download downloads backup file
	Download(backupFile, outputFile string) error

	// GetReader returns reader for given backup file
	GetReader(backupFile string) (io.ReadCloser, error)

	// GetBackupFile returns name of created backup file
	GetBackupFile() (string, error)

	// IsBackupCreated returns true if backup created and ready for download
	IsBackupCreated() (bool, error)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Config is backuper configuration struct
type Config struct {
	Account         string
	Email           string
	APIKey          string
	WithAttachments bool
	ForCloud        bool
}

type ProgressInfo struct {
	Message  string
	Progress int
}

// ////////////////////////////////////////////////////////////////////////////////// //

var (
	ErrEmptyAccount    = fmt.Errorf("Configuration validation error: account is empty")
	ErrEmptyEmail      = fmt.Errorf("Configuration validation error: email is empty")
	ErrEmptyAPIKey     = fmt.Errorf("Configuration validation error: API key is empty")
	ErrEmptyOutputFile = fmt.Errorf("Configuration validation error: output file is empty")
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Validate validates configuration struct
func (c Config) Validate() error {
	switch {
	case c.Account == "":
		return ErrEmptyAccount
	case c.Email == "":
		return ErrEmptyEmail
	case c.APIKey == "":
		return ErrEmptyAPIKey
	}

	return nil
}

// AccountURL returns URL of account
func (c Config) AccountURL() string {
	return "https://" + c.Account + ".atlassian.net"
}
