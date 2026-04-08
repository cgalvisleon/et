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

# Build
go build ./...
```

## Architecture

This is a **modular utility library** for building Go microservices. Each directory is an independent package imported separately. There is no central entry point — consumers import only the packages they need.

### Core type: `et.Json`

`et/json.go` defines `Json` (`map[string]interface{}`), the primary data structure used throughout the entire library. It has typed accessors (`Str`, `Int`, `Bool`, `Time`, `Json`, `Array`, etc.) with a default-value pattern (`ValStr(def, keys...)`) and nested key traversal via variadic `atribs ...string`. This type is the lingua franca across all packages.

`et/list.go` defines `List` — the standard paginated result type (`Rows`, `All`, `Count`, `Page`, `Start`, `End`, `Result []Json`).

`et/item.go` and `et/items.go` define single-item and multi-item result wrappers.

### Infrastructure packages (require external services)

- **`cache/`** — Redis client (requires `REDIS_HOST`, optionally `REDIS_PASSWORD`, `REDIS_DB`). `cache.Load()` initializes; provides `Set`, `Get`, `Delete`, `Pub`, `Sub`.
- **`event/`** — NATS pub/sub (requires `NATS_HOST`, optionally `NATS_USER`, `NATS_PASSWORD`). `event.Load()` initializes; provides `Subscribe`, `Publish`, `Stack`.
- **`ettp/v2/`** — HTTP server built on `go-chi/chi`. `ettp.New(name, config)` calls `cache.Load()` + `event.Load()` internally. Router state is synchronized across instances via NATS events.
- **`jrpc/`** — JSON-RPC inter-service communication over NATS.

### Self-contained utility packages

- **`config/`** — App config/env with getters `GetStr`, `GetInt`, `GetBool`, `GetFloat`, `GetTime` and CLI param helpers `ParamStr`, `ParamInt`, etc. The `config.App` struct holds `name`, `version`, `company`, `host`, `port`, `stage`.
- **`envar/`** — Low-level env var access; `envar.Validate([]string{...})` checks required vars exist.
- **`logs/`** — Structured logging. Functions: `Log`, `Info`, `Infof`, `Alert`, `Alertf`, `Error`, `Errorf`, `Debug`, `Debugf`, `Fatal`, `Tracer`. All route through `stdrout` for colorized output.
- **`claim/`** — JWT claims with `tenantId` (not `projectId`).
- **`crontab/`** — Job scheduler. `crontab.Load(tag)` initializes; `AddJob`, `AddOneShotJob`, `AddEventJob` register jobs. Supports `robfig/cron` spec format including seconds (`"0 * * * * *"`).
- **`strs/`** — String utilities.
- **`utility/`** — Crypto, validation, general helpers.
- **`middleware/`** — HTTP middleware (CORS, request ID, logger, auth).
- **`response/`** — Unified HTTP response helpers.
- **`ws/`** — WebSocket support via `gorilla/websocket`.

### Application-layer packages

- **`vm/`** — Embeds a JavaScript runtime (`dop251/goja`) for executing JS from Go. Three modes: `Develop` (reads files directly, hot-reloads via `file.Watcher`), `Production` (loads from a `Store`), `Building` (compiles + stores with semver bumping). Global wrappers provide `console.*`, `ctx.*`, `fetch()`, and CommonJS-style `require()`. `RunDev(baseDir)` and `RunProd(store)` are the entry points.
- **`ia/`** — OpenAI agent integration (`openai-go/v3`). Manages agents with conversation tracking, event handlers, and instance state via a caller-provided `instances.Store`.
- **`workflow/`** — Workflow orchestration with multi-step execution, instance state, and resilience patterns. Integrates with `resilience/`, `instances/`, and `event/` (NATS) for async state sync.
- **`graph/`** — Neo4j connectivity (`neo4j-go-driver/v5`). `graph.Load()` returns a `*Conn` with the Neo4j driver.
- **`wsp/`** — WhatsApp Business API client. `NewWhatsapp(token, phoneNumberId)` produces a message builder; uses Facebook Graph API (configurable via `WHATSAPP_API_URL`).
- **`instances/`** — `Store` interface (`Set`, `Get`, `Delete`, `Query`) used by `ia` and `workflow` for state persistence. Implementations are caller-provided.
- **`resilience/`** — Resilience patterns (circuit breaker, etc.) used by `workflow`.
- **`reg/`** — Service registration/discovery used by `ia` and `workflow`.
- **`file/`** — File operations and watching (`FileInfo`, `Watcher`, `ExistPath()`); used by `vm` for hot-reload.
- **`mem/`** — Shared memory and sync primitives.
- **`ephemeral/`** — Ephemeral/temporary data structures.
- **`iterate/`** — Iteration control with time support.
- **`race/`** — Race condition detection helpers.
- **`cmds/`** — Command/stage execution system (distinct from the `cmd/` CLI binaries).
- **`timezone/`**, **`units/`**, **`color/`** — Timezone handling, unit conversions, terminal color utilities.

### CLI (`cmd/`)

Each subdirectory under `cmd/` is a standalone binary:
- `cmd/et/` — Main CLI using `cobra`
- `cmd/apigateway/` — API Gateway/proxy
- `cmd/daemon/` — Background service with systemd integration
- `cmd/create/` — Project/code scaffolding
- `cmd/vm/` — JavaScript VM runner; `go run ./cmd/vm` starts `vm.RunDev("./cmd/vm")` with hot-reload

### Code generation (`create/`)

Templates and generators for new microservices, projects, and deployments (Kubernetes). Used by the `cmd/create` CLI.

## Key patterns

- **Initialization pattern**: Infrastructure packages expose a `Load()` function that reads env vars via `envar` and establishes connections. Call `Load()` once at startup; subsequent calls are no-ops.
- **Error handling**: `logs.Fatal(err)` calls `os.Exit(1)`. Use `logs.Alert` / `logs.Error` for non-fatal errors.
- **Event-driven coordination**: `ettp/v2` server syncs router state across replicas via NATS (`router.EVENT_SET_ROUTER`, `EVENT_REMOVE_ROUTER`, `EVENT_RESET_ROUTER`). The `m.Myself` flag prevents self-processing.
- **`msg/` packages**: Each package has a local `msg/` or `msg.go` file with error message constants — use these instead of hardcoded strings.
- **Store interface pattern**: `vm`, `workflow`, and `ia` accept a caller-provided `instances.Store` for persistence — the library defines the interface, consumers implement it.

## Required environment variables

| Package | Variable | Purpose |
|---------|----------|---------|
| `cache` | `REDIS_HOST` | Redis connection |
| `event` | `NATS_HOST` | NATS connection |
| `event` | `NATS_USER`, `NATS_PASSWORD` | NATS auth (optional) |
| `wsp` | `WHATSAPP_API_URL` | WhatsApp Graph API base URL (optional) |
