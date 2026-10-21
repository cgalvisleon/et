# Especificación: `websocket`

Referencia del paquete `websocket`: define el protocolo y lo que garantiza cada operación. Sale de `feature.md`; el estado de cumplimiento de cada requisito se lleva en `brecha.md`.

**Regla de mantenimiento:** todo cambio en `websocket` que altere el protocolo, la API pública o un comportamiento descrito aquí actualiza este archivo en el mismo cambio, junto con `brecha.md` y, si cambia la API pública, `test/websocket/test.go` (lista `publicAPI` y casos).

Versión del documento: 4 — 2026-10-21 (el paquete pasa de josefina `internal/websocket` a `github.com/cgalvisleon/et/websocket`, con sus mensajes en `et/msg` y el verificador en `test/websocket`; sin cambios de API; versión 3: cliente Go: `NewClient`/`Connect` devuelven un `Client`; la conexión vista desde el servidor pasa a llamarse `Conn`; mensaje `connected`).

## 1. Objetivo (de `feature.md`)

Servidor WebSocket sobre `github.com/gorilla/websocket` que permita conectarse, suscribirse (topic), suscribirse de tipo queue (un miembro por grupo, por turnos), desuscribirse y publicar desde el servidor o un cliente; con `OnConnect`, `OnDisconnect`, `OnSubscribe` y `OnUnsubscribe` para ampliar sus capacidades (`OnConnect` y `OnSubscribe` pueden rechazar); montable en cualquier router o escuchando solo; sin que un cliente lento frene a los demás; con un cliente Go (`NewClient` y `Connect` devuelven un `Client`) para usar el servidor; y con los métodos públicos agrupados en `catalog.go`.

**Decisión:** `et/jws` ya tiene un hub WebSocket (canales topic/queue/stack), pero no tiene `OnSubscribe`/`OnUnsubscribe` ni permite rechazar una conexión o suscripción. Este paquete es propio, sobre Gorilla directamente, y sigue el modelo de `tcp` (implementación privada + `catalog.go`).

## 2. Requisitos

| ID | Requisito | Origen |
|---|---|---|
| WS-01 | El servidor usa `github.com/gorilla/websocket` para el protocolo WebSocket. | feature |
| WS-02 | Un cliente se conecta por HTTP (upgrade a WebSocket). El servidor se puede montar en cualquier router (`http.Handler`) o escuchar solo (`Listen`). | feature: conectarse |
| WS-03 | Un cliente se suscribe a un canal (tipo **topic**): recibe **todos** los mensajes publicados en él. | feature: suscribir |
| WS-04 | Un cliente se suscribe a un canal de tipo **queue**, dentro de un grupo: cada mensaje del canal lo recibe **un solo** miembro del grupo, repartidos por turnos. | feature: suscribirse de tipo queue |
| WS-05 | Un cliente se desuscribe de un canal. | feature: desuscribirse |
| WS-06 | Se publican mensajes en un canal, desde el servidor (`Publish`) y desde un cliente (mensaje `publish`). | feature: publicar |
| WS-07 | `OnConnect`: funciones que se llaman al conectarse un cliente; pueden **rechazar** la conexión. | feature: OnConnect puede rechazar |
| WS-08 | `OnDisconnect`: funciones que se llaman al desconectarse un cliente, con el motivo. | feature |
| WS-09 | `OnSubscribe`: funciones que se llaman antes de cada suscripción (topic o queue); pueden **rechazarla**. | feature: OnSubscribe puede rechazar |
| WS-10 | `OnUnsubscribe`: funciones que se llaman después de cada desuscripción, también las implícitas al desconectarse. | feature |
| WS-11 | Los métodos públicos son envoltorios en `catalog.go` de implementaciones privadas con el mismo nombre en minúscula. | feature: modelo catalog |
| WS-12 | Todas las operaciones son seguras bajo concurrencia; un cliente lento no frena a los demás. | feature: cliente lento |
| WS-13 | Un panic en una función de evento no cae el proceso. | calidad (igual que `tcp`) |
| WS-14 | Cliente Go: `NewClient()` devuelve un `*Client`; `Connect(url)` se conecta y espera a que el servidor acepte; si `OnConnect` del servidor rechaza, `Connect` retorna el motivo. | feature: cliente con NewClient y Connect |
| WS-15 | Cliente Go: `Subscribe`, `SubscribeQueue`, `Unsubscribe` y `Publish` esperan la confirmación del servidor y retornan su error (p. ej. un rechazo de `OnSubscribe`). | feature: cliente |
| WS-16 | Cliente Go: `OnMessage` y `OnDirect` reciben los mensajes en el orden en que llegaron, sin frenar las confirmaciones. | feature: cliente |
| WS-17 | Cliente Go: `OnConnect` y `OnDisconnect`; tras desconectarse se puede volver a llamar a `Connect` (también desde `OnDisconnect`). | feature: cliente |

## 3. Protocolo

Mensajes de texto con un objeto JSON cada uno. Los mensajes binarios, el JSON inválido o un `type` desconocido se contestan con `error`; la conexión sigue abierta.

### 3.1 Cliente → servidor

| `type` | Campos | Efecto | Requisito |
|---|---|---|---|
| `subscribe` | `channel` | Suscripción topic. | WS-03 |
| `queue` | `channel`, `queue` (grupo; opcional, por defecto `""`) | Suscripción queue en el grupo. | WS-04 |
| `unsubscribe` | `channel` | Quita la suscripción al canal, sea topic o queue. | WS-05 |
| `publish` | `channel`, `data` (cualquier JSON; opcional) | Publica en el canal. | WS-06 |

Todos aceptan `id` (texto, opcional): se devuelve en la respuesta para correlacionarla.

### 3.2 Servidor → cliente

| `type` | Campos | Cuándo |
|---|---|---|
| `ack` | `id`; en `publish` además `delivered` (cuántos clientes lo recibieron) | La operación se hizo. |
| `error` | `id`, `message` | La operación falló o el mensaje era inválido. |
| `message` | `channel`, `queue` (solo si llegó por una suscripción queue), `from` (id del cliente que publicó, `""` si fue el servidor), `data` | Un mensaje publicado en un canal suscrito. |
| `direct` | `data` | Mensaje enviado a este cliente con `Conn.Send`. |
| `connected` | `id` (el id del cliente) | Una vez, cuando `OnConnect` acepta la conexión (§6.1). |

Toda operación del cliente recibe exactamente un `ack` o un `error`.

## 4. Suscripciones

- Un canal es un nombre no vacío; coincide exacto (sin comodines). Existe mientras tenga suscriptores.
- Un cliente tiene **a lo sumo una suscripción por canal**: topic, o queue en un grupo.
  - Repetir la misma suscripción no hace nada (`ack`, sin llamar a `OnSubscribe`).
  - Suscribirse distinto a un canal ya suscrito (topic ↔ queue, u otro grupo) → `MSG_WS_ALREADY_SUBSCRIBED`; hay que desuscribirse antes.
- `unsubscribe` de un canal no suscrito → `MSG_WS_NOT_SUBSCRIBED`.
- Canal vacío → `MSG_WS_CHANNEL_REQUIRED`.
- El servidor también puede suscribir y desuscribir a un cliente (`Conn.Subscribe`, `Conn.SubscribeQueue`, `Conn.Unsubscribe`), con las mismas reglas y los mismos eventos.

## 5. Entrega (`Publish`)

Al publicar en un canal:

1. Cada suscriptor **topic** recibe el mensaje una vez.
2. Cada **grupo queue** del canal lo entrega a **un** miembro: el siguiente en turno (round-robin) que tenga espacio en su buffer; si ninguno tiene espacio, ese grupo no lo recibe.
3. Quien publica también lo recibe si está suscrito.

- `Publish` retorna cuántos clientes lo recibieron (`delivered`). Publicar en un canal sin suscriptores no es error: `delivered = 0`.
- **Garantía:** a lo sumo una vez, en memoria. No hay persistencia ni reintentos: un cliente desconectado pierde lo que se publique mientras tanto.
- **Orden:** los mensajes de un mismo publicador en un mismo canal llegan a cada suscriptor en el orden en que se publicaron. Entre publicadores distintos no hay orden garantizado.
- `data` del servidor (`Publish`, `Send`) se codifica con `encoding/json`; para enviar JSON ya codificado, usar `json.RawMessage`. `data` de un cliente se reenvía tal cual.

## 6. Conexión

### 6.1 Conectar (WS-02, WS-07)

1. El servidor hace el upgrade con Gorilla (origen: la regla por defecto de Gorilla — mismo origen, o sin encabezado `Origin`).
2. Crea el cliente con un id nuevo (ULID) y empieza a enviarle.
3. Llama a las funciones de `OnConnect` en orden. En ellas se puede usar `Send`, `Subscribe`, `SubscribeQueue` y leer `Request()` (encabezados, query) para autenticar.
   - Si una retorna error (o hace panic): la conexión se **rechaza** — se deshacen las suscripciones hechas en `OnConnect` (llamando a `OnUnsubscribe`), se envía un cierre `1008` (policy violation) con el texto del error y se cierra. No se llama a `OnDisconnect` y nunca cuenta en `Count`.
4. Si todas aceptan: el cliente queda **conectado** (cuenta en `Count`), recibe `connected` con su id y se empiezan a leer sus mensajes. Lo que se le envió en `OnConnect` (`Send`, publicaciones) le llega antes que `connected`.

### 6.2 Desconectar (WS-08, WS-10)

Una conexión termina cuando el cliente cierra, falla la red, se vence el keepalive, llega un mensaje de más de 1 MiB, es un consumidor lento (§7), o el servidor la cierra (`Conn.Close`, `Server.Close`). En ese momento, una sola vez:

1. Se cierra el socket y el cliente deja de recibir.
2. Se quitan todas sus suscripciones, llamando a `OnUnsubscribe` por cada una.
3. Si estaba conectado, se llama a las funciones de `OnDisconnect` con `(client, err)`. `err` es `MSG_WS_CLOSED` en un cierre normal (del cliente o del servidor) y `MSG_WS_CLOSED: <causa>` en otro caso.

### 6.3 Servidor

- `ServeHTTP(w, r)`: atiende una conexión; bloquea hasta que termina. Así se monta en cualquier router.
- `Listen(addr)`: además abre un servidor HTTP propio en `addr` que atiende todas las rutas con `ServeHTTP` (`":0"` elige puerto; `Addr()` lo retorna). Un segundo `Listen` → `MSG_WS_ALREADY_LISTENING`.
- `Close()`: deja de aceptar (los nuevos upgrades reciben `503` con `MSG_WS_SERVER_CLOSED`), cierra el servidor HTTP propio si lo hay, cierra todos los clientes con cierre normal y **espera** a que terminen sus `OnUnsubscribe`/`OnDisconnect`. Es definitivo. No se debe llamar desde una función de evento del mismo servidor.

## 7. Envío, keepalive y límites

- Cada cliente tiene un buffer de salida de **256** mensajes y una goroutine que los escribe en orden.
- **Consumidor lento:** si al publicar el buffer de un suscriptor topic está lleno, ese cliente se desconecta (`MSG_WS_SLOW_CONSUMER`); los demás no esperan. En queue, se salta al siguiente miembro (§5).
- `Conn.Send` con buffer lleno retorna `MSG_WS_SLOW_CONSUMER` sin desconectar.
- Keepalive: el servidor envía ping cada 54 s; si no recibe nada (pong u otro mensaje) en 60 s, desconecta. Cada escritura tiene 10 s de plazo.
- Tamaño máximo de un mensaje del cliente: 1 MiB.

## 8. Funciones de evento

| Evento | Firma | Cuándo | Puede rechazar |
|---|---|---|---|
| `OnConnect` | `func(c *Conn) error` | §6.1 | sí: error → conexión rechazada |
| `OnDisconnect` | `func(c *Conn, err error)` | §6.2 | no |
| `OnSubscribe` | `func(c *Conn, sub Subscription) error` | antes de registrar la suscripción | sí: error → la suscripción no se hace y el cliente recibe `error` con ese texto |
| `OnUnsubscribe` | `func(c *Conn, sub Subscription)` | después de quitar la suscripción | no |

- Se agregan (no reemplazan) y se llaman en el orden en que se agregaron. En las que pueden rechazar, la primera que retorna error detiene las siguientes.
- `Subscription` = `{Channel, Queue string, Type}` con `Type` = `TypeTopic` o `TypeQueue`.
- Se llaman **sin locks del servidor tomados**: pueden usar cualquier método, también sobre el mismo cliente (por ejemplo, suscribirlo a otro canal desde `OnSubscribe`).
- **Panic:** se recupera y se registra en el log (`MSG_WS_HANDLER_PANIC`). En `OnConnect`/`OnSubscribe` cuenta como rechazo; en `OnDisconnect`/`OnUnsubscribe` se sigue con las demás.

## 9. Concurrencia

- Todos los métodos de `Server`, `Conn` y `Client` se pueden usar desde varias goroutines a la vez.
- Los mensajes de un cliente se procesan en orden, uno a la vez; cada cliente en su propia goroutine.
- `Publish` nunca bloquea esperando a un cliente: solo deja el mensaje en su buffer.
- Las operaciones de suscripción de un mismo cliente se aplican una a la vez; entre la llamada a `OnSubscribe` y el registro, se vuelve a validar (si mientras tanto el cliente se desconectó → `MSG_WS_CLOSED`).

## 10. API pública (`catalog.go`)

**Servidor:** `NewServer() (*Server, error)`, `ServeHTTP(w, r)`, `Listen(addr) error`, `Addr() string`, `Close() error`, `Count() int`, `Publish(channel string, data any) (int, error)`, `OnConnect`, `OnDisconnect`, `OnSubscribe`, `OnUnsubscribe`.

**Conexión (`Conn`, un cliente visto desde el servidor):** `ID() string`, `Request() *http.Request`, `Send(data any) error`, `Subscribe(channel string) error`, `SubscribeQueue(channel, queue string) error`, `Unsubscribe(channel string) error`, `Subscriptions() []Subscription`, `Close() error`.

**Cliente (`Client`, §13):** `NewClient() (*Client, error)`, `Connect(url string) error`, `Close() error`, `ID() string`, `Subscribe(channel string) error`, `SubscribeQueue(channel, queue string) error`, `Unsubscribe(channel string) error`, `Publish(channel string, data any) (int, error)`, `Subscriptions() []Subscription`, `OnConnect(func(c *Client))`, `OnDisconnect(func(c *Client, err error))`, `OnMessage(func(c *Client, m Message))`, `OnDirect(func(c *Client, data json.RawMessage))`.

## 11. Errores

Textos en `internal/msg`, en inglés o español según `LANG`.

| Constante | Cuándo |
|---|---|
| `MSG_WS_CLOSED` | Motivo de desconexión; operación sobre un cliente ya desconectado. |
| `MSG_WS_SERVER_CLOSED` | Upgrade después de `Server.Close`. |
| `MSG_WS_ALREADY_LISTENING` | Segundo `Listen`. |
| `MSG_WS_CHANNEL_REQUIRED` | Canal vacío. |
| `MSG_WS_ALREADY_SUBSCRIBED` | Suscripción distinta a un canal ya suscrito. |
| `MSG_WS_NOT_SUBSCRIBED` | Desuscribirse de un canal no suscrito. |
| `MSG_WS_INVALID_MESSAGE` | Mensaje binario o JSON inválido. |
| `MSG_WS_UNKNOWN_TYPE` | `type` desconocido. |
| `MSG_WS_SLOW_CONSUMER` | Buffer de salida lleno. |
| `MSG_WS_HANDLER_PANIC` | Una función de evento hizo panic. |
| `MSG_WS_NOT_CONNECTED` | Operación del `Client` sin conexión. |
| `MSG_WS_ALREADY_CONNECTED` | `Connect` con una conexión abierta. |
| `MSG_WS_TIMEOUT` | El servidor no confirmó `Connect` o una operación a tiempo (10 s). |
| `MSG_WS_REJECTED` | `OnConnect` del servidor rechazó la conexión; incluye el motivo. |

## 13. Cliente Go (`Client`)

Cliente para este servidor, sobre Gorilla (WS-14 a WS-17).

- **Conectar:** `NewClient()` crea el cliente sin conectar; `Connect(url)` marca, espera el mensaje `connected` (hasta 10 s) y guarda el id que asigna el servidor (`ID()`). Si el servidor cierra con `1008`, `Connect` retorna `MSG_WS_REJECTED` con el motivo; si el upgrade falla, el error incluye el código HTTP. Con conexión abierta, `MSG_WS_ALREADY_CONNECTED`. Al aceptarse, llama a las funciones de `OnConnect` antes de retornar.
- **Operaciones:** `Subscribe`, `SubscribeQueue`, `Unsubscribe` y `Publish` envían el mensaje del §3.1 con un `id` propio y esperan su `ack` o `error` (hasta 10 s, `MSG_WS_TIMEOUT`). El error del servidor se retorna con su texto. `Publish` retorna `delivered`. Se pueden llamar desde varias goroutines a la vez.
- **Suscripciones:** `Subscriptions()` lista las confirmadas por el servidor. Al desconectarse se vacía: el servidor las olvidó, y al reconectar hay que volver a suscribirse.
- **Recepción:** una goroutine lee del servidor y entrega los `ack`/`error` a quien los espera; los `message` y `direct` pasan, en orden, a otra goroutine que llama a `OnMessage` (con `Message{Channel, Queue, From, Data}`) y a `OnDirect`. Así una función lenta no frena las confirmaciones, y una función puede llamar a `Subscribe` o `Publish`. Si 256 mensajes esperan a sus funciones, la lectura se detiene (y el servidor puede desconectar al cliente por lento).
- **Desconectar:** `Close()` envía un cierre normal y cierra la conexión; no espera a `OnDisconnect`, que llega después. Cuando la conexión termina (por cualquier lado), las operaciones en espera retornan el motivo, se entregan los mensajes pendientes y luego se llama a `OnDisconnect` con `(client, err)`. Desde ahí se puede volver a llamar a `Connect`.
- Las funciones `On…` se agregan, se llaman en orden y un panic en ellas se registra sin cortar las demás.

## 14. Verificación

`test/websocket/test.go` prueba cada requisito con clientes Gorilla reales y verifica que todos los métodos públicos estén cubiertos; el resultado de cada requisito se refleja en `brecha.md`.

```bash
go run -race ./test/websocket   # resultados en test/websocket/result/
```
