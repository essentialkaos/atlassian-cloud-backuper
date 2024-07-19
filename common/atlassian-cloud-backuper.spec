################################################################################

%global crc_check pushd ../SOURCES ; sha512sum -c %{SOURCE100} ; popd

################################################################################

%define debug_package  %{nil}

################################################################################

Summary:        Tool for backuping Atlassian cloud services
Name:           atlassian-cloud-backuper
Version:        0.0.4
Release:        0%{?dist}
Group:          Applications/System
License:        Apache License, Version 2.0
URL:            https://kaos.sh/atlassian-cloud-backuper

Source0:        https://source.kaos.st/%{name}/%{name}-%{version}.tar.bz2

Source100:      checksum.sha512

BuildRoot:      %{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

BuildRequires:  golang >= 1.22

Provides:       %{name} = %{version}-%{release}

################################################################################

%description
Tool for backuping Atlassian cloud services (Jira and Confluence).

################################################################################

%prep
%{crc_check}

%setup -q

%build
if [[ ! -d "%{name}/vendor" ]] ; then
  echo -e "----\nThis package requires vendored dependencies\n----"
  exit 1
elif [[ -f "%{name}/%{name}" ]] ; then
  echo -e "----\nSources must not contain precompiled binaries\n----"
  exit 1
fi

pushd %{name}
  %{__make} %{?_smp_mflags} all
popd

%install
rm -rf %{buildroot}

install -dDm 755 %{buildroot}%{_bindir}
install -dDm 755 %{buildroot}%{_sysconfdir}/logrotate.d
install -dDm 755 %{buildroot}%{_localstatedir}/log/%{name}

install -pm 755 %{name}/%{name} \
                %{buildroot}%{_bindir}/

install -pm 644 %{name}/common/%{name}.knf \
                %{buildroot}%{_sysconfdir}/

install -pm 644 %{name}/common/%{name}.logrotate \
                %{buildroot}%{_sysconfdir}/logrotate.d/%{name}

install -pDm 644 %{name}/common/%{name}.cron \
                 %{buildroot}%{_sysconfdir}/cron.d/%{name}

install -pDm 644 %{name}/common/%{name}-confluence.service \
                 %{buildroot}%{_unitdir}/%{name}-confluence.service
install -pDm 644 %{name}/common/%{name}-confluence.service \
                 %{buildroot}%{_unitdir}/%{name}-confluence.timer
install -pDm 644 %{name}/common/%{name}-jira.service \
                 %{buildroot}%{_unitdir}/%{name}-jira.service
install -pDm 644 %{name}/common/%{name}-jira.service \
                 %{buildroot}%{_unitdir}/%{name}-jira.timer

# Generate man page
install -dDm 755 %{buildroot}%{_mandir}/man1
./%{name}/%{name} --generate-man > %{buildroot}%{_mandir}/man1/%{name}.1

# Generate completions
install -dDm 755 %{buildroot}%{_sysconfdir}/bash_completion.d
install -dDm 755 %{buildroot}%{_datadir}/zsh/site-functions
install -dDm 755 %{buildroot}%{_datarootdir}/fish/vendor_completions.d
./%{name}/%{name} --completion=bash 1> %{buildroot}%{_sysconfdir}/bash_completion.d/%{name}
./%{name}/%{name} --completion=zsh 1> %{buildroot}%{_datadir}/zsh/site-functions/_%{name}
./%{name}/%{name} --completion=fish 1> %{buildroot}%{_datarootdir}/fish/vendor_completions.d/%{name}.fish

%clean
rm -rf %{buildroot}

################################################################################

%files
%defattr(-,root,root,-)
%doc %{name}/LICENSE
%dir %{_localstatedir}/log/%{name}
%config(noreplace) %{_sysconfdir}/%{name}.knf
%config(noreplace) %{_sysconfdir}/logrotate.d/%{name}
%config(noreplace) %{_unitdir}/%{name}-*
%config(noreplace) %{_sysconfdir}/cron.d/%{name}
%{_bindir}/%{name}
%{_mandir}/man1/%{name}.1.*
%{_sysconfdir}/bash_completion.d/%{name}
%{_datadir}/zsh/site-functions/_%{name}
%{_datarootdir}/fish/vendor_completions.d/%{name}.fish

################################################################################

%changelog
* Fri Jul 19 2024 Anton Novojilov <andy@essentialkaos.com> - 0.0.4-0
- Dependencies update

* Wed Jun 12 2024 Anton Novojilov <andy@essentialkaos.com> - 0.0.3-0
- Dependencies update

* Thu Apr 04 2024 Anton Novojilov <andy@essentialkaos.com> - 0.0.2-0
- Added cloud/serverless function for Yandex.Cloud
- Fixed bug with handling STORAGE_S3_REGION value from options and environment
  variables
- Code refactoring
- Dependencies update

* Tue Mar 26 2024 Anton Novojilov <andy@essentialkaos.com> - 0.0.1-0
- Initial build for kaos-repo
