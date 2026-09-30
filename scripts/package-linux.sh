#!/usr/bin/env bash
# Package Linux .deb (system install) and .tar.gz (user install).
set -euo pipefail
cd "$(dirname "$0")/.."

BIN="${BIN:-build/bin/STAPT}"
[[ -x "$BIN" ]] || { echo "Build the Linux binary first (build/bin/STAPT)"; exit 1; }

VERSION="${VERSION:-}"
if [[ -z "$VERSION" ]]; then
  VERSION="$(python3 -c "import json; print(json.load(open('wails.json'))['info']['productVersion'])")"
fi
VERSION="${VERSION#v}"

mkdir -p dist
ICON="build/appicon.png"
[[ -f "$ICON" ]] || ICON="docs/stapt-logo.png"

# --- user tarball ---
TGZ_DIR="$(mktemp -d /tmp/stapt-tgz-XXXXXX)"
DEB_DIR=""
trap 'rm -rf "$TGZ_DIR" "$DEB_DIR"' EXIT
mkdir -p "$TGZ_DIR/STAPT-Linux-x64"
cp "$BIN" "$TGZ_DIR/STAPT-Linux-x64/STAPT"
chmod 755 "$TGZ_DIR/STAPT-Linux-x64/STAPT"
if [[ -f "$ICON" ]]; then
  cp "$ICON" "$TGZ_DIR/STAPT-Linux-x64/icon.png"
fi
cat > "$TGZ_DIR/STAPT-Linux-x64/install.sh" <<'EOS'
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
DEST_DIR="${HOME}/.local/share/stapt"
mkdir -p "${DEST_DIR}" "${HOME}/.local/bin" "${HOME}/.local/share/applications"
install -m 755 STAPT "${DEST_DIR}/STAPT"
ln -sf "${DEST_DIR}/STAPT" "${HOME}/.local/bin/stapt"
if [[ -f icon.png ]]; then
  install -m 644 icon.png "${DEST_DIR}/icon.png"
  ICON_LINE="${DEST_DIR}/icon.png"
else
  ICON_LINE="utilities-terminal"
fi
cat > "${HOME}/.local/share/applications/stapt.desktop" <<EOF
[Desktop Entry]
Name=STAPT
Comment=SSH terminal and SFTP file manager
Exec=${DEST_DIR}/STAPT
Icon=${ICON_LINE}
Terminal=false
Type=Application
Categories=Network;Development;
StartupNotify=true
EOF
echo "STAPT installed. Run: stapt"
echo "Or open STAPT from your application menu."
EOS
chmod +x "$TGZ_DIR/STAPT-Linux-x64/install.sh"
cat > "$TGZ_DIR/STAPT-Linux-x64/README.txt" <<EOF
STAPT ${VERSION} for Linux
========================

1. Double-click install.sh  (or:  bash install.sh)
2. Open STAPT from the application menu, or run:  stapt

Needs GTK 3 and WebKitGTK (Ubuntu: sudo apt install libgtk-3-0 libwebkit2gtk-4.0-37).
EOF
tar -czf dist/STAPT-Linux-x64.tar.gz -C "$TGZ_DIR" STAPT-Linux-x64
echo "Created dist/STAPT-Linux-x64.tar.gz"

# --- .deb ---
DEB_DIR="$(mktemp -d /tmp/stapt-deb-XXXXXX)"
mkdir -p "$DEB_DIR/DEBIAN" \
  "$DEB_DIR/usr/bin" \
  "$DEB_DIR/usr/share/applications" \
  "$DEB_DIR/usr/share/pixmaps" \
  "$DEB_DIR/usr/share/icons/hicolor/256x256/apps"
install -m 755 "$BIN" "$DEB_DIR/usr/bin/stapt"
if [[ -f "$ICON" ]]; then
  install -m 644 "$ICON" "$DEB_DIR/usr/share/pixmaps/stapt.png"
  install -m 644 "$ICON" "$DEB_DIR/usr/share/icons/hicolor/256x256/apps/stapt.png"
fi
cat > "$DEB_DIR/usr/share/applications/stapt.desktop" <<EOF
[Desktop Entry]
Name=STAPT
Comment=SSH terminal and SFTP file manager
Exec=/usr/bin/stapt
Icon=stapt
Terminal=false
Type=Application
Categories=Network;Development;
StartupNotify=true
EOF
SIZE_KB="$(du -sk "$DEB_DIR" | awk '{print $1}')"
cat > "$DEB_DIR/DEBIAN/control" <<EOF
Package: stapt
Version: ${VERSION}
Section: net
Priority: optional
Architecture: amd64
Maintainer: CodersCircle <circlecoders@gmail.com>
Installed-Size: ${SIZE_KB}
Depends: libgtk-3-0, libwebkit2gtk-4.0-37 | libwebkit2gtk-4.1-0
Description: SSH terminal and SFTP file manager
 Native desktop SSH terminal, SFTP files, and a lightweight editor.
EOF
if dpkg-deb --root-owner-group --build "$DEB_DIR" dist/STAPT-Linux-amd64.deb 2>/dev/null; then
  echo "Created dist/STAPT-Linux-amd64.deb"
else
  dpkg-deb --build "$DEB_DIR" dist/STAPT-Linux-amd64.deb
  echo "Created dist/STAPT-Linux-amd64.deb"
fi
