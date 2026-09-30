#!/bin/bash
# Installs STAPT into Applications and clears macOS download quarantine
# so the app can open. Double-click this file (do not open STAPT.app here).
set -euo pipefail
export COPYFILE_DISABLE=1

DIR="$(cd "$(dirname "$0")" && pwd)"
SRC=""
for candidate in "$DIR/.payload/STAPT.app" "$DIR/STAPT.app"; do
  if [[ -x "${candidate}/Contents/MacOS/STAPT" ]]; then
    SRC="$candidate"
    break
  fi
done

if [[ -z "$SRC" ]]; then
  osascript -e 'display alert "STAPT installer" message "STAPT.app was not found next to this installer. Re-download the DMG from GitHub Releases." as critical' 2>/dev/null || true
  echo "STAPT.app not found. Re-download the DMG."
  read -r -p "Press Enter to close..."
  exit 1
fi

if /usr/bin/touch /Applications/.stapt-write-test 2>/dev/null; then
  rm -f /Applications/.stapt-write-test
  DEST="/Applications/STAPT.app"
else
  mkdir -p "${HOME}/Applications"
  DEST="${HOME}/Applications/STAPT.app"
fi

echo "Installing STAPT to ${DEST} ..."
rm -rf "${DEST}"
ditto --norsrc --noextattr --noqtn "${SRC}" "${DEST}"
find "${DEST}" -name '._*' -delete 2>/dev/null || true
xattr -cr "${DEST}" 2>/dev/null || true
xattr -d com.apple.quarantine "${DEST}" 2>/dev/null || true
codesign --force --deep --sign - "${DEST}" 2>/dev/null || true

echo ""
echo "STAPT is installed."
echo "Open it from Launchpad or Spotlight (Cmd+Space → STAPT)."
open "${DEST}"
osascript -e 'display notification "STAPT is in Applications. Open it from Spotlight or Launchpad." with title "STAPT installed"' 2>/dev/null || true
sleep 1
