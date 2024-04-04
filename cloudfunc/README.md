### Cloud function

#### Targets

- `jira` — Create backup of Jira instance;
- `confluence` — Create backup of Confluence instance.

#### Stages

- `create` — generate backup file for given target;
- `download` — download backup file for given target.

#### Recommended roles for service account (Yandex.Cloud)

- `storage.uploader` — for uploading data to S3;
- `lockbox.payloadViewer` — for viewing secrets in Lockbox;
- `functions.functionInvoker` — for working with triggers.

#### Handlers

- `cloudfunc/ycfunc.Request` — handler for HTTP requests
- `cloudfunc/ycfunc.Trigger` — handler for events from timer trigger

###### `Request` configuration

You must pass `target` and `stage` using query string.

Example:

```bash
# Create Jira backup (asynchronously)
curl 'https://functions.yandexcloud.net/abcdeabcdeabcdeabcde?target=jira&stage=create'

# Download Jira backup
curl 'https://functions.yandexcloud.net/abcdeabcdeabcdeabcde?target=jira&stage=download'

# Create Confluence backup (asynchronously)
curl 'https://functions.yandexcloud.net/abcdeabcdeabcdeabcde?target=confluence&stage=create'

# Download Confluence backup
curl 'https://functions.yandexcloud.net/abcdeabcdeabcdeabcde?target=confluence&stage=download'
```

HTTP status codes:

- `200` — request successfully processed;
- `400` — query validation error;
- `500` — configuration validation error.

###### `Trigger` configuration

You must pass `target` and `stage` using event payload using ';' as a separator.

Supported payload values:

```
jira;create
jira;download
confluence;create
confluence;download
```

#### Environment variables

| Env | Type | Required | Description |
|-----|------|----------|-------------|
| `ACCESS_ACCOUNT`                 | sᴛʀɪɴɢ  | Yes | _Account name_ |
| `ACCESS_EMAIL`                   | sᴛʀɪɴɢ  | Yes | _User email with access to API_ |
| `ACCESS_API_KEY`                 | sᴛʀɪɴɢ  | Yes | _API key_ |
| `STORAGE_TYPE`                   | sᴛʀɪɴɢ  | Yes | _Storage type (fs/sftp/s3)_ |
| `STORAGE_FS_PATH`                | sᴛʀɪɴɢ  | No  | _Path on system for backups_ |
| `STORAGE_FS_MODE`                | sᴛʀɪɴɢ  | No  | _File mode on system_ |
| `STORAGE_SFTP_HOST`              | sᴛʀɪɴɢ  | No  | _SFTP host_ |
| `STORAGE_SFTP_USER`              | sᴛʀɪɴɢ  | No  | _SFTP user name_ |
| `STORAGE_SFTP_KEY`               | sᴛʀɪɴɢ  | No  | _Base64-encoded private key_ |
| `STORAGE_SFTP_PATH`              | sᴛʀɪɴɢ  | No  | _Path on SFTP_ |
| `STORAGE_SFTP_MODE`              | sᴛʀɪɴɢ  | No  | _File mode on SFTP_ |
| `STORAGE_S3_HOST`                | sᴛʀɪɴɢ  | No  | _S3 host_ |
| `STORAGE_S3_REGION`              | sᴛʀɪɴɢ  | No  | _S3 region_ |
| `STORAGE_S3_ACCESS_KEY`          | sᴛʀɪɴɢ  | No  | _S3 access key ID_ |
| `STORAGE_S3_SECRET_KEY`          | sᴛʀɪɴɢ  | No  | _S3 access secret key_ |
| `STORAGE_S3_BUCKET`              | sᴛʀɪɴɢ  | No  | _S3 bucket_ |
| `STORAGE_S3_PATH`                | sᴛʀɪɴɢ  | No  | _Path for backups_ |
| `JIRA_OUTPUT_FILE`               | sᴛʀɪɴɢ  | No  | _Jira backup output file name template_ |
| `JIRA_INCLUDE_ATTACHMENTS`       | ʙᴏᴏʟᴇᴀɴ | No  | _Include attachments to Jira backup_ |
| `JIRA_CLOUD_FORMAT`              | ʙᴏᴏʟᴇᴀɴ | No  | _Create Jira backup for Cloud_ |
| `CONFLUENCE_OUTPUT_FILE`         | sᴛʀɪɴɢ  | No  | _Confluence backup output file name template_ |
| `CONFLUENCE_INCLUDE_ATTACHMENTS` | ʙᴏᴏʟᴇᴀɴ | No  | _Include attachments to Confluence backup_ |
| `CONFLUENCE_CLOUD_FORMAT`        | ʙᴏᴏʟᴇᴀɴ | No  | _Create Confluence backup for Cloud_ |
| `LOG_LEVEL`                      | sᴛʀɪɴɢ  | No  | _Log level (debug,info,warn,error)_ |
