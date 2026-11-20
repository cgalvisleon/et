#!/bin/bash
# Cortes de energía reales con LazyFS (tareas 33 y 50 de F-23). Corre DENTRO de la VM etstore; desde la Mac:
#   ./test/linuxvm/vmrun.sh powercut-lazyfs.sh [ROUNDS=20] [SEED=1] [MAXCUT=6]
# Cada ronda monta LazyFS, lanza ~/powercut -mode load (registro en el disco normal de la VM) y, a un tiempo al azar,
# provoca el corte según el tipo de ronda (ronda mod 4):
#   0, 1  clear   kill -9 de la carga y lazyfs::clear-cache (se pierde todo lo que no tuvo fsync)
#   2     crash   lazyfs::crash antes del próximo fsync o write de un archivo del motor (LazyFS muere con su caché)
#   3     torn    lazyfs::torn-op sobre el segmento 1 del WAL: una escritura partida en 3, solo la 1.ª parte llega
# Luego desmonta, vuelve a montar y corre -mode verify. Un fallo deja su carpeta con salidas, registro y motor.
set -uo pipefail

ROUNDS=${ROUNDS:-20}
SEED=${SEED:-1}
MAXCUT=${MAXCUT:-6}
BIN=${POWERCUT_BIN:-$HOME/powercut}
LAZY=$HOME/lazyfs/lazyfs
OUT=$HOME/powercut-results/lazyfs-$(date +%Y%m%d-%H%M%S)
W=/tmp/pclfs
MNT=$W/mnt ROOTD=$W/root FIFO=$W/faults.fifo
export STORE_CACHE_SIZE=8 WAL_SEGMENT_SIZE=16 MAX_WAL_SIZE=48 CHECKPOINT_TIMEOUT=2 VACUUM_INTERVAL=1
mkdir -p "$OUT"
SUMMARY=$OUT/summary.txt

log() { echo "$*" | tee -a "$SUMMARY"; }

# rnd <semilla> <a> <b>: entero determinista en [a, b].
rnd() { awk -v s="$1" -v a="$2" -v b="$3" 'BEGIN{srand(s); print int(a + rand()*(b-a+1))}'; }

unmount() {
  # Tras un crash de LazyFS el montaje queda «Transport endpoint is not connected» y mountpoint dice que no.
  fusermount3 -u "$MNT" 2>/dev/null || fusermount3 -uz "$MNT" 2>/dev/null
  pkill -f "lazyfs $MNT" 2>/dev/null
  sleep 0.3
}

mount_lazyfs() {
  cat > $W/lazyfs.toml <<T
[faults]
fifo_path="$FIFO"
[cache]
apply_eviction=false
[cache.simple]
custom_size="512mb"
blocks_per_page=1
[filesystem]
log_all_operations=false
logfile="$W/lazyfs.log"
T
  (cd "$LAZY" && ./scripts/mount-lazyfs.sh -c $W/lazyfs.toml -m "$MNT" -r "$ROOTD" >>$W/mount.out 2>&1)
  for _ in $(seq 40); do mountpoint -q "$MNT" && [ -p "$FIFO" ] && return 0; sleep 0.25; done
  return 1
}

fifo() { timeout 10 bash -c "echo '$1' > '$FIFO'"; }

[ -x "$BIN" ] || { echo "falta $BIN (vmrun.sh lo compila)"; exit 2; }
log "== LazyFS: $ROUNDS rondas desde la semilla $SEED, corte entre 1 y $MAXCUT s — $(date)"
ok=0; fails=(); confirmed=0
for r in $(seq 1 "$ROUNDS"); do
  [ -n "${ONLY:-}" ] && [ "$r" != "$ONLY" ] && continue   # ONLY=<ronda> repite solo esa
  seed=$((SEED + r - 1))
  case $((r % 4)) in 2) kind=crash ;; 3) kind=torn ;; *) kind=clear ;; esac
  unmount; rm -rf "$W"; mkdir -p "$MNT" "$ROOTD"
  if ! mount_lazyfs; then log "ronda $r semilla $seed: no se pudo montar LazyFS"; fails+=("$seed:montaje"); continue; fi
  RLOG=$W/confirm.log   # fuera del montaje: disco normal de la VM
  nice -n 10 "$BIN" -mode load -dir "$MNT/db" -log "$RLOG" -seed "$seed" >$W/load.out 2>&1 &
  pid=$!
  for _ in $(seq 120); do grep -q LOAD-STARTED $W/load.out && break; sleep 0.25; done
  cut_ms=$(rnd "$seed" 1000 $((MAXCUT * 1000)))
  detail=""
  case $kind in
    clear)
      sleep "$(awk -v m="$cut_ms" 'BEGIN{print m/1000}')"
      kill -9 $pid 2>/dev/null; wait $pid 2>/dev/null
      fifo "lazyfs::clear-cache"; sleep 1
      detail="kill -9 a los ${cut_ms} ms + clear-cache" ;;
    crash)
      sleep "$(awk -v m="$cut_ms" 'BEGIN{print m/1000}')"
      targets=(fsync:wal write:wal fsync:data write:data fsync:control rename:catalog)
      t=${targets[$(rnd "$seed" 0 5)]}
      fifo "lazyfs::crash::timing=before::op=${t%%:*}::from_rgx=${t#*:}"
      for _ in $(seq 40); do pgrep -f "lazyfs $MNT" >/dev/null || break; sleep 0.25; done
      detail="crash antes de ${t%%:*} en ${t#*:} pedido a los ${cut_ms} ms"
      if pgrep -f "lazyfs $MNT" >/dev/null; then
        kill -9 $pid 2>/dev/null; wait $pid 2>/dev/null
        fifo "lazyfs::clear-cache"; sleep 1
        detail="$detail, no llegó en 10 s: kill -9 + clear-cache"
      else
        kill -9 $pid 2>/dev/null; wait $pid 2>/dev/null
        detail="$detail (LazyFS murió)"
      fi ;;
    torn)
      occ=$(rnd "$seed" 50 3000)
      fifo "lazyfs::torn-op::file=$ROOTD/db/wal/0000000001.seg::persist=1::parts=3::occurrence=$occ"
      for _ in $(seq $((MAXCUT * 4))); do pgrep -f "lazyfs $MNT" >/dev/null || break; sleep 0.25; done
      if pgrep -f "lazyfs $MNT" >/dev/null; then
        kill -9 $pid 2>/dev/null; wait $pid 2>/dev/null
        fifo "lazyfs::clear-cache"; sleep 1
        detail="torn-op en la escritura $occ del WAL no llegó en ${MAXCUT} s: kill -9 + clear-cache"
      else
        kill -9 $pid 2>/dev/null; wait $pid 2>/dev/null
        detail="torn-op en la escritura $occ del segmento 1 del WAL (LazyFS murió)"
      fi ;;
  esac
  if grep -q -e '^panic:' -e 'fatal error:' $W/load.out; then
    log "ronda $r semilla $seed: PÁNICO en la carga ($detail)"; fails+=("$seed:panico")
  fi
  unmount
  if ! mount_lazyfs; then log "ronda $r semilla $seed: no se pudo volver a montar"; fails+=("$seed:remontaje"); continue; fi
  nice -n 10 timeout 300 "$BIN" -mode verify -dir "$MNT/db" -log "$RLOG" >$W/verify.out 2>&1
  rc=$?
  line=$(grep -m1 -e VERIFY- -e POWERCUT-FATAL $W/verify.out || tail -1 $W/verify.out)
  if [ $rc = 0 ]; then
    ok=$((ok + 1))
    c=$(sed -n 's/.*confirmadas=\([0-9]*\).*/\1/p' <<<"$line"); confirmed=$((confirmed + ${c:-0}))
    log "ronda $r semilla $seed [$kind] $detail: $line"
  else
    log "ronda $r semilla $seed [$kind] $detail: FALLO rc=$rc"; cat $W/verify.out | tee -a "$SUMMARY"
    fails+=("$seed:$kind")
    unmount; mkdir -p "$OUT/ronda-$r"; cp -a $W/load.out $W/verify.out "$RLOG" "$OUT/ronda-$r/"; cp -a "$ROOTD" "$OUT/ronda-$r/root"
  fi
  unmount
done
rm -rf "$W"
log "== LazyFS: $ok/$ROUNDS rondas bien, $confirmed confirmaciones verificadas, fallos: ${fails[*]:-ninguno}"
log "== resultados en $OUT"
[ ${#fails[@]} = 0 ]
