# ET - Biblioteca Go

[![Go Version](https://img.shields.io/badge/Go-1.23.0+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Version](https://img.shields.io/badge/Version-v0.1.16-orange.svg)](https://github.com/cgalvisleon/et/releases)

ET es una biblioteca Go moderna y robusta que proporciona una amplia gama de funcionalidades para el desarrollo de aplicaciones empresariales. Diseñada para ser modular, eficiente y fácil de usar.

## 🚀 Características

- **🔐 Autenticación y Autorización**: Sistema completo de JWT y manejo de sesiones
- **🗄️ Integración Multi-DB**: Soporte para múltiples bases de datos
- **⚡ Sistema de Caché**: Integración con Redis para alto rendimiento
- **📡 Mensajería**: NATS para comunicación entre servicios
- **☁️ AWS Integration**: Servicios AWS integrados
- **🌐 WebSockets**: Comunicación en tiempo real
- **📊 Eventos**: Sistema de eventos en tiempo real
- **⏰ Tareas Programadas**: Gestión de cron jobs
- **🛠️ CLI Interactiva**: Herramientas de línea de comandos
- **📝 Logging**: Sistema de logs avanzado
- **🔗 Middleware**: Middleware para Chi Router
- **📈 GraphQL**: Soporte nativo para GraphQL
- **🌍 Zonas Horarias**: Gestión de timezones
- **⚙️ Variables de Entorno**: Configuración flexible
- **🔧 Utilidades**: Strings, rutas y más utilidades
- **🏷️ Versionado**: Sistema de versionado automático
- **📧 Brevo Integration**: Email, SMS y WhatsApp
- **🔄 Data Transfer Objects**: Manejo de objetos de datos
- **🎯 Argumentos**: Gestión de argumentos de línea de comandos
- **🛡️ Resiliencia**: Patrones de resiliencia y circuit breakers
- **📊 Métricas**: Sistema de métricas y monitoreo
- **🔍 Búsqueda**: Funcionalidades de búsqueda avanzada

## 📋 Requisitos

- **Go**: 1.23.0 o superior
- **Redis**: Para sistema de caché (opcional)
- **NATS**: Para mensajería (opcional)
- **Neo4j**: Para base de datos de grafos (opcional)
- **AWS SDK**: Para servicios AWS (opcional)
- **Brevo API**: Para servicios de comunicación (opcional)

## 🛠️ Instalación

```bash
# Instalar la librería
go get github.com/cgalvisleon/et@v0.1.16

# O usar go mod
go mod init myproject
go get github.com/cgalvisleon/et
```

## ⚙️ Configuración

### Variables de Entorno

Crea un archivo `.env` en la raíz de tu proyecto:

```env
# Configuración de la aplicación
PROJECT_ID=my-project
APP_NAME=MyApp
APP_VERSION=1.0.0

# Base de datos
DB_HOST=localhost
DB_PORT=5432
DB_NAME=mydb
DB_USER=postgres
DB_PASSWORD=password

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# NATS
NATS_URL=nats://localhost:4222

# AWS
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key

# WebSocket
WS_PORT=3300
WS_MODE=development

# Resilience
RESILIENCE_ATTEMPTS=3
RESILIENCE_TIME_ATTEMPTS=30

# Brevo (Email, SMS, WhatsApp)
BREVO_API_KEY=your-brevo-api-key
BREVO_SENDER_EMAIL=noreply@yourdomain.com
BREVO_SENDER_NAME=Your App Name

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

## 📦 Gestión de Dependencias

### Dependencias Principales

```bash
# WebSocket y comunicación en tiempo real
go get github.com/gorilla/websocket

# Router HTTP
go get github.com/go-chi/chi/v5

# Utilidades
go get github.com/fsnotify/fsnotify
go get github.com/mattn/go-colorable
go get github.com/dimiro1/banner
go get github.com/shirou/gopsutil/v3/mem

# CLI
go get github.com/spf13/cobra
go get github.com/manifoldco/promptui

# Caché y mensajería
go get github.com/redis/go-redis/v9
go get github.com/nats-io/nats.go

# Autenticación
go get github.com/golang-jwt/jwt/v4

# Base de datos
go get github.com/neo4j/neo4j-go-driver/v5

# AWS
go get github.com/aws/aws-sdk-go

# Utilidades adicionales
go get github.com/google/uuid
go get github.com/bwmarrin/snowflake
go get github.com/oklog/ulid/v2
go get github.com/rs/xid
go get github.com/robfig/cron/v3
```

## 🚀 Comandos de Ejecución

### Servicios

```bash
# Ejecutar el servicio principal
go run ./cmd/et/main.go

# Ejecutar el daemon
go run ./cmd/daemon/main.go

# Ejecutar el apigateway
go run ./cmd/apigateway

# Ejecutar con parámetros
go run ./cmd/et/main.go -port 3300 -mode production

# Ejecutar con variables de entorno
PORT=8080 MODE=production go run ./cmd/et/main.go
```

### WebSockets

```bash
# Ejecutar servidor WebSocket
go run ./cmd/ws/main.go -port 3300

# Ejecutar con modo específico
go run ./cmd/ws/main.go -port 3300 -mode production

# Ejecutar con URL master
go run ./cmd/ws/main.go -port 3300 -master-url ws://master:3300/ws
```

### Herramientas de Creación

```bash
# Crear un nuevo proyecto
go run ./cmd/create/main.go

# Preparar un proyecto existente
go run ./cmd/prepare/main.go

# Ejecutar comandos específicos
go run ./cmd/et/main.go create
go run ./cmd/et/main.go prepare
```

## 💡 Ejemplos de Uso

### WebSocket Server

```go
package main

import (
    "os"
    "os/signal"
    "syscall"

    "github.com/cgalvisleon/et/config"
    "github.com/cgalvisleon/et/logs"
    "github.com/cgalvisleon/et/ws"
)

func main() {
    // Configurar parámetros
    port := config.SetIntByArg("port", 3300)
    mode := config.SetStrByArg("mode", "development")
    masterURL := config.SetStrByArg("master-url", "")

    // Iniciar servidor WebSocket
    hub := ws.ServerHttp(port, mode, masterURL)

    logs.Log("WebSocket", "Server started on port", port)

    // Esperar señal de interrupción
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
    <-sigs

    logs.Log("WebSocket", "Shutting down server...")
}
```

### WebSocket Client

```go
package main

import (
    "time"

    "github.com/cgalvisleon/et/et"
    "github.com/cgalvisleon/et/logs"
    "github.com/cgalvisleon/et/ws"
)

func main() {
    // Crear cliente WebSocket
    client, err := ws.NewClient(&ws.ClientConfig{
        ClientId:  "my-client",
        Name:      "MyClient",
        Url:       "ws://localhost:3300/ws",
        Reconnect: 3,
    })

    if err != nil {
        logs.Alert("Error creating client:", err)
        return
    }

    // Suscribirse a canales
    client.Subscribe("notifications", func(msg ws.Message) {
        logs.Log("Client", "Notification received:", msg.ToString())
    })

    client.Subscribe("updates", func(msg ws.Message) {
        logs.Log("Client", "Update received:", msg.ToString())
    })

    // Publicar mensajes
    go func() {
        for {
            client.Publish("notifications", et.Json{
                "type": "info",
                "message": "Hello from client!",
                "timestamp": time.Now().Unix(),
            })
            time.Sleep(5 * time.Second)
        }
    }()

    // Mantener el cliente activo
    select {}
}
```

### Sistema de Caché con Redis

```go
package main

import (
    "time"

    "github.com/cgalvisleon/et/cache"
    "github.com/cgalvisleon/et/et"
    "github.com/cgalvisleon/et/logs"
)

func main() {
    // Conectar a Redis
    err := cache.Connect()
    if err != nil {
        logs.Alert("Error connecting to Redis:", err)
        return
    }

    // Guardar datos en caché
    userData := et.Json{
        "id": 123,
        "name": "Juan Pérez",
        "email": "juan@example.com",
        "created_at": time.Now().Unix(),
    }

    err = cache.Set("user:123", userData, 3600) // Expira en 1 hora
    if err != nil {
        logs.Alert("Error setting cache:", err)
        return
    }

    // Obtener datos del caché
    data, err := cache.Get("user:123")
    if err != nil {
        logs.Alert("Error getting from cache:", err)
        return
    }

    logs.Log("Cache", "User data:", data.ToString())

    // Eliminar datos del caché
    cache.Delete("user:123")
}
```

### Servicios de Comunicación con Brevo

```go
package main

import (
    "github.com/cgalvisleon/et/brevo"
    "github.com/cgalvisleon/et/logs"
)

func main() {
    // Enviar email
    emailData := brevo.EmailData{
        To: []brevo.EmailContact{
            {Email: "user@example.com", Name: "Usuario"},
        },
        Subject: "Bienvenido a nuestra aplicación",
        HTMLContent: "<h1>¡Bienvenido!</h1><p>Gracias por registrarte.</p>",
    }

    err := brevo.SendEmail(emailData)
    if err != nil {
        logs.Error("Email", "Error sending email:", err)
    } else {
        logs.Log("Email", "Email sent successfully")
    }

    // Enviar SMS
    smsData := brevo.SMSData{
        To: "+1234567890",
        Message: "Tu código de verificación es: 123456",
    }

    err = brevo.SendSMS(smsData)
    if err != nil {
        logs.Error("SMS", "Error sending SMS:", err)
    } else {
        logs.Log("SMS", "SMS sent successfully")
    }

    // Enviar WhatsApp
    whatsappData := brevo.WhatsAppData{
        To: "+1234567890",
        Message: "¡Hola! Tu pedido ha sido confirmado.",
    }

    err = brevo.SendWhatsApp(whatsappData)
    if err != nil {
        logs.Error("WhatsApp", "Error sending WhatsApp:", err)
    } else {
        logs.Log("WhatsApp", "WhatsApp message sent successfully")
    }
}
```

### Data Transfer Objects (DTO)

```go
package main

import (
    "github.com/cgalvisleon/et/dt"
    "github.com/cgalvisleon/et/logs"
)

func main() {
    // Crear un objeto de datos
    userDTO := dt.NewObject("user")
    userDTO.Set("id", 123)
    userDTO.Set("name", "Juan Pérez")
    userDTO.Set("email", "juan@example.com")

    // Validar el objeto
    if userDTO.Valid() {
        logs.Log("DTO", "User object is valid")

        // Obtener datos
        name := userDTO.Get("name")
        logs.Log("DTO", "User name:", name)
    } else {
        logs.Error("DTO", "User object is invalid")
    }

    // Crear objeto con resiliencia
    resilientDTO := dt.NewResilientObject("user", 3, 30)
    resilientDTO.Set("id", 456)
    resilientDTO.Set("name", "María García")

    // El objeto maneja automáticamente reintentos y timeouts
    if resilientDTO.Valid() {
        logs.Log("DTO", "Resilient user object created successfully")
    }
}
```

### Manejo de Argumentos

```go
package main

import (
    "github.com/cgalvisleon/et/arg"
    "github.com/cgalvisleon/et/logs"
)

func main() {
    // Configurar argumentos
    arg.Set("port", 3300, "Puerto del servidor")
    arg.Set("mode", "development", "Modo de ejecución")
    arg.Set("debug", false, "Modo debug")

    // Parsear argumentos de línea de comandos
    arg.Parse()

    // Obtener valores
    port := arg.GetInt("port")
    mode := arg.GetStr("mode")
    debug := arg.GetBool("debug")

    logs.Log("Args", "Port:", port, "Mode:", mode, "Debug:", debug)

    // Verificar si un argumento existe
    if arg.Exists("custom-flag") {
        logs.Log("Args", "Custom flag is set")
    }
}
```

### Sistema de Logs

```go
package main

import (
    "github.com/cgalvisleon/et/logs"
)

func main() {
    // Diferentes niveles de log
    logs.Log("App", "This is an info message")
    logs.Debug("App", "This is a debug message")
    logs.Alert("App", "This is an alert message")
    logs.Error("App", "This is an error message")

    // Logs con contexto
    logs.Log("Database", "Connected to PostgreSQL")
    logs.Log("Cache", "Redis connection established")
    logs.Log("WebSocket", "Client connected: client-123")
}
```

### Tareas Programadas (Crontab)

```go
package main

import (
    "github.com/cgalvisleon/et/crontab"
    "github.com/cgalvisleon/et/et"
    "github.com/cgalvisleon/et/logs"
)

func main() {
    // Crear un administrador de tareas
    jobs := crontab.New()

    // Agregar una tarea que se ejecute cada minuto
    job, err := jobs.AddJob(
        "notification-sender",                    // Nombre de la tarea
        "0 * * * *",                            // Cada hora en punto (cron format)
        "notifications",                         // Canal de eventos
        et.Json{"type": "hourly", "action": "send"}, // Parámetros
        nil,                                    // Función personalizada (opcional)
    )

    if err != nil {
        logs.Error("Crontab", "Error adding job:", err)
        return
    }

    // Cargar tareas guardadas
    err = jobs.Load()
    if err != nil {
        logs.Error("Crontab", "Error loading jobs:", err)
    }

    // Iniciar el administrador de tareas
    err = jobs.Start()
    if err != nil {
        logs.Error("Crontab", "Error starting crontab:", err)
        return
    }

    // Iniciar una tarea específica
    idx, err := jobs.StartJob("notification-sender")
    if err != nil {
        logs.Error("Crontab", "Error starting job:", err)
    } else {
        logs.Log("Crontab", "Job started with ID:", idx)
    }

    // Listar todas las tareas
    jobsList := jobs.List()
    logs.Log("Crontab", "Active jobs:", jobsList.Count)

    // Guardar configuración
    err = jobs.Save()
    if err != nil {
        logs.Error("Crontab", "Error saving jobs:", err)
    }
}
```

### Sistema de Eventos en Tiempo Real

```go
package main

import (
    "github.com/cgalvisleon/et/event"
    "github.com/cgalvisleon/et/et"
    "github.com/cgalvisleon/et/logs"
)

func main() {
    // Conectar al sistema de eventos
    err := event.Connect()
    if err != nil {
        logs.Error("Event", "Error connecting:", err)
        return
    }

    // Suscribirse a un canal
    event.Subscribe("user-actions", func(data et.Json) {
        logs.Log("Event", "User action received:", data.ToString())

        // Procesar el evento
        action := data.Str("action")
        userId := data.Str("user_id")

        switch action {
        case "login":
            logs.Log("Event", "User logged in:", userId)
        case "logout":
            logs.Log("Event", "User logged out:", userId)
        case "purchase":
            amount := data.Float("amount")
            logs.Log("Event", "Purchase made by", userId, "amount:", amount)
        }
    })

    // Publicar eventos
    event.Publish("user-actions", et.Json{
        "action":    "login",
        "user_id":   "user-123",
        "timestamp": 1634567890,
        "ip":        "192.168.1.1",
    })

    event.Publish("user-actions", et.Json{
        "action":   "purchase",
        "user_id":  "user-123",
        "amount":   99.99,
        "product":  "Premium Plan",
        "currency": "USD",
    })

    // Mantener la aplicación activa
    select {}
}
```

### Daemon y Servicios del Sistema

```go
package main

import (
    "github.com/cgalvisleon/et/cmd/daemon"
    "github.com/cgalvisleon/et/et"
    "github.com/cgalvisleon/et/logs"
)

type MyService struct {
    name    string
    version string
    status  string
}

func (s *MyService) Help(key string) {
    logs.Log("Service", "Available commands: start, stop, restart, status, version")
}

func (s *MyService) Version() string {
    version := s.version
    logs.Log("Service", "Version:", version)
    return version
}

func (s *MyService) SetConfig(cfg string) {
    logs.Log("Service", "Setting config:", cfg)
}

func (s *MyService) Status() et.Json {
    return et.Json{
        "name":    s.name,
        "version": s.version,
        "status":  s.status,
        "uptime":  "2h 30m",
    }
}

func (s *MyService) Start() et.Item {
    s.status = "running"
    logs.Log("Service", "Service started")

    return et.Item{
        Ok: true,
        Result: et.Json{
            "status": "started",
            "pid":    12345,
        },
    }
}

func (s *MyService) Stop() et.Item {
    s.status = "stopped"
    logs.Log("Service", "Service stopped")

    return et.Item{
        Ok: true,
        Result: et.Json{
            "status": "stopped",
        },
    }
}

func (s *MyService) Restart() et.Item {
    s.Stop()
    return s.Start()
}

func main() {
    // Registrar el servicio
    service := &MyService{
        name:    "my-app",
        version: "1.0.0",
        status:  "stopped",
    }

    daemon.Registry("my-app", service)

    // El daemon maneja automáticamente los comandos:
    // go run main.go start
    // go run main.go stop
    // go run main.go status
    // go run main.go restart
}
```

### Middleware HTTP Avanzado

```go
package main

import (
    "net/http"

    "github.com/cgalvisleon/et/middleware"
    "github.com/cgalvisleon/et/ettp"
    "github.com/cgalvisleon/et/logs"
    "github.com/go-chi/chi/v5"
)

func main() {
    // Crear router
    r := chi.NewRouter()

    // Aplicar middleware de ET
    r.Use(middleware.Logger)           // Logging de requests
    r.Use(middleware.Recoverer)        // Recovery de panics
    r.Use(middleware.RequestID)        // Request ID único
    r.Use(middleware.CORS)             // CORS headers
    r.Use(middleware.WrapWrite)        // Response wrapper

    // Middleware de autenticación
    r.Use(middleware.Authentication)

    // Middleware de autorización
    r.Use(middleware.Authorization)

    // Rutas protegidas
    r.Route("/api/v1", func(r chi.Router) {
        r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
            // Handler protegido
            response := map[string]interface{}{
                "users": []string{"user1", "user2"},
            }

            ettp.WriteJSON(w, http.StatusOK, response)
        })

        r.Post("/users", func(w http.ResponseWriter, r *http.Request) {
            // Crear usuario
            ettp.WriteJSON(w, http.StatusCreated, map[string]string{
                "message": "User created successfully",
            })
        })
    })

    // Iniciar servidor
    port := ":8080"
    logs.Log("Server", "Starting on port", port)
    http.ListenAndServe(port, r)
}
```

### Manejo de Variables de Entorno

```go
package main

import (
    "github.com/cgalvisleon/et/envar"
    "github.com/cgalvisleon/et/logs"
)

func main() {
    // Obtener variables con valores por defecto
    dbHost := envar.GetStr("DB_HOST", "localhost")
    dbPort := envar.GetInt("DB_PORT", 5432)
    debugMode := envar.GetBool("DEBUG", false)

    logs.Log("Config", "Database:", dbHost, "Port:", dbPort, "Debug:", debugMode)
}
```

## 🔄 Publicación y Versiones

### Publicar Nueva Versión

```bash
# Limpiar y formatear el código
go mod tidy
gofmt -w .

# Actualizar git y crear nueva versión
git add .
git commit -m "Release v0.1.16"
git tag v0.1.16
git push origin main --tags

# Instalar la nueva versión
go get github.com/cgalvisleon/et@v0.1.16
```

## 📋 Versiones y Releases

### Historial de Versiones

#### v0.1.16

- Mejoras en el sistema de WebSockets
- Optimización del rendimiento del gateway
- Corrección de condiciones de carrera
- **Nuevo**: Integración con Brevo (Email, SMS, WhatsApp)
- **Nuevo**: Data Transfer Objects (DTO)
- **Nuevo**: Sistema de argumentos mejorado
- **Nuevo**: Patrones de resiliencia
- **Nuevo**: Sistema de daemon/servicios
- **Nuevo**: Crontab avanzado con persistencia
- **Nuevo**: Sistema de eventos mejorado
- **Nuevo**: Middleware HTTP completo
- **Nuevo**: Servidor ETTP optimizado
- **Nuevo**: Utilidades de archivos y sincronización
- **Nuevo**: Sistema de validaciones y criptografía
- **Nuevo**: Documentación mejorada
- **Nuevo**: Ejemplos de uso completos

#### v0.1.3

- Mejoras en el sistema de WebSockets
- Optimización del rendimiento del gateway
- Corrección de condiciones de carrera

#### v0.1.2

- Implementación de sistema de caché con Redis
- Nuevas utilidades para manejo de strings
- Mejoras en la documentación

#### v0.1.1

- Integración inicial con AWS
- Soporte para GraphQL
- Sistema de eventos en tiempo real

#### v0.1.0

- Lanzamiento inicial
- Sistema básico de autenticación
- Integración con bases de datos

### Política de Versionado

- Seguimos el versionado semántico (MAJOR.MINOR.PATCH)
- **MAJOR**: Cambios incompatibles con versiones anteriores
- **MINOR**: Nuevas funcionalidades compatibles
- **PATCH**: Correcciones de errores compatibles

### Proceso de Release

1. Desarrollo en rama feature
2. Pruebas y revisión de código
3. Merge a main
4. Tag de versión
5. Publicación en GitHub
6. Actualización de documentación

## 🧪 Desarrollo y Pruebas

### Condiciones de Carrera

```bash
# Compilar con detección de condiciones de carrera
go build --race ./cmd/et/main.go
go build --race ./cmd/ws/main.go

# Compilación normal
go build ./cmd/et/main.go
```

### Herramientas de Desarrollo

```bash
# Ejecutar herramientas de creación
go run ./cmd/create/main.go
go run ./cmd/prepare/main.go

# Ejecutar comandos específicos
go run ./cmd/et/main.go create
go run ./cmd/et/main.go prepare
```

### Testing

```bash
# Ejecutar todos los tests
go test ./...

# Ejecutar tests con coverage
go test -cover ./...

# Ejecutar tests específicos
go test ./cache/...
go test ./ws/...
go test ./brevo/...
```

## 🏗️ Arquitectura y Estructura

### Arquitectura de ET

```
┌─────────────────────────────────────────────────────────────┐
│                     Aplicación Usuario                      │
├─────────────────────────────────────────────────────────────┤
│                    ET Library Layer                         │
├─────────────────┬─────────────────┬─────────────────────────┤
│   Comunicación  │   Persistencia  │       Utilidades        │
│   - WebSockets  │   - Cache       │   - Logs                │
│   - Events      │   - Storage     │   - Config              │
│   - Brevo       │   - Redis       │   - Args                │
│   - AWS         │                 │   - Crypto              │
├─────────────────┼─────────────────┼─────────────────────────┤
│    Servicios    │   Middleware    │       Core              │
│   - HTTP/ETTP   │   - Auth        │   - et (JSON)           │
│   - Daemon      │   - CORS        │   - dt (DTO)            │
│   - Crontab     │   - Logger      │   - utility             │
│   - Realtime    │   - Recovery    │   - mistake             │
└─────────────────┴─────────────────┴─────────────────────────┘
```

### Estructura del Proyecto

```
et/
├── arg/         # Gestión de argumentos de línea de comandos
├── aws/         # Integración con servicios AWS (S3, SES, SMS)
├── brevo/       # Servicios de comunicación (Email, SMS, WhatsApp)
├── cache/       # Sistema de caché con Redis y Pub/Sub
├── claim/       # Sistema de claims y permisos JWT
├── cmd/         # Comandos CLI y ejecutables
│   ├── create/  # Generador de proyectos y plantillas
│   ├── daemon/  # Servicios en segundo plano y systemd
│   ├── et/      # Comando principal de la biblioteca
│   ├── prepare/ # Preparador de proyectos existentes
│   └── ws/      # Servidor WebSocket dedicado
├── config/      # Configuración y parámetros de aplicación
├── console/     # Consola interactiva y terminal
├── create/      # Templates y generadores de código
├── crontab/     # Sistema de tareas programadas con persistencia
├── dt/          # Data Transfer Objects con validación
├── envar/       # Variables de entorno y configuración
├── et/          # Utilidades principales y tipos JSON
├── ettp/        # Servidor HTTP optimizado con routing
├── event/       # Sistema de eventos en tiempo real
├── file/        # Manejo de archivos y sincronización
├── graph/       # Soporte GraphQL y consultas de grafos
├── jrpc/        # JSON-RPC para comunicación entre servicios
├── logs/        # Sistema de logs estructurados y avanzados
├── mem/         # Memoria compartida y sincronización
├── middleware/  # Middleware HTTP (Auth, CORS, Logger, etc.)
├── mistake/     # Manejo centralizado de errores
├── msg/         # Mensajes del sistema y localización
├── race/        # Detección y prevención de condiciones de carrera
├── realtime/    # Funcionalidades en tiempo real
├── reg/         # Registro de servicios y discovery
├── request/     # Manejo unificado de requests HTTP
├── resilience/  # Patrones de resiliencia y circuit breakers
├── response/    # Manejo unificado de responses HTTP
├── router/      # Enrutamiento HTTP avanzado
├── server/      # Servidor HTTP base
├── service/     # Servicios y utilidades de negocio
├── stdrout/     # Rutas estándar y endpoints comunes
├── strs/        # Utilidades para manejo de strings
├── timezone/    # Gestión avanzada de zonas horarias
├── units/       # Unidades de medida y conversiones
├── utility/     # Utilidades generales (crypto, validation, etc.)
└── ws/          # WebSocket y comunicación bidireccional
```

### Módulos Principales

#### 🔌 Comunicación

- **WebSockets**: Comunicación bidireccional en tiempo real
- **Events**: Sistema de eventos pub/sub
- **Brevo**: Email, SMS y WhatsApp
- **AWS**: Integración con servicios AWS

#### 💾 Persistencia y Caché

- **Cache**: Redis con pub/sub
- **File**: Sistema de archivos con watcher
- **Storage**: Almacenamiento persistente

#### 🛡️ Seguridad y Middleware

- **Middleware**: Auth, CORS, Logger, Recovery
- **Claim**: Sistema JWT avanzado
- **Crypto**: Utilidades criptográficas

#### ⚙️ Servicios y Utilidades

- **Daemon**: Servicios del sistema
- **Crontab**: Tareas programadas
- **Config**: Configuración centralizada
- **Logs**: Sistema de logging estructurado

## 🔧 API Reference

### WebSocket

```go
// Crear servidor
hub := ws.ServerHttp(port, mode, masterURL)

// Crear cliente
client := ws.NewClient(&ws.ClientConfig{
    ClientId:  "client-id",
    Name:      "Client Name",
    Url:       "ws://localhost:3300/ws",
    Reconnect: 3,
})

// Suscribirse a canal
client.Subscribe("channel", func(msg ws.Message) {
    // Manejar mensaje
})

// Publicar mensaje
client.Publish("channel", data)
```

### Cache

```go
// Conectar
cache.Connect()

// Operaciones básicas
cache.Set(key, value, ttl)
cache.Get(key)
cache.Delete(key)
cache.Exists(key)

// Pub/Sub
cache.Publish(channel, message)
cache.Subscribe(channel, handler)
```

### Brevo (Comunicación)

```go
// Email
brevo.SendEmail(brevo.EmailData{
    To: []brevo.EmailContact{{Email: "user@example.com"}},
    Subject: "Asunto",
    HTMLContent: "<p>Contenido</p>",
})

// SMS
brevo.SendSMS(brevo.SMSData{
    To: "+1234567890",
    Message: "Mensaje SMS",
})

// WhatsApp
brevo.SendWhatsApp(brevo.WhatsAppData{
    To: "+1234567890",
    Message: "Mensaje WhatsApp",
})
```

### Data Transfer Objects

```go
// Crear objeto
obj := dt.NewObject("name")

// Establecer propiedades
obj.Set("key", value)

// Validar
if obj.Valid() {
    // Usar objeto
}

// Objeto con resiliencia
resilientObj := dt.NewResilientObject("name", attempts, timeout)
```

### Argumentos

```go
// Configurar argumentos
arg.Set("name", defaultValue, "description")

// Parsear
arg.Parse()

// Obtener valores
value := arg.GetStr("name")
number := arg.GetInt("port")
flag := arg.GetBool("debug")
```

### Crontab (Tareas Programadas)

```go
// Crear administrador de tareas
jobs := crontab.New()

// Agregar tarea
job, err := jobs.AddJob(name, spec, channel, params, fn)

// Gestión de tareas
jobs.StartJob(name)
jobs.StopJob(name)
jobs.DeleteJob(name)

// Operaciones del administrador
jobs.Start()  // Iniciar crontab
jobs.Stop()   // Detener crontab
jobs.Load()   // Cargar desde cache
jobs.Save()   // Guardar en cache
jobs.List()   // Listar tareas
```

### Eventos

```go
// Conectar al sistema de eventos
event.Connect()

// Publicar evento
event.Publish(channel, data)

// Suscribirse a eventos
event.Subscribe(channel, handler)

// Emitir eventos
event.Emit(eventName, data)
```

### Middleware

```go
// Middleware básico
middleware.Logger           // Logging de requests
middleware.Recoverer        // Recovery de panics
middleware.RequestID        // Request ID único
middleware.CORS             // CORS headers
middleware.WrapWrite        // Response wrapper

// Middleware de seguridad
middleware.Authentication   // Autenticación JWT
middleware.Authorization   // Autorización basada en roles
middleware.Telemetry      // Métricas y telemetría
```

### ETTP (Servidor HTTP)

```go
// Respuestas JSON
ettp.WriteJSON(w, statusCode, data)

// Manejo de errores
ettp.WriteError(w, err)

// Servidor con configuración
server := ettp.NewServer(config)
server.Start()
```

### Utilidades

```go
// Generación de IDs
utility.UUID()              // UUID v4
utility.NewID()            // ID personalizado
utility.GenId()            // ID generado

// Validaciones
utility.ValidStr(str, min, exclusions)
utility.ValidEmail(email)
utility.ValidUrl(url)

// Criptografía
utility.PasswordHash(password)
utility.PasswordVerify(password, hash)
utility.Encrypt(data, key)
utility.Decrypt(data, key)
```

### Sistema de Archivos

```go
// Operaciones de archivos
file.Exists(path)
file.Read(path)
file.Write(path, data)
file.Delete(path)

// Watcher de archivos
watcher := file.NewWatcher()
watcher.Add(path)
watcher.Watch(handler)

// Sincronización
file.Sync(source, target)
```

### Logs

```go
// Niveles de log
logs.Log(component, message)
logs.Debug(component, message)
logs.Alert(component, message)
logs.Error(component, message)
```

## 🎯 Mejores Prácticas

### Configuración de Proyecto

```go
package main

import (
    "github.com/cgalvisleon/et/config"
    "github.com/cgalvisleon/et/logs"
    "github.com/cgalvisleon/et/cache"
    "github.com/cgalvisleon/et/event"
)

func init() {
    // Configurar logging
    logs.SetLevel("info")

    // Cargar configuración
    config.Load()

    // Conectar servicios
    if err := cache.Connect(); err != nil {
        logs.Alert("Cache connection failed:", err)
    }

    if err := event.Connect(); err != nil {
        logs.Alert("Event system connection failed:", err)
    }
}

func main() {
    logs.Log("App", "Application started successfully")
    // Tu aplicación aquí
}
```

### Estructura de Microservicio Recomendada

```
mi-microservicio/
├── cmd/
│   └── main.go              # Punto de entrada
├── internal/
│   ├── handlers/            # Handlers HTTP
│   ├── services/           # Lógica de negocio
│   ├── models/             # Modelos de datos
│   └── middleware/         # Middleware personalizado
├── pkg/
│   └── api/                # API pública
├── configs/
│   ├── .env.development
│   ├── .env.production
│   └── config.yaml
├── scripts/
│   ├── deploy.sh
│   └── test.sh
├── docker/
│   └── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

### Patrón de Inicialización

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/cgalvisleon/et/ettp"
    "github.com/cgalvisleon/et/logs"
    "github.com/cgalvisleon/et/config"
)

func main() {
    // Configuración inicial
    cfg := config.New()

    // Crear servidor
    server := ettp.NewServer(cfg)

    // Configurar rutas
    setupRoutes(server)

    // Iniciar servidor en goroutine
    go func() {
        if err := server.Start(); err != nil {
            logs.Error("Server", "Failed to start:", err)
            os.Exit(1)
        }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logs.Log("Server", "Shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        logs.Error("Server", "Server forced to shutdown:", err)
    }

    logs.Log("Server", "Server exited")
}
```

### Manejo de Errores

```go
package main

import (
    "github.com/cgalvisleon/et/mistake"
    "github.com/cgalvisleon/et/logs"
)

func businessLogic() error {
    // Crear error con contexto
    if someCondition {
        return mistake.New("BUSINESS_ERROR", "Something went wrong", "Additional context")
    }

    // Envolver error externo
    if err := externalCall(); err != nil {
        return mistake.Wrap(err, "EXTERNAL_CALL_FAILED", "Failed to call external service")
    }

    return nil
}

func handleError(err error) {
    if mistake.Is(err, "BUSINESS_ERROR") {
        logs.Alert("Business", "Business logic error:", err)
        // Manejar error de negocio
    } else {
        logs.Error("System", "System error:", err)
        // Manejar error del sistema
    }
}
```

## 🚨 Troubleshooting

### Problemas Comunes

#### Error de conexión a Redis

```bash
# Verificar que Redis esté ejecutándose
redis-cli ping

# Verificar configuración
echo $REDIS_HOST
echo $REDIS_PORT
```

#### Error de WebSocket

```bash
# Verificar puerto disponible
netstat -an | grep 3300

# Verificar firewall
sudo ufw status
```

#### Error de compilación

```bash
# Limpiar módulos
go clean -modcache
go mod tidy

# Verificar versión de Go
go version
```

#### Error de Brevo API

```bash
# Verificar API key
echo $BREVO_API_KEY

# Verificar configuración
echo $BREVO_SENDER_EMAIL
echo $BREVO_SENDER_NAME
```

## 🤝 Contribución

Las contribuciones son bienvenidas. Por favor, sigue estos pasos:

1. Fork el proyecto
2. Crea una rama para tu feature (`git checkout -b feature/AmazingFeature`)
3. Commit tus cambios (`git commit -m 'Add some AmazingFeature'`)
4. Push a la rama (`git push origin feature/AmazingFeature`)
5. Abre un Pull Request

### Guías de Contribución

- Mantén el código limpio y bien documentado
- Añade tests para nuevas funcionalidades
- Sigue las convenciones de Go
- Actualiza la documentación cuando sea necesario
- Incluye ejemplos de uso para nuevas funcionalidades

## 📝 Licencia

Este proyecto está bajo la Licencia MIT. Ver el archivo `LICENSE` para más detalles.

## 📧 Contacto

- **GitHub Issues**: [Abrir un issue](https://github.com/cgalvisleon/et/issues)
- **Documentación**: [Wiki del proyecto](https://github.com/cgalvisleon/et/wiki)
- **Discusiones**: [GitHub Discussions](https://github.com/cgalvisleon/et/discussions)

## 🙏 Agradecimientos

- A todos los contribuidores que han ayudado a mejorar esta librería
- A la comunidad de Go por las excelentes herramientas
- A los mantenedores de las dependencias utilizadas
- A Brevo por proporcionar excelentes servicios de comunicación

---

**ET** - Simplificando el desarrollo de aplicaciones empresariales en Go 🚀
