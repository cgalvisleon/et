# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Module

`github.com/cgalvisleon/et` — Go 1.25, MIT license.

> **Note:** This codebase changes fast and in-place (commit history is mostly bare "Backup:" commits with no descriptive messages) — prefer reading the actual source over trusting prior assumptions about an API shape, including the rest of this file. `LIBRARY_CONTEXT.md` (Spanish, for external consumers of the library) and `README.es.md` can also drift out of sync with the code; verify before relying on them.

## Commands

```bash
# Run all tests
go test ./...

# Run a single package's tests
go test ./et/...
go test ./cache/...

# Format code
gofmt -w .

# Run CLI commands
go run ./cmd/et
go run ./cmd/apigateway
go run ./cmd/daemon
go run ./cmd/server          # TCP node server (default port 1377, use -port flag)
go run ./cmd/jrex            # JS VM (jrex) with hot-reload from ./cmd/jrex/
go run ./cmd/jsql            # jsql driver test/demo

# Build all binaries
go build ./...

# Semantic versioning (reads git tags, updates README.md, pushes new tag)
./version.sh --major | --minor | --request
```

> **Note:** There are currently no `*_test.go` files in the repo — `go test ./...` will compile but find nothing to run.

## Code style

### Comments

Always generate GoDoc comments for all functions.

GoDoc comments must:

- Start with the function name and Brief description.
- Be written in English.
- Describe one line to parameters and one line return values.
- Be concise and professional.

> **Note:** most existing code (e.g. `et/json.go`, and the newer `jwf/`, `resilience/`, `stores/` packages) actually uses an older `/** ... @param ... @return ... **/` block style, not real GoDoc syntax. Follow that existing convention when editing those files for consistency; use real GoDoc (`// FuncName: ...`) only where the surrounding file already does.

## Architecture

This is a **modular utility library** for building Go microservices. Each directory is an independent package imported separately. There is no central entry point — consumers import only the packages they need.

### Core type: `et.Json`

`et/json.go` defines `Json` (`map[string]interface{}`), the primary data structure used throughout the entire library. It has typed accessors (`Str`, `Int`, `Bool`, `Time`, `Json`, `Array`, etc.) with a default-value pattern (`ValStr(def, keys...)`) and nested key traversal via variadic `atribs ...string`. This type is the lingua franca across all packages.

`et/list.go` defines `List` — the standard paginated result type (`Rows`, `All`, `Count`, `Page`, `Start`, `End`, `Result []Json`).

`et/item.go` and `et/items.go` define single-item and multi-item result wrappers.

### SQL builder: `jsql/`

`jsql/` is a database-agnostic SQL builder and lightweight ORM. Entry points: `jsql.Load(tenantId string) (*DB, error)` (connects to the default DB) and `jsql.LoadTo(tenantId, name string) (*DB, error)` (connects to a named DB). There is no config object to pass in anymore — connection settings (`DB_DRIVER`, `DB_HOST`, etc.) are read internally via `config.GetStr`/`config.GetInt` (package-level functions in `config/`, backed by `envar`). `jsql.DriverPostgres`, `DriverSqlite`, `DriverMysql`, `DriverMssql`, `DriverOracle`, `DriverJosefina` are declared as driver-name constants (`jsql/driver.go`). Only `postgres` has an actual implementation under `jsql/drivers/postgres/`. `getConnection` (`jsql/jsql.go`) still has a `case DriverSqlite` branch that builds a `*SqliteConection`, but **there is no `jsql/drivers/sqlite/` directory at all** — nothing registers a `Driver` for `"sqlite"`, so setting `DB_DRIVER=sqlite` will reach `ConnectTo` and fail to resolve a driver. `mysql`/`mssql`/`oracle`/`josefina` constants exist with no backing implementation either (`jsql/drivers/mysql/` and `jsql/drivers/josefina/` are empty directories; there's no `jsql/drivers/mssql/` or `jsql/drivers/oracle/` directory at all). In practice, **`postgres` is the only working driver today.**

**Model definition:**

```go
// Full-featured model (adds id, created_at, updated_at, _source JSONB, _idx VARCHAR(80)):
model, _ := db.DefineModel("public", "users", 1)

// Manual model (add every column yourself):
model, _ := db.NewModel("public", "users", 1)
model.DefineColumn("email", jsql.TEXT, "")
model.DefinePrimaryKey("id", jsql.KEY, "")
model.DefineUnique("email", jsql.TEXT, "")
model.DefineAttrib("name", jsql.TEXT, "")   // stored inside _source JSONB
model.DefineForeignKeys(orders, map[string]string{"order_id": "id"}, true, false)
model.Init()  // executes DDL (CREATE TABLE, indexes, FK constraints)
```

**Struct-based model definition (preferred for complex models):**

```go
model, _ := db.Define(jsql.Def{
    Schema:  "public",
    Name:    "users",
    Version: 1,
    IdxField: jsql.IDX,
    Columns: []jsql.Column{
        {Name: "email", TypeData: jsql.TEXT, Default: ""},
        {Name: "name", TypeColumn: jsql.ATTRIB, TypeData: jsql.TEXT, Default: ""},
    },
    PrimaryKeys: []jsql.DefIndex{{Name: "email", Sorted: true}},
    Unique:      []jsql.DefIndex{{Name: "email"}},
})
model.Init()
```

Package-level wrapper: `jsql.Define(dbName, def)` looks up the named DB from the registry.

**Key column types (`TypeColumn`):**

| `TypeColumn`        | Meaning                                                                                              |
| ------------------- | ---------------------------------------------------------------------------------------------------- |
| `COLUMN`            | Real SQL column                                                                                      |
| `ATTRIB`            | Key inside `_source` JSONB — accessed via `_source->>'field'` or with cast for numeric/bool/datetime |
| `DETAIL` / `ROLLUP` | Virtual relationship fields, not stored as columns                                                   |
| `CALCFUNC`          | Computed column via a registered `CalcFunction` callback                                             |
| `CALC`              | Computed expression evaluated at query time                                                          |
| `AGG`               | Aggregation column                                                                                   |

**Key data types (`TypeData`):** `KEY` (VARCHAR 80, used for IDs and `_idx`), `TEXT`, `MEMO`, `INT`, `FLOAT`, `BOOLEAN`, `DATETIME`, `JSON`, `BYTES`, `GEOMETRY`, `EMBEDDING`, `ANY`.

**Column name constants** (exported from `jsql/column.go` for use in queries and `Def`):
`jsql.ID`, `jsql.IDX` (`_idx`), `jsql.IDT` (`_idt`), `jsql.SOURCE` (`_source`), `jsql.STATUS`, `jsql.TENANT_ID`, `jsql.PROJECT_ID`, `jsql.CREATED_AT`, `jsql.UPDATED_AT`.

**Status constants:** `jsql.ACTIVE`, `jsql.ARCHIVED`, `jsql.CANCELED`, `jsql.PENDING`, `jsql.APPROVED`, `jsql.REJECTED`, `jsql.OF_SYSTEM`, `jsql.FOR_DELETE`. The `jsql.Status` map holds all non-active statuses; extend it with `jsql.SetStatus(value)`.

`IdxField` (`_idx`) is `VARCHAR(80)` (`KEY` type); its value is a `reg.ULID()` set by an auto-registered `BeforeInsert` trigger — **not** a database serial/sequence.

**Model triggers:** `Model` supports six trigger slices (`beforeInserts`, `beforeUpdates`, `beforeDeletes`, `afterInserts`, `afterUpdates`, `afterDeletes`) each accepting `TriggerFunction` (`func(tx *Tx, old, new et.Json) error`). Computed columns use `CalcFunction` (`func(tx *Tx, data et.Json)`), registered via `model.calcs`.

**Query / Command API (fluent):**

```go
items, _ := model.Where(jsql.Eq("status", jsql.ACTIVE)).
    And(jsql.More("age", 18)).
    Limit(20).Page(1).All()

item, _ := model.Where(jsql.Eq("id", id)).One()

_, _ = model.Insert(et.Json{"email": "a@b.com"}).ExecTx(nil)
_, _ = model.Update(et.Json{"status": "archived"}).Where(jsql.Eq("id", id)).ExecTx(nil)
_, _ = model.Upsert(et.Json{"id": id, "email": "a@b.com"}).ExecTx(nil)
```

**Nested JSONB paths:** field names use `->` as a path separator (e.g. `"ventas->detalle->precio"`). The condition builder and `BuildSelectField` translate these to the correct PostgreSQL `->`/`->>` chain automatically, with type casts for ATTRIB leaves.

**Driver interface (`jsql/driver.go`):**

```go
type Driver interface {
    Connect(db *DB) (*sql.DB, error)
    Load(model *Model) (string, error)        // DDL generation
    Query(query *Query) (string, error)       // SELECT generation
    Command(command *Command) (string, error) // DML generation
}
```

Implementations live in `jsql/drivers/<name>/` and self-register via `init()`. The only active driver is `postgres` (`lib/pq`). Import as a side-effect: `import _ "github.com/cgalvisleon/et/jsql/drivers/postgres"`. (`go.mod` does not even list a sqlite Go driver dependency, e.g. `mattn/go-sqlite3` — confirming sqlite support was removed, not just unfinished.)

**Debug / Test mode:** both `Model`, `Query`, and `Command` support `.Debug()` (logs SQL, skips execution) and `.Test()` (generates SQL, skips execution). Both return the receiver for chaining.

### HTTP server packages

There are two HTTP server packages at different abstraction levels:

- **`server/`** — Lightweight HTTP server (`Ettp` struct wrapping `chi.Mux`). No external service dependencies. Use when Redis/NATS are not needed.
- **`ettp/v2/`** — Full-featured HTTP server. `ettp.New(name string, cnf *Config) (*Server, error)` — `Config` has `Port`, `Parent`, timeouts, `AllowOrigin`, TLS fields, and `UseCache`/`UseEvent` bool flags that conditionally call `cache.Load()`/`event.Load()` inside `New` (so Redis/NATS are only required if those flags are set). Router state is synchronized across instances via NATS events (`EVENT_SET_ROUTER`, `EVENT_REMOVE_ROUTER`, `EVENT_RESET_ROUTER`). The `m.Myself` flag prevents self-processing. `ettp/v1/` is the older version (untouched since 2026-06-02); prefer `v2`.
- **`router/`** — Standalone router package (used internally by `ettp/v2`). Can be imported directly for custom HTTP routing without the full server setup.

### Infrastructure packages (require external services)

- **`cache/`** — Redis client (requires `REDIS_HOST`, optionally `REDIS_PASSWORD`, `REDIS_DB`). `cache.Load()` initializes; provides `Set`, `Get`, `Delete`, `Pub`, `Sub`.
- **`event/`** — NATS pub/sub (requires `NATS_HOST`, optionally `NATS_USER`, `NATS_PASSWORD`). `event.Load()` initializes; provides `Subscribe`, `Publish`, `Stack`.
- **`jrpc/`** — Go `net/rpc` over TCP (not NATS). `jrpc.Mount(host, port, services, packageName)` registers a service; includes load balancing (`balancer.go`) and Raft consensus (`raft.go`).

### Self-contained utility packages

- **`config/`** — Package-level env getters: `GetStr`, `GetInt`, `GetInt64`, `GetFloat`, `GetBool`, `Get`, `Set`, `Validate`, `IsLoad` (all backed by `envar/`). There is no `config.App` struct — `config.Config` is a different, store-backed settings object (`ID`, `TenantId`, `OwnerId`, `Tag`, `Stage`, `Params`, `AuditLog`) created via `config.New(tag, stage, tenantId, ownerId, store, userId)` / loaded via `config.Load(...)`, with its own `Store` interface (`Get`/`Set`/`Delete`) for persistence — it's a per-tenant settings record, not a process-wide app descriptor.
- **`envar/`** — Low-level env var access; `envar.GetStr/GetInt/GetBool/GetFloat` read env vars with defaults, `envar.ArgStr/ArgInt/ArgBool/...` read CLI args, `envar.Validate([]string{...})` checks required vars exist.
- **`logs/`** — Structured logging. Functions: `Log`, `Info`, `Infof`, `Alert`, `Alertf`, `Error`, `Errorf`, `Debug`, `Debugf`, `Fatal`, `Panic`, `Tracer`. All route through `stdrout` for colorized output.
- **`jwt/`** — High-level token creation: `New`, `NewAuthentication`, `NewAuthorization`, `NewAppToken`. Stores tokens in `cache`. Built on top of `claim/`.
- **`claim/`** — JWT claims struct with `tenantId`. `GenToken` signs with HS256 (`golang-jwt/jwt/v4`).
- **`crontab/`** — Job scheduler. `crontab.New(tag string, store Store)` creates a scheduler (calls `event.Load()` internally — note `store` is now a required second argument, not optional); `AddJob`, `AddOneShotJob`, `AddEventJob` register jobs. Supports `robfig/cron` spec format including seconds (`cron.WithSeconds()`, e.g. `"0 * * * * *"`).
- **`jval/`** — Fluent validation rules for `et.Json`. Implements `Rule` interface with typed validators (`Str`, `Int`, `Float`, `Bool`, `Email`, `Phone`, `Time`, etc.); chainable constraints (`.NotEmpty()`, `.Min()`, `.Max()`, etc.).
- **`request/`** — Both inbound helpers (`URLParam`, `GetBody`) and outbound HTTP client utilities. `URLParam(r, "key").Str()` reads chi route params; `GetBody(r)` parses the JSON body into `et.Json`.
- **`strs/`** — String utilities.
- **`utility/`** — Crypto, validation, ID generation (UUID, Snowflake, ULID), general helpers.
- **`middleware/`** — HTTP middleware (CORS, request ID, logger, auth, telemetry, panic recovery).
- **`response/`** — Unified HTTP response helpers. Key functions: `ITEM(w, r, status, et.Item{})`, `ITEMS(w, r, status, items)`, `HTTPError(w, r, status, message)`.
- **`ws/`** — WebSocket support via `gorilla/websocket`.
- **`service/`** — OTP helpers (`SendOTPEmail`, `SendOTPSms`, `VerifyOTP`) and messaging integration; uses `tenantId`.
- **`stdrout/`** — Low-level colorized stdout routing used by `logs/`.

### Integration packages

- **`aws/`** — AWS SDK wrapper: S3, SES (email), SMS.
- **`brevo/`** — Brevo API client: email, SMS, WhatsApp.
- **`wsp/`** — WhatsApp Business API client. `wsp.NewSender(token, phoneNumberId) *Whatsapp` produces a message/webhook builder (`Debug()`, `IsTest()`, `SetVerifyToken()`, `SetEventHandler()`); uses Facebook Graph API (configurable via `WHATSAPP_API_URL`).

### Application-layer packages

- **`jrex/`** — JavaScript runtime package (`dop251/goja`), formerly named `vm`. `jrex.New(name, store)` is the entry point — `store` is a caller-provided `jrex.Store` (`SetModule`/`GetModule`/`DeleteModule`, may be `nil` in dev mode); three modes: `Develop` (reads files, hot-reloads via `file.Watcher`), `Production` (loads from the `Store`), `Building` (compiles + stores with semver bumping). Global wrappers provide `console.*`, `ctx.*`, `fetch()`, and CommonJS-style `require()`. The `cmd/jrex` binary runs this in dev mode via `jrex.RunDev(baseDir)`. There is also a `jcli/` package at the repo root (not under `jrex/`) holding a Bubble Tea CLI model (`cliModel`, `App` interface) — its file still declares `package jrex` and nothing imports `github.com/cgalvisleon/et/jcli`, so treat it as in-progress/orphaned work extracting the dev CLI out of `jrex/`, not a wired feature.
- **`ia/`** — OpenAI agent integration (`openai-go/v3`). `ia.New(tenantId, tag string, store Store) (*Ia, error)` (direct) or `ia.Load(tenantId, tag string, store Store) error` (singleton) initializes the package, which calls `event.Load()` internally (no `cache.Load()`). `Store` is a locally-defined interface in `ia/ia.go` (`Set(id, tag, tenantId, ownerId string, obj any, userId string) error`, `Get(id, tag string, dest any) (bool, error)`, `Delete(id, tag string) error`, `Query(query et.Json) (et.Items, error)`) — caller-provided, no shared package defines it. `OPENAI_API_KEY` is read directly via `config.GetStr`, not passed in. Manages `Agent`s, `Participant`s, and `Conversation`s (with `Message`s) per tenant; `Skill` (e.g. `ApiSkill`) lets agents call external APIs.
- **`jwf/`** — Workflow orchestration (replaces the old `workflow/` package, which was deleted; a separate `wf/` scratch package was also deleted). Calls `cache.Load()` + `event.Load()` internally. See detail below.
- **`graph/`** — Neo4j connectivity (`neo4j-go-driver/v5`). `graph.Load()` returns a `*Conn` with the Neo4j driver.
- **`stores/`** — jsql-backed persistence helpers: `stores.DefineInstance`/`LoadInstance`/`DefineInstanceBite`/`LoadInstanceBite` (kind: `KindJson` or `KindBite`) and `stores.DefineAuthorization`. **Caveat:** `(*stores.Instance).Get(id string, dest any) (bool, error)` only takes one string key, so it does *not* structurally satisfy the two-string-key `Store.Get` interfaces defined locally in `ia/` and `jwf/` (`Get(id, tag string, dest any)` / `Get(collection, id string, dest any)`) — nothing in the repo currently wires `stores/` into `ia` or `jwf` (both are exercised with a `nil` store in their `cmd/` examples). Check signatures before assuming it's a drop-in.
- **`dt/`** — Cache-backed object store. `dt.Up(key, data)` writes an object (uses Redis in production, reads from file in dev based on `PRODUCTION` env var); `dt.Get(key)` retrieves it, `dt.Drop(key)` removes it. HTTP handler support via `handler.go`.
- **`resilience/`** — Resilience/retry pattern. `resilience.New(store)` (store: local `Store` interface — `Set`/`Get`/`Delete`/`Query`), `LoadInstance(Params{TenantId, Id, Tag, Description, OwnerId, TotalAttempts, Interval, Tags, UserId, Fn, FnArgs})` returns an `*Instance`, then `instance.Run(userId)` executes `Fn` with retry/interval semantics. Used by `jwf` for step-level resilience; see `cmd/resilience` for a standalone example.
- **`reg/`** — Service registration/discovery; provides ID generation helpers (ULID, etc.) used by `claim` and others.
- **`file/`** — File operations and watching (`FileInfo`, `Watcher`, `ExistPath()`); used by `jrex` for hot-reload.
- **`mem/`** — Shared memory and sync primitives.
- **`ephemeral/`** — Ephemeral/temporary data structures.
- **`iterate/`** — Iteration control with time support.
- **`race/`** — Race condition detection helpers.
- **`cmds/`** — Command/stage execution system (distinct from the `cmd/` CLI binaries).
- **`timezone/`**, **`units/`**, **`color/`** — Timezone handling, unit conversions, terminal color utilities.

### TCP cluster: `tcp/`

`tcp/` implements a distributed TCP node with Raft-style leader election. Modes: `Follower`, `Candidate`, `Leader`, `Proxy`. `tcp.NewNode(port)` is used by `cmd/server`.

### CLI (`cmd/`)

Each subdirectory under `cmd/` is a standalone binary:

- `cmd/et/` — Main CLI using `cobra`
- `cmd/apigateway/` — API Gateway/proxy using `ettp.New`
- `cmd/daemon/` — Background service with systemd integration (start/stop/restart/status/conf/version)
- `cmd/create/` — Project/code scaffolding
- `cmd/server/` — TCP node server (`tcp.NewNode(port)`)
- `cmd/jrex/` — JavaScript VM runner; `go run ./cmd/jrex` starts `jrex.New("jrex", nil).RunDev(baseDir)` with hot-reload
- `cmd/jsql/` — jsql driver demo: DDL generation, condition building, SELECT field resolution, live DB connection
- `cmd/client/` — Test client
- `cmd/install/` — Installation utility
- `cmd/whatcher/` — Filesystem change watcher
- `cmd/jwf/` — `jwf/` (workflow) usage example: builds a flow with `NewFloW`/`Step`, runs it with `wf.Run(...)`
- `cmd/resilience/` — `resilience/` usage example: `resilience.New(nil)`, `LoadInstance(Params{...})`, `ins.Run(userId)`
- `cmd/wsp/` — `wsp/` (WhatsApp) usage example

### Code generation (`create/`)

Templates and generators for new microservices, projects, and Kubernetes deployments. Used by the `cmd/create` CLI.

### `jwf/` package detail

> **Note:** This package replaces what used to be two separate packages: a top-level `workflow/` package and a `wf/` scratch/rewrite area — both were deleted (last seen 2026-06-16 and 2026-06-18 respectively) and consolidated into `jwf/`, which has a different API from either predecessor. References to `workflow.RunInstance`, `instances.Store`, etc. elsewhere (old branches, comments) describe the removed package, not `jwf/`.

`jwf.New(tenantId string, store Store) (*WorkFlow, error)` calls `cache.Load()` + `event.Load()` internally. `jwf.Load(tenantId string, store Store, userId string) (*WorkFlow, error)` is the store-backed variant that loads-or-creates a persisted `WorkFlow` (errors if `store` is `nil`, unlike `New`).

**Type hierarchy** (graph-based, not a linear step list):

```
WorkFlow (tenant-scoped container)
  -- Flows map[string]*Flow
        -- Steps map[string]*Step       (pool of steps, including one or more Triggers)
        -- Connections []*Connection    (Source/Target step + port: input/output/error)
        -- Triggers []*Trigger          (tag -> starting Step.ID)
  -- Instances map[string]*Instance     (runtime, in-memory unless backed by Store)
```

A `Flow` is built fluently: `flow.Step(tag, title, fn)` adds the first step as a `KindTrigger` (registering a `Trigger`) or chains an action step via an output `Connection`; `flow.Error(tag, version, title, fn)` attaches an error-port step to the most recently added step. `fn` has signature `func(instance *jwf.Instance, ctx et.Json) (et.Json, error)`; `Step.Definition` also accepts a JS string/`[]byte` body, executed via an embedded `jrex.Instance` instead of a Go closure.

```go
wf, _ := jwf.New(tenantId, nil) // nil store: in-memory only, nothing persisted
flow := wf.NewFloW("add", "add item", "1.0.0", userId). // note: "FloW", not "Flow"
    Step("add", "add item", func(instance *jwf.Instance, ctx et.Json) (et.Json, error) {
        return instance.SetParams(et.Json{"step1": "step1"}), nil
    }).
    Step("add", "add item", func(instance *jwf.Instance, ctx et.Json) (et.Json, error) {
        return instance.SetParams(et.Json{"step2": "step2"}), nil
    })

result, err := wf.Run(flow.ID, "add", "" /* instance id, blank = new */, projectId, et.Json{}, et.Json{}, userId)
```

`Instance` tracks `Status` (`CREATED`, `PENDING`, `RUNNING`, `ROLLBACK`, `DONE`, `FAILED`, `CANCEL`, `STOP`), advances step-to-step via `next()` following `Connections`, and on a step error falls back to `resilience.New` (when `Flow.TotalAttempts > 0`) before giving up.

**HTTP routing:** `(*WorkFlow).LoadRouter(r Router)` wires a small set of *lowercase* (unexported) handlers — `httpGetStep`/`httpNewStep`/`httpUpdateStep`/`httpSetDefinitionStep`/`httpDeleteStep` for steps, plus stub handlers for flows and instances (`httpGetFlow`, `httpSetFlow`, `httpStatusFlow`, `httpDeleteFlow`, `httpGetInstance`, `httpDeleteInstance`, `httpRunInstance` — currently **empty function bodies** in `jwf/router.go`, not yet implemented). `Router` is a minimal local interface (`Protect(method, path string, handler func(http.ResponseWriter, *http.Request))`), unrelated to the repo's `router/` package. These handlers call `response.JSON(...)` (not `response.ITEM`/`ITEMS`) and `request.UserId(r)` to read the authenticated user.

### HTTP handler pattern

All `handler.go` files across packages follow this pattern:

```go
func (s *T) HttpFoo(w http.ResponseWriter, r *http.Request) {
    // URL path params (chi router)
    id := request.URLParam(r, "id").Str()
    index := request.URLParam(r, "index").Int()

    // JSON body
    body, err := request.GetBody(r)
    if err != nil {
        response.HTTPError(w, r, http.StatusBadRequest, err.Error())
        return
    }
    tag  := body.Str("tag")
    idx  := body.Int("index")
    stop := body.Bool("stop")
    ctx  := body.Json("ctx")   // nested object
    tags := body.Json("tags")  // nested object

    // Responses
    response.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: data})
    response.HTTPError(w, r, http.StatusBadRequest, err.Error())
}
```

Use `http.StatusCreated` for POST handlers that create new resources, `http.StatusOK` for queries and mutations.

## Key patterns

- **Initialization pattern**: Infrastructure packages expose a `Load()` function that reads env vars via `envar` and establishes connections. Call `Load()` once at startup; subsequent calls are no-ops.
- **Error handling**: `logs.Fatal(err)` calls `os.Exit(1)`. Use `logs.Alert` / `logs.Error` for non-fatal errors.
- **Event-driven coordination**: `ettp/v2` server syncs router state across replicas via NATS. The `m.Myself` flag prevents self-processing.
- **`msg/` packages**: Each package has a local `msg/` or `msg.go` file with error message constants — use these instead of hardcoded strings.
- **Store interface pattern**: `jwf` and `ia` each accept a caller-provided `Store` for persistence — these are now two separately-defined, structurally-similar (but not identical or shared) interfaces local to each package (there is no shared `instances` package anymore); `jrex` accepts its own `jrex.Store` (`SetModule`/`GetModule`/`DeleteModule`) for compiled-module persistence in `Production`/`Building` modes; `resilience` and `config` each define their own local `Store` too. In every case the library defines the interface and consumers implement it — check the exact method signatures per package before reusing one implementation across packages (see the `stores/` caveat above).

## Required environment variables

| Package | Variable                                                                                  | Purpose                                |
| ------- | ----------------------------------------------------------------------------------------- | -------------------------------------- |
| `jsql`  | `DB_DRIVER`                                                                               | Driver name (`postgres`, `sqlite`, …)  |
| `jsql`  | `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`                                 | Database connection                    |
| `jsql`  | `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_CONN_LIFETIME`, `DB_POOL_CONN_IDLE_TIME` | Connection pool (optional)             |
| `cache` | `REDIS_HOST`                                                                              | Redis connection                       |
| `event` | `NATS_HOST`                                                                               | NATS connection                        |
| `event` | `NATS_USER`, `NATS_PASSWORD`                                                              | NATS auth (optional)                   |
| `graph` | `NEO4J_HOST`, `NEO4J_USER`, `NEO4J_PASSWORD`                                              | Neo4j connection                       |
| `ia`    | `OPENAI_API_KEY`                                                                          | OpenAI agent integration               |
| `dt`    | `PRODUCTION`                                                                              | `true` = use Redis, `false` = use file |
| `wsp`   | `WHATSAPP_API_URL`                                                                        | WhatsApp Graph API base URL (optional) |
| `claim` | `SECRET`                                                                                  | JWT signing key (default: `"1977"`)    |
