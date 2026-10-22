%global debug_package %{nil}

Name:           whatsdown
Version:        0.2.0
Release:        1%{?dist}
Summary:        Native Linux WhatsApp client built with GTK 4 and Libadwaita

License:        MIT
URL:            https://github.com/EmirOven/WhatsDown
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.22
BuildRequires:  gtk4-devel >= 4.14
BuildRequires:  libadwaita-devel >= 1.5
BuildRequires:  sqlite-devel
BuildRequires:  blueprint-compiler
BuildRequires:  pkgconfig(gobject-introspection-1.0)
BuildRequires:  gcc
BuildRequires:  desktop-file-utils

Requires:       gtk4 >= 4.14
Requires:       libadwaita >= 1.5
Requires:       sqlite
Requires:       hicolor-icon-theme

%description
WhatsDown is a modern native Linux WhatsApp desktop client built with Go,
GTK 4, and Libadwaita, utilizing whatsmeow and gotk4.

%prep
%autosetup -n WhatsDown-%{version}

%build
# Compile Blueprint UI files
blueprint-compiler batch-compile ui ui ui/*.blp ui/components/*.blp

# Build Go binary with release flags
export CGO_ENABLED=1
export GOFLAGS="-trimpath -mod=vendor -modcacherw"
go build -ldflags="-s -w" -v -o whatsdown .

%install
install -D -p -m 0755 whatsdown %{buildroot}%{_bindir}/whatsdown
install -D -p -m 0644 com.postbetasoftware.WhatsDown.desktop %{buildroot}%{_datadir}/applications/com.postbetasoftware.WhatsDown.desktop
install -D -p -m 0644 com.postbetasoftware.WhatsDown.svg %{buildroot}%{_datadir}/icons/hicolor/scalable/apps/com.postbetasoftware.WhatsDown.svg

%check
desktop-file-validate %{buildroot}%{_datadir}/applications/com.postbetasoftware.WhatsDown.desktop

%files
%license LICENSE
%doc README.md
%{_bindir}/whatsdown
%{_datadir}/applications/com.postbetasoftware.WhatsDown.desktop
%{_datadir}/icons/hicolor/scalable/apps/com.postbetasoftware.WhatsDown.svg

%changelog
* Thu Oct 22 2026 Emir Öven <emiroven@postbetasoftware.com> - 0.2.0-1
- v0.2.0 (Colourful): custom chat wallpapers, window tinting, sidebar colour handles, group creation and permissions, @mention autocomplete, and Demo Mode

* Wed Oct 21 2026 Emir Öven <emiroven@postbetasoftware.com> - 0.1.0-1
- Initial release of WhatsDown for Fedora COPR
