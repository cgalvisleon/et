# ET: Librería de servicios y herramientas en Go

> [English version](../README.md)

## Instalación

Requiere Go 1.25. Agrega el módulo e importa solo los paquetes que uses:

```bash
go get github.com/cgalvisleon/et@latest
go get github.com/cgalvisleon/et@v1.0.47
```

```go
import (
    "github.com/cgalvisleon/et/et"
    "github.com/cgalvisleon/et/jsql"
    _ "github.com/cgalvisleon/et/jsql/drivers/postgres" // los drivers SQL se registran solos al importarse
)
```

## Qué es

Una librería modular para microservicios, CLIs y aplicaciones web en Go. Cada directorio es un paquete independiente; no hay un punto de entrada central. Módulo `github.com/cgalvisleon/et`, licencia MIT.

`et.Json` (`map[string]interface{}`) es la moneda común: casi toda la API lo recibe o lo devuelve.

Los paquetes de infraestructura (`cache`, `event`, `jsql`) leen su configuración de variables de entorno y se conectan una sola vez en `Load()`; las llamadas siguientes no hacen nada.

## Actualizar a v1.0.47

- `router.Publish` / `router.With` ya no anteponen `Route.PackageName` a `Route.Path`: pon la ruta completa en la ruta.
- Los modelos de `jsql` exigen un esquema. En SQLite, MySQL y Oracle la tabla se llama `schema_nombre` (antes SQLite usaba el nombre a secas), así que las bases SQLite existentes necesitan renombrar sus tablas.
- `cache.Get*` devuelven `(valor, existe, error)`; `cache.ErrNotFound` ya no existe.
- `jwf.New(id)` recibe solo el id; los flujos se persisten mediante `OnChange`.
- Scripts de jsql (goja): `where` de un modelo, consulta o comando recibe una condición en JSON o una lista (`[{status: {eq: "active"}}, {and: {age: {more: 18}}}]`); ya no hay `and`/`or`. Sirve un objeto a secas (`{status: {eq: "active"}}`) o dentro de `where`.
- `router.Api.Public` / `Session` / `Protected` reciben `group` primero.

## Cambios después de v1.0.47

- Las condiciones se evalúan en memoria como en SQL: `AND` liga más que `OR`, y una condición sin conector después de la primera es un `AND` (antes se ignoraba en memoria). Revisa los filtros en memoria que dependían del orden de izquierda a derecha.
- `like` en memoria acepta `%` y `_` y no distingue mayúsculas; `*` sigue valiendo como `%`.
- `et.Wheres.AdddResult` queda obsoleto: usa `AddResult`.
- Las claves del límite de peticiones de `router` son `ratelimit:<MÉTODO>:<ruta>`; las claves viejas sin TTL se pueden borrar a mano.
- `jsql` tiene un sexto driver, `josefina`, que se conecta al protocolo Postgres de Josefina.

## Paquetes

Cada tabla lista la API pública principal de cada paquete. `go doc github.com/cgalvisleon/et/<paquete>` muestra la lista completa.

### Datos base

| Paquete | API pública | Ejemplo |
| --- | --- | --- |
| `et` | `Json` (`Str`, `Int`, `Bool`, `Num`, `Time`, `Json`, `ArrayStr`, `ArrayJson`, `ValStr`, `ValInt`, `Set`, `Put`, `Clear`, `Get`, `Update`, `Clone`, `DeepClone`, `ToString`, `ToObject`), `ToJS`/`FromJS`/`JsonOf`/`GoArg` (un `Json` en scripts de goja: un objeto sobre el mismo mapa, claves como propiedades y sus métodos con su nombre de Go), `Item`, `Items`, `List`, condiciones (`Eq`, `More`, `In`, `Between`, `ToCondition`, `AddCondition`, `EvaluateObject`, `EvaluateValue`, `Fld` (una referencia a un campo, `{field: "nombre"}` en JSON)), `From(items).Where(...).All()` | `data.ValStr("Cali", "user", "city")` |
| `jsql` | `Load`, `LoadTo`, `ConnectTo`, `CreateDB`, `DropDB`, `TestConnection`; `DB.Define`, `DefineModel`, `Query`, `Sql`; `Model.Insert`, `Update`, `Upsert`, `Delete`, `Bulk`, `Select`, `Where`, `WhereByJson`; `Query.All`, `One`, `Limit`, `Count`; `NewTx`, `DefineSeries` (`GenSerie`, `GenSerieTx`, `GenValue`, `GenValueTx`); `Bind(source, vars)` (variables `{{name}}`); `Wrapper`, `BindWrapper`, `ToJsMapWrapper` (goja) | `users.Where(jsql.Eq("id", id)).One()` |
| `stores` | `DefineInstance` (`Set`, `Get`, `Delete`, `Query`), `DefineAuthorization` (`SetAuthor`, `Author`, `RemoveAuthor`), `DefineConfig` (`SetConfig`, `Get`), `DefineFormState` | `ok, _ := auth.Author(profileId, "GET", "/users")` |
| `mem` | `Set`, `Get`, `GetStr`, `GetInt`, `GetJson`, `Delete`, `Exists`, `Keys`, `Clear` | `mem.Set("k", "v", time.Minute)` |
| `ephemeral` | `NewInstance(expiration)`, `Set`, `Get`, `Del` | `e := ephemeral.NewInstance(time.Minute)` |
| `dt` | `Up(key, data)`, `Get(key).Json()`, `GetObject`, `Drop` | `dt.Up("cfg", et.Json{"a": 1})` |
| `race` | `NewValue(v)`, `Get`, `Set`, `Int`, `Increase`, `Map`, `Range` | `c := race.NewValue(0); c.Increase(1)` |

### HTTP y RPC

| Paquete | API pública | Ejemplo |
| --- | --- | --- |
| `router` | `NewApi(name, path, host, port, rpc, version)`, `Api.Public`, `Protected`, `Session`, `Item`, `Items`, `List`, `UseAuthentication`, `UseAuthorization`, `UseLimitRate` | `api.Protected("users", router.GET, "/users/{id}", "Get user", api.Item(200, getUser))` |
| `server` | `New(name, port)` → `Ettp` (`Mount`, `HandleFunc`, `Use`, `Start`, `Close`); `NewApi` | `srv := server.New("api", 3000); srv.Start()` |
| `ettp/v2` | `New(name, *Config)` → `Server` (`Public`, `Private`, `SetRouter`, `Use`, `Start`, `Save`, `Reset`) | `ettp.New("gateway", &ettp.Config{Port: 3300})` |
| `ettp/v1` | Gateway anterior: `New(*Config)`, `SetRouter`, `SetProxy`, `SetPortForward`, `PublicRoute`, `PrivateRoute` | — |
| `middleware` | `Authentication`, `Logger`, `Recoverer`, `RequestID`, `AllowAll`, `GetBearerToken`, `NewMetric` | `r.Use(middleware.Logger)` |
| `request` | Entrada: `GetBody`, `URLParam`, `Query`, `UserID`, `TenantID`; salida: `Get`, `Post`, `Put`, `Delete`, `Fetch`, `…WithTimeout`, `…WithTls` | `body, status := request.Get(url, header)` |
| `response` | `ITEM`, `ITEMS`, `LIST`, `JSON`, `RESULT`, `HTTPError`, `Unauthorized`, `Forbidden`, `Stream` | `response.ITEM(w, r, 200, item)` |
| `jrpc` | `Mount(host, port, services, packageName)`, `Start`, `Call`, `CallJson`, `CallItem`, `CallItems` | `jrpc.CallJson("Users.Get", et.Json{"id": id})` |
| `tcp` | `NewServer(id, name)` / `NewClient(id, name)` (`Listen`, `ConnectTo`, `Send`, `Request`, streams), `NewHub(id, name)` (`LoadServer`, `ConnectTo`, `Send`, `SendTo`, `Heartbeat`), `SetSecurity`, `NewTLS` | `srv, _ := tcp.NewServer("", "api"); srv.Listen(":1377")` |
| `websocket` | `NewServer()` (tópicos, colas, `OnConnect`/`OnSubscribe` que pueden rechazar), `NewClient`/`Connect` → `Client` (`Subscribe`, `Publish`) | `srv.Publish("orders", msg)` |

### Infraestructura

| Paquete | API pública | Ejemplo |
| --- | --- | --- |
| `cache` | `Load`, `Set`, `Get`, `GetStr`, `GetInt`, `GetJson`, `GetObject`, `SetObject`, `Delete`, `DeleteByPrefix`, `Incr`, `Exists`, `Expire`, `LPush`, `LRange`, `ObjectSet`/`ObjectGet`, `CollectionPut`/`CollectionFind` | `v, ok, err := cache.Get("key", "")` |
| `event` | `Load`, `Publish`, `Subscribe`, `Queue`, `Stack`, `Source`, `On`, `Unsubscribe`, `Close` | `event.Publish("user:created", data)` |
| `graph` | `Load()` → `Conn` (`HealthCheck`, `Close`) | `conn, _ := graph.Load()` |
| `queue` | `New[T](size, maxEvents, period, handler)`, `Push`, `Len`, `Close` | `q.Push(item)` |
| `crontab` | `Load(tag, store)`, `CronJob`, `ScheduleJob`, `HttpStartJob`, `HttpStopJob`, `HttpRemoveJob` | `crontab.CronJob(tag, owner, "0 * * * *", 0, nil, fn)` |
| `resilience` | `New()`, `LoadInstance(Params)`, `Instance.Run`, `IsDone`, `IsFailed`, `Stop` | `inst, _ := r.LoadInstance(resilience.Params{Fn: send, TotalAttempts: 3})` |
| `store` | `Open(pathData, pathWal, name, mode)` → `FileStore` (`Insert`, `Update`, `Delete`, `Get`, `GetVersion`, `ForEach`, `Keys`, `Deferred`/`WaitDurable`, `Compact`, `Sync`, `WalSince`/`ApplyWalEntry`, `Stats`), `Recover` | `fs, _ := store.Open("./data", "./wal", "customers", store.ReadWrite)` |

### Seguridad

| Paquete | API pública | Ejemplo |
| --- | --- | --- |
| `jwt` | `NewToken`, `NewAuthentication`, `NewAuthorization`, `NewAppToken`, `NewEphemeralToken`, `Validate`, `RenewToken`, `DeleteToken` | `claim, err := jwt.Validate(token)` |
| `claim` | `NewToken`, `NewAuthenticationToken`, `ParseToken`, `Claim.ToJson` | `c, _ := claim.ParseToken(token)` |
| `service` | `LoadSend(tenantId, name, email)` → `Send` (`SendOTPEmail`, `SendOTPSMS`, `VerifyOTP`, `SendEmail`, `SendSMS`, `SendWhatsapp`); `LoadLocalStorage` | `ok, _ := send.VerifyOTP(key, otp)` |
| `utility` | `Hash`/`Match` (bcrypt), `EncryptAES`/`DecryptAES`, `HashSHA256`, `GetOTP`, `ValidEmail`, `ValidPhone`, `ToBase64` | `h, _ := utility.Hash(password)` |

### Scripting, workflows e IA

| Paquete | API pública | Ejemplo |
| --- | --- | --- |
| `jwf` | `New(id)` → `WorkFlow` (`NewFlow`, `Run`, `GetInstance`, `RunInstance`, `StopInstance`, `OnChange`); `Flow.Step`, `Error`, `AddStep`; `LoadFlow`, `Run` (código, bindings, lecturas → último valor y las variables leídas), `RunCode`, `RunScripts`, `Bindings` (los bindings estándar: console, fetch, cache, event, jrpc…), `Binder`, `GoValue` (un valor de Go para los scripts); `Instance.OnLog` / `Step.OnLog` (consola); `ctx`/`params` en los scripts (`set`, `put`, `view`, `omit`, `get`…) y bindings de estado (`statusActive`, `statusList()`…); `evaluateObject`/`evaluateValue` y referencias a campos (`field`, `sum`, `year`…) | `wf.Run("signup", "signup", "", "", ctx, nil, true, userId)` |
| `jia` | `New(tag, store, userId)`, `Load(id, store)` → `Ia` (`SetModelAgent`, `SetContextAgent`, `SetSkillAgent`, `Conversation`, `Embed`, `LoadRouter`); `NewApiSkill` | `conv, _ := ai.Conversation(ctx, "support", to, name, prompt)` |
| `ia` | `Load(db, schema, Config)` → `Rag` (`IngestFile`, `IngestSQL`, `Ask`, `ListDocuments`, `DeleteDocument`, `ListMessages`, `LoadRouter`); `LoadIndex` → `IngestText`, `Search`, `DeleteDocumentsOf`, `DeleteProject` con un `Embedder` (`HttpEmbedder`); `ExtractFile`, `RowsText` | `rag.Ask(ctx, tenant, project, conv, user, "question")` |
| `cmds` | `NewStage`, `NewStep`, `AppendCmd`, `RunOS`, `RunSSH`, `Load(file)` | `cmds.NewStep("build", "").AppendCmd("go build ./...")` |

### Mensajería

| Paquete | API pública | Ejemplo |
| --- | --- | --- |
| `aws` | `NewS3AWS(Params)` (`Uploader`, `UploaderB64`, `UploaderFile`, `Download`, `Delete`; `UploadStream`, `List` y `Open` para archivos grandes en flujo; `Params.Endpoint`/`PathStyle` para servicios compatibles con S3 como MinIO o SeaweedFS), `NewSenderAWS(Params).SendSMS` | `s3.Uploader(bucket, name, "image/png", data)` |
| `brevo` | `SendEmail…`, `SendSms…`, `SendWhatsapp…` (`Transactional` / `Promotional`) | `brevo.SendEmailTransactional(sender, to, subject, html, params)` |
| `infobip` | `NewSenderInfobip(Params)` (`SendSMS`, `SendEmail`, `SendWhatsApp`) | `s.SendSMS([]string{"+57..."}, "Hi {{name}}", params, "Transactional")` |
| `jwsp` | `NewSender(token, phoneId)` → `Whatsapp` (`SendTextMessage`, `SendTemplate`, `SendImageMessageByURL`, `SendDocumentByURL`, `SendList`, `Webhooks`) | `wa.SendTextMessage("57300...", "Hello")` |

### Utilidades

| Paquete | API pública | Ejemplo |
| --- | --- | --- |
| `envar` | `GetStr`, `GetInt`, `GetInt64`, `GetBool`, `GetFloat`, `GetDuration`, `Set*`, `Arg*`, `Validate` | `port := envar.GetInt("PORT", 3000)` |
| `logs` | `Info`, `Infof`, `Debug`, `Warn`, `Alert`, `Alertf`, `Error`, `Errorf`, `Fatal`, `Panic` | `logs.Errorf("failed: %v", err)` |
| `stdrout`, `color` | `Printl`, `Traces`, `SetStdout`; `Red`, `Green`, `Yellow`, `Blue`… | `fmt.Println(color.Green("ok"))` |
| `validator` | `New()`, `Field`, `Required`, `NotEmpty`, `Min`, `Max`, `Between`, `MinLength`, `MaxLength`, `Pattern`, `Values`, `IsEmail`, `IsPhone`, `Validate`, `ByJson`, `ToJson`, `Wrapper` (goja) | `validator.New().Field("email").Required().IsEmail().Validate(body)` |
| `reg` | `UUID` (v7, ordenable), `ULID`, `XID`, `GenSnowflake`, `GetUUID(id)`, `GenULID(tag)`, `TagULID` | `id := reg.GetULID("")` |
| `strs` | `Format`, `Parse`, `Titlecase`, `Lowcase`, `RemoveAccents`, `Split`, `IsEmpty`, `HtmlToText` | `strs.Parse("Hi {{name}}", vars)` |
| `timezone` | `Now`, `NowStr`, `Add`, `Format`, `Parse`, `Location` | `t := timezone.Now()` |
| `units` | `NewQuantity(val, unit)`, `Load`, `To`, `ToStr` | `q.To(units.UnityKilometer)` |
| `xls`, `csv` | `ReadXls*` / `ReadCsv*` (`GetSheet`, `GetData`, `GetRows`); `NewXls` / `NewCsv` (`ToFile`, `ToWriter`, `ToHttp`) | `csv.NewCsv(rows).ToHttp(w, "report.csv")` |
| `file` | `Load[T]`, `Save`, `LoadOrSave[T]`, `MakeFile`, `MakeFolder`, `NewWatcher` (`OnWrite`, `OnCreate`, `Load`) | `file.Save("cfg.json", cfg)` |
| `iterate` | `Start`, `Segment`, `End` (tiempos) | `iterate.Start("load")` |
| `msg` | Constantes de mensajes compartidas | — |

## Ejemplos de uso

### `et.Json`

Los accesores tipados recorren claves anidadas; las variantes `Val*` reciben un valor por defecto.

```go
data := et.Json{"user": et.Json{"name": "Ana", "age": 30}}
name := data.Str("user", "name")            // "Ana"
age := data.Int("user", "age")              // 30
city := data.ValStr("Cali", "user", "city") // valor por defecto si no existe
data.Set("active", true)
```

El texto JSON se lee de forma tolerante: `et.ToJson`, `Scan`, `Json(clave)`, `Array`/`ArrayJson` también aceptan saltos de línea reales dentro de los textos, texto escapado una vez más (`{\"a\":1}`) o codificado dos veces. Los números conservan su forma: `Str` da `"1"` para un entero y `"3.1"` para `3.10` (sin decimales agregados ni perdidos), los enteros mayores que 2^53 se mantienen exactos (`int64`), y `ArrayInt`, `MapInt`… leen cualquier número o texto numérico. `Bool` acepta `true`/`false`, `1`/`0` y los textos `yes`/`no`, `si`/`no`, `on`/`off`.

```go
j, _ := et.ToJson("{\"note\": \"línea 1\nlínea 2\", \"id\": 9007199254740993, \"price\": 3.10, \"active\": \"1\"}")
j.Str("id")      // "9007199254740993"
j.Str("price")   // "3.1"
j.Bool("active") // true
```

### `jsql`: modelos y consultas

Un modelo crea su tabla en `Init()`. Las claves sin columna se guardan en la columna JSON `_source` y se pueden consultar como cualquier columna.

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

Las consultas y los comandos también se pueden describir en JSON:

```go
items, _ := db.Query(et.Json{
    "select":   []any{"U.name", "count(O.id):orders"},
    "from":     "public.users:U",
    "left join": []any{et.Json{"to": "public.orders:O", "on": []any{et.Json{"O.user_id": et.Json{"eq": "U.id"}}}}},
    "group by": []any{"U.name"},
    "limit":    20,
})
```

Drivers: `postgres`, `sqlite` (sin CGO), `oracle` 19c+, `mysql` 8.0+, `mssql` 2022+ y `josefina` (protocolo Postgres de Josefina; no crea ni elimina bases, y guarda los atributos de un modelo en la raíz del documento en lugar de en `_source`). El comportamiento completo está en [`jsql/spec.md`](../jsql/spec.md); la API pública está reunida en [`jsql/catalog.go`](../jsql/catalog.go).

### API HTTP con `router`

```go
api := router.NewApi("users", "/api/users", "localhost", 3000, 4000, 1)
api.UseAuthentication(middleware.Authentication)
api.Public("version", router.GET, "/version", "Get the version", api.Item(http.StatusOK, getVersion))
api.Protected("users", router.POST, "/query", "Query users", api.List(queryUsers))
http.ListenAndServe(":3000", api.Router)
```

Un handler simple usa `request` para leer y `response` para escribir:

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

### Caché y eventos

`cache.Get` devuelve `(valor, existe, error)`: una clave inexistente es `def, false, nil`.

```go
cache.Load() // REDIS_HOST
cache.Set("session:u1", "token", time.Hour)
token, ok, _ := cache.Get("session:u1", "")

event.Load() // NATS_HOST
event.Subscribe("user:created", func(m event.Message) { logs.Info(m.ToString()) })
event.Publish("user:created", et.Json{"id": "u1"})
```

### Condiciones en memoria y en SQL

**Un where da las mismas filas en memoria que en la base.** `et.EvaluateObject`, `et.EvaluateValue` y `et.From(...).Where(...)` siguen a SQL: `AND` liga más que `OR`, una condición sin conector después de la primera es un `AND`, los enteros se comparan exactos (también IDs más allá de 2^53), `like` acepta `%` y `_` sin distinguir mayúsculas, y `between` acepta `[min, max]` o `{min, max}`. Las reglas están en `et/spec.md`.

```go
conds, _ := et.ToConditions([]et.Json{
    {"name": et.Json{"like": "%maria%"}},
    {"or": et.Json{"vip": et.Json{"eq": true}}},
    {"and": et.Json{"age": et.Json{"between": []any{18, 65}}}},
})
ok := et.EvaluateObject(row, conds) // name LIKE '%maria%' OR (vip AND age BETWEEN 18 AND 65)
```

### Condiciones con referencias a campos

**Comparar contra otro campo, no contra un literal.** Un valor `{"field": "nombre"}` en una condición nombra un campo: en memoria es otro atributo del objeto evaluado (una agregación como `sum(items->price)` se aplica sobre él); en una consulta de jsql es una columna de la consulta, también en `havings`.

```go
conds, _ := et.ToConditions([]et.Json{
    {"spent": et.Json{"more": et.Json{"field": "budget"}}},
    {"or": et.Json{"total": et.Json{"less": et.Json{"field": "sum(items->price)"}}}},
})
ok := et.EvaluateObject(row, conds)

over, _ := budgets.Query(et.Json{
    "selects": []any{"A.id"},
    "where":   []any{et.Json{"A.spent": et.Json{"more": et.Json{"field": "A.budget"}}}},
}).All()
```

En los scripts de jwf y jsql, `field`, `sum`, `count`, `year`… arman la referencia:

```js
evaluateObject(params, [{age: {more_eq: field("min_age")}}, {and: {total: {eq: sum("items->price")}}}])
```

### Validación

```go
ok, err := validator.New().
    Field("email").Required().IsEmail().
    Field("age").Min(18).Max(120).
    Validate(body)
```

Las mismas reglas pueden venir de un JSON (`ByJson`) y guardarse de vuelta con `ToJson`. `values` (`Values(...)` en Go) admite solo esos valores; en una lista, cada elemento. Una condición desconocida o un patrón inválido devuelven un error.

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

### Cola por lotes

El handler se ejecuta cuando se juntan `maxEvents` elementos o pasa `period`; `Close()` vacía la cola.

```go
q := queue.New[et.Json](1000, 100, 5*time.Second, func(ctx context.Context, batch []et.Json) error {
    return save(batch)
})
q.Push(et.Json{"id": 1})
defer q.Close()
```

### Store embebido clave/valor

`store` es un store de un solo nodo, de solo agregado, que guarda `id → []byte` en archivos de segmento con un índice en memoria; `Open` se repara solo después de una caída. No hay upsert: se llama `Update` y, si el id no existía, `Insert`. Formato y garantías: `store/spec.md`.

```go
fs, err := store.Open("./data/collections", "./data/wal", "customers", store.ReadWrite)
if err != nil {
    return err
}
defer fs.Close()

ok, err := fs.Insert("c-001", []byte(`{"name":"Ana"}`)) // false si ya existía
ok, err = fs.Update("c-001", []byte(`{"name":"Ana María"}`))
data, exists, err := fs.Get("c-001")
err = fs.ForEach(func(id string, data []byte) (bool, error) {
    return true, nil // false detiene el recorrido
}, true, 0, 100)
```

### Workflows con `jwf`

`jwf.New` llama a `cache.Load()` y `event.Load()`. El primer paso es el trigger; los siguientes se encadenan en orden. Persiste los flujos mediante `OnChange`.

```go
wf, _ := jwf.New("")
wf.NewFlow("signup", "Sign up", "1.0.0", ownerId, userId).
    Step("signup", "create user", func(i *jwf.Instance, ctx et.Json) (et.Json, error) { return ctx, nil }).
    Step("signup", "send welcome", sendWelcome)

result, err := wf.Run("signup", "signup", "", "", et.Json{"email": "ana@x.com"}, et.Json{}, true, userId)
```

`OnLog` recibe cada línea de `console` de los pasos JavaScript: `{level, message, at, step_id, instance_id}`.

```go
instance, _ := wf.GetInstance("signup", "signup", "", "", et.Json{}, userId)
instance.OnLog(func(line et.Json) { logs.Info(line.ToString()) })
result, err := wf.RunInstance(instance, et.Json{"email": "ana@x.com"}, et.Json{}, true, userId)
```

### Crontab

```go
crontab.Load("billing", store) // store implementa crontab.Store
job, err := crontab.CronJob("billing", ownerId, "0 2 * * *", 0, et.Json{},
    func(params et.Json) error { return runBilling() }) // id = tag; registrarlo de nuevo lo reemplaza
crontab.ScheduleJob("billing:once", ownerId, time.Now().Add(time.Hour), et.Json{}, fn)
crontab.RemoveJob("billing")
```

### Scripts JavaScript e IA

```go
bindings := jwf.Bindings(jsql.Wrapper(db)) // console, fetch, cache… más db, jsql
bindings["NEW"] = record                    // un et.Json: NEW.name = "x" cambia record
result, globals, err := jwf.Run(`NEW.code = NEW.Str("name").toUpperCase(); NEW`, bindings, "NEW")

rag, _ := ia.Load(db, "public", ia.Config{}) // OPENAI_API_KEY
rag.IngestFile(ctx, tenantId, projectId, "manual.pdf", data, userId)
answer, _ := rag.Ask(ctx, tenantId, projectId, conversationId, userId, "How do I reset the router?")

// Solo el índice (sin clave de API): cualquier endpoint /embeddings compatible con OpenAI
index, _ := ia.LoadIndex(db, "public", ia.Config{})
embed := ia.HttpEmbedder("http://localhost:11434/v1", "", "nomic-embed-text")
index.IngestText(ctx, ia.Document{TenantId: tenantId, ProjectId: projectId, Id: "faq-1", Name: "FAQ", Text: text}, embed)
hits, _ := index.Search(ctx, tenantId, projectId, "reset the router", 5, 0.3, embed)
```

## Variables de entorno

| Paquete | Variables |
| --- | --- |
| `jsql` | `DB_DRIVER` (`postgres`), `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_RECORD_LIMIT` (1000), `DB_IS_DEBUG`, `DB_APP_NAME`; postgres `DB_SSLMODE`; oracle `DB_SSL`, `DB_SSL_VERIFY`; toda conexión `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_LIFETIME`, `DB_POOL_IDLE_TIME` |
| `cache` | `REDIS_HOST`, `REDIS_PASSWORD`, `REDIS_DB` |
| `event` | `NATS_HOST`, `NATS_USER`, `NATS_PASSWORD` |
| `graph` | `NEO4J_HOST`, `NEO4J_USER`, `NEO4J_PASSWORD` |
| `claim` | `SECRET` (por defecto `"1977"`) |
| `jia`, `ia` | `OPENAI_API_KEY` (`ia.LoadIndex` no la necesita); `ia` además `IA_EMBEDDING_MODEL`, `IA_CHAT_MODEL`, `IA_CHUNK_SIZE`, `IA_CHUNK_OVERLAP`, `IA_TOP_K` |
| `dt` | `PRODUCTION` (`true` guarda en Redis) |
| `jwf`, `jsql` | `MAX_AUDIT_LOG` (1000) |
| `jwsp` | `WHATSAPP_API_URL` |
| `store` | `RELSEG_SIZE` (MB por segmento, 128), `SYNC_ON_WRITE` (true), `MIN_THRESHOLD_COMPACT` (1000) |
| `validator` | `LANG` |

## Binarios CLI

```bash
go run ./cmd/create      # scaffolding: proyecto, microservicio, modelo, rpc
go run ./cmd/apigateway  # API gateway (ettp)
go run ./cmd/server      # nodo TCP (puerto 1377); ./cmd/client se conecta a él
go run ./cmd/ia          # API HTTP de RAG (PORT, por defecto 3300)
```

También `cmd/et`, `cmd/daemon`, `cmd/jsql`, `cmd/install`, `cmd/watcher`, `cmd/resilience`, `cmd/wsp` y `cmd/jwf`.

## Desarrollo

```bash
gofmt -w .
go build ./... && go vet ./...
go test -race ./et/ ./jsql/ ./jwf/   # núcleo, helpers de jsql y scripts (sin servicios)
cd jsql/drivers/test && go test -v   # jsql contra cada driver disponible
go run ./test/storebench -d 10s      # carga sobre store (escritores, lectores)
./version.sh --minor                 # sube el tag de git, actualiza los README y hace push
```

La suite de jsql omite cualquier motor al que no pueda conectarse, así que revisa las líneas `SKIP`. Las pruebas de `cache`, `aws` y `brevo` necesitan servicios en vivo. Cada paquete con spec (`et/spec.md`, `jsql/spec.md`, `store/spec.md`, `tcp/spec.md`, `websocket/spec.md`) se actualiza en el mismo cambio que su código.

## Documentación

Guías adicionales (en español) en [`doc/`](.):

- [`LIBRARY_CONTEXT.md`](LIBRARY_CONTEXT.md) — contexto para asistentes de IA en proyectos que usan `et` (`go run ./cmd/install` lo copia al proyecto).
- [`ARCHITECTURE_SUMMARY.md`](ARCHITECTURE_SUMMARY.md) — capas, dependencias internas, patrones y problemas conocidos.
- [`COMPONENT_CATALOG.md`](COMPONENT_CATALOG.md) — API pública completa, generada desde `go doc`.
- [`AI_USAGE_GUIDE.md`](AI_USAGE_GUIDE.md) — reglas y trampas para asistentes de IA.
- [`NEW_PROJECT_GUIDE.md`](NEW_PROJECT_GUIDE.md) — plantillas para arrancar un proyecto nuevo sobre `et`.

## Licencia

MIT. Ver [`LICENSE`](../LICENSE).
