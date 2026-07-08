# COMPONENT_CATALOG.md

> Catálogo exhaustivo de componentes de `github.com/cgalvisleon/et` (Go 1.25). Cada entrada cita archivo:línea real al momento de generar este documento — el repo cambia con frecuencia, verifica contra el código antes de depender de una firma exacta en una decisión crítica. Para la narrativa de arquitectura, ver `ARCHITECTURE_SUMMARY.md`; para reglas de uso por IA, ver `AI_USAGE_GUIDE.md`; para la visión general, ver `LIBRARY_CONTEXT.md`.

---

## Índice

1. Núcleo de datos: `et/`
2. Persistencia: `jsql/`, `stores/`, `dt/`
3. HTTP: `server/`, `ettp/v1`, `ettp/v2`, `router/`, `middleware/`, `response/`, `request/`, `jws/`
4. Validación: `jval/`, `validator/`
5. Infraestructura: `cache/`, `event/`, `graph/`
6. Entorno y logging: `envar/`, `logs/`, `stdrout/`, `color/`
7. Identidad/seguridad: `claim/`, `jwt/`, `reg/`, `utility/`, `strs/`
8. Orquestación: `crontab/`, `jwf/`, `resilience/`, `jia/`, `jrex/`, `service/`
9. Comunicación de bajo nivel: `jrpc/`, `jtcp/`
10. Integraciones externas: `aws/`, `brevo/`, `jwsp/`
11. Concurrencia/memoria: `mem/`, `ephemeral/`, `race/`, `iterate/`, `queue/`
12. Tiempo/unidades/archivos: `timezone/`, `units/`, `file/`
13. Herramientas de desarrollo: `cmds/`, `create/`, `cmd/*`, `jcli/`, `infobip/`
14. Patrón transversal `msg/`

---

## 1. Núcleo de datos: `et/`

**Descripción:** tipo de datos universal (`Json`) y wrappers de resultado (`List`, `Item`, `Items`) usados en todas las capas.

**API pública (`et/json.go`):**
- `func (s *Json) Scan(src interface{}) error` — :32 (implementa `sql.Scanner`)
- `func (s *Json) ScanRows(rows *sql.Rows) error` — :62
- `func (s Json) ToByte() ([]byte, error)` — :115
- `func (s Json) ToString() string` — :128
- `func (s Json) ToEscapeHTML() string` — :141
- `func (s Json) ToMap() map[string]interface{}` — :160
- `func (s Json) IsEmpty() bool` — :168
- `func (s Json) IsExist(key string) bool` — :177
- `func (s Json) Clone() Json` — :186
- `func (s Json) ValAny/ValStr/ValInt/ValInt64/ValNum/ValBool/ValTime/ValDuration/ValJson/ValArray(def, atribs ...string) ...` — :197-554 (patrón "valor con default + ruta anidada", recorre `atribs` como path a través de `Json`/`map[string]interface{}` anidados)
- `func (s Json) Any/Str/String/Int/Int64/Num/Bool/Byte/Time/Json(atrib) (atribs ...string)` — :562-681
- `func (s Json) FromBase64/ToBase64(atribs ...string) string` — :589-611
- `func (s Json) MapStr/MapInt/MapFloat(atrib string) map[string]...` — :688-729
- `func (s Json) Array/ArrayBytes/ArrayStr/ArrayInt/ArrayInt64/ArrayNumber/ArrayJson(atribs ...string)` — :736-866
- `func (s Json) Update(from Json)` — :873 (mezcla in-place, `maps.Copy`)
- `func (s Json) Compare(from Json) Json` — :882 (diff)
- `func (s Json) Append(from Json) Json` — :897 (solo agrega claves que no existían)
- `func (s Json) IsChanged(from Json) bool` — :912
- `func (s Json) IsDeferent(atrib string, val interface{}) bool` — :941
- `func (s Json) Get(keys ...string) (result interface{})` — :954
- `func (j Json) SetNested(keys []string, value interface{})` — :989
- `func (s Json) Set(key string, val interface{})` — :1019 (soporta `key` con `->` como separador de nesting, delega a `SetNested`)
- `func (s Json) Delete(keys []string) bool` — :1038
- `func (s Json) Exist(key string) bool` — :1059
- `func (s Json) Remove(keys ...string)` — :1068
- `func (s Json) Select(keys []string) Json` — :1079 (whitelist)
- `func (s Json) Hidden(keys []string) Json` — :1096 (blacklist)
- `func (s Json) From(as string) *Where` / `ApplyCondition(condition *Condition) bool` — :1117,1126 (integración con el mini-query-builder de `et` sobre slices en memoria, distinto del builder de `jsql`)

**`et/list.go`:** `List{Rows, All, Count, Page, Start, End, Result []Json}`, `.ToJson()/.ToString()/.ToMap()/.From(as)`.

**`et/item.go`:** `Item{Ok bool, Result Json}`, `NewItem(data Json) Item`, mismos accesores tipados que `Json` delegando a `.Result`.

**`et/items.go`:** `Items{Ok bool, Count int, Result []Json}`, `NewItems(data []Json) Items`, `Add(item ...Json)`, `AddMany(items []Json)`, accesores indexados (`Str(idx, atribs...)`, etc.), `One(idx int) (Item, error)` (índice 1-based o negativo desde el final), `First()`/`Last()`, `ToList(all, page, rows int) List`.

**`et/msg.go`:** constantes de error propias del paquete `et` (`MSG_FIELD_NOT_FOUND`, `MSG_INDEX_OUT_OF_RANGE`, `MSG_FAILED_TO_UNMARSHAL_JSON_VALUE`, etc.) — **distinto** del paquete raíz `msg/` (§14).

**Ejemplo:**
```go
data := et.Json{"user": et.Json{"name": "Ana", "age": 30}}
name := data.Str("user", "name")
age := data.ValInt(0, "user", "age")
```

---

## 2. Persistencia

### 2.1 `jsql/`

**Descripción:** SQL builder agnóstico de motor + ORM ligero con esquema híbrido relacional/JSONB.

**Entrada (`jsql/jsql.go`):**
- `func Load() (*DB, error)` — conecta a la DB por defecto; **sin `tenantId`**, se lee vía `envar.GetStr("DB_TENANT_ID", "tenant:root")` internamente
- `func LoadTo(name string) (*DB, error)` — conecta a una DB nombrada; mismo cambio, sin `tenantId`
- `func GetDb(name string) (*DB, error)`, `func GetModel(db, schema, name string) (*Model, error)`
- `func Define(dbName string, def Def) (*Model, error)` — atajo de paquete que delega en `(*DB).Define`
- `func Insert/Update/Delete/Upsert(model *Model, data et.Json) *Command`

**`(*DB).NewDB(tenantId, host, name, driver string) (*DB, error)`** (`jsql/db.go:42`) — constructor de bajo nivel usado internamente por `Load`/`LoadTo`; no confundir con conectar directamente.

**`jsql.Store` — es un STRUCT concreto, no una interfaz** (`jsql/store.go:11`):
```go
type Store struct {
    model *Model
}
func DefineStore(db *DB, schema string) (*Store, error)
```
Es la ex-`stores.Catalog` (tabla genérica `kind`+`id`), movida dentro de `jsql`. Se usa opcionalmente para persistir metadata del propio `*DB`/`*Model`:
- `func (s *DB) Save(store *Store) error` — `jsql/db.go:158`
- `func LoadDb(store *Store, id string) (*DB, error)` — `jsql/db.go:87` (nótese "LoadDb")
- `func (s *Model) Save(store *Store) error` — `jsql/model.go:107`

Ambos `Save` son no-op si `store == nil` — el flujo normal de conexión no lo requiere. **No existe ninguna interfaz `Store` inyectable en `jsql`**, a diferencia de `jia`/`jwf`/`resilience`/`crontab` (una versión previa de este catálogo lo afirmaba incorrectamente).

**Drivers (`jsql/driver.go`):**
- Constantes: `DriverPostgres`, `DriverSqlite`, `DriverMysql`, `DriverMssql`, `DriverOracle`, `DriverJosefina` — :9-14
- `type Driver interface { Connect(db *DB) (*sql.DB, error); Load(model *Model) (string, error); Query(query *Query) (string, error); Command(command *Command) (string, error) }` — :20-26
- `func Register(name string, driver Driver)` — :39
- **Solo `jsql/drivers/postgres/` tiene archivos.** `jsql/drivers/mysql/` y `jsql/drivers/josefina/` existen vacíos; **no existe `jsql/drivers/sqlite/`** aunque `DriverSqlite` siga declarada.

**Definición de modelo (`jsql/define.go`, `jsql/db.go`):**
- `func (s *DB) DefineModel(schema, name string, version int) (*Model, error)`
- `func (s *DB) NewModel(schema, name string, version int, userId string) *Model` — `jsql/db.go:400`
- `func (s *DB) Define(define Def) (*Model, error)`
- `type Def struct { Schema, Name string; Version int; IdxField, IdtField string; PrimaryKeys, ForeignKeys, Indexes, Unique, Required []DefIndex/DefForeignKeys; Columns []Column; SourceField string; Hiddens []string; Details map[string]DefDetail; Rollups map[string]DefRollup; IsCore, IsDebug, IsTest bool }`

**Tipos de columna (`TypeColumn`):** `COLUMN`, `ATTRIB`, `DETAIL`, `ROLLUP`, `CALCFUNC`, `CALC`, `AGG`.
**Tipos de dato (`TypeData`):** `KEY`, `TEXT`, `MEMO`, `INT`, `FLOAT`, `BOOLEAN`, `DATETIME`, `JSON`, `BYTES`, `GEOMETRY`, `EMBEDDING`, `ANY`.
**Constantes de columna:** `ID`, `IDX` (`_idx`), `IDT` (`_idt`), `SOURCE` (`_source`), `STATUS`, `TENANT_ID`, `PROJECT_ID`, `CREATED_AT`, `UPDATED_AT`.
**Constantes de estado:** `ACTIVE`, `ARCHIVED`, `CANCELED`, `PENDING`, `APPROVED`, `REJECTED`, `OF_SYSTEM`, `FOR_DELETE`; mapa `Status`; `SetStatus(value)`.

**Query/Command fluido:**
```go
items, _ := model.Where(jsql.Eq("status", jsql.ACTIVE)).And(jsql.More("age", 18)).Limit(20).Page(1).All()
item, _  := model.Where(jsql.Eq("id", id)).One()
_, _      = model.Insert(et.Json{"email": "a@b.com"}).ExecTx(nil)
_, _      = model.Update(et.Json{"status": "archived"}).Where(jsql.Eq("id", id)).ExecTx(nil)
_, _      = model.Upsert(et.Json{"id": id}).ExecTx(nil)
```
- `.Debug()` / `.Test()` en `Model`, `Query` y `Command`.
- Triggers: `beforeInserts/Updates/Deletes`, `afterInserts/Updates/Deletes` (`TriggerFunction = func(tx *Tx, old, new et.Json) error`); columnas calculadas vía `CalcFunction`.
- Paths anidados JSONB: `"ventas->detalle->precio"` se traduce a `->`/`->>` automáticamente.
- `jsql.Series` (`jsql/series.go:122`) tiene un método `GenSerie` — no relacionado con ningún `Store` de otro paquete; si ves "GenSerie" mencionado como parte de `jwf.Store`, es un error (ver `ARCHITECTURE_SUMMARY.md` §3.2).

**Errores comunes:**
- Configurar `DB_DRIVER=sqlite`/`mysql`/`mssql`/`oracle`/`josefina` — falla al resolver driver.
- Pasar `tenantId` a `jsql.Load`/`LoadTo` — ya no existe ese parámetro.
- Confundir el `*jsql.Store` (struct concreto, opcional) con una interfaz inyectable de persistencia.

### 2.2 `stores/`

**Descripción:** helpers de persistencia jsql-backed: instancias genéricas, autorización, y configuración por tenant. **Ya no incluye un "Catalog" genérico** — eso se movió a `jsql.Store` (§2.1).

**API:**
- `func DefineInstance(db *DB, tenantId, schema string) (*Instance, error)` — `stores/instances.go`
- `func (s *Instance) Set(tag, id, ownerId string, obj any) error`
- `func (s *Instance) Get(id string, dest any) (bool, error)` — **un solo parámetro de clave**
- `func (s *Instance) Delete(id string) error`
- `func (s *Instance) Query(query et.Json) (et.Items, error)`
- `func DefineAuthorization(db *DB, tenantId, schema string) (*Authorization, error)` — `stores/authorization.go` (ACL tenant/perfil/método/ruta, cacheado vía `dt`)
- `func DefineConfig(db *DB, tenantId, schema, stage, tag string) (*Config, error)` — `stores/config.go` (reemplazo del viejo `config.Config`): `type Config struct { TenantId, Stage, Tag string; model *Model }` — mucho más simple que el `config.Config` eliminado (ese tenía además `ID`, `OwnerId`, `Params`, `AuditLog`)

**Compatibilidad con la forma unificada de `Store` (`Set(collection, id, ownerId, obj)`/`Get(collection, id, dest) (bool, error)`/`Delete(collection, id)`/`Query`, ver `ARCHITECTURE_SUMMARY.md` §3.2):**
- `stores.Instance` — **no calza**: `Get`/`Delete` reciben una sola clave string, no dos.
- Nada en el repo conecta hoy `stores/` con `jia`/`jwf`/`resilience`/`crontab` (todos se ejercitan con `store=nil` en sus ejemplos de `cmd/`).
- Por contraste, `jsql.Store` (el struct concreto de §2.1) **sí** tiene `Set(collection, id, ownerId, obj) error` / `Get(collection, id, dest) (bool, error)` / `Delete(collection, id) error` / `Query(query et.Json) (et.Items, error)` — calza exactamente con `jia.Store`, pero es un tipo concreto, no algo que implementes tú; conectarlo requeriría un adaptador que delegue en un `*jsql.Store` ya construido.

### 2.3 `dt/`

**Descripción:** cache de objetos liviano sobre `cache/` (Redis), condicionado por `PRODUCTION`.

**API (`dt/handler.go`, `dt/object.go`):**
- `func Up(key string, data any) Object` — prefixea `"object:"+key`; **solo si `PRODUCTION=true` persiste de verdad** en `cache.Set` — si `PRODUCTION=false` (o no está seteado), el objeto se construye solo en memoria y **nunca se persiste** (no hay fallback a filesystem, a diferencia de lo que sugería documentación previa)
- `func Get(key string) *Object` — lee vía `cache.GetObject`, `nil` si no existe
- `func Drop(key string) error` — `cache.Delete("object:"+key)`
- `func GetObject(key string, dest any) (bool, error)` — deserializa el payload a `dest`
- `type Object` con `Value any, Ok bool, Type string, Key string, Expire time.Duration` (default 5m vía env `CACHE_DURATION`) y accesores tipados: `Byte, String, Int, Int64, Float, Bool, Time, Duration, Array, Json, Item, Items, List, Array*`
- **Nota**: pese al nombre del archivo, `dt/handler.go` no contiene handlers HTTP — es la API de acceso a datos completa del paquete.

---

## 3. HTTP

### 3.1 `server/`

**Descripción:** servidor HTTP ligero, sin `cache`/`event`, solo `chi` + `http.Server`.

**API (`server/server.go`):**
- `type Ettp struct { ... }` (envuelve `*chi.Mux`)
- `func New(name string, port int) *Ettp` — si `port == 0`, el mux queda `nil` y todos los métodos se vuelven no-op
- `(*Ettp) Use(middlewares ...func(http.Handler) http.Handler)`, `NotFound(fn)`, `HandleFunc(pattern, fn)`, `Mount(pattern, handler)`
- `(*Ettp) Start()` — arranca en goroutine, corre hooks `OnStart`, imprime tabla de rutas (`chi.Walk`), muestra banner ASCII, bloquea en `utility.AppWait()`
- `(*Ettp) Close()`, `OnClose(fn)`, `OnStart(fn)`

**`server/api.go`:** un segundo tipo `Api`, `NewApi(name, path, host string, port int, version string) *Api` — instala `middleware.Logger`+`Recoverer`, delega en `router.Publish`/`router.With` para anunciar rutas a un gateway compartido. Casi duplicado del `Api` de `router/api.go` (que además recibe un parámetro `rpc int`) — no son el mismo tipo aunque el nombre coincida.

### 3.2 `ettp/v2`

**Descripción:** servidor HTTP completo con router sincronizado entre réplicas vía NATS; cache/eventos ya **no** son automáticos.

**API (`ettp/v2/server.go`):**
- `type Config struct { Port, RpcPort int; Parent string; ReadTimeout, WriteTimeout, IdleTimeout, Timeout time.Duration; AllowOrigin []string; IsTLS bool; CertFile, KeyFile string; Transport *TransportConfig; Debug bool }` — **no tiene** `UseCache`/`UseEvent`
- `func New(name string, cnf *Config) (*Server, error)` — nunca llama `cache.Load()`/`event.Load()` por sí solo
- Registrar handlers: `(*Server).Public(method, path string, fn http.HandlerFunc, packageName string) (*Solver, error)`, `Private(...)` (igual + middlewares de autenticación)
- Registrar proxy/API remota: `(*Server).SetRouter(method, path, resolve string, typeHeader int, header et.Json, excludeHeader []string, version int, packageName string, saved bool) (*Solver, error)`
- Tipos internos clave: `Solver` (definición de ruta persistible), `Package` (agrupa Solvers por nombre), `Router` (trie por segmento de path, uno por método HTTP), `Resolver` (una petición resuelta en curso, con `ServiceId`, `Status`, timer de limpieza)
- Métodos adicionales: `SetRouter`, `Use`, `UseAutentication` (sic), `RemoveRouterById`, `FindResolver`, `Start`, `Close`, `Reset`, `Save`, `HTTPError`, `HTTPSuccess`
- Rutas base (`basicRoutes()`): `GET /version`, `GET /reset` (privada), `POST /events`, `GET/POST /routes`, `DELETE /routes/{id}`, `GET /packages`, `DELETE /packages/{name}`, `GET/DELETE /cache`, `GET /cache/{key}`
- **RPC interno no conectado**: `Config.RpcPort` abre un `net.Listen("tcp", ...)` en `New()` y hay un codec gob (`pipe.go`) listo para atender esas conexiones, pero la función `startPipe()` que haría el `Accept()` loop **nunca es invocada** en ningún punto del código — es una funcionalidad iniciada pero no conectada. El RPC funcional real del repo es `jrpc/` (independiente de `ettp/v2`).
- Sincronización de router: suscribe (`initEvents()`) a `router.APIGATEWAY_SET_ROUTER`/`APIGATEWAY_REMOVE_ROUTER`/`APIGATEWAY_RESET_ROUTER` (y variantes v0); todo handler chequea `if m.Myself { return }`.
- Storage: `Save()` persiste todos los `Solvers` (excepto los de tipo `TpHandler`, que se re-registran en código en cada arranque) en `cache` si está cargado, o en un archivo `apigateway.json` si no.

**`ettp/v1/`:** no es simplemente "legado" — recibe commits recientes, y tiene archivos completos sin equivalente en v2 (`server-apigateway.go`, `server-proxy.go`, `server-token.go`, `server-method.go`, `server-cache.go`). Verifica cuál importa el binario que estás tocando antes de asumir "usa siempre v2".

### 3.3 `router/`

**Descripción:** routing standalone con anuncio de rutas vía NATS, usado internamente por `ettp/v2` (la dependencia va en un solo sentido: `ettp/v2` importa `router`, no al revés).

**API (`router/router.go`):**
- `func PushApiGateway(method, path, resolve string, tpHeader TpHeader, header et.Json, excludeHeader []string, version int, packageName string)`
- `func RemoveApiGateway(id string)`, `func GetRoutes() map[string]et.Json`
- `func SetChannels(channels *Channels)`, `func GetChanels() et.Json` (sic — typo confirmado, no "GetChannels")
- `func UseAutentication(fn func(http.Handler) http.Handler)` (sic — typo confirmado, no "Authentication")
- `func Public/Private/Protect/With(r *chi.Mux, method, path string, ...) *chi.Mux` — registran la ruta en el `chi.Mux` **y** anuncian la ruta al gateway vía eventos
- Constantes de evento reales: `APIGATEWAY_SET_ROUTER`, `APIGATEWAY_REMOVE_ROUTER`, `APIGATEWAY_RESET_ROUTER` (v1) y `APIGATEWAY_SET_RESOLVE`/`APIGATEWAY_DELETE_RESOLVE`/`APIGATEWAY_RESET` (v0, heredadas)
- `router/api.go`: `NewApi(name, path, host string, port, rpc int, version string) *Api`
- `Router` interface local (`UseAutentication`, `Protect`, `Public`, `With`) — no implementada por ningún tipo en el repo hoy, parece vestigial
- `router/hadler.go` (sic, no "handler.go"): `HttpSet/HttpGet/HttpDelete` — CRUD HTTP genérico sobre un store externo de solvers (`SetFn`/`GetFn`/`DeleteFn` en `storage.go`)
- Uso standalone: sí, no depende de `ettp/v2`.

### 3.4 `middleware/`

**Descripción:** middlewares HTTP para `chi`/`net/http`.

**API:**
- `func AllowAll(allowedOrigins []string) *cors.Cors` — `middleware/cors.go:10`
- `func RequestID(next http.Handler) http.Handler` — `middleware/request_id.go:67` (+ `GetReqID(ctx)`, `NextRequestID()`)
- `func Logger(next http.Handler) http.Handler` — `middleware/logger.go:43` (debe registrarse antes que `Recoverer`)
- `func Authentication(next http.Handler) http.Handler` — `middleware/authentication.go:43` (no "Authenticate"; valida Bearer JWT vía `jwt.Validate`, puebla contexto de `request/`, publica `PushTokenLastUse`)
- `func Recoverer(next http.Handler) http.Handler` — `middleware/recoverer.go:23` (+ `PrintPrettyStack`)
- `type Metrics struct{...}` + `NewMetric(r *http.Request) *Metrics` / `NewRpcMetric(method string) *Metrics` / `GetMetrics(r) *Metrics` (auto-crea si no existe) — `middleware/telemetry.go` — métodos `RESULT/JSON/ITEM/ITEMS/HTTPError/WriteResponse` (mismos nombres que `response/`, pero con telemetría añadida — no intercambiables) más `CallSearchTime/CallResponseTime/CallLatency/DoneHTTP/DoneRpc`
- `wrap_write.go`: `NewWrapResponseWriter`/`WrapResponseWriter` — wrapper de `http.ResponseWriter` estilo chi (`Flush`/`Hijack`/`Push`/`Tee`/`Unwrap`), redundante con el `ResponseWriterWrapper` que ya define `telemetry.go` para el mismo propósito
- `msg.go`: dos constantes (`ERR_AUTORIZATION_IS_REQUIRED`, `ERR_INVALID_AUTORIZATION_FORMAT`) que **no se usan** — `authentication.go` construye el mensaje de error con un literal duplicado en vez de citarlas

### 3.5 `response/`

**Descripción:** capa de salida HTTP unificada.

**API (`response/response.go`):**
- Lectura: `ScanBody(r io.Reader)`, `ScanStr(value string)`, `ScanJson(map[string]interface{})`, `GetArray(r) ([]et.Json, error)`, `GetQuery(r) et.Json`, `GetParam(r, key) string`
- `func WriteResponse(w, statusCode int, e []byte) error` — escritor crudo, fija `Content-Type: application/json; charset=utf-8`
- `func RESULT(w, r, statusCode int, data interface{}) error` — marshal directo, sin envoltorio
- `func JSON(w, r, statusCode int, data interface{}) error` — envuelve en `{ok, result}` (`ok = 200<=status<300`)
- `func ITEM(w, r, statusCode int, data et.Item) error`, `ITEMS(w, r, statusCode int, data et.Items) error`, `DATA(w, r, statusCode int, data et.Json) error`
- `func ANY(w, r, statusCode int, result interface{}) error` — despacha por tipo a `ITEM`/`ITEMS`/`JSON`
- `func HTTPError(w, r, statusCode int, message string) error` — envoltorio `{"message": message}`
- `func HTTPAlert(w, r, message string) error` (= `HTTPError` 400), `Unauthorized(w, r)`, `InternalServerError(w, r, err)`, `Forbidden(w, r)`
- `func Stream(w, r, rows int, getData DataFunction) error` — `DataFunction = func(page, rows int) (et.Items, error)`, streaming de array JSON paginado con flushing manual
- **Bug confirmado**: el chequeo interno "está vacío" en `ITEM`/`ITEMS`/`DATA` (`if &data == (&et.Item{})` y análogos) compara punteros de dos valores stack distintos — **siempre falso**, nunca detecta el caso de valor cero. El mismo idiom roto está copiado en `middleware/telemetry.go`.
- **Colisión de nombres a tener presente**: `response.JSON/ITEM/ITEMS/HTTPError` y `middleware.Metrics.JSON/ITEM/ITEMS/HTTPError` tienen los mismos nombres pero son conjuntos de funciones distintos (uno con telemetría, otro sin ella) — elige uno consistentemente por handler, no los mezcles.

### 3.6 `request/`

**Descripción:** capa de entrada HTTP + propagación de contexto de usuario/tenant + cliente HTTP saliente.

**API inbound (`request/request.go`):**
- `type Body` con `ToJson/ToItem/ToItems/ToArrayJson/ToString/ToInt/ToInt64/ToFloat/ToBool/ToTime`
- `func ReadBody(body io.ReadCloser) (*Body, error)`, `func GetBody(r *http.Request) (et.Json, error)`
- `func URLParam(r *http.Request, key string) *Value` (wrapper de `chi.URLParam`), `func Query(r *http.Request, key string) *Value`
- `type Value` con `.Str()/.Int()/.Float()/.Bool()/.DateTime()/.Object() et.Json/.Array() []any/.ArrayString()/.ArrayInt()/.ArrayFloat()/.ArrayJson()`

**Contexto (`request/ctx.go`):**
- Claves: `DurationKey, PayloadKey, ServiceIdKey, AppKey, DeviceKey, UserIdKey, UsernameKey, TenantIdKey, ProfileIdKey, TokenKey`
- Getters: `Duration(r), Payload(r), ServiceId(r), App(r), Device(r), Username(r)` (default `"Anonimo"`, no "Anonymous" — locale español consistente), `UserId(r), TenantId(r), ProfileId(r)`
- Setters: `SetDuration/SetPayload/SetServiceId/SetApp/SetDevice/SetUserId/SetUsername/SetTenantId/SetProfileId(ctx, val) context.Context`

**Cliente HTTP saliente (`request/fetch.go`):**
- `func HttpWithContext(ctx, method, url string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status)` — implementación core
- `func Http(method, url string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status)`, `func Fetch(method, url string, header, body et.Json) (*Body, Status)`
- Verbos: `Post/Get/Put/Delete/Patch/Options(url, header[, body] et.Json) (*Body, Status)` + variantes `*WithTls`
- `func NewTlsConfig(caFile, certFile, keyFile string) (*tls.Config, error)`
- `type Status struct { Ok bool; Code int; Message string }` con `.ToJson()/.ToString()` — **tipo distinto** de `et.Item`/`et.Json` usados en el resto de la librería para resultados; las llamadas salientes requieren una conversión manual (`body.ToItem()`, etc.) si quieres homogeneizar con el resto del código.

### 3.7 `jws/` (antes `ws/`)

**Descripción:** WebSocket sobre `gorilla/websocket`, modelo de `Hub` con tópicos/colas/pila.

**API:**
- `func New() *Hub`, `func Upgrader(w, r) (*websocket.Conn, error)` (con `CheckOrigin` siempre `true`)
- `(*Hub) Start()/Close()`
- `(*Hub) Connect(socket *websocket.Conn, ctx context.Context) (*Client, error)` — dedupe por username (`ctx.Value("username")`), reconexión reutiliza el mismo `Client`
- `(*Hub) Topic/Queue/Stack(channel string) *Channel` — broadcast / round-robin single-delivery / round-robin reverso
- `(*Hub) Publish(channel string, message Message) ([]string, error)`, `(*Hub) SendTo(to []string, message Message) ([]string, error)`
- `(*Hub) Subscribe/Unsubscribe(channel, subscribe string) error`
- Callbacks: `OnListener, OnConnection, OnDisconnection, OnChannel, OnRemove, OnPublish, OnSend`
- `Client` (`jws/client.go`): `read()`/`write()` loops, `Send`, `SendMessage`, `SendError`, `SendHola`

---

## 4. Validación

### 4.1 `jval/` (`jval/validate.go`)

**`Rule` interface:** `Validate(et.Json) error`, `Name() string`.

Constructores tipados y encadenables:
- `Str(name) *StringRule` + `.NotEmpty()`
- `Int(name) *IntRule` + `.Min(v)/.Max(v)`
- `Float(name) *FloatRule` + `.Min(v)/.Max(v)`
- `Array(name) *ArrayRule` + `.NotEmpty()`
- `Email(name) *EmailRule` (usa `net/mail.ParseAddress`)
- `Date(name) *DateRule` + `.Layout(layout)` (default `"2006-01-02"`)
- `Enum(name string, vals ...string) *EnumRule`
- `Phone(name) *PhoneRule` + `.CountryCode(code)/.Length(n)` (regex E.164-ish)
- `Between(name string, min, max float64) *BetweenRule`
- `Validate(name string, rules ...Rule) *ObjectRule` (objeto anidado — nombre de función igual al método `Rule.Validate`, colisión de nombre legal en Go pero confusa)

**No existen constructores `Bool`/`Time`** pese a lo que sugería documentación previa — solo `Date` (basado en layout de string).

**Invocación:**
- `func Require(data et.Json, rules ...Rule) error` — todas las reglas son obligatorias, retorna el primer error
- `func Maybe(data et.Json, rules ...Rule) error` — **bug/comportamiento a verificar**: si el primer campo de la lista está ausente en `data`, retorna `nil` inmediatamente **sin seguir evaluando los campos siguientes** de esa misma llamada — no es "salta el que falta y continúa", es "corta al primer ausente". Si necesitas validar varios campos opcionales de forma independiente, llama `Maybe` una vez por campo en vez de agrupar reglas.

Mensajes de error centralizados en el paquete raíz `msg/` (§14).

### 4.2 `validator/`

**Descripción:** un segundo paquete de validación, **no relacionado** con `jval/` (tipos distintos, sin interfaz compartida).

- `type Validator struct { Fields map[string]*Field }`, `func New() *Validator`
- `type Field struct { Name string; Condition *Condition }`
- `type Condition struct { required bool; min, max float64; minLength, maxLength int; pattern, name string; validator *Validator }` con builder chainable (`.NotEmpty()`, `.Min()`, `.Max()`, etc. — mismos nombres de método que `jval` por convención, pero tipos distintos)
- `validator/msg.go`: constantes i18n (`MSG_VALIDATOR_REQUIRED`, etc.), seleccionadas en `init()` según env `LANG` (`"es"` activa mensajes en español; por defecto ya están en español en este archivo, revisar si se agregan más idiomas)

No mezcles instancias/tipos de `jval` y `validator` en la misma validación — son independientes.

---

## 5. Infraestructura

### 5.1 `cache/`

**Descripción:** cliente Redis con operaciones tipadas, pub/sub, colecciones, métricas.

**API destacada:**
- `func Load() error` (idempotente) · `func IsLoad() bool` · `func New() (*Conn, error)` (constructor de bajo nivel; requiere `REDIS_HOST`, valida `runtime.GOOS` ∈ `{linux,darwin,windows}`)
- **Bug confirmado**: `(*Conn) Close()` se llama a sí mismo en vez de cerrar el `redis.Client` embebido — recursión infinita / stack overflow. No lo uses para shutdown ordenado.
- `HealthCheck()`, `FromId()`
- `Set/SetObject(key, val, expiration)`, `Get(key, def) (string, error)`, `GetObject(key, dest) (bool, error)`, `Exists(key) bool`, `Delete(key) (int64, error)`
- `Incr/Decr`, `Expire`, `LPush/LRem/LRange/LTrim` (listas)
- `SetH/SetD/SetW/SetM/SetY` (atajos de expiración por hora/día/semana/mes/año)
- `CollectionSet/Get/Put/Find/Delete` (hash de Redis)
- `ObjetSet/Get/Delete/All` (sic — "Objet", no "Object"; objetos JSON)
- `AllCache(search string, page, rows int) (et.List, error)` — SCAN completo + paginación en memoria
- `GetJson/GetItem/GetItems(key) (et.Json/Item/Items, error)`
- `SetVerify/GetVerify/DeleteVerify(device, key, ...)` — patrón OTP get-then-delete
- `Pub(channel, message []byte) error` / `Sub(channel, f func(*redis.Message))` / `Unsub(channel) error`
- `type Metrics` + `CallMetrics(key string, limit int64) (Metrics, error)` — contadores de rate-limit por segundo/minuto/hora/día
- HTTP: `GET /cache`, `GET /cache/{key}`, `DELETE /cache` vía `LoadRouter(r Router)`

**Env vars:** `REDIS_HOST` (requerido), `REDIS_PASSWORD`, `REDIS_DB` (opcionales).

### 5.2 `event/`

**Descripción:** pub/sub sobre NATS con logging asíncrono y soporte de colas/pila.

**API:**
- `func Load() error` (idempotente, requiere `NATS_HOST`) · `Close()/IsLoad()/HealthCheck()`
- **Mismo bug que `cache`**: `(*Conn) Close()` se llama a sí mismo en vez del `nats.Conn.Close()` embebido.
- `func Publish(channel string, data et.Json) error`
- `func Subscribe(channel string, f func(Message)) error` — broadcast (marca `Myself` si el emisor es la propia conexión)
- `func Queue(channel, queue string, f func(Message)) error` — reparto entre workers (QueueSubscribe)
- `func Stack(channel string, f func(Message)) error` — alias de `Queue(channel, "stack", f)`
- `func Source(channel string, f func(Message)) error` — alias de `Subscribe`
- `func Unsubscribe(channel string) error`
- `func Log(event string, data et.Json)` / `Overflow(data)` / `Error(event string, err error) error` — logging asíncrono no bloqueante (buffer de 256, se descarta si está lleno)
- `type Message struct { CreatedAt time.Time; FromId, Id, Channel string; Data et.Json; Myself bool }`
- `Work(event string, data et.Json) et.Json` / `State(id string, status WorkStatus, data et.Json)` — convención de tracking de trabajo sobre pub/sub, no persistida

### 5.3 `graph/`

**Descripción:** wrapper mínimo del driver Neo4j — prácticamente un stub.

**API (`graph/graph.go`, archivo completo, ~33 líneas):**
```go
type Conn struct { driver neo4j.DriverWithContext; id, host string }  // todos los campos sin exportar
func Load() (*Conn, error) {
    // URL y credenciales hardcodeadas:
    // "neo4j://localhost:7687", neo4j.BasicAuth("neo4j", "password", "")
}
```
**No lee ninguna variable de entorno** (`NEO4J_HOST`/`USER`/`PASSWORD` no existen en el código, pese a documentación previa) y **`*Conn` no tiene ni un solo método** — ni consulta, ni sesión, ni `Close()`. Para usar Neo4j de verdad hoy hay que usar `neo4j-go-driver/v5` directamente.

---

## 6. Entorno y logging

### 6.1 `envar/`

**El único paquete de acceso a entorno** — `config/` fue eliminado por completo.

- `GetStr/GetInt/GetInt64/GetFloat/GetBool(name, def)` — lectura de env vars con default
- `Get(name string, def interface{}) interface{}` / `Set(name string, value interface{}) interface{}` — acceso a nivel de proceso, con un `_store` interno opcional pluggable
- `ArgStr/ArgInt/ArgInt64/ArgFloat64/ArgBool(name, defaultVal)` — lectura de argumentos CLI
- `Validate(keys []string) error`

### 6.2 `logs/`, `stdrout/`, `color/`

- `logs/`: `Log, Info(f), Alert(f), Error(f), Debug(f), Fatal, Panic, Tracer`.
- `stdrout/`: `type Stdout interface { Notify(kind, message string) }`; `SetStdout(v Stdout)`; `Color(s *string, color, format string, args ...) *string`; `CW(w io.Writer, color []byte, format string, args ...)`; `Printl(kind, color string, args ...any) string`; `Traces/ErrorTraces(err error)`; constantes ANSI y `GetFunctionName(idx int) string`.
- `color/`: `Purple/Green/Red/Yellow/Blue/Cyan/White/Black(str string) string` — envuelven con ANSI + reset.

---

## 7. Identidad y seguridad

### 7.1 `claim/`

- `type Claim struct { jwt.StandardClaims; ID, Salt string; Duration time.Duration; App, Device, UserId, Username, TenantId, ProfileId string; Payload et.Json }`
- `func NewClaim(duration time.Duration) *Claim`
- `func NewToken(app, device, userId, username, tenantId, profileId string, payload et.Json, duration time.Duration) (string, error)` — HS256 vía `golang-jwt/jwt/v4`; el firmado real ocurre en una función **privada** `genToken(c *Claim, secret string) (string, error)` — **no existe un `GenToken` exportado** pese a documentación previa.
- `func ParceToken(token string) (*Claim, error)` — sic, "Parce" no "Parse", typo consistente en todo el repo (también en `jwt.go`)
- Secreto: env `SECRET` (default `"1977"`, inseguro) leído una sola vez vía `sync.Once` en `getSecret()`.

### 7.2 `jwt/`

- `func NewToken(app, device, userId, username, tenantId, profileId string, payload et.Json, duration time.Duration) (string, error)` — requiere `cache.IsLoad()`; delega el firmado a `claim.NewToken`, guarda el token en cache bajo clave `"app:device:username"` con la misma TTL (habilita revocación server-side)
- `func NewAuthentication(app, device, userId, username string, duration time.Duration) (string, error)` — sin tenant/perfil
- `func NewAuthorization(app, device, userId, username, tenantId, profileId string, duration time.Duration) (string, error)` — con tenant/perfil
- `func NewAppToken(app, device string, duration time.Duration) (string, error)` — token de aplicación (usa `app` como userId y username)
- `func NewEphemeralToken(...) (string, error)` — duración forzada a máx. 15 min
- `func Validate(token string) (*claim.Claim, error)` — valida firma y confirma contra el valor cacheado (revoca si no coincide)
- `func RenewToken`, `GetToken`, `DeleteToken(app, device, username) error`, `DeleteTokeByToken(token) error` (logout)

### 7.3 `reg/`

**Solo generación de IDs — no hay descubrimiento de servicios** pese al nombre del paquete.
- `UUID() / ULID() / XID() string`
- `GenKey/GenUUId/GenULID/GenXID(tag string) string` (sic: `GenUUId`, no `GenUUID`)
- `GetUUID/GetULID/GetXID(id string) string` — si `id` es `""`/`"*"`/`"new"` genera uno nuevo, si no retorna `id` tal cual
- `TagUUID/TagULID/TagXID(tag, id string) string`
- `GenSnowflake() string` — no usa realmente el generador de `bwmarrin/snowflake` (solo su constante `Epoch` en `init()`) — vestigial/incompleto
- `GenIndex() int64` (timestamp en nanosegundos), `GenHashKey(args ...interface{}) string` (base64 de `GenKey`)

### 7.4 `utility/`

- IDs: `UUID()`, `GetOTP(length)`, `GetRandomString(length)`, `GenId(id)`, `GenKey(id)` (duplica el concepto de `reg.GenKey` con un set de "vacío" ligeramente distinto)
- Cripto: `Encrypt(value, cryptoType) (string, error)` (MD5/SHA1/SHA256/SHA512/AES; clave AES = env `SECRET`, default inseguro `"1977"`, sin validar longitud de clave AES-128/192/256), `DecryptoAES(value) (string, error)` (sic, "Decrypto" no "Decrypt"), `Hash(password) (string, error)`/`Match(hash, password) bool` (bcrypt costo 5), `Sha256`, `Md5`
- Validación: `ValidStr/ValidIn/ValidId/ValidKey/ValidInt/ValidNum/ValidName/ValidEmail/ValidPhone` (exactamente 10 dígitos, no internacional)/`ValidUUID/ValidCode/ValidWord`
- Misc: `App() context.Context` (contexto sensible a señales), `AppWait()` (bloquea hasta SIGINT/SIGTERM, usado por varios `cmd/*`), `More(tag string, expiration time.Duration) int64` (**bug**: siempre resetea el contador interno a 0 tras leerlo, así que el valor devuelto/almacenado nunca sube más allá de 0/1 pese a llamadas repetidas), `Contains/InStr/InInt`, `ExtractMencion` (@menciones), `Quote/Unquote/ParamQuote/Params`, `ToBase64/FromBase64`, `Normalize`, `Add[T any](slice, item) []T` genérico
- También: `algorithm.go` (`BinarySearch`, `Dijkstra`, `QuickSort` — utilidades genéricas, poco relacionadas con el resto), `list.go`, `math.go` (`DivNum/DivInt` división segura), `pid.go` (`GetPidByPort(port) int`, cross-platform)

### 7.5 `strs/`

`Format/FormatUppCase/FormatLowCase/FormatDateTime/FormatSerie`, `Contains/Replace/ReplaceAll/Change` (regex cacheado en `sync.Map`), `Name/Trim/NotSpace/DaskSpace` (sic, "Dask" no "Dash"), `Uppcase/Lowcase/Titlecase/Same`, `Split/GetSplitIndex` (soporta índice negativo), `Append/AppendAny/JoinQuoted`, `StrToTime/StrToBool/HtmlToText/RemoveAcents`, `MaskToken(token, length)`, `Parse(str, vars et.Json)` (reemplaza `{{key}}`).

---

## 8. Orquestación

### 8.1 `crontab/`

> **Rediseño**: la API pública documentada en versiones anteriores (`AddJob/AddOneShotJob/AddEventJob/DeleteJob/StartJob/StopJob` como métodos de instancia) **ya no existe**. Modelo orientado a eventos con singleton de paquete.

- `type Store interface { Set(collection, id, ownerId string, obj any) error; Get(collection, id string, dest any) (bool, error); Delete(collection, id string) error; Query(query et.Json) (et.Items, error) }` (misma forma que `jia.Store`/`resilience.Store`)
- `func New(tag string, store Store) (*Crontab, error)` — solo llama `event.Load()`; inicia `cron.New(cron.WithSeconds(), cron.WithLocation(loc))`
- `func Load(tag string, store Store) error` — crea el `*Crontab` vía `New` y lo guarda en un **singleton de paquete**; llama `eventInit()`
- `type Cron struct { DayOfWeek, Month, DayOfMonth, Hour, Minute string }` — spec **estructurada**, no un string crudo; `(*Cron).toString()` valida cada campo con regex
- `func CronJob(tag, ownerId string, spec Cron, repetitions int, params et.Json, fn func(params et.Json) error) error` (reemplaza `AddJob`)
- `func ScheduleJob(tag, ownerId string, spec time.Time, params et.Json, fn func(params et.Json) error) error` (una sola ejecución, reemplaza `AddOneShotJob`)
- `func HttpRemoveJob/HttpStopJob/HttpStartJob(w, r)` — publican eventos de control en vez de mutar el job directamente
- **Nada en el repo importa `crontab` actualmente.**

### 8.2 `jwf/` (workflows — sustituye a `workflow/`/`instances/`, eliminados)

- `type Store interface { Set(collection, id, ownerId string, obj any) error; Get(collection, id string, dest any) (bool, error); Delete(collection, id string) error; Query(collection string, query et.Json) (et.Items, error) }` (`jwf/store.go:12`) — **nótese el `collection` extra en `Query`; NO tiene `GenSerie`**, corrección importante vs. documentación previa
- `func New(store Store, userID string) (*WorkFlow, error)` — `cache.Load()` + `event.Load()`; `WorkFlow.ID = reg.UUID()`; registra auditoría `"new_workflow"` para `userID`
- `func Load(store Store, id string) (*WorkFlow, error)` — error si `store == nil`; carga por el propio ID vía `store.Get("workflows", id, &def)`, rehidrata flows/steps
- **No existe un mapa `WorkFlow.Instances`** — las instancias se crean bajo demanda (`newInstance`) o se cargan por ID vía `store.Get("instances", id, ...)`; un marcador en cache (`instance:<id>:status`, TTL = `Flow.TimeAwait`) solo detecta "ya está corriendo"
- `(*WorkFlow) NewFloW(tag, title, version, userId string) *Flow` (sic: "FloW")
- `(*Flow) Step(tag, title string, fn func(*Instance, et.Json) (et.Json, error)) *Flow` — primer `Step` se registra como `KindTrigger`; siguientes se conectan vía `Connection{Kind: PortOutput}`
- `(*Flow) Error(tag, version, title string, fn func(*Instance, et.Json) (et.Json, error)) *Flow` — puerto de error (`PortError`); si se llama antes de tener algún Step, guarda un error recuperable vía `(*Flow).IsError() error`
- `(*WorkFlow) Run(flowId, triggerTag, id, projectId, code string, ctx, tags et.Json, userId string) (et.Json, error)` — `id` vacío crea una instancia nueva (vía `reg.GetULID("")`)
- `(*Instance) SetParams(params et.Json) et.Json` — **existe y funciona**, contradice documentación previa que decía que se había eliminado
- Estados de `Instance`: `CREATED, PENDING, RUNNING, ROLLBACK, DONE, FAILED, CANCEL, STOP`
- Reintentos: en error de paso, si `Flow.TotalAttempts != 0`, llama `resilience.New(workflow.store)` + `LoadInstance(resilience.Params{..., Fn: instance.run, ...})` antes de seguir la conexión de puerto `PortError`
- `(*WorkFlow) LoadRouter(r Router)` — registra `httpGetStep/httpSetStep/httpUpdateStep/httpDeleteStep` (**implementados**) y `httpGetFlow/httpSetFlow/httpStatusFlow/httpDeleteFlow/httpGetInstance/httpDeleteInstance/httpRunInstance` (**cuerpo vacío**, sin implementar)
- Quirk menor: en `Flow.Step`/`Flow.Error`, el parámetro que llega a `newStep` como "userId" es en realidad `flow.ID` — no afecta la compilación ni la ejecución, solo el audit log de esos steps.

### 8.3 `resilience/`

- `type Store interface { Set(collection, id, ownerId string, obj any) error; Get(collection, id string, dest any) (bool, error); Delete(collection, id string) error; Query(query et.Json) (et.Items, error) }` (misma forma que `jia.Store`/`crontab.Store`)
- `func New(store Store) (*Resilience, error)` — solo llama `event.Load()`
- `type Params struct { Id, Tag, Description string; TotalAttempts int; Interval time.Duration; Tags et.Json; Fn interface{}; FnArgs []interface{} }` — sin `TenantId`/`OwnerId`/`UserId`
- `(*Resilience) LoadInstance(params Params) *Instance` — defaults `TotalAttempts=3`, `Interval=30s` si no se especifican
- `(*Instance) Run(userId string) ([]any, error)`

### 8.4 `jia/` (antes `ia/`)

- `type Store interface { Set(collection, id, ownerId string, obj any) error; Get(collection, id string, dest any) (bool, error); Delete(collection, id string) error; Query(query et.Json) (et.Items, error) }` (`jia/ia.go:25`, misma forma unificada) — **sin `store.go` separado**, definida directamente en `ia.go`
- `func New(tag string, store Store, userId string) (*Ia, error)` (`ia.go:58`) — solo `event.Load()`, **sin** `cache.Load()`; `Ia.ID = reg.UUID()`, sin `tenantId`; `OPENAI_API_KEY` vía `envar.GetStr`
- `func Load(id string, store Store) (*Ia, error)` (`ia.go:84`) — **caveat de orden de argumentos**: llama internamente `store.Get(id, packageName, &result)` — es decir, pasa el `id` recibido como si fuera el `collection` y el literal `"ia"` como si fuera el `id`, en el orden opuesto al resto del paquete (`save()` usa `Set("ia", s.ID, ...)`, orden `collection` primero). Verifica el call site si implementas un `Store` real para `jia`.
- Estructuras: `Ia{Agents, Participants, Conversations map[string]*...}`, `Agent{IaID, ID, Tag, Name, Skills map[string]Skill, ...}`, `Participant{IAID, ID, To, ConvID, ...}`, `Conversation{IAID, ID, Type, Messages []*Message, Participants, ...}`, `Message{IAID, ConversationID, ID, Type, Content, MessageStatuses, ...}`
- `Skill` interface (`Tag/Name/Description/Execute(ctx, input et.Json) (et.Json, error)`); `ApiSkill` implementación concreta (`Url, Method, Headers, Body et.Json`)
- `jia/router.go` — rutas HTTP para agentes/conversaciones/participantes
- `jia/sender.go` — `Sender` interface (`SendTextMessage(to, content string) (et.Item, error)`), usada por `Conversation.SendTextMessage`; **no hay setter exportado** hoy — efectivamente sin conectar a ninguna implementación real
- `jia/event.go`/`jia/msg.go` — constantes de evento e i18n
- **Sin `cmd/jia`** — a diferencia de `jwf`/`jrex`, no hay ejemplo runnable en `cmd/`
- **Otros bugs de orden de argumentos detectados**: `Agent.save()` usa la colección `"step"` en vez de `"agent"` (copia/pega de `jwf`); `Conversation.loadConversation` y `Message.delete()` también invierten el orden `(collection, id)` en algunas llamadas — revisa el call site exacto antes de asumir consistencia.

### 8.5 `jrex/`

- `func Load(tag string, store Store) (*Jrex, error)` — carga-o-crea un `Jrex` persistido bajo la colección `"jrex"`; `store = nil` usa por defecto un `FileStore` rooted en `./src`
- `type Store interface { Set(collection, id, ownerId string, obj any) error; Get(collection, id string, dest any) (bool, error) }` — **subconjunto** de 2 métodos (sin `Delete`/`Query`)
- `(*Jrex) Set(name string, value interface{}) *Jrex` — inyecta un binding Go en cada `Instance` nueva
- `(*Jrex) Run()` / `RunModule(tag)` — ejecuta el módulo `"index"` (u otro) en una `Instance` `goja` nueva
- `(*Jrex) RunDev()` — corre una vez y bloquea vía `utility.AppWait()`; con el `FileStore` por defecto, cambios de archivo bajo `BaseDir` disparan un re-run vía `file.NewWatcher` (hot reload)
- Globales JS: `console.*`, `ctx.*`, `fetch()`, `require()` (`jrex/wrapper.go`, `jrex/lib.go`)
- **No existen** modos `Develop`/`Production`/`Building` ni métodos `SetModule`/`GetModule`/`DeleteModule` en el código actual.

### 8.6 `service/`

- `func VerifyOTP(channel string, otp, createdBy string) (bool, error)`
- `func SendOTPSMS(tenantId, serviceId, sender, countryCode, phoneNumber string, length int, duration time.Duration, createdBy string) (et.Items, error)`
- `func SendOTPEmail(...)`, `func SendOTPByTemplateId(...)`
- `func SendSms/SendWhatsapp/SendEmail/SendEmailByTemplateId(...)` — delegan en `aws`/`brevo`
- `type TpMessage` (enum: `TypeNotification, TypeComercial, TypeAutentication`)

---

## 9. Comunicación de bajo nivel

### 9.1 `jrpc/`

- `func Mount(host string, port int, services any, packageName string) (*Package, error)` — reflexiona la struct de servicios, la registra con `net/rpc` estándar (`rpc.Register`)
- `func Start(port int) error` — `rpc.ServeConn(conn)` por conexión entrante
- `func Call/CallJson/CallItems/CallItem(method string, args ...) (..., error)` — resuelven host/puerto vía el registro global de `Solver`s y llaman por `net/rpc` (gob), con `et.Json/Item/Items/List` pre-registrados vía `gob.Register`
- `func GetSolver(method string) (*Solver, error)`
- `type Solver struct { Host, Port string; Inputs, Output []string }` / `type Package struct { Name, Host, Port string; Solvers map[string]*Solver }`
- **Confirmado: sin balanceador de carga ni Raft** — eso vive en `jtcp/`.
- Código muerto detectado: `rpcs map[string]et.Json` se inicializa pero nunca se escribe; `listRouters()`/`HttpListRouters` siempre retorna lista vacía.

### 9.2 `jtcp/` (antes `tcp/`)

- `const (Follower, Candidate, Leader, Proxy Mode)` (iota)
- `func NewNode(port int, tlsConfig ...*tls.Config) *Node` — env `TIMEOUT` (default 10s), `WORKER_COUNT` (default 1000), `CONFIG_FILE` (default `./envar.json`)
- Elección de líder (`raft.go`): Raft simplificado hecho a mano — timeout aleatorio 1.5-3s sin heartbeat dispara elección, vota por sí mismo, pide votos a los peers; con mayoría pasa a `Leader` y emite heartbeats cada 500ms; cualquier peer viendo un término mayor vuelve a `Follower`. Callbacks `OnBecomeLeader`/`OnChangeLeader`.
- `balancer.go`: proxy TCP L4 round-robin simple, solo activo cuando `Node.mode == Proxy` — pipa bytes en ambas direcciones vía `io.Copy`.
- Protocolo de mensaje (`message.go`): frame con prefijo de longitud (4 bytes big-endian) + JSON `Message{ID, Type TpMessage, Method, Payload, Error, Args, IsResponse, Timeout}`; tipos `ACKMessage/CloseMessage/ErrorMessage/Heartbeat/RequestVote/Method`.
- Otros: `client.go` (Client con pending-requests por ID + timeout), `service.go` (`Service` interface + `Tcp.Ping` built-in), `tls.go` (mTLS/self-signed helpers), `cli.go` (`StartConsole`, REPL interactivo).
- Nota: `Response.Error` es tipo `error` con tag `json` — `encoding/json` no lo serializa útilmente (produce `{}`), solo el acceso directo al campo Go es confiable.

---

## 10. Integraciones externas

### 10.1 `aws/`

**Solo S3 y SNS/SMS — no hay SES** pese al nombre general del paquete.
- `type Params struct { Region, KeyId, Secret, Token string }`
- `func NewS3AWS(params Params) (*S3AWS, error)` → `(*S3AWS).Uploader/UploaderFile/UploaderB64/Delete/Download(...)`
- `func NewSenderAWS(params Params) (*SenderAWS, error)` → `(*SenderAWS).SendSMS(contactNumbers []string, content string, params et.Json, tpMessage string) (et.Item, error)` — `tpMessage` ∈ `{"Transactional","Promotional"}`, plantillado `{{key}}` vía `strs.Replace`
- Requiere las cuatro credenciales de `Params` no vacías para construir la sesión AWS.

### 10.2 `brevo/`

Pura capa HTTP vía `request.Fetch`/`request.Post` (sin SDK). Auth: env `BREVO_SEND_PATH` (base URL) + `BREVO_SEND_KEY` (header `api-key`).
- `func SendEmail(sender et.Json, to []et.Json, subject, htmlContent string, params et.Json, tp string) (et.Items, error)` + `SendEmailTransactional/SendEmailPromotional`
- `func SendSmsTransactional/SendSmsPromotional(...)` (interno `sendSms`)
- `func SendWhatsapp(contactNumbers []string, templateId string, params []et.Json, tp string) (et.Items, error)` + `SendWhatsappTransactional/SendWhatsappPromotional` — requiere además env `BREVO_SENDER`

### 10.3 `jwsp/` (antes `wsp/`)

- `func NewSender(token, phoneNumberId string) *Whatsapp` — env `WHATSAPP_API_URL` (default `https://graph.facebook.com/v22.0`)
- Builder: `.Debug()/.IsTest()` (modo test: no llama a la API real, retorna el body construido)/`.SetVerifyToken(...)/.SetEventHandler(fn)/.SetEventHandlerError(fn)`
- `(*Whatsapp) Webhooks(w, r)` — maneja el handshake GET de Facebook (`hub.mode`/`hub.verify_token`/`hub.challenge`) y el POST de conversación
- Decenas de `Send*`/`SendReply*` — texto, imagen, audio, video, documento, sticker, ubicación, contacto, plantillas, catálogo, listas/botones interactivos
- **Bug confirmado**: `SendReplyVideoMessageByURL(to, url, videoCaptionText string)` asigna `url` al campo `MessageID` del mensaje, en vez de recibir un `messageID` separado como su hermano correcto `SendReplyVideoMessageById(to, messageID, videoCaptionText, videoObjectID string)`.
- `jwsp/event.go` es un stub vacío (solo `package jwsp`, sin código).

---

## 11. Concurrencia y memoria

### 11.1 `mem/`

- `(*Mem) Set(key, value, expiration) (*Entry, error)`; getters tipados `(T, bool, error)`: `GetStr/GetInt/GetInt64/GetFloat/GetBool/GetTime/GetDuration/GetJson/GetArrayStr/GetArrayInt/GetArrayFloat/GetArrayJson`
- `(*Mem) More(key, expiration) (int64, error)` — contador atómico
- `(*Mem) Clear(match)`, `Empty()`, `Len()`, `Keys()`, `Values()`
- Package-level: mismas funciones delegando a un singleton (`mem/handler.go`)
- `mem/sync.go`: `Peticiones` — limitador de concurrencia (`NewPeticiones(capacity, timeWait)`, `(*Peticiones).Ejecucion(fn, params) (et.Items, error)`)

### 11.2 `ephemeral/`

`func NewInstance(expiration time.Duration) *Instance` — `.Set(key, value)` (resetea timer), `.Del(key)`, `.Get(key) (interface{}, bool)` (resetea timer al leer). Más simple que `mem/`: todo `interface{}`, sin accesores tipados.

### 11.3 `race/`

`func NewValue(value interface{}) *Value` — wrapper thread-safe con `sync.RWMutex`. `.Set/.Delete/.Get/.String/.Int/.Float64/.Bool/.Time/.Array/.Map/.StringArray/.IntArray/.Float64Array/.IsNil/.MapRange/.ArrayRange/.Range/.Increase(n)`.

### 11.4 `iterate/`

Medición de tiempo entre checkpoints vía singleton de paquete. `func Start(tag string)`, `func Segment(tag, msg string, isDebug bool) time.Duration`, `func End(tag, msg string, isDebug bool) time.Duration` — logging vía `logs.Infof`.

### 11.5 `queue/`

Cola de batching genérica en memoria. `func New[T any](queueSize, maxEvents int, period time.Duration, handler func(context.Context, []T) error) *Queue[T]` — `.Push(item)` agrupa ítems y despacha el lote a `handler` cuando ocurre primero: `maxEvents` alcanzado o `period` vencido. `.Close()` flushea y detiene. Tiene el único test real del repo (`queue/example_test.go`, `Example_queue`).

---

## 12. Tiempo, unidades y archivos

### 12.1 `timezone/`

Constantes de layout (`RFC3339Nano`, `RFC3339`, etc.). `func Now()/NowStr()/Add(d)/Location()/Format(t, layout)/Parse(layout, value) (time.Time, error)/FormatMDYYYY(value string) string`. Zona vía env `TIMEZONE` (default `America/Bogota`), formato por defecto vía `LAYOUT_TIME`.

### 12.2 `units/`

`type TypeUnity` (constantes de distancia/masa/volumen). `func NewQuantity(val float64, unit TypeUnity) *Quantity`, `(*Quantity).Load(val interface{}) error`, `.To(unit) error` (convierte in-place; masa↔volumen tratados 1:1 g/ml), `.ToStr()/.ToJson()`. `func Load(val any) (*Quantity, error)` (factory desde float/map/`et.Json`/string).

### 12.3 `file/`

`type FileInfo struct { Path string; Info os.FileInfo; Error error; IsDir, Exist bool }` + `.Json() et.Json`. `func ExistPath(path string) FileInfo`, `GetExtencion(filename) string` (sic, "Extencion"), `MakeFolder(names ...string) (string, error)`, `MakeFile(path, name, model string, args ...any) (string, error)` (sustituye `$1,$2...`), `Remove(path) (bool, error)`, `Save/Load[T any]/LoadOrSave[T any](path, obj) ...` (persistencia JSON genérica).

`type Watcher` (`fsnotify`-backed): `NewWatcher(root) (*Watcher, error)`, builder `.Debug()/.OnCreate/.OnWrite/.OnRemove/.OnRename/.OnChmod/.OnReload(fn)/.OnError(fn)/.Close()/.Load()` (bloqueante, auto-watch de subdirectorios nuevos). `func WatcherPath(root string) error` — atajo simple sin builder, usado por `cmd/whatcher`.

---

## 13. Herramientas de desarrollo

### 13.1 `cmds/`

`Stage{Id, Name, Description, Steps []*Step}`, `Step{Id, Name, Description, Commands []*Cmd}`. `(*Step).RunOS(idx, args et.Json) ([]byte, error)` ejecuta un comando local sustituyendo variables. `(*Step).RunSSH(idx, args et.Json) ([]byte, error)` es **idéntico a `RunOS`** — no implementa SSH real.

### 13.2 `create/`

Scaffolder Cobra + `promptui`, orientado a datos (`create/template/templates.go`, >1000 líneas de plantillas Go raw-string): microservicios, modelos, RPC, deployments de Kubernetes (StatefulSet/Deployment). Punto de entrada: comando `create.Create` (Cobra) → `PrompCreate()` → `MkProject`/`MkMicroservice`/`MkMolue`/`MkRpc`. **Nota de calidad**: la plantilla de `server.New("$2")` (un solo argumento) está desactualizada frente a la firma real `server.New(name string, port int) *Ettp` — si generas un servicio nuevo con `create/`, corrige esa llamada a mano.

### 13.3 `cmd/*` (binarios)

| Binario | Propósito | Dependencias principales |
|---|---|---|
| `cmd/et` | CLI principal, demuestra el API tipo LINQ de `et.Json` (`et.From(...).Where(...).Order(...).Join(...)`) | `et`, `logs` |
| `cmd/apigateway` | Arranca un API Gateway sobre `ettp.New` | `envar`, `ettp/v2`, `logs` |
| `cmd/daemon` | Servicio en background con integración systemd, patrón `Registry`/plugin | `et`, `os` |
| `cmd/create` | Shim CLI de `create/`, corre `go mod tidy` al final | `create`, `spf13/cobra` |
| `cmd/server` | Nodo TCP (`jtcp.NewNode(port)`) | `envar`, `jtcp`, `utility` |
| `cmd/jrex` | Runner de `jrex` en modo dev con hot-reload, conecta Postgres | `jrex`, `jsql` + driver postgres |
| `cmd/jsql` | Demo del driver `jsql` (DDL, condiciones, consultas) | `jsql` + driver postgres |
| `cmd/client` | Cliente TCP de consola (`jtcp.StartConsole`) | `envar`, `jtcp` |
| `cmd/install` | Bootstrap: instala dependencias fijas + copia `LIBRARY_CONTEXT.md` al proyecto consumidor y lo referencia en su `CLAUDE.md` | stdlib |
| `cmd/whatcher` | Observa el directorio actual con `file.WatcherPath(".")` | `file` |
| `cmd/jwf` | Ejemplo de `jwf` (`NewFloW`/`Step`/`Run`) | `et`, `jwf` |
| `cmd/resilience` | Ejemplo de `resilience` | `et`, `resilience`, `utility` |
| `cmd/wsp` | Ejemplo de `jwsp` en modo test (conserva el nombre de directorio antiguo) | `jwsp` |

No existe `cmd/jia` — a diferencia de `jwf`/`jrex`, `jia` no tiene ejemplo runnable en el repo.

### 13.4 `jcli/` (huérfano, en progreso)

`jcli/jcli.go` implementa un modelo de CLI Bubble Tea (`cliModel`, interfaz `App{RunCli() error}`) pero **declara `package jrex`** estando en el directorio `jcli/`, y **nada en el repo importa `github.com/cgalvisleon/et/jcli`**. Parece ser una extracción en progreso del CLI de desarrollo de `jrex/` hacia su propio paquete, sin terminar de conectar.

### 13.5 `infobip/` (stub vacío)

`infobip/infobip.go` contiene únicamente `package infobip` — sin ningún código. Placeholder para una integración futura, no usable hoy.

---

## 14. Patrón transversal: `msg/`

El paquete raíz `msg/` (`msg/msg.go`) centraliza mensajes de error compartidos entre el resto de paquetes como constantes string (`MSG_*`): `MSG_ATRIB_REQUIRED`, `MSG_RECORD_NOT_FOUND`, `MSG_TOKEN_INVALID`, `MSG_TOKEN_EXPIRED`, etc. Es distinto de `et/msg.go` (§1), que cubre solo los mensajes internos del paquete `et`. La mayoría de los demás paquetes (`jia`, `jwf`, `resilience`, `crontab`, `claim`, `validator`, etc.) además mantienen su **propio** `msg.go` local con constantes específicas de ese paquete, siguiendo el mismo patrón. Al generar código que interactúa con `et`, prefiere comparar/propagar estos mensajes en vez de inventar nuevos strings literales equivalentes.
