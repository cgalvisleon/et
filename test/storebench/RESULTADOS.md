# Resultados de `test/storebench` y de los benchmarks del store (F-23, tareas 29 y 33)

Fecha: **2026-11-20**. Código: el árbol de trabajo de `et` (rama `ia`, sin commit) copiado con `git ls-files` como hace
`test/linuxvm/run.sh`. Valores de 100 B, claves `k0000000000…` de 11 B, lotes de 20.000 (`Batch` + `Commit`),
configuración por omisión del motor (`STORE_CACHE_SIZE` 128 MB fuera del heap, `WAL_SEGMENT_SIZE` 128 MB,
`SYNC_ON_WRITE` true, `MAX_WAL_SIZE` 1 GB). Un solo proceso de medida a la vez, con `nice -n 10`; la memoria libre de IA
estuvo entre 61 % y 83 % en todas las corridas (Ollama del corrector siguió corriendo sin tocarlo).

## Máquinas

| | macOS (IA directo) | VM Linux `etstore` (en IA) |
|---|---|---|
| CPU | Apple M2 Pro, 10 núcleos | 3 vCPU (Lima `vz`, aarch64) |
| RAM | 16 GB | 4 GiB, sin swap |
| Disco | SSD NVMe interno de IA (APFS) | `virtio` de 100 GiB sparse sobre ese SSD (ext4, `discard`) |
| Go | 1.26.3 darwin/arm64 | 1.25.14 linux/arm64 |
| fsync | `F_FULLFSYNC` (vacía la caché del SSD) | `fdatasync` sobre virtio: el anfitrión no lo lleva hasta el SSD con la misma fuerza, por eso la escritura durable es ~35× más rápida que en macOS |

## Comandos

```bash
# en IA (macOS), binario compilado con GOWORK=off; TMPDIR en un directorio propio, borrado al final
/usr/bin/time -l nice -n 10 ./storebench -n 1000000  -batch 20000 -rate 5s
/usr/bin/time -l nice -n 10 ./storebench -n 10000000 -batch 20000
GOWORK=off nice -n 10 go test -p 1 -run '^$' -bench . -benchmem -count 3 ./store/

# en la VM (los mismos tres) y además
/usr/bin/time -v nice -n 10 ~/storebench -n 100000000 -batch 20000 -reopen -dir ~/sb100m
# Open repetido sobre lo ya cargado (dos veces con la caché del SO vaciada, una sin vaciar)
sync; echo 3 | sudo tee /proc/sys/vm/drop_caches; ~/storebench -n 100000000 -dir ~/sb100m -load=false
```

«Lectura fría» = 100.000 `Read` al azar sobre las N claves recién reabiertas (la caché de páginas del motor empieza
vacía; la del sistema operativo puede tener parte del archivo: con 1 y 10 millones el archivo cabe en ella, con 100
millones —16 GB contra 4 GiB de RAM— no). «En caché» = 1.000 claves leídas 200 veces.

## Tarea 29 — escala (VM Linux)

| claves | carga (claves/s) | disco | heap vivo tras cargar / tras reabrir y leer | RSS tras cargar / tras reabrir y leer (máx.) | `Open` tras `Close` | lectura fría p50 / p99 / p999 | en caché p50 / p99 / p999 |
|---|---|---|---|---|---|---|---|
| 1 M | 2,8 s (355.892) | 1.228 MB¹ | 6 / 6 MB | 147 / 151 MB (155) | 36 ms | 5,5 / 20,5 / 51 µs | 4,5 / 5,7 / 12 µs |
| 10 M | 26,7 s (374.284) | 2.244 MB | 7 / 6 MB | 146 / 150 MB (150) | 56 ms; repetido 34, 60, 21 ms | 18 / 49 / 122 µs² | 5,3 / 6,8 / 14 µs |
| **100 M** | **7 min 13 s (231.003)** | **16.685 MB** | **7 / 6 MB** | **148 / 151 MB (152)** | **97 ms; repetido 64, 76, 79 ms** | **127 / 201 / 295 µs** | **6,1 / 7,3 / 10,5 µs** |

¹ Incluye los stores de `-rate` y segmentos de WAL. ² Con la caché del SO vaciada: p50 51 µs, p99 112–122 µs.

macOS (IA), mismas escalas: 1 M → 4,2 s (236.105 claves/s), 1.102 MB¹, heap 6 MB, RSS 159/160 MB, `Open` 42 ms, fría
5,3/20/59 µs, en caché 4,5/4,9/17 µs. 10 M → 41,4 s (241.339 claves/s), 2.244 MB, heap 7/6 MB, RSS 150/155 MB,
`Open` 46 ms, fría 20/30/85 µs, en caché 5,1/5,6/12 µs.

**Reglas con 100 millones.**

- **Memoria**: heap vivo de 6–7 MB a cualquier escala (100 mil → 100 millones), RSS máximo 152 MB = caché de 128 MB +
  19 % (dentro de caché + 10–20 %). Se cumple.
- **`Open`**: entre 21 y 60 ms con 10 millones y entre 64 y 97 ms con 100 millones (cuatro aperturas de cada uno, en
  frío y en tibio). No crece en proporción a las claves (×10 claves → ×1,6 tiempo) y queda bajo 100 ms, pero tampoco es
  plano: hay ~30–40 ms más con 100 millones. Probable causa (sin perfilar): los 16 archivos de datos de 1 GB y las
  páginas de catálogo/meta leídas en frío; el WAL a rehacer es casi nulo tras el checkpoint final de `Close`. Vale
  perfilarlo en la corrida de 1.000 millones (tarea 51).
- **Disco**: ~167 B por clave a 100 millones (11 B de clave + 100 B de valor + cabeceras de versión y de página). A
  1.000 millones serían ~167 GB: no caben en la VM de 100 GiB.
- **Carga**: baja de ~370 mil a 231 mil claves/s al pasar de 10 a 100 millones (el árbol ya no cabe en la caché del SO
  de la VM y los checkpoints escriben más páginas).

## Tarea 33 — benchmarks (IA macOS NVMe vs VM Linux)

### `storebench -rate 5s` (Put suelto por segundo; diferido incluye el `WaitDurable` final)

| escritores | macOS durable | macOS diferido | VM durable | VM diferido |
|---|---|---|---|---|
| 1 | 340 | 82.203 | 12.465 | 93.418 |
| 8 | 1.330 | 167.325 | 16.177 | 185.035 |
| 64 | 10.216 | 158.315 | 21.321 | 185.192 |

El durable en macOS lo fija `F_FULLFSYNC` (~3 ms por fsync): el group commit escala casi lineal con los escritores
(340 → 1.330 → 10.216). En la VM el fsync es barato y el techo lo pone la CPU.

### Lectura y `Open` (de la tabla de arriba)

| | macOS 1 M | macOS 10 M | VM 1 M | VM 10 M | VM 100 M |
|---|---|---|---|---|---|
| fría p50 / p99 | 5,3 / 20 µs | 20 / 30 µs | 5,5 / 21 µs | 18 / 49 µs | 127 / 201 µs |
| en caché p50 / p99 | 4,5 / 4,9 µs | 5,1 / 5,6 µs | 4,5 / 5,7 µs | 5,3 / 6,8 µs | 6,1 / 7,3 µs |
| `Open` | 42 ms | 46 ms | 36 ms | 56 ms (21–60) | 97 ms (64–97) |

### `go test -bench` del store (`store/recursive_test.go`, mediana de 3, ns/op)

| benchmark | macOS | VM | B/op · allocs/op |
|---|---|---|---|
| PutPlain/commit | 2.994.641 | 73.682 | 2.617 · 40 (VM 2.710 · 44) |
| PutPlain/deferred | 10.787 | 7.758 | 2.702 · 42 |
| PutIndexed/idx=1/commit | 2.977.320 | 79.851 | 3.724 · 55 (VM 4.066 · 61) |
| PutIndexed/idx=1/deferred | 21.478 | 16.371 | 4.099 · 60 |
| PutIndexed/idx=2/commit | 2.976.948 | 81.017 | 4.503 · 68 (VM 4.855 · 74) |
| PutIndexed/idx=2/deferred | 25.267 | 18.773 | 4.876 · 72 |
| QueryByIndex | 875.202 | 883.741 | 190.409 · 4.049 |
| ScanFilter | 13.925.720 | 13.565.603 | 12.841.576 · 280.281 |

No hay otros `func Benchmark` en `store/` (incluidos los paquetes internos).

## Limpieza

Los datos de 100 millones (`~/sb100m`, 16 GB) y de 10 millones se borraron y se corrió `fstrim` (la VM volvió a 3,0 GB
usados); el directorio temporal de IA (`~/etbench`) se borró; la VM quedó detenida.

## Qué falta

- Perfilar `Open` con 100 millones o más (sube de ~50 a ~80 ms entre 10 y 100 millones): tarea 51, con 1.000 millones,
  que necesita ~170 GB de disco (más que la VM).
- Benchmarks en Linux con NVMe propio (aplazado por decisión del usuario).
- Recorridos (`ForEach`/`ForEachRange`), p99 de lectura con 64 escritores y amplificación de escritura: tarea 41
  (fase 2).

---

# Tareas 71, 72 y 73 — medidas de producción (bloque A de F-23)

Fecha: **2026-11-20**. Copia medida en 71 y 72: **`c28bc39` con `store/` sin cambios** (el sello dice `+cambios` por
lo nuevo de `test/storebench`; a esa hora el árbol de `store/` estaba igual al commit). Subida con
`test/linuxvm/run.sh` (`ETSTORE_SYNC_ONLY=1`) y guardada aparte en IA (`~/etbench/et.tgz`) para que otra subida no
cambiara lo medido. Todo bajo el candado de `test/ci` (`~/etstore-ci/lock`), un proceso pesado a la vez, `nice`; la
memoria libre de IA estuvo entre 83 % y 85 % (Ollama y el corrector siguieron sin tocarlos). La VM quedó en 4 GiB
(no hizo falta subirla: el proceso no pasa de 567 MB) y detenida.

Herramientas nuevas: `storebench -crashwal <MB> -dir d` (carga lotes durables de 20.000 claves sin `Close` hasta que el
WAL por rehacer llega a esos MB y se mata con `SIGKILL`) y `storebench -recover -dir d` (mide `Open`, muestra `Count` y
`Engine.Verify` y cierra); `test/storebench/medir.sh memoria|caida-mac|caida-vm` las corre en IA.

```bash
./test/storebench/medir.sh memoria                 # 71 (VM): STORE_CACHE_SIZE=512, 100 M claves, y dos reaperturas en frío
./test/storebench/medir.sh caida-mac               # 72 (macOS de IA), 3 vueltas: CHECKPOINT_TIMEOUT=3600 storebench -crashwal 1024, luego -recover
./test/storebench/medir.sh caida-vm                # 72 (VM), 3 vueltas, drop_caches antes de cada Open
CRASH_MB=2048 REPS=1 ./test/storebench/medir.sh caida-mac   # 72: el máximo que acumula la configuración por omisión
CRASH_MB=2048 REPS=1 ./test/storebench/medir.sh caida-vm
```

## Tarea 71 — memoria con caché de 512 MB y 100 millones de claves (VM)

`STORE_CACHE_SIZE=512`, claves de 11 B, valores de 100 B, lotes de 20.000. Una corrida que carga, cierra, reabre y lee
(un proceso) y dos reaperturas en procesos nuevos con la caché del SO vaciada (`drop_caches`). RSS máximo de
`getrusage` (el de `/usr/bin/time -v` entre paréntesis).

| | carga | heap vivo | RSS actual | RSS máximo | caché + % | `Open` | lectura fría p50 / p99 / p999 | en caché p50 / p99 |
|---|---|---|---|---|---|---|---|---|
| tras cargar (100 M) | 6 min 59 s (238.324 claves/s), 16.685 MB | 21 MB | 562 MB | **566 MB** (567) | **+10,5 %** | — | — | — |
| tras reabrir y leer (mismo proceso) | — | 17 MB | 559 MB | 566 MB | +10,5 % | 172 ms | 133 / 350 / 721 µs | 6,0 / 7,5 µs |
| proceso nuevo, frío, vuelta 1 | — | 17 MB | 555 MB | **555 MB** (555) | **+8,4 %** | 43 ms | 138 / 283 / 1.272 µs | 6,0 / 7,3 µs |
| proceso nuevo, frío, vuelta 2 | — | 17 MB | 555 MB | 554 MB (555) | +8,3 % | 36 ms | 132 / 204 / 354 µs | 6,0 / 7,5 µs |

**Resultado: casi.** Reabrir y leer cumple (+8,4 %); la carga pasa el criterio por 3 MB (566 MB contra el tope de
563 MB, +10,5 %). Lo que no es caché sube con la caché: con 128 MB eran ~24 MB (heap vivo 6–7 MB), con 512 MB son
~43–54 MB (heap vivo 17 MB tras abrir, 21 MB durante la carga). El heap crece unos 11 MB con la caché (estructuras por
búfer en el heap de Go, probable; sin perfilar) y durante la carga se suma la holgura del recolector. Para cumplir
también al cargar haría falta bajar el heap por búfer o el pico de la carga (perfilar con `pprof` el heap vivo con
512 MB), o aceptar la meta como «caché + 10 % en régimen, + 11 % cargando». Los datos se borraron y `fstrim` liberó
76,7 GiB (la VM volvió a 3,1 GB usados).

## Tarea 72 — `Open` tras `kill -9` con ~1 GB de WAL por rehacer

Configuración por omisión (`MAX_WAL_SIZE` 1024 MB, `WAL_SEGMENT_SIZE` 128 MB, `STORE_CACHE_SIZE` 128 MB,
`SYNC_ON_WRITE` true) salvo `CHECKPOINT_TIMEOUT=3600`. Motor nuevo, lotes durables de 20.000 claves secuenciales de
100 B (~200 B de WAL por clave). `Open` medido dentro del proceso (sin contar arrancarlo); después `Count` y
`Engine.Verify` dieron lo esperado y limpio en todas las vueltas. En la VM la caché del SO se vació antes de cada
`Open`; en macOS no se puede sin `sudo` (`purge`), así que el WAL pudo estar en la caché del SO.

| equipo | vuelta | claves | WAL por rehacer | `Open` | MB/s | RSS máx. al recuperar |
|---|---|---|---|---|---|---|
| macOS IA (NVMe) | 1 | 5,10 M | 1.025 MB | 2,386 s | 430 | 152 MB |
| macOS IA (NVMe) | 2 | 5,10 M | 1.025 MB | 2,422 s | 423 | 152 MB |
| macOS IA (NVMe) | 3 | 5,10 M | 1.025 MB | 2,402 s | 427 | 152 MB |
| VM Linux (frío) | 1 | 5,10 M | 1.025 MB | 2,453 s | 418 | 148 MB |
| VM Linux (frío) | 2 | 5,10 M | 1.025 MB | 2,477 s | 414 | 148 MB |
| VM Linux (frío) | 3 | 5,10 M | 1.025 MB | 2,453 s | 418 | 148 MB |
| macOS IA, máximo natural | 1 | 13,62 M | 1.071 MB (pico visto 1.125) | 2,520 s | 425 | 152 MB |
| VM Linux, máximo natural | 1 | 13,74 M | 1.077 MB (pico visto 1.133) | 2,492 s | 432 | 148 MB |

**Resultado: cumple** (≤ 5 s): ~2,4–2,5 s por GB en los dos equipos, ~420 MB/s de rehacer (lo pone la CPU, no el
disco: macOS con el WAL en caché y la VM en frío dan lo mismo).

**El checkpoint por volumen.** Se dispara a `MAX_WAL_SIZE / 1,9` = 539 MB, pero el WAL por rehacer se cuenta desde el
punto de rehacer del último checkpoint **terminado**, y mientras uno corre (escribe páginas, 7–11 s con esta carga) el
WAL sigue creciendo. Con carga continua el primero no había terminado al llegar a 1.025 MB, así que el GB por rehacer
se alcanza con la configuración por omisión. Dejando correr tres checkpoints, el máximo que se acumula es
~1.125–1.133 MB (539 MB + lo escrito durante el checkpoint); el tope duro es el freno de los escritores en
2 × `MAX_WAL_SIZE` = 2 GB, que a ~420 MB/s se rehace en ~4,9 s. **La cota de 5 s se cumple con `MAX_WAL_SIZE` = 1024
aun en el peor caso** (2 GB); con 1.536 MB el peor caso (3 GB) daría ~7 s. Cota propuesta para la documentación: «~2,5
s por GB de WAL; con la configuración por omisión ≤ 3 s en régimen y ≤ 5 s en el tope del freno».

**`Stats.WALSinceCheckpoint` se queda corto mientras corre un checkpoint.** Mide `SizeSince(RedoPoint())`, y
`BeginCheckpoint` mueve el punto de rehacer al *empezar*: en las vueltas de arriba marcaba 420–537 MB con 1.025–1.077
MB por rehacer. Lo que rehace la recuperación es `walBacklog()` (desde el del último checkpoint terminado), que no se
expone. `storebench -crashwal` lo reconstruye (`redoTracker`: suma lo escrito y anota dónde empezó cada checkpoint,
error ≤ un lote ≈ 4 MB). Queda como hallazgo para `store/` (no tocado): exponer el backlog real en `Stats` o aclarar
el comentario del campo.

## Tarea 73 — fuzzers 8 h (macOS de IA)

`./test/ci/fuzzlong.sh`: los 10 `func Fuzz` de `store/` uno a la vez, `-fuzztime 48m -parallel 4`, `nice -n 15`,
`-fuzzminimizetime 10s` (`page` 1x), con `nohup` bajo el candado. Lanzada el 2026-11-20 a las 11:38 sobre `c28bc39` con
`store/` sin cambios (una primera de 11:37 tomó la copia de trabajo con la tarea 70 a medio hacer —
`FuzzOpenDamaged/seed#0` fallaba con «root, fast root or deleted head is the meta page»— y se descartó:
`~/etstore-ci/descartada-fuzzlong-20261120-113725`). Fin previsto **19:58**: no pasa por las 03:00, el nocturno queda
igual. Estado: **en curso**; consultar con `./test/ci/status.sh fuzzlong` (resumen en
`~/etstore-ci/fuzzlong-20261120-113812/summary.txt`, al terminar `~/etstore-ci/last-fuzzlong.txt`; casos nuevos en
`~/etstore-ci/fuzz-found/`). Mientras corre, `ci.sh` espera el candado (hasta 1 h) y falla si no lo obtiene.
