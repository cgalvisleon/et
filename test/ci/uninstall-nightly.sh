#!/bin/bash
# Quita de IA el LaunchAgent com.cgalvisleon.etstore-nightly. Desde la Mac: ./test/ci/uninstall-nightly.sh
# Deja ~/etstore-ci (resultados, línea base, casos de fuzz) tal cual.
set -euo pipefail

HOST=${ETSTORE_HOST:-ia}
LABEL=com.cgalvisleon.etstore-nightly
ssh -o BatchMode=yes "$HOST" "launchctl bootout gui/\$(id -u)/$LABEL 2>/dev/null || true
  rm -f ~/Library/LaunchAgents/$LABEL.plist
  launchctl list | grep -q $LABEL && echo 'sigue cargado' || echo 'nocturno quitado'"
