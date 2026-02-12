package ettp

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/response"
	rt "github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/utility"
)

type Proxy struct {
	server      *Server                `json:"-"`
	pkg         *Package               `json:"-"`
	Id          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Path        string                 `json:"path"`
	Solver      string                 `json:"solver"`
	Kind        TypeApi                `json:"kind"`
	PackageName string                 `json:"package_name"`
	proxy       *httputil.ReverseProxy `json:"-"`
	RemoteHost  string                 `json:"remote_host"`
	RemotePort  int                    `json:"remote_port"`
	LocalPort   int                    `json:"local_port"`
	listener    net.Listener           `json:"-"`
	wg          sync.WaitGroup         `json:"-"`
	cancel      context.CancelFunc     `json:"-"`
	started     bool                   `json:"-"`
}

/**
* NewProxy
* @param id, path, name, description, solver, packageName string, s *Server
* @return *Proxy
**/
func NewProxy(id, path, name, description, solver, packageName string, s *Server) *Proxy {
	id = utility.GenId(id)
	return &Proxy{
		Id:          id,
		Name:        name,
		Description: description,
		Path:        path,
		Solver:      solver,
		PackageName: packageName,
		server:      s,
	}
}

/**
* ToJson
* @return et.Json
**/
func (p *Proxy) ToJson() et.Json {
	if p.Kind == TpPortForward {
		return et.Json{
			"id":           p.Id,
			"name":         p.Name,
			"description":  p.Description,
			"kind":         p.Kind.String(),
			"remote_host":  p.RemoteHost,
			"remote_port":  p.RemotePort,
			"local_port":   p.LocalPort,
			"package_name": p.PackageName,
		}
	}

	return et.Json{
		"id":           p.Id,
		"name":         p.Name,
		"description":  p.Description,
		"kind":         p.Kind.String(),
		"path":         p.Path,
		"solver":       p.Solver,
		"package_name": p.PackageName,
	}
}

/**
* handlerConnection
* @param clientConn *net.Conn
**/
func (s *Proxy) handlerConnection(clientConn net.Conn) {
	dialer := net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	remoteConn, err := dialer.Dial("tcp", s.Solver)
	if err != nil {
		logs.Alertf("Error conectando al destino: %s", err.Error())
		clientConn.Close()
		return
	}

	go io.Copy(remoteConn, clientConn)
	go io.Copy(clientConn, remoteConn)
}

/**
* setProxy
* @param id, path, name, description, solver, packageName string, save bool
* @return *Proxy, error
**/
func (s *Server) SetProxy(id, path, name, description, solver, packageName string, save bool) (*Proxy, error) {
	if !utility.ValidStr(path, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "path")
	}

	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}

	if !utility.ValidStr(solver, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "solver")
	}

	if !utility.ValidStr(packageName, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "package_name")
	}

	if !utility.ValidStr(id, 0, []string{"", "new", "-1"}) {
		id = utility.UUID()
	}

	confirm := func(action string) {
		logs.Logf(packageName, `[%s] %s -> %s | %s | %s`, action, name, path, solver, description)
	}

	result, ok := s.proxys[path]
	if !ok {
		result = NewProxy(id, path, name, description, solver, packageName, s)
		result.Kind = TpProxy
		s.proxys[path] = result
		confirm("SET")
	} else {
		result.Id = id
		result.Name = name
		result.Description = description
		result.Kind = TpProxy
		result.Solver = solver
		result.PackageName = packageName
		confirm("RESET")
	}

	target, err := url.Parse(solver)
	if err != nil {
		return nil, err
	}

	result.proxy = httputil.NewSingleHostReverseProxy(target)
	result.setPakage(packageName)

	if save {
		if err := s.save(); err != nil {
			logs.Alertf("Failed to save routes: %s", err.Error())
		}
	}

	return result, nil
}

/**
* handlerReverseProxy
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) handlerReverseProxy(w http.ResponseWriter, r *http.Request) {
	/* Begin telemetry */
	metric := middleware.NewMetric(r)
	w.Header().Set("ServiceId", metric.ServiceId)
	ctx := context.WithValue(r.Context(), MetricKey, metric)
	r = r.WithContext(ctx)

	request := et.Json{
		"method":   r.Method,
		"url":      r.URL,
		"host":     r.Host,
		"path":     r.URL.Path,
		"rawquery": r.URL.RawQuery,
		"header":   r.Header,
		"body":     r.Body,
	}

	proxy := s.getProxyByPath(r.URL.Path)
	if proxy == nil {
		request["proxy"] = "not found"
		logs.Debug(packageName, "proxy:", request.ToString())
		s.notFoundHandler.ServeHTTP(w, r)
		return
	}

	if s.debug {
		request["proxy"] = proxy.Solver
		logs.Debug(packageName, "proxy:", request.ToString())
	}

	if proxy.Kind == TpPortForward {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_IS_PORTFORWARD)
		return
	}

	/* Call search time since begin */
	metric.CallSearchTime()
	metric.SetPath(proxy.Solver)
	proxy.ServeHTTP(w, r)
}

/**
* setPortForward
* @param id, path, name, description, remoteHost string, remotePort, localPort int, packageName string, save bool
* @return *Proxy, error
**/
func (s *Server) SetPortForward(id, name, description, remoteHost string, remotePort, localPort int, packageName string, save bool) (*Proxy, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "name")
	}

	if !utility.ValidStr(remoteHost, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "remote_host")
	}

	if !utility.ValidInt(remotePort, []int{0, 5432, 6379, 4222, 443}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "remote_port")
	}

	if !utility.ValidInt(localPort, []int{0, 5432, 6379, 4222, 443}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "local_port")
	}

	if !utility.ValidStr(packageName, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "package_name")
	}

	if !utility.ValidStr(id, 0, []string{"", "new", "-1"}) {
		id = utility.UUID()
	}

	confirm := func(action string) {
		logs.Logf(packageName, `[%s] %s -> %d -> %s:%d`, action, name, localPort, remoteHost, remotePort)
	}

	remoteAddr := fmt.Sprintf("%s:%d", remoteHost, remotePort)
	port := fmt.Sprintf(":%d", localPort)
	result, ok := s.proxys[port]
	if !ok {
		result = NewProxy(id, port, name, description, remoteAddr, packageName, s)
		result.Kind = TpPortForward
		result.RemoteHost = remoteHost
		result.RemotePort = remotePort
		result.LocalPort = localPort
		s.proxys[port] = result
		confirm("SET")
	} else {
		result.Id = id
		result.Name = name
		result.Description = description
		result.Kind = TpPortForward
		result.RemoteHost = remoteHost
		result.RemotePort = remotePort
		result.LocalPort = localPort
		result.PackageName = packageName
		confirm("RESET")
	}

	result.StartPortForward()

	if save {
		if err := s.save(); err != nil {
			logs.Alertf("Failed to save routes: %s", err.Error())
		}
	}

	return result, nil
}

/**
* StartPortForward
* @return error
**/
func (s *Proxy) StartPortForward() error {
	if s.Kind != TpPortForward {
		return fmt.Errorf(MSG_INVALID_KIND, "portforward")
	}

	if s.started {
		return nil
	}

	if s.listener != nil {
		return nil
	}

	var ctx context.Context
	ctx, s.cancel = context.WithCancel(context.Background())

	ln, err := net.Listen("tcp", s.Path)
	if err != nil {
		return err
	}
	s.listener = ln

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			for {
				conn, err := ln.Accept()
				if err != nil {
					select {
					case <-ctx.Done():
						return // Se pidió detener
					default:
						logs.Logf(packageName, "Error aceptando conexión: %s", err.Error())
						continue
					}
				}

				go s.handlerConnection(conn)
			}
		}
	}()

	s.started = true
	logs.Logf(packageName, "Port-forwarding iniciado: %s -> %s", s.Path, s.Solver)
	return nil
}

/**
* StopPortForward
* @return error
**/
func (s *Proxy) StopPortForward() error {
	if s.Kind != TpPortForward {
		return fmt.Errorf(MSG_INVALID_KIND, "portforward")
	}

	if !s.started {
		return nil
	}

	logs.Logf(packageName, "Deteniendo port-forward...")
	s.cancel()
	s.listener.Close()
	s.wg.Wait()
	s.started = false
	logs.Logf(packageName, "Port-forward detenido.")
	return nil
}

/**
* ResetPortForward
* @return error
**/
func (s *Proxy) ResetPortForward() error {
	if s.Kind != TpPortForward {
		return fmt.Errorf(MSG_INVALID_KIND, "portforward")
	}

	if err := s.StopPortForward(); err != nil {
		return err
	}

	if err := s.StartPortForward(); err != nil {
		return err
	}

	return nil
}

/**
* ServeHTTP
* @param w http.ResponseWriter, r *http.Request
**/
func (s *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(w, r)
}

/**
* getProxyById
* @param id string
* @return *Proxy
**/
func (s *Server) getProxyById(id string) *Proxy {
	for _, proxy := range s.proxys {
		if proxy.Id == id {
			return proxy
		}
	}

	return nil
}

/**
* getProxyByPath
* @param path string
* @return *Proxy
**/
func (s *Server) getProxyByPath(path string) *Proxy {
	for _, proxy := range s.proxys {
		if proxy.Path == path {
			return proxy
		}
	}

	return nil
}

/**
* setPakage
* @param packageName string
* @return *Proxy
**/
func (s *Proxy) setPakage(packageName string) *Proxy {
	if len(packageName) == 0 {
		return s
	}

	if s.PackageName == packageName {
		return s
	}

	old := getPackageByName(s.server, s.PackageName)
	if old != nil {
		old.deleteProxy(s)
	}

	pkg := getPackageByName(s.server, packageName)
	if pkg == nil {
		pkg = newPakage(s.server, packageName)
	}

	s.PackageName = packageName
	s.pkg = pkg
	pkg.addProxy(s)

	return s
}

/**
* mountProxy
* @param proxy *Proxy
**/
func (s *Server) mountProxy(proxy *Proxy) {
	if proxy.Kind == TpPortForward {
		s.SetPortForward(
			proxy.Id,
			proxy.Name,
			proxy.Description,
			proxy.RemoteHost,
			proxy.RemotePort,
			proxy.LocalPort,
			proxy.PackageName,
			false,
		)
	} else {
		s.SetProxy(
			proxy.Id,
			proxy.Path,
			proxy.Name,
			proxy.Description,
			proxy.Solver,
			proxy.PackageName,
			false,
		)
	}
}

/**
* DeleteProxyById
* @param id string
* @return error
**/
func (s *Server) DeleteProxyById(id string, save bool) error {
	path := ""
	for _, proxy := range s.proxys {
		if proxy.Id == id {
			path = proxy.Path
			break
		}
	}

	proxy, ok := s.proxys[path]
	if !ok {
		return fmt.Errorf(MSG_ROUTE_NOT_FOUND)
	}

	pkg := proxy.pkg
	if pkg != nil {
		pkg.deleteProxyById(id)
	}
	delete(s.proxys, path)

	if save {
		if err := s.save(); err != nil {
			logs.Alertf("Failed to save routes: %s", err.Error())
		}
	}

	return nil
}

/**
* listProxies
* @return et.Items
**/
func (s *Server) listProxies() et.Items {
	items := et.Items{Result: []et.Json{}}

	for _, proxy := range s.proxys {
		items.Add(proxy.ToJson())
	}

	return items
}

/**
* GetProxys
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) GetProxys(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	result := s.listProxies()
	metric.ITEMS(w, r, http.StatusOK, result)
}

/**
* GetProxysById
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) GetProxysById(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	id := r.PathValue("id")
	result := s.getProxyById(id)
	if result == nil {
		metric.HTTPError(w, r, http.StatusNotFound, MSG_PROXY_NOT_FOUND)
		return
	}

	metric.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result.ToJson(),
	})
}

/**
* SetProxys
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) SetProxys(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	result := et.Items{Result: []et.Json{}}
	body, _ := response.GetArray(r)
	n := len(body)
	for i, item := range body {
		id := item.ValStr("", "id")
		name := item.Str("name")
		path := item.Str("path")
		description := item.Str("description")
		solver := item.Str("solver")
		packageName := item.Str("package_name")
		saved := i == n-1
		proxy, err := s.SetProxy(id, path, name, description, solver, packageName, saved)
		if err != nil {
			metric.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		result.Add(proxy.ToJson())
		event.Publish(rt.EVENT_SET_ROUTER, et.Json{
			"id":           proxy.Id,
			"path":         proxy.Path,
			"name":         proxy.Name,
			"description":  proxy.Description,
			"solver":       proxy.Solver,
			"package_name": proxy.PackageName,
		})
	}

	metric.ITEMS(w, r, http.StatusOK, result)
}

/**
* SetPortForwards
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) SetPortForwards(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	result := et.Items{Result: []et.Json{}}
	body, _ := response.GetArray(r)
	n := len(body)
	for i, item := range body {
		id := item.ValStr("", "id")
		name := item.Str("name")
		description := item.Str("description")
		remoteHost := item.Str("remote_host")
		remotePort := item.Int("remote_port")
		localPort := item.Int("local_port")
		packageName := item.Str("package_name")
		saved := i == n-1
		portForward, err := s.SetPortForward(id, name, description, remoteHost, remotePort, localPort, packageName, saved)
		if err != nil {
			metric.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		result.Add(portForward.ToJson())
		event.Publish(rt.EVENT_SET_ROUTER, et.Json{
			"id":           portForward.Id,
			"path":         portForward.Path,
			"name":         portForward.Name,
			"description":  portForward.Description,
			"solver":       portForward.Solver,
			"package_name": portForward.PackageName,
		})
	}

	metric.ITEMS(w, r, http.StatusOK, result)
}

/**
* DeleteProxys
* @params w http.ResponseWriterz
* @params r *http.Request
**/
func (s *Server) DeleteProxys(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	id := r.PathValue("id")
	err := s.DeleteProxyById(id, true)
	if err != nil {
		metric.HTTPError(w, r, http.StatusNotFound, err.Error())
		return
	}

	event.Publish(rt.EVENT_REMOVE_ROUTER, et.Json{
		"id": id,
	})
	metric.ITEM(w, r, http.StatusOK, et.Item{
		Ok: true,
		Result: et.Json{
			"message": MSG_PROXY_DELETE,
		},
	})
}
