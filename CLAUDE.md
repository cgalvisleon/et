# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Module

`github.com/cgalvisleon/et` — Go 1.23+, MIT license.

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

All doc comments for functions, methods, and types must use this block style:

```go
/**
* FunctionName: Brief description.
* @param paramName type
* @return type
**/
```

Each `@param` and `@return` must be its own single line. For types and interfaces use only the first line (no `@param`/`@return`).

## Architecture

This is a **modular utility library** for building Go microservices. Each directory is an independent package imported separately. There is no central entry point — consumers import only the packages they need.

### Core type: `et.Json`

`et/json.go` defines `Json` (`map[string]interface{}`), the primary data structure used throughout the entire library. It has typed accessors (`Str`, `Int`, `Bool`, `Time`, `Json`, `Array`, etc.) with a default-value pattern (`ValStr(def, keys...)`) and nested key traversal via variadic `atribs ...string`. This type is the lingua franca across all packages.

`et/list.go` defines `List` — the standard paginated result type (`Rows`, `All`, `Count`, `Page`, `Start`, `End`, `Result []Json`).

`et/item.go` and `et/items.go` define single-item and multi-item result wrappers.

### SQL builder: `jsql/`

`jsql/` is a database-agnostic SQL builder and lightweight ORM. Entry points: `jsql.Load(config)` (connects to the default DB) and `jsql.LoadTo(config, name)` (connects to a named DB). `config` is a `jsql.Config` interface (`GetStr`/`GetInt`/`GetBool` — satisfied by `*config.App`); pass `nil` to read connection settings purely from env vars (`DB_*`).

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

| `TypeColumn` | Meaning |
|---|---|
| `COLUMN` | Real SQL column |
| `ATTRIB` | Key inside `_source` JSONB — accessed via `_source->>'field'` or with cast for numeric/bool/datetime |
| `DETAIL` / `ROLLUP` | Virtual relationship fields, not stored as columns |
| `CALCFUNC` | Computed column via a registered `CalcFunction` callback |
| `CALC` | Computed expression evaluated at query time |
| `AGG` | Aggregation column |

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

Implementations live in `jsql/drivers/<name>/` and self-register via `init()`. Active drivers: `postgres` (`lib/pq`), `sqlite` (`mattn/go-sqlite3`). Import as a side-effect: `import _ "github.com/cgalvisleon/et/jsql/drivers/postgres"`. The `josefina` and `mysql` driver directories exist but contain no files yet.

**Debug / Test mode:** both `Model`, `Query`, and `Command` support `.Debug()` (logs SQL, skips execution) and `.Test()` (generates SQL, skips execution). Both return the receiver for chaining.

### HTTP server packages

There are two HTTP server packages at different abstraction levels:

- **`server/`** — Lightweight HTTP server (`Ettp` struct wrapping `chi.Mux`). No external service dependencies. Use when Redis/NATS are not needed.
- **`ettp/v2/`** — Full-featured HTTP server. `ettp.New(name, config)` calls `cache.Load()` + `event.Load()` internally. Router state is synchronized across instances via NATS events (`EVENT_SET_ROUTER`, `EVENT_REMOVE_ROUTER`, `EVENT_RESET_ROUTER`). The `m.Myself` flag prevents self-processing. `ettp/v1/` is the older version; prefer `v2`.
- **`router/`** — Standalone router package (used internally by `ettp/v2`). Can be imported directly for custom HTTP routing without the full server setup.

### Infrastructure packages (require external services)

- **`cache/`** — Redis client (requires `REDIS_HOST`, optionally `REDIS_PASSWORD`, `REDIS_DB`). `cache.Load()` initializes; provides `Set`, `Get`, `Delete`, `Pub`, `Sub`.
- **`event/`** — NATS pub/sub (requires `NATS_HOST`, optionally `NATS_USER`, `NATS_PASSWORD`). `event.Load()` initializes; provides `Subscribe`, `Publish`, `Stack`.
- **`jrpc/`** — Go `net/rpc` over TCP (not NATS). `jrpc.Mount(host, port, services, packageName)` registers a service; includes load balancing (`balancer.go`) and Raft consensus (`raft.go`).

### Self-contained utility packages

- **`config/`** — App config/env with getters `GetStr`, `GetInt`, `GetBool`, `GetFloat`, `GetTime` and CLI param helpers `ParamStr`, `ParamInt`, etc. The `config.App` struct holds `name`, `version`, `company`, `host`, `port`, `stage`.
- **`envar/`** — Low-level env var access; `envar.Validate([]string{...})` checks required vars exist.
- **`logs/`** — Structured logging. Functions: `Log`, `Info`, `Infof`, `Alert`, `Alertf`, `Error`, `Errorf`, `Debug`, `Debugf`, `Fatal`, `Tracer`. All route through `stdrout` for colorized output.
- **`jwt/`** — High-level token creation: `New`, `NewAuthentication`, `NewAuthorization`, `NewAppToken`. Stores tokens in `cache`. Built on top of `claim/`.
- **`claim/`** — JWT claims struct with `tenantId`. `GenToken` signs with HS256.
- **`crontab/`** — Job scheduler. `crontab.New(tag)` creates a scheduler (calls `event.Load()` internally); `AddJob`, `AddOneShotJob`, `AddEventJob` register jobs. Supports `robfig/cron` spec format including seconds (`"0 * * * * *"`).
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
- **`wsp/`** — WhatsApp Business API client. `NewWhatsapp(token, phoneNumberId)` produces a message builder; uses Facebook Graph API (configurable via `WHATSAPP_API_URL`).

### Application-layer packages

- **`jrex/`** — JavaScript runtime package (`dop251/goja`), formerly named `vm`. `jrex.New(name, store)` is the entry point — `store` is a caller-provided `jrex.Store` (`SetModule`/`GetModule`/`DeleteModule`, may be `nil` in dev mode); three modes: `Develop` (reads files, hot-reloads via `file.Watcher`), `Production` (loads from the `Store`), `Building` (compiles + stores with semver bumping). Global wrappers provide `console.*`, `ctx.*`, `fetch()`, and CommonJS-style `require()`. The `cmd/jrex` binary runs this in dev mode via `jrex.RunDev(baseDir)`.
- **`ia/`** — OpenAI agent integration (`openai-go/v3`). `ia.New(tenantId, tag string, store Store, config Config) (*Ia, error)` (direct) or `ia.Load(tenantId, tag string, store Store, config Config) error` (singleton) initializes the package, which calls `event.Load()` internally. `Store` (`Set`/`Get`/`Delete`/`Query`) and `Config` (`GetStr`/`GetInt`/`GetBool` — satisfied by `*config.App`) are caller-provided, mirroring the `instances.Store` pattern; `stores/` provides a jsql-backed `Store`. Manages `Agent`s, `Participant`s, and `Conversation`s (with `Message`s) per tenant; `Skill` (e.g. `ApiSkill`) lets agents call external APIs. Requires `OPENAI_API_KEY` via `Config`.
- **`workflow/`** — Workflow orchestration with multi-step execution, instance state, and resilience patterns. `workflow.New(store)` creates a new instance; `workflow.Load(store)` is the singleton wrapper (no-op if already initialized). Integrates with `resilience/`, `instances/`, and `event/` (NATS) for async state sync. See detail below.
- **`graph/`** — Neo4j connectivity (`neo4j-go-driver/v5`). `graph.Load()` returns a `*Conn` with the Neo4j driver.
- **`instances/`** — `Store` interface (`Set`, `Get`, `Delete`, `Query`) used by `ia` and `workflow` for state persistence. Implementations are caller-provided.
- **`stores/`** — jsql-backed implementation of `instances.Store`. `stores.NewInstance(db, schema, name, kind)` creates a DB-backed store (kind: `KindJson` or `KindBite`). Use this as the concrete `Store` when a `jsql.DB` is available.
- **`dt/`** — Cache-backed object store. `dt.Up(key, data)` writes an object (uses Redis in production, reads from file in dev based on `PRODUCTION` env var); `dt.Get(key)` retrieves it. HTTP handler support via `handler.go`.
- **`resilience/`** — Resilience patterns (circuit breaker, etc.) used by `workflow`.
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

### Code generation (`create/`)

Templates and generators for new microservices, projects, and Kubernetes deployments. Used by the `cmd/create` CLI.

### `workflow/` package detail

> **Note:** `wf/` also declares `package workflow` and duplicates much of this package (flow, step, instance, router). It is an in-progress rewrite/scratch area not imported by anything else in the repo — treat `workflow/` as the active package unless told otherwise.

`workflow.Load(store instances.Store)` initializes the singleton and calls `event.Load()` internally.

**Type hierarchy:**

```
Flow (definition)
  └── Steper (named path/lane, identified by tag)
        └── Steps []int  (indexes into Flow.Steps)
  └── Steps []*Step  (all step definitions, shared pool)

Instance (runtime)
  └── runs one Steper of a Flow for a specific entity ID
```

- **`Flow`** — top-level definition (`tag`, `version`, `name`, `description`). Created via `NewFlow(tag, version, name, description, username)`.
- **`Steper`** — a named ordered path through steps within a flow. Multiple stepers can share steps from the same pool.
- **`Step`** — individual unit: `Definition` (executable), `Undo` (rollback), `Stop bool`.
- **`Instance`** — runtime execution. `ToJson()` returns `(et.Json, error)` (two values). `GetInstance(id string)` takes only `id`.

**Instance lifecycle methods on `*WorkFlow`:**

| Method | Purpose |
|---|---|
| `RunInstance(id, tag, step, ctx, tags, username)` | Start or resume an instance |
| `GetInstance(id)` | Fetch instance by ID |
| `ResetInstance(id, username)` | Reset to step 0, status PENDING |
| `RollbackInstance(id, username)` | Execute undo chain |
| `StopInstance(id, username)` | Halt execution |

**HTTP handlers in `handler.go`** (all on `*WorkFlow`):

Flow: `HttpGetFlow`, `HttpNewFlow`, `HttpDeleteFlow`
Step: `HttpNewStep`, `HttpSetStep`, `HttpDeleteStep`
Steper: `HttpNewSteper`, `HttpSetSteper`, `HttpDeleteSteper`
Steper↔Step wiring: `HttpAddStepFromSteper`, `HttpRemoveStepFromSteper`, `HttpMoveStepFromSteper`
Instance: `HttpGetInstance`, `HttpRunInstance`, `HttpResetInstance`, `HttpRollbackInstance`, `HttpStopInstance`

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
- **Store interface pattern**: `workflow` and `ia` accept a caller-provided `instances.Store` for persistence; `jrex` accepts its own `jrex.Store` (`SetModule`/`GetModule`/`DeleteModule`) for compiled-module persistence in `Production`/`Building` modes. In each case the library defines the interface and consumers implement it. Use `stores/` for the jsql-backed implementation of `instances.Store`.

## Required environment variables

| Package | Variable                                     | Purpose                                |
| ------- | -------------------------------------------- | -------------------------------------- |
| `jsql`  | `DB_DRIVER`                                  | Driver name (`postgres`, `sqlite`, …)  |
| `jsql`  | `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | Database connection       |
| `jsql`  | `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_CONN_LIFETIME`, `DB_POOL_CONN_IDLE_TIME` | Connection pool (optional) |
| `cache` | `REDIS_HOST`                                 | Redis connection                       |
| `event` | `NATS_HOST`                                  | NATS connection                        |
| `event` | `NATS_USER`, `NATS_PASSWORD`                 | NATS auth (optional)                   |
| `graph` | `NEO4J_HOST`, `NEO4J_USER`, `NEO4J_PASSWORD` | Neo4j connection                       |
| `ia`    | `OPENAI_API_KEY`                             | OpenAI agent integration               |
| `dt`    | `PRODUCTION`                                 | `true` = use Redis, `false` = use file |
| `wsp`   | `WHATSAPP_API_URL`                           | WhatsApp Graph API base URL (optional) |
