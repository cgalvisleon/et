# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Module

`github.com/cgalvisleon/et` — Go 1.25, MIT license. Part of the parent workspace `go.work`.

> The codebase changes fast (commits are bare "Backup:" WIP checkpoints) — **read the source before trusting any API shape here**. Docs: only `README.md` lives at the root; everything else is in `doc/` — `doc/README.es.md` (Spanish translation of `README.md`, keep both in sync), `doc/LIBRARY_CONTEXT.md` (copied into consumer projects by `go run ./cmd/install`), `doc/ARCHITECTURE_SUMMARY.md`, `doc/COMPONENT_CATALOG.md`, `doc/AI_USAGE_GUIDE.md` (the last four are what the `lib-ctx-gen` skill generates); they can drift from the code. `doc/NEW_PROJECT_GUIDE.md` has verified templates for starting a new project on `et` (HTTP microservice like `core-studio/api`, CLI like `tick`); `go run ./cmd/create` scaffolds `Project`/`Microservice`/`Modelo`/`Rpc` (`create modelo` prints a manual wiring step since `file.MakeFile` never overwrites).

## Commands

```bash
go build ./...                       # build + vet are currently clean
go vet ./...
gofmt -w .
go test ./queue/ -run Example_queue -race   # prefer one package over go test ./...

go run ./cmd/server   # TCP node (port 1377, -port flag)
go run ./cmd/jrex     # JS runtime, hot-reload from ./cmd/jrex/src/
go run ./cmd/jsql     # jsql driver demo
go run ./cmd/ia       # RAG HTTP API (PORT, default 3300)
docker compose up -d  # optional local Ollama on 11434 (see docker-compose.yml header)

./version.sh --major | --minor | --request   # bumps git tag, updates README.md + doc/README.es.md, pushes
```

Tests exist only in `queue/` (`Example_queue`), `ia/loader_test.go`, `jsql/drivers/test/`, and `jrpc/`, `jws/`, `cache/`, `aws/`, `brevo/` — the latter need live Redis/AWS/Brevo and fail or hang without them. `jwf` and `ia`'s ingest/ask path have no tests.

`doc/COMPONENT_CATALOG.md` is generated from `go doc` (every exported func/method/interface per package); regenerate it rather than hand-editing when the API changes. `LIBRARY_CONTEXT.md`, `ARCHITECTURE_SUMMARY.md` and `AI_USAGE_GUIDE.md` were re-verified against the code on 2026-10-10.

`jsql/drivers/test/` is the cross-driver suite (`TestCatalog`, `TestInsertUpdate`, `TestJoinGroupHaving`, `TestWrapper`, `TestCreateDropDB`): it runs every case against sqlite (temp file), postgres, mysql, mssql and oracle, reading connection settings from `et/.env` (`DB_HOST`/`DB_PORT`/`DB_NAME`/`DB_USER`/`DB_PASSWORD`, `ORACLE_HOST`/`ORACLE_SERVICE`/`ORACLE_USER`…) and `t.Skip`-ing any engine it can't reach, so a green run may have only exercised sqlite — check the skip lines with `-v`. `JSQL_DEBUG=1` logs the generated SQL in the tests (`DB_IS_DEBUG=true` does it for a regular `DB`). `TestCatalog` writes `result/*` only with `-update` (`go test -run TestCatalog -update`); an engine not reachable in that run keeps its previous `result/<driver>.json` in the summary. `result/<driver>.sql|.json` hold per-driver outputs.

```bash
cd jsql/drivers/test && go test -v -run 'TestInsertUpdate/sqlite' .
```

## Code style

GoDoc for every function, in English: starts with the function name + brief description, one line for params, one for returns. **But** most existing files (`et/json.go`, `jwf/`, `resilience/`, `stores/`) use a `/** ... @param ... @return ... **/` block style — match whatever the surrounding file uses.

## Architecture

Modular utility library for Go microservices: every directory is an independent package, no central entry point.

### Core type: `et.Json`

`et/json.go`: `Json` (`map[string]interface{}`) is the lingua franca of the whole library — typed accessors (`Str`, `Int`, `Bool`, `Time`, `Json`, `Array`…), default-value variants (`ValStr(def, keys...)`), nested traversal via variadic keys. `ScanRows` decodes `[]byte` and JSON-looking `string` values (`{`/`[` prefix). `et/list.go` `List` is the paginated result (`Rows`, `All`, `Count`, `Page`, `Start`, `End`, `Result []Json`); `et/item.go`/`items.go` are single/multi result wrappers. Data types (`et.TypeData`, `et/condition.go`): `KEY` (VARCHAR 80), `TEXT`, `MEMO`, `INT`, `FLOAT`, `BOOL`, `DATETIME`, `JSON`, `BYTE`, `ANY`, `ARRAY`, `ARRAY_JSON|STRING|INT|FLOAT|BOOL|DATETIME`. There is no `GEOMETRY`/`EMBEDDING`/`STRING`.

### SQL builder / ORM: `jsql/`

- `jsql.Load()` / `jsql.LoadTo(dbName, hostName...)` → `*DB`; config is read internally via `envar` (`DB_TENANT_ID` is no longer read; only `create/template/templates.go` still writes it).
- Drivers in `jsql/drivers/<name>/`, self-register via `init()`, imported as side effect: `postgres` (`lib/pq`), `sqlite` (`modernc.org/sqlite`), `mysql`, `mssql`, `oracle` — each with the same file set (`connect`, `load` = DDL, `query`, `command`, `quoted`, `types`, `test`). `josefina` is a constant only. A change to SQL generation usually has to be mirrored in all five.
- **Table names**: the core sets `Model.Table` (in `newModel` and `Model.Init`, `Model.setTable`) from `Driver.UseSchema()`: `schema.name` for postgres/mssql, `schema_name` for sqlite/mysql/oracle (no `CREATE SCHEMA`, tables live in the connection's database/user). Drivers build SQL from `model.Table`/`From.Table`, not `Schema`+`Name` (mssql splits it back to quote `[schema].[name]`). Raw SQL against a sqlite table must use the prefixed name (e.g. `main_tasks`).
- `jsql/catalog.go` is the whole public API in one file: every exported function/method is a thin wrapper over a private implementation elsewhere in the package, grouped into numbered sections. New public API goes there (and `TestCatalog` checks the sections). `jsql/spec.md` is the formal spec with a numbered gap list ("brechas", e.g. B8); `feature.md` is its source.
- `jsql/jquery.go` parses JSON query/command descriptors (`select`/`from`/`join`/`where`/…, `insert`/`update`/`delete`/`upsert`/`bulk`); `from` and join `to` take the standard object `{database, schema, model, as}` or the short text `"schema.model:as"` (one parser, `fromRefs`/`parseFromRef` + `DB.modelOf` in `query.go`; `database` must match the connection's); `jsql/wrapper.go` `Wrapper(db)` exposes jsql to goja scripts (triggers, `DefineCalc`) and `jsql/jsql.d.ts` is the matching TypeScript declaration — keep both in sync when changing the wrapper.
- `Driver` interface (`jsql/driver.go`): `UseSchema()`, `CreateDB`, `DropDB`, `Connect(db)`, `ExistModel`, `Load(model)` (DDL text), `Drop(model)`, `Query(query)` (SELECT text), `Command(command)` (DML text); all but `UseSchema` take `timeout ...time.Duration`. Optional `jsql.Tester` (see below).
- sqlite quirk: queries return each row as one JSON object; a nested JSON column inside it comes back double-encoded (see `ia/similarity.go` `chunkEmbedding`).

```go
model, _ := db.DefineModel("public", "users", 1, userId) // adds id, created_at, updated_at, _source, _idx

model, _ := db.Define(jsql.Define{                            // struct-based, preferred for complex models
    Schema: "public", Name: "users", Version: 1, IdxField: jsql.IDX,
    Columns: []jsql.Column{
        {Name: "email", TypeData: et.TEXT, Default: ""},
        {Name: "name", TypeColumn: jsql.ATTRIB, TypeData: et.TEXT, Default: ""},
    },
    PrimaryKeys: []jsql.DefIndex{{Name: "email", Sorted: true}},
    Unique:      []jsql.DefIndex{{Name: "email"}},
})
model.Init() // runs DDL

items, _ := model.Where(jsql.Eq("status", jsql.ACTIVE)).And(jsql.More("age", 18)).Limit(1, 20) // page, rows
item, _  := model.Where(jsql.Eq("id", id)).One()
_, _ = model.Insert(et.Json{...}).ExecTx(nil)   // also Update(...).Where(...), Upsert(...)
```

Manual alternative: `db.NewModel(...)` + `DefineColumn`/`DefinePrimaryKey`/`DefineUnique`/`DefineAttrib`/`DefineForeignKeys`. `db.LoadModel(schema, json)` and the JSON `define` descriptor build the same `Define`.

- **`TypeColumn`**: `COLUMN` (real column), `ATTRIB` (key inside `_source` JSONB, cast on read), `DETAIL`/`MASTER`/`ROLLUP` (virtual relations; `MASTER` goes through a bridge table), `CALCFUNC` (Go `CalcFunction`), `CALC` (JS script run per row).
- **Column constants** (`jsql/column.go`): `ID`, `IDX` (`_idx`), `SOURCE` (`_source`), `RESULT`, `STATUS`, `VERSION`, `TENANT_ID`, `PROJECT_ID`, `CREATED_AT`, `UPDATED_AT`, `CODE`, `KIND`. `_idx` is a `reg.UUID()` set by an auto `BeforeInsert` trigger, not a sequence.
- **Status constants**: `ACTIVE`, `ARCHIVED`, `CANCELED`, `PENDING`, `APPROVED`, `REJECTED`, `OF_SYSTEM`, `FOR_DELETE`, `IN_PROCESS`, `FAILED`; `jsql.Status` is the map of valid ones, `jsql.StatusList()` lists them.
- **Triggers**: before/after × insert/update/delete, `TriggerFunction func(tx *Tx, old, new et.Json) error`.
- **Nested JSONB paths**: `"a->b->c"` field names are translated to `->`/`->>` chains with casts on ATTRIB leaves.
- `.Debug()` logs SQL and skips execution; `.Test()` generates SQL without executing (on `Model`, `Query`, `Command`).
- `jsql.DefineSeries(db, schema)` → series/code generator (`GenSerie`, `GenValue`, `SetSeries`, `Model()`). There is no `jsql.Store`/`DefineStore` anymore.

### HTTP servers

- **`server/`** — lightweight `chi.Mux` wrapper, no Redis/NATS.
- **`ettp/v2/`** — `ettp.New(name, *Config)`; `Config` has `Port`, `RpcPort` (zero = OS-assigned), `Parent`, timeouts, `AllowOrigin`, TLS, `Transport`, `Debug`. It does **not** call `cache.Load()`/`event.Load()` — callers must. Syncs router state across replicas over NATS (`EVENT_SET_ROUTER`/`REMOVE`/`RESET`; `m.Myself` prevents self-processing).
- **`ettp/v1/`** — older but not dead (has apigateway/proxy/token/cache files with no v2 equivalent); check which version a `cmd/` imports.
- **`router/`** — `router.NewApi(name, path, host, port, rpc, version)` → `*Api` over `chi.Mux`; `Public`/`Session`/`Protected(group, method, path, name, handler)`; `Api.Item(status, fn)`/`Items(fn)`/`List(fn)` wrap business functions; `UseAuthentication`, `UseAuthorization`, `UseLimitRate`. Used by `core-studio/api`, `postick/api` and `ettp/v2`.

### Infrastructure (external services)

- **`cache/`** Redis (`Set`/`Get`/`Delete`/`Pub`/`Sub`), **`event/`** NATS (`Subscribe`/`Publish`/`Stack`), **`graph/`** Neo4j.
- **`jrpc/`** — `net/rpc` over TCP; `jrpc.Mount(host, port, services, packageName)` with a simple `Solver` registry.
- **`jtcp/`** — distributed TCP node with Raft-style election (`Follower`/`Candidate`/`Leader`/`Proxy`), balancer; `jtcp.NewNode(port)` used by `cmd/server`.

### Utility packages

- **`envar/`** — `GetStr/GetInt/GetInt64/GetFloat/GetBool` with defaults, `Arg*` for CLI args, `Validate([]string)`. The old `config/` package is gone (only stale strings remain in `create/template/templates.go`).
- **`logs/`** (`Info`, `Alert`, `Error`, `Debug`, `Fatal`→`os.Exit(1)`, `…f` variants) over **`stdrout/`**.
- **`jwt/`** (tokens stored in `cache`) over **`claim/`** (HS256, `SECRET`).
- **`crontab/`** — event-driven; `crontab.Load(tag, store)` sets a package singleton, then `crontab.CronJob(tag, ownerId, Cron{...}, repetitions, params, fn)` / `ScheduleJob(tag, ownerId, time.Time, params, fn)`. HTTP handlers publish control events.
- **`validator/`** — fluent validator (`jval/` no longer exists). Conditions: `Required`/`Min`/`Max`/`Between`/`MinLength`/`MaxLength`/`Pattern`/`NotEmpty` and "at least one" checks `IsLetters`/`IsNumbers`/`IsSpecialCharacters`; handles `[]any` from decoded JSON, objects (`et.Json`, `map[string]any`, `[]et.Json`; `Required` means non-empty, like arrays) and `nil`. `Validate` fails a `Required` field that is missing from the json (2026-09-28; before, it only checked the keys present). `ByJson(params) (*Validator, error)` defines it from `{not_empty, fields: {name: {required, not_empty, is_letters, is_numbers, is_special_characters, is_email, is_phone, min, max, min_length, max_length, pattern}}}` (`validator/json.go`; an unknown condition or a pattern that does not compile is an error naming the field) and `Validator.ToJson()`/`Condition.ToJson()` give it back. `validator.Wrapper()` (2026-10-11, `validator/wrapper.go`) is the `validator` binding of jwf scripts: `validator(def?)` → camelCase, chainable (`field`, `required`, `isEmail`, `min`, `pattern`…, `byJson`, `validate` —true or throws—, `isValid`, `errorOf`, `toJson`); every error is a Go error that goja throws as a JS `Error` (`try/catch`, `e.message`), and `pattern` checks the regex (an invalid one used to panic in `validate`). `validate` accepts `int32`/`int64`/`float32` (goja exports JS integers as `int64`; before, they were "of invalid type"). Caveat: `min`/`max` only apply when > 0 and `pattern` is compiled on every validation.
- **`queue/`** — `queue.New[T](queueSize, maxEvents, period, handler)`; flushes on `maxEvents` or `period`, `.Close()` flushes.
- **`request/`** (`URLParam`, `GetBody`, outbound `Fetch` — JSON/form only), **`response/`** (`ITEM`, `ITEMS`, `HTTPError`), **`middleware/`**, **`msg/`** (shared message constants).
- **`xls/`**, **`csv/`** — same API shape: `Read*(data|multipart|file)` → reader; `New*(data, columns)` → `ToFile`/`ToWriter`/`ToHttp` (columns derived from keys if omitted).
- **`utility/`**, **`reg/`** (IDs: UUID/ULID/Snowflake), **`file/`** (watcher), **`dt/`** (Redis when `PRODUCTION` is true — the default; otherwise memory only, nothing persisted), **`jws/`** (websocket), **`service/`** (OTP), **`mem/`**, **`ephemeral/`**, **`iterate/`**, **`race/`**, **`cmds/`**, **`timezone/`**, **`units/`**, **`color/`**.
- Messaging: **`aws/`** (S3/SES/SMS), **`brevo/`**, **`jwsp/`** (WhatsApp Graph API), **`infobip/`** (`NewSenderInfobip(Params{BaseUrl, ApiKey, Sender})`; `SendSMS`/`SendEmail` (API v4 JSON)/`SendWhatsApp` (templates only) take `tpMessage` `"Transactional"|"Promotional"`; `{{key}}` substitution like brevo).

### Application packages

- **`jrex/`** — JS runtime (`goja`). `jrex.Load(tag, store)` (nil store → `FileStore` at `./src`); `Set(name, value)` injects Go bindings; `Run()`/`RunModule(tag)`; `RunDev()` hot-reloads via `file.NewWatcher`. Globals: `console`, `ctx`, `fetch`, `require`. `jcli/` (Bubble Tea, declares `package jrex`) is orphaned WIP.
- **`jia/`** — OpenAI agents (`openai-go/v3`): `jia.New(tag, store, userId)`, `jia.Load(id, store)`; Agents, Participants, Conversations/Messages, Skills (`ApiSkill`); not tenant-scoped. **Bug:** `jia/ia.go:90` calls `store.Get(id, packageName, …)` with args swapped.
- **`ia/`** — multitenant RAG (unrelated to `jia`). `ia.Load(db, schema, Config)` defines `ia_documents`/`ia_chunks`/`ia_conversations`/`ia_messages`, all with `tenant_id` + `project_id`. `IngestFile` (pdf/docx/xlsx/csv/txt/md loaders in `ia/loader_*.go`) and `IngestSQL` → chunk → embed → store. No pgvector: embeddings are JSON `[]float64`, and `Ask` ranks by cosine similarity in Go over up to `DB_RECORD_LIMIT` (default 1000) chunks, then answers with `Chat.Completions`. `embedFn`/`answerFn` are swappable for tests. `LoadRouter(chi.Router)`; design in `featureIa.md`.
- **`stores/`** — jsql-backed `DefineInstance`/`LoadInstance` (Json/Bite kinds), `DefineAuthorization` (ACL cached through `dt`), `DefineConfig` (per-tenant settings). `stores.Instance.Get(id, dest)` has one key, so it doesn't satisfy the `Store` interfaces below.
- **`resilience/`** — retry: `resilience.New()` (calls `event.Load()`, no store), `LoadInstance(Params{Id, Tag, Description, TotalAttempts, Interval, Tags, Fn, FnArgs})`, `instance.Run() ([]any, error)`, `Stop(id)`. Used by `jwf`.
- **`jwf/`** — workflow engine, detailed below.

- **Crear / eliminar la base** (`Driver.CreateDB` / `Driver.DropDB`, expuestos como `jsql.CreateDB(connection, timeout...)` / `jsql.DropDB(db, timeout...)`; los métodos de `Driver` reciben `timeout ...time.Duration` y usan `jsql.TimeoutContext`: sin timeout o con `0` nunca fallan por timeout): idempotentes, implementados en los cinco drivers (`drivers/<x>/connect.go`); `Connect` de postgres, mysql y mssql usa `CreateDB`; oracle no crea ni elimina el servicio (`DropDB` devuelve `MSG_DRIVER_CANNOT_DROP_DB`). `TestCreateDropDB` (`jsql/drivers/test/createdb_test.go`) corre en los cinco drivers (en oracle solo comprueba el servicio y el error de `DropDB`).
- **Probar una conexión sin tocar nada** (`jsql/tester.go`): `jsql.TestConnection(ctx, connection, query)` busca el driver por `connection.GetParams()["driver"]` y, si implementa la interfaz opcional `jsql.Tester` (`Test(ctx, connection, query)`), abre, hace ping, corre la consulta (si hay) y cierra con `jsql.Probe`; si no, `MSG_DRIVER_CANNOT_TEST`. Los cinco drivers la implementan (`drivers/<x>/test.go`) reutilizando su `connectTo`/`dsn`/`chain`: **no crean la base** (a diferencia de `Connect`, que en postgres, mysql y mssql la crea si falta); postgres usa `postgres` y mssql `master` si no viene base; sqlite exige que el archivo exista (abrirlo lo crearía). El contexto acota cuánto tarda.

### `jwf/` (workflows)

`jwf.New(id)` calls `cache.Load()` + `event.Load()` and uses `reg.GetUUID(id)` (blank → new UUID). There is **no** store, `jwf.Load`, `DefineStore` or `LoadRouter` anymore: flows live in memory (`WorkFlow.Flows`, keyed by tag) and are persisted by the caller through `WorkFlow.OnChange(fn)` / `Flow.OnChange(fn)`; `jwf.LoadFlow(json)` / `wf.SetFlow` rebuild them.

```
WorkFlow (ID, TimeAwait, AuditLog)
  Flows map[tag]*Flow
    Steps        map[id]*Step (incl. triggers)
    Connections  Source/Target {StepId, Port, Index} + Kind input/output/error
    Triggers     {Type, Tag, StartId}
    TotalAttempts / TimeAttempts / TimeAwait, Constants, SaveInstances, Published
```

```go
wf, _ := jwf.New("")
wf.NewFlow("add", "add item", "1.0.0", ownerId, userId).
    Step("add", "step 1", func(instance *jwf.Instance, ctx et.Json) (et.Json, error) { return ctx, nil }).
    Step("add", "step 2", fn)
// Run(tag, triggerTag, id, code string, ctx, tags et.Json, await bool, userId string)
result, err := wf.Run("add", "add", reg.GetULID(""), "", et.Json{}, et.Json{}, true, userId)
```

- **`params` and `ctx` in a script are goja dynamic objects** (2026-10-11, `jsonObject` + `jsonMethods` + `jsonBinder` in `jwf/wrapper.go`, over `Step.Params` and `Instance.Ctx`): `ctx.user.name` reads and `ctx.total = …` writes, `JSON.stringify(ctx)`/`Object.keys(ctx)` list the keys, and the methods (`str`, `int`, `get`, `set`, `json`…) still work — a method hides a key with the same name (read it with `ctx.get("str")`). Values go out through `plainOf` (an `et.Json`, nested too, becomes `map[string]any`: goja would expose the named type with its methods and no keys), so a nested change (`ctx.user.name = …`) does not persist: reassign the top key or use `set`. Passed to Go (`console.debug(params)`, an `et.Json` argument) it shows its values (`String`/`MarshalJSON`). It is built per run through `jwf.Binder` (`func(vm *goja.Runtime) goja.Value`): a binding of that type is called by `RunCode` with the runtime instead of being set as is.
- **Console output** (2026-10-11, `jwf/console.go`): `consoleOf(prefix, emit)` builds the `console` binding (`log`, `debug`, `info`, `warn`, `error`; each argument a string as is or json) that writes to the server log and, with `emit`, sends a line `{level, message, at}` (message cut at `MAX_LOG_MESSAGE`, 4096). `Instance.OnLog(fn)` gets the lines of every step of the instance (with `step_id`, `instance_id`); `Step.OnLog(fn)` the lines of a step run alone (`RunScript(nil, …)`, a Test: it sets its own `console` then). `RunScript` returns the instance `Ctx` when it has one and, alone, the completion value of the scripts (the last expression, `RunScripts` → `RunCode`) when it is an object (`objectOf`: a JS object exports as `map[string]any`); anything else gives `{}`.
- **Go values handed to goja as they are** (checked 2026-10-11): an `et.Json` returned by a Go function (`db.Query(...).Result[0]`, `Json`, `.Json("x")`) is a Go object with its methods (`.Str("name")`, `.ToString()`) but its keys are **not** readable (`row.name` is `undefined`, `JSON.stringify(row)` is `{}`); `timeNow()` is a `time.Time` (`.Format`, `.Unix`), not a `Date`. (`validator()` has its own wrapper since 2026-10-11.) Only `params`/`ctx` (`jsonObject` + `plainOf`) give plain objects.
- First `Step` becomes a trigger; the next ones chain through output connections. `Flow.Error(...)` attaches an error-port step to the last step. A step runs a Go `fnStep` or a JS script (`RunCode`/`RunScripts`, goja, with `cache`/`fetch` bindings from `wrapper.go`).
- Instances are stored in Redis under `instance:<id>` (`cache.SetObject`, no TTL); `GetInstance` loads one or creates it, `RunInstance` runs it, `StopInstance(id)` sets `instance:<id>:stop` for `TimeAwait`. `Run` returns only the result ctx: pass your own id to read the instance afterwards. Status: `CREATED`, `PENDING`, `RUNNING`, `ROLLBACK`, `DONE`, `FAILED`, `CANCEL`, `COMPLETED`.
- On a step error it retries through `resilience` when `Flow.TotalAttempts != 0`, then follows the error connection.
- `AuditLog []et.Json` + `onChange` hooks (`MAX_AUDIT_LOG`, default 1000) is a reusable pattern; `jwf/step.go` (`Step`) and `jwf/flow.go` (`Flow`) are the reference implementation.

### `cmd/` binaries

`et` (cobra CLI), `apigateway` (ettp), `daemon` (systemd), `create` (scaffolder, templates in `create/`), `server`/`client` (jtcp), `jrex`, `jsql`, `install`, `whatcher`, `resilience`, `wsp`, `ia` (on `server/`), `jwf` (current `jwf.New("")` example).

## Key patterns

- **`Load()` initialization**: infrastructure packages read env via `envar`, connect once; later calls are no-ops.
- **Local `Store` interfaces**: each package (`jwf`, `jia`, `jrex`, `resilience`, `crontab`) defines its own and the caller implements it. They differ: `jwf.Store.Query` takes a leading `collection`, `jia.Store.Query` doesn't (`jsql.Store` satisfies `jia.Store` exactly, not `jwf.Store`); `jrex.Store` has only `Set`/`Get`. Check signatures before reusing an implementation.
- **Messages**: use the package-local `msg.go`/`msg/` constants, not hardcoded strings.
- **Errors**: `logs.Fatal` exits; use `logs.Alert`/`logs.Error` for non-fatal.
- **HTTP handlers** (`handler.go` everywhere):

```go
func (s *T) HttpFoo(w http.ResponseWriter, r *http.Request) {
    id := request.URLParam(r, "id").Str()
    body, err := request.GetBody(r)
    if err != nil {
        response.HTTPError(w, r, http.StatusBadRequest, err.Error())
        return
    }
    tag, ctx := body.Str("tag"), body.Json("ctx")
    response.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: data}) // StatusCreated for creating POSTs
}
```

## Environment variables

| Package | Variables |
| --- | --- |
| `jsql` | `DB_DRIVER`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_RECORD_LIMIT`, `DB_IS_DEBUG`, `DB_APP_NAME`, `DB_SSLMODE` (postgres), `DB_SSL`/`DB_SSL_VERIFY` (oracle); `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_LIFETIME`, `DB_POOL_IDLE_TIME` (every connection; server ones embed `jsql.Pool`) |
| `cache` | `REDIS_HOST`, `REDIS_PASSWORD`, `REDIS_DB` |
| `event` | `NATS_HOST`, `NATS_USER`, `NATS_PASSWORD` |
| `graph` | `NEO4J_HOST`, `NEO4J_USER`, `NEO4J_PASSWORD` |
| `jia`, `ia` | `OPENAI_API_KEY` |
| `ia` | `IA_SCHEMA` (`public`, cmd/ia), `IA_EMBEDDING_MODEL`, `IA_CHAT_MODEL`, `IA_CHUNK_SIZE`, `IA_CHUNK_OVERLAP`, `IA_TOP_K` |
| `dt` | `PRODUCTION` (`true` = Redis) |
| `jwsp` | `WHATSAPP_API_URL` |
| `claim` | `SECRET` (default `"1977"`) |
| `validator` | `LANG` |
| `jwf` | `MAX_AUDIT_LOG` (1000) |

## Cambios del 2026-09-28 (desde core-studio)

- `et.ToCondition` acepta `{"where": {campo: {op: valor}}}` como primera condición (sin conector), igual que `and`/`or`; antes `where` caía en el caso por defecto y armaba una condición sobre un campo llamado `where`.
- `jsql.Series.Model()` expone el modelo de la tabla de series, para publicarlo como cualquier otro.
- `jsql` `newSeries` crea la serie con `value` 0: el primer `GenSerie` devuelve 1 (antes 2).
- `jwf.WorkFlow.SetFlow(def, userId)` carga un flujo entero desde su JSON (steps como mapa nodo → paso, connections, triggers) conservando los ids de los nodos; sin triggers, los arma de los pasos de kind trigger.
- `jwf` `getInstance` y `cache.GetObject` tratan `cache.ErrNotFound` como clave inexistente (antes solo `redis.Nil`, que `cache.Get` ya no devuelve: toda instancia nueva fallaba con «not found»).

- `ettp/v2` `migrate` (al arrancar un gateway sin rutas guardadas en Redis) trata `cache.ErrNotFound` como «no hay rutas v0.0.1 que migrar»; antes solo `redis.Nil`, así que el gateway se caía con `FATAL: not found`.
- `ettp/v2`: `Save` serializa `Solvers` bajo `muRoutes` (varias suscripciones de NATS registran rutas a la vez y el gateway se caía con `concurrent map iteration and map write`); `findResolver` busca con `URL.EscapedPath()` y el resolver reenvía los parámetros sin decodificar (y los pasa decodificados a `SetPathValue`): un id con `%2F` ya no se parte en dos segmentos.
- `router.Api`: `Public(method, path, name, handler)` y `Session(method, path, name, handler)` llevan `name`, como `Protected`; las tres anteponen el `Path` del Api (`getPath`) y publican la ruta al catálogo (`RegisterEndpoint`, grupo vacío en las públicas y de sesión). Llamadores actualizados: los routers de `core-studio/api/pkg/*` y `postick/api/pkg/postick`.

## Cambios del 2026-10-02 (cache)

- `cache.GetCtx`/`cache.Get` devuelven `(valor, existe, error)`; `cache.ErrNotFound` ya no existe: una clave inexistente es `def, false, nil`. Igual `GetJson`/`GetItem`/`GetItems`/`GetVerify`/`CollectionFind`/`ObjetGet`, y los nuevos `GetStr`/`GetInt`/`GetInt64`/`GetFloat`/`GetBool`/`GetTime`/`GetDuration`/`GetBytes`/`GetList` (valor que no parsea → `def, true, err`). `Set` guarda `float64` con `strconv.FormatFloat(v, 'f', -1, 64)` (antes `%f`, 6 decimales). `HttpGet` de `cache` y de `ettp` v1/v2 responden `et.Item{Ok: existe, Result: {key, value}}`.

## Cambios del 2026-10-10 (documentación)

- `README.md` reescrito: instalación primero, tabla por paquete con API pública y ejemplo, ejemplos verificados con `go vet`.
- `LIBRARY_CONTEXT.md`, `ARCHITECTURE_SUMMARY.md`, `AI_USAGE_GUIDE.md` reescritos desde el código; `COMPONENT_CATALOG.md` generado desde `go doc`.
- `router.Api.Public/Session/Protected` llevan hoy `group` primero: `(group, method, path, name, handler)`.
- `jsql/spec.md` §12 recalculado: B6 cerrada; nuevas B9 (`LoadDb` en pánico), B10 (`audit_log` del modelo no es lista) B11 (`TestWrapper` espera `conn.id` indefinido) y B12 (`Define` exige esquema; en sqlite las tablas pasaron a `schema_nombre`).
- `router.Publish`/`router.With` ya no anteponen `Route.PackageName` al `Path` (rompió postick: rutas montadas en la raíz).

## Cambios del 2026-10-11 (código)

- `validator`: `ByJson`/`ToJson`; los mensajes `MSG_VALIDATOR_MIN|MAX|BETWEEN|MIN_LENGTH|MAX_LENGTH` usan `%v` (con `%s` salían `%!s(float64=18)`).
- `validator`: `Wrapper()` para goja (el binding `validator` de jwf) y `int32`/`int64`/`float32` en `validate`.
- `jwf`: `Binder` y `params` y `ctx` como objetos dinámicos (`jsonObject`); `console` con `OnLog` en `Instance` y `Step` y el nivel `warn`, y `console.error` acepta varios argumentos; `Step.RunScript` sin instancia devuelve el último valor de los scripts si es un objeto (`objectOf`) (ver «`jwf/`»).
## Cambios del 2026-10-12 (código)

- `jwf`: `GoValue(value any) Binder` (`jwf/goobject.go`) expone un valor de Go (un `*jsql.DB`, un struct con métodos, un mapa de ellos) con sus nombres de Go, pero convierte lo que devuelve: `et.Json`, `et.Item`, `et.Items` y mapas o listas de datos llegan como objetos planos (`item.result.name`), un struct con métodos vuelve a ser un objeto de Go (`goObject`, un `DynamicObject`: `db.GetModel(…).Query(…).All()`), un `error` devuelto se lanza y los argumentos van a su tipo de Go (un `goObject` devuelve su valor; `params`/`ctx`, su `et.Json`). `Keys` solo lista los campos de datos (los que apuntan a otros objetos harían infinito un `JSON.stringify`). Lo usan `db` y `dbs.get` de los pasos de core-studio. Probado con sqlite desechable (insert, query con `where`, campos, error atrapado). `jsql.Wrapper` **se mantiene**: lo usan los scripts de eventos de un modelo (`Model.wrapper`, `model.go`) y `TestWrapper`.
- `jsql` wrapper (goja): en un modelo, `beforeInsert`/`beforeUpdate`/`beforeDelete`/`afterInsert`/`afterUpdate`/`afterDelete` reciben ahora `(name, code, version)` y llaman a `Define…` (un script que el modelo guarda y corre con `OLD`/`NEW`; un before puede cambiar `NEW` y un `throw` cancela); se quitaron los `define…` del wrapper y las versiones con función JS del modelo (el `*Model` es compartido y la función quedaba atada a una VM ya terminada; además no leía ni cambiaba el registro). `beforeInsertOrUpdate`/`afterInsertOrUpdate` definen el mismo script en los dos. Los triggers de un **comando** siguen recibiendo una función. Probado con sqlite desechable.
- `jsql` triggers por script y series: los scripts de un trigger (`BeforeInserts`…, corridos por `Command` con jrex) reciben `TX`, la transacción del comando (`Model.wrapper(instance, tx)`); lo que el trigger escribe debe ir en ella (`.oneTx(TX)`, `series.genSerie(tag, TX)`): con SQLite, escribir fuera esperaba para siempre la única conexión (se colgó probando). `Series` suma `GenSerieTx`/`GenValueTx` (`genSerie`/`genValue` reciben `tx`), y el wrapper `genSerie(tag, tx?)`/`genValue(tag, tx?)` y `db.series(schema)` (una `DefineSeries` por db y esquema, en `seriesCache`; `jsql.defineSeries` la define en cada llamada). `Wrapper` suma `bind`. Probado con sqlite desechable: insert/update/delete con before que asignan `code` de una serie, `status`, `version` y una auditoría en el delete, y un ejemplo con `bind`. B11 cerrada el mismo día: `wrapConnection` pone `id` siempre (es parte de `Connection`), y `TestWrapper` (`connId` = `function`) y `jsql.d.ts` (`id(): string`) lo dicen; `TestWrapper` pasa en sqlite y postgres.
- `jsql/spec.md` §12: **cerradas todas salvo B5** con la batería en los cinco motores (contenedores desechables `jsql-mysql`, `jsql-mssql`, `jsql-oracle` con las credenciales por defecto de la batería; SQL Server necesita `CREATE DATABASE jsql`): `TestCatalog` 102/103 (1 omitido) en sqlite, postgres, oracle, mysql y mssql. B9 `LoadDb` reconstruye conexión y `DB` (`ConnectionFromJson` con el tipo de `params.driver`, `newDB`, esquemas y `audit_log`; sin `Init`); B10 `et.SafeData` se serializa como lista (`All`, `MarshalJSON`/`UnmarshalJSON` que acepta `{data}`); B13 `jsql.Pool` embebido en las conexiones de servidor (`DB_POOL_*`, solo los > 0 van en `GetParams`); B12 esquema obligatorio y renombrado automático en sqlite de `nombre` a `schema_nombre` (`Sqlite.ExistModel`); B3 `Define` lee `masters` y `master`; B7 `-update`; B4/B8 SQL Server y `TestCreateDropDB` en los cinco (sqlite `DropDB` respeta un timeout vencido); B5 sigue abierta: el usuario pidió no implementar todavía el driver de josefina (se conserva la constante `DriverJosefina`).
- `jsql`: `Bind(source, vars et.Json) et.Json` (`helpers.go`) aplica las variables `{{nombre}}` a un json: `"{{x}}"` solo toma el valor tal cual (número, lista, objeto); dentro de un texto, su texto; en una lista, una lista se expande; `{{user.id}}` es una ruta en vars; las desconocidas quedan igual. No es para SQL escrito a mano (los valores van como parámetros). `BindWrapper()` (`wrapper.go`) es su versión para goja (devuelve un objeto plano, `toJsMap`): `jsql.bind` en `wrapPackage` y el global `bind` de los pasos de core-studio (engine). Declarado en `jsql.d.ts`. Probado con un test temporal (Go y goja).

- **`router` rate limit, pendiente**: `rateLimitMiddleware` cuenta por `MÉTODO:ruta` con `cache.Incr(key, 1s)`, y `IncrCtx` solo pone el vencimiento cuando el contador nace (`result == 1`). Si ese `Expire` falla una vez, la clave queda sin TTL, sigue sumando y al pasar `LimitRate` (1000) la ruta responde 429 para siempre (pasó con `POST:/api/tenancy/steps/query` en core-studio; se arregló borrando la clave en Redis). Falta hacerlo atómico (`INCR` + `EXPIRE … NX` en un pipeline) o reponer el TTL cuando es -1.

## Cambios del 2026-10-11 (documentación)

- `doc/COMPONENT_CATALOG.md`: secciones `jwf` y `validator` regeneradas desde `go doc` (ByJson, ToJson, OnLog, Binder…). `LIBRARY_CONTEXT.md`, `AI_USAGE_GUIDE.md`, `ARCHITECTURE_SUMMARY.md` (flujo de `jwf` y dos problemas conocidos: el TTL del límite de peticiones y `validator`) y la tabla de paquetes de `README.md` / `doc/README.es.md` al día con los cambios de código del 2026-10-11.

- Todos los `.md` salvo `README.md` (y `CLAUDE.md`) pasaron a `doc/`. `doc/README.es.md` es ahora una traducción completa de `README.md`; `version.sh` actualiza la versión en ambos.
- Se eliminó `AUDITLOG.md` (describía `onSave`/`onDelete`/`store` en `jwf`, que ya no existen).
- `cmd/install` copia `doc/LIBRARY_CONTEXT.md`; el skill `lib-ctx-gen` genera sus cuatro archivos en `doc/`.
- README (en/es): ejemplos de `validator.ByJson` y `jwf` `Instance.OnLog` (verificados con `go vet`); tabla de variables de `jsql` corregida desde el código.
- `jsql/spec.md` §10 (variables corregidas: pool `DB_POOL_LIFETIME`/`DB_POOL_IDLE_TIME`, `DB_SSLMODE`, `DB_APP_NAME`, sin `DB_TENANT_ID`) y §12 recalculado con la batería del 2026-10-11 (sqlite y postgres: 100/103; sin cambios en B3–B12); nueva B13 (el pool solo se configura en sqlite).
