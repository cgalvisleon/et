# NEW_PROJECT_GUIDE.md

> Documento de contexto persistente para asistentes de IA (Claude Code y otros) que van a **crear un proyecto nuevo desde cero** usando `github.com/cgalvisleon/et` como librería primaria.
>
> Diferencia con los demás documentos de este repo: `LIBRARY_CONTEXT.md`/`AI_USAGE_GUIDE.md`/`COMPONENT_CATALOG.md`/`ARCHITECTURE_SUMMARY.md` asumen que ya existe un proyecto que depende de `et` y responden "¿cómo uso tal cosa?". Este documento responde "¿cómo arranco un proyecto que todavía no existe?" — estructura de carpetas, primer `main.go`, primer modelo, primeras variables de entorno.
>
> **Verificado contra el código fuente real de `et`**, contra dos consumidores que compilan hoy (`core-studio/api`: microservicio HTTP multi-tenant; `tick`: CLI), y contra el generador `create/` (ver nota abajo) — cada escenario (`project`, `micro` con/sin `schema`, `modelo`, `rpc`) fue generado y compilado (`go build`/`go vet`/`gofmt -l`) en un harness aislado antes de escribir este documento. Fecha de verificación: 2026-09.

---

## Nota: `create/` / `cmd/create` ya está corregido

Hasta 2026-09 el generador de proyectos que trae este repo (`go run ./cmd/create`, paquete `create/`) estaba obsoleto: sus plantillas (`create/template/templates.go`) importaban `github.com/cgalvisleon/et/config` (paquete eliminado) y un módulo `github.com/cgalvisleon/jdb` inexistente, y llamaban funciones que ya no existen (`envar.Reload()`) o con firmas viejas. Fue corregido y reverificado contra el `et` actual — hoy `go run ./cmd/create` (interactivo: elige "Project"/"Microservice"/"Modelo"/"Rpc") es la forma más rápida de obtener el esqueleto de §3/§4 sin copiar nada a mano. Si necesitas invocarlo sin el menú interactivo, sus funciones son públicas: `create.MkProject(modulePath, name, author, schema)`, `create.MkMicroservice(modulePath, name, schema)`, `create.MkMolue(modulePath, packageName, modelo, schema)`, `create.MkRpc(name)` (`schema == ""` genera la variante sin base de datos).

Sigue siendo cierto que `create modelo` (agregar un segundo recurso a un servicio ya generado) no reescribe automáticamente `model.go`/`router.go` — `file.MakeFile` nunca sobreescribe un archivo existente, así que el comando termina imprimiendo el fragmento exacto a copiar a mano (wiring de `Define<Modelo>(db)` en `initModels`, y las rutas nuevas en `Routes()`). Si ves ese mensaje, es el flujo esperado, no un fallo.

Este documento sigue siendo la referencia de **por qué** el esqueleto tiene la forma que tiene (y sirve para reconstruirlo a mano si prefieres no usar el generador, o para un tipo de proyecto que `create/` no cubre, como un CLI estilo `tick`) — `create/` solo cubre el patrón de microservicio HTTP de §3, no el patrón CLI de §4.

---

## 1. Decide qué tipo de proyecto vas a crear

| Si el proyecto es... | Sigue el patrón de... | Sección |
| --- | --- | --- |
| Un servicio HTTP / microservicio (API REST, posiblemente multi-tenant) | `core-studio/api` | §3 |
| Un CLI (comandos tipo git, sin servidor HTTP) | `tick` | §4 |
| Un worker/job sin HTTP ni CLI interactivo | subconjunto de §3 sin `server`/`ettp` | — |

En ambos casos, antes de escribir una sola línea, repasa la Matriz de Prioridad de `AI_USAGE_GUIDE.md` (§1): usa un componente de `et` tal cual → extiéndelo → adáptalo → implementa sin `et` → dependencia externa, en ese orden, y dilo explícitamente si terminas en las últimas dos opciones.

---

## 2. Bootstrap común (aplica a ambos tipos)

```bash
mkdir <nombre-proyecto> && cd <nombre-proyecto>
go mod init github.com/cgalvisleon/<nombre-proyecto>
go get github.com/cgalvisleon/et@latest
```

**Si el proyecto va a vivir dentro de este workspace (`cgalvisleon/`)** y quieres que los cambios locales a `et` se reflejen sin publicar versión, agrégalo al `go.work` de la raíz (ver `/Users/cesargalvisleon/Projects/cgalvisleon/CLAUDE.md`):

```
go work use ./<nombre-proyecto>
```

**Variables de entorno**: `envar/` carga `.env` automáticamente al importarse (`_ "github.com/joho/godotenv/autoload"` como side-effect dentro del propio paquete) — no hay que llamar nada explícito para que se lea. Crea `.env` en la raíz del proyecto con las variables que vayas a necesitar (ver tablas de cada plantilla abajo). No necesitas `envar.Reload()` — no existe.

**Formato**: `gofmt -w .` antes de cada build/commit, como en el resto del workspace.

---

## 3. Plantilla A — Microservicio HTTP (patrón `core-studio/api`)

### Estructura de carpetas

```
<proyecto>/
  cmd/<servicio>/main.go
  internal/services/<servicio>/service.go
  internal/services/<servicio>/v1/api.go
  internal/models/<dominio>/<dominio>.go
  pkg/<servicio>/router.go
  go.mod
  .env
```

### `cmd/<servicio>/main.go`

```go
package main

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	serv "github.com/cgalvisleon/<proyecto>/internal/services/<servicio>"
)

func main() {
	port := envar.SetIntByArg("-port", "PORT", 3000)
	rpcPort := envar.SetIntByArg("-rpc_port", "RPC_PORT", 4000)

	srv, err := serv.New(port, rpcPort)
	if err != nil {
		logs.Fatal(err)
	}

	srv.Start()
}
```

(`envar.SetIntByArg(arg, name string, def int) int` — el primer argumento es el flag de línea de comandos, el segundo el nombre de la variable de entorno a fijar. Confirmado en `envar/set.go`.)

### `internal/services/<servicio>/service.go` — arranque de infraestructura

Sigue exactamente la secuencia de `core-studio/api/internal/services/apps/v1/api.go` (verificado, real, compila):

```go
package servicio

import (
	"github.com/cgalvisleon/et/server"
	v1 "github.com/cgalvisleon/<proyecto>/internal/services/<servicio>/v1"
)

func New(port, rpcPort int) (*server.Ettp, error) {
	result, err := server.New("<servicio>", port)
	if err != nil {
		return nil, err
	}

	latest := v1.New()
	result.Mount("/", latest)
	result.Mount("/v1", latest)
	result.OnClose(v1.Close)

	return result, nil
}
```

```go
// internal/services/<servicio>/v1/api.go
package v1

import (
	"net/http"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	_ "github.com/cgalvisleon/et/jsql/drivers/postgres" // side-effect: registra el driver
	pkg "github.com/cgalvisleon/<proyecto>/pkg/<servicio>"
)

func New() http.Handler {
	if err := cache.Load(); err != nil {
		logs.Panic(err)
	}
	if err := event.Load(); err != nil {
		logs.Panic(err)
	}

	db, err := jsql.Load() // o jsql.LoadTo("<nombre-db>") para una BD con nombre distinto a DB_NAME
	if err != nil {
		logs.Panic(err)
	}

	return pkg.Routes("<servicio>", "v1.0.0", db)
}

func Close() {
	jrpc.Close()
	// cache.Close()/event.Close() tienen un bug conocido de recursión infinita
	// (ver LIBRARY_CONTEXT.md, Anti-Patrones #9) — no dependas de que cierren limpio.
}
```

> Nota: `cache.Load()`/`event.Load()` ya **no** son flags automáticos de `ettp/v2` ni de `server/` — si necesitas Redis/NATS, llámalos tú mismo antes de construir el router, como arriba.

### `internal/models/<dominio>/<dominio>.go` — primer modelo

Patrón real verificado (`core-studio/api/internal/models/apps/tenant.go` y `et/CLAUDE.md`):

```go
package dominio

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

const schema = "<dominio>"

var store *jsql.Model

func Define(db *jsql.DB) error {
	if store != nil {
		return nil
	}

	columns := []jsql.Column{
		{Name: jsql.CREATED_AT, TypeColumn: jsql.COLUMN, TypeData: jsql.DATETIME, Default: ""},
		{Name: jsql.UPDATED_AT, TypeColumn: jsql.COLUMN, TypeData: jsql.DATETIME, Default: ""},
		{Name: jsql.STATUS, TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: jsql.ACTIVE},
		{Name: jsql.ID, TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
		{Name: "name", TypeColumn: jsql.COLUMN, TypeData: jsql.TEXT, Default: ""},
	}

	def := jsql.Def{
		Schema:      schema,
		Name:        "<tabla>",
		Version:     1,
		Columns:     columns,
		PrimaryKeys: []jsql.DefIndex{{Name: jsql.ID, Sorted: true}},
		Unique:      []jsql.DefIndex{{Name: "name", Sorted: true}},
		IdxField:    jsql.IDX,
		SourceField: jsql.SOURCE,
	}

	result, err := db.Define(def)
	if err != nil {
		return err
	}
	if err := result.Init(); err != nil {
		return err
	}

	store = result
	return nil
}
```

Consulta con el builder fluido: `store.Where(jsql.Eq("id", id)).One()`, `store.Where(jsql.Eq(jsql.STATUS, jsql.ACTIVE)).Limit(20).Page(1).All()`.

### `pkg/<servicio>/router.go` — HTTP mínimo

```go
package servicio

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/response"
	"github.com/go-chi/chi/v5"
)

func Routes(name, version string, db *jsql.DB) http.Handler {
	r := chi.NewRouter()

	r.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		response.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: et.Json{
			"service": name,
			"version": version,
		}})
	})

	return r
}
```

### Variables de entorno mínimas (Postgres)

| Variable | Ejemplo | Usada por |
| --- | --- | --- |
| `DB_DRIVER` | `postgres` | `jsql.Load`/`LoadTo` |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | — | conexión |
| `DB_TENANT_ID` | `tenant:root` (default) | tenant leído internamente por `jsql` |
| `REDIS_HOST` | — | `cache.Load()` |
| `NATS_HOST` | — | `event.Load()` |
| `PORT`, `RPC_PORT` | `3000`, `4000` | `main.go` |

Si no necesitas Redis/NATS, simplemente no llames `cache.Load()`/`event.Load()` — no son obligatorios (a diferencia de lo que sugería una plantilla vieja del `create/`).

---

## 4. Plantilla B — CLI (patrón `tick`)

Para un CLI local (sin servidor HTTP), usa Cobra + `jsql` sobre **SQLite** en vez de Postgres. El driver SQLite **sí existe y funciona** hoy (`jsql/drivers/sqlite/`, basado en `modernc.org/sqlite`, puro Go, se auto-registra vía `init()`) — no confíes en la afirmación contraria de `LIBRARY_CONTEXT.md` §Anti-Patrones #1, quedó desactualizada; verifica la existencia del directorio si tienes dudas.

### Estructura de carpetas

```
<proyecto>/
  cmd/main.go
  pkg/<proyecto>/root.go          (raíz de Cobra + comandos, uno por archivo)
  internal/store/db.go            (Open, modelos jsql sobre sqlite)
  internal/findroot/findroot.go   (opcional: localizar un directorio de datos subiendo desde cwd, como .git)
  go.mod
```

### `cmd/main.go`

```go
package main

import (
	"fmt"
	"os"

	"github.com/cgalvisleon/<proyecto>/pkg/<proyecto>"
)

func main() {
	if err := <proyecto>.Root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
```

### `internal/store/db.go` — apertura de la BD embebida

```go
package store

import (
	"github.com/cgalvisleon/et/jsql"
	_ "github.com/cgalvisleon/et/jsql/drivers/sqlite" // side-effect: registra el driver
)

func Open(path string) (*jsql.DB, error) {
	// Ajusta DB_DRIVER/DB_HOST/DB_NAME vía envar.Set antes de Load(),
	// o usa el constructor de DB que exponga jsql para apuntar a un archivo concreto —
	// revisa jsql/db.go / jsql.NewDB en el código actual antes de asumir la firma exacta,
	// este paquete cambia rápido (ver advertencia de vigencia en CLAUDE.md raíz de et).
	return jsql.Load()
}
```

> A diferencia del microservicio, un CLI normalmente no lee `.env` de variables globales sino que resuelve la ruta del archivo `.db` relativa al proyecto del usuario (como hace `tick/internal/findroot`, imitando cómo git encuentra `.git`). Si necesitas ese patrón, cópialo de `tick/internal/findroot/`.

### Comandos Cobra

Un archivo por comando, registrado sobre una variable `Root` compartida (patrón `tick/pkg/tick/root.go`):

```go
package proyecto

import "github.com/spf13/cobra"

var Root = &cobra.Command{Use: "<proyecto>"}
```

```go
package proyecto

import "github.com/spf13/cobra"

func init() {
	Root.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new project",
	RunE: func(cmd *cobra.Command, args []string) error {
		// ...
		return nil
	},
}
```

---

## 5. Checklist antes de dar por "arrancado" el proyecto

- [ ] `go.mod` apunta a `github.com/cgalvisleon/et` (última versión, o `go.work` si vive en este workspace).
- [ ] `.env` (o `.env.<stage>`) con solo las variables que de verdad usas — no copies la lista completa de `et/CLAUDE.md` si no vas a usar `cache`/`event`/`graph`.
- [ ] Si usas Postgres: `import _ ".../jsql/drivers/postgres"`. Si usas SQLite: `import _ ".../jsql/drivers/sqlite"`. Nunca ambos a la vez sin necesitarlo.
- [ ] Mensajes de error centralizados en un `msg.go` propio (patrón usado en todo `et`), no strings repetidos.
- [ ] Respuestas HTTP (si aplica) siempre vía `response.ITEM`/`ITEMS`/`HTTPError`, nunca `json.NewEncoder(w).Encode(...)` a mano.
- [ ] `gofmt -w .` corrido antes del primer commit.
- [ ] No copiaste nada de `create/template/templates.go` sin verificarlo contra el código fuente actual primero.

## 6. Después del bootstrap

Una vez el esqueleto compila, para cualquier funcionalidad nueva sigue en este orden:

1. `AI_USAGE_GUIDE.md` — matriz de prioridad y tabla rápida "necesito X → uso Y".
2. `COMPONENT_CATALOG.md` — firma exacta de la función/tipo antes de usarla (no la asumas por analogía con otro paquete).
3. `LIBRARY_CONTEXT.md` (Anti-Patrones, Migration Guide) — para no chocar con bugs conocidos o imports eliminados.
4. `et/CLAUDE.md` — convenciones de comentarios y comandos del propio repo `et`, si vas a modificar `et` mismo en vez de solo consumirlo.
