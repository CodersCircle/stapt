#!/usr/bin/env bash
# Install STAPT to ~/.local and the application menu.
set -euo pipefail
cd "$(dirname "$0")"

BIN_SRC="build/bin/STAPT"
if [[ ! -x "$BIN_SRC" ]]; then
  echo "Run ./build.sh first."
  exit 1
fi

DEST_DIR="${HOME}/.local/share/stapt"
mkdir -p "${DEST_DIR}" "${HOME}/.local/bin" "${HOME}/.local/share/applications"
install -m 755 "$BIN_SRC" "${DEST_DIR}/STAPT"
ln -sf "${DEST_DIR}/STAPT" "${HOME}/.local/bin/stapt"

ICON_LINE="utilities-terminal"
if [[ -f build/appicon.png ]]; then
  install -m 644 build/appicon.png "${DEST_DIR}/icon.png"
  ICON_LINE="${DEST_DIR}/icon.png"
elif [[ -f docs/stapt-logo.png ]]; then
  install -m 644 docs/stapt-logo.png "${DEST_DIR}/icon.png"
  ICON_LINE="${DEST_DIR}/icon.png"
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

echo "Installed. Run: stapt   (ensure ~/.local/bin is in PATH)"
echo "Or find STAPT in your application menu."
