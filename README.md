<p align="center"><a href="#readme"><img src="https://gh.kaos.st/atlassian-cloud-backuper.png" /></a></p>

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
```
Usage: atlassian-cloud-backuper {options} target

Options

  --interactive, -I    Interactive mode
  --no-color, -nc      Disable colors in output
  --help, -h           Show this help message
  --version, -v        Show version

Examples

  atlassian-cloud-backuper jira
  Create backup of Jira data

  atlassian-cloud-backuper confluence
  Create backup of Confluence data
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
