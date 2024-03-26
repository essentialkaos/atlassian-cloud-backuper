<p align="center"><a href="#readme"><img src="https://gh.kaos.st/atlassian-cloud-backuper.svg" /></a></p>

<p align="center">
  <a href="https://kaos.sh/w/atlassian-cloud-backuper/ci"><img src="https://kaos.sh/w/atlassian-cloud-backuper/ci.svg" alt="GitHub Actions CI Status" /></a>
  <a href="https://kaos.sh/l/atlassian-cloud-backuper"><img src="https://kaos.sh/l/c742a6f5789762426f97.svg" alt="Code Climate Maintainability" /></a>
  <a href="https://kaos.sh/b/atlassian-cloud-backuper"><img src="https://kaos.sh/b/f337729e-ce98-4c15-9123-420f9feb443f.svg" alt="Codebeat badge" /></a>
  <a href="https://kaos.sh/w/atlassian-cloud-backuper/codeql"><img src="https://kaos.sh/w/atlassian-cloud-backuper/codeql.svg" alt="GitHub Actions CodeQL Status" /></a>
  <a href="#license"><img src="https://gh.kaos.st/apache2.svg"></a>
</p>

<br/>

`atlassian-cloud-backuper` is tool for backuping Atlassian cloud services (_Jira and Confluence_).

### Installation

#### From [ESSENTIAL KAOS Public Repository](https://kaos.sh/kaos-repo)

```bash
sudo yum install -y https://pkgs.kaos.st/kaos-repo-latest.el$(grep 'CPE_NAME' /etc/os-release | tr -d '"' | cut -d':' -f5).noarch.rpm
sudo yum install atlassian-cloud-backuper
```

### Usage

#### Standalone
```
Usage: atlassian-cloud-backuper {options} target

Options

  --config, -c file    Path to configuration file
  --interactive, -I    Interactive mode
  --no-color, -nc      Disable colors in output
  --help, -h           Show this help message
  --version, -v        Show version
```

#### Container

If `atlassian-cloud-backuper` runs inside a container, it allows you to use united configuration (_knf file + options + environment variables_).

```
Usage: atlassian-cloud-backuper {options} target

Options

  --config, -c file                          Path to configuration file
  --interactive, -I                          Interactive mode
  --no-color, -nc                            Disable colors in output
  --help, -h                                 Show this help message
  --version, -v                              Show version

  --access-account name                      Account name (ACCESS_ACCOUNT)
  --access-email email                       User email with access to API (ACCESS_EMAIL)
  --access-api-key key                       API key (ACCESS_API_KEY)
  --storage-type fs/sftp/s3                  Storage type (STORAGE_TYPE)
  --storage-fs-path path                     Path on system for backups (STORAGE_FS_PATH)
  --storage-fs-mode mode                     File mode on system (STORAGE_FS_MODE)
  --storage-sftp-host host                   SFTP host (STORAGE_SFTP_HOST)
  --storage-sftp-user name                   SFTP user name (STORAGE_SFTP_USER)
  --storage-sftp-key key                     SFTP user private key (STORAGE_SFTP_KEY)
  --storage-sftp-path path                   Path on SFTP (STORAGE_SFTP_PATH)
  --storage-sftp-mode mode                   File mode on SFTP (STORAGE_SFTP_MODE)
  --storage-s3-host host                     S3 host (STORAGE_S3_HOST)
  --storage-s3-access-key id                 S3 access key ID (STORAGE_S3_ACCESS_KEY)
  --storage-s3-secret-key key                S3 access secret key (STORAGE_S3_SECRET_KEY)
  --storage-s3-bucket name                   S3 bucket (STORAGE_S3_BUCKET)
  --storage-s3-path path                     Path for backups (STORAGE_S3_PATH)
  --jira-output-file template                Jira backup output file name template (JIRA_OUTPUT_FILE)
  --jira-include-attachments yes/no          Include attachments to Jira backup (JIRA_INCLUDE_ATTACHMENTS)
  --jira-cloud-format yes/no                 Create Jira backup for Cloud (JIRA_CLOUD_FORMAT)
  --confluence-output-file template          Confluence backup output file name template (CONFLUENCE_OUTPUT_FILE)
  --confluence-include-attachments yes/no    Include attachments to Confluence backup (CONFLUENCE_INCLUDE_ATTACHMENTS)
  --confluence-cloud-format yes/no           Create Confluence backup for Cloud (CONFLUENCE_CLOUD_FORMAT)
  --temp-dir path                            Path to directory for temporary data (TEMP_DIR)
  --log-format text/json                     Log format (LOG_FORMAT)
  --log-level level                          Log level (LOG_LEVEL)
```

### CI Status

| Branch | Status |
|--------|----------|
| `master` | [![CI](https://kaos.sh/w/atlassian-cloud-backuper/ci.svg?branch=master)](https://kaos.sh/w/atlassian-cloud-backuper/ci?query=branch:master) |
| `develop` | [![CI](https://kaos.sh/w/atlassian-cloud-backuper/ci.svg?branch=develop)](https://kaos.sh/w/atlassian-cloud-backuper/ci?query=branch:develop) |

### Contributing

Before contributing to this project please read our [Contributing Guidelines](https://github.com/essentialkaos/contributing-guidelines#contributing-guidelines).

### License

[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0)

<p align="center"><a href="https://essentialkaos.com"><img src="https://gh.kaos.st/ekgh.svg"/></a></p>
