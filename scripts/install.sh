#!/usr/bin/env bash
# One-command install for STAPT on macOS and Linux.
#   curl -fsSL https://raw.githubusercontent.com/CodersCircle/stapt/main/scripts/install.sh | bash
set -euo pipefail

REPO="CodersCircle/stapt"
BASE="https://github.com/${REPO}/releases/latest/download"

need() {
  command -v "$1" >/dev/null 2>&1 || { echo "Missing $1"; exit 1; }
}

install_macos() {
  need curl
  need hdiutil
  local arch file tmp mnt dest
  arch="$(uname -m)"
  if [[ "$arch" == "arm64" ]]; then
    file="STAPT-macOS-arm64.dmg"
  else
    file="STAPT-macOS-amd64.dmg"
  fi
  tmp="$(mktemp -d /tmp/stapt-install-XXXXXX)"
  echo "Downloading ${file} ..."
  curl -fL "${BASE}/${file}" -o "${tmp}/stapt.dmg"
  mnt="$(hdiutil attach "${tmp}/stapt.dmg" -nobrowse | awk '/\/Volumes\//{print $NF; exit}')"
  if [[ -z "$mnt" || ! -d "$mnt" ]]; then
    echo "Could not mount the installer disk image."
    hdiutil attach "${tmp}/stapt.dmg" -nobrowse
    exit 1
  fi
  if /usr/bin/touch /Applications/.stapt-write-test 2>/dev/null; then
    rm -f /Applications/.stapt-write-test
    dest="/Applications/STAPT.app"
  else
    mkdir -p "${HOME}/Applications"
    dest="${HOME}/Applications/STAPT.app"
  fi
  echo "Installing to ${dest} ..."
  rm -rf "$dest"
  if [[ -x "${mnt}/Install STAPT.command" ]]; then
    bash "${mnt}/Install STAPT.command"
  else
    local src=""
    for candidate in "${mnt}/.payload/STAPT.app" "${mnt}/STAPT.app"; do
      if [[ -x "${candidate}/Contents/MacOS/STAPT" ]]; then
        src="$candidate"
        break
      fi
    done
    [[ -n "$src" ]] || { echo "STAPT.app missing from DMG"; hdiutil detach "$mnt" -quiet || true; exit 1; }
    ditto --norsrc --noextattr --noqtn "$src" "$dest"
    xattr -cr "$dest" 2>/dev/null || true
    codesign --force --deep --sign - "$dest" 2>/dev/null || true
    open "$dest"
  fi
  hdiutil detach "$mnt" -quiet || true
  rm -rf "$tmp"
  echo "STAPT is installed. Open it from Spotlight (Cmd+Space → STAPT)."
}

install_linux() {
  need curl
  local tmp
  tmp="$(mktemp -d /tmp/stapt-install-XXXXXX)"
  if command -v dpkg >/dev/null 2>&1 && command -v sudo >/dev/null 2>&1; then
    echo "Downloading STAPT-Linux-amd64.deb ..."
    if curl -fL "${BASE}/STAPT-Linux-amd64.deb" -o "${tmp}/stapt.deb"; then
      echo "Installing system package (password may be required)..."
      if sudo dpkg -i "${tmp}/stapt.deb"; then
        sudo apt-get install -f -y >/dev/null 2>&1 || true
        rm -rf "$tmp"
        echo "STAPT is installed. Open it from the application menu, or run: stapt"
        return
      fi
    fi
    echo "Deb install failed, using the user installer..."
  fi
  echo "Downloading STAPT-Linux-x64.tar.gz ..."
  curl -fL "${BASE}/STAPT-Linux-x64.tar.gz" -o "${tmp}/stapt.tgz"
  tar -xzf "${tmp}/stapt.tgz" -C "$tmp"
  bash "$tmp"/STAPT-Linux-x64/install.sh
  rm -rf "$tmp"
}

case "$(uname -s)" in
  Darwin) install_macos ;;
  Linux) install_linux ;;
  MINGW*|MSYS*|CYGWIN*)
    echo "On Windows, in PowerShell run:"
    echo "  irm https://raw.githubusercontent.com/CodersCircle/stapt/main/scripts/install.ps1 | iex"
    exit 1
    ;;
  *)
    echo "Unsupported OS: $(uname -s)"
    exit 1
    ;;
esac
