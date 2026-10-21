## Hub

Crea un Hub TCP tomando como punto de partida el siguiente struct y el struct `Message` para el envío y la recepción.

```go
type Hub struct {
	clients       map[string]*Client
	servers       map[string]*Server
	timeoutServer *duration // compartido con todos los servidores
	timeoutClient *duration // compartido con todos los clientes
}

const (
	TypeBeat = iota // 0
	TypeInt         // 1
	TypeFloat       // 2
	TypeText        // 3
	TypeJson        // 4
)

type Message struct {
	From    et.Json
	Header  et.Json
	Type    int
	Message any
}
```

### Objetivo

- Crear un hub TCP de conexiones con múltiples clientes y múltiples servidores, para llevar tráfico entre ellos (ejemplo: replicación).
- El Hub debe tener un `NewHub` que solo crea el Hub.
- Debe tener métodos para cargar o dar de baja un servidor: `LoadServer(port)` y `DownServer(port)`.
- Debe permitir saber que se recibe un mensaje con un `OnMessage` por servidor, identificando en cuál servidor y enviado por cuál cliente.
- El servidor debe tener métricas por cliente conectado (cuáles están conectados) y un `OnConnect`; cuando un cliente se conecta, envía también su nombre.
- También `OnDisconnect`, para saber qué cliente se desconectó.
- Debe permitir conectarse como cliente a diversos servidores con `ConnectTo(host:port)` y también cerrar esa conexión.
- Debe tener `Heartbeat`, que permita conocer el estado de los clientes a los que está conectado.
- Debe permitir enviar mensajes con la estructura `Message`.

### Complementos (revisión 2026-09-29)

Lo que el objetivo deja implícito y hace falta para que funcione; el detalle normativo está en `spec.md` §14.

- Identidad del Hub: `NewHub(id, name)`. Todos sus servidores y clientes usan esa identidad; el nombre es el que viaja al conectarse (saludo `msgHello` que ya existe) y el que va en `Message.From`.
- `From` lo pone el Hub al enviar (`{"id","name"}` del Hub); el usuario llena `Header`, `Type` y `Message`.
- Tipo y valor coinciden: `TypeInt` → `int64`, `TypeFloat` → `float64`, `TypeText` → `string`, `TypeJson` → `et.Json`, `TypeBeat` → sin valor. Un mensaje que no cumple no se envía.
- Orden: para replicación, los mensajes de una misma conexión llegan a `OnMessage` en el orden en que se enviaron (el `Send` de `tcp` no lo garantiza, así que el Hub lo resuelve).
- Heartbeat entre Hubs: los latidos (`TypeBeat`) los contesta el Hub remoto y nunca llegan a `OnMessage`. Tras varios latidos sin respuesta, la conexión se da por muerta y se cierra.
- Envío en los dos sentidos: además de cliente → servidor (`Send(addr, msg)`), un servidor puede enviar a un cliente conectado (`SendTo(port, clientID, msg)`); lo que llega por una conexión que abrió el Hub también pasa por `OnMessage`, con servidor `nil`.
- Timeouts compartidos: `SetTimeoutServer` y `SetTimeoutClient` cambian el timeout de todos los servidores o de todos los clientes del Hub, también los que se creen después.
- Cierre: `Close()` detiene el heartbeat, cierra los clientes y da de baja los servidores.
- Modelo catalog: los métodos públicos del Hub van en `catalog.go`, como el resto del paquete.

### Decisiones a confirmar

- `OnMessage`, `OnConnect` y `OnDisconnect` son del Hub (reciben el servidor como parámetro) y no por servidor. Si se prefiere uno por servidor, cambia la firma de `LoadServer`.
- Un servidor se identifica por su puerto (`LoadServer(0)` toma un puerto libre); un cliente, por la dirección pasada a `ConnectTo`.
- Una conexión caída queda en el Hub en estado `down` (visible en el estado del heartbeat) hasta `Disconnect`; volver a llamar a `ConnectTo` la reconecta. No hay reconexión automática.

### Seguridad (revisión 2026-10-16)

El hub lleva la replicación y las escrituras reenviadas entre nodos. Hasta ahora era TCP plano y aceptaba a cualquiera que supiera el puerto.

- **Autenticación mutua:** solo conversan los dos lados que conocen un secreto compartido. Ninguno envía el secreto; cada uno prueba que lo conoce, y una prueba capturada no sirve para otra conexión.
- **Cifrado opcional:** TLS 1.3. Con un certificado propio, o uno autofirmado en memoria cuando solo se quiere cifrar; con una CA, TLS mutuo verificado.
- Se configura en el cliente, en el servidor y en el Hub (que lo comparte con todos sus servidores y clientes). Sin configurarlo, todo sigue como antes.
- Detalle normativo: `spec.md` §7.4.

### Fuera de alcance

Reconexión automática, broadcast a todos los clientes, confirmación de entrega (ack) y persistencia de mensajes. Rotar el secreto sin cortar las conexiones.
