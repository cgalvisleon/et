# Resultados: `websocket`

- Fecha: 2026-10-21 14:54:04 -05
- Go: go1.25.0, darwin/arm64
- Casos: **15/15 correctos**
- Requisitos de `spec.md`: **17/17 cumplidos**
- Métodos públicos probados: **32/32**
- Duración total: 2.288s

Generado con `go run ./test/websocket` (desde la raíz del repo).

## Requisitos

| ID | Requisito | Estado | Casos |
|---|---|---|---|
| WS-01 | Usa gorilla/websocket | ✅ cumple | 1, 15 |
| WS-02 | Conectarse (http.Handler o Listen) | ✅ cumple | 1, 6 |
| WS-03 | Suscribirse tipo topic | ✅ cumple | 2, 4, 10, 13 |
| WS-04 | Suscribirse tipo queue | ✅ cumple | 3, 4, 10, 13 |
| WS-05 | Desuscribirse | ✅ cumple | 4 |
| WS-06 | Publicar (servidor y cliente) | ✅ cumple | 2, 3, 5, 10, 13 |
| WS-07 | OnConnect (puede rechazar) | ✅ cumple | 7, 12 |
| WS-08 | OnDisconnect | ✅ cumple | 8, 14 |
| WS-09 | OnSubscribe (puede rechazar) | ✅ cumple | 9, 13 |
| WS-10 | OnUnsubscribe | ✅ cumple | 8, 9 |
| WS-11 | Modelo catalog | ✅ cumple | 15 |
| WS-12 | Concurrencia y consumidor lento | ✅ cumple | 10, 11 |
| WS-13 | Panic en eventos | ✅ cumple | 7, 9 |
| WS-14 | Cliente: NewClient y Connect devuelven un Client | ✅ cumple | 12 |
| WS-15 | Cliente: suscribir, queue, desuscribir y publicar | ✅ cumple | 13 |
| WS-16 | Cliente: recibir mensajes en orden | ✅ cumple | 13 |
| WS-17 | Cliente: OnConnect, OnDisconnect y reconexión | ✅ cumple | 12, 14 |

## Casos

| # | Caso | Requisitos | Resultado | Duración |
|---|---|---|---|---|
| 1 | Conexión: Listen, ServeHTTP y Close | WS-01, WS-02 | ✅ PASS | 7ms |
| 2 | Suscripción topic | WS-03, WS-06 | ✅ PASS | 4ms |
| 3 | Suscripción queue: reparto por turnos | WS-04, WS-06 | ✅ PASS | 619ms |
| 4 | Desuscribirse y reglas de suscripción | WS-05, WS-03, WS-04 | ✅ PASS | 157ms |
| 5 | Publicar desde un cliente y mensajes directos | WS-06 | ✅ PASS | 5ms |
| 6 | Protocolo: mensajes inválidos | WS-02 | ✅ PASS | 3ms |
| 7 | OnConnect: autenticar y rechazar | WS-07, WS-13 | ✅ PASS | 55ms |
| 8 | OnDisconnect: cierre del cliente, del servidor y Server.Close | WS-08, WS-10 | ✅ PASS | 5ms |
| 9 | OnSubscribe: rechazar y ampliar | WS-09, WS-10, WS-13 | ✅ PASS | 3ms |
| 10 | Concurrencia: 20 clientes y 5 publicadores | WS-12, WS-03, WS-04, WS-06 | ✅ PASS | 358ms |
| 11 | Consumidor lento | WS-12 | ✅ PASS | 969ms |
| 12 | Cliente Go: NewClient, Connect y rechazo | WS-14, WS-17, WS-07 | ✅ PASS | 2ms |
| 13 | Cliente Go: suscribir, queue, publicar y recibir en orden | WS-15, WS-16, WS-03, WS-04, WS-06, WS-09 | ✅ PASS | 85ms |
| 14 | Cliente Go: desconexión y reconexión | WS-17, WS-08 | ✅ PASS | 7ms |
| 15 | Modelo catalog (análisis del código) | WS-11, WS-01 | ✅ PASS | 9ms |

## Detalle

### 1. Conexión: Listen, ServeHTTP y Close

Métodos: `NewServer`, `Server.Listen`, `Server.Addr`, `Server.Count`, `Server.ServeHTTP`, `Server.Close`

- Listen en 127.0.0.1:55347
- segundo Listen -> websocket: server already listening
- 3 clientes Gorilla conectados: Count=3
- montado en http.ServeMux en /ws: conecta (Count=1)
- Close cerró los 3 clientes; Count=0
- upgrade después de Close -> HTTP 503

### 2. Suscripción topic

Métodos: `Server.Publish`, `Server.OnConnect`, `Conn.Subscriptions`

- repetir la misma suscripción -> ack
- Publish del servidor -> delivered=3; los 3 suscriptores topic lo recibieron (from="")
- Subscriptions() = [{Channel:news Queue: Type:topic}]

### 3. Suscripción queue: reparto por turnos

Métodos: `Server.Publish`

- 30 mensajes: grupo a (3 miembros) [10 10 10], grupo b (2 miembros) [15 15], topic 30
- cada grupo recibió cada mensaje exactamente una vez; el topic los 30

### 4. Desuscribirse y reglas de suscripción

Métodos: `Server.Publish`

- queue sobre canal ya suscrito como topic -> websocket: already subscribed to x
- después de unsubscribe no llega nada (delivered=0)
- unsubscribe no suscrito -> websocket: not subscribed to x
- canal vacío -> websocket: channel is required
- tras desuscribirse, se puede suscribir como queue

### 5. Publicar desde un cliente y mensajes directos

Métodos: `Conn.ID`, `Conn.Send`, `Server.OnConnect`

- Client.Send en OnConnect -> cada cliente recibió su mensaje direct
- publish de A -> B lo recibe con from=01M5G0E8…; ack delivered=1
- publish en canal sin suscriptores -> ack delivered=0

### 6. Protocolo: mensajes inválidos

- inválido -> error "websocket: invalid message"
- inválido -> error "websocket: invalid message"
- tipo desconocido -> websocket: unknown message type otro
- la conexión sigue abierta: subscribe -> ack

### 7. OnConnect: autenticar y rechazar

Métodos: `Server.OnConnect`, `Conn.Request`, `Conn.Subscribe`, `Conn.SubscribeQueue`, `Server.Count`

- token inválido -> cierre 1008 "token inválido"
- panic en OnConnect -> rechazada: "websocket: handler panic: token malo"
- token válido -> conectado (Count=1) y suscrito desde OnConnect a un topic y a una queue

### 8. OnDisconnect: cierre del cliente, del servidor y Server.Close

Métodos: `Server.OnDisconnect`, `Server.OnUnsubscribe`, `Conn.Close`, `Server.Close`

- el cliente cierra -> [unsubscribe:dos unsubscribe:uno disconnect:websocket: connection closed]
- el servidor cierra al cliente (Client.Close) -> disconnect:websocket: connection closed; el cliente recibió websocket: close 1000 (normal)
- Server.Close espera a OnUnsubscribe y OnDisconnect -> [unsubscribe:tres disconnect:websocket: connection closed]

### 9. OnSubscribe: rechazar y ampliar

Métodos: `Server.OnSubscribe`, `Server.OnUnsubscribe`, `Conn.Subscribe`, `Conn.Unsubscribe`, `Conn.Subscriptions`

- OnSubscribe rechaza -> el cliente recibe "sin permiso para privado" y no queda suscrito
- panic en OnSubscribe -> websocket: handler panic: boom
- OnSubscribe recibe el tipo y el grupo: {Channel:cola Queue:g Type:queue}
- OnSubscribe amplía: subscribe a sala -> también sala.presencia (Subscriptions=3)
- OnUnsubscribe amplía: unsubscribe de sala -> también sala.presencia (panic en OnUnsubscribe registrado, sin efecto)

### 10. Concurrencia: 20 clientes y 5 publicadores

Métodos: `Server.Publish`

- 500 publicaciones concurrentes en 338ms
- 10 topic: cada uno recibió las 500 en orden por publicador; queue (10 miembros): 500 en total

### 11. Consumidor lento

Métodos: `Server.Publish`, `Server.OnDisconnect`

- tras 267 mensajes de 64 KiB, el lector lento se desconectó: websocket: connection closed: websocket: slow consumer, send buffer full
- Publish nunca se bloqueó (747ms en total)
- el cliente rápido recibió los 267

### 12. Cliente Go: NewClient, Connect y rechazo

Métodos: `NewClient`, `Client.Connect`, `Client.ID`, `Client.Close`, `Client.OnConnect`, `Conn.ID`

- Connect rechazado por OnConnect del servidor -> websocket: connection rejected: token inválido
- Connect aceptado: Client.ID()=01M5G0EA… coincide con Conn.ID() del servidor; OnConnect del cliente llamado
- segundo Connect -> websocket: already connected
- Close; operación sin conexión -> websocket: not connected
- Connect a un servidor cerrado -> dial tcp 127.0.0.1:55425: connect: connection refused

### 13. Cliente Go: suscribir, queue, publicar y recibir en orden

Métodos: `Client.Subscribe`, `Client.SubscribeQueue`, `Client.Unsubscribe`, `Client.Publish`, `Client.Subscriptions`, `Client.OnMessage`, `Client.OnDirect`, `Conn.Send`

- OnDirect recibió el Conn.Send del servidor: "bienvenido"
- Subscribe rechazado por el servidor -> sin permiso
- 500 Publish desde un Client (delivered=1500): los topic recibieron 500 en orden con From=ID del publicador; la queue repartió 250/250
- Unsubscribe confirmado: la siguiente publicación llega a 2 (el publicador y la queue)
- Subscribe desde dentro de OnMessage -> ok (sin bloqueo)

### 14. Cliente Go: desconexión y reconexión

Métodos: `Client.OnDisconnect`, `Client.Connect`, `Client.Subscriptions`, `Conn.Close`

- el servidor cierra la conexión -> OnDisconnect: websocket: connection closed
- se reconectó desde OnDisconnect con otro ID; las suscripciones se limpian y se vuelven a hacer
- el servidor se apaga -> OnDisconnect: websocket: connection closed

### 15. Modelo catalog (análisis del código)

- usa github.com/gorilla/websocket: true
- 32 métodos públicos, todos en catalog.go como envoltorios de su versión privada

## Cobertura de métodos públicos

| Método | Probado en |
|---|---|
| `NewServer` | 1 |
| `Server.ServeHTTP` | 1 |
| `Server.Listen` | 1 |
| `Server.Addr` | 1 |
| `Server.Close` | 1, 8 |
| `Server.Count` | 1, 7 |
| `Server.Publish` | 2, 3, 4, 10, 11 |
| `Server.OnConnect` | 2, 5, 7 |
| `Server.OnDisconnect` | 8, 11 |
| `Server.OnSubscribe` | 9 |
| `Server.OnUnsubscribe` | 8, 9 |
| `Conn.ID` | 5, 12 |
| `Conn.Request` | 7 |
| `Conn.Send` | 5, 13 |
| `Conn.Subscribe` | 7, 9 |
| `Conn.SubscribeQueue` | 7 |
| `Conn.Unsubscribe` | 9 |
| `Conn.Subscriptions` | 2, 9 |
| `Conn.Close` | 8, 14 |
| `NewClient` | 12 |
| `Client.Connect` | 12, 14 |
| `Client.Close` | 12 |
| `Client.ID` | 12 |
| `Client.Subscribe` | 13 |
| `Client.SubscribeQueue` | 13 |
| `Client.Unsubscribe` | 13 |
| `Client.Publish` | 13 |
| `Client.Subscriptions` | 13, 14 |
| `Client.OnConnect` | 12 |
| `Client.OnDisconnect` | 14 |
| `Client.OnMessage` | 13 |
| `Client.OnDirect` | 13 |
