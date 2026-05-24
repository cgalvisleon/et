# ET: Librería de servicios y herramientas en Go

> [English version](README.md)

Librería modular en Go para microservicios, CLIs y aplicaciones web. Importa solo los paquetes que necesitas — no hay punto de entrada central.

- Módulo: `github.com/cgalvisleon/et` — Go 1.23+ — MIT

## Instalación

```bash
go get github.com/cgalvisleon/et@latest
go get github.com/cgalvisleon/et@v1.0.24
```

```go
import (
    "github.com/cgalvisleon/et/et"
    "github.com/cgalvisleon/et/cache"
)
```

## Paquetes

| Paquete       | Descripción                                                                         |
| ------------- | ----------------------------------------------------------------------------------- |
| `et/`         | Tipos centrales: `Json`, `List`, `Item`, `Items`                                    |
| `cache/`      | Cliente Redis con Pub/Sub. Init: `cache.Load()`                                     |
| `event/`      | Pub/Sub sobre NATS. Init: `event.Load()`                                            |
| `ettp/v2/`    | Servidor HTTP completo (go-chi + Redis + NATS). Init: `ettp.New(name, config)`      |
| `server/`     | Servidor HTTP ligero (solo chi, sin dependencias externas). Init: `server.New()`    |
| `router/`     | Router HTTP independiente con sincronización entre instancias vía NATS              |
| `jrpc/`       | `net/rpc` de Go sobre TCP con balanceo de carga y consenso Raft                     |
| `jwt/`        | Creación de tokens JWT (`New`, `NewAuthentication`, `NewAuthorization`)             |
| `claim/`      | Claims JWT con campo `tenantId`. Firma con HS256                                    |
| `crontab/`    | Scheduler cron con soporte de segundos. Init: `crontab.New(tag)`                   |
| `middleware/` | CORS, auth, logger, request ID, telemetría, recuperación de pánico                  |
| `response/`   | Respuestas HTTP unificadas: `ITEM`, `ITEMS`, `HTTPError`                            |
| `request/`    | Helpers de entrada (`URLParam`, `GetBody`) y cliente HTTP de salida                 |
| `ws/`         | WebSocket bidireccional (gorilla/websocket)                                         |
| `jsql/`       | Constructor SQL agnóstico a la base de datos y ORM ligero                           |
| `jval/`       | Validación fluida de JSON: validadores tipados con restricciones encadenables       |
| `logs/`       | Logging estructurado por niveles con salida colorizada                              |
| `strs/`       | Utilidades de strings                                                               |
| `utility/`    | Crypto, hashing, generación de IDs (UUID, ULID, Snowflake)                         |
| `envar/`      | Acceso a variables de entorno y argumentos CLI                                      |
| `config/`     | Configuración de aplicación (`GetStr`, `GetInt`, `GetBool`, `GetFloat`, `GetTime`) |
| `service/`    | Helpers OTP (`SendOTPEmail`, `SendOTPSms`, `VerifyOTP`) y mensajería               |
| `mem/`        | Caché en memoria con expiración y primitivas de sincronización                      |
| `ephemeral/`  | Estructuras de datos temporales de corta vida                                       |
| `vm/`         | Runtime JavaScript embebido (goja) con hot-reload                                  |
| `ia/`         | Integración con agentes OpenAI (`openai-go/v3`) con seguimiento de conversación     |
| `workflow/`   | Orquestación de flujos multi-paso con estado de instancia y resiliencia             |
| `graph/`      | Conectividad Neo4j (`neo4j-go-driver/v5`)                                           |
| `instances/`  | Interfaz `Store` para persistencia de estado usada por `ia`, `workflow`, `vm`       |
| `resilience/` | Circuit breaker y patrones de resiliencia                                           |
| `reg/`        | Registro de servicios y helpers de generación de IDs (ULID, etc.)                  |
| `aws/`        | Wrapper del SDK de AWS: S3, SES (email), SMS                                        |
| `brevo/`      | Cliente de la API de Brevo: email, SMS, WhatsApp                                    |
| `wsp/`        | Cliente de la API de WhatsApp Business                                              |
| `tcp/`        | Nodo TCP distribuido con elección de líder estilo Raft                              |
| `file/`       | Operaciones de archivo y watcher de cambios en el sistema de archivos               |
| `color/`      | Colores ANSI para terminal                                                           |
| `stdrout/`    | Salida estándar colorizada de bajo nivel usada por `logs`                           |
| `timezone/`   | Helpers de zona horaria                                                              |
| `units/`      | Utilidades de conversión de unidades                                                |
| `race/`       | Helpers de detección de condiciones de carrera                                      |
| `cmds/`       | Sistema de comandos y etapas de ejecución                                           |
| `iterate/`    | Control de iteraciones con soporte de tiempo                                        |
| `create/`     | Templates de código para microservicios y despliegues en Kubernetes                 |
| `cmd/`        | Binarios CLI: `et`, `apigateway`, `daemon`, `server`, `vm`, `jsql`, …              |

## Tipo central: `et.Json`

`et.Json` (`map[string]interface{}`) es el tipo universal de la librería. Accesores tipados con patrón de valor por defecto y traversal de claves anidadas:

```go
data := et.Json{"user": et.Json{"name": "Ana", "age": 30}}
name := data.Str("user", "name")       // "Ana"
age  := data.Int("user", "age")        // 30
ok   := data.Bool("active")            // false (valor cero)
data.Set("active", true)
```

`et.List` — resultado paginado (`Rows`, `All`, `Count`, `Page`, `Start`, `End`, `Result []Json`).  
`et.Item` / `et.Items` — wrappers de resultado individual y múltiple.

## Patrón de inicialización

`Load()` es idempotente y thread-safe. Llama una vez al arrancar:

```go
cache.Load()  // requiere REDIS_HOST
event.Load()  // requiere NATS_HOST
```

`ettp.New(name, config)` llama a `cache.Load()` y `event.Load()` internamente.

## Constructor SQL: `jsql/`

Constructor SQL agnóstico a la base de datos y ORM ligero. Soporta PostgreSQL y SQLite.

```go
import _ "github.com/cgalvisleon/et/jsql/drivers/postgres"

db, _ := jsql.Load() // lee DB_DRIVER, DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME

// Definir un modelo (agrega id, created_at, updated_at, _source JSONB, _idx)
model, _ := db.DefineModel("public", "users", 1)
model.DefineAttrib("name", jsql.TEXT, "")
model.Init()

// Consultas
items, _ := model.Where(jsql.Eq("status", "active")).Limit(20).Page(1).All()
item,  _ := model.Where(jsql.Eq("id", id)).One()

// Comandos
_, _ = model.Insert(et.Json{"email": "a@b.com"}).ExecTx(nil)
_, _ = model.Update(et.Json{"status": "archivado"}).Where(jsql.Eq("id", id)).ExecTx(nil)
_, _ = model.Upsert(et.Json{"id": id, "email": "a@b.com"}).ExecTx(nil)
```

Tanto `Query` como `Command` soportan `.Debug()` (registra el SQL, omite la ejecución) y `.Test()` (devuelve el SQL sin ejecutarlo).

## Orquestación de flujos: `workflow/`

Motor de flujos multi-paso con estado de instancia, rollback y resiliencia.

**Jerarquía de tipos:**

```
Flow  (definición)
  ├── Steper  (camino/carril con nombre, identificado por tag)
  │     └── Steps []int  (índices en Flow.Steps)
  └── Steps  []*Step  (pool compartido de pasos: Definition, Undo, Stop)

Instance  (ejecución en tiempo real de un Flow para un ID de entidad específico)
```

```go
workflow.Load(myStore) // myStore implementa instances.Store

// Definir un flujo
flow, _ := workflow.NewFlow("onboarding", "v1", "Onboarding", "Flujo de incorporación", "admin")
flow.Resilence(3, 5*time.Second, "equipo-ops", "high")

steper, _ := flow.NewSteper("principal", "Camino Principal", "Pasos principales de incorporación")
step, _   := flow.NewStep(workflow.Def{
    Name:       "enviar-bienvenida",
    Definition: `/* JS o definición */`,
    Undo:       `/* lógica de rollback */`,
    Stop:       false,
})

// Ejecutar una instancia
result, err := workflow.RunInstance(entityID, "onboarding", 0, ctx, tags, username)

// Operaciones del ciclo de vida
workflow.ResetInstance(id, username)    // reinicia al paso 0
workflow.RollbackInstance(id, username) // ejecuta la cadena de undo
workflow.StopInstance(id, username)     // detiene la ejecución
```

Handlers HTTP expuestos en `*WorkFlow`: `HttpGetFlow`, `HttpNewFlow`, `HttpDeleteFlow`, `HttpNewStep`, `HttpSetStep`, `HttpDeleteStep`, `HttpNewSteper`, `HttpSetSteper`, `HttpDeleteSteper`, `HttpAddStepFromSteper`, `HttpRemoveStepFromSteper`, `HttpMoveStepFromSteper`, `HttpGetInstance`, `HttpRunInstance`, `HttpResetInstance`, `HttpRollbackInstance`, `HttpStopInstance`.

## Crontab

```go
ct := crontab.New("mi-servicio")

ct.AddJob("job-1", "0 * * * * *", et.Json{"msg": "hola"}, 0, true,
    func(job *crontab.Job) { logs.Infof("[cron] %s", job.Params.ToString()) },
)

ct.AddEventJob("job-2", "@every 30s", "mi-canal", 0, true,
    et.Json{"foo": "bar"},
    func(msg event.Message) { logs.Infof("[event] %s", msg.Data.ToString()) },
)
```

API: `New`, `AddJob`, `AddOneShotJob`, `AddEventJob`, `AddOneShotEventJob`, `DeleteJob`, `StartJob`, `StopJob`, `Stop`.

## Runtime JavaScript (`vm/`)

Runtime goja embebido con tres modos: `Develop` (hot-reload desde archivos), `Production` (carga desde Store), `Building` (compila + bump de semver). Expone `console.*`, `ctx.*`, `fetch()`, `require()`.

```go
v := vm.New("mi-vm")
v.RunDev("./cmd/vm")
```

## Agentes IA (`ia/`)

```go
agent := ia.New(name, myStore) // myStore implementa instances.Store
```

Gestiona seguimiento de conversación, event handlers y estado por instancia. Requiere `OPENAI_API_KEY`.

## Logging

```go
logs.Info("mensaje")
logs.Errorf("falló: %v", err)
logs.Fatal(err)                  // llama a os.Exit(1)
logs.EnableCallerInfo = false    // desactiva runtime.Callers en Error() — recomendado en producción
```

## Variables de entorno requeridas

| Paquete | Variable                                                                     | Descripción                                  |
| ------- | ---------------------------------------------------------------------------- | -------------------------------------------- |
| `cache` | `REDIS_HOST`                                                                 | Host de Redis                                |
| `cache` | `REDIS_PASSWORD`, `REDIS_DB`                                                 | Auth y base de datos de Redis (opcionales)   |
| `event` | `NATS_HOST`                                                                  | Host de NATS                                 |
| `event` | `NATS_USER`, `NATS_PASSWORD`                                                 | Auth de NATS (opcionales)                    |
| `claim` | `SECRET`                                                                     | Clave de firma JWT (por defecto: `"1977"`)   |
| `jsql`  | `DB_DRIVER`                                                                  | Nombre del driver: `postgres` o `sqlite`     |
| `jsql`  | `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`                    | Conexión a la base de datos                  |
| `jsql`  | `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_CONN_LIFETIME`, `DB_POOL_CONN_IDLE_TIME` | Pool de conexiones (opcionales) |
| `graph` | `NEO4J_HOST`, `NEO4J_USER`, `NEO4J_PASSWORD`                                 | Conexión a Neo4j                             |
| `ia`    | `OPENAI_API_KEY`                                                             | Clave de API de OpenAI                       |
| `wsp`   | `WHATSAPP_API_URL`                                                           | URL base de la API de WhatsApp Graph (opcional)|

## Binarios CLI

```bash
go run ./cmd/et                          # CLI principal (cobra)
go run ./cmd/apigateway                  # API Gateway / proxy
go run ./cmd/daemon                      # servicio en segundo plano con integración systemd
go run ./cmd/server                      # servidor de nodo TCP (puerto por defecto 1377, usa -port)
go run ./cmd/vm                          # VM JS con hot-reload
go run ./cmd/jsql                        # demo y prueba del driver jsql
go run ./cmd/client -addr localhost:1377 # cliente TCP de prueba
```

## Desarrollo

```bash
gofmt -w .           # formatear código
go build ./...       # compilar todos los paquetes
go test ./...        # ejecutar pruebas (aún no hay archivos *_test.go — solo compila)
./version.sh --minor # bump de versión semántica (lee tags de git, actualiza README, hace push del tag)
```

## Licencia

MIT. Ver `LICENSE`.
