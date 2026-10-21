# Brecha: `websocket`

Distancia entre el objetivo (`feature.md`, detallado en `spec.md`) y lo implementado. Se actualiza con cada cambio del paquete y cada corrida de `go run -race ./test/websocket` (resultados en `test/websocket/result/`).

## Estado actual

**Objetivo del feature: cumplido.** 17/17 requisitos de `spec.md` verificados; 15/15 casos correctos; 32/32 métodos públicos probados (corrida del 2026-09-27, `-race`, 3 veces seguidas).

| ID | Requisito | Estado | Evidencia (caso en `result.md`) |
|---|---|---|---|
| WS-01 | Usa gorilla/websocket | ✅ | 1, 12 (análisis del código: importa `github.com/gorilla/websocket` v1.5.3) |
| WS-02 | Conectarse (`http.Handler` o `Listen`) | ✅ | 1 (Listen, montado en `http.ServeMux`, 503 tras `Close`), 6 (mensajes inválidos no cortan la conexión) |
| WS-03 | Suscribirse tipo topic | ✅ | 2, 4, 10 |
| WS-04 | Suscribirse tipo queue | ✅ | 3 (30 mensajes: grupo de 3 → 10/10/10, grupo de 2 → 15/15, sin duplicados), 4, 10 |
| WS-05 | Desuscribirse | ✅ | 4 |
| WS-06 | Publicar (servidor y cliente) | ✅ | 2, 3, 5 (`from` del cliente, `delivered`), 10 |
| WS-07 | `OnConnect` (puede rechazar) | ✅ | 7 (token por query; rechazo con cierre 1008 y motivo; suscripciones desde `OnConnect`) |
| WS-08 | `OnDisconnect` | ✅ | 8 (cierre del cliente, `Client.Close`, `Server.Close` que espera), 11 (motivo "consumidor lento") |
| WS-09 | `OnSubscribe` (puede rechazar) | ✅ | 9 (rechazo, tipo y grupo, ampliar suscribiendo a otro canal) |
| WS-10 | `OnUnsubscribe` | ✅ | 8 (orden: unsubscribe de cada canal y luego disconnect), 9 |
| WS-11 | Modelo catalog | ✅ | 12 (todo método público está en `catalog.go` y solo llama a su versión privada) |
| WS-12 | Concurrencia y consumidor lento | ✅ | 10 (20 clientes, 5 publicadores, orden por publicador), 11 (el lento se desconecta, `Publish` no se bloquea, el rápido recibe todo) |
| WS-13 | Panic en eventos | ✅ | 7, 9 |
| WS-14 | Cliente: `NewClient` y `Connect` devuelven un `Client` | ✅ | 12 (rechazo con motivo, `ID()` igual al del servidor, `OnConnect` antes de retornar, error contra servidor cerrado) |
| WS-15 | Cliente: suscribir, queue, desuscribir y publicar | ✅ | 13 (rechazo de `OnSubscribe` como error, `delivered`, `Unsubscribe` confirmado) |
| WS-16 | Cliente: recibir mensajes en orden | ✅ | 13 (500 mensajes en orden con `From`; `OnDirect`; `Subscribe` dentro de `OnMessage` sin bloqueo) |
| WS-17 | Cliente: `OnConnect`, `OnDisconnect` y reconexión | ✅ | 12, 14 (el servidor corta → reconecta desde `OnDisconnect` con otro id; suscripciones limpias) |

## Brecha pendiente

Ningún requisito del feature está pendiente. Lo que sigue **no** lo pide el feature; se lista para decidir los próximos pasos:

| Tema | Situación actual | Para cerrarla |
|---|---|---|
| Reconexión automática del cliente | Se hace a mano desde `OnDisconnect`, y hay que volver a suscribirse. | Opción de reconectar y volver a suscribir sola. |
| Encabezados en `Connect` | El token va en la URL (query); `Connect` no recibe encabezados. | `Connect` con `http.Header` (p. ej. `Authorization`). |
| Integración en el servidor de Josefina | El paquete no está montado en `pkg/server` ni en `internal/server`. | Montar `Server` como ruta (p. ej. `GET /ws`) en el router de `et`, y decidir su ciclo de vida junto con `jdb`. |
| Autenticación | Se puede hacer en `OnConnect` con `Request()`, pero no está conectada a las sesiones de `jdb` ni a `et/claim`. | `OnConnect` que valide el bearer token con `et/claim` y la sesión de `jdb`. |
| Origen (CORS) | Regla por defecto de Gorilla (mismo origen o sin `Origin`); no configurable. | Exponer la configuración de `CheckOrigin`. |
| Persistencia y reintentos | Entrega a lo sumo una vez, en memoria. | Solo si se necesita: guardar mensajes de queue o reintentar. |
| Varios nodos | Los canales viven en un solo proceso. | Puente con `et/event` (NATS) para publicar entre nodos. |
| Comodines en canales | Coincidencia exacta. | Solo si se necesita (p. ej. `sala.*`). |
| Límites configurables | Buffer 256, 1 MiB por mensaje, keepalive 54/60 s: constantes. | Pasarlos a configuración si hace falta ajustarlos. |

## Historial

| Fecha | Cambio | Requisitos cumplidos |
|---|---|---|
| 2026-10-21 | Movido de josefina (`internal/websocket`) a `et/websocket`: mensajes `MSG_WS_*` en `et/msg`, verificador en `et/test/websocket`; reemplaza al `jws` anterior. Sin cambios de API. `go run -race ./test/websocket`: 15/15 casos, 17/17 requisitos, 32/32 métodos. | 17/17 |
| 2026-09-27 | Cliente Go (`NewClient`, `Connect` → `Client`) con WS-14 a WS-17; la conexión del lado del servidor pasa a llamarse `Conn` (cambia la firma de los eventos del servidor); el servidor envía `connected` con el id al aceptar. Spec v3. Primera corrida: 15/15. | 17/17 |
| 2026-09-27 | `feature.md` actualizado con lo implementado (publicar, topic/queue, rechazo en eventos, montaje, cliente lento); spec v2 solo cambia el origen de los requisitos WS-06, WS-07, WS-09 y WS-12. Sin cambios de código. | 13/13 |
| 2026-09-27 | Spec v1 a partir de `feature.md`; implementación inicial; verificador `test/websocket`. Primera corrida: 11/12 casos (falla del verificador al nombrar `ID` → `id`, no del paquete); corregido el verificador. | 13/13 |
