# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Module

`github.com/cgalvisleon/et` — Go 1.25, MIT license. Part of the parent workspace `go.work`.

> The codebase changes fast (commits are bare "Backup:" WIP checkpoints) — **read the source before trusting any API shape here**. History of fixes (what changed, when, how it was tested) lives in `doc/CHANGELOG.md`, not here.

**Docs**: only `README.md` at the root; the rest in `doc/` — `doc/README.es.md` (Spanish translation, keep both in sync: same sections, same order, installation first), `doc/LIBRARY_CONTEXT.md` (copied into consumers by `go run ./cmd/install`), `doc/ARCHITECTURE_SUMMARY.md`, `doc/COMPONENT_CATALOG.md` (generated from `go doc` — regenerate, don't hand-edit), `doc/AI_USAGE_GUIDE.md` (these four come from the `lib-ctx-gen` skill and can drift), `doc/NEW_PROJECT_GUIDE.md` (verified templates for a new HTTP microservice or CLI on `et`), `doc/CHANGELOG.md`. The root `spec.md` is the stability plan (phases 0–5, detail in `doc/review-2026-10-27/`), not a library spec.

**Features**: since 2026-11-16 they live in the workspace’s single «Mesa de trabajo», `../features/` (outside this repo; `features/` here only keeps a README pointing there). When an agent starts, advances or finishes an et feature, it updates `../features/data/<id>.json` (with `"project": "et"`; status, plan, and a dated `log` entry) and regenerates the HTML with `node ../features/build.mjs`; only `data/` is edited by hand (format in `../features/README.md`). Numbers are global (crontab is F-3). The learning lessons that were in `learn/` moved to `../features/learn/`.

**After a code change**: update `README.md` + `doc/README.es.md` together, the touched package's spec (`jsql/spec.md` §12 gaps, `et/spec.md` §7 core, `tcp/spec.md`, `websocket/spec.md`), and add an entry to `doc/CHANGELOG.md`. Changes to `et` reach consumers live through `go.work`; for consumers built with `GOWORK=off` or Docker images, publish a tag and bump their `go.mod`.

## Commands

```bash
go build ./...                       # build + vet are clean
go vet ./...
gofmt -w .
go test ./queue/ -run Example_queue -race   # prefer one package over go test ./...
GOWORK=off go build ./...            # if go.work lists a missing module, build et on its own

go run ./cmd/server   # TCP echo node (port 1377, -port flag); cmd/client is its stdin console
go run ./cmd/jsql     # jsql driver demo
go run ./cmd/store    # store example: Open, Store, Put, Read, Insert, Update, Delete, Remove, Close
go run ./cmd/ia       # RAG HTTP API (PORT, default 3300)
go run ./cmd/create   # scaffolds Project/Microservice/Modelo/Rpc (`create modelo` prints a manual wiring step: file.MakeFile never overwrites)
docker compose up -d  # optional local Ollama on 11434

./version.sh --major | --minor | --request   # bumps git tag, updates README.md + doc/README.es.md, pushes
```

**Tests**: every package with `*_test.go` runs offline (use `-race`), **except** `cache/cache_test.go`, `cache/handler_test.go`, `crontab/TestJobs` (`NATS_HOST=localhost:4222`; the rest of `crontab` runs on an in-memory bus), `jrpc/`, `aws/`, `brevo/`, which need live Redis/NATS/AWS/Brevo and fail or hang without them. `cache/memory_test.go` and `middleware/` (telemetry with `httptest`) need nothing. `ia/` tests use a sqlite temp file + `httptest`; `ia`'s `Ask`/`IngestFile`/`IngestSQL` (OpenAI SDK) are untested. Verifier programs (no external services): `go run -race ./test/tcp`, `go run -race ./test/websocket`, .

**Cross-driver suite** `jsql/drivers/test/` (`TestCatalog`, `TestInsertUpdate`, `TestJoinGroupHaving`, `TestWrapper`, `TestCreateDropDB`, `TestFieldValue`, `TestMemoryMatchesSql`, `TestAttribsAtRoot`, `TestSearch`, `TestFullText`, templates…): runs each case on sqlite (temp file), postgres, mysql, mssql, oracle and josefina, reading settings from `et/.env` (`DB_*`, `ORACLE_*`, `JOSEFINA_*`) and `t.Skip`-ing unreachable engines — **a green run may have exercised only sqlite; check skips with `-v`**. `JSQL_DEBUG=1` logs the generated SQL (`DB_IS_DEBUG=true` for a regular `DB`). `et/.env` points at the workspace's local Postgres: for a disposable one export `DB_HOST`/`DB_PORT`/`DB_NAME`/`DB_USER`/`DB_PASSWORD` (godotenv does not override them) and `MYSQL_PORT`, `JOSEFINA_PORT`…; an unreachable port (e.g. `ORACLE_PORT=1`) skips an engine. Known (2026-11-14): Josefina rejects `ddlLock` (`hashtext`/`pg_advisory_xact_lock`, added to postgres `Load` on 2026-11-01), so every test that creates a table fails on josefina. `TestCatalog` writes `result/*` only with `-update`.

```bash
cd jsql/drivers/test && go test -v -run 'TestInsertUpdate/sqlite' .
```

## Code style

Code must be understandable by a human: clear names, short functions, one place per rule. GoDoc for every function, in English and **brief**: name + one or two lines on what it does (and why if not obvious), one line for params, one for returns; no narration of fix history (that goes in the spec/changelog). Most existing files (`et/json.go`, `jwf/`, `resilience/`, `stores/`) use a `/** ... @param ... @return ... **/` block style — match the surrounding file. `store/` comments are in Spanish.

**Catalog layout** (`jsql/`, `store/`, `tcp/`, `websocket/`): the whole public API lives in `catalog.go` as thin wrappers grouped in sections; the implementation is private (same name lowercased) in other files. New public API = private implementation + wrapper in `catalog.go`.

## Architecture

Modular utility library for Go microservices: every directory is an independent package, no central entry point.

### Core type: `et.Json` (`et/`)

`Json` (`map[string]interface{}`) is the lingua franca of the whole library: typed accessors (`Str`, `Int`, `Bool`, `Time`, `Json`, `Array`…), default-value variants (`ValStr(def, keys...)`), nested traversal via variadic keys. `Clone` is shallow (`DeepClone` deep); `Put(val)` replaces content in the same map.

- **Files**: `json.go` (Json, parsing, dates, `Serialize`/`ToJson`), `result.go` (`Item`, `Items`, `List` — the paginated result with `Rows`, `All`, `Count`, `Page`, `Start`, `End`, `Result`), `condition.go`, `evaluate.go`, `search.go` (text search: `ToSearch`, `MatchSearch`, `FoldText`, `SearchPattern`), `where.go`, `goja.go`, `safemap.go`, `msg.go`.
- **Parsing**: `decodeJSON` turns numbers into `float64` (many consumers type-assert it) except integers beyond 2^53 (`int64`); `parseJSON` also repairs raw control chars, double escaping and double encoding. Accessor conversions live in `stringOf`/`intOf`/`floatOfAny`/`boolOf`.
- **Conditions** (`et/spec.md`): one in-memory evaluator (`evaluate`) behind `EvaluateObject`, `EvaluateValue` and `Wheres`, with exactly SQL's semantics — AND binds tighter than OR, a condition without connector after the first is `AND`, exact integer comparison, case-insensitive `like` (`%`/`_`/`*`), `between` from `{min,max}` or `[min,max]`, `search` (web syntax of `websearch_to_tsquery`: words, `"phrase"`, `-excluded`, `or`; in memory whole words without case nor accents — SQL engines that fall back to `LIKE` match parts of words, see `et/spec.md` §3.7). `TestMemoryMatchesSql` checks memory = SQL. A value `{"field": "<ref>"}` is a field reference (`name`, `a->b`, `sum(items)`, `extract_year(x)`…), resolved from the same object in memory and as a column/aggregate in SQL (unresolvable → `1 = 0`).
- **Goja bridge** (`goja.go`): `et.ToJS(vm, v)` wraps a `Json`/`Item`/`Items`/`List` as a dynamic object over the same map (JS writes land in Go) whose prototype exposes every `Json` method by its Go name; `et.FromJS` goes back (an `et` object gives its same map). **Objects reach Go as `et.Json`, not `map[string]any`: every type switch on data must accept both.**
- Data types (`et.TypeData`): `KEY`, `TEXT`, `MEMO`, `INT`, `FLOAT`, `BOOL`, `DATETIME`, `JSON`, `BYTE`, `ANY`, `ARRAY`, `ARRAY_JSON|STRING|INT|FLOAT|BOOL|DATETIME`.

### SQL builder / ORM: `jsql/`

- `jsql.Load()` / `jsql.LoadTo(dbName, hostName...)` → `*DB`, config read via `envar`. `jsql/catalog.go` is the whole public API (`TestCatalog` checks its sections). `jsql/spec.md` is the formal spec with a numbered gap list (`B<n>`).
- **Drivers** in `jsql/drivers/<name>/`, self-registering via `init()` (import for side effect): `postgres`, `sqlite`, `mysql`, `mssql`, `oracle`, each with the same files (`connect`, `load` = DDL, `query`, `command`, `quoted`, `types`, `test`), plus `josefina` (embeds postgres over Josefina's Postgres protocol; implements `jsql.RootAttribs` so ATTRIB fields live at the document root, not `_source`; cannot create databases). **A change to SQL generation usually has to be mirrored in all five** (josefina inherits postgres).
- **`Driver` interface** (`connection.go`): `UseSchema()`, `CreateDB`, `DropDB`, `Connect`, `ExistModel`, `Load(model)` (DDL text), `Drop`, `Query` (SELECT text), `Command` (DML text); all but `UseSchema` take `timeout ...time.Duration` (none or 0 = no timeout, via `jsql.TimeoutContext`). `CreateDB`/`DropDB` are idempotent; postgres/mysql/mssql `Connect` creates the DB if missing; oracle never creates/drops. Optional `jsql.Tester`: `jsql.TestConnection(ctx, conn, query)` opens, pings, queries and closes **without creating the DB**. Postgres `Load` takes an advisory lock so concurrent services can run DDL at startup.
- **Table names**: `Model.Table` is `schema.name` (postgres/mssql) or `schema_name` (sqlite/mysql/oracle — no `CREATE SCHEMA`). Drivers build SQL from `model.Table`, so raw SQL against sqlite must use the prefixed name (`main_tasks`). A schema is mandatory in `Define`.
- **Concurrency**: `DB.Schemas` and `AuditLog` are mutex-guarded; definitions on a model are serialized by `Model.muDef` and replace maps/`Columns` copy-on-write because drivers read them lockless. Rules in the `Concurrency` block over `type Model` (`model.go`).
- **`TypeColumn`**: `COLUMN`, `ATTRIB` (key inside `_source` JSONB, cast on read; a non-column name is an attribute), `DETAIL`/`MASTER` (bridge table)/`ROLLUP` (virtual relations, resolved in batches of 500 parents), `CALCFUNC` (Go), `CALC` (JS per row), `TEMPLATE` (text built from `{{field|format:"arg"}}` at query time, never stored; can't be used in where/order by).
- Column constants (`define.go`): `ID`, `IDX` (`_idx`, text of `UnixMilli()` set by an auto `BeforeInsert`), `SOURCE`, `STATUS`, `VERSION`, `TENANT_ID`, `PROJECT_ID`, `CREATED_AT`, `UPDATED_AT`, `CODE`, `KIND`… Status constants `ACTIVE`, `ARCHIVED`, `CANCELED`, `PENDING`, … (`jsql.StatusList()`).
- **Triggers**: before/after × insert/update/delete, Go `func(tx *Tx, old, new et.Json) error` or JS scripts run by `jwf` with `OLD`/`NEW`/`TX`. **Whatever a script trigger writes must go through `TX`** (`.oneTx(TX)`, `GenSerieTx`): with sqlite, writing outside waits forever for the single connection.
- **Text search** (`search.go`, `jsql/spec.md` §6.1): `searches` in `Define`/model JSON or `model.DefineSearch(name, lang, fields...)` is a virtual field (TypeColumn `SEARCH` in `Query.getField`, no column; only in a `search` condition or `orders`, else `MSG_SEARCH_ONLY_SEARCH`). **`full_text`** (`"full_text": [fields]`, `full_text_lang`, `full_text_langs`; `Define.FullText…`, `model.DefineFullText`/`DefineFullTextLang`) is just the search named `full_text` (`jsql.FULL_TEXT`); `ToJson` writes it with its own keys (not in `searches`), `Model.UnmarshalJSON` rebuilds it, and having both is an error. A JSQL query's `"search": "text"` / `{text, lang}` (`Query.Search`, `.search` in scripts) is kept in `Query.fullText` and joined at render time by `Query.withFullText` (in `DB.query`, restored afterwards): AND into every OR group via `rollupConditions`, plus a relevance order when there are no `orders` (not for count/exists); no `full_text` → `MSG_FULLTEXT_UNDEFINED`. **Langs are ISO codes** (`es`, `en`, `pt-BR`) mapped by `jsql.SearchConfig` to a whitelist of Postgres configs (unknown → `simple`; engine names still accepted); default `DB_SEARCH_LANG` or **`spanish`** (was `simple` until 2026-11-14). The condition value may be `{"text", "lang"}` (`et.SearchValue`, `jsql.SearchOf` — drivers must read the value through it, never `.(string)`). Postgres: `to_tsvector(config, COALESCE(a,'') || ' ' || …) @@ websearch_to_tsquery` + one GIN per config of `lang`+`langs` (`jsql.SearchIndexConfigs`) over the identical expression + `ts_rank`; mysql: `MATCH … AGAINST` + one `FULLTEXT` (lang-independent) when all fields are text columns, else LIKE; josefina (`jsql.LikeSearch`), sqlite, oracle, mssql: `jsql.SearchLike`. Indexes that must reach existing tables (searches, `"method": "trgm"` → `pg_trgm` GIN) are applied on **every** `Init` (`Load` runs only for a new table) through the optional **`jsql.IndexApplier.ApplyIndexes`** (postgres: `CREATE/DROP INDEX CONCURRENTLY` one by one outside a tx on a dedicated `*sql.Conn` with a retried `pg_try_advisory_lock(hashtext('et.jsql.indexes'))` — never blocking `pg_advisory_lock`, whose open snapshot would deadlock the other service's concurrent build —, invalid indexes rebuilt, orphans dropped only if named `_fts_<8 hex>` and GIN over `to_tsvector`; mysql: `ALTER TABLE … ALGORITHM=INPLACE, LOCK=SHARED` with `GET_LOCK`; josefina overrides it to run its own `LoadIndexes` in a tx, otherwise it would inherit postgres'), falling back to the older `jsql.Indexer.LoadIndexes` (DDL run in a tx). `DB_SEARCH_INDEXES=off` skips it at `Init`; `db.ApplySearchIndexes(model)` applies it by hand. Tests: `TestSearch`, `TestFullText`, `TestSearchIndexesLock` (drivers/test; postgres: three services starting at once against the lock, and a `CREATE INDEX CONCURRENTLY` really cancelled with `pg_cancel_backend` left invalid and rebuilt by the next start), `jsql/search_test.go`.
- **`from` reaches any table** (`describe.go`): `DB.modelOf` resolves defined models first; with a schema, an undefined table is read from the catalog through the optional `jsql.Describer` (all five drivers, rows via `jsql.DescribeRows`, types via `typeDataOf`) and registered in its schema as an initialized model (`_source`/`_idx` become SourceField/IdxField; no keys, relations or triggers; never re-read if the table changes). Commands use it too. A `select` with two fields giving the same result name fails (`checkSelectNames`, `MSG_SELECT_DUPLICATED`). Tests: `TestReflectModel`, `TestTypeDataOf`.
- **Programs** (`program.go`, spec §8.4): `{"steps": {const, vars, do, max_steps}}` runs descriptors in one transaction (rolled back on error); `Bind` is applied before each step (never to `before_*`/`after_*` scripts); control steps `if` (value/data/query, match/exists, then/else), `for` (range/counter/while, break/continue, `max` = `DB_RECORD_LIMIT`), `set`, `return`, `error`, `script` (jwf with `vars`, `consts`, `TX`); `{"query": {...}, "one"?}` as a value is a subquery run in Go (over `DB_RECORD_LIMIT` rows without own limit = error). SQL subqueries are deliberately not implemented (each driver returns rows as one JSON object). Tests: `TestProgram`, `TestProgramSubqueryLimit`.
- `jquery.go` parses **JSQL** (the name of jsql's JSON format: "a JSQL query", "a JSQL command") query/command descriptors (`select`/`from`/`join`/`where`…, `insert`/`update`/`delete`/`upsert`/`bulk`; `admin.go` handles `drop`/`truncate`/`create`/`alter`). `from`/join `to` accept `{database, schema, model, as}` or `"schema.model:as"`.
- `wrapper.go` `Wrapper(db)` exposes jsql to goja; **`jsql/jsql.d.ts` is its TypeScript declaration — keep both in sync**. `Bind(source, vars)` applies `{{name}}` variables to a JSON (not for hand-written SQL).
- sqlite quirk: rows come back as one JSON object each; a nested JSON column inside is double-encoded.
- `.Debug()` logs SQL and skips execution; `.Test()` generates SQL only. `jsql.DefineSeries(db, schema)` → series/code generator.

```go
model, _ := db.Define(jsql.Define{
    Schema: "public", Name: "users", Version: 1, IdxField: jsql.IDX,
    Columns: []jsql.Column{
        {Name: "email", TypeData: et.TEXT, Default: ""},
        {Name: "name", TypeColumn: jsql.ATTRIB, TypeData: et.TEXT, Default: ""},
    },
    PrimaryKeys: []jsql.DefIndex{{Name: "email", Sorted: true}},
})
model.Init() // runs DDL
items, _ := model.Where(jsql.Eq("status", jsql.ACTIVE)).And(jsql.More("age", 18)).Limit(1, 20) // page, rows
_, _ = model.Insert(et.Json{...}).ExecTx(nil)
```

### HTTP servers

- **`server/`** — lightweight `chi.Mux` wrapper, no Redis/NATS. `Start` blocks until SIGINT/SIGTERM and then `Close` (HTTP `Shutdown` up to 10 s, then the `OnClose` hooks, once).
- **`router/`** — `router.NewApi(name, path, host, port, rpc, version)`; `Public`/`Session`/`Protected(group, method, path, name, handler)` prefix the Api path and register the endpoint (`Api.Endpoints()`); `Item`/`Items`/`List` wrap business functions; `UseAuthentication`, `UseAuthorization`, `UseLimitRate` (Redis keys `ratelimit:*`); `MountRpc(services)`. Used by `core-studio/api`, `postick/api`, `ettp/v2`. Routes can also be published to the gateway at any time with `PushApiGateway(method, path, resolve, …)` and removed with `RemoveApiGateway("METHOD:path")` (core-studio's engine does it for its webhooks); the package registry of published routes (re-published by `eventActionReset` on a gateway reset) is guarded by a mutex since 2026-11-16 and `GetRoutes()` returns a copy.
- **`ettp/v2/`** — gateway: `ettp.New(name, *Config)`; does **not** call `cache.Load()`/`event.Load()` — callers must. Syncs router state across replicas over NATS. The route tree resolves fixed segments and `{param}` only (no wildcards); since 2026-11-16 `Router.delete` clears the solver and prunes only empty nodes (before, deleting `/a/b` also dropped `/a` and deleting `/a` dropped `/a/b`; `router_test.go`). Events marked `Myself` are ignored: a gateway in the same process as the service never sees its own routes. **`ettp/v1/`** is older but not dead (apigateway/proxy/token files without v2 equivalent); check which version a `cmd/` imports.

### Infrastructure (external services)

- **`cache/`** — Redis, or an in-process memory store when `REDIS_HOST` is empty. `Get`-style calls return `(value, exists, error)`; a missing key is `def, false, nil` (there is no `ErrNotFound`).
- **`event/`** NATS (`Subscribe`/`Publish`/`Stack`/`Log`; `Subscribe`/`Queue`/`Stack` keep one subscription per channel —a second one on the same channel replaces it in the map and `Unsubscribe(channel)` only cuts the last—, so several listeners of one channel use `SubscribeWith(key, channel, f)`, `QueueWith(key, channel, queue, f)`, `StackWith(key, channel, f)` + `UnsubscribeKey(key)`/`HasKey(key)`: own map by key, the same key replaces its previous subscription, `MSG_ERR_NOT_CONNECT` without a connection; since 2026-11-16, F-11; tests in `event/handler_test.go` need `NATS_HOST`, otherwise skip), **`graph/`** Neo4j.
- **`jrpc/`** — `net/rpc` over TCP; `jrpc.Mount(host, port, services, packageName)`, `SetResolver(fn)` for methods mounted elsewhere, `Methods()`. Types sent through gob must be registered (lists and `time.Time` are).
- **`tcp/`** — TCP client/server with concurrent `Send`/`Request`, streams with credit flow control, identity hello, and a **Hub** (servers + clients, typed in-order `Message`s, heartbeat, metrics) used by josefina's cluster; optional HMAC + TLS 1.3 (`SetSecurity`, `NewTLS`). **Every change must update `tcp/spec.md` and `brecha.md` (bump "Versión del documento") and `test/tcp/test.go`'s `publicAPI` list/cases**; the verifier fails if a public method is untested.
- **`websocket/`** — pub/sub server + Go client (topics, round-robin queue groups, hooks that can reject). **Every change must update `spec.md` and `brecha.md`, `test/websocket/test.go`, rerun the verifier and record the result in `brecha.md`'s history.**

### `store/` — rewritten from scratch (F-23), PostgreSQL-style engine

The package is being rebuilt as a storage engine modeled on PostgreSQL's storage (proposal `../features/propuestas/et-store-arquitectura-postgres.md`, 56 decisions in `../features/data/et-store-motor.json`). **`store/doc.go` is the reference** (formats byte by byte, change order, lock order, checkpoint, recovery, failure rules, env vars, planned API); every internal package has its own `doc.go` with its format table — keep them in sync. Comments in Spanish. The old log-with-in-memory-index code was deleted on 2026-11-19 (its log lives on in `internal/wal`), and **wave 4 (2026-11-19) gave it its phase-1A API back**: `Open(dir)` → `*Engine` (`Store(name)` opens or creates, `DropStore`, `Stores`, `Checkpoint`, `Close` = final checkpoint) → `*Store` (`Put`, `Remove`, `Insert`, `Update`, `Delete`, `Read`, `Count`); each single operation is its own transaction (Read Committed, waits up to `LOCK_TIMEOUT` for another writer); `cmd/store` is the example. Engine files: `catalog.go` (public wrappers), `engine.go`, `store.go`, `checkpoint.go` (steps 1–6, background checkpointer, `commitMu` waits for in-flight commits like PostgreSQL's delayChkpt), `recovery.go` (dispatches each WAL kind to its owner's `Redo`), `ddl.go`, `counts.go`, `records.go`.

- Two user rules: nothing of the B+tree in RAM beyond a fixed page cache (no structure grows with the number of keys), and `Open` does not grow with the index (control file + WAL replay since the last checkpoint).
- Work goes by waves, one internal package per piece, several agents in parallel (each edits only its package): 0 formats/interfaces (done); 1 `page`, `vfs` (`vfs.OS()`), `crashfs` (`crashfs.New()`, simulated crashes: lose/reorder/torn unsynced writes, a failed fsync forgets pending data), `smgr` (1 GB files per store, fixed descriptor cache), `control` (two alternating 512 B copies + atomic catalog), `wal` (from the old log: reservation under `writeMu`, parallel writes, contiguous `published`, group commit; S2-compressed pre-change page images, `ErrNeedImage`, segment recycling), `xact` (2 bits per xid, fixed SLRU-like cache, commit lsn per group in memory) — done; 2 `buffer` (fixed page cache, `PinZero` for redo), `fsm` (free list + visibility map) — done; 3 `btree` (Lehman-Yao, two-step splits, suffix truncation, fast-path insert, scans both ways, `Verify`), `overflow` (chains of one version each) — done; 4 checkpoints, recovery, engine and API — done; 5 batches, snapshots, MVCC waits, vacuum (phase 1B).
- Invariants that bite: «WAL before the page»; never wait for a transaction or a fsync with a page locked; a failed write or fsync makes the engine unusable (never retry a fsync); every WAL reservation must be written, even on error, or `published` stalls and `Close` hangs.
- Tests: the engine's `TestModelCrash` (random ops on 3 stores vs a map, crash at a random point in the 3 `crashfs` modes × 8 seeds, 3 rounds, `btree.Verify` + `Count` checked) and `TestCrashBetweenCheckpointSteps` (a crash after each checkpoint step) are the key ones. `go test ./store/... -race` (all offline) **and also without `-race`**: an atomic load on an unaligned address passed under `-race` and crashed with SIGBUS on arm64 without it (btree, ola 3). Fuzzers per package (`go test ./store/internal/<pkg> -run '^$' -fuzz FuzzX -fuzztime 1m`; for `page`, add `-fuzzminimizetime 1x`). Don't benchmark while other agents run.

### `jwf/` (workflows and scripts)

All JavaScript in the library runs through `jwf` on goja: `jwf.Run(code, bindings, reads...)`, `RunCode`/`RunScripts`, `jwf.NewRunner(bindings)` (one runtime for many runs — jsql `CALC` columns), `jwf.Bindings(map)` adds the standard ones (console, basic functions, `fetch`, `jrpc`, `cache`, `event`, `sleep`, `validator`, `evaluateObject`/`evaluateValue`, field helpers). Script errors come back as the plain JS message, without `at <eval>`.

```
WorkFlow (ID, TimeAwait, AuditLog)
  Flows map[tag]*Flow
    Steps map[id]*Step, Connections {Source, Target, Kind input/output/error}, Triggers,
    TotalAttempts / TimeAttempts / TimeAwait, Constants, SaveInstances, Published
```

```go
wf, _ := jwf.New("") // calls cache.Load() + event.Load()
wf.NewFlow("add", "add item", "1.0.0", ownerId, userId).
    Step("add", "step 1", func(instance *jwf.Instance, ctx et.Json) (et.Json, error) { return ctx, nil })
// Run(tag, triggerTag, id, code, ctx, tags, await, userId) — returns only the result ctx
result, err := wf.Run("add", "add", reg.GetULID(""), "", et.Json{}, et.Json{}, true, userId)
```

- **No store**: flows live in memory and are persisted by the caller via `OnChange`; `jwf.LoadFlow(json)` / `wf.SetFlow(def, userId)` rebuild them.
- **Error port by node key (2026-11-16)**: `Instance.runError` finds the `kind: error` connection by `StepId` (the node's key in `flow.Steps`, what connections name), not by `Step.ID` (in coreStudio nodes it is the catalog step id, e.g. `error:throw`, so the error port never ran); after the error-port step runs the instance ends `FAILED` (it was left `RUNNING`). `TestErrorPortByNodeKey`.
- Instances live in Redis (`instance:<id>`, no TTL); `StopInstance(id)`. A step error retries via `resilience` when `TotalAttempts != 0`, then follows the error connection; `goTo(n)` to an unconnected output ends the run.
- **Steps of a `Flow` are shared by every concurrent run**: bindings are per run, status goes to the instance. Never mutate a step of a shared flow while running it — test a copy (`LoadStep(step.ToJson())`).
- In scripts, `params`/`ctx` are dynamic objects (`ctx.user.name`, `ctx.total = …`, plus methods `str`, `get`, `set`, `put`, `view`, `omit`…); a nested change doesn't persist — reassign the top key or use `set`. Other Go values returned to JS (`et.Json` from a Go func) expose methods but not keys unless wrapped with `GoValue`.
- `console` lines reach `Instance.OnLog` / `Step.OnLog` (a step run alone).

### Other packages

- **Utility**: `envar/` (`GetStr/GetInt/...` with defaults; values are cached — use `envar.SetInt` in tests, not `t.Setenv`), `logs/` (`Fatal` exits — use `Alert`/`Error` otherwise) over `stdrout/`, `jwt/` over `claim/` (HS256, `SECRET` required), `validator/` (fluent, `ByJson`/`ToJson`, goja `Wrapper()`), `queue/` (`queue.New[T](size, maxEvents, period, handler)`), `request/` (`URLParam`, `GetBody`, outbound `Fetch`; context getters/setters `UserID`, `TenantID`, `RoleID`, `Payload`… and, since 2026-11-16, `Token(r)`/`SetToken(ctx, token)` — the token `middleware.Authentication` stored already validated under `TokenKey`, without «Bearer »: read it instead of `r.Header.Get("Authorization")`), `response/` (`ITEM`, `ITEMS`, `HTTPError`), `middleware/` (`Logger` request telemetry, `Recoverer`), `crontab/` (`New(tag, store, opts...)` instances — `WithHandler(fn)` serves every job without its own fn (`CronJob`/`ScheduleJob` with fn nil; neither → error) and is subscribed to the reloaded jobs before the replica can lead and to the ones the leader broadcasts, so a catch-up fire is never lost; `ScheduleJob` with a past `at` keeps an existing job with the same definition, `Load` = the global one behind the package functions; replicas (processes) with the same tag form a group over NATS: `hello` every `CRONTAB_HEARTBEAT`, the oldest live one leads — schedules, dispatches, writes the Store, broadcasts state — and the others keep copies and take over; `New` waits 1.5 heartbeats to receive the leader's jobs; job id = job tag, a different definition replaces it (greater UUID v7 `Rev` wins), the same one keeps its state; jobs reload from the Store (`Query({collection, crontab})`); lifecycle `JobStatus` apart from each `Run` (retries, timeout, skipped); lock order `electMu` → `Job.mu` → `Crontab.mu`; tests run on an in-memory bus, only `TestJobs` needs NATS), `resilience/` (synchronous retry), `reg/` (UUID v7, ULID, XID, `SID` — 11-char base62, sortable, 64 bits = ms + node + sequence; node from `SID_NODE` or `SetSIDNode`, random if unset; `GenSnowflake` = the process `SIDInt()` as 19 zero-padded digits), `xls/`, `csv/`, `utility/`, `file/`, `dt/`, `timezone/` (`Parse` takes a key of its layouts table, not a Go layout), `strs/`, `msg/`, and smaller helpers.
- **Messaging**: `aws/` (S3 incl. S3-compatible endpoints, SES, SMS), `brevo/`, `jwsp/` (WhatsApp), `infobip/`.
- **`ia/`** — multitenant RAG (`tenant_id` + `project_id`): `Load`/`LoadIndex`, `IngestText`/`IngestFile`/`IngestSQL`, `Search`, `Ask`. No pgvector: embeddings are JSON and ranked by cosine in Go; `Embedder`/`HttpEmbedder` for any OpenAI-compatible endpoint. jsql does not migrate: new columns on an existing table need a manual `ALTER` (see `doc/CHANGELOG.md` 2026-10-12).
- **`jia/`** — OpenAI agents (unrelated to `ia`), no consumer in the workspace. **`stores/`** — jsql-backed instance/authorization/config helpers.
- **`cmd/`**: `et` (cobra CLI), `apigateway`, `create`, `server`/`client`, `jsql`, `install`, `watcher`, `resilience`, `wsp`, `ia`, `jwf`; `daemon` is `package daemon`, not runnable.

## Key patterns

- **`Load()` initialization**: infrastructure packages read env via `envar` and connect once; later calls are no-ops.
- **Local `Store` interfaces**: `jwf`, `jia`, `resilience`, `crontab` each define their own and the caller implements it. Signatures differ (`jsql.Store` satisfies `jia.Store`, not `jwf.Store`) — check before reusing.
- **Messages**: use package-local `msg.go`/`msg/` constants, not hardcoded strings.
- **HTTP handlers** (`handler.go` everywhere):

```go
func (s *T) HttpFoo(w http.ResponseWriter, r *http.Request) {
    body, err := request.GetBody(r)
    if err != nil {
        response.HTTPError(w, r, http.StatusBadRequest, err.Error())
        return
    }
    response.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: data}) // StatusCreated for creating POSTs
}
```

## Environment variables

| Package | Variables |
| --- | --- |
| `jsql` | `DB_DRIVER`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_RECORD_LIMIT`, `DB_SEARCH_LANG` (`spanish`), `DB_SEARCH_INDEXES` (`on`), `DB_IS_DEBUG`, `DB_APP_NAME`, `DB_SSLMODE` (postgres), `DB_SSL`/`DB_SSL_VERIFY` (oracle), `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_LIFETIME`, `DB_POOL_IDLE_TIME` |
| `cache` | `REDIS_HOST`, `REDIS_PASSWORD`, `REDIS_DB` |
| `event` | `NATS_HOST`, `NATS_USER`, `NATS_PASSWORD` |
| `graph` | `NEO4J_HOST`, `NEO4J_USER`, `NEO4J_PASSWORD` |
| `jia`, `ia` | `OPENAI_API_KEY` |
| `ia` | `IA_SCHEMA`, `IA_EMBEDDING_MODEL`, `IA_CHAT_MODEL`, `IA_CHUNK_SIZE`, `IA_CHUNK_OVERLAP`, `IA_TOP_K` |
| `store` | `STORE_CACHE_SIZE` (128 MB), `WAL_SEGMENT_SIZE` (128 MB), `SYNC_ON_WRITE` (true), `CHECKPOINT_TIMEOUT` (300 s), `MAX_WAL_SIZE` (1024 MB), `LOCK_TIMEOUT` (5 s); `TRANSACTION_TIMEOUT` from phase 1B |
| `dt`, `ettp` | `PRODUCTION` (`true` = Redis) |
| `jwsp` | `WHATSAPP_API_URL` |
| `claim` | `SECRET` (required: `logs.Fatal` without it) |
| `validator` | `LANG` |
| `jwf`, `crontab` | `MAX_AUDIT_LOG` (1000) |
| `crontab` | `CRONTAB_RUN_TIMEOUT` (1h; `SetRunTimeout`/`WithTimeout`), `CRONTAB_HEARTBEAT` (1s), `CRONTAB_CATCH_UP` (1h: a schedule late by less runs when a leader takes over) |
| `middleware` | `REQUESTS_LIMIT` (400/s per method + path), `LATENCY_LIMIT` (1000 ms), `TELEMETRY_FORMAT` (`text`/`json`/`off`), `TELEMETRY_LOG` (`all`/`errors`) |
| `jwt` | `JWT_EPHEMERAL_DURATION` (15 min) |
| `reg` | `SID_NODE` (0..1023; unset = random node per process) |
| `ettp/v2` | `COMPANY`, `WEB`, `HELP` |
