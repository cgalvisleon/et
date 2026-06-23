# COMPONENT_CATALOG.md

> Catálogo exhaustivo de componentes de `github.com/cgalvisleon/et` (Go 1.25). Cada entrada cita archivo:línea real al momento de generar este documento — el repo cambia con frecuencia (ver advertencia en `LIBRARY_CONTEXT.md`), así que verifica contra el código antes de depender de una firma exacta en una decisión crítica. Para la narrativa de arquitectura, ver `ARCHITECTURE_SUMMARY.md`; para reglas de uso por IA, ver `AI_USAGE_GUIDE.md`.

---

## Índice

1. Núcleo de datos: `et/`
2. Persistencia: `jsql/`, `stores/`, `dt/`
3. HTTP: `server/`, `ettp/v1`, `ettp/v2`, `router/`, `middleware/`, `response/`, `request/`, `jws/`
4. Validación: `jval/`
5. Infraestructura: `cache/`, `event/`, `graph/`
6. Configuración/entorno/log: `config/`, `envar/`, `logs/`, `stdrout/`, `color/`
7. Identidad/seguridad: `claim/`, `jwt/`, `reg/`, `utility/`, `strs/`
8. Orquestación: `crontab/`, `jwf/`, `resilience/`, `jia/`, `jrex/`, `service/`
9. Comunicación de bajo nivel: `jrpc/`, `jtcp/`
10. Integraciones externas: `aws/`, `brevo/`, `jwsp/`
11. Concurrencia/memoria: `mem/`, `ephemeral/`, `race/`, `iterate/`
12. Tiempo/unidades/archivos: `timezone/`, `units/`, `file/`
13. Herramientas de desarrollo: `cmds/`, `create/`, `cmd/*`, `jcli/`
14. Patrón transversal `msg/`

---

## 1. Núcleo de datos: `et/`

**Descripción:** tipo de datos universal (`Json`) y wrappers de resultado (`List`, `Item`, `Items`) usados en todas las capas.

**API pública (`et/json.go`):**
- `func (s Json) ToByte() ([]byte, error)` — :115
- `func (s Json) ToString() string` — :128
- `func (s Json) ToEscapeHTML() string` — :141
- `func (s Json) ToMap() map[string]interface{}` — :160
- `func (s Json) IsEmpty() bool` — :168
- `func (s Json) IsExist(key string) bool` — :177
- `func (s Json) Clone() Json` — :186
- `func (s Json) ValAny/ValStr/ValInt/ValInt64/ValNum/ValBool/ValTime/ValJson/ValArray(def, atribs ...string) ...` — :197-455 (patrón "valor con default + ruta anidada")
- `func (s Json) Str/String/Int/Int64/Num/Bool/Time/Json(atrib)/Byte` (atribs ...string) — :546-630
- `func (s Json) FromBase64/ToBase64(...)` — :564-587 (codificación auxiliar)
- `func (s Json) MapStr/MapInt/MapFloat(atrib string) map[string]...` — :663-704
- `func (s Json) Array/ArrayBytes/ArrayStr/ArrayInt/ArrayInt64/ArrayNumber/ArrayJson(atribs ...string)` — :711-841
- `func (s Json) Update(from Json)` — :848 (mezcla in-place)
- `func (s Json) Compare(from Json) Json` — :858 (diff)
- `func (s Json) Append(from Json) Json` — :873
- `func (s Json) IsChanged(from Json) bool` — :887
- `func (s Json) IsDeferent(atrib string, val interface{}) bool` — :917
- `func (s Json) Get(keys ...string) (result interface{})` — :929
- `func (j Json) SetNested(keys []string, value interface{})` — :965
- `func (s Json) Set(key string, val interface{})` — :995
- `func (s Json) Delete(keys []string) bool` — :1014
- `func (s Json) Exist(key string) bool` — :1035
- `func (s Json) Remove(keys ...string)` — :1044
- `func (s Json) Select(keys []string) Json` — :1055
- `func (s Json) Hidden(keys []string) Json` — :1072

**`et/list.go`:** `List{Rows, All, Count, Page, Start, End, Result []Json}`.

**`et/item.go` / `et/items.go`:** `Item{Ok bool, Result Json}` con los mismos accesores tipados; `Items` con `Add`, `AddMany`, `One(idx)`, `First`, `Last`, `ToList(all, page, rows)`.

**`et/msg.go`:** constantes de error propias del paquete `et` (`MSG_FIELD_NOT_FOUND`, `MSG_DATA_NOT_FOUND`, `MSG_INDEX_OUT_OF_RANGE`, `MSG_FAILED_TO_UNMARSHAL_JSON_VALUE`, con variantes en inglés/español) — **distinto** del paquete raíz `msg/` (ver §14), que es el catálogo de mensajes compartido entre el resto de paquetes.

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

**Descripción:** SQL builder agnóstico de motor + ORM ligero con esquema híbrido relacional/JSONB. Internamente, `jsql` define su **propio** `Store` (idéntico en forma al de `jia`/`jwf`/`resilience`/`crontab` — ver `ARCHITECTURE_SUMMARY.md` §3.2) para persistir, de forma opcional, la metadata de sus propios objetos `DB`/`Model`.

**Entrada (`jsql/jsql.go`):**
- `func Load(tenantId string) (*DB, error)` — :86
- `func LoadTo(tenantId, name string) (*DB, error)` — :73
- `func ConnectTo(connect Connection) (*DB, error)` — :46
- `func GetDb(name string) (*DB, error)` — :142
- `func GetModel(db, schema, name string) (*Model, error)` — :155
- `func Define(dbName string, def Def) (*Model, error)` — :187 (atajo de paquete que delega en `(*DB).Define`)
- `func Insert/Update/Delete/Upsert(model *Model, data et.Json) *Command` — :151-180
- `func NewDB(params et.Json, store Store, userId string) (*DB, error)` — :99 (**registra** un `*DB` respaldado por `Store`, distinto del flujo de conexión `Load`/`LoadTo`)
- `func LoadDB(id string, store Store) (*DB, error)` — :113 (carga un `*DB` previamente registrado por su propio ID)

**`jsql.Store` (`jsql/db.go:20`):**
```go
type Store interface {
    Set(collection, id, ownerId string, obj any) error
    Get(collection, id string, dest any) (bool, error)
    Delete(collection, id string) error
    Query(query et.Json) (et.Items, error)
}
```
`(*DB).save()` (`jsql/db.go:159`) y `(*Model).save()` (`jsql/model.go:246`) son no-op si `store == nil` — es decir, este `Store` es completamente opcional; `jsql.Load`/`LoadTo` no lo usan ni lo requieren para el flujo normal de conexión/consulta.

**Drivers (`jsql/driver.go`):**
- Constantes: `DriverPostgres`, `DriverSqlite`, `DriverMysql`, `DriverMssql`, `DriverOracle`, `DriverJosefina` — :9-14
- `type Driver interface { Connect(db *DB) (*sql.DB, error); Load(model *Model) (string, error); Query(query *Query) (string, error); Command(command *Command) (string, error) }` — :20-26
- `func Register(name string, driver Driver)` — :39
- **Solo `jsql/drivers/postgres/` tiene archivos.** `jsql/drivers/mysql/` y `jsql/drivers/josefina/` existen vacíos; **no existe `jsql/drivers/sqlite/`** aunque `DriverSqlite` y `sqliteConection()` (`jsql/conection.go:98`) sigan en el código.

**Definición de modelo (`jsql/define.go`, `jsql/db.go`):**
- `func (s *DB) DefineModel(schema, name string, version int) (*Model, error)` — `jsql/define.go:461`
- `func (s *DB) NewModel(schema, name string, version int) (*Model, error)` — `jsql/db.go:251`
- `func (s *DB) Define(define Def) (*Model, error)` — `jsql/db.go:351`
- `type Def struct { Schema, Name string; Version int; IdxField, IdtField string; PrimaryKeys, ForeignKeys, Indexes, Unique, Required []DefIndex/DefForeignKeys; Columns []Column; SourceField string; Hiddens []string; Details map[string]DefDetail; Rollups map[string]DefRollup; IsCore, IsDebug, IsTest bool }` — `jsql/define.go:43-65`

**Tipos de columna (`TypeColumn`, `jsql/column.go:31-38`):** `COLUMN`, `ATTRIB`, `DETAIL`, `ROLLUP`, `CALCFUNC`, `CALC`, `AGG`.
**Tipos de dato (`TypeData`, `jsql/column.go:54-66`):** `KEY`, `TEXT`, `MEMO`, `INT`, `FLOAT`, `BOOLEAN`, `DATETIME`, `JSON`, `BYTES`, `GEOMETRY`, `EMBEDDING`, `ANY`.
**Constantes de columna (`jsql/column.go:4-14`):** `ID`, `IDX` (`_idx`), `IDT` (`_idt`), `SOURCE` (`_source`), `STATUS`, `TENANT_ID`, `PROJECT_ID`, `CREATED_AT`, `UPDATED_AT`.
**Constantes de estado (`jsql/column.go:69-95`):** `ACTIVE`, `ARCHIVED`, `CANCELED`, `PENDING`, `APPROVED`, `REJECTED`, `OF_SYSTEM`, `FOR_DELETE`; mapa `Status`; `SetStatus(value)`.

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
- Confundir el `Store` opcional de `jsql.NewDB`/`LoadDB` (registro de metadata) con la conexión real a la base de datos (`jsql.Load`/`LoadTo`) — son mecanismos independientes.

### 2.2 `stores/`

**Descripción:** helpers de persistencia jsql-backed: "instancias" genéricas, autorización y un catálogo `kind`+`id` genérico.

**API:**
- `func DefineInstance(db *DB, tenantId, schema string) (*Instance, error)` — `stores/instances.go:26`
- `func (s *Instance) Set(tag, id, ownerId string, obj any) error` — :88
- `func (s *Instance) Get(id string, dest any) (bool, error)` — :134 (**un solo parámetro de clave**)
- `func (s *Instance) Delete(id string) error` — :181
- `func (s *Instance) Query(query et.Json) (et.Items, error)` — :201
- `func DefineAuthorization(db *DB, tenantId, schema string) (*Authorization, error)` — `stores/authorization.go`
- `func DefineCatalog(db *DB, tenantId, schema string) (*Catalog, error)` — `stores/catalog.go:22`
- `func (s *Catalog) Set(collection, id, ownerId string, obj any) error` — :81
- `func (s *Catalog) Get(collection, id string, dest any) error` — :115 (**dos claves, pero devuelve solo `error`**)
- `func (s *Catalog) Delete(collection, id string) error` — :147
- `func (s *Catalog) Query(query et.Json) (et.Items, error)` — :165

**Compatibilidad con la forma unificada de `Store` (`Set(collection, id, ownerId, obj)`/`Get(collection, id, dest) (bool, error)`/`Delete(collection, id)`/`Query`, ver `ARCHITECTURE_SUMMARY.md` §3.2):**
- `stores.Catalog` — **casi calza**: `Set`/`Delete`/`Query` son idénticos; solo `Get` rompe la compatibilidad porque devuelve `error` en vez de `(bool, error)`. Es la implementación más cercana a un adaptador trivial.
- `stores.Instance` — **no calza**: `Get`/`Delete` reciben una sola clave string, no dos (`collection`+`id`).
- Nada en el repo conecta hoy `stores/` con `jia`/`jwf`/`resilience`/`crontab`/`jsql` (todos se ejercitan con `store=nil` en sus ejemplos de `cmd/`).

### 2.3 `dt/`

**Descripción:** cache de objetos liviano: Redis en producción, filesystem en desarrollo (según `PRODUCTION`).

**API (`dt/handler.go`):** `func Up(key string, data any) *Object` — :15, `func Get(key string) *Object` — :29, `func Drop(key string) error` — :47.

---

## 3. HTTP

### 3.1 `server/`

**Descripción:** servidor HTTP ligero, sin `cache`/`event`, solo `chi` + `http.Server`.

**API (`server/server.go`):**
- `type Ettp struct { ... }` (envuelve `*chi.Mux`) — :19
- `func New(name string, port int) *Ettp` — :34
- `(*Ettp) Use(middlewares ...func(http.Handler) http.Handler)` — :102
- `(*Ettp) NotFound(handlerFn http.HandlerFunc)` — :114
- `(*Ettp) HandleFunc(pattern string, handlerFn http.HandlerFunc)` — :126
- `(*Ettp) Mount(pattern string, handler http.Handler)` — :138
- `(*Ettp) Start()` — :176
- `(*Ettp) Close()` — :74
- `(*Ettp) OnClose(fn func())` / `OnStart(fn func())` — :86, :94

**`server/api.go`:** `NewApi(name, path, host string, port int, version string) *Api` — :24; métodos `Use`, `UseAutentication`, `Public`, `Private`.

### 3.2 `ettp/v2`

**Descripción:** servidor HTTP completo con cache/eventos opcionales y sincronización de router entre réplicas vía NATS.

**API (`ettp/v2/server.go`):**
- `type Config struct { Port, RpcPort int; Parent string; ReadTimeout, WriteTimeout, IdleTimeout, Timeout time.Duration; AllowOrigin []string; IsTLS bool; CertFile, KeyFile string; Transport *TransportConfig; UseCache, UseEvent, Debug bool }` — :43
- `func New(name string, cnf *Config) (*Server, error)` — :95 (llama `cache.Load()`/`event.Load()` internamente **solo si** `cnf.UseCache`/`cnf.UseEvent`)
- `RpcPort` ya no tiene fallback interno (`envar.GetInt("RPC_PORT", 4200)`) — si se deja en cero, el `net.Listen` del RPC interno se bindea a un puerto asignado por el SO.
- Sincronización de router: eventos `EVENT_SET_ROUTER`, `EVENT_REMOVE_ROUTER`, `EVENT_RESET_ROUTER`; bandera `m.Myself` evita auto-procesamiento.

**`ettp/v1/`:** versión anterior (`Config`/`Server` en `ettp/v1/server.go`, constructor `New(cnf *Config) (*Server, error)`), último cambio fue un sweep mecánico `config.GetBool`→`envar.GetBool`, no una feature nueva — más "full-featured a la antigua" (incluye proxy/resolver/packages). Prefiere `v2`.

### 3.3 `router/`

**Descripción:** routing standalone con sincronización de estado entre instancias vía NATS, usado internamente por `ettp/v2`.

**API (`router/router.go`):**
- `func PushApiGateway(method, path, resolve string, tpHeader TpHeader, header et.Json, excludeHeader []string, version int, packageName string)` — :165
- `func RemoveApiGateway(id string)` — :187
- `func GetRoutes() map[string]et.Json` — :202
- `func SetChannels(channels *Channels)` — :62
- `func GetChanels() et.Json` — :72 (nota: typo confirmado en el código, no es error de doc — no `GetChannels`)
- `func UseAutentication(fn func(http.Handler) http.Handler)` — :14 (typo confirmado: "Autentication", no "Authentication")
- `func Public/Private/Protect/With(r *chi.Mux, method, path string, ...) *chi.Mux` — :237-310
- `router/api.go`: `NewApi(name, path, host string, port, rpc int, version string) *Api` — :26

### 3.4 `middleware/`

**Descripción:** middlewares HTTP para `chi`/`net/http`.

**API:**
- `func AllowAll(allowedOrigins []string) *cors.Cors` — `middleware/cors.go:10`
- `func RequestID(next http.Handler) http.Handler` — `middleware/request_id.go:67` (+ `GetReqID(ctx)`, `RequestIDHeader`)
- `func Logger(next http.Handler) http.Handler` — `middleware/logger.go:43` (+ `RequestLogger(f LogFormatter)`, `GetLogEntry(r)`)
- `func Authenticate(next http.Handler) http.Handler` — `middleware/autentication.go:43` (valida Bearer JWT vía `jwt.Validate`, puebla contexto de `request/`)
- `func Recoverer(next http.Handler) http.Handler` — `middleware/recoverer.go:23` (+ `PrintPrettyStack`)
- `type Metrics struct{...}` + `NewMetric(r *http.Request) *Metrics` — `middleware/telemetry.go:188` / `NewRpcMetric(method string) *Metrics` — :246 (latencia, tamaño, status; `DoneHTTP`, `WriteResponse/RESULT/JSON/ITEM/ITEMS/HTTPError`)

### 3.5 `response/`

**Descripción:** capa de salida HTTP unificada.

**API (`response/response.go`):**
- Lectura: `ScanBody`, `ScanStr`, `ScanJson`, `GetArray`, `GetQuery`, `GetParam` — :24-96
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
- Getters: `Duration(r), Payload(r), ServiceId(r), App(r), Device(r), Username(r), UserId(r), TenantId(r), ProfileId(r)`
- Setters: `SetDuration/SetPayload/SetServiceId/SetApp/SetDevice/SetUserId/SetUsername/SetTenantId/SetProfileId(ctx, val)`

### 3.7 `jws/` (antes `ws/`)

**Descripción:** WebSocket sobre `gorilla/websocket`, modelo de `Hub` con tópicos/colas/pila.

**API:**
- `func New() *Hub` — `jws/hub.go:27`
- `(*Hub) Start()` — :95
- `(*Hub) Connect(socket *websocket.Conn, ctx context.Context) (*Client, error)` — :127
- `(*Hub) SendTo(to []string, message Message) ([]string, error)` — :229
- `(*Hub) Topic/Queue/Stack(channel string) *Channel` — :272-294 (broadcast / round-robin / LIFO)
- `(*Hub) Publish(channel string, message Message) ([]string, error)` — :375
- `(*Hub) Subscribe/Unsubscribe(channel, subscribe string) error` — :334-355
- Callbacks: `OnListener/OnConnection/OnDisconnection/OnChannel/OnRemove/OnPublish/OnSend`
- `func Upgrader(w, r) (*websocket.Conn, error)` — `jws/ws.go:39`
- `Client` (`jws/client.go:37`): `Send`, `SendMessage`, `SendError`, `SendHola`

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

**API destacada:**
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

**API (`graph/graph.go`, archivo completo, 33 líneas):**
```go
type Conn struct { driver neo4j.DriverWithContext; id, host string }
func Load() (*Conn, error) { // :17
    // URL y credenciales hardcodeadas:
    // "neo4j://localhost:7687", neo4j.BasicAuth("neo4j", "password", "")
}
```
**No hay métodos de consulta, sesión ni transacción expuestos.** Para usar Neo4j de verdad hoy, hay que usar `neo4j-go-driver/v5` directamente o extender este paquete.

---

## 6. Configuración, entorno y logging

### 6.1 `config/`

**Descripción:** registro de configuración por tenant, respaldado por un `Store` propio (no es un descriptor de app global — no existe `config.App`).

**`config.Store` (`config/config.go:18`) — la única forma de `Store` del repo que NO sigue el esquema `(collection, id, ownerId)` (ver `ARCHITECTURE_SUMMARY.md` §3.2):**
```go
type Store interface {
    Set(tag, stage, tenantId, ownerId string, obj any) error
    Get(tag, stage string, dest any) (bool, error)
    Delete(tag, stage string) error
}
```
**Bug confirmado**: `(*Config).Save()` (`config/config.go:126`) invoca `s.store.Set(s.Tag, s.Stage, s.OwnerId, s.TenantId, s)` en la línea 136 — los argumentos 3 y 4 (`tenantId`, `ownerId`) están **intercambiados** respecto al orden declarado en la interfaz (`tag, stage, tenantId, ownerId, obj`). Cualquier implementación real de `Store` que distinga ambos campos los persistirá cruzados.

- `func New(tag, stage, tenantId, ownerId string, store Store, userId string) (*Config, error)` — :49
- `func Load(tag, stage string, store Store, userId string) error` — :81
- `func GetStr/GetInt/GetInt64/GetFloat/GetBool(key string, def ...) ...` — `config/handler.go` (package-level, backed by `envar`)
- `func Get(key string, def interface{}) interface{}` / `Set(param map[string]interface{}) *Config` / `Validate(keys []string) error` / `IsLoad() bool`
- `type Config struct { CreatedAt, UpdatedAt time.Time; ID, TenantId, OwnerId, Tag, Stage string; Params et.Json; AuditLog []et.Json }` — :24-37

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
- `func NewClaim(duration time.Duration) *Claim` — :108
- `func NewToken(app, device, userId, username, tenantId, profileId string, payload et.Json, duration time.Duration) (string, error)` — `claim/claim.go` (HS256, `golang-jwt/jwt/v4`)
- `func ParceToken(token string) (*Claim, error)` — :140 (typo confirmado en el código: "Parce", no "Parse")
- Secreto: env `SECRET` (default `"1977"`) vía `getSecret()` — :27-33

### 7.2 `jwt/`

- `func NewToken(app, device, userId, username, tenantId, profileId string, payload et.Json, duration time.Duration) (string, error)` — `jwt/jwt.go:35`
- `func NewAuthentication(app, device, userId, username string, duration time.Duration) (string, error)` — :56
- `func NewAuthorization(app, device, userId, username, tenantId, profileId string, duration time.Duration) (string, error)` — :75
- `func NewAppToken(app, device string, duration time.Duration) (string, error)` — :100
- `func NewEphemeralToken(...) (string, error)` — :116 (pensado para duraciones cortas tipo OTP)
- `func Validate(token string) (*claim.Claim, error)` — :181 (valida también contra cache)
- `func RenewToken(token string, duration time.Duration) (string, error)` — :226
- `func DeleteToken(app, device, username string) error` — :149 / `DeleteTokeByToken(token string) error` — logout

### 7.3 `reg/`

- `UUID() / ULID() / XID() string` — `reg/id.go:21,29,39`
- `GenKey/GenUUId/GenULID/GenXID(tag string) string` — :49-83 (typo confirmado: `GenUUId`, no `GenUUID`)
- `GetUUID/GetULID/GetXID(id string) string` — :90-122 (si `id` es `""`/`"*"`/`"new"`, genera nuevo; si no, retorna `id`)
- `TagUUID/TagULID/TagXID(tag, id string) string` — :156-188
- `GenSnowflake() string` / `GenIndex() int64` / `GenHashKey(args ...interface{}) string` — :128-146

### 7.4 `utility/`

- IDs: `UUID()`, `GetOTP(length)`, `GetRandomString(length)`, `GenId(id)`, `GenKey(id)`
- Cripto: `Encrypt(value, cryptoType) (string, error)` (MD5/SHA1/SHA256/SHA512/AES), `DecryptoAES(value) (string, error)` — :166 (typo confirmado: "Decrypto", no "Decrypt"), `GetCryptoType(value)`
- Validación: `ValidStr/ValidIn/ValidId/ValidKey/ValidInt/ValidNum/ValidName/ValidEmail/ValidPhone/ValidUUID/ValidCode/ValidWord`

### 7.5 `strs/`

- Formato: `Format/FormatUppCase/FormatLowCase/FormatDateTime/FormatSerie`
- Manipulación: `Contains/Replace/ReplaceAll/Change/Name/Trim/NotSpace/DaskSpace` — :180 (typo confirmado: "Dask", no "Dash")
- Casing: `Uppcase/Lowcase/Titlecase/Same`
- Arrays: `Split/GetSplitIndex/Append/AppendAny/JoinQuoted`
- Conversión: `StrToTime/StrToBool/HtmlToText/RemoveAcents`
- Otros: `MaskToken(token, length)`, `Parse(str, vars et.Json)` (reemplaza `{{key}}`)

---

## 8. Orquestación

### 8.1 `crontab/`

> **Rediseño reciente**: la API pública documentada en versiones anteriores (`AddJob/AddOneShotJob/AddEventJob/AddOneShotJob/DeleteJob/StartJob/StopJob/Stop` como métodos de `*Crontab`) **ya no existe**. El paquete pasó a un modelo orientado a eventos con un singleton de paquete.

- `type Store interface { Set(collection, id, ownerId string, obj any) error; Get(collection, id string, dest any) (bool, error); Delete(collection, id string) error; Query(query et.Json) (et.Items, error) }` — `crontab/crontab.go:24` (misma forma que `jia.Store`/`jwf.Store`/`resilience.Store`)
- `func New(tag string, store Store) (*Crontab, error)` — :47 (solo llama `event.Load()`; inicia `cron.New(cron.WithSeconds(), cron.WithLocation(loc))`)
- `func Load(tag string, store Store) error` — `crontab/handler.go:42` (crea el `*Crontab` vía `New` y lo guarda en un **singleton de paquete** `var crontab *Crontab`; llama `crontab.eventInit()`)
- `type Cron struct { DayOfWeek, Month, DayOfMonth, Hour, Minute string }` — `crontab/handler.go:52` (spec **estructurada**, ya no un string crudo tipo `"0 * * * * *"`; `(*Cron).toString()` valida cada campo con regex antes de convertir a spec de `robfig/cron`)
- `func CronJob(tag, ownerId string, spec Cron, repetitions int, params et.Json, fn func(params et.Json) error) error` — :95 (reemplaza al antiguo `AddJob`; opera sobre el singleton, falla si `Load` no se llamó antes)
- `func ScheduleJob(tag, ownerId string, spec time.Time, params et.Json, fn func(params et.Json) error) error` — :114 (job de una sola ejecución; reemplaza al antiguo `AddOneShotJob`)
- `func HttpRemoveJob/HttpStopJob/HttpStartJob(w http.ResponseWriter, r *http.Request)` — :127,146,165 (handlers HTTP que **publican eventos** `EVENT_CRONTAB_REMOVE/STOP/START` en vez de mutar el job directamente)
- Internamente (`crontab/event.go`): `newJob(...)` publica `EVENT_CRONTAB_SET`; `(*Crontab).eventSet/eventRemove/eventStop/eventStart` (suscritos en `eventInit`, :23) son los que finalmente llaman a los métodos **privados** `addJob/removeJob/startJob/stopJob` (`crontab/crontab.go:92,124,147,168`).
- **Nada en el repo importa `crontab` actualmente** — no hay ejemplo en `cmd/` que lo ejercite.

### 8.2 `jwf/` (workflows — sustituye a `workflow/`/`instances/`, eliminados)

- `type Store interface { Set(collection, id, ownerId string, obj any) error; Get(collection, id string, dest any) (bool, error); Delete(collection, id string) error; Query(query et.Json) (et.Items, error); GenSerie(tag string) (string, error) }` — `jwf/workflow.go:23`
- `func New(store Store) (*WorkFlow, error)` — :53 (`cache.Load()` + `event.Load()`; `WorkFlow.ID = reg.UUID()`, ya no recibe `tenantId`)
- `func Load(id string, store Store) (*WorkFlow, error)` — :81 (error si `store == nil`; carga por el propio ID, vía `store.Get("workflow", id, ...)`)
- `(*WorkFlow) NewFloW(tag, title, version, userId string) *Flow` — :368 (sic: "FloW")
- `(*WorkFlow) Run(flowId, triggerTag, id, projectId string, ctx, tags et.Json, userId string) (et.Json, error)` — :380
- `(*Flow) Step(tag, title string, fn func(*Instance, et.Json) (et.Json, error)) *Flow` — `jwf/flow.go:482`
- `(*Flow) Error(tag, version, title string, fn func(*Instance, et.Json) (et.Json, error)) *Flow` — :504 (puerto de error)
- `(*Instance) SetParams(params et.Json) et.Json` — `jwf/instance.go:537`
- Estados de `Instance`: `CREATED, PENDING, RUNNING, ROLLBACK, DONE, FAILED, CANCEL, STOP`
- `(*WorkFlow) LoadRouter(r Router)` — `jwf/router.go:22` — registra `httpGetStep/httpNewStep/httpUpdateStep/httpSetDefinitionStep/httpDeleteStep` (implementados) y `httpGetFlow/httpSetFlow/httpStatusFlow/httpDeleteFlow/httpGetInstance/httpDeleteInstance/httpRunInstance` (**cuerpo vacío**, sin implementar)

### 8.3 `resilience/`

- `type Store interface { Set(collection, id, ownerId string, obj any) error; Get(collection, id string, dest any) (bool, error); Delete(collection, id string) error; Query(query et.Json) (et.Items, error) }` — `resilience/resilience.go:19` (misma forma que `jia.Store`/`jwf.Store`/`crontab.Store`)
- `func New(store Store) (*Resilience, error)` — :42 (solo llama `event.Load()`)
- `type Params struct { Id, Tag, Description string; TotalAttempts int; Interval time.Duration; Tags et.Json; Fn interface{}; FnArgs []interface{} }` — :180-189 (sin `TenantId`/`OwnerId`/`UserId`)
- `(*Resilience) LoadInstance(params Params) *Instance` — :196 (default `TotalAttempts=3`, `Interval=30s` si no se especifican)
- `(*Instance) Run(userId string) ([]any, error)` — `resilience/instance.go:250`

### 8.4 `jia/` (antes `ia/`)

- `type Store interface { Set(collection, id, ownerId string, obj any) error; Get(collection, id string, dest any) (bool, error); Delete(collection, id string) error; Query(query et.Json) (et.Items, error) }` — `jia/ia.go:25` (misma forma unificada)
- `func New(tag string, store Store, userId string) (*Ia, error)` — :58 (solo `event.Load()`, **sin** `cache.Load()`; `Ia.ID = reg.UUID()`, ya no recibe `tenantId`; `OPENAI_API_KEY` vía `envar.GetStr`)
- `func Load(id string, store Store) (*Ia, error)` — :84 (carga por el propio ID vía `store.Get(id, "ia", &result)`)
- Modela `Agent`, `Participant`, `Conversation` (con `Message`)
- `(*Ia) Embed(ctx context.Context, agentName, text string) ([]float64, error)` — :487
- `(*Ia) Conversation(ctx context.Context, tagAgent, to, name, prompt string) (*Conversation, error)` — :519 (firma real: **4 strings** `tagAgent, to, name, prompt`, sin `userId` — versiones previas de esta tabla citaban mal el orden de parámetros)
- `Skill` (p. ej. `ApiSkill`) permite que un agente llame APIs externas.

### 8.5 `jrex/`

- `func Load(tag string, store Store) (*Jrex, error)` — `jrex/jrex.go:59`
- `type Store interface { Set(collection, id, ownerId string, obj any) error; Get(collection, id string, dest any) (bool, error) }` — `jrex/store.go:13-14` (**subconjunto** de 2 métodos de la forma unificada — sin `Delete`/`Query`)
- `type FileStore` (`jrex/store.go:17`) — implementación basada en filesystem con hot-reload (`fsnotify`)
- `(*Jrex) Set(name string, value interface{}) *Jrex` — binding Go→JS
- `(*Jrex) NewInstance(module string) (*Instance, error)` / `RunInstance` / `RunModule` / `Run()`
- `type Instance` (`jrex/instance.go`): `NewInstance() *Instance` (standalone), `.Set(name, value)`, `.SetCtx(ctx et.Json)`, `.RunString(code string) (goja.Value, error)`, `.Ctx et.Json`, `.Run() (et.Json, error)`
- Globales JS: `console.*`, `ctx.*`, `fetch()`, `require()`
- **No existen** modos `Develop`/`Production`/`Building` en el código actual (confirmado por búsqueda directa de esos identificadores) — cualquier mención a ellos en documentación es residual de una versión anterior.

### 8.6 `service/`

- `func VerifyOTP(channel string, otp, createdBy string) (bool, error)` — `service/otp.go:18`
- `func SendOTPSMS(tenantId, serviceId, sender, countryCode, phoneNumber string, length int, duration time.Duration, createdBy string) (et.Items, error)` — :37
- `func SendOTPEmail(tenantId, serviceId string, from et.Json, name, email string, length int, duration time.Duration, createdBy string) (et.Items, error)` — :76
- `func SendOTPByTemplateId(tenantId, serviceId string, from et.Json, name, email string, length int, duration time.Duration, templateId, createdBy string) (et.Items, error)` — :115
- `func SendSms(tenantId, serviceId string, contactNumbers []string, content string, params et.Json, tp TpMessage, createdBy string) (et.Items, error)` — `service/send.go:33`
- `func SendWhatsapp(...)` — :63 · `func SendEmail(...)` — :93 · `func SendEmailByTemplateId(...)` — :125 (delegan en `aws`/`brevo`)
- `type TpMessage` (enum: `TypeNotification, TypeComercial, TypeAutentication`) — :12

---

## 9. Comunicación de bajo nivel

### 9.1 `jrpc/`

- `func Mount(host string, port int, services any, packageName string) (*Package, error)` — `jrpc/jrpc.go:36` (reflexiona la struct de servicios, registra con `net/rpc`)
- `func Start(port int) error` — :79
- `func Call/CallJson/CallItems/CallItem(method string, args ...) (..., error)` — :128-188
- `func GetSolver(method string) (*Solver, error)` — `jrpc/handler.go:22`
- `type Solver struct { Host, Port string; Inputs, Output []string }` / `type Package struct { Name, Host, Port string; Solvers map[string]*Solver }` — `jrpc/package.go`
- **Confirmado: no hay `balancer.go` ni `raft.go` en `jrpc/`** — solo resolución simple por `Solver{Host,Port}`. Esos dos archivos viven en `jtcp/`.

### 9.2 `jtcp/` (antes `tcp/`)

- `const (Follower, Candidate, Leader, Proxy Mode)` — `jtcp/node.go:31-34`
- `func NewNode(port int, tlsConfig ...*tls.Config) *Node` — :91 (env `TIMEOUT` default "10s", `WORKER_COUNT` default 1000, `CONFIG_FILE` default "./config.json")
- `(*Node) run()/connect/disconnect/send/inbox/Error/closeAllClients` — gestión de ciclo de vida de conexiones y mensajes
- Callbacks públicos (capitalizados, confirmado): `OnConnect` — :846, `OnDisconnect` — :854, `OnError` — :862, `OnInbox` — :870, `OnSend` — :878, `OnBecomeLeader` — :886, `OnChangeLeader` — :894
- `type Raft struct { ctx, node, addr, state Mode, term int, votedFor, leaderID string, lastHeartbeat time.Time, mu sync.Mutex }` — `jtcp/raft.go:73` (implementación propia confirmada, **no usa una librería externa de Raft**)

---

## 10. Integraciones externas

### 10.1 `aws/`

- `func UploaderS3/UploaderFile/UploaderB64/DeleteS3/DownloadS3/DeleteFile/DownloaderFile(...)` — `aws/s3.go`
- `func SendSMS(contactNumbers []string, content string, params et.Json, tp string) (et.Items, error)` — `aws/sms.go:19` (`tp`: "Transactional"/"Promotional"; reemplaza `{{key}}` con `params`; usa AWS SNS)
- Env vars: `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, `BUCKET`, `STORAGE_TYPE`, `HOSTNAME`

### 10.2 `brevo/`

- `func SendEmail/SendEmailTransactional/SendEmailPromotional(...)` — `brevo/email.go`
- `func SendSmsTransactional/SendSmsPromotional(...)` — `brevo/sms.go`
- `func SendWhatsapp/SendWhatsappTransactional/SendWhatsappPromotional(contactNumbers []string, templateId string, params []et.Json, tp string) (et.Items, error)` — `brevo/whatsapp.go:19,88,97` (WhatsApp **templado**, vía Brevo — distinto de `jwsp/`)
- Env vars: `BREVO_SEND_PATH`, `BREVO_SEND_KEY`, `BREVO_SENDER`

### 10.3 `jwsp/` (antes `wsp/`)

- `func NewSender(token, phoneNumberId string) *Whatsapp` — `jwsp/wsp.go:24` (lee `WHATSAPP_API_URL`, default `https://graph.facebook.com/v22.0`)
- `.Debug()/.IsTest()/.SetVerifyToken(...)/.SetEventHandler(fn)/.SetEventHandlerError(fn)` — :38-77
- `(*Whatsapp) Webhooks(w, r)` — `jwsp/handler.go:19`
- Decenas de `Send*`/`SendReply*` para texto, imagen, audio, documento, sticker, video, contacto, ubicación, plantilla, catálogo, lista — `jwsp/handler.go:108-689`
- `SendReplyVideoMessageById(to, messageID, videoCaptionText, videoObjectID string)` — :348 (hermano correcto, recibe `messageID` aparte)
- **Bug confirmado**: `SendReplyVideoMessageByURL(to, url, videoCaptionText string)` — `jwsp/handler.go:369` — en la línea 372 asigna `url` al campo `MessageID` del mensaje, en vez de recibir un `messageID` separado como su hermano `SendReplyVideoMessageById`.

---

## 11. Concurrencia y memoria

### 11.1 `mem/`

- `func Load() *Mem` (instancia local) + singleton de paquete con mismas funciones a nivel de paquete (`Set/Get/Delete/Exists/GetEntry/.../More/Clear/Empty/Len/Keys/Values`) — `mem/mem.go:23`
- `(*Mem) Set(key string, value interface{}, expiration time.Duration) (*Entry, error)`
- Getters tipados con bool de "existe" y error, todos `(T, bool, error)`: `GetStr` — :151, `GetInt` — :170, `GetInt64` — :189, `GetFloat` — :208, `GetBool` — :227, más `GetTime/GetDuration/GetJson/GetArrayStr/GetArrayInt/GetArrayFloat/GetArrayJson`
- `(*Mem) More(key string, expiration time.Duration) (int64, error)` — contador atómico (primera llamada retorna 1)
- `(*Mem) Clear(match string)` — borra claves que coincidan con regex
- `type Entry` con getters tipados similares y serialización a `[]byte`
- `NewPeticiones(capacity, timeWait int) *Peticiones` + `(*Peticiones) Ejecucion(fn, params) (et.Items, error)` — limitador de concurrencia

### 11.2 `ephemeral/`

- `func NewInstance(expiration time.Duration) *Instance`
- `(*Instance) Set(key, value)` (resetea timer), `Del(key)`, `Get(key) (interface{}, bool)` (resetea timer al leer)
- Más simple que `mem/`: todo `interface{}`, sin accesores tipados, sin persistencia granular.

### 11.3 `race/`

- `func NewValue(value interface{}) *Value` — `race/race.go:16` — wrapper thread-safe con `sync.RWMutex`
- `.Set/.Delete/.Get/.String/.Int/.Float64/.Bool/.Time/.Array/.Map/.StringArray/.IntArray/.Float64Array/.IsNil/.MapRange/.ArrayRange/.Range/.Increase(n)` — :27-244

### 11.4 `iterate/`

**Descripción:** medición de tiempo entre checkpoints con logging, vía un singleton de paquete (`var iterate *Iterate` en `iterate/handler.go:5`, inicializado en `init()`).

- `func Start(tag string)` — `iterate/handler.go:17` (delega en el método privado `(*Iterate).start`, `iterate/iterate.go:18`)
- `func Segment(tag, msg string, isDebug bool) time.Duration` — `iterate/handler.go:29` (delega en `(*Iterate).segment`, `iterate/iterate.go:27`)
- `func End(tag, msg string, isDebug bool) time.Duration` — `iterate/handler.go:41` (delega en `(*Iterate).end`, `iterate/iterate.go:53`)

---

## 12. Tiempo, unidades y archivos

### 12.1 `timezone/`

- Constantes de layout: `RFC3339Nano`, `RFC3339`, `YYYYMMDDTHHMMSSZ`, `YYYYMMDDTHHMMSSSSZ`
- `func Now() time.Time` / `NowStr() string` / `Add(d) time.Time` / `Location() *time.Location` / `Format(t, layout) string` / `Parse(layout, value) (time.Time, error)` / `FormatMDYYYY(value string) string`
- Zona vía env `TIMEZONE` (default `"America/Bogota"`), formato por defecto vía `LAYOUT_TIME`.

### 12.2 `units/`

- `type TypeUnity` con constantes de distancia (km/m/cm/mm), masa (mg/g/kg/ton/lb/oz), volumen (ml/cl/l/m³)
- `func NewQuantity(val float64, unit TypeUnity) *Quantity` — :110 / `(*Quantity) Load(val interface{}) error` / `.To(unit) error` (convierte in-place) / `.ToStr()/.ToJson()`
- `func Load(val any) (*Quantity, error)` — :475 (factory desde float/map/`et.Json`/string)

### 12.3 `file/`

- `type FileInfo struct { Path string; Info os.FileInfo; Error error; IsDir, Exist bool }` — :18 + `.Json() et.Json`
- `func ExistPath(path string) FileInfo` — :78 / `GetExtencion(filename) string` — :60 (typo confirmado: "Extencion", no "Extension") / `MakeFolder(names ...string) (string, error)` — :118 / `MakeFile(path, name, model string, args ...any) (string, error)` — :149
- `type Watcher` (`fsnotify`-backed): `NewWatcher(root) (*Watcher, error)` — :38, builder `.OnCreate/.OnWrite/.OnRemove/.OnRename/.OnChmod/.OnReload/.OnError(fn) *Watcher` — :123-183, `.Load() error` (bloqueante) — :223, `.Close()` — :95, `.Debug()` — :103
- `func WatcherPath(root string) error` — :281 (atajo simple)

---

## 13. Herramientas de desarrollo

### 13.1 `cmds/`

- `func Load(fileName string) (*Stage, error)` — carga un `Stage` desde archivo
- `Stage{Id, Name, Description, Steps []*Step}`, `NewStage`, `.AppendStep`
- `Step{Id, Name, Description, Commands []*Cmd}`, `NewStep`, `.AppendCmd(args string)` (parsea `"cmd arg1 arg2"`)
- `(*Step) RunOS(idx int, args et.Json) ([]byte, error)` — ejecuta comando local sustituyendo variables
- `(*Step) RunSSH(idx int, args et.Json) ([]byte, error)` — **idéntico a `RunOS`** (usa `exec.Command` local, no SSH real)

### 13.2 `create/`

- Comando Cobra que invoca `PrompCreate()` — :10 (menú interactivo: Project, Microservice, Modelo, Rpc)
- `PrompStr(label string, require bool) (string, error)` — :54 / `PrompInt(label string, require bool) (int, error)` — :77 — prompts vía `promptui`
- Genera estructura de carpetas (`cmd/`, `deployments/`, `pkg/`, `rest/`, `test/`, `web/`), archivos Go base, posible YAML de Kubernetes.

### 13.3 `cmd/*` (binarios)

| Binario | Propósito |
|---|---|
| `cmd/et` | CLI principal (Cobra) |
| `cmd/apigateway` | API Gateway/proxy sobre `ettp.New` |
| `cmd/daemon` | Servicio en background con integración systemd |
| `cmd/server` | Nodo TCP (`jtcp.NewNode(port)`) |
| `cmd/jrex` | Runner de `jrex` en modo dev con hot-reload |
| `cmd/jsql` | Demo del driver `jsql` |
| `cmd/jwf` | Ejemplo de uso de `jwf` (`NewFloW`/`Step`/`Run`) |
| `cmd/resilience` | Ejemplo de uso de `resilience` |
| `cmd/wsp` | Ejemplo de uso de `jwsp` (conserva el nombre antiguo de directorio) |
| `cmd/client` | Cliente de prueba |
| `cmd/install` | Utilidad de instalación |
| `cmd/whatcher` | Observador de cambios de filesystem |
| `cmd/create` | Generador de proyectos |

### 13.4 `jcli/` (huérfano, en progreso)

`jcli/jcli.go` implementa un modelo de CLI Bubble Tea (`cliModel`, interfaz `App{RunCli() error}`) pero **declara `package jrex`** estando en el directorio `jcli/`, y **nada en el repo importa `github.com/cgalvisleon/et/jcli`** (confirmado por búsqueda directa). Parece ser una extracción en progreso del CLI de desarrollo de `jrex/` hacia su propio paquete, sin terminar de conectar.

---

## 14. Patrón transversal: `msg/`

El paquete raíz `msg/` (`msg/msg.go`) centraliza mensajes de error compartidos entre el resto de paquetes como constantes string (`MSG_*`): `MSG_ATRIB_REQUIRED`, `MSG_RECORD_NOT_FOUND`, `MSG_TOKEN_INVALID`, `MSG_TOKEN_EXPIRED`, etc. Es distinto de `et/msg.go` (ver §1), que cubre solo los mensajes internos del paquete `et`. La mayoría de los demás paquetes (`jia`, `jwf`, `resilience`, `crontab`, `claim`, etc.) además mantienen su **propio** `msg.go` local con constantes específicas de ese paquete, siguiendo el mismo patrón. Al generar código que interactúa con `et`, prefiere comparar/propagar estos mensajes en vez de inventar nuevos strings literales equivalentes.
