et/
├── arg/ # Gestión de argumentos de línea de comandos
├── aws/ # Integración con servicios AWS (S3, SES, SMS)
├── brevo/ # Servicios de comunicación (Email, SMS, WhatsApp)
├── cache/ # Sistema de caché con Redis y Pub/Sub
├── claim/ # Sistema de claims y permisos JWT
├── cmd/ # Comandos CLI y ejecutables
│ ├── apigateway/ # API Gateway y proxy de servicios
│ ├── context/ # Gestión de contexto de aplicación
│ ├── create/ # Generador de proyectos y plantillas
│ ├── daemon/ # Servicios en segundo plano y systemd
│ ├── et/ # Comando principal de la biblioteca
│ ├── prepare/ # Preparador de proyectos existentes
│ └── ws/ # Servidor WebSocket dedicado
├── cmds/ # Sistema de comandos y etapas de ejecución
├── config/ # Configuración y parámetros de aplicación
├── console/ # Consola interactiva y terminal
├── create/ # Templates y generadores de código
├── crontab/ # Sistema de tareas programadas con persistencia
├── dt/ # Data Transfer Objects con validación
├── envar/ # Variables de entorno y configuración
├── ephemeral/ # Gestión de datos temporales y efímeros
├── et/ # Utilidades principales y tipos JSON
├── ettp/ # Servidor HTTP optimizado con routing
├── event/ # Sistema de eventos en tiempo real
├── file/ # Manejo de archivos y sincronización
├── graph/ # Soporte GraphQL y consultas de grafos
├── jrpc/ # JSON-RPC para comunicación entre servicios
├── logs/ # Sistema de logs estructurados y avanzados
├── mem/ # Memoria compartida y sincronización
├── middleware/ # Middleware HTTP (Auth, CORS, Logger, etc.)
├── msg/ # Mensajes del sistema y localización
├── race/ # Detección y prevención de condiciones de carrera
├── realtime/ # Funcionalidades en tiempo real
├── reg/ # Registro de servicios y discovery
├── request/ # Manejo unificado de requests HTTP
├── resilience/ # Patrones de resiliencia y circuit breakers
├── response/ # Manejo unificado de responses HTTP
├── router/ # Enrutamiento HTTP avanzado
├── server/ # Servidor HTTP base
├── service/ # Servicios y utilidades de negocio
├── stdrout/ # Rutas estándar y endpoints comunes
├── strs/ # Utilidades para manejo de strings
├── timezone/ # Gestión avanzada de zonas horarias
├── units/ # Unidades de medida y conversiones
├── utility/ # Utilidades generales (crypto, validation, etc.)
├── workflow/ # Sistema de flujos de trabajo y procesos
└── ws/ # WebSocket y comunicación bidireccional
