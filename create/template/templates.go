package template

const ModelDockerfile = `# Versión de Go como argumento3
ARG GO_VERSION=1.25

# Stage 1: Compilación (builder)
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

# Argumentos para el sistema operativo y la arquitectura
ARG TARGETOS
ARG TARGETARCH

# Instalación de dependencias necesarias
RUN apk update && apk add --no-cache ca-certificates openssl git \
    && update-ca-certificates

# Configuración de las variables de entorno para la build
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

# Directorio de trabajo
WORKDIR /src

# Descargar dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar el código fuente
COPY . .

# Formatear el código Go
RUN gofmt -w .

# Compilar el binario
RUN go build -a -v -o /$1 ./cmd/$1

# Cambiar permisos del binario
RUN chmod +x /$1

# Stage 2: Imagen final mínima
FROM scratch

# Copiar certificados y binario
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /$1 /$1

# Establecer el binario como punto de entrada
ENTRYPOINT ["/$1"]
`

// ModelMain: cmd/$2/main.go — $1 = module path (go.mod's module), $2 = service name.
const ModelMain = `package main

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	serv "$1/internal/services/$2"
)

func main() {
	port := envar.SetIntByArg("-port", "PORT", 3300)
	rpcPort := envar.SetIntByArg("-rpc_port", "RPC_PORT", 4200)

	srv, err := serv.New(port, rpcPort)
	if err != nil {
		logs.Fatal(err)
	}

	srv.Start()
}
`

// ModelApi: internal/services/$2/v1/api.go, no-DB variant — $1 = module path, $2 = service name.
const ModelApi = `package v1

import (
	"net/http"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	pkg "$1/pkg/$2"
)

func New(rpcPort int) http.Handler {
	if err := pkg.LoadConfig(); err != nil {
		logs.Panic(err)
	}

	if err := cache.Load(); err != nil {
		logs.Panic(err)
	}

	if err := event.Load(); err != nil {
		logs.Panic(err)
	}

	if err := pkg.StartRpcServer(rpcPort); err != nil {
		logs.Panic(err)
	}

	return pkg.Routes()
}

func Close() {
	jrpc.Close()
	// cache.Close()/event.Close() no cierran limpio (bug conocido de recursión
	// infinita en et/cache y et/event) — no se llaman aquí a propósito.
}
`

// ModelDbApi: internal/services/$2/v1/api.go, DB-backed variant — $1 = module path, $2 = service name.
const ModelDbApi = `package v1

import (
	"net/http"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/jsql"
	_ "github.com/cgalvisleon/et/jsql/drivers/postgres"
	"github.com/cgalvisleon/et/logs"
	pkg "$1/pkg/$2"
)

func New(rpcPort int) http.Handler {
	if err := pkg.LoadConfig(); err != nil {
		logs.Panic(err)
	}

	if err := cache.Load(); err != nil {
		logs.Panic(err)
	}

	if err := event.Load(); err != nil {
		logs.Panic(err)
	}

	if err := pkg.StartRpcServer(rpcPort); err != nil {
		logs.Panic(err)
	}

	db, err := jsql.Load()
	if err != nil {
		logs.Panic(err)
	}

	return pkg.Routes(db)
}

func Close() {
	jrpc.Close()
	// cache.Close()/event.Close() no cierran limpio (bug conocido de recursión
	// infinita en et/cache y et/event) — no se llaman aquí a propósito.
}
`

// ModelService: internal/services/$2/service.go — $1 = module path, $2 = service name.
const ModelService = `package $2

import (
	"github.com/cgalvisleon/et/server"
	v1 "$1/internal/services/$2/v1"
)

func New(port, rpcPort int) (*server.Ettp, error) {
	result := server.New("$2", port)

	latest := v1.New(rpcPort)
	result.Mount("/", latest)
	result.Mount("/v1", latest)
	result.OnClose(v1.Close)

	return result, nil
}
`

// ModelConfig: pkg/$1/config.go — $1 = package name.
const ModelConfig = `package $1

import "github.com/cgalvisleon/et/envar"

/**
* LoadConfig: Validates that the environment is ready before the service starts.
* Extend the required-keys list as this service grows.
* @return error
**/
func LoadConfig() error {
	return envar.Validate([]string{"PORT"})
}
`

// ModelDbController: pkg/$1/controller.go, DB-backed variant — $1 = package name.
const ModelDbController = `package $1

import (
	"context"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

type Controller struct {
	Db *jsql.DB
}

/**
* Version: Returns basic service metadata.
* @param ctx context.Context
* @return et.Json, error
**/
func (c *Controller) Version(ctx context.Context) (et.Json, error) {
	return et.Json{
		"service": PackageName,
		"version": PackageVersion,
	}, nil
}

/**
* Init: Defines models and wires event subscriptions. Called once when the router is built.
* @param ctx context.Context
* @return error
**/
func (c *Controller) Init(ctx context.Context) error {
	if err := initModels(c.Db); err != nil {
		return err
	}

	return initEvents()
}

type Repository interface {
	Version(ctx context.Context) (et.Json, error)
	Init(ctx context.Context) error
}
`

// ModelController: pkg/$1/controller.go, no-DB variant — $1 = package name.
const ModelController = `package $1

import (
	"context"

	"github.com/cgalvisleon/et/et"
)

type Controller struct {
}

/**
* Version: Returns basic service metadata.
* @param ctx context.Context
* @return et.Json, error
**/
func (c *Controller) Version(ctx context.Context) (et.Json, error) {
	return et.Json{
		"service": PackageName,
		"version": PackageVersion,
	}, nil
}

/**
* Init: Wires event subscriptions. Called once when the router is built.
* @param ctx context.Context
* @return error
**/
func (c *Controller) Init(ctx context.Context) error {
	return initEvents()
}

type Repository interface {
	Version(ctx context.Context) (et.Json, error)
	Init(ctx context.Context) error
}
`

// ModelEvent: pkg/$1/event.go — $1 = package name.
const ModelEvent = `package $1

import (
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
)

/**
* initEvents: Subscribes to the events this service reacts to. Requires event.Load()
* to have already been called (see internal/services/$1/v1/api.go).
* @return error
**/
func initEvents() error {
	return event.Stack("$1", func(m event.Message) {
		logs.Log("event", m.Data)
	})
}
`

// ModelData: internal/models/$4/$name.go — $1 = field/var lowercase name, $2 = CamelCase model
// name, $3 = table name, $4 = schema (also the package name of internal/models/$4).
const ModelData = `package $4

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

var $2 *jsql.Model

/**
* Define$2: Defines and initializes the "$3" table.
* @param db *jsql.DB
* @return error
**/
func Define$2(db *jsql.DB) error {
	if $2 != nil {
		return nil
	}

	def := jsql.Def{
		Schema:  "$4",
		Name:    "$3",
		Version: 1,
		Columns: []jsql.Column{
			{Name: jsql.CREATED_AT, TypeColumn: jsql.COLUMN, TypeData: jsql.DATETIME, Default: ""},
			{Name: jsql.UPDATED_AT, TypeColumn: jsql.COLUMN, TypeData: jsql.DATETIME, Default: ""},
			{Name: jsql.STATUS, TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: jsql.ACTIVE},
			{Name: jsql.ID, TypeColumn: jsql.COLUMN, TypeData: jsql.KEY, Default: ""},
			{Name: "name", TypeColumn: jsql.COLUMN, TypeData: jsql.TEXT, Default: ""},
			{Name: "description", TypeColumn: jsql.COLUMN, TypeData: jsql.MEMO, Default: ""},
		},
		PrimaryKeys: []jsql.DefIndex{
			{Name: jsql.ID, Sorted: true},
		},
		Indexes: []jsql.DefIndex{
			{Name: "name", Sorted: true},
		},
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

	$2 = result
	return nil
}

/**
* Get$2ById: Returns a single "$3" row by id.
* @param id string
* @return et.Item, error
**/
func Get$2ById(id string) (et.Item, error) {
	if $2 == nil {
		return et.Item{}, errors.New(MSG_MODEL_NOT_DEFINED)
	}

	return $2.Where(jsql.Eq(jsql.ID, id)).One()
}

/**
* Upsert$2: Creates or updates a "$3" row.
* @param id, name, description string
* @return et.Item, error
**/
func Upsert$2(id, name, description string) (et.Item, error) {
	if $2 == nil {
		return et.Item{}, errors.New(MSG_MODEL_NOT_DEFINED)
	}

	if name == "" {
		return et.Item{}, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}

	id = reg.GetUUID(id)
	now := timezone.Now()
	_, err := $2.
		Upsert(et.Json{
			jsql.ID:       id,
			"name":        name,
			"description": description,
		}).
		BeforeInsert(func(tx *jsql.Tx, old, data et.Json) error {
			data[jsql.CREATED_AT] = now
			data[jsql.UPDATED_AT] = now
			data[jsql.STATUS] = jsql.ACTIVE
			return nil
		}).
		BeforeUpdate(func(tx *jsql.Tx, old, data et.Json) error {
			data[jsql.UPDATED_AT] = now
			return nil
		}).
		Where(jsql.Eq(jsql.ID, id)).
		Exec()
	if err != nil {
		return et.Item{}, err
	}

	return et.Item{
		Ok: true,
		Result: et.Json{
			"message": MSG_RECORD_SAVED,
			jsql.ID:   id,
		},
	}, nil
}

/**
* State$2: Changes the status of a "$3" row (also used to soft-delete via jsql.FOR_DELETE).
* @param id, status string
* @return et.Item, error
**/
func State$2(id, status string) (et.Item, error) {
	if $2 == nil {
		return et.Item{}, errors.New(MSG_MODEL_NOT_DEFINED)
	}

	now := timezone.Now()
	items, err := $2.
		Update(et.Json{
			jsql.STATUS:     status,
			jsql.UPDATED_AT: now,
		}).
		Where(jsql.Eq(jsql.ID, id)).
		Exec()
	if err != nil {
		return et.Item{}, err
	}

	if items.Count == 0 {
		return et.Item{}, errors.New(MSG_RECORD_NOT_FOUND)
	}

	return et.Item{
		Ok: true,
		Result: et.Json{
			"message": MSG_RECORD_UPDATED,
		},
	}, nil
}

/**
* Query$2: Runs a query.QL-style query (see jsql.Model.Query) against "$3".
* @param query et.Json
* @return et.Items, error
**/
func Query$2(query et.Json) (et.Items, error) {
	if $2 == nil {
		return et.Items{}, errors.New(MSG_MODEL_NOT_DEFINED)
	}

	return $2.Query(query)
}
`

// ModelDbHandler: pkg/$1/router-<name>.go, DB-backed variant.
// $1 = package name, $2 = CamelCase model name, $3 = module path, $4 = schema.
const ModelDbHandler = `package $1

import (
	"net/http"

	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
	"$3/internal/models/$4"
)

/**
* upsert$2
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (rt *Router) upsert$2(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	id := body.Str(jsql.ID)
	name := body.Str("name")
	description := body.Str("description")
	result, err := $4.Upsert$2(id, name, description)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusCreated, result)
}

/**
* get$2ById
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (rt *Router) get$2ById(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	result, err := $4.Get$2ById(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, result)
}

/**
* state$2
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (rt *Router) state$2(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	status := body.Str(jsql.STATUS)
	result, err := $4.State$2(id, status)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, result)
}

/**
* delete$2
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (rt *Router) delete$2(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()
	result, err := $4.State$2(id, jsql.FOR_DELETE)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, result)
}

/**
* query$2
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (rt *Router) query$2(w http.ResponseWriter, r *http.Request) {
	body, err := request.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	query := body.Json("query")
	result, err := $4.Query$2(query)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEMS(w, r, http.StatusOK, result)
}

/** Copy these lines into the Routes() func in router.go — mount them under their own
    prefix (e.g. r.Route("/$2", func(r chi.Router) {...})) so they don't collide with
    the routes already registered for the first resource:
	// $2
	r.Get("/{id}", rt.get$2ById)
	r.Post("/", rt.upsert$2)
	r.Put("/{id}/status", rt.state$2)
	r.Delete("/{id}", rt.delete$2)
	r.Get("/", rt.query$2)
**/

/** Copy this line into initModels() in model.go:
	if err := $4.Define$2(db); err != nil {
		return err
	}
**/
`

// ModelHandler: pkg/$1/h$2.go, no-DB variant — $1 = package name, $2 = CamelCase model name.
const ModelHandler = `package $1

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
)

/**
* Get$2: Replace this with the real lookup for your resource.
* @param id string
* @return et.Item, error
**/
func Get$2(id string) (et.Item, error) {
	return et.Item{
		Ok: true,
		Result: et.Json{
			"id": id,
		},
	}, nil
}

/**
* get$2
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (rt *Router) get$2(w http.ResponseWriter, r *http.Request) {
	id := request.URLParam(r, "id").Str()

	result, err := Get$2(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, result)
}

/** Copy this line into the Routes() func in router.go:
	// $2
	r.Get("/$2/{id}", rt.get$2)
**/
`

// ModelModel: pkg/$1/model.go — $1 = package name, $2 = schema (import segment / package
// alias, same value declared as `package $2` in internal/models/$2), $3 = module path,
// $4 = CamelCase model name.
const ModelModel = `package $1

import (
	"github.com/cgalvisleon/et/jsql"
	"$3/internal/models/$2"
)

func initModels(db *jsql.DB) error {
	return $2.Define$4(db)
}
`

const ModelhRpc = `package $1

import (
	"encoding/json"
	"net/rpc"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

var PackageName = "$1"
var initRpc bool

type Service et.Item

func InitRpc() error {
	service := new(Service)

	err := rpc.Register(service)
	if err != nil {
		return logs.Error(err)
	}

	initRpc = true

	return nil
}

func (c *Service) Version(require []byte, response *[]byte) error {
	if !initRpc {
		return nil
	}

	var rq et.Json
	if err := json.Unmarshal(require, &rq); err != nil {
		return err
	}
	help := rq.Str("help")

	result := et.Item{
		Ok: true,
		Result: et.Json{
			"service": PackageName,
			"help":    help,
		},
	}

	bt, err := result.ToByte()
	if err != nil {
		return err
	}

	*response = bt

	return nil
}
`

// ModelMsg: pkg/$1/msg.go and internal/models/$1/msg.go — $1 = package name.
const ModelMsg = `package $1

const (
	MSG_MODEL_NOT_DEFINED = "model not defined, call Define first"
	MSG_RECORD_NOT_FOUND  = "record not found"
	MSG_RECORD_SAVED      = "record saved"
	MSG_RECORD_UPDATED    = "record updated"
)
`

// ModelDbRouter: pkg/$1/router.go, DB-backed variant — $1 = package name, $2 = CamelCase model name.
const ModelDbRouter = `package $1

import (
	"context"
	"net/http"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/response"
	"github.com/go-chi/chi/v5"
)

var PackageName = "$1"
var PackageVersion = envar.GetStr("VERSION", "0.0.0")

type Router struct {
	Repository Repository
}

/**
* Routes: Builds the HTTP handler for this package, initializing its models and
* event subscriptions once (via Repository.Init) before serving.
* @param db *jsql.DB
* @return http.Handler
**/
func Routes(db *jsql.DB) http.Handler {
	rt := &Router{Repository: &Controller{Db: db}}
	if err := rt.Repository.Init(context.Background()); err != nil {
		logs.Panic(err)
	}

	r := chi.NewRouter()
	r.Get("/version", rt.version)
	// $2
	r.Get("/{id}", rt.get$2ById)
	r.Post("/", rt.upsert$2)
	r.Put("/{id}/status", rt.state$2)
	r.Delete("/{id}", rt.delete$2)
	r.Get("/", rt.query$2)

	logs.Logf(PackageName, "Router version:%s", PackageVersion)

	return r
}

func (rt *Router) version(w http.ResponseWriter, r *http.Request) {
	result, err := rt.Repository.Version(r.Context())
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: result})
}
`

// ModelRouter: pkg/$1/router.go, no-DB variant — $1 = package name, $2 = CamelCase model name.
const ModelRouter = `package $1

import (
	"context"
	"net/http"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/response"
	"github.com/go-chi/chi/v5"
)

var PackageName = "$1"
var PackageVersion = envar.GetStr("VERSION", "0.0.0")

type Router struct {
	Repository Repository
}

/**
* Routes: Builds the HTTP handler for this package, running Repository.Init once
* (event subscriptions, etc.) before serving.
* @return http.Handler
**/
func Routes() http.Handler {
	rt := &Router{Repository: &Controller{}}
	if err := rt.Repository.Init(context.Background()); err != nil {
		logs.Panic(err)
	}

	r := chi.NewRouter()
	r.Get("/version", rt.version)
	// $2
	r.Get("/{id}", rt.get$2)

	logs.Logf(PackageName, "Router version:%s", PackageVersion)

	return r
}

func (rt *Router) version(w http.ResponseWriter, r *http.Request) {
	result, err := rt.Repository.Version(r.Context())
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: result})
}
`

// ModelRpc: pkg/$1/rpc.go — $1 = package name. Exposes the service over the classic
// net/rpc-over-TCP transport (github.com/cgalvisleon/et/jrpc), mirroring the pattern
// documented for core-studio/api's pkg/<service>/rpc.go.
const ModelRpc = `package $1

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
)

type Services struct{}

/**
* StartRpcServer: Registers Services over net/rpc and starts listening on rpcPort.
* @param rpcPort int
* @return error
**/
func StartRpcServer(rpcPort int) error {
	host := envar.GetStr("RPC_HOST", "localhost")
	if _, err := jrpc.Mount(host, rpcPort, new(Services), PackageName); err != nil {
		return err
	}

	return jrpc.Start(rpcPort)
}

func (c *Services) Version(require et.Json, response *et.Item) error {
	response.Ok = true
	response.Result = et.Json{
		"service": PackageName,
		"version": PackageVersion,
	}

	return logs.Log("rpc", response.ToJson())
}
`

const RestHttp = `@host=localhost:3300
@token=

###
GET /version HTTP/1.1
Host: {{host}}

###
POST /api/$1 HTTP/1.1
Host: {{host}}
Content-Type: application/json
Authorization: Bearer {{token}}

{
}
`

const ModelReadme = `
## Project $1

## Create project

$3
go mod init $2
$3

## Dependencias

$3
go get github.com/cgalvisleon/et@latest
$3

## Run

$3
gofmt -w . && go run ./cmd/$1 -port 3300 -rpc_port 4200
$3
`

const ModelEnvar = `APP=$1
PORT=3300
VERSION=0.0.0
COMPANY=Company
WEB=https://www.home.com
PRODUCTION=false
PATH_URL=/api/$1
HOST=http://localhost
# HOST=http://host.docker.internal

# RPC
RPC_HOST=localhost
RPC_PORT=4200

# DB
DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=test
DB_USER=test
DB_PASSWORD=test
DB_TENANT_ID=tenant:root

# REDIS
REDIS_HOST=localhost:6379
REDIS_PASSWORD=test
REDIS_DB=0

# NATS
NATS_HOST=localhost:4222
NATS_USER=
NATS_PASSWORD=

# SESSION
SECRET=test
`

const ModelDeploy = `version: "3"

networks:
  $3:
    external: true

services:
  $1:
    image: $1:latest
    logging:
      driver: "json-file"
      options:
        max-size: "1m"
        max-file: "2"
    networks:
      - $3
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.$1.rule=PathPrefix($2)"
      - "traefik.http.services.$1.loadbalancer.server.port=3300"
    deploy:
      replicas: 1
    environment:
      - "APP=$APP"
      - "PORT=$PORT"
      - "VERSION=$RELEASE"
      - "COMPANY=$COMPANY"
      - "WEB=$WEB"
      - "PATH_URL=/api/$1"
      - "PRODUCTION=true"
      - "HOST=stack"
      # DB
      - "DB_DRIVER=postgres"
      - "DB_HOST="
      - "DB_PORT=5432"
      - "DB_NAME=internet"
      - "DB_USER=internet"
      - "DB_PASSWORD="
      # REDIS
      - "REDIS_HOST="
      - "REDIS_PASSWORD="
      - "REDIS_DB=0"
      # NATS
      - "NATS_HOST=nats:4222"
      # SECRET
      - "SECRET="
      # RPC
      - "RPC_HOST=$1"
      - "RPC_PORT=4200"
`
