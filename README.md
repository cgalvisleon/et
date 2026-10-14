# ET: Go Services and Tools Library

> [Versión en español](doc/README.es.md)

## Installation

Requires Go 1.25. Add the module and import only the packages you use:

```bash
go get github.com/cgalvisleon/et@latest
go get github.com/cgalvisleon/et@v1.0.38
```

```go
import (
    "github.com/cgalvisleon/et/et"
    "github.com/cgalvisleon/et/jsql"
    _ "github.com/cgalvisleon/et/jsql/drivers/postgres" // SQL drivers register themselves on import
)
```

## What it is

A modular library for Go microservices, CLIs and web apps. Every directory is an independent package; there is no central entry point. Module `github.com/cgalvisleon/et`, MIT license.

`et.Json` (`map[string]interface{}`) is the common currency: almost every API receives or returns it.

Infrastructure packages (`cache`, `event`, `jsql`) read their configuration from environment variables and connect once in `Load()`; later calls are no-ops.

## Upgrading to v1.0.38

- `router.Publish` / `router.With` no longer prefix `Route.PackageName` onto `Route.Path`: put the full path in the route.
- `jsql` models require a schema. On SQLite, MySQL and Oracle the table is `schema_name` (before, SQLite used the bare name), so existing SQLite databases need their tables renamed.
- `cache.Get*` return `(value, exists, error)`; `cache.ErrNotFound` is gone.
- `jwf.New(id)` takes only the id; flows are persisted through `OnChange`.
- jsql scripts (goja): `where` of a model, query or command takes a JSON condition or a list (`[{status: {eq: "active"}}, {and: {age: {more: 18}}}]`); `and`/`or` are gone. A single object must be wrapped in `where` (`{where: {status: {eq: "active"}}}`) — a bare `{status: {…}}` is ignored for now (`jsql/spec.md` B14).
- `router.Api.Public` / `Session` / `Protected` take `group` first.

## Packages

Each table lists the main public API of every package. `go doc github.com/cgalvisleon/et/<package>` shows the full list.

### Core data

| Package | Public API | Example |
| --- | --- | --- |
| `et` | `Json` (`Str`, `Int`, `Bool`, `Num`, `Time`, `Json`, `ArrayStr`, `ArrayJson`, `ValStr`, `ValInt`, `Set`, `Put`, `Clear`, `Get`, `Update`, `Clone`, `ToString`, `ToObject`), `Item`, `Items`, `List`, conditions (`Eq`, `More`, `In`, `Between`, `ToCondition`, `AddCondition`), `From(items).Where(...).All()` | `data.ValStr("Cali", "user", "city")` |
| `jsql` | `Load`, `LoadTo`, `ConnectTo`, `CreateDB`, `DropDB`, `TestConnection`; `DB.Define`, `DefineModel`, `Query`, `Sql`; `Model.Insert`, `Update`, `Upsert`, `Delete`, `Bulk`, `Select`, `Where`, `WhereByJson`; `Query.All`, `One`, `Limit`, `Count`; `NewTx`, `DefineSeries` (`GenSerie`, `GenSerieTx`, `GenValue`, `GenValueTx`); `Bind(source, vars)` (`{{name}}` variables); `Wrapper`, `BindWrapper`, `ToJsMapWrapper` (goja) | `users.Where(jsql.Eq("id", id)).One()` |
| `stores` | `DefineInstance` (`Set`, `Get`, `Delete`, `Query`), `DefineAuthorization` (`SetAuthor`, `Author`, `RemoveAuthor`), `DefineConfig` (`SetConfig`, `Get`), `DefineFormState` | `ok, _ := auth.Author(profileId, "GET", "/users")` |
| `mem` | `Set`, `Get`, `GetStr`, `GetInt`, `GetJson`, `Delete`, `Exists`, `Keys`, `Clear` | `mem.Set("k", "v", time.Minute)` |
| `ephemeral` | `NewInstance(expiration)`, `Set`, `Get`, `Del` | `e := ephemeral.NewInstance(time.Minute)` |
| `dt` | `Up(key, data)`, `Get(key).Json()`, `GetObject`, `Drop` | `dt.Up("cfg", et.Json{"a": 1})` |
| `race` | `NewValue(v)`, `Get`, `Set`, `Int`, `Increase`, `Map`, `Range` | `c := race.NewValue(0); c.Increase(1)` |

### HTTP and RPC

| Package | Public API | Example |
| --- | --- | --- |
| `router` | `NewApi(name, path, host, port, rpc, version)`, `Api.Public`, `Protected`, `Session`, `Item`, `Items`, `List`, `UseAuthentication`, `UseAuthorization`, `UseLimitRate` | `api.Protected("users", router.GET, "/users/{id}", "Get user", api.Item(200, getUser))` |
| `server` | `New(name, port)` → `Ettp` (`Mount`, `HandleFunc`, `Use`, `Start`, `Close`); `NewApi` | `srv := server.New("api", 3000); srv.Start()` |
| `ettp/v2` | `New(name, *Config)` → `Server` (`Public`, `Private`, `SetRouter`, `Use`, `Start`, `Save`, `Reset`) | `ettp.New("gateway", &ettp.Config{Port: 3300})` |
| `ettp/v1` | Older gateway: `New(*Config)`, `SetRouter`, `SetProxy`, `SetPortForward`, `PublicRoute`, `PrivateRoute` | — |
| `middleware` | `Authentication`, `Logger`, `Recoverer`, `RequestID`, `AllowAll`, `GetBearerToken`, `NewMetric` | `r.Use(middleware.Logger)` |
| `request` | Inbound: `GetBody`, `URLParam`, `Query`, `UserID`, `TenantID`; outbound: `Get`, `Post`, `Put`, `Delete`, `Fetch`, `…WithTimeout`, `…WithTls` | `body, status := request.Get(url, header)` |
| `response` | `ITEM`, `ITEMS`, `LIST`, `JSON`, `RESULT`, `HTTPError`, `Unauthorized`, `Forbidden`, `Stream` | `response.ITEM(w, r, 200, item)` |
| `jrpc` | `Mount(host, port, services, packageName)`, `Start`, `Call`, `CallJson`, `CallItem`, `CallItems` | `jrpc.CallJson("Users.Get", et.Json{"id": id})` |
| `jtcp` | `NewNode(port, tls...)` (`Mount`, `Start`, `Request`, `Broadcast`, `LeaderID`), `NewClient(addr)` (`Start`, `Request`, `Send`), TLS helpers | `node := jtcp.NewNode(1377); node.Start()` |
| `jws` | `New()` → `Hub` (`Start`, `Connect`, `Subscribe`, `Publish`, `SendTo`, `Topic`, `Queue`), `Upgrader` | `hub.Publish("orders", msg)` |

### Infrastructure

| Package | Public API | Example |
| --- | --- | --- |
| `cache` | `Load`, `Set`, `Get`, `GetStr`, `GetInt`, `GetJson`, `GetObject`, `SetObject`, `Delete`, `DeleteByPrefix`, `Incr`, `Exists`, `Expire`, `LPush`, `LRange`, `ObjetSet`/`ObjetGet`, `CollectionPut`/`CollectionFind` | `v, ok, err := cache.Get("key", "")` |
| `event` | `Load`, `Publish`, `Subscribe`, `Queue`, `Stack`, `Source`, `On`, `Unsubscribe`, `Close` | `event.Publish("user:created", data)` |
| `graph` | `Load()` → `Conn` (`HealthCheck`, `Close`) | `conn, _ := graph.Load()` |
| `queue` | `New[T](size, maxEvents, period, handler)`, `Push`, `Len`, `Close` | `q.Push(item)` |
| `crontab` | `Load(tag, store)`, `CronJob`, `ScheduleJob`, `HttpStartJob`, `HttpStopJob`, `HttpRemoveJob` | `crontab.CronJob(tag, owner, crontab.Cron{Minute: "0"}, 0, nil, fn)` |
| `resilience` | `New()`, `LoadInstance(Params)`, `Instance.Run`, `IsDone`, `IsFailed`, `Stop` | `inst, _ := r.LoadInstance(resilience.Params{Fn: send, TotalAttempts: 3})` |

### Security

| Package | Public API | Example |
| --- | --- | --- |
| `jwt` | `NewToken`, `NewAuthentication`, `NewAuthorization`, `NewAppToken`, `NewEphemeralToken`, `Validate`, `RenewToken`, `DeleteToken` | `claim, err := jwt.Validate(token)` |
| `claim` | `NewToken`, `NewAuthenticationToken`, `ParceToken`, `Claim.ToJson` | `c, _ := claim.ParceToken(token)` |
| `service` | `LoadSend(tenantId, name, email)` → `Send` (`SendOTPEmail`, `SendOTPSMS`, `VerifyOTP`, `SendEmail`, `SendSMS`, `SendWhatsapp`); `LoadLocalStorage` | `ok, _ := send.VerifyOTP(key, otp)` |
| `utility` | `Hash`/`Match` (bcrypt), `EncryptAES`/`DecryptAES`, `HashSHA256`, `GetOTP`, `ValidEmail`, `ValidPhone`, `ToBase64` | `h, _ := utility.Hash(password)` |

### Scripting, workflows and AI

| Package | Public API | Example |
| --- | --- | --- |
| `jrex` | `Load(tag, store)` → `Jrex` (`Set`, `Run`, `RunDev`, `RunModule`, `NewModule`); `NewInstance`, `NewStore` | `rt.Set("db", db).Run()` |
| `jwf` | `New(id)` → `WorkFlow` (`NewFlow`, `Run`, `GetInstance`, `RunInstance`, `StopInstance`, `OnChange`); `Flow.Step`, `Error`, `AddStep`; `LoadFlow`, `RunCode`, `Binder`, `GoValue` (a Go value for scripts); `Instance.OnLog` / `Step.OnLog` (console); script `ctx`/`params` (`set`, `put`, `view`, `omit`, `get`…) and status bindings (`statusActive`, `statusList()`…) | `wf.Run("signup", "signup", "", "", ctx, nil, true, userId)` |
| `jia` | `New(tag, store, userId)`, `Load(id, store)` → `Ia` (`SetModelAgent`, `SetContextAgent`, `SetSkillAgent`, `Conversation`, `Embed`, `LoadRouter`); `NewApiSkill` | `conv, _ := ai.Conversation(ctx, "support", to, name, prompt)` |
| `ia` | `Load(db, schema, Config)` → `Rag` (`IngestFile`, `IngestSQL`, `Ask`, `ListDocuments`, `DeleteDocument`, `ListMessages`, `LoadRouter`); `LoadIndex` → `IngestText`, `Search`, `DeleteDocumentsOf`, `DeleteProject` with an `Embedder` (`HttpEmbedder`); `ExtractFile`, `RowsText` | `rag.Ask(ctx, tenant, project, conv, user, "question")` |
| `cmds` | `NewStage`, `NewStep`, `AppendCmd`, `RunOS`, `RunSSH`, `Load(file)` | `cmds.NewStep("build", "").AppendCmd("go build ./...")` |

### Messaging

| Package | Public API | Example |
| --- | --- | --- |
| `aws` | `NewS3AWS(Params)` (`Uploader`, `UploaderB64`, `UploaderFile`, `Download`, `Delete`), `NewSenderAWS(Params).SendSMS` | `s3.Uploader(bucket, name, "image/png", data)` |
| `brevo` | `SendEmail…`, `SendSms…`, `SendWhatsapp…` (`Transactional` / `Promotional`) | `brevo.SendEmailTransactional(sender, to, subject, html, params)` |
| `infobip` | `NewSenderInfobip(Params)` (`SendSMS`, `SendEmail`, `SendWhatsApp`) | `s.SendSMS([]string{"+57..."}, "Hi {{name}}", params, "Transactional")` |
| `jwsp` | `NewSender(token, phoneId)` → `Whatsapp` (`SendTextMessage`, `SendTemplate`, `SendImageMessageByURL`, `SendDocumentByURL`, `SendList`, `Webhooks`) | `wa.SendTextMessage("57300...", "Hello")` |

### Utilities

| Package | Public API | Example |
| --- | --- | --- |
| `envar` | `GetStr`, `GetInt`, `GetInt64`, `GetBool`, `GetFloat`, `GetDuration`, `Set*`, `Arg*`, `Validate` | `port := envar.GetInt("PORT", 3000)` |
| `logs` | `Info`, `Infof`, `Debug`, `Warn`, `Alert`, `Alertf`, `Error`, `Errorf`, `Fatal`, `Panic` | `logs.Errorf("failed: %v", err)` |
| `stdrout`, `color` | `Printl`, `Traces`, `SetStdout`; `Red`, `Green`, `Yellow`, `Blue`… | `fmt.Println(color.Green("ok"))` |
| `validator` | `New()`, `Field`, `Required`, `NotEmpty`, `Min`, `Max`, `Between`, `MinLength`, `MaxLength`, `Pattern`, `Values`, `IsEmail`, `IsPhone`, `Validate`, `ByJson`, `ToJson`, `Wrapper` (goja) | `validator.New().Field("email").Required().IsEmail().Validate(body)` |
| `reg` | `UUID`, `ULID`, `XID`, `GenSnowflake`, `GetUUID(id)`, `GenULID(tag)`, `TagULID` | `id := reg.GetULID("")` |
| `strs` | `Format`, `Parse`, `Titlecase`, `Lowcase`, `RemoveAcents`, `Split`, `IsEmpty`, `HtmlToText` | `strs.Parse("Hi {{name}}", vars)` |
| `timezone` | `Now`, `NowStr`, `Add`, `Format`, `Parse`, `Location` | `t := timezone.Now()` |
| `units` | `NewQuantity(val, unit)`, `Load`, `To`, `ToStr` | `q.To(units.UnityKilometer)` |
| `xls`, `csv` | `ReadXls*` / `ReadCsv*` (`GetSheet`, `GetData`, `GetRows`); `NewXls` / `NewCsv` (`ToFile`, `ToWriter`, `ToHttp`) | `csv.NewCsv(rows).ToHttp(w, "report.csv")` |
| `file` | `Load[T]`, `Save`, `LoadOrSave[T]`, `MakeFile`, `MakeFolder`, `NewWatcher` (`OnWrite`, `OnCreate`, `Load`) | `file.Save("cfg.json", cfg)` |
| `iterate` | `Start`, `Segment`, `End` (timings) | `iterate.Start("load")` |
| `msg` | Shared message constants | — |

## Usage examples

### `et.Json`

Typed accessors walk nested keys; the `Val*` variants take a default.

```go
data := et.Json{"user": et.Json{"name": "Ana", "age": 30}}
name := data.Str("user", "name")            // "Ana"
age := data.Int("user", "age")              // 30
city := data.ValStr("Cali", "user", "city") // default when missing
data.Set("active", true)
```

### `jsql`: models and queries

A model creates its table on `Init()`. Keys without a column are stored in the JSON `_source` column and can be queried like any column.

```go
db, _ := jsql.Load() // DB_DRIVER, DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME

users, _ := db.DefineModel("public", "users", 1, userId) // id, status, created_at, updated_at, _source, _idx
users.DefineColumn("name", et.TEXT, "")
users.DefineUnique("email", et.TEXT, "")
users.DefineHidden("password")
users.Init()

users.Insert(et.Json{"id": "u1", "name": "Ana", "email": "ana@x.com", "city": "Cali"}).One()
users.Update(et.Json{"addr->zip": "76001"}).Where(jsql.Eq("id", "u1")).One()
items, _ := users.Select("id", "name", "city").Where(jsql.Eq("city", "Cali")).OrderBy("name", true).Limit(1, 20)
```

Queries and commands can also be described in JSON:

```go
items, _ := db.Query(et.Json{
    "select":   []any{"U.name", "count(O.id):orders"},
    "from":     "public.users:U",
    "left join": []any{et.Json{"to": "public.orders:O", "on": []any{et.Json{"O.user_id": et.Json{"eq": "U.id"}}}}},
    "group by": []any{"U.name"},
    "limit":    20,
})
```

Drivers: `postgres`, `sqlite` (no CGO), `oracle` 19c+, `mysql` 8.0+ and `mssql` 2022+. The full behavior is in [`jsql/spec.md`](jsql/spec.md); the public API is gathered in [`jsql/catalog.go`](jsql/catalog.go).

### HTTP API with `router`

```go
api := router.NewApi("users", "/api/users", "localhost", 3000, 4000, 1)
api.UseAuthentication(middleware.Authentication)
api.Public("version", router.GET, "/version", "Get the version", api.Item(http.StatusOK, getVersion))
api.Protected("users", router.POST, "/query", "Query users", api.List(queryUsers))
http.ListenAndServe(":3000", api.Router)
```

A plain handler uses `request` to read and `response` to write:

```go
func getUser(w http.ResponseWriter, r *http.Request) {
    id := request.URLParam(r, "id").Str()
    item, err := users.Where(jsql.Eq("id", id)).One()
    if err != nil {
        response.HTTPError(w, r, http.StatusBadRequest, err.Error())
        return
    }
    response.ITEM(w, r, http.StatusOK, item)
}
```

### Cache and events

`cache.Get` returns `(value, exists, error)`: a missing key is `def, false, nil`.

```go
cache.Load() // REDIS_HOST
cache.Set("session:u1", "token", time.Hour)
token, ok, _ := cache.Get("session:u1", "")

event.Load() // NATS_HOST
event.Subscribe("user:created", func(m event.Message) { logs.Info(m.ToString()) })
event.Publish("user:created", et.Json{"id": "u1"})
```

### Validation

```go
ok, err := validator.New().
    Field("email").Required().IsEmail().
    Field("age").Min(18).Max(120).
    Validate(body)
```

The same rules can come from JSON (`ByJson`) and be saved back with `ToJson`. `values` (`Values(...)` in Go) allows only those values; on a list, each item. An unknown condition or an invalid pattern returns an error.

```go
v, err := validator.New().ByJson(et.Json{
    "fields": et.Json{
        "email": et.Json{"required": true, "is_email": true},
        "age":   et.Json{"min": 18, "max": 120},
        "role":  et.Json{"values": []any{"admin", "user"}},
    },
})
ok, err := v.Validate(body)
```

### Batching queue

The handler runs when `maxEvents` items gather or `period` elapses; `Close()` flushes.

```go
q := queue.New[et.Json](1000, 100, 5*time.Second, func(ctx context.Context, batch []et.Json) error {
    return save(batch)
})
q.Push(et.Json{"id": 1})
defer q.Close()
```

### Workflows with `jwf`

`jwf.New` calls `cache.Load()` and `event.Load()`. The first step is the trigger; the next ones chain in order. Persist flows through `OnChange`.

```go
wf, _ := jwf.New("")
wf.NewFlow("signup", "Sign up", "1.0.0", ownerId, userId).
    Step("signup", "create user", func(i *jwf.Instance, ctx et.Json) (et.Json, error) { return ctx, nil }).
    Step("signup", "send welcome", sendWelcome)

result, err := wf.Run("signup", "signup", "", "", et.Json{"email": "ana@x.com"}, et.Json{}, true, userId)
```

`OnLog` receives each `console` line of the JavaScript steps: `{level, message, at, step_id, instance_id}`.

```go
instance, _ := wf.GetInstance("signup", "signup", "", "", et.Json{}, userId)
instance.OnLog(func(line et.Json) { logs.Info(line.ToString()) })
result, err := wf.RunInstance(instance, et.Json{"email": "ana@x.com"}, et.Json{}, true, userId)
```

### Crontab

```go
crontab.Load("billing", store) // store implements crontab.Store
crontab.CronJob("billing", ownerId, crontab.Cron{Hour: "2", Minute: "0"}, 0, et.Json{},
    func(params et.Json) error { return runBilling() })
crontab.ScheduleJob("billing", ownerId, time.Now().Add(time.Hour), et.Json{}, fn)
```

### JavaScript runtime and AI

```go
rt, _ := jrex.Load("scripts", nil) // nil store → files in ./src
rt.Set("db", jsql.Wrapper(db)).Run()

rag, _ := ia.Load(db, "public", ia.Config{}) // OPENAI_API_KEY
rag.IngestFile(ctx, tenantId, projectId, "manual.pdf", data, userId)
answer, _ := rag.Ask(ctx, tenantId, projectId, conversationId, userId, "How do I reset the router?")

// Index only (no API key): any OpenAI-compatible /embeddings endpoint
index, _ := ia.LoadIndex(db, "public", ia.Config{})
embed := ia.HttpEmbedder("http://localhost:11434/v1", "", "nomic-embed-text")
index.IngestText(ctx, ia.Document{TenantId: tenantId, ProjectId: projectId, Id: "faq-1", Name: "FAQ", Text: text}, embed)
hits, _ := index.Search(ctx, tenantId, projectId, "reset the router", 5, 0.3, embed)
```

## Environment variables

| Package | Variables |
| --- | --- |
| `jsql` | `DB_DRIVER` (`postgres`), `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_RECORD_LIMIT` (1000), `DB_IS_DEBUG`, `DB_APP_NAME`; postgres `DB_SSLMODE`; oracle `DB_SSL`, `DB_SSL_VERIFY`; every connection `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_LIFETIME`, `DB_POOL_IDLE_TIME` |
| `cache` | `REDIS_HOST`, `REDIS_PASSWORD`, `REDIS_DB` |
| `event` | `NATS_HOST`, `NATS_USER`, `NATS_PASSWORD` |
| `graph` | `NEO4J_HOST`, `NEO4J_USER`, `NEO4J_PASSWORD` |
| `claim` | `SECRET` (default `"1977"`) |
| `jia`, `ia` | `OPENAI_API_KEY` (not needed by `ia.LoadIndex`); `ia` also `IA_EMBEDDING_MODEL`, `IA_CHAT_MODEL`, `IA_CHUNK_SIZE`, `IA_CHUNK_OVERLAP`, `IA_TOP_K` |
| `dt` | `PRODUCTION` (`true` stores in Redis) |
| `jwf`, `jsql` | `MAX_AUDIT_LOG` (1000) |
| `jwsp` | `WHATSAPP_API_URL` |
| `validator` | `LANG` |

## CLI binaries

```bash
go run ./cmd/create      # scaffolding: project, microservice, model, rpc
go run ./cmd/apigateway  # API gateway (ettp)
go run ./cmd/server      # TCP node (port 1377); ./cmd/client connects to it
go run ./cmd/jrex        # JS runtime with hot-reload from ./cmd/jrex/src
go run ./cmd/ia          # RAG HTTP API (PORT, default 3300)
```

Also `cmd/et`, `cmd/daemon`, `cmd/jsql`, `cmd/install`, `cmd/whatcher`, `cmd/resilience`, `cmd/wsp` and `cmd/jwf`.

## Development

```bash
gofmt -w .
go build ./... && go vet ./...
cd jsql/drivers/test && go test -v   # jsql against every available driver
./version.sh --minor                 # bump the git tag, update the READMEs and push
```

The jsql suite skips any engine it cannot reach, so check the `SKIP` lines. `cache`, `aws` and `brevo` tests need live services.

## Documentation

More guides (in Spanish) in [`doc/`](doc/):

- [`LIBRARY_CONTEXT.md`](doc/LIBRARY_CONTEXT.md) — context for AI assistants in projects that use `et` (`go run ./cmd/install` copies it into the project).
- [`ARCHITECTURE_SUMMARY.md`](doc/ARCHITECTURE_SUMMARY.md) — layers, internal dependencies, patterns and known issues.
- [`COMPONENT_CATALOG.md`](doc/COMPONENT_CATALOG.md) — full public API, generated from `go doc`.
- [`AI_USAGE_GUIDE.md`](doc/AI_USAGE_GUIDE.md) — rules and pitfalls for AI assistants.
- [`NEW_PROJECT_GUIDE.md`](doc/NEW_PROJECT_GUIDE.md) — templates for starting a new project on `et`.

## License

MIT. See [`LICENSE`](LICENSE).
