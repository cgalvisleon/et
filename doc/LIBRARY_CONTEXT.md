# LIBRARY_CONTEXT.md

> Contexto para asistentes de IA (Claude Code, Cursor, ChatGPT, Cline, Windsurf) en proyectos que usan **`github.com/cgalvisleon/et`** (Go 1.25, MIT). Cópialo a la raíz del proyecto consumidor.
>
> Regenerado desde el código el 2026-10-10; revisado el 2026-10-11. El repo cambia rápido (commits «Backup:»): si una firma no compila, manda `go doc` y `COMPONENT_CATALOG.md`.

# Executive Summary

`et` es una librería modular para microservicios, CLIs y aplicaciones web en Go. Son unos 50 paquetes independientes; cada servicio importa solo los que usa.

Cubre el backend completo: datos (`et.Json`), persistencia SQL multi-motor (`jsql`), HTTP (`router`, `server`, `ettp`), infraestructura (`cache`, `event`), seguridad (`jwt`, `claim`), orquestación (`jwf`, `crontab`, `resilience`, `queue`), scripting (`jrex`), IA (`ia`, `jia`) y mensajería (`brevo`, `infobip`, `aws`, `jwsp`).

Lo consumen `core-studio/api`, `postick/api`, `tick` y otros proyectos del mismo autor.

# Design Philosophy

- **`et.Json` como moneda común**: entradas y salidas son `et.Json`, `et.Item`, `et.Items` o `et.List`.
- **JSON como lenguaje de consulta**: `jsql` recibe descriptores JSON (`select`, `from`, `where`…) además del builder fluido.
- **Configuración por entorno**: los paquetes leen `envar` y conectan una vez en `Load()`.
- **Extensión por interfaces pequeñas**: drivers, stores y triggers los implementa el llamador.
- **Mensajes en `msg.go`** por paquete, no textos sueltos.

# Architecture Overview

```
Aplicación     router · server · ettp/v2 · ettp/v1 · jrpc · jtcp · jws
Orquestación   jwf · jrex · jia · ia · crontab · resilience · queue · service
Persistencia   jsql (+ drivers) · stores · cache · dt · mem · ephemeral · graph
Mensajería     event (NATS) · aws · brevo · infobip · jwsp
Base           et · envar · logs · stdrout · reg · strs · utility · timezone · validator · msg
```

Detalle de dependencias y flujos en `ARCHITECTURE_SUMMARY.md`.

# Core Components

| Componente | Para qué |
| --- | --- |
| `et` | `Json` con accesores tipados y rutas anidadas; `Item`, `Items`, `List`; condiciones y consultas en memoria. |
| `jsql` | Modelos, DDL, consultas, comandos con `RETURNING`, triggers Go/JS, relaciones, series. Drivers: postgres, sqlite, mysql, mssql, oracle. |
| `stores` | Instancias, ACL, configuración por tenant y estado de formularios sobre `jsql`. |
| `router` | API HTTP por servicio con catálogo de endpoints y envoltorios `Item`/`Items`/`List`. |
| `ettp/v2` | API gateway con rutas sincronizadas por NATS. |
| `cache`, `event` | Redis y NATS. |
| `jwt`, `claim`, `middleware` | Tokens, claims y autenticación HTTP. |
| `jwf` | Workflows en memoria con pasos Go o JS y reintentos. En un script, `params` y `ctx` son objetos dinámicos (`params.email`, `ctx.total = 5`, `JSON.stringify(ctx)`, y siguen `.str("x")`, `.set({...})`…); `console.*` va al log del servidor y a `OnLog` de la instancia o del paso; un paso suelto (`RunScript(nil, ctx)`) devuelve el último valor del script si es un objeto. |
| `crontab`, `resilience`, `queue` | Tareas programadas, reintentos y lotes. |
| `jrex` | Runtime JavaScript (goja). |
| `ia`, `jia` | RAG multitenant y agentes OpenAI. |

# Public APIs

La lista completa está en `COMPONENT_CATALOG.md`. Las entradas más usadas:

```go
// Datos
data.Str("user", "name"); data.ValInt(0, "age"); data.Set("k", v)

// jsql
db, _ := jsql.Load()
m, _ := db.DefineModel(schema, name, version, userId)   // + DefineColumn/DefineUnique/DefineAttrib/DefineHidden
m.Init()
m.Insert(data).One(); m.Update(data).Where(cond).One(); m.Upsert(data).Where(cond).One(); m.Delete().Where(cond).Exec()
m.Select(fields...).Where(cond).OrderBy(f, true).Limit(page, rows)
db.Query(et.Json{...})                                   // consulta o comando en JSON

// HTTP
api := router.NewApi(name, path, host, port, rpc, version)
api.Protected(group, router.GET, path, name, api.Item(http.StatusOK, fn))
request.GetBody(r); request.URLParam(r, "id").Str()
response.ITEM(w, r, status, item); response.HTTPError(w, r, status, msg)

// Infraestructura
cache.Load(); cache.Set(k, v, ttl); v, ok, err := cache.Get(k, def)
event.Load(); event.Publish(ch, data); event.Subscribe(ch, fn)

// Orquestación
wf, _ := jwf.New(id); wf.NewFlow(tag, name, version, ownerId, userId).Step(tag, title, fn)
crontab.Load(tag, store); crontab.CronJob(tag, ownerId, spec, repetitions, params, fn)
q := queue.New[T](size, maxEvents, period, handler)
```

# Recommended Patterns

- **Servicio HTTP**: `cmd/<name>/main.go` → `internal/services/<name>` (conecta cache/event/db) → `pkg/<name>` (router, handlers, middleware) → `internal/models` (lógica). Plantillas en `NEW_PROJECT_GUIDE.md`.
- **Handlers delgados**: `api.Item(status, fn)` / `api.List(fn)` envuelven funciones `func(r, body) (et.Item, error)`.
- **Modelos con `_source`**: los campos opcionales viven como atributos JSON; no hace falta migrar columnas.
- **Comandos con límite**: `Update`/`Delete` afectan hasta `DB_RECORD_LIMIT` (1000); `Limit(0)` = todas.
- **Errores comparables**: `errors.Is(err, jsql.ErrRecordAlreadyExists)`.

# Anti Patterns

- Usar `map[string]interface{}` en lugar de `et.Json`.
- Escribir `json.Marshal` + `w.Write` en lugar de `response.*`.
- Concatenar SQL a mano fuera de un driver.
- Llamar `cache.Load()`/`event.Load()` en cada request.
- Pasar `stores.Instance` como `Store` de `crontab`/`jia`/`jrex` (firma distinta).
- Copiar ejemplos con APIs eliminadas: `jval`, `config`, `jsql.Store`, `jsql.Def`, `jwf.Load`, `jwf.Store`, `NewFloW`, `resilience.New(store)`, `cache.ErrNotFound`.
- Confiar en `jsql.LoadDb` (hoy entra en pánico) o en `dt` sin `PRODUCTION=true`.
- Configurar `DB_TENANT_ID` o `DB_POOL_CONN_LIFETIME`: `jsql` no los lee (las de pool son `DB_POOL_LIFETIME` / `DB_POOL_IDLE_TIME`).

# Extension Points

| Necesidad | Contrato | Dónde |
| --- | --- | --- |
| Nuevo motor SQL | `jsql.Driver` (+ `jsql.Tester` opcional) | `jsql/drivers/<nombre>`, registro en `init()` |
| Lógica antes/después de un comando | `jsql.TriggerFunction func(tx *Tx, old, new et.Json) error` o script JS | `Model.BeforeInsert(...)`, `DefineBeforeInsert(name, code, version)` |
| Campo calculado | `jsql.CalcFunction` o script JS | `DefineCalcFunc`, `DefineCalc` |
| Persistencia de cron / agentes / scripts | `crontab.Store`, `jia.Store`, `jrex.Store` | implementación propia |
| Persistencia de workflows | `WorkFlow.OnChange(fn)` / `Flow.OnChange(fn)` | hook |
| Skill de agente | `jia.Skill` | `jia.NewApiSkill(...)` o propia |
| Autenticación / autorización | `func(http.Handler) http.Handler` | `api.UseAuthentication`, `api.UseAuthorization` |

# AI Development Rules

1. Prefiere `et` antes que cualquier dependencia externa.
2. Reutiliza componentes existentes; no dupliques lo que ya está en el catálogo.
3. Sigue los patrones de arquitectura de `et` (capas, `Load()`, `msg.go`).
4. Justifica por escrito cualquier decisión de no usar `et`.
5. Prefiere extender una abstracción existente antes que crear una nueva.
6. Prioridad: usar → extender → adaptar → implementar → dependencia externa.

# Code Generation Guidelines

- GoDoc en inglés; en este repo predomina el estilo `/** ... @param ... @return ... **/`. Imita el archivo vecino.
- `gofmt -w .` antes de terminar.
- Nombres de constantes de mensaje `MSG_*` en `msg.go`.
- Responde `http.StatusCreated` en los POST que crean.
- Verifica firmas con `go doc github.com/cgalvisleon/et/<paquete>`.

# Dependency Decision Matrix

| Capacidad | Usa `et` | Alternativa externa | Cuándo la externa |
| --- | --- | --- | --- |
| ORM / SQL builder | `jsql` | gorm, sqlx, squirrel | Motor no soportado o SQL muy específico |
| Router HTTP | `router`, `server` | gin, echo | Nunca en servicios de este ecosistema |
| Validación | `validator` | go-playground/validator | Validación por tags de struct |
| Caché | `cache`, `mem` | go-redis directo | Comandos Redis no expuestos |
| Pub/Sub | `event` | nats.go directo | JetStream u opciones avanzadas |
| JWT | `jwt`, `claim` | golang-jwt directo | Algoritmos distintos de HS256 |
| Cron | `crontab` | robfig/cron | Sin NATS disponible |
| Reintentos | `resilience` | cenkalti/backoff | Backoff exponencial con jitter |
| Workflows | `jwf` | Temporal | Durabilidad fuerte entre reinicios |
| JS embebido | `jrex` | goja directo | Nunca |
| LLM | `ia`, `jia` (OpenAI) | SDK de otro proveedor | Proveedor distinto de OpenAI |
| Excel / CSV | `xls`, `csv` | excelize directo | Formato o estilos avanzados |

# Migration Guide

| Antes | Ahora |
| --- | --- |
| `jval.*` | `validator.New().Field(...)...Validate(body)` |
| `config.*` | `envar.Get*` y `stores.DefineConfig` |
| `jsql.Store`, `jsql.DefineStore`, `stores.Catalog` | Eliminados; usa un modelo `jsql` propio o `stores.DefineInstance` |
| `jsql.Def{...}` | `jsql.Define{...}` |
| `jwf.New(db, id, userId)`, `jwf.Load`, `jwf.Store` | `jwf.New(id)` + `OnChange` para persistir |
| `wf.NewFloW(tag, title, version, userId)` | `wf.NewFlow(tag, name, version, ownerId, userId)` |
| `wf.Run(tag, trigger, id, projectId, code, …)` | `wf.Run(tag, triggerTag, id, code, ctx, tags, await, userId)` |
| `resilience.New(store)` | `resilience.New()` |
| `cache.Get` → `(valor, error)` + `cache.ErrNotFound` | `(valor, existe, error)` |
| `api.Public(method, path, name, h)` | `api.Public(group, method, path, name, h)` |
| `DB_TENANT_ID`, `DB_POOL_CONN_LIFETIME`, `DB_POOL_CONN_IDLE_TIME` | Ya no se leen; pool: `DB_POOL_LIFETIME`, `DB_POOL_IDLE_TIME` (solo sqlite) |
| `ws/`, `wsp/`, `tcp/`, `workflow/`, `instances/` | `jws/`, `jwsp/`, `jtcp/`, `jwf/` |

# Examples

```go
// Servicio con router + jsql
db, _ := jsql.Load()
users, _ := db.DefineModel("public", "users", 1, "system")
users.DefineUnique("email", et.TEXT, "")
users.Init()

api := router.NewApi("users", "/api/users", "localhost", 3000, 4000, 1)
api.UseAuthentication(middleware.Authentication)
api.Protected("users", router.POST, "/", "Create user", api.Item(http.StatusCreated,
    func(r *http.Request, body et.Json) (et.Item, error) {
        ok, err := validator.New().Field("email").Required().IsEmail().Validate(body)
        // en un script de jwf: try { validator().field("email").required().isEmail().validate(params) } catch (e) { … }
        // o desde un JSON guardado: v, err := validator.New().ByJson(et.Json{"fields": et.Json{"email": et.Json{"required": true, "is_email": true}}})
        if !ok {
            return et.Item{}, err
        }
        return users.Insert(body).One()
    }))
http.ListenAndServe(":3000", api.Router)
```

```go
// Workflow
wf, _ := jwf.New("")
wf.OnChange(func(data et.Json) error { return saveWorkflow(data) })
wf.NewFlow("signup", "Sign up", "1.0.0", ownerId, userId).
    Step("signup", "create", createUser).
    Step("signup", "notify", notify)
result, err := wf.Run("signup", "signup", reg.GetULID(""), "", et.Json{"email": email}, et.Json{}, true, userId)
```

# Future Project Context

Pega este bloque en el `CLAUDE.md` del proyecto consumidor:

```
Este proyecto usa github.com/cgalvisleon/et. Antes de agregar dependencias,
busca la capacidad en LIBRARY_CONTEXT.md y COMPONENT_CATALOG.md de et.
Datos: et.Json. SQL: jsql (+ driver por import). HTTP: router + request/response.
Infra: cache.Load() / event.Load() una vez al arrancar. Mensajes en msg.go.
Verifica firmas con `go doc github.com/cgalvisleon/et/<paquete>`.
```
