# ET: Librería para servicios y herramientas en Go

ET es una librería modular en Go para construir servicios, CLIs y aplicaciones web con componentes reutilizables: servidor HTTP y routing, caché, eventos, jobs, WebSockets, seguridad (JWT), utilidades, y más.

## Estado del proyecto

- Módulo: `github.com/cgalvisleon/et`
- Go: 1.23+
- Licencia: MIT

## Instalación

```bash
go get github.com/cgalvisleon/et@latest
go get github.com/cgalvisleon/et@v1.0.21
```

Para usar paquetes individuales, importa el paquete correspondiente:

```go
import (
    "github.com/cgalvisleon/et/et"
    "github.com/cgalvisleon/et/logs"
)
```

## Estructura del repositorio

```
et/
├── aws/           # Integración con servicios AWS (S3, SES, SMS)
├── brevo/         # Servicios de comunicación (Email, SMS, WhatsApp)
├── cache/         # Caché (Redis) y Pub/Sub
├── claim/         # Claims y permisos (JWT)
├── cmd/           # Comandos CLI y ejecutables
│   ├── apigateway/  # API Gateway y proxy inverso
│   ├── client/      # Cliente de prueba
│   ├── create/      # Generación de proyectos y plantillas
│   ├── daemon/      # Servicio en segundo plano con soporte systemd
│   ├── et/          # Comando principal de la CLI
│   ├── install/     # Instalador
│   ├── jql/         # Cliente JQL
│   ├── server/      # Servidor HTTP de ejemplo
│   └── whatcher/    # Observador de cambios
├── cmds/          # Sistema de comandos y etapas de ejecución
├── color/         # Utilidades de color para terminal
├── create/        # Templates y generadores de código (microservicios, K8s)
├── crontab/       # Tareas programadas (cron y one-shot)
├── dt/            # Data Transfer Objects (DTOs) y validación
├── envar/         # Variables de entorno, argumentos CLI y configuración
├── ephemeral/     # Datos temporales y efímeros
├── et/            # Tipos principales: Json, List, Item, Items
├── ettp/          # Servidor HTTP y routing (v1 y v2, basado en go-chi)
├── event/         # Sistema de eventos pub/sub (NATS)
├── file/          # Manejo de archivos y sincronización
├── graph/         # Soporte GraphQL / Graph
├── iterate/       # Control de iteraciones con tiempo
├── jrpc/          # JSON-RPC entre servicios vía NATS
├── jwt/           # Generación y validación de tokens JWT (usa claim + cache)
├── logs/          # Logs estructurados con niveles y colores
├── mem/           # Memoria compartida y sincronización
├── middleware/    # Middleware HTTP (auth, CORS, logger, request ID)
├── msg/           # Mensajes del sistema y constantes de error
├── race/          # Detección de condiciones de carrera
├── reg/           # Registro de servicios y service discovery
├── request/       # Cliente HTTP unificado
├── resilience/    # Patrones de resiliencia (circuit breaker, etc.)
├── response/      # Respuestas HTTP unificadas
├── router/        # Enrutamiento HTTP con sincronización entre instancias
├── server/        # Servidor HTTP base
├── service/       # Servicios de negocio (OTP, envío de mensajes)
├── sql/           # Constructor de queries SQL (INSERT, UPDATE, DELETE, WHERE)
├── stdrout/       # Salida estándar con formato y colores
├── strs/          # Utilidades de strings
├── tcp/           # Comunicación TCP con balanceo y Raft
├── timezone/      # Zonas horarias
├── units/         # Unidades de medida y conversiones
├── utility/       # Utilidades generales (crypto, hashing, validación)
└── ws/            # WebSocket bidireccional (gorilla/websocket)
```

## Paquetes clave

- **HTTP**: `ettp/`, `router/`, `middleware/`, `server/`, `stdrout/`, `request/`, `response/`.
- **Mensajería/IPC**: `event/` (NATS), `jrpc/`, `tcp/`.
- **Persistencia temporal**: `cache/` (Redis), `mem/`, `ephemeral/`.
- **Planificación**: `crontab/`.
- **Seguridad**: `claim/` (claims JWT), `jwt/` (tokens, usa `claim` + `cache`).
- **Integraciones externas**: `aws/`, `brevo/`.
- **Soporte de dominio**: `service/`, `dt/`, `sql/`.
- **Utilidades**: `utility/`, `strs/`, `color/`, `timezone/`, `units/`, `logs/`.
- **CLI**: `cmd/`, `cmds/`, `envar/` (argumentos CLI y env vars).

## Tipo central: `et.Json`

`et.Json` (`map[string]interface{}`) es el tipo de datos principal usado en toda la librería. Ofrece accesores tipados con valor por defecto y traversal anidado:

```go
data := et.Json{"user": et.Json{"name": "Ana", "age": 30}}

name := data.Str("user", "name")       // "Ana"
age  := data.Int("user", "age")        // 30
data.Set("active", true)
data.Update(et.Json{"role": "admin"})
```

Tipos de respuesta estándar: `et.List` (paginado), `et.Item`, `et.Items`.

## Patrón de inicialización

Los paquetes de infraestructura exponen una función `Load()` que lee variables de entorno y establece la conexión. Las llamadas siguientes son no-op:

```go
cache.Load()  // Requiere REDIS_HOST
event.Load()  // Requiere NATS_HOST
```

`ettp.New(name, config)` llama internamente a `cache.Load()` y `event.Load()`.

## Configuración y entorno

Variables de entorno y argumentos CLI se gestionan desde `envar/`:

```go
host := envar.GetStr("REDIS_HOST", "localhost")
port := envar.GetInt("PORT", 3300)
envar.Validate([]string{"NATS_HOST", "REDIS_HOST"}) // retorna error si falta alguna
```

### Variables de entorno requeridas

| Paquete | Variable        | Descripción                |
| ------- | --------------- | -------------------------- |
| `cache` | `REDIS_HOST`    | Conexión a Redis           |
| `event` | `NATS_HOST`     | Conexión a NATS            |
| `event` | `NATS_USER`     | Usuario NATS (opcional)    |
| `event` | `NATS_PASSWORD` | Contraseña NATS (opcional) |

## Crontab

```go
crontab.Load("mi-servicio")

// Job recurrente (spec con segundos)
crontab.AddJob("job-1", "0 * * * * *", et.Json{"msg": "hola"}, 0, true,
    func(job *crontab.Job) {
        logs.Infof("[cron] %s", job.Params.ToString())
    },
)

// Job basado en evento
crontab.AddEventJob("job-2", "@every 30s", "canal:mi-evento", 0, true,
    et.Json{"foo": "bar"},
    func(msg event.Message) {
        logs.Infof("[event] %s", msg.Data.ToString())
    },
)
```

APIs disponibles: `Load`, `AddJob`, `AddOneShotJob`, `AddEventJob`, `AddOneShotEventJob`, `DeleteJob`, `StartJob`, `StopJob`, `Stop`.

## Integraciones

- **AWS**: S3, SES, SMS en `aws/` (depende de `github.com/aws/aws-sdk-go`).
- **Brevo**: email, SMS y WhatsApp en `brevo/`.

## Cambios recientes relevantes

- **Claim (JWT)**: `projectId` renombrado a `tenantId` en `claim.Claim`, helpers `TenantIdKey` y `TenantId(r)`.
- **Service**: nuevos helpers OTP en `service/otp.go` (`SendOTPEmail`, `SendOTPSms`, `VerifyOTP`); firmas actualizadas para usar `tenantId`.
- **Eventos**: canal global `event.EVENT`; publicación adicional con `{channel, data}` en `event/handler.go`.
- **Cache**: `cache.Set` garantiza expiración mínima de 1s.
- **Crontab**: nuevo orquestador `crontab.Jobs` con integración de eventos vía canales `EVENT_CRONTAB_*`.

## Desarrollo

```bash
# Ejecutar todas las pruebas
go test ./...

# Ejecutar pruebas de un paquete específico
go test ./et/...

# Formatear código
gofmt -w .

# Construir todos los paquetes
go build ./...
```

## VM

```bash
# Ejecutar el VM
gofmt -w . && go run ./cmd/vm
gofmt -w . && go run ./cmd/server
gofmt -w . && go run ./cmd/client -addr localhost:1377
```

## Contribuir

Issues y PRs son bienvenidos. Sigue el estilo del proyecto y añade pruebas cuando aplique.

## Licencia

MIT. Ver `LICENSE`.
