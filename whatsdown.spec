%global debug_package %{nil}

Name:           whatsdown
Version:        0.1.0
Release:        0
Summary:        Native Linux WhatsApp client built with GTK 4 and Libadwaita

License:        MIT
Group:          Productivity/Networking/Instant Messenger
URL:            https://github.com/EmirOven/WhatsDown

BuildRequires:  golang >= 1.22
BuildRequires:  pkgconfig
BuildRequires:  pkgconfig(gtk4) >= 4.14
BuildRequires:  pkgconfig(libadwaita-1) >= 1.5
BuildRequires:  pkgconfig(sqlite3)
BuildRequires:  pkgconfig(gobject-introspection-1.0)
BuildRequires:  blueprint-compiler
BuildRequires:  gcc
%if 0%{?suse_version}
BuildRequires:  update-desktop-files
%else
BuildRequires:  desktop-file-utils
%endif

Requires:       gtk4 >= 4.14
Requires:       libadwaita >= 1.5
Requires:       sqlite
Requires:       hicolor-icon-theme

%description
WhatsDown is a modern native Linux WhatsApp desktop client built with Go,
GTK 4, and Libadwaita, utilizing whatsmeow and gotk4.

%prep
# In SCM sync, source files are already present in the build root

%build
# Compile Blueprint UI definitions into XML UI
blueprint-compiler batch-compile ui ui ui/*.blp ui/components/*.blp

# Build Go binary using offline vendored dependencies
export CGO_ENABLED=1
export GOFLAGS="-trimpath -mod=vendor -modcacherw"
go build -ldflags="-s -w" -v -o whatsdown .

%install
install -D -p -m 0755 whatsdown %{buildroot}%{_bindir}/whatsdown
install -D -p -m 0644 com.postbetasoftware.WhatsDown.desktop %{buildroot}%{_datadir}/applications/com.postbetasoftware.WhatsDown.desktop
install -D -p -m 0644 com.postbetasoftware.WhatsDown.svg %{buildroot}%{_datadir}/icons/hicolor/scalable/apps/com.postbetasoftware.WhatsDown.svg
install -D -p -m 0644 com.postbetasoftware.WhatsDown.metainfo.xml %{buildroot}%{_datadir}/metainfo/com.postbetasoftware.WhatsDown.metainfo.xml

%if 0%{?suse_version}
%suse_update_desktop_file -r com.postbetasoftware.WhatsDown Network InstantMessaging
%endif

%check
%if ! 0%{?suse_version}
desktop-file-validate %{buildroot}%{_datadir}/applications/com.postbetasoftware.WhatsDown.desktop
%endif

%files
%license LICENSE
%doc README.md
%{_bindir}/whatsdown
%{_datadir}/applications/com.postbetasoftware.WhatsDown.desktop
%{_datadir}/icons/hicolor/scalable/apps/com.postbetasoftware.WhatsDown.svg
%{_datadir}/metainfo/com.postbetasoftware.WhatsDown.metainfo.xml

%changelog
