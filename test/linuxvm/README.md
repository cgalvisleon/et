# VM Linux para validar `store/` (tareas 32, 33 y 50 de F-23)

Una máquina virtual Ubuntu 24.04 arm64 (Lima, `vmType: vz`) en el equipo **IA**
para correr la suite de `et/store` en Linux y probar fallos de disco reales con
**LazyFS** (cortes de energía), **dm-flakey** (escrituras perdidas, errores de E/S)
y un loop ext4 chico (disco lleno). Los scripts se lanzan desde la Mac
y llegan a IA con `ssh ia`.

## Topes (para no bloquear IA)

- 3 CPU, 4 GiB de RAM, disco virtual de 100 GiB (sparse: crece solo con el uso).
- Sin montajes del host (`mounts: []`): el código entra copiado, nunca se
  escribe en IA desde la VM.
- Las pruebas corren con `go test -p 1` y `nice -n 10`.
- Un vigilante (`vigilante.service` en la VM, log en `/var/log/vigilante.log`)
  mata cualquier `*.test` con más de 2 GB de RSS (o el mayor si la suma pasa de 3 GB).
- Sin arranque automático. IA también corre Ollama (el corrector): deja la VM
  detenida cuando no la uses.

## Uso

```bash
./test/linuxvm/setup.sh            # instala Lima en IA, crea/arranca la VM, Go 1.25.x, build-essential, dmsetup, fuse3
./test/linuxvm/setup.sh --smoke    # + prueba de humo de dm-flakey (loop de 64 MB, ext4, drop_writes)
./test/linuxvm/setup.sh --lazyfs   # + compila LazyFS en ~/lazyfs de la VM y prueba clear-cache

./test/linuxvm/run.sh                      # sincroniza y corre go test -p 1 ./store/...
./test/linuxvm/run.sh -race ./store/...    # cualquier argumento de go test
ETSTORE_STOP=1 ./test/linuxvm/run.sh       # detiene la VM al terminar
```

`run.sh` empaqueta los archivos que git conoce (sin `.git`, ignorados ni
binarios), los copia a IA (`~/etstore/et.tgz`) y a la VM (`limactl copy`),
y corre `go test` en `~/et` con `GOWORK=off` bajo `/usr/bin/time -v`
(imprime el RSS máximo). Arranca la VM si estaba detenida.

`setup.sh` es idempotente; la definición de la VM es `etstore.yaml`
(en IA queda copiada en `~/etstore/etstore.yaml`).

## Fallos de disco reales (tareas 33 y 50)

```bash
./test/linuxvm/vmrun.sh powercut-lazyfs.sh [ROUNDS=20] [SEED=1] [MAXCUT=6]
./test/linuxvm/vmrun.sh powercut-flakey.sh [DROP_ROUNDS=10] [ERROR_ROUNDS=5] [SEED=101] [MAXCUT=6]
./test/linuxvm/vmrun.sh diskfull.sh [SEED=201]
ETSTORE_STOP=1 ./test/linuxvm/vmrun.sh …   # detiene la VM al terminar
```

`vmrun.sh` espera a que `./test/powercut` compile en la Mac (otro trabajo puede
tener `store/` a medio editar), sincroniza igual que `run.sh`, compila
`test/powercut` en la VM (`~/powercut`) y corre el script dentro de ella. Cada
script corre un solo proceso de carga a la vez, con `nice -n 10`, motores chicos
(`STORE_CACHE_SIZE=8`, `WAL_SEGMENT_SIZE=16` o 4) y datos de pocos MB.

`test/powercut` (solo la API pública de `store`) tiene un escritor por store
(4 por omisión) con `Put`, `Delete`, lotes con `Commit`, operaciones diferidas
con `WaitDurable`, `Empty` y `DropStore`, más checkpoints y limpiezas a mano.
Anota cada intento (`I`), cada operación **confirmada y durable** (`C`) y cada
rechazo sin efecto (`F`) en un registro que vive en el disco normal de la VM,
nunca en el que se corta. `-mode verify` reabre y exige que cada store sea el
estado tras un prefijo de las operaciones de su escritor que llegue al menos a
la última confirmada (nada confirmado falta, un lote está entero o no está), que
`Count` coincida con el recorrido y que `Engine.Verify` dé limpio. Como en las
pruebas del motor (`checkAfterCrash`), tras un corte se aceptan páginas
perdidas, cadenas de desborde huérfanas y archivos huérfanos (escritos antes de
un registro que no llegó a ser durable): se reparan con `Repair` y el `Verify`
siguiente debe dar limpio (`huerfanos_reparados` los cuenta).

| script | qué corta | cómo |
|---|---|---|
| `powercut-lazyfs.sh` | lo que no tuvo fsync | ronda mod 4: 0 y 1 `kill -9` + `lazyfs::clear-cache`; 2 `lazyfs::crash` antes del próximo `fsync`/`write`/`rename` de un archivo del motor (WAL, datos, control, catálogo); 3 `lazyfs::torn-op` sobre el segmento 1 del WAL (una escritura partida en 3, solo llega la 1.ª parte). Luego desmontar, montar y `verify`. |
| `powercut-flakey.sh` | `drop_writes` | loop de 256 MB, ext4 sobre `/dev/mapper/pcflakey`: SIGSTOP a la carga, tabla a `drop_writes` (`dmsetup suspend --noflush --nolockfs`: así ext4 no vacía su caché), `kill -9`, desmontar (lo sucio se pierde), tabla normal, montar y `verify`. Equivale a un corte en ese instante. |
| `powercut-flakey.sh` | `error_writes` | `-mode ioerror`: con la carga corriendo, la tabla pasa a `error_writes` (write y fsync dan EIO). Espera que todos los escritores paren, prueba `Put`, `Read`, `WaitDurable` y `Close` (cada uno con tope de tiempo: un cuelgue es fallo) y sale. Luego tabla normal, desmontar, `e2fsck -fy`, montar y `verify`. |
| `diskfull.sh` | lugar | loop ext4 de 128 MB sin bloques reservados y 32 MB de relleno; `-mode diskfull` llena hasta `ErrDiskFull`, comprueba las lecturas, borra el relleno y exige que las escrituras vuelvan sin reabrir (error limpio) o que `Close` + `Open` recupere todo (solo lectura); `Count` y `Verify` limpios, también tras reabrir. Casos: segmentos de 16 MB, de 4 MB, de 4 MB con `SYNC_ON_WRITE=false`, y de 16 MB con `-closefull` (`Close` y `Open` con el disco todavía lleno antes de liberar). |

**Cómo leer el resultado.** Cada script imprime una línea por ronda y la deja en
`~/powercut-results/<script>-<fecha>/summary.txt` de la VM; la última línea dice
`N/M rondas bien, X confirmaciones verificadas, fallos: <semilla>:<tipo> …`.
En una ronda buena, `VERIFY-OK open=… confirmadas=… ops=… claves=… inciertas=…
sobrevivieron=… huerfanos_reparados=… problemas=0`: `inciertas` son las
operaciones posteriores a la última confirmada de cada escritor (podían estar o
no) y `sobrevivieron` cuántas quedaron. Una ronda fallida imprime `VERIFY-FAIL`
con cada problema y deja en `ronda-<n>/` sus salidas, el registro y el motor
(`root/` o `db/`). Para repetirla: el mismo script con los mismos `ROUNDS`/`SEED` y
`ONLY=<ronda>` (la semilla y el tipo salen del número de ronda; el instante del
corte también, aunque lo que alcanza a escribir la carga en ese tiempo varía), o
a mano en la VM con
`~/powercut -mode verify -dir <motor> -log <confirm.log>`. En `error_writes` la
línea lleva también `IOERROR escritores_parados=…`, lo que devolvieron `Put`,
`Read`, `WaitDurable` y `Close` después, y el código de `e2fsck`. En
`diskfull.sh` cada caso imprime sus líneas `DISKFULL …` y `DISKFULL-OK` o
`DISKFULL-FAIL`.

## Detener o borrar

```bash
ssh ia 'PATH=/opt/homebrew/bin:$PATH limactl stop etstore'
ssh ia 'PATH=/opt/homebrew/bin:$PATH limactl delete etstore'   # borra el disco
ssh ia 'PATH=/opt/homebrew/bin:$PATH limactl shell etstore'    # entrar a la VM
```

## Qué falta

- NVMe real (la VM ve un disco virtio sobre el SSD de IA: el orden y la
  atomicidad de las escrituras de un NVMe de verdad no se prueban acá) y
  benchmarks sobre él.
- `diskfull.sh` no llega al modo de solo lectura en ext4 (todo disco lleno sale
  limpio, en la entrada o al extender): para eso haría falta un fsync que falle
  por lugar (volumen fino, NFS).
