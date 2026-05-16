# ET: Librería para servicios y herramientas en Go

Librería modular en Go para microservicios, CLIs y aplicaciones web. Importa solo los paquetes que necesitas — no hay punto de entrada central.

- Módulo: `github.com/cgalvisleon/et` — Go 1.23+ — MIT

## Instalación

```bash
go get github.com/cgalvisleon/et@latest
go get github.com/cgalvisleon/et@v0.0.1
```

```go
import (
    "github.com/cgalvisleon/et/et"
    "github.com/cgalvisleon/et/cache"
)
```

## Estructura

| Paquete       | Descripción                                                           |
| ------------- | --------------------------------------------------------------------- |
| `et/`         | Tipos centrales: `Json`, `List`, `Item`, `Items`                      |
| `cache/`      | Cliente Redis con Pub/Sub. Init: `cache.Load()`                       |
| `event/`      | Pub/Sub sobre NATS. Init: `event.Load()`                              |
| `ettp/v2/`    | Servidor HTTP (go-chi). Init: `ettp.New(name, config)`                |
| `jrpc/`       | JSON-RPC entre servicios vía NATS                                     |
| `jwt/`        | Generación de tokens JWT (usa `claim` + `cache`)                      |
| `claim/`      | Claims JWT con campo `tenantId`                                       |
| `crontab/`    | Scheduler cron con soporte de segundos. Init: `crontab.New(tag)`      |
| `middleware/` | CORS, auth, logger, request ID, telemetría                            |
| `response/`   | Respuestas HTTP unificadas                                            |
| `request/`    | Cliente HTTP para peticiones salientes                                |
| `router/`     | Routing HTTP con sincronización entre instancias vía NATS             |
| `ws/`         | WebSocket bidireccional (gorilla/websocket)                           |
| `logs/`       | Logging estructurado por niveles con colores                          |
| `strs/`       | Utilidades de strings                                                 |
| `utility/`    | Crypto, hashing, generación de IDs (UUID, ULID, Snowflake)            |
| `envar/`      | Lectura de variables de entorno y argumentos CLI                      |
| `config/`     | Configuración de aplicación (`GetStr`, `GetInt`, `GetBool`, …)        |
| `service/`    | OTP (`SendOTPEmail`, `SendOTPSms`, `VerifyOTP`) y envío de mensajes   |
| `sql/`        | Constructor de queries SQL (INSERT, UPDATE, DELETE, WHERE)            |
| `jql/`        | Lenguaje de consulta sobre `et.Json` (filter, join, order)            |
| `mem/`        | Caché en memoria con expiración y sincronización                      |
| `ephemeral/`  | Datos temporales de corta vida                                        |
| `js/`         | Runtime JavaScript embebido (goja) con hot-reload                     |
| `ia/`         | Agentes IA sobre OpenAI (`openai-go/v3`)                              |
| `workflow/`   | Orquestación de flujos multi-paso con resiliencia y estado            |
| `graph/`      | Conectividad Neo4j (`neo4j-go-driver/v5`)                             |
| `instances/`  | Interfaz `Store` para persistencia de estado (`ia`, `workflow`, `js`) |
| `resilience/` | Circuit breaker y patrones de resiliencia                             |
| `reg/`        | Registro de servicios y generación de IDs                             |
| `aws/`        | S3, SES, SMS vía AWS SDK                                              |
| `brevo/`      | Email, SMS y WhatsApp vía Brevo API                                   |
| `wsp/`        | Cliente WhatsApp Business API                                         |
| `tcp/`        | Nodo TCP con balanceo y Raft                                          |
| `file/`       | Operaciones de archivo y watcher de cambios                           |
| `color/`      | Colores ANSI para terminal                                            |
| `stdrout/`    | Salida estándar formateada con colores                                |
| `timezone/`   | Helpers de zona horaria                                               |
| `units/`      | Conversiones de unidades                                              |
| `race/`       | Detección de condiciones de carrera                                   |
| `cmds/`       | Sistema de comandos y etapas de ejecución                             |
| `iterate/`    | Control de iteraciones con tiempo                                     |
| `create/`     | Templates de código (microservicios, K8s)                             |
| `cmd/`        | Binarios CLI (`et`, `apigateway`, `daemon`, `server`, `vm`, …)        |

## Tipo central: `et.Json`

`et.Json` (`map[string]interface{}`) es el tipo transversal de la librería. Accesores tipados con valor por defecto y traversal anidado:

```go
data := et.Json{"user": et.Json{"name": "Ana", "age": 30}}
name := data.Str("user", "name")   // "Ana"
age  := data.Int("user", "age")    // 30
data.Set("active", true)
```

`et.List` — resultado paginado (`Rows`, `All`, `Count`, `Page`, `Start`, `End`, `Result []Json`).  
`et.Item` / `et.Items` — wrappers de resultado individual y múltiple.  
`items.AddMany([]Json)` — bulk insert con capacidad pre-alocada.

## Patrón de inicialización

`Load()` es idempotente y thread-safe (usa mutex interno). Llama una vez al arrancar:

```go
cache.Load()  // Requiere REDIS_HOST
event.Load()  // Requiere NATS_HOST
```

`ettp.New(name, config)` llama a `cache.Load()` y `event.Load()` internamente.

## Variables de entorno requeridas

| Paquete | Variable                                     | Descripción                            |
| ------- | -------------------------------------------- | -------------------------------------- |
| `cache` | `REDIS_HOST`                                 | Host Redis                             |
| `cache` | `REDIS_PASSWORD`, `REDIS_DB`                 | Auth y base (opcionales)               |
| `event` | `NATS_HOST`                                  | Host NATS                              |
| `event` | `NATS_USER`, `NATS_PASSWORD`                 | Auth NATS (opcionales)                 |
| `claim` | `SECRET`                                     | Clave de firma JWT (default: `"1977"`) |
| `graph` | `NEO4J_HOST`, `NEO4J_USER`, `NEO4J_PASSWORD` | Conexión Neo4j                         |
| `ia`    | `OPENAI_API_KEY`                             | API key de OpenAI                      |
| `wsp`   | `WHATSAPP_API_URL`                           | URL base WhatsApp Graph API (opcional) |

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

## JavaScript embebido (`js/`)

Runtime goja con tres modos: `Develop` (hot-reload), `Production` (desde Store), `Building` (compila + semver).  
Expone `console.*`, `ctx.*`, `fetch()`, `require()`.

```go
v := js.New("mi-vm")
v.RunDev("./cmd/vm")
```

## Agentes IA (`ia/`)

```go
agent := ia.New(name, myStore) // myStore implementa instances.Store
```

Seguimiento de conversación, event handlers y estado por instancia. Requiere `OPENAI_API_KEY`.

## Logging

```go
logs.Info("mensaje")
logs.Errorf("falló: %v", err)
logs.Fatal(err)              // os.Exit(1)
logs.EnableCallerInfo = false // desactiva runtime.Callers en Error() — recomendado en producción
```

## Desarrollo

```bash
gofmt -w .           # formatear
go build ./...       # compilar
go test ./...        # pruebas
./version.sh --minor # bump de versión semántica
```

## Binarios CLI

```bash
go run ./cmd/et                          # CLI principal
go run ./cmd/apigateway                  # API Gateway
go run ./cmd/daemon                      # Servicio systemd
go run ./cmd/server                      # Nodo TCP (:1377)
go run ./cmd/vm                          # VM JS con hot-reload
go run ./cmd/client -addr localhost:1377 # Cliente TCP de prueba
```

## Licencia

MIT. Ver `LICENSE`.

A

