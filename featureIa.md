# Feature: paquete `ia/` — Rag

## Contexto

Se busca implementar en la carpeta `ia/` (vacía actualmente) una implementacion RAG que permita utilizar funtes de datos de archivos que se carguen como PDF, Holas de calculo de Excel, Documneto de Word, CSV, TXT, MD, SQL a bases de datos, este RAG debe ser Multitenant, quiere decir que en la base verstorial debe existere un campo TenenatID y ProjectID, para poder diponibilizar en varios proyectos. la implementacion debe incluir en la carpeta `cmd/ia` una API que use la carpeta ia usando el packete server de la libreria et. el rag debe tener un structu de configuracion para api de IA requerida para el embedidn y la consulta, tambien debe incluirse un enpoint /conversation par ainter actuar con el RAG.

## Plan

### Decisiones de diseño

1. **Persistencia**: `jsql` (ya usado en todo el repo) en vez de un `Store` genérico tipo `jia`/`jwf`. El RAG necesita filtrar/consultar por `tenant_id` + `project_id`, y `jsql` ya trae esas dos constantes de columna (`jsql.TENANT_ID`, `jsql.PROJECT_ID`) y helpers `DefineTenantModel`/`DefineProjectModel` (que solo cubren un campo cada uno) — se define un helper propio que agrega ambos.
2. **Vector store**: `jsql.EMBEDDING`/`VECTOR` (postgres) existe solo como mapeo de tipo DDL (`jsql/drivers/postgres/types.go`); no hay marshalling de valores, operador de similitud, ni `CREATE EXTENSION vector` en ningún lado del repo, y **no** hay ningún ejemplo de uso. Implementarlo en serio (pgvector + operadores `<=>`) sería una apuesta no verificable en este entorno (no hay Postgres corriendo). Decisión: el embedding se guarda como columna `JSON` (arreglo de floats) y la búsqueda por similitud (coseno) se calcula en Go, trayendo los chunks candidatos del tenant/proyecto (acotados por `DB_RECORD_LIMIT`, default 1000) y rankeando en memoria. Es portable entre `postgres` y `sqlite` (ambos drivers activos en el repo) y queda documentado como limitación conocida, no como bug.
3. **Proveedor de IA**: OpenAI vía `github.com/openai/openai-go/v3` (ya es dependencia del repo, usado en `jia/`). Se reusa el mismo patrón de `jia/agent.go` (`openai.NewClient(option.WithAPIKey(key))`) pero con la API clásica `Chat.Completions.New` (no `Responses`/`Conversations` de `jia`, porque acá el historial de conversación se persiste en las propias tablas `jsql` multitenant, no en el estado del lado de OpenAI).
4. **Ingesta de archivos**:
   - TXT, MD: texto plano tal cual.
   - CSV: reusa el paquete `csv/` del repo (`csv.ReadCsv`).
   - XLSX: usa `excelize/v2` directamente (ya es dependencia; `xls.XlsReader` no expone la lista de hojas, así que se evita tocar ese paquete).
   - SQL: ejecuta un query contra un `*jsql.DB` (`db.Sql(query)`) y vuelca las filas como texto.
   - DOCX: extractor propio mínimo con `archive/zip` + `encoding/xml` sobre `word/document.xml` (sin dependencia nueva).
   - PDF: nueva dependencia `github.com/ledongthuc/pdf` (fork Go puro de `rsc.io/pdf`, `GetPlainText()`), agregada con `go get` (verificado que resuelve contra el proxy de Go).
5. **Chunking**: por palabras, con tamaño y solapamiento configurables (`Config.ChunkSize`/`ChunkOverlap`).
6. **API HTTP**: `cmd/ia/main.go` usa `server.New(name, port)` (paquete `server/`, tal como pide el feature — no `ettp/v2`), conecta `jsql.Load()` y monta las rutas de `ia.LoadRouter`.
7. **Endpoints**: `POST /documents` (multipart, ingesta de archivo), `POST /documents/sql` (ingesta de fuente SQL), `GET /documents` (listado), `DELETE /documents/{id}`, `POST /conversation` (pregunta → respuesta RAG con fuentes), `GET /conversation/{id}` (historial).
8. **Convenciones del repo**: comentarios estilo `/** ... @param ... @return ... **/` (igual que `jwf`/`jia`/`stores`), `msg.go` local i18n (`LANG` env var, igual que `jia/msg.go`), patrón de handler HTTP documentado en `CLAUDE.md` (`request.URLParam`/`request.GetBody`, `response.ITEM`/`response.HTTPError`).

### Modelo de datos (jsql, todas con `tenant_id` + `project_id`)

- `ia_documents`: id, tenant_id, project_id, name, source (pdf/docx/xlsx/csv/txt/md/sql), status, chunk_count, _source (metadata: filename, content_type, query sql).
- `ia_chunks`: id, tenant_id, project_id, document_id, idx, content, embedding (JSON []float64).
- `ia_conversations`: id, tenant_id, project_id, user_id, title.
- `ia_messages`: id, tenant_id, project_id, conversation_id, role, content, sources (JSON).

### Archivos a crear

- `ia/msg.go`, `ia/config.go`, `ia/store.go` (definición de modelos + `Rag`/`Load`), `ia/chunk.go`, `ia/similarity.go`, `ia/client.go` (embeddings + chat OpenAI), `ia/loader.go` + `ia/loader_csv.go` + `ia/loader_xlsx.go` + `ia/loader_docx.go` + `ia/loader_pdf.go` + `ia/loader_sql.go`, `ia/document.go`, `ia/conversation.go`, `ia/handler.go`, `ia/router.go`.
- `cmd/ia/main.go`.
- `go.mod`/`go.sum`: + `github.com/ledongthuc/pdf`.

### Fuera de alcance (documentado, no implementado)

- Prueba en vivo contra OpenAI real (requiere `OPENAI_API_KEY` y red hacia OpenAI; no disponible en este entorno). El código sigue el patrón ya probado de `jia/agent.go`.
- pgvector / `CREATE EXTENSION vector` (ver decisión #2).
- UI / frontend.

## Ejecución

- [x] Investigación de primitivas del repo (`jsql` Define/Query/Command, `server.Ettp`, `openai-go/v3` Chat Completions, `csv/`/`xls/` readers, patrón `jia/agent.go`).
- [x] `go get github.com/ledongthuc/pdf` (resuelto contra proxy.golang.org).
- [x] `ia/msg.go`, `ia/config.go`.
- [x] `ia/store.go` (modelos jsql multitenant + `Rag`/`Load`; agrega también `embedFn`/`answerFn` como puntos de inyección para poder testear sin red).
- [x] `ia/chunk.go`, `ia/similarity.go`.
- [x] `ia/client.go` (Embed + Chat contra OpenAI, API clásica `Chat.Completions`).
- [x] Loaders: `ia/loader.go`, `_csv`, `_xlsx`, `_docx`, `_pdf`, `_sql`.
- [x] `ia/document.go` (ingesta: loader → chunk → embed → persistir).
- [x] `ia/conversation.go` (`Ask`: embed pregunta → similitud → prompt con contexto → chat → persistir mensaje).
- [x] `ia/handler.go` + `ia/router.go`.
- [x] `cmd/ia/main.go` — reemplaza los `cmd/ia/main.go`/`handler.go` que ya estaban borrados (sin commitear) al iniciar esta tarea: en HEAD referenciaban un `ia.Engine`/`Learn`/`Revise`/`Verify`/`Unload`/`IsLoaded` de un intento anterior cuyo paquete `ia/` nunca llegó a existir en el repo (build roto en HEAD); no se recupera ese diseño porque no coincide con el `featureIa.md` vigente (que pide el paquete `server/`, no `ettp/v2`).
- [x] `gofmt -w .`, `go build ./...`, `go vet ./...` (repo completo, limpio).
- [x] Prueba funcional end-to-end (`ia/ia_test.go`, sin llamadas reales a OpenAI vía `embedFn`/`answerFn` sustituidos) contra `sqlite` embebido (`jsql/drivers/sqlite`, driver nuevo en el repo, sin uso previo): ingesta multi-tenant, ranking por similitud coseno y aislamiento tenant/project — `go test ./ia/...` en verde (9 tests). También cubre loaders `csv`/`xlsx`/`docx` con fixtures sintéticos y `chunkText`/`cosineSimilarity`.
- [x] **Bug encontrado y corregido en el camino** (`et/json.go`, `Json.ScanRows`): el driver `sqlite` (recién agregado, sin ejemplos de uso en el repo) devuelve las columnas JSON como `string` en vez de `[]byte`, así que `ScanRows` nunca las decodificaba — cualquier fila con columnas JSON venía como `{"result": "<json crudo>"}` en vez de aplanarse. Se agregó un `case string:` que solo intenta decodificar cuando el valor luce como objeto/arreglo JSON (`{`/`[`), replicando la lógica ya existente para `[]byte` pero sin arriesgar columnas de texto plano (`"123"`, `"true"`) — cambio acotado, con `go test ./...` del repo completo verificando que nada más se rompió. El doble-encodeo del propio campo `embedding` (columna JSON anidada dentro del objeto-fila que arma el driver sqlite) se resolvió aparte, dentro de `ia/similarity.go` (`chunkEmbedding`), sin tocar más el driver — no se intentó arreglar `jsql/drivers/sqlite/query.go` por ser un cambio de mayor alcance y riesgo no pedido por este feature.
- [x] Actualizar `et/CLAUDE.md` con la nueva sección `ia/` y la corrección del estado del driver `sqlite` (ya existe, a diferencia de lo que decía la nota anterior).

### Cómo correr `cmd/ia`

```bash
export DB_DRIVER=postgres DB_HOST=... DB_NAME=... DB_USER=... DB_PASSWORD=...
export OPENAI_API_KEY=sk-...
export PORT=3300              # opcional, default 3300
export IA_SCHEMA=public       # opcional, default public
go run ./cmd/ia
```

Endpoints: `POST /documents` (multipart `file` + `tenant_id`/`project_id`/`user_id`), `POST /documents/sql` (JSON `tenant_id`/`project_id`/`name`/`query`/`user_id`), `GET /documents?tenant_id=..&project_id=..`, `DELETE /documents/{id}?tenant_id=..&project_id=..`, `POST /conversation` (JSON `tenant_id`/`project_id`/`conversation_id`/`user_id`/`question`), `GET /conversation/{id}?tenant_id=..&project_id=..`.

### Limitaciones conocidas (documentadas, no bugs)

- Búsqueda por similitud en memoria (Go), acotada por `DB_RECORD_LIMIT` (default 1000 chunks por tenant/proyecto) — ver decisión de diseño #2. Para un volumen mayor, la migración natural es pgvector + un operador de similitud nativo en el driver postgres.
- No se probó contra OpenAI real (sin `OPENAI_API_KEY` ni red en este entorno); el código de `ia/client.go` sigue el patrón ya usado en producción por `jia/agent.go`.
