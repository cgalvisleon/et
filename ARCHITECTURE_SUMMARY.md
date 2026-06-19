# ARCHITECTURE_SUMMARY.md

> Resumen de arquitectura de `github.com/cgalvisleon/et` (Go 1.25). Complementa a `LIBRARY_CONTEXT.md` (visión general para IA) y `COMPONENT_CATALOG.md` (referencia exhaustiva de API). Generado a partir de lectura directa del código fuente.

---

## 1. Mapa de capas

```
┌──────────────────────────────────────────────────────────────────────┐
│ Capa 6 — Herramientas de desarrollo                                  │
│   cmd/* (binarios) · create/ (generador) · cmds/ (pipelines) · jcli/ │
├──────────────────────────────────────────────────────────────────────┤
│ Capa 5 — Integraciones externas                                      │
│   aws/ (S3, SNS)  ·  brevo/ (email/SMS/WhatsApp templado)            │
│   wsp/ (WhatsApp Business Graph API)                                 │
├──────────────────────────────────────────────────────────────────────┤
│ Capa 4 — Aplicación / orquestación                                   │
│   jwf/ (workflows)  ·  ia/ (agentes IA)  ·  jrex/ (runtime JS)       │
│   crontab/  ·  resilience/  ·  service/  ·  jrpc/  ·  tcp/           │
├──────────────────────────────────────────────────────────────────────┤
│ Capa 3 — HTTP / routing (sobre go-chi)                               │
│   server/  ·  ettp/v1  ·  ettp/v2  ·  router/                       │
│   middleware/  ·  response/  ·  request/  ·  ws/                    │
├──────────────────────────────────────────────────────────────────────┤
│ Capa 2 — Infraestructura (servicios externos)                        │
│   cache/ → Redis   ·   event/ → NATS   ·   graph/ → Neo4j (stub)    │
│   jsql/ → PostgreSQL (único driver funcional)                        │
├──────────────────────────────────────────────────────────────────────┤
│ Capa 1 — Utilidades autosuficientes (sin red)                        │
│   et · utility · strs · reg · jval · logs · stdrout · color         │
│   config · envar · mem · ephemeral · iterate · race · timezone       │
│   units · file                                                       │
├──────────────────────────────────────────────────────────────────────┤
│ Persistencia de aplicación (cruza capas 2 y 4)                       │
│   stores/ (helpers jsql-backed: instancias, autorización)            │
└──────────────────────────────────────────────────────────────────────┘
```

No existe un punto de entrada único (`et.App`, `et.Run()`, etc.). Cada binario en `cmd/` compone manualmente las capas que necesita. La dirección de dependencia es de arriba hacia abajo: una capa puede importar capas inferiores, nunca al revés (con la excepción de `jwf/` → `jrex/`, que es capa 4 → capa 4, ambas de orquestación).

---

## 2. Grafo de dependencias entre paquetes internos (verificado)

| Paquete | Importa de `et` (capas inferiores/hermanas) | Notas |
|---|---|---|
| `jsql/` | `config`, `et` | Postgres vía `jsql/drivers/postgres` (auto-registro `init()`) |
| `cache/` | `config`, `et`, `msg`, `utility`, `reg` | Redis |
| `event/` | `et`, `msg`, `logs`, `timezone`, `reg` | NATS |
| `ettp/v2` | `cache`, `event`, `config`, `et`, `router` (implícito vía sincronización) | `Config.UseCache`/`UseEvent` deciden si llama `cache.Load()`/`event.Load()` |
| `server/` | `et` (mínimo) | Sin `cache`/`event` — deliberadamente ligero |
| `middleware/` | `jwt`, `request`, `response`, `event` (telemetría) | |
| `response/`, `request/` | `et` | Capa de E/S HTTP compartida por `server`, `ettp`, `jwf`, etc. |
| `jwf/` | `cache`, `config`, `et`, `event`, `reg`, `resilience`, `timezone`, `jrex`, `logs`, `request`, `response` | Motor de workflows; usa `jrex` para pasos definidos en JS |
| `ia/` | `config`, `et`, `event`, `logs`, `msg`, `timezone`, `utility`, `openai-go/v3` | Sin `cache` (a diferencia de versiones previas) |
| `crontab/` | `cache`, `event`, `timezone`, `utility` | |
| `resilience/` | `et` + su propio `Store` local | Usado por `jwf` para reintentos de paso |
| `jrex/` | `file`, `config`, `timezone`, `utility`, `reg`, `et`, `logs`, `goja`, `fsnotify` | VM JS embebida |
| `stores/` | `dt`, `et`, `event`, `jsql`, `msg`, `timezone`, `utility` | Helpers de persistencia sobre `jsql` |
| `dt/` | `cache` (producción) o filesystem (dev, según `PRODUCTION`) | Cache de objetos liviano |
| `service/` | `aws`, `brevo`, `cache` | Orquesta envío de OTP/mensajes |
| `claim/` | `config`, `et`, `msg`, `timezone`, `utility`, `reg`, `golang-jwt/jwt/v4` | |
| `jwt/` | `claim`, `cache`, `et`, `msg` | |
| `tcp/` | `et`, `config`, `file`, `logs`, `msg`, `color` | Sin dependencia de consenso externo — Raft propio |
| `jrpc/` | `et`, `net/rpc` (stdlib) | Sin balanceador ni Raft (a diferencia de lo sugerido en documentación previa) |

**Lectura del grafo**: las únicas dependencias "hacia arriba" llamativas son `jwf/` → `jrex/` y `jwf/` → `resilience/` (capa 4 consumiendo capa 4 — orquestación componiendo orquestación, lo cual es esperado) y `stores/` → `dt/` + `jsql/` (persistencia de aplicación apoyándose en infraestructura, también esperado).

---

## 3. Patrones de diseño con evidencia en código

### 3.1 `Load()` idempotente vs `New()` explícito

- **Singleton perezoso** (`cache.Load()`, `event.Load()`): primera llamada conecta, llamadas posteriores son no-op. Pensado para llamarse una vez por proceso, desde cualquier paquete que lo necesite, sin coordinación.
- **Instancia explícita** (`jwf.New(tenantId, store)`, `ia.New(tenantId, tag, store)`, `crontab.New(tag, store)`, `resilience.New(store)`): cada llamada crea un objeto independiente — útil para multi-tenant en el mismo proceso, pero significa que **no hay un único "workflow global"** como sí lo hay para cache/event.

### 3.2 Store inyectado, interfaces no compartidas

Cinco paquetes definen su propia interfaz `Store` local, con formas parecidas pero no idénticas:

```go
// ia/ia.go
type Store interface {
    Set(id, tag, tenantId, ownerId string, obj any, userId string) error
    Get(id, tag string, dest any) (bool, error)
    Delete(id, tag string) error
    Query(query et.Json) (et.Items, error)
}

// jwf/workflow.go
type Store interface {
    Set(collection, id, tenantId, ownerId string, obj any, userId string) error
    Get(collection, id string, dest any) (bool, error)
    Delete(collection, id string) error
    Query(query et.Json) (et.Items, error)
    GetCode(tag string) (string, error)
}

// crontab/crontab.go
type Store interface {
    Set(id, tag, tenantId, ownerId string, obj any, userId string) error
    Get(id string, dest any) (bool, error)
    Delete(id string) error
    Query(query et.Json) (et.Items, error)
}

// resilience/resilience.go
type Store interface {
    Set(tag, id, tenantId, ownerId string, obj any, userId string) error
    Get(id string, dest any) (bool, error)
    Delete(id string) error
    Query(query et.Json) (et.Items, error)
}
```

`Set` es casi idéntico entre los cuatro (4 strings + `obj any` + `userId string`), pero `Get`/`Delete` varían entre 1 y 2 parámetros string. **Una sola implementación no satisface las cuatro interfaces simultáneamente** sin un adaptador. `stores/` (jsql-backed) implementa una forma con `Get(id string, dest any)` de un solo parámetro — compatible estructuralmente con `crontab.Store`/`resilience.Store`, pero no con `ia.Store`/`jwf.Store`.

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

`jsql.Load(tenantId)` resuelve el driver por nombre (`DB_DRIVER`, leído vía `config.GetStr`) contra el registro interno (`jsql/driver.go: var drivers map[string]Driver`). Si el nombre no está registrado (caso `sqlite` hoy), falla en tiempo de ejecución, no de compilación.

### 3.4 Builder fluido (`jsql.Query`/`jsql.Command`, `jwf.Flow`)

```go
model.Where(jsql.Eq("status", jsql.ACTIVE)).And(jsql.More("age", 18)).Limit(20).Page(1).All()

flow.Step("a", "Paso A", fnA).Step("b", "Paso B", fnB).Error("err", "1.0.0", "Manejo de error", fnErr)
```

Ambos retornan el receptor (`*Query`/`*Flow`) para encadenar, y acumulan estado mutable internamente (condiciones, conexiones) hasta que se ejecuta un método terminal (`.All()`/`.One()`/`.Exec()` en `jsql`; el `Flow` no tiene un "terminal" — se ejecuta después, vía `WorkFlow.Run(flow.ID, ...)`).

### 3.5 Workflow como grafo, no como lista

```
Flow
 ├─ Steps map[string]*Step          (pool de pasos, por ID)
 ├─ Connections []*Connection       (Source.StepId --port--> Target.StepId)
 └─ Triggers []*Trigger             (tag -> Step.ID de inicio)

Instance.next():
  1. Si Current == nil: salta al Step del Trigger
  2. Si no: busca la Connection cuyo Source == Current.ID y avanza a Target
  3. Si el Step falla: intenta resilience.New(...) (si Flow.TotalAttempts > 0); si también falla, sigue el puerto "error"
```

Esto permite ramas de error explícitas (puerto `PortError`) separadas del flujo normal (`PortOutput`), a diferencia de un pipeline lineal con manejo de errores implícito.

### 3.6 Sincronización de router entre réplicas (NATS)

```
Instancia A: router.PushApiGateway(...) --publish--> NATS topic EVENT_SET_ROUTER
                                                          │
Instancia B: event.Subscribe(EVENT_SET_ROUTER, fn) <─────┘
             fn actualiza su copia local de Routes
```

El campo `Myself` en `event.Message` permite que una instancia ignore sus propios mensajes si el publish y el subscribe ocurren en el mismo proceso (evita procesar dos veces el propio cambio).

---

## 4. Ciclo de vida de una petición HTTP típica

```
1. chi.Mux enruta la petición (server/ o ettp/v2)
2. middleware.RequestID      -> inyecta X-Request-Id en el contexto
3. middleware.Logger         -> loguea inicio (método, path)
4. middleware.Authenticate   -> valida Bearer JWT (jwt.Validate), puebla contexto
                                 (request.SetUserId, SetTenantId, SetProfileId, ...)
5. middleware.Recoverer      -> envuelve todo, captura panics -> 500
6. Handler de negocio:
     id   := request.URLParam(r, "id").Str()
     body, err := request.GetBody(r)
     userId := request.UserId(r)   // leído del contexto poblado en (4)
     ...
     response.ITEM(w, r, http.StatusOK, item)   // o response.HTTPError(...)
7. middleware.Telemetry (Metrics) -> registra latencia, status, tamaño de respuesta
```

`jwf/router.go` es una excepción parcial: usa `response.JSON(...)` en vez de `response.ITEM`/`ITEMS`, y varios de sus handlers (flows, instances) aún no tienen cuerpo implementado.

---

## 5. Ciclo de vida de un workflow (`jwf`)

```
1. wf, _ := jwf.New(tenantId, store)        // cache.Load() + event.Load()
2. flow := wf.NewFloW(tag, title, version, userId)
3. flow.Step(tag, title, fn)                // 1er Step -> KindTrigger + Trigger{Tag, StartId}
   flow.Step(tag2, title2, fn2)             // Steps siguientes -> Connection PortOutput
   flow.Error(tagErr, ver, titleErr, fnErr) // Step de error -> Connection PortError
4. result, err := wf.Run(flow.ID, triggerTag, instanceId, projectId, ctx, tags, userId)
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

---

## 6. Componentes incompletos o experimentales (impacto arquitectónico)

| Componente | Estado | Impacto |
|---|---|---|
| `jsql` drivers `sqlite`/`mysql`/`mssql`/`oracle`/`josefina` | Solo constantes, sin implementación (`sqlite` ni siquiera tiene directorio) | La capa de persistencia es, en la práctica, **Postgres-only** |
| `graph/` | Solo `Load()` con credenciales hardcodeadas, sin API de consultas | No usable como capa de persistencia en grafo real todavía |
| `jrpc/` | Sin balanceador de carga ni Raft (solo registro `Solver{Host,Port}`) | RPC simple punto a punto, no un mesh distribuido |
| `tcp/` | Consenso Raft implementado a mano, sin librería externa | Validar cuidadosamente antes de usar en clúster crítico — es código propio, no una implementación de Raft madura y probada externamente |
| `jwf/router.go` | Handlers de Flow/Instance vacíos; solo Step está implementado | La gestión de workflows por HTTP aún no es funcional completa |
| `cmds.RunSSH` | Idéntico a `RunOS` (ejecución local) | No usar para automatización remota real |
| `jcli/` | Declara `package jrex` en directorio `jcli/`, no importado por nadie | Código huérfano/en progreso, no parte de ninguna ruta activa |

Ver `LIBRARY_CONTEXT.md` (sección Anti-Patterns) y `COMPONENT_CATALOG.md` para el detalle de cada caso.
