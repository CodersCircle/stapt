#!/usr/bin/env bash
# Install STAPT to ~/.local/share/stapt and ~/.local/bin/stapt
set -euo pipefail
cd "$(dirname "$0")"

BIN_SRC="build/bin/STAPT"
if [[ ! -x "$BIN_SRC" ]]; then
  echo "Run ./build.sh first."
  exit 1
fi

DEST_DIR="${HOME}/.local/share/stapt"
mkdir -p "${DEST_DIR}" "${HOME}/.local/bin"
install -m 755 "$BIN_SRC" "${DEST_DIR}/STAPT"
ln -sf "${DEST_DIR}/STAPT" "${HOME}/.local/bin/stapt"

DESKTOP="${HOME}/.local/share/applications/stapt.desktop"
cat > "$DESKTOP" <<EOF
[Desktop Entry]
Name=STAPT
Comment=SSH terminal and SFTP file manager
Exec=${DEST_DIR}/STAPT
Icon=utilities-terminal
Terminal=false
Type=Application
Categories=Network;Development;
EOF

echo "Installed. Run: stapt   (ensure ~/.local/bin is in PATH)"
echo "Or find STAPT in your application menu after re-login."
