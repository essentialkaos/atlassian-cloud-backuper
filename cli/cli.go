package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/errutil"
	"github.com/essentialkaos/ek/v12/events"
	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/knf"
	"github.com/essentialkaos/ek/v12/log"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/path"
	"github.com/essentialkaos/ek/v12/req"
	"github.com/essentialkaos/ek/v12/spinner"
	"github.com/essentialkaos/ek/v12/support"
	"github.com/essentialkaos/ek/v12/support/deps"
	"github.com/essentialkaos/ek/v12/system/container"
	"github.com/essentialkaos/ek/v12/terminal/tty"
	"github.com/essentialkaos/ek/v12/timeutil"
	"github.com/essentialkaos/ek/v12/tmp"
	"github.com/essentialkaos/ek/v12/usage"
	"github.com/essentialkaos/ek/v12/usage/completion/bash"
	"github.com/essentialkaos/ek/v12/usage/completion/fish"
	"github.com/essentialkaos/ek/v12/usage/completion/zsh"
	"github.com/essentialkaos/ek/v12/usage/man"
	"github.com/essentialkaos/ek/v12/usage/update"

	knfu "github.com/essentialkaos/ek/v12/knf/united"
	knfv "github.com/essentialkaos/ek/v12/knf/validators"
	knff "github.com/essentialkaos/ek/v12/knf/validators/fs"
	knfn "github.com/essentialkaos/ek/v12/knf/validators/network"

	"github.com/essentialkaos/atlassian-cloud-backuper/backuper"
	"github.com/essentialkaos/atlassian-cloud-backuper/backuper/confluence"
	"github.com/essentialkaos/atlassian-cloud-backuper/backuper/jira"

	"github.com/essentialkaos/atlassian-cloud-backuper/uploader"
	"github.com/essentialkaos/atlassian-cloud-backuper/uploader/fs"
	"github.com/essentialkaos/atlassian-cloud-backuper/uploader/s3"
	"github.com/essentialkaos/atlassian-cloud-backuper/uploader/sftp"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Basic utility info
const (
	APP  = "Atlassian Cloud Backuper"
	VER  = "0.0.2"
	DESC = "Tool for backuping Atlassian cloud services (Jira and Confluence)"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Options
const (
	OPT_CONFIG      = "c:config"
	OPT_INTERACTIVE = "I:interactive"
	OPT_NO_COLOR    = "nc:no-color"
	OPT_HELP        = "h:help"
	OPT_VER         = "v:version"

	OPT_VERB_VER     = "vv:verbose-version"
	OPT_COMPLETION   = "completion"
	OPT_GENERATE_MAN = "generate-man"
)

const (
	ACCESS_ACCOUNT                 = "access:account"
	ACCESS_EMAIL                   = "access:email"
	ACCESS_API_KEY                 = "access:api-key"
	STORAGE_TYPE                   = "storage:type"
	STORAGE_FS_PATH                = "storage-fs:path"
	STORAGE_FS_MODE                = "storage-fs:mode"
	STORAGE_SFTP_HOST              = "storage-sftp:host"
	STORAGE_SFTP_USER              = "storage-sftp:user"
	STORAGE_SFTP_KEY               = "storage-sftp:key"
	STORAGE_SFTP_PATH              = "storage-sftp:path"
	STORAGE_SFTP_MODE              = "storage-sftp:mode"
	STORAGE_S3_HOST                = "storage-s3:host"
	STORAGE_S3_REGION              = "storage-s3:region"
	STORAGE_S3_ACCESS_KEY          = "storage-s3:access-key"
	STORAGE_S3_SECRET_KEY          = "storage-s3:secret-key"
	STORAGE_S3_BUCKET              = "storage-s3:bucket"
	STORAGE_S3_PATH                = "storage-s3:path"
	JIRA_OUTPUT_FILE               = "jira:output-file"
	JIRA_INCLUDE_ATTACHMENTS       = "jira:include-attachments"
	JIRA_CLOUD_FORMAT              = "jira:cloud-format"
	CONFLUENCE_OUTPUT_FILE         = "confluence:output-file"
	CONFLUENCE_INCLUDE_ATTACHMENTS = "confluence:include-attachments"
	CONFLUENCE_CLOUD_FORMAT        = "confluence:cloud-format"
	TEMP_DIR                       = "temp:dir"
	LOG_DIR                        = "log:dir"
	LOG_FILE                       = "log:file"
	LOG_FORMAT                     = "log:format"
	LOG_MODE                       = "log:perms"
	LOG_LEVEL                      = "log:level"
)

const (
	TARGET_JIRA       = "jira"
	TARGET_CONFLUENCE = "confluence"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// optMap contains information about all supported options
var optMap = options.Map{
	OPT_CONFIG:      {Value: "/etc/atlassian-cloud-backuper.knf"},
	OPT_INTERACTIVE: {Type: options.BOOL},
	OPT_NO_COLOR:    {Type: options.BOOL},
	OPT_HELP:        {Type: options.BOOL},
	OPT_VER:         {Type: options.MIXED},

	OPT_VERB_VER:     {Type: options.BOOL},
	OPT_COMPLETION:   {},
	OPT_GENERATE_MAN: {Type: options.BOOL},
}

// temp is temp data manager
var temp *tmp.Temp

// color tags for app name and version
var colorTagApp, colorTagVer string

// ////////////////////////////////////////////////////////////////////////////////// //

// Run is main utility function
func Run(gitRev string, gomod []byte) {
	preConfigureUI()
	addExtraOptions(optMap)

	args, errs := options.Parse(optMap)

	if len(errs) != 0 {
		printError(errs[0].Error())
		os.Exit(1)
	}

	configureUI()

	switch {
	case options.Has(OPT_COMPLETION):
		os.Exit(printCompletion())
	case options.Has(OPT_GENERATE_MAN):
		printMan()
		os.Exit(0)
	case options.GetB(OPT_VER):
		genAbout(gitRev).Print(options.GetS(OPT_VER))
		os.Exit(0)
	case options.GetB(OPT_VERB_VER):
		support.Collect(APP, VER).
			WithRevision(gitRev).
			WithDeps(deps.Extract(gomod)).
			WithChecks(getServiceStatus("Jira Software")).
			WithChecks(getServiceStatus("Jira Service Management")).
			WithChecks(getServiceStatus("Jira Work Management")).
			WithChecks(getServiceStatus("Confluence")).
			Print()
		os.Exit(0)
	case options.GetB(OPT_HELP) || len(args) == 0:
		genUsage().Print()
		os.Exit(0)
	}

	err := errutil.Chain(
		loadConfig,
		validateConfig,
		setupLogger,
		setupTemp,
	)

	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	log.Divider()
	log.Aux("%s %s starting…", APP, VER)

	if !process(args.Get(0).String()) {
		os.Exit(1)
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// preConfigureUI preconfigures UI based on information about user terminal
func preConfigureUI() {
	if !tty.IsTTY() {
		fmtc.DisableColors = true
	}

	switch {
	case fmtc.IsTrueColorSupported():
		colorTagApp, colorTagVer = "{*}{#00AFFF}", "{#00AFFF}"
	case fmtc.Is256ColorsSupported():
		colorTagApp, colorTagVer = "{*}{#39}", "{#39}"
	default:
		colorTagApp, colorTagVer = "{*}{c}", "{c}"
	}
}

// addExtraOptions adds additional options for configuration when the application
// is running inside a container
func addExtraOptions(m options.Map) {
	if !container.IsContainer() {
		return
	}

	knfu.AddOptions(m,
		ACCESS_ACCOUNT,
		ACCESS_EMAIL,
		ACCESS_API_KEY,
		STORAGE_TYPE,
		STORAGE_FS_PATH,
		STORAGE_FS_MODE,
		STORAGE_SFTP_HOST,
		STORAGE_SFTP_USER,
		STORAGE_SFTP_KEY,
		STORAGE_SFTP_PATH,
		STORAGE_SFTP_MODE,
		STORAGE_S3_HOST,
		STORAGE_S3_ACCESS_KEY,
		STORAGE_S3_SECRET_KEY,
		STORAGE_S3_BUCKET,
		STORAGE_S3_PATH,
		JIRA_OUTPUT_FILE,
		JIRA_INCLUDE_ATTACHMENTS,
		JIRA_CLOUD_FORMAT,
		CONFLUENCE_OUTPUT_FILE,
		CONFLUENCE_INCLUDE_ATTACHMENTS,
		CONFLUENCE_CLOUD_FORMAT,
		TEMP_DIR,
		LOG_FORMAT,
		LOG_LEVEL,
	)
}

// configureUI configures user interface
func configureUI() {
	if options.GetB(OPT_NO_COLOR) {
		fmtc.DisableColors = true
	}

	req.SetUserAgent("AtlassianCloudBackuper", VER)
}

// loadConfig loads configuration file
func loadConfig() error {
	config, err := knf.Read(options.GetS(OPT_CONFIG))

	if err != nil {
		return fmt.Errorf("Can't load configuration: %w", err)
	}

	if !container.IsContainer() {
		knfu.Combine(config)
	} else {
		knfu.CombineSimple(
			config,
			ACCESS_ACCOUNT, ACCESS_EMAIL, ACCESS_API_KEY,
			STORAGE_TYPE,
			STORAGE_FS_PATH, STORAGE_FS_MODE,
			STORAGE_SFTP_HOST, STORAGE_SFTP_USER, STORAGE_SFTP_KEY,
			STORAGE_SFTP_PATH, STORAGE_SFTP_MODE,
			STORAGE_S3_HOST, STORAGE_S3_REGION, STORAGE_S3_ACCESS_KEY,
			STORAGE_S3_SECRET_KEY, STORAGE_S3_BUCKET, STORAGE_S3_PATH,
			JIRA_OUTPUT_FILE, JIRA_INCLUDE_ATTACHMENTS, JIRA_CLOUD_FORMAT,
			CONFLUENCE_OUTPUT_FILE, CONFLUENCE_INCLUDE_ATTACHMENTS, CONFLUENCE_CLOUD_FORMAT,
			TEMP_DIR,
			LOG_DIR, LOG_FILE, LOG_MODE, LOG_LEVEL,
		)
	}

	return nil
}

// validateConfig validates configuration file values
func validateConfig() error {
	validators := []*knf.Validator{
		{ACCESS_ACCOUNT, knfv.Empty, nil},
		{ACCESS_EMAIL, knfv.Empty, nil},
		{ACCESS_API_KEY, knfv.Empty, nil},
		{ACCESS_EMAIL, knfn.Mail, nil},
		{STORAGE_TYPE, knfv.NotContains, []string{
			"fs", "sftp", "s3",
		}},
		{LOG_FORMAT, knfv.NotContains, []string{
			"", "text", "json",
		}},
		{LOG_LEVEL, knfv.NotContains, []string{
			"", "debug", "info", "warn", "error", "crit",
		}},
		{TEMP_DIR, knff.Perms, "DW"},
	}

	switch knfu.GetS(STORAGE_TYPE) {
	case "fs":
		validators = append(validators,
			&knf.Validator{STORAGE_FS_PATH, knff.Perms, "DRW"},
		)

	case "sftp":
		validators = append(validators,
			&knf.Validator{STORAGE_SFTP_HOST, knfv.Empty, nil},
			&knf.Validator{STORAGE_SFTP_USER, knfv.Empty, nil},
			&knf.Validator{STORAGE_SFTP_KEY, knfv.Empty, nil},
			&knf.Validator{STORAGE_SFTP_PATH, knfv.Empty, nil},
		)

	case "s3":
		validators = append(validators,
			&knf.Validator{STORAGE_S3_HOST, knfv.Empty, nil},
			&knf.Validator{STORAGE_S3_ACCESS_KEY, knfv.Empty, nil},
			&knf.Validator{STORAGE_S3_SECRET_KEY, knfv.Empty, nil},
			&knf.Validator{STORAGE_S3_BUCKET, knfv.Empty, nil},
			&knf.Validator{STORAGE_S3_PATH, knfv.Empty, nil},
		)
	}

	errs := knfu.Validate(validators)

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// setupLogger configures logger subsystem
func setupLogger() error {
	var err error

	if knfu.GetS(LOG_FILE) != "" {
		err = log.Set(knfu.GetS(LOG_FILE), knfu.GetM(LOG_MODE, 640))

		if err != nil {
			return err
		}
	}

	err = log.MinLevel(knfu.GetS(LOG_LEVEL, "info"))

	if err != nil {
		return err
	}

	if knfu.GetS(LOG_FORMAT) == "" && container.IsContainer() {
		log.Global.UseJSON = true
	} else {
		switch knfu.GetS(LOG_FORMAT) {
		case "json":
			log.Global.UseJSON = true
		case "text", "":
			// default
		default:
			return fmt.Errorf("Unknown log format %q", knfu.GetS(LOG_FORMAT))
		}
	}

	return nil
}

// setupTemp configures temporary directory
func setupTemp() error {
	var err error

	temp, err = tmp.NewTemp(knfu.GetS(TEMP_DIR, "/tmp"))

	return err
}

// process starts backup creation
func process(target string) bool {
	var dispatcher *events.Dispatcher

	if options.GetB(OPT_INTERACTIVE) {
		dispatcher = events.NewDispatcher()
		addEventsHandlers(dispatcher)
	}

	defer temp.Clean()

	bkpr, err := getBackuper(target)

	if err != nil {
		log.Crit("Can't start backuping process: %v", err)
		return false
	}

	bkpr.SetDispatcher(dispatcher)

	outputFileName := getOutputFileName(target)
	tmpFile := path.Join(temp.MkName(".zip"), outputFileName)

	err = bkpr.Backup(tmpFile)

	if err != nil {
		spinner.Done(false)
		log.Crit("Error while backuping process: %v", err)
		return false
	}

	log.Info("Backup process successfully finished!")

	updr, err := getUploader(target)

	if err != nil {
		log.Crit("Can't start uploading process: %v", err)
		return false
	}

	updr.SetDispatcher(dispatcher)

	err = updr.Upload(tmpFile, outputFileName)

	if err != nil {
		spinner.Done(false)
		log.Crit("Error while uploading process: %v", err)
		return false
	}

	return true
}

// getBackuper returns backuper instances
func getBackuper(target string) (backuper.Backuper, error) {
	var err error
	var bkpr backuper.Backuper

	bkpConfig, err := getBackuperConfig(target)

	if err != nil {
		return nil, err
	}

	switch target {
	case TARGET_JIRA:
		bkpr, err = jira.NewBackuper(bkpConfig)
	case TARGET_CONFLUENCE:
		bkpr, err = confluence.NewBackuper(bkpConfig)
	}

	return bkpr, nil
}

// getOutputFileName returns name for backup output file
func getOutputFileName(target string) string {
	var template string

	switch target {
	case TARGET_JIRA:
		template = knfu.GetS(JIRA_OUTPUT_FILE, `jira-backup-%Y-%m-%d`) + ".zip"
	case TARGET_CONFLUENCE:
		template = knfu.GetS(JIRA_OUTPUT_FILE, `confluence-backup-%Y-%m-%d`) + ".zip"
	}

	return timeutil.Format(time.Now(), template)
}

// getBackuperConfig returns configuration for backuper
func getBackuperConfig(target string) (*backuper.Config, error) {
	switch target {
	case TARGET_JIRA:
		return &backuper.Config{
			Account:         knfu.GetS(ACCESS_ACCOUNT),
			Email:           knfu.GetS(ACCESS_EMAIL),
			APIKey:          knfu.GetS(ACCESS_API_KEY),
			WithAttachments: knfu.GetB(JIRA_INCLUDE_ATTACHMENTS),
			ForCloud:        knfu.GetB(JIRA_CLOUD_FORMAT),
		}, nil

	case TARGET_CONFLUENCE:
		return &backuper.Config{
			Account:         knfu.GetS(ACCESS_ACCOUNT),
			Email:           knfu.GetS(ACCESS_EMAIL),
			APIKey:          knfu.GetS(ACCESS_API_KEY),
			WithAttachments: knfu.GetB(CONFLUENCE_INCLUDE_ATTACHMENTS),
			ForCloud:        knfu.GetB(CONFLUENCE_CLOUD_FORMAT),
		}, nil
	}

	return nil, fmt.Errorf("Unknown target %q", target)
}

// getUploader returns uploader instance
func getUploader(target string) (uploader.Uploader, error) {
	var err error
	var updr uploader.Uploader

	switch knfu.GetS(STORAGE_TYPE) {
	case "fs":
		updr, err = fs.NewUploader(&fs.Config{
			Path: path.Join(knfu.GetS(STORAGE_FS_PATH), target),
			Mode: knfu.GetM(STORAGE_FS_MODE, 0600),
		})

	case "sftp":
		keyData, err := readPrivateKeyData()

		if err != nil {
			return nil, err
		}

		updr, err = sftp.NewUploader(&sftp.Config{
			Host: knfu.GetS(STORAGE_SFTP_HOST),
			User: knfu.GetS(STORAGE_SFTP_USER),
			Key:  keyData,
			Path: path.Join(knfu.GetS(STORAGE_SFTP_PATH), target),
			Mode: knfu.GetM(STORAGE_SFTP_MODE, 0600),
		})

	case "s3":
		updr, err = s3.NewUploader(&s3.Config{
			Host:        knfu.GetS(STORAGE_S3_HOST),
			Region:      knfu.GetS(STORAGE_S3_REGION),
			AccessKeyID: knfu.GetS(STORAGE_S3_ACCESS_KEY),
			SecretKey:   knfu.GetS(STORAGE_S3_SECRET_KEY),
			Bucket:      knfu.GetS(STORAGE_S3_BUCKET),
			Path:        path.Join(knfu.GetS(STORAGE_S3_PATH), target),
		})
	}

	return updr, err
}

// readPrivateKeyData reads private key data
func readPrivateKeyData() ([]byte, error) {
	if fsutil.IsExist(knfu.GetS(STORAGE_SFTP_KEY)) {
		return os.ReadFile(knfu.GetS(STORAGE_SFTP_KEY))
	}

	return base64.StdEncoding.DecodeString(knfu.GetS(STORAGE_SFTP_KEY))
}

// addEventsHandlers registers events handlers
func addEventsHandlers(dispatcher *events.Dispatcher) {
	dispatcher.AddHandler(backuper.EVENT_BACKUP_STARTED, func(payload any) {
		fmtc.NewLine()
		spinner.Show("Starting downloading process")
	})

	dispatcher.AddHandler(backuper.EVENT_BACKUP_PROGRESS, func(payload any) {
		p := payload.(*backuper.ProgressInfo)
		spinner.Update("[%d%%] %s", p.Progress, p.Message)
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
			"[%s] Uploading file (%s/%s)",
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

// printError prints error message to console
func printError(f string, a ...interface{}) {
	if len(a) == 0 {
		fmtc.Fprintln(os.Stderr, "{r}"+f+"{!}")
	} else {
		fmtc.Fprintf(os.Stderr, "{r}"+f+"{!}\n", a...)
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// getServiceStatus returns service status from status API
func getServiceStatus(service string) support.Check {
	chk := support.Check{support.CHECK_ERROR, service, ""}
	serviceName := strings.ReplaceAll(strings.ToLower(service), " ", "-")

	resp, err := req.Request{
		URL:         fmt.Sprintf("https://%s.status.atlassian.com/api/v2/status.json", serviceName),
		AutoDiscard: true,
	}.Get()

	if err != nil {
		chk.Message = "Can't send request to status API"
		return chk
	}

	if resp.StatusCode != 200 {
		chk.Message = fmt.Sprintf("Status API returned non-ok status code (%d)", resp.StatusCode)
		return chk
	}

	type StatusInfo struct {
		Desc      string `json:"description"`
		Indicator string `json:"indicator"`
	}

	type StatusResp struct {
		Status *StatusInfo `json:"status"`
	}

	status := &StatusResp{}
	err = resp.JSON(status)

	if err != nil {
		chk.Message = err.Error()
		return chk
	}

	switch status.Status.Indicator {
	case "minor":
		chk.Status = support.CHECK_WARN
	case "none":
		chk.Status = support.CHECK_OK
	}

	chk.Message = status.Status.Desc

	return chk
}

// printCompletion prints completion for given shell
func printCompletion() int {
	info := genUsage()

	switch options.GetS(OPT_COMPLETION) {
	case "bash":
		fmt.Print(bash.Generate(info, "atlassian-cloud-backuper"))
	case "fish":
		fmt.Print(fish.Generate(info, "atlassian-cloud-backuper"))
	case "zsh":
		fmt.Print(zsh.Generate(info, optMap, "atlassian-cloud-backuper"))
	default:
		return 1
	}

	return 0
}

// printMan prints man page
func printMan() {
	fmt.Println(man.Generate(genUsage(), genAbout("")))
}

// addUnitedOption adds info about option from united config
func addUnitedOption(info *usage.Info, prop, desc, value string) {
	info.AddOption(knfu.O(prop), desc+" {s-}("+knfu.E(prop)+"){!}", value).ColorTag = "{b}"
}

// genUsage generates usage info
func genUsage() *usage.Info {
	info := usage.NewInfo("", "target")

	info.AddOption(OPT_CONFIG, "Path to configuration file", "file")
	info.AddOption(OPT_INTERACTIVE, "Interactive mode")
	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

	if container.IsContainer() {
		addUnitedOption(info, ACCESS_ACCOUNT, "Account name", "name")
		addUnitedOption(info, ACCESS_EMAIL, "User email with access to API", "email")
		addUnitedOption(info, ACCESS_API_KEY, "API key", "key")
		addUnitedOption(info, STORAGE_TYPE, "Storage type", "fs/sftp/s3")
		addUnitedOption(info, STORAGE_FS_PATH, "Path on system for backups", "path")
		addUnitedOption(info, STORAGE_FS_MODE, "File mode on system", "mode")
		addUnitedOption(info, STORAGE_SFTP_HOST, "SFTP host", "host")
		addUnitedOption(info, STORAGE_SFTP_USER, "SFTP user name", "name")
		addUnitedOption(info, STORAGE_SFTP_KEY, "Base64-encoded private key", "key")
		addUnitedOption(info, STORAGE_SFTP_PATH, "Path on SFTP", "path")
		addUnitedOption(info, STORAGE_SFTP_MODE, "File mode on SFTP", "mode")
		addUnitedOption(info, STORAGE_S3_HOST, "S3 host", "host")
		addUnitedOption(info, STORAGE_S3_REGION, "S3 region", "region")
		addUnitedOption(info, STORAGE_S3_ACCESS_KEY, "S3 access key ID", "id")
		addUnitedOption(info, STORAGE_S3_SECRET_KEY, "S3 access secret key", "key")
		addUnitedOption(info, STORAGE_S3_BUCKET, "S3 bucket", "name")
		addUnitedOption(info, STORAGE_S3_PATH, "Path for backups", "path")
		addUnitedOption(info, JIRA_OUTPUT_FILE, "Jira backup output file name template", "template")
		addUnitedOption(info, JIRA_INCLUDE_ATTACHMENTS, "Include attachments to Jira backup", "yes/no")
		addUnitedOption(info, JIRA_CLOUD_FORMAT, "Create Jira backup for Cloud", "yes/no")
		addUnitedOption(info, CONFLUENCE_OUTPUT_FILE, "Confluence backup output file name template", "template")
		addUnitedOption(info, CONFLUENCE_INCLUDE_ATTACHMENTS, "Include attachments to Confluence backup", "yes/no")
		addUnitedOption(info, CONFLUENCE_CLOUD_FORMAT, "Create Confluence backup for Cloud", "yes/no")
		addUnitedOption(info, TEMP_DIR, "Path to directory for temporary data", "path")
		addUnitedOption(info, LOG_FORMAT, "Log format", "text/json")
		addUnitedOption(info, LOG_LEVEL, "Log level", "level")
	}

	return info
}

// genAbout generates info about version
func genAbout(gitRev string) *usage.About {
	about := &usage.About{
		App:     APP,
		Version: VER,
		Desc:    DESC,
		Year:    2009,
		Owner:   "ESSENTIAL KAOS",

		AppNameColorTag: colorTagApp,
		VersionColorTag: colorTagVer,
		DescSeparator:   "{s}—{!}",

		License:       "Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>",
		BugTracker:    "https://github.com/essentialkaos/atlassian-cloud-backuper/issues",
		UpdateChecker: usage.UpdateChecker{"essentialkaos/atlassian-cloud-backuper", update.GitHubChecker},
	}

	if gitRev != "" {
		about.Build = "git:" + gitRev
	}

	return about
}

// ////////////////////////////////////////////////////////////////////////////////// //
