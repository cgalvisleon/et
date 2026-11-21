# Manual de operación de `et/store`

Este manual cubre la tarea 49 de F-23. Explica cómo dimensionar, operar, respaldar, replicar, verificar y reparar el
motor de almacenamiento `github.com/cgalvisleon/et/store`, y qué hacer ante cada error.

La referencia completa del diseño es `store/doc.go`, y la API real está en `store/catalog.go`. Si este manual y el
código difieren, manda el código. Los números salen de `store/doc.go`, del código y de
`test/storebench/RESULTADOS.md`. Lo que no está claro se marca **«a confirmar»**.

Siglas que se usan en el manual:

- **WAL** (registro previo de escritura): todo cambio se anota ahí antes de tocar las páginas.
- **lsn** (número de secuencia del registro): la posición de un registro en el WAL.
- **xid** (número de transacción, de 64 bits).
- **MVCC** (control de concurrencia multiversión).
- **PITR** (recuperación a un punto en el tiempo).
- **DDL** (cambios de estructura): crear, vaciar y borrar stores.
- **RSS** (memoria residente del proceso).
- **SO** (sistema operativo).
- **CI** (integración continua).
- **VM** (máquina virtual).

---

## 1. Qué es y cuándo usarlo

`et/store` es un motor de pares `(id string, data []byte)` sobre el que se construye una base de datos. Josefina es su
consumidor principal. Copia la arquitectura de almacenamiento de PostgreSQL:

- un árbol B+ en disco con páginas de 8 KB;
- una caché de páginas de tamaño fijo;
- un WAL con imágenes de página entera tras cada checkpoint;
- MVCC con xid de 64 bits y un archivo de estados;
- checkpoints repartidos en el tiempo y recuperación desde el último checkpoint.

Un **motor** es un directorio con una caché y un WAL para todos sus **stores**. Cada store es un árbol propio con sus
archivos. Un proceso puede abrir varios motores, cada uno sobre un directorio distinto. El archivo `LOCK` impide que
dos motores abran el mismo directorio.

### Las dos reglas

1. **La memoria está acotada por la caché.** Nada del árbol vive en RAM fuera de la caché de páginas, y ninguna
   estructura en memoria crece con el número de claves. La caché vive fuera del heap de Go (`mmap` anónimo en darwin,
   linux y los BSD; heap en windows), así que el recolector de basura no la duplica.
2. **`Open` no crece con los datos.** `Open` lee el control y el catálogo, y reproduce solo el WAL escrito desde el
   punto de rehacer del último checkpoint terminado. `MAX_WAL_SIZE` acota ese volumen. La caché empieza vacía.

Medido con 100 millones de claves (`RESULTADOS.md`, VM Linux): el heap vivo quedó en 6–7 MB a cualquier escala, y
`Open` tras `Close` tardó entre 64 y 97 ms.

### Cuándo usarlo

- Hay más claves de las que caben en RAM, y se necesitan recorridos ordenados (por prefijo y por rango) y escrituras
  atómicas de varias claves (lotes).
- Se necesita durabilidad con un tiempo de arranque acotado, respaldo en caliente, archivo continuo del WAL, PITR,
  réplica física o un flujo lógico de cambios.

### Límites

| Qué | Límite |
|---|---|
| id | de 1 a 2.000 B (`MaxIDLen`) |
| valor | hasta 64 MiB (`MaxValueLen`). Desde ~2 KB va a páginas de desborde: en línea solo si 28 + id + valor ≤ 2.032 B |
| nombre de un store | UTF-8 válido, sin el carácter nulo, de 1 a 255 B (`MaxNameLen`) |
| lotes y snapshots | 10 minutos (`TRANSACTION_TIMEOUT`). `SnapshotFor(0)` no tiene tiempo máximo |
| ranuras de replicación | 16, con nombres de 1 a 64 B |
| aislamiento | snapshot (como Repeatable Read). No hay detección de interbloqueos: dos lotes que se esperan salen por `LOCK_TIMEOUT` con `ErrConflict` |
| recorrido dentro de un lote | no existe (`ForEachIn`). Se recorre con un snapshot y, dentro del lote, se lee por id con `ReadIn` |
| réplica síncrona | no existe: el primario no espera a la réplica al confirmar |
| líneas de tiempo | no existen: el primario viejo no puede seguir a una réplica promovida |
| formato en disco | versión 1, todavía **no congelado** (tarea 54; ver la sección 10) |

### Abrir y cerrar

```go
e, err := store.Open("/var/lib/miapp/db") // lo crea si no existe
if err != nil {
	log.Fatal(err)
}
defer e.Close() // checkpoint final

users, err := e.Store("users") // abre el store, o lo crea
if err != nil {
	log.Fatal(err)
}
if err := users.Put("u1", []byte(`{"name":"Ana"}`)); err != nil {
	log.Fatal(err)
}
```

`Close` no espera a los lotes abiertos: quedan inservibles (`ErrClosed`) y se leen abortados al reabrir. También
detiene la limpieza de fondo y hace el checkpoint final.

---

## 2. Dimensionar

### Memoria: `STORE_CACHE_SIZE`

La memoria del proceso es la caché más una parte pequeña que no es caché. Medido en `RESULTADOS.md`:

| Caché | RSS máximo | Por encima de la caché | Notas |
|---|---|---|---|
| 128 MB (por omisión) | 148–152 MB | ~20–24 MB (+16–19 %) | heap vivo 6–7 MB; de 1 a 100 millones de claves |
| 512 MB, en régimen | 555 MB | **+8,4 %** | proceso nuevo, reabrir y leer 100 millones de claves |
| 512 MB, cargando | 566 MB | +10,5 % | heap vivo 21 MB durante la carga |

**Regla práctica:** con una caché de 512 MB o más, cuenta la caché más un 8–11 %. Con cachés chicas, cuenta la caché
más unos 25 MB fijos. La parte que no es caché crece algo con la caché (~11 MB de heap al pasar de 128 a 512 MB, sin
perfilar). La decisión 72 fija que el criterio se mide en régimen.

Elige la caché según el conjunto de trabajo: la lectura en caché tarda ~5–6 µs (p50); la lectura fría con 100
millones de claves, 127–138 µs (p50) en la VM. Un recorrido grande no vacía la caché (usa un anillo de búferes).

### Tiempo de `Open` tras una caída: `MAX_WAL_SIZE`

`MAX_WAL_SIZE` (1.024 MB por omisión) fija cuánto WAL rehace `Open` tras una caída. Rehacer va a **~420 MB/s** y lo
limita la CPU, no el disco: macOS con el WAL en la caché del SO y la VM Linux en frío dieron lo mismo.

| WAL por rehacer | `Open` medido o calculado |
|---|---|
| tras `Close` (casi nada) | 36–97 ms (de 1 a 100 millones de claves) |
| 1 GB | **2,39–2,48 s** (medido, macOS y VM) |
| ~1,13 GB (lo máximo visto con carga continua y la configuración por omisión) | 2,49–2,52 s (medido) |
| 2 GB = 2 × `MAX_WAL_SIZE` (el tope del freno de escritores) | **~4,9 s** (calculado a 420 MB/s) |

Cómo funciona:

- Un checkpoint se dispara al llegar a `MAX_WAL_SIZE / 1,9` (539 MB con 1 GB) o cada `CHECKPOINT_TIMEOUT`.
- El WAL por rehacer se cuenta desde el último checkpoint **terminado**. Mientras uno corre, el WAL sigue creciendo.
- Con más de 2 × `MAX_WAL_SIZE` sin reciclar, los escritores esperan un checkpoint y, tras `LOCK_TIMEOUT`, reciben
  `ErrThrottled`.

Regla: ~2,5 s por GB de WAL. Con la configuración por omisión, `Open` tarda ≤ 3 s en régimen y ≤ 5 s en el tope del
freno. Si subes `MAX_WAL_SIZE` a 1.536 MB, el peor caso (3 GB) da ~7 s.

`MAX_WAL_SIZE` también fija los segmentos de repuesto del WAL: `MAX_WAL_SIZE / 2 / WAL_SEGMENT_SIZE` (4 por omisión).

### Disco

Cada versión ocupa ~32 B fijos (28 B de encabezado y 4 B del puntero) más el id y el valor. Las hojas quedan al ~87 %.

Medido en estado estable (`TestDiskSteadyState` en `doc.go`: ventana deslizante de ids de 13 B, uno de cada cuatro
reemplazado, limpieza de fondo frecuente). La meta del bloque A es ≤ 1,5 × los datos vivos.

| Valor | Disco del store / datos vivos (ids + valores) |
|---|---|
| 100 B | 1,65 × (pesan los 32 B por versión) |
| **256 B** | **1,48 ×** (dentro de la meta) |
| 1 KB | 1,40 × |

En una carga sin reemplazos (`RESULTADOS.md`, 100 millones de claves de 11 B con valores de 100 B) se midieron
~167 B por clave: 16,7 GB en total.

Además del store, cuenta con lo siguiente:

- **`wal/`**: los segmentos se preasignan enteros con ceros, y hay repuestos. Mientras rige el freno, el WAL sin
  reciclar llega hasta ~2 × `MAX_WAL_SIZE`, más los repuestos. Un respaldo en curso, el archivo continuo (`WAL_ARCHIVE`)
  y las ranuras retienen segmentos además de eso. La cota exacta del directorio está **a confirmar**.
- **`xact/`**: 2 bits por xid, en archivos de 32 MB (134.217.728 xids cada uno). El congelado los trunca. Queda acotado
  por lo que separa el horizonte del xid vivo más viejo, más dos archivos.
- **`data/`**: archivos de 1 GB por store. El archivo no se encoge: las páginas libres se reutilizan.
- **Versiones muertas**: quedan hasta que pasa la limpieza. Un snapshot o un lote largo las retiene (sección 4).
- **Respaldo a directorio**: `Backup` **copia** los datos (no usa enlaces), así que pide libre lo que ocupa la base.
  `BackupTo` hacia un flujo no necesita lugar local.

### Otras variables

- **`WAL_SEGMENT_SIZE`** (128 MB): el tamaño de cada segmento del WAL. Es la unidad del archivo continuo y del
  reciclado. Un segmento más chico archiva antes (y pesa menos cada `SwitchWAL`), a cambio de más archivos. Josefina
  abre su motor local con 4 MB.
- **`WAL_BUFFERS`** (1.024 KB): los registros sellados que esperan en memoria antes de escribirse en tandas de un solo
  `pwrite`. Antes, el `pwrite` por registro se llevaba el 73 % de la CPU de la carga masiva. Junto con los otros dos
  arreglos de la tarea 100, la carga masiva de josefina subió de 35.216 a ~75.000 documentos/s. Con 0, se hace una
  escritura por registro. No cambia la durabilidad: lo confirmado sigue siendo durable al responder.
- **`CHECKPOINT_TIMEOUT`** (300 s): el intervalo entre checkpoints por tiempo. Cada checkpoint reparte sus escrituras
  en el 90 % del intervalo. Más corto escribe más imágenes de página en el WAL; más largo no cambia la cota de `Open`,
  que fija `MAX_WAL_SIZE`.
- **`VACUUM_INTERVAL`** (60 s; 0 la apaga) y **`VACUUM_THRESHOLD`** (1.000 escrituras): la limpieza de fondo corre
  cada intervalo si hubo al menos ese número de escrituras desde la última vuelta. También corre cada medio archivo
  de estados de xids asignados (67 millones). Los valores están **a confirmar** (ver la decisión 56).
- **`LOCK_TIMEOUT`** (5 s): cuánto espera una escritura a otra transacción sobre el mismo id (`ErrConflict`), y cuánto
  espera un escritor frenado (`ErrThrottled`). También es lo que espera `Store(name)` a un lote que crea ese store.
- **`TRANSACTION_TIMEOUT`** (600 s; 0 sin tope): la duración máxima de un lote o de un `Snapshot()`. Al vencer, el lote
  se aborta (`ErrTimeout`) aunque nadie lo use. Para copias largas, usa `SnapshotFor(0)` (sin tope) y ciérralo.
- **`WAL_WRITER_DELAY`** (200 ms; 0 lo apaga): cada cuánto el escritor de fondo hace durable lo confirmado de forma
  diferida. Acota lo que un corte puede perder.
- **`SYNC_ON_WRITE`** (true): con `false`, **todas** las confirmaciones pasan a ser diferidas (ver la sección 3),
  salvo el commit de un lote con DDL y los registros de crear, vaciar o borrar un store, que siempre esperan su fsync.

### `Options` con `OpenWith` frente a las variables de entorno

Las variables se leen al abrir con `envar`. Ten en cuenta tres cosas:

- `envar` **guarda en caché el primer valor que lee** de cada variable. Cambiar el entorno del proceso entre dos
  `Open` no tiene efecto.
- `envar` carga el `.env` del directorio de trabajo (`godotenv/autoload`).
- Todos los motores abiertos con `Open` en un proceso comparten la configuración.

Para dar a cada motor su propia configuración, usa `OpenWith` (o `OpenReplicaWith`) con `Options`:

- Un campo en cero toma su variable, así que `Options{}` es lo mismo que `Open`.
- En los campos donde 0 apaga algo (`WALWriterDelay`, `VacuumInterval`, `TransactionTimeout`, `ArchiveMaxMB`,
  `SlotKeepMB`, `WALBuffersKB`), un valor negativo lo apaga.
- `SyncOnWrite` y `Archive` son punteros: `nil` toma la variable.

```go
off := false
e, err := store.OpenWith("/var/lib/miapp/db", store.Options{
	CacheMB:           512,             // STORE_CACHE_SIZE
	MaxWALMB:          1024,            // MAX_WAL_SIZE
	WALSegmentMB:      128,             // WAL_SEGMENT_SIZE
	CheckpointTimeout: 5 * time.Minute, // CHECKPOINT_TIMEOUT
	LockTimeout:       5 * time.Second, // LOCK_TIMEOUT
	VacuumInterval:    time.Minute,     // VACUUM_INTERVAL
	Archive:           &off,            // WAL_ARCHIVE
	SlotKeepMB:        4096,            // MAX_SLOT_WAL_KEEP_SIZE
})
```

Josefina abre dos motores por nodo:

- el de **datos**, con `CacheMB` de `JDB_STORE_CACHE_MB`;
- uno **local** con `CacheMB` 4, `WALSegmentMB` 4, `MaxWALMB` 64 y `WALBuffersKB` 64.

`OpenAt` y `OpenAtTime` no aceptan `Options`: usan las variables de entorno.

### Tabla de variables

| Variable | Por omisión | Campo de `Options` | Cuándo cambiarla |
|---|---|---|---|
| `STORE_CACHE_SIZE` | 128 (MB, mínimo 1) | `CacheMB` | Según el conjunto de trabajo y la RAM. La memoria es la caché + 8–11 % (con 512 MB o más). |
| `WAL_SEGMENT_SIZE` | 128 (MB, mínimo 1) | `WALSegmentMB` | Bájala para archivar o replicar segmentos más chicos, o en motores chicos. |
| `WAL_BUFFERS` | 1024 (KB; 0 = una escritura por registro) | `WALBuffersKB` (< 0 = sin búfer) | Casi nunca. Bájala en motores chicos (josefina local: 64). |
| `SYNC_ON_WRITE` | true | `SyncOnWrite` (`*bool`) | `false` solo si se acepta perder ~200 ms de confirmaciones ante un corte (sección 3). |
| `CHECKPOINT_TIMEOUT` | 300 (s, mínimo 1) | `CheckpointTimeout` | Rara vez. La cota de `Open` la fija `MAX_WAL_SIZE`. |
| `MAX_WAL_SIZE` | 1024 (MB) | `MaxWALMB` | Bájala si necesitas un `Open` más rápido tras una caída (~2,5 s por GB); súbela si los escritores se frenan (`PressureCause = wal`) y aceptas más tiempo de `Open`. |
| `LOCK_TIMEOUT` | 5 (s) | `LockTimeout` | Súbela si hay `ErrConflict` o `ErrThrottled` por esperas legítimas; bájala para fallar antes. |
| `TRANSACTION_TIMEOUT` | 600 (s; 0 = sin tope) | `TransactionTimeout` (< 0 = sin tope) | Mejor no apagarla: un lote olvidado retiene la limpieza. Para copias largas, `SnapshotFor(0)`. |
| `VACUUM_INTERVAL` | 60 (s; 0 = apagada) | `VacuumInterval` (< 0 = apagada) | Bájala con mucho reemplazo de claves (menos disco); apágala solo si llamas a `Vacuum()` tú. |
| `VACUUM_THRESHOLD` | 1000 (escrituras) | `VacuumThreshold` | Súbela en motores con poco cambio. |
| `WAL_WRITER_DELAY` | 200 (ms; 0 = apagado) | `WALWriterDelay` (< 0 = apagado) | Bájala para perder menos con escrituras diferidas. No la apagues si usas `Deferred` sin `WaitDurable`. |
| `WAL_ARCHIVE` | off (`on`, `true`, `1` o `yes` lo encienden) | `Archive` (`*bool`) | Enciéndela para PITR. Se fija al abrir. |
| `WAL_ARCHIVE_MAX_SIZE` | 0 (MB; 0 = sin tope: frena a los escritores) | `ArchiveMaxMB` (< 0 = sin tope) | Ponle tope si prefieres perder continuidad del archivo antes que frenar las escrituras. |
| `MAX_SLOT_WAL_KEEP_SIZE` | 4096 (MB; 0 = sin tope: frena) | `SlotKeepMB` (< 0 = sin tope) | Según el disco que aceptas dar a una réplica caída antes de invalidar su ranura. |

Constantes sin variable:

- archivos de datos de 1 GB;
- reparto del checkpoint en el 90 % de su intervalo;
- `FREEZE_MIN_AGE` = 0;
- `FREEZE_TABLE_AGE` = un archivo de estados (134.217.728 xids);
- freno del WAL en 2 × `MAX_WAL_SIZE`;
- freno de las escrituras diferidas en 8.192 registros sin fsync;
- espera de la réplica a sus snapshots: 30 s;
- memoria del búfer del flujo lógico: 16 MB por lector.

---

## 3. Durabilidad

### Durables frente a diferidas

| Forma | Cuándo vuelve | Qué pierde un corte |
|---|---|---|
| `Put`, `Insert`, `Update`, `Delete`, `Remove` sueltos; `Batch.Commit` (con `SYNC_ON_WRITE=true`) | tras el fsync del WAL (un fsync compartido por todos los que confirman a la vez: *group commit*) | nada de lo que volvió bien |
| `Store.Deferred()` (`Put`, `Remove`, `Insert`, `Update`, `Delete`) y `Batch.CommitDeferred` | sin esperar el fsync: lo confirmado se ve enseguida | lo confirmado después del último fsync: siempre entero (un lote está completo o no está) y en el orden del WAL |
| todo, con `SYNC_ON_WRITE=false` | como las diferidas | ídem |

Tres mecanismos acotan lo que se pierde con las diferidas:

- **`Engine.WaitDurable()`** espera a que todo lo confirmado antes de llamar, en cualquier store, esté en disco.
- **El escritor de fondo del WAL** hace un `WaitDurable` cada `WAL_WRITER_DELAY` (200 ms). Lo que un corte puede perder
  queda en esa pausa más un fsync.
- **El freno de las diferidas**: con más de 8.192 registros sin fsync, una confirmación diferida espera el fsync hasta
  su lsn − 8.192.

Una confirmación durable que sigue en el WAL a una diferida también hace durable a la diferida: lo durable es siempre
un prefijo del WAL.

```go
// Carga masiva: confirmar sin esperar el fsync y hacer durable todo al final.
d := s.Deferred()
for id, v := range rows {
	if err := d.Put(id, v); err != nil {
		return err
	}
}
return e.WaitDurable() // desde aquí, todo lo anterior sobrevive a un corte

// Un lote diferido.
b := e.NewBatch()
if err := s.PutIn(b, "a", []byte("1")); err != nil {
	b.Rollback()
	return err
}
if err := b.CommitDeferred(); err != nil {
	return err
}
return e.WaitDurable()
```

Cuidados:

- Una lectura que ve algo diferido no espera su fsync. Si un corte pierde esa versión, lo que contestó deja de valer.
- La versión (`GetVersion`, el lsn) de algo diferido solo vale hasta el próximo corte o hasta `WaitDurable`. Tras un
  corte, ese lsn puede volver a usarse para otra escritura.
- Los commits de lotes con DDL (`StoreIn`, `DropStoreIn`) y los registros de crear, vaciar y borrar un store **siempre**
  esperan su fsync, también con `CommitDeferred` y con `SYNC_ON_WRITE=false`.

### fsync en cada sistema

| SO | Qué hace `Sync` | Costo medido |
|---|---|---|
| macOS | `F_FULLFSYNC`: vacía también la caché del SSD (el fsync simple de macOS no lo hace) | ~3–4 ms por fsync. Pone el techo de las escrituras durables: 340/s con un escritor, 10.216/s con 64 (group commit) |
| Linux | `fdatasync` | En la VM (virtio) es ~35 veces más barato que en macOS: de 12.465/s con un escritor a 21.321/s con 64, y el techo lo pone la CPU. **Sobre un NVMe real no está medido.** |
| otros (windows) | `(*os.File).Sync` | sin medir |

El directorio también recibe su fsync tras crear, renombrar y borrar archivos. Un fsync fallido nunca se reintenta: el
motor queda inutilizable (sección 8), porque tras un fsync fallido el SO puede haber descartado las páginas.

Con escrituras diferidas, el Put suelto llega a 82.000–167.000/s en macOS (`RESULTADOS.md`, `storebench -rate`).

---

## 4. Operación diaria

### Métricas

`Engine.Stats()` es una foto barata (solo atómicos, sin recorrer el árbol). Los contadores son desde `Open`: para
tasas, resta dos fotos. `OnStats(interval, fn)` llama a `fn` cada `interval` sin candados tomados y devuelve `stop`.
`fn` no debe llamar a `Close`, porque `Close` espera a la llamada en curso.

```go
stop := e.OnStats(10*time.Second, func(st store.Stats) {
	if st.ReadOnly {
		log.Printf("ALERTA: motor de solo lectura por disco lleno: liberar lugar, cerrar y abrir")
	}
	if st.Pressure >= 0.7 {
		log.Printf("presión %.2f (%s)", st.Pressure, st.PressureCause)
	}
	if st.SlotsInvalid > 0 || st.ArchiveLost > 0 {
		log.Printf("ranuras invalidadas %d, pérdidas del archivo %d", st.SlotsInvalid, st.ArchiveLost)
	}
	log.Printf("WAL por rehacer %d MB, fsync p99 %v, commit p99 %v, horizonte atrasado %d xids",
		st.WALSinceCheckpoint>>20, st.FsyncLatency.P99, st.CommitLatency.P99, st.NextXID-st.OldestXID)
})
defer stop()
```

`Engine.Pressure()` devuelve un `float64`: 0 = sin presión, ≥ 1 = ya frena. Es el máximo de varias razones, y la mayor
va en `Stats.PressureCause`:

| Causa | Razón | Qué frena al llegar a 1 |
|---|---|---|
| `wal` | WAL sin reciclar / (2 × `MAX_WAL_SIZE`) | los escritores esperan un checkpoint y, tras `LOCK_TIMEOUT`, `ErrThrottled` |
| `archive` | WAL sin archivar (sin tope) / el mismo tope | ídem, esperando al archivador |
| `slot` | WAL retenido por la ranura más atrasada / `MAX_SLOT_WAL_KEEP_SIZE` (o / el tope del freno, sin tope) | con tope, la ranura se invalida; sin tope, frena como `archive` |
| `deferred` | registros sin fsync / 8.192 | la confirmación diferida espera un fsync |
| `dirty` | páginas sucias / búferes de la caché | nadie: un escritor que necesita un búfer escribe antes una página sucia |
| `read_only` | el motor quedó de solo lectura por disco lleno (= 1) | toda escritura: `ErrDiskFull` + `ErrFailed` |
| `failed` | el motor quedó inutilizable por otro fallo (= 1) | toda escritura: `ErrFailed` |

`Pressure()` solo lee atómicos, así que se puede llamar antes de cada escritura masiva. Josefina admite las cargas
masivas libremente por debajo de 0,40 (`JDB_BULK_RELAX`). Por encima, una carga masiva espera turno y, mientras la
presión pase de 0,70 (`JDB_BULK_PRESSURE`), espera hasta 30 s y después contesta 503.

`Engine.DirUsage()` mide el disco: `Total`, `Data`, `WAL`, `Xact`, `Control` y `Stores` por nombre. No lee páginas,
pero no es una foto atómica. `Store.DirUsage()` da la parte de un store.

```go
u, err := e.DirUsage()
if err != nil {
	return err
}
fmt.Printf("total %d MB: data %d, wal %d, xact %d, control %d\n",
	u.Total>>20, u.Data>>20, u.WAL>>20, u.Xact>>20, u.Control>>20)
for name, bytes := range u.Stores {
	fmt.Printf("  %s %d MB\n", name, bytes>>20)
}
```

#### Umbrales sugeridos

Los umbrales sin una medición que los respalde están marcados como «sugerido».

| Señal | Umbral | Qué hacer |
|---|---|---|
| `Stats.ReadOnly` | `true` | Urgente: liberar disco, `Close` y `Open` (sección 9). |
| `Pressure()` | ≥ 0,40 aviso, ≥ 0,70 frenar cargas, ≥ 1 ya frena (los dos primeros son los de josefina) | Mirar `PressureCause` (sección 9). |
| `WALSinceCheckpoint` | sugerido: aviso > `MAX_WAL_SIZE` (en régimen ronda 539 MB–1,13 GB con 1 GB) | Los checkpoints no dan abasto: disco lento, o `MAX_WAL_SIZE` chico para la carga. Desde la tarea 82 cuenta desde el último checkpoint terminado, que es lo que rehace `Open`. |
| `LastCheckpoint` | sugerido: hace más de 2 × `CHECKPOINT_TIMEOUT` | Un checkpoint no termina (disco lleno en los pasos 2 o 3, o disco lento). Ojo: `LastCheckpointTook` de uno de fondo ronda el 90 % del intervalo, y eso es normal. |
| `Throttled`, `DiskFull` | sugerido: cualquier aumento | Rechazos sin efecto: hubo freno o falta de lugar. |
| `FsyncLatency.P99` | sugerido: muy por encima de la línea base (macOS ~4 ms con `F_FULLFSYNC`) | Disco degradado o compartido. |
| `CommitLatency.P99` | sugerido: muy por encima de la línea base | Group commit saturado o disco lento. |
| `NextXID − OldestXID` | sugerido: que crezca sin bajar | Un lote o snapshot largo retiene el horizonte (abajo). |
| `OpenBatches`, `OpenSnapshots` | sugerido: más que la concurrencia esperada | Lotes o snapshots olvidados. La edad de un snapshot no se expone (**a confirmar**). |
| `FrozenAge` | sugerido: > 134.217.728 (un archivo de estados) | El congelado no avanza: un snapshot largo, o la limpieza apagada. |
| `ArchivePending`, `ArchiveRetained` | sugerido: que crezcan entre vueltas del archivador | El archivador está caído o atrasado. |
| `ArchiveLost`, `ArchiveGap` | > 0 | Se perdió continuidad: hace falta un `Backup` nuevo para PITR (sección 5). |
| `SlotsInvalid`, `SlotsInvalidated` | > 0 | Rehacer el consumidor de esa ranura (sección 6). |
| `SlotRetained` | sugerido: > 50 % de `MAX_SLOT_WAL_KEEP_SIZE` | Una réplica o un consumidor lógico va atrasado. |
| `RecoveryConflicts` (réplica) | sugerido: cualquier aumento | Snapshots de la réplica cancelados (`ErrRecoveryConflict`): son demasiado largos. |
| `DirUsage().Data` / datos vivos | sugerido: > 1,5 × con valores de 256 B | La limpieza no alcanza, o algo retiene el horizonte. |
| `CacheHits / (CacheHits + CacheMisses)` | sin umbral medido | Si baja de forma sostenida, subir `STORE_CACHE_SIZE`. |

### Checkpoints

El checkpointer de fondo mira cada segundo si toca uno: cada `CHECKPOINT_TIMEOUT` o al llegar a `MAX_WAL_SIZE / 1,9`
de WAL. Reparte la escritura de páginas en el 90 % del intervalo. Al terminar, recicla los segmentos anteriores al
punto de rehacer, salvo los que retienen un respaldo, el archivo continuo o las ranuras.

`Engine.Checkpoint()` fuerza uno ahora, sin repartir. No hace falta antes de un `Backup`, que hace el suyo.

Si un checkpoint falla por disco lleno en sus pasos 2 (archivo de estados) o 3 (catálogo), aborta sin daño y se
reintenta en la próxima vuelta. Cualquier otro fallo deja el motor inutilizable.

`Close` hace un checkpoint final. Si un checkpoint de fondo está repartiendo sus escrituras y a la vez hay un
`Checkpoint()` del usuario esperando, `Close` puede tardar lo que falte del checkpoint de fondo: medido 4,8 s, y hasta
~4,5 min en producción (hallazgo N3 de `doc/review-store-2026-11-20.md`, **pendiente**). Al detener el servicio,
deja margen para el `Close`.

### Limpieza (vacuum), congelado y snapshots largos

La limpieza quita las versiones que ningún lote ni snapshot abierto puede ver, libera sus cadenas de desborde, devuelve
a la lista de libres las hojas vacías y los padres sin hijos, congela lo confirmado y pone el mapa de visibilidad. Corre
de fondo (`VACUUM_INTERVAL`/`VACUUM_THRESHOLD`), y `Engine.Vacuum()` la fuerza. Puede correr con lecturas y escrituras.

El **horizonte** es el xid en curso más viejo, o el `Xmin` del lote o snapshot abierto más viejo. Un lote o snapshot
que sigue abierto (también un `SnapshotFor(0)` olvidado o un recorrido detenido a medias) hace lo siguiente:

- **retiene las versiones muertas**: el disco crece con cada reemplazo;
- **retiene las páginas borradas**: no se reciclan mientras su `safeXID` no quede por debajo del horizonte;
- **frena el congelado**: el archivo de estados no se trunca y crece `FrozenAge`;
- **hace que `Empty` devuelva `ErrBusy`**, y que `DropStoreIn` espere.

En la práctica:

- Cierra siempre los snapshots con `defer v.Close()`. Los recorridos sueltos (`ForEach`, `Keys`…) cierran el suyo solos.
- Para copias largas, usa `SnapshotFor(0)` y vigila `OpenSnapshots` y `NextXID − OldestXID`.
- En una réplica, un snapshot largo atrasa al aplicador hasta 30 s y después se cancela (`ErrRecoveryConflict`).

---

## 5. Respaldo y restauración

### Respaldo en caliente: `Backup` y `BackupTo`

`Backup(dir)` copia el motor a un directorio vacío o inexistente sin detener las escrituras, los lotes ni el DDL, como
`pg_basebackup`. Hace un checkpoint (el inicio), copia el catálogo, los datos y los estados, escribe el registro de fin
de respaldo y copia el WAL hasta él. Mientras dura, el WAL del motor no se recicla desde el inicio: el disco del WAL
crece con lo que se escriba durante la copia.

```go
info, err := e.Backup("/backups/db-2026-11-20")
if err != nil {
	return err // ErrBackupDir (destino no vacío), ErrDiskFull (en el destino: el motor sigue)
}
log.Printf("respaldo: inicio %d, fin %d, %d archivos, %d MB, %v",
	info.StartLSN, info.EndLSN, info.Files, info.Bytes>>20, info.Took)
```

`BackupTo(sink)` entrega los mismos archivos a un `BackupSink`, uno por uno, enteros y con su largo, como pide un tar.
Puede ir comprimido, cifrado o subido a S3. No necesita lugar local.

```go
// tarSink entrega cada archivo del respaldo como una entrada de tar.
type tarSink struct{ tw *tar.Writer }

type tarEntry struct{ tw *tar.Writer }

func (t tarEntry) Write(p []byte) (int, error) { return t.tw.Write(p) }
func (t tarEntry) Close() error                { return nil }

func (s tarSink) File(name string, size int64) (io.WriteCloser, error) {
	hdr := &tar.Header{Name: name, Mode: 0o600, Size: size, ModTime: time.Now(), Typeflag: tar.TypeReg}
	if err := s.tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	return tarEntry{s.tw}, nil
}

func respaldoATar(e *store.Engine, path string) (store.BackupInfo, error) {
	f, err := os.Create(path)
	if err != nil {
		return store.BackupInfo{}, err
	}
	defer f.Close()
	tw := tar.NewWriter(f)
	info, err := e.BackupTo(tarSink{tw})
	if err != nil {
		return info, err
	}
	if err := tw.Close(); err != nil {
		return info, err
	}
	return info, f.Sync()
}
```

El marcador `backup_label` sale dos veces: «en curso» al principio y el final al terminar. Al extraer, el segundo pisa
al primero. Una copia cortada no abre (`ErrBackupIncomplete`).

Guarda `BackupInfo` (`StartLSN`, `EndLSN`) y el `Count()` de cada store junto con el respaldo: sirven para probarlo y
para PITR. Guarda también la versión de `et` con la que se hizo (sección 10).

### Restaurar

Restaurar es dejar los archivos en un directorio y abrirlo:

```bash
mkdir -p /var/lib/miapp/db-restaurada
tar -xf /backups/db-2026-11-20.tar -C /var/lib/miapp/db-restaurada
```

Después, `store.Open("/var/lib/miapp/db-restaurada")`. `Open` hace tres cosas:

1. Comprueba, sin escribir nada, que el WAL llega hasta el fin del respaldo. Si no llega, devuelve
   `ErrBackupIncomplete` y no toca la copia.
2. Rehace el WAL como tras un corte.
3. Hace un checkpoint y borra `backup_label`. Desde ahí, la copia es un motor más.

**Ojo:** `OpenAt` y `OpenAtTime` exigen `backup_label`. Si vas a necesitar PITR, no abras con `Open` la única copia
del respaldo: trabaja sobre una copia de ella.

### Archivo continuo: `WAL_ARCHIVE`, `PendingArchive`, `SetArchived` y `SwitchWAL`

Con `WAL_ARCHIVE=on` (o `Options.Archive`), ningún segmento sellado se recicla hasta que el archivador lo marque como
copiado. El archivador es tuyo, como el `archive_command` de PostgreSQL:

- `PendingArchive()` da hasta 256 segmentos sellados sin archivar, en orden. Vuelve a pedirlo si hay más.
- Basta copiar los `Size` primeros bytes de cada segmento, con su nombre (`WALSegment.Name`).
- `SetArchived(seg)` es durable al volver. Llámalo en orden, después de que la copia sea durable en el destino.
- `SwitchWAL()` sella el segmento activo aunque no esté lleno, para archivarlo ya. Fija la precisión del PITR.
- El archivador tiene que ser idempotente: un corte entre la copia y `SetArchived` hace archivar de nuevo.

```go
func archivar(e *store.Engine, destino string) error {
	if _, err := e.SwitchWAL(); err != nil { // opcional: sella el activo para archivarlo ya
		return err
	}
	segs, err := e.PendingArchive()
	if err != nil {
		return err
	}
	for _, seg := range segs {
		if err := copiarPrefijo(seg.Path, filepath.Join(destino, seg.Name), seg.Size); err != nil {
			return err
		}
		if err := e.SetArchived(seg.Segment); err != nil {
			return err
		}
	}
	return nil
}

// copiarPrefijo copia los primeros n bytes de src a dst con fsync (idempotente: pisa lo que había).
func copiarPrefijo(src, dst string, n int64) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.CopyN(out, in, n); err != nil {
		out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}
```

Si el archivador se atrasa, el resultado depende del tope:

- **Sin tope** (`WAL_ARCHIVE_MAX_SIZE=0`, por omisión): lo retenido suma al freno. Pasado 2 × `MAX_WAL_SIZE`, los
  escritores esperan y reciben `ErrThrottled` (`PressureCause = archive`). Nunca se pierde continuidad, pero un
  archivador caído detiene las escrituras.
- **Con tope**: no frena. En el checkpoint, si lo retenido pasa el tope, el motor suelta lo anterior y anota la
  pérdida (`Stats.ArchiveGap`, `Stats.ArchiveLost`). Un respaldo anterior a la pérdida ya no llega más allá: haz uno
  nuevo.

Abrir con el archivo apagado un motor que archivaba deja reciclar segmentos sin archivar. Al volver a encenderlo se
anota la pérdida. Abre siempre con el mismo `WAL_ARCHIVE`, **también con `cmd/storecheck`** (sección 7).

`Backup` no copia `archive_status`: la copia restaurada empieza su propio archivo, y conviene mandarlo a otro destino.

### Recuperación a un punto en el tiempo: `OpenAt` y `OpenAtTime`

La base es una copia de `Backup` (con su `backup_label`) más los segmentos archivados desde antes de su fin.

1. Restaura la copia en un directorio nuevo (tar o copia del directorio), sin abrirla.
2. Copia a `wal/` de la copia los segmentos archivados con su nombre, desde el primero que ya tiene la copia. El
   último de la copia lo cortó el respaldo en su fin: la versión archivada lo reemplaza.
3. Abre, con una de estas tres llamadas:
   - `store.Open(dir)` rehace hasta el final de lo archivado.
   - `store.OpenAt(dir, lsn)` rehace hasta el registro `lsn` inclusive. El lsn tiene que ser ≥ `BackupInfo.EndLSN` y
     ≥ el checkpoint del control. Sirven `Stats.LSN`, `BackupInfo.EndLSN` y `WALSegment.EndLSN`.
   - `store.OpenAtTime(dir, t)` rehace hasta el último registro antes del primero con hora posterior a `t`. Llevan hora
     los commits de lote, los checkpoints y el fin del respaldo; las escrituras sueltas van con lo que las rodea. Si
     nada en el WAL es posterior a `t`, rehace todo.

Si el objetivo no se alcanza (sin `backup_label`, antes del fin del respaldo o después del final del WAL), devuelven
`ErrRecoveryTarget` y la copia queda como estaba. Lo confirmado después del objetivo no está, y lo que estaba a medias
queda deshecho, como tras un corte.

```go
e, err := store.OpenAtTime(copia, hasta)
if errors.Is(err, store.ErrRecoveryTarget) {
	return nil, fmt.Errorf("objetivo inalcanzable (la copia quedó como estaba): %w", err)
}
```

Pendiente (**a confirmar**, tarea 66): el objetivo por xid, las líneas de tiempo, un comando archivador y la
compresión de los segmentos archivados.

### Cómo probar que un respaldo sirve

Un respaldo que nunca se restauró no está probado. No hay `VerifyBackup`: `Open` restaura y `Verify` revisa.

```go
func probarRespaldo(copia string, esperado map[string]int64) error {
	e, err := store.Open(copia) // restaura: rehace el WAL hasta EndLSN
	if err != nil {
		return err // ErrBackupIncomplete: la copia no sirve
	}
	defer e.Close()
	rep, err := e.Verify()
	if err != nil {
		return err
	}
	if !rep.Clean() {
		return fmt.Errorf("la copia no verifica:\n%s", rep.String())
	}
	for name, n := range esperado {
		s, err := e.Store(name)
		if err != nil {
			return err
		}
		if got := s.Count(); got != n {
			return fmt.Errorf("store %s: Count %d, se esperaba %d", name, got, n)
		}
	}
	return nil
}
```

`esperado` es el `Count()` de cada store anotado al terminar el respaldo. Con escrituras durante la copia, el valor
exacto en `EndLSN` puede diferir; en ese caso, compara contra un motor quieto o acepta la diferencia de lo escrito
durante la copia.

### Procedimiento paso a paso

**Respaldo diario con PITR:**

1. Abre el motor con `WAL_ARCHIVE=on` y un archivador que corra cada pocos segundos (`SwitchWAL`, `PendingArchive`,
   copia, `SetArchived`).
2. Haz un `BackupTo` (o `Backup`) a la hora de menos carga. Guarda el tar, el `BackupInfo`, los `Count` y la versión
   de `et`.
3. Restaura ese respaldo en otra máquina o directorio y córrele `probarRespaldo`. Hazlo al menos tras cada cambio de
   versión de `et`.
4. Vigila `ArchivePending`, `ArchiveLost` y `ArchiveGap`. Si `ArchiveLost` sube, vuelve al paso 2.
5. Conserva los segmentos archivados desde el inicio del respaldo más viejo que quieras poder restaurar.

**Restaurar tras un desastre:**

1. Detén el servicio. Si el motor viejo sigue abierto, ciérralo: `LOCK` impide abrir el mismo directorio dos veces.
2. Extrae el último respaldo bueno en un directorio nuevo. No reutilices el directorio dañado.
3. Para PITR, copia a `wal/` los segmentos archivados (paso 2 de la sección anterior).
4. Abre con `Open`, `OpenAt` u `OpenAtTime`.
5. Corre `Verify()` y compara los `Count`.
6. Apunta el servicio al directorio restaurado. El archivo continuo empieza de nuevo: haz pronto un respaldo nuevo.

---

## 6. Réplicas

El transporte no es del motor: josefina lleva los `WalEntry` y los `Change` por su hub TCP.

### Réplica física

La réplica tiene el WAL idéntico byte a byte, es de solo lectura (`ErrReadOnly` en las escrituras, `NewBatch`,
`Vacuum`, `Repair` y `Backup`) y sus checkpoints son *restartpoints* (puntos de reinicio).

1. **En el primario**, crea la ranura **antes** del respaldo, así queda retenido todo lo que la réplica necesita:

   ```go
   if _, err := primario.CreateSlot("r1"); err != nil && !errors.Is(err, store.ErrSlotExists) {
   	return err
   }
   _, err := primario.Backup("/srv/replica1") // se lleva al equipo de la réplica
   ```

2. **En la réplica**, abre la copia como réplica: `replica, err := store.OpenReplica("/srv/replica1")`. Un directorio
   sin motor da `ErrNotReplica`.
3. **Bucle de aplicación** (aquí los dos motores están en el mismo proceso):

   ```go
   func replicar(primario, replica *store.Engine, parar <-chan struct{}) error {
   	r, err := primario.WalSince(replica.ReplayLSN())
   	if err != nil {
   		return err
   	}
   	for {
   		select {
   		case <-parar:
   			return nil
   		default:
   		}
   		entries, err := r.Next(1 << 20)
   		if errors.Is(err, store.ErrWalRemoved) {
   			return fmt.Errorf("la réplica quedó atrás: hace falta un Backup nuevo: %w", err)
   		}
   		if err != nil {
   			return err
   		}
   		if len(entries) == 0 {
   			time.Sleep(50 * time.Millisecond)
   			continue
   		}
   		if err := replica.ApplyWal(entries); errors.Is(err, store.ErrWalGap) {
   			if r, err = primario.WalSince(replica.ReplayLSN()); err != nil { // pedir de nuevo desde ReplayLSN
   				return err
   			}
   			continue
   		} else if err != nil {
   			return err
   		}
   		if err := replica.WaitDurable(); err != nil {
   			return err
   		}
   		if err := primario.AdvanceSlot("r1", replica.ReplayLSN()); err != nil {
   			return err
   		}
   	}
   }
   ```

Reglas del bucle:

- `WalSince` solo entrega WAL ya durable en el primario. `ApplyWal` salta lo ya aplicado (reenviar es seguro) y no
  hace fsync.
- Confirma en la ranura el `ReplayLSN` **después** de `WaitDurable` en la réplica.
- Las lecturas, los recorridos, los snapshots, `Verify` y `Checkpoint` (un *restartpoint*) andan en la réplica.
- Un snapshot de la réplica ve lo confirmado al tomarlo. Si el aplicador necesita rehacer algo que ese snapshot no debe
  ver, lo espera hasta 30 s y después lo cancela con `ErrRecoveryConflict`. Mantén cortos los snapshots de la réplica.
- Un corte de la réplica se recupera reabriéndola con `OpenReplica`: sigue desde lo durable.

### Promover: `Promote`

`replica.Promote()` vuelve primario a la réplica sin cerrarla:

- deja de aplicar WAL (`ApplyWal` da `ErrNotReplica`);
- aborta lo que seguía en curso en el primario;
- hace el checkpoint de fin de recuperación;
- desde ahí acepta escrituras; las lecturas y los snapshots abiertos siguen.

Cerrar la réplica y abrirla con `store.Open` hace lo mismo. Los stores abiertos antes de promover no ponen marcas de
visibilidad hasta reabrir el motor (solo afecta al rendimiento de lectura).

Para conmutar (*failover*):

1. Asegúrate de que el primario viejo esté detenido de verdad: no hay protección contra dos primarios.
2. Elige la réplica con el mayor `ReplayLSN()`.
3. Llama a `Promote()`.
4. Apunta los clientes a ella.
5. El primario viejo **no puede seguirla** (no hay líneas de tiempo, y su WAL puede tener lo que la réplica no
   recibió): vuelve como réplica desde un `Backup` nuevo del nuevo primario.

**Nunca abras con `store.Open` (ni con `cmd/storecheck`) el directorio de una réplica que quieres seguir usando como
réplica: eso la promueve.**

### Flujo lógico: ranuras, `ChangesWith` y `ChangesFrom`

El flujo entrega, por transacción confirmada y en orden de confirmación, cada cambio: `put` (con el valor nuevo),
`delete`, `create`, `drop` o `empty`, con `TxInfo{XID, CommitLSN, Origin}`.

- **Con ranura** (`CreateLogicalSlot`, después `Changes` o `ChangesWith`): la ranura retiene el WAL hasta que
  `Confirm` lo libera. `CreateLogicalSlot` espera, hasta `TRANSACTION_TIMEOUT`, a que terminen las transacciones en
  curso (si no, `ErrBusy`). No anda en una réplica (`ErrReadOnly`).
- **Sin ranura** (`ChangesFrom(pos, f)`, desde `ChangeHead()` o desde un `Pos()` guardado): no retiene nada (si el WAL
  ya se recicló, `ErrWalRemoved`) y anda también en una réplica. Quien lee guarda `Pos()`. `AdvanceLogicalSlot` lleva
  una ranura a una posición sin abrir su lector (para repartir un flujo entre varios consumidores y avanzar al mínimo).
- **Filtros** (`ChangeFilter`): `Store` por cambio, `ID` por cambio (antes de armar un valor en desborde) y `Origin`
  por transacción. `Batch.SetOrigin(n)` marca un lote. Así, un nodo que aplica cambios de otro no los vuelve a
  publicar.

```go
if _, err := e.CreateLogicalSlot("eventos"); err != nil && !errors.Is(err, store.ErrSlotExists) {
	return err
}
r, err := e.ChangesWith("eventos", store.ChangeFilter{
	Store:  func(name string) bool { return name == "users" },
	Origin: func(origin uint16) bool { return origin == 0 }, // solo lo escrito en este motor
})
if err != nil {
	return err
}
defer r.Close()
for {
	n, err := r.Next(100, func(tx store.TxInfo, c store.Change) error {
		fmt.Println(tx.CommitLSN, c.Op, c.Store, c.ID, len(c.Data))
		return nil
	})
	if err != nil {
		return err // ErrWalRemoved: la ranura se invalidó; crear otra y empezar de nuevo
	}
	if n == 0 {
		time.Sleep(100 * time.Millisecond)
		continue
	}
	if err := r.Confirm(); err != nil {
		return err
	}
}
```

Cuidados:

- **Entrega al menos una vez.** Los avances de una ranura se guardan en el checkpoint, así que tras un corte del
  primario se puede repetir lo posterior. Descarta los repetidos por `TxInfo.CommitLSN`, que crece de una transacción
  a la siguiente.
- Si `fn` devuelve un error, el lector queda inservible: ciérralo y abre otro.
- `fn` corre con el motor tomado en lectura. No llames a `Close` desde `fn`, y evita leer del motor dentro de `fn`
  mientras otro hilo pueda estar cerrándolo: un `Read` ahí puede colgarse detrás del `Close` (hallazgo R3 de la
  revisión, **pendiente**). Copia lo que necesites y procésalo fuera de `fn`.
- Cada lector usa hasta 16 MB en memoria. Una transacción más grande se derrama a `replslot/` y el archivo se borra
  solo.

### El tope `MAX_SLOT_WAL_KEEP_SIZE` y la ranura invalidada

`Slots()` da por ranura `RestartLSN`, `ConfirmedLSN`, `Retained` (bytes) y `Lag` (registros sin confirmar).

- **Con tope** (4.096 MB por omisión): en el checkpoint, una ranura que retiene más se **invalida**. Suelta su
  retención, queda marcada en el archivo `slots`, y suben `Stats.SlotsInvalid` y `Stats.SlotsInvalidated`. El motor
  sigue sano. `AdvanceSlot`, `WalSince` y `Changes` de ella dan `ErrWalRemoved`.
- **Sin tope** (0): la ranura más atrasada frena a los escritores, como el archivo sin tope (`PressureCause = slot`).

Qué hacer con una ranura invalidada:

- **Réplica física:** `DropSlot` y `CreateSlot` en el primario, un `Backup` nuevo, y `OpenReplica` sobre la copia
  nueva (no sobre el directorio viejo).
- **Consumidor lógico:** `DropSlot`, `CreateLogicalSlot`, y una copia completa del estado (un `SnapshotFor(0)` con
  `ChangeHead()` para la posición y `ForEachRangeAt` para los datos). Después, el flujo desde esa posición.
- Una ranura que ya no se usa también retiene WAL: bórrala con `DropSlot`.

---

## 7. Verificar y reparar

`Engine.Verify()` revisa con el motor abierto y no cambia nada. Frena solo las operaciones del store que revisa
mientras dura. Revisa:

- el control y el catálogo;
- los archivos de `data/` sin store;
- de cada store: el árbol, las cadenas de desborde, la lista de libres, el mapa de visibilidad, el congelado y `Count`.

`Engine.Repair()` hace el mismo recorrido y arregla lo reparable, con WAL: un corte en medio deja un motor que abre.

Su error es solo `ErrClosed` o `ErrFailed`. El daño va en el `Report`: los totales, hasta 1.000 hallazgos con
detalle, `Total`, `Repaired`, `Irreparable`, `Clean()`, `Pending()` y `String()`.

Medido en un M5 sobre disco real: 97 MB (1 millón de ids) en 119 ms.

```go
func revisar(e *store.Engine, reparar bool) error {
	check := e.Verify
	if reparar {
		check = e.Repair
	}
	rep, err := check()
	if err != nil {
		return err // solo ErrClosed o ErrFailed: el daño va en el informe
	}
	fmt.Print(rep.String())
	for _, f := range rep.Findings {
		if !f.Repairable {
			fmt.Printf("irreparable: %s %s %s bloque %d: %s\n", f.Kind, f.Store, f.File, f.Block, f.Desc)
		}
	}
	if rep.Pending() > 0 {
		return fmt.Errorf("%d hallazgos pendientes (%d irreparables)", rep.Pending(), rep.Irreparable)
	}
	return nil
}
```

### `cmd/storecheck`

```bash
go run ./cmd/storecheck -dir /var/lib/miapp/db            # Verify
go run ./cmd/storecheck -dir /var/lib/miapp/db -repair    # Repair
echo $?   # 0 = nada pendiente, 1 = hallazgos sin arreglar, 2 = no pudo revisar
```

`storecheck` abre el motor con `store.Open`, recuperación incluida. Eso implica tres cosas:

- **El motor tiene que estar cerrado.** Si no, `Open` devuelve `ErrLocked` (salida 2). Para revisar un motor en uso,
  llama a `Verify()` desde el proceso que lo tiene abierto.
- **Corre con las mismas variables que el servicio**, sobre todo `WAL_ARCHIVE`. Si no, el checkpoint de fin de
  recuperación y el de `Close` reciclan segmentos sin archivar.
- **No lo corras sobre una réplica** (la promueve) **ni sobre la única copia de un respaldo** (la restaura y borra
  `backup_label`, que `OpenAt` necesita). Para una réplica, usa `Verify()` en su proceso. Para un respaldo, trabaja
  sobre una copia.

### Qué arregla y qué no

| Hallazgo (`FindingKind`) | ¿Repair? | Qué hace o qué hacer |
|---|---|---|
| `orphan-file`: archivos de datos de un número sin store | sí | Los borra. Quedan tras un `DropStore`/`Empty` cortado o un respaldo interrumpido. |
| `lost-page`: páginas sin dueño | sí (si en el store no hay nada que haga dudar de quién es cada página) | Vuelven a la lista de libres. Normal tras un corte (una extensión sin su registro). |
| `orphan-chain`: páginas de desborde sin versión | sí (ídem) | Vuelven a la lista de libres. Normal tras un corte. |
| `visibility`: bit del mapa mal puesto | sí | Lo borra (siempre es seguro). |
| `count`: `Count` no coincide con el recorrido | sí | Lo corrige con una confirmación propia. |
| `half-dead`: borrado de páginas cortado a medias | sí | Lo termina (también lo hace la próxima limpieza). |
| `unknown-file`: un archivo en `data/` con un nombre que no es de datos | no | Se informa sin tocarlo: revisa quién lo dejó ahí. |
| `no-meta`: store sin página meta | no | `Store(name)` lo crea vacío al abrirlo. Si tenía datos, restaura. |
| `tree`: el árbol no pasa la verificación o una página no se lee | no | El resto del store no es confiable. Restaura el respaldo (o PITR). |
| `chain`: cadena de desborde viva rota | no | Ese valor se perdió: `Read` del id da error. Restaura, o reescribe el id si tienes el valor en otro lado. |
| `double-page`: página con dos dueños | no | Restaura. |
| `free-list`: la lista de libres apunta a una página no libre o tiene un bucle | no | Restaura. |
| `frozen`: versión sin congelar por debajo del frozenXID | no | Restaura. |
| `status`: el archivo de estados no da el estado de una versión | no | Restaura. |
| `control` o `catalog`: no se leen o no coinciden | no | `Open` ya falla si no se leen. Restaura. |
| el WAL | — | `Open` lo valida: corta la cola rota del segmento activo y falla si el daño está en un segmento sellado. |

**Ante lo irreparable:**

1. No sigas escribiendo en ese store.
2. Guarda una copia del directorio tal cual, para el análisis.
3. Restaura el último respaldo bueno y, si hay archivo continuo, haz PITR hasta antes del daño.
4. Si no hay respaldo, lo que no depende del daño sigue legible: copia los datos sanos a un motor nuevo con un
   recorrido sobre un snapshot.

Tras un corte, `test/powercut` acepta páginas perdidas, cadenas huérfanas y archivos huérfanos, los repara con
`Repair` y exige que el `Verify` siguiente salga limpio. Ese es el comportamiento esperado.

---

## 8. Qué hacer ante cada error

Compara siempre con `errors.Is`. Los errores pueden venir envueltos: un disco lleno irreversible trae `ErrFailed` **y**
`ErrDiskFull` a la vez, así que mira `ErrFailed` primero.

| Error | Qué significa | ¿Se reintenta? | Qué hace el operador | ¿Reabrir? |
|---|---|---|---|---|
| `ErrLocked` | otro motor (de este u otro proceso) tiene el directorio | no | Busca el otro proceso; no borres `LOCK` a mano con el otro vivo. | — |
| `ErrClosed` | el motor ya se cerró | no | Error de la aplicación (uso tras `Close`). | sí, si hace falta |
| `ErrFailed` | una escritura o un fsync falló: el motor quedó inutilizable | no | Revisa el disco (`dmesg`, SMART, espacio). `Close` no escribe nada. | **sí**: `Close` y `Open` recuperan lo durable |
| `ErrFailed` + `ErrDiskFull` | disco lleno donde ya no se podía volver atrás: **solo lectura** (`Stats.ReadOnly`) | no (las lecturas siguen) | Libera disco (sección 9). | **sí** |
| `ErrDiskFull` (solo) | sin lugar; la operación no hizo nada; un lote queda abortado | sí, con lugar | Libera disco. Las escrituras vuelven solas, sin reabrir. | no |
| `ErrThrottled` | el WAL sin reciclar (o sin archivar, o retenido por una ranura sin tope) pasa de 2 × `MAX_WAL_SIZE` desde hace `LOCK_TIMEOUT`; no hizo nada | sí, con espera | Mira `PressureCause`: disco lento (`wal`), archivador caído (`archive`) o réplica caída (`slot`). | no |
| `ErrCorrupt` | el WAL, el control o el catálogo no son coherentes (falta el registro del checkpoint, un registro mal formado, un segmento sellado del WAL dañado, `archive_status` o `slots` dañados) | no | Guarda el directorio; restaura el respaldo. El daño en un segmento sellado también cumple `errors.Is(err, ErrCorrupt)` (desde la tarea 101; el mensaje trae además el detalle de `wal`). | `Open` falla |
| `ErrFormatVersion` | el directorio es de otra versión del formato en disco: más nueva que la que lee el binario, o anterior y sin migración (el mensaje dice cuál archivo y qué versión) | no | `Open` no tocó nada. Con una más nueva, abre con la versión de `et` que lo escribió (o una posterior); con una anterior, migra (sección 10). | `Open` falla |
| `ErrBackupIncomplete` | `Open` de una copia que no terminó o cuyo WAL no llega a su fin | no | La copia no sirve; la copia no se tocó. Usa otro respaldo o completa `wal/`. | — |
| `ErrBackupDir` | el destino de `Backup` no está vacío | no | Usa un directorio nuevo. | no |
| `ErrRecoveryTarget` | `OpenAt`/`OpenAtTime` con un objetivo inalcanzable | no | Corrige el objetivo o agrega los segmentos que faltan; la copia sigue igual. | — |
| `ErrArchiveOff` | `PendingArchive`/`SetArchived` con el archivo apagado | no | Abre con `WAL_ARCHIVE=on`. | sí, para encenderlo |
| `ErrNotSealed` | `SetArchived` de un segmento todavía no sellado | no | Archiva solo lo que da `PendingArchive`. | no |
| `ErrReadOnly` | escritura (o `NewBatch`, `Vacuum`, `Repair`, `Backup`, ranura lógica) en una **réplica** | no | Escribe en el primario. No confundir con el solo lectura por disco lleno. | no |
| `ErrNotReplica` | `ApplyWal` en un motor que no es réplica (o ya promovida), u `OpenReplica` de un directorio sin motor | no | La réplica empieza desde un `Backup`. | — |
| `ErrWalGap` | `ApplyWal` recibió un registro que no sigue | sí | Pide de nuevo desde `ReplayLSN()`. | no |
| `ErrWalRemoved` | el WAL pedido ya se recicló, o la ranura se invalidó | no | Réplica: `Backup` nuevo. Consumidor lógico: ranura nueva y copia completa. | no |
| `ErrSlotNotFound`, `ErrSlotExists`, `ErrSlotLimit` | la ranura no existe; ya existe; hay 16 o el nombre no mide de 1 a 64 B | no | Corrige el nombre o borra ranuras sin uso. | no |
| `ErrRecoveryConflict` | snapshot de la réplica cancelado para aplicar el WAL | sí, con un snapshot nuevo | Acorta los snapshots de la réplica (josefina: 503 y reintento). | no |
| `ErrSerialization` | un lote escribió un id que otra transacción confirmó después de que el lote empezó; el lote quedó abortado | sí: `Rollback` y repetir **el lote entero** | Normal con concurrencia; si es frecuente, lotes más cortos. | no |
| `ErrConflict` | otra transacción tuvo el id más de `LOCK_TIMEOUT` (en un lote, queda abortado) | sí | Si es frecuente: interbloqueos entre lotes o lotes largos. | no |
| `ErrTimeout` | el lote o el snapshot pasó su tiempo máximo | sí, más corto | Lotes más cortos, o `SnapshotFor`. | no |
| `ErrTxDone` | el lote o el snapshot ya terminó | no | Error de la aplicación. | no |
| `ErrOtherEngine` | el lote o el snapshot es de otro motor | no | Error de la aplicación. | no |
| `ErrBusy` | `Empty` con lotes o snapshots abiertos; `DropStoreIn` con otros lotes que escribieron en el store; `CreateLogicalSlot` que no pudo esperar | sí, cuando terminen | Cierra snapshots y lotes largos. | no |
| `ErrStorePending` | el store lo crea o lo borra un lote abierto | sí, cuando ese lote termine | — | no |
| `ErrStoreNotFound` | `DropStore` de un store que no existe, o `Store` de uno inexistente en una réplica | no | — | no |
| `ErrStoreDropped` | el store se borró; su `*Store` ya no sirve | no | Pide el store de nuevo con `Store(name)`. | no |
| `ErrBadID`, `ErrValueTooLarge`, `ErrBadName` | id fuera de 1–2.000 B, valor de más de 64 MiB, nombre inválido | no | Error de la aplicación. | no |

Un ayudante para reintentar:

```go
// reintentable dice si una escritura que falló se puede repetir tal cual.
func reintentable(err error) bool {
	switch {
	case errors.Is(err, store.ErrFailed): // también con ErrDiskFull: solo lectura, hay que reabrir
		return false
	case errors.Is(err, store.ErrDiskFull), errors.Is(err, store.ErrThrottled):
		return true // no hizo nada; reintentar con espera
	case errors.Is(err, store.ErrSerialization), errors.Is(err, store.ErrConflict), errors.Is(err, store.ErrTimeout):
		return true // el lote quedó abortado: Rollback y repetir el lote entero
	case errors.Is(err, store.ErrBusy), errors.Is(err, store.ErrStorePending), errors.Is(err, store.ErrRecoveryConflict):
		return true
	}
	return false
}

func conReintento(e *store.Engine, s *store.Store, id string, v []byte) error {
	for intento := 0; ; intento++ {
		b := e.NewBatch()
		err := s.PutIn(b, id, v)
		if err == nil {
			err = b.Commit()
		}
		if err == nil {
			return nil
		}
		b.Rollback()
		if !reintentable(err) || intento == 5 {
			return err
		}
		time.Sleep(time.Duration(10<<intento) * time.Millisecond)
	}
}
```

Un error de lectura por una página dañada se devuelve sin inutilizar el motor: corre `Verify` (sección 7).

---

## 9. Disco lleno o lento

### Disco lleno

El motor distingue dónde faltó el lugar:

| Dónde | Resultado |
|---|---|
| a la entrada de una escritura (cada escritura exige medio segmento del WAL preasignado) | `ErrDiskFull` limpio: no se escribió nada y el motor sigue |
| al extender un archivo de datos (cada página nueva se escribe entera en ceros, sin huecos: escribirla después nunca pide lugar nuevo en ext4 o XFS) | `ErrDiskFull` limpio |
| en el archivo de estados o el catálogo durante un checkpoint | el checkpoint aborta y se reintenta; el motor sigue |
| en el WAL en medio de una operación, al escribir una página sucia o el control, o cualquier fsync | **solo lectura**: `ErrDiskFull` + `ErrFailed`, `Stats.ReadOnly = true`, `PressureCause = read_only` |

En **solo lectura**:

- siguen las lecturas, los recorridos, los snapshots y `Store` de uno que existe;
- las escrituras, `NewBatch`, `Checkpoint`, `Vacuum`, `WaitDurable` y crear un store dan `ErrDiskFull` + `ErrFailed`;
- no se pierde nada confirmado y durable. Las confirmaciones diferidas sin fsync se ven mientras dure el motor, pero al
  reabrir pueden faltar (siempre un prefijo, en el orden del WAL).

En Linux, con un ext4 lleno de verdad (`test/linuxvm/diskfull.sh`), el disco lleno siempre salió limpio: el solo
lectura no se alcanzó. Haría falta un fsync que falle por lugar (volumen fino, NFS). En APFS, la copia en escritura
podría pedir lugar al sobrescribir un bloque preasignado (**a confirmar**).

**Recuperarse:**

1. **`ErrDiskFull` limpio:** libera disco. Las escrituras vuelven solas, sin reabrir.
2. **Solo lectura:**
   1. Libera disco.
   2. Llama a `Close`, que no escribe nada y devuelve el fallo.
   3. Llama a `Open`, que recupera todo lo durable desde el WAL. `Open` con el disco todavía lleno falla con
      `ErrDiskFull` (el checkpoint de fin de recuperación necesita lugar), pero nunca corrompe.
3. **Prevención:** vigila el espacio libre desde fuera del motor, con una reserva. Josefina usa un vigía de `statfs`
   con `JDB_DISK_RESERVE_MB`, 128 MB por omisión.
4. **Qué liberar:** primero, lo que el motor retiene sin necesidad:
   - ranuras sin uso (`DropSlot`);
   - el archivador atrasado (`SetArchived` tras copiar);
   - respaldos a directorio en el mismo disco.
   
   Nunca borres a mano archivos de `wal/`, `data/` ni `xact/`.

### Disco lento

Nada crece en el heap de Go cuando el disco tarda:

- **Group commit:** un fsync confirma todo lo publicado.
- **Páginas sucias:** la caché es fija. Si las reemplazables están sucias, quien necesita un búfer escribe una antes.
- **WAL sin reciclar:** pasado 2 × `MAX_WAL_SIZE`, los escritores esperan un checkpoint (o al archivador, o a la ranura
  sin tope) hasta `LOCK_TIMEOUT` y después reciben `ErrThrottled` sin efecto (`Stats.Throttled`). Un checkpoint
  atrasado se apura solo.
- **Escrituras diferidas:** con más de 8.192 registros sin fsync, esperan un fsync.

**Qué hacer según la causa:**

| `PressureCause` | Qué hacer |
|---|---|
| `wal` | Los checkpoints no terminan a tiempo: disco lento, o un checkpoint que falla por disco lleno. Revisa `FsyncLatency` y el espacio libre. Si la carga lo justifica, sube `MAX_WAL_SIZE`, a costa de más tiempo de `Open`. |
| `archive` | El archivador no avanza: arréglalo, o ponle tope (`WAL_ARCHIVE_MAX_SIZE`) si prefieres perder continuidad a frenar. |
| `slot` | Una réplica o un consumidor sin tope está caído: recupéralo o bórrale la ranura. |
| `deferred` | Llama a `WaitDurable` más seguido, o baja `WAL_WRITER_DELAY`. |
| `dirty` | La caché está casi toda sucia: el disco no da abasto escribiendo páginas. Sube la caché o mejora el disco. |

---

## 10. Actualizar de versión

**El formato en disco todavía no está congelado.** La tarea 54 (versión 1.0: formato congelado; todo cambio posterior
con migración y prueba de ida y vuelta) está **en curso**: ya están la versión por archivo, `ErrFormatVersion` y la
prueba de ida y vuelta, pero falta guardar el directorio de referencia `store/testdata/format-v1`, que espera dos
decisiones que cambiarían el formato (compactar el encabezado de cada versión, tarea 96, y un encabezado con versión
en los segmentos del WAL). Hasta la versión 1.0, actualizar `et` puede exigir migrar los datos. Desde la 1.0, todo
cambio de formato llevará versión nueva, migración y prueba de ida y vuelta.

Lo que hay hoy:

- **Una sola versión de formato.** El control (cada copia), el catálogo, `backup_label`, `archive_status` y `slots`
  llevan `FormatVersion = 1`; las páginas, la suya en `pd_pagesize_version`, y la meta de cada store, la suya. El WAL y
  el archivo de estados no llevan versión propia: los cubre la del control. La tabla completa está en `store/doc.go`
  («Versiones del formato»).
- **Otra versión hace fallar `Open` sin tocar nada**, con `ErrFormatVersion` (`errors.Is(err, store.ErrFormatVersion)`).
  El mensaje nombra el archivo y dice si la versión es más nueva («upgrade et») o anterior («not migratable»). Basta
  con que una copia válida del control sea de otra versión: `Open` no vuelve a la copia vieja.
- **Cambios compatibles dentro de la versión 1.** El formato cambió sin subir la versión, de modo que el código nuevo lee
  lo viejo:
  - la pila de páginas borradas de la meta (tarea 70: una meta anterior se lee con la pila vacía);
  - la cola de DDL y el origen en `KindCommit` (tarea 83: los registros viejos se leen igual);
  - la marca de sellado de los segmentos del WAL (tarea 90).

  Lo contrario (un binario viejo que lee lo que escribió uno nuevo) no está garantizado (**a confirmar**): **no vuelvas
  atrás de versión sobre el mismo directorio**.
- **La réplica física copia el WAL byte a byte.** El primario y sus réplicas tienen que correr la misma versión de `et`.

Procedimiento hasta la 1.0:

1. Fija la versión de `et` en el `go.mod` del servicio con un tag (`./version.sh` en `et`). No dependas del `go.work`
   en producción.
2. Antes de actualizar, lee `doc/CHANGELOG.md` en busca de cambios de formato.
3. Haz un `Backup` con la versión vieja. Ábrelo con el binario nuevo **sobre una copia**, corre `Verify` y compara los
   `Count`. Si `Open` falla, ese salto exige migrar.
4. **Migrar** es exportar con la versión vieja y cargar con la nueva:
   - recorre cada store sobre un `SnapshotFor(0)` con el binario viejo;
   - escribe en un motor nuevo con el binario nuevo (en lotes o con `Deferred` + `WaitDurable`).

   No hay herramienta de migración (**a confirmar**).
5. Con réplicas, actualiza todos los nodos juntos: rehaz cada réplica desde un `Backup` del primario ya actualizado.
6. Guarda la versión de `et` junto a cada respaldo: un respaldo se restaura con una versión que lea su formato.

---

## 11. Pruebas y validación antes de producción

### En la máquina de desarrollo

```bash
cd et
go build ./store/... && go vet ./store/...
go test -p 1 ./store/...           # todo offline
go test -p 1 -race ./store/...     # y sin -race: en arm64, un acceso atómico desalineado solo falló sin -race
go run ./cmd/store                 # el ejemplo de la API
go run ./cmd/storecheck -dir <dir> [-repair]
```

Las corridas largas (`-count` ≥ 20) van de a un binario por vez, midiendo la memoria con `/usr/bin/time -l`.

### `test/ci` (CI en el equipo IA)

| Script | Qué hace |
|---|---|
| `./test/ci/ci.sh` | Verificación de un cambio (~10 min). En la VM Linux: `go test -p 1 ./store/...`, `-race`, cortes con LazyFS (5 rondas) y con dm-flakey (3 + 2). En macOS: `go test -p 1 ./store/...`. Sale con 0 si todo está bien. Córrelo antes de un `git backup`. |
| `./test/ci/nightly.sh` | Cada noche a las 03:00, por LaunchAgent: benchmarks contra la línea base (FALLA si una métrica empeora más del 10 %), los 10 fuzzers de `store/` y la suite completa de cortes y de disco lleno. |
| `./test/ci/fuzzlong.sh` | Corrida larga de fuzzers: 10 × 48 min = 8 h (tarea 73). |
| `./test/ci/status.sh [ci\|nightly\|fuzzlong]` | El último resultado. |

### `test/linuxvm` y `test/powercut` (fallos de disco reales)

```bash
./test/linuxvm/vmrun.sh powercut-lazyfs.sh ROUNDS=20    # cortes de energía: se pierde lo que no tuvo fsync
./test/linuxvm/vmrun.sh powercut-flakey.sh              # drop_writes y error_writes (EIO en write y fsync)
./test/linuxvm/vmrun.sh diskfull.sh                     # ext4 chico que se llena
```

`test/powercut` usa solo la API pública: un escritor por store, con `Put`, `Delete`, lotes, diferidas con
`WaitDurable`, `Empty` y `DropStore`. Anota fuera del disco que se corta cada operación confirmada y durable. Después,
`powercut -mode verify -dir <motor> -log <registro>` exige tres cosas:

- nada confirmado falta, y cada lote está entero o no está;
- `Count` coincide con el recorrido;
- `Verify` sale limpio tras `Repair`.

Usa esta misma herramienta para validar el disco de producción (no está probado sobre un NVMe real).

### `test/storebench` (dimensionar con tu hardware)

```bash
go build -o storebench ./test/storebench
./storebench -n 10000000 -batch 20000 -value 256         # carga, RSS, Open, lectura fría y en caché
./storebench -n 1000000 -rate 5s -writers 1,8,64         # Put/s durable y diferido por escritores
STORE_CACHE_SIZE=512 ./storebench -n 100000000 -dir /datos/sb   # memoria con tu caché
CHECKPOINT_TIMEOUT=3600 ./storebench -crashwal 1024 -dir /datos/c   # carga hasta 1 GB de WAL y se mata (kill -9)
./storebench -recover -dir /datos/c                      # mide Open tras la caída, Count y Verify
```

### Lista mínima antes de producción

- [ ] `ci.sh` en verde, y el nocturno en verde durante 30 días (criterio del bloque D).
- [ ] Revisión del código sin hallazgos graves abiertos. El crítico y los altos de `doc/review-store-2026-11-20.md`
      están arreglados (tareas 88–95), y también los tres altos de la segunda parte (C1, C2 y V1: tareas 102–104) y
      el medio D1 (un `DropStore` que podía dejar un archivo huérfano tras un corte). Siguen pendientes los medios,
      entre ellos:
  - N3: `Close` detrás de un checkpoint repartido;
  - N4: `DropStore` suelto no mira los lotes que escribieron en el store (sus escrituras se pierden en silencio);
  - R3: `fn` del flujo lógico con el motor tomado;
  - R4: `CreateLogicalSlot` espera con el motor tomado;
  - S2: `SyncAll` puede devolver nil tras un fsync fallido;
  - V2: la limpieza de un store grande frena el DDL sobre ese store, y con él el checkpoint.
- [ ] `storebench` en el hardware de producción: Put/s durable, `Open` tras `-crashwal` con tu `MAX_WAL_SIZE`, y RSS
      con tu caché.
- [ ] `powercut` sobre el disco y el sistema de archivos de producción.
- [ ] Un respaldo restaurado y verificado (sección 5), y un PITR de prueba si usas el archivo continuo.
- [ ] Alertas de la sección 4 conectadas.
- [ ] La versión de `et` fijada con un tag (sección 10).
