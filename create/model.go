package create

const modelDockerfile = `ARG GO_VERSION=1.22

FROM golang:${GO_VERSION}-alpine AS builder

RUN apk update && apk add --no-cache ca-certificates openssl git tzdata
RUN update-ca-certificates

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /src

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN gofmt -w . && go build ./cmd/$1

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /src/$1 ./$1

ENTRYPOINT ["./$1"]
`

const modelMain = `package main

import (
	"os"
	"os/signal"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	serv "$1/internal/service/$2"	
)

func main() {
	envar.SetInt("port", 3000, "Port server", "PORT")
	envar.SetInt("rpc", 4200, "Port rpc server", "RPC")
	envar.SetStr("dbhost", "localhost", "Database host", "DB_HOST")
	envar.SetInt("dbport", 5432, "Database port", "DB_PORT")
	envar.SetStr("dbname", "", "Database name", "DB_NAME")
	envar.SetStr("dbuser", "", "Database user", "DB_USER")
	envar.SetStr("dbpass", "", "Database password", "DB_PASSWORD")

	serv, err := serv.New()
	if err != nil {
		logs.Fatal(err)
	}

	go serv.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	serv.Close()
}
`

const modelService = `package module

import (
	"net"
	"net/http"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jdb"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/strs"
	"github.com/go-chi/chi/v5"
	v1 "$1/internal/service/$2/v1"	
	"github.com/rs/cors"
)

type Server struct {
	http *http.Server
	rpc  *net.Listener
}

func New() (*Server, error) {
	err := cache.Load()
	if err != nil {
		panic(err)
	}

	err = event.Load()
	if err != nil {
		panic(err)
	}

	_, err = jdb.Load()
	if err != nil {
		panic(err)
	}

	/**
	* HTTP
	**/

	server := Server{}

	port := envar.GetInt(3300, "PORT")

	if port != 0 {
		r := chi.NewRouter()

		r.Use(middleware.Logger)
		r.Use(middleware.Recoverer)

		latest := v1.New()

		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			response.HTTPError(w, r, http.StatusNotFound, "404 Not Found")
		})

		r.Mount("/", latest)
		r.Mount("/v1", latest)

		handler := cors.AllowAll().Handler(r)
		addr := strs.Format(":%d", port)
		serv := &http.Server{
			Addr:    addr,
			Handler: handler,
		}

		server.http = serv
	}

	/**
	 * RPC
	 **/
	rpc := envar.GetInt(0, "RPC")

	if rpc != 0 {
		serv := v1.NewRpc(rpc)

		server.rpc = &serv
	}

	return &server, nil
}

func (serv *Server) Close() error {
	v1.Close()
	return nil
}

func (serv *Server) Start() {
	go func() {
		if serv.http == nil {
			return
		}

		svr := serv.http
		logs.Log("Http", "Running on http://localhost%s", svr.Addr)
		logs.Fatal(serv.http.ListenAndServe())
	}()

	go func() {
		if serv.rpc == nil {
			return
		}

		svr := *serv.rpc
		logs.Log("RPC", "Running on tcp:localhost:%s", svr.Addr().String())
		http.Serve(svr, nil)
	}()

	v1.Banner()

	<-make(chan struct{})
}
`

const modelApi = `package v1

import (
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"time"

	"github.com/cgalvisleon/et/utility"
	"github.com/dimiro1/banner"
	"github.com/go-chi/chi/v5"
	"github.com/mattn/go-colorable"	
	pkg "$1/pkg/$2"	
)

func New() http.Handler {
	r := chi.NewRouter()
	
	_pkg := &pkg.Router{
		Repository: &pkg.Controller{
		},
	}

	r.Mount(pkg.PackagePath, _pkg.Routes())

	return r
}

func Close() {
}

func NewRpc(port int) net.Listener {
	rpc.HandleHTTP()

	result, err := net.Listen("tcp", utility.Address("0.0.0.0", port))
	if err != nil {
		panic(err)
	}

	return result
}

func Banner() {
	time.Sleep(3 * time.Second)
	templ := utility.BannerTitle(pkg.PackageName, pkg.PackageVersion, 4)
	banner.InitString(colorable.NewColorableStdout(), true, true, templ)
	fmt.Println()
}
`

const modelEvent = `package $1

import (
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/message"
)

func initEvents() {
	err := event.Stack("<channel>", eventAction)
	if err != nil {
		logs.Error(err)
	}

}

func eventAction(m message.Message) {
	data, err := m.Json()
	if err != nil {
		logs.Alert(err)
	}

	logs.Log("eventAction", data)
}
`

const modelModel = `package $1

import (
	"github.com/cgalvisleon/et/logs"
)

func initModels() error {
	if err := Define$2(); err != nil {
		return logs.Panic(err)
	}

	return nil
}
`

const modelSchema = `package $1

import "github.com/cgalvisleon/et/linq"

var $2 *linq.Schema

func defineSchema() error {
	if $2 == nil {
		$2 = linq.NewSchema(0, "$3", true, false, true)
	}

	return nil
}
`

const modelhRpc = `package $1

import (
	"net/rpc"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/et"
)

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

	rq := et.ByteToJson(require)
	help := rq.Str("help")

	result := et.Item{
		Ok: true,
		Result: et.Json{
			"service": PackageName,
			"host":    HostName,
			"help":    help,
		},
	}

	*response = result.ToByte()

	return nil
}
`

const modelMsg = `package $1

const (
	// MSG
	MSG_ATRIB_REQUIRED = "Atributo requerido (%s)"
	MSG_VALUE_REQUIRED = "Atributo requerido (%s) value:%s"
	RECORD_NOT_FOUND   = "Registro no encontrado"
	RECORD_NOT_UPDATE  = "Registro no actualizado"
)
`

const modelDbController = `package $1

import (
	"context"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

type Controller struct{}

func (c *Controller) Version(ctx context.Context) (et.Json, error) {
	company := envar.GetStr("", "COMPANY")
	web := envar.GetStr("", "WEB")
	version := envar.GetStr("", "VERSION")
  service := et.Json{
		"version": version,
		"service": PackageName,
		"host":    HostName,
		"company": company,
		"web":     web,
		"help":    "",
	}

	return service, nil
}

func (c *Controller) Init(ctx context.Context) {
	initModels()
	initEvents()
}

type Repository interface {
	Version(ctx context.Context) (et.Json, error)
	Init(ctx context.Context)
}
`

const modelController = `package $1

import (
	"context"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/jdb"
	"github.com/cgalvisleon/et/et"
)

type Controller struct {
	Db *jdb.Conn
}

func (c *Controller) Version(ctx context.Context) (et.Json, error) {
	company := envar.GetStr("", "COMPANY")
	web := envar.GetStr("", "WEB")
	version := envar.GetStr("", "VERSION")
  service := et.Json{
		"version": version,
		"service": PackageName,
		"host":    HostName,
		"company": company,
		"web":     web,
		"help":    "",
	}

	return service, nil
}

func (c *Controller) Init(ctx context.Context) {
	initEvents()
}

type Repository interface {
	Version(ctx context.Context) (et.Json, error)
	Init(ctx context.Context)
}
`

const modelDbRouter = `package $1

import (
	"context"
	"net/http"
	"os"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
	"github.com/go-chi/chi/v5"
)

var PackageName = "$1"
var PackageTitle = "$1"
var PackagePath = envar.GetStr("/api/$1", "PATH_URL")
var PackageVersion = envar.GetStr("0.0.1", "VERSION")
var HostName, _ = os.Hostname()

type Router struct {
	Repository Repository
}

func (rt *Router) Routes() http.Handler {
	var host = strs.Format("%s:%d", envar.GetStr("http://localhost", "HOST"), envar.GetInt(3300, "PORT"))

	r := chi.NewRouter()

	router.Public(r, router.Get, "/version", rt.version, PackageName, PackagePath, host)
	// $2
	router.Protect(r, router.Get, "/{id}", rt.get$2ById, PackageName, PackagePath, host)
	router.Protect(r, router.Post, "/", rt.upSert$2, PackageName, PackagePath, host)
	router.Protect(r, router.Put, "/state/{id}", rt.state$2, PackageName, PackagePath, host)
	router.Protect(r, router.Delete, "/{id}", rt.delete$2, PackageName, PackagePath, host)
	router.Protect(r, router.Get, "/all", rt.all$2, PackageName, PackagePath, host)

	ctx := context.Background()
	rt.Repository.Init(ctx)

	logs.Log(PackageName, "Router version:%s", PackageVersion)
	return r
}

func (rt *Router) version(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result, err := rt.Repository.Version(ctx)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}
`

const modelRouter = `package $1

import (
	"context"
	"net/http"
	"os"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
	"github.com/go-chi/chi/v5"
)

var PackageName = "$1"
var PackageTitle = "$1"
var PackagePath = envar.GetStr("/api/$1", "PATH_URL")
var PackageVersion = envar.GetStr("0.0.1", "VERSION")
var HostName, _ = os.Hostname()

type Router struct {
	Repository Repository
}

func (rt *Router) Routes() http.Handler {
	var host = strs.Format("%s:%d", envar.GetStr("http://localhost", "HOST"), envar.GetInt(3300, "PORT"))

	r := chi.NewRouter()

	router.Public(r, router.Get, "/version", rt.version, PackageName, PackagePath, host)
	// $2
	router.Protect(r, router.Post, "/", rt.$2, PackageName, PackagePath, host)
	
	ctx := context.Background()
	rt.Repository.Init(ctx)

	logs.Log(PackageName, "Router version:%s", PackageVersion)
	return r
}

func (rt *Router) version(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result, err := rt.Repository.Version(ctx)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}
`

const restHttp = `@host=localhost:3300
@token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6IlVTRVIuQURNSU4iLCJhcHAiOiJEZXZvcHMtSW50ZXJuZXQiLCJuYW1lIjoiQ2VzYXIgR2FsdmlzIExlw7NuIiwia2luZCI6ImF1dGgiLCJ1c2VybmFtZSI6Iis1NzMxNjA0Nzk3MjQiLCJkZXZpY2UiOiJkZXZlbG9wIiwiZHVyYXRpb24iOjI1OTIwMDB9.dexIOute7r9o_P8U3t6l9RihN8BOnLl4xpoh9QbQI4k

###
GET /auth HTTP/1.1
Host: {{host}}/version
Authorization: Bearer {{token}}

###
POST /api/test/test HTTP/1.1
Host: {{host}}
Content-Type: application/json
Authorization: Bearer {{token}}
Content-Length: 227

{
}
`

const modelDbHandler = `package $1

import (
	"net/http"

	"github.com/cgalvisleon/et/core"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/generic"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
	"github.com/go-chi/chi/v5"
)

var $2 *linq.Model

func Define$2() error {
	if err := defineSchema(); err != nil {
		return logs.Panic(err)
	}

	if $2 != nil {
		return nil
	}

	$2 = linq.NewModel($3, "$4", "Tabla", 1)
	$2.DefineColum("date_make", "", "TIMESTAMP", "NOW()")
	$2.DefineColum("date_update", "", "TIMESTAMP", "NOW()")
	$2.DefineColum("_state", "", "VARCHAR(80)", utility.ACTIVE)
	$2.DefineColum("_id", "", "VARCHAR(80)", "-1")
	$2.DefineColum("project_id", "", "VARCHAR(80)", "-1")
	$2.DefineColum("name", "", "VARCHAR(250)", "")
	$2.DefineColum("description", "", "TEXT", "")
	$2.DefineColum("_data", "", "JSONB", "{}")
	$2.DefineColum("index", "", "INTEGER", 0)
	$2.DefinePrimaryKey([]string{"_id"})
	$2.DefineIndex([]string{
		"date_make",
		"date_update",
		"_state",
		"project_id",
		"name",
		"index",
	})
	$2.DefineRequired([]string{
		"name:Atributo requerido (name)",
	})
	$2.IntegrityAtrib(true)
	$2.Trigger(linq.BeforeInsert, func(model *linq.Model, old, new *et.Json, data et.Json) error {
		return nil
	})
	$2.Trigger(linq.AfterInsert, func(model *linq.Model, old, new *et.Json, data et.Json) error {
		return nil
	})
	$2.Trigger(linq.BeforeUpdate, func(model *linq.Model, old, new *et.Json, data et.Json) error {
		return nil
	})
	$2.Trigger(linq.AfterUpdate, func(model *linq.Model, old, new *et.Json, data et.Json) error {
		return nil
	})
	$2.Trigger(linq.BeforeDelete, func(model *linq.Model, old, new *et.Json, data et.Json) error {
		return nil
	})
	$2.Trigger(linq.AfterDelete, func(model *linq.Model, old, new *et.Json, data et.Json) error {
		return nil
	})
	$2.OnListener = func(data et.Json) {
		logs.Debug(data.ToString())
	}
	
	if err := core.InitModel($2); err != nil {
		return logs.Panic(err)
	}

	return nil
}

/**
*	Handler for CRUD data
 */
func Get$2ById(id string) (et.Item, error) {	
	result, err := $2.Data().
		Where($2.Column("_id").Eq(id)).
		First()
	if err != nil {
		return et.Item{}, err
	}
	
	return result, nil	
}

func Value$2ById(_default any, id, atrib string) *generic.Any {
	item, err := $2.Data(atrib).
		Where($2.Column("_id").Eq(id)).
		First()
	if err != nil {
		return &generic.Any{}
	}

	return item.Any(_default, atrib)
}

func UpSert$2(project_id, id string, data et.Json) (et.Item, error) {
	if !utility.ValidId(project_id) {
		return et.Item{}, logs.Alertf(MSG_ATRIB_REQUIRED, "project_id")
	}

	if !utility.ValidId(id) {
		return et.Item{}, logs.Alertf(MSG_ATRIB_REQUIRED, "_id")
	}

	id = utility.GenId(id)
	data["project_id"] = project_id
	data["_id"] = id
	return $2.UpSert(data).
		Where($2.Column("_id").Eq(id)).
		CommandOne()
}

func State$2(id, state string) (et.Item, error) {
	if !utility.ValidId(state) {
		return et.Item{}, logs.Alertf(MSG_ATRIB_REQUIRED, "state")
	}

	item, err := $2.Data("_state").
		Where($2.Column("_id").Eq(id)).
		First()
	if err != nil {
		return et.Item{}, err
	}

	if !item.Ok {
		return et.Item{}, logs.Alertm(RECORD_NOT_FOUND)
	}

	old_state := item.Key("_state")
	if old_state == state {
		return et.Item{
			Ok: true,
			Result: et.Json{
				"message": RECORD_NOT_UPDATE,
			}}, nil
	}

	return $2.Update(et.Json{
		"_state":   state,
	}).
		Where($2.Column("_id").Eq(id)).
		CommandOne()	
}

func Delete$2(id string) (et.Item, error) {
	return State$2(id, utility.FOR_DELETE)
}

func All$2(project_id, state, search string, page, rows int, _select string) (et.List, error) {	
	if state == "" {
		state = utility.ACTIVE
	}

	auxState := state

	if search != "" {
		return $2.Data(_select).
			Where($2.Column("project_id").In("-1", project_id)).
			And($2.Concat("NAME:", $2.Column("name"), "DESCRIPTION:", $2.Column("description"), "DATA:", $2.Column("_data"), ":").Like("%"+search+"%")).
			OrderBy($2.Column("name"), true).
			List(page, rows)
	} else if auxState == "*" {
		state = utility.FOR_DELETE

		return $2.Data(_select).
			Where($2.Column("_state").Neg(state)).
			And($2.Column("project_id").In("-1", project_id)).
			OrderBy($2.Column("name"), true).
			List(page, rows)
	} else if auxState == "0" {
		return $2.Data(_select).
			Where($2.Column("_state").In("-1", state)).
			And($2.Column("project_id").In("-1", project_id)).
			OrderBy($2.Column("name"), true).
			List(page, rows)
	} else {
		return $2.Data(_select).
			Where($2.Column("_state").Eq(state)).
			And($2.Column("project_id").In("-1", project_id)).
			OrderBy($2.Column("name"), true).
			List(page, rows)
	}
}

/**
* Router
**/
func (rt *Router) upSert$2(w http.ResponseWriter, r *http.Request) {
	body, _ := response.GetBody(r)
	project_id := body.Str("project_id")
	id := body.Str("id")	

	result, err := UpSert$2(project_id, id, body)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, result)
}

func (rt *Router) get$2ById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := Get$2ById(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, result)
}

func (rt *Router) state$2(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, _ := response.GetBody(r)
	state := body.Str("state")

	result, err := State$2(id, state)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, result)
}

func (rt *Router) delete$2(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := Delete$2(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, result)
}

func (rt *Router) all$2(w http.ResponseWriter, r *http.Request) {
	query := response.GetQuery(r)
	project_id := query.Str("project_id")
	state := query.Str("state")
	search := query.Str("search")
	page := query.ValInt(1, "page")
	rows := query.ValInt(30, "rows")
	_select := query.Str("select")

	result, err := All$2(project_id, state, search, page, rows, _select)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}

/** Copy this code to router.go
	// $2
	er.ProtectRoute(r, er.Get, "/$5/{id}", rt.get$2ById, PackageName, PackagePath, Host)
	er.ProtectRoute(r, er.Post, "/$5", rt.upSert$2, PackageName, PackagePath, Host)
	er.ProtectRoute(r, er.Put, "/$5/state/{id}", rt.state$2, PackageName, PackagePath, Host)
	er.ProtectRoute(r, er.Delete, "/$5/{id}", rt.delete$2, PackageName, PackagePath, Host)
	er.ProtectRoute(r, er.Get, "/$5/all", rt.all$2, PackageName, PackagePath, Host)
**/

/** Copy this code to func initModel in model.go
	if err := Define$2(); err != nil {
		return logs.Panic(err)
	}
**/
`

const modelHandler = `package $1

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/response"
)

func $2(project_id, id string, params et.Json) (et.Item, error) {

	return et.Item{}, nil
}


/**
* Router
**/
func (rt *Router) $3(w http.ResponseWriter, r *http.Request) {
	body, _ := response.GetBody(r)
	project_id := body.Str("project_id")
	id := body.Str("id")	

	result, err := $2(project_id, id, body)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, result)
}

/** Copy this code to router.go
	// $2
	er.ProtectRoute(r, er.Post, "/$3", rt.$2, PackageName, PackagePath, Host)	
**/
`

const modelReadme = `
## Project $1

## Create project

go mod init github.com/$1/api

### Dependencias

go get -u github.com/joho/godotenv/autoload &&
go get -u github.com/redis/go-redis/v9 &&
go get -u github.com/google/uuid &&
go get -u github.com/nats-io/nats.go &&
go get -u golang.org/x/crypto/bcrypt &&
go get -u golang.org/x/exp/slices &&
go get -u github.com/manifoldco/promptui &&
go get -u github.com/schollz/progressbar/v3 &&
go get -u github.com/spf13/cobra &&
go get -u github.com/cgalvisleon/et

### Crear projecto, microservicios, modelos

go run github.com/cgalvisleon/et/cmd/create-go create
`

const modelEnvar = `APP=
PORT=3300
VERSION=0.0.0
COMPANY=Company
PATH_URL=
WEB=https://www.celsia.com
PRODUCTION=false
PATH_URL=/api/$1
HOST=localhost

# DB
DB_DRIVE=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=test
DB_USER=test
DB_PASSWORD=test

# REDIS
REDIS_HOST=localhost:6379
REDIS_PASSWORD=test
REDIS_DB=0

# NATS
NATS_HOST=localhost:4222

# CALM
SECRET=test
`

const modelDeploy = `version: "3"

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
      - "APP=Celsia Internet - Event Stack"
      - "PORT=3300"
      - "VERSION=1.0.1"
      - "COMPANY=Celsia Internet"
      - "WEB=https://www.internet.celsia.com"
      - "PATH_URL=/api/$1"
      - "PRODUCTION=true"
      - "HOST=stack"
      # DB
      - "DB_DRIVE=postgres"
      - "DB_HOST="
      - "DB_PORT=5432"
      - "DB_NAME=internet"
      - "DB_USER=internet"
      - "DB_PASSWORD="
      - "DB_APPLICATION_NAME=$1"
      # REDIS
      - "REDIS_HOST="
      - "REDIS_PASSWORD="
      - "REDIS_DB=0"
      # NATS
      - "NATS_HOST=nats:4222"
      # CALM
      - "SECRET="
      # RPC
      - "PORT_RPC=4200"
`
