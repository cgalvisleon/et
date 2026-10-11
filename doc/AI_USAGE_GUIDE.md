# AI_USAGE_GUIDE.md

> Guía de bolsillo para asistentes de IA que escriben código sobre `github.com/cgalvisleon/et`. Verificada contra el código el 2026-10-10. Firmas exactas en `COMPONENT_CATALOG.md`; arquitectura en `ARCHITECTURE_SUMMARY.md`; contexto general en `LIBRARY_CONTEXT.md`.

## 1. Orden de decisión

1. **Usa** un componente de `et` tal cual. Busca primero en `COMPONENT_CATALOG.md`.
2. **Extiende** un punto de extensión: `jsql.Driver`, un `Store` de paquete, `TriggerFunction`, `CalcFunction`, un middleware.
3. **Adapta**: si la firma no calza (por ejemplo, un `Store`), escribe un adaptador delgado.
4. **Implementa** con la librería estándar, siguiendo los patrones de `et` (`et.Json`, `Load()` idempotente, `msg.go`).
5. **Agrega una dependencia externa** solo con justificación explícita.

Si terminas en 4 o 5, dilo en la respuesta: «No usé `et` para X porque Y».

## 2. Necesito X → uso Y

| Necesito | Paquete | Entrada |
| --- | --- | --- |
| Datos JSON dinámicos | `et` | `et.Json{}`, `.Str(keys...)`, `.ValStr(def, keys...)` |
| Resultado paginado | `et` | `et.List` |
| Conectar a una base | `jsql` | `jsql.Load()` + `import _ ".../jsql/drivers/<driver>"` |
| Probar una conexión sin crear la base | `jsql` | `jsql.TestConnection(ctx, connection, query)` |
| Definir una tabla | `jsql` | `db.Define(jsql.Define{...})` o `db.DefineModel(schema, name, version, userId)` |
| Consultar | `jsql` | `model.Where(jsql.Eq(...)).And(...).Limit(page, rows)` |
| Consulta descrita en JSON | `jsql` | `db.Query(et.Json{"select": ..., "from": ...})` |
| Transacción | `jsql` | `tx := jsql.NewTx()` + `ExecTx(tx)` + `tx.Commit()` |
| Consecutivos | `jsql` | `jsql.DefineSeries(db, schema)` + `GenSerie(tag)` |
| API HTTP de un servicio | `router` | `router.NewApi(...)` + `Protected(group, method, path, name, handler)` |
| Servidor HTTP mínimo | `server` | `server.New(name, port)` |
| API gateway | `ettp/v2` | `ettp.New(name, &ettp.Config{...})` (llama tú `cache.Load`/`event.Load`) |
| Leer el request | `request` | `request.GetBody(r)`, `request.URLParam(r, "id").Str()` |
| Responder | `response` | `response.ITEM/ITEMS/LIST/HTTPError(w, r, status, ...)` |
| Llamar otra API | `request` | `request.Fetch(method, url, header, body, timeout, def)` |
| Validar un payload | `validator` | `validator.New().Field("x").Required()...Validate(body)` |
| Caché / Redis | `cache` | `cache.Set`, `cache.Get` → `(valor, existe, error)` |
| Pub/Sub | `event` | `event.Publish`, `event.Subscribe` |
| JWT | `jwt` | `jwt.NewAuthorization(...)`, `jwt.Validate(token)` |
| IDs | `reg` | `reg.UUID()`, `reg.GetULID(id)` |
| Variables de entorno | `envar` | `envar.GetStr/GetInt/GetBool(name, def)` |
| Logs | `logs` | `logs.Info`, `logs.Errorf`, `logs.Alert` |
| Lotes asíncronos | `queue` | `queue.New[T](size, maxEvents, period, handler)` |
| Cron | `crontab` | `crontab.Load(tag, store)` + `CronJob` / `ScheduleJob` |
| Reintentos | `resilience` | `resilience.New()` + `LoadInstance(Params{...})` + `Run()` |
| Workflow | `jwf` | `jwf.New(id)` + `NewFlow(...).Step(...)` + `wf.Run(...)` |
| Script JavaScript | `jrex` / `jwf` | `jrex.Load(tag, store)`, `jwf.RunCode(code, ctx, bindings)` |
| RAG sobre documentos | `ia` | `ia.Load(db, schema, ia.Config{})` + `IngestFile` + `Ask` |
| Agentes OpenAI | `jia` | `jia.New(tag, store, userId)` |
| Email/SMS/WhatsApp | `brevo`, `infobip`, `aws`, `jwsp`, `service` | ver `COMPONENT_CATALOG.md` |
| Excel / CSV | `xls`, `csv` | `ReadXls*`, `NewCsv(data).ToHttp(w, name)` |

## 3. Trampas verificadas

- **`cache.Get` devuelve tres valores** `(valor, existe, error)`; `cache.ErrNotFound` ya no existe. Una clave inexistente es `def, false, nil`.
- **`jwf.New(id)`** recibe solo el id. No hay `jwf.Load`, `jwf.Store` ni `DefineStore`: persiste con `OnChange`. El método es `NewFlow` (ya no `NewFloW`) y recibe `ownerId`.
- **`resilience.New()`** no recibe store.
- **`router.Api.Public/Session/Protected`** reciben `group` primero: `(group, method, path, name, handler)`.
- **`jsql` acepta cinco drivers**: `postgres`, `sqlite`, `mysql`, `mssql`, `oracle`. Hay que importar el driver por efecto lateral.
- **Tablas sin esquema**: en sqlite, mysql y oracle la tabla se llama `schema_nombre`. El SQL crudo debe usar ese nombre (`main_tasks`).
- **`router.Publish`/`router.With`** ya no anteponen `Route.PackageName` al `Path`: la ruta debe venir completa.
- **`jsql` exige esquema** (`MSG_SCHEMA_REQUIRED`); en sqlite las tablas son `schema_nombre` desde v1.0.36.
- **`jsql.LoadTo(dbName, hostName...)`**, no `LoadTo(name)` a secas.
- **`jsql.Define`** es el struct de definición (no `jsql.Def`).
- **No existen** `jval/`, `config/`, `jsql.Store`, `stores.Catalog`, `ws/`, `wsp/`, `tcp/`, `workflow/`, `instances/`.
- **`stores.Instance.Get(id, dest)`** tiene una sola clave: no cumple los `Store` de `crontab`, `jia` ni `jrex`.
- **`jia.Load`** tiene los argumentos de `store.Get` invertidos (`jia/ia.go:90`).
- **`dt` con `PRODUCTION=false`** no persiste nada.
- **`cmds.RunSSH`** ejecuta localmente.
- **`utility.More`** no acumula: reinicia el contador.
- **`jsql.LoadDb`** entra en pánico hoy (brecha B9 de `jsql/spec.md`).

## 4. Reglas

1. Los datos dinámicos son `et.Json`, no `map[string]interface{}`.
2. Los handlers responden con `response.*`, nunca `json.Marshal` + `w.Write`.
3. Las consultas usan el builder de `jsql` o su JSON, no SQL concatenado (salvo dentro de un driver).
4. Un `Store` inyectado implementa la interfaz exacta del paquete que lo recibe.
5. `cache.Load()` y `event.Load()` se llaman una vez al arrancar.
6. Los mensajes de error van en un `msg.go` local.
7. En desarrollo, `.Debug()` registra el SQL y `.Test()` lo genera sin ejecutarlo.
8. Si una firma de este documento no compila, confía en el código (`go doc`) y corrige el documento.
