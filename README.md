# ET: Librería para servicios y herramientas en Go

ET es una librería modular en Go para construir servicios, CLIs y aplicaciones web con componentes reutilizables: HTTP server y routing, caché, eventos, jobs, workflows, seguridad (JWT claims), utilidades, y más.

## Estado del proyecto

- Módulo: `github.com/cgalvisleon/et`
- Go: 1.23+
- Licencia: MIT

## Instalación

```bash
go get github.com/cgalvisleon/et@latest
```

Para usar paquetes individuales, importa el módulo y el paquete correspondiente, por ejemplo:

```go
import (
    "github.com/cgalvisleon/et/et"
)
```

## Estructura del repositorio

Estructura actual de `et/` con una breve descripción por paquete:

````
et/
├── arg/           # Gestión de argumentos de línea de comandos
├── aws/           # Integración con servicios AWS (S3, SES, SMS)
├── brevo/         # Servicios de comunicación (Email, SMS, WhatsApp)
├── cache/         # Caché (Redis) y Pub/Sub
├── claim/         # Claims y permisos (JWT)
├── cmd/           # Comandos CLI y ejecutables
│   ├── apigateway/  # API Gateway y proxy
│   ├── context/     # Gestión de contexto de aplicación
│   ├── create/      # Generación de proyectos y plantillas
│   ├── daemon/      # Servicios en segundo plano y systemd
│   ├── et/          # Comando principal
│   ├── prepare/     # Preparador de proyectos existentes
│   └── ws/          # Servidor WebSocket dedicado
├── cmds/          # Sistema de comandos y etapas de ejecución
├── config/        # Configuración y parámetros de aplicación
├── create/        # Templates y generadores de código
├── crontab/       # Tareas programadas con persistencia
├── dt/            # Data Transfer Objects (DTOs) y validación
├── envar/         # Variables de entorno / configuración
├── ephemeral/     # Datos temporales y efímeros
├── et/            # Utilidades principales y tipos JSON
├── ettp/          # Servidor HTTP y routing
├── event/         # Sistema de eventos
├── file/          # Manejo de archivos y sincronización
├── graph/         # Soporte Graph/GraphQL
├── jrpc/          # JSON-RPC entre servicios
├── logs/          # Logs estructurados
├── mem/           # Memoria compartida y sincronización
├── middleware/    # Middleware HTTP (auth, CORS, logger, etc.)
├── msg/           # Mensajes del sistema y localización
├── race/          # Detección de condiciones de carrera
├── reg/           # Registro de servicios y service discovery
├── request/       # Request HTTP unificado
├── resilience/    # Patrones de resiliencia (circuit breakers, etc.)
├── response/      # Response HTTP unificado
├── router/        # Enrutamiento HTTP
├── server/        # Servidor HTTP base
├── service/       # Servicios y utilidades de negocio
├── stdrout/       # Rutas estándar y endpoints comunes
├── strs/          # Utilidades de strings
├── timezone/      # Zonas horarias
├── units/         # Unidades de medida y conversiones
├── utility/       # Utilidades generales (crypto, validación, etc.)
├── vm/            # Máquinas virtuales / helpers de ejecución
├── workflow/      # Flujos de Trabajo y procesos
└── ws/            # WebSocket y comunicación bidireccional

## Paquetes clave (visión general)
- **HTTP**: `ettp/`, `router/`, `middleware/`, `server/`, `stdrout/`, `request/`, `response/`.
- **Mensajería/IPC**: `event/`, `jrpc/`.
- **Persistencia temporal**: `cache/`, `mem/`, `ephemeral/`.
- **Planificación**: `crontab/`.
- **Workflows**: `workflow/`.
- **Seguridad**: `claim/` (JWT).
- **Integraciones**: `aws/`, `brevo/`.
- **Soporte de dominio**: `service/`, `dt/`.
- **Utilities**: `utility/`, `strs/`, `timezone/`, `units/`, `logs/`.
- **CLI**: `cmd/`, `cmds/`, `arg/`.

## CLI y comandos
- Los ejecutables y entradas de CLI se encuentran en `cmd/`.
- El sistema de orquestación de comandos y etapas está en `cmds/`.
- Revisa cada subdirectorio de `cmd/` para ver el propósito de cada comando (por ejemplo, `cmd/et/`).

## Configuración y entorno
- Variables y carga de entorno: `envar/`, archivo `.env` y `config/`.
- Recomendado: documentar las variables requeridas por cada paquete que las consuma.

### API de configuración (nueva)
Funciones principales en `config/`:
- **Lectura**: `GetStr`, `GetInt`, `GetInt64`, `GetBool`, `GetFloat`, `GetTime`.
- **Escritura**: `Set(key, val)` y `SetEnvar(values et.Json)`.
- **Parámetros CLI**: `ParamStr`, `ParamInt`, `ParamInt64`, `ParamBool`, `ParamFloat`, `ParamTime`.
- **App metadata**: `config.App` mantiene `name`, `version`, `company`, `host`, `port`, `stage`, etc. Puedes serializar con `App.ToJson()`.

Ejemplo rápido:
```go
name := config.GetStr("NAME", "et")
port := config.GetInt("PORT", 3300)
config.Set("DEBUG", true)
argsPort := config.ParamInt("port", port)
_ = config.App.ToJson() // et.Json con la configuración activa
````

## Integraciones

- **AWS**: utilidades para S3, SES, SMS en `aws/` (depende de `github.com/aws/aws-sdk-go`).
- **Brevo**: email/SMS/WhatsApp en `brevo/`.

## Cambios recientes relevantes

- **Claim (JWT)**: se renombra `projectId` a `tenantId` en `claim.Claim` y helpers (`TenantIdKey`, `TenantId(r)`), y en logs/contexto.
- **Service (mensajería)**:
  - Nuevos mensajes constantes en `service/msg.go` (`MSG_SEND_SMS`, `MSG_SEND_WHATSAPP`, `MSG_SEND_EMAIL`, `MSG_OTP`).
  - Nuevos helpers OTP en `service/otp.go` (`SendOTPEmail`, `SendOTPSms`, `VerifyOTP`), con plantillas en `service/templates.go`.
  - Firmas actualizadas para usar `tenantId` en `service/send.go` y `service/event.go`.
- **Eventos**: se añade el canal global `event.EVENT` y publicación adicional del evento con `{channel, data}` en `event/handler.go`.
- **Cache**: `cache.Set` asegura expiración mínima de 1s cuando se pasa un valor inferior.
- **Config**: unificación de getters `Get*` y CLI `Param*`; `App.Set(key, val)` actualiza env y `config.App`.

## Desarrollo

- Requisitos: Go 1.23+.
- Dependencias: ver `go.mod` y `go.sum`.
- Pruebas: usa `go test ./...` en la raíz del módulo.
- Versionado: consulta `version.sh` si aplica a pipelines/builds.

## Ejemplos y guías

- Este README describe el mapa de paquetes y cambios recientes. Para ejemplos de uso, consulta READMEs locales por paquete o `create/` para plantillas.

```bash
gofmt -w . && go run ./cmd/flow
```

## Contribuir

- Issues y PRs son bienvenidos. Sigue el estilo del proyecto y añade pruebas cuando aplique.

## Licencia

MIT. Ver `LICENSE`.
