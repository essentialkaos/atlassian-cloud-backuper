package backuper

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"

	"github.com/essentialkaos/ek/v12/events"
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
	// Backup starts backup process
	Backup() error

	// SetDispatcher sets events dispatcher
	SetDispatcher(d *events.Dispatcher)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Config is backuper configuration struct
type Config struct {
	Account         string
	Email           string
	APIKey          string
	OutputFile      string
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
	case c.OutputFile == "":
		return ErrEmptyOutputFile
	}

	return nil
}

// AccountURL returns URL of account
func (c Config) AccountURL() string {
	return "https://" + c.Account + ".atlassian.net"
}
