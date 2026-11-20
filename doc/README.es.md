# ET: Librería de servicios y herramientas en Go

> [English version](../README.md)

## Instalación

Requiere Go 1.25. Agrega el módulo e importa solo los paquetes que uses:

```bash
go get github.com/cgalvisleon/et@latest
go get github.com/cgalvisleon/et@v1.0.52
```

Para crear un servicio nuevo, corre el generador dentro de una carpeta que ya tenga su `go.mod` (`go mod init <módulo>`); pregunta qué crear (Project, Microservice, Modelo, Rpc) y al final corre `go mod tidy`:

```bash
go run github.com/cgalvisleon/et/cmd/create@latest go
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

## Actualizar a v1.0.52

- `router.Publish` / `router.With` ya no anteponen `Route.PackageName` a `Route.Path`: pon la ruta completa en la ruta.
- Los modelos de `jsql` exigen un esquema. En SQLite, MySQL y Oracle la tabla se llama `schema_nombre` (antes SQLite usaba el nombre a secas), así que las bases SQLite existentes necesitan renombrar sus tablas.
- `cache.Get*` devuelven `(valor, existe, error)`; `cache.ErrNotFound` ya no existe.
- `jwf.New(id)` recibe solo el id; los flujos se persisten mediante `OnChange`.
- Scripts de jsql (goja): `where` de un modelo, consulta o comando recibe una condición en JSON o una lista (`[{status: {eq: "active"}}, {and: {age: {more: 18}}}]`); ya no hay `and`/`or`. Sirve un objeto a secas (`{status: {eq: "active"}}`) o dentro de `where`.
- `router.Api.Public` / `Session` / `Protected` reciben `group` primero.

## Cambios después de v1.0.52

- Las condiciones se evalúan en memoria como en SQL: `AND` liga más que `OR`, y una condición sin conector después de la primera es un `AND` (antes se ignoraba en memoria). Revisa los filtros en memoria que dependían del orden de izquierda a derecha.
- `like` en memoria acepta `%` y `_` y no distingue mayúsculas; `*` sigue valiendo como `%`.
- `et.Wheres.AdddResult` queda obsoleto: usa `AddResult`.
- Las claves del límite de peticiones de `router` son `ratelimit:<MÉTODO>:<ruta>`; las claves viejas sin TTL se pueden borrar a mano.
- `jsql` tiene un sexto driver, `josefina`, que se conecta al protocolo Postgres de Josefina.
- `db.Query` con `from` (y el `to` de los joins) alcanza cualquier tabla de la base: un modelo que este proceso no definió se lee del catálogo de la base cuando la referencia trae esquema (`schema.model`), para consultas y para `insert`/`update`/`delete`/`upsert`/`bulk`.
- `db.Query` corre programas: `{"steps": {"const", "vars", "do": [...]}}` ejecuta una lista de descriptores en una transacción con `{{variables}}`, `as`, `set`, `if`/`then`/`else`, `for` al estilo de Go (range, contador, while, `break`, `continue`, con tope), subconsultas `{"query": {...}}` como valores, `script` (JavaScript) y `return`/`error`.
- Un `select` con dos campos que dan el mismo nombre (`C.name`, `P.name`) ahora falla en lugar de que un valor pise al otro sin avisar: nombra uno con `campo:alias`.
- Búsqueda de texto: un modelo declara `searches` (`{"name": "fts", "fields": ["name", "tags"], "lang": "simple"}`, o `model.DefineSearch`) y un where pide `{"fts": {"search": "bomba agua -diesel"}}` (palabras, `"frase"`, `-excluida`, `or`); `"orders": [{"fts": false}]` ordena por relevancia. Postgres usa un índice GIN de `tsvector` y `ts_rank`, MySQL un índice `FULLTEXT` y `MATCH`; los demás motores buscan con `LIKE`. El índice se crea también en una tabla que ya existe. `"indexes": [{"name": "name", "method": "trgm"}]` agrega un índice `pg_trgm` para `like '%x%'` en Postgres. Detalle en `jsql/spec.md` §6.1.
- `full_text`, la búsqueda de un modelo: `"full_text": ["code", "name", "description", "tags"]`, `"full_text_lang": "es"`, `"full_text_langs": ["en"]` (o `model.DefineFullText(fields...)`, `DefineFullTextLang(lang, langs...)`), y una consulta JSQL (la consulta descrita en JSON) pide `"search": "bomba agua"` o `"search": {"text": "pumps", "lang": "en"}` (`Query.Search(texto, lang...)`, `.search(texto, lang)` en scripts): se une con `AND` a todo el where y, sin `orders`, ordena por relevancia. Sin `full_text` la consulta falla con `MSG_FULLTEXT_UNDEFINED`. El idioma es un código ISO (`es` → `spanish`, `en` → `english`…; uno desconocido es `simple`), también en el operador: `{"fts": {"search": {"text": "…", "lang": "en"}}}`.
- **Cambio de comportamiento:** el idioma por omisión de una búsqueda es `spanish` (era `simple`; `DB_SEARCH_LANG` lo cambia). Postgres construye un índice GIN por idioma declarado, los crea y borra con `CREATE/DROP INDEX CONCURRENTLY` fuera de transacción (sin bloquear escrituras), rehace uno inválido y borra los índices de búsqueda que hizo jsql y que ya no usa ninguna búsqueda; MySQL usa `ALGORITHM=INPLACE, LOCK=SHARED` (las escrituras esperan mientras se construye un `FULLTEXT`). `DB_SEARCH_INDEXES=off` lo apaga al iniciar y `db.ApplySearchIndexes(model)` los aplica a mano.

## Paquetes

Cada tabla lista la API pública principal de cada paquete. `go doc github.com/cgalvisleon/et/<paquete>` muestra la lista completa.

### Datos base

| Paquete | API pública | Ejemplo |
| --- | --- | --- |
| `et` | `Json` (`Str`, `Int`, `Bool`, `Num`, `Time`, `Json`, `ArrayStr`, `ArrayJson`, `ValStr`, `ValInt`, `Set`, `Put`, `Clear`, `Get`, `Update`, `Clone`, `DeepClone`, `ToString`, `ToObject`), `ToJS`/`FromJS`/`JsonOf`/`GoArg` (un `Json` en scripts de goja: un objeto sobre el mismo mapa, claves como propiedades y sus métodos con su nombre de Go), `Item`, `Items`, `List`, condiciones (`Eq`, `More`, `In`, `Between`, `ToCondition`, `AddCondition`, `EvaluateObject`, `EvaluateValue`, `Fld` (una referencia a un campo, `{field: "nombre"}` en JSON)), `From(items).Where(...).All()` | `data.ValStr("Cali", "user", "city")` |
| `jsql` | `Load`, `LoadTo`, `ConnectTo`, `CreateDB`, `DropDB`, `TestConnection`; `DB.Define`, `DefineModel`, `Query` (JSQL, consultas y comandos en JSON: consultas, comandos, `define`, y `drop`/`truncate`/`create`/`alter` de modelos, schemas, bases y usuarios), `Sql`; `Model.Insert`, `Update`, `Upsert`, `Delete`, `Bulk`, `Select`, `Where`, `WhereByJson`; `Query.All`, `One`, `Limit`, `Count`; `NewTx`, `DefineSeries` (`GenSerie`, `GenSerieTx`, `GenValue`, `GenValueTx`); `Bind(source, vars)` (variables `{{name}}`); `Wrapper`, `BindWrapper`, `ToJsMapWrapper` (goja) | `users.Where(jsql.Eq("id", id)).One()` |
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
| `middleware` | `Authentication`, `Logger`, `Recoverer`, `RequestID`, `AllowAll`, `GetBearerToken`, `NewMetric`, `SetTelemetryOutput` | `r.Use(middleware.Logger)` |
| `request` | Entrada: `GetBody`, `URLParam`, `Query`, `UserID`, `TenantID`; salida: `Get`, `Post`, `Put`, `Delete`, `Fetch`, `…WithTimeout`, `…WithTls` | `body, status := request.Get(url, header)` |
| `response` | `ITEM`, `ITEMS`, `LIST`, `JSON`, `RESULT`, `HTTPError`, `Unauthorized`, `Forbidden`, `Stream` | `response.ITEM(w, r, 200, item)` |
| `jrpc` | `Mount(host, port, services, packageName)`, `Start`, `Call`, `CallJson`, `CallItem`, `CallItems` | `jrpc.CallJson("Users.Get", et.Json{"id": id})` |
| `tcp` | `NewServer(id, name)` / `NewClient(id, name)` (`Listen`, `ConnectTo`, `Send`, `Request`, streams), `NewHub(id, name)` (`LoadServer`, `ConnectTo`, `Send`, `SendTo`, `Heartbeat`), `SetSecurity`, `NewTLS` | `srv, _ := tcp.NewServer("", "api"); srv.Listen(":1377")` |
| `websocket` | `NewServer()` (tópicos, colas, `OnConnect`/`OnSubscribe` que pueden rechazar), `NewClient`/`Connect` → `Client` (`Subscribe`, `Publish`) | `srv.Publish("orders", msg)` |

### Infraestructura

| Paquete | API pública | Ejemplo |
| --- | --- | --- |
| `cache` | `Load`, `Set`, `Get`, `GetStr`, `GetInt`, `GetJson`, `GetObject`, `SetObject`, `Delete`, `DeleteByPrefix`, `Incr`, `Exists`, `Expire`, `LPush`, `LRange`, `ObjectSet`/`ObjectGet`, `CollectionPut`/`CollectionFind` | `v, ok, err := cache.Get("key", "")` |
| `event` | `Load`, `Publish`, `Subscribe`, `Queue`, `Stack`, `Source`, `On`, `Unsubscribe`, `SubscribeWith`, `QueueWith`, `StackWith` (por clave: varios oyentes por canal), `UnsubscribeKey`, `HasKey`, `Close` | `event.Publish("user:created", data)` |
| `graph` | `Load()` → `Conn` (`HealthCheck`, `Close`) | `conn, _ := graph.Load()` |
| `queue` | `New[T](size, maxEvents, period, handler)`, `Push`, `Len`, `Close` | `q.Push(item)` |
| `crontab` | `New(tag, store, opts...)` (instancia; `WithHandler`) o `Load` (global), `CronJob`, `ScheduleJob`, `StopJob`, `StartJob`, `RemoveJob`, `GetJob`, `Http*`; estados `pending`→`scheduled`⇄`running`→`finished`/`failed`, `paused`, `missed`, `canceled`; `LastRun`, `WithRetries`, `WithTimeout`; las réplicas con el mismo tag eligen una líder y la relevan | `crontab.CronJob(tag, owner, "0 * * * *", 0, nil, fn)` |
| `resilience` | `New()`, `LoadInstance(Params)`, `Instance.Run`, `IsDone`, `IsFailed`, `Stop` | `inst, _ := r.LoadInstance(resilience.Params{Fn: send, TotalAttempts: 3})` |
| `store` | `Open(dir)` → `Engine` (`Store`, `DropStore`, `Stores`, `Checkpoint`, `Vacuum`, `NewBatch`, `Snapshot`, `SnapshotFor`, `WaitDurable`, `Verify`, `Repair`, `Stats`, `OnStats`, `Pressure`, `DirUsage`, `StoreIn`, `DropStoreIn`, `Backup`, `PendingArchive`, `SetArchived`, `SwitchWAL`, `WalSince`, `ApplyWal`, `ReplayLSN`, `CreateSlot`, `CreateLogicalSlot`, `DropSlot`, `Slots`, `AdvanceSlot`, `Changes`, `Close`; `OpenAt(dir, lsn)`, `OpenReplica(dir)`) → `Store` (`Name`, `Put`, `Remove`, `Insert`, `Update`, `Delete`, `Read`, `GetVersion`, `Exists`, `Count`, `Empty`, `Deferred`, `DirUsage`, `Keys`, `KeysAfter`, `KeysPrefix`, `ForEach`, `ForEachPrefix`, `ForEachRange`, `ForEachBorrowed`, sus formas `…In(lote, …)`, `ReadAt`/`GetVersionAt`/`ExistsAt`/`ForEachAt`/`CountAt(snapshot, …)`); `Batch` (`Commit`, `CommitDeferred`, `Rollback`); motor al estilo PostgreSQL con MVCC (F-23) | `s, _ := e.Store("users"); s.Put("id", data)` |

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
| `reg` | `UUID` (v7, ordenable), `ULID`, `XID`, `SID` (11 caracteres, ordenable; nodo de `SID_NODE`), `SIDInt`, `EncodeSID`/`DecodeSID`/`SIDParts`, `NewSIDGenerator(nodo)`, `GenSnowflake` (el SID en 19 dígitos), `GetUUID(id)`, `GenULID(tag)`, `TagULID` | `id := reg.SID() // "0S2ZTw7OCm0"` |
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

Las consultas y los comandos también se pueden describir en JSON (el formato **JSQL**: una consulta JSQL, un comando JSQL):

```go
items, _ := db.Query(et.Json{
    "select":   []any{"U.name", "count(O.id):orders"},
    "from":     "public.users:U",
    "left join": []any{et.Json{"to": "public.orders:O", "on": []any{et.Json{"O.user_id": et.Json{"eq": "U.id"}}}}},
    "group by": []any{"U.name"},
    "limit":    20,
})
```

`from` y el `to` de los joins pueden nombrar cualquier tabla de la base, definida en este proceso o no: con esquema (`"sales.clients:C"`), una tabla que el proceso no definió se lee del catálogo de la base, con los atributos de `_source`, y también puede ser el destino de `insert`, `update`, `delete`, `upsert` y `bulk`. Dos campos seleccionados no pueden dar el mismo nombre en el resultado; usa `campo:alias` (`"P.name:product"`).

Varios descriptores pueden correr como un programa, en una transacción (ver [`jsql/spec.md`](../jsql/spec.md) §8.4):

```go
items, _ := db.Query(et.Json{"steps": et.Json{
    "vars": et.Json{"client": "c1"},
    "do": []any{
        et.Json{"as": "orders", "select": []any{"id", "total"}, "from": "sales.orders", "where": []any{et.Json{"client_id": et.Json{"eq": "{{client}}"}}}},
        et.Json{"for": et.Json{"range": "{{orders}}", "value": "o"}, "do": []any{
            et.Json{"if": et.Json{"data": "{{o}}", "where": []any{et.Json{"total": et.Json{"eq": 0}}}}, "then": []any{et.Json{"continue": true}}},
            et.Json{"update": et.Json{"from": "sales.orders", "data": et.Json{"state": "billed"}, "where": []any{et.Json{"id": et.Json{"eq": "{{o.id}}"}}}}},
        }},
        et.Json{"return": et.Json{"orders": "{{orders}}"}},
    },
}})
```

Drivers: `postgres`, `sqlite` (sin CGO), `oracle` 19c+, `mysql` 8.0+, `mssql` 2022+ y `josefina` (protocolo Postgres de Josefina; no crea bases, y guarda los atributos de un modelo en la raíz del documento en lugar de en `_source`). El comportamiento completo está en [`jsql/spec.md`](../jsql/spec.md); la API pública está reunida en [`jsql/catalog.go`](../jsql/catalog.go).

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

### Telemetría de peticiones y Grafana

`middleware.Logger` (lo usan `router` y `server`) escribe una línea por petición en stdout. `TELEMETRY_FORMAT` elige cuál: `text` (la línea con colores), `json` (un objeto JSON por petición, para Loki/Grafana) u `off`; sin definir, es `text` en una terminal y `json` en otro caso (un contenedor). `TELEMETRY_LOG=errors` escribe solo las peticiones fallidas o lentas (estado ≥ 500 o latencia ≥ `LATENCY_LIMIT` ms; por defecto `all`). `SetTelemetryOutput(w, format, filter)` cambia el destino, el formato y el filtro en tiempo de ejecución (vacío = la variable). El query string nunca se escribe. Con `off` y sin NATS el middleware no hace nada (sin métrica ni cabecera `ServiceId`); los contadores de `cache.CallMetrics` solo corren para la línea de texto, NATS o `NewMetric` (el 429 de `ettp`).

```json
{"time":"2026-10-29T15:04:05.123-05:00","level":"INFO","msg":"http.request","http.request.method":"GET","http.route":"/users/{id}","url.path":"/users/123","url.scheme":"http","http.response.status_code":200,"duration_ms":12.84,"response_time_ms":11.9,"search_time_ms":0,"http.request.body.size":0,"http.response.body.size":2048,"client.address":"10.0.0.4","server.address":"pod-1","user_agent.original":"curl/8.7.1","trace_id":"","service_id":"…","app_name":"…"}
```

`level` es `ERROR` en un 5xx y `WARN` en un 4xx o con una latencia mayor a `LATENCY_LIMIT` ms. `http.route` es el patrón de la ruta (el de chi, o el que pone `ettp`), así que no crece con los ids. Grafana Alloy o Promtail llevan el stdout del contenedor a Loki; deja como etiquetas de Loki solo el servicio y `level`, y consulta el resto desde el JSON (el parser `json` cambia los puntos por `_`):

```logql
sum by (http_route) (rate({service="tenants"} | json [1m]))                                        # peticiones por segundo
quantile_over_time(0.95, {service="tenants"} | json | unwrap duration_ms [5m]) by (http_route)     # latencia p95
sum(rate({service="tenants"} | json | http_response_status_code >= 500 [5m]))                      # 5xx por segundo
```

Con NATS conectado se siguen publicando los eventos `telemetry`, `telemetry:log` y `telemetry:overflow`.

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

`store` es un motor de almacenamiento para construir bases de datos sobre él, tomando como referencia el almacenamiento de PostgreSQL (F-23): un árbol B+ en disco en páginas de 8 KB, una caché de páginas de tamaño fijo (nada del índice vive en RAM), un WAL por página con imágenes enteras tras cada checkpoint, MVCC con ids de transacción de 64 bits, checkpoints en segundo plano y una recuperación que solo reproduce el WAL desde el último checkpoint. Un motor es un directorio (un proceso a la vez) con varios stores de `(id string, data []byte)`; los ids miden de 1 a 2.000 bytes y los valores hasta 64 MiB (los grandes van a páginas de desborde). Diseño, formatos y orden de los bloqueos: [`store/doc.go`](../store/doc.go).

**Operaciones sueltas.** `Put` y `Remove` son ciegos; `Insert`, `Update` y `Delete` devuelven si escribieron; `Read`, `Exists` (sin leer el valor), `Count` (un contador, O(1)) y `Empty` (O(1), como `TRUNCATE`; `ErrBusy` mientras haya lotes o snapshots abiertos). Cada una es su propia transacción: lee la versión más nueva confirmada (Read Committed), espera hasta `LOCK_TIMEOUT` a otro escritor del mismo id (después `ErrConflict`) y, con `SYNC_ON_WRITE` (por omisión), vuelve cuando el WAL está en disco; los escritores comparten cada fsync.

**Lotes y snapshots.** Un lote (`NewBatch`, los métodos `…In`, `Commit`/`Rollback`) es atómico entre stores, ve sus propios cambios y lee de su snapshot (Repeatable Read: escribir un id que otro confirmó después de que empezó el lote da `ErrSerialization`; se reintenta el lote). Un `Snapshot` da lecturas consistentes con `ReadAt`, `ExistsAt`, `GetVersionAt`, `ForEachAt` y `CountAt` (O(n)). Los dos terminan al pasar `TRANSACTION_TIMEOUT` (`ErrTimeout`). Las versiones viejas se limpian en segundo plano (`VACUUM_INTERVAL`, `VACUUM_THRESHOLD`) o con `Vacuum()`, que además borra las páginas del árbol que quedan vacías (como el nbtree de PostgreSQL) y las devuelve al espacio libre: con una carga que reemplaza y borra, el disco se estabiliza en unas 1,5 × los datos vivos (`Stats.PagesDeleted`, `Stats.PagesRecycled`).

**Recorridos.** Siguen las hojas en orden de id, ascendente o descendente, con offset y limit, por prefijo o rango `[from, to)`: `ForEach`, `ForEachPrefix`, `ForEachRange`, `Keys`, `KeysPrefix` y `KeysAfter` (paginar) toman un snapshot propio; las formas `…At` usan el tuyo. El callback no tiene nada tomado mientras corre (puede usar el motor), `ForEachBorrowed` presta `data` solo dentro del callback, y los recorridos grandes usan un anillo de búferes en vez de vaciar la caché.

**Escritura diferida y versiones.** `store.Deferred()` (`Put`, `Remove`, `Insert`, `Update`, `Delete`) y `batch.CommitDeferred()` confirman sin esperar el fsync, como `synchronous_commit = off` de PostgreSQL: se ven enseguida y un corte puede perderlas enteras (nunca a medias) hasta que `engine.WaitDurable()` vuelva. Un escritor de fondo del WAL las hace durables igual en `WAL_WRITER_DELAY`, y `SYNC_ON_WRITE=false` difiere toda confirmación. `GetVersion`, `GetVersionIn` y `GetVersionAt` devuelven el valor con su versión (el LSN que lo escribió, 0 si no existe); la escritura condicional es un lote, que da `ErrSerialization` si el id cambió después de que empezó.

**DDL transaccional.** `engine.StoreIn(tx, name)` abre un store para un lote o lo crea dentro del lote, y `engine.DropStoreIn(tx, name)` lo borra cuando el lote confirma, así un catálogo y sus stores nacen o mueren juntos. Hasta el `Commit` solo el lote ve el store que creó (`Store(name)` desde fuera espera hasta `LOCK_TIMEOUT` y después da `ErrStorePending`). `Rollback`, el tiempo máximo o un corte no dejan rastro. Un lote con DDL siempre espera el fsync, también con `CommitDeferred`.

**Revisión y reparación.** `engine.Verify()` revisa el motor abierto sin cambiar nada (control, catálogo, archivos huérfanos y, de a un store, su árbol, sus cadenas de desborde, la lista de libres, el mapa de visibilidad, el congelado y `Count`) y devuelve un `Report`. `engine.Repair()` además arregla, con WAL, lo que se puede sin perder nada sano (archivos huérfanos, páginas sin dueño, cadenas huérfanas, bits del mapa mal puestos, un `Count` equivocado, un borrado de páginas cortado por una caída) e informa lo demás. `go run ./cmd/storecheck -dir <dir> [-repair]` imprime el informe.

**Estadísticas, presión y uso de disco.** `engine.Stats()` es una foto barata de contadores atómicos: caché, WAL, checkpoints, limpieza, lotes y snapshots abiertos, XID, lecturas, escrituras, confirmaciones, estado del disco, del archivo continuo, la presión y latencias p50/p99 de confirmaciones, lecturas puntuales y fsync del WAL. `engine.OnStats(intervalo, fn)` manda una en cada intervalo hasta su `stop` o `Close`. `engine.Pressure()` dice qué tan cerca está el motor de frenar a los escritores (0 = nada, ≥ 1 = ya frena): la mayor de cuatro razones (WAL sin reciclar, WAL que espera al archivador, registros del WAL sin fsync, páginas sucias), o 1 con el motor de solo lectura o inutilizable; `Stats.PressureCause` la nombra. `engine.DirUsage()` y `store.DirUsage()` dan los bytes en disco, del largo de los archivos.

**Disco lleno o lento.** Los segmentos del WAL se preasignan con ceros y los archivos de datos crecen escribiendo antes la página nueva, así el disco lleno casi siempre sale antes de escribir nada: `ErrDiskFull` (se compara con `errors.Is`), nada cambió y las escrituras vuelven solas cuando hay lugar. Donde ya no puede volver atrás (un fsync fallido) el motor queda de solo lectura: las lecturas siguen, las escrituras devuelven `ErrDiskFull` con `ErrFailed`, y `Close` más `Open` recupera todo lo confirmado. Un disco lento frena a los escritores en vez de crecer: con más de 2 × `MAX_WAL_SIZE` de WAL sin reciclar esperan a un checkpoint (después `ErrThrottled`), y las confirmaciones diferidas esperan un fsync cuando hay 8.192 registros sin fsync.

**Respaldo en caliente y restauración.** `engine.Backup(dir)` copia el motor abierto a un directorio vacío mientras siguen las escrituras, los lotes y el DDL, como `pg_basebackup`, y `store.Open(dir)` restaura la copia: comprueba primero que el WAL copiado llegue al fin del respaldo (si no, `ErrBackupIncomplete`) y después lo rehace, así la copia tiene exactamente lo confirmado al terminar el respaldo. `BackupInfo` informa los LSN, los archivos, los bytes y la duración.

**Archivo continuo del WAL y recuperación a un punto en el tiempo.** Con `WAL_ARCHIVE=on` ningún segmento sellado del WAL se recicla hasta que tu archivador llame a `engine.SetArchived(seg)`; `engine.PendingArchive()` lista los sellados sin archivar (con `Size`, los bytes que vale la pena copiar) y `engine.SwitchWAL()` sella antes el activo. Un archivador atrasado frena a los escritores, salvo que `WAL_ARCHIVE_MAX_SIZE` ponga tope al WAL retenido (el corte se ve en `Stats.ArchiveGap`). Una copia de `Backup` más los segmentos archivados copiados a su `wal/` se restaura hasta el final de lo archivado con `Open`, o hasta un LSN con `store.OpenAt(dir, lsn)` (`ErrRecoveryTarget` si no se puede alcanzar).

**Replicación.** Una réplica empieza de una copia de `Backup` abierta con `store.OpenReplica(dir)`: es de solo lectura para el usuario (`ErrReadOnly`) y aplica el WAL del primario byte a byte, `primary.WalSince(replica.ReplayLSN())` → `Next` → `replica.ApplyWal`, después `replica.WaitDurable()` y `primary.AdvanceSlot(nombre, replica.ReplayLSN())`. El transporte es tuyo. Una ranura (`CreateSlot`, antes del respaldo) retiene el WAL que la réplica todavía necesita hasta `MAX_SLOT_WAL_KEEP_SIZE` (después se invalida y la réplica necesita un respaldo nuevo; `Stats` y `Pressure` muestran ranuras, atraso y WAL retenido). Para un flujo de cambios por id, `CreateLogicalSlot` + `engine.Changes(ranura)` entrega cada transacción confirmada en orden de confirmación (put, delete, create, drop, empty), nunca una abortada; las transacciones grandes pasan a disco. `Open` sobre el directorio de una réplica la promueve.

**Configuración.** Las variables de entorno (tamaños en MB y tiempos en segundos, salvo que se diga otra cosa) están en la fila `store` de [Variables de entorno](#variables-de-entorno); se leen al abrir el motor.

```go
engine, err := store.Open("./data")
if err != nil {
    return err
}
defer engine.Close() // checkpoint final

customers, err := engine.Store("customers") // lo abre o lo crea
err = customers.Put("c-001", []byte(`{"name":"Ana"}`))
data, found, err := customers.Read("c-001")
created, err := customers.Insert("c-002", []byte(`{"name":"Luis"}`)) // false si c-002 ya existe
changed, err := customers.Update("c-001", []byte(`{"name":"Ana M."}`)) // false si no existe o no cambió
deleted, err := customers.Delete("c-002")                             // false si no existe
err = customers.Remove("c-003")                                       // borrado ciego
n := customers.Count()

// Un lote: atómico entre stores, un fsync en Commit
orders, _ := engine.Store("orders")
tx := engine.NewBatch()
_ = customers.PutIn(tx, "c-001", []byte(`{"name":"Ana","orders":1}`))
_, _ = orders.InsertIn(tx, "o-001", []byte(`{"customer":"c-001"}`))
if err := tx.Commit(); err != nil { // ErrSerialization: reintentar el lote
    return err
}

// Un snapshot: lecturas consistentes mientras otros escriben
view := engine.Snapshot()
defer view.Close()
data, found, err = customers.ReadAt(view, "c-001")

// Recorridos: ids en orden sobre un snapshot; devolver false corta
err = customers.ForEach(func(id string, data []byte) (bool, error) {
    fmt.Println(id, string(data))
    return true, nil
}, true, 0, 100) // ascendente, offset 0, hasta 100
page, err := customers.KeysAfter("", 50) // página siguiente: KeysAfter(page[len(page)-1], 50)
exists, err := customers.Exists("c-001")

// Escritura diferida: sin fsync por escritura; un WaitDurable para todo el grupo
deferred := customers.Deferred()
for _, ev := range events {
    _ = deferred.Put(ev.ID, ev.Data) // visible ya, durable tras WaitDurable
}
err = engine.WaitDurable()

// Versiones: el LSN de la versión visible (0 si no existe)
data, version, found, err := customers.GetVersion("c-001")

// DDL transaccional: un store creado y llenado en un lote, y uno viejo borrado con él
tx = engine.NewBatch()
v2, err := engine.StoreIn(tx, "customers_v2") // solo tx lo ve hasta el Commit
_ = v2.PutIn(tx, "c-001", []byte(`{"name":"Ana"}`))
_ = engine.DropStoreIn(tx, "orders") // se borra cuando tx confirma
if err := tx.Commit(); err != nil { // con error o Rollback: sin customers_v2, orders intacto
    return err
}

// Verify (solo lectura) o Repair (arregla lo que puede, seguro ante un corte); el daño va en el informe
report, err := engine.Verify()
fmt.Print(report.String()) // totales y una línea por hallazgo
if !report.Clean() {
    report, err = engine.Repair() // report.Pending(): lo que queda
}

// Stats: una foto barata; OnStats manda una en cada intervalo hasta stop() o Close
st := engine.Stats()
fmt.Println(st.CacheDirty, st.WALSinceCheckpoint, st.CommitLatency.P99, st.FsyncLatency.P50)
stop := engine.OnStats(10*time.Second, func(st store.Stats) { log.Printf("reads %d, commits %d", st.Reads, st.Commits) })
defer stop()

// Presión y uso de disco: frenar antes de una carga masiva si el motor está cerca de frenar
if engine.Pressure() >= 0.8 {
    time.Sleep(100 * time.Millisecond)
}
usage, _ := engine.DirUsage()
fmt.Println(usage.Total, usage.WAL, usage.Stores["customers"])

// Respaldo en caliente: las escrituras siguen mientras copia; Open sobre la copia la restaura
info, err := engine.Backup("/backups/2026-11-20")
if err != nil {
    return err
}
fmt.Println(info.EndLSN, info.Bytes, info.Took)
restored, err := store.Open("/backups/2026-11-20") // rehace su WAL hasta el fin del respaldo

// Archivador (WAL_ARCHIVE=on): copiar cada segmento sellado y después marcarlo archivado
segments, err := engine.PendingArchive()
for _, seg := range segments {
    if err := copyFile(seg.Path, filepath.Join("/archive", seg.Name), seg.Size); err != nil {
        break // se reintenta en la próxima vuelta: SetArchived solo con la copia a salvo
    }
    _ = engine.SetArchived(seg.Segment)
}
// A un punto en el tiempo: la copia del respaldo más los segmentos archivados en su wal/, hasta un LSN
pitr, err := store.OpenAt("/backups/2026-11-20", targetLSN)

// Replicación: primero la ranura, después el respaldo del que parte la réplica
_, _ = primary.CreateSlot("replica-1")
_, _ = primary.Backup("/replica")
replica, err := store.OpenReplica("/replica")
reader, _ := primary.WalSince(replica.ReplayLSN())
for {
    entries, err := reader.Next(1 << 20) // solo lo durable; ErrWalRemoved = hace falta un respaldo nuevo
    if err != nil || len(entries) == 0 {
        break // se vuelve a pedir después
    }
    _ = replica.ApplyWal(entries) // en la vida real viaja por tu transporte
}
_ = replica.WaitDurable()
_ = primary.AdvanceSlot("replica-1", replica.ReplayLSN())

// Flujo lógico de cambios: transacciones confirmadas, en orden de confirmación
_, _ = primary.CreateLogicalSlot("search-index")
feed, _ := primary.Changes("search-index")
_, err = feed.Next(0, func(tx store.TxInfo, c store.Change) error {
    fmt.Println(tx.CommitLSN, c.Op, c.Store, c.ID, len(c.Data))
    return nil
})
_ = feed.Confirm() // el próximo lector de la ranura sigue desde acá
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

// Una instancia propia (varias por proceso); las réplicas (procesos) con el mismo tag eligen líder a la más antigua,
// que programa todos los jobs; otra la releva si se detiene. El Store recarga los jobs al arrancar.
c, _ := crontab.New("reports", store)
defer c.Close()
c.CronJob("daily", ownerId, "0 6 * * *", 0, nil, fn, // error o panic de fn → last_run "failed"
    crontab.WithRetries(3, time.Minute))             // hasta 3 reintentos; registrarlo de nuevo conserva su estado
c.StopJob("daily")                                    // → paused; StartJob → scheduled
job, _ := c.GetJob("daily")                           // job.Str("status"), job.Json("last_run")

// Un handler para todos los jobs de la instancia, suscrito antes de que pueda liderar: los jobs recargados del Store
// (y el disparo de catch-up de un schedule atrasado) se atienden sin registrarlos otra vez. fn nil lo usa; sin ninguno, error.
c, _ = crontab.New("engine:"+tenantId, store, crontab.WithHandler(func(params et.Json) error { return run(params) }))
c.ScheduleJob("once", ownerId, at, et.Json{"job": "once"}, nil) // una fecha pasada solo vale si ya existe ese mismo job
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
| `jsql` | `DB_DRIVER` (`postgres`), `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_RECORD_LIMIT` (1000), `DB_IS_DEBUG`, `DB_APP_NAME`; postgres `DB_SSLMODE`; oracle `DB_SSL`, `DB_SSL_VERIFY`; toda conexión `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_LIFETIME`, `DB_POOL_IDLE_TIME`; búsqueda `DB_SEARCH_LANG` (`spanish`), `DB_SEARCH_INDEXES` (`on`) |
| `cache` | `REDIS_HOST`, `REDIS_PASSWORD`, `REDIS_DB` |
| `event` | `NATS_HOST`, `NATS_USER`, `NATS_PASSWORD` |
| `graph` | `NEO4J_HOST`, `NEO4J_USER`, `NEO4J_PASSWORD` |
| `claim` | `SECRET` (por defecto `"1977"`) |
| `jia`, `ia` | `OPENAI_API_KEY` (`ia.LoadIndex` no la necesita); `ia` además `IA_EMBEDDING_MODEL`, `IA_CHAT_MODEL`, `IA_CHUNK_SIZE`, `IA_CHUNK_OVERLAP`, `IA_TOP_K` |
| `dt` | `PRODUCTION` (`true` guarda en Redis) |
| `jwf`, `jsql` | `MAX_AUDIT_LOG` (1000) |
| `jwsp` | `WHATSAPP_API_URL` |
| `store` | `STORE_CACHE_SIZE` (MB, 128), `WAL_SEGMENT_SIZE` (MB, 128), `SYNC_ON_WRITE` (true), `CHECKPOINT_TIMEOUT` (s, 300), `MAX_WAL_SIZE` (MB, 1024), `LOCK_TIMEOUT` (s, 5), `TRANSACTION_TIMEOUT` (s, 600; 0 = sin tope), `VACUUM_INTERVAL` (s, 60; 0 = apagada), `VACUUM_THRESHOLD` (escrituras, 1000), `WAL_WRITER_DELAY` (ms, 200; 0 = apagado), `WAL_ARCHIVE` (`off`; `on`/`true`/`1`/`yes` lo activan), `WAL_ARCHIVE_MAX_SIZE` (MB, 0 = sin tope: frena), `MAX_SLOT_WAL_KEEP_SIZE` (MB, 4096; 0 = sin tope: frena) |
| `validator` | `LANG` |
| `middleware` | `TELEMETRY_FORMAT` (`text` en una terminal, `json` en otro caso, `off`), `TELEMETRY_LOG` (`all`, `errors`), `LATENCY_LIMIT` (1000 ms, `WARN` por encima), `REQUESTS_LIMIT` (400 por segundo por método y ruta) |

## Binarios CLI

```bash
go run ./cmd/create      # scaffolding: proyecto, microservicio, modelo, rpc
go run ./cmd/apigateway  # API gateway (ettp)
go run ./cmd/server      # nodo TCP (puerto 1377); ./cmd/client se conecta a él
go run ./cmd/ia          # API HTTP de RAG (PORT, por defecto 3300)
go run ./cmd/store       # ejemplo de store: Open, Store, Put, Read, Insert, Update, Delete, Remove, batch, snapshot, Vacuum, Close
go run ./cmd/storecheck -dir <dir> [-repair]  # revisa (o repara) un motor de store e imprime el informe
```

También `cmd/et`, `cmd/daemon`, `cmd/jsql`, `cmd/install`, `cmd/watcher`, `cmd/resilience`, `cmd/wsp` y `cmd/jwf`.

## Desarrollo

```bash
gofmt -w .
go build ./... && go vet ./...
go test -race ./et/ ./jsql/ ./jwf/   # núcleo, helpers de jsql y scripts (sin servicios)
cd jsql/drivers/test && go test -v   # jsql contra cada driver disponible
go test -p 1 ./store/...             # el motor store, sin servicios (un binario de prueba a la vez)
./version.sh --minor                 # sube el tag de git, actualiza los README y hace push
```

La suite de jsql omite cualquier motor al que no pueda conectarse, así que revisa las líneas `SKIP`. Las pruebas de `cache`, `aws` y `brevo` necesitan servicios en vivo. Cada paquete con spec (`et/spec.md`, `jsql/spec.md`, `tcp/spec.md`, `websocket/spec.md`) se actualiza en el mismo cambio que su código.

## Documentación

Guías adicionales (en español) en [`doc/`](.):

- [`LIBRARY_CONTEXT.md`](LIBRARY_CONTEXT.md) — contexto para asistentes de IA en proyectos que usan `et` (`go run ./cmd/install` lo copia al proyecto).
- [`ARCHITECTURE_SUMMARY.md`](ARCHITECTURE_SUMMARY.md) — capas, dependencias internas, patrones y problemas conocidos.
- [`COMPONENT_CATALOG.md`](COMPONENT_CATALOG.md) — API pública completa, generada desde `go doc`.
- [`AI_USAGE_GUIDE.md`](AI_USAGE_GUIDE.md) — reglas y trampas para asistentes de IA.
- [`NEW_PROJECT_GUIDE.md`](NEW_PROJECT_GUIDE.md) — plantillas para arrancar un proyecto nuevo sobre `et`.
- [`CHANGELOG.md`](CHANGELOG.md) — historial de arreglos: qué cambió, cuándo y cómo se probó.

## Licencia

MIT. Ver [`LICENSE`](../LICENSE).
