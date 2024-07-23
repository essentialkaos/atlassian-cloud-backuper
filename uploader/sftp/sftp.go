package sftp

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
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

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/essentialkaos/katana"

	"github.com/essentialkaos/atlassian-cloud-backuper/uploader"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Config is configuration for SFTP uploader
type Config struct {
	Host   string
	User   string
	Key    []byte
	Path   string
	Mode   os.FileMode
	Secret *katana.Secret
}

// SFTPUploader is SFTP uploader instance
type SFTPUploader struct {
	config     *Config
	dispatcher *events.Dispatcher
}

// ////////////////////////////////////////////////////////////////////////////////// //

// validate backuper interface
var _ uploader.Uploader = (*SFTPUploader)(nil)

// ////////////////////////////////////////////////////////////////////////////////// //

// NewUploader creates new SFTP uploader instance
func NewUploader(config *Config) (*SFTPUploader, error) {
	err := config.Validate()

	if err != nil {
		return nil, err
	}

	return &SFTPUploader{config, nil}, nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// SetDispatcher sets events dispatcher
func (u *SFTPUploader) SetDispatcher(d *events.Dispatcher) {
	if u != nil {
		u.dispatcher = d
	}
}

// Upload uploads given file to SFTP storage
func (u *SFTPUploader) Upload(file, fileName string) error {
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
func (u *SFTPUploader) Write(r io.ReadCloser, fileName string, fileSize int64) error {
	u.dispatcher.DispatchAndWait(uploader.EVENT_UPLOAD_STARTED, "SFTP")

	var w io.Writer

	lastUpdate := time.Now()
	outputFile := path.Join(u.config.Path, fileName)

	log.Info(
		"Uploading backup file to %s@%s~%s/%s…",
		u.config.User, u.config.Host, u.config.Path, fileName,
	)

	sftpClient, err := u.connectToSFTP()

	if err != nil {
		return fmt.Errorf("Can't connect to SFTP: %v", err)
	}

	defer sftpClient.Close()

	_, err = sftpClient.Stat(u.config.Path)

	if err != nil {
		err = sftpClient.MkdirAll(u.config.Path)

		if err != nil {
			return fmt.Errorf("Can't create directory for backup: %v", err)
		}
	}

	fd, err := sftpClient.OpenFile(outputFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY)

	if err != nil {
		return fmt.Errorf("Can't create file of SFTP: %v", err)
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

	_, err = io.Copy(w, r)

	if err != nil {
		return fmt.Errorf("Can't upload file to SFTP: %v", err)
	}

	err = sftpClient.Chmod(outputFile, u.config.Mode)

	if err != nil {
		log.Error("Can't change file mode for uploaded file: %v", err)
	}

	log.Info("File successfully uploaded to SFTP!")
	u.dispatcher.DispatchAndWait(uploader.EVENT_UPLOAD_DONE, "SFTP")

	return nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// connectToSFTP connects to SFTP storage
func (u *SFTPUploader) connectToSFTP() (*sftp.Client, error) {
	signer, _ := ssh.ParsePrivateKey(u.config.Key)

	sshClient, err := ssh.Dial("tcp", u.config.Host, &ssh.ClientConfig{
		User:            u.config.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		Timeout:         5 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})

	if err != nil {
		return nil, fmt.Errorf("Can't connect to SSH: %v", err)
	}

	return sftp.NewClient(sshClient, sftp.UseConcurrentWrites(true))
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Validate validates configuration
func (c *Config) Validate() error {
	switch {
	case c == nil:
		return fmt.Errorf("Configuration validation error: config is nil")

	case c.Host == "":
		return fmt.Errorf("Configuration validation error: host is empty")

	case !strings.Contains(c.Host, ":"):
		return fmt.Errorf("Configuration validation error: host doesn't contain port number")

	case c.User == "":
		return fmt.Errorf("Configuration validation error: user is empty")

	case c.Path == "":
		return fmt.Errorf("Configuration validation error: path is empty")

	case len(c.Key) == 0:
		return fmt.Errorf("Configuration validation error: key is empty")

	case c.Mode == 0:
		return fmt.Errorf("Configuration validation error: invalid file mode %v", c.Mode)
	}

	_, err := ssh.ParsePrivateKey(c.Key)

	if err != nil {
		return fmt.Errorf("Configuration validation error: invalid key: %v", err)
	}

	return nil
}
