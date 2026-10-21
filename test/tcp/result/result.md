# Resultados: `tcp`

- Fecha: 2026-10-21 14:54:00 -05
- Go: go1.25.0, darwin/arm64
- Casos: **25/25 correctos**
- Métodos públicos probados: **70/70**
- Duración total: 2.041s

Generado con `go run ./test/tcp` (desde la raíz del repo).

## Casos

| # | Caso | Resultado | Duración |
|---|---|---|---|
| 1 | Servidor: ciclo de vida | ✅ PASS | 2ms |
| 2 | Identificación: saludo, ID, Name y ToJson | ✅ PASS | 1ms |
| 3 | Saludo fallido | ✅ PASS | 303ms |
| 4 | Cliente: conexión | ✅ PASS | 3ms |
| 5 | Timeout | ✅ PASS | 253ms |
| 6 | Send y OnSend en los dos sentidos | ✅ PASS | 12ms |
| 7 | Request y OnRequest: orden, error, sin handler y panic | ✅ PASS | 2ms |
| 8 | Concurrencia: 5 clientes, en los dos sentidos | ✅ PASS | 71ms |
| 9 | OnConnect y OnDisconnect en los dos lados | ✅ PASS | 13ms |
| 10 | Reconexión desde OnDisconnect | ✅ PASS | 6ms |
| 11 | SendStream: orden y control de flujo | ✅ PASS | 225ms |
| 12 | RequestStream: 20 a la vez | ✅ PASS | 78ms |
| 13 | ReceiveStream en los dos sentidos | ✅ PASS | 130ms |
| 14 | Streams del servidor al cliente | ✅ PASS | 1ms |
| 15 | Streams: errores y cancelación | ✅ PASS | 52ms |
| 16 | Hub: creación, servidores y baja (HUB-01, 02, 03) | ✅ PASS | 22ms |
| 17 | Hub: mensajes tipados, orden y eventos (HUB-04, 05, 06, 11) | ✅ PASS | 39ms |
| 18 | Hub: métricas y ToJson (HUB-08) | ✅ PASS | 14ms |
| 19 | Hub: clientes, caída, reconexión y Disconnect (HUB-07, 09) | ✅ PASS | 15ms |
| 20 | Hub: heartbeat, Status y timeouts compartidos (HUB-10, 12) | ✅ PASS | 414ms |
| 21 | Hub: mensajes ilegibles, panics y Close (HUB-13, 14) | ✅ PASS | 15ms |
| 22 | Seguridad: secreto en el saludo (§7.4) | ✅ PASS | 107ms |
| 23 | Seguridad: TLS, TLS mutuo y canal ligado (§7.4) | ✅ PASS | 64ms |
| 24 | Seguridad: Hub (§7.4, §14) | ✅ PASS | 12ms |
| 25 | Seguridad: rotación del secreto, dos secretos durante una ventana (§7.4) | ✅ PASS | 186ms |

## Detalle

### 1. Servidor: ciclo de vida

Métodos: `NewServer`, `Server.Listen`, `Server.Addr`, `Server.Count`, `Server.Close`

- escuchando en 127.0.0.1:55208
- segundo Listen -> tcp: server already listening
- Count con 3 clientes = 3
- después de Close: Count=0, Addr vacío
- volvió a escuchar en 127.0.0.1:55212

### 2. Identificación: saludo, ID, Name y ToJson

Métodos: `NewClient`, `Client.ID`, `Client.Name`, `Client.To`, `Client.ServerID`, `Client.ServerName`, `Client.ToJson`, `NewServer`, `Server.ID`, `Server.Name`, `Server.SetName`, `Server.ToJson`, `Server.OnConnect`, `Server.OnSend`

- NewServer/NewClient conservan id y nombre; con id vacío generan uno (01a16007…)
- el cliente conoce al servidor por el saludo: ServerID=nodo-1 ServerName=nodo-a To=127.0.0.1:55213
- en OnConnect del servidor, la conexión trae la identidad del cliente: cliente-1/replica
- OnSend del servidor sabe quién envió: cliente-1/replica
- Server.ToJson -> map[addr:127.0.0.1:55213 id:nodo-1 name:nodo-a peers:[map[auth:false id:cliente-1 name:replica server:map[id:nodo-1 name:nodo-a] tls:false to:127.0.0.1:55214]]]
- Client.ToJson -> map[auth:false id:cliente-1 name:replica server:map[id:nodo-1 name:nodo-a] tls:false to:127.0.0.1:55213]

### 3. Saludo fallido

Métodos: `Client.ConnectTo`, `Client.SetTimeout`, `Server.SetTimeout`, `Server.Count`

- servidor que no saluda -> ConnectTo: tcp: handshake failed: read tcp 127.0.0.1:55217->127.0.0.1:55216: i/o timeout
- cliente que no saluda -> el servidor lo cierra (EOF); Count=0, sin OnConnect

### 4. Cliente: conexión

Métodos: `NewClient`, `Client.ConnectTo`, `Client.Close`, `Client.Send`, `Client.Request`

- Send sin conexión -> tcp: not connected
- Request sin conexión -> tcp: not connected
- segundo ConnectTo -> tcp: already connected
- Close y reconexión correctos

### 5. Timeout

Métodos: `Client.SetTimeout`, `Server.SetTimeout`

- Client.SetTimeout(0) -> tcp: timeout must be greater than zero
- Server.SetTimeout(-1s) -> tcp: timeout must be greater than zero
- timeout 50ms (conexión abierta) -> tcp: request timeout en 50ms
- timeout 1s -> respuesta "tarde"

### 6. Send y OnSend en los dos sentidos

Métodos: `Client.Send`, `Server.OnSend`, `Client.OnSend`, `Server.OnConnect`

- cliente→servidor: 100, servidor→cliente: 100, segunda función del servidor: 100

### 7. Request y OnRequest: orden, error, sin handler y panic

Métodos: `Client.Request`, `Server.OnRequest`, `Client.OnRequest`

- (nil, nil) cede a la siguiente -> "segunda"
- error del handler -> falló a propósito
- ninguna responde -> tcp: no request handler
- panic -> tcp: handler panic: boom
- servidor→cliente Request -> "cliente:x"

### 8. Concurrencia: 5 clientes, en los dos sentidos

Métodos: `Client.Send`, `Client.Request`, `Server.OnRequest`, `Server.OnSend`, `Client.OnRequest`, `Client.OnSend`, `Server.OnConnect`

- 1000 Request y 1000 Send en cada sentido en 32ms

### 9. OnConnect y OnDisconnect en los dos lados

Métodos: `Client.OnConnect`, `Client.OnDisconnect`, `Server.OnConnect`, `Server.OnDisconnect`, `Client.Close`, `Server.Close`

- OnConnect: cliente 2, servidor 2 (con un panic en otra función)
- cierra el cliente -> OnDisconnect en ambos lados
- cierra el servidor -> OnDisconnect en ambos lados; motivo en el cliente: tcp: connection closed

### 10. Reconexión desde OnDisconnect

Métodos: `Client.OnDisconnect`, `Client.ConnectTo`, `Server.OnConnect`

- el servidor cortó, el cliente se reconectó solo y respondió "de nuevo"

### 11. SendStream: orden y control de flujo

Métodos: `Client.SendStream`, `Server.OnSendStream`, `StreamWriter.Write`, `StreamWriter.Close`, `StreamReader.Read`

- 2000 trozos en orden
- escritor frenado por el lector lento: 224ms
- Request atendidos mientras tanto por la misma conexión: 4153

### 12. RequestStream: 20 a la vez

Métodos: `Client.RequestStream`, `Server.OnRequestStream`, `StreamWriter.Write`, `StreamWriter.CloseAndReceive`, `StreamReader.Read`

- 20 RequestStream de 300 trozos, cada uno con su suma correcta
- CloseAndReceive en un SendStream -> tcp: not a request stream

### 13. ReceiveStream en los dos sentidos

Métodos: `Client.ReceiveStream`, `Server.OnReceiveStream`, `Client.OnReceiveStream`, `StreamWriter.Write`, `StreamReader.Read`

- 20 ReceiveStream de 500 trozos a la vez: 10 del cliente al servidor y 10 del servidor al cliente

### 14. Streams del servidor al cliente

Métodos: `Client.OnSendStream`, `Client.OnRequestStream`, `Client.SendStream`, `Client.RequestStream`, `StreamWriter.Write`, `StreamWriter.Close`, `StreamWriter.CloseAndReceive`, `StreamReader.Read`

- SendStream servidor→cliente -> el cliente recibió [a b c]
- RequestStream servidor→cliente -> el cliente respondió "xyz"

### 15. Streams: errores y cancelación

Métodos: `StreamWriter.CloseWithError`, `StreamReader.Close`, `Server.OnSendStream`, `Server.OnRequestStream`, `Server.OnReceiveStream`, `Client.SendStream`, `Client.RequestStream`, `Client.ReceiveStream`

- RequestStream sin handler -> tcp: no request handler
- ReceiveStream sin handler -> tcp: no request handler
- CloseWithError -> el lector recibe "el productor falló"
- ReceiveStream con error en el handler -> leyó [a] y luego el generador falló
- ReceiveStream con panic en el handler -> leyó [a] y luego tcp: handler panic: boom
- panic en OnRequestStream -> tcp: handler panic: boom
- StreamReader.Close -> el escritor recibe tcp: stream canceled by the reader
- conexión cerrada a mitad del stream -> leyó [1] y luego tcp: connection closed

### 16. Hub: creación, servidores y baja (HUB-01, 02, 03)

Métodos: `NewHub`, `Hub.ID`, `Hub.Name`, `Hub.LoadServer`, `Hub.DownServer`, `Hub.Servers`, `Hub.Metrics`

- NewHub sin id -> 01a16007-1463-7be6-a92c-77811eb7ea97; no arranca goroutines ni escucha
- LoadServer(0) en :55267 y :[::]:55268; repetido -> MSG_HUB_SERVER_EXISTS
- DownServer cierra las conexiones; repetido y Metrics -> MSG_HUB_SERVER_NOT_FOUND

### 17. Hub: mensajes tipados, orden y eventos (HUB-04, 05, 06, 11)

Métodos: `Hub.ConnectTo`, `Hub.Send`, `Hub.SendTo`, `Hub.OnMessage`, `Hub.OnConnect`

- 500 TypeInt en orden, como int64, con From del Hub (el From del llamador se ignora)
- OnConnect: servidor ve al cliente "nodo-b"; cliente ve al servidor "nodo-a"
- TypeFloat -> float64, TypeText -> string, TypeJson -> et.Json
- tipo fuera de rango, TypeBeat, valor que no corresponde y nil -> MSG_HUB_INVALID_MESSAGE
- SendTo servidor -> cliente llega con srv nil; id desconocido -> hub: client not found: otro; addr desconocido -> hub: client not found: 127.0.0.1:1; puerto -> hub: no server loaded on port 1

### 18. Hub: métricas y ToJson (HUB-08)

Métodos: `Hub.Metrics`, `Hub.ToJson`

- Metrics: {"addr":"127.0.0.1:55273","bytes_in":851,"bytes_out":80,"connected_at":"2026-10-21T14:53:59.201462-05:00","id":"hub-b","last_beat":null,"last_message":"2026-10-21T14:53:59.203204-05:00","messages_in":10,"messages_out":1,"name":"nodo-b"}
- al cerrarse, la conexión sale de las métricas

### 19. Hub: clientes, caída, reconexión y Disconnect (HUB-07, 09)

Métodos: `Hub.ConnectTo`, `Hub.Disconnect`, `Hub.Clients`, `Hub.OnDisconnect`, `Hub.Status`

- el servidor cae -> el cliente queda registrado en down; Send -> tcp: not connected
- ConnectTo sobre un cliente caído reconecta el mismo *Client
- OnDisconnect, una vez por conexión y lado: [a srv=true nodo-b b srv=false b srv=false a srv=true nodo-b]

### 20. Hub: heartbeat, Status y timeouts compartidos (HUB-10, 12)

Métodos: `Hub.Heartbeat`, `Hub.Status`, `Hub.SetTimeoutClient`, `Hub.SetTimeoutServer`

- Status: {"addr":"127.0.0.1:55278","connected_at":"2026-10-21T14:53:59.231261-05:00","failures":0,"last_beat":"2026-10-21T14:53:59.251947-05:00","latency_ms":0.259,"server":{"id":"hub-a","name":"nodo-a"},"state":"up"}
- 3 latidos sin respuesta -> cierre con hub: no heartbeat response
- Heartbeat(0) lo detiene

### 21. Hub: mensajes ilegibles, panics y Close (HUB-13, 14)

Métodos: `Hub.Close`, `Hub.OnMessage`, `Hub.Servers`, `Hub.Clients`

- un panic en OnMessage se registra; la siguiente función y el siguiente mensaje siguen
- seq 1 inválido se descarta y cuenta; seq 2 (llegado antes) se entrega
- Data sin seq legible -> el Hub cierra la conexión con hub: invalid message
- un Request con JSON que no es latido (sin from.id) llega al OnRequest del usuario
- Close vacía el Hub y se puede volver a usar (LoadServer en [::]:55284)

### 22. Seguridad: secreto en el saludo (§7.4)

Métodos: `Client.SetSecurity`, `Server.SetSecurity`, `Client.ToJson`

- mismo secreto: conecta, auth=true en los dos lados, el mensaje pasa
- secreto distinto -> tcp: handshake failed: tcp: authentication failed
- cliente sin secreto -> tcp: handshake failed: tcp: authentication required
- servidor sin secreto -> tcp: handshake failed: tcp: authentication failed

### 23. Seguridad: TLS, TLS mutuo y canal ligado (§7.4)

Métodos: `NewTLS`, `Client.SetSecurity`, `Server.SetSecurity`, `Client.ToJson`

- TLS 1.3 autofirmado + secreto: conecta, tls=true y auth=true, Request pasa
- cliente sin TLS contra servidor TLS -> tcp: handshake failed: EOF
- intermediario TLS que reenvía el saludo -> tcp: handshake failed: tcp: authentication failed
- contraprueba: sin secreto (TLS sin CA) el intermediario pasa
- TLS mutuo con CA (certificado para 127.0.0.1): conecta
- cliente con certificado ajeno a la CA -> tcp: handshake failed: remote error: tls: certificate required
- NewTLS con certificado sin llave -> tcp: TLS needs both the certificate and the key

### 24. Seguridad: Hub (§7.4, §14)

Métodos: `Hub.SetSecurity`, `Hub.ConnectTo`, `Hub.Send`, `Hub.Clients`

- dos hubs con secreto + TLS: conectan, el mensaje llega, tls=true y auth=true
- hub con otro secreto -> tcp: handshake failed: tcp: authentication failed

### 25. Seguridad: rotación del secreto, dos secretos durante una ventana (§7.4)

Métodos: `Client.SetSecurity`, `Server.SetSecurity`, `Client.ToJson`

- servidor con los dos, cliente con el viejo: conecta, auth=true en los dos lados
- servidor con el viejo, cliente con los dos: conecta, auth=true en los dos lados
- servidor con los dos, cliente con el nuevo: conecta, auth=true en los dos lados
- servidor con el nuevo, cliente con los dos: conecta, auth=true en los dos lados
- los dos con los dos: conecta, auth=true en los dos lados
- servidor solo con el nuevo, cliente solo con el viejo -> tcp: handshake failed: tcp: authentication failed
- servidor solo con el viejo, cliente solo con el nuevo -> tcp: handshake failed: tcp: authentication failed
- dos secretos en cada lado, ninguno en común -> tcp: handshake failed: tcp: authentication failed

## Cobertura de métodos públicos

| Método | Probado en |
|---|---|
| `NewClient` | 2, 4 |
| `Client.ToJson` | 2, 22, 23, 25 |
| `Client.ID` | 2 |
| `Client.Name` | 2 |
| `Client.To` | 2 |
| `Client.ServerID` | 2 |
| `Client.ServerName` | 2 |
| `Client.ConnectTo` | 3, 4, 10 |
| `Client.Close` | 4, 9 |
| `Client.SetTimeout` | 3, 5 |
| `Client.Send` | 4, 6, 8 |
| `Client.Request` | 4, 7, 8 |
| `Client.SendStream` | 11, 14, 15 |
| `Client.RequestStream` | 12, 14, 15 |
| `Client.ReceiveStream` | 13, 15 |
| `Client.OnConnect` | 9 |
| `Client.OnDisconnect` | 9, 10 |
| `Client.OnSend` | 6, 8 |
| `Client.OnRequest` | 7, 8 |
| `Client.OnSendStream` | 14 |
| `Client.OnRequestStream` | 14 |
| `Client.OnReceiveStream` | 13 |
| `NewServer` | 1, 2 |
| `Server.ToJson` | 2 |
| `Server.ID` | 2 |
| `Server.Name` | 2 |
| `Server.SetName` | 2 |
| `Server.Listen` | 1 |
| `Server.Addr` | 1 |
| `Server.Close` | 1, 9 |
| `Server.Count` | 1, 3 |
| `Server.SetTimeout` | 3, 5 |
| `Server.OnConnect` | 2, 6, 8, 9, 10 |
| `Server.OnDisconnect` | 9 |
| `Server.OnSend` | 2, 6, 8 |
| `Server.OnRequest` | 7, 8 |
| `Server.OnSendStream` | 11, 15 |
| `Server.OnRequestStream` | 12, 15 |
| `Server.OnReceiveStream` | 13, 15 |
| `StreamWriter.Write` | 11, 12, 13, 14 |
| `StreamWriter.Close` | 11, 14 |
| `StreamWriter.CloseWithError` | 15 |
| `StreamWriter.CloseAndReceive` | 12, 14 |
| `StreamReader.Read` | 11, 12, 13, 14 |
| `StreamReader.Close` | 15 |
| `NewHub` | 16 |
| `Hub.ID` | 16 |
| `Hub.Name` | 16 |
| `Hub.ToJson` | 18 |
| `Hub.LoadServer` | 16 |
| `Hub.DownServer` | 16 |
| `Hub.Servers` | 16, 21 |
| `Hub.Metrics` | 16, 18 |
| `Hub.ConnectTo` | 17, 19, 24 |
| `Hub.Disconnect` | 19 |
| `Hub.Clients` | 19, 21, 24 |
| `Hub.Send` | 17, 24 |
| `Hub.SendTo` | 17 |
| `Hub.OnMessage` | 17, 21 |
| `Hub.OnConnect` | 17 |
| `Hub.OnDisconnect` | 19 |
| `Hub.Heartbeat` | 20 |
| `Hub.Status` | 19, 20 |
| `Hub.SetTimeoutServer` | 20 |
| `Hub.SetTimeoutClient` | 20 |
| `Hub.Close` | 21 |
| `Client.SetSecurity` | 22, 23, 25 |
| `Server.SetSecurity` | 22, 23, 25 |
| `Hub.SetSecurity` | 24 |
| `NewTLS` | 23 |
