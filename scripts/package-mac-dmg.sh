#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export COPYFILE_DISABLE=1

APP="build/bin/STAPT.app"
OUT="dist/STAPT-macOS-arm64.dmg"

[[ -d "$APP" ]] || { echo "Run ./build.sh first"; exit 1; }

mkdir -p dist
rm -f "$OUT"
hdiutil create -volname STAPT -srcfolder "$APP" -ov -format UDZO "$OUT"
echo "Created $OUT"
