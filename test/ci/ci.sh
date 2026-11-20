#!/bin/bash
# Verificación de un cambio del motor store (tarea 48 de F-23), en IA.
#
# Desde la Mac (repo et):   ./test/ci/ci.sh
#   1. espera a que store/, test/powercut y test/storebench compilen (hasta 10 min: otro agente puede estar editando);
#   2. sube los archivos de git a IA con test/linuxvm/run.sh (ETSTORE_SYNC_ONLY=1), sin git push;
#   3. instala estos scripts en IA (~/etstore-ci/bin) y corre allí la parte de IA.
# En IA:                    ~/etstore-ci/bin/ci.sh     (prueba la última copia subida, ~/etstore/et.tgz)
#   VM Linux: go test -p 1 ./store/..., lo mismo con -race, cortes cortos con LazyFS y dm-flakey; detiene la VM;
#   macOS:    go test -p 1 ./store/...
# Resumen en IA: ~/etstore-ci/<fecha>/summary.txt (copia en ~/etstore-ci/last-ci.txt), un <paso>.log por paso.
# Variables: CI_LAZYFS_ROUNDS (5), CI_DROP_ROUNDS (3), CI_ERROR_ROUNDS (2), MIN_FREE (30 %).
set -uo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
HOST=${ETSTORE_HOST:-ia}

# ---- en la Mac: sincronizar y lanzar en IA ----
if git -C "$HERE" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  ROOT=$(cd "$HERE/../.." && pwd)
  cd "$ROOT" || exit 2
  echo "== compilando store/ en la Mac (solo build y vet, sin pruebas)"
  for i in $(seq 20); do
    go build ./store/... ./test/powercut ./test/storebench && go vet ./store/... && break
    [ "$i" = 20 ] && { echo "store/ no compila tras 10 min: no se sincroniza. Reintenta más tarde."; exit 1; }
    echo "   (store/ no compila todavía; espero 30 s)"; sleep 30
  done
  ETSTORE_SYNC_ONLY=1 ETSTORE_HOST=$HOST bash test/linuxvm/run.sh || exit 1
  echo "== instalando test/ci en $HOST:~/etstore-ci/bin"
  ssh -o BatchMode=yes "$HOST" 'set -e; c=~/etstore-ci; mkdir -p $c; rm -rf $c/bin.new; mkdir $c/bin.new
    tar xzf ~/etstore/et.tgz -C $c/bin.new --strip-components 2 test/ci
    rm -rf $c/bin; mv $c/bin.new $c/bin' || exit 1
  exec ssh -o BatchMode=yes "$HOST" "CI_LAZYFS_ROUNDS=${CI_LAZYFS_ROUNDS:-5} CI_DROP_ROUNDS=${CI_DROP_ROUNDS:-3} \
CI_ERROR_ROUNDS=${CI_ERROR_ROUNDS:-2} MIN_FREE=${MIN_FREE:-30} bash ~/etstore-ci/bin/ci.sh"
fi

# ---- en IA ----
# shellcheck source=lib.sh
source "$HERE/lib.sh"
take_lock 3600 || exit 1
take_snapshot || exit 1
OUT=$CI/$(date +%Y%m%d-%H%M%S)
mkdir -p "$OUT"
SUMMARY=$OUT/summary.txt
start=$(date +%s)

say "== ci.sh en IA, $(date '+%F %T') — copia: $(cat "$CI/src.stamp")"
step vm-test    1200 vm_test ./store/...
step vm-race    1200 vm_test -race ./store/...
step vm-lazyfs  1200 vm_script powercut-lazyfs.sh "ROUNDS=${CI_LAZYFS_ROUNDS:-5}"
step vm-flakey  1200 vm_script powercut-flakey.sh "DROP_ROUNDS=${CI_DROP_ROUNDS:-3}" "ERROR_ROUNDS=${CI_ERROR_ROUNDS:-2}"
vm_stop
step mac-test   1200 mac_go test -p 1 ./store/...

if [ "$failures" = 0 ]; then result=OK; else result="FALLA ($failures pasos)"; fi
say "== resultado: $result en $(( ($(date +%s) - start) / 60 )) min — VM detenida — $OUT"
cp "$SUMMARY" "$CI/last-ci.txt"
prune '20[0-9]*' 30
[ "$failures" = 0 ]
