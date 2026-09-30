# STAPT — Install guide

STAPT is a **native desktop app** (not a website). Install it on the computer, then open it like any other app.

**[→ Download latest installers](https://github.com/CodersCircle/stapt/releases/latest)**

---

## macOS

1. Download `STAPT-macOS-arm64.dmg` (Apple Silicon) or `STAPT-macOS-amd64.dmg` (Intel).
2. Open the DMG.
3. Double-click **Install STAPT.command** (not a yellow app icon).
4. STAPT copies itself into **Applications** and opens.

If macOS says *“STAPT.app Not Opened”* or *cannot verify*:

1. Click **Done**.
2. Open **System Settings → Privacy & Security**.
3. Scroll down and click **Open Anyway**.
4. Run **Install STAPT.command** again.

Terminal (always works, even if double-click is blocked):

```bash
curl -fsSL https://raw.githubusercontent.com/CodersCircle/stapt/main/scripts/install.sh | bash
```

Or, if the DMG is already open:

```bash
bash "/Volumes/STAPT/Install STAPT.command"
```

After install: **Cmd+Space** → type **STAPT** → Enter.

---

## Windows

1. Download `STAPT-Windows-x64-setup.exe`.
2. Run it. If **Windows protected your PC** appears: **More info** → **Run anyway**.
3. Open **Start Menu → STAPT**.

The installer does **not** need Administrator. It puts STAPT in your user Programs folder and adds a Start Menu shortcut.

PowerShell one-liner:

```powershell
irm https://raw.githubusercontent.com/CodersCircle/stapt/main/scripts/install.ps1 | iex
```

---

## Linux

### Debian / Ubuntu / Mint (`.deb`)

```bash
sudo dpkg -i STAPT-Linux-amd64.deb
sudo apt-get install -f -y
stapt
```

Needs GTK 3 and WebKitGTK (`libgtk-3-0` and `libwebkit2gtk-4.0-37` or `libwebkit2gtk-4.1-0`).

### Any Linux (tarball)

```bash
tar -xzf STAPT-Linux-x64.tar.gz
cd STAPT-Linux-x64
bash install.sh
stapt
```

That installs to `~/.local` and adds an application-menu entry.

One-liner:

```bash
curl -fsSL https://raw.githubusercontent.com/CodersCircle/stapt/main/scripts/install.sh | bash
```

---

## Daily use

| OS | Open app |
|----|----------|
| macOS | **Cmd+Space** → **STAPT** |
| Windows | **Start Menu** → **STAPT** |
| Linux | Application menu → **STAPT**, or `stapt` |

Data is stored locally: `~/.stapt` (Windows: `%USERPROFILE%\.stapt`).

---

## Build from source (developers)

Requires [Go 1.22+](https://go.dev/dl/) and [Wails prerequisites](https://wails.io/docs/gettingstarted/installation).

### macOS

```bash
git clone https://github.com/CodersCircle/stapt.git
cd stapt
chmod +x build.sh install-mac.sh
./build.sh
./install-mac.sh
```

Or double-click **`Install STAPT (Mac).command`** once.

### Windows

```bat
git clone https://github.com/CodersCircle/stapt.git
cd stapt
build.bat
powershell -ExecutionPolicy Bypass -File install-windows.ps1
```

### Linux

```bash
git clone https://github.com/CodersCircle/stapt.git
cd stapt
chmod +x build.sh install-linux.sh
./build.sh
./install-linux.sh
stapt
```

---

## Build installers (maintainers)

```bash
git tag v1.0.1
git push origin v1.0.1
```

GitHub Actions builds the DMG, Windows setup EXE, Linux `.deb`, and `.tar.gz` and attaches them to the release.
