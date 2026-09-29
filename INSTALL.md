# STAPT — Install guide

STAPT is a **native desktop app** (not a website). Pick **installer** (recommended) or **source code** (developers).

---

## Download installers (recommended)

Get the latest files from **GitHub Releases** (one click, no Terminal):

**[→ Download STAPT releases](https://github.com/CodersCircle/stapt/releases/latest)**

| Platform | File | After download |
|----------|------|----------------|
| **macOS** (Apple Silicon) | `STAPT-macOS-arm64.dmg` | Open DMG → drag **STAPT** to **Applications** → open from Launchpad |
| **macOS** (Intel) | `STAPT-macOS-amd64.dmg` | Same as above |
| **Windows** | `STAPT-Windows-x64.exe` | Run installer → Start Menu → **STAPT** |
| **Linux** | `STAPT-Linux-x64.tar.gz` | Extract → run `STAPT`, or use `install-linux.sh` from source |

First launch on macOS: if blocked, **System Settings → Privacy & Security → Open Anyway**.

Data is stored locally: `~/.stapt` (Windows: `%USERPROFILE%\.stapt`).

---

## Install from source (developers)

### macOS

```bash
git clone git@github.com:CodersCircle/stapt.git
cd stapt
chmod +x build.sh install-mac.sh
./build.sh
./install-mac.sh
```

Or double-click **`Install STAPT (Mac).command`** once.

### Windows

```bat
git clone git@github.com:CodersCircle/stapt.git
cd stapt
build.bat
powershell -ExecutionPolicy Bypass -File install-windows.ps1
```

### Linux

```bash
git clone git@github.com:CodersCircle/stapt.git
cd stapt
chmod +x build.sh install-linux.sh
./build.sh
./install-linux.sh
stapt
```

Requires [Go 1.22+](https://go.dev/dl/) and [Wails prerequisites](https://wails.io/docs/gettingstarted/installation).

---

## Daily use (after install)

| OS | Open app |
|----|----------|
| macOS | **Cmd+Space** → type **STAPT** → Enter |
| Windows | **Start Menu** → **STAPT** |
| Linux | Application menu → **STAPT**, or run `stapt` |

No Terminal or `.command` files needed for end users.

---

## Build installers (maintainers)

Tag a version to build all platforms on GitHub Actions:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Artifacts appear on the Releases page automatically.
