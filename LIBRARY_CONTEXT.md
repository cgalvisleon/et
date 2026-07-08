# LIBRARY_CONTEXT.md

> Documento de contexto persistente para asistentes de IA (Claude, ChatGPT, Cursor, Windsurf, Cline, etc.)
> Librería: **`github.com/cgalvisleon/et`** — Go 1.25 — MIT
> Copia este archivo en la raíz de cualquier proyecto que dependa de `et` para que el asistente diseñe soluciones coherentes con la librería.
>
> **Advertencia de vigencia:** este repositorio cambia muy rápido y de forma poco documentada (el historial de commits son casi todos "Backup:" sin mensaje descriptivo). Este documento fue regenerado leyendo el código fuente real. La versión anterior de este archivo tenía deriva significativa (ver "Correcciones importantes vs. versiones previas" más abajo) — antes de confiar en una firma o ruta de archivo aquí citada para una decisión importante, verifícala contra el código. Ver `CLAUDE.md` en la raíz del repo para contexto operativo (comandos, convenciones).

---

# Corrección importante vs. versiones previas de este documento

Si ya usaste una versión anterior de `LIBRARY_CONTEXT.md`/`COMPONENT_CATALOG.md`, estas son las correcciones de mayor impacto:

1. **El paquete `config/` fue eliminado por completo** — no existe ni el directorio. No hay `config.Config`, `config.Store`, `config.New`, `config.GetStr`, etc. Sus dos responsabilidades se separaron: los getters de entorno (`GetStr/GetInt/...`) viven ahora en `envar/`; el registro de configuración por tenant es `stores.Config`/`stores.DefineConfig` (forma distinta, ver más abajo). Cualquier referencia a `config.*` en documentación antigua describe un paquete que ya no existe.
2. **`jsql.Store` no es una interfaz inyectable** — es un **struct concreto** (`jsql/store.go`), la tabla genérica `kind`+`id` que antes se llamaba `stores.Catalog` y se movió dentro de `jsql`. Se crea con `jsql.DefineStore(db *DB, schema string) (*Store, error)`. La persistencia opcional de metadata de `*DB`/`*Model` usa este mismo struct concreto vía `(*DB).Save(store *Store) error` / `jsql.LoadDb(store *Store, id string) (*DB, error)` — **no** hay una interfaz `Store` inyectable en `jsql` como sí la hay en `jia`/`jwf`/`resilience`/`crontab`.
3. **`jwf.Store` no tiene un método `GenSerie`.** Su interfaz es exactamente `Set(collection, id, ownerId string, obj any) error` / `Get(collection, id string, dest any) (bool, error)` / `Delete(collection, id string) error` / `Query(collection string, query et.Json) (et.Items, error)` — el `Query` sí lleva un `collection` extra al inicio (a diferencia de `jia.Store.Query(query et.Json)`), pero no hay `GenSerie` en ningún lado de esta interfaz.
4. **`stores.Catalog` ya no existe** — se movió a `jsql.Store` (punto 2). `stores/` hoy expone `DefineInstance`/`LoadInstance` (+ variantes `Bite`), `DefineAuthorization`, y `DefineConfig` (el reemplazo del viejo `config.Config`, con forma mucho más simple: solo `TenantId`/`Stage`/`Tag`).
5. **`jsql.Load`/`LoadTo` ya no reciben `tenantId`**: son `jsql.Load() (*DB, error)` y `jsql.LoadTo(name string) (*DB, error)`. El tenant se lee internamente vía `envar.GetStr("DB_TENANT_ID", "tenant:root")`.

---

# Executive Summary

`et` es una **librería modular de utilidades para construir microservicios, CLIs y aplicaciones web en Go**. No es un framework monolítico: es un conjunto de más de 50 paquetes independientes, cada uno importable por separado, que cubren de punta a punta las necesidades habituales de un backend:

- **Modelo de datos universal**: `et.Json`, `et.List`, `et.Item`, `et.Items`.
- **Persistencia** agnóstica de motor con ORM ligero: `jsql/` (en la práctica, solo PostgreSQL funciona — ver Anti-Patrones).
- **Servidores HTTP** en varios niveles de abstracción: `server/` (ligero), `ettp/v2` (completo), `ettp/v1` (una variante paralela, no simplemente "legado" — ver Anti-Patrones), más `router/` y `middleware/`.
- **Validación declarativa** de payloads: `jval/` (y un segundo paquete de validación no relacionado, `validator/` — ver Anti-Patrones).
- **Autenticación/JWT**: `jwt/` + `claim/`.
- **Infraestructura**: Redis (`cache/`), NATS (`event/`), Neo4j (`graph/`, muy incompleto).
- **Entorno**: `envar/` (ya no existe `config/` como paquete separado — ver corrección #1 arriba).
- **Logging estructurado**: `logs/` (+ `stdrout/`, `color/`).
- **Orquestación**: cron (`crontab/`), workflows multi-paso basados en grafo (`jwf/`), agentes de IA sobre OpenAI (`jia/`), runtime JS embebido (`jrex/`), resiliencia/reintentos (`resilience/`).
- **Integraciones externas**: AWS (S3 + SNS/SMS — no SES pese al nombre del paquete), Brevo (email/SMS/WhatsApp templado), WhatsApp Business Graph API (`jwsp/`).
- **Comunicación**: WebSocket (`jws/`), RPC sobre TCP (`jrpc/`), nodo TCP con consenso tipo Raft propio (`jtcp/`).
- **Utilidades transversales**: IDs (ULID/UUID/XID/Snowflake en `reg/`), criptografía y validación de formato (`utility/`), strings (`strs/`), memoria/concurrencia (`mem/`, `ephemeral/`, `race/`, `iterate/`), tiempo/unidades (`timezone/`, `units/`), colas en memoria (`queue/`), generación de proyectos (`create/`).

**Idea central**: cualquier dato dinámico que entra o sale de un servicio (body HTTP, fila de base de datos, mensaje de evento, claim JWT) se representa como `et.Json` (`map[string]interface{}`) con accesores tipados y valor por defecto. Esto evita el patrón `val, ok := m["x"].(string)` repetido en toda la base de código.

**Estado real del proyecto** (importante para decisiones de adopción): es una librería viva, en evolución muy activa y a veces inconsistente entre paquetes hermanos. Paquetes enteros documentados en versiones anteriores de este archivo (`config/`, `stores.Catalog`) ya no existen. Algunos paquetes están notablemente incompletos o tienen bugs confirmados en código (`graph/` es casi un stub; `cache.Close()`/`event.Close()` tienen recursión infinita; `response.ITEM`/`ITEMS`/`DATA` tienen un chequeo de "vacío" que nunca se cumple; `jval.Maybe` corta la validación de más campos al primero ausente). Trátala como una librería de utilidades sólida en su núcleo (`et`, `jsql` sobre Postgres, `cache`, `event`, HTTP) y experimental/con bugs puntuales en sus bordes (`jwf`, `jtcp`, `graph`, partes de `ettp/v2`).

---

# Design Philosophy

1. **"JSON como lingua franca"**: `et.Json` cruza todas las capas — body HTTP, filas de base de datos (`_source` JSONB), mensajes NATS, claims JWT, resultados de workflows. Un solo tipo, un solo conjunto de accesores (`Str`, `Int`, `Int64`, `Num`, `Bool`, `Time`, `Json(attr)`, `Array...`, `ValStr(def, ...atribs)`, etc. — ver `et/json.go`).
2. **Modularidad por import, no por imposición**: no hay un "framework" que envuelva la app. Cada paquete se importa solo si se necesita. `jsql` no requiere `cache`; `cache` no requiere `event`; `ettp/v2` **ya no tiene** flags `UseCache`/`UseEvent` — nunca llama `cache.Load()`/`event.Load()` por sí solo, el consumidor debe llamarlos antes si los necesita.
3. **`Load()` idempotente vs. `New()`/`Load(...)` explícito**: los paquetes de infraestructura (`cache`, `event`) exponen `Load()` seguro de llamar varias veces (singleton perezoso). Los paquetes de orquestación (`jia`, `jwf`, `crontab`, `resilience`) usan `New(...)` (crea una instancia nueva con `ID` propio vía `reg.UUID()`) y `Load(...)` (carga una instancia existente por ese ID) — no hay singleton global en ninguno de ellos.
4. **Inversión de dependencias vía interfaces pequeñas y locales, con convergencia real de forma**: la librería define interfaces de persistencia (`jia.Store`, `jwf.Store`, `resilience.Store`, `crontab.Store`) y el consumidor las implementa. `jia.Store`, `resilience.Store` y `crontab.Store` comparten **exactamente** la misma forma: `Set(collection, id, ownerId string, obj any) error` / `Get(collection, id string, dest any) (bool, error)` / `Delete(collection, id string) error` / `Query(query et.Json) (et.Items, error)`. `jwf.Store` es casi igual pero su `Query` lleva un `collection string` extra al inicio — **no** tiene `GenSerie` (una versión previa de este documento lo afirmaba; es incorrecto). `jrex.Store` es un subconjunto intencional de 2 métodos (`Set`/`Get`, sin `Delete`/`Query`). `jsql` **no** tiene una interfaz `Store` inyectable — usa un struct concreto propio (`jsql.Store`, la ex-`stores.Catalog`) para persistir su propia metadata, de forma opcional.
5. **APIs fluidas / encadenables**: `model.Where(...).And(...).Limit(20).Page(1).All()`, `flow.Step(tag, title, fn).Step(...)`, `jval.Require(body, jval.Str("email").NotEmpty())`.
6. **Agnosticismo de driver (en teoría)**: `jsql` define el contrato (`Driver` interface en `jsql/driver.go`) y los drivers se auto-registran con `init()` al importarse como side-effect. En la práctica solo `postgres` tiene implementación real hoy.
7. **Esquema híbrido relacional/documental**: las tablas de `jsql` tienen columnas reales (`COLUMN`) y atributos dentro de una columna `_source JSONB` (`ATTRIB`), permitiendo evolución de esquema sin migraciones constantes, sin perder capacidad de consulta SQL (`_source->>'campo'`).
8. **Mensajes de error centralizados**: casi todos los paquetes tienen un `msg.go` (o el paquete raíz `msg/`) con constantes de error — usarlas en vez de strings literales repetidos.
9. **Respuestas HTTP unificadas, con una colisión de nombres a tener presente**: `response.ITEM`/`ITEMS`/`JSON`/`HTTPError` son la capa "plana"; `middleware.Metrics` (usada por `ettp/v2`) tiene métodos con **los mismos nombres** (`JSON`, `ITEM`, `ITEMS`, `HTTPError`) pero además registra telemetría — no son intercambiables aunque el nombre sea igual.
10. **Estilo de comentarios no-GoDoc real**: la mayoría del código usa bloques `/** ... @param ... @return ... **/` en vez de comentarios GoDoc estándar (`// Func: ...`). Si generas código para este repo, sigue la convención del archivo que estés editando.

---

# Architecture Overview

No existe un punto de entrada central ni un `et.App` que arranque "todo". Cada servicio compone los paquetes que necesita.

```
Capa 1 — Utilidades autosuficientes (sin servicios externos)
  et, utility, strs, reg, jval, validator, logs, stdrout, color, envar,
  mem, ephemeral, iterate, race, timezone, units, file, queue, msg

Capa 2 — Infraestructura (requieren servicios externos vía env vars)
  cache    -> Redis
  event    -> NATS
  graph    -> Neo4j (credenciales hardcodeadas, sin API de consultas)
  jsql     -> PostgreSQL (único driver funcional)

Capa 3 — HTTP / routing (construidos sobre go-chi)
  server, ettp/v1, ettp/v2, router, middleware, response, request, jws

Capa 4 — Aplicación / orquestación (compone capas 1-3)
  jwf (workflows), jia (agentes IA), jrex (runtime JS), crontab,
  resilience, service, jrpc, jtcp

Capa 5 — Integraciones externas
  aws (S3/SNS), brevo (email/SMS/WhatsApp templado), jwsp (WhatsApp Graph API)

Capa 6 — Persistencia de aplicación
  stores (instancias, autorización, config por tenant — todo jsql-backed)

Herramientas de desarrollo
  cmd/* (binarios), create (generador de proyectos), cmds (ejecución de pipelines), jcli (huérfano)
```

**Patrones estructurales clave:**

- **Sincronización entre réplicas vía eventos NATS**: `ettp/v2` sincroniza el estado del router entre instancias usando eventos publicados por el paquete `router/` (constantes reales: `APIGATEWAY_SET_ROUTER`/`APIGATEWAY_REMOVE_ROUTER`/`APIGATEWAY_RESET_ROUTER`, no literalmente `EVENT_*`), con bandera `m.Myself` para evitar bucles de auto-procesamiento.
- **Patrón Store inyectado**: `jia.New(tag, store, userId)`, `jwf.New(store, userID)`, `crontab.New(tag, store)`, `resilience.New(store)` — cada uno con su propio `ID` generado internamente (`reg.UUID()`), sin `tenantId`. Cargar una instancia existente es `jia.Load(id, store)` / `jwf.Load(store, id)` (nótese el orden: `jwf.Load` recibe `store` primero).
- **Patrón Driver auto-registrado**: `import _ "github.com/cgalvisleon/et/jsql/drivers/postgres"` registra el driver vía `init()`; `jsql.Load()` lo resuelve internamente leyendo `DB_DRIVER` con `envar.GetStr`.
- **Debug/Test transversal en `jsql`**: `Model`, `Query` y `Command` soportan `.Debug()` (loguea SQL sin ejecutar) y `.Test()` (devuelve SQL sin ejecutar).
- **Contexto de request enriquecido**: `request/ctx.go` propaga `tenantId`, `userId`, `username`, `profileId`, `app`, `device`, `payload` a través de `context.Context`, poblado por `middleware.Authentication` y leído por handlers (`request.TenantId(r)`, `request.UserId(r)`, etc.).
- **Motor de workflows basado en grafo, no en lista lineal**: `jwf/` modela un `Flow` como un conjunto de `Step`s conectados por `Connection`s con puertos (`output`/`error`), no como una secuencia fija.
- **RPC "de facto" para servicios**: `jrpc/` (registro simple `Solver{Host,Port}` + `net/rpc` estándar) es el mecanismo de RPC real usado; `ettp/v2` también abre un listener TCP (`Config.RpcPort`) y tiene un codec gob propio en `pipe.go`, pero la función que aceptaría conexiones en ese listener (`startPipe`) **nunca se llama** en ningún lado del código — es una funcionalidad iniciada pero no conectada.

---

# Core Components

### Núcleo de datos — `et/`

| Tipo                                 | Propósito                | API clave                                                                                                                                                                                                                                                                                                                                                                                              |
| ------------------------------------- | ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `et.Json` (`map[string]interface{}`) | Tipo universal de datos  | `Str`, `Int`, `Int64`, `Num`, `Bool`, `Time`, `Json(attr)`, `Array`, `ArrayStr/Int/Int64/Number/Bytes/Json`, `MapStr/Int/Float`, `ValStr/ValInt/ValInt64/ValNum/ValBool/ValTime/ValJson/ValArray(def, ...atribs)`, `Get`, `Set`, `SetNested`, `Delete`, `Exist`, `Remove`, `Select`, `Hidden`, `Clone`, `Update`, `Compare`, `Append`, `IsChanged`, `IsDeferent`, `ToByte/ToString/ToMap/ToEscapeHTML` |
| `et.List`                            | Resultado paginado        | `Rows`, `All`, `Count`, `Page`, `Start`, `End`, `Result []Json`                                                                                                                                                                                                                                                                                                                                        |
| `et.Item`                            | Resultado de un registro  | `Ok bool`, `Result Json` + mismos accesores tipados que `Json`, `NewItem(data)`                                                                                                                                                                                                                                                                                                                        |
| `et.Items`                           | Resultado multi-registro  | `NewItems(data)`, `Add`, `AddMany`, `One(idx)` (índice 1-based o negativo desde el final), `First`, `Last`, `ToList(all, page, rows)`, accesores indexados                                                                                                                                                                                                                                            |

> **Regla de oro**: para leer/escribir datos dinámicos (JSON, filas de DB, payloads), usa `et.Json` y sus accesores — nunca `map[string]interface{}` a mano ni _type assertions_ manuales.

### Persistencia — `jsql/`, `stores/`

- `jsql.Load() (*DB, error)` / `jsql.LoadTo(name string) (*DB, error)` — **sin `tenantId` como parámetro**; se lee internamente vía `envar.GetStr("DB_TENANT_ID", "tenant:root")`. Todo lo demás (`DB_DRIVER`, `DB_HOST`, etc.) también vía `envar` directo — ya no hay paquete `config`.
- Modelos: `db.DefineModel(schema, name, version)` (agrega `id`, `created_at`, `updated_at`, `_source`, `_idx`), `db.NewModel(...)` (manual), `db.Define(jsql.Def{...})` (declarativo, preferido).
- Tipos de columna: `COLUMN`, `ATTRIB` (dentro de `_source` JSONB), `DETAIL`/`ROLLUP` (relaciones virtuales), `CALCFUNC`/`CALC` (computadas), `AGG` (agregaciones).
- Consultas/comandos fluidos: `.Where(jsql.Eq(...)).And(...).Limit().Page().All()/.One()`, `.Insert(...)`, `.Update(...)`, `.Upsert(...)`, `.Delete()`, todos con `.ExecTx(tx)`/`.Exec()`.
- Triggers: `beforeInserts/Updates/Deletes`, `afterInserts/Updates/Deletes` (`TriggerFunction`); columnas calculadas vía `CalcFunction`.
- Paths anidados JSONB: `"ventas->detalle->precio"` se traduce automáticamente a `->`/`->>` con casts.
- **`jsql.Store`** (`jsql/store.go`) es un **struct concreto**, no una interfaz — la tabla genérica `kind`+`id` (ex-`stores.Catalog`): `jsql.DefineStore(db, schema) (*Store, error)`. Se usa opcionalmente para persistir la metadata del propio `*DB`/`*Model` vía `(*DB).Save(store *Store) error` / `jsql.LoadDb(store *Store, id string) (*DB, error)` — si no se usa, no pasa nada, el flujo normal de conexión (`Load`/`LoadTo`) no lo requiere.
- `stores/` — helpers jsql-backed **distintos** de `jsql.Store`: `DefineInstance`/`LoadInstance` (+ `Bite`) (registro genérico tipo "instancia", solo 1 clave string), `DefineAuthorization` (permisos, cacheado vía `dt`), `DefineConfig(db, tenantId, schema, stage, tag) (*Config, error)` (el reemplazo del viejo `config.Config`, mucho más simple: solo `TenantId`/`Stage`/`Tag`).

### HTTP y routing

| Paquete       | Nivel                    | Cuándo usarlo                                                                                                                                                                                     |
| ------------- | ------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `server/`     | Ligero (`Ettp` sobre `chi.Mux`) | Servicios sin Redis/NATS. `server.New(name, port) *Ettp`, `.Use(...)`, `.HandleFunc`, `.Mount`, `.Start()`, `.OnStart/.OnClose`                                                              |
| `ettp/v2`     | Completo                  | `ettp.New(name string, cnf *Config) (*Server, error)` — **ya no** hay `UseCache`/`UseEvent`; `New` nunca llama `cache.Load()`/`event.Load()`, el consumidor debe hacerlo antes si los necesita |
| `ettp/v1`     | Paralelo, no simple legado | Sigue recibiendo commits; tiene archivos sin equivalente en v2 (`server-apigateway.go`, `server-proxy.go`, etc.) — verifica cuál importa tu binario antes de asumir "usa siempre v2"           |
| `router/`     | Standalone                 | Routing con anuncio de rutas vía NATS: `Public/Private/Protect/With`, `PushApiGateway`, `RemoveApiGateway`, `GetRoutes`                                                                          |
| `middleware/` | Transversal                | `Authentication` (no "Authenticate"), `Logger`, `Recoverer`, `RequestID`, `AllowAll` (CORS), `Metrics`/telemetría                                                                                |
| `response/`   | Salida                     | `ITEM`, `ITEMS`, `DATA`, `JSON`, `RESULT`, `ANY`, `Stream`, `HTTPError`, `HTTPAlert`, `Unauthorized`, `Forbidden`, `InternalServerError`                                                        |
| `request/`    | Entrada                   | `GetBody(r)`, `URLParam(r, key).Str()/.Int()/...`, `Query(r, key)`, contexto: `TenantId`, `UserId`, `Username`, `ProfileId`, `App`, `Device`, `Payload`; cliente HTTP saliente (`Post/Get/Put/...`) con su propio tipo `Status` |
| `jws/`        | WebSocket                 | `jws.New() *Hub`, `hub.Connect/SendTo/Topic/Queue/Stack/Publish`                                                                                                                                 |

### Validación

- `jval/` — validadores tipados y encadenables sobre `et.Json`: `Str`, `Int`, `Float`, `Bool`... **no**, en realidad no hay `Bool`/`Time` — los constructores reales son `Str`, `Int`, `Float`, `Array`, `Email`, `Date`, `Enum`, `Phone`, `Between`, y `Validate(name, rules...)` para objetos anidados. Se agrupan con `jval.Require(data, rules...)` (todas obligatorias) o `jval.Maybe(data, rules...)` (opcionales — con una particularidad, ver Anti-Patrones).
- `validator/` — un **segundo** paquete de validación, no relacionado con `jval/` (tipos `Validator`/`Field`/`Condition`, mensajes i18n vía env `LANG`). No compartas tipos entre ambos.

### Infraestructura — `cache/`, `event/`, `graph/`

- `cache/` — Redis. `cache.Load()`, `Set/Get/Delete/Exists/Expire/Incr/Decr`, listas (`LPush/LRange/...`), colecciones hash, objetos JSON (`SetObject/GetObject`, `ObjetSet/ObjetGet` — sic, no "Object"), verificación/OTP (`SetVerify/GetVerify`), `cache.Metrics`. **`(*Conn).Close()` tiene un bug de recursión infinita** (se llama a sí mismo en vez del `Close` del cliente Redis embebido) — no lo uses para cerrar limpiamente.
- `event/` — NATS. `event.Load()`, `Publish(channel, data)`, `Subscribe(channel, fn)` (broadcast), `Queue(channel, queue, fn)` (reparto entre workers), `Stack(channel, fn)` (alias de `Queue` con cola `"stack"`), `event.Message{Data et.Json, Myself bool, ...}`. **Mismo bug de recursión infinita en `(*Conn).Close()`.**
- `graph/` — Neo4j. `graph.Load() (*Conn, error)` con URL y credenciales **hardcodeadas** (`neo4j://localhost:7687`, `neo4j`/`password`) — no lee ninguna variable de entorno, y `*Conn` no tiene ni un solo método de consulta/sesión. No usable en producción tal cual.

### Entorno y logging

- `envar/` — acceso a variables de entorno (`GetStr/GetInt/GetInt64/GetFloat/GetBool`, `Get/Set` a nivel de proceso) y argumentos CLI (`ArgStr/ArgInt/...`), `Validate(keys)`. **Es el único paquete de acceso a entorno** — `config/` fue eliminado.
- `logs/` — `Log`, `Info(f)`, `Alert(f)`, `Error(f)`, `Debug(f)`, `Fatal`, `Panic`, `Tracer`, todo vía `stdrout/` (color ANSI, `color/`).

### Identidad, autenticación y seguridad

- `reg/` — generadores de ID: `UUID()`, `ULID()`, `XID()`, más variantes "tag" y "get-or-generate" (`GetULID(id)` retorna `id` si ya es válido, o genera uno nuevo si está vacío/`"*"`/`"new"`). **Es solo generación de IDs, no hay descubrimiento de servicios** pese al nombre del paquete. `GenSnowflake()` no usa realmente el generador de la librería `bwmarrin/snowflake` (solo su constante `Epoch`) — es vestigial.
- `claim/` — claims JWT (`tenantId`, `profileId`, `payload`), `NewClaim`, `NewToken` (HS256 vía `golang-jwt/jwt/v4`), `ParceToken` (sic, no "Parse" — typo consistente en todo el repo). Secreto vía env `SECRET`, default inseguro `"1977"`.
- `jwt/` — capa sobre `claim/` + `cache/` (revocación server-side): `NewAuthentication`, `NewAuthorization`, `NewAppToken`, `NewEphemeralToken`, `Validate`, `RenewToken`, `DeleteToken` (logout).
- `utility/` — criptografía (`Encrypt` con MD5/SHA1/SHA256/SHA512/AES, `DecryptoAES` — sic), generación de IDs simples, validadores de formato (`ValidEmail`, `ValidPhone` — solo 10 dígitos, no internacional), `AppWait()` (bloquea hasta SIGINT/SIGTERM, usado por varios `cmd/*`).

### Orquestación

- `crontab/` — modelo orientado a eventos con singleton de paquete: `crontab.Load(tag, store) error` (arranca el singleton); registra jobs con `crontab.CronJob(tag, ownerId, spec Cron, repetitions, params, fn) error` (recurrente, `Cron{DayOfWeek, Month, DayOfMonth, Hour, Minute}` estructurado) o `crontab.ScheduleJob(tag, ownerId, spec time.Time, params, fn) error` (una sola ejecución). Nada en el repo importa `crontab` hoy.
- `jwf/` — workflows basados en grafo. `jwf.New(store, userID) (*WorkFlow, error)` (llama `cache.Load()`+`event.Load()`, genera `WorkFlow.ID` con `reg.UUID()`); `jwf.Load(store, id) (*WorkFlow, error)` carga por ese ID. **No** tiene un mapa `Instances` en memoria — las instancias se cargan/guardan bajo demanda vía el `Store`. Ver detalle completo en `COMPONENT_CATALOG.md`.
- `resilience/` — reintentos con intervalo: `resilience.New(store)`, `LoadInstance(Params{...})`, `instance.Run(userId)`.
- `jia/` — agentes sobre OpenAI (`openai-go/v3`): `jia.New(tag, store, userId) (*Ia, error)`; `jia.Load(id, store) (*Ia, error)`. Gestiona `Agent`/`Participant`/`Conversation` (con `Message`s), expone su propio router HTTP y un `Sender` para envío externo (sin setter público hoy — efectivamente sin conectar).
- `jrex/` — runtime JS embebido (`dop251/goja`) con hot-reload, usado tanto standalone como motor de ejecución de pasos de `jwf/` cuando `Step.Definition` es un string JS.
- `service/` — OTP y mensajería multicanal (`SendOTPEmail/SendOTPSms/VerifyOTP`, `SendSms/SendWhatsapp/SendEmail`), delega en `aws/`/`brevo/`.

### Comunicación de bajo nivel

- `jrpc/` — RPC sobre TCP con `net/rpc` estándar. `jrpc.Mount(host, port, services, packageName) (*Package, error)` registra un servicio. **No tiene balanceador de carga ni consenso Raft** — eso vive en `jtcp/`, no aquí.
- `jtcp/` — nodo TCP con consenso Raft **implementado a mano** (no una librería externa) — modos `Follower`/`Candidate`/`Leader`/`Proxy`, `jtcp.NewNode(port, tlsConfig...) *Node`, balanceador round-robin L4 (`balancer.go`) cuando el modo es `Proxy`.

### Integraciones externas

- `aws/` — solo **S3** y **SNS/SMS** (no hay SES pese al nombre del paquete): `UploaderS3/UploaderFile/UploaderB64/DeleteS3/DownloadS3`, `SendSMS`.
- `brevo/` — Email/SMS/WhatsApp **templado** vía API HTTP de Brevo (`SendEmail*`, `SendSms*`, `SendWhatsapp*`).
- `jwsp/` — WhatsApp Business (Graph API de Meta) directo: `jwsp.NewSender(token, phoneNumberId) *Whatsapp`, decenas de `Send*`. **No confundir con `brevo/`** — integraciones distintas.

### Utilidades transversales

`strs/` (formateo y manipulación de strings), `mem/` (cache en memoria thread-safe con TTL, tipado), `ephemeral/` (TTL simple sin tipado), `race/` (wrapper thread-safe genérico), `iterate/` (medición de tiempo entre checkpoints), `timezone/` (`Now`, `Format`, `Parse`, zona vía env `TIMEZONE`), `units/` (conversión de unidades), `file/` (`FileInfo`, `Watcher` con `fsnotify`), `queue/` (cola de batching genérica en memoria, con el único test real del repo), `validator/` (ver Validación), `msg/` (constantes de error compartidas).

### Herramientas de desarrollo

`create/` (generador interactivo de proyectos/microservicios/k8s vía Cobra + promptui — nota: su plantilla `server.New("$2")` está desactualizada frente a la firma real `server.New(name, port)`), binarios en `cmd/*`, `cmds/` (pipelines de comandos OS — `RunSSH` no ejecuta SSH real, es alias de `RunOS`), `jcli/` (huérfano: declara `package jrex` en su propio directorio, no lo importa nadie).

---

# Public APIs

```go
// Datos
data := et.Json{"user": et.Json{"name": "Ana"}}
name := data.Str("user", "name")

// Persistencia (Postgres)
import _ "github.com/cgalvisleon/et/jsql/drivers/postgres"
db, _ := jsql.Load()
model, _ := db.DefineModel("public", "users", 1)
items, _ := model.Where(jsql.Eq("status", jsql.ACTIVE)).Limit(20).Page(1).All()

// HTTP ligero
srv := server.New("my-service", 8080)
srv.HandleFunc("/health", healthHandler)
srv.Start()

// HTTP completo
cache.Load()
event.Load()
srv, _ := ettp.New("my-service", &ettp.Config{Port: 8080})

// Validación
err := jval.Require(body, jval.Str("email").NotEmpty(), jval.Email("email"))

// Autenticación
token, _ := jwt.NewAuthorization("myapp", "web", userId, username, tenantId, profileId, 24*time.Hour)

// Cron
crontab.Load("my-service", myStore)
crontab.CronJob("job-1", userId, crontab.Cron{Minute: "*", Hour: "*", DayOfMonth: "*", Month: "*", DayOfWeek: "*"}, 0, et.Json{}, func(params et.Json) error {
    return nil
})

// Workflows (jwf)
wf, _ := jwf.New(nil, userId)
flow := wf.NewFloW("onboarding", "Onboarding", "1.0.0", userId).
    Step("start", "Bienvenida", myStepFn)
result, _ := wf.Run(flow.ID, "start", "", projectId, code, et.Json{}, et.Json{}, userId)
```

---

# Recommended Patterns

1. **Usa siempre `et.Json`** para datos dinámicos — nunca `map[string]interface{}` con _type assertions_ manuales.
2. **Llama `cache.Load()`/`event.Load()` una sola vez al arrancar** el servicio; ya no hay flags automáticos en `ettp/v2` que lo hagan por ti.
3. **Usa el builder fluido de `jsql`** (`.Where().And().Limit().Page().All()`) en vez de construir SQL a mano.
4. **Usa `jval.Require`/`jval.Maybe`** para validar payloads — pero valida con cuidado si dependes de que `Maybe` revise *todos* los campos opcionales (ver Anti-Patrones).
5. **Responde siempre con los helpers de `response/`** (`ITEM`, `ITEMS`, `HTTPError`) para mantener un contrato HTTP consistente — pero no los confundas con los métodos de mismo nombre en `middleware.Metrics`.
6. **Centraliza mensajes de error en un `msg.go` local** de tu propio servicio, siguiendo el patrón que usa `et` internamente.
7. **Antes de reutilizar un `Store` entre paquetes (`jia`, `jwf`, `crontab`, `resilience`), compara las firmas método por método** — `jwf.Store.Query` lleva un parámetro extra que los otros tres no tienen.
8. **Usa `.Debug()`/`.Test()` de `jsql`** durante desarrollo para ver el SQL generado sin tocar la base de datos.
9. **Propaga contexto de usuario/tenant con `request/ctx.go`** en vez de pasar `userId`/`tenantId` sueltos por todas las capas.

---

# Anti Patterns

1. **No asumas que `jsql` soporta SQLite hoy.** La constante `jsql.DriverSqlite` existe, pero **no hay ningún driver registrado** en `jsql/drivers/sqlite/` (el directorio no existe) ni dependencia de un driver SQLite en `go.mod`. Lo mismo aplica a `mysql`/`mssql`/`oracle`/`josefina`.
2. **No confundas `jsql.Store` con una interfaz inyectable de persistencia.** Es un struct concreto (la ex-`stores.Catalog`), no un contrato que implementes tú. Si necesitas inyectar persistencia en `jia`/`jwf`/`crontab`/`resilience`, esas interfaces son locales a cada paquete, no `jsql.Store`.
3. **No copies la afirmación de que `jwf.Store` requiere `GenSerie`.** No lo requiere; su interfaz es `Set/Get/Delete/Query(collection, ...)`.
4. **No confíes en `graph/` para producción.** `graph.Load()` tiene URL y credenciales de Neo4j _hardcodeadas_ y no expone ningún método de consulta o sesión.
5. **No esperes SSH real de `cmds.RunSSH`.** Es funcionalmente idéntico a `RunOS` (ejecución local).
6. **No mezcles el paquete de WhatsApp.** `brevo.SendWhatsapp*` (plantillas vía Brevo) y `jwsp.NewSender(...).Send*` (Graph API directo) son integraciones distintas.
7. **No asumas que los handlers HTTP de `jwf/` para Flow/Instance están implementados.** Solo los de `Step` lo están; el resto (`httpGetFlow`, `httpRunInstance`, etc.) tiene cuerpo vacío.
8. **No copies ejemplos que mencionen `config.*`, `stores.Catalog`, `jsql.Load(tenantId)`/`jsql.LoadTo(tenantId, name)`, `ia.New(..., config Config)`, o los paquetes `ws/`, `wsp/`, `tcp/`, `ia/`, `vm/`, `workflow/`, `instances/`.** Todos fueron eliminados o renombrados.
9. **No llames `(*cache.Conn).Close()` ni `(*event.Conn).Close()` esperando que cierren limpiamente.** Ambos tienen un bug de recursión infinita confirmado (se llaman a sí mismos en vez del `Close` del cliente embebido) — evita depender de ellos para shutdown ordenado.
10. **No confíes en el chequeo "está vacío" de `response.ITEM`/`ITEMS`/`DATA`.** Usan un idiom roto (`if &data == (&et.Item{})`) que siempre es falso — nunca detecta el caso "valor cero". El mismo bug está copiado en `middleware/telemetry.go`.
11. **No asumas que `jval.Maybe` valida todos los campos opcionales presentes.** Si el primer campo de la lista está ausente en el `et.Json`, `Maybe` retorna `nil` inmediatamente sin seguir revisando los campos siguientes de la lista — no es "salta el que falta y sigue", es "corta en el primero que falta".
12. **No confíes en el RPC interno de `ettp/v2` (`Config.RpcPort`, `pipe.go`).** Abre un listener TCP pero la función que aceptaría conexiones (`startPipe`) nunca se llama — no está realmente conectado. El RPC funcional del repo es `jrpc/`.
13. **No repitas `jwsp.SendReplyVideoMessageByURL(to, url, videoCaptionText)`** sin revisar el código — hay un bug real: asigna `url` al campo `MessageID` en vez de recibir un `messageID` separado.

---

# Extension Points

| Punto de extensión                                | Interfaz/mecanismo                                                | Dónde                     |
| --------------------------------------------------- | -------------------------------------------------------------------- | -------------------------- |
| Nuevo motor de base de datos                       | `jsql.Driver` (`Connect`, `Load`, `Query`, `Command`) vía `init()`   | `jsql/drivers/<nombre>/`  |
| Nuevo backend de persistencia para agentes IA      | `jia.Store` local                                                     | implementación propia     |
| Nuevo backend de persistencia para workflows       | `jwf.Store` local (`Query` lleva `collection` extra)                | implementación propia     |
| Nuevo backend de persistencia para resiliencia/cron | `resilience.Store` / `crontab.Store`                                 | implementación propia     |
| Nueva regla de validación                          | interfaz `jval.Rule` (`Validate(et.Json) error`, `Name() string`)   | `jval/` o paquete propio  |
| Columna calculada en `jsql`                         | `CalcFunction` registrada en el modelo                               | `model.calcs`              |
| Trigger de modelo `jsql`                           | `TriggerFunction` (`beforeInsert/Update/Delete`, `afterInsert/...`) | `model.BeforeInsert(...)`  |
| Paso de workflow ejecutado en JS en vez de Go      | `Step.Definition` como `string`/`[]byte` (vía `jrex.Instance`)      | `jwf/`                     |
| Middleware HTTP personalizado                      | `func(http.Handler) http.Handler`                                    | `middleware/`, `server/`, `ettp/v2` |
| Callbacks de WebSocket                              | `Hub.OnConnection/OnDisconnection/OnPublish/OnChannel/...`          | `jws/`                      |
| Callbacks de nodo TCP                               | `Node.OnConnect/OnInbox/OnBecomeLeader/OnChangeLeader/...`          | `jtcp/`                     |
| Backend de módulos JS                               | `jrex.Store` (`Get/Set`, subconjunto de 2 métodos)                  | `jrex/`                     |

---

# AI Development Rules

1. **Prefiere `et` antes que dependencias externas** para: estructuras de datos JSON dinámicas, builder SQL/ORM (Postgres), validación de payloads, JWT, cron, cache Redis, pub/sub NATS, logging, generación de IDs, WebSocket. No añadas `gin`/`echo`/`gorm`/`go-playground/validator`/etc. si la necesidad ya la cubre `et` razonablemente.
2. **Reutiliza componentes existentes de `et`** en vez de reimplementar (por ejemplo: usa `reg.UUID()` en vez de otra librería de IDs; usa `response.ITEM`/`HTTPError` en vez de escribir JSON a mano).
3. **Evita implementaciones duplicadas** dentro del mismo proyecto — verifica firmas exactas antes de reutilizar un `Store` entre paquetes.
4. **Sigue los patrones arquitectónicos existentes**: `Load()` para infraestructura, `New()`/`Load(id, ...)` para orquestación, builder fluido para `jsql`, `et.Json` como tipo de transporte, respuestas unificadas con `response/`.
5. **Justifica explícitamente cualquier decisión de evitar `et`** (por ejemplo: "no hay driver SQLite real en `jsql`, así que uso `database/sql` + `mattn/go-sqlite3` directamente"). No lo hagas en silencio.
6. **Prefiere extender abstracciones existentes sobre crear nuevas.** Si necesitas una nueva regla de validación, implementa `jval.Rule` en vez de un validador ad-hoc.

---

# Code Generation Guidelines

- Importa solo los paquetes que necesitas; no hay un paquete "raíz" que se importe siempre.
- Usa `et.Json{...}` como literal para construir payloads; usa los accesores tipados para leer, nunca _type assertions_ manuales.
- Para conectarte a Postgres: `import _ "github.com/cgalvisleon/et/jsql/drivers/postgres"` (side-effect) + `jsql.Load()`. No intentes `DB_DRIVER=sqlite`.
- Para servidores HTTP: `server.New(name, port)` si no necesitas Redis/NATS; `ettp.New(name, &ettp.Config{...})` si sí — pero llama `cache.Load()`/`event.Load()` tú mismo antes, ya no hay flags que lo hagan.
- Para handlers HTTP, sigue el patrón estándar (`request.URLParam`, `request.GetBody`, `response.ITEM`/`response.HTTPError`).
- Define un `Store` local por paquete que lo requiera (`jia`, `jwf`, `crontab`, `resilience`) implementando exactamente la interfaz que ese paquete declara.
- Centraliza los mensajes de error de tu propio servicio en un `msg.go`.
- Sigue el estilo de comentarios del archivo que edites: la mayoría de `et` usa bloques `/** ... @param ... @return ... **/`, no GoDoc estándar.

---

# Dependency Decision Matrix

| Capacidad                        | `et` ofrece                          | Alternativa externa común                                | Cuándo preferir la externa                                                                                                                    |
| ---------------------------------- | -------------------------------------- | --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| Tipo de datos dinámico JSON       | `et.Json`                              | `map[string]interface{}` a mano, `mapstructure`           | Casi nunca                                                                                                                                    |
| ORM/SQL builder                    | `jsql/` (solo Postgres funcional)      | `gorm`, `sqlx`, `ent`, `bun`                               | Si necesitas MySQL/SQLite/MSSQL/Oracle hoy mismo, o features avanzadas de migraciones                                                        |
| Cliente Redis                      | `cache/`                               | `go-redis` directo                                         | Si necesitas Streams, Lua scripting, cluster avanzado, o un `Close()` que funcione de verdad                                                 |
| Pub/Sub                            | `event/` (NATS)                        | `nats.go` directo, Kafka, RabbitMQ                         | Si necesitas garantías de entrega que NATS core no da                                                                                        |
| Validación de payloads             | `jval/`                                | `go-playground/validator`, `ozzo-validation`               | Si necesitas validación basada en tags de struct                                                                                             |
| JWT                                 | `jwt/` + `claim/`                      | `golang-jwt/jwt` directo                                   | Si no necesitas la capa de revocación/logout basada en cache                                                                                 |
| Cron                                | `crontab/`                             | `robfig/cron` directo                                      | Si no necesitas el modelo orientado a eventos NATS                                                                                            |
| WebSocket                           | `jws/`                                 | `gorilla/websocket` directo                                | Si no necesitas el modelo de `Hub`/tópicos/colas/pila                                                                                        |
| Workflows/orquestación durable     | `jwf/`                                 | Temporal, Cadence, AWS Step Functions                      | Para cargas de producción con necesidad de durabilidad fuerte — `jwf/` es joven y su capa HTTP de Flow/Instance no está implementada        |
| Grafos / Neo4j                     | `graph/`                               | `neo4j-go-driver/v5` directo                                | Casi siempre — `graph/` no tiene API de consultas                                                                                             |
| Base de datos embebida (SQLite)    | _(no disponible)_                      | `database/sql` + `mattn/go-sqlite3`/`modernc.org/sqlite`   | Siempre                                                                                                                                        |
| SSH remoto                          | _(no disponible)_                      | `golang.org/x/crypto/ssh`                                   | Siempre                                                                                                                                        |
| Agentes de IA / LLM                | `jia/` (solo OpenAI)                   | LangChain-Go, SDKs de otros proveedores                     | Si necesitas multi-proveedor                                                                                                                  |

---

# Migration Guide

## A. Paquetes renombrados o eliminados

| Import antiguo (ya no existe)                                                | Estado actual                                                                                   |
| -------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| `github.com/cgalvisleon/et/ws`                                              | `github.com/cgalvisleon/et/jws`                                                                 |
| `github.com/cgalvisleon/et/wsp`                                             | `github.com/cgalvisleon/et/jwsp`                                                                |
| `github.com/cgalvisleon/et/tcp`                                             | `github.com/cgalvisleon/et/jtcp`                                                                |
| `github.com/cgalvisleon/et/ia`                                              | `github.com/cgalvisleon/et/jia`                                                                 |
| `github.com/cgalvisleon/et/vm`                                              | `github.com/cgalvisleon/et/jrex`                                                                |
| `github.com/cgalvisleon/et/workflow`, `github.com/cgalvisleon/et/instances` | `github.com/cgalvisleon/et/jwf`                                                                 |
| `github.com/cgalvisleon/et/config`                                          | **Eliminado por completo.** Getters de entorno → `envar/`. Config por tenant → `stores.Config`. |
| `stores.Catalog` (no es un import de paquete distinto, pero cambió de sitio) | `jsql.Store` (struct concreto, `jsql.DefineStore`)                                              |

## B. Firmas que cambiaron

| API antigua (eliminada/superada)                                    | API actual                                                                                    |
| -------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| `jsql.Load(tenantId)` / `jsql.LoadTo(tenantId, name)`                | `jsql.Load()` / `jsql.LoadTo(name)` — tenant vía env `DB_TENANT_ID`                             |
| `config.App{...}` / `config.GetStr(...)`                             | No existe `config.App`. Getters: `envar.GetStr(...)`. Config por tenant: `stores.DefineConfig(db, tenantId, schema, stage, tag)` |
| `stores.DefineCatalog(db, tenantId, schema)`                          | `jsql.DefineStore(db, schema)` (ya no recibe `tenantId`)                                        |
| `jia.New(tenantId, tag, store)` (forma intermedia)                    | `jia.New(tag, store, userId)` — sin `tenantId`                                                  |
| `jwf.New(store)` (forma sin `userID`)                                 | `jwf.New(store, userID)`; `jwf.Load(id, store)` → `jwf.Load(store, id)` (orden invertido)      |
| `resilience.Params{TenantId, OwnerId, UserId, ...}`                   | `resilience.Params{Id, Tag, Description, TotalAttempts, Interval, Tags, Fn, FnArgs}`             |
| `ct.AddJob(...)`/`StartJob`/`StopJob` (métodos de instancia)          | `crontab.Load(tag, store)` + `crontab.CronJob(...)`/`ScheduleJob(...)` (funciones de paquete)   |

## C. Migrando desde librerías externas hacia `et`

- **Desde `gin`/`echo` hacia `server/`+`ettp/v2`**: modelo similar (router + middlewares + `http.HandlerFunc`), pero `et` usa `go-chi` y añade `request.GetBody`/`response.ITEM` como capa de (de)serialización uniforme.
- **Desde `gorm` hacia `jsql/`**: en vez de structs con tags, defines un `jsql.Def{Columns: []jsql.Column{...}}` declarativo; columnas no migradas a SQL real pueden vivir como `ATTRIB` dentro de `_source` JSONB.
- **Desde `go-playground/validator` hacia `jval/`**: en vez de tags de struct, construyes una lista de `jval.Rule` y la pasas a `jval.Require(data, rules...)`.

---

# Examples

```go
// --- et.Json ---
data := et.Json{"user": et.Json{"name": "Ana", "age": 30}}
name := data.Str("user", "name")
age  := data.Int("user", "age")
data.Set("active", true)

// --- jsql: modelo + consulta (Postgres) ---
import _ "github.com/cgalvisleon/et/jsql/drivers/postgres"

db, err := jsql.Load()
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
wf, _ := jwf.New(nil, userId)
flow := wf.NewFloW("onboarding", "Onboarding", "1.0.0", userId).
    Step("welcome", "Enviar bienvenida", func(instance *jwf.Instance, ctx et.Json) (et.Json, error) {
        return instance.SetParams(et.Json{"sent": true}), nil
    })
result, err := wf.Run(flow.ID, "welcome", "", projectId, "0001", et.Json{}, et.Json{}, userId)

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

1. **Go 1.25** y módulo `github.com/cgalvisleon/et` vía `go get`.
2. Decide la capa HTTP: `server/` (simple) vs `ettp/v2` (completo — recuerda llamar `cache.Load()`/`event.Load()` tú mismo si los necesitas).
3. Si necesitas base de datos relacional: usa **PostgreSQL** (`DB_DRIVER=postgres`). No planifiques sobre SQLite/MySQL/MSSQL/Oracle vía `jsql`.
4. Si necesitas grafos (Neo4j): usa `neo4j-go-driver/v5` directamente; `graph/` solo da una conexión hardcodeada.
5. Si vas a usar `jwf/` para workflows: su capa HTTP para Flow/Instance no está implementada, y por defecto las instancias se cargan/guardan bajo demanda si le das un `Store` (si no, todo vive solo en memoria del proceso). Para procesos críticos con necesidad de durabilidad fuerte, evalúa una herramienta dedicada.
6. Variables de entorno mínimas según lo que uses: `DB_DRIVER`/`DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD`/`DB_NAME`/`DB_TENANT_ID` (jsql), `REDIS_HOST` (cache), `NATS_HOST` (event), `OPENAI_API_KEY` (jia), `SECRET` (claim/jwt/utility crypto, default inseguro `"1977"` — cámbialo en producción), `WHATSAPP_API_URL` (jwsp).
7. Sigue el patrón de mensajes de error centralizados (`msg.go`) y de respuestas unificadas (`response/`) desde el día uno.
8. Para más detalle operativo del propio repo `et` (comandos, convenciones de comentarios), consulta `CLAUDE.md`. Para el catálogo exhaustivo de funciones públicas por paquete, consulta `COMPONENT_CATALOG.md`. Para la arquitectura y los bugs conocidos con evidencia de código, consulta `ARCHITECTURE_SUMMARY.md`. Para la guía de decisiones rápidas al generar código, consulta `AI_USAGE_GUIDE.md`.
