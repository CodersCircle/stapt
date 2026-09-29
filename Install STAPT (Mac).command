#!/bin/bash
cd "$(dirname "$0")"
chmod +x install-mac.sh 2>/dev/null
./install-mac.sh
read -r -p "Press Enter to close..."
