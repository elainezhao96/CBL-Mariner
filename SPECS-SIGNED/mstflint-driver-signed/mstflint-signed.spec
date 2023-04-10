%define mstflint_unsigned_name kernel-mstflint
%global kver %(/bin/rpm -q --queryformat '%{RPMTAG_VERSION}-%{RPMTAG_RELEASE}' $(/bin/rpm -q --whatprovides kernel-devel))
%global install_mod_dir %{_libdir}/modules/%{kver}/extra/%{mstflint_unsigned_name}
%define mstflint_module_name mstflint_access.ko
%define upper_kernel_ver 5.16.0
%define lower_kernel_ver 5.15.0
Summary:        Mellanox firmware burning tool
Name:           %{mstflint_unsigned_name}-signed
Version:        4.21.0
Release:        4%{?dist}
License:        Dual BSD/GPL
Vendor:         Microsoft Corporation
Distribution:   Mariner
Group:          System Environment/Kernel
URL:            https://github.com/Mellanox/%{name}
Source0:        https://github.com/Mellanox/%{name}/releases/download/v%{version}-1/%{name}-%{version}-1.tar.gz#%{mstflint_module_name}
BuildRequires:  kernel-headers >= %{lower_kernel_ver}
BuildRequires:  kernel-headers < %{upper_kernel_ver}
Requires:       kernel >= %{lower_kernel_ver}
Requires:       kernel < %{upper_kernel_ver}

%install
install -dm 755 %{buildroot}%{install_mod_dir}
install -m 644 %{SOURCE0} %{buildroot}%{install_mod_dir}

%post -n %{mstflint_unsigned_name}
depmod %{kver}

%postun -n %{mstflint_unsigned_name}
depmod %{kver}

%files -n %{mstflint_unsigned_name}
%defattr(-,root,root,-)
/%{install_mod_dir}/

%changelog
* Thu Mar 23 2023 Elaheh Dehghani <edehghani@microsoft.com> - 4.21.0-4
- Add mstflint driver for secure boot.
- Original version for CBL-Mariner.
- License verified.