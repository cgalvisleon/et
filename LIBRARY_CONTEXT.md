# LIBRARY_CONTEXT.md

> Documento de contexto persistente para asistentes de IA (Claude, ChatGPT, Cursor, Windsurf, Cline, etc.)
> Librería: **`github.com/cgalvisleon/et`** — Go 1.23+ — MIT
> Copia este archivo en la raíz de cualquier proyecto que dependa de `et` para que el asistente diseñe soluciones coherentes con la librería.

---

## 1. Resumen ejecutivo

`et` es una **librería modular de utilidades para construir microservicios, CLIs y aplicaciones web en Go**. No es un framework monolítico: es un conjunto de ~40 paquetes independientes que cubren, de punta a punta, las necesidades habituales de un backend:

- **Modelo de datos universal** (`et.Json`, `et.List`, `et.Item`, `et.Items`).
- **Persistencia** agnóstica de motor con ORM ligero (`jsql/`).
- **Servidores HTTP** en dos niveles de abstracción (`server/`, `ettp/v2`) + router con sincronización distribuida (`router/`).
- **Validación declarativa** de payloads (`jval/`).
- **Autenticación/JWT** (`jwt/`, `claim/`) y middlewares HTTP (`middleware/`).
- **Infraestructura**: Redis (`cache/`), NATS (`event/`), Neo4j (`graph/`).
- **Configuración y entorno** (`config/`, `envar/`).
- **Logging estructurado** (`logs/`).
- **Orquestación**: cron (`crontab/`), workflows multi-paso (`workflow/`), agentes IA (`ia/`), runtime JS embebido (`jrex/`).
- **Integraciones externas**: AWS (S3/SES/SMS), Brevo, WhatsApp Business (`wsp/`).
- **Utilidades transversales**: IDs (ULID/UUID/Snowflake), criptografía, validación de formatos, manejo de tiempo, etc.

**Idea central**: cualquier dato dinámico que entra o sale de un servicio (JSON de HTTP, filas de DB, mensajes de eventos, configuración) se representa como `et.Json` (`map[string]interface{}`) con accesores tipados y valores por defecto. Esto evita el patrón `val, ok := m["x"].(string)` en todo el código.

---

## 2. Filosofía de diseño

1. **"Json as lingua franca"**: `et.Json` es el tipo que cruza todas las capas — HTTP body, filas de base de datos (`_source JSONB`), mensajes NATS, configuración, claims JWT, resultados de workflows. Un solo tipo, un solo conjunto de accesores (`Str`, `Int`, `Bool`, `Time`, `Json`, `Array`, `ValStr(def, ...)`, etc.).
2. **Modularidad por import, no por imposición**: no hay "framework" que envuelva la app. Cada paquete se importa solo si se necesita. `jsql` no requiere `cache`; `cache` no requiere `event`; `ettp/v2` sí necesita ambos (lo declara explícitamente).
3. **`Load()` idempotente**: los paquetes de infraestructura (`cache`, `event`, `config`, `crontab`, `workflow`, `ia`) exponen `Load()` (o `New()`), seguro de llamar múltiples veces, que lee variables de entorno vía `envar` y establece conexiones una sola vez.
4. **Inversión de dependencias vía interfaces pequeñas**: la librería define interfaces (`jsql.Driver`, `jsql.Config`, `instances.Store`, `jrex.Store`, `config.Store`) y el consumidor las implementa. `stores/` ofrece una implementación lista basada en `jsql` para `instances.Store`.
5. **APIs fluidas / encadenables**: `model.Where(...).And(...).Limit(20).Page(1).All()`, `jval.Validate("user", jval.Str("email").NotEmpty())`, `flow.NewSteper(...)`.
6. **Agnosticismo de driver**: `jsql` define el contrato (`Driver` interface) y los drivers (`postgres`, `sqlite`) se auto-registran con `init()` al importarse como side-effect.
7. **Esquema híbrido relacional/documental**: las tablas tienen columnas reales (`COLUMN`) y atributos dentro de una columna `_source JSONB` (`ATTRIB`), permitiendo evolución de esquema sin migraciones constantes, sin perder capacidad de consulta SQL (`_source->>'campo'`).
8. **Mensajes de error centralizados**: cada paquete tiene `msg/` o `msg.go` con constantes de error — nunca strings literales repetidos.
9. **Respuestas HTTP unificadas**: toda respuesta de API pasa por `response.ITEM` / `response.ITEMS` / `response.DATA` / `response.HTTPError`, envolviendo `et.Item` / `et.Items` / `et.Json`.
10. **Documentación de código estandarizada**: comentarios de funciones en bloque `/** ... @param ... @return ... **/` (ver `CLAUDE.md`).

---

## 3. Principios arquitectónicos

- **Sin punto de entrada central**: no existe un `et.App` que arranque "todo". Cada servicio compone los paquetes que necesita.
- **Capas**:
  1. *Utilidades autosuficientes* (`et`, `utility`, `strs`, `reg`, `jval`, `logs`, `config`, `envar`, `mem`, `ephemeral`, `iterate`, `timezone`, `units`, `color`, `race`, `file`) — sin dependencias externas de servicios.
  2. *Infraestructura* (`cache` → Redis, `event` → NATS, `graph` → Neo4j, `jsql` → SQL) — requieren servicios externos vía variables de entorno.
  3. *HTTP / routing* (`server`, `ettp/v1`, `ettp/v2`, `router`, `middleware`, `response`, `request`) — construidos sobre `chi`; `ettp/v2` añade sincronización de rutas vía NATS.
  4. *Capa de aplicación* (`workflow`, `ia`, `jrex`, `crontab`, `service`) — orquestación de negocio, construida sobre las capas 1–3.
  5. *Integraciones externas* (`aws`, `brevo`, `wsp`, `graph`).
- **Sincronización entre réplicas vía eventos**: `ettp/v2` sincroniza el estado del router entre instancias usando eventos NATS (`EVENT_SET_ROUTER`, `EVENT_REMOVE_ROUTER`, `EVENT_RESET_ROUTER`), con bandera `m.Myself` para evitar bucles de auto-procesamiento. `router/` usa el mismo patrón de forma standalone.
- **Patrón Store inyectado**: `workflow.Load(store instances.Store)`, `ia.New(tenantId, tag, store, config)`, `jrex.New(name, store)` — la librería define el contrato de persistencia, el consumidor decide el backend (`stores/` ofrece uno basado en `jsql`).
- **Patrón Driver auto-registrado**: `import _ "github.com/cgalvisleon/et/jsql/drivers/postgres"` registra el driver vía `init()`; `jsql.Load(config)` lo resuelve por `DB_DRIVER`.
- **Debug/Test transversal**: `Model`, `Query` y `Command` de `jsql` soportan `.Debug()` (loguea SQL sin ejecutar) y `.Test()` (devuelve SQL sin ejecutar) — útil para pruebas sin DB real.
- **Contexto de request enriquecido**: `request/ctx.go` propaga `tenantId`, `userId`, `username`, `profileId`, `app`, `device`, `payload` a través de `context.Context`, leídos por middlewares y handlers (`request.TenantId(r)`, `request.UserId(r)`, etc.).

---

## 4. Componentes principales

### 4.1 Núcleo de datos (`et/`)

| Tipo | Propósito | API clave |
|---|---|---|
| `et.Json` (`map[string]interface{}`) | Tipo universal de datos | `Str`, `Int`, `Int64`, `Num`, `Bool`, `Time`, `Json(attr)`, `Array`, `ArrayStr/Int/Json`, `Map*`, `ValStr/ValInt/.../ValJson(def, ...atribs)`, `Get`, `Set`, `SetNested`, `Delete`, `Exist`, `Select`, `Hidden`, `Clone`, `Update`, `Compare`, `Append`, `IsChanged`, `ToByte/ToString/ToMap` |
| `et.List` | Resultado paginado | `Rows`, `All`, `Count`, `Page`, `Start`, `End`, `Result []Json`, `ToJson/ToString/ToMap` |
| `et.Item` | Resultado de un registro (`Ok bool`, `Result Json`) | mismos accesores tipados que `Json` (`Str`, `Int`, `Bool`, `Json`, `Array...`), `NewItem(data)` |
| `et.Items` | Resultado multi-registro | `Add`, `AddMany`, `One(idx)`, `First`, `Last`, `ToList(all, page, rows)`, accesores indexados (`Str(idx, ...)`, `Int(idx, ...)`, etc.) |

> **Regla de oro**: si vas a leer/escribir datos dinámicos (JSON, filas de DB, payloads), usa `et.Json` y sus accesores — **nunca** `map[string]interface{}` a mano ni *type assertions* manuales.

### 4.2 Persistencia (`jsql/`, `stores/`, `instances/`)

- **`jsql/`** — SQL builder agnóstico + ORM ligero. Entradas: `jsql.Load(config)` / `jsql.LoadTo(config, name)`. Drivers: `postgres`, `sqlite` (auto-registro vía `init()`).
- Modelos: `db.DefineModel(schema, name, version)` (full: agrega `id`, `created_at`, `updated_at`, `_source`, `_idx`), `db.NewModel(...)` (manual), o `db.Define(jsql.Def{...})` (declarativo, preferido para modelos complejos).
- Columnas: `COLUMN`, `ATTRIB` (dentro de `_source` JSONB), `DETAIL`/`ROLLUP` (relaciones virtuales), `CALCFUNC`/`CALC` (computadas), `AGG` (agregaciones).
- Consultas/comandos fluidos: `.Where(jsql.Eq(...)).And(...).Limit().Page().All()/.One()`, `.Insert(...)`, `.Update(...)`, `.Upsert(...)`, todos con `.ExecTx(tx)`.
- Triggers: `beforeInserts/Updates/Deletes`, `afterInserts/Updates/Deletes` (`TriggerFunction`), columnas calculadas vía `CalcFunction`.
- Paths anidados JSONB: `"ventas->detalle->precio"` se traduce automáticamente a `->`/`->>` con casts.
- **`instances.Store`** — interfaz (`Set`, `Get`, `Delete`, `Query`) usada por `workflow`, `ia`, `jrex` para persistir estado.
- **`stores/`** — implementación de `instances.Store` basada en `jsql`: `stores.NewInstance(db, schema, name, kind)` (`KindJson` / `KindBite`).

### 4.3 HTTP y routing

| Paquete | Nivel | Cuándo usarlo |
|---|---|---|
| `server/` | Ligero (`Ettp` sobre `chi.Mux`) | Servicios sin Redis/NATS. `server.New(name, port)`, `.Use(...)`, `.HandleFunc`, `.Mount`, `.Start()` |
| `ettp/v2` | Completo | `ettp.New(name, cfg)` — llama `cache.Load()` + `event.Load()`; router sincronizado entre réplicas vía NATS; `.Public(method, path, handler, pkg)`, `.Private(...)`, `.UseAutentication(mw)`, `.SetRouter(...)` |
| `ettp/v1` | Versión anterior | Prefiere `v2` |
| `router/` | Standalone | Routing con sincronización NATS sin el resto de `ettp` |
| `middleware/` | Transversal | `Authenticate`, `Logger`/`RequestLogger`, `Recoverer`, `RequestID`, `AllowAll` (CORS), `Metrics`/`PushTelemetry*` |
| `response/` | Salida | `ITEM`, `ITEMS`, `DATA`, `JSON`, `RESULT`, `ANY`, `Stream`, `HTTPError`, `HTTPAlert`, `Unauthorized`, `Forbidden`, `InternalServerError` |
| `request/` | Entrada | `GetBody(r)`, `URLParam(r, key).Str()/.Int()/.../Object()/.ArrayJson()`, `Query(r, key)`, contexto: `TenantId`, `UserId`, `Username`, `ProfileId`, `App`, `Device`, `Payload`, `ServiceId`, `SetXxx(ctx, ...)` |

**Patrón de handler estándar** (ver §6 para ejemplo completo):

```go
func (s *T) HttpFoo(w http.ResponseWriter, r *http.Request) {
    id := request.URLParam(r, "id").Str()
    body, err := request.GetBody(r)
    if err != nil {
        response.HTTPError(w, r, http.StatusBadRequest, err.Error())
        return
    }
    response.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: data})
}
```

### 4.4 Validación (`jval/`)

Validadores tipados, encadenables, que operan sobre `et.Json`:

```go
err := jval.Require(body,
    jval.Str("email").NotEmpty(),
    jval.Email("email"),
    jval.Int("age").Min(18).Max(120),
    jval.Enum("role", "admin", "user"),
    jval.Array("tags").NotEmpty(),
)
```

Reglas disponibles: `Str`, `Int`, `Float`, `Array`, `Email`, `Date(layout)`, `Enum(name, vals...)`, `Phone` (`CountryCode`, `Length`), `Between(min,max)`, `Validate(name, rules...)` (objetos anidados). Helpers: `jval.Require(data, rules...)` (todas obligatorias), `jval.Maybe(data, rules...)` (valida solo si el campo existe).

### 4.5 Autenticación (`jwt/`, `claim/`)

- `claim.Claim` — claims JWT con `tenantId`, firmado HS256 (clave: env `SECRET`, default `"1977"`).
- `jwt.NewAuthentication(app, device, userId, username, duration)` / `jwt.NewAuthorization(app, device, userId, username, tenantId, profileId, duration)` / `jwt.NewAppToken(app, device, duration)` / `jwt.NewEphemeralToken(...)`.
- Tokens almacenados en `cache` (Redis); `jwt.Validate(token)`, `jwt.RenewToken`, `jwt.DeleteToken`, `jwt.DeleteTokeByToken`.
- `middleware.Authenticate` — valida el Bearer token y puebla el contexto del request.

### 4.6 Infraestructura

- **`cache/`** (Redis, `REDIS_HOST`): `cache.Load()`, `Set/Get/Delete`, `SetWithDuration`, `SetH/D/W/M/Y` (atajos de expiración hora/día/semana/mes/año), `Incr/Decr`, `LPush/LRange/LTrim`, `SetObject/GetObject` (serialización automática), `Collection*` (hashes), `Pub/Sub` vía `event`.
- **`event/`** (NATS, `NATS_HOST`): `event.Load()`, `Publish(channel, data)`, `Subscribe(channel, fn)`, `Queue(channel, queue, fn)` (balanceo de carga), `Stack`/`Source`, `Work`/`State` (seguimiento de tareas).
- **`graph/`** (Neo4j): `graph.Load()` → `*Conn` con driver Neo4j.

### 4.7 Configuración y entorno

- **`config/`** — `config.New(tag, stage, tenantId, ownerId, store, userId)` / `config.Load(...)`; getters `GetStr/GetInt/GetInt64/GetFloat/GetBool/GetMap` con default; `Set`, `Save`, `Remove`.
- **`envar/`** — acceso a variables de entorno y parámetros CLI; `envar.Validate([]string{...})` verifica variables requeridas.

### 4.8 Logging (`logs/`)

`logs.Info`, `Infof`, `Alert`/`Alertf`/`Alertm`, `Error`/`Errorf`/`Errorm`, `Warn`, `Debug`, `Tracer`, `Fatal` (→ `os.Exit(1)`), `Panic`. Salida colorizada vía `stdrout`. `logs.EnableCallerInfo = false` recomendado en producción.

### 4.9 Orquestación

- **`crontab/`** — `crontab.New(tag)` (llama `event.Load()`); `AddJob`, `AddOneShotJob`, `AddEventJob`, `AddOneShotEventJob`, `DeleteJob`, `StartJob/StopJob/Stop`. Soporta spec con segundos (`"0 * * * * *"`).
- **`workflow/`** — `workflow.Load(store)` / `workflow.New(store)`. Jerarquía `Flow → Steper → Step` + `Instance` en runtime. `RunInstance`, `GetInstance`, `ResetInstance`, `RollbackInstance`, `StopInstance`. Handlers HTTP listos (`HttpRunInstance`, etc.).
- **`ia/`** — `ia.New(tenantId, tag, store, config)` / `ia.Load(...)` (singleton). Gestiona `Agent`, `Participant`, `Conversation`/`Message` por tenant. `Skill` (p.ej. `ApiSkill`) permite que agentes llamen APIs externas. Requiere `OPENAI_API_KEY`.
- **`jrex/`** — runtime JS embebido (`goja`). Modos `Develop` (hot-reload), `Production` (carga desde `Store`), `Building` (compila + versiona). Expone `console.*`, `ctx.*`, `fetch()`, `require()`.

### 4.10 Integraciones externas

- **`aws/`** — S3, SES (email), SMS.
- **`brevo/`** — email, SMS, WhatsApp vía Brevo.
- **`wsp/`** — WhatsApp Business API (Graph API), `NewWhatsapp(token, phoneNumberId)`.
- **`service/`** — OTP (`SendOTPEmail`, `SendOTPSms`, `VerifyOTP`) multi-tenant.

### 4.11 Utilidades transversales

- **`reg/`** — generación de IDs: `UUID()`, `ULID()`, `XID()`, `GenULID/GenUUId/GenXID(tag)`, `GenSnowflake()`, `GenHashKey`, `TagULID/TagUUID/TagXID(tag, id)`.
- **`utility/`** — `ValidEmail/ValidPhone/ValidUUID/ValidName/...`, `GetRandom`, `GetOTP`, `Encrypt`/`DecryptoAES` (MD5/SHA1/SHA256/SHA512/AES), `Now()`, `TimeDifference`, `Contains`.
- **`strs/`** — utilidades de strings.
- **`mem/`** — caché en memoria con expiración y primitivas de sincronización.
- **`ephemeral/`**, **`iterate/`**, **`race/`**, **`timezone/`**, **`units/`**, **`color/`**, **`file/`** — soporte transversal (datos temporales, iteración con control de tiempo, detección de condiciones de carrera, husos horarios, conversión de unidades, color de terminal, watcher de archivos).

### 4.12 Redes y CLI

- **`tcp/`** — nodo TCP distribuido con elección de líder estilo Raft (`Follower/Candidate/Leader/Proxy`); `tcp.NewNode(port)`.
- **`jrpc/`** — `net/rpc` sobre TCP con balanceo de carga y consenso Raft.
- **`ws/`** — WebSocket bidireccional (`gorilla/websocket`).
- **`cmd/`** — binarios independientes (`et`, `apigateway`, `daemon`, `server`, `jrex`, `jsql`, `client`, `create`, `install`, `whatcher`).
- **`create/`** — plantillas de scaffolding para microservicios y despliegues Kubernetes.

---

## 5. Casos de uso comunes

| Necesidad | Componente(s) | Notas |
|---|---|---|
| API REST con persistencia | `ettp/v2` o `server` + `jsql` + `response` + `request` | Patrón handler estándar (§4.3) |
| Validar payloads de entrada | `jval` | `Require`/`Maybe` en el primer paso del handler |
| Login / sesiones | `jwt` + `claim` + `middleware.Authenticate` + `cache` | Tokens en Redis |
| Multi-tenant | `request.TenantId(r)`, `claim.Claim.tenantId`, `jsql.TENANT_ID` | Propagado por contexto |
| Cache de resultados / rate limiting | `cache` (`SetWithDuration`, `Incr`) | Atajos `SetH/D/W/M/Y` |
| Comunicación entre microservicios | `event` (pub/sub NATS) o `jrpc` (RPC balanceado) | `event.Queue` para balanceo tipo cola |
| Tareas programadas | `crontab` | Soporta segundos y jobs basados en eventos |
| Llamadas a APIs externas | `request.Get/Post/Put/Delete/Patch` (+ `WithTls`) | Devuelve `*request.Body, request.Status` |
| Procesos de negocio multi-paso con rollback | `workflow` | `Flow → Steper → Step`, `RollbackInstance` |
| Agentes conversacionales / IA | `ia` + `stores` (Store sobre `jsql`) | Requiere `OPENAI_API_KEY` |
| Scripts dinámicos / lógica configurable en runtime | `jrex` | Hot-reload en dev, `Store` en producción |
| Notificaciones (email/SMS/WhatsApp) | `aws`, `brevo`, `wsp`, `service` (OTP) | Elegir según proveedor |
| IDs únicos (orden temporal, distribuidos) | `reg.ULID()`, `reg.GenULID(tag)` | `_idx` de `jsql` usa ULID, no secuencias |
| Configuración por entorno/tenant | `config` + `envar` | `config.New(...)` por tenant si aplica |

---

## 6. Ejemplos de integración

### 6.1 Microservicio mínimo con `server/` + `jsql` + `jval` + `response`

```go
package main

import (
    "net/http"

    "github.com/cgalvisleon/et/et"
    "github.com/cgalvisleon/et/jsql"
    _ "github.com/cgalvisleon/et/jsql/drivers/postgres"
    "github.com/cgalvisleon/et/jval"
    "github.com/cgalvisleon/et/logs"
    "github.com/cgalvisleon/et/request"
    "github.com/cgalvisleon/et/response"
    "github.com/cgalvisleon/et/server"
)

var users *jsql.Model

func main() {
    db, err := jsql.Load(nil) // lee DB_* desde env
    if err != nil {
        logs.Fatal(err)
    }

    users, err = db.Define(jsql.Def{
        Schema:  "public",
        Name:    "users",
        Version: 1,
        IdxField: jsql.IDX,
        Columns: []jsql.Column{
            {Name: "email", TypeData: jsql.TEXT, Default: ""},
            {Name: "name", TypeColumn: jsql.ATTRIB, TypeData: jsql.TEXT, Default: ""},
        },
        Unique: []jsql.DefIndex{{Name: "email"}},
    })
    if err != nil {
        logs.Fatal(err)
    }
    if err := users.Init(); err != nil {
        logs.Fatal(err)
    }

    srv := server.New("users-svc", 8080)
    srv.HandleFunc("/users", httpCreateUser)
    srv.HandleFunc("/users/{id}", httpGetUser)
    srv.Start()
}

func httpCreateUser(w http.ResponseWriter, r *http.Request) {
    body, err := request.GetBody(r)
    if err != nil {
        response.HTTPError(w, r, http.StatusBadRequest, err.Error())
        return
    }

    if err := jval.Require(body,
        jval.Email("email"),
        jval.Str("name").NotEmpty(),
    ); err != nil {
        response.HTTPError(w, r, http.StatusBadRequest, err.Error())
        return
    }

    item, err := users.Insert(et.Json{
        "email": body.Str("email"),
        "name":  body.Str("name"),
    }).ExecTx(nil)
    if err != nil {
        response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
        return
    }

    response.ITEM(w, r, http.StatusCreated, item)
}

func httpGetUser(w http.ResponseWriter, r *http.Request) {
    id := request.URLParam(r, "id").Str()

    item, err := users.Where(jsql.Eq("id", id)).One()
    if err != nil {
        response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
        return
    }
    if item.IsEmpty() {
        response.HTTPError(w, r, http.StatusNotFound, "user not found")
        return
    }

    response.ITEM(w, r, http.StatusOK, item)
}
```

### 6.2 Autenticación con `jwt` + `middleware`

```go
token, err := jwt.NewAuthorization("my-app", "web", userId, username, tenantId, profileId, 24*time.Hour)
// ...
srv.Use(middleware.Authenticate) // valida Bearer token y puebla el contexto

func httpProtected(w http.ResponseWriter, r *http.Request) {
    tenantId := request.TenantId(r)
    userId   := request.UserId(r)
    response.DATA(w, r, http.StatusOK, et.Json{"tenantId": tenantId, "userId": userId})
}
```

### 6.3 Eventos entre servicios

```go
event.Load() // requiere NATS_HOST

event.Subscribe("user.created", func(msg event.Message) {
    data, _ := msg.ToJson()
    logs.Infof("nuevo usuario: %s", data.Str("email"))
})

event.Publish("user.created", et.Json{"email": "a@b.com", "tenantId": tenantId})
```

### 6.4 Llamadas HTTP salientes

```go
body, status := request.Post("https://api.externa.com/v1/orders",
    et.Json{"Authorization": "Bearer " + token},
    et.Json{"sku": "ABC", "qty": 2},
)
if !status.Ok {
    return fmt.Errorf("error externo: %s", status.ToString())
}
result, err := body.ToJson()
```

---

## 7. Buenas prácticas

1. **Usa `et.Json` y sus accesores tipados** (`Str`, `Int`, `ValStr(def, ...)`, etc.) para cualquier dato dinámico — nunca *type assertions* manuales sobre `map[string]interface{}`.
2. **Define modelos con `db.Define(jsql.Def{...})`** en lugar de SQL crudo; usa `ATTRIB` para campos flexibles que no requieren índices ni FKs.
3. **Centraliza mensajes de error** en un `msg/` o `msg.go` del paquete, como hace el resto de la librería — evita strings repetidos.
4. **Llama `Load()` una sola vez** al arrancar el servicio (`cache.Load()`, `event.Load()`, `jsql.Load()`); es idempotente, así que llamarlo de más no rompe nada, pero estructura el arranque para que sea explícito.
5. **Usa `response.ITEM/ITEMS/DATA/HTTPError`** para todas las respuestas HTTP — mantiene un contrato consistente (`{Ok, Result}` / errores) entre todos los servicios.
6. **Valida en el borde con `jval`** (`Require`/`Maybe`) antes de tocar la base de datos o lógica de negocio.
7. **Genera IDs con `reg`** (`ULID`, `UUID`, `GenULID(tag)`) — son ordenables temporalmente y consistentes con `_idx` de `jsql`.
8. **Inyecta `Store`/`Config` por interfaz** cuando uses `workflow`, `ia`, `jrex`, `config` — usa `stores/` como implementación por defecto sobre `jsql`.
9. **Propaga contexto de request** (`request.TenantId/UserId/Username`) en lugar de pasar estos valores manualmente por parámetros.
10. **Usa `.Debug()`/`.Test()` de `jsql`** durante desarrollo para inspeccionar el SQL generado sin tocar la base de datos.
11. **Sigue el estilo de comentarios `/** ... @param ... @return **/`** para nuevas funciones (consistencia con el resto del código).

---

## 8. Anti-patrones

| ❌ Anti-patrón | ✅ Alternativa en `et` |
|---|---|
| `m["email"].(string)` con verificación manual de `ok` | `data.Str("email")` o `data.ValStr("default", "email")` |
| `database/sql` con queries en strings concatenados | `jsql.Model` con `.Where`, `.Insert`, `.Update`, `.Upsert` |
| Validación manual con `if`/`else` anidados | `jval.Require/Maybe` con reglas declarativas |
| JWT hecho a mano con `golang-jwt` directo | `jwt.NewAuthentication/NewAuthorization` + `claim.Claim` |
| Cliente Redis (`go-redis`) directo en el código de negocio | `cache.Set/Get/SetWithDuration/Incr/...` |
| Cliente NATS directo | `event.Publish/Subscribe/Queue` |
| `uuid.New()` / `time.Now().UnixNano()` para IDs | `reg.ULID()`, `reg.GenULID(tag)`, `reg.UUID()` |
| Respuestas HTTP con `json.NewEncoder(w).Encode(...)` ad hoc | `response.ITEM/ITEMS/DATA/JSON/HTTPError` |
| `os.Getenv("X")` disperso por el código | `envar`/`config` (`config.GetStr/GetInt/...`, `envar.Validate`) |
| `cron.New()` (robfig) directo + goroutines propias para pub/sub | `crontab.New(tag)` + `AddJob/AddEventJob` (ya integra NATS) |
| Reimplementar paginación (`limit`, `offset`, `total`) por servicio | `et.List` / `model.Limit(n).Page(p).All()` |
| Middlewares de logging/CORS/recovery propios | `middleware.Logger/AllowAll/Recoverer/RequestID/Authenticate` |
| Cliente HTTP propio con `net/http` + manejo de errores ad hoc | `request.Get/Post/Put/Delete/Patch(...)` → `(*Body, Status)` |
| Definir `_idx`/`createdAt`/`updatedAt` a mano en cada tabla | `db.DefineModel(...)` (full) que ya los agrega |

---

## 9. Guía de migración desde alternativas

| Desde | Hacia (`et`) | Notas de migración |
|---|---|---|
| `gin`, `echo`, `fiber` | `server/` (ligero) o `ettp/v2` (completo, con sync NATS) | `ettp/v2` requiere Redis + NATS; `server/` no. Las rutas usan `chi` por debajo, sintaxis de patrones compatible (`/users/{id}`). |
| `gorm`, `sqlx`, `ent` | `jsql/` | Define modelos con `jsql.Def{}`; usa `ATTRIB` para columnas JSONB en vez de structs con `gorm.Model`. Drivers: `postgres`, `sqlite` (vía `init()`). |
| `go-playground/validator`, `ozzo-validation` | `jval/` | Reemplaza tags de struct por reglas fluidas sobre `et.Json`: `jval.Require(body, jval.Str("name").NotEmpty(), jval.Email("email"))`. |
| `golang-jwt/jwt` crudo | `jwt/` + `claim/` | `jwt.NewAuthentication/NewAuthorization/NewAppToken` ya gestionan firma (HS256, env `SECRET`), expiración y almacenamiento en `cache`. |
| `go-redis/redis` crudo | `cache/` | `cache.Load()` lee `REDIS_HOST` (+ `REDIS_PASSWORD`, `REDIS_DB`); usa `SetObject/GetObject` para serialización automática, `SetH/D/W/M/Y` para TTLs comunes. |
| `nats.go` crudo | `event/` | `event.Load()` lee `NATS_HOST`; usa `Queue` para balanceo de carga (cola), `Subscribe` para fan-out, `Stack`/`Source` para flujos especializados. |
| `robfig/cron` crudo | `crontab/` | `crontab.New(tag)` ya integra `event`; usa `AddJob`/`AddEventJob` con specs de 6 campos (con segundos). |
| `google/uuid`, `oklog/ulid`, `rs/xid` directos | `reg/` | `reg.UUID()/ULID()/XID()`, con variantes con prefijo `GenULID(tag)`, `TagULID(tag, id)`. |
| `logrus`, `zap`, `zerolog` | `logs/` | API más simple (`Info/Error/Alert/Fatal/Debug`), salida colorizada vía `stdrout`. Si necesitas structured logging avanzado (JSON, sinks múltiples), evalúa si `logs/` cubre el caso o documenta la excepción (ver §10, regla de justificación). |
| Validación de email/teléfono/UUID con regex propias | `utility/` (`ValidEmail`, `ValidPhone`, `ValidUUID`, `ValidName`, ...) | |
| Cliente HTTP propio (`resty`, `net/http` envuelto) | `request/` | `request.Get/Post/Put/Delete/Patch/Options` (+ `WithTls`) devuelven `(*request.Body, request.Status)`; `Body` tiene `ToJson/ToItem/ToItems/ToArrayJson/ToInt/...`. |
| Orquestadores de workflow propios (state machines ad hoc) | `workflow/` | Modela como `Flow → Steper → Step`; usa `RunInstance/ResetInstance/RollbackInstance/StopInstance`. |
| Integración directa con OpenAI SDK | `ia/` | `ia.New(tenantId, tag, store, config)` gestiona `Agent`/`Conversation`/`Message` y persistencia vía `instances.Store`. |

---

## 10. AI Development Rules

> **Estas reglas son vinculantes para cualquier asistente de IA que genere o modifique código en un proyecto que dependa de `github.com/cgalvisleon/et`.**

1. **Prioriza `et` antes que cualquier dependencia externa.** Antes de proponer `go get` para una nueva librería (validación, HTTP, JWT, cache, pub/sub, cron, IDs, logging, llamadas HTTP, workflows), **verifica primero si `et` ya resuelve el problema** (ver §4 y §5 de este documento).
2. **Antes de crear un nuevo componente, busca un equivalente en `et`.** Ejemplos: ¿necesitas paginación? → `et.List` / `model.Limit().Page()`. ¿Necesitas un wrapper de respuesta HTTP? → `et.Item`/`et.Items` + `response.*`. ¿Necesitas un generador de IDs? → `reg.*`.
3. **Antes de escribir lógica personalizada, revisa si `et` ya la ofrece.** Ejemplos: parsing de body JSON (`request.GetBody`), extracción de claims (`request.TenantId/UserId/...`), validación de email/teléfono (`utility.ValidEmail/ValidPhone`), cifrado (`utility.Encrypt`).
4. **No dupliques capacidades existentes.** Si el proyecto ya usa `jsql` para persistencia, no introduzcas `database/sql` crudo "solo para esta query"; extiende el modelo existente o usa `.Debug()`/`.Test()` para depurar.
5. **Mantén consistencia con los patrones de la librería**:
   - `et.Json` como tipo de datos dinámico, con accesores tipados.
   - `Load()` idempotente para inicialización de infraestructura.
   - Interfaces `Store`/`Config`/`Driver` para inyección de dependencias.
   - Constantes de error en `msg/` o `msg.go`.
   - Comentarios de función en estilo `/** ... @param ... @return **/`.
   - Handlers HTTP con el patrón de §4.3 (`request.GetBody` → `jval.Require` → lógica → `response.ITEM/HTTPError`).
6. **Favorece la composición sobre la implementación ad hoc.** Combina `jsql` + `jval` + `response` + `middleware` antes de escribir un endpoint desde cero con `net/http` puro.
7. **Si decides NO usar `et` para algo que tiene equivalente interno, justifícalo explícitamente** en el código (comentario breve) y en tu respuesta al usuario: por qué la alternativa interna no aplica (ej. requisito de protocolo específico, limitación de rendimiento documentada, incompatibilidad de tipos). No reemplaces silenciosamente un componente de `et` por otro.
8. **No reescribas paquetes de `et`** salvo que el usuario lo solicite explícitamente. Si encuentras una limitación, prefiere extender (wrapper/adaptador) antes que forkear o copiar código del paquete.
9. **Respeta las variables de entorno y convenciones de configuración** documentadas en `CLAUDE.md` y este archivo (`DB_*`, `REDIS_*`, `NATS_*`, `OPENAI_API_KEY`, `SECRET`, etc.) — no introduzcas un sistema de configuración paralelo.
10. **Cuando el usuario pida "agregar X"**, responde primero identificando si `et` ya tiene un componente para `X` (cítalo con su paquete y función), y solo después propone una implementación nueva si no existe equivalente.

---

## 11. Code Generation Guidelines

### 11.1 Priority Matrix (orden de decisión obligatorio)

Al generar código para resolver una necesidad, evalúa **en este orden** y detente en la primera opción viable:

| Prioridad | Acción | Ejemplo |
|---|---|---|
| **1** | **Usar un componente nativo de `et` tal cual** | Validar email → `jval.Email("email")`; generar ID → `reg.ULID()`; responder JSON → `response.ITEM(...)` |
| **2** | **Extender un componente de `et`** (nuevas reglas, modelos, middlewares que se integran con su API) | Crear una nueva `jval.Rule` personalizada implementando la interfaz `Rule`; agregar un `TriggerFunction` a un modelo `jsql` |
| **3** | **Crear un adaptador alrededor de `et`** (cuando el contrato externo no coincide pero la lógica interna sí debe usar `et`) | Adaptador que traduce un webhook externo a `et.Json` y lo persiste vía `jsql.Model` |
| **4** | **Implementar una solución personalizada** (sin tocar dependencias externas) | Lógica de negocio específica del dominio que no tiene equivalente genérico |
| **5** | **Incorporar una dependencia externa** (última opción, requiere justificación explícita) | Un protocolo o SDK de terceros que `et` no cubre (ej. un proveedor de pagos específico) |

### 11.2 Componentes y APIs preferidas (resumen rápido)

| Necesidad | Preferido |
|---|---|
| Tipo de dato dinámico | `et.Json` |
| Resultado paginado / item / lista | `et.List`, `et.Item`, `et.Items` |
| Acceso a DB | `jsql.Model` (`Define`, `Where`, `Insert`, `Update`, `Upsert`) |
| Validación de entrada | `jval.Require` / `jval.Maybe` |
| Respuesta HTTP | `response.ITEM/ITEMS/DATA/HTTPError` |
| Lectura de request | `request.GetBody`, `request.URLParam`, `request.Query`, `request.TenantId/UserId/...` |
| Llamadas salientes | `request.Get/Post/Put/Delete/Patch` |
| Auth | `jwt.NewAuthentication/NewAuthorization` + `middleware.Authenticate` |
| Cache | `cache.Set/Get/SetObject/GetObject/SetWithDuration` |
| Pub/Sub | `event.Publish/Subscribe/Queue` |
| Cron | `crontab.New(tag).AddJob/AddEventJob` |
| IDs | `reg.ULID/UUID/XID`, `reg.GenULID(tag)` |
| Logging | `logs.Info/Error/Alert/Fatal/Debug` |
| Config/env | `config.GetStr/GetInt/...`, `envar.Validate` |
| Workflows | `workflow.NewFlow/RunInstance/RollbackInstance` |

### 11.3 Convenciones de nombres

- Paquetes en minúsculas, sin guiones (`jsql`, `jval`, `ettp`).
- Funciones exportadas en `PascalCase`, comenzando con verbo cuando aplica (`NewToken`, `GetBody`, `ValidEmail`).
- Constantes de mensajes de error: `MSG_<DESCRIPCION>` en `msg.go` (ver `msg/msg.go`).
- Handlers HTTP: `Http<Acción><Recurso>` (`HttpGetUser`, `HttpNewFlow`, `HttpRunInstance`).
- Columnas/atributos de `jsql`: `snake_case` para nombres de columna SQL; constantes exportadas en `PascalCase` desde `jsql/column.go` (`jsql.ID`, `jsql.IDX`, `jsql.TENANT_ID`).
- Comentarios de función: bloque `/** ... **/` con `FunctionName: descripción`, `@param`/`@return` una línea cada uno (ver `CLAUDE.md`).

### 11.4 Estructura de carpetas recomendada para un servicio consumidor

```
my-service/
├── cmd/
│   └── my-service/main.go        // arranque: Load() de cache/event/jsql, server/ettp.New, registro de rutas
├── internal/
│   ├── models/                   // definiciones jsql.Def por entidad
│   ├── handlers/                 // Http<Accion><Recurso>, usa request/response/jval
│   ├── services/                 // lógica de negocio (usa jsql.Model, cache, event)
│   ├── msg/                      // constantes MSG_* del servicio
│   └── workflows/                // flows/steps si aplica
├── go.mod
└── CLAUDE.md / LIBRARY_CONTEXT.md
```

### 11.5 Patrón de inyección de dependencias

- Las dependencias de infraestructura (`*jsql.DB`, implementaciones de `instances.Store`, `*config.Config`) se construyen **una vez en `main`** y se inyectan a structs de servicio/handler como campos.
- Para `workflow`/`ia`/`jrex`, implementa la interfaz `Store` correspondiente (o usa `stores.NewInstance(db, schema, name, kind)`) y pásala a `Load`/`New`.
- No uses variables globales mutables salvo las que la propia librería expone como singletons (`cache`, `event`, `config` tras `Load()`).

```go
type UserService struct {
    db    *jsql.Model
    cache bool // o referencia explícita si se requiere
}

func NewUserService(db *jsql.Model) *UserService {
    return &UserService{db: db}
}
```

### 11.6 Manejo de errores

- Funciones de negocio devuelven `error` estándar de Go.
- Mensajes de error como constantes `msg.MSG_*` (con `fmt.Errorf(msg.MSG_X, args...)` cuando llevan formato).
- En el borde HTTP: `response.HTTPError(w, r, statusCode, err.Error())`.
- Errores fatales de arranque: `logs.Fatal(err)`.
- Errores no fatales pero relevantes: `logs.Error(err)` / `logs.Alert(err)`.

### 11.7 Testing

- El repo de `et` no tiene `*_test.go` aún; `go test ./...` compila pero no ejecuta nada.
- Para servicios consumidores: usa `.Test()`/`.Debug()` de `jsql.Model`/`Query`/`Command` para verificar el SQL generado sin DB real.
- Para handlers HTTP, usa `httptest` estándar de Go + `request`/`response` helpers (son funciones puras sobre `http.ResponseWriter`/`*http.Request`, fácilmente testeables).
- Para `jval`, las reglas son testeables unitariamente: `rule.Validate(et.Json{...})`.

### 11.8 Observabilidad

- Logging: `logs.*` para todo logging de aplicación (no introducir otro logger salvo justificación, ver Regla 7 de §10).
- Métricas/telemetría HTTP: `middleware.Metrics` (`NewMetric`, `PushTelemetry`, `PushTelemetryLog`, `PushTelemetryOverflow`, `DoneHTTP`).
- Request ID: `middleware.RequestID` + `middleware.GetReqID(ctx)`.
- Recuperación de pánicos con stack legible: `middleware.Recoverer`.

### 11.9 Ejemplos correctos vs incorrectos

**❌ Incorrecto** — type assertions manuales, SQL crudo, respuesta ad hoc:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    var body map[string]interface{}
    json.NewDecoder(r.Body).Decode(&body)
    email, _ := body["email"].(string)

    rows, _ := db.Query("SELECT id, name FROM users WHERE email = $1", email)
    // ...
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "data": rows})
}
```

**✅ Correcto** — `et.Json`, `jval`, `jsql`, `response`:

```go
func httpGetUserByEmail(w http.ResponseWriter, r *http.Request) {
    body, err := request.GetBody(r)
    if err != nil {
        response.HTTPError(w, r, http.StatusBadRequest, err.Error())
        return
    }

    if err := jval.Require(body, jval.Email("email")); err != nil {
        response.HTTPError(w, r, http.StatusBadRequest, err.Error())
        return
    }

    item, err := users.Where(jsql.Eq("email", body.Str("email"))).One()
    if err != nil {
        response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
        return
    }

    response.ITEM(w, r, http.StatusOK, item)
}
```

---

## 12. Qué reemplaza `et` (y qué dependencias evita)

### 12.1 Frameworks/librerías que `et` reemplaza total o parcialmente

| Categoría | Alternativas externas comunes | Reemplazo en `et` |
|---|---|---|
| Framework web | `gin`, `echo`, `fiber` | `server/`, `ettp/v2` (sobre `chi`) |
| ORM / query builder | `gorm`, `ent`, `sqlx`, `squirrel` | `jsql/` |
| Validación de structs/JSON | `go-playground/validator`, `ozzo-validation` | `jval/` |
| JWT | `golang-jwt/jwt` (uso directo) | `jwt/` + `claim/` |
| Cliente Redis | `go-redis/redis` (uso directo) | `cache/` |
| Cliente NATS / mensajería | `nats.go` (uso directo), `rabbitmq` para pub/sub simple | `event/` |
| Cron | `robfig/cron` (uso directo) | `crontab/` |
| Generación de IDs | `google/uuid`, `oklog/ulid`, `rs/xid`, `bwmarrin/snowflake` (uso directo) | `reg/` |
| Cliente HTTP | `net/http` envuelto a mano, `resty`, `go-resty` | `request/` |
| Logging | `logrus`, `zap`, `zerolog` (para casos simples) | `logs/` |
| Middlewares chi comunes | `go-chi/cors`, `go-chi/middleware` reimplementado | `middleware/` (incluye versiones propias de logger, recoverer, request ID, telemetry) |
| Orquestación de procesos / sagas | Implementaciones propias de state machines, `temporal` (para casos simples) | `workflow/` |
| Integración OpenAI con tracking | `openai-go` SDK directo + tracking propio | `ia/` |
| Validación de formatos (email, teléfono, UUID) | `regexp` propias, `asaskevich/govalidator` | `utility/` |

### 12.2 Dependencias futuras que `et` puede evitar

Al planear nuevas features, considera que `et` ya cubre lo siguiente — **no agregues estas dependencias salvo justificación**:

- Librerías de paginación/serialización de resultados → `et.List`/`et.Item`/`et.Items`.
- SDKs de Redis/NATS adicionales → `cache/`, `event/`.
- Librerías de scheduling (`gocron`, etc.) → `crontab/`.
- Librerías de circuit breaker / retry (`sony/gobreaker`, `hashicorp/go-resiliency`) → `resilience/` (usado por `workflow`).
- Librerías de generación de tokens OTP → `utility.GetOTP` + `service` (`SendOTPEmail/SendOTPSms/VerifyOTP`).
- Clientes para AWS S3/SES/SNS, Brevo, WhatsApp Business → `aws/`, `brevo/`, `wsp/`.
- Librerías de manejo de zonas horarias / unidades → `timezone/`, `units/`.
- Librerías de colores en terminal (`fatih/color`) → `color/`, `stdrout/`.
- Watchers de filesystem (`fsnotify` envuelto) → `file/` (usado por `jrex` para hot-reload).
- Drivers Neo4j envueltos → `graph/`.

### 12.3 Límites conocidos (no inventar capacidades)

- **No hay `*_test.go` en `et`** — no asumas helpers de testing propios de la librería más allá de `.Debug()`/`.Test()`.
- **Drivers `jsql`**: solo `postgres` y `sqlite` están implementados; `josefina` y `mysql` existen como directorios vacíos — no asumas soporte MySQL/josefina sin verificarlo en el código.
- **`wf/`** es una reescritura en progreso de `workflow/`, no usada por nada más — no la uses como referencia de API estable.
- **`ettp/v1`** es la versión anterior de `ettp/v2` — prefiere `v2` salvo que el proyecto ya dependa de `v1`.
