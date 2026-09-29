#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

if ! command -v go >/dev/null 2>&1; then
  echo "Go 1.22+ is required: https://go.dev/dl/"
  exit 1
fi
if ! command -v wails >/dev/null 2>&1; then
  echo "Installing Wails CLI..."
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  export PATH="${PATH}:$(go env GOPATH)/bin"
fi

export COPYFILE_DISABLE=1

set +e
wails build -clean "$@"
build_rc=$?
set -e

APP="build/bin/STAPT.app"
if [[ -d "$APP" ]]; then
  find build/bin -name '._*' -delete 2>/dev/null || true
  dot_clean -m "$APP" 2>/dev/null || true
  xattr -cr "$APP" 2>/dev/null || true
  codesign --force --deep -s - "$APP" 2>/dev/null || true
fi

if [[ $build_rc -ne 0 && -x "${APP}/Contents/MacOS/STAPT" ]]; then
  echo "Note: Wails codesign failed (common on USB volumes); app was re-signed locally."
  build_rc=0
fi

[[ $build_rc -eq 0 ]] || exit $build_rc

echo ""
echo "Built: build/bin/STAPT.app"
echo "Install (once): ./install-mac.sh"
echo "Package DMG: ./scripts/package-mac-dmg.sh"
