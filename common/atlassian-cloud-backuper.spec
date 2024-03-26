################################################################################

%global crc_check pushd ../SOURCES ; sha512sum -c %{SOURCE100} ; popd

################################################################################

%define debug_package  %{nil}

################################################################################

Summary:        atlassian-cloud-backuper
Name:           Tool for backuping Atlassian cloud services
Version:        0.0.1
Release:        0%{?dist}
Group:          Applications/System
License:        Apache License, Version 2.0
URL:            https://kaos.sh/atlassian-cloud-backuper

Source0:        https://source.kaos.st/%{name}/%{name}-%{version}.tar.bz2

Source100:      checksum.sha512

BuildRoot:      %{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

BuildRequires:  golang >= 1.21

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
  echo "This package requires vendored dependencies"
  exit 1
fi

pushd %{name}
  %{__make} %{?_smp_mflags} all
popd

%install
rm -rf %{buildroot}

install -dDm 755 %{buildroot}%{_bindir}
install -dDm 755 %{buildroot}%{_sysconfdir}/logrotate.d
install -dDm 755 %{buildroot}%{_logdir}/%{name}
install -dDm 755 %{buildroot}%{_mandir}/man1

install -pm 755 %{name}/%{name} \
                %{buildroot}%{_bindir}/

install -pm 644 %{name}/common/%{name}.knf \
                %{buildroot}%{_sysconfdir}/

install -pm 644 %{name}/common/%{name}.logrotate \
                %{buildroot}%{_sysconfdir}/logrotate.d/%{name}

# Generate man page
./%{name}/%{name} --generate-man > %{buildroot}%{_mandir}/man1/%{name}.1

# Generate completions
./%{name}/%{name} --completion=bash 1> %{buildroot}%{_sysconfdir}/bash_completion.d/%{name}
./%{name}/%{name} --completion=zsh 1> %{buildroot}%{_datadir}/zsh/site-functions/_%{name}
./%{name}/%{name} --completion=fish 1> %{buildroot}%{_datarootdir}/fish/vendor_completions.d/%{name}.fish

%clean
rm -rf %{buildroot}

################################################################################

%files
%defattr(-,root,root,-)
%doc LICENSE
%dir %{_logdir}/%{name}
%config(noreplace) %{_sysconfdir}/%{name}.knf
%config(noreplace) %{_sysconfdir}/logrotate.d/%{name}
%{_bindir}/%{name}
%{_mandir}/man1/%{name}.1.*
%{_sysconfdir}/bash_completion.d/%{name}
%{_datadir}/zsh/site-functions/_%{name}
%{_datarootdir}/fish/vendor_completions.d/%{name}.fish

################################################################################

%changelog
* Mon Mar 04 2024 Anton Novojilov <andy@essentialkaos.com> - 0.0.1-0
- Initial build for kaos-repo
