# ET - Biblioteca Go

ET es una biblioteca Go moderna y robusta que proporciona una amplia gama de funcionalidades para el desarrollo de aplicaciones empresariales.

## 🚀 Características

- Manejo de autenticación y autorización
- Integración con múltiples bases de datos
- Sistema de caché con Redis
- Mensajería con NATS
- Integración con AWS
- Manejo de WebSockets
- Sistema de eventos en tiempo real
- Gestión de tareas programadas
- CLI interactiva
- Manejo de logs
- Middleware para Chi Router
- Soporte para GraphQL
- Gestión de zonas horarias
- Manejo de variables de entorno
- Utilidades para strings y rutas
- Sistema de versionado

## 📋 Requisitos

- Go 1.23.0 o superior
- Redis
- NATS (opcional)
- Neo4j (opcional)
- AWS SDK (opcional)

## 🛠️ Instalación

```bash
go get github.com/cgalvisleon/et
```

## 📦 Gestión de Dependencias

### Dependencias Principales

```bash
go get github.com/fsnotify/fsnotify
go get github.com/gorilla/websocket
go get github.com/mattn/go-colorable
go get github.com/dimiro1/banner
go get github.com/go-chi/chi/v5
go get github.com/shirou/gopsutil/v3/mem
go get github.com/googollee/go-socket.io
go get github.com/satyakb/go-socket.io-redis
```

## 🚀 Comandos de Ejecución

### Servicios

```bash
# Ejecutar el servicio principal
go run ./cmd/service/main.go

# Ejecutar el gateway
go run ./cmd/gateway/main.go -port 3300 -rpc 4200
```

## 🌐 WebSockets

### Servidor y Cliente

```bash
# Ejecutar el servidor WebSocket
go run ./cmd/ws/server

# Ejecutar el cliente WebSocket
go run ./cmd/ws/client
```

## 🔄 Publicación y Versiones

### Publicar Nueva Versión

```bash
# Limpiar y formatear el código
go mod tidy
gofmt -w .

# Actualizar git y crear nueva versión
git update
git tag v0.1.4
git tags

# Instalar la nueva versión
go get github.com/cgalvisleon/et@v0.1.4
```

## 📋 Versiones y Releases

### Historial de Versiones

#### v0.1.4

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
- MAJOR: Cambios incompatibles con versiones anteriores
- MINOR: Nuevas funcionalidades compatibles
- PATCH: Correcciones de errores compatibles

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

## 🏗️ Estructura del Proyecto

```
et/
├── aws/         # Integración con AWS
├── cache/       # Sistema de caché
├── cmd/         # Comandos CLI
├── config/      # Configuración
├── event/       # Sistema de eventos
├── graph/       # Soporte GraphQL
├── middleware/  # Middleware HTTP
├── msg/         # Mensajes del sistema
├── realtime/    # Funcionalidades en tiempo real
├── router/      # Enrutamiento
├── service/     # Servicios
└── utility/     # Utilidades generales
```

## 🚀 Inicio Rápido

```go
package main

import (
    "github.com/cgalvisleon/et"
)

func main() {
    // Inicializar la aplicación
    app := et.New()

    // Configurar y ejecutar
    app.Run()
}
```

## 💡 Ejemplos de Uso

### WebSocket Server

```go
package main

import (
    "github.com/cgalvisleon/et/config"
    "github.com/cgalvisleon/et/ws"
)

func main() {
    port := config.SetIntByArg("port", 3300)
    mode := config.SetStrByArg("mode", "")
    masterURL := config.SetStrByArg("master-url", "")

    // Iniciar servidor WebSocket
    hub := ws.ServerHttp(port, mode, masterURL)

    // El servidor se ejecuta hasta recibir señal de interrupción
    select {}
}
```

### WebSocket Client

```go
package main

import (
    "github.com/cgalvisleon/et/et"
    "github.com/cgalvisleon/et/ws"
)

func main() {
    client, err := ws.NewClient(&ws.ClientConfig{
        ClientId:  "my-client",
        Name:      "MyClient",
        Url:       "ws://localhost:3300/ws",
        Reconnect: 3,
    })

    if err != nil {
        panic(err)
    }

    // Suscribirse a un canal
    client.Subscribe("notifications", func(msg ws.Message) {
        println("Mensaje recibido:", msg.ToString())
    })

    // Publicar mensaje
    client.Publish("notifications", et.Json{
        "type": "info",
        "message": "Hola mundo!",
    })
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

### Sistema de Caché

```go
package main

import (
    "github.com/cgalvisleon/et/cache"
    "github.com/cgalvisleon/et/et"
)

func main() {
    // Conectar a Redis
    err := cache.Connect()
    if err != nil {
        panic(err)
    }

    // Guardar datos en caché
    cache.Set("user:123", et.Json{
        "id": 123,
        "name": "Juan Pérez",
        "email": "juan@example.com",
    }, 3600) // Expira en 1 hora

    // Obtener datos del caché
    data, err := cache.Get("user:123")
    if err == nil {
        println("Usuario:", data.ToString())
    }
}
```

## 📝 Licencia

Este proyecto está bajo la Licencia MIT. Ver el archivo `LICENSE` para más detalles.

## 👥 Contribución

Las contribuciones son bienvenidas. Por favor, lee las guías de contribución antes de enviar un pull request.

## 📧 Contacto

Para soporte o consultas, por favor abre un issue en el repositorio.
