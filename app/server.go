package app

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v13/log"
	"github.com/essentialkaos/ek/v13/strutil"

	knfu "github.com/essentialkaos/ek/v13/knf/united"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// startServer starts app in server mode
func startServer() error {
	port := strutil.Q(os.Getenv("PORT"), knfu.GetS(SERVER_PORT))
	ip := knfu.GetS(SERVER_IP)

	log.Info(
		"Starting HTTP server",
		log.F{"server-ip", strutil.Q(ip, "localhost")},
		log.F{"server-port", port},
	)

	mux := http.NewServeMux()

	server := &http.Server{
		Addr:         ip + ":" + port,
		Handler:      mux,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	mux.HandleFunc("/create", createBackupHandler)
	mux.HandleFunc("/download", downloadBackupHandler)

	return server.ListenAndServe()
}

// ////////////////////////////////////////////////////////////////////////////////// //

// createHandler is handler for caching booking data
func createBackupHandler(rw http.ResponseWriter, r *http.Request) {
	updateResponseHeaders(rw)

	log.Info("Got create request", getConfigurationFields())

	target := strings.ToLower(r.URL.Query().Get("target"))
	token := r.URL.Query().Get("token")
	force := r.URL.Query().Get("force") != ""

	err := validateRequestQuery(target, token)

	if err != nil {
		log.Error("Invalid request query: %v", err.Error())
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	bkpr, err := getBackuper(target)

	if err != nil {
		sendUpdownPulse(false, err.Error())
		log.Error("Can't create backuper instance: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	taskID, err := bkpr.Start(force)

	if err != nil {
		sendUpdownPulse(false, err.Error())
		log.Error("Can't create backup: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
		return
	}

	log.Info("Backup request successfully created", log.F{"task-id", taskID})

	sendUpdownPulse(true, "create-backup")

	rw.WriteHeader(http.StatusOK)
}

// createHandler is handler for caching booking data
func downloadBackupHandler(rw http.ResponseWriter, r *http.Request) {
	var lf log.Fields

	updateResponseHeaders(rw)

	log.Info("Got download request", getConfigurationFields())

	target := strings.ToLower(r.URL.Query().Get("target"))
	token := r.URL.Query().Get("token")

	err := validateRequestQuery(target, token)

	if err != nil {
		sendUpdownPulse(false, err.Error())
		log.Error("Invalid request query: %v", err.Error())
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	bkpr, err := getBackuper(target)

	if err != nil {
		sendUpdownPulse(false, err.Error())
		log.Error("Can't create backuper instance: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	backupFile, err := bkpr.GetBackupFile()

	if err != nil {
		sendUpdownPulse(false, err.Error())
		log.Error("Can't find backup file: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
		return
	}

	log.Info("Starting downloading of backup", log.F{"backup-file", backupFile})

	br, err := bkpr.GetReader(backupFile)

	if err != nil {
		sendUpdownPulse(false, err.Error())
		log.Error("Can't get reader for backup file: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	updr, err := getUploader(target)

	if err != nil {
		sendUpdownPulse(false, err.Error())
		log.Error("Can't create uploader instance: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	outputFile := getOutputFileName(target)

	lf.Add(
		log.F{"backup-file", backupFile},
		log.F{"output-file", outputFile},
	)

	log.Info("Uploading backup to storage", lf)

	err = updr.Write(br, outputFile, 0)

	if err != nil {
		sendUpdownPulse(false, err.Error())
		log.Error("Can't upload backup file: %v", err, lf)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Info("Backup successfully uploaded", lf)

	sendUpdownPulse(true, "upload-backup")

	rw.WriteHeader(http.StatusOK)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// validateRequestQuery validates request query arguments
func validateRequestQuery(target, token string) error {
	switch {
	case target == "":
		return fmt.Errorf("target is empty")
	case knfu.GetS(SERVER_ACCESS_TOKEN) != "" && token == "":
		return fmt.Errorf("token is empty")
	case target != TARGET_JIRA && target != TARGET_CONFLUENCE:
		return fmt.Errorf("Unknown target %q", target)
	case knfu.GetS(SERVER_ACCESS_TOKEN) != "" && token == knfu.GetS(SERVER_ACCESS_TOKEN):
		return fmt.Errorf("Invalid access token")
	}

	return nil
}

// getConfigurationFields returns log fields
func getConfigurationFields() *log.Fields {
	lf := &log.Fields{}

	lf.Add(
		log.Field{"access-account", knfu.GetS(ACCESS_ACCOUNT)},
		log.Field{"access-email", knfu.GetS(ACCESS_EMAIL)},
		log.Field{"access-key", knfu.GetS(ACCESS_API_KEY) != ""},
		log.Field{"storage-type", knfu.GetS(STORAGE_TYPE)},
	)

	switch strings.ToLower(knfu.GetS(STORAGE_TYPE)) {
	case STORAGE_FS:
		lf.Add(
			log.Field{"storage-fs-path", knfu.GetS(STORAGE_FS_PATH)},
		)

	case STORAGE_SFTP:
		lf.Add(
			log.Field{"storage-sftp-host", knfu.GetS(STORAGE_SFTP_HOST)},
			log.Field{"storage-sftp-user", knfu.GetS(STORAGE_SFTP_USER)},
			log.Field{"storage-sftp-path", knfu.GetS(STORAGE_SFTP_PATH)},
		)

	case STORAGE_S3:
		lf.Add(
			log.Field{"storage-s3-host", knfu.GetS(STORAGE_S3_HOST)},
			log.Field{"storage-s3-bucket", knfu.GetS(STORAGE_S3_BUCKET)},
			log.Field{"storage-s3-path", knfu.GetS(STORAGE_S3_PATH)},
			log.Field{"storage-s3-key-id", knfu.GetS(STORAGE_S3_ACCESS_KEY)},
		)
	}

	return lf
}

// updateResponseHeaders updates response headers
func updateResponseHeaders(rw http.ResponseWriter) {
	rw.Header().Set("X-Powered-By", "EK|"+APP)
	rw.Header().Set("X-App-Version", VER)
}
