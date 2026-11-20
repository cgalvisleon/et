#!/bin/bash
# Medidas de producción del motor store (F-23, tareas 71 y 72) en IA, bajo el candado de test/ci (un solo proceso
# pesado a la vez, memoria libre ≥ MIN_FREE %). Se lanza desde la Mac:
#   ./test/storebench/medir.sh memoria     # 71: VM, STORE_CACHE_SIZE=512, 100 M claves de 100 B; RSS tras cargar y tras reabrir
#   ./test/storebench/medir.sh caida-mac   # 72: macOS de IA, carga hasta CRASH_MB de WAL sin Close, kill -9, Open (REPS veces)
#   ./test/storebench/medir.sh caida-vm    # 72: lo mismo en la VM, con la caché del SO vaciada antes de cada Open
# Variables: N (100000000), CACHE_MB (512), CRASH_MB (1024), REPS (3), CKPT_TIMEOUT (3600), MAX_WAL (vacío = por omisión).
# Sube la copia de trabajo (si compila) con test/linuxvm/run.sh (ETSTORE_SYNC_ONLY=1) y la guarda en IA como
# ~/etbench/et.tgz + et.stamp, así otra subida no cambia lo que se mide. Salida en IA: ~/etbench/<modo>-<fecha>.log.
# Al terminar borra los datos (en la VM además fstrim) y deja la VM detenida.
set -uo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
mode=${1:?uso: medir.sh memoria|caida-mac|caida-vm}

# ---- en la Mac: compilar, subir y lanzar en IA ----
if git -C "$HERE" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  ROOT=$(cd "$HERE/../.." && pwd)
  cd "$ROOT" || exit 2
  for i in $(seq 20); do
    go build ./store/... ./test/storebench && break
    [ "$i" = 20 ] && { echo "store/ no compila tras 10 min"; exit 1; }
    echo "   (store/ no compila todavía; espero 30 s)"; sleep 30
  done
  ETSTORE_SYNC_ONLY=1 bash test/linuxvm/run.sh || exit 1
  ssh -o BatchMode=yes ia 'set -e; mkdir -p ~/etbench; cp ~/etstore/et.tgz ~/etbench/et.tgz; cp ~/etstore/et.stamp ~/etbench/et.stamp
    rm -rf ~/etbench/src; mkdir ~/etbench/src; tar xzf ~/etbench/et.tgz -C ~/etbench/src 2>/dev/null' || exit 1
  exec ssh -o BatchMode=yes ia "N=${N:-100000000} CACHE_MB=${CACHE_MB:-512} CRASH_MB=${CRASH_MB:-1024} REPS=${REPS:-3} \
CKPT_TIMEOUT=${CKPT_TIMEOUT:-3600} MAX_WAL=${MAX_WAL:-} MIN_FREE=${MIN_FREE:-30} \
bash ~/etbench/src/test/storebench/medir.sh $mode 2>&1 | tee ~/etbench/$mode-\$(date +%Y%m%d-%H%M%S).log"
fi

# ---- en IA ----
B=$HOME/etbench
# shellcheck source=../ci/lib.sh
source "$B/src/test/ci/lib.sh"
SUMMARY=/dev/null
take_lock 10800 || exit 1
wait_memory 3600 || { echo "memoria libre $(mem_free) % < $MIN_FREE %: no se mide"; exit 1; }
VM=etstore
envwal=""; [ -n "${MAX_WAL:-}" ] && envwal="MAX_WAL_SIZE=$MAX_WAL"
echo "== medir.sh $mode en IA, $(date '+%F %T') — copia: $(cat "$B/et.stamp") — memoria libre $(mem_free) %"

# vm_prepare: arranca la VM, copia la copia medida y compila storebench en ~/storebench.
vm_prepare() {
  [ "$(limactl list --format '{{.Status}}' $VM)" = Running ] || limactl start --tty=false $VM >/dev/null 2>&1
  limactl copy "$B/et.tgz" $VM:/tmp/et.tgz
  limactl shell $VM -- bash -lc 'rm -rf ~/et && mkdir -p ~/et && tar xzf /tmp/et.tgz -C ~/et 2>/dev/null && rm -f /tmp/et.tgz
    cd ~/et && GOWORK=off nice -n 10 go build -o ~/storebench ./test/storebench && go version && free -m && df -h ~ | tail -1'
}

# vm <comando>: corre un comando en la VM.
vm() { limactl shell $VM -- bash -lc "$1"; }

case $mode in
memoria)
  vm_prepare || exit 1
  echo "== 71: carga de $N claves con STORE_CACHE_SIZE=$CACHE_MB, Close, Open y lecturas (un proceso)"
  vm "rm -rf ~/sbmem; STORE_CACHE_SIZE=$CACHE_MB /usr/bin/time -v nice -n 10 ~/storebench -n $N -batch 20000 -dir ~/sbmem 2>&1 | grep -v -e '^	[A-Z].*: 0\$'"
  for r in 1 2; do
    echo "== 71: solo reabrir y leer (proceso nuevo, caché del SO vaciada), vuelta $r"
    vm "sync; echo 3 | sudo tee /proc/sys/vm/drop_caches >/dev/null; STORE_CACHE_SIZE=$CACHE_MB /usr/bin/time -v nice -n 10 ~/storebench -n $N -dir ~/sbmem -load=false 2>&1 | grep -e '^Open' -e '^lectura' -e '^memoria' -e 'Maximum resident' -e 'Elapsed'"
  done
  vm "du -sh ~/sbmem; rm -rf ~/sbmem; sudo fstrim -av; df -h ~ | tail -1"
  ;;
caida-mac)
  (cd "$B/src" && GOWORK=off nice -n 15 go build -o "$B/storebench" ./test/storebench) || exit 1
  for r in $(seq "$REPS"); do
    wait_memory 3600 || exit 1
    echo "== 72 macOS, vuelta $r: carga hasta $CRASH_MB MB de WAL sin Close (CHECKPOINT_TIMEOUT=$CKPT_TIMEOUT ${envwal:-MAX_WAL_SIZE por omisión})"
    rm -rf "$B/crash"
    env CHECKPOINT_TIMEOUT="$CKPT_TIMEOUT" $envwal nice -n 15 "$B/storebench" -crashwal "$CRASH_MB" -batch 20000 -dir "$B/crash"
    echo "   salió con $? (137 = kill -9)"
    env $envwal /usr/bin/time -l nice -n 15 "$B/storebench" -recover -dir "$B/crash" 2>&1 | grep -e '^Open' -e '^Verify' -e '^memoria' -e 'real' -e 'maximum resident'
  done
  rm -rf "$B/crash" "$B/storebench"
  ;;
caida-vm)
  vm_prepare || exit 1
  for r in $(seq "$REPS"); do
    echo "== 72 VM, vuelta $r: carga hasta $CRASH_MB MB de WAL sin Close (CHECKPOINT_TIMEOUT=$CKPT_TIMEOUT ${envwal:-MAX_WAL_SIZE por omisión})"
    vm "rm -rf ~/crash; env CHECKPOINT_TIMEOUT=$CKPT_TIMEOUT $envwal nice -n 10 ~/storebench -crashwal $CRASH_MB -batch 20000 -dir ~/crash; echo \"   salió con \$? (137 = kill -9)\""
    vm "sync; echo 3 | sudo tee /proc/sys/vm/drop_caches >/dev/null; env $envwal /usr/bin/time -v nice -n 10 ~/storebench -recover -dir ~/crash 2>&1 | grep -e '^Open' -e '^Verify' -e '^memoria' -e 'Maximum resident' -e 'Elapsed'"
  done
  vm "rm -rf ~/crash; sudo fstrim -av"
  ;;
*) echo "modo desconocido: $mode"; exit 2 ;;
esac
echo "== medir.sh $mode terminó $(date '+%F %T')"
