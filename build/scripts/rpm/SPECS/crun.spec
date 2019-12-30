Name:           %{_product_name}
Version:        %{_product_version}

Release:        1.el%{_rhel_version}
Summary:        Command execution wrapper.
Group:          Development/Tools
License:        MIT
Source0:        %{name}_linux_amd64.zip
Source1:        crun.toml
Source2:        crun.logrotate
BuildRoot:      %(mktemp -ud %{_tmppath}/%{name}-%{version}-%{release}-XXXXXX)

%description
Command execution wrapper.

%prep
%setup -q -c

%install
mkdir -p %{buildroot}/%{_bindir}
cp %{name} %{buildroot}/%{_bindir}

mkdir -p %{buildroot}/%{_sysconfdir}/crun
cp %{SOURCE1} %{buildroot}/%{_sysconfdir}/crun/crun.toml

mkdir -p %{buildroot}/%{_sysconfdir}/logrotate.d/
cp %{SOURCE2} %{buildroot}/%{_sysconfdir}/logrotate.d/crun

mkdir -p %{buildroot}/var/log/crun

%pre

%post

%preun

%clean
rm -rf %{buildroot}

%files
%defattr(-,root,root,-)
%attr(755, root, root) %{_bindir}/%{name}
%config(noreplace) %{_sysconfdir}/crun/crun.toml
%attr(644, root, root) %{_sysconfdir}/logrotate.d/crun
%dir %attr(777, root, root) /var/log/crun

%doc
