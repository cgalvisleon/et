# ARCHITECTURE_SUMMARY.md

> Resumen de arquitectura de `github.com/cgalvisleon/et` (Go 1.25). Complementa a `LIBRARY_CONTEXT.md` (visión general para IA) y `COMPONENT_CATALOG.md` (referencia exhaustiva de API). Regenerado a partir de lectura directa del código fuente — corrige deriva significativa de versiones previas (ver nota al final de cada sección relevante).

---

## 1. Mapa de capas

```
┌──────────────────────────────────────────────────────────────────────┐
│ Capa 6 — Herramientas de desarrollo                                  │
│   cmd/* (binarios) · create/ (generador) · cmds/ (pipelines) · jcli/ │
├──────────────────────────────────────────────────────────────────────┤
│ Capa 5 — Integraciones externas                                      │
│   aws/ (S3, SNS/SMS — no SES)  ·  brevo/ (email/SMS/WhatsApp templado)│
│   jwsp/ (WhatsApp Business Graph API)                                │
├──────────────────────────────────────────────────────────────────────┤
│ Capa 4 — Aplicación / orquestación                                   │
│   jwf/ (workflows)  ·  jia/ (agentes IA)  ·  jrex/ (runtime JS)      │
│   crontab/  ·  resilience/  ·  service/  ·  jrpc/  ·  jtcp/          │
├──────────────────────────────────────────────────────────────────────┤
│ Capa 3 — HTTP / routing (sobre go-chi)                               │
│   server/  ·  ettp/v1  ·  ettp/v2  ·  router/                       │
│   middleware/  ·  response/  ·  request/  ·  jws/                   │
├──────────────────────────────────────────────────────────────────────┤
│ Capa 2 — Infraestructura (servicios externos)                        │
│   cache/ → Redis   ·   event/ → NATS   ·   graph/ → Neo4j (stub)    │
│   jsql/ → PostgreSQL (único driver funcional)                        │
├──────────────────────────────────────────────────────────────────────┤
│ Capa 1 — Utilidades autosuficientes (sin red)                        │
│   et · utility · strs · reg · jval · validator · logs · stdrout      │
│   color · envar · mem · ephemeral · iterate · race · timezone        │
│   units · file · queue · msg                                        │
├──────────────────────────────────────────────────────────────────────┤
│ Persistencia de aplicación (cruza capas 2 y 4)                       │
│   stores/ (instancias, autorización, config por tenant — jsql-backed)│
└──────────────────────────────────────────────────────────────────────┘
```

No existe un punto de entrada único (`et.App`, `et.Run()`, etc.). Cada binario en `cmd/` compone manualmente las capas que necesita.

> **Corrección vs. versiones previas:** el paquete `config/` que aparecía en Capa 1 en versiones anteriores de este documento **fue eliminado por completo** — no existe como paquete separado. Sus dos mitades quedaron: los getters de entorno viven ahora directamente en `envar/`, y el registro de configuración por tenant es `stores.Config` (Capa "persistencia de aplicación", jsql-backed). Se agregaron además `queue/`, `validator/` y `msg/` (raíz) a Capa 1, no documentados previamente.

---

## 2. Grafo de dependencias entre paquetes internos (verificado contra código actual)

| Paquete | Importa de `et` (capas inferiores/hermanas) | Notas |
|---|---|---|
| `jsql/` | `et`, `event`, `envar`, `reg`, `timezone`, `utility` | Postgres vía `jsql/drivers/postgres` (auto-registro `init()`). **Ya no importa `config`** (fue eliminado) — todos sus env vars se leen vía `envar` directo. Define su propio struct concreto `Store` (`jsql/store.go`, ex-`stores.Catalog`) para persistir metadata de `DB`/`Model`, opcional vía `(*DB).Save(store)`/`jsql.LoadDb(store, id)` |
| `cache/` | `et`, `msg`, `utility`, `reg` | Redis. **`(*Conn).Close()` tiene un bug de recursión infinita** — se llama a sí mismo en vez de cerrar el `redis.Client` embebido |
| `event/` | `et`, `msg`, `logs`, `timezone`, `reg` | NATS. **Mismo bug de recursión infinita en `Close()`** |
| `ettp/v2` | `event`/`cache` (usados condicionalmente, no cargados automáticamente), `router`, `middleware`, `et` | `Config` **ya no tiene** `UseCache`/`UseEvent` — `New` nunca llama `cache.Load()`/`event.Load()`; el consumidor debe llamarlos antes si los necesita. Abre un listener TCP para RPC (`Config.RpcPort`) que queda sin usar (ver §6) |
| `server/` | `et` (mínimo) | Sin `cache`/`event` — deliberadamente ligero. Expone también un segundo tipo `Api` casi duplicado del de `router/` |
| `middleware/` | `jwt`, `request`, `response`, `event` (telemetría) | El middleware de auth se llama `Authentication`, no `Authenticate` |
| `response/`, `request/` | `et` | Capa de E/S HTTP compartida por `server`, `ettp`, `jwf`, etc. `request` define su propio tipo `Status` para llamadas salientes, distinto de `et.Item` |
| `jwf/` | `cache`, `event`, `et`, `reg`, `jsql` (tipos), `jrex` (pasos JS), `resilience` (reintentos de paso) | `New(store, userID)` llama `cache.Load()` **y** `event.Load()`. Su `Store` local no tiene `GenSerie` (corrección — ver §3.2) |
| `jia/` | `et`, `event`, `envar`, `reg`, `timezone`, `utility`, `openai-go/v3` | `New(tag, store, userId)` solo llama `event.Load()` — **sin** `cache.Load()` |
| `crontab/` | `et`, `event`, `logs`, `timezone`, `robfig/cron` | `New(tag, store)` solo llama `event.Load()` |
| `resilience/` | `cache` (solo el tipo `cache.Metrics`, sin llamar `cache.Load()`), `event`, `et`, `reg`, `timezone` | `New(store)` llama `event.Load()` |
| `jrex/` | `file`, `envar`, `timezone`, `utility`, `reg`, `et`, `logs`, `goja`, `fsnotify` | VM JS embebida; `Store` es un subconjunto de 2 métodos (`Set`/`Get`, sin `Delete`/`Query`) |
| `stores/` | `dt`, `et`, `jsql`, `timezone`, `event` | Helpers de persistencia sobre `jsql` (`Instance`, `Authorization`, `Config`) — **no incluye `Catalog`**, movido a `jsql.Store` |
| `dt/` | `cache` (si `PRODUCTION=true`; si no, no persiste nada — no hay fallback a filesystem) | Cache de objetos liviano |
| `service/` | `aws`, `brevo`, `et` | Orquesta envío de OTP/mensajes |
| `claim/` | `et`, `msg`, `timezone`, `utility`, `reg`, `golang-jwt/jwt/v4` | |
| `jwt/` | `claim`, `cache`, `et`, `msg` | |
| `jtcp/` | `et`, `envar`, `file`, `logs`, `msg`, `color` | Sin dependencia de consenso externo — Raft propio (`jtcp/raft.go`) |
| `jrpc/` | `et`, `net/rpc` (stdlib) | Sin balanceador ni Raft — eso vive en `jtcp/`, no aquí |

**Lectura del grafo**: las únicas dependencias "hacia arriba" llamativas son `jwf/` → `jrex/` y `jwf/` → `resilience/` (orquestación componiendo orquestación, esperado) y `stores/` → `dt/` + `jsql/` (persistencia de aplicación apoyándose en infraestructura, también esperado).

---

## 3. Patrones de diseño con evidencia en código

### 3.1 `Load()` idempotente vs `New()`/`Load(...)` explícito

- **Singleton perezoso** (`cache.Load()`, `event.Load()`): primera llamada conecta, llamadas posteriores son no-op.
- **Instancia explícita** (`jwf.New(store, userID)`, `jia.New(tag, store, userId)`, `crontab.New(tag, store)`, `resilience.New(store)`): cada llamada crea un objeto independiente con su propio `ID` (`reg.UUID()`). Cargar una instancia existente es `jia.Load(id, store)` — pero **`jwf.Load` invierte el orden de parámetros**: `jwf.Load(store, id)`, no `(id, store)`. Verifica el orden exacto por paquete, no lo asumas por simetría con el hermano.

### 3.2 Store inyectado — forma casi unificada, con una excepción real

Tres paquetes comparten **exactamente** la misma forma de `Store` (tipos Go distintos, sin interfaz compartida en el lenguaje, pero firma idéntica):

```go
// jia/ia.go, resilience/resilience.go, crontab/crontab.go — misma forma en los tres:
type Store interface {
    Set(collection, id, ownerId string, obj any) error
    Get(collection, id string, dest any) (bool, error)
    Delete(collection, id string) error
    Query(query et.Json) (et.Items, error)
}
```

`jwf.Store` (`jwf/store.go:12-17`) es casi idéntico pero **su `Query` lleva un parámetro `collection string` adicional al inicio**:

```go
type Store interface {
    Set(collection, id, ownerId string, obj any) error
    Get(collection, id string, dest any) (bool, error)
    Delete(collection, id string) error
    Query(collection string, query et.Json) (et.Items, error)  // <- diferencia real
}
```

**Corrección importante vs. versiones previas de este documento**: `jwf.Store` **no** tiene un método `GenSerie(tag string) (string, error)`. Esa afirmación era incorrecta — verificado directamente contra `jwf/store.go`, donde la interfaz solo tiene los cuatro métodos de arriba. `GenSerie` sí existe en el repo, pero como método de `jsql.Series` (`jsql/series.go`), un tipo completamente distinto y no relacionado con ningún `Store`.

`jrex.Store` es un **subconjunto** intencional de solo 2 métodos (`Set`/`Get`, sin `Delete`/`Query`) — coincide en la forma de esos dos métodos, pero no implementa la forma completa.

**`jsql` no tiene una interfaz `Store` inyectable.** Otra corrección importante: versiones previas de este documento describían un `jsql.Store` interface con la misma forma que `jia`/`jwf`. Eso no existe — lo que sí existe es:

```go
// jsql/store.go — es un STRUCT concreto, no una interfaz:
type Store struct {
    model *Model
}
func DefineStore(db *DB, schema string) (*Store, error) { ... }
```

Este `jsql.Store` es la tabla genérica `kind`+`id` que antes vivía en `stores/` como `stores.Catalog` — fue movida dentro de `jsql`, no es un contrato que el consumidor implemente. Se usa opcionalmente como backend de persistencia para la metadata del propio `*DB`/`*Model`:

```go
func (s *DB) Save(store *Store) error                      // jsql/db.go:158
func LoadDb(store *Store, id string) (*DB, error)           // jsql/db.go:87 (nótese "LoadDb", no "LoadDB")
func (s *Model) Save(store *Store) error                    // jsql/model.go:107
```

Ambos `Save` son no-op si `store == nil` — el flujo normal de conexión (`jsql.Load()`/`LoadTo(name)`) no usa ni requiere esto en absoluto.

**Conclusión práctica**: `jia.Store`, `resilience.Store` y `crontab.Store` son intercambiables entre sí con un solo adaptador (misma forma exacta). `jwf.Store` necesita ese mismo adaptador **más** un parámetro `collection` en `Query`. `jsql.Store` no es intercambiable con ninguno de los anteriores — es un tipo concreto de otro propósito.

### 3.3 Driver auto-registrado (`jsql`)

```go
// jsql/drivers/postgres/driver.go (patrón, no cita literal)
func init() {
    jsql.Register("postgres", &PostgresDriver{})
}
```

El consumidor solo necesita el import por side-effect:

```go
import _ "github.com/cgalvisleon/et/jsql/drivers/postgres"
```

`jsql.Load()` resuelve el driver por nombre (`DB_DRIVER`, leído vía `envar.GetStr` — ya no `config.GetStr`, ese paquete no existe) contra el registro interno. Si el nombre no está registrado (caso `sqlite` hoy), falla en tiempo de ejecución, no de compilación. El tenant activo se lee vía `envar.GetStr("DB_TENANT_ID", "tenant:root")` — `Load`/`LoadTo` ya no reciben ese parámetro.

### 3.4 Builder fluido (`jsql.Query`/`jsql.Command`, `jwf.Flow`)

```go
model.Where(jsql.Eq("status", jsql.ACTIVE)).And(jsql.More("age", 18)).Limit(20).Page(1).All()

flow.Step("a", "Paso A", fnA).Step("b", "Paso B", fnB).Error("err", "1.0.0", "Manejo de error", fnErr)
```

Ambos retornan el receptor para encadenar. `jsql` acumula condiciones hasta un método terminal (`.All()`/`.One()`/`.Exec()`); `Flow` no tiene terminal — se ejecuta después vía `WorkFlow.Run(flow.ID, ...)`.

### 3.5 Workflow como grafo, no como lista

```
Flow
 ├─ Steps map[string]*Step          (pool de pasos, por ID)
 ├─ Connections []*Connection       (StepConnection{StepId, Port, Index} Source --Kind--> Target)
 └─ Triggers []*Trigger             (Tag -> StartId de inicio)

Instance.next():
  1. Si Current == nil: salta al Step del Trigger
  2. Si no: busca la Connection cuyo Source == Current.ID y avanza a Target
  3. Si el Step falla: intenta resilience.New(workflow.store) + LoadInstance(...) si Flow.TotalAttempts != 0;
     si también falla, sigue la Connection de puerto PortError
```

**No hay un mapa `WorkFlow.Instances` en memoria** (corrección vs. versiones previas que sí lo mencionaban) — las instancias se crean bajo demanda (`newInstance`) o se cargan por su ID vía `store.Get("instances", id, ...)`. Un marcador liviano en cache (`instance:<id>:status`, TTL = `Flow.TimeAwait`) solo sirve para detectar si una instancia ya está corriendo, no para guardar su estado completo.

### 3.6 Sincronización de router entre réplicas (NATS)

```
Instancia A: router.PushApiGateway(...) --publish--> NATS topic APIGATEWAY_SET_ROUTER
                                                          │
Instancia B: event.Subscribe(APIGATEWAY_SET_ROUTER, fn) <┘
             fn actualiza su copia local de Routes
```

**Corrección**: las constantes reales publicadas/suscritas son `router.APIGATEWAY_SET_ROUTER` / `APIGATEWAY_REMOVE_ROUTER` / `APIGATEWAY_RESET_ROUTER` (y variantes v0 heredadas `APIGATEWAY_SET_RESOLVE`/`APIGATEWAY_RESET`) — no literalmente `EVENT_SET_ROUTER` como decían versiones previas de este documento. El campo `Myself` en `event.Message` permite que una instancia ignore sus propios mensajes.

---

## 4. Ciclo de vida de una petición HTTP típica

```
1. chi.Mux enruta la petición (server/ o ettp/v2)
2. middleware.RequestID      -> inyecta X-Request-Id en el contexto
3. middleware.Logger         -> loguea inicio (método, path)
4. middleware.Authentication -> valida Bearer JWT (jwt.Validate), puebla contexto
                                 (request.SetUserId, SetTenantId, SetProfileId, ...)
5. middleware.Recoverer      -> envuelve todo, captura panics -> 500
6. Handler de negocio:
     id   := request.URLParam(r, "id").Str()
     body, err := request.GetBody(r)
     userId := request.UserId(r)   // leído del contexto poblado en (4)
     ...
     response.ITEM(w, r, http.StatusOK, item)   // o response.HTTPError(...)
7. middleware.Metrics (telemetría) -> registra latencia, status, tamaño de respuesta
```

Notas:
- El middleware de autenticación se llama **`Authentication`**, no `Authenticate` (`middleware/authentication.go:43`).
- `response.ITEM`/`ITEMS`/`DATA` tienen un chequeo interno de "está vacío" (`if &data == (&et.Item{})`) que compara direcciones de memoria de dos valores distintos — **siempre es falso**, nunca detecta el caso de valor cero. El mismo idiom roto está copiado en `middleware/telemetry.go`. No depende de él para lógica condicional.
- `jwf/router.go` es una excepción parcial: usa `response.JSON(...)` en vez de `response.ITEM`/`ITEMS`, y sus handlers de Flow/Instance aún no tienen cuerpo implementado.

---

## 5. Ciclo de vida de un workflow (`jwf`)

```
1. wf, _ := jwf.New(store, userID)          // cache.Load() + event.Load(); WorkFlow.ID = reg.UUID()
2. flow := wf.NewFloW(tag, title, version, userId)   // sic: "FloW"
3. flow.Step(tag, title, fn)                // 1er Step -> KindTrigger + Trigger{Tag, StartId}
   flow.Step(tag2, title2, fn2)             // Steps siguientes -> Connection PortOutput
   flow.Error(tagErr, ver, titleErr, fnErr) // Step de error -> Connection PortError
4. result, err := wf.Run(flow.ID, triggerTag, instanceId, projectId, code, ctx, tags, userId)
     -> si instanceId == "": crea Instance nueva (status CREATED -> PENDING)
     -> instance.run(ctx, userId):
          for instance.next() {
              step := instance.Current
              result, err = step.run(instance, ctx, userId)   // Go func o jrex si Definition es string/[]byte
              if err != nil { intenta resilience, luego puerto error }
              instance.setResult(result, err, userId)         // status RUNNING/FAILED/DONE, push a event+cache
          }
5. (opcional) (*WorkFlow).LoadRouter(r) expone los Steps vía HTTP — Flow/Instance HTTP aún no implementado
```

En `Flow.Step`/`Flow.Error`, el parámetro que llega como `userId` a `newStep` internamente es en realidad `flow.ID` (el ID del propio flow), no un usuario real — un detalle menor que afecta solo el audit log de esos pasos, no la ejecución.

---

## 6. Componentes incompletos, experimentales, o con bugs confirmados

| Componente | Estado | Impacto |
|---|---|---|
| `jsql` drivers `sqlite`/`mysql`/`mssql`/`oracle`/`josefina` | Solo constantes, sin implementación (`sqlite` ni siquiera tiene directorio) | La capa de persistencia es, en la práctica, **Postgres-only** |
| `graph/` | Solo `Load()` con credenciales hardcodeadas (`neo4j://localhost:7687`, `neo4j`/`password`), sin ningún método de consulta/sesión | No usable como capa de persistencia en grafo real todavía |
| `jrpc/` | Sin balanceador de carga ni Raft (solo registro `Solver{Host,Port}`) | RPC simple punto a punto, no un mesh distribuido |
| `jtcp/` | Consenso Raft implementado a mano, sin librería externa | Validar cuidadosamente antes de usar en clúster crítico |
| `jwf/router.go` | Handlers de Flow/Instance vacíos; solo Step está implementado | La gestión de workflows por HTTP aún no es funcional completa |
| `ettp/v2` RPC (`Config.RpcPort`, `pipe.go`) | Abre el listener TCP en `New()`, pero `startPipe()` (la función que haría `Accept()` sobre ese listener) **nunca se llama** en ningún punto del código | El RPC de `ettp/v2` está iniciado pero no conectado; el RPC real y funcional del repo es `jrpc/` |
| `cache.(*Conn).Close()` / `event.(*Conn).Close()` | Ambos se llaman a sí mismos en vez de cerrar el cliente Redis/NATS embebido | Recursión infinita — no usar para shutdown ordenado |
| `response.ITEM`/`ITEMS`/`DATA` (y `middleware/telemetry.go`, mismo bug copiado) | El chequeo "está vacío" (`&data == &Type{}`) nunca es verdadero | No depender de esa rama para detectar valores vacíos |
| `jval.Maybe` | Corta la validación de la lista completa en el primer campo ausente, en vez de saltarlo y seguir con los siguientes | Validaciones opcionales posteriores en la lista pueden no ejecutarse nunca si un campo anterior falta |
| `cmds.RunSSH` | Idéntico a `RunOS` (ejecución local) | No usar para automatización remota real |
| `jcli/` | Declara `package jrex` en directorio `jcli/`, no importado por nadie | Código huérfano/en progreso |
| `jwsp.SendReplyVideoMessageByURL` | Asigna el parámetro `url` al campo `MessageID` del mensaje en vez de un `messageID` separado | Bug funcional confirmado en ese método específico |
| `dt/` en modo desarrollo (`PRODUCTION=false`) | No hay fallback a filesystem — simplemente no persiste nada (el objeto vive solo en memoria del proceso) | No asumir "modo dev = guarda en archivo", como sugería documentación previa |

Ver `LIBRARY_CONTEXT.md` (sección Anti-Patterns) y `COMPONENT_CATALOG.md` para el detalle de cada caso.
