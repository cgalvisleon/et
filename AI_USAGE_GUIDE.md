# AI_USAGE_GUIDE.md

> Guía operativa para asistentes de IA (Claude Code, Cursor, ChatGPT, Cline, Windsurf, etc.) que generan código en un proyecto que depende de `github.com/cgalvisleon/et`. Es el documento "de bolsillo": decisiones rápidas y checklists. Para contexto narrativo completo ver `LIBRARY_CONTEXT.md`, para arquitectura ver `ARCHITECTURE_SUMMARY.md`, para la API exacta de cada función ver `COMPONENT_CATALOG.md`.

---

## 1. Procedimiento antes de escribir código

Cuando necesites implementar una capacidad (HTTP, persistencia, validación, mensajería, etc.), sigue este orden — **es la Matriz de Prioridad obligatoria**:

```
┌─────────────────────────────────────────────────────────────────┐
│ Prioridad 1 — Usa un componente existente de et tal cual        │
│   ¿Ya hay una función/tipo en et que hace exactamente esto?     │
│   -> Revisa COMPONENT_CATALOG.md antes de escribir nada nuevo.  │
├─────────────────────────────────────────────────────────────────┤
│ Prioridad 2 — Extiende un componente existente de et            │
│   ¿Hay una interfaz de extensión (Driver, Store, Rule,          │
│   CalcFunction, TriggerFunction, middleware) que cubre el caso  │
│   con una implementación nueva pero siguiendo el contrato?      │
├─────────────────────────────────────────────────────────────────┤
│ Prioridad 3 — Crea un adaptador alrededor de et                 │
│   ¿La forma exacta no calza (p. ej. un Store con firma          │
│   distinta), pero la funcionalidad de base sí existe?           │
│   -> Escribe un adaptador delgado, no una reimplementación.     │
├─────────────────────────────────────────────────────────────────┤
│ Prioridad 4 — Implementa una solución propia, sin et            │
│   ¿et genuinamente no cubre esto (ver Tabla 3 y "Huecos          │
│   conocidos")? Implementa con la librería estándar de Go         │
│   cuando sea razonable, siguiendo los patrones de et             │
│   (et.Json como transporte, Load() idempotente, etc.)           │
├─────────────────────────────────────────────────────────────────┤
│ Prioridad 5 — Introduce una dependencia externa                 │
│   Última opción. Requiere justificación explícita (ver §4).     │
└─────────────────────────────────────────────────────────────────┘
```

**Regla dura:** nunca saltes directo a la Prioridad 5 sin pasar (mentalmente) por 1-4. Si terminas en 4 o 5, dilo explícitamente en tu respuesta al usuario ("no usé `et` para X porque..."), no en silencio.

---

## 2. Tabla rápida "necesito X → usa Y"

| Necesito...                         | Usa de `et`      | Función/tipo de entrada                                        |
| ----------------------------------- | ---------------- | -------------------------------------------------------------- |
| Representar datos JSON dinámicos    | `et`             | `et.Json{...}`, `.Str()/.Int()/.ValStr(def, ...)`              |
| Resultado paginado de una lista     | `et`             | `et.List{Rows, All, Count, Page, Result}`                      |
| Conectar a Postgres                 | `jsql`           | `jsql.Load(tenantId)` + `import _ ".../jsql/drivers/postgres"` |
| Definir una tabla/modelo            | `jsql`           | `db.Define(jsql.Def{...})` o `db.DefineModel(...)`             |
| Consultar con filtros               | `jsql`           | `model.Where(jsql.Eq(...)).And(...).Limit().Page().All()`      |
| Servidor HTTP simple                | `server`         | `server.New(name, port)`                                       |
| Servidor HTTP con cache/eventos     | `ettp/v2`        | `ettp.New(name, &ettp.Config{UseCache: true, UseEvent: true})` |
| Middleware de auth/CORS/logging     | `middleware`     | `Authenticate`, `AllowAll`, `Logger`, `Recoverer`              |
| Responder JSON desde un handler     | `response`       | `response.ITEM/ITEMS/HTTPError(w, r, status, ...)`             |
| Leer body/params de un request      | `request`        | `request.GetBody(r)`, `request.URLParam(r, "id").Str()`        |
| Validar un payload                  | `jval`           | `jval.Require(data, jval.Str("x").NotEmpty(), ...)`            |
| Cache / Redis                       | `cache`          | `cache.Load()`, `cache.Set/Get/SetObject/GetObject`            |
| Pub/Sub entre servicios             | `event`          | `event.Load()`, `event.Publish/Subscribe/Queue`                |
| Emitir/validar JWT                  | `jwt` + `claim`  | `jwt.NewAuthorization(...)`, `jwt.Validate(token)`             |
| Generar IDs (ULID/UUID/XID)         | `reg`            | `reg.UUID()`, `reg.GetULID(id)`                                |
| Cron jobs                           | `crontab`        | `crontab.New(tag, store)` + `AddJob/AddEventJob`               |
| Workflows multi-paso                | `jwf`            | `jwf.New(store)` + `flow.Step(...)` + `wf.Run(...)` (ya no recibe `tenantId`) |
| Reintentos con backoff              | `resilience`     | `resilience.New(store).LoadInstance(Params{...}).Run(userId)`  |
| Agente sobre OpenAI                 | `jia`            | `jia.New(tag, store, userId)` (ya no recibe `tenantId`)        |
| Ejecutar JS embebido                | `jrex`           | `jrex.Load(tag, store)` (entry point real; `jrex.NewInstance()` es de bajo nivel) |
| WebSocket                           | `jws`            | `jws.New()` (Hub) + `.Connect/.Publish/.SendTo`                |
| Subir archivos a S3                 | `aws`            | `aws.UploaderFile/UploaderB64`                                 |
| Enviar WhatsApp (Graph API directo) | `jwsp`           | `jwsp.NewSender(token, phoneNumberId).SendTextMessage(...)`    |
| Enviar WhatsApp/Email/SMS templado  | `brevo`          | `brevo.SendWhatsapp*/SendEmail*/SendSms*`                      |
| Logging estructurado                | `logs`           | `logs.Info/Error/Fatal/Panic`                                  |
| Variables de entorno                | `envar`/`config` | `config.GetStr(key, def)`, `envar.Validate([...])`             |
| Medir tiempo entre pasos            | `iterate`        | `iterate.Start/Segment/End(tag, ...)`                          |
| Cache en memoria con TTL            | `mem`            | `mem.Set/Get/GetInt/.../More`                                  |
| Valor compartido thread-safe        | `race`           | `race.NewValue(v)`                                             |
| Zona horaria / formateo de fechas   | `timezone`       | `timezone.Now()/Format(t, layout)`                             |

---

## 3. Huecos conocidos — aquí SÍ se justifica salir de `et` (Prioridad 4/5)

| Necesidad                                                      | Por qué `et` no alcanza hoy                                                           | Qué usar en su lugar                                                                                |
| -------------------------------------------------------------- | ------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| Base de datos SQLite/MySQL/MSSQL/Oracle                        | `jsql` solo tiene driver real para Postgres; `sqlite` ni tiene directorio             | `database/sql` + driver nativo del motor, o espera a que se implemente el driver en `jsql/drivers/` |
| Consultas/sesiones Neo4j                                       | `graph.Load()` solo abre una conexión hardcodeada a `localhost`, sin API de consultas | `github.com/neo4j/neo4j-go-driver/v5` directamente                                                  |
| Ejecución remota vía SSH                                       | `cmds.RunSSH` es idéntico a `RunOS` (local)                                           | `golang.org/x/crypto/ssh`                                                                           |
| Balanceo de carga / consenso RPC robusto                       | `jrpc` no tiene balancer ni Raft real                                                 | Diseña tu propio mecanismo o usa una librería RPC madura (gRPC, etc.) si el caso lo amerita         |
| Orquestación de workflows con durabilidad fuerte a gran escala | `jwf` es joven, en memoria por defecto, capa HTTP de Flow/Instance sin implementar    | Temporal, Cadence, AWS Step Functions, según el contexto                                            |
| Multi-proveedor de LLM o agentes complejos                     | `jia` solo soporta OpenAI                                                             | SDK del proveedor específico, o una librería de orquestación de agentes                             |

Antes de añadir cualquier dependencia externa para cubrir estos casos, confírmalo releyendo el código real (estos huecos pueden cerrarse en futuras versiones del repo — no asumas que seguirán igual para siempre).

---

## 4. Plantilla de justificación al salir de `et`

Cuando decidas usar Prioridad 4 o 5, exprésalo así (adapta al caso):

> "No usé `et` para \_\_\_ porque \_\_\_ (cita el hueco concreto, p. ej. 'jsql no tiene driver SQLite real'). En su lugar usé \_\_\_ (librería/patrón), siguiendo de todas formas el patrón de `et` de \_\_\_ (p. ej. 'envolver el resultado en et.Json para mantener consistencia con el resto del servicio')."

No lo hagas en silencio — el usuario necesita saber cuándo y por qué se desvió la solución de la librería base del proyecto.

---

## 5. Banderas rojas — verifica antes de copiar un patrón

Estas son trampas reales detectadas en el código, no hipotéticas. Si tu solución toca alguno de estos puntos, detente y revisa `COMPONENT_CATALOG.md`/`LIBRARY_CONTEXT.md` antes de continuar:

- ⚠️ **`DB_DRIVER=sqlite`/`mysql`/`mssql`/`oracle`/`josefina`** → no resolverá ningún driver. Solo `postgres` funciona.
- ⚠️ **Reutilizar `stores.Instance` o `stores.Catalog` como `Store` de `jia`/`jwf`** → incompatible (`Get` con 1 string o que devuelve solo `error`, vs. los 2 strings + `(bool, error)` que exigen `jia.Store`/`jwf.Store`). Verifica firma exacta antes de inyectar.
- ⚠️ **`graph.Load()` en producción** → credenciales y host hardcodeados (`localhost`, `neo4j`/`password`), sin API de consultas.
- ⚠️ **`cmds.RunSSH`** → no ejecuta SSH real, ejecuta localmente igual que `RunOS`.
- ⚠️ **`jwsp.SendReplyVideoMessageByURL`** → bug conocido, asigna `url` a `MessageID`.
- ⚠️ **Handlers HTTP de `jwf` para Flow/Instance** (`httpGetFlow`, `httpRunInstance`, etc.) → cuerpo vacío, no implementados. Solo los handlers de `Step` funcionan.
- ⚠️ **Los paquetes `ws/`, `wsp/`, `tcp/`, `ia/`, `vm/` ya no existen** → fueron renombrados a `jws/`, `jwsp/`, `jtcp/`, `jia/`, `jrex/`. Cualquier ejemplo o ruta de import con el nombre viejo no compilará.
- ⚠️ **`jia.New(tenantId, tag, store)` / `jwf.New(tenantId, store)`** → forma intermedia ya obsoleta. Hoy ninguno de los dos recibe `tenantId`: `jia.New(tag, store, userId)` / `jwf.New(store)`, ambos generan su propio `ID` con `reg.UUID()`.
- ⚠️ **Ejemplos/documentación que mencionen `workflow.RunInstance`, `instances.Store`, `ia.New(..., config Config)`, `jsql.Load(config)`, `config.App`, `crontab.New(tag)` (sin store), `wsp.NewWhatsapp`, `resilience.Params{TenantId/OwnerId/UserId, ...}`** → APIs **eliminadas o renombradas**. Ver tabla de migración en `LIBRARY_CONTEXT.md` (sección Migration Guide).
- ⚠️ **El repo cambia rápido** (commits "Backup:" sin mensaje) → si una firma citada aquí no compila, confía en el código actual, no en este documento; actualízalo si detectas el drift.

---

## 6. Reglas duras (resumen ejecutable)

1. Toda estructura de datos dinámica en código nuevo es `et.Json`, no `map[string]interface{}`.
2. Toda respuesta HTTP de un handler de negocio pasa por `response.ITEM`/`ITEMS`/`HTTPError` (o `response.JSON` si seguís el patrón de `jwf`), nunca `json.Marshal` + `w.Write` manual.
3. Toda consulta a Postgres usa el builder fluido de `jsql`, no SQL crudo concatenado a mano (salvo dentro de un nuevo `Driver`).
4. Todo `Store` inyectado en `jia`/`jwf`/`crontab`/`resilience`/`config` implementa la interfaz exacta de **ese** paquete — cópiala de `COMPONENT_CATALOG.md`, no la adivines por similitud con otra.
5. Toda inicialización de infraestructura (`cache`, `event`) ocurre una vez al arrancar el proceso, no por request.
6. Todo mensaje de error de tu propio servicio vive en un `msg.go` local, igual que en `et`.
7. Todo uso de `jsql` en desarrollo puede (y debería) pasar por `.Debug()`/`.Test()` antes de tocar una base de datos real.
