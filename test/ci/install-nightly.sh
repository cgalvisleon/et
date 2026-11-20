#!/bin/bash
# Instala en IA el LaunchAgent de usuario com.cgalvisleon.etstore-nightly (sin sudo) que corre
# ~/etstore-ci/bin/nightly.sh a las 03:00. Desde la Mac: ./test/ci/install-nightly.sh
# Copia también test/ci a ~/etstore-ci/bin (lo mismo que hace ci.sh). Quitarlo: ./test/ci/uninstall-nightly.sh
set -euo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
HOST=${ETSTORE_HOST:-ia}
LABEL=com.cgalvisleon.etstore-nightly
SSH=(ssh -o BatchMode=yes "$HOST")

echo "== copiando test/ci a $HOST:~/etstore-ci/bin"
COPYFILE_DISABLE=1 tar czf - -C "$HERE" . \
  | "${SSH[@]}" 'set -e; c=~/etstore-ci; mkdir -p $c; rm -rf $c/bin.new; mkdir $c/bin.new; tar xzf - -C $c/bin.new
      rm -rf $c/bin; mv $c/bin.new $c/bin'

echo "== instalando ~/Library/LaunchAgents/$LABEL.plist"
"${SSH[@]}" "set -e
  [ -f ~/etstore/et.tgz ] || echo 'aviso: todavía no hay copia subida (~/etstore/et.tgz): corre ./test/ci/ci.sh'
  plist=~/Library/LaunchAgents/$LABEL.plist
  sed \"s|__HOME__|\$HOME|g\" ~/etstore-ci/bin/$LABEL.plist > \$plist
  plutil -lint \$plist
  launchctl bootout gui/\$(id -u)/$LABEL 2>/dev/null || true
  launchctl bootstrap gui/\$(id -u) \$plist
  launchctl print gui/\$(id -u)/$LABEL | grep -E 'state|path' | head -3"
echo "== listo: corre todas las noches a las 03:00 (resultado: ./test/ci/status.sh)"
