#!/bin/bash
# Muestra desde la Mac el último resultado de la CI de et/store en IA.
#   ./test/ci/status.sh            último ci.sh, último nightly y la corrida larga de fuzzers
#   ./test/ci/status.sh ci         solo ci.sh   (~/etstore-ci/last-ci.txt)
#   ./test/ci/status.sh nightly    solo nightly (~/etstore-ci/last-nightly.txt)
#   ./test/ci/status.sh fuzzlong   solo la corrida larga de fuzzers (la última terminada o, si corre, la en curso)
set -uo pipefail

HOST=${ETSTORE_HOST:-ia}
what=${1:-todo}
ssh -o BatchMode=yes "$HOST" "export PATH=/opt/homebrew/bin:\$PATH; c=~/etstore-ci
  if [ $what = todo ] || [ $what = ci ]; then echo '### último ci.sh'; cat \$c/last-ci.txt 2>/dev/null || echo '(ninguno)'; echo; fi
  if [ $what = todo ] || [ $what = nightly ]; then echo '### último nightly'; cat \$c/last-nightly.txt 2>/dev/null || echo '(ninguno)'; echo; fi
  if [ $what = todo ] || [ $what = fuzzlong ]; then
    f=\$(ls -1d \$c/fuzzlong-*/ 2>/dev/null | tail -1)
    if [ -n \"\$f\" ] && ! grep -q '^== resultado' \$f/summary.txt 2>/dev/null; then
      echo \"### fuzzlong en curso (\$f)\"; cat \$f/summary.txt 2>/dev/null
      ps -axo etime,command | grep -v grep | grep -o -m1 '[0-9:-]* .*-fuzz [^ ]*' | sed 's/^/ahora: /'
    else echo '### último fuzzlong'; cat \$c/last-fuzzlong.txt 2>/dev/null || echo '(ninguno)'; fi
    ls \$c/skip-nightly-* 2>/dev/null | sed 's/^/marca para saltar el nocturno: /'
    echo
  fi
  echo '### ahora'
  [ -d \$c/lock ] && echo \"corriendo (pid \$(cat \$c/lock/pid 2>/dev/null))\" || echo 'nada corriendo'
  launchctl list | grep -q com.cgalvisleon.etstore-nightly && echo 'nocturno instalado (03:00)' || echo 'nocturno no instalado'
  echo \"copia subida: \$(cat ~/etstore/et.stamp 2>/dev/null || echo '?')\"
  ls \$c/fuzz-found 2>/dev/null | grep -q . && echo \"casos de fuzz guardados: \$(find \$c/fuzz-found -type f | wc -l | tr -d ' ')\"
  echo \"VM: \$(limactl list --format '{{.Status}}' etstore 2>/dev/null)\""
