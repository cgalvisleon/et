#!/bin/bash
# Sincroniza el módulo et de esta Mac a la VM `etstore` (en IA) y corre
# `go test` allí con -p 1, nice -n 10, GOWORK=off y /usr/bin/time -v.
# Uso:  ./test/linuxvm/run.sh [args de go test]      (por defecto ./store/...)
#   ./test/linuxvm/run.sh -race ./store/...
#   ./test/linuxvm/run.sh -run TestModelCrash -count 3 ./store/
# Arranca la VM si está detenida; con ETSTORE_STOP=1 la detiene al final.
# ETSTORE_SYNC_ONLY=1: solo empaqueta y sube a IA (~/etstore/et.tgz y et.stamp), sin tocar la VM.
# ETSTORE_HOST=local: corre en IA mismo (lo usa test/ci): no empaqueta y copia a la VM ETSTORE_TGZ.
set -euo pipefail

HOST=${ETSTORE_HOST:-ia}
VM=etstore
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
TGZ=${ETSTORE_TGZ:-'~/etstore/et.tgz'}
if [ "$HOST" = local ]; then SSH=(bash -c); else SSH=(ssh -o BatchMode=yes "$HOST"); fi
REMOTE_PATH='export PATH=/opt/homebrew/bin:$PATH'
[ $# -gt 0 ] || set -- ./store/...

inicio=$(date +%s)

cd "$ROOT"
if [ "$HOST" != local ]; then
  echo "== empaquetando $ROOT (archivos de git, sin .git ni ignorados)"
  git ls-files -co --exclude-standard -z \
    | while IFS= read -r -d '' f; do [ -f "$f" ] && printf '%s\0' "$f"; done \
    | COPYFILE_DISABLE=1 tar czf - --null -T - \
    | "${SSH[@]}" 'mkdir -p ~/etstore && cat > ~/etstore/et.tgz'
  # Qué se subió (lo leen test/ci/ci.sh y nightly.sh): fecha, commit y si había cambios sin commit.
  echo "$(date '+%F %T') $(git rev-parse --short HEAD)$(git diff --quiet HEAD -- . || echo ' +cambios')" \
    | "${SSH[@]}" 'cat > ~/etstore/et.stamp'
  [ "${ETSTORE_SYNC_ONLY:-0}" = 1 ] && { echo "== subido a $HOST:~/etstore/et.tgz"; exit 0; }
fi

echo "== copiando a la VM"
"${SSH[@]}" "$REMOTE_PATH
st=\$(limactl list --format '{{.Status}}' $VM)
[ \"\$st\" = Running ] || limactl start --tty=false $VM >/dev/null 2>&1
limactl copy $TGZ $VM:/tmp/et.tgz
limactl shell $VM -- bash -c 'rm -rf ~/et && mkdir -p ~/et && tar xzf /tmp/et.tgz -C ~/et 2>/dev/null && rm -f /tmp/et.tgz'"

args=$(printf '%q ' "$@")
echo "== go test -p 1 $args (en la VM)"
set +e
"${SSH[@]}" "$REMOTE_PATH; limactl shell $VM -- bash -lc 'cd ~/et && export GOWORK=off && go version && /usr/bin/time -v nice -n 10 go test -p 1 $args'"
rc=$?
set -e

if [ "${ETSTORE_STOP:-0}" = 1 ]; then
  "${SSH[@]}" "$REMOTE_PATH; limactl stop $VM" >/dev/null 2>&1 || true
fi
echo "== run.sh terminó en $(( $(date +%s) - inicio )) s (go test salió con $rc)"
exit $rc
