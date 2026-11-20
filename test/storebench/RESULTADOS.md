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
