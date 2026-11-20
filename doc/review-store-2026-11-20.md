# Revisión independiente de `et/store` — 2026-11-20 (tarea 53 de F-23, primera parte)

Revisión de solo lectura, hecha por quien no escribió el código. Se miró contra `store/doc.go`, contra el `doc.go` de cada paquete y contra las invariantes de `et/CLAUDE.md`.

## Alcance

**Revisado:**

- **Paquetes internos:** `page`, `vfs`, `crashfs`, `smgr`, `control`, `wal`, `xact`, `buffer`, `overflow`, `metric`.
- **Archivos de `store/`:** `engine`, `checkpoint`, `recovery`, `batch`, `snapshot`, `store`, `durable`, `space`, `backup`, `restore`, `archive`, `replica`, `slots`, `walsince`, `decode`, `ddl`, `ddlbatch`, `empty`.

**Pendiente**, para la segunda parte, después de la tarea 70: `btree`, `fsm`, `vacuum.go`, `verify.go`, `storecheck.go`.

**Cómo se confirmó:**

- Cada hallazgo dice si está **confirmado** (con una prueba o leyendo el código que lo rodea) o si es **plausible**.
- Las pruebas temporales se corrieron con `go test -p 1` y se borraron. No queda ningún archivo en el repo.
- Las pruebas de todos los paquetes revisados pasan.

## Resumen

| Gravedad | Hallazgos |
|---|---|
| Crítico | 1 |
| Alto | 6 |
| Medio | 11 |
| Bajo | 23 |
| Estilo | 9 |
| **Total** | **50** |

### Los 5 más importantes

> **Estado (2026-11-20):** el crítico y los 6 altos están arreglados, cada uno con su prueba de regresión (tareas 88–94 de F-23; detalle debajo de cada hallazgo). Los medios, bajos y de estilo siguen pendientes, salvo la parte de DropSlot de R6.

1. **Crítico, confirmado con prueba.** Con `SYNC_ON_WRITE=false`, el commit de un lote con DDL no espera el fsync, pero el catálogo y los archivos sí se cambian de forma durable. Un corte borra un store y sus datos durables, aunque el lote quedó abortado. `Verify` da limpio. Está en `ddlbatch.go:312` y `durable.go:26`.
2. **Alto, confirmado con prueba.** Una escritura suelta espera a otra transacción con `Store.mu` y `closeMu` tomados. Esa espera cierra un ciclo con `DropStoreIn`, `DropStore` y `Close`. Con `LOCK_TIMEOUT=0` el interbloqueo es permanente; con un límite, todo el motor se frena hasta que vence. Está en `store.go:279` y `batch.go:353`.
3. **Alto, confirmado con prueba.** Un registro dañado en un segmento sellado se corta en silencio, y `cutTail` borra los registros durables que siguen. `doc.go` promete que `Open` falla. Está en `wal/scan.go:72-82` y `wal/log.go:100`.
4. **Alto, confirmado por lectura.** El flush de la víctima del reemplazo toma `content.RLock()` sin condición mientras quien pide la página tiene otras bloqueadas. Esto puede cerrar un ciclo con una partición del árbol, y el checkpoint se cuelga detrás. Está en `buffer/replace.go:61` y `buffer/flush.go:44`.
5. **Alto, confirmado por lectura.** El reciclaje del WAL puede borrar segmentos que el archivo `slots` guardado todavía promete. Las réplicas y los lectores lógicos quedan invalidados tras un corte. Está en `checkpoint.go:70-73` y `slots.go:411-435`.

**Los otros altos:**

- `smgr/manager.go:262-271`: un fsync del checkpoint se hace con el candado del fork tomado. Bloquea `Extend` y `Read` de páginas con candados del árbol tomados. Confirmado con prueba.
- `decode.go:230,272-302`: el flujo lógico diverge con un DDL deshecho dentro de un lote. Confirmado con prueba.

---

## Detalle por paquete

Formato de cada hallazgo: **[gravedad, estado]** `archivo:línea`, con qué pasa, un escenario y una sugerencia.

### store/ — núcleo (engine, checkpoint, recovery, batch, snapshot, store, durable, empty, ddl, ddlbatch)

**N1. [Crítico, confirmado con prueba] `ddlbatch.go:312` y `durable.go:26`: el KindCommit con DDL no espera el fsync cuando `SYNC_ON_WRITE=false`.**

- **Qué pasa:** `finishDDL` llama a `b.logEnd(commit, false)`, y `syncCommit` solo frena (`deferredBrake`) cuando `!e.cfg.syncOnWrite`. Después, `applyDDL` guarda un catálogo durable y borra los archivos del store (`removeFiles`).
- **Escenario** (crashfs `LoseUnsynced`):
  1. `s.Put(x)` y un `Checkpoint`.
  2. Un lote con `t.PutIn(y)`, `DropStoreIn("s")` y `Commit`.
  3. Un corte y la reapertura.
  - **Resultado:** `Stores()=[t]` y falta `t.y`. El store `s` se perdió y el lote quedó abortado. `Verify` da limpio.
  - **El caso inverso:** un store creado en un lote abortado sobrevive.
- **Sugerencia:** en `finishDDL`, cuando hay commit, esperar siempre `WaitDurable(lsn)`; por ejemplo, con un parámetro «forzar» en `logEnd` o `syncCommit`. Agregar `syncOnWrite=false` a `TestDDLBatchCrash`.

> **Arreglado (tarea 88, 2026-11-20).** `finishDDL` confirma con `syncAlways` (nuevo `syncMode` en `durable.go`: `syncConfig`, `syncDeferred`, `syncAlways`): el KindCommit de un lote con DDL espera siempre su fsync, también con `SYNC_ON_WRITE=false` y con `CommitDeferred`. Prueba: `TestDDLBatchCrash` ganó la variante `nosync=true` (`SYNC_ON_WRITE=false` + `CommitDeferred`, un corte en cada operación de disco × 3 modos); sin el arreglo falla con «the state is neither before nor after: stores [a c]».

**N2. [Alto, confirmado con prueba] `store.go:279` (`writeLoop` → `txs.Wait`) y `batch.go:353` (`batchLoop`, también con `b.mu`): se espera una transacción con `Store.mu` y `closeMu` tomados.**

- **Escenario (a):**
  1. El lote B hace `PutIn(S,x)`.
  2. La escritura suelta W hace `S.Put(x)` y espera a B con `S.mu.R`.
  3. B hace `DropStoreIn(S)`. Su `markDrop` (`ddlbatch.go:184`) pide `S.mu.Lock` con `b.mu` y `ddlMu` tomados.
  - **Con `LOCK_TIMEOUT=2s`:** todo el motor queda 1,8 s detenido, incluidos un `Store()` ajeno y el paso 3 del checkpoint, y W recibe un ErrConflict espurio.
  - **Con `LOCK_TIMEOUT=0`:** `config.go:88` lo acepta y el interbloqueo es permanente. `expire` necesita `b.mu`, así que el temporizador del lote no lo destraba.
- **Escenario (b):** `Close` pide `closeMu.Lock` y entonces el `enter()` del `Commit` de B queda bloqueado.
- **Escenario (c):** un `DropStore` suelto, que tiene `ddlMu`, espera `S.mu`, mientras el `finishDDL` de un lote espera `ddlMu`.
- **Sugerencia:** soltar `s.end()` antes de `txs.Wait` y volver a entrar al repetir, revisando `dropped`, `file` y `tree`. Como mínimo, no aceptar `LOCK_TIMEOUT=0`. Corregir también «Orden de los bloqueos» en `doc.go`.

> **Arreglado (tarea 89, 2026-11-20).** Una escritura suelta (`write` → `writeRound`/`writeLeaf`) o de un lote (`writeIn` → `writeInRound`/`batchLeaf`) que choca con otra transacción libera su desborde, sale del store y del motor (`Store.mu`, `closeMu`) y, en un lote, suelta `b.mu` (`Batch.waitOther`); espera y repite desde la entrada, como `XactLockTableWait`. `waitDDL` usa la misma espera; `abortLocked` es idempotente. `LOCK_TIMEOUT=0` sigue aceptado: ya no hay ciclo. Pruebas (`lockwait_test.go`, `LOCK_TIMEOUT=0` y un plazo propio): `TestLockWaitDropStoreIn` (a), `TestLockWaitClose` (b), `TestLockWaitDropStore` (c), `TestLockWaitBatchClose` y `TestLockWaitBatchRollback`; las cinco se colgaban sin el arreglo.

**N3. [Medio, confirmado con prueba] `engine.go:195` y `checkpoint.go:92-101,255`: Close queda detrás de un checkpoint de fondo que reparte sus escrituras.**

- **Qué pasa:** si un `Checkpoint()` del usuario espera `ckptMu` con `closeMu.R` tomado, `Close` no puede tomar `closeMu` hasta que el checkpoint de fondo termine. Ese checkpoint solo se apura con `e.stop`, que `Close` cierra después. El Lock pendiente bloquea todo `enter()`.
- **Medida:** con `checkpointTimeout=6s`, Close tardó 4,8 s. En producción puede llegar a unos 4,5 min con el motor detenido.
- **Sugerencia:** cerrar `stop` (o un canal propio para apurar el checkpoint) antes de `closeMu.Lock`. Que `Checkpoint()` también apure al de fondo.

**N4. [Medio, confirmado por lectura] `ddl.go:296-302`: `DropStore` suelto no mira los lotes que escribieron en el store.**

- **Qué pasa:** el `Commit` posterior del lote devuelve nil y sus escrituras se pierden en silencio. `DropStoreIn` sí devuelve ErrBusy por `batchWriters`.
- **Sugerencia:** el mismo chequeo, o documentarlo como decisión. Se relaciona con la tarea 86, la del DDL en lote según 73 y 74.

**N5. [Bajo, plausible] `empty.go:20` y `ddl.go:286`: `Empty` y `DropStore` no pasan por `writable()`/`admit()`.**

- **Qué pasa:** un `ErrNoRoom` de `Reserve` hace `e.fail` y el motor queda de solo lectura, en lugar de devolver un ErrDiskFull limpio. Justo `DropStore` es lo que se usa para liberar espacio.
- **Sugerencia:** llamar a `e.writable()` antes del registro.

**N6. [Bajo, confirmado] `batch.go:278-281`: `txError` incluye `ErrThrottled`, que nunca llega.**

- **Qué pasa:** `throttle` va antes de `acquire`. El comentario dice que el freno aborta el lote, y no es así.
- **Sugerencia:** quitarlo de `txError` o corregir el comentario.

**N7. [Estilo] Comentarios desactualizados.**

- `recovery.go:20` dice «pasos 1–5», pero se hacen los 9 de `doc.go`.
- `recovery.go:162`: el `nextXID` no menciona el frozenXID.
- `engine.go:28-34`: la tabla de candados duplica la de `doc.go` y le falta `Batch.mu → closeMu → ddlMu → commitMu`. Conviene remitir a `doc.go`.

**N8. [Estilo] Código repetido.** `store.go:337-368` (`delete`/`remove`) y `batch.go:408-436` (`deleteIn`/`removeIn`) son la misma escritura y solo cambian el valor que devuelven. Se pueden unificar.

**Lo que está bien:**

- **Checkpoint:** el punto de rehacer y los contadores se toman bajo `writeMu` con `hintGate`. `waitChanges` va antes de `FlushAll`. `commitMu` actúa como `delayChkpt`. El catálogo se guarda antes del registro y el control después del fsync.
- **Recuperación:** cada Kind del 1 al 47 tiene su `Redo`, y crear, borrar y vaciar son idempotentes por número de archivo. `countLSN` evita contar dos veces. `nextXID` es el máximo de control, el último xid visto + 1 y frozen.
- **Reservas del WAL:** toda reserva se escribe, también en los caminos de error.
- **Close:** detiene los temporizadores y las goroutines.
- **Memoria:** está acotada.

### store/ — respaldo, archivo, réplica, ranuras y flujo lógico (space, backup, restore, archive, replica, slots, walsince, decode)

**R1. [Alto, confirmado con prueba] `decode.go:230,272-302`: el flujo lógico diverge con un DDL deshecho dentro de un lote.**

- **Qué pasa:** `storeDDL` toma todo `KindDropStore` como un DDL suelto con xid 0, lo entrega enseguida y borra `r.names[file]`. Pero ese registro también lo escriben, con xid 0:
  - `dropOwnCreate` (`ddlbatch.go:200`);
  - `undoDDL` (`ddlbatch.go:352`);
  - `createIn` con el disco lleno.
- **Escenario:**
  1. Un lote con `StoreIn(x)`, `PutIn(x,k)`, `DropStoreIn(x)` y `Commit`.
  2. Otro lote con `StoreIn(y)` y `Rollback`.
  - **Lo que entregó el flujo:** `[drop:x, create:x, put:/k, drop:y]`.
  - **Resultado:** el consumidor queda con `{x:{}, "":{k:v}}`, mientras que el motor está vacío.
- **Sugerencia:** si el drop es de un archivo creado por un `KindCreateStoreIn` que sigue en el búfer, quitar del búfer de ese xid el alta y sus cambios, y no entregar el drop. La otra opción es escribir ese drop con el xid del lote o con una marca.

> **Arreglado (tarea 94, 2026-11-20).** El lector anota qué archivos crea cada lote sin confirmar (`creating`, `txBuffer.creates`). Un `KindDropStore` de xid 0 sobre uno de ellos cancela el store dentro de ese lote (`txBuffer.canceled`: al confirmar se saltan su create y sus cambios, también los derramados), y sobre uno de un lote ya abortado se descarta (`ghosts`); ninguno sale como DDL suelto. Prueba: `TestChangesBatchDDLUndo` (el escenario del informe, con un valor en desborde y un lote que crea y confirma; sin el arreglo, el consumidor queda con 4 stores y el motor con 2).

**R2. [Alto, confirmado por lectura] `checkpoint.go:70-73` y `slots.go:411-435`: el reciclaje puede pasar lo que promete el archivo `slots` guardado.**

- **Qué pasa:** `checkpointSlots` suelta `slots.mu` antes de `log.Recycle`. Hay dos caminos al problema:
  - un `AdvanceSlot` o un `Confirm` que llega en medio mueve la retención solo en memoria;
  - un disco lleno limpio al escribir `slots.tmp` se ignora, y `Recycle` corre igual con las retenciones adelantadas en memoria.
- **Consecuencia:** si hay un corte antes del siguiente checkpoint, `loadSlots` encuentra el `RestartLSN` en un segmento reciclado y la ranura queda invalidada (ErrRemoved). La réplica necesita un Backup nuevo y el lector lógico pierde su flujo. `doc.go` dice que lo que se recicló siempre sale de lo guardado.
- **Sugerencia:** retener según los valores guardados, no según los de memoria. La otra opción es mantener `slots.mu` hasta después de `Recycle`; el orden `ckptMu → slots.mu → writeMu` es válido. Con el disco lleno, no reciclar.

> **Arreglado (tarea 92, 2026-11-20).** Una retención de ranura (`wal.Hold`) solo pasa a otro segmento después de guardar el archivo `slots` que la promete (`moveHold`): `AdvanceSlot`/`Confirm` guardan antes de moverla (una vez por segmento de avance) y el paso 6 guarda y recién entonces mueve todas. Con un disco lleno al guardar, las retenciones quedan en lo guardado y `Recycle` no las pasa. `DropSlot` guarda antes de soltar la retención (y si no puede, la ranura sigue: cubre la parte de DropSlot de R6). Prueba: `TestSlotSavedRetention` (guardados de `slots` con disco lleno, checkpoint, corte y reapertura: la ranura sigue válida y su WAL se lee; sin el arreglo el WAL ya no estaba).

**R3. [Medio, confirmado por lectura] `decode.go:146-149`: `ChangeReader.next` llama a `fn` con `closeMu.RLock` tomado, y el doc dice que `fn` puede leer.**

- **Qué pasa:** un `Read` dentro de `fn` mientras `Close` espera en `closeMu.Lock` es un RLock anidado detrás de un escritor pendiente. Close y el lector quedan colgados para siempre.
- **Sugerencia:** soltar `closeMu` alrededor de `fn`, como hacen los recorridos.

**R4. [Medio, confirmado por lectura] `slots.go:181-189`: `waitConsistent` espera hasta TRANSACTION_TIMEOUT (600 s) dentro de `enterSlots`, o sea con `closeMu.RLock` tomado.**

- **Qué pasa:** el comentario dice «sin candados», y no es cierto. Un `Close` en medio congela el motor: los `Commit` que se esperan no pueden entrar.
- **Sugerencia:** soltar `closeMu` durante la espera.

**R5. [Medio, confirmado por lectura] `decode.go:363-365,596`: el nombre del derrame es `hex(slot)-nextFile.spill`, y `nextFile` arranca en 1 en cada lector.**

- **Qué pasa:** dos `Changes(name)` abiertos a la vez sobre la misma ranura abren el mismo archivo con `O_TRUNC` y mezclan los valores sin error. Además, `removeSpills` del segundo borra los del primero.
- **Sugerencia:** marcar la ranura activa (ErrBusy para un segundo lector), o usar un nombre único por lector.

**R6. [Medio, confirmado por lectura] `slots.go:295-298` y `196-199`: un DropSlot o un CreateLogicalSlot con el disco lleno dejan la memoria distinta del archivo, sin `dirty`.**

- **DropSlot:** la ranura borrada reaparece al reabrir y retiene WAL; un nuevo DropSlot devuelve ErrSlotNotFound.
- **CreateLogicalSlot:** se vuelve al `confirmed` de antes de la espera, y pueden salir transacciones a medias.
- **Sugerencia:** marcar `dirty` o deshacer el cambio en memoria cuando falla el guardado.

**R7. [Bajo, confirmado] `slots.go:453`, `pressure.go:43-45` y `space.go:159`: `slotRetained` toma `slots.mu`.**

- **Qué pasa:** ese candado se tiene durante `SegmentOf` (que lee el disco) y durante fsync + rename + fsync del directorio. Por eso `Pressure` y `throttle` pueden esperar un fsync, aunque `doc.go` dice que `Pressure` solo lee atómicos.
- **Sugerencia:** publicar en un atómico el segmento retenido más viejo.

**R8. [Bajo, confirmado] `decode.go:128-130`: los nombres de los stores vaciados (`Empty`) antes de abrir el lector se pierden.** Los cambios salen con `Store ""`. Es el mismo mecanismo que `doc.go` documenta para DropStore, pero para `Empty` no está documentado. Hay que documentarlo o tomar el nombre de `KindEmptyStore`.

**R9. [Bajo, confirmado] `replica.go:245-249`: pasado el plazo, `quiesceSnapshots` hace `continue` sin dormir.** Es una espera activa. Además, `cancelSnapshots` cuenta en `RecoveryConflicts` aunque `release` no haga nada.

**R10. [Bajo, confirmado] `restore.go:133`, `archive.go:317` y `slots.go:427`: `n, _ := f.ReadAt(...)`.** Un EIO se informa como ErrBackupIncomplete o ErrCorrupt. Hay que propagar todo error que no sea EOF.

**R11. [Bajo, plausible, solo con datos propios dañados] El derrame del búfer de reordenamiento no valida lo que lee.**

- `decode.go:463`: el largo leído no se valida antes de `make`, así que puede reservar hasta 4 GB.
- `decode.go:445`: `le.Uint32(data)` entra en pánico con menos de 4 bytes.
- `assemble` puede quedar en un bucle sin fin si hay un ciclo de trozos vacíos.

**R12. [Bajo, plausible] `wal/stream.go:68` (`Locate`) y `SegmentOf`: un segmento reciclado entre `listSegments` y `firstLSN` devuelve ENOENT crudo, no ErrRemoved.** Por eso `WalSince` no da ErrWalRemoved.

**R13. [Estilo]**

- `backup.go:360` repite el formato `%010d.seg`; conviene usar `wal.SegmentName`.
- `slots.go:176`: un nombre inválido devuelve ErrSlotLimit, lo que confunde.

**R14. [Estilo] Lo que dice la documentación y no hace el código.**

- `doc.go` dice que `WalSince` de una ranura da ErrWalRemoved, pero `WalSince` no recibe ranura.
- El comentario de `waitConsistent` dice «sin candados» (ver R4).

**Lo que está bien:**

- **Respaldo en caliente:**
  - `ForceImages` va antes del checkpoint de inicio.
  - La retención se toma con `ckptMu`.
  - El catálogo y la lista de `data/` se toman juntos con `ddlMu`.
  - `KindBackupEnd` se escribe y se espera durable.
  - Hay fsync de cada archivo y de cada directorio, y `backup_label` es atómico.
  - `unforce` y `unhold` van con `sync.Once`.
- **Restauración y PITR:** se valida todo antes de cambiar nada, y `StopLSN` corta.
- **Archivo continuo:** `archive_status` es atómico y se guarda antes de mover la retención.
- **Réplica:** el registro va al WAL local antes del redo, y lo que llega por la red pasa por el CRC.
- **Búfer de reordenamiento:** la cuenta de memoria es consistente, y la poda en `KindCheckpoint` es correcta.

### store/internal/wal

**W1. [Alto, confirmado con prueba] `scan.go:72-82` y `log.go:100`: un registro inválido en un segmento sellado se corta en silencio.**

- **Qué pasa:** solo se detecta si el segmento siguiente ya tiene un registro válido con un lsn mayor. Si el siguiente existe pero está vacío, `scan` lo toma por el final del WAL y `cutTail` trunca en el registro dañado. Eso pasa con un segmento preasignado por `prepareNext`/`rotate`, con un `Switch` sin escrituras después o con el sellado al abrir con el archivo continuo.
- **Prueba:**
  1. Se escriben 10 registros y se esperan durables.
  2. `Switch` y `Close`.
  3. Se invierte un byte del registro 5.
  - **Resultado:** `Open` devolvió `records=4, err=nil`. Se perdieron 6 confirmaciones durables y la copia archivada dejó de coincidir con el WAL local.
- **Atenuante:** hace falta un daño en el medio, como bit rot o una escritura perdida por el disco.
- **Sugerencia:** anotar en el control el último segmento sellado y su fin, y fallar si el corte cae antes. La otra opción es fallar si después del corte quedan más de `MaxRecordSize` de bytes no nulos en un segmento que no es el último con datos. Como mínimo, corregir `doc.go:824,1243`.

> **Arreglado (tarea 90, 2026-11-20).** Marca de sellado (`wal/seal.go`, formato en `wal/doc.go`): al sellar un segmento, después del fsync de sus registros, se escriben en sus últimos 32 bytes su número y el fin de sus registros (con CRC) y otro fsync; los registros terminan antes (`capacity`). `Open` (`checkSealed`) falla con `ErrCorrupted` sin cortar nada si el final válido cae antes del fin que promete la marca de su segmento o si queda un segmento sellado después (salvo el corte de `StopLSN`). Un segmento reciclado conserva la marca de su número viejo, que no vale. Pruebas: `TestSealedCorruptionFails` (el escenario del informe, con y sin preasignación), `TestSealedAfterEndFails`, `TestTornTailActiveStillCut` (la cola rota del activo, también reciclado, se sigue cortando).

**W2. [Medio, plausible] `log.go:218-227`: un pánico entre `Reserve` y `Write` (en `underLock` o en `applyBlock` sobre una página dañada) deja un lsn reservado que nunca se escribe.**

- **Qué pasa:** `inflight` y `lastLSN` ya subieron. `published` se estanca, todo `WaitDurable` cuelga y `Close` cuelga en `seal`. Solo importa si algún llamador recupera el pánico.
- **Sugerencia:** en `Reserve`, un `defer` con `recover` que haga `fail` y `inflight.Done()`. En `logChange`, un `defer` que haga el `Write` pendiente.

**W3. [Bajo, confirmado] `log.go:215,452-457`: en modo respaldo, `Reserve` decide la imagen contra `imageLSN()` = max(rehacer, forceFrom), pero `btree/record.go:61` y `fsm/record.go:169` la arman contra `RedoPoint().LSN`.**

- **Qué pasa:** entre `ForceImages` y `BeginCheckpoint` hay un giro de CPU con páginas bloqueadas en exclusivo, que reintenta `ErrNeedImage`. `wal/doc.go:121` y `wal.go:96` describen la regla vieja.
- **Sugerencia:** exportar `ImageLSN()` y usarla en los dos llamadores.

**W4. [Bajo, confirmado] `archive.go:70` y `firstLSN`: las lecturas reservan búferes de más.**

- **Qué pasa:** `SegmentInfo` crea un scanner nuevo (1 MiB + 64 KiB) en lugar de usar el pool, hasta 256 veces por `pendingArchive` (unos 270 MB de basura), y relee segmentos ya anotados. `firstLSN`, al que llaman `Locate` y `SegmentOf` en cada `slots.hold`, lee hasta 1 MiB por segmento.
- **Sugerencia:** usar `getScanner`; leer solo el encabezado con `ReadAt`; si el segmento ya está anotado (`known`), no abrirlo.

**W5. [Bajo, confirmado; figura «a confirmar» en el doc] `log.go:96-100`: lo que se rehace cuenta como durable antes del fsync del último segmento.**

- **Qué pasa:** ese fsync recién llega en `cutTail`. Durante el redo, `walProxy` (`counts.go:85-135`) da todo por durable, así que puede expulsarse una página con un lsn que no tiene fsync.
- **Escenario:** el proceso se cae, se reabre y el sistema operativo se cae en medio de la recuperación.
- **Sugerencia:** en `Open`, hacer fsync del segmento final antes de rehacer.

### store/internal/xact

**X1. [Bajo, plausible] `status.go:314-344,584`: `getSlot` revisa `oldest` solo al entrar, no después de `cond.Wait()`.**

- **Qué pasa:** un `Status` que esperaba en medio de un `Truncate` vuelve a crear con `O_CREATE` un archivo `0001` ya borrado. `Recover` lo corrige en la próxima apertura.
- **Otra ventana:** existe una ventana mínima para un `nil` en `syncFiles` (`status.go:502-511`).
- **Sugerencia:** volver a comprobar `pageNo < pageOf(oldest)` en cada vuelta de `getSlot`.

**X2. [Bajo, confirmado] `status.go:459`: `readPage` crea el archivo de cualquier xid consultado.**

- **Qué pasa:** un xid basura que sale de una página dañada crea archivos `xact/NNNN` lejanos. Esto contradice dos documentos: `xact/doc.go` («leer más allá del final da ceros») y Verify («no cambia nada»).
- **Sugerencia:** abrir sin `O_CREATE` y tratar «no existe» como ceros.

**Lo que está bien:**

- La caché es fija: 128 ranuras.
- «WAL antes de la página» se cumple en `writePage`.
- Los hint bits se ponen solo con un commit durable.
- El mutex se suelta en toda E/S.
- `Truncate` va con `SyncDir`.

### store/internal/metric

**M1. [Estilo] `histogram.go:65-70`:** `Count` sale de las cubetas y `Mean` del campo `count`, así que con lecturas concurrentes pueden diferir. Conviene quitar `count`.

### store/internal/buffer

**B1. [Alto, confirmado por lectura] `replace.go:61` → `flush.go:44`: `victim` hace `flush` con `d.content.RLock()` incondicional.**

- **Qué pasa:** `clockSweep` deja la víctima en el mapa con su tag viejo. Entre la elección y el `RLock`, otra goroutine puede fijarla con `pinExisting` (`pool.go:209-229`) y bloquearla en exclusivo.
- **Escenario:**
  1. T1 parte la hoja L: tiene L en exclusivo y está en `fsm.Allocate`, con la meta en exclusivo.
  2. T1 hace `Pin(head)`, la víctima es V (sucia) y se queda esperando `RLock(V)`.
  3. T2 tiene V en exclusivo, parte y espera la meta.
  - **Resultado:** un ciclo. La variante sin la meta es una hoja V a la izquierda de L.
  - **El checkpoint se cuelga detrás:** T1 tiene `V.io`, así que `FlushAll` también se bloquea.
- **Contradicción con la documentación:** `doc.go:697-701` dice que esa espera «no puede cerrar un ciclo». Solo cuenta la espera al WAL.
- **Sugerencia:** en el camino de `victim`, usar `TryRLock` y, si falla, soltar y elegir otra víctima, como `LWLockConditionalAcquire` en BufferAlloc. Corregir `doc.go`.

> **Arreglado (tarea 91, 2026-11-20).** `flush(d, try)`: el reemplazo (`victim`) usa `io.TryLock` y `content.TryRLock`; si la víctima está tomada devuelve `errVictimBusy`, se suelta y se elige otra (como `LWLockConditionalAcquire` en BufferAlloc). `FlushAll` y el escritor de fondo siguen esperando (no tienen páginas). `doc.go` («Orden de los bloqueos») y `buffer/doc.go` corregidos. Prueba: `TestVictimBusySkipped` (la víctima sucia con el contenido tomado; sin el arreglo, `Pin` queda colgado).

**B2. [Bajo, confirmado] `pool.go:155-166`: si `Extend` funciona y luego falla el `pin` (ErrNoBuffers o la escritura de la víctima), queda un bloque huérfano.** Repair lo recupera. Además, `extendMu` es global y se tiene durante el flush de la víctima, o sea durante `WaitDurable`, así que todas las extensiones de todos los stores esperan un fsync. Sugerencia: elegir la víctima antes de `Extend` o fuera del candado.

**B3. [Estilo] Comentarios desactualizados.**

- `buffer.go:92`: el comentario de `Pool.Close` no dice que libera el arena.
- `buffer.go:52` y `desc.go:109`: `MarkDirty(lsn)` ignora `lsn`.

**B4. [Estilo] `pool.go:30,76` y `flush.go:42`:** el `sync.Pool` guarda un `page.Page`, que es un slice, y cada `Put` reserva memoria (SA6002). Conviene guardar `*[page.Size]byte`.

**Lo que está bien:**

- **Escritura:** «WAL antes de la página» se cumple contra el pd_lsn de la copia. La copia se hace con el candado compartido. `justDirtied` y el bit de sucio están bien.
- **Lectura y mapa:** una lectura fallida sale del mapa, y dos lecturas no cargan dos búferes.
- **Reemplazo:** el clock-sweep termina con ErrNoBuffers.
- **Atómicos:** son tipados y alineados.
- **Cierre y candados:** Close y el último unpin liberan el arena sin SIGSEGV. El orden io → content → mu se respeta.

### store/internal/overflow

**O1. [Medio, confirmado con prueba temporal] `manager.go:280-307` (`readPage`): `Read` y `Free` no comprueban el dueño (`PruneXID`).**

- **Qué pasa:** un `next` que salta a la página k+1 de otra cadena, donde las páginas del medio miden igual, pasa la validación.
- **Prueba:** `Read` devolvió datos mezclados sin error, y `Free` liberó páginas de la cadena B, todavía viva. Eso corrompe la lista de libres.
- **Qué dice la documentación:** `doc.go:14-17` promete ErrBadChain ante «un salto a otra cadena».
- **Cuándo pasa:** solo con un `next` dañado.
- **Sugerencia:** pasar el owner a `Read` y `Free`. Los llamadores lo tienen: `store.go:451`, `store.go:221`, `batch.go:333`. Validar en una sola función compartida con `ownPage`, que hoy repite casi las mismas comprobaciones (`manager.go:206-232`).

**O2. [Bajo, confirmado] `manager.go:88-104`: si `Reserve` falla, `writePage` suelta la página que dio `Allocate` sin devolverla.** Queda huérfana hasta un Repair. Sugerencia: devolverla con `m.free.Free`.

**O3. [Bajo, plausible] `redo.go:51-66`: `ensureBlock` no acota `b.Block` y extiende página por página pasando por la caché.** Sugerencia: poner un tope y usar `smgr.Extend` directamente.

**O4. [Bajo, confirmado] `manager.go:284`: `Read` de un valor grande usa `buffer.Normal`.** Un valor de 64 MiB son unas 8.200 páginas, que vacían la caché. Sugerencia: usar `Ring` cuando pase el tamaño del anillo, como en `Pages`.

**Lo que está bien:**

- Los recorridos están acotados por `pagesOf(ref.Len)`, así que un ciclo no da un bucle infinito, y las longitudes se validan.
- `FreeTail` libera de forma reanudable, y no se libera dos veces.
- El redo es idempotente.
- `writePage` siempre escribe la reserva.

### store/internal/smgr

**S1. [Alto, confirmado con prueba] `manager.go:262-271`: `Sync` mantiene `st.mu.RLock` del fork durante `syncSegs`, o sea durante el fsync.**

- **Qué pasa:** `Extend` (`:154`) pide `Lock`. Se llega ahí desde una partición del árbol, por `fsm.Allocate` → `PinNew` → `smgr.Extend`, con páginas bloqueadas. Como el RWMutex da preferencia al escritor que espera, también se bloquean todos los `Read` y `Write` posteriores del fork.
- **Prueba:** con un fsync de 600 ms, `Extend` esperó 500 ms y un `Read` 449 ms. Esto viola «nunca esperar un fsync con una página tomada».
- **Además:** `Extend` hace fsync bajo el lock exclusivo cada vez que nace un segmento de 1 GB. Su doc lo explica, pero `store/doc.go` no lo registra como excepción.
- **Sugerencia:** tomar los descriptores (`refs++`) bajo el RLock, soltar el candado y hacer el fsync fuera. Que `drop` y `unlinkFork` esperen a `refs==0`.

> **Arreglado (tarea 93, 2026-11-20).** `Sync` toma los descriptores con el fork en compartido y hace los fsync con el fork suelto (`acquireSegs` + `syncHandles`); `fileCache.drop` (Unlink) no cierra un descriptor en uso: lo marca y lo cierra su último `release`. El fsync de `Extend` al nacer un archivo de 1 GB queda como la única excepción, documentada en `store/doc.go` y `smgr/doc.go`. Prueba: `TestSyncOutsideForkLock` (con el fsync trabado, `Extend`, `Read`, `Write` y `Unlink` del fork no esperan; sin el arreglo, `Extend` espera).

**S2. [Medio, código confirmado y disparo plausible] `manager.go:563-576,581-591`: `SyncAll` puede devolver nil después de un fsync fallido.**

- **Qué pasa:** `evict` anota `m.failed` cuando falla su fsync, pero solo lo ven las llamadas que empiezan después.
- **Escenario:** en Linux (errseq), el `syncSegs` del checkpoint abre un descriptor nuevo y ve éxito. El checkpoint avanza el punto de rehacer y las páginas se pierden.
- **Sugerencia:** terminar `Sync` y `SyncAll` con `return m.check()`.

**S3. [Medio, confirmado] `filecache.go:94-101,186-190` y `manager.go:519`: el fsync de la expulsión de un descriptor sucio corre en la goroutine que lo suelta.**

- **Qué pasa:** esa goroutine puede ser un `Read` de otro store, todavía con el candado de su fork y con el búfer en E/S. Con más de 128 stores activos, una lectura paga el fsync de un archivo ajeno.
- **Sugerencia:** que `takeVictims` saltee los descriptores sucios.

**S4. [Bajo, confirmado] `manager.go:223-229`:** se mira `n < page.Size` antes que `err`, así que un EIO sale como `ErrShortFile`, sin envolver. Sugerencia: mirar primero el error.

**S5. [Bajo, confirmado; camino sin uso] `manager.go:171-190`:** un `Extend` que salta dos o más segmentos crea el intermedio antes del fsync del anterior. Sugerencia: exigir `block == NBlocks`.

**S6. [Bajo, plausible] `manager.go:444-453`:** `state()` crea una entrada del mapa `forks` para todo FileNo consultado, y solo `unlinkFork` la borra. El mapa crece con los números de archivo a lo largo de la vida del motor, no con las claves.

**Lo que está bien:**

- Los offsets son correctos, y `load` valida los segmentos de 1 GB.
- Al nacer un segmento, el fsync del anterior va antes que el de `data/`.
- `Unlink` borra del último al primero y hace `SyncDir`.
- La caché LRU nunca cierra un descriptor con `refs > 0`.
- Un fsync fallido no se reintenta.

### store/internal/crashfs

**C1. [Medio, plausible: da falsa confianza] `file.go:111-113` (`dropPending`) y `crash.go:69-85`:** tras un Sync fallido, las escrituras pendientes nunca llegan a `durable`, así que ningún corte posterior las conserva. En un disco real pueden haber llegado en parte. Ese caso, que `doc.go` admite («Qué se pierde en solo lectura»), nunca se prueba. Sugerencia: una lista de escrituras «quizá» que `Crash` aplique por sector.

**Lo que está bien:**

- Los sectores son de 512 B o 4 KB.
- Se conserva un prefijo ordenado de los cambios por directorio, y un nombre no es durable sin SyncDir.
- Un Sync fallido no vuelve durable nada después.
- La misma seed da el mismo disco.

### store/internal/control

**K1. [Bajo, plausible] `file.go:33-35`:** las dos copias de 512 B están en el mismo bloque de 4 KB. En un disco 4Kn, o en un SSD sin protección ante cortes, se pueden romper las dos a la vez. crashfs trata el bloque de 4 KB como atómico y no lo prueba. Sugerencia: poner la copia B en el byte 4096.

**K2. [Bajo, confirmado] `codec.go:59-63,137-144`:** `Control.Slots` es código muerto; `doc.go` lo da por reservado y en cero. `decodeControl` acepta valores distintos de cero. Sugerencia: exigir ceros o quitar el campo.

**K3. [Bajo, confirmado] `catalog_codec.go:88-114` y `catalog_file.go:96,111`:** `encodeCatalog` no aplica `maxCatalogSize` y `readAll` sí, así que se puede guardar un catálogo que después no abre. Hacen falta unos 3,8 M de stores. Sugerencia: validar el tamaño al codificar.

**Lo que está bien:**

- Las dos copias alternan por `seq % 2` y gana la válida con mayor seq.
- Si las dos están rotas, da `ErrNoValidCopy`.
- El catálogo se guarda con tmp, fsync, rename y fsync del directorio.
- `entryCount` se acota antes de reservar memoria.

### store/internal/vfs y page

**V1. [Estilo] Referencias rotas.** `vfs/os.go:81` dice «ver lock_*.go», un archivo que no existe; el código está en `fs_unix.go`. En `crashfs/doc.go:7-8` hay una frase cortada.

**page:** sin hallazgos. Todos los accesos están acotados aun con un encabezado o punteros dañados. `Repair` es correcto. El checksum CRC-32C con el número de bloque, plegado a 16 bits, coincide con el doc.

**vfs:** F_FULLFSYNC en macOS funciona también sobre directorios; se comprobó en APFS. En Linux, fdatasync. EINTR se reintenta. `MkdirAll` hace fsync de los padres. `flock` va con LOCK_NB.

---

## Funciones más largas (sin tests, dentro del alcance)

| # | Líneas | Función | Ubicación |
|---|---|---|---|
| 1 | 73 | `Engine.recover` | `store/recovery.go:25` |
| 2 | 63 | `wal.Open` | `store/internal/wal/log.go:68` |
| 3 | 55 | `manager.Extend` | `store/internal/smgr/manager.go:150` |
| 4 | 53 | `ChangeReader.replay` | `store/decode.go:426` |
| 5 | 51 | `Engine.checkpointLocked` | `store/checkpoint.go:36` |
| 6 | 51 | `ChangeReader.next` | `store/decode.go:144` |
| 7 | 50 | `decodeControl` | `store/internal/control/codec.go:100` |
| 8 | 47 | `pool.load` | `store/internal/buffer/pool.go:268` |
| 9 | 46 | `Engine.restartpoint` | `store/replica.go:364` |
| 10 | 46 | `scan` | `store/internal/wal/scan.go:43` |
| 11 | 45 | `decodeCatalog` | `store/internal/control/catalog_codec.go:120` |
| 12 | 42 | `open` | `store/engine.go:145` |
| 13 | 41 | `WAL.Recycle` | `store/internal/wal/log.go:407` |
| 14 | 41 | `decodeSlots` | `store/slots.go:521` |
| 15 | 38 | `decodeRecord` | `store/internal/wal/record.go:124` |

Ninguna pasa de 75 líneas. Las de decodificación son largas por la secuencia de campos, no por la lógica. Las que más conviene partir son:

- `Engine.recover`: un paso por función, igual que los 9 pasos de `doc.go`;
- `manager.Extend`: separar el nacimiento de un segmento;
- `ChangeReader.replay`.
