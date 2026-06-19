# COMPONENT_CATALOG.md

> Catálogo exhaustivo de componentes de `github.com/cgalvisleon/et` (Go 1.25). Cada entrada cita archivo:línea real al momento de generar este documento — el repo cambia con frecuencia (ver advertencia en `LIBRARY_CONTEXT.md`), así que verifica contra el código antes de depender de una firma exacta en una decisión crítica. Para la narrativa de arquitectura, ver `ARCHITECTURE_SUMMARY.md`; para reglas de uso por IA, ver `AI_USAGE_GUIDE.md`.

---

## Índice

1. Núcleo de datos: `et/`
2. Persistencia: `jsql/`, `stores/`, `dt/`
3. HTTP: `server/`, `ettp/v1`, `ettp/v2`, `router/`, `middleware/`, `response/`, `request/`, `ws/`
4. Validación: `jval/`
5. Infraestructura: `cache/`, `event/`, `graph/`
6. Configuración/entorno/log: `config/`, `envar/`, `logs/`, `stdrout/`, `color/`
7. Identidad/seguridad: `claim/`, `jwt/`, `reg/`, `utility/`, `strs/`
8. Orquestación: `crontab/`, `jwf/`, `resilience/`, `ia/`, `jrex/`, `service/`
9. Comunicación de bajo nivel: `jrpc/`, `tcp/`
10. Integraciones externas: `aws/`, `brevo/`, `wsp/`
11. Concurrencia/memoria: `mem/`, `ephemeral/`, `race/`, `iterate/`
12. Tiempo/unidades/archivos: `timezone/`, `units/`, `file/`
13. Herramientas de desarrollo: `cmds/`, `create/`, `cmd/*`, `jcli/`
14. Patrón transversal `msg/`

---

## 1. Núcleo de datos: `et/`

**Descripción:** tipo de datos universal (`Json`) y wrappers de resultado (`List`, `Item`, `Items`) usados en todas las capas.

**API pública (`et/json.go`):**
- `func (s Json) ToByte() ([]byte, error)` — :116
- `func (s Json) ToString() string` — :129
- `func (s Json) ToEscapeHTML() string` — :142
- `func (s Json) ToMap() map[string]interface{}` — :161
- `func (s Json) IsEmpty() bool` — :169
- `func (s Json) IsExist(key string) bool` — :178
- `func (s Json) Clone() Json` — :187
- `func (s Json) ValAny/ValStr/ValInt/ValInt64/ValNum/ValBool/ValTime/ValJson/ValArray(def, atribs ...string) ...` — :198-538 (patrón "valor con default + ruta anidada")
- `func (s Json) Str/String/Int/Int64/Num/Bool/Time/Json(atrib)/Byte` (atribs ...string) — :547-655
- `func (s Json) MapStr/MapInt/MapFloat(atrib string) map[string]...` — :664-712
- `func (s Json) Array/ArrayBytes/ArrayStr/ArrayInt/ArrayInt64/ArrayNumber/ArrayJson(atribs ...string)` — :712-849
- `func (s Json) Update(from Json)` — :849 (mezcla in-place)
- `func (s Json) Compare(from Json) Json` — :858 (diff)
- `func (s Json) Append(from Json) Json` — :873
- `func (s Json) IsChanged(from Json) bool` — :888
- `func (s Json) IsDeferent(atrib string, val interface{}) bool` — :917
- `func (s Json) Get(keys ...string) (result interface{})` — :930
- `func (j Json) SetNested(keys []string, value interface{})` — :965
- `func (s Json) Set(key string, val interface{})` — :995
- `func (s Json) Delete(keys []string) bool` — :1014
- `func (s Json) Exist(key string) bool` — :1035
- `func (s Json) Remove(keys ...string)` — :1044
- `func (s Json) Select(keys []string) Json` — :1055
- `func (s Json) Hidden(keys []string) Json` — :1072

**`et/list.go`:** `List{Rows, All, Count, Page, Start, End, Result []Json}`.

**`et/item.go` / `et/items.go`:** `Item{Ok bool, Result Json}` con los mismos accesores tipados; `Items` con `Add`, `AddMany`, `One(idx)`, `First`, `Last`, `ToList(all, page, rows)`.

**Ejemplo:**
```go
data := et.Json{"user": et.Json{"name": "Ana", "age": 30}}
name := data.Str("user", "name")
age := data.ValInt(0, "user", "age")
```

**Notas:** es el tipo más reutilizado de todo el repo — virtualmente todo paquete depende de `et`.

---

## 2. Persistencia

### 2.1 `jsql/`

**Descripción:** SQL builder agnóstico de motor + ORM ligero con esquema híbrido relacional/JSONB.

**Entrada (`jsql/jsql.go`):**
- `func Load(tenantId string) (*DB, error)` — :85
- `func LoadTo(tenantId, name string) (*DB, error)` — :72
- `func ConnectTo(connect Connection) (*DB, error)` — :45
- `func GetDb(name string) (*DB, error)` — :98
- `func GetModel(db, schema, name string) (*Model, error)` — :111
- `func Define(dbName string, def Def) (*Model, error)` — :187
- `func Insert/Update/Delete/Upsert(model *Model, data et.Json) *Command` — :151-180

**Drivers (`jsql/driver.go`):**
- Constantes: `DriverPostgres`, `DriverSqlite`, `DriverMysql`, `DriverMssql`, `DriverOracle`, `DriverJosefina` — :9-14
- `type Driver interface { Connect(db *DB) (*sql.DB, error); Load(model *Model) (string, error); Query(query *Query) (string, error); Command(command *Command) (string, error) }` — :20
- `func Register(name string, driver Driver)` — :39
- **Solo `jsql/drivers/postgres/` tiene archivos.** `jsql/drivers/mysql/` y `jsql/drivers/josefina/` existen vacíos; **no existe `jsql/drivers/sqlite/`** aunque `DriverSqlite` y `sqliteConection()` (`jsql/conection.go:98`) sigan en el código.

**Definición de modelo (`jsql/define.go`, `jsql/db.go`):**
- `func (s *DB) DefineModel(schema, name string, version int) (*Model, error)` — `jsql/define.go:461`
- `func (s *DB) NewModel(schema, name string, version int) (*Model, error)` — `jsql/db.go:251`
- `func (s *DB) Define(define Def) (*Model, error)` — `jsql/db.go:351`
- `type Def struct { Schema, Name string; Version int; IdxField, IdtField string; PrimaryKeys, ForeignKeys, Indexes, Unique, Required []DefIndex/DefForeignKeys; Columns []Column; SourceField string; Hiddens []string; Details map[string]DefDetail; Rollups map[string]DefRollup; IsCore, IsDebug, IsTest bool }` — `jsql/define.go:43`

**Tipos de columna (`TypeColumn`):** `COLUMN`, `ATTRIB`, `DETAIL`, `ROLLUP`, `CALCFUNC`, `CALC`, `AGG`.
**Tipos de dato (`TypeData`):** `KEY`, `TEXT`, `MEMO`, `INT`, `FLOAT`, `BOOLEAN`, `DATETIME`, `JSON`, `BYTES`, `GEOMETRY`, `EMBEDDING`, `ANY`.
**Constantes de columna (`jsql/column.go`):** `ID`, `IDX` (`_idx`), `IDT` (`_idt`), `SOURCE` (`_source`), `STATUS`, `TENANT_ID`, `PROJECT_ID`, `CREATED_AT`, `UPDATED_AT`.
**Constantes de estado:** `ACTIVE`, `ARCHIVED`, `CANCELED`, `PENDING`, `APPROVED`, `REJECTED`, `OF_SYSTEM`, `FOR_DELETE`; mapa `Status`; `SetStatus(value)`.

**Query/Command fluido:**
```go
items, _ := model.Where(jsql.Eq("status", jsql.ACTIVE)).And(jsql.More("age", 18)).Limit(20).Page(1).All()
item, _  := model.Where(jsql.Eq("id", id)).One()
_, _      = model.Insert(et.Json{"email": "a@b.com"}).ExecTx(nil)
_, _      = model.Update(et.Json{"status": "archived"}).Where(jsql.Eq("id", id)).ExecTx(nil)
_, _      = model.Upsert(et.Json{"id": id}).ExecTx(nil)
```
- `.Debug()` / `.Test()` en `Model`, `Query` y `Command` — loguea/devuelve SQL sin ejecutar.
- Triggers: `beforeInserts/Updates/Deletes`, `afterInserts/Updates/Deletes` (`TriggerFunction = func(tx *Tx, old, new et.Json) error`); columnas calculadas vía `CalcFunction`.
- Paths anidados JSONB: `"ventas->detalle->precio"` se traduce a `->`/`->>` automáticamente.

**Errores comunes:**
- Configurar `DB_DRIVER=sqlite`/`mysql`/`mssql`/`oracle`/`josefina` — falla al resolver driver (ninguno registrado salvo `postgres`).
- Asumir que `jsql.Load`/`LoadTo` aceptan un objeto `Config` — ya no existe ese parámetro.

### 2.2 `stores/`

**Descripción:** helpers de persistencia jsql-backed para "instancias" genéricas y autorización.

**API:**
- `func DefineInstance/LoadInstance/DefineInstanceBite/LoadInstanceBite(db *DB, schema, name string) (*Instance, error)` — `stores/instances.go:100-129`
- `func (s *Instance) Set(id, tag, tenantId, ownerId string, obj any, userId string) error` — :136
- `func (s *Instance) Get(id string, dest any) (bool, error)` — :181 (**un solo parámetro de clave**)
- `func (s *Instance) Delete(id string) error` — :229
- `func (s *Instance) Query(query et.Json) (et.Items, error)` — :248
- `func DefineAuthorization(db *DB, schema string) (*Authorization, error)` — `stores/authorization.go`

**Errores comunes:** `Get(id string, dest any)` de un solo parámetro **no** satisface `ia.Store.Get(id, tag string, dest any)` ni `jwf.Store.Get(collection, id string, dest any)` (requieren 2 strings). Nada en el repo conecta hoy `stores/` con `ia`/`jwf`.

### 2.3 `dt/`

**Descripción:** cache de objetos liviano: Redis en producción, filesystem en desarrollo (según `PRODUCTION`).

**API:** `dt.Up(key, data)`, `dt.Get(key)`, `dt.Drop(key)`.

---

## 3. HTTP

### 3.1 `server/`

**Descripción:** servidor HTTP ligero, sin `cache`/`event`, solo `chi` + `http.Server`.

**API (`server/server.go`):**
- `type Ettp struct { ... }` (envuelve `*chi.Mux`) — :19
- `func New(name string, port int) *Ettp` — constructor
- `(*Ettp) Use(middlewares ...func(http.Handler) http.Handler)` — :102
- `(*Ettp) NotFound(handlerFn http.HandlerFunc)` — :114
- `(*Ettp) HandleFunc(pattern string, handlerFn http.HandlerFunc)` — :126
- `(*Ettp) Mount(pattern string, handler http.Handler)` — :138
- `(*Ettp) Start()` — :176
- `(*Ettp) Close()` — :74
- `(*Ettp) OnClose(fn func())` / `OnStart(fn func())` — :86, :94

**`server/api.go`:** `NewApi(name, path, host string, port int, version string) *Api`; métodos `Use`, `UseAutentication`, `Public`, `Private`.

### 3.2 `ettp/v2`

**Descripción:** servidor HTTP completo con cache/eventos opcionales y sincronización de router entre réplicas vía NATS.

**API (`ettp/v2/server.go`):**
- `type Config struct { Port int; Parent string; ReadTimeout, WriteTimeout, IdleTimeout, Timeout time.Duration; AllowOrigin []string; IsTLS bool; CertFile, KeyFile string; Transport *TransportConfig; UseCache, UseEvent, Debug bool }` — :42
- `func New(name string, cnf *Config) (*Server, error)` — :95 (llama `cache.Load()`/`event.Load()` internamente **solo si** `cnf.UseCache`/`cnf.UseEvent` — :165-175)
- Sincronización de router: eventos `EVENT_SET_ROUTER`, `EVENT_REMOVE_ROUTER`, `EVENT_RESET_ROUTER`; bandera `m.Myself` evita auto-procesamiento.

**`ettp/v1/`:** versión anterior (`Config`/`Server` en `ettp/v1/server.go:34,53`, constructor `New(cnf *Config) (*Server, error)` :88), sin tocar desde 2026-06-02 — más "full-featured a la antigua" (incluye proxy/resolver/packages). Prefiere `v2`.

### 3.3 `router/`

**Descripción:** routing standalone con sincronización de estado entre instancias vía NATS, usado internamente por `ettp/v2`.

**API (`router/router.go`):**
- `func PushApiGateway(method, path, resolve string, tpHeader TpHeader, header et.Json, excludeHeader []string, version int, packageName string)` — :165
- `func RemoveApiGateway(id string)` — :187
- `func GetRoutes() map[string]et.Json` — :202
- `func SetChannels(channels *Channels)` — :62
- `func GetChanels() et.Json` — :72 (nota: typo en el nombre, no `GetChannels`)
- `func UseAutentication(fn func(http.Handler) http.Handler)` — :225
- `func Public/Private/Protect/With(r *chi.Mux, method, path string, ...) *chi.Mux` — :237-310
- `router/api.go`: `NewApi(name, path, host string, port, rpc int, version string) *Api`

### 3.4 `middleware/`

**Descripción:** middlewares HTTP para `chi`/`net/http`.

**API:**
- `func AllowAll(allowedOrigins []string) *cors.Cors` — `middleware/cors.go:10`
- `func RequestID(next http.Handler) http.Handler` — `middleware/request_id.go:67` (+ `GetReqID(ctx)`, `RequestIDHeader`)
- `func Logger(next http.Handler) http.Handler` — `middleware/logger.go:43` (+ `RequestLogger(f LogFormatter)`, `GetLogEntry(r)`)
- `func Authenticate(next http.Handler) http.Handler` — `middleware/autentication.go:43` (valida Bearer JWT vía `jwt.Validate`, puebla contexto de `request/`)
- `func Recoverer(next http.Handler) http.Handler` — `middleware/recoverer.go:23` (+ `PrintPrettyStack`)
- `type Metrics struct {...}` + `NewMetric(r *http.Request) *Metrics` / `NewRpcMetric(method string) *Metrics` — `middleware/telemetry.go` (latencia, tamaño, status; `DoneHTTP`, `WriteResponse/RESULT/JSON/ITEM/ITEMS/HTTPError`)

**Nota:** `logger.go` tiene una construcción de coloreado de puntero potencialmente nulo sin verificación explícita (líneas ~137-151) — revisar si se modifica ese archivo.

### 3.5 `response/`

**Descripción:** capa de salida HTTP unificada.

**API (`response/response.go`):**
- Lectura: `ScanBody`, `ScanStr`, `ScanJson`, `GetArray`, `GetQuery`, `GetParam` — :24-94
- `func WriteResponse(w, statusCode int, e []byte) error` — :103
- `func RESULT(w, r, statusCode int, data interface{}) error` — :132
- `func JSON(w, r, statusCode int, data interface{}) error` — :152
- `func ITEM(w, r, statusCode int, data et.Item) error` — :177
- `func ITEMS(w, r, statusCode int, data et.Items) error` — :197
- `func DATA(w, r, statusCode int, data et.Json) error` — :217
- `func ANY(w, r, statusCode int, result interface{}) error` — :237 (despacha por tipo a ITEM/ITEMS/JSON)
- `func HTTPError(w, r, statusCode int, message string) error` — :253
- `func HTTPAlert / Unauthorized / InternalServerError / Forbidden(w, r, ...)` — :266-290
- `func Stream(w, r, rows int, getData DataFunction) error` — :300 (streaming de array JSON paginado)

### 3.6 `request/`

**Descripción:** capa de entrada HTTP + propagación de contexto de usuario/tenant.

**API (`request/request.go`):**
- `type Body` con `ToJson/ToItem/ToItems/ToArrayJson/ToString/ToInt/ToInt64/ToFloat/ToBool/ToTime`
- `func ReadBody(body io.ReadCloser) (*Body, error)` — :218
- `func GetBody(r *http.Request) (et.Json, error)` — :247
- `func URLParam(r *http.Request, key string) *Value` — :398
- `func Query(r *http.Request, key string) *Value` — :409
- `type Value` con `.Str()/.Int()/.Float()/.Bool()/.DateTime()/.Object()/.Array()/.ArrayString()/.ArrayInt()/.ArrayFloat()/.ArrayJson()`

**Contexto (`request/ctx.go`):**
- Claves: `DurationKey, PayloadKey, ServiceIdKey, AppKey, DeviceKey, UserIdKey, UsernameKey, TenantIdKey, ProfileIdKey, TokenKey`
- Getters: `Duration(r), Payload(r), ServiceId(r), App(r), Device(r), Username(r), UserId(r) — :136, TenantId(r), ProfileId(r)`
- Setters: `SetDuration/SetPayload/SetServiceId/SetApp/SetDevice/SetUserId/SetUsername/SetTenantId/SetProfileId(ctx, val)`

### 3.7 `ws/`

**Descripción:** WebSocket sobre `gorilla/websocket`, modelo de `Hub` con tópicos/colas/pila.

**API:**
- `func New() *Hub` — `ws/hub.go:27` / `ws/ws.go:15`
- `(*Hub) Start()` — :95
- `(*Hub) Connect(socket *websocket.Conn, ctx context.Context) (*Client, error)` — :127
- `(*Hub) SendTo(to []string, message Message) ([]string, error)` — :229
- `(*Hub) Topic/Queue/Stack(channel string) *Channel` — :272-294 (broadcast / round-robin / LIFO)
- `(*Hub) Publish(channel string, message Message) ([]string, error)` — :375
- `(*Hub) Subscribe/Unsubscribe(channel, subscribe string) error` — :334-355
- Callbacks: `OnListener/OnConnection/OnDisconnection/OnChannel/OnRemove/OnPublish/OnSend`
- `func Upgrader(w, r) (*websocket.Conn, error)` — `ws/ws.go:39`
- `Client` (`ws/client.go:37`): `Send`, `SendMessage`, `SendError`, `SendHola`

---

## 4. Validación: `jval/`

**Descripción:** validadores tipados encadenables sobre `et.Json`.

**API (`jval/validate.go`):**
- `type Rule interface { Validate(et.Json) error; Name() string }` — :15
- `Str(name) *StringRule` + `.NotEmpty()`
- `Int(name) *IntRule` + `.Min(v)/.Max(v)`
- `Float(name) *FloatRule` + `.Min(v)/.Max(v)`
- `Array(name) *ArrayRule` + `.NotEmpty()`
- `Email(name) *EmailRule`
- `Date(name) *DateRule` + `.Layout(layout)`
- `Enum(name string, vals ...string) *EnumRule`
- `Phone(name) *PhoneRule` + `.CountryCode(code)/.Length(n)` (valida E.164)
- `Between(name string, min, max float64) *BetweenRule`
- `Validate(name string, rules ...Rule) *ObjectRule` (objeto anidado) — :406
- `func Require(data et.Json, rules ...Rule) error` — :581 (todas obligatorias)
- `func Maybe(data et.Json, rules ...Rule) error` — :595 (valida solo si el campo existe)

**Ejemplo:**
```go
err := jval.Require(body,
    jval.Str("email").NotEmpty(),
    jval.Email("email"),
    jval.Int("age").Min(0).Max(120),
)
```

---

## 5. Infraestructura

### 5.1 `cache/`

**Descripción:** cliente Redis con operaciones tipadas, pub/sub, colecciones, métricas.

**API destacada (`cache/handler.go` salvo donde se indique):**
- `func Load() error` — `cache/handler.go:22` · `func IsLoad() bool` — :51
- `func New() (*Conn, error)` — `cache/cache.go:41` · `(*Conn) Close()/HealthCheck()` — :84,94
- `Set/SetWithDuration/SetObject(key, val, expiration)`, `Get/GetObject(key, dest)`, `Exists(key)`, `Delete(key)`
- `LPush/LRem/LRange/LTrim` (listas), `Expire/Incr/IncrDuration/Decr` (contadores/TTL)
- `SetH/SetD/SetW/SetM/SetY` (atajos de expiración por horas/días/semanas/meses/años)
- `CollectionSet/Get/Delete/Put/Find` (hash de Redis)
- `ObjetAll/ObjetSet/ObjetGet/ObjetDelete` (objetos JSON)
- `SetVerify/GetVerify/DeleteVerify` (patrón OTP: get-then-delete)
- `AllCache(search string, page, rows int) (et.List, error)` (scan paginado)
- `GetJson/GetItem/GetItems(key)` (deserialización tipada)
- `type Metrics struct{...}`

**Env vars:** `REDIS_HOST` (requerido), `REDIS_PASSWORD`, `REDIS_DB` (opcionales).

### 5.2 `event/`

**Descripción:** pub/sub sobre NATS con logging asíncrono y soporte de colas/pila.

**API (`event/handler.go`, `event/message.go`):**
- `func Load() error` — :31 · `Close()/IsLoad()/HealthCheck()` — :48,58,66
- `func Publish(channel string, data et.Json) error` — :99
- `func Subscribe(channel string, f func(Message)) error` — :140 (broadcast)
- `func Queue(channel, queue string, f func(Message)) error` — :186 (reparto entre workers)
- `func Stack(channel string, f func(Message)) error` — :235
- `func Source(channel string, f func(Message)) error` — :244 (alias de `Subscribe`)
- `func Unsubscribe(channel string) error` — :112
- `func Log(event string, data et.Json)` / `Overflow(data)` / `Error(event string, err error) error` — :252-281 (logging async, no bloqueante)
- `type Message struct { CreatedAt time.Time; FromId, Id, Channel string; Data et.Json; Myself bool }` — `event/message.go:12`; `NewEvenMessage`, `Encode`, `ToJson`, `ToString`, `DecodeMessage`

### 5.3 `graph/`

**Descripción:** wrapper mínimo del driver Neo4j.

**API (`graph/graph.go`, archivo completo):**
```go
type Conn struct { driver neo4j.DriverWithContext; id, host string }
func Load() (*Conn, error) {
    // URL y credenciales hardcodeadas:
    // "neo4j://localhost:7687", neo4j.BasicAuth("neo4j", "password", "")
}
```
**No hay métodos de consulta, sesión ni transacción expuestos.** Para usar Neo4j de verdad hoy, hay que usar `neo4j-go-driver/v5` directamente o extender este paquete.

---

## 6. Configuración, entorno y logging

### 6.1 `config/`

- `func GetStr/GetInt/GetInt64/GetFloat/GetBool(key string, def ...) ...` — `config/handler.go:64+` (package-level, backed by `envar`)
- `func Get(key string, def interface{}) interface{}` / `Set(param map[string]interface{}) *Config` / `Validate(keys []string) error` / `IsLoad() bool`
- `type Config struct { ID, TenantId, OwnerId, Tag, Stage string; Params map[string]interface{}; AuditLog []map[string]interface{} }` — `config/config.go:20` (registro de configuración por tenant, respaldado por un `Store{Get,Set,Delete}`)
- `func New(tag, stage, tenantId, ownerId string, store Store, userId string) (*Config, error)` — :40
- `func Load(tag, stage, tenantId, ownerId string, store Store, userId string) error` — :72
- **No existe `config.App`.**

### 6.2 `envar/`

- `GetStr/GetInt/GetInt64/GetFloat/GetBool(name, def)` — lectura de env vars con default
- `ArgStr/ArgInt/ArgInt64/ArgFloat64/ArgBool(name, defaultVal)` — lectura de argumentos CLI
- `Str/Int/Int64/Float/Bool(name)` — lectura sin default
- `Set/SetStr/SetInt/SetInt64/SetNumber/SetBool(name, value)` — escritura en proceso
- `Validate(keys []string) error`

### 6.3 `logs/`, `stdrout/`, `color/`

- `logs/`: `Log, Info(f), Alert(f), Error(f), Debug(f), Fatal, Panic, Tracer`; `EnableCallerInfo` (bool, desactivar en producción).
- `stdrout/`: `type Stdout interface { Notify(kind, message string) }`; `SetStdout(v Stdout)`; `Color(s *string, color, format string, args ...) *string`; `CW(w io.Writer, color []byte, format string, args ...)`; `Printl(kind, color string, args ...any) string`; `Traces/ErrorTraces(err error)`; constantes ANSI (`Red`, `Green`, etc.) y `IsTTY`.
- `color/`: `Purple/Green/Red/Yellow/Blue/Cyan/White/Black(str string) string` — envuelven con ANSI + reset.

---

## 7. Identidad y seguridad

### 7.1 `claim/`

- `type Claim struct { jwt.StandardClaims; ID, Salt, Duration, App, Device, UserId, Username, TenantId, ProfileId string; Payload et.Json }` — `claim/claim.go:43`
- `func NewClaim(duration time.Duration) *Claim` — :112
- `func NewToken(app, device, userId, username, tenantId, profileId string, payload et.Json, duration time.Duration) (string, error)` — :242 (HS256, `golang-jwt/jwt/v4`)
- `func ParceToken(token string) (*Claim, error)` — :144
- Secreto: env `SECRET` (default `"1977"`) vía `getSecret()` — :27

### 7.2 `jwt/`

- `func NewToken(app, device, userId, username, tenantId, profileId string, payload et.Json, duration time.Duration) (string, error)` — `jwt/jwt.go:35`
- `func NewAuthentication(app, device, userId, username string, duration time.Duration) (string, error)` — :56
- `func NewAuthorization(app, device, userId, username, tenantId, profileId string, duration time.Duration) (string, error)` — :75
- `func NewAppToken(app, device string, duration time.Duration) (string, error)` — :100
- `func NewEphemeralToken(...) (string, error)` — :116 (pensado para duraciones cortas tipo OTP)
- `func Validate(token string) (*claim.Claim, error)` — :181 (valida también contra cache)
- `func RenewToken(token string, duration time.Duration) (string, error)` — :226
- `func DeleteToken(app, device, username string) error` / `DeleteTokeByToken(token string) error` — logout

### 7.3 `reg/`

- `UUID() / ULID() / XID() string` — `reg/id.go:21,29,39`
- `GenKey/GenUUId/GenULID/GenXID(tag string) string` — :49-81
- `GetUUID/GetULID/GetXID(id string) string` — :90-122 (si `id` es `""`/`"*"`/`"new"`, genera nuevo; si no, retorna `id`)
- `TagUUID/TagULID/TagXID(tag, id string) string` — :156-188
- `GenSnowflake() string` / `GenIndex() int64` / `GenHashKey(args ...interface{}) string` — :128-146

### 7.4 `utility/`

- IDs: `UUID()`, `GetOTP(length)`, `GetRandomString(length)`, `GenId(id)`, `GenKey(id)`
- Cripto: `Encrypt(value, cryptoType) (string, error)` (MD5/SHA1/SHA256/SHA512/AES), `DecryptoAES(value) (string, error)`, `GetCryptoType(value)`
- Validación: `ValidStr/ValidIn/ValidId/ValidKey/ValidInt/ValidNum/ValidName/ValidEmail/ValidPhone/ValidUUID/ValidCode/ValidWord`

### 7.5 `strs/`

- Formato: `Format/FormatUppCase/FormatLowCase/FormatDateTime/FormatSerie`
- Manipulación: `Contains/Replace/ReplaceAll/Change/Name/Trim/NotSpace/DaskSpace`
- Casing: `Uppcase/Lowcase/Titlecase/Same`
- Arrays: `Split/GetSplitIndex/Append/AppendAny/JoinQuoted`
- Conversión: `StrToTime/StrToBool/HtmlToText/RemoveAcents`
- Otros: `MaskToken(token, length)`, `Parse(str, vars et.Json)` (reemplaza `{{key}}`)

---

## 8. Orquestación

### 8.1 `crontab/`

- `type Store interface { Set(id, tag, tenantId, ownerId string, obj any, userId string) error; Get(id string, dest any) (bool, error); Delete(id string) error; Query(query et.Json) (et.Items, error) }` — `crontab/crontab.go:29`
- `func New(tag string, store Store) (*Crontab, error)` — :54 (llama `cache.Load()` + `event.Load()`; `store` ya **obligatorio**, no opcional)
- `AddJob/AddOneShotJob/AddEventJob/AddOneShotEventJob/DeleteJob/StartJob/StopJob/Stop`
- Usa `robfig/cron` con `cron.WithSeconds()` — soporta spec de 6 campos (`"0 * * * * *"`)

### 8.2 `jwf/` (workflows — sustituye a `workflow/`/`instances/`, eliminados)

- `type Store interface { Set(collection, id, tenantId, ownerId string, obj any, userId string) error; Get(collection, id string, dest any) (bool, error); Delete(collection, id string) error; Query(query et.Json) (et.Items, error); GetCode(tag string) (string, error) }` — `jwf/workflow.go:22`
- `func New(tenantId string, store Store) (*WorkFlow, error)` — :52 (`cache.Load()` + `event.Load()`)
- `func Load(tenantId string, store Store, userId string) (*WorkFlow, error)` — :89 (error si `store == nil`)
- `(*WorkFlow) NewFloW(tag, title, version, userId string) *Flow` — :220 (sic: "FloW")
- `(*WorkFlow) Run(flowId, triggerTag, id, projectId string, ctx, tags et.Json, userId string) (et.Json, error)` — :231
- `(*Flow) Step(tag, title string, fn func(*Instance, et.Json) (et.Json, error)) *Flow` — `jwf/flow.go:515`
- `(*Flow) Error(tag, version, title string, fn ...) *Flow` — :541 (puerto de error)
- `(*Instance) SetParams(params et.Json) et.Json` — `jwf/instance.go:544`
- Estados de `Instance`: `CREATED, PENDING, RUNNING, ROLLBACK, DONE, FAILED, CANCEL, STOP`
- `(*WorkFlow) LoadRouter(r Router)` — `jwf/router.go:22` — registra `httpGetStep/httpNewStep/httpUpdateStep/httpSetDefinitionStep/httpDeleteStep` (implementados) y `httpGetFlow/httpSetFlow/httpStatusFlow/httpDeleteFlow/httpGetInstance/httpDeleteInstance/httpRunInstance` (**cuerpo vacío**, sin implementar)

### 8.3 `resilience/`

- `type Store interface { Set(tag, id, tenantId, ownerId string, obj any, userId string) error; Get(id string, dest any) (bool, error); Delete(id string) error; Query(query et.Json) (et.Items, error) }` — `resilience/resilience.go:18`
- `func New(store Store) (*Resilience, error)`
- `(*Resilience) LoadInstance(Params{TenantId, Id, Tag, Description, OwnerId, TotalAttempts, Interval, Tags, UserId, Fn, FnArgs}) *Instance` — :193
- `(*Instance) Run(userId string) ([]interface{}, error)` — `resilience/instance.go:248`

### 8.4 `ia/`

- `type Store interface { Set(id, tag, tenantId, ownerId string, obj any, userId string) error; Get(id, tag string, dest any) (bool, error); Delete(id, tag string) error; Query(query et.Json) (et.Items, error) }` — `ia/ia.go:25`
- `func New(tenantId, tag string, store Store) (*Ia, error)` — :53 (`event.Load()`, **sin** `cache.Load()`; `OPENAI_API_KEY` vía `config.GetStr`)
- `func Load(tenantId, tag string, store Store) error` — :92 (singleton)
- Modela `Agent`, `Participant`, `Conversation` (con `Message`); `(*Ia) Conversation(ctx, tagAgent, to, prompt, userId string) (*Conversation, error)` — :557
- `(*Ia) Embed(ctx, agentName, text string) ([]float64, error)` — :525
- `Skill` (p. ej. `ApiSkill`) permite que un agente llame APIs externas.

### 8.5 `jrex/`

- `func Load(tag string, store Store) (*Jrex, error)` — `jrex/jrex.go:59`
- `type Store interface { Get(collection, id string, dest any) (bool, error); Set(collection, id, ownerId string, obj any, userId string) error }` — `jrex/store.go:12`
- `type FileStore` (`jrex/store.go:17`) — implementación basada en filesystem con hot-reload (`fsnotify`)
- `(*Jrex) Set(name string, value interface{}) *Jrex` — binding Go→JS
- `(*Jrex) NewInstance(module string) (*Instance, error)` / `RunInstance` / `RunModule` / `Run()`
- `type Instance` (`jrex/instance.go`): `NewInstance() *Instance` (standalone), `.Set(name, value)`, `.SetCtx(ctx et.Json)`, `.RunString(code string) (goja.Value, error)`, `.Ctx et.Json`, `.Run() (et.Json, error)`
- Globales JS: `console.*`, `ctx.*`, `fetch()`, `require()`
- Modos: `Develop` (hot-reload desde archivos), `Production` (carga desde `Store`), `Building` (compila + guarda con bump semver)

### 8.6 `service/`

- `func VerifyOTP(channel string, otp, createdBy string) (bool, error)` — `service/otp.go:18`
- `func SendOTPSMS(...)` / `SendOTPEmail(...)` / `SendOTPByTemplateId(...)` — `service/otp.go:37,76,115` (delegan en `aws.SendSMS`/`brevo.SendEmail*`)
- `func SendSms/SendWhatsapp/SendEmail/SendEmailByTemplateId(...)` — `service/send.go:33-125` (delegan en `aws`/`brevo`)
- `type TpMessage` (enum: `TypeNotification, TypeComercial, TypeAutentication`)

---

## 9. Comunicación de bajo nivel

### 9.1 `jrpc/`

- `func Mount(host string, port int, services any, packageName string) (*Package, error)` — `jrpc/jrpc.go:36` (reflexiona la struct de servicios, registra con `net/rpc`)
- `func Start(port int) error` — :79
- `func Call/CallJson/CallItems/CallItem(method string, args ...) (..., error)` — :128-188
- `func GetSolver(method string) (*Solver, error)` — `jrpc/handler.go:22`
- `type Solver struct { Host, Port string; Inputs, Output []string }` / `type Package struct { Name, Host, Port string; Solvers map[string]*Solver }` — `jrpc/package.go`
- **No hay `balancer.go` ni `raft.go`** — solo resolución simple por `Solver{Host,Port}`.

### 9.2 `tcp/`

- `const (Follower, Candidate, Leader, Proxy Mode)` — `tcp/node.go:27`
- `func NewNode(port int, tlsConfig ...*tls.Config) *Node` — :91 (env `TIMEOUT` default "10s", `WORKER_COUNT` default 1000, `CONFIG_FILE` default "./config.json")
- `(*Node) run()/connect/disconnect/send/inbox/Error/closeAllClients` — gestión de ciclo de vida de conexiones y mensajes
- Callbacks: `onConnect, onDisconnect, onError, onInbox, onSend, onBecomeLeader, onChangeLeader`
- `type Raft struct { ctx, node, addr, state Mode, term int, votedFor, leaderID string, lastHeartbeat time.Time, mu sync.Mutex }` — `tcp/raft.go:73` (implementación propia, **no usa una librería externa de Raft**; solo importa paquetes internos de `et` + stdlib)

---

## 10. Integraciones externas

### 10.1 `aws/`

- `func UploaderS3/UploaderFile/UploaderB64/DeleteS3/DownloadS3/DeleteFile/DownloaderFile(...)` — `aws/s3.go`
- `func SendSMS(contactNumbers []string, content string, params et.Json, tp string) (et.Items, error)` — `aws/sms.go:19` (`tp`: "Transactional"/"Promotional"; reemplaza `{{key}}` con `params`; usa AWS SNS)
- Env vars: `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, `BUCKET`, `STORAGE_TYPE`, `HOSTNAME`

### 10.2 `brevo/`

- `func SendEmail/SendEmailTransactional/SendEmailPromotional(...)` — `brevo/email.go`
- `func SendSmsTransactional/SendSmsPromotional(...)` — `brevo/sms.go`
- `func SendWhatsapp/SendWhatsappTransactional/SendWhatsappPromotional(contactNumbers []string, templateId string, params []et.Json, tp string) (et.Items, error)` — `brevo/whatsapp.go:19,88,97` (WhatsApp **templado**, vía Brevo — distinto de `wsp/`)
- Env vars: `BREVO_SEND_PATH`, `BREVO_SEND_KEY`, `BREVO_SENDER`

### 10.3 `wsp/`

- `func NewSender(token, phoneNumberId string) *Whatsapp` — `wsp/wsp.go:24` (lee `WHATSAPP_API_URL`, default `https://graph.facebook.com/v22.0`)
- `.Debug()/.IsTest()/.SetVerifyToken(...)/.SetEventHandler(fn)/.SetEventHandlerError(fn)` — :38-77
- `(*Whatsapp) Webhooks(w, r)` — `wsp/handler.go:19`
- Decenas de `Send*`/`SendReply*` para texto, imagen, audio, documento, sticker, video, contacto, ubicación, plantilla, catálogo, lista — `wsp/handler.go:108-689`
- **Bug conocido**: `SendReplyVideoMessageByURL(to, url, videoCaptionText string)` (`wsp/handler.go:368`) asigna `url` al campo `MessageID` del mensaje — no recibe un `messageID` separado como sus hermanos `SendReply*ById`.

---

## 11. Concurrencia y memoria

### 11.1 `mem/`

- `func Load() *Mem` (instancia local) + singleton de paquete con mismas funciones a nivel de paquete (`Set/Get/Delete/Exists/GetEntry/GetStr/GetInt/.../More/Clear/Empty/Len/Keys/Values`)
- `(*Mem) Set(key string, value interface{}, expiration time.Duration) (*Entry, error)`
- Getters tipados con bool de "existe" y error: `GetStr/GetInt/GetInt64/GetFloat/GetBool/GetTime/GetDuration/GetJson/GetArrayStr/GetArrayInt/GetArrayFloat/GetArrayJson(key, def) (T, bool, error)`
- `(*Mem) More(key string, expiration time.Duration) (int64, error)` — contador atómico (primera llamada retorna 1)
- `(*Mem) Clear(match string)` — borra claves que coincidan con regex
- `type Entry` con getters tipados similares y serialización a `[]byte`
- `NewPeticiones(capacity, timeWait int) *Peticiones` + `(*Peticiones) Ejecucion(fn, params) (et.Items, error)` — limitador de concurrencia

### 11.2 `ephemeral/`

- `func NewInstance(expiration time.Duration) *Instance`
- `(*Instance) Set(key, value)` (resetea timer), `Del(key)`, `Get(key) (interface{}, bool)` (resetea timer al leer)
- Más simple que `mem/`: todo `interface{}`, sin accesores tipados, sin persistencia granular.

### 11.3 `race/`

- `func NewValue(value interface{}) *Value` — wrapper thread-safe con `sync.RWMutex`
- `.Set/.Delete/.Get/.String/.Int/.Float64/.Bool/.Time/.Array/.Map/.StringArray/.IntArray/.Float64Array/.IsNil/.MapRange/.ArrayRange/.Range/.Increase(n)`

### 11.4 `iterate/`

- `func Start(tag string)` / `Segment(tag, msg string, isDebug bool) time.Duration` / `End(tag, msg string, isDebug bool) time.Duration` — medición de tiempo entre checkpoints con logging.

---

## 12. Tiempo, unidades y archivos

### 12.1 `timezone/`

- Constantes de layout: `RFC3339Nano`, `RFC3339`, `YYYYMMDDTHHMMSSZ`, `YYYYMMDDTHHMMSSSSZ`
- `func Now() time.Time` / `NowStr() string` / `Add(d) time.Time` / `Location() *time.Location` / `Format(t, layout) string` / `Parse(layout, value) (time.Time, error)` / `FormatMDYYYY(value string) string`
- Zona vía env `TIMEZONE` (default `"America/Bogota"`), formato por defecto vía `LAYOUT_TIME`.

### 12.2 `units/`

- `type TypeUnity` con constantes de distancia (km/m/cm/mm), masa (mg/g/kg/ton/lb/oz), volumen (ml/cl/l/m³)
- `func NewQuantity(val float64, unit TypeUnity) *Quantity` / `(*Quantity) Load(val interface{}) error` / `.To(unit) error` (convierte in-place) / `.ToStr()/.ToJson()`
- `func Load(val any) (*Quantity, error)` (factory desde float/map/`et.Json`/string)

### 12.3 `file/`

- `type FileInfo struct { Path string; Info os.FileInfo; Error error; IsDir, Exist bool }` + `.Json() et.Json`
- `func ExistPath(path string) FileInfo` / `GetExtencion(filename) string` / `MakeFolder(names ...string) (string, error)` / `MakeFile(path, name, model string, args ...any) (string, error)`
- `type Watcher` (`fsnotify`-backed): `NewWatcher(root) (*Watcher, error)`, builder `.OnCreate/.OnWrite/.OnRemove/.OnRename/.OnChmod/.OnReload/.OnError(fn) *Watcher`, `.Load() error` (bloqueante), `.Close()`, `.Debug()`
- `func WatcherPath(root string) error` (atajo simple)

---

## 13. Herramientas de desarrollo

### 13.1 `cmds/`

- `func Load(fileName string) (*Stage, error)` — carga un `Stage` desde archivo
- `Stage{Id, Name, Description, Steps []*Step}`, `NewStage`, `.AppendStep`
- `Step{Id, Name, Description, Commands []*Cmd}`, `NewStep`, `.AppendCmd(args string)` (parsea `"cmd arg1 arg2"`)
- `(*Step) RunOS(idx int, args et.Json) ([]byte, error)` — ejecuta comando local sustituyendo variables
- `(*Step) RunSSH(idx int, args et.Json) ([]byte, error)` — **idéntico a `RunOS`** (usa `exec.Command` local, no SSH real)

### 13.2 `create/`

- Comando Cobra (`create.go`) que invoca `PrompCreate()` (menú interactivo: Project, Microservice, Modelo, Rpc)
- `PrompStr/PrompInt(label string, require bool) (T, error)` — prompts vía `promptui`
- Genera estructura de carpetas (`cmd/`, `deployments/`, `pkg/`, `rest/`, `test/`, `web/`), archivos Go base, posible YAML de Kubernetes.

### 13.3 `cmd/*` (binarios)

| Binario | Propósito |
|---|---|
| `cmd/et` | CLI principal (Cobra) |
| `cmd/apigateway` | API Gateway/proxy sobre `ettp.New` |
| `cmd/daemon` | Servicio en background con integración systemd |
| `cmd/server` | Nodo TCP (`tcp.NewNode(port)`) |
| `cmd/jrex` | Runner de `jrex` en modo dev con hot-reload |
| `cmd/jsql` | Demo del driver `jsql` |
| `cmd/jwf` | Ejemplo de uso de `jwf` (`NewFloW`/`Step`/`Run`) |
| `cmd/resilience` | Ejemplo de uso de `resilience` |
| `cmd/wsp` | Ejemplo de uso de `wsp` |
| `cmd/client` | Cliente de prueba |
| `cmd/install` | Utilidad de instalación |
| `cmd/whatcher` | Observador de cambios de filesystem |
| `cmd/create` | Generador de proyectos |

### 13.4 `jcli/` (huérfano, en progreso)

`jcli/jcli.go` implementa un modelo de CLI Bubble Tea (`cliModel`, interfaz `App{RunCli() error}`) pero **declara `package jrex`** estando en el directorio `jcli/`, y **nada en el repo importa `github.com/cgalvisleon/et/jcli`**. Parece ser una extracción en progreso del CLI de desarrollo de `jrex/` hacia su propio paquete, sin terminar de conectar.

---

## 14. Patrón transversal: `msg/`

La mayoría de los paquetes (incluyendo el paquete raíz `msg/`) centralizan sus mensajes de error como constantes string (`MSG_*`), normalmente en un archivo `msg.go` local al paquete. Ejemplo (`msg/msg.go`): `MSG_ATRIB_REQUIRED`, `MSG_RECORD_NOT_FOUND`, `MSG_TOKEN_INVALID`, `MSG_TOKEN_EXPIRED`, etc. Al generar código que interactúa con `et`, prefiere comparar/propagar estos mensajes en vez de inventar nuevos strings literales equivalentes.
