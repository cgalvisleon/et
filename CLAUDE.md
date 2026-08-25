# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Module

`github.com/cgalvisleon/et` — Go 1.25, MIT license.

> **Note:** This codebase changes fast and in-place (commit history is mostly bare "Backup:" commits with no descriptive messages, saved as periodic WIP checkpoints) — prefer reading the actual source over trusting prior assumptions about an API shape, including the rest of this file. Because commits can land mid-refactor, `go build ./...` can intermittently fail on a freshly-cloned `HEAD` (e.g. a half-renamed function or a stray syntax error in one file); if it does, check whether the specific files involved look incomplete before assuming the wider architecture described here is wrong. `LIBRARY_CONTEXT.md`, `ARCHITECTURE_SUMMARY.md`, `COMPONENT_CATALOG.md`, `AI_USAGE_GUIDE.md` (all Spanish, generated for external consumers of the library) and `README.es.md`/`README.md` can also drift out of sync with the code; verify before relying on them. (`README.md` in particular currently still documents several removed packages by their old names — `ws/`, `ia/`, `workflow/`, `tcp/`, `wsp/`, `instances/` — and stale APIs for `crontab`/`jrex`; don't trust its package table or code samples without checking the source.)

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
go run ./cmd/jrex            # JS runtime (jrex) with hot-reload from ./cmd/jrex/src/
go run ./cmd/jsql            # jsql driver test/demo

# Build all binaries
go build ./...

# Semantic versioning (reads git tags, updates README.md, pushes new tag)
./version.sh --major | --minor | --request
```

> **Note:** Almost the entire repo has no `*_test.go` files — `go test ./...` mostly compiles and finds nothing to run. The exceptions are `queue/example_test.go` (a runnable `Example_queue`) and `ia/*_test.go` (unit + an sqlite-backed integration test); don't assume either generalizes to other packages.

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

`jsql/` is a database-agnostic SQL builder and lightweight ORM. Entry points: `jsql.Load() (*DB, error)` (connects to the default DB) and `jsql.LoadTo(name string) (*DB, error)` (connects to a named DB) — **neither takes a `tenantId` parameter anymore**; the tenant id is read internally via `envar.GetStr("DB_TENANT_ID", "tenant:root")` (`jsql/jsql.go`). Connection settings (`DB_DRIVER`, `DB_HOST`, etc.) are likewise read internally via `envar.GetStr`/`GetInt`/`GetBool` directly — there is no `config` package anymore (it was deleted; see the `envar/` note below). `jsql.DriverPostgres`, `DriverSqlite`, `DriverMysql`, `DriverMssql`, `DriverOracle`, `DriverJosefina` are declared as driver-name constants (`jsql/driver.go`). **`postgres` and `sqlite` both have working implementations** under `jsql/drivers/postgres/` and `jsql/drivers/sqlite/` (the latter is a newer addition — pure-Go `modernc.org/sqlite`, self-registers under `jsql.DriverSqlite` via `init()`; an earlier version of this doc said the directory didn't exist at all, which is no longer true and had no prior usage anywhere in the repo until `ia/ia_test.go` started exercising it). `mysql`/`mssql`/`oracle`/`josefina` constants still exist with no backing implementation — none of `jsql/drivers/mysql/`, `mssql/`, `oracle/`, `josefina/` exist as directories at all (previously they existed as empty stub directories; those were removed too). One caveat found while wiring `sqlite` up for tests: `Model.Query`'s SQLite path composes each result row as a single nested JSON object (unlike postgres, which returns real per-column values), and `et.Json.ScanRows` (`et/json.go`) only auto-decoded `[]byte` JSON values, not `string` — the modernc.org/sqlite driver returns `string` for TEXT columns, so query results silently came back as `{"result": "<json text>"}` instead of a flattened row. `ScanRows` now also decodes `string` values that look like a JSON object/array (`{`/`[` prefix guard, to avoid misinterpreting an ordinary text value like `"123"` or `"true"`); a *nested* JSON-typed column inside that row-object (e.g. a hand-rolled JSON column holding an array) still comes back double-encoded as a JSON string one level down — `ia/similarity.go`'s `chunkEmbedding` works around that locally rather than deeper surgery on `jsql/drivers/sqlite/query.go`.

**Model definition:**

```go
// Full-featured model (adds id, created_at, updated_at, _source JSONB, _idx VARCHAR(80)):
model, _ := db.DefineModel("public", "users", 1, userId)

// Manual model (add every column yourself):
model := db.NewModel("public", "users", 1, userId)
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

`IdxField` (`_idx`) is `VARCHAR(80)` (`KEY` type); its value is a `reg.UUID()` set by an auto-registered `BeforeInsert` trigger — **not** a database serial/sequence.

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
- **`ettp/v2/`** — Full-featured HTTP server. `ettp.New(name string, cnf *Config) (*Server, error)` — `Config` has `Port`, `RpcPort`, `Parent`, timeouts, `AllowOrigin`, TLS fields (`IsTLS`, `CertFile`, `KeyFile`), `Transport *TransportConfig`, `Debug bool`. **There is no `UseCache`/`UseEvent` flag** — `New` never calls `cache.Load()`/`event.Load()` itself; `cache` is instead consulted opportunistically elsewhere (guarded by `cache.IsLoad()` in `ettp/v2/storage.go`), so callers must call `cache.Load()`/`event.Load()` themselves beforehand if they need those features. `RpcPort` is caller-supplied — a zero-value `Config{}` will make the internal RPC `net.Listen` bind to an OS-assigned port instead of a fixed default. Router state is synchronized across instances via NATS events (`EVENT_SET_ROUTER`, `EVENT_REMOVE_ROUTER`, `EVENT_RESET_ROUTER`). The `m.Myself` flag prevents self-processing. `ettp/v1/` is the older version; it is not simply a subset of `v2` (v1 has ~19 files/3300+ lines including `server-apigateway.go`/`server-proxy.go`/`server-token.go`/`server-method.go`/`server-cache.go` with no `v2` equivalent) and both versions still receive commits, so don't assume `v1` is dead code — check which one a given `cmd/` binary actually imports before assuming "prefer v2" applies.
- **`router/`** — Standalone router package (used internally by `ettp/v2`). Can be imported directly for custom HTTP routing without the full server setup.

### Infrastructure packages (require external services)

- **`cache/`** — Redis client (requires `REDIS_HOST`, optionally `REDIS_PASSWORD`, `REDIS_DB`). `cache.Load()` initializes; provides `Set`, `Get`, `Delete`, `Pub`, `Sub`.
- **`event/`** — NATS pub/sub (requires `NATS_HOST`, optionally `NATS_USER`, `NATS_PASSWORD`). `event.Load()` initializes; provides `Subscribe`, `Publish`, `Stack`.
- **`jrpc/`** — Go `net/rpc` over TCP (not NATS). `jrpc.Mount(host, port, services, packageName) (*Package, error)` registers a service under a simple `Solver{Host, Port, Inputs, Output}` registry (`jrpc/package.go`) — no load balancing or Raft logic lives here. `balancer.go` and `raft.go` are in `jtcp/`, not `jrpc/`; an older note attributing them to `jrpc` (also still present in `README.md`) was wrong.

### Self-contained utility packages

- **`envar/`** — Low-level env var access; `envar.GetStr/GetInt/GetInt64/GetFloat/GetBool` read env vars with defaults, `envar.Get`/`Set` (package-level, backed by an optional pluggable `_store`), `envar.ArgStr/ArgInt/ArgBool/...` read CLI args, `envar.Validate([]string{...})` checks required vars exist. **The old top-level `config/` package (package-level `GetStr`/`GetInt`/etc. plus a separate store-backed `config.Config` settings object) has been deleted entirely** — its env-getter half moved into `envar/` as shown here; its store-backed settings-record half was replaced by `stores.Config` (`stores/config.go`, `stores.DefineConfig(db, tenantId, schema, stage, tag) (*Config, error)`, jsql-backed, fields `TenantId`/`Stage`/`Tag` only — no `ID`/`OwnerId`/`Params`/`AuditLog`). The only leftover references to `github.com/cgalvisleon/et/config` are stale import strings inside `create/template/templates.go` (code-scaffolding templates) — there is no such importable package in this repo anymore.
- **`logs/`** — Structured logging. Functions: `Log`, `Info`, `Infof`, `Alert`, `Alertf`, `Error`, `Errorf`, `Debug`, `Debugf`, `Fatal`, `Panic`, `Tracer`. All route through `stdrout` for colorized output.
- **`jwt/`** — High-level token creation: `New`, `NewAuthentication`, `NewAuthorization`, `NewAppToken`. Stores tokens in `cache`. Built on top of `claim/`.
- **`claim/`** — JWT claims struct with `tenantId`. `GenToken` signs with HS256 (`golang-jwt/jwt/v4`).
- **`crontab/`** — Job scheduler, recently redesigned to be event-driven. `crontab.New(tag string, store Store) (*Crontab, error)` builds an instance (calls `event.Load()` only); `crontab.Load(tag string, store Store) error` is the usual entry point — it builds a `*Crontab` via `New` and stores it in a **package-level singleton**, then wires `event.Subscribe`/`event.Stack` listeners (`crontab/event.go`). The old instance methods `AddJob`/`AddOneShotJob`/`AddEventJob`/`StartJob`/`StopJob`/`DeleteJob` **no longer exist** — register jobs with the package-level `crontab.CronJob(tag, ownerId string, spec Cron, repetitions int, params et.Json, fn func(et.Json) error) error` (recurring, spec is now a structured `Cron{DayOfWeek, Month, DayOfMonth, Hour, Minute}`, not a raw cron string) or `crontab.ScheduleJob(tag, ownerId string, spec time.Time, params et.Json, fn func(et.Json) error) error` (one-shot); both fail if `Load` wasn't called first. `crontab.HttpRemoveJob`/`HttpStopJob`/`HttpStartJob` are HTTP handlers that publish control events rather than mutating a job directly. Nothing in the repo currently imports `crontab`.
- **`jval/`** — Fluent validation rules for `et.Json`. Implements `Rule` interface with typed validators (`Str`, `Int`, `Float`, `Bool`, `Email`, `Phone`, `Time`, etc.); chainable constraints (`.NotEmpty()`, `.Min()`, `.Max()`, etc.).
- **`validator/`** — A second, separate fluent validation package (`Validator`/`Field`/`Condition`, `.NotEmpty()`/`.Min()`/`.Max()`/etc.), with its own i18n'd (`LANG` env var) message constants in `validator/msg.go`. Overlaps in purpose with `jval/` but is a distinct, unrelated type — don't assume they share types or can be mixed.
- **`msg/`** — Root-level package of shared error-message string constants (`MSG_ATRIB_REQUIRED`, `MSG_RECORD_NOT_FOUND`, `MSG_TOKEN_INVALID`, etc.), separate from the per-package local `msg.go` files described under Key Patterns below.
- **`queue/`** — Generic in-memory batching queue. `queue.New[T any](queueSize, maxEvents int, period time.Duration, handler func(context.Context, []T) error) *Queue[T]`; `.Push(item)` buffers items and flushes a batch to `handler` when either `maxEvents` is reached or `period` elapses, whichever first; `.Close()` flushes and stops. Has the repo's only real test (`queue/example_test.go`, a runnable `Example_queue`) — the "no `*_test.go` files" note above no longer applies to this package.
- **`request/`** — Both inbound helpers (`URLParam`, `GetBody`) and outbound HTTP client utilities. `URLParam(r, "key").Str()` reads chi route params; `GetBody(r)` parses the JSON body into `et.Json`.
- **`strs/`** — String utilities.
- **`xls/`** — Excel read/write via `excelize/v2`. `xls.ReadXls(data []byte)` / `ReadXlsMultipart(multipart.File)` / `ReadXlsFile(path)` return an `*XlsReader`; `.GetSheet(nameSheet, columns)` reads rows into `[]et.Json` keyed by header. `xls.NewXls(data []et.Json, nameSheet string, columns []xls.Column)` builds an `*Xls` workbook (`.Add(...)` appends further sheets — omitting `columns` derives them from the union of keys in `data`); `.ToFile(path)` / `.ToWriter(w)` / `.ToHttp(w, filename)` export it.
- **`csv/`** — CSV read/write, mirroring `xls/`'s API shape. `csv.ReadCsv(data []byte, comma rune)` / `ReadCsvMultipart(multipart.File, comma)` / `ReadCsvFile(path, comma)` return a `*CsvReader` (pass `comma = 0` for the default `,`). `csv.NewCsv(data []et.Json, columns ...csv.Column)` builds a `*Csv` document (omitting `columns` derives them from the sorted union of keys in `data`, using the key as title); export via its writer methods (`ToFile`/`ToWriter`/`ToHttp`, same pattern as `xls`).
- **`utility/`** — Crypto, validation, ID generation (UUID, Snowflake, ULID), general helpers.
- **`middleware/`** — HTTP middleware (CORS, request ID, logger, auth, telemetry, panic recovery).
- **`response/`** — Unified HTTP response helpers. Key functions: `ITEM(w, r, status, et.Item{})`, `ITEMS(w, r, status, items)`, `HTTPError(w, r, status, message)`.
- **`jws/`** — WebSocket support via `gorilla/websocket` (formerly named `ws`).
- **`service/`** — OTP helpers (`SendOTPEmail`, `SendOTPSms`, `VerifyOTP`) and messaging integration; uses `tenantId`.
- **`stdrout/`** — Low-level colorized stdout routing used by `logs/`.

### Integration packages

- **`aws/`** — AWS SDK wrapper: S3, SES (email), SMS.
- **`brevo/`** — Brevo API client: email, SMS, WhatsApp.
- **`jwsp/`** — WhatsApp Business API client (formerly named `wsp`). `jwsp.NewSender(token, phoneNumberId) *Whatsapp` produces a message/webhook builder (`Debug()`, `IsTest()`, `SetVerifyToken()`, `SetEventHandler()`); uses Facebook Graph API (configurable via `WHATSAPP_API_URL`).

### Application-layer packages

- **`jrex/`** — JavaScript runtime package (`dop251/goja`), formerly named `vm`. Entry point: `jrex.Load(tag string, store Store) (*Jrex, error)` — loads-or-creates a persisted `Jrex` (looked up under collection `"jrex"`); pass `store = nil` to default to a `FileStore` rooted at `./src` (`jrex.NewStore(baseDir)`). `jrex.Store` is a 2-method interface (`Set(collection, id, ownerId string, obj any) error`, `Get(collection, id string, dest any) (bool, error)`) — there are no `Develop`/`Production`/`Building` modes or `SetModule`/`GetModule`/`DeleteModule` methods in the current code (older docs describing those are stale). `(*Jrex).Set(name, value)` injects a Go binding into every new `Instance`; `(*Jrex).Run()`/`RunModule(tag)` executes the `"index"` module (or another by tag) in a fresh `goja` `Instance`; `(*Jrex).RunDev()` runs once and blocks via `utility.AppWait()` — with the default `FileStore`, file changes under `BaseDir` also trigger a re-run via `file.NewWatcher` (hot reload). Global wrappers still provide `console.*`, `ctx.*`, `fetch()`, and CommonJS-style `require()` (`jrex/wrapper.go`, `jrex/lib.go`). There is also a `jcli/` package at the repo root (not under `jrex/`) holding a Bubble Tea CLI model (`cliModel`, `App` interface) — its file still declares `package jrex` and nothing imports `github.com/cgalvisleon/et/jcli`, so treat it as in-progress/orphaned work extracting the dev CLI out of `jrex/`, not a wired feature. (Similarly, `infobip/infobip.go` at the repo root is currently just a bare `package infobip` declaration with no code — an empty stub, not yet a real integration.)
- **`jia/`** — OpenAI agent integration (`openai-go/v3`), formerly named `ia`. `jia.New(tag string, store Store, userId string) (*Ia, error)` generates a fresh `reg.UUID()` as `Ia.ID`; `jia.Load(id string, store Store) (*Ia, error)` loads an existing `Ia` *by its own ID* via `store.Get`. Neither takes a `tenantId` — `Ia` has no `TenantId` field, it is no longer tenant-scoped (same shift as `jwf`, below). Calls `event.Load()` internally (no `cache.Load()`). `Store` is a locally-defined interface in `jia/ia.go` (`Set(collection, id, ownerId string, obj any) error`, `Get(collection, id string, dest any) (bool, error)`, `Delete(collection, id string) error`, `Query(query et.Json) (et.Items, error)`) — caller-provided, no shared package defines it. `OPENAI_API_KEY` is read directly via `envar.GetStr` (the `config` package it might once have used no longer exists at all). Manages `Agent`s, `Participant`s, and `Conversation`s (with `Message`s); `Skill` (e.g. `ApiSkill`) lets agents call external APIs. Also has `jia/router.go` (HTTP routes for agents/conversations/participants), `jia/sender.go` (`Sender` interface used by `Conversation.SendTextMessage`, with no exported setter currently — effectively unwired), and `jia/event.go`/`jia/msg.go` (event-name and i18n message constants). Unlike `jwf`/`jrex`, there is no `cmd/jia` runnable example. **Caveat:** `jia/ia.go:90`'s `Load` calls `store.Get(id, packageName, &result)` — args swapped vs. the interface's `Get(collection, id, dest)` convention. (A previously-documented sibling bug — `Agent.save()` in `agent.go` passing the wrong collection name `"step"` instead of `"agent"` — has since been fixed; every `jia/*.go` file now keys its own `store.Set`/`Get` calls off a local `storeXxx` collection-name constant, e.g. `storeAgents`, `storeConversations`, `storeMessages`, `storeParticipants`, `storeIa`.)
- **`jwf/`** — Workflow orchestration (replaces the old `workflow/` package, which was deleted; a separate `wf/` scratch package was also deleted). Calls `cache.Load()` + `event.Load()` internally. See detail below.
- **`ia/`** — Multitenant RAG (retrieval-augmented generation), not to be confused with `jia/` above (whose old name was also `ia`) — the two are unrelated, separate implementations. `ia.Load(db *jsql.DB, schema string, cnf Config) (*Rag, error)` defines/initializes four jsql models under `schema` (`ia_documents`, `ia_chunks`, `ia_conversations`, `ia_messages`), every one carrying both `tenant_id` and `project_id` indexed columns (via a local `defineTenantProjectModel` helper — `jsql.DefineTenantModel`/`DefineProjectModel` each only add one of the two). `Config` (embedding/chat model names, chunk size/overlap, top-K, `ApiKey`) is filled from env vars (`IA_EMBEDDING_MODEL`, `IA_CHAT_MODEL`, `IA_CHUNK_SIZE`, `IA_CHUNK_OVERLAP`, `IA_TOP_K`, `OPENAI_API_KEY`) by `defaultConfig` for any zero-valued field. `(*Rag).IngestFile(ctx, tenantId, projectId, name string, data []byte, userId string)` infers the source from the filename extension (pdf/docx/xlsx/csv/txt/md) and extracts text via a format-specific loader (`ia/loader_*.go`: `csv/` package reuse, `excelize/v2` directly for xlsx, a hand-rolled `archive/zip`+`encoding/xml` extractor for docx, `github.com/ledongthuc/pdf` for pdf — a dependency added specifically for this package); `(*Rag).IngestSQL(ctx, tenantId, projectId, name, query string, sourceDB *jsql.DB, userId string)` ingests a SQL query's result rows instead (via `db.Sql(query)`; `sourceDB = nil` defaults to the Rag's own DB). Both funnel into a shared `ingest` that chunks the text (`chunkText`, word-based, `Config.ChunkSize`/`ChunkOverlap`), embeds every chunk, and stores them. **Vector search is not backed by pgvector** — `jsql.EMBEDDING`/`VECTOR` (see the `jsql/` section above) has no value marshalling or similarity operator wired anywhere in the repo, so embeddings are stored as a plain `JSON` column (`[]float64`) and `(*Rag).Ask(ctx, tenantId, projectId, conversationId, userId, question string)` does the similarity search itself: fetch every chunk for the tenant/project (bounded by `DB_RECORD_LIMIT`, default 1000) and rank by cosine similarity in Go (`ia/similarity.go`) — a documented scaling limitation, not a bug. `Ask` embeds the question, keeps the top `Config.TopK` chunks as context, asks the chat model to answer using only that context (`ia/client.go`, classic `Chat.Completions.New`, not the `Responses`/`Conversations` API `jia/agent.go` uses — conversation history here lives in `ia_conversations`/`ia_messages`, not OpenAI-side state), and persists both the question and answer as messages (creating a conversation first if `conversationId` is empty). `Rag.embedFn`/`Rag.answerFn` are swappable function fields (defaulted to the real OpenAI-backed methods in `Load`) that exist specifically so tests can stub out network calls — see `ia/ia_test.go`, which exercises the full ingest → ask → multitenant-isolation path against the `sqlite` driver with fake embeddings (nothing in this package has been tested against live OpenAI). `(*Rag).LoadRouter(mux chi.Router)` wires `POST/GET /documents`, `POST /documents/sql`, `DELETE /documents/{id}`, `POST /conversation`, `GET /conversation/{id}`; `cmd/ia` boots it on the plain `server/` package (not `ettp/v2`) per the feature's own spec — see `featureIa.md` for the full design write-up and decision log.
- **`graph/`** — Neo4j connectivity (`neo4j-go-driver/v5`). `graph.Load()` returns a `*Conn` with the Neo4j driver.
- **`stores/`** — jsql-backed persistence helpers: `stores.DefineInstance`/`LoadInstance`/`DefineInstanceBite`/`LoadInstanceBite` (kind: `KindJson` or `KindBite`), `stores.DefineAuthorization` (tenant/profile/method/path ACL checks, cached through `dt`), and `stores.DefineConfig` (per-tenant settings record — see the `envar/` note above). **The old generic `stores.DefineCatalog`/`Catalog` (`kind`+`id` key-value table) was moved out of `stores/` entirely** — it now lives as `jsql.DefineStore(db, schema) (*jsql.Store, error)` (`jsql/store.go`, table renamed `db_catalogs`). **Caveat:** `(*stores.Instance).Get(id string, dest any) (bool, error)` only takes one string key, so it still does _not_ structurally satisfy the two-string-key `Store.Get` interfaces defined locally in `jia/`/`jwf/` — nothing in the repo currently wires `stores/` into `jia` or `jwf` (both are exercised with a `nil` store in their `cmd/` examples). By contrast, `jsql.Store` now has `Set(collection, id, ownerId string, obj any) error` / `Get(collection, id string, dest any) (bool, error)` / `Delete(collection, id string) error` / `Query(query et.Json) (et.Items, error)` — this **does** structurally satisfy `jia.Store` exactly, but not `jwf.Store` (whose `Query` additionally takes a leading `collection string` param). Check signatures before assuming any of these is a drop-in for another package's `Store`.
- **`dt/`** — Cache-backed object store. `dt.Up(key, data)` writes an object (uses Redis in production, reads from file in dev based on `PRODUCTION` env var); `dt.Get(key)` retrieves it, `dt.Drop(key)` removes it. HTTP handler support via `handler.go`.
- **`resilience/`** — Resilience/retry pattern. `resilience.New(store)` (store: local `Store` interface — `Set`/`Get`/`Delete`/`Query`), `LoadInstance(Params{Id, Tag, Description, TotalAttempts, Interval, Tags, Fn, FnArgs})` returns an `*Instance` (note: `Params` has no `TenantId`/`OwnerId`/`UserId` fields), then `instance.Run(userId)` executes `Fn` with retry/interval semantics. Used by `jwf` for step-level resilience; see `cmd/resilience` for a standalone example.
- **`reg/`** — Service registration/discovery; provides ID generation helpers (ULID, etc.) used by `claim` and others.
- **`file/`** — File operations and watching (`FileInfo`, `Watcher`, `ExistPath()`); used by `jrex` for hot-reload.
- **`mem/`** — Shared memory and sync primitives.
- **`ephemeral/`** — Ephemeral/temporary data structures.
- **`iterate/`** — Iteration control with time support.
- **`race/`** — Race condition detection helpers.
- **`cmds/`** — Command/stage execution system (distinct from the `cmd/` CLI binaries).
- **`timezone/`**, **`units/`**, **`color/`** — Timezone handling, unit conversions, terminal color utilities.

### TCP cluster: `jtcp/`

`jtcp/` (formerly named `tcp`) implements a distributed TCP node with Raft-style leader election. Modes: `Follower`, `Candidate`, `Leader`, `Proxy`. `jtcp.NewNode(port)` is used by `cmd/server`.

### CLI (`cmd/`)

Each subdirectory under `cmd/` is a standalone binary:

- `cmd/et/` — Main CLI using `cobra`
- `cmd/apigateway/` — API Gateway/proxy using `ettp.New`
- `cmd/daemon/` — Background service with systemd integration (start/stop/restart/status/conf/version)
- `cmd/create/` — Project/code scaffolding
- `cmd/server/` — TCP node server (`jtcp.NewNode(port)`)
- `cmd/jrex/` — JavaScript runtime example; connects a `jsql.DB`, then `jrex.Load("jrex", nil)`, binds `db`/`getDb`/the model via `v.Set(...)`, and runs `v.RunDev()` (hot-reloads scripts under `cmd/jrex/src/`)
- `cmd/jsql/` — jsql driver demo: DDL generation, condition building, SELECT field resolution, live DB connection
- `cmd/client/` — Test client (`jtcp`)
- `cmd/install/` — Installation utility
- `cmd/whatcher/` — Filesystem change watcher
- `cmd/jwf/` — `jwf/` (workflow) usage example: builds a flow with `NewFloW`/`Step`, runs it with `wf.Run(...)`
- `cmd/resilience/` — `resilience/` usage example: `resilience.New(nil)`, `LoadInstance(Params{...})`, `ins.Run(userId)`
- `cmd/wsp/` — `jwsp/` (WhatsApp) usage example
- `cmd/ia/` — `ia/` (RAG) HTTP API: connects `jsql.Load()` (postgres), `ia.Load(db, schema, ia.Config{})`, serves `rag.LoadRouter` on `server.New("ia", port)` (`PORT` env, default 3300)

### Code generation (`create/`)

Templates and generators for new microservices, projects, and Kubernetes deployments. Used by the `cmd/create` CLI.

### `jwf/` package detail

> **Note:** This package replaces what used to be two separate packages: a top-level `workflow/` package and a `wf/` scratch/rewrite area — both were deleted (last seen 2026-06-16 and 2026-06-18 respectively) and consolidated into `jwf/`, which has a different API from either predecessor. References to `workflow.RunInstance`, `instances.Store`, etc. elsewhere (old branches, comments) describe the removed package, not `jwf/`.

`jwf.New(store Store, userID string) (*WorkFlow, error)` (`jwf/workflow.go`) calls `cache.Load()` + `event.Load()` internally, assigns a fresh `reg.UUID()` as the new `WorkFlow.ID`, records a `"new_workflow"` audit entry for `userID`, and persists via `Save()` — it takes a `userID` for auditing, not a `tenantId`. `jwf.Load(store Store, id string) (*WorkFlow, error)` is the store-backed variant that loads an existing `WorkFlow` *by its own ID* from `store.Get("workflows", id, &def)` (errors if `store` is `nil`), rehydrating its flows/steps individually.

**Type hierarchy** (graph-based, not a linear step list):

```
WorkFlow (container, identified by its own ID — not tenant-scoped)
  -- Steps map[string]*Step        (pool of steps shared across the workflow)
  -- Flows map[string]*Flow
        -- Steps map[string]*Step       (this flow's subset of the step pool, including one or more Triggers)
        -- Connections []*Connection    (Source/Target *StepConnection{StepId, Port, Index} + Kind: input/output/error)
        -- Triggers []*Trigger          (Tag -> StartId, the starting Step.ID)
```

**There is no `WorkFlow.Instances` map** — instances are not held in memory as a registry. `Run` builds one on demand (`newInstance`) or loads it via `store.Get("instances", id, ...)`; a lightweight cache-backed marker (`instance:<id>:status`, TTL = `Flow.TimeAwait`) is used only to detect an already-running instance, not to hold full state.

A `Flow` is built fluently: `flow.Step(tag, title, fn)` adds the first step as a `KindTrigger` (registering a `Trigger`) or chains an action step via an output `Connection` to the previously-added step; `flow.Error(tag, version, title, fn)` attaches an error-port step to the most recently added step (`Connection` with `Kind: PortError`), or sets a retrievable build error (`flow.IsError()`) if called before any step exists. `fn` has signature `func(instance *jwf.Instance, ctx et.Json) (et.Json, error)`; `Step.Definition` also accepts a JS string/`[]byte` body, executed via an embedded `jrex.Instance` instead of a Go closure.

```go
wf, _ := jwf.New(nil, userId) // nil store: in-memory only, nothing persisted
flow := wf.NewFloW("add", "add item", "1.0.0", userId). // note: "FloW", not "Flow" (still not fixed)
    Step("add", "add item", func(instance *jwf.Instance, ctx et.Json) (et.Json, error) {
        instance.SetParams(et.Json{"step1": "step1"})
        return et.Json{"step1": "step1"}, nil
    }).
    Step("add", "add item", func(instance *jwf.Instance, ctx et.Json) (et.Json, error) {
        return et.Json{"step2": "step2"}, nil
    })

result, err := wf.Run(flow.ID, "add", "" /* instance id, blank = new */, projectId, code, et.Json{}, et.Json{}, userId)
```

> **Note:** `Instance.Params et.Json` and `(*Instance).SetParams(params et.Json) et.Json` both still exist (`jwf/instance.go`) — an earlier version of this doc claimed they'd been removed and that `cmd/jwf/main.go` failed to build as a result; that is no longer true, `go vet ./jwf/... ./cmd/jwf/...` is currently clean. Treat any claim of a specific field/method being "recently removed" in this package with suspicion and grep first — this package churns quickly.
>
> Minor quirk worth knowing: in `Flow.Step`/`Flow.Error`, the `userId` argument passed down into `newStep` is actually `s.ID` (the flow's own ID), not a real user id — so the audit log on those steps records the flow ID as "user". Doesn't affect compilation or the fluent chain's behavior.

`Instance` tracks `Status` (`CREATED`, `PENDING`, `RUNNING`, `ROLLBACK`, `DONE`, `FAILED`, `CANCEL`, `STOP`), advances step-to-step via `next()` following `Connections`, and on a step error calls `resilience.New(workflow.store)` + `LoadInstance(resilience.Params{..., Fn: instance.run, ...})` (when `Flow.TotalAttempts != 0`) before falling back to the error-branch connection.

**`Store` interface** (`jwf/store.go`): `Set(collection, id, ownerId string, obj any) error`, `Get(collection, id string, dest any) (bool, error)`, `Delete(collection, id string) error`, `Query(collection string, query et.Json) (et.Items, error)`, plus five series-generator methods mirroring `jsql.Series`'s own API — `SetSeries(tag, format string, value int) error`, `GetSeries(tag string) (et.Item, error)`, `DeleteSeries(tag string) error`, `GenSerie(tag string) (string, error)`, `GenValue(tag string) (int, error)` (an earlier version of this doc said `Store` had no `GenSerie` method; that changed — a custom `Store` implementation must now provide all five or it no longer satisfies the interface). A default jsql-backed implementation ships as `jwf.Storage`, built via `jwf.DefineStore(db *jsql.DB, schema string) (*Storage, error)` (`jwf/store.go`), managing `"workflows"`/`"flows"`/`"steps"`/`"instances"` jsql models plus a `jsql.Series` (`jwf/store.go`'s own `storeWorkflows`/`storeFlows`/`storeSteps`/`storeInstances` constants back the model-name strings) — it delegates the five series methods straight through to `jsql.DefineSeries(db, schema)`. `newInstance` (`jwf/instance.go`) uses this to auto-generate an instance's `code` via `store.GenSerie(tag+":"+projectId)` when the caller passes `code == ""`. Relatedly, `StoreDefine` (`jwf/store.go`) dropped its `tenant_id` column/index — workflow/flow/step/instance storage rows are no longer tenant-scoped at all, consistent with `WorkFlow` itself having no `TenantId` field.

**HTTP routing:** `(*WorkFlow).LoadRouter(r Router)` wires 8 routes (`jwf/router.go`). Only the **Steps** sub-API is actually implemented: `httpGetStep`, `httpSetStep`, `httpUpdateStep`, `httpDeleteStep`. The **Flows** (`httpGetFlow`, `httpSetFlow`, `httpStatusFlow`, `httpDeleteFlow`) and **Instances** (`httpGetInstance`, `httpDeleteInstance`, `httpRunInstance`) handlers are registered but still have **empty function bodies** — not yet implemented. `Router` is a minimal local interface (`Protect(method, path string, handler func(http.ResponseWriter, *http.Request))`), unrelated to the repo's `router/` package.

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
- **Store interface pattern**: `jwf` and `jia` each accept a caller-provided `Store` for persistence — separately-defined, structurally near-identical interfaces local to each package (there is no shared `instances` package anymore). The one difference: `jwf.Store.Query` takes a leading `collection string` param (`Query(collection string, query et.Json) (et.Items, error)`) that `jia.Store.Query` does not (`Query(query et.Json) (et.Items, error)`) — `jsql.Store` (see `stores/` above) happens to structurally satisfy `jia.Store` exactly but not `jwf.Store` for this reason. `jrex` accepts its own, narrower `jrex.Store` (`Set(collection, id, ownerId string, obj any) error` / `Get(collection, id string, dest any) (bool, error)`, no `Delete`); `resilience` defines its own local `Store` too. In every case the library defines the interface and consumers implement it — check the exact method signatures per package before reusing one implementation across packages (see the `stores/` caveat above).

## Required environment variables

| Package | Variable                                                                                  | Purpose                                |
| ------- | ----------------------------------------------------------------------------------------- | -------------------------------------- |
| `jsql`  | `DB_DRIVER`                                                                               | Driver name (`postgres`, `sqlite`, …)  |
| `jsql`  | `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`                                 | Database connection                    |
| `jsql`  | `DB_TENANT_ID` (default `tenant:root`)                                                    | Tenant id used by `Load`/`LoadTo`       |
| `jsql`  | `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_CONN_LIFETIME`, `DB_POOL_CONN_IDLE_TIME` | Connection pool (optional)             |
| `cache` | `REDIS_HOST`                                                                              | Redis connection                       |
| `event` | `NATS_HOST`                                                                               | NATS connection                        |
| `event` | `NATS_USER`, `NATS_PASSWORD`                                                              | NATS auth (optional)                   |
| `graph` | `NEO4J_HOST`, `NEO4J_USER`, `NEO4J_PASSWORD`                                              | Neo4j connection                       |
| `jia`   | `OPENAI_API_KEY`                                                                          | OpenAI agent integration               |
| `ia`    | `OPENAI_API_KEY`                                                                          | RAG embeddings + chat (required by `ia.Load`) |
| `ia`    | `IA_SCHEMA` (default `public`, read by `cmd/ia`), `IA_EMBEDDING_MODEL`, `IA_CHAT_MODEL`, `IA_CHUNK_SIZE`, `IA_CHUNK_OVERLAP`, `IA_TOP_K` | RAG config defaults (optional)         |
| `dt`    | `PRODUCTION`                                                                              | `true` = use Redis, `false` = use file |
| `jwsp`  | `WHATSAPP_API_URL`                                                                        | WhatsApp Graph API base URL (optional) |
| `claim` | `SECRET`                                                                                  | JWT signing key (default: `"1977"`)    |
