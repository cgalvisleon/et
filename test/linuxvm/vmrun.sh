#!/bin/bash
# Sincroniza el módulo et a la VM `etstore` (en IA), compila test/powercut allí (~/powercut) y corre dentro de la VM
# uno de los scripts de fallos de disco de esta carpeta.
# Uso (desde la Mac):  ./test/linuxvm/vmrun.sh <script> [VAR=valor …]
#   ./test/linuxvm/vmrun.sh powercut-lazyfs.sh ROUNDS=20 SEED=1
#   ./test/linuxvm/vmrun.sh powercut-flakey.sh DROP_ROUNDS=10 ERROR_ROUNDS=5
#   ./test/linuxvm/vmrun.sh diskfull.sh
#   ETSTORE_STOP=1 ./test/linuxvm/vmrun.sh diskfull.sh      # detiene la VM al terminar
# Los resultados quedan en la VM en ~/powercut-results/<script>-<fecha>/ (summary.txt y, por ronda fallida, sus
# salidas y su registro).
# ETSTORE_HOST=local: corre en IA mismo (lo usa test/ci): no compila en la Mac ni empaqueta; copia ETSTORE_TGZ.
set -euo pipefail

HOST=${ETSTORE_HOST:-ia}
VM=etstore
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
TGZ=${ETSTORE_TGZ:-'~/etstore/et.tgz'}
if [ "$HOST" = local ]; then SSH=(bash -c); else SSH=(ssh -o BatchMode=yes "$HOST"); fi
REMOTE_PATH='export PATH=/opt/homebrew/bin:$PATH'
[ $# -ge 1 ] || { echo "uso: $0 <script> [VAR=valor …]" >&2; exit 2; }
script=$1; shift
[ -f "$ROOT/test/linuxvm/$script" ] || { echo "no existe test/linuxvm/$script" >&2; exit 2; }

cd "$ROOT"
if [ "$HOST" != local ]; then
  # Si otro trabajo deja store/ a medio editar, espera (hasta 10 min) a que compile antes de copiar.
  for _ in $(seq 20); do go build -o /dev/null ./test/powercut/ 2>/dev/null && break; echo "   (store/ no compila todavía; espero 30 s)"; sleep 30; done
  echo "== empaquetando $ROOT"
  git ls-files -co --exclude-standard -z \
    | while IFS= read -r -d '' f; do [ -f "$f" ] && printf '%s\0' "$f"; done \
    | COPYFILE_DISABLE=1 tar czf - --null -T - \
    | "${SSH[@]}" 'mkdir -p ~/etstore && cat > ~/etstore/et.tgz'
  echo "$(date '+%F %T') $(git rev-parse --short HEAD)$(git diff --quiet HEAD -- . || echo ' +cambios')" \
    | "${SSH[@]}" 'cat > ~/etstore/et.stamp'
fi

echo "== copiando a la VM y compilando test/powercut"
"${SSH[@]}" "$REMOTE_PATH
st=\$(limactl list --format '{{.Status}}' $VM)
[ \"\$st\" = Running ] || limactl start --tty=false $VM >/dev/null 2>&1
limactl copy $TGZ $VM:/tmp/et.tgz
limactl shell $VM -- bash -lc 'rm -rf ~/et && mkdir -p ~/et && tar xzf /tmp/et.tgz -C ~/et 2>/dev/null && rm -f /tmp/et.tgz && cd ~/et && GOWORK=off nice -n 10 go build -o ~/powercut ./test/powercut'"

envs=$(printf '%q ' "$@")
echo "== $script $envs(en la VM)"
set +e
"${SSH[@]}" "$REMOTE_PATH; limactl shell $VM -- bash -lc 'cd ~/et && env $envs bash test/linuxvm/$script'"
rc=$?
set -e
if [ "${ETSTORE_STOP:-0}" = 1 ]; then
  "${SSH[@]}" "$REMOTE_PATH; limactl stop $VM" >/dev/null 2>&1 || true
fi
exit $rc
