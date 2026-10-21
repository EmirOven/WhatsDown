# Packaging WhatsDown

This directory contains configuration files and instructions for building and distributing WhatsDown on **Arch Linux (AUR)**, **Fedora (COPR)**, and **Ubuntu (Launchpad PPA)**.

---

## 1. Arch Linux (AUR)

Package files: [`packaging/aur/PKGBUILD`](PKGBUILD) and [`packaging/aur/.SRCINFO`](.SRCINFO).

### How to publish to the AUR:
1. Register an account at [aur.archlinux.org](https://aur.archlinux.org/) and add your SSH public key in your account settings.
2. Clone the empty AUR repository:
   ```bash
   git clone ssh://aur@aur.archlinux.org/whatsdown-git.git
   cd whatsdown-git
   ```
3. Copy the packaging files from this repository:
   ```bash
   cp /path/to/WhatsDown/packaging/aur/PKGBUILD .
   cp /path/to/WhatsDown/packaging/aur/.SRCINFO .
   ```
4. Test building locally with `makepkg`:
   ```bash
   makepkg -sic
   ```
5. Whenever you update `PKGBUILD`, regenerate `.SRCINFO` before committing:
   ```bash
   makepkg --printsrcinfo > .SRCINFO
   git add PKGBUILD .SRCINFO
   git commit -m "Update WhatsDown package"
   git push origin master
   ```

Users can install it with:
```bash
yay -S whatsdown-git
# or
paru -S whatsdown-git
```

---

## 2. Fedora (COPR)

Package files: [`packaging/copr/whatsdown.spec`](copr/whatsdown.spec) and [`.copr/Makefile`](../.copr/Makefile).

COPR builds RPM packages in the cloud for Fedora 40, Fedora 41, Rawhide, etc., and can automatically build new commits pushed to GitHub.

### How to set up Fedora COPR:
1. Log in to [copr.fedorainfracloud.org](https://copr.fedorainfracloud.org/) using your Fedora Account (FAS).
2. Click **New Project**:
   - **Project Name:** `whatsdown`
   - **Chroots:** Select current Fedora versions (`fedora-40-x86_64`, `fedora-41-x86_64`, `fedora-rawhide-x86_64`).
3. Click **New Package** within your project:
   - **Package Name:** `whatsdown`
   - **Source type:** Select **SCM**
   - **Clone URL:** `https://github.com/EmirOven/WhatsDown.git`
   - **Committish:** `main`
   - **Source build method:** Select **`make_srpm`** (uses `.copr/Makefile` to archive the git tree automatically)
   - **Spec file:** `packaging/copr/whatsdown.spec`
   - **Subdirectory:** *(Leave completely empty/blank)*
   - Make sure **"Enable internet access during build"** is checked in project settings.
4. Enable **Auto-rebuilds** / Webhook:
   - Under project settings, copy the GitHub Webhook URL provided by COPR and paste it into `GitHub Repository -> Settings -> Webhooks` with Content-type `application/json`.
5. Trigger your first build by clicking **Rebuild**.

Users can install it on Fedora with:
```bash
sudo dnf copr enable <your-username>/whatsdown
sudo dnf install whatsdown
```

---

## 3. Ubuntu (Launchpad PPA)

Package files: [`debian/`](../debian/).

Launchpad builds and hosts Debian packages for Ubuntu distributions (e.g. `noble`, `oracular`).

### Option A: Launchpad Git Recipe (Automated Cloud Builds - Recommended)
1. Log in to [launchpad.net](https://launchpad.net/) and create a PPA (e.g. `ppa:your-username/whatsdown`).
2. Link your GitHub account or import your GitHub repository into Launchpad under your profile (`Code -> Import a repository`).
3. Click **Create packaging recipe**:
   - **Recipe name:** `whatsdown-daily`
   - **Recipe text:**
     ```
     # git-build-recipe format 0.4 deb-version {debversion}+git{revtime}
     https://github.com/EmirOven/WhatsDown main
     ```
   - **Target PPA:** `ppa:your-username/whatsdown`
   - **Target series:** `noble` (Ubuntu 24.04), `oracular`, etc.
4. Launchpad will automatically build directly from your Git branch using the `debian/` directory in the repository!

### Option B: Upload Source Package with `dput`
To manually build and upload a source package to your PPA:
1. Ensure your GPG key is registered with Launchpad:
   ```bash
   sudo apt install -y devscripts debhelper build-essential dput
   ```
2. Build the source package:
   ```bash
   # From the root of the WhatsDown repo:
   debuild -S -sa
   ```
3. Upload to your Launchpad PPA:
   ```bash
   dput ppa:your-username/whatsdown ../whatsdown_0.1.0-1_source.changes
   ```

Users can install it on Ubuntu with:
```bash
sudo add-apt-repository ppa:your-username/whatsdown
sudo apt update
sudo apt install whatsdown
```
