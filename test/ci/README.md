# CI de `et/store` en IA (tarea 48 de F-23)

La integración continua del motor corre en el equipo **IA** (macOS arm64) y en su VM Linux `etstore`
(`test/linuxvm/`). Son scripts bash; desde esta Mac solo se compila (build + vet), se sincroniza y se lanza.

| script | dónde se lanza | qué hace |
|---|---|---|
| `ci.sh` | Mac (o IA) | verificación de un cambio, ~10 min: sube la copia, VM: `go test -p 1 ./store/...`, `-race`, `powercut-lazyfs.sh ROUNDS=5`, `powercut-flakey.sh DROP_ROUNDS=3 ERROR_ROUNDS=2`; detiene la VM; macOS: `go test -p 1 ./store/...` |
| `nightly.sh` | IA (LaunchAgent 03:00) o Mac | benchmarks contra la línea base, fuzzers de `store/` y la suite completa de cortes |
| `status.sh` | Mac | muestra el último `ci.sh`, el último nocturno y qué corre ahora |
| `install-nightly.sh` / `uninstall-nightly.sh` | Mac | pone o quita el LaunchAgent `com.cgalvisleon.etstore-nightly` en IA |
| `lib.sh` | — | funciones comunes (candado, memoria, pasos, resumen) |

## Uso

```bash
./test/ci/ci.sh                 # antes de un `git backup`: OK o FALLA por paso; sale 0 si todo OK
./test/ci/status.sh             # último resultado (ci y nightly); `status.sh ci` o `status.sh nightly`
./test/ci/nightly.sh            # lanza el nocturno ya mismo en IA (segundo plano); FUZZTIME=30s para uno corto
ssh ia cat ~/etstore-ci/last-nightly.txt
```

`ci.sh` en la Mac espera hasta 10 min a que `store/` compile (otro agente puede estar editándolo), sube los archivos
de git con `test/linuxvm/run.sh` (`ETSTORE_SYNC_ONLY=1`, sin `git push`), copia `test/ci` a `~/etstore-ci/bin` de IA
y corre allí el resto. En IA, `~/etstore-ci/bin/ci.sh` prueba la última copia subida. Variables: `CI_LAZYFS_ROUNDS`,
`CI_DROP_ROUNDS`, `CI_ERROR_ROUNDS`, `MIN_FREE`; en `nightly.sh`: `FUZZTIME` (5m), `FUZZ_PARALLEL` (4),
`FUZZ_MINIMIZE` (10s), `THRESHOLD` (10 %), `BENCH_RUNS` (3).

**«En cada cambio» sin hooks.** El repo no instala hooks: córrelo a mano antes de un `git backup`. Si quieres que
corra solo, copia esto en `.git/hooks/pre-push` y dale `chmod +x` (opcional; `git backup` ya habrá hecho el commit,
pero si la CI falla el push no sale; `git push --no-verify` lo salta):

```bash
#!/bin/bash
exec ./test/ci/ci.sh
```

## Cuidados con IA

IA corre Ollama y el corrector: no se tocan. Todo va con `nice` (`-n 15` en macOS; la VM mantiene sus topes de 3 CPU,
4 GiB y el vigilante de 2 GB), un solo proceso pesado a la vez (candado `~/etstore-ci/lock`, que `ci.sh` y el
nocturno esperan) y antes de cada paso se mira `memory_pressure`: con menos de 30 % libre espera (20 min en `ci.sh`,
hasta 2 h en el nocturno) y si no baja marca el paso `SALTADO`. Al salir, aun con error, la VM queda detenida.

## Resultados (en IA)

| | dónde |
|---|---|
| cada `ci.sh` | `~/etstore-ci/<fecha>/summary.txt` + un `<paso>.log` por paso; se guardan las últimas 30; copia en `~/etstore-ci/last-ci.txt` |
| cada noche | `~/etstore-ci/nightly-<fecha>/` (resumen, logs, `bench-metrics.tsv`, `bench-compare.log`, `fuzz-found/`); las últimas 14; copia en `~/etstore-ci/last-nightly.txt` |
| casos de fuzz | `~/etstore-ci/fuzz-found/<paquete>/<Fuzzer>/` (se acumulan; cópialos a `testdata/fuzz/` del repo para fijarlos) |
| cortes fallidos | dentro de la VM, `~/powercut-results/<script>-<fecha>/ronda-<n>/` (ver `test/linuxvm/README.md`) |
| lo que se probó | `~/etstore-ci/src.stamp` (fecha, commit y `+cambios` si había cambios sin commit) |

## El nocturno

Corre sobre **la última copia sincronizada** (`~/etstore/et.tgz`, la que dejó el último `ci.sh`, `run.sh` o
`vmrun.sh`): IA no tiene acceso al repo, así que la noche prueba lo último subido, no lo último del repo.

1. Benchmarks en macOS: `storebench -n 1000000 -batch 20000 -rate 5s` tres veces y
   `go test -run '^$' -bench . -benchmem -count 3 ./store/`; mediana de 3 por métrica.
2. Fuzzers: cada `func Fuzz` de `store/` (hoy 10: `page`, `wal`, `btree`, `control` y `FuzzOpenDamaged`), uno a la
   vez, `-fuzztime 5m -parallel 4 -fuzzminimizetime 10s` (`page` con `1x`). Un caso nuevo hace FALLA y queda
   guardado. La minimización se acota porque Go tarda hasta 60 s en minimizar cada entrada nueva: con un `-fuzztime`
   corto el fuzzer sale con `context deadline exceeded` sin haber encontrado nada (pasó con `FuzzReadNode` a 30 s).
3. VM: `powercut-lazyfs.sh` (20 rondas), `powercut-flakey.sh` (10 + 5), `diskfull.sh`; luego la detiene.

Dura algo más de 1 h con 5 min por fuzzer (la corrida de prueba, con 30 s por fuzzer, tardó 16 min). Instalar y quitar (sin sudo; `~/Library/LaunchAgents/com.cgalvisleon.etstore-nightly.plist`):

```bash
./test/ci/install-nightly.sh       # copia test/ci a IA y carga el agente (03:00)
./test/ci/uninstall-nightly.sh     # lo quita; deja ~/etstore-ci
```

La salida de launchd va a `~/etstore-ci/nightly-launchd.log`. IA no duerme (`caffeinate`), así que corre a su hora.

## Línea base

`~/etstore-ci/baseline.tsv` (métrica, valor, `higher`/`lower` = qué es mejor): `sb.*` de storebench (carga, Put
durable y diferido con 1/8/64 escritores, `Open`, lectura fría y en caché p50/p99) y `go.*` de `go test -bench`
(ns/op y allocs/op). Una métrica que empeora más de `THRESHOLD` % (10) frente a la base es `PEOR` y el paso
`bench-compare` da FALLA; una que falta es `FALTA`. Si no hay base, la primera corrida la crea.

Actualizarla tras un cambio aceptado (o una mejora):

```bash
ssh ia 'cp ~/etstore-ci/nightly-<fecha>/bench-metrics.tsv ~/etstore-ci/baseline.tsv'
```

La base guardada salió de la corrida de prueba del 2026-11-20 (copia `468707d +cambios`) y cuadra con
`test/storebench/RESULTADOS.md` (macOS de IA) dentro del 1–2 %, salvo dos métricas ruidosas que se dejaron con el
valor de `RESULTADOS.md`: `sb.put_durable_64` = 10.216 (las tres corridas dieron 10.246, 13.548 y 13.412: con la
mediana 13.412 una noche de ~10.200 daría una falsa FALLA del 24 %) y `sb.open_us` = 42.000 (34–41 ms en las
corridas; 10 % de 38 ms son 4 ms de ruido). La original quedó en `~/etstore-ci/baseline.orig.tsv`.
