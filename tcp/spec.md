# Especificación: `tcp`

Este documento es la referencia del paquete `tcp`: define el protocolo en la conexión y lo que garantiza cada operación. Ante una diferencia con el código, se corrige uno de los dos en el mismo cambio.

**Regla de mantenimiento:** todo cambio en `tcp` que altere el protocolo, la API pública o un comportamiento descrito aquí actualiza este archivo en el mismo cambio, junto con `brecha.md` (estado de los requisitos del Hub) y, si cambia la API pública, `test/tcp/test.go` (lista `publicAPI` y casos).

Las secciones 1–13 describen el cliente, el servidor y los streams; la §14, el Hub (`hub.go`, `message.go`). Todo está implementado.

Versión del documento: 9 — 2026-10-21 (§7.4: `NewTLS` vuelve a leer el certificado, la llave y la CA cuando cambian los archivos, sin reiniciar; versión 8: el paquete pasa de josefina `internal/tcp` a `github.com/cgalvisleon/et/tcp`, con sus mensajes en `et/msg` y el verificador en `test/tcp`; sin cambios de protocolo ni de API; versión 7: §7.4 rotación del secreto: `Security.Previous`, `kids` en el primer saludo, el servidor prueba con el secreto que los dos conocen; versión 6: §7.4 seguridad: secreto compartido con autenticación mutua en el saludo, ligada al canal TLS; TLS 1.3 opcional, autofirmado o con CA; `SetSecurity` en `Client`, `Server` y `Hub`, `NewTLS`; `tls`/`auth` en `ToJson`; §12 sin "sin autenticación ni cifrado").

Historial: v5 — 2026-10-12 (§14.3: un request es latido solo si trae `from.id`; antes cualquier JSON sin `type` se tomaba por latido y no llegaba al `OnRequest` del usuario). v4 — 2026-10-12 (§14 Hub implementado: HUB-01…HUB-14; §7.3 cierre con un motivo propio; §14.11.1 detalles).
; v3 — 2026-09-29 (§14 Hub especificado, sin implementar). v2 — 2026-09-28 (identidad: `NewClient(id, name)`, `NewServer(id, name)`, saludo `msgHello` al conectar, `ID`/`Name`/`To`/`ServerID`/`ServerName`, `ToJson`, `SetName`).

## 1. Alcance

- Transporte: TCP. Una conexión lleva, al mismo tiempo y en los dos sentidos, mensajes sueltos (`Send`), solicitudes con respuesta (`Request`) y streams.
- Los dos extremos de una conexión son simétricos: cualquiera puede enviar, pedir, abrir streams y atenderlos. Lo único asimétrico es quién marca (`Client.ConnectTo`) y quién escucha (`Server.Listen`).
- Del lado del servidor, cada conexión aceptada es un `*Client` (llamado aquí **conexión**), con los mismos métodos de envío que un cliente. Una conexión representa al cliente remoto: su `ID()` y `Name()` son los que ese cliente envió al saludar (§7.0).
- Los datos son `[]byte` opacos; el paquete no interpreta su contenido.

## 2. Formato de trama

Cada mensaje en la conexión es una trama:

```
+---------+---------+---------+------------------+
| Len (4) | Type(1) | ID (8)  | Data (Len bytes) |
+---------+---------+---------+------------------+
```

| Campo | Tamaño | Codificación | Significado |
|---|---|---|---|
| `Len` | 4 bytes | uint32 big-endian | Longitud de `Data`. Máximo `64 MiB` (`64 << 20`). |
| `Type` | 1 byte | bits 0–6: tipo; bit 7: `toOpener` | Ver §3. |
| `ID` | 8 bytes | uint64 big-endian | Correlación (solicitudes y streams). `0` en `msgSend`. |
| `Data` | `Len` bytes | opaco | Contenido según el tipo. |

- Encabezado fijo de 13 bytes.
- Cada trama se escribe completa con una sola escritura, bajo un lock por conexión: tramas de goroutines distintas nunca se intercalan.
- **Recepción inválida:** si `Len` supera el máximo (`MSG_DATA_TOO_LARGE`) o el tipo (sin el bit 7) no está entre 1 y 13 (`MSG_TCP_INVALID_FRAME`), el receptor cierra la conexión.

## 3. Tipos de trama

| Valor | Nombre | Enviada por | `ID` | `Data` |
|---|---|---|---|---|
| 1 | `msgSend` | cualquiera | 0 | mensaje |
| 2 | `msgRequest` | quien pide | id de la solicitud | mensaje |
| 3 | `msgResponse` | quien atiende | id de la solicitud | respuesta |
| 4 | `msgError` | quien atiende | id de la solicitud | texto del error |
| 5 | `msgOpenSend` | quien abre | id del stream | vacío |
| 6 | `msgOpenRequest` | quien abre | id del stream (= id de la respuesta) | vacío |
| 7 | `msgOpenReceive` | quien abre | id del stream | mensaje |
| 8 | `msgData` | escritor del stream | id del stream | un trozo |
| 9 | `msgEnd` | escritor del stream | id del stream | vacío |
| 10 | `msgFail` | escritor del stream | id del stream | texto del error |
| 11 | `msgAck` | lector del stream | id del stream | vacío |
| 12 | `msgCancel` | lector del stream | id del stream | motivo |
| 13 | `msgHello` | cada lado, al conectar | 0 | `{"id","name"}` en JSON (§7.0) |

`msgResponse` y `msgError` responden tanto a `msgRequest` como a `msgOpenRequest`.

### 3.1 IDs y el bit `toOpener`

- Cada extremo numera lo que **inicia** (solicitudes y streams) con un contador propio que empieza en 1 y crece de a uno. Los dos extremos usan el mismo rango, así que un mismo número puede designar a la vez una solicitud o stream de cada lado.
- `msgResponse`/`msgError` siempre se interpretan en el espacio de IDs de quien las **recibe** (respuestas a sus propias solicitudes).
- En las tramas de stream (5–12), el bit 7 de `Type` (`toOpener`) indica el dueño del ID:
  - `toOpener = 0`: la trama va hacia el extremo que **no** abrió el stream (el ID es del emisor). Las tramas de apertura (5–7) siempre llevan 0.
  - `toOpener = 1`: la trama va hacia el extremo que **abrió** el stream (el ID es del receptor).
- El receptor busca el stream entre los que abrió él (`toOpener = 1`) o entre los que abrió el otro (`toOpener = 0`). Así los streams de ambos lados nunca se confunden.

## 4. Mensajes sueltos: `Send`

1. El emisor escribe `msgSend`.
2. El receptor llama, **en una goroutine nueva por mensaje**, a cada función de `OnSend` en el orden en que se agregaron, con `(conn, message)`.

- No hay confirmación: `Send` retorna al escribir la trama.
- **Orden:** las tramas llegan en el orden en que se escribieron, pero cada mensaje se atiende en su propia goroutine, así que **el orden en que las funciones procesan mensajes sucesivos no está garantizado**. Para datos que deben procesarse en orden, usar un stream.
- Sin funciones de `OnSend`, el mensaje se descarta.

## 5. Solicitudes: `Request`

1. Quien pide toma un ID nuevo, registra una espera para él y escribe `msgRequest`.
2. Quien atiende prueba, en una goroutine nueva por solicitud, las funciones de `OnRequest` en orden:
   - retorna `error` → responde `msgError` con `err.Error()`;
   - retorna respuesta distinta de `nil` → responde `msgResponse`;
   - retorna `(nil, nil)` → pasa a la siguiente función;
   - hace panic → responde `msgError` con `MSG_TCP_HANDLER_PANIC` y lo registra en el log;
   - ninguna responde → `msgError` con `MSG_TCP_NO_HANDLER`.
3. `Request` retorna la respuesta, o `errors.New(texto)` si llegó `msgError`.

- El error remoto viaja como texto: quien pide no recibe el mismo valor de error (`errors.Is` no aplica).
- Termina antes con `MSG_TCP_TIMEOUT` (vence el timeout) o con el motivo de cierre (se cae la conexión).
- Una respuesta que llega tarde, repetida o para un ID desconocido se descarta.

## 6. Streams

Un stream es una secuencia ordenada de trozos de un **escritor** a un **lector**, con fin explícito. En cada extremo un stream tiene un solo papel.

| Operación | Abre | Escritor | Lector | Respuesta |
|---|---|---|---|---|
| `SendStream()` | `msgOpenSend` | quien abre | quien atiende (`OnSendStream`) | ninguna |
| `RequestStream()` | `msgOpenRequest` | quien abre | quien atiende (`OnRequestStream`) | una, con `CloseAndReceive` |
| `ReceiveStream(m)` | `msgOpenReceive` + `m` | quien atiende (`OnReceiveStream`) | quien abre | es el stream |

### 6.1 Trozos y fin

- El escritor envía cada trozo como `msgData` y termina con `msgEnd` (normal) o `msgFail` (con error). Solo el primer cierre tiene efecto.
- El lector recibe los trozos **en el orden en que se escribieron**. Tras `msgEnd`, `Read` retorna `io.EOF`; tras `msgFail`, retorna `errors.New(texto)`. Los trozos que ya llegaron se entregan antes del fin.
- `Write` después de cerrar retorna `MSG_TCP_STREAM_CLOSED`.

### 6.2 Control de flujo

- Ventana fija de **32 trozos** por stream.
- El escritor empieza con 32 créditos; cada `msgData` consume uno. Sin créditos, `Write` espera (hasta el timeout).
- Cada `Read` que entrega un trozo envía `msgAck`, que devuelve un crédito.
- El lector nunca tiene más de 32 trozos sin leer, así que recibir no bloquea la lectura de la conexión: un lector lento frena solo a su escritor, y el resto del tráfico sigue.
- Si el escritor envía más trozos de los permitidos, el lector cancela el stream con `MSG_TCP_INVALID_FRAME`.

### 6.3 Cancelación

- El lector puede dejar de leer (`StreamReader.Close`, o al retornar la función que atiende): envía `msgCancel` con el motivo y descarta lo que llegue después.
- El escritor, al recibir `msgCancel`, falla sus `Write` pendientes y siguientes con ese motivo (`MSG_TCP_STREAM_CANCELED` en una cancelación normal).
- Un `msgData` para un stream que el receptor no tiene (ya terminó o fue cancelado) se contesta con `msgCancel` (`MSG_TCP_STREAM_CLOSED`). Las demás tramas para streams desconocidos se descartan.

### 6.4 Quien atiende

Cada tipo de stream tiene **una sola función** (`OnSendStream`, `OnRequestStream`, `OnReceiveStream`); definirla de nuevo reemplaza a la anterior. Se ejecuta en una goroutine nueva por stream.

| Stream | Sin función | La función retorna | Panic |
|---|---|---|---|
| `SendStream` | `msgCancel` con `MSG_TCP_NO_HANDLER` | se cancela lo no leído | se registra; se cancela lo no leído |
| `RequestStream` | `msgError` `MSG_TCP_NO_HANDLER` + `msgCancel` | `msgResponse` (respuesta, vacía si es `nil`) o `msgError`; luego se cancela lo no leído | `msgError` `MSG_TCP_HANDLER_PANIC` |
| `ReceiveStream` | `msgFail` `MSG_TCP_NO_HANDLER` | `nil` → `msgEnd`; error → `msgFail` | `msgFail` `MSG_TCP_HANDLER_PANIC` |

- `CloseAndReceive` (solo `RequestStream`) envía `msgEnd` y espera la respuesta hasta el timeout. En otros streams retorna `MSG_TCP_NOT_REQUEST_STREAM`.
- Un `RequestStream` debe terminar con `CloseAndReceive`; si no, su registro vive hasta que se cierra la conexión.

## 7. Conexión

### 7.0 Identidad y saludo

- `NewClient(id, name)` y `NewServer(id, name)` fijan la identidad al crear; con `id` vacío se genera un UUID (`reg.GetUUID`). El id no cambia después (tampoco al volver a llamar a `Listen` o `ConnectTo`). El nombre del servidor se puede cambiar con `SetName`; las conexiones que se acepten después lo reciben actualizado.
- **Saludo:** al conectar, antes de cualquier otra trama, el cliente envía `msgHello` con su `{"id","name"}` y el servidor responde `msgHello` con el suyo. Cada lado espera el saludo del otro hasta el timeout. Con secreto (§7.4) el saludo lleva además `nonce` y `proof`, y hay un tercer `msgHello` del cliente.
  - Si el saludo no llega a tiempo o la primera trama no es `msgHello`, se cierra la conexión con `MSG_TCP_HANDSHAKE`: `ConnectTo` retorna ese error; el servidor no la registra, no llama a `OnConnect` ni cuenta en `Count`.
  - Un `msgHello` después del saludo se ignora.
- Resultado:

  | | `ID()` / `Name()` | `To()` | `ServerID()` / `ServerName()` |
  |---|---|---|---|
  | Cliente que marcó | los suyos | dirección del servidor | los del servidor (del saludo) |
  | Conexión en el servidor | los del cliente (del saludo) | dirección del cliente | los del servidor |

  Así las funciones del servidor (`OnConnect`, `OnSend`, …) saben con `conn.ID()`/`conn.Name()` qué cliente envió cada mensaje. El servidor no verifica que los ids sean únicos.
- `To`, `ServerID` y `ServerName` quedan con los valores de la última conexión aunque se cierre; antes de la primera son `""`.
- `ToJson()` del cliente: `{"id","name","to","server":{"id","name"},"tls","auth"}` (`tls`/`auth`: la última conexión fue por TLS / se autenticó con el secreto, §7.4). Del servidor: `{"id","name","addr","peers":[ToJson de cada conexión]}`; `addr` es `""` si no escucha.

### 7.1 Cliente

- `ConnectTo(addr)`: si ya hay conexión, retorna `MSG_TCP_ALREADY_CONNECTED` sin marcar. Marca con el timeout, saluda (§7.0), empieza a leer y llama a las funciones de `OnConnect` **en la goroutine de quien llamó, antes de retornar**; en ellas ya se puede usar `Send`, `Request` y streams.
- `Close()`: cierra la conexión y espera a que termine la lectura. Sin conexión, no hace nada.
- Sin conexión, las operaciones de envío retornan `MSG_TCP_NOT_CONNECTED`.
- Tras cerrarse la conexión (por cualquier lado) el cliente queda sin conexión y se puede volver a llamar a `ConnectTo`, también desde `OnDisconnect`.

### 7.2 Servidor

- `Listen(addr)`: si ya escucha, `MSG_TCP_ALREADY_LISTENING`. Acepta en segundo plano. Un error de `Accept` que no sea el cierre se registra en el log y se reintenta a los 10 ms.
- Por cada conexión aceptada, en su propia goroutine (un cliente lento en saludar no frena a los demás): saluda (§7.0); si el saludo falla, se cierra. Si no: se registra (cuenta en `Count`), empieza a leer y se llama a las funciones de `OnConnect` **en otra goroutine**, en paralelo con la lectura: puede llegar un mensaje de esa conexión antes de que `OnConnect` termine.
- `Close()`: deja de aceptar, cierra todas las conexiones (también las que están saludando) y **espera** a que terminen sus lecturas y sus `OnDisconnect`. Después se puede volver a llamar a `Listen`. No se debe llamar desde un `OnDisconnect` del mismo servidor (se esperaría a sí mismo).
- Las conexiones del servidor comparten sus funciones (`On…`) y su timeout: llamar a `On…` o `SetTimeout` sobre una de ellas cambia los del servidor.

### 7.3 Cierre y `OnDisconnect`

- Una conexión termina cuando la lectura falla: el otro lado cerró, se llamó a `Close` en este lado, error de red o trama inválida.
- Motivo (`err`): `MSG_TCP_CLOSED` si fue un cierre (EOF o cierre local); si no, `MSG_TCP_CLOSED: <causa>`.
- Al terminar, en este orden: se cierra el socket; toda espera en curso (`Request`, `CloseAndReceive`, `Read`, `Write` de stream) retorna el motivo; el cliente queda sin conexión (el servidor además la quita de `Count`); se llama a cada función de `OnDisconnect` con `(conn, err)`, una sola vez por conexión, en la goroutine que leía.
- `Client.Close` no espera a que termine `OnDisconnect`; `Server.Close` sí.
- Internamente, `closeWith(reason)` (privado, en `link` y `Client`) cierra igual que `Close`, pero el motivo que recibe `OnDisconnect` de este lado es `reason` y no `MSG_TCP_CLOSED`. El Hub lo usa para `MSG_HUB_HEARTBEAT` y `MSG_HUB_INVALID_MESSAGE` (§14.3, §14.7). El otro lado sigue viendo `MSG_TCP_CLOSED`.

### 7.4 Seguridad

`Security{Secret, Previous, ServerTLS, ClientTLS}` se fija con `SetSecurity` en un `Client`, un `Server` o un `Hub` (que la comparte con todos sus servidores y clientes, también los que cree después). Aplica a las conexiones que se abran o se acepten **después** de llamarlo; las abiertas siguen como estaban. La configuración vacía (por defecto) es el comportamiento anterior: TCP plano y saludo sin prueba.

**TLS:**
- `ServerTLS` (servidor) y `ClientTLS` (cliente): con uno no nulo, la conexión pasa por TLS. El handshake TLS va **antes** del saludo, dentro del mismo timeout, en la goroutine de cada conexión (el servidor no frena `Accept`). Si falla: `MSG_TCP_HANDSHAKE`.
- Si `ClientTLS` verifica certificados y no trae `ServerName`, el cliente usa el host de la dirección de `ConnectTo` (una IP se verifica contra las IP del certificado).
- `NewTLS(certFile, keyFile, caFile)` arma las dos configuraciones con TLS 1.3 como mínimo:
  - sin `certFile` ni `keyFile`: un certificado ECDSA P-256 autofirmado, solo en memoria (cambia en cada arranque); con uno solo de los dos: `MSG_TCP_TLS_KEYPAIR`;
  - sin `caFile`: el cliente no verifica el certificado del servidor (`InsecureSkipVerify`). Cifra, pero **no autentica**: la autenticación queda en el secreto, que sí detecta a un intermediario (ver abajo);
  - con `caFile`: TLS mutuo. El servidor exige un certificado de cliente firmado por esa CA y el cliente verifica el del servidor contra ella. Un archivo sin certificados: `MSG_TCP_TLS_CA`.
  - **Recarga sin reiniciar:** con archivos, las configuraciones no guardan el certificado ni la CA fijos: los piden en cada handshake (`GetCertificate`, `GetClientCertificate`, y la verificación en `VerifyConnection` contra la CA vigente, con el nombre de `ServerName` en el cliente). Si cambió la fecha o el tamaño de alguno de los archivos se vuelven a leer; si no se pueden leer o no sirven, sigue lo último válido. Así se renueva un certificado o se cambia la CA escribiendo los archivos: las conexiones nuevas usan lo nuevo y las abiertas siguen con su sesión. Para cambiar la CA de un clúster sin cortes, el archivo de CA de cada nodo lleva un tiempo las dos (vieja y nueva) mientras se reemplazan los certificados.

**Secreto (autenticación mutua en el saludo):**

```
cliente → servidor   msgHello {id, name, nonce: Nc, kids: [kid(secreto)...]}
servidor → cliente   msgHello {id, name, nonce: Ns, proof: P(server)}     o {error}
cliente → servidor   msgHello {proof: P(client)}
```

- `Nc` y `Ns`: 32 bytes aleatorios por conexión (`crypto/rand`). En JSON viajan en base64.
- `P(dominio) = HMAC-SHA256(secret, dominio ‖ Nc ‖ Ns ‖ canal ‖ idCliente ‖ idServidor)`, cada parte precedida por su largo (4 bytes, big-endian). Los dominios son `josefina-tcp/server` y `josefina-tcp/client`, así una prueba no sirve como la del otro lado.
- **Canal:** con TLS, 32 bytes exportados de la sesión (`ExportKeyingMaterial("EXPORTER-josefina-tcp")`, RFC 5705); sin TLS, vacío. Un intermediario que termina TLS en los dos lados tiene dos sesiones distintas, así que las pruebas que reenvía no validan. Sin TLS, el secreto autentica el saludo pero el tráfico posterior viaja en claro.
- Los ids entran en la prueba: un lado no puede presentarse con otro id que el que usó para probar.
- El secreto nunca viaja. La comparación es en tiempo constante (`hmac.Equal`).
- **Rechazos** (el saludo falla con `MSG_TCP_HANDSHAKE: <motivo>`; el servidor no registra la conexión ni llama a `OnConnect`):

  | Servidor | Cliente | Resultado |
  |---|---|---|
  | con secreto | sin secreto (sin `nonce`) | el servidor responde `{error: MSG_TCP_AUTH_REQUIRED}` y cierra; `ConnectTo` retorna ese motivo |
  | con secreto | otro secreto | el cliente no valida `P(server)`: `ConnectTo` retorna `MSG_TCP_AUTH_FAILED` y avisa al servidor con `{error}` |
  | sin secreto | con secreto | el servidor no envía prueba: `ConnectTo` retorna `MSG_TCP_AUTH_FAILED` |
  | con secreto | `P(client)` inválida | el servidor cierra sin registrarla (solo pasa si alguien fabrica el tercer saludo) |

- **Rotación del secreto** (`Security.Previous`): un lado puede aceptar, además del secreto vigente, los anteriores. `secrets()` = vigente + anteriores, sin vacíos ni repetidos. El cliente envía en el primer saludo el `kid` de cada uno de sus secretos, el vigente primero (`kid(s)` = los 8 primeros bytes de `HMAC-SHA256(s, "josefina-tcp/kid")` en hexadecimal: identifica el secreto sin revelar más que lo que ya revela una prueba). El servidor elige el **primero de sus secretos** cuyo `kid` está en la lista y prueba con él; sin ningún `kid` en común responde `{error: MSG_TCP_AUTH_FAILED}`; un cliente anterior a esta versión (sin `kids`) se trata con el vigente. El cliente verifica `P(server)` con cada uno de sus secretos y responde `P(client)` con el que validó (así un servidor anterior, que prueba con el suyo sin mirar `kids`, sigue entrando). Para cambiar el secreto de un clúster: cada nodo pasa a `{Secret: nuevo, Previous: [viejo]}` de uno en uno (en cada momento todos conversan, con el viejo o con el nuevo); cuando todos lo tienen, se retira `Previous` también de uno en uno. Un lado solo con el viejo y otro solo con el nuevo no conversan.
- Con los dos lados sin secreto, el saludo es el de §7.0 (dos tramas). Un nodo con esta versión y secreto no conversa con uno anterior a la versión 6: hay que actualizar todos juntos.
- Al terminar el saludo, la conexión guarda si fue por TLS y si se autenticó (`tls` y `auth` en `ToJson`).

## 8. Timeout

- Uno por cliente o por servidor (compartido con sus conexiones). Por defecto **10 s**. `SetTimeout(d)` exige `d > 0` (si no, `MSG_TCP_INVALID_TIMEOUT`) y aplica de inmediato, también a conexiones abiertas.
- Se aplica a: marcar (`ConnectTo`), escribir cada trama, esperar la respuesta de `Request` y `CloseAndReceive`, esperar crédito en `StreamWriter.Write` y esperar cada trozo en `StreamReader.Read`.
- En streams el timeout es **por trozo**, no por stream: un stream puede durar lo que haga falta mientras no pase más de un timeout entre trozos.

## 9. Concurrencia

- Todos los métodos de `Client` y `Server` se pueden llamar desde varias goroutines a la vez.
- Una conexión admite a la vez cualquier cantidad de `Send`, `Request` y streams en curso, en los dos sentidos.
- Cada mensaje, solicitud y stream entrante se atiende en su propia goroutine: una función lenta no frena la lectura, y una función puede llamar a `Request` o abrir streams sobre la misma conexión.
- Un `StreamWriter` o `StreamReader` debe usarse desde una sola goroutine: el orden de los trozos es el de las llamadas.
- Las funciones `On…` se pueden agregar o reemplazar en cualquier momento; un mensaje en curso usa las que había cuando empezó a atenderse.

## 10. Panics en funciones

Ningún panic en una función del usuario detiene el proceso ni la conexión. Se registra en el log (`MSG_TCP_HANDLER_PANIC`) y:

- `OnConnect`, `OnDisconnect`, `OnSend`: se sigue con las demás funciones.
- `OnRequest`, `OnRequestStream`, `OnReceiveStream`: el otro lado recibe `MSG_TCP_HANDLER_PANIC` como error.
- `OnSendStream`: se cancela el stream.

## 11. Errores

Textos en `internal/msg`, en inglés o español según `LANG`. Los errores que viajan por la red (`msgError`, `msgFail`, `msgCancel`) llevan el texto en el idioma de quien los envía.

| Constante | Cuándo |
|---|---|
| `MSG_TCP_NOT_CONNECTED` | Operación de envío sin conexión. |
| `MSG_TCP_ALREADY_CONNECTED` | `ConnectTo` con conexión abierta. |
| `MSG_TCP_ALREADY_LISTENING` | `Listen` cuando ya escucha. |
| `MSG_TCP_CLOSED` | Motivo de cierre de la conexión. |
| `MSG_TCP_TIMEOUT` | Venció el timeout de una espera. |
| `MSG_TCP_INVALID_TIMEOUT` | `SetTimeout` con `d <= 0`. |
| `MSG_TCP_NO_HANDLER` | No hay función que atienda la solicitud o el stream. |
| `MSG_TCP_HANDLER_PANIC` | Una función hizo panic. |
| `MSG_TCP_INVALID_FRAME` | Tipo de trama desconocido, o escritor que no respetó la ventana. |
| `MSG_TCP_STREAM_CLOSED` | `Write` tras cerrar; datos para un stream que ya no existe. |
| `MSG_TCP_STREAM_CANCELED` | El lector dejó de leer. |
| `MSG_TCP_NOT_REQUEST_STREAM` | `CloseAndReceive` en un stream que no es `RequestStream`. |
| `MSG_TCP_HANDSHAKE` | El saludo no llegó a tiempo, la primera trama no fue `msgHello`, falló el handshake TLS o la autenticación (§7.4). |
| `MSG_TCP_AUTH_REQUIRED` | §7.4: el servidor tiene secreto y el cliente no. |
| `MSG_TCP_AUTH_FAILED` | §7.4: una prueba no validó, o el cliente tiene secreto y el servidor no. |
| `MSG_TCP_TLS_KEYPAIR` | `NewTLS` con el certificado sin la llave o al revés. |
| `MSG_TCP_TLS_CA` | `NewTLS` con un archivo de CA sin certificados. |
| `MSG_DATA_TOO_LARGE` | Mensaje o trozo de más de 64 MiB. |

## 12. Límites conocidos

- Seguridad apagada por defecto (§7.4): sin `SetSecurity`, TCP plano y sin autenticación. El secreto no se rota sin reconectar, y un certificado autofirmado cambia en cada arranque.
- Sin reconexión automática (se hace desde `OnDisconnect`).
- Sin heartbeat en `Client`/`Server`: una conexión muerta sin cierre TCP se detecta al vencer una escritura o una espera. El heartbeat es del Hub (§14.7).
- Ventana de stream y tamaño máximo de trama fijos.
- Los errores remotos llegan como texto.

## 13. Verificación

`test/tcp/test.go` prueba este comportamiento sobre conexiones reales y verifica que todos los métodos públicos estén cubiertos:

```bash
go run -race ./test/tcp   # resultados en test/tcp/result/
```

Los casos 16–21 cubren el Hub (HUB-01…HUB-14) y sus 21 métodos públicos. Los casos 22–24 cubren §7.4: secreto (aceptación y los tres rechazos), TLS autofirmado, cliente plano contra servidor TLS, intermediario TLS que reenvía el saludo (falla con secreto y pasa sin él, como contraprueba), TLS mutuo con CA y certificado ajeno, y dos Hubs con `SetSecurity`. `brecha.md` registra cada corrida.

## 14. Hub

Objetivo (de `feature.md`): un Hub que agrupa varios servidores y varios clientes TCP de este paquete para llevar tráfico entre nodos (p. ej. replicación), con mensajes tipados (`Message`), métricas por cliente conectado, eventos de conexión y un heartbeat. Se construye **encima** de `Client`/`Server` (§1–§11) sin cambiar su protocolo: los dos extremos de una conexión del Hub son Hubs.

### 14.1 Requisitos

| ID | Requisito | Origen |
|---|---|---|
| HUB-01 | `NewHub(id, name)` solo crea el Hub: no escucha, no conecta y no arranca goroutines. Con `id` vacío se genera un UUID. | feature: NewHub |
| HUB-02 | `LoadServer(port)` crea un `Server` con la identidad del Hub, lo pone a escuchar en `:port` y lo registra. | feature: LoadServer |
| HUB-03 | `DownServer(port)` cierra el servidor (espera sus `OnDisconnect`) y lo quita del Hub. | feature: DownServer |
| HUB-04 | `OnMessage` recibe cada `Message` que llega, con el servidor que lo recibió y la conexión (cliente) que lo envió. | feature: OnMessage por servidor |
| HUB-05 | Los mensajes de una misma conexión llegan a `OnMessage` en el orden en que se enviaron, de a uno. | complemento: replicación |
| HUB-06 | `OnConnect` del Hub con el servidor y la conexión; la conexión trae el id y el **nombre** del cliente. | feature: OnConnect + nombre |
| HUB-07 | `OnDisconnect` del Hub con el servidor, la conexión y el motivo. | feature: OnDisconnect |
| HUB-08 | Métricas por servidor: qué clientes están conectados y, por cliente, desde cuándo y cuántos mensajes/bytes entraron y salieron. | feature: métricas por cliente |
| HUB-09 | `ConnectTo(host:port)` abre un cliente del Hub hacia un servidor; `Disconnect(host:port)` lo cierra. Varios a la vez. | feature: ConnectTo + cerrar |
| HUB-10 | `Heartbeat(interval)` envía latidos por cada cliente del Hub y `Status()` informa el estado de cada uno; una conexión que no contesta se cierra. | feature: Heartbeat |
| HUB-11 | `Send(addr, msg)` (cliente → servidor) y `SendTo(port, clientID, msg)` (servidor → cliente) envían un `Message`; `From` lo pone el Hub. | feature: enviar Message |
| HUB-12 | `SetTimeoutServer`/`SetTimeoutClient` cambian el timeout compartido por todos los servidores / clientes del Hub, también los que se creen después. | feature: timeouts compartidos |
| HUB-13 | `Close()` detiene el heartbeat, cierra todos los clientes y da de baja todos los servidores. | complemento |
| HUB-14 | Métodos públicos en `catalog.go` (sección Hub); todo es seguro bajo concurrencia; un panic en una función del usuario no cae el proceso ni la conexión (§10). | complemento: modelo del paquete |

### 14.2 Tipos

```go
const (
	TypeBeat  = iota // 0: latido; sin valor
	TypeInt          // 1: int64
	TypeFloat        // 2: float64
	TypeText         // 3: string
	TypeJson         // 4: et.Json
)

type Message struct {
	From    et.Json // {"id","name"} del Hub que envía; lo pone el Hub, lo que traiga se reemplaza
	Header  et.Json // metadatos libres; nil viaja como {}
	Type    int
	Message any
}

type Hub struct {
	mu            sync.RWMutex       // protege clients, servers, metrics y el estado del heartbeat
	hubID         string             // fijo desde NewHub
	hubName       string
	clients       map[string]*Client // clave: addr tal como se pasó a ConnectTo
	servers       map[string]*Server // clave: puerto en decimal ("1377")
	timeoutServer *duration          // el mismo puntero en todos los servidores del Hub
	timeoutClient *duration          // el mismo puntero en todos los clientes del Hub
	// + funciones On…, métricas por conexión y estado del heartbeat (privados)
}
```

**Tipo y valor** (lo valida `Send`/`SendTo`; si no cumple, `MSG_HUB_INVALID_MESSAGE` y no se envía):

| `Type` | Valores aceptados al enviar | Valor al recibir |
|---|---|---|
| `TypeBeat` | cualquiera (se ignora) | `nil` |
| `TypeInt` | `int`, `int8…int64`, `uint8…uint32` | `int64` |
| `TypeFloat` | `float32`, `float64` | `float64` |
| `TypeText` | `string` | `string` |
| `TypeJson` | `et.Json`, `map[string]any` | `et.Json` |

Un `Type` fuera de 0–4 también es `MSG_HUB_INVALID_MESSAGE`. `Send`/`SendTo` no aceptan `TypeBeat`: los latidos los envía solo el heartbeat.

### 14.3 En la conexión

- Cada `Message` viaja como **una trama `msgSend`** (§4) cuyo `Data` es JSON:
  `{"seq":n,"from":{"id","name"},"header":{…},"type":t,"message":v}`.
- `seq` (uint64) numera los mensajes de cada conexión **en cada sentido**, empezando en 1 en cada conexión nueva. El receptor entrega a `OnMessage` en orden de `seq`: guarda los que llegan adelantados (los `OnSend` de `tcp` corren en goroutines distintas, §4) y los libera cuando llega el que falta. Como TCP no pierde tramas, el hueco siempre se llena; si la conexión se cierra, lo pendiente se descarta.
- El receptor decodifica números con `UseNumber` y convierte según `type` (tabla §14.2). Un mensaje con `seq` legible pero `type` o valor inválido se registra en el log y se descarta; su `seq` cuenta igual para no detener la entrega. Un `Data` del que no se puede leer `seq` (JSON inválido) rompe el orden de esa conexión: se registra y **se cierra la conexión** con `MSG_HUB_INVALID_MESSAGE` como motivo.
- Los latidos viajan como `msgRequest` (§5) con un `Message` `TypeBeat`; el Hub remoto responde `msgResponse` con otro `TypeBeat` (con su `From`). No usan `seq` y nunca llegan a `OnMessage`. Un `msgRequest` se toma como latido solo si su JSON trae `type` 0, no trae `seq` y trae `from.id`. Cualquier otro request (también un JSON que no tenga `type`) se cede a las funciones `OnRequest` que el usuario agregue al servidor o al cliente.
- Las demás funciones de `tcp` (`Request`, streams) siguen disponibles sobre los mismos `*Client`; el Hub no las usa ni las intercepta.

### 14.4 Servidores

- `LoadServer(port int) (*Server, error)`: crea el servidor con `hubID`/`hubName`, le asigna `timeoutServer` y sus funciones internas, y llama a `Listen(":<port>")`. Con `port = 0` el sistema elige uno libre y la clave es el puerto real (`Addr()`).
  - Puerto ya cargado → `MSG_HUB_SERVER_EXISTS`. Error de `Listen` → se retorna y el servidor no queda registrado.
- `DownServer(port int) error`: `Server.Close()` (espera las lecturas y los `OnDisconnect`, §7.2) y lo quita del mapa. Puerto no cargado → `MSG_HUB_SERVER_NOT_FOUND`. No se debe llamar desde un `OnDisconnect` de ese mismo servidor (§7.2).
- `Servers() []*Server`: los servidores cargados.

### 14.5 Eventos

Todas las funciones son del Hub, se agregan con `On…` y se llaman en el orden en que se agregaron. `srv` indica el lado:

- `srv != nil`: la conexión la aceptó ese servidor del Hub; `conn.ID()`/`conn.Name()` son los del cliente remoto (saludo, §7.0).
- `srv == nil`: la conexión la abrió el Hub con `ConnectTo`; `conn` es su cliente (`conn.To()` es la dirección, `conn.ServerID()`/`ServerName()` el Hub remoto).

| Función | Firma | Cuándo |
|---|---|---|
| `OnMessage` | `func(srv *Server, conn *Client, msg *Message)` | Cada `Message` (no latido) recibido, en orden por conexión (§14.3). Para una misma conexión no se llama en paralelo; conexiones distintas sí. |
| `OnConnect` | `func(srv *Server, conn *Client)` | Al aceptar una conexión (goroutine aparte, §7.2) o al conectar `ConnectTo` (antes de que retorne, §7.1). |
| `OnDisconnect` | `func(srv *Server, conn *Client, err error)` | Una vez por conexión, al cerrarse por cualquier lado, con el motivo (§7.3). |

- Una función lenta de `OnMessage` frena solo la entrega de su conexión; la lectura sigue (los mensajes se acumulan en el Hub).
- Un panic se registra y se sigue con la siguiente función y el siguiente mensaje (§10).

### 14.6 Clientes

- `ConnectTo(addr string) (*Client, error)`: crea un `Client` con `hubID`/`hubName` y `timeoutClient`, conecta y lo registra con clave `addr`.
  - `addr` ya registrado y conectado → `MSG_TCP_ALREADY_CONNECTED`. Registrado pero caído (`down`) → reconecta el mismo `Client`.
  - Si falla la conexión o el saludo, se retorna el error y un `addr` nuevo no queda registrado.
- `Disconnect(addr string) error`: `Client.Close()` y lo quita del mapa (y de `Status`). No registrado → `MSG_HUB_CLIENT_NOT_FOUND`.
- Si la conexión se cae sola, el cliente **queda registrado en estado `down`** hasta `Disconnect` o hasta un nuevo `ConnectTo`. No hay reconexión automática (se hace desde `OnDisconnect` con `srv == nil`).
- `Clients() []*Client`: los clientes registrados, conectados o no.

### 14.7 Heartbeat

- `Heartbeat(interval time.Duration)`: arranca o cambia el latido de **todos** los clientes del Hub (también los que se conecten después). `interval <= 0` lo detiene. Por defecto está apagado.
- En cada intervalo, cada cliente conectado envía un latido (§14.3) y espera respuesta hasta `timeoutClient`.
  - Respuesta: `failures = 0`, se guardan `last_beat` y `latency`.
  - Sin respuesta o error: `failures++`. Con **3** fallas seguidas (`beatMaxFailures`) el Hub cierra la conexión con `MSG_HUB_HEARTBEAT` como motivo: queda `down` y se llama a `OnDisconnect`.
  - Un latido no empieza si el anterior de ese cliente sigue esperando.
- Lado servidor: el Hub responde siempre, tenga o no el heartbeat encendido, y guarda `last_beat` en las métricas de la conexión (§14.8).
- `Status() []et.Json`, uno por cliente registrado:
  `{"addr","server":{"id","name"},"state":"up"|"down","connected_at","last_beat","latency_ms","failures"}`. Sin latidos todavía, `last_beat` es `null` y `latency_ms` 0.

### 14.8 Métricas

- Por conexión aceptada, el Hub lleva: `connected_at`, `messages_in`, `messages_out`, `bytes_in`, `bytes_out` (de los `Data` de mensajes del Hub, sin latidos), `last_message` y `last_beat`.
- `Metrics(port int) (et.Json, error)`: `{"port","addr","count","clients":[{"id","name","addr","connected_at","messages_in","messages_out","bytes_in","bytes_out","last_message","last_beat"}]}`. Puerto no cargado → `MSG_HUB_SERVER_NOT_FOUND`.
- Al cerrarse, la conexión sale de las métricas (después de su `OnDisconnect`).
- `ToJson() et.Json`: `{"id","name","servers":[Metrics de cada uno],"clients":Status()}`.

### 14.9 Envío

- `Send(addr string, msg *Message) error`: por el cliente registrado con `addr`. No registrado → `MSG_HUB_CLIENT_NOT_FOUND`; registrado pero `down` → `MSG_TCP_NOT_CONNECTED`.
- `SendTo(port int, clientID string, msg *Message) error`: a **cada** conexión de ese servidor cuyo `ID()` sea `clientID` (el servidor no verifica ids únicos, §7.0). Puerto no cargado → `MSG_HUB_SERVER_NOT_FOUND`; ninguna conexión con ese id → `MSG_HUB_CLIENT_NOT_FOUND`.
- Los dos validan el mensaje (§14.2), ponen `From` y el `seq` de la conexión, y retornan al escribir la trama (sin confirmación de entrega, como §4). No modifican el `*Message` del llamador.
- Asignar `seq` y escribir la trama ocurren bajo un mismo lock por conexión, para que el orden de `seq` sea el de escritura.

### 14.10 Timeouts y cierre

- `timeoutServer` y `timeoutClient` empiezan en 10 s (§8). `SetTimeoutServer(d)`/`SetTimeoutClient(d)` exigen `d > 0` (`MSG_TCP_INVALID_TIMEOUT`) y aplican de inmediato a todo lo del Hub, porque todos comparten el mismo `*duration`. Llamar `SetTimeout` sobre un servidor o cliente del Hub cambia también el del Hub.
- `Close() error`: detiene el heartbeat, cierra todos los clientes, da de baja todos los servidores y vacía los mapas. Después el Hub se puede volver a usar. Retorna el primer error encontrado.

### 14.11 Errores nuevos (`internal/msg`, inglés/español)

| Constante | Cuándo |
|---|---|
| `MSG_HUB_SERVER_EXISTS` | `LoadServer` con un puerto ya cargado. |
| `MSG_HUB_SERVER_NOT_FOUND` | `DownServer`, `Metrics` o `SendTo` con un puerto no cargado. |
| `MSG_HUB_CLIENT_NOT_FOUND` | `Disconnect` o `Send` con una dirección no registrada; `SendTo` sin conexión con ese id. |
| `MSG_HUB_INVALID_MESSAGE` | `Type` fuera de rango, `TypeBeat` en `Send`/`SendTo`, o valor que no corresponde al tipo; motivo de cierre si llega un mensaje sin `seq` legible. |
| `MSG_HUB_HEARTBEAT` | Motivo de cierre tras 3 latidos sin respuesta. |

### 14.11.1 Detalles de la implementación

- **Errores:** los textos de `MSG_HUB_SERVER_EXISTS` y `MSG_HUB_SERVER_NOT_FOUND` incluyen el puerto, y `MSG_HUB_CLIENT_NOT_FOUND` la dirección o el id.
- **`ConnectTo` registra antes de conectar:** así un `OnConnect` puede usar `Send(addr)`. Si una dirección nueva no conecta, se quita. Un segundo `ConnectTo` a la misma dirección mientras otro está en curso retorna `MSG_TCP_ALREADY_CONNECTED`.
- **Estado por conexión:** `seq`, mensajes pendientes y métricas se crean con el primer evento de la conexión, porque un mensaje puede llegar antes que `OnConnect`. Se descartan después de `OnDisconnect`, y una conexión ya cerrada no lo recrea.
- **Escritura fallida:** si falla por un error de red o de timeout (no por "sin conexión" ni por "mensaje demasiado grande"), el Hub cierra esa conexión, porque una trama a medio escribir dejaría el orden roto. El `seq` solo avanza cuando la escritura sale bien.
- **Cierre sin errores falsos:** `Close` y `Disconnect` no reportan como error un `net.ErrClosed`, que aparece cuando el otro lado cerró la conexión mientras este la cerraba.
- **`Status()`:** `latency_ms` es un número con decimales (milisegundos).

### 14.12 API pública del Hub (sección nueva de `catalog.go`)

`NewHub`, `Hub.ID`, `Hub.Name`, `Hub.ToJson`, `Hub.LoadServer`, `Hub.DownServer`, `Hub.Servers`, `Hub.Metrics`, `Hub.ConnectTo`, `Hub.Disconnect`, `Hub.Clients`, `Hub.Send`, `Hub.SendTo`, `Hub.OnMessage`, `Hub.OnConnect`, `Hub.OnDisconnect`, `Hub.Heartbeat`, `Hub.Status`, `Hub.SetTimeoutServer`, `Hub.SetTimeoutClient`, `Hub.Close` (21 métodos, que se suman a `publicAPI` en `test/tcp/test.go`). `Hub.SetSecurity` (§7.4) va en la misma sección; `Client.SetSecurity`, `Server.SetSecurity` y `NewTLS` en las suyas.
