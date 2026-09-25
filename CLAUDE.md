# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Module

`github.com/cgalvisleon/et` — Go 1.25, MIT license. Part of the parent workspace `go.work`.

> The codebase changes fast (commits are bare "Backup:" WIP checkpoints) — **read the source before trusting any API shape here**. `README.md`/`README.es.md`, `LIBRARY_CONTEXT.md`, `ARCHITECTURE_SUMMARY.md`, `COMPONENT_CATALOG.md`, `AI_USAGE_GUIDE.md` drift out of sync (README still lists removed packages `ws/`, `ia/`, `workflow/`, `tcp/`, `wsp/`, `instances/`). `NEW_PROJECT_GUIDE.md` has verified templates for starting a new project on `et` (HTTP microservice like `core-studio/api`, CLI like `tick`); `go run ./cmd/create` scaffolds `Project`/`Microservice`/`Modelo`/`Rpc` (`create modelo` prints a manual wiring step since `file.MakeFile` never overwrites).

## Commands

```bash
go build ./...                       # build + vet are currently clean
go vet ./...
gofmt -w .
go test ./jwf/ -run TestName -race   # prefer one package over go test ./...

go run ./cmd/server   # TCP node (port 1377, -port flag)
go run ./cmd/jrex     # JS runtime, hot-reload from ./cmd/jrex/src/
go run ./cmd/jsql     # jsql driver demo
go run ./cmd/ia       # RAG HTTP API (PORT, default 3300)

./version.sh --major | --minor | --request   # bumps git tag, updates README.md, pushes
```

Tests exist only in `queue/` (`Example_queue`), `ia/loader_test.go`, `jwf/workflow_test.go` (use `-race`), and `jrpc/`, `jws/`, `cache/`, `aws/`, `brevo/` — the latter need live Redis/AWS/Brevo and fail or hang without them. The sqlite driver and `ia`'s ingest/ask path have no tests.

## Code style

GoDoc for every function, in English: starts with the function name + brief description, one line for params, one for returns. **But** most existing files (`et/json.go`, `jwf/`, `resilience/`, `stores/`) use a `/** ... @param ... @return ... **/` block style — match whatever the surrounding file uses.

## Architecture

Modular utility library for Go microservices: every directory is an independent package, no central entry point.

### Core type: `et.Json`

`et/json.go`: `Json` (`map[string]interface{}`) is the lingua franca of the whole library — typed accessors (`Str`, `Int`, `Bool`, `Time`, `Json`, `Array`…), default-value variants (`ValStr(def, keys...)`), nested traversal via variadic keys. `ScanRows` decodes `[]byte` and JSON-looking `string` values (`{`/`[` prefix). `et/list.go` `List` is the paginated result (`Rows`, `All`, `Count`, `Page`, `Start`, `End`, `Result []Json`); `et/item.go`/`items.go` are single/multi result wrappers. Data types (`et.TypeData`, `et/condition.go`): `KEY` (VARCHAR 80), `TEXT`, `MEMO`, `INT`, `FLOAT`, `BOOL`, `DATETIME`, `JSON`, `BYTE`, `ANY`, `ARRAY`, `ARRAY_JSON|STRING|INT|FLOAT|BOOL|DATETIME`. There is no `GEOMETRY`/`EMBEDDING`/`STRING`.

### SQL builder / ORM: `jsql/`

- `jsql.Load()` / `jsql.LoadTo(name)` → `*DB`; config and tenant (`DB_TENANT_ID`, default `tenant:root`) are read internally via `envar`.
- Drivers in `jsql/drivers/<name>/`, self-register via `init()`, imported as side effect: `postgres` (`lib/pq`) and `sqlite` (`modernc.org/sqlite`). `mysql`/`mssql`/`oracle`/`josefina` are constants only, no implementation.
- `Driver` interface (`jsql/driver.go`): `Connect(db)`, `Load(model)` (DDL), `Query(query)` (SELECT), `Command(command)` (DML) — each returns SQL text.
- sqlite quirk: queries return each row as one JSON object; a nested JSON column inside it comes back double-encoded (see `ia/similarity.go` `chunkEmbedding`).

```go
model, _ := db.DefineModel("public", "users", 1, userId) // adds id, created_at, updated_at, _source, _idx

model, _ := db.Define(jsql.Def{                            // struct-based, preferred for complex models
    Schema: "public", Name: "users", Version: 1, IdxField: jsql.IDX,
    Columns: []jsql.Column{
        {Name: "email", TypeData: et.TEXT, Default: ""},
        {Name: "name", TypeColumn: jsql.ATTRIB, TypeData: et.TEXT, Default: ""},
    },
    PrimaryKeys: []jsql.DefIndex{{Name: "email", Sorted: true}},
    Unique:      []jsql.DefIndex{{Name: "email"}},
})
model.Init() // runs DDL

items, _ := model.Where(jsql.Eq("status", jsql.ACTIVE)).And(jsql.More("age", 18)).Limit(20).Page(1).All()
item, _  := model.Where(jsql.Eq("id", id)).One()
_, _ = model.Insert(et.Json{...}).ExecTx(nil)   // also Update(...).Where(...), Upsert(...)
```

Manual alternative: `db.NewModel(...)` + `DefineColumn`/`DefinePrimaryKey`/`DefineUnique`/`DefineAttrib`/`DefineForeignKeys`. Package-level `jsql.Define(dbName, def)` uses the DB registry.

- **`TypeColumn`**: `COLUMN` (real column), `ATTRIB` (key inside `_source` JSONB, cast on read), `DETAIL`/`ROLLUP` (virtual relations), `CALCFUNC` (`CalcFunction` callback), `CALC` (query-time expression), `AGG`.
- **Column constants** (`jsql/column.go`): `ID`, `IDX` (`_idx`), `IDT`, `SOURCE` (`_source`), `STATUS`, `TENANT_ID`, `PROJECT_ID`, `CREATED_AT`, `UPDATED_AT`. `_idx` is a `reg.UUID()` set by an auto `BeforeInsert` trigger, not a sequence.
- **Status constants**: `ACTIVE`, `ARCHIVED`, `CANCELED`, `PENDING`, `APPROVED`, `REJECTED`, `OF_SYSTEM`, `FOR_DELETE`; extend `jsql.Status` with `jsql.SetStatus`.
- **Triggers**: before/after × insert/update/delete, `TriggerFunction func(tx *Tx, old, new et.Json) error`.
- **Nested JSONB paths**: `"a->b->c"` field names are translated to `->`/`->>` chains with casts on ATTRIB leaves.
- `.Debug()` logs SQL and skips execution; `.Test()` generates SQL without executing (on `Model`, `Query`, `Command`).
- `jsql.DefineStore(db, schema)` → `*jsql.Store` (generic `db_catalogs` kind+id store); `jsql.DefineSeries(db, schema)` → series/code generator.

### HTTP servers

- **`server/`** — lightweight `chi.Mux` wrapper, no Redis/NATS.
- **`ettp/v2/`** — `ettp.New(name, *Config)`; `Config` has `Port`, `RpcPort` (zero = OS-assigned), `Parent`, timeouts, `AllowOrigin`, TLS, `Transport`, `Debug`. It does **not** call `cache.Load()`/`event.Load()` — callers must. Syncs router state across replicas over NATS (`EVENT_SET_ROUTER`/`REMOVE`/`RESET`; `m.Myself` prevents self-processing).
- **`ettp/v1/`** — older but not dead (has apigateway/proxy/token/cache files with no v2 equivalent); check which version a `cmd/` imports.
- **`router/`** — standalone router used by `ettp/v2`.

### Infrastructure (external services)

- **`cache/`** Redis (`Set`/`Get`/`Delete`/`Pub`/`Sub`), **`event/`** NATS (`Subscribe`/`Publish`/`Stack`), **`graph/`** Neo4j.
- **`jrpc/`** — `net/rpc` over TCP; `jrpc.Mount(host, port, services, packageName)` with a simple `Solver` registry.
- **`jtcp/`** — distributed TCP node with Raft-style election (`Follower`/`Candidate`/`Leader`/`Proxy`), balancer; `jtcp.NewNode(port)` used by `cmd/server`.

### Utility packages

- **`envar/`** — `GetStr/GetInt/GetInt64/GetFloat/GetBool` with defaults, `Arg*` for CLI args, `Validate([]string)`. The old `config/` package is gone (only stale strings remain in `create/template/templates.go`).
- **`logs/`** (`Info`, `Alert`, `Error`, `Debug`, `Fatal`→`os.Exit(1)`, `…f` variants) over **`stdrout/`**.
- **`jwt/`** (tokens stored in `cache`) over **`claim/`** (HS256, `SECRET`).
- **`crontab/`** — event-driven; `crontab.Load(tag, store)` sets a package singleton, then `crontab.CronJob(tag, ownerId, Cron{...}, repetitions, params, fn)` / `ScheduleJob(tag, ownerId, time.Time, params, fn)`. HTTP handlers publish control events.
- **`jval/`** and **`validator/`** — two unrelated fluent validators; don't mix types. `validator` conditions: `Required`/`Min`/`Max`/`Between`/`MinLength`/`MaxLength`/`Pattern`/`NotEmpty` and "at least one" checks `IsLetters`/`IsNumbers`/`IsSpecialCharacters`; handles `[]any` from decoded JSON.
- **`queue/`** — `queue.New[T](queueSize, maxEvents, period, handler)`; flushes on `maxEvents` or `period`, `.Close()` flushes.
- **`request/`** (`URLParam`, `GetBody`, outbound `Fetch` — JSON/form only), **`response/`** (`ITEM`, `ITEMS`, `HTTPError`), **`middleware/`**, **`msg/`** (shared message constants).
- **`xls/`**, **`csv/`** — same API shape: `Read*(data|multipart|file)` → reader; `New*(data, columns)` → `ToFile`/`ToWriter`/`ToHttp` (columns derived from keys if omitted).
- **`utility/`**, **`reg/`** (IDs: UUID/ULID/Snowflake), **`file/`** (watcher), **`dt/`** (Redis in `PRODUCTION`, file otherwise), **`jws/`** (websocket), **`service/`** (OTP), **`mem/`**, **`ephemeral/`**, **`iterate/`**, **`race/`**, **`cmds/`**, **`timezone/`**, **`units/`**, **`color/`**.
- Messaging: **`aws/`** (S3/SES/SMS), **`brevo/`**, **`jwsp/`** (WhatsApp Graph API), **`infobip/`** (`NewSenderInfobip(Params{BaseUrl, ApiKey, Sender})`; `SendSMS`/`SendEmail` (API v4 JSON)/`SendWhatsApp` (templates only) take `tpMessage` `"Transactional"|"Promotional"`; `{{key}}` substitution like brevo).

### Application packages

- **`jrex/`** — JS runtime (`goja`). `jrex.Load(tag, store)` (nil store → `FileStore` at `./src`); `Set(name, value)` injects Go bindings; `Run()`/`RunModule(tag)`; `RunDev()` hot-reloads via `file.NewWatcher`. Globals: `console`, `ctx`, `fetch`, `require`. `jcli/` (Bubble Tea, declares `package jrex`) is orphaned WIP.
- **`jia/`** — OpenAI agents (`openai-go/v3`): `jia.New(tag, store, userId)`, `jia.Load(id, store)`; Agents, Participants, Conversations/Messages, Skills (`ApiSkill`); not tenant-scoped. **Bug:** `jia/ia.go:90` calls `store.Get(id, packageName, …)` with args swapped.
- **`ia/`** — multitenant RAG (unrelated to `jia`). `ia.Load(db, schema, Config)` defines `ia_documents`/`ia_chunks`/`ia_conversations`/`ia_messages`, all with `tenant_id` + `project_id`. `IngestFile` (pdf/docx/xlsx/csv/txt/md loaders in `ia/loader_*.go`) and `IngestSQL` → chunk → embed → store. No pgvector: embeddings are JSON `[]float64`, and `Ask` ranks by cosine similarity in Go over up to `DB_RECORD_LIMIT` (default 1000) chunks, then answers with `Chat.Completions`. `embedFn`/`answerFn` are swappable for tests. `LoadRouter(chi.Router)`; design in `featureIa.md`.
- **`stores/`** — jsql-backed `DefineInstance`/`LoadInstance` (Json/Bite kinds), `DefineAuthorization` (ACL cached through `dt`), `DefineConfig` (per-tenant settings). `stores.Instance.Get(id, dest)` has one key, so it doesn't satisfy the `Store` interfaces below.
- **`resilience/`** — retry: `resilience.New(store)`, `LoadInstance(Params{Id, Tag, Description, TotalAttempts, Interval, Tags, Fn, FnArgs})`, `instance.Run(userId)`. Used by `jwf`.
- **`jwf/`** — workflow engine, detailed below.

### `jwf/` (workflows)

`jwf.New(db *jsql.DB, id, userID)` calls `cache.Load()` + `event.Load()`, builds its own `Storage` (schema hard-coded `"workflows"`), uses `reg.GetUUID(id)` (blank → new UUID). `jwf.Load(db, id, userId)` falls back to `New` if not found. A nil db is not supported.

```
WorkFlow (own ID, not tenant-scoped)
  Steps map[string]*Step           shared step pool
  Flows map[string]*Flow
    Steps        subset of the pool, incl. triggers
    Connections  Source/Target {StepId, Port, Index} + Kind input/output/error
    Triggers     Tag -> StartId
```

```go
wf, _ := jwf.New(db, "", userId)
flow := wf.NewFloW("add", "add item", "1.0.0", userId). // "FloW" typo is the real name
    Step("add", "step 1", func(instance *jwf.Instance, ctx et.Json) (et.Json, error) { return et.Json{}, nil }).
    Step("add", "step 2", fn)
// Run(tag, triggerTag, id, projectId, code string, ctx, tags et.Json, await bool, userId string)
result, err := wf.Run(flow.ID, "add", "", projectId, "", et.Json{}, et.Json{}, true, userId)
```

- First `Step` becomes a trigger; the next ones chain through output connections. `flow.Error(...)` attaches an error-port step to the last step. `Step.Definition` can also be a JS body run through `jrex`.
- Instances are not kept in memory: `Run` creates one (`newInstance`, code from `store.GenSerie(tag+":"+projectId)` when empty) or loads it from `"instances"`; a cache key `instance:<id>:status` (TTL `Flow.TimeAwait`) flags running ones. Status: `CREATED`, `PENDING`, `RUNNING`, `ROLLBACK`, `DONE`, `FAILED`, `CANCEL`, `STOP`. On a step error it retries via `resilience` (if `TotalAttempts != 0`), then follows the error connection.
- `jwf.Store`: `Set`/`Get`/`Delete`/`Query(collection, query)` + `SetSeries`/`GetSeries`/`DeleteSeries`/`GenSerie`/`GenValue`. Default impl `jwf.DefineStore(db, schema)` (models `workflows`/`flows`/`steps`/`instances` + `jsql.Series`).
- `LoadRouter(Router)` registers 11 routes; only the Steps handlers are implemented — the Flows/Instances handler bodies are empty.
- Quirk: `Flow.Step`/`Error` pass the flow ID as "userId" into step audit logs.
- `AuditLog []et.Json` + `onSave`/`onDelete` hooks (`MAX_AUDIT_LOG`, default 1000) is a reusable pattern documented in `AUDITLOG.md`.

### `cmd/` binaries

`et` (cobra CLI), `apigateway` (ettp), `daemon` (systemd), `create` (scaffolder, templates in `create/`), `server`/`client` (jtcp), `jrex`, `jsql`, `install`, `whatcher`, `resilience`, `wsp`, `ia` (on `server/`), `jwf` — **stale**: calls `jwf.New(nil, …)`, so it compiles but fails at runtime.

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
| `jsql` | `DB_DRIVER`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_TENANT_ID` (`tenant:root`), `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_CONN_LIFETIME`, `DB_POOL_CONN_IDLE_TIME`, `DB_RECORD_LIMIT` |
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
