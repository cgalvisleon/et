# LIBRARY_CONTEXT.md

> Documento de contexto persistente para asistentes de IA (Claude, ChatGPT, Cursor, Windsurf, Cline, etc.)
> Librería: **`github.com/cgalvisleon/et`** — Go 1.25 — MIT
> Copia este archivo en la raíz de cualquier proyecto que dependa de `et` para que el asistente diseñe soluciones coherentes con la librería.
>
> **Advertencia de vigencia:** este repositorio cambia muy rápido y de forma poco documentada (el historial de commits son casi todos "Backup:" sin mensaje descriptivo). Este documento fue generado leyendo el código fuente real en la fecha de generación. Antes de confiar en una firma o ruta de archivo aquí citada para una decisión importante, verifícala contra el código — ver `CLAUDE.md` en la raíz del repo para más contexto operativo (comandos, convenciones).

---

# Executive Summary

`et` es una **librería modular de utilidades para construir microservicios, CLIs y aplicaciones web en Go**. No es un framework monolítico: es un conjunto de ~50 paquetes independientes, cada uno importable por separado, que cubren de punta a punta las necesidades habituales de un backend:

- **Modelo de datos universal**: `et.Json`, `et.List`, `et.Item`, `et.Items`.
- **Persistencia** agnóstica de motor con ORM ligero: `jsql/` (hoy, en la práctica, solo PostgreSQL funciona — ver Anti-Patrones).
- **Servidores HTTP** en varios niveles de abstracción: `server/` (ligero), `ettp/v2` (completo, con cache/eventos), `ettp/v1` (legado), más `router/` y `middleware/`.
- **Validación declarativa** de payloads: `jval/`.
- **Autenticación/JWT**: `jwt/` + `claim/`.
- **Infraestructura**: Redis (`cache/`), NATS (`event/`), Neo4j (`graph/`, muy incompleto).
- **Configuración y entorno**: `config/`, `envar/`.
- **Logging estructurado**: `logs/` (+ `stdrout/`, `color/`).
- **Orquestación**: cron (`crontab/`), workflows multi-paso basados en grafo (`jwf/`), agentes de IA sobre OpenAI (`jia/`), runtime JS embebido (`jrex/`), resiliencia/reintentos (`resilience/`).
- **Integraciones externas**: AWS (S3/SNS), Brevo (email/SMS/WhatsApp templado), WhatsApp Business Graph API (`jwsp/`).
- **Comunicación**: WebSocket (`jws/`), RPC sobre TCP (`jrpc/`), nodo TCP con consenso tipo Raft propio (`jtcp/`).
- **Utilidades transversales**: IDs (ULID/UUID/XID/Snowflake en `reg/`), criptografía y validación de formato (`utility/`), strings (`strs/`), memoria/concurrencia (`mem/`, `ephemeral/`, `race/`, `iterate/`), tiempo/unidades (`timezone/`, `units/`), generación de proyectos (`create/`).

**Idea central**: cualquier dato dinámico que entra o sale de un servicio (body HTTP, fila de base de datos, mensaje de evento, configuración, claim JWT) se representa como `et.Json` (`map[string]interface{}`) con accesores tipados y valor por defecto. Esto evita el patrón `val, ok := m["x"].(string)` repetido en toda la base de código.

**Estado real del proyecto** (importante para decisiones de adopción): es una librería viva, en evolución activa y a veces inconsistente entre paquetes hermanos. Varias piezas documentadas en versiones anteriores de este archivo (paquetes `workflow/` e `instances/`) ya no existen — fueron remplazadas por `jwf/`. Algunos paquetes están notablemente incompletos (`graph/`, partes de `jrpc/`, `cmds.RunSSH`). Trátalo como una librería de utilidades sólida en su núcleo (`et`, `jsql` sobre Postgres, `cache`, `event`, HTTP) y experimental en sus bordes (`jwf`, `jtcp`, `graph`).

---

# Design Philosophy

1. **"JSON como lingua franca"**: `et.Json` cruza todas las capas — body HTTP, filas de base de datos (`_source` JSONB), mensajes NATS, configuración, claims JWT, resultados de workflows. Un solo tipo, un solo conjunto de accesores (`Str`, `Int`, `Int64`, `Num`, `Bool`, `Time`, `Json(attr)`, `Array...`, `ValStr(def, ...atribs)`, etc. — ver `et/json.go`).
2. **Modularidad por import, no por imposición**: no hay un "framework" que envuelva la app. Cada paquete se importa solo si se necesita. `jsql` no requiere `cache`; `cache` no requiere `event`; `ettp/v2` sí puede necesitar ambos, pero solo si se activan los flags `Config.UseCache`/`Config.UseEvent`.
3. **`Load()`/`New()` mayormente idempotente**: los paquetes de infraestructura (`cache`, `event`) exponen `Load()` seguro de llamar varias veces. Los paquetes de orquestación (`jia`, `jwf`, `crontab`, `config`) en cambio usan `New(...)`/`Load(...)` que crean y devuelven una instancia explícita — no hay singletons globales en ninguno de ellos hoy (`jia.Load(id, store)` y `jwf.Load(id, store)` cargan una instancia existente por su propio ID, no son setters de un global de paquete).
4. **Inversión de dependencias vía interfaces pequeñas y locales**: la librería define interfaces de persistencia (`jsql.Driver`, `jia.Store`, `jwf.Store`, `resilience.Store`, `config.Store`, `crontab.Store`, `jrex.Store`) y el consumidor las implementa. **Importante**: ya no existe una interfaz `Store` compartida (el antiguo paquete `instances/` fue eliminado) — cada paquete define la suya, con firmas parecidas pero no intercambiables sin verificar (`jwf.Store` exige además un método `GenSerie(tag) (string, error)` que `jia.Store` no tiene).
5. **APIs fluidas / encadenables**: `model.Where(...).And(...).Limit(20).Page(1).All()`, `flow.Step(tag, title, fn).Step(...)`, `jval.Require(body, jval.Str("email").NotEmpty())`.
6. **Agnosticismo de driver (en teoría)**: `jsql` define el contrato (`Driver` interface en `jsql/driver.go`) y los drivers se auto-registran con `init()` al importarse como side-effect. En la práctica solo `postgres` tiene implementación real hoy.
7. **Esquema híbrido relacional/documental**: las tablas de `jsql` tienen columnas reales (`COLUMN`) y atributos dentro de una columna `_source JSONB` (`ATTRIB`), permitiendo evolución de esquema sin migraciones constantes, sin perder capacidad de consulta SQL (`_source->>'campo'`).
8. **Mensajes de error centralizados**: casi todos los paquetes tienen un `msg.go` (o paquete `msg/`) con constantes de error — usarlas en vez de strings literales repetidos.
9. **Respuestas HTTP unificadas**: la salida de una API pasa por `response.ITEM` / `response.ITEMS` / `response.DATA` / `response.JSON` / `response.HTTPError`, envolviendo `et.Item` / `et.Items` / `et.Json`. (`jwf/` es la excepción que usa `response.JSON` directo — ver Anti-Patrones.)
10. **Estilo de comentarios no-GoDoc real**: la mayoría del código usa bloques `/** ... @param ... @return ... **/` en vez de comentarios GoDoc estándar (`// Func: ...`). Si generas código para este repo, sigue la convención del archivo que estés editando.

---

# Architecture Overview

No existe un punto de entrada central ni un `et.App` que arranque "todo". Cada servicio compone los paquetes que necesita. La arquitectura se entiende mejor en capas de dependencia:

```
Capa 1 — Utilidades autosuficientes (sin servicios externos)
  et, utility, strs, reg, jval, logs, stdrout, color, config, envar,
  mem, ephemeral, iterate, race, timezone, units, file

Capa 2 — Infraestructura (requieren servicios externos vía env vars)
  cache    -> Redis
  event    -> NATS
  graph    -> Neo4j (muy incompleto)
  jsql     -> PostgreSQL (único driver funcional)

Capa 3 — HTTP / routing (construidos sobre go-chi)
  server, ettp/v1, ettp/v2, router, middleware, response, request, jws

Capa 4 — Aplicación / orquestación (compone capas 1-3)
  jwf (workflows), jia (agentes IA), jrex (runtime JS), crontab,
  resilience, service, jrpc, jtcp

Capa 5 — Integraciones externas
  aws (S3/SNS), brevo (email/SMS/WhatsApp templado), jwsp (WhatsApp Graph API)

Capa 6 — Persistencia de aplicación
  stores (helpers jsql-backed: instancias, autorización)

Herramientas de desarrollo
  cmd/* (binarios), create (generador de proyectos), cmds (ejecución de pipelines), jcli (en progreso/huérfano)
```

**Patrones estructurales clave:**

- **Sincronización entre réplicas vía eventos NATS**: `ettp/v2` sincroniza el estado del router entre instancias usando eventos (`EVENT_SET_ROUTER`, `EVENT_REMOVE_ROUTER`, `EVENT_RESET_ROUTER`), con bandera `m.Myself` para evitar bucles de auto-procesamiento. `router/` usa un patrón similar de forma standalone (`PushApiGateway`, `RemoveApiGateway`).
- **Patrón Store inyectado, pero no unificado**: `jia.New(tag, store, userId)`, `jwf.New(store)`, `crontab.New(tag, store)`, `resilience.New(store)`, `jrex.Load(tag, store)` — cada uno define su propio `Store` local. Ni `jia.New` ni `jwf.New` reciben ya un `tenantId` (ambos generan su propio `ID` internamente con `reg.UUID()`; cargar una instancia existente es `jia.Load(id, store)` / `jwf.Load(id, store)`). Antes de reutilizar una implementación entre paquetes, compara las firmas exactas (ver Anti-Patrones — el caso `stores.Instance` es el ejemplo real de una incompatibilidad).
- **Patrón Driver auto-registrado**: `import _ "github.com/cgalvisleon/et/jsql/drivers/postgres"` registra el driver vía `init()`; `jsql.Load(tenantId)` lo resuelve internamente leyendo `DB_DRIVER` con `config.GetStr`.
- **Debug/Test transversal en `jsql`**: `Model`, `Query` y `Command` soportan `.Debug()` (loguea SQL sin ejecutar) y `.Test()` (devuelve SQL sin ejecutar) — útil para pruebas sin DB real.
- **Contexto de request enriquecido**: `request/ctx.go` propaga `tenantId`, `userId`, `username`, `profileId`, `app`, `device`, `payload` a través de `context.Context`, poblado por `middleware.Authenticate` y leído por handlers (`request.TenantId(r)`, `request.UserId(r)`, etc.).
- **Motor de workflows basado en grafo, no en lista lineal**: `jwf/` modela un `Flow` como un conjunto de `Step`s conectados por `Connection`s con puertos (`input`/`output`/`error`), no como una secuencia fija — más cercano a un diagrama de flujo que a un pipeline.

---

# Core Components

### Núcleo de datos — `et/`

| Tipo                                 | Propósito                | API clave                                                                                                                                                                                                                                                                                                                                                                                              |
| ------------------------------------ | ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `et.Json` (`map[string]interface{}`) | Tipo universal de datos  | `Str`, `Int`, `Int64`, `Num`, `Bool`, `Time`, `Json(attr)`, `Array`, `ArrayStr/Int/Int64/Number/Bytes/Json`, `MapStr/Int/Float`, `ValStr/ValInt/ValInt64/ValNum/ValBool/ValTime/ValJson/ValArray(def, ...atribs)`, `Get`, `Set`, `SetNested`, `Delete`, `Exist`, `Remove`, `Select`, `Hidden`, `Clone`, `Update`, `Compare`, `Append`, `IsChanged`, `IsDeferent`, `ToByte/ToString/ToMap/ToEscapeHTML` |
| `et.List`                            | Resultado paginado       | `Rows`, `All`, `Count`, `Page`, `Start`, `End`, `Result []Json`                                                                                                                                                                                                                                                                                                                                        |
| `et.Item`                            | Resultado de un registro | `Ok bool`, `Result Json` + mismos accesores tipados que `Json`                                                                                                                                                                                                                                                                                                                                         |
| `et.Items`                           | Resultado multi-registro | `Add`, `AddMany`, `One(idx)`, `First`, `Last`, `ToList(all, page, rows)`, accesores indexados                                                                                                                                                                                                                                                                                                          |

> **Regla de oro**: para leer/escribir datos dinámicos (JSON, filas de DB, payloads), usa `et.Json` y sus accesores — nunca `map[string]interface{}` a mano ni _type assertions_ manuales.

### Persistencia — `jsql/`, `stores/`

- `jsql.Load(tenantId) (*DB, error)` / `jsql.LoadTo(tenantId, name) (*DB, error)` — sin objeto de configuración, lee `DB_*` vía `config`/`envar`. **Solo `postgres` tiene driver real** (ver Anti-Patrones).
- Modelos: `db.DefineModel(schema, name, version)` (agrega `id`, `created_at`, `updated_at`, `_source`, `_idx`), `db.NewModel(...)` (manual), `db.Define(jsql.Def{...})` (declarativo, preferido).
- Tipos de columna: `COLUMN`, `ATTRIB` (dentro de `_source` JSONB), `DETAIL`/`ROLLUP` (relaciones virtuales), `CALCFUNC`/`CALC` (computadas), `AGG` (agregaciones).
- Consultas/comandos fluidos: `.Where(jsql.Eq(...)).And(...).Limit().Page().All()/.One()`, `.Insert(...)`, `.Update(...)`, `.Upsert(...)`, `.Delete()`, todos con `.ExecTx(tx)`/`.Exec()`.
- Triggers: `beforeInserts/Updates/Deletes`, `afterInserts/Updates/Deletes` (`TriggerFunction`); columnas calculadas vía `CalcFunction`.
- Paths anidados JSONB: `"ventas->detalle->precio"` se traduce automáticamente a `->`/`->>` con casts.
- `stores/` — helpers jsql-backed: `DefineInstance`/`LoadInstance`/`DefineInstanceBite`/`LoadInstanceBite` (registro genérico tipo "instancia"), `DefineAuthorization` (registro de permisos), `DefineCatalog` (tabla genérica `kind`+`id`, sin caché — el resto sí cachea vía `dt`). Ver Anti-Patrones sobre su (in)compatibilidad con `jia.Store`/`jwf.Store`.

### HTTP y routing

| Paquete       | Nivel                                | Cuándo usarlo                                                                                                                                                                        |
| ------------- | ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `server/`     | Ligero (`Ettp` sobre `chi.Mux`)      | Servicios sin Redis/NATS. `server.New(name, port) *Ettp`, `.Use(...)`, `.HandleFunc`, `.Mount`, `.Start()`, `.OnStart/.OnClose`                                                      |
| `ettp/v2`     | Completo                             | `ettp.New(name string, cnf *Config) (*Server, error)` — `Config.UseCache`/`UseEvent` activan `cache.Load()`/`event.Load()` internamente; router sincronizado entre réplicas vía NATS |
| `ettp/v1`     | Anterior, último cambio 2026-06-21 (sweep mecánico `config`→`envar`, no es una feature nueva) | Prefiere `v2`                                                                                                                                                                        |
| `router/`     | Standalone                           | Routing con sincronización NATS sin el resto de `ettp`: `Public/Private/Protect/With`, `PushApiGateway`, `SetChannels`                                                               |
| `middleware/` | Transversal                          | `Authenticate`, `Logger`, `Recoverer`, `RequestID`, `AllowAll` (CORS), `Metrics`/`PushTelemetry*`                                                                                    |
| `response/`   | Salida                               | `ITEM`, `ITEMS`, `DATA`, `JSON`, `RESULT`, `ANY`, `Stream`, `HTTPError`, `HTTPAlert`, `Unauthorized`, `Forbidden`, `InternalServerError`                                             |
| `request/`    | Entrada                              | `GetBody(r)`, `URLParam(r, key).Str()/.Int()/...`, `Query(r, key)`, contexto: `TenantId`, `UserId`, `Username`, `ProfileId`, `App`, `Device`, `Payload`                              |
| `jws/`        | WebSocket (antes `ws/`)              | `jws.New() *Hub`, `hub.Connect/SendTo/Topic/Queue/Stack/Publish`                                                                                                                     |

### Validación — `jval/`

Validadores tipados y encadenables sobre `et.Json`: `Str`, `Int`, `Float`, `Bool`, `Array`, `Email`, `Date`, `Enum`, `Phone`, `Between`, `Object` (anidado), agrupados con `jval.Require(data, rules...)` (todas obligatorias) o `jval.Maybe(data, rules...)` (opcionales).

### Infraestructura — `cache/`, `event/`, `graph/`

- `cache/` — Redis. `cache.Load()`, `Set/Get/Delete/Exists/Expire/Incr/Decr`, listas (`LPush/LRange/...`), colecciones hash, objetos JSON (`SetObject/GetObject`, `ObjetSet/ObjetGet`), verificación/OTP (`SetVerify/GetVerify`), `cache.Metrics`.
- `event/` — NATS. `event.Load()`, `Publish(channel, data)`, `Subscribe(channel, fn)` (broadcast), `Queue(channel, queue, fn)` (reparto entre workers), `Stack(channel, fn)`, `event.Message{Data et.Json, Myself bool, ...}`.
- `graph/` — Neo4j. **Solo tiene `graph.Load() (*Conn, error)`**, con URL y credenciales _hardcodeadas_ (`neo4j://localhost:7687`, `neo4j`/`password`) — no hay métodos de consulta/sesión expuestos. No usable en producción tal cual.

### Configuración, entorno y logging

- `config/` — getters de paquete (`GetStr/GetInt/GetInt64/GetFloat/GetBool/Get/Set/Validate/IsLoad`), respaldados por `envar/`. `config.Config` es un registro de configuración por tenant respaldado por un `Store` (no es un descriptor de app global).
- `envar/` — acceso a variables de entorno (`GetStr/GetInt/...`) y argumentos CLI (`ArgStr/ArgInt/...`), `Validate(keys)`.
- `logs/` — `Log`, `Info(f)`, `Alert(f)`, `Error(f)`, `Debug(f)`, `Fatal`, `Panic`, `Tracer`, todo vía `stdrout/` (color ANSI, `color/`).

### Identidad, autenticación y seguridad

- `reg/` — generadores de ID: `UUID()`, `ULID()`, `XID()`, `GenSnowflake()`, más variantes "tag" y "get-or-generate" (`GetULID(id)` retorna `id` si ya es válido, o genera uno nuevo si está vacío/`"*"`/`"new"`).
- `claim/` — claims JWT (`tenantId`, `profileId`, `payload`), `NewClaim`, `NewToken` (HS256 vía `golang-jwt/jwt/v4`), `ParceToken`.
- `jwt/` — capa sobre `claim/` + `cache/`: `NewAuthentication`, `NewAuthorization`, `NewAppToken`, `NewEphemeralToken`, `Validate`, `RenewToken`, `DeleteToken` (logout).
- `utility/` — criptografía (`Encrypt` con MD5/SHA1/SHA256/SHA512/AES, `DecryptoAES`), generación de IDs simples (`UUID`, `GetOTP`, `GenId`), validadores de formato (`ValidEmail`, `ValidPhone`, `ValidUUID`, etc.).

### Orquestación

- `crontab/` — `crontab.New(tag, store)` (requiere `Store`, ya no es opcional), `AddJob/AddOneShotJob/AddEventJob`, soporta segundos vía `robfig/cron` (`cron.WithSeconds()`).
- `jwf/` — workflows basados en grafo (ver detalle completo en `COMPONENT_CATALOG.md` y `CLAUDE.md`). Sustituye a los antiguos `workflow/` e `instances/` (eliminados). `jwf.New(store) (*WorkFlow, error)` ya no recibe `tenantId` — genera su propio `ID` (`reg.UUID()`); `jwf.Load(id, store)` carga una instancia existente por ese `ID`.
- `resilience/` — reintentos con intervalo: `resilience.New(store)`, `LoadInstance(Params{...})`, `instance.Run(userId)`. `Params` ya no tiene campos `TenantId`/`OwnerId`/`UserId`.
- `jia/` (antes `ia/`) — agentes sobre OpenAI (`openai-go/v3`): `jia.New(tag, store, userId) (*Ia, error)` ya no recibe `tenantId` (`Ia` no tiene campo `TenantId`); `jia.Load(id, store)` carga una instancia existente por su propio `ID`. Gestiona `Agent`/`Participant`/`Conversation`.
- `jrex/` — runtime JS embebido (`dop251/goja`) con hot-reload, usado tanto standalone como motor de ejecución de pasos de `jwf/` cuando `Step.Definition` es un string JS.
- `service/` — OTP y mensajería multicanal (`SendOTPEmail/SendOTPSms/VerifyOTP`, `SendSms/SendWhatsapp/SendEmail`), delega en `aws/`/`brevo/`.

### Comunicación de bajo nivel

- `jrpc/` — RPC sobre TCP con `net/rpc` estándar. `jrpc.Mount(host, port, services, packageName) (*Package, error)` registra un servicio. **No tiene balanceador de carga ni consenso Raft** — es un registro simple de `Solver{Host, Port, Inputs, Output}` por método (`jrpc/package.go`). `balancer.go` y `raft.go` (que sí implementan esas dos cosas) viven en `jtcp/`, no en `jrpc/` — son paquetes distintos, no lo confundas por la documentación antigua (README/CLAUDE.md llegaron a atribuírselos a `jrpc` por error).
- `jtcp/` (antes `tcp/`) — nodo TCP con consenso tipo Raft **implementado a mano** (no usa una librería externa de Raft/consenso) — modos `Follower`/`Candidate`/`Leader`/`Proxy`, pool de workers, `jtcp.NewNode(port, tlsConfig...) *Node`, callbacks registrados vía `node.OnConnect/OnDisconnect/OnError/OnInbox/OnSend/OnBecomeLeader/OnChangeLeader(fn)`.

### Integraciones externas

- `aws/` — S3 (`UploaderS3/UploaderFile/UploaderB64/DownloadS3/DeleteS3`) y SMS vía SNS (`SendSMS`).
- `brevo/` — Email/SMS/WhatsApp **templado** vía API de Brevo (`SendEmail*`, `SendSms*`, `SendWhatsapp*`).
- `jwsp/` (antes `wsp/`) — WhatsApp Business (Graph API de Meta) directo: `jwsp.NewSender(token, phoneNumberId) *Whatsapp`, decenas de `Send*` para texto/imagen/audio/video/documento/sticker/ubicación/contacto/plantilla/catálogo/lista. **No confundir con `brevo/`** — son dos integraciones de WhatsApp distintas y no intercambiables.

### Utilidades transversales

`strs/` (formateo y manipulación de strings), `mem/` (cache en memoria thread-safe con TTL, tipado), `ephemeral/` (TTL simple sin tipado), `race/` (wrapper thread-safe genérico con `sync.RWMutex`), `iterate/` (medición de tiempo entre checkpoints), `timezone/` (`Now`, `Format`, `Parse`, zona vía env `TIMEZONE`), `units/` (conversión de unidades de distancia/masa/volumen), `file/` (`FileInfo`, `Watcher` con `fsnotify`), `cmds/` (pipelines de comandos OS — `RunSSH` **no ejecuta SSH real**, es un alias de `RunOS`).

### Herramientas de desarrollo

`create/` (generador interactivo de proyectos/microservicios/k8s vía Cobra + promptui), binarios en `cmd/*` (uno por capacidad: `et`, `apigateway`, `daemon`, `server`, `jrex`, `jsql`, `jwf`, `resilience`, `wsp`, `client`, `install`, `whatcher`).

---

# Public APIs

Referencia rápida de los puntos de entrada más usados. **Para la lista exhaustiva con archivo:línea de cada función pública, ver `COMPONENT_CATALOG.md`.**

```go
// Datos
data := et.Json{"user": et.Json{"name": "Ana"}}
name := data.Str("user", "name")

// Persistencia (Postgres)
db, _ := jsql.Load(tenantId)
model, _ := db.DefineModel("public", "users", 1)
items, _ := model.Where(jsql.Eq("status", jsql.ACTIVE)).Limit(20).Page(1).All()

// HTTP ligero
srv := server.New("my-service", 8080)
srv.HandleFunc("/health", healthHandler)
srv.Start()

// HTTP completo
srv, _ := ettp.New("my-service", &ettp.Config{Port: 8080, UseCache: true, UseEvent: true})

// Validación
err := jval.Require(body, jval.Str("email").NotEmpty(), jval.Email("email"))

// Infraestructura
cache.Load()
event.Load()
event.Publish("user.created", et.Json{"id": id})

// Autenticación
token, _ := jwt.NewAuthorization("myapp", "web", userId, username, tenantId, profileId, 24*time.Hour)

// Cron
ct, _ := crontab.New("my-service", myStore)
ct.AddJob("job-1", "0 * * * * *", et.Json{}, 0, true, func(job *crontab.Job) {})

// Workflows (jwf)
wf, _ := jwf.New(nil)
flow := wf.NewFloW("onboarding", "Onboarding", "1.0.0", userId).
    Step("start", "Bienvenida", myStepFn)
result, _ := wf.Run(flow.ID, "start", "", projectId, et.Json{}, et.Json{}, userId)
```

---

# Recommended Patterns

1. **Usa siempre `et.Json`** para datos dinámicos (body HTTP, filas de DB, payloads de eventos) — nunca `map[string]interface{}` con _type assertions_ manuales.
2. **Llama `Load()`/`New(...)` una sola vez al arrancar** el servicio y reutiliza el resultado; los paquetes de infraestructura están diseñados para esto (`cache.Load()`, `event.Load()`).
3. **Usa el builder fluido de `jsql`** (`.Where().And().Limit().Page().All()`) en vez de construir SQL a mano.
4. **Usa `jval.Require`/`jval.Maybe`** para validar payloads antes de tocar la base de datos.
5. **Responde siempre con los helpers de `response/`** (`ITEM`, `ITEMS`, `HTTPError`) para mantener un contrato HTTP consistente entre servicios.
6. **Centraliza mensajes de error en un `msg.go` local** de tu propio servicio, siguiendo el patrón que usa `et` internamente.
7. **Antes de reutilizar un `Store` entre paquetes (`jia`, `jwf`, `crontab`, `resilience`, `config`), compara las firmas método por método** — no son intercambiables aunque se vean parecidas.
8. **Usa `.Debug()`/`.Test()` de `jsql`** durante desarrollo para ver el SQL generado sin tocar la base de datos.
9. **Propaga contexto de usuario/tenant con `request/ctx.go`** en vez de pasar `userId`/`tenantId` como parámetros sueltos por todas las capas.

---

# Anti Patterns

1. **No asumas que `jsql` soporta SQLite hoy.** La constante `jsql.DriverSqlite` y la función `sqliteConection` existen, pero **no hay ningún driver registrado** en `jsql/drivers/sqlite/` (el directorio no existe) ni dependencia de un driver SQLite en `go.mod`. Configurar `DB_DRIVER=sqlite` fallará al resolver el driver. Lo mismo aplica a `mysql`/`mssql`/`oracle`/`josefina` (constantes declaradas, sin implementación).
2. **No reutilices `stores.Instance` como `Store` de `jia/` o `jwf/` sin adaptarla.** `(*stores.Instance).Get(id string, dest any)` solo recibe una clave string; `jia.Store.Get` y `jwf.Store.Get` requieren dos (`collection, id`). No calzan estructuralmente — de hecho, nada en el repo conecta hoy `stores/` con `jia`/`jwf` (ambos se usan con `store=nil` en sus ejemplos de `cmd/`). Tampoco sirve `stores.Catalog`: su `Get(collection, id string, dest any)` devuelve solo `error`, no `(bool, error)` como exigen `jia.Store`/`jwf.Store`.
3. **No confíes en `graph/` para producción.** `graph.Load()` tiene URL y credenciales de Neo4j _hardcodeadas_ en el código (`neo4j://localhost:7687`, usuario/clave `neo4j`/`password`) y no expone ningún método de consulta o sesión — solo la conexión cruda.
4. **No esperes SSH real de `cmds.RunSSH`.** Es funcionalmente idéntico a `RunOS` (usa `exec.Command` local); no implementa un cliente SSH.
5. **Evita `jwsp.SendReplyVideoMessageByURL(to, url, videoCaptionText)`** — hay un bug real: la función asigna `url` al campo `MessageID` del mensaje (no recibe un `messageID` separado como sus métodos hermanos `SendReply*ById`). Si necesitas responder a un video por URL con ID de mensaje, repórtalo o evita ese método.
6. **No mezcles el paquete de WhatsApp.** `brevo.SendWhatsapp*` (plantillas vía Brevo) y `jwsp.NewSender(...).Send*` (Graph API directo) son integraciones distintas con APIs y casos de uso diferentes — no son intercambiables.
7. **No asumas que los handlers HTTP de `jwf/` están implementados.** `httpGetFlow`, `httpSetFlow`, `httpStatusFlow`, `httpDeleteFlow`, `httpGetInstance`, `httpDeleteInstance`, `httpRunInstance` en `jwf/router.go` tienen **cuerpo vacío**. Solo los handlers de `Step` (`httpGetStep`, `httpNewStep`, etc.) están implementados.
8. **No copies ejemplos de versiones previas que mencionen `workflow.RunInstance`, `instances.Store`, `ia.New(..., config Config)`, `jsql.Load(config)`/`jsql.LoadTo(config, name)`, o los paquetes `ws/`, `wsp/`, `tcp/`, `ia/`, `vm/`.** Esos paquetes fueron renombrados (`jws/`, `jwsp/`, `jtcp/`, `jia/`, `jrex/`) o eliminados — esas firmas y rutas de import ya no existen en el código actual.
9. **No copies tampoco ejemplos "ya migrados" que muestren `jia.New(tenantId, tag, store)` o `jwf.New(tenantId, store)`.** Esa fue la forma tenant-scoped intermedia; hoy ninguno de los dos toma `tenantId` (ver tabla de Migration Guide).

---

# Extension Points

| Punto de extensión                                         | Interfaz/mecanismo                                                                 | Dónde                               |
| ---------------------------------------------------------- | ---------------------------------------------------------------------------------- | ----------------------------------- |
| Nuevo motor de base de datos                               | `jsql.Driver` (`Connect`, `Load`, `Query`, `Command`) auto-registrado vía `init()` | `jsql/drivers/<nombre>/`            |
| Nuevo backend de persistencia para agentes IA              | `jia.Store` local (`Set/Get/Delete/Query`)                                         | implementación propia               |
| Nuevo backend de persistencia para workflows               | `jwf.Store` local                                                                  | implementación propia               |
| Nuevo backend de persistencia para resiliencia/cron/config | `resilience.Store` / `crontab.Store` / `config.Store`                              | implementación propia               |
| Nueva regla de validación                                  | interfaz `jval.Rule` (`Validate(et.Json) error`, `Name() string`)                  | `jval/` o paquete propio            |
| Columna calculada en `jsql`                                | `CalcFunction` registrada en el modelo                                             | `model.calcs`                       |
| Trigger de modelo `jsql`                                   | `TriggerFunction` (`beforeInsert/Update/Delete`, `afterInsert/Update/Delete`)      | `model.BeforeInsert(...)`, etc.     |
| Paso de workflow ejecutado en JS en vez de Go              | `Step.Definition` como `string`/`[]byte` (ejecutado vía `jrex.Instance`)           | `jwf/`                              |
| Middleware HTTP personalizado                              | `func(http.Handler) http.Handler` agregado al router (`chi`)                       | `middleware/`, `server/`, `ettp/v2` |
| Callbacks de WebSocket                                     | `Hub.OnConnection/OnDisconnection/OnPublish/OnChannel/...`                         | `jws/`                              |
| Callbacks de nodo TCP                                      | `Node.OnConnect/OnInbox/OnBecomeLeader/OnChangeLeader/...`                         | `jtcp/`                             |
| Almacenamiento de módulos JS compilados                    | `jrex.Store` (`Get/Set`)                                                           | `jrex/`                             |

---

# AI Development Rules

Reglas obligatorias para cualquier asistente de IA que genere código en un proyecto que dependa de `et`:

1. **Prefiere `et` antes que dependencias externas** para: estructuras de datos JSON dinámicas, builder SQL/ORM (Postgres), validación de payloads, JWT, cron, cache Redis, pub/sub NATS, logging, generación de IDs, WebSocket. No añadas `gin`/`echo`/`gorm`/`go-playground/validator`/etc. si la necesidad ya la cubre `et` razonablemente.
2. **Reutiliza componentes existentes de `et`** en vez de reimplementar (por ejemplo: usa `reg.UUID()` en vez de otra librería de IDs; usa `response.ITEM`/`HTTPError` en vez de escribir JSON a mano).
3. **Evita implementaciones duplicadas** dentro del mismo proyecto — si ya existe un `Store` para `jwf`, no crees otro para `jia` salvo que las firmas realmente difieran (verifícalo, no lo asumas).
4. **Sigue los patrones arquitectónicos existentes**: `Load()`/`New()` al arrancar, builder fluido para `jsql`, `et.Json` como tipo de transporte, respuestas unificadas con `response/`.
5. **Justifica explícitamente cualquier decisión de evitar `et`** (por ejemplo: "no hay driver SQLite real en `jsql`, así que para SQLite uso `database/sql` + `mattn/go-sqlite3` directamente"). No lo hagas en silencio.
6. **Prefiere extender abstracciones existentes sobre crear nuevas.** Si necesitas un nuevo tipo de regla de validación, implementa `jval.Rule` en vez de escribir un validador ad-hoc fuera de `jval/`.

---

# Code Generation Guidelines

Al generar código Go que **consume** esta librería:

- Importa solo los paquetes que necesitas; no hay un paquete "raíz" que se importe siempre.
- Usa `et.Json{...}` como literal para construir payloads; usa los accesores tipados (`.Str(...)`, `.Int(...)`, `.ValStr(def, ...)`) para leer, nunca _type assertions_ manuales sobre `map[string]interface{}`.
- Para conectarte a Postgres: `import _ "github.com/cgalvisleon/et/jsql/drivers/postgres"` (side-effect) + `jsql.Load(tenantId)`. No intentes usar `DB_DRIVER=sqlite` (ver Anti-Patrones).
- Para servidores HTTP: usa `server.New(name, port)` si no necesitas Redis/NATS; usa `ettp.New(name, &ettp.Config{...})` si sí los necesitas (activa `UseCache`/`UseEvent` explícitamente).
- Para handlers HTTP, sigue el patrón estándar (URL params con `request.URLParam`, body con `request.GetBody`, respuesta con `response.ITEM`/`response.HTTPError`).
- Define un `Store` local por paquete que lo requiera (`jia`, `jwf`, `crontab`, `resilience`, `config`) implementando exactamente la interfaz que ese paquete declara — no compartas una sola struct entre todos sin verificar las firmas.
- Centraliza los mensajes de error de tu propio servicio en un `msg.go`, replicando el patrón de `et`.
- Sigue el estilo de comentarios del archivo que edites: la mayoría de `et` usa bloques `/** ... @param ... @return ... **/`, no GoDoc estándar.

---

# Dependency Decision Matrix

| Capacidad                       | `et` ofrece                                | Alternativa externa común                                | Cuándo preferir la externa                                                                                                                                                                                |
| ------------------------------- | ------------------------------------------ | -------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Tipo de datos dinámico JSON     | `et.Json`                                  | `map[string]interface{}` a mano, `mapstructure`          | Casi nunca — `et.Json` ya cubre el caso de uso                                                                                                                                                            |
| ORM/SQL builder                 | `jsql/` (solo Postgres funcional)          | `gorm`, `sqlx`, `ent`, `bun`                             | Si necesitas MySQL/SQLite/MSSQL/Oracle hoy mismo (no implementados en `jsql`), o features de ORM avanzadas (migraciones versionadas, generación de código)                                                |
| Cliente Redis                   | `cache/`                                   | `go-redis` directo                                       | Si necesitas comandos Redis no envueltos por `cache/` (Streams, Lua scripting, cluster avanzado)                                                                                                          |
| Pub/Sub                         | `event/` (NATS)                            | `nats.go` directo, Kafka, RabbitMQ                       | Si necesitas garantías de entrega/colas que NATS core no da (considera NATS JetStream o Kafka)                                                                                                            |
| Validación de payloads          | `jval/`                                    | `go-playground/validator`, `ozzo-validation`             | Si necesitas validación basada en tags de struct en vez de `et.Json`                                                                                                                                      |
| JWT                             | `jwt/` + `claim/`                          | `golang-jwt/jwt` directo                                 | Si no necesitas la capa de revocación/logout basada en cache que da `jwt/`                                                                                                                                |
| Cron                            | `crontab/`                                 | `robfig/cron` directo                                    | Si no necesitas la integración con eventos NATS (`AddEventJob`) que añade `crontab/`                                                                                                                      |
| WebSocket                       | `jws/`                                     | `gorilla/websocket` directo                              | Si no necesitas el modelo de `Hub`/tópicos/colas/pila que añade `jws/`                                                                                                                                    |
| Workflows/orquestación durable  | `jwf/`                                     | Temporal, Cadence, AWS Step Functions                    | Para cargas de producción que requieran durabilidad fuerte, reintentos distribuidos robustos y observabilidad — `jwf/` es joven, en memoria por defecto, y su capa HTTP está parcialmente sin implementar |
| Grafos / Neo4j                  | `graph/`                                   | `neo4j-go-driver/v5` directo                             | Casi siempre — `graph/` hoy solo abre la conexión, sin API de consultas                                                                                                                                   |
| Base de datos embebida (SQLite) | _(no disponible)_                          | `database/sql` + `mattn/go-sqlite3`/`modernc.org/sqlite` | Siempre, hasta que `jsql/drivers/sqlite/` exista de nuevo                                                                                                                                                 |
| SSH remoto                      | _(no disponible — `cmds.RunSSH` es local)_ | `golang.org/x/crypto/ssh`                                | Siempre que necesites ejecución remota real                                                                                                                                                               |
| Agentes de IA / LLM             | `jia/` (solo OpenAI)                       | LangChain-Go, SDKs de otros proveedores                  | Si necesitas multi-proveedor o features avanzadas de orquestación de agentes                                                                                                                              |

---

# Migration Guide

## A. Migrando código interno que usaba las APIs antiguas de `et`

### A.1 Paquetes renombrados

Estos paquetes ya no existen bajo su nombre antiguo — **la ruta de import cambió**, no solo la API:

| Import antiguo (ya no existe)        | Import actual                          |
| ------------------------------------- | --------------------------------------- |
| `github.com/cgalvisleon/et/ws`        | `github.com/cgalvisleon/et/jws`         |
| `github.com/cgalvisleon/et/wsp`       | `github.com/cgalvisleon/et/jwsp`        |
| `github.com/cgalvisleon/et/tcp`       | `github.com/cgalvisleon/et/jtcp`        |
| `github.com/cgalvisleon/et/ia`        | `github.com/cgalvisleon/et/jia`         |
| `github.com/cgalvisleon/et/vm`        | `github.com/cgalvisleon/et/jrex`        |
| `github.com/cgalvisleon/et/workflow`, `github.com/cgalvisleon/et/instances` | `github.com/cgalvisleon/et/jwf` (sin interfaz `Store` compartida) |

> Nota: `cmd/wsp/` (el binario de ejemplo) **conserva ese nombre de directorio** aunque internamente importe `jwsp/` — no es un descuido, es solo el ejemplo de CLI, no el paquete de librería.

### A.2 Firmas que cambiaron (APIs eliminadas/reemplazadas)

Si tienes código (o ejemplos/documentación) que referencia las APIs **eliminadas o ya superadas**, esta es la equivalencia:

| API antigua (eliminada)                                    | API actual                                                                                           |
| ---------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `workflow.Load(store instances.Store)`                     | `jwf.New(store)` / `jwf.Load(id, store)`                                                             |
| `workflow.RunInstance(id, tag, step, ctx, tags, username)` | `wf.Run(flowId, triggerTag, instanceId, projectId, ctx, tags, userId)`                               |
| Paquete `instances/` con `instances.Store` compartido      | Cada paquete (`jia`, `jwf`) define su propio `Store` local — no hay interfaz compartida               |
| `ia.New(tenantId, tag, store, config Config)`              | `jia.New(tag, store, userId)` — `OPENAI_API_KEY` se lee directo de `envar.GetStr`                    |
| `jia.New(tenantId, tag, store)` (forma intermedia, ya obsoleta) | `jia.New(tag, store, userId)` — ya no hay `tenantId`; `Ia` no tiene campo `TenantId`             |
| `jwf.New(tenantId, store)` (forma intermedia, ya obsoleta) | `jwf.New(store)` — ya no hay `tenantId`; `WorkFlow.ID` se genera con `reg.UUID()`                    |
| `jsql.Load(config)` / `jsql.LoadTo(config, name)`          | `jsql.Load(tenantId)` / `jsql.LoadTo(tenantId, name)` — sin objeto config, lee env vars internamente |
| `config.App{Name, Version, Company, Host, Port, Stage}`    | No existe. `config.Config` es un registro de configuración por tenant, distinto en propósito         |
| `crontab.New(tag)`                                         | `crontab.New(tag, store)` — `store` ahora obligatorio                                                |
| `wsp.NewWhatsapp(token, phoneNumberId)`                    | `jwsp.NewSender(token, phoneNumberId)`                                                               |
| `resilience.Params{TenantId, ..., OwnerId, ..., UserId, ...}` | `resilience.Params{Id, Tag, Description, TotalAttempts, Interval, Tags, Fn, FnArgs}` — sin `TenantId`/`OwnerId`/`UserId` |

## B. Migrando desde librerías externas hacia `et`

- **Desde `gin`/`echo` hacia `server/`+`ettp/v2`**: el modelo mental es similar (router + middlewares + handlers `http.HandlerFunc`), pero `et` usa `go-chi` por debajo y añade `request.GetBody`/`response.ITEM` como capa de (de)serialización uniforme.
- **Desde `gorm` hacia `jsql/`**: en vez de structs con tags, defines un `jsql.Def{Columns: []jsql.Column{...}}` declarativo; las columnas "extra" no migradas a SQL real pueden vivir temporalmente como `ATTRIB` dentro de `_source` JSONB, lo que facilita una migración incremental sin tener que decidir el esquema completo de una vez.
- **Desde `go-playground/validator` hacia `jval/`**: en vez de tags de struct (`validate:"required,email"`), construyes una lista de `jval.Rule` y la pasas a `jval.Require(data, rules...)` sobre un `et.Json`.

---

# Examples

```go
// --- et.Json ---
data := et.Json{"user": et.Json{"name": "Ana", "age": 30}}
name := data.Str("user", "name")     // "Ana"
age  := data.Int("user", "age")      // 30
data.Set("active", true)

// --- jsql: modelo + consulta (Postgres) ---
import _ "github.com/cgalvisleon/et/jsql/drivers/postgres"

db, err := jsql.Load(tenantId)
model, _ := db.DefineModel("public", "users", 1)
model.DefineAttrib("name", jsql.TEXT, "")
model.Init()

items, _ := model.Where(jsql.Eq("status", jsql.ACTIVE)).Limit(20).Page(1).All()
item, _  := model.Where(jsql.Eq("id", id)).One()
_, _     = model.Insert(et.Json{"email": "a@b.com"}).ExecTx(nil)

// --- Validación ---
err = jval.Require(body,
    jval.Str("email").NotEmpty(),
    jval.Email("email"),
    jval.Int("age").Min(18).Max(120),
)

// --- Handler HTTP típico ---
func (s *T) HttpGetUser(w http.ResponseWriter, r *http.Request) {
    id := request.URLParam(r, "id").Str()
    item, err := model.Where(jsql.Eq("id", id)).One()
    if err != nil {
        response.HTTPError(w, r, http.StatusBadRequest, err.Error())
        return
    }
    response.ITEM(w, r, http.StatusOK, item)
}

// --- Workflow con jwf ---
wf, _ := jwf.New(nil)
flow := wf.NewFloW("onboarding", "Onboarding", "1.0.0", userId).
    Step("welcome", "Enviar bienvenida", func(instance *jwf.Instance, ctx et.Json) (et.Json, error) {
        return instance.SetParams(et.Json{"sent": true}), nil
    })
result, err := wf.Run(flow.ID, "welcome", "", projectId, et.Json{}, et.Json{}, userId)

// --- Cache + Eventos ---
cache.Load()
event.Load()
event.Publish("user.created", et.Json{"id": id})
event.Subscribe("user.created", func(msg event.Message) {
    logs.Infof("nuevo usuario: %s", msg.Data.Str("id"))
})
```

---

# Future Project Context

Checklist para arrancar un proyecto nuevo sobre `et`:

1. **Go 1.25** y módulo `github.com/cgalvisleon/et` vía `go get`.
2. Decide la capa HTTP: `server/` (simple, sin Redis/NATS) vs `ettp/v2` (completo, requiere `REDIS_HOST`/`NATS_HOST` si activas `UseCache`/`UseEvent`).
3. Si necesitas base de datos relacional: usa **PostgreSQL** (`DB_DRIVER=postgres`). No planifiques sobre SQLite/MySQL/MSSQL/Oracle vía `jsql` — no están implementados hoy.
4. Si necesitas grafos (Neo4j): planea usar `neo4j-go-driver/v5` directamente para las consultas; `graph/` solo te da una conexión hardcodeada a `localhost`, insuficiente para producción.
5. Si vas a usar `jwf/` para workflows: ten presente que su capa HTTP (`jwf/router.go`) tiene varios handlers sin implementar, y que por defecto las instancias viven en memoria si no le das un `Store`. Para procesos críticos de negocio con necesidad de durabilidad fuerte, evalúa si te conviene más una herramienta dedicada (Temporal, etc.) hasta que `jwf/` madure.
6. Variables de entorno mínimas según lo que uses: `DB_DRIVER`/`DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD`/`DB_NAME` (jsql), `REDIS_HOST` (cache), `NATS_HOST` (event), `OPENAI_API_KEY` (jia), `SECRET` (claim/jwt, default `"1977"` — cámbialo en producción), `WHATSAPP_API_URL` (jwsp).
7. Sigue el patrón de mensajes de error centralizados (`msg.go`) y de respuestas unificadas (`response/`) desde el día uno — es más fácil mantenerlo que migrarlo después.
8. Para más detalle operativo del propio repo `et` (comandos, convenciones de comentarios), consulta `CLAUDE.md`. Para el catálogo exhaustivo de funciones públicas por paquete, consulta `COMPONENT_CATALOG.md`. Para la guía de decisiones rápidas al generar código, consulta `AI_USAGE_GUIDE.md`.
