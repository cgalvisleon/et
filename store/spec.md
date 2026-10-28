# Especificación: `store`

Este documento es la referencia del paquete `github.com/cgalvisleon/et/store` (hasta el 2026-10-25, `internal/store` de Josefina, que desde entonces usa solo Pebble): define el formato en disco y lo que garantiza cada operación. Ante una diferencia con el código, se corrige uno de los dos en el mismo cambio.

**Regla de mantenimiento:** todo cambio en `store` que altere el formato en disco, la API pública o un comportamiento descrito aquí actualiza este archivo en el mismo cambio, junto con la sección del store en `README.md`, `doc/README.es.md` y `CLAUDE.md` de `et` cuando cambie la API.

Versión del documento: 10 — 2026-10-27 (`Debug()` es el nombre del activador de logs, `IsDebug()` queda como alias en desuso; nombres de los mensajes `MSG_STORE_*` y el momento en que se lee la configuración corregidos en §7, §8 y §10; `Empty` devuelve el error de borrar el directorio de segmentos, antes lo perdía en un `defer`; `FileStore.ID` es el nombre normalizado, antes quedaba vacío en `ToJson`; versión 9: 2026-10-25, el paquete pasa de `josefina/internal/store` a `et/store`; los mensajes de error son `MSG_STORE_*` de `et/msg`; el verificador de carga es `go run ./test/storebench` de `et`; versión 8: funciones privadas sin uso quitadas: `segment.readHeader`, `FileStore.syncRef`, `read`, `readHeader` y `prune`; ninguna estaba en la API ni en el comportamiento descrito; versión 7: 2026-10-18, `Open` se recupera solo: restaura una compactación interrumpida y trunca la cola cortada del segmento activo; versión 6: `mmap` también del segmento activo; versión 5: escrituras diferidas, `Deferred` y `WaitDurable`; versión 4: índice por fragmentos, `mmap` de segmentos sellados, CRC en el índice para `Update`, contadores atómicos; versión 3: group commit).

## 1. Alcance

- Store de **un solo nodo**, concurrente, de solo agregado (tipo WAL), que guarda `id → []byte`.
- Guarda bytes opacos; codificar (JSON u otro) es tarea de quien lo usa.
- Es la base por nodo para alta disponibilidad: la replicación será una capa **encima** (`Sync` + `ApplyWalEntry` + `WalSince`), no dentro.
- Un store abierto pertenece a un solo proceso. Abrir el mismo store en dos procesos a la vez no está soportado.

## 2. Estructura en disco

Para `Open(pathData, pathWald, name, mode)`, con `name` normalizado (§10):

| Ruta | Contenido |
|---|---|
| `pathData/segments/<name>/segment-NNNNNN.dat` | Segmentos del log, numerados desde `000001`. |
| `pathWald/snapshot/<name>/state-<name>.snap` | Snapshot del índice (opcional). |
| `pathWald/compact/<name>/segments-<name>.tmp/` | Segmentos nuevos durante una compactación. |
| `pathData/segments/<name>.segments.old/` | Segmentos viejos durante el intercambio de una compactación. |
| `pathWald/recover/<name>/<segmento>.<fecha>.corrupt` | Bytes descartados por `Recover`. |

Todos los enteros en disco son **big-endian**. Los CRC son CRC-32 **Castagnoli**.

## 3. Registro

```
[LSN:8][DataLen:4][CRC:4][IDLen:2][ID:IDLen][Status:1][Data:DataLen]
```

| Campo | Tipo | Significado |
|---|---|---|
| `LSN` | uint64 | Número de secuencia del registro (§5). |
| `DataLen` | uint32 | Longitud de `Data`. |
| `CRC` | uint32 | CRC de `Data` **solamente** (no cubre LSN, ID ni estado). |
| `IDLen` | uint16 | Longitud del id: 1 a 65535 bytes. |
| `ID` | bytes | El id. |
| `Status` | byte | `Active` (1) o `Deleted` (2). |
| `Data` | bytes | Los datos; vacío en `Deleted` (tombstone). |

- Encabezado fijo de 19 bytes más el id.
- Encabezado y datos se escriben con una sola escritura.
- Un registro se considera válido si el encabezado se puede leer completo, `IDLen` está entre 1 y 65535 y, cuando hay datos, el CRC coincide.

## 4. Segmentos

- Solo el último segmento (el **activo**) recibe escrituras; los demás están sellados (fsync y solo lectura).
- **Lectura de segmentos sellados:** en plataformas Unix cada segmento sellado (al rotar, al compactar y, al abrir, todos menos el último; en `ReadOnly`, todos) se mapea en memoria solo lectura (`mmap`), y sus lecturas son copias de memoria, sin llamadas al sistema. El **activo** también se mapea, con capacidad `MaxSegment` (o su tamaño, si es mayor) desde que se crea o se abre: el mapeo es compartido, así que lo que `write` agrega se ve en él, y el segmento lleva la cuenta de los bytes ya escritos (`visible`). Una lectura que cae entera dentro de esos bytes se copia del mapeo; cualquier otra (un registro que no entra en la capacidad, o leer pasado el final) va a `ReadAt`, con el mismo resultado de siempre. Si el mapeo falla, o en otras plataformas, se lee con `ReadAt`. El mapeo se libera al cerrar el archivo, cuando ya no quedan lectores (§13). Ningún proceso debe truncar un segmento con el store abierto (solo truncan `Recover`, que exige el store cerrado, y `Open` en `ReadWrite`, sobre el activo y antes de retornar; §7).
- **Rotación:** antes de escribir un registro, si `tamaño activo + tamaño del registro > RELSEG_SIZE` MB, se sella el activo, se crea el siguiente y se escribe un snapshot nuevo (§6). Un registro más grande que el máximo se escribe igual, solo, en un segmento nuevo.
- **Lectura del log (`scan`):** recorre los registros en orden y se detiene **en silencio** en el primer registro inválido o al final del archivo. Lo que sigue a un registro inválido no se lee.

## 5. LSN

- El store mantiene un contador `WAL` = el LSN más alto escrito.
- Cada escritura local (`Insert`, `Update`, `Delete`) toma `WAL + 1`.
- `ApplyWalEntry` conserva el LSN recibido y deja `WAL = max(WAL, LSN)`.
- La compactación conserva el LSN original de cada registro que copia.
- Al abrir, `WAL` se restaura al máximo entre el LSN guardado en el snapshot y los LSN vistos al recorrer segmentos. **Nunca retrocede**: la compactación conserva el tombstone más reciente para garantizarlo.

## 6. Snapshot

```
"SNAP"[4] Version:uint16 Count:uint64 WAL:uint64
  Count × ( IDLen:uint16 ID SegIndex:uint32 Offset:int64 DataLen:uint32 )
CRC:uint32   (de todo lo anterior)
```

- Versión actual: **2**. La versión 1 (sin `WAL`) se sigue leyendo.
- Contiene las entradas del índice que están en segmentos **sellados**; excluye las del segmento activo.
- Se escribe en `<ruta>.tmp`, con fsync, y se renombra encima del anterior (reemplazo atómico).
- Se escribe en cada rotación y al terminar cada compactación. Se borra al empezar el intercambio de una compactación y en `Recover`.
- Es **opcional**: si falta, está incompleto, tiene otra marca o versión, o el CRC no coincide, se ignora y el índice se reconstruye recorriendo todos los segmentos.

## 7. Apertura y cierre

### `Open`

1. `ReadWrite`: repara una **compactación interrumpida** como el paso 1 de `Recover` (§12): restaura `.segments.old` si falta el directorio de segmentos (y borra el snapshot), borra `.segments.old` si existen los dos, y borra el temporal de compactación. Después crea los directorios que falten. `ReadOnly`: si el directorio de segmentos no existe, `MSG_STORE_NOT_FOUND`; no crea ni repara nada.
2. Abre los segmentos en orden de nombre. `ReadWrite` sin segmentos crea `segment-000001.dat`.
3. Carga el índice:
   - con snapshot válido: lo carga y aplica encima el segmento activo;
   - sin snapshot: recorre todos los segmentos. En ambos casos, en orden: un registro `Active` apunta el id a él; un `Deleted` lo quita.
4. `ReadWrite`: si el segmento **activo** tiene bytes después de su último registro válido (un registro cortado por una caída), los copia a `pathWald/recover/<name>/<segmento>.<fecha>.corrupt`, trunca el activo en ese punto con fsync y deja escribir desde ahí; lo registra en el log (`open:repair`). Solo el activo puede quedar cortado: la rotación hace fsync del anterior antes de crear el siguiente (§4). Un registro corrupto en medio del activo se trata igual (lo que sigue queda en la cuarentena, no se borra); uno en un segmento sellado no se toca y ahí el recorrido se detiene como siempre (eso es para `Recover`).
5. La configuración (§11) se lee al construir el store, antes del paso 1 (`newFileStore`).

- Tras un apagado inesperado ya no hace falta `Recover` antes de `Open`: lo escrito y confirmado después de reabrir queda a continuación de lo válido, no detrás de la basura. `Recover` sigue sirviendo para revisar **todos** los segmentos con el store cerrado.
- El contador de tombstones empieza en 0 al abrir: no se persiste.

### `Close`

Detiene `OnStats`, espera cualquier compactación en curso, hace fsync de todos los segmentos y los retira (§13). Después de `Close` el store no se debe usar.

### `Empty`

Cierra el store, deja el índice y los contadores en cero, borra el snapshot y borra sus segmentos (si borrar el directorio falla, devuelve ese error): al volver a abrirlo con el mismo nombre queda vacío y con el contador `WAL` en 0 (antes el snapshot traía de vuelta el índice y el contador viejos; una réplica que vaciaba un store para una copia completa descartaba como ya aplicadas las entradas con LSN menor que el contador viejo).

## 8. Escritura

Todas las escrituras se serializan. Para cada una, la verificación de existencia, la escritura en el log y la actualización del índice ocurren **como un solo paso**: ninguna otra escritura se intercala.

| Operación | Escribe si | Resultado | Registro |
|---|---|---|---|
| `Insert(id, data)` | `id` no existe | `true` si insertó; `false` si ya existía | `Active` |
| `Update(id, data)` | `id` existe **y** `data` difiere byte a byte de lo guardado | `true` si actualizó; `false` si no existe o es idéntico | `Active` |
| `Delete(id)` | `id` existe | `true` si eliminó; `false` si no existía | `Deleted` |

- No hay upsert: se compone con `Update` y luego `Insert` (y `Update` de nuevo si el `Insert` perdió una carrera).
- Errores: `id` vacío → `MSG_STORE_ID_IS_REQUIRED`; id de más de 65535 bytes → `MSG_STORE_INVALID_ID_LENGTH`; datos de más de 4 GiB → `MSG_DATA_TOO_LARGE`; store `ReadOnly` → `MSG_STORE_IS_READ_ONLY`.
- **Durabilidad:** con `SYNC_ON_WRITE=true` (por defecto) cada escritura retorna cuando su registro ya está en disco (fsync). Con `false`, lo escrito puede perderse ante un corte de energía (no ante una caída del proceso).
- **Group commit:** el fsync no se hace dentro de la escritura serializada. Cada escritura (también `ApplyWalEntry`) escribe en el log, actualiza el índice y avisa a `Sync`; suelta el turno y luego espera el fsync. Si no hay uno en curso, ella misma lo hace (líder) sobre el segmento activo y cubre todo lo escrito hasta ese momento; si lo hay, espera a que termine y, si no la cubrió, repite. Un fsync atiende así a todas las escrituras que llegaron mientras corría el anterior. Lo que estaba en segmentos sellados ya está en disco (la rotación y la compactación hacen fsync antes de cambiar el activo; `Close` sella todo).
- **Escrituras diferidas:** `Deferred()` retorna una vista con `Insert`, `Update` y `Delete` idénticos a los de arriba (mismas condiciones, mismo resultado, mismo registro, misma visibilidad y mismo aviso a `Sync`), salvo que retornan sin esperar el fsync. `WaitDurable()` espera a que esté en disco todo lo escrito hasta el momento de la llamada (diferido o no), con el mismo group commit; sin `SYNC_ON_WRITE` retorna de inmediato, y si hubo un error de fsync lo retorna. Llamarlo después de `Close` no falla (`Close` ya forzó todo a disco). Quien escribe diferido debe llamar `WaitDurable` antes de confirmar su escritura: es lo que permite a una capa superior escribir varias cosas bajo su propio lock y esperar el disco después de soltarlo.
- **Visibilidad antes de durabilidad:** lo escrito es visible para lecturas y para `Sync` desde que la escritura suelta su turno, antes de su fsync (una ventana de un fsync). Ante una caída del proceso no se pierde nada (el `write` ya está en el kernel); solo un corte de energía o una caída del sistema operativo en esa ventana dejaría a un lector, o a una réplica, habiendo visto un registro que el store pierde. Quien escribió no recibió respuesta de esa escritura.
- El primer error de fsync queda fijo: todas las escrituras siguientes con `SYNC_ON_WRITE=true` fallan con él.
- El primer error de escritura en un segmento queda fijo: todas las escrituras siguientes en ese segmento fallan con él.
- `Update` compara sin leer el disco cuando puede: el índice guarda el CRC de cada registro, y si la longitud o el CRC difieren el dato es distinto. Solo con CRC y longitud iguales (o con una referencia cargada del snapshot, que no guarda el CRC) lee el registro y compara byte a byte.
- `Update` y `Delete` cuentan un tombstone (el registro anterior queda obsoleto) y pueden lanzar la compactación automática (§9).

## 9. Compactación

- **Automática:** después de un `Update` o `Delete`, si `tombstones > max(10 % de los ids vivos, MIN_THRESHOLD_COMPACT)`, se lanza en segundo plano. Nunca hay más de una a la vez.
- **Manual:** `Compact()` la ejecuta en la goroutine que llama; si hay otra en curso, espera a que termine.
- **Fases:**
  1. Con las escrituras detenidas un instante: copia del índice y punto de corte (segmento y posición actuales).
  2. Sin bloquear nada: copia los registros vivos, en orden de id, a segmentos nuevos en el directorio temporal.
  3. Con escrituras y lecturas del índice detenidas: aplica en orden lo escrito después del corte (registros vivos copiados; eliminaciones escritas como tombstone), borra el snapshot, intercambia los directorios con dos renombres, retira los segmentos viejos y escribe un snapshot nuevo.
- Resultado: sin registros sobrescritos ni tombstones, salvo los tombstones de la fase 3 y el más reciente si hace falta para el WAL (§5). El contador de tombstones queda en los de la fase 3.
- Las lecturas en curso no se interrumpen: terminan sobre los segmentos viejos, que se cierran al terminar su último lector.
- Si falla antes del intercambio, se descartan los segmentos nuevos y el store sigue con los viejos. Si falla el segundo renombre, se restaura el directorio original.

## 10. Lectura

- `Get(id)` → `(datos, existe, error)`. Los datos son una copia nueva: quien llama puede guardarlos. Si el registro en disco no coincide con el índice (id, longitud, estado o CRC), `MSG_STORE_CORRUPTED_RECORD`.
- `GetVersion(id)` → `(datos, versión, existe, error)`: como `Get`, más la **versión** de `id`: el LSN de su último registro (`0` si no existe). Cambia con cada `Insert`, `Update` o `Delete` que escribe, y se conserva al compactar y al replicar (`ApplyWalEntry` mantiene el LSN). Sirve para concurrencia optimista: si la versión no cambió, nadie escribió `id` (así lo usaban las transacciones interactivas de Josefina).
- `IsExist(id)`, `Count()`, `Keys(asc, offset, limit)` usan solo el índice en memoria, sin leer el disco.
- **Orden:** `Keys` y `ForEach` ordenan por id como texto (orden de bytes), ascendente o descendente. `offset < 0` cuenta como 0; `limit <= 0` es sin límite.
- `ForEach(fn, asc, offset, limit)`:
  - fija el conjunto de registros al empezar; lo escrito después no aparece;
  - lee del disco en paralelo (hasta `NumCPU` goroutines), pero llama a `fn` **de a uno, en orden, en la goroutine que llamó**: `fn` no necesita locks;
  - no sostiene locks mientras llama a `fn`: `fn` puede usar el store (leer o escribir);
  - si `fn` retorna `false` se detiene ahí; si retorna un error, `ForEach` lo retorna; un error de lectura también lo detiene.
- `Normalize(nombre)`: quita espacios de los extremos, cambia espacios por `_`, elimina todo lo que no sea letra ASCII, dígito, `_` o `.`, y quita los dígitos iniciales.

## 11. Configuración

Leída en `Open`, por variable de entorno:

| Variable | Por defecto | Efecto |
|---|---|---|
| `RELSEG_SIZE` | `128` | Tamaño máximo de segmento, en MB. |
| `SYNC_ON_WRITE` | `true` | Cada escritura espera su fsync (agrupado, §8). |
| `MIN_THRESHOLD_COMPACT` | `1000` | Mínimo de tombstones para la compactación automática. |

## 12. Recuperación: `Recover`

Repara el store en disco después de una falla. `Open` en `ReadWrite` ya hace por sí solo el paso 1 y el paso 3 sobre el segmento activo (§7); `Recover` revisa además los segmentos sellados y borra el snapshot. Requisitos: el store está **cerrado en todos los procesos**. Pasos, en orden:

1. **Compactación interrumpida:**
   - falta el directorio de segmentos y existe `.segments.old` → se restaura `.segments.old` (la caída fue entre los dos renombres);
   - existen los dos → se borra `.segments.old` (el intercambio terminó);
   - no existe ninguno → `MSG_STORE_NOT_FOUND`.
   Además se borra el directorio temporal de compactación si existe.
2. Borra el snapshot, para que el próximo `Open` reconstruya el índice completo.
3. Recorre cada segmento; si tiene bytes después del último registro válido, los **copia** a `pathWald/recover/<name>/` y trunca el segmento en ese punto (con fsync). Nunca borra nada sin dejar copia.

Retorna un reporte: `name`, `restored_segments`, `removed_old_segments`, `removed_compact_tmp`, `removed_snapshot`, `records`, `truncated_bytes` y, por segmento, `name`, `size`, `records`, `valid`, `truncated` y `quarantine` (si se truncó).

## 13. Concurrencia

- Todas las operaciones públicas se pueden llamar desde varias goroutines a la vez.
- **Escrituras:** una a la vez (`Insert`, `Update`, `Delete`, `ApplyWalEntry` y las fases 1 y 3 de la compactación). La espera del fsync queda fuera de ese turno (§8): varias escrituras esperan el mismo fsync.
- **Lecturas:** en paralelo entre sí y con las escrituras. Nunca sostienen el lock del índice mientras leen el disco: fijan el segmento, sueltan el lock, leen y liberan.
- **Índice por fragmentos:** el índice en memoria está repartido en 64 fragmentos por hash del id, cada uno con su lock. Las operaciones sobre un id (`Get`, `GetVersion`, `IsExist` y la actualización del índice de cada escritura) toman el lock de estructura compartido y el de su fragmento: una escritura bloquea solo su fragmento, no a todos los lectores. Lo que recorre el índice (`Count`, `Keys`, `ForEach`, snapshot, compactación) toma todos los fragmentos a la vez, así que ve el índice en un solo instante. Cambiar los segmentos o reemplazar el índice entero (rotación, intercambio de la compactación, carga del snapshot, `Empty`) toma el lock de estructura exclusivo.
- Los contadores (`WAL`, tombstones, tamaño, operaciones en curso) son atómicos, sin lock.
- Un registro publicado en el índice ya está escrito y se puede leer.
- Un segmento retirado (por compactación o `Close`) solo se cierra cuando termina su último lector.
- Orden de locks interno: compactación → escritura → índice (estructura) → fragmentos (en orden de número).

## 14. Monitoreo y replicación

### `Stats` y `OnStats`

- `Stats()` retorna `count` (ids vivos), `size` (bytes en segmentos), `wal`, `tomb_stones` y las operaciones en ejecución `reading` (`Get`, `ForEach`), `inserting`, `updating`, `deleting`. Una escritura que espera su turno cuenta como en ejecución. `ApplyWalEntry` no se cuenta.
- `OnStats(fn)`: `fn` recibe `Stats` cada vez que cambian las operaciones en ejecución, desde **una sola goroutine de fondo**; varios cambios seguidos se entregan como el último estado. Nunca frena las operaciones. Se detiene en `Close`.

### `Sync`

- `Sync(fn)` recibe cada cambio aplicado como `Change{Op, ID, Data, LSN}`:

| Origen | `Op` |
|---|---|
| `Insert` | `OpInsert` |
| `Update` | `OpUpdate` |
| `Delete` | `OpDelete` (`Data` = `nil`) |
| `ApplyWalEntry` `Active`, id nuevo | `OpInsert` |
| `ApplyWalEntry` `Active`, id existente | `OpUpdate` |
| `ApplyWalEntry` `Deleted`, id existente | `OpDelete` |

- Se llama **durante la escritura**, justo después del log y el índice y **antes de su fsync** (§8): los cambios llegan en orden exacto de LSN y sin huecos.
- Las escrituras que no cambian nada no emiten (`Insert` de un id existente, `Update` idéntico o de un id inexistente, `Delete` de un id inexistente, `ApplyWalEntry` `Deleted` de un id inexistente).
- `fn` debe ser rápida, **no debe escribir en el mismo store** (se bloquearía) y debe copiar `Data` si la guarda.

### `ApplyWalEntry` y `WalSince`

- `ApplyWalEntry(entry)` escribe el registro recibido con su LSN y estado, y actualiza el índice (`Active` apunta el id; `Deleted` lo quita). No compara con lo existente ni valida el orden de los LSN: **quien replica debe aplicar cada entrada una sola vez y en orden**. No lanza compactación.
- `WalSince(since)` retorna, en orden de escritura, todos los registros del log con `LSN > since` (vivos y tombstones).
- La compactación descarta registros sobrescritos y tombstones: un nodo que quedó atrás de una compactación no puede ponerse al día con `WalSince` (le faltarían eliminaciones); necesita una copia completa.

## 15. Límites y defectos conocidos

Límites aceptados:

- Una escritura a la vez en el log; con `SYNC_ON_WRITE=true` cada fsync atiende a todas las que esperan (group commit), así que una sola escritura sigue pagando la latencia completa del disco.
- Lo escrito es visible antes de su fsync (§8).
- El CRC protege los datos, no el encabezado ni el id.
- Una escritura con el turno de escritura tomado hace su `write` al archivo. Si las lecturas ocupan todos los procesadores sin pausa (lectores en un bucle cerrado en el mismo proceso), la escritura que tiene el turno espera un procesador al volver de esa llamada y las escrituras caen a cientos por segundo. Con lectores que esperan entre operaciones (por ejemplo, peticiones por red) no ocurre.
- La rotación crea un segmento nuevo aunque el activo esté vacío: un registro más grande que `RELSEG_SIZE` deja un segmento vacío antes de él.

Defectos conocidos: ninguno abierto. `Empty` no borraba el snapshot (verificado el 2026-09-26); corregido el 2026-10-13 (§7). `Open` sin `Recover` tras una caída perdía lo escrito después de un registro cortado (verificado el 2026-10-18: 100 de 100 escrituras confirmadas tras reabrir, y una compactación interrumpida dejaba 186 de 200); corregido el 2026-10-18 (§7).

## 16. Verificación

No hay tests versionados. Las pruebas se hacen con archivos temporales en el paquete: tras `Open`, bajar `MaxSegment` a unos KB para forzar rotaciones; `SYNC_ON_WRITE=false` para velocidad; `MIN_THRESHOLD_COMPACT` para controlar la compactación automática; siempre con `-race`; y al final reabrir el store y comparar contra el estado esperado en memoria.
