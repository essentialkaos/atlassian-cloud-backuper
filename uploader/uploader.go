package uploader

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"io"

	"github.com/essentialkaos/ek/v13/events"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	EVENT_UPLOAD_STARTED  = "upload-started"
	EVENT_UPLOAD_PROGRESS = "upload-progress"
	EVENT_UPLOAD_DONE     = "upload-done"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type ProgressInfo struct {
	Progress float64
	Current  int64
	Total    int64
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Uploader is generic uploader interface
type Uploader interface {
	// SetDispatcher sets events dispatcher
	SetDispatcher(d *events.Dispatcher)

	// Upload uploads given file to storage
	Upload(file, fileName string) error

	// Write writes data from given reader to given file
	Write(r io.ReadCloser, fileName string, fileSize int64) error
}

// ////////////////////////////////////////////////////////////////////////////////// //
