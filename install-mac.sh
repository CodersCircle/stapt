#!/usr/bin/env bash
# One-time install: STAPT → ~/Applications (normal double-click app).
set -euo pipefail
cd "$(dirname "$0")"
export PATH="/opt/homebrew/bin:/usr/local/bin:${HOME}/go/bin:${PATH}"
export COPYFILE_DISABLE=1

SRC="build/bin/STAPT.app"
DEST="${HOME}/Applications/STAPT.app"

if [[ ! -x "${SRC}/Contents/MacOS/STAPT" ]]; then
  echo "Building STAPT..."
  ./build.sh
fi

echo "Installing to ${DEST} ..."
rm -rf "${DEST}"
mkdir -p "${HOME}/Applications"
ditto --noextattr --noqtn "${SRC}" "${DEST}"

find "${DEST}" -name '._*' -delete 2>/dev/null || true
dot_clean -m "${DEST}" 2>/dev/null || true
xattr -cr "${DEST}" 2>/dev/null || true
codesign --force --deep -s - "${DEST}"

echo ""
echo "STAPT is installed. Open from Launchpad or Spotlight (Cmd+Space → STAPT)."
open "${DEST}"
