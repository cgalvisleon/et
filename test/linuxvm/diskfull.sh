#!/bin/bash
# Disco lleno real (tarea 33 de F-23, «Disco lleno o lento» en store/doc.go). Corre DENTRO de la VM etstore (usa sudo
# de la VM); desde la Mac:
#   ./test/linuxvm/vmrun.sh diskfull.sh [SEED=201]
# Por cada caso monta un loop ext4 chico (128 MB, sin bloques reservados), pone un relleno de 32 MB (más que un
# segmento del WAL, para que liberarlo alcance) y corre
# ~/powercut -mode diskfull: llena el disco hasta ErrDiskFull, comprueba las lecturas, borra el relleno y exige que
# las escrituras vuelvan solas (error limpio) o que Close + Open recupere todo (solo lectura); al final Count y
# Verify limpios, también tras reabrir.
#   caso 1  WAL_SEGMENT_SIZE=16 MB: lo más probable es que falte lugar al preasignar un segmento del WAL
#   caso 2  WAL_SEGMENT_SIZE=4 MB: los datos crecen más que el WAL; falta lugar al extender un archivo de datos
#   caso 3  como el 2 con SYNC_ON_WRITE=false (todas las confirmaciones diferidas)
#   caso 4  como el 1 con -closefull: Close y Open con el disco todavía lleno (pueden dar ErrDiskFull, nunca colgarse
#           ni perder nada) y recién entonces liberar
set -uo pipefail

SEED=${SEED:-201}
BIN=${POWERCUT_BIN:-$HOME/powercut}
OUT=$HOME/powercut-results/diskfull-$(date +%Y%m%d-%H%M%S)
W=$HOME/pcfull
MNT=/mnt/pcfull
mkdir -p "$OUT" $W
SUMMARY=$OUT/summary.txt
log() { echo "$*" | tee -a "$SUMMARY"; }

cleanup() {
  mountpoint -q $MNT && sudo umount $MNT
  [ -n "${LOOP:-}" ] && sudo losetup -d "$LOOP" 2>/dev/null
  LOOP=""
  rm -f $W/full.img
}
trap cleanup EXIT
[ -x "$BIN" ] || { echo "falta $BIN (vmrun.sh lo compila)"; exit 2; }

log "== disco lleno: 4 casos sobre un ext4 de 128 MB — $(date)"
ok=0; fails=()
cases=("WAL_SEGMENT_SIZE=16 MAX_WAL_SIZE=32 SYNC_ON_WRITE=true"
       "WAL_SEGMENT_SIZE=4 MAX_WAL_SIZE=16 SYNC_ON_WRITE=true"
       "WAL_SEGMENT_SIZE=4 MAX_WAL_SIZE=16 SYNC_ON_WRITE=false"
       "WAL_SEGMENT_SIZE=16 MAX_WAL_SIZE=32 SYNC_ON_WRITE=true -closefull")
for i in "${!cases[@]}"; do
  n=$((i + 1)); seed=$((SEED + i)); cfg=${cases[$i]%% -*}; args=""
  [ "$cfg" = "${cases[$i]}" ] || args="-${cases[$i]#* -}"
  cleanup
  truncate -s 128M $W/full.img
  LOOP=$(sudo losetup -f --show $W/full.img)
  sudo mkfs.ext4 -q -m 0 "$LOOP"
  sudo mkdir -p $MNT && sudo mount "$LOOP" $MNT && sudo chown "$(id -u):$(id -g)" $MNT
  dd if=/dev/urandom of=$MNT/relleno bs=1M count=32 conv=fsync status=none
  rm -f $W/confirm.log
  env STORE_CACHE_SIZE=8 CHECKPOINT_TIMEOUT=2 VACUUM_INTERVAL=1 $cfg \
    nice -n 10 timeout 1200 "$BIN" -mode diskfull -dir $MNT/db -log $W/confirm.log -seed "$seed" -filler $MNT/relleno $args \
    >$W/diskfull.out 2>&1
  rc=$?
  df=$(df -k --output=used,avail $MNT | tail -1)
  log "caso $n semilla $seed ($cfg $args) rc=$rc, df usado/libre KB al final: $df"
  grep -e '^DISKFULL' -e '^POWERCUT' -e '^panic' $W/diskfull.out | sed 's/^/  /' | tee -a "$SUMMARY"
  if [ $rc = 0 ] && grep -q DISKFULL-OK $W/diskfull.out; then
    ok=$((ok + 1))
  else
    fails+=("$seed:caso$n"); mkdir -p "$OUT/caso-$n"; cp -a $W/diskfull.out $W/confirm.log "$OUT/caso-$n/"
  fi
done
log "== disco lleno: $ok/${#cases[@]} casos bien, fallos: ${fails[*]:-ninguno}"
log "== resultados en $OUT"
[ ${#fails[@]} = 0 ]
