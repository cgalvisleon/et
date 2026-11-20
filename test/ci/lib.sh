#!/bin/bash
# Funciones comunes de ci.sh y nightly.sh cuando corren en IA (se cargan con `source`).
# Carpetas en IA:
#   ~/etstore/et.tgz, et.stamp   la última copia subida desde la Mac (run.sh, vmrun.sh o ci.sh)
#   ~/etstore-ci/src.tgz, src/   la copia que se está probando (tomada bajo el candado)
#   ~/etstore-ci/bin/            estos scripts (los instala ci.sh o install-nightly.sh)
#   ~/etstore-ci/lock/           candado: una sola corrida pesada a la vez

export PATH=/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin
CI=$HOME/etstore-ci
SRC=$CI/src
MIN_FREE=${MIN_FREE:-30}   # % de memoria libre mínimo antes de cada paso
export TMPDIR=$CI/tmp      # temporales de go test, fuzz y storebench: se borran al salir
mkdir -p "$TMPDIR"

failures=0

# say <texto>: imprime y anota en el resumen.
say() { echo "$*" | tee -a "$SUMMARY"; }

# mem_free: porcentaje de memoria libre de macOS (memory_pressure).
mem_free() { memory_pressure 2>/dev/null | awk -F': ' '/free percentage/ {gsub("%", "", $2); print $2}'; }

# wait_memory <segundos máx>: espera a que haya al menos MIN_FREE % libre; falla si no llega a tiempo.
wait_memory() {
  local waited=0 free
  while :; do
    free=$(mem_free); free=${free:-0}
    [ "$free" -ge "$MIN_FREE" ] && return 0
    [ "$waited" -ge "$1" ] && return 1
    [ "$waited" = 0 ] && echo "   memoria libre $free % (< $MIN_FREE %): espero"
    sleep 60; waited=$((waited + 60))
  done
}

# take_lock <segundos máx>: candado de ~/etstore-ci (un directorio con el pid); libera uno de un proceso muerto.
take_lock() {
  local waited=0 pid
  mkdir -p "$CI"
  until mkdir "$CI/lock" 2>/dev/null; do
    pid=$(cat "$CI/lock/pid" 2>/dev/null)
    if [ -n "$pid" ] && ! kill -0 "$pid" 2>/dev/null; then rm -rf "$CI/lock"; continue; fi
    [ "$waited" -ge "$1" ] && { echo "otra corrida tiene el candado (pid ${pid:-?}) desde hace más de $1 s"; return 1; }
    [ "$waited" = 0 ] && echo "   otra corrida tiene el candado (pid ${pid:-?}): espero"
    sleep 30; waited=$((waited + 30))
  done
  echo $$ > "$CI/lock/pid"
  trap 'vm_stop; rm -rf "$TMPDIR" "$CI/lock"' EXIT   # al salir, aun por error: VM detenida y candado libre
}

# take_snapshot: copia la última subida (~/etstore/et.tgz) a src.tgz y la descomprime en src/ (cambio atómico).
take_snapshot() {
  [ -f "$HOME/etstore/et.tgz" ] || { echo "no hay ~/etstore/et.tgz: corre ci.sh desde la Mac primero"; return 1; }
  cp "$HOME/etstore/et.tgz" "$CI/src.tgz"
  cp "$HOME/etstore/et.stamp" "$CI/src.stamp" 2>/dev/null || echo "(sin sello)" > "$CI/src.stamp"
  rm -rf "$SRC.new" && mkdir -p "$SRC.new" && tar xzf "$CI/src.tgz" -C "$SRC.new" 2>/dev/null || return 1
  rm -rf "$SRC" && mv "$SRC.new" "$SRC"
}

# step <nombre> <seg. máx. de espera de memoria> <comando…>: corre el comando con su salida en <OUT>/<nombre>.log y
# anota en el resumen OK, FALLA o SALTADO con su duración.
step() {
  local name=$1 memwait=$2 t0 rc; shift 2
  if ! wait_memory "$memwait"; then
    say "$(printf 'SALTADO  %-22s memoria libre %s %% < %s %%' "$name" "$(mem_free)" "$MIN_FREE")"
    failures=$((failures + 1)); return 1
  fi
  echo "-- $name: $*"
  t0=$(date +%s)
  "$@" >"$OUT/$name.log" 2>&1; rc=$?
  if [ $rc = 0 ]; then
    say "$(printf 'OK       %-22s %4d s' "$name" $(($(date +%s) - t0)))"
  else
    say "$(printf 'FALLA    %-22s %4d s  (rc=%d, ver %s.log)' "$name" $(($(date +%s) - t0)) $rc "$name")"
    tail -n 15 "$OUT/$name.log" | sed 's/^/         | /'
    failures=$((failures + 1))
  fi
  return $rc
}

# vm <args>: test/linuxvm/run.sh en modo local sobre la copia en prueba.
vm_test() { ETSTORE_HOST=local ETSTORE_TGZ="$CI/src.tgz" bash "$SRC/test/linuxvm/run.sh" "$@"; }
# vm_script <script> [VAR=valor …]: test/linuxvm/vmrun.sh en modo local.
vm_script() { ETSTORE_HOST=local ETSTORE_TGZ="$CI/src.tgz" bash "$SRC/test/linuxvm/vmrun.sh" "$@"; }
# vm_stop: deja la VM detenida.
vm_stop() { limactl stop etstore >/dev/null 2>&1 || true; }

# mac_go <args>: go en macOS de IA sobre la copia en prueba, con GOWORK=off y nice -n 15.
mac_go() { (cd "$SRC" && GOWORK=off nice -n 15 go "$@"); }

# prune <patrón> <cuántas>: borra las carpetas más viejas de ~/etstore-ci que cumplen el patrón.
prune() {
  # shellcheck disable=SC2012,SC2086
  ls -1d "$CI"/$1 2>/dev/null | sort | awk -v k="$2" '{a[NR] = $0} END {for (i = 1; i <= NR - k; i++) print a[i]}' \
    | while read -r d; do rm -rf "$d"; done
}

# fuzz_one <paquete> <Fuzzer>: corre un fuzzer y guarda los casos nuevos que deje en testdata/fuzz/<Fuzzer>.
fuzz_one() {
  local pkg=$1 fn=$2 corpus before extra=()
  corpus=$SRC/$pkg/testdata/fuzz/$fn
  before=$(ls "$corpus" 2>/dev/null)
  # Minimizar una entrada nueva dura hasta 60 s por omisión: con un -fuzztime corto el fuzzer termina con
  # «context deadline exceeded» sin haber fallado. Se acota (page, como siempre, con 1x).
  extra=(-fuzzminimizetime "$FUZZ_MINIMIZE")
  [ "$pkg" = store/internal/page ] && extra=(-fuzzminimizetime 1x)
  local log rc f found=0
  log=$(mktemp)
  mac_go test "./$pkg" -run '^$' -fuzz "^$fn\$" -fuzztime "$FUZZTIME" -parallel "$FUZZ_PARALLEL" "${extra[@]}" 2>&1 | tee "$log"
  rc=${PIPESTATUS[0]}
  for f in $(ls "$corpus" 2>/dev/null); do
    grep -qx "$f" <<<"$before" && continue
    for dest in "$OUT/fuzz-found/$pkg/$fn" "$CI/fuzz-found/$pkg/$fn"; do mkdir -p "$dest" && cp "$corpus/$f" "$dest/"; done
    echo "CASO NUEVO: $pkg/testdata/fuzz/$fn/$f"
    found=1
  done
  # Al vencer -fuzztime mientras Go minimiza una entrada «interesante» (no un fallo), el fuzzer sale con
  # «context deadline exceeded» sin guardar caso: no es un hallazgo.
  if [ "$rc" -ne 0 ] && [ "$found" -eq 0 ] && grep -q 'context deadline exceeded' "$log" && ! grep -q 'Failing input' "$log"; then
    echo "AVISO: venció el tiempo minimizando una entrada; sin caso guardado: no es un hallazgo"
    rc=0
  fi
  rm -f "$log"
  return $rc
}

# fuzz_all: corre fuzz_one con cada func Fuzz de store/ (orden alfabético por archivo), cada uno como un paso.
# Usa OUT, SUMMARY, FUZZTIME, FUZZ_PARALLEL y FUZZ_MINIMIZE.
fuzz_all() {
  local file fn pkg
  while read -r file fn; do
    pkg=$(dirname "${file#"$SRC"/}")
    if ! step "fuzz-${pkg##*/}-$fn" 1800 fuzz_one "$pkg" "$fn"; then
      grep '^CASO NUEVO' "$OUT/fuzz-${pkg##*/}-$fn.log" | sed 's/^/         /' | tee -a "$SUMMARY"
    fi
  done < <(grep -rH '^func Fuzz' --include='*_test.go' "$SRC/store" | sed -E 's/^([^:]+):func (Fuzz[A-Za-z0-9_]*).*/\1 \2/' | sort)
}
