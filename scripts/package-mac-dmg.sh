#!/usr/bin/env bash
# Build a Mac installer DMG: hidden app payload + Install STAPT.command.
set -euo pipefail
cd "$(dirname "$0")/.."
export COPYFILE_DISABLE=1

APP="${APP:-build/bin/STAPT.app}"
OUT="${1:-dist/STAPT-macOS-arm64.dmg}"

[[ -x "${APP}/Contents/MacOS/STAPT" ]] || { echo "Run ./build.sh first"; exit 1; }

STAGE="$(mktemp -d /tmp/stapt-dmg-XXXXXX)"
cleanup() { rm -rf "$STAGE"; }
trap cleanup EXIT

mkdir -p "$STAGE/.payload"
ditto --norsrc --noextattr --noqtn "$APP" "$STAGE/.payload/STAPT.app"
find "$STAGE/.payload/STAPT.app" -name '._*' -delete 2>/dev/null || true
xattr -cr "$STAGE/.payload/STAPT.app" 2>/dev/null || true
codesign --force --deep --sign - "$STAGE/.payload/STAPT.app" 2>/dev/null || true

cp "scripts/dmg/Install-STAPT.command" "$STAGE/Install STAPT.command"
cp "scripts/dmg/How-to-Install.txt" "$STAGE/How to Install.txt"
chmod 755 "$STAGE/Install STAPT.command"
chmod 644 "$STAGE/How to Install.txt"
xattr -cr "$STAGE" 2>/dev/null || true

mkdir -p "$(dirname "$OUT")"
rm -f "$OUT"
hdiutil create -volname STAPT -srcfolder "$STAGE" -ov -format UDZO "$OUT"
echo "Created $OUT"
