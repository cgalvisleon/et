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

## 📦 Dependencias Principales

- `github.com/aws/aws-sdk-go` - Integración con AWS
- `github.com/go-chi/chi` - Router HTTP
- `github.com/golang-jwt/jwt` - Manejo de JWT
- `github.com/gorilla/websocket` - Soporte WebSocket
- `github.com/nats-io/nats.go` - Mensajería
- `github.com/neo4j/neo4j-go-driver` - Driver Neo4j
- `github.com/redis/go-redis` - Cliente Redis
- `github.com/spf13/cobra` - CLI
- `github.com/robfig/cron` - Tareas programadas

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

## 📝 Licencia

Este proyecto está bajo la Licencia MIT. Ver el archivo `LICENSE` para más detalles.

## 👥 Contribución

Las contribuciones son bienvenidas. Por favor, lee las guías de contribución antes de enviar un pull request.

## 📧 Contacto

Para soporte o consultas, por favor abre un issue en el repositorio.

```
go mod init github.com/cgalvisleon/et
go get github.com/cgalvisleon/et/@v1.0.10
```

## Dependencis

```
go get github.com/fsnotify/fsnotify
go get github.com/gorilla/websocket
go get github.com/mattn/go-colorable
go get github.com/dimiro1/banner
go get github.com/go-chi/chi/v5
go get github.com/shirou/gopsutil/v3/mem
go get github.com/googollee/go-socket.io
go get github.com/satyakb/go-socket.io-redis
```

## CDM

```
go run ./cmd/service/main.go
go run ./cmd/gateway/main.go -port 3300 -rpc 4200
```

## WS

```
go run ./cmd/ws/server
go run ./cmd/ws/client
```

# Public

```
go mod tidy &&
gofmt -w . &&
git update &&
git tag v0.1.3 &&
git tags

go get github.com/cgalvisleon/et@v0.1.3
```

## Condicion de carrera

```
go build --race ./cmd/gateway/main.go
go build --race ./cmd/serive/main.go

go build ./cmd/gateway/main.go

go run github.com/cgalvisleon/et/cmd/create go
go run github.com/cgalvisleon/et/cmd/cmd

go run ./cmd/create go
go run ./cmd
```
