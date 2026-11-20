#!/bin/bash
# Muestra desde la Mac el último resultado de la CI de et/store en IA.
#   ./test/ci/status.sh            último ci.sh y último nightly
#   ./test/ci/status.sh ci         solo ci.sh   (~/etstore-ci/last-ci.txt)
#   ./test/ci/status.sh nightly    solo nightly (~/etstore-ci/last-nightly.txt)
set -uo pipefail

HOST=${ETSTORE_HOST:-ia}
what=${1:-todo}
ssh -o BatchMode=yes "$HOST" "export PATH=/opt/homebrew/bin:\$PATH; c=~/etstore-ci
  if [ $what != nightly ]; then echo '### último ci.sh'; cat \$c/last-ci.txt 2>/dev/null || echo '(ninguno)'; echo; fi
  if [ $what != ci ]; then echo '### último nightly'; cat \$c/last-nightly.txt 2>/dev/null || echo '(ninguno)'; echo; fi
  echo '### ahora'
  [ -d \$c/lock ] && echo \"corriendo (pid \$(cat \$c/lock/pid 2>/dev/null))\" || echo 'nada corriendo'
  launchctl list | grep -q com.cgalvisleon.etstore-nightly && echo 'nocturno instalado (03:00)' || echo 'nocturno no instalado'
  echo \"copia subida: \$(cat ~/etstore/et.stamp 2>/dev/null || echo '?')\"
  ls \$c/fuzz-found 2>/dev/null | grep -q . && echo \"casos de fuzz guardados: \$(find \$c/fuzz-found -type f | wc -l | tr -d ' ')\"
  echo \"VM: \$(limactl list --format '{{.Status}}' etstore 2>/dev/null)\""
