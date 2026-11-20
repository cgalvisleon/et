#!/bin/bash
# Corrida nocturna del motor store (tarea 48 de F-23), en IA. La lanza el LaunchAgent
# com.cgalvisleon.etstore-nightly a las 03:00 (install-nightly.sh) o se corre a mano:
#   en IA:            ~/etstore-ci/bin/nightly.sh
#   desde la Mac:     ./test/ci/nightly.sh          (la lanza en IA en segundo plano; ver con status.sh)
# Prueba la ÚLTIMA COPIA SUBIDA a IA (~/etstore/et.tgz, la deja ci.sh): IA no tiene acceso al repo.
#   1. benchmarks en macOS: storebench -n 1000000 -rate 5s (3 veces) y go test -bench (-count 3); mediana de 3 contra
#      ~/etstore-ci/baseline.tsv: FALLA si algo empeora más de THRESHOLD % (10). Sin línea base, la crea.
#   2. fuzzers de store/ en macOS, uno a la vez, FUZZTIME cada uno (5m), -parallel FUZZ_PARALLEL (4),
#      -fuzzminimizetime FUZZ_MINIMIZE (10s; page 1x); un caso
#      encontrado se guarda en ~/etstore-ci/fuzz-found/ y en la carpeta de la noche.
#   3. cortes completos en la VM (rondas por omisión): powercut-lazyfs.sh, powercut-flakey.sh, diskfull.sh; VM detenida.
# Todo queda en ~/etstore-ci/nightly-<fecha>/ (se conservan las últimas 14); resumen en ~/etstore-ci/last-nightly.txt.
set -uo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)

# ---- en la Mac: lanzar en IA en segundo plano ----
if git -C "$HERE" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  ssh -o BatchMode=yes "${ETSTORE_HOST:-ia}" "FUZZTIME=${FUZZTIME:-5m} THRESHOLD=${THRESHOLD:-10} \
nohup bash ~/etstore-ci/bin/nightly.sh >~/etstore-ci/nightly-manual.log 2>&1 </dev/null &" || exit 1
  echo "nightly lanzado en IA; progreso: ssh ia tail -f ~/etstore-ci/nightly-manual.log; resultado: ./test/ci/status.sh"
  exit 0
fi

# ---- en IA ----
# shellcheck source=lib.sh
source "$HERE/lib.sh"
FUZZTIME=${FUZZTIME:-5m}
FUZZ_PARALLEL=${FUZZ_PARALLEL:-4}
FUZZ_MINIMIZE=${FUZZ_MINIMIZE:-10s}
THRESHOLD=${THRESHOLD:-10}
BENCH_RUNS=${BENCH_RUNS:-3}
BASELINE=$CI/baseline.tsv

take_lock 10800 || exit 1
take_snapshot || exit 1
OUT=$CI/nightly-$(date +%Y%m%d-%H%M%S)
mkdir -p "$OUT"
SUMMARY=$OUT/summary.txt
start=$(date +%s)
say "== nightly.sh en IA, $(date '+%F %T') — copia: $(cat "$CI/src.stamp") — fuzz $FUZZTIME c/u, umbral $THRESHOLD %"

# ---------- 1. benchmarks ----------

# storebench_runs: compila storebench y lo corre BENCH_RUNS veces a escala chica.
storebench_runs() {
  mac_go build -o "$OUT/storebench" ./test/storebench || return 1
  for i in $(seq "$BENCH_RUNS"); do
    echo "## corrida $i"
    nice -n 15 "$OUT/storebench" -n 1000000 -batch 20000 -rate 5s || return 1
  done
  rm -f "$OUT/storebench"
}

# metrics <storebench.log> <gobench.log>: imprime «métrica valor mejor» (mediana de las corridas; mejor = higher|lower).
metrics() {
  awk '
    # ns: duración de Go (1m2.5s, 36.5ms, 5.5µs, 812ns) en nanosegundos.
    function ns(d,   t, n, u) {
      t = 0
      while (match(d, /^[0-9.]+/)) {
        n = substr(d, 1, RLENGTH) + 0; d = substr(d, RLENGTH + 1)
        if (d ~ /^ns/) { u = 1; d = substr(d, 3) }
        else if (d ~ /^us/) { u = 1e3; d = substr(d, 3) }
        else if (index(d, "µs") == 1) { u = 1e3; d = substr(d, length("µs") + 1) }
        else if (d ~ /^ms/) { u = 1e6; d = substr(d, 3) }
        else if (d ~ /^s/) { u = 1e9; d = substr(d, 2) }
        else if (d ~ /^m/) { u = 6e10; d = substr(d, 2) }
        else if (d ~ /^h/) { u = 3.6e12; d = substr(d, 2) }
        else return -1
        t += n * u
      }
      return t
    }
    function add(name, v, better) { vals[name] = vals[name] " " v; dir[name] = better }
    FILENAME == ARGV[1] && /^carga:/ { gsub(/[(]/, "", $6); add("sb.carga_claves_s", $6, "higher") }
    FILENAME == ARGV[1] && /^Put / { add("sb.put_" $2 "_" $4, $6, "higher") }
    FILENAME == ARGV[1] && /^Open tras Close:/ { add("sb.open_us", ns($4) / 1e3, "lower") }
    FILENAME == ARGV[1] && /^lectura al azar/ { add("sb.fria_p50_us", ns($9) / 1e3, "lower"); add("sb.fria_p99_us", ns($11) / 1e3, "lower") }
    FILENAME == ARGV[1] && /^lectura en caché/ { add("sb.cache_p50_us", ns($9) / 1e3, "lower"); add("sb.cache_p99_us", ns($11) / 1e3, "lower") }
    FILENAME == ARGV[2] && /^Benchmark/ {
      name = $1; sub(/-[0-9]+$/, "", name)
      for (i = 2; i < NF; i++) {
        if ($(i + 1) == "ns/op") add("go." name ".ns_op", $i, "lower")
        if ($(i + 1) == "allocs/op") add("go." name ".allocs_op", $i, "lower")
      }
    }
    END {
      for (m in vals) {
        n = split(substr(vals[m], 2), a, " ")
        for (i = 1; i <= n; i++) for (j = i + 1; j <= n; j++) if (a[j] + 0 < a[i] + 0) { x = a[i]; a[i] = a[j]; a[j] = x }
        printf "%s\t%s\t%s\n", m, a[int((n + 1) / 2)], dir[m]
      }
    }' "$1" "$2" | sort
}

# compare <actual.tsv>: compara con la línea base; FALLA si una métrica empeora más de THRESHOLD %.
compare() {
  if [ ! -s "$BASELINE" ]; then
    cp "$1" "$BASELINE"
    echo "no había línea base: queda esta corrida en $BASELINE"
    return 0
  fi
  awk -F'\t' -v thr="$THRESHOLD" '
    NR == FNR { base[$1] = $2; next }
    {
      seen[$1] = 1
      if (!($1 in base)) { printf "NUEVA  %-45s %14.1f\n", $1, $2; next }
      b = base[$1] + 0; c = $2 + 0
      worse = (b == 0) ? 0 : (($3 == "higher") ? (b - c) / b * 100 : (c - b) / b * 100)
      tag = (worse > thr) ? "PEOR " : "ok   "
      if (worse > thr) bad++
      printf "%s  %-45s base %14.1f  hoy %14.1f  %+6.1f %%\n", tag, $1, b, c, -worse
    }
    END {
      for (m in base) if (!(m in seen)) { printf "FALTA  %s\n", m; bad++ }
      exit (bad > 0 ? 1 : 0)
    }' "$BASELINE" "$1"
}

step bench-storebench 7200 storebench_runs
step bench-go         7200 mac_go test -p 1 -run '^$' -bench . -benchmem -count "$BENCH_RUNS" ./store/
metrics "$OUT/bench-storebench.log" "$OUT/bench-go.log" > "$OUT/bench-metrics.tsv"
if [ -s "$OUT/bench-metrics.tsv" ]; then
  step bench-compare 0 compare "$OUT/bench-metrics.tsv"
  grep -e '^PEOR' -e '^FALTA' -e '^no había' "$OUT/bench-compare.log" | sed 's/^/         /' | tee -a "$SUMMARY"
else
  say "FALLA    bench-compare          sin métricas (¿fallaron los benchmarks?)"; failures=$((failures + 1))
fi

# ---------- 2. fuzzers ----------

# fuzz_one <paquete> <Fuzzer>: corre un fuzzer y guarda los casos nuevos que deje en testdata/fuzz/<Fuzzer>.
fuzz_one() {
  local pkg=$1 fn=$2 corpus before extra=()
  corpus=$SRC/$pkg/testdata/fuzz/$fn
  before=$(ls "$corpus" 2>/dev/null)
  # Minimizar una entrada nueva dura hasta 60 s por omisión: con un -fuzztime corto el fuzzer termina con
  # «context deadline exceeded» sin haber fallado. Se acota (page, como siempre, con 1x).
  extra=(-fuzzminimizetime "$FUZZ_MINIMIZE")
  [ "$pkg" = store/internal/page ] && extra=(-fuzzminimizetime 1x)
  mac_go test "./$pkg" -run '^$' -fuzz "^$fn\$" -fuzztime "$FUZZTIME" -parallel "$FUZZ_PARALLEL" "${extra[@]}"
  local rc=$? f
  for f in $(ls "$corpus" 2>/dev/null); do
    grep -qx "$f" <<<"$before" && continue
    for dest in "$OUT/fuzz-found/$pkg/$fn" "$CI/fuzz-found/$pkg/$fn"; do mkdir -p "$dest" && cp "$corpus/$f" "$dest/"; done
    echo "CASO NUEVO: $pkg/testdata/fuzz/$fn/$f"
  done
  return $rc
}

while read -r file fn; do
  pkg=$(dirname "${file#"$SRC"/}")
  if ! step "fuzz-${pkg##*/}-$fn" 1800 fuzz_one "$pkg" "$fn"; then
    grep '^CASO NUEVO' "$OUT/fuzz-${pkg##*/}-$fn.log" | sed 's/^/         /' | tee -a "$SUMMARY"
  fi
done < <(grep -rH '^func Fuzz' --include='*_test.go' "$SRC/store" | sed -E 's/^([^:]+):func (Fuzz[A-Za-z0-9_]*).*/\1 \2/' | sort)

# ---------- 3. cortes en la VM ----------

step vm-lazyfs   3600 vm_script powercut-lazyfs.sh
step vm-flakey   3600 vm_script powercut-flakey.sh
step vm-diskfull 3600 vm_script diskfull.sh
vm_stop

if [ "$failures" = 0 ]; then result=OK; else result="FALLA ($failures pasos)"; fi
say "== resultado: $result en $(( ($(date +%s) - start) / 60 )) min — VM detenida — $OUT"
cp "$SUMMARY" "$CI/last-nightly.txt"
prune 'nightly-*' 14
[ "$failures" = 0 ]
