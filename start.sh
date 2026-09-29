#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
export PATH="/opt/homebrew/bin:/usr/local/bin:${HOME}/go/bin:${PATH}"

APP="build/bin/STAPT.app"
BIN="${APP}/Contents/MacOS/STAPT"

if [[ ! -x "$BIN" ]]; then
  echo "STAPT is not built. Run: ./build.sh"
  read -r -p "Press Enter to close..." || true
  exit 1
fi

find "$APP" -name '._*' -delete 2>/dev/null || true
dot_clean -m "$APP" 2>/dev/null || true
xattr -cr "$APP" 2>/dev/null || true
codesign --force --deep -s - "$APP" 2>/dev/null || true

echo "Starting STAPT..."
open "$APP" 2>/dev/null || exec "$BIN" &
sleep 2
osascript -e 'tell application "STAPT" to activate' 2>/dev/null || true

if [[ -t 0 ]]; then
  read -r -p "Press Enter to close..." || true
fi
