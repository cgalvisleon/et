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
	envar.SetStr("dbapp", "Test", "Database app name", "DB_APP_NAME")

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
		logs.Logf("Http", "Running on http://localhost%s", svr.Addr)
		logs.Fatal(serv.http.ListenAndServe())
	}()

	go func() {
		if serv.rpc == nil {
			return
		}

		svr := *serv.rpc
		logs.Logf("RPC", "Running on tcp:localhost:%s", svr.Addr().String())
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

	"github.com/cgalvisleon/et/linq"
	_ "github.com/cgalvisleon/et/lib/pg"
	"github.com/cgalvisleon/et/utility"
	"github.com/dimiro1/banner"
	"github.com/go-chi/chi/v5"
	"github.com/mattn/go-colorable"
	pkg "$1/pkg/$2"	
)

var db *linq.DB

func New() http.Handler {
	r := chi.NewRouter()

	var err error
	db, err = linq.Core()
	if err != nil {
		panic(err)
	}
	
	_pkg := &pkg.Router{
		Repository: &pkg.Controller{
			Origin: db,
		},
	}

	r.Mount(pkg.PackagePath, _pkg.Routes())

	return r
}

func Close() {
	if db != nil {
		db.Close()
	}
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
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
)

func initModels(db *linq.DB) error {
	if err := Define$2(db); err != nil {
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
		$2 = linq.NewSchema("$3", "")
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
	MSG_ATRIB_REQUIRED   = "Atributo requerido (%s)"
	MSG_VALUE_REQUIRED 	 = "Atributo requerido (%s) value:%s"
	MSG_STATE_NOT_ACTIVE = "Estado no activo (%s)"
	RECORD_NOT_FOUND     = "Registro no encontrado"
	RECORD_NOT_UPDATE    = "Registro no actualizado"
)
`

const modelDbController = `package $1

import (
	"context"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/linq"
)

type Controller struct {
	Origin *linq.DB
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
	initModels(c.Origin)
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
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/linq"
)

type Controller struct {
	Origin *linq.DB
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

	logs.Logf(PackageName, "Router version:%s", PackageVersion)
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

	logs.Logf(PackageName, "Router version:%s", PackageVersion)
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

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
	"github.com/go-chi/chi/v5"
)

var $2 *linq.Model

func Define$2(db *linq.DB) error {
	if err := defineSchema(); err != nil {
		return logs.Panic(err)
	}

	if $2 != nil {
		return nil
	}

	$2 = linq.NewModel($3, "$4", "Tabla", 1)
	$2.DefineColumn("date_make", "", linq.TpDate, "NOW()")
	$2.DefineColumn("date_update", "", linq.TpDate, "NOW()")
	$2.DefineColumn("project_id", "", linq.TpKey, "-1")
	$2.DefineColumn("_state", "", linq.TpStatus, utility.ACTIVE)
	$2.DefineColumn("_id", "", linq.TpKey, "-1")	
	$2.DefineColumn("name", "", linq.TpText, "")
	$2.DefineColumn("description", "", linq.TpMemo, "")
	$2.DefineColumn("_data", "", linq.TpSource, "{}")
	$2.DefineColumn("index", "", linq.TpSerie, 0)
	$2.DefinePrimaryKey([]string{"_id"})
	$2.DefineIndex([]string{
		"date_make",
		"date_update",
		"_state",
		"project_id",
		"name",
		"index",
	}, true)
	$2.DefineRequired([]linq.ColRequired{
		{
			Name:    "name",
			Message: "Atributo requerido - (name)",
		},
	})
	$2.DefineIntegrity(true)
	$2.DefineTrigger(linq.BeforeInsert, func(model *linq.Model, values *linq.Values) error {
		return nil
	})
	$2.DefineTrigger(linq.AfterInsert, func(model *linq.Model, values *linq.Values) error {
		return nil
	})
	$2.DefineTrigger(linq.BeforeUpdate, func(model *linq.Model, values *linq.Values) error {
		return nil
	})
	$2.DefineTrigger(linq.AfterUpdate, func(model *linq.Model, values *linq.Values) error {
		return nil
	})
	$2.DefineTrigger(linq.BeforeDelete, func(model *linq.Model, values *linq.Values) error {
		return nil
	})
	$2.DefineTrigger(linq.AfterDelete, func(model *linq.Model, values *linq.Values) error {
		return nil
	})
	$2.OnListener = func(data et.Json) {
		logs.Debug(data.ToString())
	}
	
	if err := $2.Init(db); err != nil {
		return logs.Panic(err)
	}

	return nil
}

/**
* Get$2ById
* @param id string
* @return et.Item
* @return error
**/
func Get$2ById(id string) (et.Item, error) {	
	if !utility.ValidId(id) {
		return et.Item{}, logs.Nerrorf(msg.MSG_ATRIB_REQUIRED, "_id")
	}

	item, err := $2.Data().
		Where($2.Col("_id").Eq(id)).
		First()
	if err != nil {
		return et.Item{}, err
	}
	
	return item, nil	
}

/**
* Insert$2
* @param project_id string
* @param state string
* @param id string
* @param data et.Json
* @return et.Item
* @return error
**/
func Insert$2(project_id, state, id string, data et.Json, user_id string) (et.Item, error) {
	if !utility.ValidId(project_id) {
		return et.Item{}, logs.Alertf(MSG_ATRIB_REQUIRED, "project_id")
	}

	if !utility.ValidId(id) {
		return et.Item{}, logs.Alertf(MSG_ATRIB_REQUIRED, "_id")
	}

	id = utility.GenId(id)
	item, err := $2.Data("_state", "_id").
		Where($2.Col("_id").Eq(id)).
		First()
	if err != nil {
		return et.Item{}, err
	}

	if item.Ok {
		return et.Item{
			Ok: false,
			Result: item.Result,
		}, nil
	}
	
	data["project_id"] = project_id
	data["_state"] = state
	data["_id"] = id
	data["user_id"] = user_id
	item, err = $2.Insert(data).		
		Exec()
	if err != nil {
		return et.Item{}, err
	}

	return item, nil
}

/**
* UpSert$2
* @param project_id string
* @param id string
* @param data et.Json
* @param user_id string
* @return et.Item
* @return error
**/
func UpSert$2(project_id, id string, data et.Json, user_id string) (et.Item, error) {
	item, err := Insert$2(project_id, utility.ACTIVE, id, data, user_id)
	if err != nil {
		return et.Item{}, err
	}

	if item.Ok {
		item, err = Get$2ById(id)
		if err != nil {
			return et.Item{}, err
		}

		return item, nil
	}

	current_state := item.Key("_state")
	if current_state != utility.ACTIVE {
		return et.Item{}, logs.Alertf(MSG_STATE_NOT_ACTIVE, current_state)
	}
	
	data["user_id"] = user_id	
	item, err = $2.Update(data).
		Where($2.Col("_id").Eq(id)).
		Exec()
	if err != nil {
		return et.Item{}, err
	}

	item, err = Get$2ById(id)
	if err != nil {
		return et.Item{}, err
	}

	return item, nil
}

/**
* State$2
* @param id string
* @param state string
* @return et.Item
* @return error
**/
func State$2(id, state string) (et.Item, error) {
	if !utility.ValidId(state) {
		return et.Item{}, logs.Alertf(MSG_ATRIB_REQUIRED, "state")
	}

	item, err := $2.Data("_state").
		Where($2.Col("_id").Eq(id)).
		First()
	if err != nil {
		return et.Item{}, err
	}

	if !item.Ok {
		return et.Item{}, logs.Alertm(msg.RECORD_NOT_FOUND)
	}

	old_state := item.Key("_state")
	if old_state == state {
		return et.Item{
			Ok: true,
			Result: et.Json{
				"message": msg.RECORD_NOT_UPDATE,
			}}, nil
	}

	return $2.Update(et.Json{
		"_state":   state,
	}).
		Where($2.Col("_id").Eq(id)).
		Exec()	
}

/**
* Delete$2
* @param id string
* @return et.Item
* @return error
**/
func Delete$2(id string) (et.Item, error) {
	return State$2(id, utility.FOR_DELETE)
}

/**
* All$2
* @param project_id string
* @param state string
* @param search string
* @param page int
* @param rows int
* @param _select string
* @return et.List
* @return error
**/
func All$2(project_id, state, search string, page, rows int, _select string) (et.List, error) {	
	if state == "" {
		state = utility.ACTIVE
	}

	auxState := state

	if search != "" {
		return $2.Data(_select).
			Where($2.Col("project_id").In("-1", project_id)).
			And(linq.Concat("NAME:", $2.Col("name"), "DESCRIPTION:", $2.Col("description"), "DATA:", $2.Col("_data"), ":").Like("%"+search+"%")).
			OrderBy(true, $2.Col("name")).
			List(page, rows)
	} else if auxState == "*" {
		state = utility.FOR_DELETE

		return $2.Data(_select).
			Where($2.Col("_state").Neg(state)).
			And($2.Col("project_id").In("-1", project_id)).
			OrderBy(true, $2.Col("name")).
			List(page, rows)
	} else if auxState == "0" {
		return $2.Data(_select).
			Where($2.Col("_state").In("-1", state)).
			And($2.Col("project_id").In("-1", project_id)).
			OrderBy(true, $2.Col("name")).
			List(page, rows)
	} else {
		return $2.Data(_select).
			Where($2.Col("_state").Eq(state)).
			And($2.Col("project_id").In("-1", project_id)).
			OrderBy(true, $2.Col("name")).
			List(page, rows)
	}
}

/**
* insert$2
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (rt *Router) upSert$2(w http.ResponseWriter, r *http.Request) {
	body, _ := response.GetBody(r)
	project_id := body.Str("project_id")
	id := body.Str("id")
	user_id := body.Str("user_id")

	result, err := UpSert$2(project_id, id, body, user_id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, result)
}

/**
* get$2ById
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (rt *Router) get$2ById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := Get$2ById(id)
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

/**
* delete$2
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (rt *Router) delete$2(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := Delete$2(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, result)
}

/**
* all$2
* @param w http.ResponseWriter
* @param r *http.Request
**/
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
	router.Protect(r, router.Get, "/$5/{id}", rt.get$2ById, PackageName, PackagePath, host)
	router.Protect(r, router.Post, "/$5", rt.upSert$2, PackageName, PackagePath, host)
	router.Protect(r, router.Put, "/$5/state/{id}", rt.state$2, PackageName, PackagePath, host)
	router.Protect(r, router.Delete, "/$5/{id}", rt.delete$2, PackageName, PackagePath, host)
	router.Protect(r, router.Get, "/$5/all", rt.all$2, PackageName, PackagePath, host)
**/

/** Copy this code to func initModel in model.go
	if err := Define$2(db); err != nil {
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
	router.Protect(r, router.Post, "/$3", rt.$2, PackageName, PackagePath, host)	
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
WEB=https://www.home.com
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
      - "WEB=https://www.home.com"
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
