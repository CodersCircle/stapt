#!/usr/bin/env bash
# One-time install: STAPT → /Applications (or ~/Applications if not writable).
set -euo pipefail
cd "$(dirname "$0")"
export PATH="/opt/homebrew/bin:/usr/local/bin:${HOME}/go/bin:${PATH}"
export COPYFILE_DISABLE=1

SRC="build/bin/STAPT.app"

if [[ ! -x "${SRC}/Contents/MacOS/STAPT" ]]; then
  echo "Building STAPT..."
  ./build.sh
fi

if /usr/bin/touch /Applications/.stapt-write-test 2>/dev/null; then
  rm -f /Applications/.stapt-write-test
  DEST="/Applications/STAPT.app"
else
  mkdir -p "${HOME}/Applications"
  DEST="${HOME}/Applications/STAPT.app"
fi

echo "Installing to ${DEST} ..."
rm -rf "${DEST}"
mkdir -p "$(dirname "${DEST}")"
ditto --norsrc --noextattr --noqtn "${SRC}" "${DEST}"

find "${DEST}" -name '._*' -delete 2>/dev/null || true
dot_clean -m "${DEST}" 2>/dev/null || true
xattr -cr "${DEST}" 2>/dev/null || true
xattr -d com.apple.quarantine "${DEST}" 2>/dev/null || true
codesign --force --deep -s - "${DEST}"

echo ""
echo "STAPT is installed. Open from Launchpad or Spotlight (Cmd+Space → STAPT)."
open "${DEST}"
