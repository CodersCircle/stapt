# STAPT

Native desktop **SSH terminal**, **SFTP file manager**, and **lightweight code editor**.

Built with Go, Wails, HTML, JavaScript, and Tailwind CSS. Opens as a real app window — not a browser tab.

![STAPT](build/appicon.png)

---

## Who are you?

### I want the app — download & install

**[Download latest installers (macOS / Windows / Linux)](https://github.com/CodersCircle/stapt/releases/latest)**

| Platform | Direct download |
|----------|-----------------|
| macOS (Apple Silicon) | [STAPT-macOS-arm64.dmg](https://github.com/CodersCircle/stapt/releases/latest/download/STAPT-macOS-arm64.dmg) |
| macOS (Intel) | [STAPT-macOS-amd64.dmg](https://github.com/CodersCircle/stapt/releases/latest/download/STAPT-macOS-amd64.dmg) |
| Windows 64-bit | [STAPT-Windows-x64.exe](https://github.com/CodersCircle/stapt/releases/latest/download/STAPT-Windows-x64.exe) |
| Linux 64-bit | [STAPT-Linux-x64.tar.gz](https://github.com/CodersCircle/stapt/releases/latest/download/STAPT-Linux-x64.tar.gz) |

Step-by-step: **[INSTALL.md](INSTALL.md)**

### I want the source code — build myself

```bash
git clone git@github.com:CodersCircle/stapt.git
cd stapt
```

Then follow **[INSTALL.md](INSTALL.md)** for your OS.

---

## Features

- **Servers** — any SSH host (password or private key + optional passphrase)
- **Root paths** — allowed remote folders; terminal and files stay inside them
- **Terminal** — interactive PTY (bash, git, npm, composer, systemctl, …)
- **Files** — browse, edit, upload, download via SFTP
- **Editor** — syntax highlighting for PHP/Blade, JS, HTML, CSS, JSON, YAML, SQL, Markdown

Credentials are encrypted in `~/.stapt`. Secrets are never sent back to the UI after save.

---

## Quick build (developers)

**macOS / Linux:** `./build.sh`  
**Windows:** `build.bat`  

Output: `build/bin/STAPT.app` | `STAPT.exe` | `STAPT`

---

## License

Use and distribute per your project policy. Repository: [CodersCircle/stapt](https://github.com/CodersCircle/stapt).
