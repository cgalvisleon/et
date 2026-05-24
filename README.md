# ET: Go Services and Tools Library

Modular Go library for microservices, CLIs, and web applications. Import only the packages you need — there is no central entry point.

- Module: `github.com/cgalvisleon/et` — Go 1.23+ — MIT

## Installation

```bash
go get github.com/cgalvisleon/et@latest
go get github.com/cgalvisleon/et@v1.0.24
```

```go
import (
    "github.com/cgalvisleon/et/et"
    "github.com/cgalvisleon/et/cache"
)
```

## Packages

| Package       | Description                                                                  |
| ------------- | ---------------------------------------------------------------------------- |
| `et/`         | Core types: `Json`, `List`, `Item`, `Items`                                  |
| `cache/`      | Redis client with Pub/Sub. Init: `cache.Load()`                              |
| `event/`      | Pub/Sub over NATS. Init: `event.Load()`                                      |
| `ettp/v2/`    | Full-featured HTTP server (go-chi + Redis + NATS). Init: `ettp.New(name, config)` |
| `server/`     | Lightweight HTTP server (chi only, no external deps). Init: `server.New()`  |
| `router/`     | Standalone HTTP router with cross-instance sync via NATS                    |
| `jrpc/`       | Go `net/rpc` over TCP with load balancing and Raft consensus                |
| `jwt/`        | JWT token creation (`New`, `NewAuthentication`, `NewAuthorization`)          |
| `claim/`      | JWT claims struct with `tenantId` field. Signs with HS256                   |
| `crontab/`    | Cron scheduler with seconds support. Init: `crontab.New(tag)`               |
| `middleware/` | CORS, auth, logger, request ID, telemetry, panic recovery                   |
| `response/`   | Unified HTTP responses: `ITEM`, `ITEMS`, `HTTPError`                        |
| `request/`    | Inbound helpers (`URLParam`, `GetBody`) and outbound HTTP client            |
| `ws/`         | Bidirectional WebSocket (gorilla/websocket)                                 |
| `jsql/`       | Database-agnostic SQL builder and lightweight ORM                           |
| `jval/`       | Fluent JSON validation: typed validators with chainable constraints         |
| `logs/`       | Structured leveled logging with colorized output                            |
| `strs/`       | String utilities                                                             |
| `utility/`    | Crypto, hashing, ID generation (UUID, ULID, Snowflake)                      |
| `envar/`      | Environment variable access and CLI argument helpers                        |
| `config/`     | App configuration (`GetStr`, `GetInt`, `GetBool`, `GetFloat`, `GetTime`)    |
| `service/`    | OTP helpers (`SendOTPEmail`, `SendOTPSms`, `VerifyOTP`) and messaging       |
| `mem/`        | In-memory cache with expiration and sync primitives                         |
| `ephemeral/`  | Short-lived temporary data structures                                       |
| `vm/`         | Embedded JavaScript runtime (goja) with hot-reload                         |
| `ia/`         | OpenAI agent integration (`openai-go/v3`) with conversation tracking        |
| `workflow/`   | Multi-step workflow orchestration with instance state and resilience        |
| `graph/`      | Neo4j connectivity (`neo4j-go-driver/v5`)                                   |
| `instances/`  | `Store` interface for state persistence used by `ia`, `workflow`, `vm`      |
| `resilience/` | Circuit breaker and resilience patterns                                     |
| `reg/`        | Service registration and ID generation helpers (ULID, etc.)                 |
| `aws/`        | AWS SDK wrapper: S3, SES (email), SMS                                       |
| `brevo/`      | Brevo API client: email, SMS, WhatsApp                                      |
| `wsp/`        | WhatsApp Business API client                                                |
| `tcp/`        | Distributed TCP node with Raft-style leader election                        |
| `file/`       | File operations and filesystem watcher                                      |
| `color/`      | ANSI terminal colors                                                         |
| `stdrout/`    | Low-level colorized stdout routing used by `logs`                           |
| `timezone/`   | Timezone helpers                                                             |
| `units/`      | Unit conversion utilities                                                   |
| `race/`       | Race condition detection helpers                                             |
| `cmds/`       | Command and stage execution system                                          |
| `iterate/`    | Iteration control with time support                                         |
| `create/`     | Code templates for microservices and Kubernetes deployments                 |
| `cmd/`        | CLI binaries: `et`, `apigateway`, `daemon`, `server`, `vm`, `jsql`, …      |

## Core type: `et.Json`

`et.Json` (`map[string]interface{}`) is the library's universal data type. Typed accessors with default-value pattern and nested key traversal:

```go
data := et.Json{"user": et.Json{"name": "Ana", "age": 30}}
name := data.Str("user", "name")       // "Ana"
age  := data.Int("user", "age")        // 30
ok   := data.Bool("active")            // false (zero value)
data.Set("active", true)
```

`et.List` — paginated result (`Rows`, `All`, `Count`, `Page`, `Start`, `End`, `Result []Json`).  
`et.Item` / `et.Items` — single and multi-item result wrappers.

## Initialization pattern

`Load()` is idempotent and thread-safe. Call once at startup:

```go
cache.Load()  // requires REDIS_HOST
event.Load()  // requires NATS_HOST
```

`ettp.New(name, config)` calls `cache.Load()` and `event.Load()` internally.

## SQL builder: `jsql/`

Database-agnostic SQL builder and lightweight ORM. Supports PostgreSQL and SQLite.

```go
import _ "github.com/cgalvisleon/et/jsql/drivers/postgres"

db, _ := jsql.Load() // reads DB_DRIVER, DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME

// Define a model (adds id, created_at, updated_at, _source JSONB, _idx)
model, _ := db.DefineModel("public", "users", 1)
model.DefineAttrib("name", jsql.TEXT, "")
model.Init()

// Query
items, _ := model.Where(jsql.Eq("status", "active")).Limit(20).Page(1).All()
item,  _ := model.Where(jsql.Eq("id", id)).One()

// Commands
_, _ = model.Insert(et.Json{"email": "a@b.com"}).ExecTx(nil)
_, _ = model.Update(et.Json{"status": "archived"}).Where(jsql.Eq("id", id)).ExecTx(nil)
_, _ = model.Upsert(et.Json{"id": id, "email": "a@b.com"}).ExecTx(nil)
```

Both `Query` and `Command` support `.Debug()` (logs SQL, skips execution) and `.Test()` (returns SQL string without executing).

## Workflow orchestration: `workflow/`

Multi-step workflow engine with instance state, rollback, and resilience.

**Type hierarchy:**

```
Flow  (definition)
  ├── Steper  (named path/lane, identified by tag)
  │     └── Steps []int  (indexes into Flow.Steps)
  └── Steps  []*Step  (shared step pool: Definition, Undo, Stop)

Instance  (runtime execution of a Flow for a specific entity ID)
```

```go
workflow.Load(myStore) // myStore implements instances.Store

// Define a flow
flow, _ := workflow.NewFlow("onboarding", "v1", "Onboarding", "User onboarding flow", "admin")
flow.Resilence(3, 5*time.Second, "ops-team", "high")

steper, _ := flow.NewSteper("main", "Main Path", "Primary onboarding steps")
step, _   := flow.NewStep(workflow.Def{
    Name:       "send-welcome",
    Definition: `/* JS or definition */`,
    Undo:       `/* rollback logic */`,
    Stop:       false,
})

// Run an instance
result, err := workflow.RunInstance(entityID, "onboarding", 0, ctx, tags, username)

// Lifecycle operations
workflow.ResetInstance(id, username)    // reset to step 0
workflow.RollbackInstance(id, username) // execute undo chain
workflow.StopInstance(id, username)     // halt execution
```

HTTP handlers exposed on `*WorkFlow`: `HttpGetFlow`, `HttpNewFlow`, `HttpDeleteFlow`, `HttpNewStep`, `HttpSetStep`, `HttpDeleteStep`, `HttpNewSteper`, `HttpSetSteper`, `HttpDeleteSteper`, `HttpAddStepFromSteper`, `HttpRemoveStepFromSteper`, `HttpMoveStepFromSteper`, `HttpGetInstance`, `HttpRunInstance`, `HttpResetInstance`, `HttpRollbackInstance`, `HttpStopInstance`.

## Crontab

```go
ct := crontab.New("my-service")

ct.AddJob("job-1", "0 * * * * *", et.Json{"msg": "hello"}, 0, true,
    func(job *crontab.Job) { logs.Infof("[cron] %s", job.Params.ToString()) },
)

ct.AddEventJob("job-2", "@every 30s", "my-channel", 0, true,
    et.Json{"foo": "bar"},
    func(msg event.Message) { logs.Infof("[event] %s", msg.Data.ToString()) },
)
```

API: `New`, `AddJob`, `AddOneShotJob`, `AddEventJob`, `AddOneShotEventJob`, `DeleteJob`, `StartJob`, `StopJob`, `Stop`.

## JavaScript runtime (`vm/`)

Embedded goja runtime with three modes: `Develop` (hot-reload from files), `Production` (loads from Store), `Building` (compiles + semver bump). Exposes `console.*`, `ctx.*`, `fetch()`, `require()`.

```go
v := vm.New("my-vm")
v.RunDev("./cmd/vm")
```

## AI agents (`ia/`)

```go
agent := ia.New(name, myStore) // myStore implements instances.Store
```

Manages conversation tracking, event handlers, and per-instance state. Requires `OPENAI_API_KEY`.

## Logging

```go
logs.Info("message")
logs.Errorf("failed: %v", err)
logs.Fatal(err)                  // calls os.Exit(1)
logs.EnableCallerInfo = false    // disable runtime.Callers in Error() — recommended in production
```

## Required environment variables

| Package | Variable                                                                    | Purpose                              |
| ------- | --------------------------------------------------------------------------- | ------------------------------------ |
| `cache` | `REDIS_HOST`                                                                | Redis host                           |
| `cache` | `REDIS_PASSWORD`, `REDIS_DB`                                                | Redis auth and database (optional)   |
| `event` | `NATS_HOST`                                                                 | NATS host                            |
| `event` | `NATS_USER`, `NATS_PASSWORD`                                                | NATS auth (optional)                 |
| `claim` | `SECRET`                                                                    | JWT signing key (default: `"1977"`)  |
| `jsql`  | `DB_DRIVER`                                                                 | Driver name: `postgres` or `sqlite`  |
| `jsql`  | `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`                   | Database connection                  |
| `jsql`  | `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_CONN_LIFETIME`, `DB_POOL_CONN_IDLE_TIME` | Connection pool (optional) |
| `graph` | `NEO4J_HOST`, `NEO4J_USER`, `NEO4J_PASSWORD`                                | Neo4j connection                     |
| `ia`    | `OPENAI_API_KEY`                                                            | OpenAI API key                       |
| `wsp`   | `WHATSAPP_API_URL`                                                          | WhatsApp Graph API base URL (optional)|

## CLI binaries

```bash
go run ./cmd/et                          # main CLI (cobra)
go run ./cmd/apigateway                  # API Gateway / proxy
go run ./cmd/daemon                      # background service with systemd integration
go run ./cmd/server                      # TCP node server (default port 1377, use -port flag)
go run ./cmd/vm                          # JS VM with hot-reload
go run ./cmd/jsql                        # jsql driver demo and test
go run ./cmd/client -addr localhost:1377 # TCP test client
```

## Development

```bash
gofmt -w .           # format code
go build ./...       # compile all packages
go test ./...        # run tests (no *_test.go files yet — compiles only)
./version.sh --minor # semantic version bump (reads git tags, updates README, pushes tag)
```

## License

MIT. See `LICENSE`.
