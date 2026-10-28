# Plan de estabilidad y consistencia de `et`

Siete revisiones (`et`, `jsql`, drivers, `tcp`/`websocket`/`store`, HTTP/infra, orquestación, utilidades) con el mismo criterio: código entendible y mantenible por un humano, **cambios pequeños y seguros, sin refactor**. `go vet` sale limpio en toda la librería; `staticcheck` marca ~35 avisos menores. No he aplicado nada de esto; el detalle con `archivo:línea` de cada punto está en `la sección «Detalle» de este archivo`.

Reglas que seguiré al aplicar: público con errata → nombre nuevo y el viejo como alias `Deprecated` (los consumidores no se tocan); privado → renombrar; muerto público → `Deprecated`; muerto privado o roto sin usos → borrar; cada spec se actualiza con su código; verificación con build/vet/staticcheck, tests con `-race`, batería en seis motores, verificadores de `tcp`/`websocket` y compilación de core-studio, postick, josefina y tick.

## Estado de ejecución (2026-10-27, sin commit)

Aplicadas las fases 0, 1, 3 y 5. Las fases 2 y 4 no se pidieron y no se tocaron. Cada grupo cerró con `go build ./...`, `go vet ./...`, `gofmt -l`, `go test -race` de los paquetes tocados, la batería de drivers (sqlite, postgres, oracle, mysql, mssql; josefina se saltó con `-skip /josefina` a pedido, para revisarla aparte), `go run -race ./test/tcp` (26/26, 70/70), `go run -race ./test/websocket` (15/15, 17/17, 32/32) y la compilación de core-studio/api, postick/api, josefina y tick/api.

- **Fase 0:** aplicados todos los bugs salvo **B9** (`OverLimit` del gateway), que sigue esperando la decisión: activar el 429 por `REQUESTS_LIMIT` o quitar el campo. B4 se hizo con el arreglo mínimo (`COALESCE` en la ruta `et.AGGREGATE`); unificar los dos agregadores de pg/sqlite es Fase 2. B10 trajo un ajuste extra: `setDone` limpia el error del intento anterior (si no, un reintento exitoso devolvía el error viejo). B26 borró `middleware/cors.go` (0 usos) y B28 `utility.More`, `UUIndex` y `utility/list.go` (rotos, 0 usos). El driver josefina recibió el mismo escape de usuario/contraseña que postgres (`url.UserPassword`; `QueryEscape` convertía un espacio en `+`).
- **Fase 1:** renombres con alias `Deprecated` para lo público (consumidores sin tocar) y directos para lo privado; archivos movidos con `git mv` (`router/handler.go`, `event/emitter.go`, `create/prompts.go`, `create/template/statefulset_template.go`, `aws/session.go`, `cmd/watcher`). `UseAutentication` se renombró sin alias (0 usos). Mensajes escritos a mano pasados a constantes; 18 `MSG_TCP_*` del `jtcp` y los duplicados de `msg/` quedan `Deprecated`; 9 `MSG_*` sin uso de `jsql` se quitaron. `MSG_DB_NOT_FOUND` y `MSG_RECORD_NOT_FOUND` de `jsql` se conservan (los usa `stores`).
- **Fase 3:** comentarios, idioma, invariantes y specs. Bloque `Concurrency` sobre `jsql.Model`; orden de locks del `Hub` verificado en el código (`sendMu` sí anida `Client.mu`, a diferencia de lo que decía el plan). Nuevo `TestWrapperDeclared` (`jsql/jsql_test.go`): cada clave de `wrapper.go` debe estar en `jsql.d.ts`. `doc/COMPONENT_CATALOG.md` regenerado desde `go doc` (sin `jtcp` ni `jws`). Saltados por ser código, no comentario: la sentencia muerta `ddlTable(model)` de `postgres/load.go`, el parámetro `dx` de `Items.Str`, la clave `"update_at"` de `event/workState.go`, el texto `"username"` de un error de `claim`, y los textos en español que son cadenas (`cmd/daemon/msg.go`, `cmd/install`, `utility/pid.go`, meses de `timezone`).
- **Fase 5:** movimientos hechos con un movedor por AST (declaración + comentario, sin reescribir): O1 en `et/et` y O2 en `jsql`. O3 ya estaba (import de sqlite); el orden entre drivers es Fase 2.
- **staticcheck:** 44 avisos, todos de categorías previas (22 `U1000` de código muerto de la Fase 4, 8 `S1025`, 4 `ST1001`, 4 `ST1005`, 3 `SA1019`…); ninguno nuevo de esta sesión.

## Fase 0 — Bugs de comportamiento confirmados (32, cada uno ≤ 20 líneas)

Los que afectan a core-studio/josefina hoy:

| Id | Qué pasa | Arreglo |
|---|---|---|
| B1 | `Count()` con `OrderBy` falla en postgres (pg/sqlite emiten `ORDER BY` en `COUNT(*)`) | retorno temprano como mysql/mssql/oracle |
| B2 | Oracle `is`/`is_not` con valor no es null-safe | `DECODE(a, b, 1, 0) = 1` |
| B3 | DSN de postgres no escapa la contraseña; `chain`/`defaultChain` duplicadas | una `chain` con el escape que ya usa josefina |
| B4/B5 | pg/sqlite: `SUM` sin `COALESCE` por una de las dos rutas; `between`/`in` citan con el `Quoted` del núcleo | un agregador único por driver; `Quoted` del driver |
| B6 | `ettp/v2/events.go` suscribe los canales V_1 dos veces y deja los V_0 sin suscriptor | 2 constantes |
| B7 | `event.Log` ignora el canal: `event.Error` nunca sale por `event:set:error`, que core-studio escucha | 1 línea |
| B8 | `middleware.PrintPrettyStack` imprime el panic sin frames | `new(string)` |
| B10 | `resilience.Run` no espera los reintentos: todo paso con `total_attempts > 0` falla con "no result" y se re-ejecuta solo 30 s después | `Run` síncrono |
| B12 | `validator.Pattern` compila en cada validación (panic con regex inválida); `Min(0)`/negativos no aplican | compilar una vez, `hasMin`/`hasMax` |
| B14 | binding `Json` de los scripts es un mapa compartido entre corridas | objeto nuevo por runtime |
| B19 | `utility.ValidStr/ValidIn/ValidInt` nunca excluyen la lista (el proxy de ettp/v1 deja pasar `"new"`, `"-1"`) | `slices.Contains` |
| B20 | lo que genera `cmd/create` no compila (`jsql.Def`, `Query` sin `.All()`, `PackageName` duplicado) | 4 líneas |

Y 20 más en paquetes sin consumidor o con 0 usos (B9 `OverLimit` del gateway nunca se asigna —**decidir**: activar el 429 o quitarlo—; B11 mensajes con `%s` faltante; B13 `ia.LoadIndex` + `Ask` hace panic; B15/B16/B17/B18 `dt`, `resilience.readInstance`, `stores.ErrorSetAuthor`, `ephemeral` con un solo timer; B21 `jwsp` reply de video y `tumbnail_`; B22 `envar.Load` invertido y caché con otra clave; B23 `stdrout` en Windows; B24 `file.MakeFile` devuelve la carpeta; B25 ocho bugs en `jia`; B26–B32 menores).

## Fase 1 — Nombres y mensajes
- **Erratas públicas** con alias: `IsDeferent`, `ArgWhitAs`, `cache.Objet*`, `jwt.DeleteTokeByToken`, `claim.ParceToken`, `UseIdenpotency`, `UseAutentication`, `ErrrStepNotFound` y familia, `TypeComercial`, `GetExtencion`, `GenUUId`, `UnityMiligram`, `SELECt`, `DaskSpace`… y archivos (`hadler.go`, `promps.go`, `statefulset_tempalte`, `sesion.go`, `cmd/whatcher`). `PgConection` y `ToWheRes` **no** (tienen usos; solo comentario).
- **Privados**: `runing`, `runResilence`, `loadAgend`, `deFault`, `muIdx`→`muReplace`, closure `condition` que oculta la función del paquete.
- **Mensajes a constantes**: ~30 textos escritos a mano donde ya existe el `MSG_*` (drivers, middleware, ettp, jrpc, crontab, tcp); duplicados y derivas en/es en `msg/`, `jsql`, `jwf`, `jia`, `service`; 18 `MSG_TCP_*` del `jtcp` eliminado.

## Fase 2 — Duplicados pequeños (~−600 líneas, sin cambio de comportamiento)
- `et`: una sola lista de formatos de fecha (hoy tres), `const pathSep`, `Wheres.Add` usa `AddCondition`, tabla de funciones, `Get`→`ValAny`, `hidden()`→`Json.Hidden`, `jsonOfValue` ×4, `conditionScalar` ×3.
- `jsql`: `setScript` ×6, `addIndex` ×4 (y cierra la invariante copy-on-write), `runTrigger*` ×12 bucles, `addToSection`, `renderSQL`/`scalarTx`, `toIndexes`, regex a variables.
- drivers: ~150 líneas de helpers **idénticos byte a byte** en los cinco (`NestSource`, `ArrayElemType`, `ConnectWithRetry`, `masterRows`…) a `jsql`; nombres y orden alineados para que un arreglo se repita por paralelismo.
- HTTP: CORS ×4 → una; `response.GetBody` → `request.GetBody`; `event.Message.Serialize`≡`Encode`; `cache.Empty` reutiliza el borrado por lotes.
- Orquestación: `addAuditLog` ×6, claves `instance:<id>` ×5, `scope()` ya existe pero no se usa en `ia`.
- Utilidades: sustitución `{{key}}` ×5 → `strs.Parse` que ya existe; `isNew` ×6 en `reg`; regex compiladas por llamada.
- `tcp`/`websocket`/`store`: `call()` ×3, `closeReason`, `close`→`closeWith(nil)`, `sortSubs` ×3, `segment.read` inalcanzable.

## Fase 3 — Comentarios, idioma, invariantes y specs
- ~80 comentarios que describen otra función o parámetros que no existen; ~250 exportados sin comentario (el que más falta: `ettp/v2.Config`); español en archivos en inglés (drivers incluido `josefina/driver.go`, `utility`, `file`) e inglés en `store`/`tcp`.
- **Invariantes en un solo lugar**: bloque `Concurrency` sobre `jsql.Model` (hoy en siete sitios y con una imprecisión: índices y scripts no son copy-on-write), orden de locks del `Hub`, claves de Redis que escribe la librería (tabla única), reglas de `router`/`event`/`jwt`, variables de entorno sin documentar (`REQUESTS_LIMIT`, `LATENCY_LIMIT`, `JWT_EPHEMERAL_DURATION`…).
- **Specs desfasadas**: `store/spec.md` nombra constantes que ya no existen; `tcp`/`websocket` apuntan a `internal/msg`; `jsql/spec.md` dice `jrex`; `CLAUDE.md` dice que `_idx` es UUID (es `UnixMilli`), que core-studio usa `stores` (ya no), `SECRET` con default (hoy `Fatal`), `cmd/daemon` como binario (no es `package main`).
- `jsql/catalog.go`: índice dice 200 exportados, hay 206; ~12 exportados fuera del catálogo; test que verifique `wrapper.go` ↔ `jsql.d.ts`.

## Fase 4 — Código muerto
~120 símbolos públicos con 0 usos en los cuatro consumidores → `Deprecated` (listas por paquete en el plan); borrar solo lo privado y lo roto sin usos: `utility.More/List/iterate/cmds`, `file.WatcherPath`, `cmd/daemon`, `server/api.go`, `router/hadler.go`, 164 líneas de Goji en `middleware/wrap_write.go`, 7 campos U1000 de `jwf`, `service/service.go`.

## Fase 5 — Orden interno
Mover funciones al archivo donde un humano las busca (`asJson`→`json.go`, referencias a campos→`condition.go`, `QueryDetail`/`QueryRollups`→`relation.go`, métodos de `Model` fuera de `command.go`…). Solo `git mv` de código y `gofmt`.

## No recomendado ahora
Anotado en el plan, no se hace: funciones largas, firmas públicas (`PgConection`, `WheRes`, `ValidStr`, `NewClient` con error nil), partir archivos grandes, unificar `Query`/`Command`, eliminar `ettp/v1` o los paquetes sin consumidor, un solo limitador de peticiones.

**Decisiones que necesito de ti:** B9 (activar o quitar el 429 por `REQUESTS_LIMIT`) y si en la Fase 4 prefieres `Deprecated` en todo lo público o borrar directamente lo que no tiene ningún uso en el workspace.

---

# Detalle

# Plan de estabilidad y consistencia de `et` (2026-10-27)

Objetivo: estabilidad y consistencia de lo existente. Sin partir archivos, sin cambiar firmas públicas, sin reescribir funciones.
Detalle con `archivo:línea` en review_et.md, review_jsql.md, review_drivers.md, review_net_store.md, review_http.md, review_orch_util.md (en `doc/review-2026-10-27/`).
Reglas de ejecución: públicos con errata → nombre nuevo + viejo como alias `// Deprecated: use X`; privados → renombrar directo; muerto público → `Deprecated`, muerto privado o roto sin usos → borrar; cada paquete con spec (et, jsql, store, tcp, websocket) se actualiza en el mismo cambio; verificar con build/vet/staticcheck, `go test -race ./et/ ./jsql/ ./jwf/ ./cache/ ./validator/ ./ia/ ./request/ ./queue/`, batería 6 motores, `go run -race ./test/tcp` y `./test/websocket`, y compilación de core-studio/api, postick/api, josefina, tick.
Ya aplicado: DB_IS_DEBUG en newDB y "id" en SqliteConection.GetParams.

## Fase 0 — Bugs de comportamiento confirmados (cada uno ≤ 20 líneas; decidir uno a uno)
B1  jsql/drivers postgres+sqlite: COUNT(*) emite ORDER BY/LIMIT (postgres falla con OrderBy) → retorno temprano IsCount + guardia !IsExists (drivers #4). + caso TestCatalog.
B2  oracle IS/IS_NOT con valor no null-safe → DECODE (drivers #5).
B3  postgres DSN sin escapar y chain/defaultChain duplicadas → una chain con escape de josefina (drivers #6).
B4  pg/sqlite SUM sin COALESCE por la ruta et.AGGREGATE → pgAggregate/sqliteAggregate únicos (drivers #2).
B5  pg/sqlite BETWEEN/IN con Quoted del núcleo (bool→true, time sin pgQuoteTime) → Quoted del driver, un case (drivers #9).
B6  ettp/v2/events.go:32,37 bloque V_0 suscribe canales de V_1 (remove/reset ×2, canales V_0 sin suscriptor) → 2 constantes (http #2).
B7  event/handler.go:260 Log ignora `event` (Error nunca sale por event:set:error, que core-studio escucha) → asyncMsg{event,data} (http #3).
B8  middleware/recoverer.go PrintPrettyStack imprime sin frames (`var w *string` nil) → new(string) (http #1).
B9  middleware/telemetry.go:104 Metrics.OverLimit nunca se asigna (429 del gateway nunca) → DECIDIR: copiar flag o quitar campo (http #5).
B10 resilience Run() no espera reintentos: jwf siempre MSG_RESILIENCE_NO_RESULT y re-ejecuta el paso en background → Run síncrono + vaciar Result (orch #1). Test temporal.
B11 jwf/instance.go:674-683 Errorf con MSG sin %s; jia/ia.go:497,532 igual (orch #2).
B12 validator Pattern MustCompile por validación (panic) y Min/Max solo > 0 → compilar una vez, hasMin/hasMax (orch #3). Actualizar CLAUDE.md.
B13 ia LoadIndex + IngestFile/Ask → panic func nil → guardia (orch #4).
B14 jwf/wrapper.go:108 binding Json compartido entre corridas → Binder por runtime (orch #5).
B15 dt.Get devuelve nil con error; Byte()/Json() fallan tras JSON → newObject + cases (orch #6).
B16 resilience readInstance devuelve nil,false tras leer; clave sin prefijo → return result,true + instanceKey (orch #9).
B17 stores ErrorSetAuthor vs trigger errors.New distinto → ErrRecordExists compartido (orch #10).
B18 ephemeral un timer para todas las claves + escritura fuera del lock → timers map (orch #11).
B19 utility.ValidStr/ValidIn/ValidInt/ValidNum nunca excluyen (Contains con la lista como un elemento) → slices.Contains (util #1). 48 usos en consumidores, casi todos []string{""} → sin cambio para ellos; ettp/v1 proxy empieza a rechazar "new"/"-1" como pretendía.
B20 create/template: jsql.Def→Define, Query(...).All(), quitar ModelhRpc PackageName, DB_TENANT_ID → el scaffold compila (util #2).
B21 jwsp SendReplyVideoMessageByURL firma (to, messageID, url, caption); "tumbnail_"→"thumbnail_" (util #3,#4).
B22 envar.Load `!= nil` invertido; envar/get.go ToUpper antes del lookup (util #5,#16).
B23 stdrout Windows: mapa colors antes del blanqueo; Traces filtra "ErrorM"/"ErrorF" (son Errorm/Errorf); logs.go:66 if siempre cierto (util #15).
B24 file.MakeFile devuelve carpeta en vez de archivo; Remove registra al revés (util #24). create/progress.go acumula; CmdProject.Use "micro" (util #25).
B25 jia: Get/Delete con args invertidos y colecciones plural/singular; skill.go ValidStr sin `!`; conversation.go bucle duplica mensajes, LastMessage nil; agent.go Sprintf sin %s, error ignorado; router args corridos (orch #7,#8). Paquete sin consumidores.
B26 claim.NewAuthenticationToken args en orden equivocado (0 usos) → corregir (http #6). jrpc/handler.go:77 return faltante (http #7). middleware/cors.go AllowAll con credenciales (0 usos) → borrar (http #4).
B27 service/storage.go Path solo al borrar; mapas sin mutex; send.go nil,nil (orch #23).
B28 utility.More/UUIndex/List.Remove rotos y sin usos → borrar (util #6,#7). cmds.RunSSH = RunOS → Deprecated delegando; Load error (util #8).
B29 jwf Flow.ToJson omite app_id/save_instances; Instance.ref omite flow_tag/name (orch #25).
B30 store: empty() pierde el error de RemoveAll; FileStore.ID nunca asignado → ID: name (net #19,#8).
B31 tcp/hub.go:343 compara err.Error() → sentinelas + errors.Is (net #16).
B32 request/response: Metrics.JSON Ok ==200 vs 2xx; 6 ramas `&dt == (&et.Item{})` muertas → IsStatusOk + borrar (http #16).

## Fase 1 — Nombres (erratas) y mensajes
N1  et: IsDeferent→IsDifferent (+alias), Json.Exist delega en IsExist (Deprecated); closure `condition`→conditionOf; fldCondition if redundante (et #14,#20).
N2  jsql: ArgWhitAs/ArgWhitSchema → ArgWith* (+alias; también wrapper.go y jsql.d.ts); `deFault`→defaultValue; jquery.go toJson libre→jsonOf; Query.page alias de setPage → uno; muIdx→muReplace y quitar de getIdx (jsql #15,#11).
N3  http: cache.Objet*→Object*, jwt.DeleteTokeByToken→DeleteTokenByToken, claim.ParceToken→ParseToken, router UseIdenpotency→UseIdempotency (todos +alias; consumidores no se tocan); UseAutentication→UseAuthentication en server/api.go, ettp/v2, interfaz router.Router; event.NewEvenMessage, EventEmiter/Emiter, jrpc imputs, router/hadler.go→handler.go (http #8).
N4  orch: ErrrStepNotFound/ErrrFlowNotFound/ErrrTriggerNotFound/ErrorInstanceNotFound → Err* (+alias); service TypeComercial/TypeAutentication (+alias); privados runing/runResilence/resilence, loadAgend; MSG_STEPER_NOT_FOUND Deprecated (tag steper_id NO) (orch #17,#18).
N5  util: SELECt, ExtractMencion, GetExtencion (+alias, 2 usos en aws), DaskSpace/ApendAny/RemoveAcents, GenUUId→GenUUID, UnityMiligram, MkMolue→MkModel, promps.go→prompts.go Promp*→Prompt*, statefulset_tempalte→template (3 sitios), aws/sesion.go→session.go, cmd/whatcher→cmd/watcher, "bolean", "clave de secret" (util #11).
N6  store: IsDebug() setter → añadir Debug(), IsDebug Deprecated; actualizar CLAUDE.md/README store (net #17).
M1  Mensajes a constantes: drivers "database is required"×7, "service_name is required"×2 → MSG_ATRIB_REQUIRED; 5 MSG_* nuevos en jsql/msg.go (QUERY_NO_FROM, NO_COLUMNS_INSERT, NO_COLUMNS_UPDATE, REFUSE_WITHOUT_WHERE, UNSUPPORTED_COMMAND) (drivers #7). middleware/authentication.go ×3 → constantes de middleware/msg.go corregidas; event "panic in Queue channel"; ettp/v2 "path %s is invalid", "router not found", "error llamando al servicio"; jrpc "solver not found"; claim; server "📌 Rutas cargadas" (http #10). crontab "job not found" → MSG_JOB_NOT_FOUND (orch #12). tcp/security.go:334 → MSG_TCP_TLS_NO_CERT + fila spec §11 (net #15).
M2  msg/msg.go duplicados (MSG_PACKAGE_NOT_FOUND≡ERR_, METHOD, NOT_CACHE_SERVICE≡CACHE_NOT_LOAD) → igualar + Deprecated; MSG_ERR_INVALID_CLAIM, MSG_ERR_AUTORIZATION textos; 18 MSG_TCP_* del jtcp eliminado → Deprecated (http #9, net #7). jsql/msg.go MSG_BRIDGE_MODEL_REQUIRED sin traducción; 9 MSG_* sin uso → quitar (jsql #9). jwf/jia/service msg.go derivas en/es (orch #21).

## Fase 2 — Duplicados pequeños, regex y helpers (sin cambio de comportamiento)
D1  et: dateLayouts única en json.go (ValTime, parseDate; Time() Deprecated); const pathSep "->" ×5; Wheres.Add → AddCondition; ToWheRes comentario; ToOperator mapa a var de paquete + FIELD en TypeValues; Function.Str/ToFunction tabla; hidden()→Json.Hidden; Get→ValAny; jsonOfValue ×4; equalsAny bool + equalScalar lo usa; conditionScalar ×3 (mantener matiz eq/objeto; test antes); ValArray→listOf; fieldValue→pathValue (o comentario en Value.Fields) (et #1-#13).
D2  jsql: setScript ×6; addIndex ×4 (con slices.Clone + muDef → cierra invariante); runTriggerFuncs/runTriggerScripts ×12 bucles; addToSection (and/or); renderSQL + scalarTx (exists/count/all); borrar idxColumn, defineColumn usa newColumn; toIndexes + defineRollupDef en DB.define; rollupWhere/rollupJsonOf; defaultSerieFormat; regex fieldAs/schemaTable a vars; quitar DefineSource repetido en defineTenantModel/defineProjectModel (jsql #1-#7,#16,#17,#19,#20).
D3  drivers: helpers idénticos a jsql (NestSource, ArrayElemType, IsJsonType, MasterRows, ConnectWithRetry, Model.ColumnType; sección Utilidades de catalog.go) −150; consistencia: my/ms ColumnRefValue→JoinColumnRef, refs→isJoin, masterRows en my/ms, pgAggExpr switch sobre et.COUNT, oraOrderExpr, findField muerto ×3, pgValueCast a query.go, sqlite orden Conds, forma compacta de relaciones en pg/sqlite (opcional), pg/sqlite QuoteAny (opcional) (drivers #1,#3,#10,#11,#12,#15,#16).
D4  http: CORS única en middleware.AllowAll (cuerpo de v2); event.Message.Serialize Deprecated, cache.Message borrar; IntToTpHeader Deprecated + ettp/v2 TpHeader alias de router; response.GetBody → request.GetBody + comentario GetParam/URLParam; cache.IncrDuration Deprecated, Empty reutiliza scanDelete; cache/msg.go MSG_ATRIB_REQUIRED → msg.; Subscribe sin conexión devuelve MSG_ERR_NOT_CONNECT; HGetCtx .Result(); staticcheck (hostName, os, loaded, return, select) y chequeo SO (http #11-#15,#18,#21).
D5  orch: appendAudit por paquete ×6 + notify ×2; instanceKey/stopKey jwf ×5; ia scope() en 4 selects + DeleteDocument transaccional; validateArray* → validateArrayLength; mem arm(); stores/instances.go triggers repetidos; jwf/flow.go code sin uso; MANUAL=TRIGGER_MANUAL; crontab channelOf ×2; dot imports de stores → jsql.; stores.Instance.Get (false,nil) (orch #13-#16,#12,#19,#22).
D6  util: {{key}} ×5 → strs.Parse; file.joinSep→strs.Append; xls/csv IndexOf → slices.Index; reg isNew ×6; reg.UUID llama utility.UUID (ciclo); Normalize/ExtractMencion/validate regex a vars; Titlecase rama muerta; S1025 Sprintf("%s") ×8; x/exp/slices → stdlib; xls.Add copia slice; envar SetFloat (+SetNumber alias); logs.Warnf; aws/s3 consts; cmd/install solo installContext; cmd/create error de modTidy (util #9,#10,#18,#19,#20,#22,#23).
D7  tcp/websocket/store: call() ×3 en tcp; closeReason en tcp; Client.close→closeWith(nil); sortSubs ×3; errNotFound ×2; validMessage; compacting atomic.Bool; segment.read muerto; Deferred a store.go; peer.toJson; NewTimer en websocket client; staticcheck de test/* (net #2,#9-#14,#18,#23,#24).

## Fase 3 — Comentarios, idioma, invariantes y specs
C1  Comentarios desfasados (listas en cada review: et #15, jsql #10,#14, drivers #14, http #19, orch #20, util #12, net #21). Solo texto.
C2  Exportados sin comentario, 1–2 líneas en el estilo del archivo (et #16, jsql #14, http #20 —ettp/v2.Config el más importante—, orch #20, util #13 —jwsp, create—, net #22).
C3  Idioma: español en archivos en inglés (drivers #13 incl. josefina/driver.go; util #21; et #15 SetNested/Caso; jsql #13 STAR/HasStar) y viceversa en store/tcp (net #20).
C4  Narrativos → resumidos (et #17; drivers #14 Quoted; jsql withEntry ya hecho).
C5  Invariantes en un solo lugar: jsql bloque Concurrency sobre type Model (qué protege muSchemas/Schema.mu/muAudit/muDef, lista de campos copy-on-write y los que no, drivers sin lock, orden de locks) (jsql #11); http bloque por paquete: router (event.Load/cache.Load antes de registrar rutas), event (una suscripción por canal, no cancela la previa), cache (tabla de claves Redis y formatos), jwt (Validate nil,nil; token distinto borra clave), claim (SECRET Fatal), ettp/v2; env no documentadas (REQUESTS_LIMIT, LATENCY_LIMIT, JWT_EPHEMERAL_DURATION, PRODUCTION, COMPANY, WEB, HELP, LANG) (http #23,#24); orch doc.go por paquete (claves Redis/TTL, mutex, qué persiste) (orch #24); tcp Hub orden de locks en hub.go + spec §14.2 (net #4); et checklist para operador/función/tipo nuevo en Operator/Function/TypeData (et #23); índices 1-based vs 0-based en Items.One/Wheres.One (et #21).
C6  Specs/docs desfasadas: store/spec.md MSG_STORE_* y paso 5; tcp/websocket spec `internal/msg`→`et/msg`; tcp/brecha.md certificados y §13 casos 22-26; websocket spec decisión jws, brecha.md fecha; jsql/spec.md jrex→jwf (:41,:289), Command sin modelo por driver §9.2/9.3, tabla archivo→contenido §2; CLAUDE.md `_idx` (UnixMilli, no UUID), `stores` ya no lo usa core-studio, SECRET sin default 1977, cmd/daemon no es binario, jia bug; et/spec.md §3.2 coerción fechas; jsql connection.go DriverJosefina "pending B5" + case en getConnection (net #1,#3,#5,#6; jsql #10; orch; util).
C7  jsql catalog.go: banner §1, contadores (206), wrappers no delgados (Query.AddCondition, DB.LoadModel), exportados fuera (Replace, ToArrayAny, Bind, TimeoutContext, TestConnection, Probe, ConnectionFromJson, HasStar/StarFields/SourceHiddens) o nota en encabezado; privados→privados (mecánico al tocar) (jsql #12,#13,#18). Test TestWrapperDeclared wrapper.go↔jsql.d.ts (jsql #24).

## Fase 4 — Código muerto
X1  et públicos 0 usos → Deprecated: GetItem, Json.Query, ArrayBytes, ApplyToIndex, Wheres.IsDebug/Workers, Source.Data, Prefixer, Value.Raw, Exp, ToConnector, ErrorDataNotFound, SafeMap; privados borrar (isDebug, Workers, 3 MSG) (et #19).
X2  jsql: TypeJoin/JOIN…/Joins Deprecated; isChanged ×2, EVENT_MODEL_SET, Calc.Module, ConnectParams ya resuelto (jsql #8,#9).
X3  http (*=borrar seguro): server/api.go*, router Router/hadler.go/storage.go*, cache/router.go, event/router.go (conservar HttpEventPublish)*, event/emiter.go, workState.go, ErrorDetails/Context*; Deprecated: cache IsNil, SetH/D/W/M/Y, *Verify, Collection*, FromId, HealthCheck ×2, Conn.Pub/Sub/Unsub; middleware wrap_write.go, request_id.go, logger.go:73-206, NewRpcMetric/DoneRpc, msg.go; response Scan*/DATA/ANY/HTTPAlert/Stream/InternalServerError; request GetStr, Status.ToJson/ToString, Value.Int… (comentario trampa); claim SetPayload; ettp/v2 StringToTypeRouter; ettp/v1 doc.go "legacy" (http #17).
X4  orch: U1000 jwf 7 campos + Flow/WorkFlow isDebug/isChanged (borrar); crontab running; jia privados muertos; service/service.go borrar; públicos sin uso Deprecated (GetArgs, KindList, StepStatusList, FlowStatusList, FlowDefinition, Node; MSG_* sin uso agrupados) (orch #21,#22).
X5  util (borrar solo roto*): utility More*/UUIndex*/List*/GeneratePortNumber*, iterate*, cmds* (o Deprecated), file.WatcherPath* (cmd/watcher usa NewWatcher), timezone.FormatMDYYYY*, cmd/daemon*, reg snowflake init/dep, utility CFB → AES-GCM o borrar (0 usos), EncryptMD5/HashSHA256 Deprecated; el resto de la lista util #14 → Deprecated (strs 19 funciones, envar, logs, stdrout, create Make*/Delete*, jwsp/event.go).
X6  net: MSG_TCP_* del jtcp (ya en M2); segment.read (ya en D7).

## Fase 5 — Orden interno (solo mover + gofmt)
O1  et: asJson/sameMap→json.go; parseDate/dateLayouts→json.go; Field.*/pathValue/fieldRef y jsonValue/toConditionValue→condition.go; getBetweenRange junto a ToBetween; Command/WheRes con los tipos; públicas de evaluate.go al inicio; deepEqualValue/equalScalar al bloque privado (et #18).
O2  jsql: withoutOmitUpdates/isOmitUpdate/restoreOmitUpdates→model.go; QueryDetail/QueryRollups+métodos→relation.go; Index→define.go (jsql #21).
O3  sqlite/connect.go import en grupo; drivers consistencia de orden (ya en D3).

## No recomendado ahora (anotado, no se hace)
et: funciones largas (ToJS, Wheres.All, ToCondition, valueType, applyFunction, ScanRows, Joingy, MatchLike, fromJS); API (ApplyToValue tabla, WheRes→Conditions, In/NotIn firma, Where dup de Eq, atribs, Str/String, Items.One base).
jsql: pipeline insert/update/delete; partir DB.define/loadQuery/setRollup/wrapModel; ConnectionFromJson por tipo; Model.Replace; Query/Command sqlBase; PgConection→PgConnection.
drivers: FieldExpr/ScalarExpr, Froms, capa común de condiciones, Command sin modelo en mysql/mssql, QuotedAs pg/sqlite, exportar postgres.chain, partir query.go.
http: eliminar ettp/v1; Metrics.* con response.*; partir cache/handler.go; request.Value parseando; logger.go; SetRouter(TpHeader); retirar nombres con errata; Subscribe cancelando previa; un solo limitador; msg.go locales a msg/.
orch: ia.ingest con IngestText; jia persistencia; steper_id; resilience con contexto; mem decode[T]; crontab EVENT_* a campos; service Storage; validator orden errores; partir jwf/instance.go; eliminar paquetes sin consumidor.
util: jwsp orden de params; units tabla; partir utility.go; firmas ValidStr/xls.Add/logs.Warn/envar.Get; brevo struct; stdrout Writer; Quote/Unquote; ModelhRpc; Watcher.Load.
net: NewClient/NewServer error; paquete común tcp/websocket; `recover` privada; partir store.go; Hub.close; config; hub receive.
