package app

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"strings"

	"github.com/essentialkaos/ek/v13/errors"
	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/knf"
	"github.com/essentialkaos/ek/v13/log"
	"github.com/essentialkaos/ek/v13/options"
	"github.com/essentialkaos/ek/v13/req"
	"github.com/essentialkaos/ek/v13/support"
	"github.com/essentialkaos/ek/v13/support/deps"
	"github.com/essentialkaos/ek/v13/system/container"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/terminal/tty"
	"github.com/essentialkaos/ek/v13/tmp"
	"github.com/essentialkaos/ek/v13/usage"
	"github.com/essentialkaos/ek/v13/usage/completion/bash"
	"github.com/essentialkaos/ek/v13/usage/completion/fish"
	"github.com/essentialkaos/ek/v13/usage/completion/zsh"
	"github.com/essentialkaos/ek/v13/usage/man"
	"github.com/essentialkaos/ek/v13/usage/update"

	knfu "github.com/essentialkaos/ek/v13/knf/united"
	knfv "github.com/essentialkaos/ek/v13/knf/validators"
	knff "github.com/essentialkaos/ek/v13/knf/validators/fs"
	knfn "github.com/essentialkaos/ek/v13/knf/validators/network"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Basic utility info
const (
	APP  = "Atlassian Cloud Backuper"
	VER  = "0.2.1"
	DESC = "Tool for backuping Atlassian cloud services (Jira and Confluence)"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Options
const (
	OPT_CONFIG      = "c:config"
	OPT_INTERACTIVE = "I:interactive"
	OPT_SERVER      = "S:server"
	OPT_FORCE       = "F:force"
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
	SERVER_IP                      = "server:ip"
	SERVER_PORT                    = "server:port"
	SERVER_ACCESS_TOKEN            = "server:access-token"
	STORAGE_TYPE                   = "storage:type"
	STORAGE_ENCRYPTION_KEY         = "storage:encryption-key"
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
	STORAGE_S3_PART_SIZE           = "storage-s3:part-size"
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
	LOG_MODE                       = "log:mode"
	LOG_LEVEL                      = "log:level"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	TARGET_JIRA       = "jira"
	TARGET_CONFLUENCE = "confluence"
)

const (
	STORAGE_FS   = "fs"
	STORAGE_SFTP = "sftp"
	STORAGE_S3   = "s3"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// optMap contains information about all supported options
var optMap = options.Map{
	OPT_CONFIG:      {Value: "/etc/atlassian-cloud-backuper.knf"},
	OPT_FORCE:       {Type: options.BOOL},
	OPT_INTERACTIVE: {Type: options.BOOL},
	OPT_SERVER:      {Type: options.BOOL},
	OPT_NO_COLOR:    {Type: options.BOOL},
	OPT_HELP:        {Type: options.MIXED},
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

	if !errs.IsEmpty() {
		terminal.Error("Options parsing errors:")
		terminal.Error(errs.Error(" - "))
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
	case options.GetB(OPT_HELP) ||
		(!options.Has(OPT_SERVER) && len(args) == 0):
		genUsage(options.GetS(OPT_HELP)).Print()
		os.Exit(0)
	}

	err := errors.Chain(
		loadConfig,
		validateConfig,
		setupLogger,
	)

	if err != nil {
		terminal.Error(err)
		os.Exit(1)
	}

	log.Divider()
	log.Aux("%s %s starting…", APP, VER)

	err = errors.Chain(
		setupTemp,
		setupReq,
	)

	if err != nil {
		log.Crit(err.Error())
		os.Exit(1)
	}

	defer temp.Clean()

	if options.GetB(OPT_SERVER) {
		err = startServer()
	} else {
		err = startApp(args)
	}

	if err != nil {
		if options.GetB(OPT_INTERACTIVE) {
			terminal.Error(err)
		}

		log.Crit(err.Error())

		os.Exit(1)
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// preConfigureUI preconfigures UI based on information about user terminal
func preConfigureUI() {
	if !tty.IsTTY() || tty.IsSystemd() || container.GetEngine() == container.YANDEX {
		fmtc.DisableColors = true
	}

	switch {
	case fmtc.IsTrueColorSupported():
		colorTagApp, colorTagVer = "{*}{#0065FF}", "{#0065FF}"
	case fmtc.Is256ColorsSupported():
		colorTagApp, colorTagVer = "{*}{#21}", "{#21}"
	default:
		colorTagApp, colorTagVer = "{*}{b}", "{b}"
	}
}

// addExtraOptions adds additional options for configuration when the application
// is running inside a container
func addExtraOptions(m options.Map) {
	if !container.IsContainer() {
		return
	}

	knfu.AddOptions(m,
		ACCESS_ACCOUNT, ACCESS_EMAIL, ACCESS_API_KEY,
		SERVER_IP, SERVER_PORT, SERVER_ACCESS_TOKEN,
		STORAGE_TYPE, STORAGE_ENCRYPTION_KEY,
		STORAGE_FS_PATH, STORAGE_FS_MODE,
		STORAGE_SFTP_HOST, STORAGE_SFTP_USER, STORAGE_SFTP_KEY,
		STORAGE_SFTP_PATH, STORAGE_SFTP_MODE,
		STORAGE_S3_HOST, STORAGE_S3_ACCESS_KEY, STORAGE_S3_SECRET_KEY,
		STORAGE_S3_BUCKET, STORAGE_S3_PATH, STORAGE_S3_PART_SIZE,
		JIRA_OUTPUT_FILE, JIRA_INCLUDE_ATTACHMENTS, JIRA_CLOUD_FORMAT,
		CONFLUENCE_OUTPUT_FILE, CONFLUENCE_INCLUDE_ATTACHMENTS, CONFLUENCE_CLOUD_FORMAT,
		TEMP_DIR,
		LOG_FORMAT, LOG_LEVEL,
	)
}

// configureUI configures user interface
func configureUI() {
	if options.GetB(OPT_NO_COLOR) {
		fmtc.DisableColors = true
	}
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
			SERVER_IP, SERVER_PORT, SERVER_ACCESS_TOKEN,
			STORAGE_TYPE, STORAGE_ENCRYPTION_KEY,
			STORAGE_FS_PATH, STORAGE_FS_MODE,
			STORAGE_SFTP_HOST, STORAGE_SFTP_USER, STORAGE_SFTP_KEY,
			STORAGE_SFTP_PATH, STORAGE_SFTP_MODE,
			STORAGE_S3_HOST, STORAGE_S3_REGION, STORAGE_S3_ACCESS_KEY,
			STORAGE_S3_SECRET_KEY, STORAGE_S3_BUCKET, STORAGE_S3_PATH, STORAGE_S3_PART_SIZE,
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
	validators := knf.Validators{
		{ACCESS_ACCOUNT, knfv.Set, nil},
		{ACCESS_EMAIL, knfv.Set, nil},
		{ACCESS_API_KEY, knfv.Set, nil},
		{ACCESS_EMAIL, knfn.Mail, nil},
		{STORAGE_TYPE, knfv.SetToAnyIgnoreCase, []string{
			STORAGE_FS, STORAGE_SFTP, STORAGE_S3,
		}},
		{TEMP_DIR, knff.Perms, "DWRX"},
		{LOG_FORMAT, knfv.SetToAnyIgnoreCase, []string{
			"", "text", "json",
		}},
		{LOG_LEVEL, knfv.SetToAnyIgnoreCase, log.LogLevels},
	}

	validators = validators.AddIf(
		knfu.GetS(STORAGE_TYPE) == STORAGE_FS,
		knf.Validators{
			{STORAGE_FS_PATH, knff.Perms, "DRW"},
		},
	)

	validators = validators.AddIf(
		knfu.GetS(STORAGE_TYPE) == STORAGE_SFTP,
		knf.Validators{
			{STORAGE_SFTP_HOST, knfv.Set, nil},
			{STORAGE_SFTP_USER, knfv.Set, nil},
			{STORAGE_SFTP_KEY, knfv.Set, nil},
			{STORAGE_SFTP_PATH, knfv.Set, nil},
		},
	)

	validators = validators.AddIf(
		knfu.GetS(STORAGE_TYPE) == STORAGE_S3,
		knf.Validators{
			{STORAGE_S3_HOST, knfv.Set, nil},
			{STORAGE_S3_ACCESS_KEY, knfv.Set, nil},
			{STORAGE_S3_SECRET_KEY, knfv.Set, nil},
			{STORAGE_S3_BUCKET, knfv.Set, nil},
			{STORAGE_S3_PART_SIZE, knfv.TypeSize, nil},
			{STORAGE_S3_PART_SIZE, knfv.SizeGreater, 1 * 1024 * 1024},
			{STORAGE_S3_PART_SIZE, knfv.SizeLess, 100 * 1024 * 1024},
		},
	)

	validators = validators.AddIf(
		options.GetB(OPT_SERVER),
		knf.Validators{
			{SERVER_IP, knfn.IP, nil},
			{SERVER_PORT, knfn.Port, nil},
		},
	)

	validators = validators.AddIf(
		knfu.GetS(STORAGE_ENCRYPTION_KEY) != "",
		knf.Validators{
			{STORAGE_ENCRYPTION_KEY, knfv.LenGreater, 16},
			{STORAGE_ENCRYPTION_KEY, knfv.LenLess, 96},
		},
	)

	errs := knfu.Validate(validators)

	if !errs.IsEmpty() {
		return errs.First()
	}

	return nil
}

// setupLogger configures logger subsystem
func setupLogger() error {
	var err error

	if knfu.GetS(LOG_FILE) != "" {
		err = log.Set(knfu.GetS(LOG_FILE), knfu.GetM(LOG_MODE, 0644))

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
		switch strings.ToLower(knfu.GetS(LOG_FORMAT)) {
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

	temp, err = tmp.NewTemp(knfu.GetS(TEMP_DIR, os.TempDir()))

	if err != nil {
		return fmt.Errorf("Can't setup temporary data directory: %w", err)
	}

	return nil
}

// setupReq configures HTTP request engine
func setupReq() error {
	req.SetUserAgent("AtlassianCloudBackuper", VER)
	return nil
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
	info := genUsage("")

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
	fmt.Println(man.Generate(genUsage(""), genAbout("")))
}

// addUnitedOption adds info about option from united config
func addUnitedOption(info *usage.Info, prop, desc, value string) {
	info.AddOption(knfu.O(prop), desc+" {s-}("+knfu.E(prop)+"){!}", value).ColorTag = "{b}"
}

// genUsage generates usage info
func genUsage(section string) *usage.Info {
	info := usage.NewInfo("", "target")

	info.WrapLen = 100
	info.AppNameColorTag = colorTagApp

	info.AddOption(OPT_CONFIG, "Path to configuration file", "file")
	info.AddOption(OPT_INTERACTIVE, "Interactive mode")
	info.AddOption(OPT_SERVER, "Server mode")
	info.AddOption(OPT_FORCE, "Force backup generation")
	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

	if container.IsContainer() || section == "container" {
		addUnitedOption(info, ACCESS_ACCOUNT, "Account name", "name")
		addUnitedOption(info, ACCESS_EMAIL, "User email with access to API", "email")
		addUnitedOption(info, ACCESS_API_KEY, "API key", "key")
		addUnitedOption(info, SERVER_IP, "HTTP server IP", "ip")
		addUnitedOption(info, SERVER_PORT, "HTTP server port", "port")
		addUnitedOption(info, SERVER_ACCESS_TOKEN, "HTTP access token", "token")
		addUnitedOption(info, STORAGE_TYPE, "Storage type", "fs/sftp/s3")
		addUnitedOption(info, STORAGE_ENCRYPTION_KEY, "Data encryption key", "key")
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
		addUnitedOption(info, STORAGE_S3_PART_SIZE, "Uploading part size", "size")
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
