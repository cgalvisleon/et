#!/bin/bash
# Prueba de resistencia de et/store (tarea 51 de F-23): test/soak en macOS de IA, bajo el candado de la CI.
#   desde la Mac:  ./test/ci/soak.sh                      (compila, sube la copia, instala test/ci en IA y la lanza con nohup)
#                  SOAK_DURATION=10m SOAK_ARGS='-check 1m -reopen 3m -warmup 2m' ./test/ci/soak.sh   (una corta)
#   en IA:         ~/etstore-ci/bin/soak.sh
# Variables: SOAK_DURATION (72h), SOAK_RATE (2000 op/s), SOAK_CACHE (64 MB), SOAK_KEYS (50000 por store),
# SOAK_ARGS (más flags de test/soak), SOAK_MAX_RSS_MB (2048: el vigilante lo detiene si pasa), MIN_FREE (30 %).
# Resultado: ~/etstore-ci/soak-<fecha>/ (summary.md, soak.log, metrics.tsv, checks.tsv, vigilante.log), copia del
# resumen en ~/etstore-ci/last-soak.txt. El directorio del motor (db/) se borra si pasa y queda si falla.
# Salta los nocturnos (marca ~/etstore-ci/skip-nightly-<AAAAMMDD>) de cada 03:00 entre el inicio y el fin previsto.
# Se consulta con ./test/ci/status.sh soak. Detenerla antes: ssh ia 'pkill -TERM -f etstore-ci/soak-.*/soak' (escribe
# el resumen como INTERRUMPIDA).
set -uo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
HOST=${ETSTORE_HOST:-ia}
SOAK_DURATION=${SOAK_DURATION:-72h}

# ---- en la Mac: compilar, sincronizar, instalar test/ci y lanzar en IA ----
if git -C "$HERE" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  ROOT=$(cd "$HERE/../.." && pwd)
  cd "$ROOT" || exit 2
  for i in $(seq 20); do
    go build ./store/... ./test/soak && go vet ./store/... ./test/soak && break
    [ "$i" = 20 ] && { echo "store/ o test/soak no compilan tras 10 min: no se lanza"; exit 1; }
    echo "   (store/ no compila todavía; espero 30 s)"; sleep 30
  done
  ETSTORE_SYNC_ONLY=1 ETSTORE_HOST=$HOST bash test/linuxvm/run.sh || exit 1
  ssh -o BatchMode=yes "$HOST" 'set -e; c=~/etstore-ci; mkdir -p $c; rm -rf $c/bin.new; mkdir $c/bin.new
    tar xzf ~/etstore/et.tgz -C $c/bin.new --strip-components 2 test/ci
    rm -rf $c/bin; mv $c/bin.new $c/bin' || exit 1
  ssh -o BatchMode=yes "$HOST" "SOAK_DURATION=$SOAK_DURATION SOAK_RATE=${SOAK_RATE:-2000} SOAK_CACHE=${SOAK_CACHE:-64} \
SOAK_KEYS=${SOAK_KEYS:-50000} SOAK_ARGS='${SOAK_ARGS:-}' SOAK_MAX_RSS_MB=${SOAK_MAX_RSS_MB:-2048} MIN_FREE=${MIN_FREE:-30} \
nohup bash ~/etstore-ci/bin/soak.sh >~/etstore-ci/soak.log 2>&1 </dev/null &" || exit 1
  echo "soak lanzado en IA ($SOAK_DURATION); progreso: ./test/ci/status.sh soak (o ssh ia tail -f ~/etstore-ci/soak.log)"
  exit 0
fi

# ---- en IA ----
# shellcheck source=lib.sh
source "$HERE/lib.sh"
SOAK_RATE=${SOAK_RATE:-2000}
SOAK_CACHE=${SOAK_CACHE:-64}
SOAK_KEYS=${SOAK_KEYS:-50000}
SOAK_ARGS=${SOAK_ARGS:-}
SOAK_MAX_RSS_MB=${SOAK_MAX_RSS_MB:-2048}

take_lock 10800 || exit 1
take_snapshot || exit 1
OUT=$CI/soak-$(date +%Y%m%d-%H%M%S)
mkdir -p "$OUT"
SUMMARY=$OUT/run.txt
prune 'soak-2*' 10

# duración en segundos (…h, …m o …s, como el flag de Go más simple)
secs() { case $1 in *h) echo $(( ${1%h} * 3600 ));; *m) echo $(( ${1%m} * 60 ));; *s) echo "${1%s}";; *) echo "$1";; esac; }
start=$(date +%s)
end=$(( start + $(secs "$SOAK_DURATION") + 600 ))   # + ~10 min de compilar, precargar y la verificación final
say "== soak.sh en IA, $(date '+%F %T') — copia: $(cat "$CI/src.stamp") — $SOAK_DURATION, $SOAK_RATE op/s, cache $SOAK_CACHE MB, $SOAK_KEYS claves/store $SOAK_ARGS — fin previsto $(date -r $end '+%F %T')"

# Salta cada nocturno (03:00) entre el inicio y el fin previsto (+30 min).
day=$start
while [ "$day" -le "$((end + 86400))" ]; do
  d=$(date -r "$day" +%Y%m%d)
  t3=$(date -j -f '%Y%m%d %H%M%S' "$d 030000" +%s)
  if [ "$t3" -ge "$start" ] && [ "$t3" -le "$((end + 1800))" ]; then
    echo "soak $OUT (fin previsto $(date -r $end '+%F %T'))" > "$CI/skip-nightly-$d"
    say "   el nocturno del $d queda saltado (marca $CI/skip-nightly-$d)"
  fi
  day=$((day + 86400))
done

if ! (cd "$SRC" && GOWORK=off go build -o "$OUT/soak" ./test/soak) >"$OUT/build.log" 2>&1; then
  say "FALLA    compilar test/soak (ver build.log)"; cp "$SUMMARY" "$CI/last-soak.txt"; exit 1
fi
if ! wait_memory 1800; then
  say "SALTADO  memoria libre $(mem_free) % < $MIN_FREE %"; cp "$SUMMARY" "$CI/last-soak.txt"; exit 1
fi

# shellcheck disable=SC2086
nice -n 15 "$OUT/soak" -dir "$OUT/db" -out "$OUT" -duration "$SOAK_DURATION" -rate "$SOAK_RATE" -cache "$SOAK_CACHE" \
  -keys "$SOAK_KEYS" $SOAK_ARGS >"$OUT/stdout.log" 2>&1 &
pid=$!
echo "$pid" > "$OUT/pid"

# Vigilante: cada minuto mira el RSS del soak (lo detiene con SIGTERM si pasa SOAK_MAX_RSS_MB) y la memoria libre de
# IA (con menos de 10 % lo congela con SIGSTOP hasta volver a MIN_FREE: Ollama y el corrector van primero).
(
  stopped=0
  while kill -0 "$pid" 2>/dev/null; do
    rss=$(ps -o rss= -p "$pid" 2>/dev/null | tr -d ' '); rss=$(( ${rss:-0} / 1024 ))
    free=$(mem_free); free=${free:-100}
    if [ "$rss" -gt "$SOAK_MAX_RSS_MB" ]; then
      echo "$(date '+%F %T') RSS $rss MB > $SOAK_MAX_RSS_MB MB: SIGTERM" >>"$OUT/vigilante.log"
      [ $stopped = 1 ] && kill -CONT "$pid"
      kill -TERM "$pid"; echo "rss $rss MB" > "$OUT/killed"; break
    fi
    if [ $stopped = 0 ] && [ "$free" -lt 10 ]; then
      echo "$(date '+%F %T') memoria libre $free %: SIGSTOP" >>"$OUT/vigilante.log"; kill -STOP "$pid"; stopped=1
    elif [ $stopped = 1 ] && [ "$free" -ge "$MIN_FREE" ]; then
      echo "$(date '+%F %T') memoria libre $free %: SIGCONT" >>"$OUT/vigilante.log"; kill -CONT "$pid"; stopped=0
    fi
    [ $(( $(date +%s) % 600 )) -lt 60 ] && echo "$(date '+%F %T') rss $rss MB, memoria libre $free %" >>"$OUT/vigilante.log"
    sleep 60
  done
) &
wait "$pid"; rc=$?

if [ $rc = 0 ]; then
  rm -rf "$OUT/db"
  say "== resultado: OK en $(( ($(date +%s) - start) / 60 )) min — $OUT"
else
  [ -f "$OUT/killed" ] && say "   el vigilante lo detuvo: $(cat "$OUT/killed")"
  say "== resultado: FALLA (rc=$rc) en $(( ($(date +%s) - start) / 60 )) min — $OUT (el motor queda en db/)"
fi
{ cat "$SUMMARY"; echo; cat "$OUT/summary.md" 2>/dev/null; } > "$CI/last-soak.txt"
rm -f "$OUT/pid"
for f in "$CI"/skip-nightly-*; do   # quita las marcas de esta corrida para noches que todavía no llegaron
  grep -q "soak $OUT" "$f" 2>/dev/null || continue
  t3=$(date -j -f '%Y%m%d %H%M%S' "${f##*-} 030000" +%s)
  [ "$t3" -gt "$(date +%s)" ] && rm -f "$f"
done
[ $rc = 0 ]
