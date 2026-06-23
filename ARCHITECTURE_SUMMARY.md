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
│   et · utility · strs · reg · jval · logs · stdrout · color         │
│   config · envar · mem · ephemeral · iterate · race · timezone       │
│   units · file                                                       │
├──────────────────────────────────────────────────────────────────────┤
│ Persistencia de aplicación (cruza capas 2 y 4)                       │
│   stores/ (helpers jsql-backed: instancias, autorización, catálogo)  │
└──────────────────────────────────────────────────────────────────────┘
```

No existe un punto de entrada único (`et.App`, `et.Run()`, etc.). Cada binario en `cmd/` compone manualmente las capas que necesita. La dirección de dependencia es de arriba hacia abajo: una capa puede importar capas inferiores, nunca al revés (con la excepción de `jwf/` → `jrex/` y `jwf/` → `resilience/`, que son capa 4 → capa 4, orquestación componiendo orquestación).

---

## 2. Grafo de dependencias entre paquetes internos (verificado contra código actual)

| Paquete | Importa de `et` (capas inferiores/hermanas) | Notas |
|---|---|---|
| `jsql/` | `config`, `et`, `event`, `reg`, `timezone`, `utility` | Postgres vía `jsql/drivers/postgres` (auto-registro `init()`). Define su **propio** `Store` interno (ver §3.2) para persistir metadata de `DB`/`Model`, opcional vía `jsql.NewDB(params, store, userId)`/`LoadDB(id, store)` — distinto del flujo normal de conexión (`jsql.Load`/`LoadTo`) |
| `cache/` | `config`, `et`, `msg`, `utility`, `reg` | Redis |
| `event/` | `et`, `msg`, `logs`, `timezone`, `reg` | NATS |
| `ettp/v2` | `cache`, `event`, `config`, `et`, `router` | `Config.UseCache`/`UseEvent` deciden si llama `cache.Load()`/`event.Load()`; `Config.RpcPort` ya no tiene fallback interno |
| `server/` | `et` (mínimo) | Sin `cache`/`event` — deliberadamente ligero |
| `middleware/` | `jwt`, `request`, `response`, `event` (telemetría) | |
| `response/`, `request/` | `et` | Capa de E/S HTTP compartida por `server`, `ettp`, `jwf`, etc. |
| `jwf/` | `cache`, `event`, `et`, `reg`, `envar`, `timezone`, `jrex` (vía pasos JS), `resilience` (reintentos de paso) | `New(store)` llama `cache.Load()` **y** `event.Load()` |
| `jia/` | `et`, `event`, `envar`, `reg`, `timezone`, `utility`, `openai-go/v3` | `New(tag, store, userId)` solo llama `event.Load()` — **sin** `cache.Load()` |
| `crontab/` | `et`, `event`, `logs`, `timezone`, `robfig/cron` | `New(tag, store)` solo llama `event.Load()` — **sin** `cache.Load()` |
| `resilience/` | `cache` (tipo `cache.Metrics`, sin llamar `cache.Load()`), `event`, `et`, `envar`, `reg`, `timezone` | `New(store)` llama `event.Load()` |
| `jrex/` | `file`, `config`, `timezone`, `utility`, `reg`, `et`, `logs`, `goja`, `fsnotify` | VM JS embebida; `Store` es un subconjunto de 2 métodos (`Set`/`Get`, sin `Delete`/`Query`) |
| `stores/` | `dt`, `et`, `jsql`, `timezone` | Helpers de persistencia sobre `jsql` (`Instance`, `Authorization`, `Catalog`) |
| `dt/` | `cache` (producción) o filesystem (dev, según `PRODUCTION`) | Cache de objetos liviano |
| `service/` | `aws`, `brevo`, `et` | Orquesta envío de OTP/mensajes |
| `claim/` | `et`, `msg`, `timezone`, `utility`, `reg`, `golang-jwt/jwt/v4` | |
| `jwt/` | `claim`, `cache`, `et`, `msg` | |
| `jtcp/` (antes `tcp/`) | `et`, `config`, `file`, `logs`, `msg`, `color` | Sin dependencia de consenso externo — Raft propio (`jtcp/raft.go`) |
| `jrpc/` | `et`, `net/rpc` (stdlib) | Sin balanceador ni Raft — eso vive en `jtcp/`, no aquí |

**Lectura del grafo**: las únicas dependencias "hacia arriba" llamativas son `jwf/` → `jrex/` y `jwf/` → `resilience/` (capa 4 consumiendo capa 4 — orquestación componiendo orquestación, esperado) y `stores/` → `dt/` + `jsql/` (persistencia de aplicación apoyándose en infraestructura, también esperado).

---

## 3. Patrones de diseño con evidencia en código

### 3.1 `Load()` idempotente vs `New()` explícito

- **Singleton perezoso** (`cache.Load()`, `event.Load()`): primera llamada conecta, llamadas posteriores son no-op. Pensado para llamarse una vez por proceso, desde cualquier paquete que lo necesite, sin coordinación.
- **Instancia explícita** (`jwf.New(store)`, `jia.New(tag, store, userId)`, `crontab.New(tag, store)`, `resilience.New(store)`): cada llamada crea un objeto independiente con su propio `ID` (`reg.UUID()`) — ninguno de los cuatro recibe ya `tenantId`. No hay un único "workflow global" como sí lo hay para cache/event; cargar una instancia existente es `jia.Load(id, store)` / `jwf.Load(id, store)` (por **su propio ID**, no por tenant).

### 3.2 Store inyectado — convergencia real hacia una forma única

A diferencia de lo que sugería documentación anterior ("interfaces parecidas pero no idénticas"), la lectura directa del código muestra que **cinco paquetes hoy comparten exactamente la misma forma** de `Store` (siguen siendo tipos Go distintos y no intercambiables formalmente — cada paquete declara su propio `type Store interface`, no hay un `import` compartido — pero la firma es idéntica carácter por carácter):

```go
// jia/ia.go, jwf/workflow.go (+GenSerie), resilience/resilience.go,
// crontab/crontab.go, jsql/db.go — misma forma en los cinco:
type Store interface {
    Set(collection, id, ownerId string, obj any) error
    Get(collection, id string, dest any) (bool, error)
    Delete(collection, id string) error
    Query(query et.Json) (et.Items, error)
}
```

`jwf.Store` es la única variante que añade un método extra: `GenSerie(tag string) (string, error)`. `jrex.Store` es un **subconjunto** intencional de solo 2 métodos (`Set`/`Get`, sin `Delete`/`Query`) — no toda la forma, pero el `Set`/`Get` que tiene coincide exactamente.

`config.Store` es la verdadera excepción — usa una forma distinta orientada a `(tag, stage)` en vez de `(collection, id)`, y no tiene `Query`:

```go
// config/config.go:18
type Store interface {
    Set(tag, stage, tenantId, ownerId string, obj any) error
    Get(tag, stage string, dest any) (bool, error)
    Delete(tag, stage string) error
}
```

**Bug real encontrado en `config/config.go`**: la interfaz declara el orden `Set(tag, stage, tenantId, ownerId string, obj any)`, pero `(*Config).Save()` (línea 136) invoca `s.store.Set(s.Tag, s.Stage, s.OwnerId, s.TenantId, s)` — **`tenantId` y `ownerId` están intercambiados** en el sitio de la llamada respecto al orden declarado en la interfaz. Como ambos son `string`, el compilador no lo detecta; cualquier `Store` real que distinga ambos campos internamente los guardará cruzados.

`stores/` (jsql-backed) tiene dos implementaciones con grados de compatibilidad distintos con esta forma unificada:
- `stores.Catalog`: `Set(collection, id, ownerId string, obj any) error` ✅ coincide. `Delete(collection, id string) error` ✅ coincide. `Query(query et.Json) (et.Items, error)` ✅ coincide. **Solo `Get(collection, id string, dest any) error`** rompe la compatibilidad — devuelve únicamente `error`, no `(bool, error)`. Es, con diferencia, la implementación más cercana a calzar sin adaptador.
- `stores.Instance`: `Set(tag, id, ownerId string, obj any) error` (mismo shape, solo cambia el nombre del primer parámetro) ✅, pero `Get(id string, dest any) (bool, error)` y `Delete(id string) error` solo reciben **una** clave string, no dos. No calza con la forma unificada de 2 claves.

**Conclusión práctica**: ya no hace falta tratar "comparar firma por firma" como una tarea abierta entre `jia`/`jwf`/`resilience`/`crontab`/`jsql` — son intercambiables en la práctica si se define un único adaptador con esa forma. La cautela real sigue siendo necesaria solo contra `config.Store` (forma distinta + bug de orden) y contra `jrex.Store`/`stores.Instance`/`stores.Catalog` (subconjuntos o casi-calces, no calces exactos).

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

Separado de esto, `jsql.NewDB(params et.Json, store Store, userId string) (*DB, error)` / `jsql.LoadDB(id string, store Store) (*DB, error)` permiten persistir la metadata del propio `*DB`/`*Model` (registro de bases y modelos) en un `Store` inyectado — opcional; si no se pasa, `(*DB).save()`/`(*Model).save()` son no-op (`if s.store == nil { return nil }`).

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
1. wf, _ := jwf.New(store)                  // cache.Load() + event.Load(); WorkFlow.ID = reg.UUID()
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
| `jtcp/` (antes `tcp/`) | Consenso Raft implementado a mano, sin librería externa | Validar cuidadosamente antes de usar en clúster crítico — es código propio, no una implementación de Raft madura y probada externamente |
| `jwf/router.go` | Handlers de Flow/Instance vacíos; solo Step está implementado | La gestión de workflows por HTTP aún no es funcional completa |
| `config/config.go` | Bug de orden de parámetros en `Save()` (`tenantId`/`ownerId` intercambiados al llamar `store.Set`) | Cualquier `Store` real conectado a `config.Config` guardará esos dos campos cruzados — ver §3.2 |
| `cmds.RunSSH` | Idéntico a `RunOS` (ejecución local) | No usar para automatización remota real |
| `jcli/` | Declara `package jrex` en directorio `jcli/`, no importado por nadie | Código huérfano/en progreso, no parte de ninguna ruta activa |

Ver `LIBRARY_CONTEXT.md` (sección Anti-Patterns) y `COMPONENT_CATALOG.md` para el detalle de cada caso.
