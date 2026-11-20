#!/bin/bash
# Corrida larga de los fuzzers de store/ (tarea 73 de F-23), en macOS de IA, bajo el candado de la CI.
#   desde la Mac:  ./test/ci/fuzzlong.sh            (compila, sube la copia, instala test/ci en IA y la lanza con nohup)
#   en IA:         ~/etstore-ci/bin/fuzzlong.sh
# Cada func Fuzz de store/ (10 hoy) corre uno a la vez FUZZTIME (48m: 10 × 48 min = 8 h), -parallel FUZZ_PARALLEL (4),
# nice -n 15, -fuzzminimizetime FUZZ_MINIMIZE (10s; page 1x), con las mismas funciones del nocturno (lib.sh: fuzz_all).
# Resultado: ~/etstore-ci/fuzzlong-<fecha>/ (summary.txt, un .log por fuzzer, fuzz-found/), copia del resumen en
# ~/etstore-ci/last-fuzzlong.txt; casos nuevos también en ~/etstore-ci/fuzz-found/. Se consulta con ./test/ci/status.sh.
# Si la corrida prevista pasa por las 03:00, deja ~/etstore-ci/skip-nightly-<AAAAMMDD> y el nocturno salta esa noche.
set -uo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
HOST=${ETSTORE_HOST:-ia}
FUZZTIME=${FUZZTIME:-48m}

# ---- en la Mac: compilar, sincronizar, instalar test/ci y lanzar en IA ----
if git -C "$HERE" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  ROOT=$(cd "$HERE/../.." && pwd)
  cd "$ROOT" || exit 2
  for i in $(seq 20); do
    go build ./store/... && go vet ./store/... && break
    [ "$i" = 20 ] && { echo "store/ no compila tras 10 min: no se lanza"; exit 1; }
    echo "   (store/ no compila todavía; espero 30 s)"; sleep 30
  done
  ETSTORE_SYNC_ONLY=1 ETSTORE_HOST=$HOST bash test/linuxvm/run.sh || exit 1
  ssh -o BatchMode=yes "$HOST" 'set -e; c=~/etstore-ci; mkdir -p $c; rm -rf $c/bin.new; mkdir $c/bin.new
    tar xzf ~/etstore/et.tgz -C $c/bin.new --strip-components 2 test/ci
    rm -rf $c/bin; mv $c/bin.new $c/bin' || exit 1
  ssh -o BatchMode=yes "$HOST" "FUZZTIME=$FUZZTIME FUZZ_PARALLEL=${FUZZ_PARALLEL:-4} MIN_FREE=${MIN_FREE:-30} \
nohup bash ~/etstore-ci/bin/fuzzlong.sh >~/etstore-ci/fuzzlong.log 2>&1 </dev/null &" || exit 1
  echo "fuzzlong lanzado en IA (FUZZTIME=$FUZZTIME por fuzzer); progreso: ssh ia tail -f ~/etstore-ci/fuzzlong.log;"
  echo "resultado: ./test/ci/status.sh fuzzlong"
  exit 0
fi

# ---- en IA ----
# shellcheck source=lib.sh
source "$HERE/lib.sh"
FUZZ_PARALLEL=${FUZZ_PARALLEL:-4}
FUZZ_MINIMIZE=${FUZZ_MINIMIZE:-10s}

take_lock 10800 || exit 1
take_snapshot || exit 1
OUT=$CI/fuzzlong-$(date +%Y%m%d-%H%M%S)
mkdir -p "$OUT"
SUMMARY=$OUT/summary.txt
start=$(date +%s)
count=$(grep -rh '^func Fuzz' --include='*_test.go' "$SRC/store" | wc -l | tr -d ' ')
minutes=$(( count * ${FUZZTIME%m} + count * 2 ))   # FUZZTIME en minutos (…m) + ~2 min por fuzzer de compilar
end=$(( start + minutes * 60 ))
say "== fuzzlong.sh en IA, $(date '+%F %T') — copia: $(cat "$CI/src.stamp") — $count fuzzers × $FUZZTIME, -parallel $FUZZ_PARALLEL — fin previsto $(date -r $end '+%F %T')"

# Si entre el inicio y el fin previsto hay unas 03:00, salta el nocturno de esa fecha (no corre en paralelo ni espera).
for day in $(date -r "$start" +%Y%m%d) $(date -r "$end" +%Y%m%d); do
  t3=$(date -j -f '%Y%m%d %H%M%S' "$day 030000" +%s)
  if [ "$t3" -ge "$start" ] && [ "$t3" -le "$((end + 1800))" ]; then
    echo "fuzzlong $OUT (fin previsto $(date -r $end '+%F %T'))" > "$CI/skip-nightly-$day"
    say "   el nocturno del $day queda saltado (marca $CI/skip-nightly-$day)"
  fi
done

fuzz_all

if [ "$failures" = 0 ]; then result=OK; else result="FALLA ($failures fuzzers)"; fi
say "== resultado: $result en $(( ($(date +%s) - start) / 60 )) min — $OUT"
cp "$SUMMARY" "$CI/last-fuzzlong.txt"
[ "$failures" = 0 ]
