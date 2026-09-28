# ET: Go Services and Tools Library

> [Versión en español](README.es.md)

Modular Go library for microservices, CLIs and web applications. Every directory is an independent package — import only what you need; there is no central entry point.

- Module: `github.com/cgalvisleon/et` — Go 1.25 — MIT

## Installation

```bash
go get github.com/cgalvisleon/et@latest
go get github.com/cgalvisleon/et@v1.0.35
```

```go
import (
    "github.com/cgalvisleon/et/et"
    "github.com/cgalvisleon/et/jsql"
)
```

## Packages

### Core data

| Package | Description |
| --- | --- |
| `et/` | Core types: `Json` (`map[string]interface{}` with typed accessors), `List`, `Item`, `Items`, `Condition`, `Field`, `Aggregate` |
| `jsql/` | Database-agnostic SQL builder and ORM driven by JSON — see [SQL builder](#sql-builder-jsql) |
| `stores/` | jsql-backed stores: instances, authorization (ACL), per-tenant config, form state |
| `mem/`, `ephemeral/` | In-memory cache with expiration; short-lived data |
| `dt/` | Data store on Redis (`PRODUCTION=true`) or files |

### HTTP and RPC

| Package | Description |
| --- | --- |
| `server/` | Lightweight HTTP server on `chi` (no Redis/NATS) |
| `ettp/v2/` | Full HTTP server/API gateway: `ettp.New(name, *Config)`; router state synced across replicas over NATS. `ettp/v1/` is the older version (proxy, token, cache) |
| `router/` | Standalone router used by `ettp/v2` |
| `middleware/` | CORS, authentication, logger, request ID, metrics, panic recovery |
| `request/`, `response/` | Inbound helpers (`URLParam`, `GetBody`), outbound `Fetch`; unified responses (`ITEM`, `ITEMS`, `HTTPError`) |
| `jrpc/` | Go `net/rpc` over TCP: `jrpc.Mount(host, port, services, packageName)` |
| `jtcp/` | Distributed TCP node with Raft-style leader election, balancer and TLS |
| `jws/` | WebSocket hub |

### Infrastructure

| Package | Description |
| --- | --- |
| `cache/` | Redis client (`Set`, `Get`, `Delete`, collections, Pub/Sub) |
| `event/` | Pub/Sub over NATS (`Publish`, `Subscribe`, `Stack`) |
| `graph/` | Neo4j connectivity |
| `queue/` | Generic batching queue: `queue.New[T](size, maxEvents, period, handler)` |
| `crontab/` | Event-driven cron and one-shot jobs |
| `resilience/` | Retries with attempts and interval (used by `jwf`) |

### Security

| Package | Description |
| --- | --- |
| `jwt/` | Tokens (`NewToken`, `NewAuthentication`, `NewAuthorization`, `Validate`) stored in `cache` |
| `claim/` | JWT claims, HS256 signing with `SECRET` |
| `service/` | OTP by email/SMS |

### Scripting, workflows and AI

| Package | Description |
| --- | --- |
| `jrex/` | Embedded JavaScript runtime (goja) with hot-reload |
| `jwf/` | Workflow engine: flows, steps (Go or JS), instances, retries and error routes |
| `jia/` | OpenAI agents: agents, participants, conversations, skills |
| `ia/` | Multitenant RAG: ingest files/SQL, embeddings, `Ask` |

### Messaging

| Package | Description |
| --- | --- |
| `aws/` | S3, SES (email), SMS |
| `brevo/` | Email, SMS and WhatsApp (transactional and promotional) |
| `infobip/` | SMS, email and WhatsApp templates |
| `jwsp/` | WhatsApp Graph API |

### Utilities

| Package | Description |
| --- | --- |
| `envar/` | Environment variables with defaults (`GetStr`, `GetInt`, …) and CLI args |
| `logs/`, `stdrout/`, `color/` | Leveled, colorized logging |
| `validator/` | Fluent validation (`Required`, `Min`, `Max`, `Pattern`, …) |
| `reg/` | IDs: UUID, ULID, Snowflake, XID |
| `xls/`, `csv/` | Read and write spreadsheets / CSV (file, multipart, HTTP) |
| `file/` | File helpers and filesystem watcher |
| `strs/`, `utility/` | String helpers; crypto, hashing, misc |
| `timezone/`, `units/` | Time zones and dates; unit conversion |
| `iterate/`, `race/`, `cmds/` | Iteration, race helpers, command/stage execution |
| `msg/` | Shared message constants |
| `create/`, `cmd/` | Project scaffolding templates; CLI binaries |

## Core type: `et.Json`

`et.Json` (`map[string]interface{}`) is the lingua franca of the library, with typed accessors, default values and nested key traversal:

```go
data := et.Json{"user": et.Json{"name": "Ana", "age": 30}}
name := data.Str("user", "name")        // "Ana"
age  := data.Int("user", "age")         // 30
city := data.ValStr("Cali", "user", "city") // default when missing
data.Set("active", true)
```

`et.List` is the paginated result (`Rows`, `All`, `Count`, `Page`, `Start`, `End`, `Result []Json`); `et.Item` / `et.Items` wrap single and multiple results.

## Initialization pattern

Infrastructure packages read their configuration from the environment and connect once; later calls are no-ops:

```go
cache.Load() // REDIS_HOST
event.Load() // NATS_HOST
```

`ettp.New(name, config)` does **not** call them — the caller must.

## SQL builder: `jsql/`

SQL builder and ORM whose inputs are JSON structures. The public API is gathered in [`jsql/catalog.go`](jsql/catalog.go), grouped in 11 sections (connection, DDL, relations, queries, conditions, commands, triggers, transactions, series, audit, utilities). The behavior is specified in [`jsql/feature.md`](jsql/feature.md) and [`jsql/spec.md`](jsql/spec.md).

**Drivers** — imported for their side effect:

| Driver | Package | Notes |
| --- | --- | --- |
| PostgreSQL | `jsql/drivers/postgres` | JSONB `_source` |
| SQLite | `jsql/drivers/sqlite` | `modernc.org/sqlite`, no CGO |
| Oracle 19c+ | `jsql/drivers/oracle` | `go-ora`; JSON in CLOB, commands through PL/SQL blocks |
| MySQL 8.0+ | `jsql/drivers/mysql` | `go-sql-driver/mysql`; JSON column, commands as batches; no `FULL JOIN` |
| SQL Server 2022+ | `jsql/drivers/mssql` | `go-mssqldb`; JSON in `NVARCHAR(MAX)`, row JSON built by the driver |

**`SourceField`** — a model can have a JSON column (`_source`): any value without a declared column is stored there as an attribute, and can be queried, updated (merged, keeping the other keys) and returned like a column, including nested paths `a->b->c`.

```go
import _ "github.com/cgalvisleon/et/jsql/drivers/postgres"

db, _ := jsql.Load() // DB_DRIVER, DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME

// Model: id, status, created_at, updated_at, _source, _idx
users, _ := db.DefineModel("public", "users", 1, userId)
users.DefineColumn("name", et.TEXT, "")
users.DefineUnique("email", et.TEXT, "")
users.DefineAttrib("age", et.INT, 0) // stored in _source
users.DefineHidden("password")
users.Init() // creates the table if missing

// Commands return the affected rows (RETURNING)
users.Insert(et.Json{"id": "u1", "name": "Ana", "email": "ana@x.com", "age": 30, "city": "Cali"}).One()
users.Update(et.Json{"addr->zip": "76001"}).Where(jsql.Eq("id", "u1")).One()
users.Delete().Where(jsql.Eq("status", "archived")).Limit(0).Exec() // 0 = every matching row

// Fluent queries
items, _ := users.Select("id", "name", "city").
    Where(jsql.More("age", 18)).
    OrderBy("name", true).
    Limit(1, 20) // page, rows
```

**Queries in JSON** — on a model (`Model.Query`) or on the database (`DB.Query`), with SQL-like keys:

```go
items, _ := db.Query(et.Json{
    "select":    []any{"U.name", "count(O.id):orders"},
    "from":      "public.users:U",
    "left join": []any{et.Json{"to": "public.orders:O", "on": []any{et.Json{"O.user_id": et.Json{"eq": "U.id"}}}}},
    "where":     []any{et.Json{"U.age": et.Json{"more": 18}}},
    "group by":  []any{"U.name"},
    "having":    []any{et.Json{"count(O.id)": et.Json{"more": 1}}},
    "order by":  []any{et.Json{"U.name": true}},
    "limit":     20,
})
```

**Commands in JSON** — `insert`, `bulk`, `update`, `delete` and `upsert`, with JS triggers:

```go
db.Query(et.Json{"update": et.Json{
    "from":          "public.users",
    "data":          et.Json{"name": "Ana María"},
    "where":         []any{et.Json{"id": et.Json{"eq": "u1"}}},
    "limit":         100, // default: DB_RECORD_LIMIT, max 1000; 0 = all
    "before_update": []any{`NEW.updated_by = "api";`},
}})
```

**Other features**: joins (`inner`, `left`, `right`, `full`), aggregates and `having`; relations — details, masters (through a bridge model, 1-to-1 or 1-to-N) and rollups (`row`, `object`, `count`, `sum`, `avg`, `min`, `max`), resolved only when selected by name; calculated fields in Go (`DefineCalcFunc`) or JavaScript (`DefineCalc`); Go and JS triggers (`NEW` / `OLD`); unique fields validated with `ErrRecordAlreadyExists`; transactions (`NewTx`, `ExecTx`, `Commit`, `Rollback`); series (`DefineSeries`, `GenSerie`); `.Debug()` logs the SQL and `.Test()` generates it without running it.

**Tests** — [`jsql/drivers/test`](jsql/drivers/test) runs the whole catalog against the five drivers and writes the results (status, output and SQL per case) to `jsql/drivers/test/result/`:

```bash
cd jsql/drivers/test && go test -v
```

SQLite runs in a temporary file; PostgreSQL uses the `DB_*` variables of the repository `.env`; Oracle, MySQL and SQL Server use `ORACLE_*`, `MYSQL_*` and `MSSQL_*`, with defaults for local containers (`gvenzl/oracle-free`, `mysql:8.4`, `mcr.microsoft.com/mssql/server:2022-latest`). A database that is not available is skipped.

## Workflows: `jwf/`

```go
wf, _ := jwf.New(db, "", userId) // db *jsql.DB; also calls cache.Load() and event.Load()
flow := wf.NewFloW("add", "add item", "1.0.0", userId).
    Step("add", "step 1", func(instance *jwf.Instance, ctx et.Json) (et.Json, error) { return et.Json{}, nil }).
    Step("add", "step 2", fn)
result, err := wf.Run(flow.ID, "add", "", projectId, "", et.Json{}, et.Json{}, true, userId)
```

A step can also be a JavaScript body run through `jrex`. Failed steps retry through `resilience` and then follow the error connection.

## Crontab

```go
crontab.Load("my-service", store) // store implements crontab.Store

crontab.CronJob("my-service", ownerId, crontab.Cron{Hour: "*", Minute: "0"}, 0, et.Json{"msg": "hello"},
    func(params et.Json) error { logs.Info(params.ToString()); return nil })

crontab.ScheduleJob("my-service", ownerId, time.Now().Add(time.Hour), et.Json{}, fn)
```

Jobs are event-driven; the HTTP handlers (`HttpStartJob`, `HttpStopJob`, `HttpRemoveJob`) publish control events.

## JavaScript runtime: `jrex/`

```go
rt, _ := jrex.Load("my-jrex", store) // nil store → files in ./src
rt.Set("db", db)                      // inject Go bindings
rt.Run()                              // or rt.RunDev() for hot-reload
```

Scripts get `console`, `ctx`, `fetch` and `require`.

## AI

```go
agent, _ := jia.New("support", store, userId) // OpenAI agents with conversations and skills

rag, _ := ia.Load(db, "public", ia.Config{}) // RAG over jsql (tenant and project scoped)
rag.IngestFile(ctx, tenantId, projectId, "manual.pdf", data, userId)
answer, _ := rag.Ask(ctx, tenantId, projectId, conversationId, userId, "¿Cómo reinicio el router?")
```

Both require `OPENAI_API_KEY`.

## Logging

```go
logs.Info("message")
logs.Errorf("failed: %v", err)
logs.Fatal(err) // calls os.Exit(1)
```

## Environment variables

| Package | Variables | Purpose |
| --- | --- | --- |
| `jsql` | `DB_DRIVER` | `postgres`, `sqlite`, `oracle`, `mysql` or `mssql` |
| `jsql` | `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | Connection (`DB_NAME` is the service name in Oracle) |
| `jsql` | `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_CONN_LIFETIME`, `DB_POOL_CONN_IDLE_TIME` | Connection pool |
| `jsql` | `DB_RECORD_LIMIT` | Rows per query and default rows per update/delete (1000) |
| `jsql` | `DB_SSL`, `DB_SSL_VERIFY` | TLS (Oracle) |
| `cache` | `REDIS_HOST`, `REDIS_PASSWORD`, `REDIS_DB` | Redis |
| `event` | `NATS_HOST`, `NATS_USER`, `NATS_PASSWORD` | NATS |
| `graph` | `NEO4J_HOST`, `NEO4J_USER`, `NEO4J_PASSWORD` | Neo4j |
| `claim` | `SECRET` | JWT signing key (default `"1977"`) |
| `jia`, `ia` | `OPENAI_API_KEY` | OpenAI |
| `ia` | `IA_EMBEDDING_MODEL`, `IA_CHAT_MODEL`, `IA_CHUNK_SIZE`, `IA_CHUNK_OVERLAP`, `IA_TOP_K` | RAG settings |
| `dt` | `PRODUCTION` | `true` stores in Redis |
| `jwf`, `jsql` | `MAX_AUDIT_LOG` | Audit entries kept (1000) |
| `jwsp` | `WHATSAPP_API_URL` | WhatsApp Graph API base URL |
| `validator` | `LANG` | Message language |

## CLI binaries

```bash
go run ./cmd/et          # main CLI (cobra)
go run ./cmd/create      # scaffolding: project, microservice, model, rpc
go run ./cmd/apigateway  # API gateway (ettp)
go run ./cmd/daemon      # background service with systemd integration
go run ./cmd/server      # TCP node (port 1377, -port flag); ./cmd/client to connect
go run ./cmd/jrex        # JS runtime with hot-reload from ./cmd/jrex/src
go run ./cmd/jsql        # jsql demo
go run ./cmd/ia          # RAG HTTP API (PORT, default 3300)
```

Also `cmd/install`, `cmd/whatcher`, `cmd/resilience`, `cmd/wsp` and `cmd/jwf`.

## Development

```bash
gofmt -w .                            # format
go build ./... && go vet ./...        # compile and vet
go test ./jwf/ -race                  # tests of one package
cd jsql/drivers/test && go test -v    # jsql against sqlite, postgres, oracle, mysql and sql server
./version.sh --minor                  # bump the git tag, update this README and push
```

Packages with tests: `jsql/drivers/test`, `jwf`, `jrpc`, `jws`, `queue`, `ia`, `cache`, `aws`, `brevo` — some need live Redis, AWS or Brevo credentials.

## License

MIT. See `LICENSE`.
