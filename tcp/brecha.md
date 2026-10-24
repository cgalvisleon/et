# Brecha: `tcp` — Hub

Distancia entre el objetivo del Hub (`feature.md`, detallado en `spec.md` §14) y lo implementado. Se actualiza con cada cambio del paquete y cada corrida de `go run -race ./test/tcp` (resultados en `test/tcp/result/`).

## Estado actual

**Objetivo del feature: cumplido.** 14/14 requisitos implementados y verificados, más la seguridad de §7.4 (revisión 2026-10-16 de `feature.md`). Corrida del 2026-10-19 (`-race`): **25/25 casos correctos, 70/70 métodos públicos probados**.

| ID | Requisito | Estado | Evidencia (caso de `test/tcp`) |
|---|---|---|---|
| HUB-01 | `NewHub(id, name)` solo crea | ✅ | 16: no arranca goroutines; UUID con id vacío; id dado |
| HUB-02 | `LoadServer(port)` | ✅ | 16: puerto 0, identidad del Hub, repetido → `MSG_HUB_SERVER_EXISTS`, `Servers()` |
| HUB-03 | `DownServer(port)` | ✅ | 16: cierra las conexiones; repetido → `MSG_HUB_SERVER_NOT_FOUND`; 19: el cliente queda `down` |
| HUB-04 | `OnMessage(srv, conn, msg)` | ✅ | 17: servidor (`srv` ≠ nil) y cliente (`srv` nil, por `SendTo`) |
| HUB-05 | Orden por conexión | ✅ | 17: 500 mensajes en orden; 21: seq 2 llegado antes se entrega tras saltar el seq 1 inválido |
| HUB-06 | `OnConnect` + nombre | ✅ | 17: el servidor ve `nodo-b`, el cliente ve `nodo-a` |
| HUB-07 | `OnDisconnect` | ✅ | 19: una vez por conexión y lado (4 eventos); 20 y 21: motivo propio |
| HUB-08 | Métricas por cliente | ✅ | 18: `messages_in/out`, `bytes_in/out`, `connected_at`, `last_message`; sale al cerrar; 20: `last_beat` |
| HUB-09 | `ConnectTo` / `Disconnect` | ✅ | 19: repetido → ya conectado; una dirección que falla no queda; reconexión con el mismo `*Client`; `Disconnect` repetido → `MSG_HUB_CLIENT_NOT_FOUND` |
| HUB-10 | `Heartbeat` + `Status` | ✅ | 20: `last_beat`, `failures`; un servidor que no contesta se cierra con `MSG_HUB_HEARTBEAT` y queda `down`; `Heartbeat(0)` lo detiene |
| HUB-11 | `Send` / `SendTo` con `Message` | ✅ | 17: tipos y conversiones, `From` del Hub, inválidos → `MSG_HUB_INVALID_MESSAGE`, destinos desconocidos |
| HUB-12 | Timeouts compartidos | ✅ | 20: `<= 0` falla; el cliente ya creado toma el timeout nuevo |
| HUB-13 | `Close()` | ✅ | 21: vacía servidores y clientes; el Hub se reusa |
| HUB-14 | Catalog, concurrencia, panics | ✅ | 21 métodos en la sección Hub de `catalog.go`; 21: panic en `OnMessage` no corta ni la entrega ni la conexión; datos ilegibles → cierre con `MSG_HUB_INVALID_MESSAGE`; todo con `-race` |
| §7.4 | Seguridad: secreto y TLS | ✅ | 22: mismo secreto conecta (`auth` en los dos lados), secreto distinto / cliente sin secreto / servidor sin secreto se rechazan y el servidor no registra la conexión; 23: TLS autofirmado, cliente plano contra TLS, intermediario TLS que reenvía el saludo (falla con secreto, pasa sin él), TLS mutuo con CA, certificado ajeno, `NewTLS` sin llave; 24: dos Hubs con `SetSecurity` (también sobre un servidor ya cargado) y un Hub con otro secreto |

Hallazgos de la implementación (ya resueltos):
- `link.close` siempre dejaba `MSG_TCP_CLOSED` como motivo. Se agregó `closeWith` (privado, spec §7.3) para los motivos del Hub.
- Cerrar un cliente cuya conexión acababa de cerrar el otro lado devolvía "use of closed network connection". `Hub.Close`/`Disconnect` lo ignoran; `Client.Close` no cambia.

## Decisiones tomadas (de `feature.md`)

| Decisión tomada en la spec | Alternativa |
|---|---|
| `OnMessage`/`OnConnect`/`OnDisconnect` del Hub, con `srv` como parámetro. | Funciones por servidor (cambia `LoadServer`). |
| Servidor identificado por puerto; cliente por la dirección de `ConnectTo`. | Por id del servidor remoto (se conoce tras el saludo). |
| Conexión caída queda `down` hasta `Disconnect`; sin reconexión automática. | Reconexión automática con backoff. |
| Heartbeat apagado por defecto; 3 fallas seguidas cierran la conexión. | Encendido por defecto / umbral configurable. |

## Fuera del feature (próximos pasos posibles)

| Tema | Situación | Para cerrarla |
|---|---|---|
| Broadcast | `SendTo` apunta a un id. | `Broadcast(port, msg)` a todas las conexiones. |
| Confirmación de entrega | `Send` retorna al escribir (como §4). | Mensajes con ack sobre `Request`. |
| Certificados que vencen | Con CA, un certificado vencido corta las conexiones nuevas. | Recargar los archivos sin reiniciar. |

## Historial

| Fecha | Cambio | Requisitos cumplidos |
|---|---|---|
| 2026-10-21 | Recarga de certificados (spec v9 §7.4): `NewTLS` con archivos lee el certificado, la llave y la CA en cada handshake y los vuelve a leer cuando cambian (fecha o tamaño); un archivo roto deja el último válido. TLS mutuo verificado en `VerifyConnection` contra la CA vigente. Caso 26. `go run -race ./test/tcp`: 26/26 casos, 70/70 métodos. | 14/14 |
| 2026-10-21 | Movido de josefina (`internal/tcp`) a `et/tcp`: mensajes `MSG_TCP_*`/`MSG_HUB_*` en `et/msg`, verificador en `et/test/tcp`; reemplaza al `jtcp` anterior. Sin cambios de protocolo ni de API. `go run -race ./test/tcp`: 25/25 casos, 70/70 métodos. | 14/14 |
| 2026-10-19 | Rotación del secreto (spec v7 §7.4): `Security.Previous`, `kids` en el primer saludo y `pickSecret` en el servidor; el cliente valida la prueba con cualquiera de sus secretos. Caso 25 (cinco combinaciones que entran, tres que no). Sale de "Fuera del feature". | 14/14 |
| 2026-10-16 | Seguridad (spec v6 §7.4, `security.go`): secreto compartido con prueba HMAC-SHA256 en un saludo de tres tramas, ligada a la sesión TLS (RFC 5705) y a los dos ids; TLS 1.3 con `NewTLS` (autofirmado en memoria, o archivos con CA para TLS mutuo); `SetSecurity` en `Client`, `Server` y `Hub`; `tls`/`auth` en `ToJson`. Casos 22–24. Contraprueba: sin el canal en la prueba, el caso 23 falla (el intermediario TLS pasa). 24/24, 70/70. | 14/14 + §7.4 |
| 2026-10-16 | Limpieza: se quitan de "Fuera del feature" la replicación del store y la integración en el servidor, que ya hace `internal/node` (etapa 4 de la raíz). Sin cambios de código. | 14/14 |
| 2026-10-12 | Defecto encontrado al montar el hub en el clúster: un `Request` con JSON sin `type` se tomaba por latido y nunca llegaba al `OnRequest` del usuario. Un latido ahora exige `from.id` (spec v5 §14.3); el caso 21 lo cubre. 21/21. | 14/14 |
| 2026-10-12 | Hub implementado (`hub.go`, `message.go`, sección Hub de `catalog.go`, `closeWith` en `link`/`Client`); 6 casos nuevos en `test/tcp` (16–21) y 21 métodos en `publicAPI`. Spec v4. Primera corrida: 20/21 (el caso 20 comparaba `Status()` en el orden de un mapa); luego una carrera en `Close` (ver hallazgos). 15 corridas seguidas en verde. | 14/14 |
| 2026-09-29 | `feature.md` revisado y complementado; spec v3 con §14 (HUB-01…HUB-14); primera medición de la brecha. Base verificada: 15/15 casos, 45/45 métodos. Sin cambios de código. | 0/14 (5 parciales) |
