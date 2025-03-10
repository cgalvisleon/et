package create

const modelDockerfile = `# Versión de Go como argumento3
ARG GO_VERSION=1.23

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

const modelMain = `package main

import (
	"os"
	"os/signal"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/console"
	serv "$1/internal/services/$2"
)

func main() {
	envar.SetInt("port", 3000, "Port server", "PORT")
	envar.SetInt("rpc", 4200, "Port rpc server", "RPC_PORT")
	envar.SetStr("dbhost", "localhost", "Database host", "DB_HOST")
	envar.SetInt("dbport", 5432, "Database port", "DB_PORT")
	envar.SetStr("dbname", "", "Database name", "DB_NAME")
	envar.SetStr("dbuser", "", "Database user", "DB_USER")
	envar.SetStr("dbpass", "", "Database password", "DB_PASSWORD")
	envar.SetStr("dbapp", "Test", "Database app name", "DB_APP_NAME")

	srv, err := serv.New()
	if err != nil {
		console.Fatal(err)
	}

	go srv.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	srv.Close()
}
`

const modelApi = `package v1

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/utility"
	"github.com/dimiro1/banner"
	"github.com/go-chi/chi/v5"
	"github.com/mattn/go-colorable"
	pkg "$1/pkg/$2"	
)

func New() http.Handler {
	r := chi.NewRouter()

	err := pkg.LoadConfig()
	if err != nil {
		console.Alert(err)
	}

	_, err = cache.Load()
	if err != nil {
		console.Panic(err)
	}

	_, err = event.Load()
	if err != nil {
		console.Panic(err)
	}

	_pkg := &pkg.Router{
		Repository: &pkg.Controller{
		},
	}

	r.Mount(pkg.PackagePath, _pkg.Routes())

	return r
}

func Close() {
	jrpc.Close()
	cache.Close()
	event.Close()
}

func Banner() {
	time.Sleep(3 * time.Second)
	templ := utility.BannerTitle(pkg.PackageName, 4)
	banner.InitString(colorable.NewColorableStdout(), true, true, templ)
	fmt.Println()
}
`

const modelDbApi = `package v1

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/utility"
	_ "github.com/cgalvisleon/jdb/drivers/postgres"
	"github.com/cgalvisleon/jdb/jdb"
	"github.com/dimiro1/banner"
	"github.com/go-chi/chi/v5"
	"github.com/mattn/go-colorable"
	pkg "$1/pkg/$2"	
)

func New() http.Handler {
	r := chi.NewRouter()

	err := pkg.LoadConfig()
	if err != nil {
		console.Alert(err)
	}

	_, err = cache.Load()
	if err != nil {
		console.Panic(err)
	}

	_, err = event.Load()
	if err != nil {
		console.Panic(err)
	}

	db, err := jdb.Load()
	if err != nil {
		console.Panic(err)
	}

	_pkg := &pkg.Router{
		Repository: &pkg.Controller{
			Db: db,
		},
	}

	r.Mount(pkg.PackagePath, _pkg.Routes())

	return r
}

func Close() {
	jrpc.Close()
	cache.Close()
	event.Close()
}

func Banner() {
	time.Sleep(3 * time.Second)
	templ := utility.BannerTitle(pkg.PackageName, 4)
	banner.InitString(colorable.NewColorableStdout(), true, true, templ)
	fmt.Println()
}
`

const modelService = `package $2

import (
	"net/http"

	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/strs"
	"github.com/go-chi/chi/v5"
	v1 "$1/internal/services/$2/v1"
	"github.com/rs/cors"
)

type Server struct {
	http *http.Server
}

func New() (*Server, error) {
	server := Server{}

	port := envar.EnvarInt(3300, "PORT")
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

	return &server, nil
}

func (serv *Server) Close() {
	v1.Close()

	console.Log("Http", "Shutting down server...")
}

func (serv *Server) StartHttpServer() {
	if serv.http == nil {
		return
	}

	svr := serv.http
	console.Logf("Http", "Running on http://localhost%s", svr.Addr)
	console.Fatal(serv.http.ListenAndServe())
}

func (serv *Server) Background() {
	serv.StartHttpServer()

	v1.Banner()
}

func (serv *Server) Start() {
	go serv.StartHttpServer()

	v1.Banner()

	<-make(chan struct{})
}
`

const modelConfig = `package $1

import (
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/mistake"
)

func LoadConfig() error {
	StartRpcServer()

	// stage := envar.GetStr("local", "STAGE")
	// return defaultConfig(stage)
	return nil
}

func defaultConfig(stage string) error {
	name := "default"
	result, err := jrpc.CallItem("Module.Services.GetConfig", et.Json{
		"stage": stage,
		"name":  name,
	})
	if err != nil {
		return err
	}

	if !result.Ok {
		return mistake.Newf(jrpc.MSG_NOT_LOAD_CONFIG, stage, name)
	}

	cfg := result.Json("config")
	return config.Load(cfg)
}
`

const modelDbController = `package $1

import (
	"context"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/jdb/jdb"
)

type Controller struct {
	Db *jdb.DB
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
	// initModels(c.Db)
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

const modelEvent = `package $1

import (
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/console"
)

func initEvents() {
	err := event.Stack("<channel>", eventAction)
	if err != nil {
		console.Error(err)
	}

}

func eventAction(m event.EvenMessage) {
	data := m.Data

	console.Log("eventAction", data)
}
`

const modelData = `package $4

import (
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/dt"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mistake"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/jdb/jdb"
)

var $2 *jdb.Model

func Define$2(db *jdb.DB) error {
	if err := defineSchema(db); err != nil {
		return console.Panic(err)
	}

	if $2 != nil {
		return nil
	}

	$2 = jdb.NewModel(schema, "$3", 1)
	$2.DefineModel()
	$2.DefineAtribute("name", jdb.TypeDataText)
	$2.DefineIndex(true,
		"name",
	)
	$2.DefineEvent(jdb.EventInsert, func(model *jdb.Model, before et.Json, after et.Json) error {
		return nil
	})
	$2.DefineEvent(jdb.EventUpdate, func(model *jdb.Model, before et.Json, after et.Json) error {
		return nil
	})
	$2.DefineEvent(jdb.EventDelete, func(model *jdb.Model, before et.Json, after et.Json) error {
		return nil
	})

	if err := $2.Init(); err != nil {
		return console.Panic(err)
	}

	return nil
}

/**
* Get$2ById
* @param id string
* @return dt.Object, error
**/
func Get$2ById(id string) (dt.Object, error) {
	result := dt.Get(id)
	if result.Ok {
		return result, nil
	}

	return Up$2ById(id)
}

/**
* Up$2ById
* @param id string
* @return dt.Object, error
**/
func Up$2ById(id string) (dt.Object, error) {
	item, err := $2.
		Where(jdb.KEY).Eq(id).
		Data().
		One()
	if err != nil {
		return dt.Object{}, err
	}

	return dt.Up(id, item.Result), nil
}

/**
* Insert$2
* @param project string
* @param state string
* @param id string
* @param data et.Json
* @param client string
* @return dt.Object, error
**/
func Insert$2(project, state, id string, data et.Json, client string) (dt.Object, error) {
	if !utility.ValidStr(project, 0, []string{""}) {
		return dt.Object{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, "project")
	}

	if !utility.ValidStr(id, 0, []string{""}) {
		return dt.Object{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, jdb.KEY)
	}

	id = $2.GenId(id)
	current, err := $2.
		Where(jdb.KEY).Eq(id).
		Data().
		One()
	if err != nil {
		return dt.Object{}, err
	}

	if current.Ok {
		id := current.Str(jdb.KEY)
		result := dt.Up(id, current.Result)
		result.Ok = false
		return result, nil
	}

	now := utility.Now()
	data[jdb.CREATED_AT] = now
	data[jdb.UPDATED_AT] = now	
	data[jdb.STATUS] = state
	data[jdb.KEY] = id
	data["project"] = project
	data["created_by"] = client
	_, err = $2.Insert(data).
		One()
	if err != nil {
		return dt.Object{}, err
	}

	return Get$2ById(id)
}

/**
* UpSert$2
* @param project string
* @param id string
* @param data et.Json
* @param client string
* @return dt.Object, error
**/
func UpSert$2(project, id string, data et.Json, client string) (dt.Object, error) {
	current, err := Insert$2(project, utility.ACTIVE, id, data, client)
	if err != nil {
		return dt.Object{}, err
	}

	if current.Ok {
		return current, nil
	}

	current_state := current.Str(jdb.STATUS)
	if current_state != utility.ACTIVE {
		return dt.Object{}, console.Alertf(MSG_STATE_NOT_ACTIVE, current_state)
	}

	id = current.Str(jdb.KEY)
	now := utility.Now()
	data[jdb.UPDATED_AT] = now
	data["updated_by"] = client
	_, err = $2.Update(data).
		Where(jdb.KEY).Eq(id).
		One()
	if err != nil {
		return dt.Object{}, err
	}

	return Up$2ById(id)
}

/**
* State$2
* @param id string
* @param state string
* @param client string
* @return et.Item
* @return error
**/
func State$2(id, state, client string) (et.Item, error) {
	if !utility.ValidStr(state, 0, []string{""}) {
		return et.Item{}, mistake.Newf(msg.MSG_ATRIB_REQUIRED, jdb.STATUS)
	}

	current, err := $2.
		Where(jdb.KEY).Eq(id).
		Data(jdb.STATUS).
		One()
	if err != nil {
		return et.Item{}, err
	}

	if !current.Ok {
		return et.Item{}, console.Alertm(msg.RECORD_NOT_FOUND)
	}

	current_state := current.Key(jdb.STATUS)
	if current_state == state {
		return et.Item{Ok: true, Result: et.Json{"message": msg.RECORD_NOT_UPDATE}}, nil
	}

	result, err := $2.Update(et.Json{
		jdb.UPDATED_AT: utility.Now(),
		jdb.STATUS: state,
		"updated_by": client,
	}).
		Where(jdb.KEY).Eq(id).
		One()
	if err != nil {
		return et.Item{}, err
	}

	dt.Put(id, jdb.STATUS, state)

	return et.Item{
		Ok: result.Ok,
		Result: et.Json{
			"message": msg.RECORD_UPDATE,
		},
	}, nil
}

/**
* Delete$2
* @param id string
* @return et.Item
* @return error
**/
func Delete$2(id string) (et.Item, error) {
	if !utility.ValidStr(id, 0, []string{""}) {
		return et.Item{}, console.Alertf(msg.MSG_ATRIB_REQUIRED, jdb.KEY)
	}

	current, err := $2.
		Where(jdb.KEY).Eq(id).
		Data(jdb.KEY).
		One()
	if err != nil {
		return et.Item{}, err
	}

	if !current.Ok {
		return et.Item{}, console.Alertm(msg.RECORD_NOT_FOUND)
	}

	_, err = $2.Delete().
		Where(jdb.KEY).Eq(id).
		One()
	if err != nil {
		return et.Item{}, err
	}

	dt.Drop(id)

	return et.Item{
		Ok: true,
		Result: et.Json{
			"message": msg.RECORD_DELETE,
		},
	}, nil
}

/**
* Get$2
* @param search et.Json, page, rows int
* @return interface{}, error
**/
func Get$2(search et.Json, page, rows int) (interface{}, error) {
	search.Set("limit", et.Json{"page": page, "rows": rows})
	result, err := jdb.From($2).
		Query(search)
	if err != nil {
		return et.List{}, err
	}

	return result, nil
}

`

const modelDbHandler = `package $1

import (
	"net/http"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/jdb/jdb"
	"github.com/go-chi/chi/v5"
	"$3/internal/models/$4"
)

/**
* upSert$2
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (rt *Router) upSert$2(w http.ResponseWriter, r *http.Request) {
	body, _ := response.GetBody(r)
	project := body.Str("project")
	id := body.Str("id")
	user := body.Str("user")

	result, err := $4.UpSert$2(project, id, body, user)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}

/**
* get$2ById
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (rt *Router) get$2ById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := $4.Get$2ById(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}

/**
* state$2
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (rt *Router) state$2(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, _ := response.GetBody(r)
	state := body.Str(jdb.STATUS)
	client := claim.GetClientName(r)

	result, err := $4.State$2(id, state, client)
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

	result, err := $4.Delete$2(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, result)
}

/** Copy this code to router.go
	// $2
	router.Protect(r, router.Get, "/assets/{id}", rt.get$2ById, PackageName, PackagePath, host)
	router.Protect(r, router.Post, "/assets", rt.upSert$2, PackageName, PackagePath, host)
	router.Protect(r, router.Put, "/assets/{id}", rt.state$2, PackageName, PackagePath, host)
	router.Protect(r, router.Delete, "/assets/{id}", rt.delete$2, PackageName, PackagePath, host)
	router.Protect(r, router.Get, "/assets/", rt.query$2, PackageName, PackagePath, host)
**/

/** Copy this code to func initModel in model.go
	if err := Define$2(db); err != nil {
		return console.Panic(err)
	}
**/
`

const modelHandler = `package $1

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/response"
	"github.com/go-chi/chi/v5"
)


/**
* Get$2
* @param id string
* @return et.Item
* @return error
**/
func Get$2(id string) (et.Item, error) {
	
	return et.item{}, nil
}


/**
* get$2
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (rt *Router) get$2(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := Get$2(id)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, result)
}

/** Copy this code to router.go
	// $2
	router.Protect(r, router.Get, "/assets/{id}", rt.get$2, PackageName, PackagePath, host)	
**/
`

const modelModel = `package $1

import (
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/jdb/jdb"
	"$3/internal/data/$2"	
)

func initModels(db *jdb.DB) error {
	if err := $2.Define$4(db); err != nil {
		return console.Panic(err)
	}

	return nil
}
`

const modelSchema = `package $1

import (
	"github.com/cgalvisleon/jdb/jdb"
)

var schema *jdb.Schema

func defineSchema(db *jdb.DB) error {
	if schema != nil {
		return nil
	}

	var err error
	schema, err = jdb.NewSchema(db, "$1")
	if err != nil {
		return err
	}

	return nil
}
`

const modelhRpc = `package $1

import (
	"net/rpc"

	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
)

var initRpc bool

type Service et.Item

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
	MSG_VALUE_REQUIRED 	 = "Atributo requerido (%s) value:%s"
	MSG_STATE_NOT_ACTIVE = "Estado no activo (%s)"  
	RECORD_NOT_FOUND     = "Registro no encontrado"
	RECORD_NOT_UPDATE    = "Registro no actualizado"
)
`

const modelDbRouter = `package $1

import (
	"context"
	"net/http"
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/middleware"
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
	defaultHost := strs.Format("http://%s", HostName)
	var host = strs.Format("%s:%d", envar.EnvarStr(defaultHost, "HOST"), envar.EnvarInt(3300, "PORT"))

	r := chi.NewRouter()

	router.Public(r, router.Get, "/version", rt.version, PackageName, PackagePath, host)
	router.Protect(r, router.Get, "/routes", rt.routes, PackageName, PackagePath, host)
	// $2
	router.Protect(r, router.Get, "/{id}", rt.get$2ById, PackageName, PackagePath, host)
	router.Protect(r, router.Post, "/", rt.upSert$2, PackageName, PackagePath, host)
	router.Protect(r, router.Put, "/{id}", rt.state$2, PackageName, PackagePath, host)
	router.Protect(r, router.Delete, "/{id}", rt.delete$2, PackageName, PackagePath, host)
	router.Protect(r, router.Get, "/", rt.query$2, PackageName, PackagePath, host)

	ctx := context.Background()
	rt.Repository.Init(ctx)
	middleware.SetServiceName(PackageName)

	console.Logf(PackageName, "Router version:%s", PackageVersion)

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

func (rt *Router) routes(w http.ResponseWriter, r *http.Request) {
	_routes := router.GetRoutes()
	routes := []et.Json{}
	for _, route := range _routes {
		routes = append(routes, et.Json{
			"method": route.Str("method"),
			"path":   route.Str("path"),
		})
	}

	result := et.Items{
		Ok:     true,
		Count:  len(routes),
		Result: routes,
	}

	response.ITEMS(w, r, http.StatusOK, result)
}
`

const modelRouter = `package $1

import (
	"context"
	"net/http"
	"os"

	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/et"
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
	defaultHost := strs.Format("http://%s", HostName)
	var host = strs.Format("%s:%d", envar.EnvarStr(defaultHost, "HOST"), envar.EnvarInt(3300, "PORT"))

	r := chi.NewRouter()

	router.Public(r, router.Get, "/version", rt.version, PackageName, PackagePath, host)
	router.Protect(r, router.Get, "/routes", rt.routes, PackageName, PackagePath, host)
	// $2
	router.Protect(r, router.Post, "/", rt.get$2, PackageName, PackagePath, host)
	
	ctx := context.Background()
	rt.Repository.Init(ctx)
	middleware.SetServiceName(PackageName)

	console.Logf(PackageName, "Router version:%s", PackageVersion)

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

func (rt *Router) routes(w http.ResponseWriter, r *http.Request) {
	_routes := router.GetRoutes()
	routes := []et.Json{}
	for _, route := range _routes {
		routes = append(routes, et.Json{
			"method": route.Str("method"),
			"path":   route.Str("path"),
		})
	}

	result := et.Items{
		Ok:     true,
		Count:  len(routes),
		Result: routes,
	}

	response.ITEMS(w, r, http.StatusOK, result)
}
`

const modelRpc = `package $1

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/console"
)

type Services struct{}

func StartRpcServer() {
	pkg, err := jrpc.Load(PackageName)
	if err != nil {
		console.Panic(err)
	}

	services := new(Services)
	err = jrpc.Mount(services)
	if err != nil {
		console.Fatal(err)
	}

	go pkg.Start()
}

func (c *Services) Version(require et.Json, response *et.Item) error {
	company := envar.EnvarStr("", "COMPANY")
	web := envar.EnvarStr("", "WEB")
	version := envar.EnvarStr("", "VERSION")
	help := envar.EnvarStr("", "RPC_HELP")
	response.Ok = true
	response.Result = et.Json{
		"methos":  "RPC",
		"version": version,
		"service": PackageName,
		"host":    HostName,
		"company": company,
		"web":     web,
		"help":    help,
	}

	return console.Rpc(response.ToString())
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

const modelReadme = `
## Project $1

## Create project

$2
go mod init github.com/redist/$1
$2

## Dependencias

$2
go get github.com/cgalvisleon/et@v0.0.8
go get github.com/cgalvisleon/jdb@v1.0.3
$2

## Create

$2
go run github.com/cgalvisleon/et/cmd/create go
$2

## Run

$2
gofmt -w . && go run ./cmd/$1 -port 3600 -rpc 4600
$2

`

const modelEnvar = `APP=
PORT=3300
VERSION=0.0.0
COMPANY=Company
PATH_URL=
WEB=https://www.home.com
PRODUCTION=false
PATH_URL=/api/$1
HOST=http://localhost
# HOST=http://host.docker.internal

# RPC
RPC_HOST=localhost
RPC_PORT=4200

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
NATS_USER=
NATS_PASSWORD=

# SESSION
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
