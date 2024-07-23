package fs

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

	"github.com/essentialkaos/ek/v13/events"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/log"
	"github.com/essentialkaos/ek/v13/passthru"
	"github.com/essentialkaos/ek/v13/path"

	"github.com/essentialkaos/katana"

	"github.com/essentialkaos/atlassian-cloud-backuper/uploader"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Config is configuration for FS uploader
type Config struct {
	Path   string
	Mode   os.FileMode
	Secret *katana.Secret
}

// FSUploader is FS uploader instance
type FSUploader struct {
	config     *Config
	dispatcher *events.Dispatcher
}

// ////////////////////////////////////////////////////////////////////////////////// //

// validate backuper interface
var _ uploader.Uploader = (*FSUploader)(nil)

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
func (u *FSUploader) Upload(file, fileName string) error {
	err := fsutil.ValidatePerms("FRS", file)

	if err != nil {
		return err
	}

	if !fsutil.IsExist(u.config.Path) {
		err = os.MkdirAll(u.config.Path, 0750)

		if err != nil {
			return fmt.Errorf("Can't create directory for backup: %w", err)
		}
	}

	fd, err := os.Open(file)

	if err != nil {
		return fmt.Errorf("Can't open backup file: %w", err)
	}

	defer fd.Close()

	err = u.Write(fd, fileName, fsutil.GetSize(file))

	if err != nil {
		return fmt.Errorf("Can't save backup file: %w", err)
	}

	return err
}

// Write writes data from given reader to given file
func (u *FSUploader) Write(r io.ReadCloser, fileName string, fileSize int64) error {
	u.dispatcher.DispatchAndWait(uploader.EVENT_UPLOAD_STARTED, "FS")

	var w io.Writer

	lastUpdate := time.Now()
	outputFile := path.Join(u.config.Path, fileName)

	log.Info("Copying backup file to %s…", u.config.Path)

	fd, err := os.OpenFile(outputFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, u.config.Mode)

	if err != nil {
		return err
	}

	w = fd

	if u.config.Secret != nil {
		sw, err := u.config.Secret.NewWriter(fd)

		if err != nil {
			return fmt.Errorf("Can't create encrypted writer: %w", err)
		}

		defer sw.Close()

		w = sw
	}

	if fileSize > 0 {
		pw := passthru.NewWriter(w, fileSize)

		pw.Update = func(n int) {
			if time.Since(lastUpdate) < 3*time.Second {
				return
			}

			u.dispatcher.Dispatch(
				uploader.EVENT_UPLOAD_PROGRESS,
				&uploader.ProgressInfo{
					Progress: pw.Progress(),
					Current:  pw.Current(),
					Total:    pw.Total(),
				},
			)

			lastUpdate = time.Now()
		}

		w = pw
	}

	_, err = io.Copy(bufio.NewWriter(w), r)

	if err != nil {
		return fmt.Errorf("File writing error: %w", err)
	}

	u.dispatcher.DispatchAndWait(uploader.EVENT_UPLOAD_DONE, "FS")
	log.Info("Backup successfully copied to %s", u.config.Path)

	return nil
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
