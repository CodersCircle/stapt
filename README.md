# STAPT

Native desktop **SSH terminal**, **SFTP file manager**, and **lightweight editor**.

Built with Go, Wails, HTML, JavaScript, and Tailwind CSS. Opens as a real app window — not a browser tab.

![STAPT](docs/stapt-logo.png)

---

## Install (macOS / Windows / Linux)

**[Download installers](https://github.com/CodersCircle/stapt/releases/latest)** · step-by-step in **[INSTALL.md](INSTALL.md)**

| OS | File | What to do |
|----|------|------------|
| **macOS** Apple Silicon | [STAPT-macOS-arm64.dmg](https://github.com/CodersCircle/stapt/releases/latest/download/STAPT-macOS-arm64.dmg) | Open DMG → double-click **Install STAPT.command** |
| **macOS** Intel | [STAPT-macOS-amd64.dmg](https://github.com/CodersCircle/stapt/releases/latest/download/STAPT-macOS-amd64.dmg) | Same |
| **Windows** 64-bit | [STAPT-Windows-x64-setup.exe](https://github.com/CodersCircle/stapt/releases/latest/download/STAPT-Windows-x64-setup.exe) | Run the setup → Start Menu → **STAPT** |
| **Linux** Debian/Ubuntu | [STAPT-Linux-amd64.deb](https://github.com/CodersCircle/stapt/releases/latest/download/STAPT-Linux-amd64.deb) | `sudo dpkg -i STAPT-Linux-amd64.deb` |
| **Linux** any | [STAPT-Linux-x64.tar.gz](https://github.com/CodersCircle/stapt/releases/latest/download/STAPT-Linux-x64.tar.gz) | Extract → `bash install.sh` |

One-line install:

**macOS / Linux**
```bash
curl -fsSL https://raw.githubusercontent.com/CodersCircle/stapt/main/scripts/install.sh | bash
```

**Windows** (PowerShell)
```powershell
irm https://raw.githubusercontent.com/CodersCircle/stapt/main/scripts/install.ps1 | iex
```

---

## Features

- **Servers** — any SSH host (password or private key + optional passphrase)
- **Root paths** — allowed remote folders; terminal and files stay inside them
- **Terminal** — interactive PTY (bash, git, npm, composer, systemctl, …)
- **Files** — browse, edit, upload, download via SFTP
- **Editor** — syntax highlighting for PHP/Blade, JS, HTML, CSS, JSON, YAML, SQL, Markdown

Credentials are encrypted in `~/.stapt`. Secrets are never sent back to the UI after save.

---

## Build from source (developers)

```bash
git clone https://github.com/CodersCircle/stapt.git
cd stapt
```

**macOS / Linux:** `./build.sh` then `./install-mac.sh` or `./install-linux.sh`  
**Windows:** `build.bat` then `powershell -ExecutionPolicy Bypass -File install-windows.ps1`

---

## License

Use and distribute per your project policy. Repository: [CodersCircle/stapt](https://github.com/CodersCircle/stapt).
