# Feature: paquete `ia/` — Detección de verdad/mentira local

## Contexto

Se busca implementar en la carpeta `ia/` (vacía actualmente) un modelo de IA en Go que
distinga verdad de mentira a partir de texto escrito, funcionando 100% en local (sin
llamadas a APIs externas). Las "verdades" se van acumulando en una base de conocimiento
que puede cambiar conforme llega más contexto (no son etiquetas estáticas). Esta base de
conocimiento debe vivir en una estructura en memoria eficiente, que permita:

- Cargar y descargar bases de conocimiento según la conversación/contexto en uso.
- Liberar automáticamente el espacio de las que no se usan por más de 1 hora.
- Liberar manualmente una base de conocimiento por selección.

Se integrará como **paquete de librería** (no binario standalone), siguiendo las
convenciones ya usadas en `jia/` y `jwf/` (interfaz `Store`, `New`/`Load`, `msg.go`,
`Storage` respaldado por `jsql`).

`jia/` es un paquete distinto (integración con OpenAI, antes se llamó `ia`) — no hay
conflicto de nombres a nivel de import path, pero conviene no confundir ambos paquetes.

## Hallazgos de la investigación previa

- No existe en el repo una caché con expiración por inactividad reutilizable tal cual:
  - `ephemeral/` (`ephemeral/ephemeral.go`) tiene la idea correcta (TTL deslizante) pero
    su implementación usa **un solo timer compartido para todo el mapa** — con múltiples
    claves solo la última tocada expira correctamente. No reutilizable sin reescritura.
  - `mem/` (`mem/mem.go`, `mem/entry.go`) es la mejor referencia estructural: mapa +
    `sync.RWMutex` + timer por entrada vía `time.AfterFunc`, más `Clear(match)` por regex
    para borrado manual. Pero su TTL es fijo desde la escritura (no por inactividad) y no
    es genérico.
  - `dt/` depende de Redis/cache externo — no sirve para memoria pura en proceso.
  - `race/` y `queue/` no aportan primitivas de mapa con expiración (solo referencias de
    estilo: `race.Value` es un box con mutex, `queue.Queue[T]` es un ejemplo de generics).
- No hay ninguna librería NLP/ML en `go.mod` (ni gonum, ni tokenizers, ni distancia de
  edición, ni sentimiento). Todo el análisis de texto hay que escribirlo desde cero.
  Reutilizables: `strs.RemoveAcents`, `strs.DaskSpace`/`Trim`/`Split` para normalización.
- Patrón de paquetes de aplicación del repo (`jia/`, `jwf/`):
  - Interfaz `Store` local: `Set(collection, id, ownerId string, obj any) error`,
    `Get(collection, id string, dest any) (bool, error)`, `Delete(collection, id string) error`,
    `Query(query et.Json) (et.Items, error)`.
  - `New(tag string, store Store, userId string) (*T, error)` y
    `Load(id string, store Store) (*T, error)` como entrypoints.
  - `msg.go` con constantes `MSG_*` e i18n vía `envar.GetStr("LANG", "en")`.
  - `Storage` respaldado por `jsql` como implementación de referencia del `Store`
    (ver `jwf.DefineStore`, `jwf/store.go`).

## Pasos de implementación

1. **Estructuras base de conocimiento** (`ia/knowledgebase.go`, `ia/store.go`)
   - `Fact`: `ID`, `Statement`, `Normalized` (sin acentos/mayúsculas, vía
     `strs.RemoveAcents`), `Confidence`, `Status` (Active/Superseded/Retracted),
     `Version`, `SupersedesID` (enlaza la versión anterior cuando una verdad cambia con
     más contexto), `CreatedAt`/`UpdatedAt`.
   - `KnowledgeBase`: `map[string]*Fact` por ID (acceso O(1)) + índice invertido
     `map[token][]factID` para no escanear todos los hechos al buscar
     contradicciones/similitud.
   - Interfaz `Store` (igual a la de `jia`).

2. **Manager en memoria con expiración** (`ia/manager.go`) — pieza más delicada
   - `map[string]*KnowledgeBase` + `sync.RWMutex`, con **un `time.AfterFunc` por KB**
     (no uno global, para no repetir el bug de `ephemeral/`).
   - `Load(id)`: si está en memoria, la "toca" (reinicia su timer) y la devuelve; si no,
     la carga desde `Store` y arranca su timer de 1h.
   - `Unload(id)`: persiste vía `Store.Set` y libera la KB de memoria (descarga manual).
   - Al expirar el timer (1h sin uso), ejecuta el mismo camino de `Unload`
     automáticamente.

3. **Utilidades de comparación de texto** (`ia/textsim.go`)
   - Distancia de Levenshtein y similitud Jaccard sobre tokens.
   - Normalización reutilizando `strs.RemoveAcents`, `strs.DaskSpace`/`Trim`/`Split`.

4. **Extracción de features lingüísticas** (`ia/features.go`)
   - Longitud/nº de palabras, conteo de pronombres 1ª vs 3ª persona, palabras de
     duda/certeza (listas en español), nivel de detalle (fechas/números/nombres vía
     regex), y similitud/contradicción contra los hechos existentes en la KB usando el
     índice invertido del paso 1.

5. **Modelo de clasificación** (`ia/model.go`)
   - Regresión logística pura en Go (sin dependencias): `Predict(features []float64) float64`
     (sigmoide), `Train(X, y, epochs, lr)` por descenso de gradiente, serialización de
     pesos a JSON.

6. **Pipeline de entrenamiento** (`ia/dataset.go`)
   - Cargar un dataset público en CSV (recomendado: *Real-life Deception Detection
     Dataset* de Pérez-Rosas et al., o *Deceptive Opinion Spam* de Ott et al.), extraer
     features, split 80/20, entrenar, evaluar accuracy/precision/recall, guardar pesos
     entrenados.

7. **Ensamblar `Classifier` y `Engine`** (`ia/classifier.go`, `ia/ia.go`)
   - `Classifier`: features + modelo → veredicto (`IsTruth`, `Confidence`,
     `ContradictsFactID`).
   - `Engine` (API pública del paquete, patrón `New`/`Load` como en `jia`/`jwf`):
     `Learn(kbId, statement, ctx)` (agrega/actualiza una verdad), `Verify(kbId, statement)`
     (clasifica contra la KB), `Unload(kbId)`.

8. **Persistencia opcional** (`ia/store.go`)
   - `ia.DefineStore(db *jsql.DB, schema string) (*Storage, error)` respaldado por
     `jsql`, siguiendo el mismo patrón que `jwf.DefineStore` (modelo con `id`,
     `owner_id`, `definition BYTES`), para que las KB descargadas se puedan recargar
     después.

9. **Pruebas**
   - Timer de expiración por inactividad (con TTL corto en test), similitud de texto
     (casos conocidos), y accuracy del clasificador sobre un set de validación separado.

10. **GoDoc**
    - Comentarios en todas las funciones exportadas siguiendo el estilo del repo.
