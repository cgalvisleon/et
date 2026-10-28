# ARCHITECTURE_SUMMARY.md

> Arquitectura de `github.com/cgalvisleon/et`, verificada contra el código el 2026-10-27. La API exacta está en `COMPONENT_CATALOG.md` (generado desde `go doc`); las reglas para asistentes de IA, en `AI_USAGE_GUIDE.md`.

## 1. Capas

`et` es un conjunto de paquetes independientes. Ninguno es punto de entrada; cada servicio importa solo lo que usa.

```
Aplicación     router · server · ettp/v2 · ettp/v1 · jrpc · tcp · websocket
Orquestación   jwf · jia · ia · crontab · resilience · queue · service
Persistencia   jsql (+ drivers) · store · stores · cache · dt · mem · ephemeral · graph
Mensajería     event (NATS) · aws · brevo · infobip · jwsp
Base           et · envar · logs · stdrout · reg · strs · utility · timezone · validator · msg
```

## 2. Dependencias internas

Calculadas con `go list -f '{{.Imports}}'`. La flecha indica "importa".

| Paquete | Importa de `et` |
| --- | --- |
| `jsql` | `et`, `envar`, `jwf`, `logs`, `reg`, `strs`, `timezone`, `utility` |
| `jwf` | `cache`, `event`, `resilience`, `jrpc`, `request`, `et`, `envar`, `reg`, `timezone` |
| `ia` | `jsql`, `csv`, `request`, `response`, `et` |
| `stores` | `jsql`, `dt`, `event`, `et` |
| `router` | `cache`, `event`, `middleware`, `request`, `response`, `et` |
| `ettp/v2` | `ettp/v1`, `router`, `middleware`, `jwt`, `jrpc`, `cache`, `event`, `et` |
| `middleware` | `jwt`, `cache`, `event`, `request`, `response`, `et` |
| `jwt` | `claim`, `cache`, `et` |
| `dt` | `cache`, `et` |
| `cache`, `event` | `envar`, `et`, `logs`, `response` |

Consecuencia: importar `jsql` arrastra `jwf` (triggers y campos calculados en JavaScript), y con ellos `cache` y `event` como dependencias de compilación, aunque no se conecten.

## 3. Patrones

### 3.1 `Load()` idempotente

`cache.Load()`, `event.Load()` y `jsql.Load()` leen el entorno con `envar`, conectan una vez y las llamadas siguientes no hacen nada. `ettp/v2.New` **no** los llama: el servicio debe hacerlo al arrancar.

Los constructores que sí los llaman: `jwf.New` (`cache` + `event`), `jia.New`, `crontab.Load` y `resilience.New` (solo `event`).

### 3.2 Store inyectado

Cada paquete de orquestación define su propia interfaz `Store` y el llamador la implementa:

| Paquete | Métodos |
| --- | --- |
| `crontab`, `jia` | `Set(collection, id, ownerId, obj)`, `Get(collection, id, dest)`, `Delete(collection, id)`, `Query(query)` |
| `envar` | `Get(name, def) (any, bool)`, `Set(name, value)` |

`jwf` ya no tiene `Store`: los flujos viven en memoria y se persisten desde fuera con `WorkFlow.OnChange` / `Flow.OnChange`. `stores.Instance.Get(id, dest)` tiene una sola clave, así que no cumple ninguna de estas interfaces sin un adaptador.

### 3.3 Drivers auto-registrados (`jsql`)

Cada driver vive en `jsql/drivers/<nombre>` y se registra en `init()` con `jsql.Register`. Se importan por efecto lateral. La interfaz `jsql.Driver` es `UseSchema`, `CreateDB`, `DropDB`, `Connect`, `ExistModel`, `Load` (DDL), `Drop`, `Query` y `Command`; los dos últimos solo devuelven SQL. La interfaz opcional `jsql.Tester` prueba una conexión sin crear la base.

Un cambio en la generación de SQL casi siempre se repite en los cinco drivers (`postgres`, `sqlite`, `mysql`, `mssql`, `oracle`); `josefina` hereda el de postgres.

### 3.3.1 Condiciones (`et`)

Una sola evaluación (`evaluate` en `et/evaluate.go`) sirve a `EvaluateObject`, `EvaluateValue` y `Wheres`, con la precedencia de SQL. Los drivers traducen las mismas condiciones, y `TestMemoryMatchesSql` (`jsql/drivers/test`) comprueba que memoria y SQL dan las mismas filas. Reglas en `et/spec.md`.

### 3.4 Builders fluidos

`Model.Select(...).Where(...).And(...).OrderBy(...).Limit(page, rows)`, `Model.Update(data).Where(...).Limit(n).One()` y `Flow.Step(...).Step(...)`. `Query`, `Command` y `Model` aceptan también un descriptor JSON (`Model.Query(json)`, `DB.Query(json)`).

### 3.5 Router con catálogo de endpoints

`router.NewApi(name, path, host, port, rpc, version)` monta un `chi.Mux`. `Public`, `Session` y `Protected` reciben `(group, method, path, name, handler)`, anteponen el `Path` del Api y publican la ruta en el catálogo (`RegisterEndpoint`). `Api.Item`, `Items` y `List` envuelven funciones de negocio en handlers HTTP.

### 3.6 Gateway sincronizado (`ettp/v2`)

Las rutas se guardan en Redis y se propagan entre réplicas por NATS (`EVENT_SET_ROUTER`, `REMOVE`, `RESET`). `Save` serializa los solvers bajo un mutex.

## 4. Flujo de una petición HTTP

1. `router.Api` recibe la petición; `middleware.Authentication` valida el token con `jwt.Validate` (token guardado en `cache`).
2. El handler lee con `request.GetBody` / `request.URLParam`.
3. El modelo de negocio usa `jsql` y devuelve `et.Item`, `et.Items` o `et.List`.
4. `response.ITEM` / `ITEMS` / `LIST` / `HTTPError` escriben la respuesta.

## 5. Flujo de un workflow (`jwf`)

1. `wf, _ := jwf.New(id)`: conecta `cache` y `event`; `id` vacío genera un UUID.
2. `wf.NewFlow(tag, name, version, ownerId, userId)`: el primer `Step` es el trigger y los siguientes se encadenan por conexiones de salida. `Flow.Error(...)` agrega un paso al puerto de error.
3. `wf.Run(tag, triggerTag, id, code, ctx, tags, await, userId)`: busca o crea la instancia (`GetInstance`) y la ejecuta (`RunInstance`).
4. Si un paso falla y el flujo tiene intentos, reintenta con `resilience`; después sigue la conexión de error.
5. Un paso puede ser Go (`fnStep`) o JavaScript (`RunCode` con goja). En JavaScript, `params` y `ctx` son objetos dinámicos de goja (`jwf.Binder` + `NewDynamicObject`, `jsonObject`) y `console.*` (`consoleOf`) escribe en el log y emite cada línea a `Instance.OnLog` / `Step.OnLog`.

## 6. Problemas conocidos (verificados 2026-10-27)

| Componente | Problema |
| --- | --- |
| `jia/ia.go:90` | `Load` llama `store.Get(id, packageName, …)` con los argumentos invertidos. |
| `validator` | `Min`/`Max` solo se aplican si son > 0 y `Pattern` se compila en cada validación (2026-10-11). |
| `jwsp` `SendReplyVideoMessageByURL` | Asigna `url` a `MessageID`. |
| `cmds.Step.RunSSH` | Ejecuta el comando localmente, igual que `RunOS`. |
| `utility.More` | Reinicia el contador del tag a 0 en cada llamada. |
| `dt` con `PRODUCTION=false` | El objeto no se persiste: vive solo en memoria. |
| `validator.Pattern` | Una regex inválida entra en pánico al validar (`regexp.MustCompile`); `ByJson` la rechaza con error. |
| `et` en memoria | `neg`, `not_in` e `is_not` con un valor nil dan `true` (en SQL, `NULL <> x` no es verdadero). |

Resuelto el 2026-10-27: el contador del límite de peticiones de `router` podía quedar sin TTL y dejar una ruta en 429 para siempre; `cache.IncrCtx` repone el vencimiento en un script atómico y la clave lleva el prefijo `ratelimit:`.
