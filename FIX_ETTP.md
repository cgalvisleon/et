# FIX_ETTP — Revisión del paquete `ettp/v2/`

## Bugs / Correctness

| # | Archivo | Línea | Problema | Severidad |
|---|---------|-------|----------|-----------|
| 1 | `server.go` | 66–68 | **`Solvers`, `Packages`, `router` sin mutex**: los tres maps son accedidos y modificados desde handlers HTTP y eventos NATS concurrentemente sin ningún lock → data race garantizada. | **Crítica** |
| 2 | `server.go` | 437–450 | **`RemoveRouterById` modifica `Solvers` sin lock**: `delete(s.Solvers, id)` directamente sin ninguna sincronización. | **Crítica** |
| 3 | `server.go` | 535–543 | **`Reset` reemplaza tres maps sin lock**: `router`, `Solvers` y `Packages` reinicializados sin mutex mientras pueden estar siendo leídos por handlers activos. | **Crítica** |
| 4 | `server.go` | 475 | **`300 * time.Microsecond` en `FindResolver`**: casi con seguridad debería ser `Millisecond`. 300 µs es menor que la latencia de red — el timer expira antes de que el request pueda terminar. | **Alta** |
| 5 | `handler.go` | 167–176 | **`setCookie` sobrescribe cookies**: cuando ya existe `Set-Cookie`, usa `Header.Set` en lugar de `Header.Add` → solo sobrevive la última cookie de la respuesta upstream. | **Alta** |
| 6 | `handler.go` | 135 | **Proxy request sin context**: `http.NewRequest` no propaga el context del request original → si el cliente se desconecta, el request al upstream sigue vivo hasta que expira el timeout del cliente. | **Alta** |
| 7 | `server.go` | 332–337 | **`getRequest` usa `Lock` en lugar de `RLock`**: solo lee `Requests` pero bloquea escrituras innecesariamente. | **Media** |
| 8 | `server.go` | 356–366 | **`listRequests` usa `Lock` en lugar de `RLock`**: solo copia el map pero bloquea escrituras. | **Media** |

---

## Performance

| # | Archivo | Línea | Problema | Impacto |
|---|---------|-------|----------|---------|
| 9  | `handler.go` | 39, 53 | **`Save()` en cada request**: `HTTPError` y `HTTPSuccess` llaman `s.Save()` que serializa y escribe a Redis/disco en cada respuesta. Costoso bajo carga alta. | **Alto** |
| 10 | `router.go` | 98 | **Regex compilado en cada `set()`**: `regexp.MustCompile(...)` se ejecuta en cada registro de ruta. Debería ser una variable de paquete inicializada una sola vez. | **Medio** |
| 11 | `server.go` | 70, 122 | **`mu` como `map[string]*sync.Mutex`**: un map lookup en cada operación lock/unlock. Debería ser un campo `muRequests sync.RWMutex` directo en el struct. | **Medio** |
| 12 | `server.go` | 471–477 | **Timer en `FindResolver` no cancelable**: `time.AfterFunc` sin `*time.Timer` guardado — si el request termina antes, el timer sigue vivo y llama `deleteRequest` innecesariamente. | **Medio** |
| 13 | `routes.go` | 138, 151 | **`r.URL.Query()` parseado dos veces en `getRouter`**: dos llamadas independientes que parsean el mismo query string. | **Bajo** |
| 14 | `server.go` | 206 | **`time.Sleep(3s)` en `banner()`**: bloquea el goroutine 3 segundos en cada arranque sin ningún propósito técnico. | **Bajo** |

---

## Code Quality

| # | Archivo | Línea | Problema | Impacto |
|---|---------|-------|----------|---------|
| 15 | `handler.go` | 149–165 | **`setHeader` escribe header vacío**: si todos los valores de un header están excluidos, `joinedValues` queda `""` y se escribe igualmente con `Header.Set(key, "")`. | **Medio** |
| 16 | `routes.go` | 197–201 | **`upsetRouter` aborta el batch en el primer error**: si falla una ruta de 100, las 99 restantes no se procesan y no hay rollback parcial ni informe de errores por ítem. | **Medio** |
| 17 | `storage.go` | 175 | **`file.Write` error ignorado silenciosamente**: cuando `storage == nil` y se escribe el archivo inicial, el error de `file.Write` no se maneja. | **Bajo** |
| 18 | `routes.go` | 239 | **Typo `getPakages`**: debería ser `getPackages`. | **Bajo** |

---

## Resumen por archivo

| Archivo | Bugs | Performance | Quality | Total |
|---------|------|-------------|---------|-------|
| `server.go` | 5 | 4 | 0 | **9** |
| `handler.go` | 2 | 1 | 1 | **4** |
| `router.go` | 0 | 1 | 0 | **1** |
| `routes.go` | 0 | 1 | 2 | **3** |
| `storage.go` | 0 | 0 | 1 | **1** |
| **Total** | **7** | **7** | **4** | **18** |
