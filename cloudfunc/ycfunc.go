package main

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/log"
	"github.com/essentialkaos/ek/v12/req"
	"github.com/essentialkaos/ek/v12/strutil"
	"github.com/essentialkaos/ek/v12/timeutil"

	knfu "github.com/essentialkaos/ek/v12/knf/united"

	"github.com/essentialkaos/atlassian-cloud-backuper/cli"

	"github.com/essentialkaos/atlassian-cloud-backuper/backuper"
	"github.com/essentialkaos/atlassian-cloud-backuper/backuper/confluence"
	"github.com/essentialkaos/atlassian-cloud-backuper/backuper/jira"

	"github.com/essentialkaos/atlassian-cloud-backuper/uploader"
	"github.com/essentialkaos/atlassian-cloud-backuper/uploader/fs"
	"github.com/essentialkaos/atlassian-cloud-backuper/uploader/s3"
	"github.com/essentialkaos/atlassian-cloud-backuper/uploader/sftp"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	STAGE_CREATE   = "create"
	STAGE_DOWNLOAD = "download"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type Data struct {
	Messages []*Message `json:"messages"`
}

type Message struct {
	Metadata *Metadata `json:"event_metadata"`
	Details  *Details  `json:"details"`
}

type Metadata struct {
	EventType string `json:"event_type"`
}

type Details struct {
	TriggerID string `json:"trigger_id"`
	Payload   string `json:"payload"`
}

// ////////////////////////////////////////////////////////////////////////////////// //

// main is used for compilation errors
func main() {
	return
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Request is handler for HTTP requests
func Request(rw http.ResponseWriter, r *http.Request) {
	req.SetUserAgent("AtlassianCloudBackuper|YCFunction", cli.VER)
	rw.Header().Set("X-Version", cli.VER)

	log.Global.UseJSON = true
	log.Global.WithCaller = true

	defer log.Flush()

	if !validateConfiguration() {
		rw.WriteHeader(500)
		return
	}

	if !validateRequest(r) {
		rw.WriteHeader(400)
		return
	}

	target := strings.ToLower(r.URL.Query().Get("target"))
	stage := strings.ToLower(r.URL.Query().Get("stage"))

	log.Info("Got backup request", log.F{"target", target}, log.F{"stage", stage})

	var ok bool

	switch stage {
	case STAGE_CREATE:
		ok = createBackupRequest(target)
	case STAGE_DOWNLOAD:
		ok = downloadBackupData(target)
	}

	if ok {
		rw.WriteHeader(200)
	} else {
		rw.WriteHeader(500)
	}
}

// Trigger is handler for timer trigger
func Trigger(ctx context.Context, data *Data) error {
	log.Global.UseJSON = true
	log.Global.WithCaller = true

	defer log.Flush()

	if !validatePayload(data) {
		return fmt.Errorf("Error while trigger event validation")
	}

	target, stage, _ := data.GetPayload()

	log.Info("Got trigger event", log.F{"target", target}, log.F{"stage", stage})

	var ok bool

	switch stage {
	case STAGE_CREATE:
		ok = createBackupRequest(target)
	case STAGE_DOWNLOAD:
		ok = downloadBackupData(target)
	}

	if !ok {
		return fmt.Errorf("Can't handle event")
	}

	return nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// GetPayload extracts target and stage from trigger payload
func (d *Data) GetPayload() (string, string, bool) {
	payload := d.Messages[0].Details.Payload
	return strings.Cut(payload, ";")
}

// ////////////////////////////////////////////////////////////////////////////////// //

// validateRequest validates request data
func validateRequest(r *http.Request) bool {
	if r.Method != req.GET {
		log.Error("Invalid request: Unsupported method")
		return false
	}

	target := strings.ToLower(r.URL.Query().Get("target"))
	stage := strings.ToLower(r.URL.Query().Get("stage"))

	switch target {
	case cli.TARGET_JIRA, cli.TARGET_CONFLUENCE:
		// ok

	case "":
		log.Error("Invalid request: Target is empty")
		return false

	default:
		log.Error("Invalid request: Unsupported target", log.F{"target", target})
		return false
	}

	switch stage {
	case STAGE_CREATE, STAGE_DOWNLOAD:
		// ok

	case "":
		log.Error("Invalid request: Stage is empty")
		return false

	default:
		log.Error("Invalid request: Unsupported stage", log.F{"stage", stage})
		return false
	}

	return true
}

// validatePayload validates trigger payload
func validatePayload(data *Data) bool {
	switch {
	case data == nil:
		log.Error("Trigger data is nil")
		return false

	case len(data.Messages) == 0:
		log.Error("No messages in trigger event")
		return false

	case data.Messages[0].Metadata == nil:
		log.Error("No metadata in message #0")
		return false

	case data.Messages[0].Metadata.EventType != "yandex.cloud.events.serverless.triggers.TimerMessage":
		log.Error("Unsupported event type", log.F{"event-type", data.Messages[0].Metadata.EventType})
		return false

	case data.Messages[0].Details == nil:
		log.Error("No details in message #0")
		return false

	case data.Messages[0].Details.Payload == "":
		log.Error("Payload is empty")
		return false

	case !strings.Contains(data.Messages[0].Details.Payload, ";"):
		log.Error("Payload doesn't have ';' separator", log.F{"payload", data.Messages[0].Details.Payload})
		return false
	}

	target, stage, _ := data.GetPayload()

	switch target {
	case cli.TARGET_JIRA, cli.TARGET_CONFLUENCE:
		// ok

	case "":
		log.Error("Invalid trigger payload: Target is empty")
		return false

	default:
		log.Error("Invalid trigger payload: Unsupported target", log.F{"target", target})
		return false
	}

	switch stage {
	case STAGE_CREATE, STAGE_DOWNLOAD:
		// ok

	case "":
		log.Error("Invalid trigger payload: Stage is empty")
		return false

	default:
		log.Error("Invalid trigger payload: Unsupported stage", log.F{"stage", stage})
		return false
	}

	return true
}

// validateConfiguration validates configuration
func validateConfiguration() bool {
	switch {
	case getEnvVar(cli.ACCESS_ACCOUNT) == "":
		log.Error("Invalid configuration: ACCESS_ACCOUNT is empty")
		return false

	case getEnvVar(cli.ACCESS_EMAIL) == "":
		log.Error("Invalid configuration: ACCESS_EMAIL is empty")
		return false

	case getEnvVar(cli.ACCESS_API_KEY) == "":
		log.Error("Invalid configuration: ACCESS_API_KEY is empty")
		return false

	case getEnvVar(cli.STORAGE_TYPE) == "":
		log.Error("Invalid configuration: STORAGE_TYPE is empty")
		return false
	}

	switch getEnvVar(cli.STORAGE_TYPE) {
	case "fs", "sftp", "s3":
		// ok
	default:
		log.Error("Invalid configuration: invalid STORAGE_TYPE value %q", getEnvVar(cli.STORAGE_TYPE))
		return false
	}

	if getEnvVar(cli.STORAGE_TYPE) == "s3" {
		switch {
		case getEnvVar(cli.STORAGE_S3_ACCESS_KEY) == "":
			log.Error("Invalid configuration: STORAGE_S3_ACCESS_KEY is empty")
			return false
		case getEnvVar(cli.STORAGE_S3_SECRET_KEY) == "":
			log.Error("Invalid configuration: STORAGE_S3_SECRET_KEY is empty")
			return false
		case getEnvVar(cli.STORAGE_S3_BUCKET) == "":
			log.Error("Invalid configuration: STORAGE_S3_BUCKET is empty")
			return false
		}
	} else if getEnvVar(cli.STORAGE_TYPE) == "sftp" {
		switch {
		case getEnvVar(cli.STORAGE_SFTP_HOST) == "":
			log.Error("Invalid configuration: STORAGE_SFTP_HOST is empty")
			return false
		case getEnvVar(cli.STORAGE_SFTP_USER) == "":
			log.Error("Invalid configuration: STORAGE_SFTP_USER is empty")
			return false
		case getEnvVar(cli.STORAGE_SFTP_KEY) == "":
			log.Error("Invalid configuration: STORAGE_SFTP_KEY is empty")
			return false
		case getEnvVar(cli.STORAGE_SFTP_PATH) == "":
			log.Error("Invalid configuration: STORAGE_SFTP_PATH is empty")
			return false
		}
	} else {
		if getEnvVar(cli.STORAGE_FS_PATH) == "" {
			log.Error("Invalid configuration: STORAGE_FS_PATH is empty")
			return false
		}
	}

	return true
}

// createBackupRequest sends request for creating backup
func createBackupRequest(target string) bool {
	bkpr, err := getBackuper(target)

	if err != nil {
		log.Error("Can't create backuper instance: %v", err)
		return false
	}

	taskID, err := bkpr.Start()

	if err != nil {
		log.Error("Can't create backup: %v", err)
		return false
	}

	log.Info("Backup request successfully created", log.F{"task-id", taskID})

	return true
}

// downloadBackupData downloads backup data and upload it to storage
func downloadBackupData(target string) bool {
	bkpr, err := getBackuper(target)

	if err != nil {
		log.Error("Can't create backuper instance: %v", err)
		return false
	}

	backupFile, err := bkpr.GetBackupFile()

	if err != nil {
		log.Error("Can't find backup file: %v", err)
		return false
	}

	log.Info("Start downloading of backup", log.F{"backup-file", backupFile})

	r, err := bkpr.GetReader(backupFile)

	if err != nil {
		log.Error("Can't get reader for backup file: %v", err)
		return false
	}

	updr, err := getUploader(target)

	if err != nil {
		log.Error("Can't create uploader instance: %v", err)
		return false
	}

	outputFile := getOutputFile(target)

	log.Info(
		"Uploading backup to storage",
		log.F{"backup-file", backupFile},
		log.F{"output-file", outputFile},
	)

	err = updr.Write(r, outputFile)

	if err != nil {
		log.Error(
			"Can't upload backup file: %v", err,
			log.F{"backup-file", backupFile},
			log.F{"output-file", outputFile},
		)
		return false
	}

	return true
}

// getBackuper returns backuper instance
func getBackuper(target string) (backuper.Backuper, error) {
	var err error
	var bkpr backuper.Backuper

	config, err := getBackuperConfig(target)

	if err != nil {
		return nil, err
	}

	switch target {
	case cli.TARGET_JIRA:
		bkpr, err = jira.NewBackuper(config)
	case cli.TARGET_CONFLUENCE:
		bkpr, err = confluence.NewBackuper(config)
	default:
		return nil, fmt.Errorf("Unknown or unsupported target %q", target)
	}

	return bkpr, err
}

// getBackuperConfig returns configuration for backuper
func getBackuperConfig(target string) (*backuper.Config, error) {
	switch target {
	case cli.TARGET_JIRA:
		return &backuper.Config{
			Account:         getEnvVar(cli.ACCESS_ACCOUNT),
			Email:           getEnvVar(cli.ACCESS_EMAIL),
			APIKey:          getEnvVar(cli.ACCESS_API_KEY),
			WithAttachments: getEnvVarFlag(cli.JIRA_INCLUDE_ATTACHMENTS, true),
			ForCloud:        getEnvVarFlag(cli.JIRA_CLOUD_FORMAT, true),
		}, nil

	case cli.TARGET_CONFLUENCE:
		return &backuper.Config{
			Account:         getEnvVar(cli.ACCESS_ACCOUNT),
			Email:           getEnvVar(cli.ACCESS_EMAIL),
			APIKey:          getEnvVar(cli.ACCESS_API_KEY),
			WithAttachments: getEnvVarFlag(cli.CONFLUENCE_INCLUDE_ATTACHMENTS, true),
			ForCloud:        getEnvVarFlag(cli.CONFLUENCE_CLOUD_FORMAT, true),
		}, nil
	}

	return nil, fmt.Errorf("Unknown or unsupported target %q", target)
}

// getUploader returns uploader instance
func getUploader(target string) (uploader.Uploader, error) {
	var err error
	var updr uploader.Uploader

	switch getEnvVar(cli.STORAGE_TYPE) {
	case "fs":
		updr, err = fs.NewUploader(&fs.Config{
			Path: path.Join(getEnvVar(cli.STORAGE_FS_PATH), target),
			Mode: parseMode(getEnvVar(cli.STORAGE_FS_MODE, "0640")),
		})

	case "sftp":
		key, err := base64.StdEncoding.DecodeString(getEnvVar(cli.STORAGE_SFTP_KEY))

		if err != nil {
			return nil, err
		}

		updr, err = sftp.NewUploader(&sftp.Config{
			Host: getEnvVar(cli.STORAGE_SFTP_HOST),
			User: getEnvVar(cli.STORAGE_SFTP_USER),
			Key:  key,
			Path: path.Join(getEnvVar(cli.STORAGE_SFTP_PATH), target),
			Mode: parseMode(getEnvVar(cli.STORAGE_SFTP_MODE, "0640")),
		})

	case "s3":
		updr, err = s3.NewUploader(&s3.Config{
			Host:        getEnvVar(cli.STORAGE_S3_HOST, "storage.yandexcloud.net"),
			Region:      getEnvVar(cli.STORAGE_S3_REGION, "ru-central1"),
			AccessKeyID: getEnvVar(cli.STORAGE_S3_ACCESS_KEY),
			SecretKey:   getEnvVar(cli.STORAGE_S3_SECRET_KEY),
			Bucket:      getEnvVar(cli.STORAGE_S3_BUCKET),
			Path:        path.Join(getEnvVar(cli.STORAGE_S3_PATH), target),
		})
	}

	return updr, err
}

// getOutputFile returns name of output file
func getOutputFile(target string) string {
	var template string

	switch target {
	case cli.TARGET_JIRA:
		template = strutil.Q(getEnvVar(cli.JIRA_OUTPUT_FILE), `jira-backup-%Y-%m-%d`) + ".zip"
	case cli.TARGET_CONFLUENCE:
		template = strutil.Q(getEnvVar(cli.CONFLUENCE_OUTPUT_FILE), `confluence-backup-%Y-%m-%d`) + ".zip"
	}

	return timeutil.Format(time.Now(), template)
}

// getEnvVar reads environment variable
func getEnvVar(name string, defs ...string) string {
	value := os.Getenv(knfu.ToEnvVar(name))

	if value == "" && len(defs) > 0 {
		return defs[0]
	}

	return value
}

// getEnvVarFlag reads environment variable with flag
func getEnvVarFlag(name string, def bool) bool {
	switch strings.ToLower(getEnvVar(name)) {
	case "n", "no", "false", "0":
		return false
	case "y", "yes", "true", "1":
		return true
	}

	return def
}

// parseMode parses file mode
func parseMode(v string) os.FileMode {
	m, err := strconv.ParseUint(v, 8, 32)

	if err != nil {
		return 0600
	}

	return os.FileMode(m)
}
