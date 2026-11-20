#!/bin/bash
# Escrituras perdidas y errores de E/S con dm-flakey (tareas 33 y 50 de F-23). Corre DENTRO de la VM etstore (usa
# sudo de la VM); desde la Mac:
#   ./test/linuxvm/vmrun.sh powercut-flakey.sh [DROP_ROUNDS=10] [ERROR_ROUNDS=5] [SEED=101] [MAXCUT=6]
# Un loop de 256 MB con ext4 sobre un dispositivo flakey (/dev/mapper/pcflakey):
#   drop_writes   carga ~/powercut -mode load; a un tiempo al azar: SIGSTOP, tabla a drop_writes (el disco descarta
#                 en silencio todo lo que no le llegó: un corte en ese instante), kill -9, desmontar (lo sucio de
#                 ext4 se pierde), tabla normal, montar y -mode verify.
#   error_writes  carga -mode ioerror; a un tiempo al azar la tabla pasa a error_writes (write y fsync dan EIO): el
#                 motor debe quedar inutilizable sin pánico ni cuelgue. Luego tabla normal, desmontar, e2fsck -fy,
#                 montar y -mode verify.
# El registro de confirmaciones va al disco normal de la VM, nunca al dispositivo flakey.
set -uo pipefail

DROP_ROUNDS=${DROP_ROUNDS:-10}
ERROR_ROUNDS=${ERROR_ROUNDS:-5}
SEED=${SEED:-101}
MAXCUT=${MAXCUT:-6}
BIN=${POWERCUT_BIN:-$HOME/powercut}
OUT=$HOME/powercut-results/flakey-$(date +%Y%m%d-%H%M%S)
W=$HOME/pcflakey   # imagen, registro y salidas: disco normal de la VM
MNT=/mnt/pcflakey
DEV=pcflakey
export STORE_CACHE_SIZE=8 WAL_SEGMENT_SIZE=16 MAX_WAL_SIZE=48 CHECKPOINT_TIMEOUT=2 VACUUM_INTERVAL=1
mkdir -p "$OUT"
SUMMARY=$OUT/summary.txt

log() { echo "$*" | tee -a "$SUMMARY"; }
rnd() { awk -v s="$1" -v a="$2" -v b="$3" 'BEGIN{srand(s); print int(a + rand()*(b-a+1))}'; }
secs() { awk -v m="$1" 'BEGIN{print m/1000}'; }

# table normal|drop_writes|error_writes: cambia la tabla del flakey sin vaciar colas ni congelar ext4.
table() {
  local t="0 $SECT flakey $LOOP 0 60 0"
  [ "$1" = normal ] || t="0 $SECT flakey $LOOP 0 0 60 1 $1"
  sudo dmsetup suspend --noflush --nolockfs $DEV && sudo dmsetup load $DEV --table "$t" && sudo dmsetup resume $DEV
}

cleanup() {
  mountpoint -q $MNT && sudo umount $MNT
  sudo dmsetup remove $DEV 2>/dev/null
  [ -n "${LOOP:-}" ] && sudo losetup -d "$LOOP" 2>/dev/null
  rm -f $W/flakey.img
}

[ -x "$BIN" ] || { echo "falta $BIN (vmrun.sh lo compila)"; exit 2; }
sudo modprobe dm-flakey 2>/dev/null
mountpoint -q $MNT && sudo umount $MNT
sudo dmsetup remove $DEV 2>/dev/null
mkdir -p $W && rm -f $W/flakey.img && truncate -s 256M $W/flakey.img
LOOP=$(sudo losetup -f --show $W/flakey.img)
SECT=$(sudo blockdev --getsz "$LOOP")
trap cleanup EXIT
sudo dmsetup create $DEV --table "0 $SECT flakey $LOOP 0 60 0"
sudo mkfs.ext4 -q /dev/mapper/$DEV
sudo mkdir -p $MNT && sudo mount /dev/mapper/$DEV $MNT && sudo chown "$(id -u):$(id -g)" $MNT

log "== dm-flakey: $DROP_ROUNDS rondas drop_writes + $ERROR_ROUNDS error_writes desde la semilla $SEED — $(date)"
ok=0; fails=(); confirmed=0; total=$((DROP_ROUNDS + ERROR_ROUNDS))

verify_round() { # verify_round <ronda> <semilla> <tipo> <detalle>
  nice -n 10 timeout 300 "$BIN" -mode verify -dir $MNT/db -log $W/confirm.log >$W/verify.out 2>&1
  local rc=$? line
  line=$(grep -m1 -e VERIFY- -e POWERCUT-FATAL $W/verify.out || tail -1 $W/verify.out)
  if [ $rc = 0 ] && [ "${5:-ok}" = ok ]; then
    ok=$((ok + 1))
    c=$(sed -n 's/.*confirmadas=\([0-9]*\).*/\1/p' <<<"$line"); confirmed=$((confirmed + ${c:-0}))
    log "ronda $1 semilla $2 [$3] $4: $line"
  else
    log "ronda $1 semilla $2 [$3] $4: FALLO rc=$rc"; tee -a "$SUMMARY" <$W/verify.out
    fails+=("$2:$3"); mkdir -p "$OUT/ronda-$1"
    cp -a $W/*.out $W/confirm.log "$OUT/ronda-$1/" 2>/dev/null; sudo cp -a $MNT/db "$OUT/ronda-$1/db" 2>/dev/null
    sudo chown -R "$(id -u):$(id -g)" "$OUT/ronda-$1"
  fi
}

for r in $(seq 1 $total); do
  [ -n "${ONLY:-}" ] && [ "$r" != "$ONLY" ] && continue   # ONLY=<ronda> repite solo esa
  seed=$((SEED + r - 1))
  kind=drop_writes; [ $r -gt $DROP_ROUNDS ] && kind=error_writes
  rm -rf $MNT/db $W/*.out $W/confirm.log
  cut_ms=$(rnd "$seed" 1000 $((MAXCUT * 1000)))
  status=ok
  if [ $kind = drop_writes ]; then
    nice -n 10 "$BIN" -mode load -dir $MNT/db -log $W/confirm.log -seed "$seed" >$W/load.out 2>&1 &
    pid=$!
    for _ in $(seq 120); do grep -q LOAD-STARTED $W/load.out && break; sleep 0.25; done
    sleep "$(secs "$cut_ms")"
    kill -STOP $pid
    table drop_writes
    kill -9 $pid; wait $pid 2>/dev/null
    sudo umount $MNT
    table normal
    sudo mount /dev/mapper/$DEV $MNT
    detail="drop_writes a los ${cut_ms} ms"
  else
    nice -n 10 "$BIN" -mode ioerror -wait 60s -dir $MNT/db -log $W/confirm.log -seed "$seed" >$W/load.out 2>&1 &
    pid=$!
    for _ in $(seq 120); do grep -q LOAD-STARTED $W/load.out && break; sleep 0.25; done
    sleep "$(secs "$cut_ms")"
    table error_writes
    hung=0
    for _ in $(seq 480); do kill -0 $pid 2>/dev/null || break; sleep 0.25; done
    if kill -0 $pid 2>/dev/null; then hung=1; kill -9 $pid; fi
    wait $pid; lrc=$?
    table normal
    sudo umount $MNT || sudo umount -l $MNT
    sudo e2fsck -fy /dev/mapper/$DEV >$W/e2fsck.out 2>&1; frc=$?
    sudo mount /dev/mapper/$DEV $MNT
    io=$(grep -e '^IOERROR' $W/load.out | tr '\n' ' ' | cut -c1-900)
    detail="error_writes a los ${cut_ms} ms (ioerror rc=$lrc, e2fsck rc=$frc): $io"
    if [ $hung = 1 ]; then detail="$detail COLGADO 120 s"; status=fail; fi
    [ $lrc = 0 ] || status=fail
  fi
  if grep -q -e '^panic:' -e 'fatal error:' $W/load.out; then detail="$detail PÁNICO"; status=fail; fi
  verify_round "$r" "$seed" "$kind" "$detail" "$status"
done
log "== dm-flakey: $ok/$total rondas bien, $confirmed confirmaciones verificadas, fallos: ${fails[*]:-ninguno}"
log "== resultados en $OUT"
[ ${#fails[@]} = 0 ]
