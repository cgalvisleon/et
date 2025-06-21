# ET - Biblioteca Go

[![Go Version](https://img.shields.io/badge/Go-1.23.0+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Version](https://img.shields.io/badge/Version-v0.1.5-orange.svg)](https://github.com/cgalvisleon/et/releases)

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

## 📋 Requisitos

- **Go**: 1.23.0 o superior
- **Redis**: Para sistema de caché (opcional)
- **NATS**: Para mensajería (opcional)
- **Neo4j**: Para base de datos de grafos (opcional)
- **AWS SDK**: Para servicios AWS (opcional)

## 🛠️ Instalación

```bash
# Instalar la librería
go get github.com/cgalvisleon/et@v0.1.5

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
```

## 📦 Gestión de Dependencias

### Dependencias Principales

```bash
# WebSocket y comunicación en tiempo real
go get github.com/gorilla/websocket
go get github.com/googollee/go-socket.io
go get github.com/satyakb/go-socket.io-redis

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
```

## 🚀 Comandos de Ejecución

### Servicios

```bash
# Ejecutar el servicio principal
go run ./cmd/service/main.go

# Ejecutar el gateway con parámetros
go run ./cmd/gateway/main.go -port 3300 -rpc 4200

# Ejecutar con variables de entorno
PORT=8080 RPC_PORT=9090 go run ./cmd/gateway/main.go
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

### Creación de Microservicios

```bash
# Crear un nuevo microservicio interactivamente
go run ./cmd/create go

# Opciones disponibles:
# - Project: Crear un proyecto completo
# - Microservice: Crear un microservicio
# - Modelo: Crear un modelo de datos
# - Rpc: Crear un servicio RPC
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
git commit -m "Release v0.1.5"
git tag v0.1.5
git push origin main --tags

# Instalar la nueva versión
go get github.com/cgalvisleon/et@v0.1.5
```

## 📋 Versiones y Releases

### Historial de Versiones

#### v0.1.5

- Mejoras en el sistema de WebSockets
- Optimización del rendimiento del gateway
- Corrección de condiciones de carrera
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
go build --race ./cmd/gateway/main.go
go build --race ./cmd/service/main.go

# Compilación normal
go build ./cmd/gateway/main.go
```

### Herramientas de Desarrollo

```bash
# Ejecutar herramientas de creación
go run github.com/cgalvisleon/et/cmd/create go
go run github.com/cgalvisleon/et/cmd/cmd

# Ejecutar comandos locales
go run ./cmd/create go
go run ./cmd
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
```

## 🏗️ Estructura del Proyecto

```
et/
├── aws/         # Integración con AWS
├── cache/       # Sistema de caché con Redis
├── cmd/         # Comandos CLI y ejecutables
│   ├── create/  # Generador de proyectos
│   ├── ws/      # Servidor WebSocket
│   └── daemon/  # Servicios en segundo plano
├── config/      # Configuración y parámetros
├── create/      # Templates y generadores
├── event/       # Sistema de eventos
├── graph/       # Soporte GraphQL
├── middleware/  # Middleware HTTP
├── msg/         # Mensajes del sistema
├── realtime/    # Funcionalidades en tiempo real
├── resilience/  # Sistema de resiliencia
├── router/      # Enrutamiento HTTP
├── service/     # Servicios y utilidades
├── utility/     # Utilidades generales
├── ws/          # WebSocket y comunicación
└── timezone/    # Gestión de zonas horarias
```

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

### Logs

```go
// Niveles de log
logs.Log(component, message)
logs.Debug(component, message)
logs.Alert(component, message)
logs.Error(component, message)
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

---

**ET** - Simplificando el desarrollo de aplicaciones empresariales en Go 🚀
