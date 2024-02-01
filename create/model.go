package create

const modelDockerfile = `ARG GO_VERSION=1.21.3

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

	serv "$1/internal/service/$2"
	"github.com/cgalvisleon/elvis/console"
	"github.com/cgalvisleon/elvis/envar"
	_ "github.com/joho/godotenv/autoload"	
)

func main() {
	envar.SetvarInt("port", 3000, "Port server", "PORT")
	envar.SetvarInt("rpc", 4200, "Port rpc server", "RPC_PORT")
	envar.SetvarStr("dbhost", "localhost", "Database host", "DB_HOST")
	envar.SetvarInt("dbport", 5432, "Database port", "DB_PORT")
	envar.SetvarStr("dbname", "", "Database name", "DB_NAME")
	envar.SetvarStr("dbuser", "", "Database user", "DB_USER")
	envar.SetvarStr("dbpass", "", "Database password", "DB_PASSWORD")

	serv, err := serv.New()
	if err != nil {
		console.Fatal(err)
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

	v1 "$1/internal/service/$2/v1"
	"github.com/cgalvisleon/elvis/console"
	"github.com/cgalvisleon/elvis/envar"
	"github.com/cgalvisleon/elvis/middleware"
	"github.com/cgalvisleon/elvis/strs"
	"github.com/go-chi/chi"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/cors"
)

type Server struct {
	http *http.Server
	rpc  *net.Listener
}

func New() (*Server, error) {
	server := Server{}

	/**
	 * HTTP
	 **/
	port := envar.EnvarInt(3300, "PORT")

	if port != 0 {
		r := chi.NewRouter()

		r.Use(middleware.Logger)
		r.Use(middleware.Recoverer)

		latest := v1.New()

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
	port = envar.EnvarInt(4200, "RPC_PORT")

	if port != 0 {
		serv := v1.NewRpc(port)

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
		console.LogKF("Http", "Running on http://localhost%s", svr.Addr)
		console.Fatal(serv.http.ListenAndServe())
	}()

	go func() {
		if serv.rpc == nil {
			return
		}

		svr := *serv.rpc
		console.LogKF("RPC", "Running on tcp:localhost:%s", svr.Addr().String())
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

	pkg "$1/pkg/$2"
	"github.com/cgalvisleon/elvis/cache"
	"github.com/cgalvisleon/elvis/event"
	"github.com/cgalvisleon/elvis/jdb"
	"github.com/cgalvisleon/elvis/utility"
	"github.com/cgalvisleon/elvis/ws"
	"github.com/dimiro1/banner"
	"github.com/go-chi/chi"
	"github.com/mattn/go-colorable"
)

func New() http.Handler {
	r := chi.NewRouter()

	_, err := cache.Load()
	if err != nil {
		panic(err)
	}

	_, err = event.Load()
	if err != nil {
		panic(err)
	}

	_, err = ws.Load()
	if err != nil {
		panic(err)
	}

	Db, err := jdb.Load()
	if err != nil {
		panic(err)
	}

	_pkg := &pkg.Router{
		Repository: &pkg.Controller{
			Db: Db,
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
	"github.com/cgalvisleon/elvis/console"
	"github.com/cgalvisleon/elvis/event"
	e "github.com/cgalvisleon/elvis/json"
)

func initEvents() {
	err := event.Stack("<channel>", eventAction)
	if err != nil {
		console.Error(err)
	}

}

func eventAction(m event.CreatedEvenMessage) {
	data, err := e.ToJson(m.Data)
	if err != nil {
		console.Error(err)
	}

	console.Log("eventAction", data)
}
`

const modelModel = `package $1

import (
	"github.com/cgalvisleon/elvis/console"
)

func initModels() error {
	if err := Define$2(); err != nil {
		return console.PanicE(err)
	}

	return nil
}
`

const modelSchema = `package $1

import "github.com/cgalvisleon/elvis/linq"

var $2 *linq.Schema

func defineSchema() error {
	if $2 == nil {
		$2 = linq.NewSchema(0, "$3")
	}

	return nil
}
`

const modelhRpc = `package $1

import (
	"net/rpc"

	"github.com/cgalvisleon/elvis/console"
	"github.com/cgalvisleon/elvis/json"
)

var initRpc bool

type Service json.Item

func InitRpc() error {
	service := new(Service)

	err := rpc.Register(service)
	if err != nil {
		return console.Error(err)
	}

	initRpc = true

	return nil
}

func (c *Service) Version(require []byte, response *[]byte) error {
	if !initRpc {
		return nil
	}

	rq := json.ByteToJson(require)
	help := rq.Str("help")

	result := json.Item{
		Ok: true,
		Result: json.Json{
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
	MSG_ATRIB_REQUIRED      = "Atributo requerido (%s)"
	MSG_VALUE_REQUIRED      = "Atributo requerido (%s) value:%s"
)
`

const modelController = `package $1

import (
	"context"

	"github.com/cgalvisleon/elvis/envar"
	"github.com/cgalvisleon/elvis/jdb"
	e "github.com/cgalvisleon/elvis/json"
)

type Controller struct {
	Db *jdb.Conn
}

func (c *Controller) Version(ctx context.Context) (e.Json, error) {
	company := envar.EnvarStr("", "COMPANY")
	web := envar.EnvarStr("", "WEB")
	version := envar.EnvarStr("", "VERSION")
  service := e.Json{
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
	Version(ctx context.Context) (e.Json, error)
	Init(ctx context.Context)
}
`

const modelRouter = `package $1

import (
	"context"
	"net/http"
	"os"

	"github.com/cgalvisleon/elvis/console"
	"github.com/cgalvisleon/elvis/envar"
	"github.com/cgalvisleon/elvis/response"
	er "github.com/cgalvisleon/elvis/router"
	"github.com/go-chi/chi"
)

var PackageName = "$1"
var PackageTitle = "$1"
var PackagePath = "/api/$1"
var PackageVersion = envar.EnvarStr("0.0.1", "VERSION")
var HostName, _ = os.Hostname()
var Host = "$1"

type Router struct {
	Repository Repository
}

func (rt *Router) Routes() http.Handler {
	r := chi.NewRouter()

	er.PublicRoute(r, er.Get, "/version", rt.Version, PackageName, PackagePath, Host)
	// $2
	er.ProtectRoute(r, er.Get, "/$1/{id}", rt.Get$2ById, PackageName, PackagePath, Host)
	er.ProtectRoute(r, er.Post, "/$1", rt.UpSert$2, PackageName, PackagePath, Host)
	er.ProtectRoute(r, er.Put, "/$1/state/{id}", rt.State$2, PackageName, PackagePath, Host)
	er.ProtectRoute(r, er.Delete, "/$1/{id}", rt.Delete$2, PackageName, PackagePath, Host)
	er.ProtectRoute(r, er.Get, "/$1/all", rt.All$2, PackageName, PackagePath, Host)

	ctx := context.Background()
	rt.Repository.Init(ctx)

	console.LogKF(PackageName, "Router version:%s", PackageVersion)
	return r
}

func (rt *Router) Version(w http.ResponseWriter, r *http.Request) {
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

const modelHandler = `package $1

import (
	"net/http"
	"strconv"

	"github.com/cgalvisleon/elvis/console"
	"github.com/cgalvisleon/elvis/core"
	"github.com/cgalvisleon/elvis/generic"
	e "github.com/cgalvisleon/elvis/json"
	"github.com/cgalvisleon/elvis/linq"
	"github.com/cgalvisleon/elvis/msg"
	"github.com/cgalvisleon/elvis/response"
	"github.com/cgalvisleon/elvis/utility"
	"github.com/go-chi/chi"
)

var $2 *linq.Model

func Define$2() error {
	if err := defineSchema(); err != nil {
		return console.PanicE(err)
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
	$2.Trigger(linq.BeforeInsert, func(model *linq.Model, old, new *e.Json, data e.Json) error {
		return nil
	})
	$2.Trigger(linq.AfterInsert, func(model *linq.Model, old, new *e.Json, data e.Json) error {
		return nil
	})
	$2.Trigger(linq.BeforeUpdate, func(model *linq.Model, old, new *e.Json, data e.Json) error {
		return nil
	})
	$2.Trigger(linq.AfterUpdate, func(model *linq.Model, old, new *e.Json, data e.Json) error {
		return nil
	})
	$2.Trigger(linq.BeforeDelete, func(model *linq.Model, old, new *e.Json, data e.Json) error {
		return nil
	})
	$2.Trigger(linq.AfterDelete, func(model *linq.Model, old, new *e.Json, data e.Json) error {
		return nil
	})
	
	if err := core.InitModel($2); err != nil {
		return console.PanicE(err)
	}

	return nil
}

/**
*	Handler for CRUD data
 */
func Get$2ById(id string) (e.Item, error) {
	return $2.Select().
		Where($2.Column("_id").Eq(id)).
		First()
}

func Value$2ById(_default any, id, atrib string) *generic.Any {
	item, err := $2.Select(atrib).
		Where($2.Column("_id").Eq(id)).
		First()
	if err != nil {
		return &generic.Any{}
	}

	return item.Any(_default, atrib)
}

func UpSert$2(project_id, id string, data e.Json) (e.Item, error) {
	if !utility.ValidId(project_id) {
		return e.Item{}, console.AlertF(msg.MSG_ATRIB_REQUIRED, "project_id")
	}

	if !utility.ValidId(id) {
		return e.Item{}, console.AlertF(msg.MSG_ATRIB_REQUIRED, "_id")
	}

	id = utility.GenId(id)
	data["project_id"] = project_id
	data["_id"] = id
	return $2.Upsert(data).
		Where($2.Column("_id").Eq(id)).
		CommandOne()
}

func State$2(id, state string) (e.Item, error) {
	if !utility.ValidId(state) {
		return e.Item{}, console.AlertF(msg.MSG_ATRIB_REQUIRED, "state")
	}

	item, err := $2.Select("_state").
		Where($2.Column("_id").Eq(id)).
		First()
	if err != nil {
		return e.Item{}, err
	}

	if !item.Ok {
		return e.Item{}, console.Alert(msg.RECORD_NOT_FOUND)
	}

	old_state := item.Key("_state")
	if old_state == state {
		return e.Item{
			Ok: true,
			Result: e.Json{
				"message": msg.RECORD_NOT_UPDATE,
			}}, nil
	}

	return $2.Update(e.Json{
		"_state":   state,
	}).
		Where($2.Column("_id").Eq(id)).
		CommandOne()	
}

func Delete$2(id string) (e.Item, error) {
	return State$2(id, utility.FOR_DELETE)
}

func All$2(project_id, state, search string, page, rows int, _select string) (e.List, error) {	
	if state == "" {
		state = utility.ACTIVE
	}

	auxState := state

	if search != "" {
		return $2.Select(_select).
			Where($2.Column("project_id").In("-1", project_id)).
			And($2.Concat("NAME:", $2.Column("name"), "DESCRIPTION:", $2.Column("description"), "DATA:", $2.Column("_data"), ":").Like("%"+search+"%")).
			OrderBy($2.Column("name"), true).
			List(page, rows)
	} else if auxState == "*" {
		state = utility.FOR_DELETE

		return $2.Select(_select).
			Where($2.Column("_state").Neg(state)).
			And($2.Column("project_id").In("-1", project_id)).
			OrderBy($2.Column("name"), true).
			List(page, rows)
	} else if auxState == "0" {
		return $2.Select(_select).
			Where($2.Column("_state").In("-1", state)).
			And($2.Column("project_id").In("-1", project_id)).
			OrderBy($2.Column("name"), true).
			List(page, rows)
	} else {
		return $2.Select(_select).
			Where($2.Column("_state").Eq(state)).
			And($2.Column("project_id").In("-1", project_id)).
			OrderBy($2.Column("name"), true).
			List(page, rows)
	}
}

/**
* Router
**/
func (rt *Router) UpSert$2(w http.ResponseWriter, r *http.Request) {
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

func (rt *Router) Get$2ById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := Get$2ById(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, result)
}

func (rt *Router) State$2(w http.ResponseWriter, r *http.Request) {
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

func (rt *Router) Delete$2(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := Delete$2(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, result)
}

func (rt *Router) All$2(w http.ResponseWriter, r *http.Request) {
	project_id := r.URL.Query().Get("project_id")
	state := r.URL.Query().Get("state")
	search := r.URL.Query().Get("search")
	pageStr := r.URL.Query().Get("page")
	rowsStr := r.URL.Query().Get("rows")
	_select := r.URL.Query().Get("select")

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}

	rows, err := strconv.Atoi(rowsStr)
	if err != nil {
		rows = 30
	}

	result, err := All$2(project_id, state, search, page, rows, _select)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}

/** Copy this code to router.go
	// $2
	er.ProtectRoute(r, er.Get, "/$5/{id}", rt.Get$2ById, PackageName, PackagePath, Host)
	er.ProtectRoute(r, er.Post, "/$5", rt.UpSert$2, PackageName, PackagePath, Host)
	er.ProtectRoute(r, er.Put, "/$5/state/{id}", rt.State$2, PackageName, PackagePath, Host)
	er.ProtectRoute(r, er.Delete, "/$5/{id}", rt.Delete$2, PackageName, PackagePath, Host)
	er.ProtectRoute(r, er.Get, "/$5/all", rt.All$2, PackageName, PackagePath, Host)
**/

/** Copy this code to func initModel in model.go
	if err := Define$2(); err != nil {
		return console.PanicE(err)
	}
**/
`

const modelReadme = `### $1`

const modelEnvar = `APP=
PORT=3300
VERSION=0.0.0
COMPANY=Company
WEB=https://www.company.com

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

# AWS
AWS_REGION=
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=
AWS_SESSION_TOKEN=
`
