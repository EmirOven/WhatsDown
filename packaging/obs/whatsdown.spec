%global debug_package %{nil}

Name:           whatsdown
Version:        3
Release:        0
Summary:        Native Linux WhatsApp client built with GTK 4 and Libadwaita

License:        MIT
Group:          Productivity/Networking/Instant Messenger
URL:            https://github.com/EmirOven/WhatsDown
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.22
BuildRequires:  pkgconfig
BuildRequires:  pkgconfig(gstreamer-1.0)
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
%if 0%{?suse_version}
Requires:       gstreamer-plugins-base
Requires:       gstreamer-plugins-good
Requires:       poppler-tools
%else
Requires:       gstreamer1-plugins-base
Requires:       gstreamer1-plugins-good
Requires:       poppler-utils
%endif
Requires:       libadwaita >= 1.5
Requires:       sqlite
Requires:       hicolor-icon-theme

%description
WhatsDown is a modern native Linux WhatsApp desktop client built with Go,
GTK 4, and Libadwaita, utilizing whatsmeow and gotk4.

%prep
# Copy SCM sources and unpacked obscpio trees from SOURCES into current BUILD directory
cp -a %{_sourcedir}/* .
rm -f *.obscpio _scmsync.obsinfo

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
* Thu Oct 22 2026 Emir Öven <emiroven@postbetasoftware.com> - 3-0
- Release 3: reduce memory usage during chat loading and sync, add clickable URLs and message day separators.

* Thu Oct 22 2026 Emir Öven <emiroven@postbetasoftware.com> - 0.2.0-0
- v0.2.0 (Colourful): custom chat wallpapers, window tinting, sidebar colour handles, group creation and permissions, @mention autocomplete, and Demo Mode

* Wed Oct 21 2026 Emir Öven <emiroven@postbetasoftware.com> - 0.1.0-0
- Initial release of WhatsDown
