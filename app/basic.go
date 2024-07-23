package app

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"

	"github.com/essentialkaos/ek/v13/events"
	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/log"
	"github.com/essentialkaos/ek/v13/options"
	"github.com/essentialkaos/ek/v13/path"
	"github.com/essentialkaos/ek/v13/spinner"
	"github.com/essentialkaos/ek/v13/terminal"

	knfu "github.com/essentialkaos/ek/v13/knf/united"

	"github.com/essentialkaos/atlassian-cloud-backuper/backuper"
	"github.com/essentialkaos/atlassian-cloud-backuper/uploader"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// startApp starts app in basic mode
func startApp(args options.Arguments) error {
	var dispatcher *events.Dispatcher

	if options.GetB(OPT_INTERACTIVE) {
		dispatcher = events.NewDispatcher()
		addEventsHandlers(dispatcher)
	}

	if knfu.GetS(STORAGE_ENCRYPTION_KEY) != "" {
		fmtc.NewLine()
		terminal.Warn("▲ Backup will be encrypted while uploading. You will not be able to use the")
		terminal.Warn("  backup if you lose the encryption key. Keep it in a safe place.")
	}

	defer temp.Clean()

	target := args.Get(0).String()
	bkpr, err := getBackuper(target)

	if err != nil {
		return fmt.Errorf("Can't start backuping process: %v", err)
	}

	bkpr.SetDispatcher(dispatcher)

	outputFileName := getOutputFileName(target)
	tmpDir, err := temp.MkDir()

	if err != nil {
		spinner.Done(false)
		return fmt.Errorf("Can't create temporary directory: %v", err)
	}

	tmpFile := path.Join(tmpDir, outputFileName)

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

// addEventsHandlers registers events handlers
func addEventsHandlers(dispatcher *events.Dispatcher) {
	dispatcher.AddHandler(backuper.EVENT_BACKUP_STARTED, func(payload any) {
		fmtc.NewLine()
		spinner.Show("Starting downloading process")
	})

	dispatcher.AddHandler(backuper.EVENT_BACKUP_PROGRESS, func(payload any) {
		p := payload.(*backuper.ProgressInfo)
		spinner.Update("{s}(%d%%){!} %s", p.Progress, p.Message)
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
			"{s}(%5s){!} Uploading file {s-}(%7s | %7s){!}",
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
