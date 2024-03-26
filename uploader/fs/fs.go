package fs

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"

	"github.com/essentialkaos/ek/v12/events"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/log"
	"github.com/essentialkaos/ek/v12/path"

	"github.com/essentialkaos/atlassian-cloud-backuper/uploader"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Config is configuration for FS uploader
type Config struct {
	Path string
	Mode os.FileMode
}

// FSUploader is FS uploader instance
type FSUploader struct {
	config     *Config
	dispatcher *events.Dispatcher
}

// ////////////////////////////////////////////////////////////////////////////////// //

// NewUploader creates new FS uploader instance
func NewUploader(config *Config) (*FSUploader, error) {
	err := config.Validate()

	if err != nil {
		return nil, err
	}

	return &FSUploader{config, nil}, nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// SetDispatcher sets events dispatcher
func (u *FSUploader) SetDispatcher(d *events.Dispatcher) {
	if u != nil {
		u.dispatcher = d
	}
}

// Upload uploads given file to storage
func (u *FSUploader) Upload(file string) error {
	log.Info("Copying backup file to %s…", u.config.Path)

	u.dispatcher.DispatchAndWait(uploader.EVENT_UPLOAD_STARTED, "FS")

	err := fsutil.ValidatePerms("FRS", file)

	if err != nil {
		return err
	}

	fileName := path.Base(file)

	err = fsutil.CopyFile(file, path.Join(u.config.Path, fileName), u.config.Mode)

	u.dispatcher.DispatchAndWait(uploader.EVENT_UPLOAD_DONE, "FS")

	return err
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Validate validates configuration
func (c *Config) Validate() error {
	switch {
	case c == nil:
		return fmt.Errorf("Configuration validation error: config is nil")
	case c.Path == "":
		return fmt.Errorf("Configuration validation error: path is empty")
	case c.Mode == 0:
		return fmt.Errorf("Configuration validation error: invalid file mode %v", c.Mode)
	}

	return nil
}
