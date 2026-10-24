package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/tcp"
)

/**
* Prueba todos los métodos públicos de tcp contra un servidor y clientes
* reales en 127.0.0.1, y escribe los resultados en test/tcp/result/.
*
* Uso (desde la raíz del repo):
*   go run ./test/tcp          // o con -race: go run -race ./test/tcp
* Sale con código 1 si algún caso falla o si queda un método público sin probar.
**/

const resultDir = "test/tcp/result"

// publicAPI: todos los métodos públicos de tcp (catalog.go).
var publicAPI = []string{
	"NewClient", "Client.ToJson", "Client.ID", "Client.Name", "Client.To", "Client.ServerID", "Client.ServerName", "Client.ConnectTo", "Client.Close", "Client.SetTimeout",
	"Client.Send", "Client.Request",
	"Client.SendStream", "Client.RequestStream", "Client.ReceiveStream",
	"Client.OnConnect", "Client.OnDisconnect", "Client.OnSend", "Client.OnRequest",
	"Client.OnSendStream", "Client.OnRequestStream", "Client.OnReceiveStream",
	"NewServer", "Server.ToJson", "Server.ID", "Server.Name", "Server.SetName", "Server.Listen", "Server.Addr", "Server.Close", "Server.Count", "Server.SetTimeout",
	"Server.OnConnect", "Server.OnDisconnect", "Server.OnSend", "Server.OnRequest",
	"Server.OnSendStream", "Server.OnRequestStream", "Server.OnReceiveStream",
	"StreamWriter.Write", "StreamWriter.Close", "StreamWriter.CloseWithError", "StreamWriter.CloseAndReceive",
	"StreamReader.Read", "StreamReader.Close",
	"NewHub", "Hub.ID", "Hub.Name", "Hub.ToJson", "Hub.LoadServer", "Hub.DownServer", "Hub.Servers", "Hub.Metrics",
	"Hub.ConnectTo", "Hub.Disconnect", "Hub.Clients", "Hub.Send", "Hub.SendTo",
	"Hub.OnMessage", "Hub.OnConnect", "Hub.OnDisconnect", "Hub.Heartbeat", "Hub.Status",
	"Hub.SetTimeoutServer", "Hub.SetTimeoutClient", "Hub.Close",
	"Client.SetSecurity", "Server.SetSecurity", "Hub.SetSecurity", "NewTLS",
}

type testCase struct {
	Name    string   `json:"name"`
	Methods []string `json:"methods"`
	run     func(r *report) error
}

type caseResult struct {
	Name     string   `json:"name"`
	Methods  []string `json:"methods"`
	Ok       bool     `json:"ok"`
	Duration string   `json:"duration"`
	Error    string   `json:"error,omitempty"`
	Details  []string `json:"details"`
}

/**
* report: Acumula las observaciones de un caso (se escriben en el resultado).
**/
type report struct {
	mu      sync.Mutex
	details []string
}

/**
* logf: Agrega una observación al caso.
* @param format string, args ...any
**/
func (r *report) logf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.details = append(r.details, fmt.Sprintf(format, args...))
}

/**
* check: Retorna un error con msg si ok es false.
* @param ok bool, format string, args ...any
* @return error
**/
func check(ok bool, format string, args ...any) error {
	if ok {
		return nil
	}
	return fmt.Errorf(format, args...)
}

/**
* waitFor: Espera hasta 3s a que cond sea verdadera.
* @param what string, cond func() bool
* @return error
**/
func waitFor(what string, cond func() bool) error {
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout esperando: %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil
}

/**
* setup: Levanta un servidor en un puerto libre y conecta un cliente. peer retorna la
* conexión del lado del servidor. close libera todo.
* @return *tcp.Server, *tcp.Client, func() *tcp.Client, func(), error
**/
func setup() (*tcp.Server, *tcp.Client, func() *tcp.Client, func(), error) {
	srv, err := tcp.NewServer("", "")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	var peer atomic.Pointer[tcp.Client]
	srv.OnConnect(func(conn *tcp.Client) { peer.Store(conn) })
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		return nil, nil, nil, nil, err
	}
	c, err := tcp.NewClient("", "")
	if err != nil {
		srv.Close()
		return nil, nil, nil, nil, err
	}
	if err := c.ConnectTo(srv.Addr()); err != nil {
		srv.Close()
		return nil, nil, nil, nil, err
	}
	getPeer := func() *tcp.Client {
		waitFor("peer", func() bool { return peer.Load() != nil })
		return peer.Load()
	}
	closeAll := func() {
		c.Close()
		srv.Close()
	}
	return srv, c, getPeer, closeAll, nil
}

/**
* readAll: Lee un stream completo.
* @param in *tcp.StreamReader
* @return []string, error
**/
func readAll(in *tcp.StreamReader) ([]string, error) {
	var out []string
	for {
		data, err := in.Read()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return out, err
		}
		out = append(out, string(data))
	}
}

/**
* portOf: El puerto de una dirección host:port.
* @param addr string
* @return int
**/
func portOf(addr string) int {
	_, p, _ := net.SplitHostPort(addr)
	n, _ := strconv.Atoi(p)
	return n
}

/**
* hasErr: Si err no es nil y su texto contiene part.
* @param err error, part string
* @return bool
**/
func hasErr(err error, part string) bool {
	return err != nil && strings.Contains(err.Error(), part)
}

/**
* hubPair: Hub a con un servidor en un puerto libre y hub b conectado a él.
* @return a, b *tcp.Hub, port int, addr string, err error
**/
func hubPair() (*tcp.Hub, *tcp.Hub, int, string, error) {
	a := tcp.NewHub("hub-a", "nodo-a")
	srv, err := a.LoadServer(0)
	if err != nil {
		return nil, nil, 0, "", err
	}
	port := portOf(srv.Addr())
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	b := tcp.NewHub("hub-b", "nodo-b")
	return a, b, port, addr, nil
}

// ---- Casos ----

var cases = []testCase{
	{
		Name:    "Servidor: ciclo de vida",
		Methods: []string{"NewServer", "Server.Listen", "Server.Addr", "Server.Count", "Server.Close"},
		run: func(r *report) error {
			srv, err := tcp.NewServer("", "")
			if err != nil {
				return err
			}
			if err := check(srv.Addr() == "", "Addr antes de Listen = %q", srv.Addr()); err != nil {
				return err
			}
			if err := srv.Listen("127.0.0.1:0"); err != nil {
				return err
			}
			r.logf("escuchando en %s", srv.Addr())
			err = srv.Listen("127.0.0.1:0")
			if err := check(err != nil, "un segundo Listen debió fallar"); err != nil {
				return err
			}
			r.logf("segundo Listen -> %v", err)

			clients := make([]*tcp.Client, 3)
			for i := range clients {
				clients[i], _ = tcp.NewClient("", "")
				if err := clients[i].ConnectTo(srv.Addr()); err != nil {
					return err
				}
			}
			if err := waitFor("Count=3", func() bool { return srv.Count() == 3 }); err != nil {
				return err
			}
			r.logf("Count con 3 clientes = %d", srv.Count())

			if err := srv.Close(); err != nil {
				return err
			}
			if err := check(srv.Count() == 0 && srv.Addr() == "", "después de Close: Count=%d Addr=%q", srv.Count(), srv.Addr()); err != nil {
				return err
			}
			r.logf("después de Close: Count=0, Addr vacío")
			if err := srv.Listen("127.0.0.1:0"); err != nil {
				return fmt.Errorf("volver a escuchar: %w", err)
			}
			r.logf("volvió a escuchar en %s", srv.Addr())
			return srv.Close()
		},
	},
	{
		Name:    "Identificación: saludo, ID, Name y ToJson",
		Methods: []string{"NewClient", "Client.ID", "Client.Name", "Client.To", "Client.ServerID", "Client.ServerName", "Client.ToJson", "NewServer", "Server.ID", "Server.Name", "Server.SetName", "Server.ToJson", "Server.OnConnect", "Server.OnSend"},
		run: func(r *report) error {
			srv, err := tcp.NewServer("nodo-1", "primario")
			if err != nil {
				return err
			}
			auto, _ := tcp.NewServer("", "")
			if err := check(srv.ID() == "nodo-1" && srv.Name() == "primario" && auto.ID() != "", "servidor: %q %q %q", srv.ID(), srv.Name(), auto.ID()); err != nil {
				return err
			}
			srv.SetName("nodo-a")
			peers := make(chan *tcp.Client, 1)
			from := make(chan string, 1)
			srv.OnConnect(func(conn *tcp.Client) { peers <- conn })
			srv.OnSend(func(conn *tcp.Client, m []byte) { from <- conn.ID() + "/" + conn.Name() })
			if err := srv.Listen("127.0.0.1:0"); err != nil {
				return err
			}

			c, _ := tcp.NewClient("cliente-1", "replica")
			generated, _ := tcp.NewClient("", "")
			if err := check(c.ID() == "cliente-1" && c.Name() == "replica" && generated.ID() != "", "cliente: %q %q %q", c.ID(), c.Name(), generated.ID()); err != nil {
				return err
			}
			r.logf("NewServer/NewClient conservan id y nombre; con id vacío generan uno (%s…)", generated.ID()[:8])
			if err := check(c.To() == "" && c.ServerID() == "", "antes de conectar: to=%q server=%q", c.To(), c.ServerID()); err != nil {
				return err
			}

			if err := c.ConnectTo(srv.Addr()); err != nil {
				return err
			}
			if err := check(c.To() == srv.Addr() && c.ServerID() == "nodo-1" && c.ServerName() == "nodo-a", "después de conectar: to=%q server=%q/%q", c.To(), c.ServerID(), c.ServerName()); err != nil {
				return err
			}
			r.logf("el cliente conoce al servidor por el saludo: ServerID=%s ServerName=%s To=%s", c.ServerID(), c.ServerName(), c.To())

			peer := <-peers
			if err := check(peer.ID() == "cliente-1" && peer.Name() == "replica" && peer.ServerID() == "nodo-1", "conexión en el servidor: %v", peer.ToJson()); err != nil {
				return err
			}
			r.logf("en OnConnect del servidor, la conexión trae la identidad del cliente: %s/%s", peer.ID(), peer.Name())
			c.Send([]byte("hola"))
			r.logf("OnSend del servidor sabe quién envió: %s", <-from)

			if err := waitFor("peer en ToJson", func() bool { return len(srv.ToJson().ArrayJson("peers")) == 1 }); err != nil {
				return err
			}
			sj := srv.ToJson()
			if err := check(sj.Str("id") == "nodo-1" && sj.Str("name") == "nodo-a" && sj.Str("addr") == srv.Addr() && sj.ArrayJson("peers")[0].Str("id") == "cliente-1", "Server.ToJson: %v", sj); err != nil {
				return err
			}
			r.logf("Server.ToJson -> %v", sj)
			cj := c.ToJson()
			if err := check(cj.Str("id") == "cliente-1" && cj.Json("server").Str("id") == "nodo-1", "Client.ToJson: %v", cj); err != nil {
				return err
			}
			r.logf("Client.ToJson -> %v", cj)

			c.Close()
			srv.Close()
			if err := srv.Listen("127.0.0.1:0"); err != nil {
				return err
			}
			defer srv.Close()
			return check(srv.ID() == "nodo-1", "el ID cambió al volver a escuchar: %q", srv.ID())
		},
	},
	{
		Name:    "Saludo fallido",
		Methods: []string{"Client.ConnectTo", "Client.SetTimeout", "Server.SetTimeout", "Server.Count"},
		run: func(r *report) error {
			// Un servidor que acepta pero nunca saluda: ConnectTo falla al vencer el timeout.
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				return err
			}
			defer ln.Close()
			go func() {
				for {
					conn, err := ln.Accept()
					if err != nil {
						return
					}
					defer conn.Close()
				}
			}()
			c, _ := tcp.NewClient("", "")
			c.SetTimeout(150 * time.Millisecond)
			err = c.ConnectTo(ln.Addr().String())
			if err := check(err != nil && strings.Contains(err.Error(), "tcp:"), "ConnectTo sin saludo del servidor debió fallar: %v", err); err != nil {
				return err
			}
			r.logf("servidor que no saluda -> ConnectTo: %v", err)

			// Un cliente que conecta pero nunca saluda: el servidor lo cierra y no lo cuenta.
			srv, _ := tcp.NewServer("", "")
			srv.SetTimeout(150 * time.Millisecond)
			var connects atomic.Int32
			srv.OnConnect(func(conn *tcp.Client) { connects.Add(1) })
			if err := srv.Listen("127.0.0.1:0"); err != nil {
				return err
			}
			defer srv.Close()
			raw, err := net.Dial("tcp", srv.Addr())
			if err != nil {
				return err
			}
			defer raw.Close()
			raw.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, readErr := raw.Read(make([]byte, 1))
			if err := check(readErr != nil && srv.Count() == 0 && connects.Load() == 0, "el servidor debió cerrar la conexión sin saludo: %v count=%d", readErr, srv.Count()); err != nil {
				return err
			}
			r.logf("cliente que no saluda -> el servidor lo cierra (%v); Count=0, sin OnConnect", readErr)
			return nil
		},
	},
	{
		Name:    "Cliente: conexión",
		Methods: []string{"NewClient", "Client.ConnectTo", "Client.Close", "Client.Send", "Client.Request"},
		run: func(r *report) error {
			srv, _, _, closeAll, err := setup()
			if err != nil {
				return err
			}
			defer closeAll()

			c, _ := tcp.NewClient("", "")
			err = c.Send([]byte("x"))
			if err := check(err != nil, "Send sin conexión debió fallar"); err != nil {
				return err
			}
			r.logf("Send sin conexión -> %v", err)
			_, err = c.Request([]byte("x"))
			r.logf("Request sin conexión -> %v", err)

			if err := c.ConnectTo(srv.Addr()); err != nil {
				return err
			}
			err = c.ConnectTo(srv.Addr())
			if err := check(err != nil, "un segundo ConnectTo debió fallar"); err != nil {
				return err
			}
			r.logf("segundo ConnectTo -> %v", err)
			if err := c.Close(); err != nil {
				return err
			}
			err = c.Send([]byte("x"))
			if err := check(err != nil, "Send después de Close debió fallar"); err != nil {
				return err
			}
			if err := c.ConnectTo(srv.Addr()); err != nil {
				return fmt.Errorf("reconectar: %w", err)
			}
			r.logf("Close y reconexión correctos")
			return c.Close()
		},
	},
	{
		Name:    "Timeout",
		Methods: []string{"Client.SetTimeout", "Server.SetTimeout"},
		run: func(r *report) error {
			srv, c, _, closeAll, err := setup()
			if err != nil {
				return err
			}
			defer closeAll()
			srv.OnRequest(func(conn *tcp.Client, m []byte) ([]byte, error) {
				time.Sleep(200 * time.Millisecond)
				return []byte("tarde"), nil
			})

			err = c.SetTimeout(0)
			if err := check(err != nil, "SetTimeout(0) debió fallar"); err != nil {
				return err
			}
			r.logf("Client.SetTimeout(0) -> %v", err)
			err = srv.SetTimeout(-time.Second)
			if err := check(err != nil, "Server.SetTimeout(-1s) debió fallar"); err != nil {
				return err
			}
			r.logf("Server.SetTimeout(-1s) -> %v", err)

			c.SetTimeout(50 * time.Millisecond)
			start := time.Now()
			_, err = c.Request([]byte("x"))
			if err := check(err != nil, "Request con timeout 50ms debió fallar"); err != nil {
				return err
			}
			r.logf("timeout 50ms (conexión abierta) -> %v en %v", err, time.Since(start).Round(time.Millisecond))

			c.SetTimeout(time.Second)
			resp, err := c.Request([]byte("x"))
			if err := check(err == nil && string(resp) == "tarde", "con 1s: %q %v", resp, err); err != nil {
				return err
			}
			r.logf("timeout 1s -> respuesta %q", resp)
			return srv.SetTimeout(5 * time.Second)
		},
	},
	{
		Name:    "Send y OnSend en los dos sentidos",
		Methods: []string{"Client.Send", "Server.OnSend", "Client.OnSend", "Server.OnConnect"},
		run: func(r *report) error {
			srv, c, peer, closeAll, err := setup()
			if err != nil {
				return err
			}
			defer closeAll()

			var toServer, toClient, calls atomic.Int32
			var sameConn atomic.Bool
			sameConn.Store(true)
			p := peer()
			srv.OnSend(func(conn *tcp.Client, m []byte) {
				if conn != p {
					sameConn.Store(false)
				}
				toServer.Add(1)
			})
			srv.OnSend(func(conn *tcp.Client, m []byte) { calls.Add(1) }) // todas las funciones reciben cada mensaje
			c.OnSend(func(conn *tcp.Client, m []byte) {
				if conn != c {
					sameConn.Store(false)
				}
				toClient.Add(1)
			})

			for i := 0; i < 100; i++ {
				if err := c.Send([]byte(strconv.Itoa(i))); err != nil {
					return err
				}
				if err := p.Send([]byte(strconv.Itoa(i))); err != nil {
					return err
				}
			}
			if err := waitFor("100 y 100", func() bool {
				return toServer.Load() == 100 && toClient.Load() == 100 && calls.Load() == 100
			}); err != nil {
				return err
			}
			r.logf("cliente→servidor: %d, servidor→cliente: %d, segunda función del servidor: %d", toServer.Load(), toClient.Load(), calls.Load())
			return check(sameConn.Load(), "conn no corresponde a la conexión")
		},
	},
	{
		Name:    "Request y OnRequest: orden, error, sin handler y panic",
		Methods: []string{"Client.Request", "Server.OnRequest", "Client.OnRequest"},
		run: func(r *report) error {
			srv, c, peer, closeAll, err := setup()
			if err != nil {
				return err
			}
			defer closeAll()

			srv.OnRequest(func(conn *tcp.Client, m []byte) ([]byte, error) {
				switch string(m) {
				case "siguiente":
					return nil, nil
				case "error":
					return nil, errors.New("falló a propósito")
				case "panic":
					panic("boom")
				case "ninguna":
					return nil, nil
				}
				return append([]byte("eco:"), m...), nil
			})
			srv.OnRequest(func(conn *tcp.Client, m []byte) ([]byte, error) {
				if string(m) == "siguiente" {
					return []byte("segunda"), nil
				}
				return nil, nil
			})

			resp, err := c.Request([]byte("hola"))
			if err := check(err == nil && string(resp) == "eco:hola", "hola: %q %v", resp, err); err != nil {
				return err
			}
			resp, err = c.Request([]byte("siguiente"))
			if err := check(err == nil && string(resp) == "segunda", "(nil,nil) no cedió: %q %v", resp, err); err != nil {
				return err
			}
			r.logf("(nil, nil) cede a la siguiente -> %q", resp)
			_, err = c.Request([]byte("error"))
			if err := check(err != nil && err.Error() == "falló a propósito", "error: %v", err); err != nil {
				return err
			}
			r.logf("error del handler -> %v", err)
			_, err = c.Request([]byte("ninguna"))
			if err := check(err != nil, "sin respuesta debió fallar"); err != nil {
				return err
			}
			r.logf("ninguna responde -> %v", err)
			_, err = c.Request([]byte("panic"))
			if err := check(err != nil, "panic debió llegar como error"); err != nil {
				return err
			}
			r.logf("panic -> %v", err)
			resp, err = c.Request([]byte("sigue"))
			if err := check(err == nil && string(resp) == "eco:sigue", "después del panic: %v", err); err != nil {
				return err
			}

			c.OnRequest(func(conn *tcp.Client, m []byte) ([]byte, error) { return []byte("cliente:" + string(m)), nil })
			resp, err = peer().Request([]byte("x"))
			r.logf("servidor→cliente Request -> %q", resp)
			return check(err == nil && string(resp) == "cliente:x", "servidor→cliente: %q %v", resp, err)
		},
	},
	{
		Name:    "Concurrencia: 5 clientes, en los dos sentidos",
		Methods: []string{"Client.Send", "Client.Request", "Server.OnRequest", "Server.OnSend", "Client.OnRequest", "Client.OnSend", "Server.OnConnect"},
		run: func(r *report) error {
			const nc, n = 5, 200
			srv, err := tcp.NewServer("", "")
			if err != nil {
				return err
			}
			var failures, srvSends, cliSends atomic.Int32
			var srvWg sync.WaitGroup
			srv.OnRequest(func(conn *tcp.Client, m []byte) ([]byte, error) { return append([]byte("srv:"), m...), nil })
			srv.OnSend(func(conn *tcp.Client, m []byte) { srvSends.Add(1) })
			srv.OnConnect(func(conn *tcp.Client) {
				for i := 0; i < n; i++ {
					srvWg.Add(1)
					go func(m string) {
						defer srvWg.Done()
						if resp, err := conn.Request([]byte(m)); err != nil || string(resp) != "cli:"+m {
							failures.Add(1)
						}
						conn.Send([]byte(m))
					}(strconv.Itoa(i))
				}
			})
			if err := srv.Listen("127.0.0.1:0"); err != nil {
				return err
			}
			defer srv.Close()

			var wg sync.WaitGroup
			for k := 0; k < nc; k++ {
				c, _ := tcp.NewClient("", "")
				c.OnRequest(func(conn *tcp.Client, m []byte) ([]byte, error) { return append([]byte("cli:"), m...), nil })
				c.OnSend(func(conn *tcp.Client, m []byte) { cliSends.Add(1) })
				if err := c.ConnectTo(srv.Addr()); err != nil {
					return err
				}
				defer c.Close()
				for i := 0; i < n; i++ {
					wg.Add(1)
					go func(m string) {
						defer wg.Done()
						if resp, err := c.Request([]byte(m)); err != nil || string(resp) != "srv:"+m {
							failures.Add(1)
						}
						c.Send([]byte(m))
					}(fmt.Sprint(k, "-", i))
				}
			}
			start := time.Now()
			wg.Wait()
			time.Sleep(20 * time.Millisecond)
			srvWg.Wait()
			if err := waitFor("sends", func() bool { return srvSends.Load() == nc*n && cliSends.Load() == nc*n }); err != nil {
				return err
			}
			r.logf("%d Request y %d Send en cada sentido en %v", nc*n, nc*n, time.Since(start).Round(time.Millisecond))
			return check(failures.Load() == 0, "%d respuestas incorrectas", failures.Load())
		},
	},
	{
		Name:    "OnConnect y OnDisconnect en los dos lados",
		Methods: []string{"Client.OnConnect", "Client.OnDisconnect", "Server.OnConnect", "Server.OnDisconnect", "Client.Close", "Server.Close"},
		run: func(r *report) error {
			srv, err := tcp.NewServer("", "")
			if err != nil {
				return err
			}
			var srvConn, srvDisc atomic.Int32
			srv.OnConnect(func(conn *tcp.Client) { panic("panic en OnConnect") }) // no debe cortar la siguiente
			srv.OnConnect(func(conn *tcp.Client) { srvConn.Add(1) })
			srv.OnDisconnect(func(conn *tcp.Client, err error) { srvDisc.Add(1) })
			srv.OnRequest(func(conn *tcp.Client, m []byte) ([]byte, error) { return m, nil })
			if err := srv.Listen("127.0.0.1:0"); err != nil {
				return err
			}

			var cliConn, cliDisc atomic.Int32
			var lastErr atomic.Value
			newClient := func() (*tcp.Client, error) {
				c, _ := tcp.NewClient("", "")
				c.OnConnect(func(conn *tcp.Client) {
					if resp, err := conn.Request([]byte("hola")); err == nil && string(resp) == "hola" {
						cliConn.Add(1) // ya puede usar Request dentro de OnConnect
					}
				})
				c.OnDisconnect(func(conn *tcp.Client, err error) { lastErr.Store(err.Error()); cliDisc.Add(1) })
				return c, c.ConnectTo(srv.Addr())
			}
			a, err := newClient()
			if err != nil {
				return err
			}
			b, err := newClient()
			if err != nil {
				return err
			}
			if err := check(cliConn.Load() == 2, "OnConnect del cliente debe correr antes de que ConnectTo retorne"); err != nil {
				return err
			}
			if err := waitFor("OnConnect servidor", func() bool { return srvConn.Load() == 2 }); err != nil {
				return err
			}
			r.logf("OnConnect: cliente %d, servidor %d (con un panic en otra función)", cliConn.Load(), srvConn.Load())

			a.Close()
			if err := waitFor("cierra el cliente", func() bool { return srvDisc.Load() == 1 && cliDisc.Load() == 1 }); err != nil {
				return err
			}
			r.logf("cierra el cliente -> OnDisconnect en ambos lados")

			srv.Close()
			if err := check(srvDisc.Load() == 2, "Server.Close debe esperar los OnDisconnect: %d", srvDisc.Load()); err != nil {
				return err
			}
			if err := waitFor("cierra el servidor", func() bool { return cliDisc.Load() == 2 }); err != nil {
				return err
			}
			r.logf("cierra el servidor -> OnDisconnect en ambos lados; motivo en el cliente: %v", lastErr.Load())
			return b.Close()
		},
	},
	{
		Name:    "Reconexión desde OnDisconnect",
		Methods: []string{"Client.OnDisconnect", "Client.ConnectTo", "Server.OnConnect"},
		run: func(r *report) error {
			srv, err := tcp.NewServer("", "")
			if err != nil {
				return err
			}
			var peer atomic.Pointer[tcp.Client]
			srv.OnConnect(func(conn *tcp.Client) { peer.Store(conn) })
			srv.OnRequest(func(conn *tcp.Client, m []byte) ([]byte, error) { return m, nil })
			if err := srv.Listen("127.0.0.1:0"); err != nil {
				return err
			}
			defer srv.Close()
			addr := srv.Addr()

			c, _ := tcp.NewClient("", "")
			var connects atomic.Int32
			c.OnConnect(func(conn *tcp.Client) { connects.Add(1) })
			c.OnDisconnect(func(conn *tcp.Client, err error) {
				if connects.Load() < 2 {
					conn.ConnectTo(addr)
				}
			})
			if err := c.ConnectTo(addr); err != nil {
				return err
			}
			defer c.Close()
			if err := waitFor("peer", func() bool { return peer.Load() != nil }); err != nil {
				return err
			}
			first := peer.Load()
			first.Close() // el servidor corta la conexión
			if err := waitFor("reconexión", func() bool { return connects.Load() == 2 && peer.Load() != first }); err != nil {
				return err
			}
			resp, err := c.Request([]byte("de nuevo"))
			r.logf("el servidor cortó, el cliente se reconectó solo y respondió %q", resp)
			return check(err == nil && string(resp) == "de nuevo", "después de reconectar: %v", err)
		},
	},
	{
		Name:    "SendStream: orden y control de flujo",
		Methods: []string{"Client.SendStream", "Server.OnSendStream", "StreamWriter.Write", "StreamWriter.Close", "StreamReader.Read"},
		run: func(r *report) error {
			srv, c, _, closeAll, err := setup()
			if err != nil {
				return err
			}
			defer closeAll()

			const n = 2000
			got := make(chan []string, 1)
			srv.OnSendStream(func(conn *tcp.Client, in *tcp.StreamReader) {
				var out []string
				for {
					data, err := in.Read()
					if err != nil {
						break
					}
					out = append(out, string(data))
					if len(out) < 100 {
						time.Sleep(2 * time.Millisecond) // lector lento al principio
					}
				}
				got <- out
			})
			srv.OnRequest(func(conn *tcp.Client, m []byte) ([]byte, error) { return m, nil })

			var requests atomic.Int32
			stop := make(chan struct{})
			go func() {
				for {
					select {
					case <-stop:
						return
					default:
					}
					if resp, err := c.Request([]byte("p")); err == nil && string(resp) == "p" {
						requests.Add(1)
					}
				}
			}()

			w, err := c.SendStream()
			if err != nil {
				return err
			}
			start := time.Now()
			for i := 0; i < n; i++ {
				if err := w.Write([]byte(strconv.Itoa(i))); err != nil {
					return err
				}
			}
			if err := w.Close(); err != nil {
				return err
			}
			out := <-got
			close(stop)
			elapsed := time.Since(start)
			if err := check(len(out) == n, "llegaron %d de %d", len(out), n); err != nil {
				return err
			}
			for i, v := range out {
				if v != strconv.Itoa(i) {
					return fmt.Errorf("fuera de orden en %d: %s", i, v)
				}
			}
			r.logf("%d trozos en orden", n)
			r.logf("escritor frenado por el lector lento: %v", elapsed.Round(time.Millisecond))
			r.logf("Request atendidos mientras tanto por la misma conexión: %d", requests.Load())
			return check(elapsed > 100*time.Millisecond && requests.Load() > 0, "sin control de flujo o conexión bloqueada")
		},
	},
	{
		Name:    "RequestStream: 20 a la vez",
		Methods: []string{"Client.RequestStream", "Server.OnRequestStream", "StreamWriter.Write", "StreamWriter.CloseAndReceive", "StreamReader.Read"},
		run: func(r *report) error {
			srv, c, _, closeAll, err := setup()
			if err != nil {
				return err
			}
			defer closeAll()
			srv.OnRequestStream(func(conn *tcp.Client, in *tcp.StreamReader) ([]byte, error) {
				sum := 0
				for {
					data, err := in.Read()
					if err == io.EOF {
						break
					}
					if err != nil {
						return nil, err
					}
					v, _ := strconv.Atoi(string(data))
					sum += v
				}
				return []byte(strconv.Itoa(sum)), nil
			})

			var failures atomic.Int32
			var wg sync.WaitGroup
			for k := 0; k < 20; k++ {
				wg.Add(1)
				go func(k int) {
					defer wg.Done()
					w, err := c.RequestStream()
					if err != nil {
						failures.Add(1)
						return
					}
					expected := 0
					for i := 0; i < 300; i++ {
						w.Write([]byte(strconv.Itoa(i + k)))
						expected += i + k
					}
					resp, err := w.CloseAndReceive()
					if err != nil || string(resp) != strconv.Itoa(expected) {
						failures.Add(1)
					}
				}(k)
			}
			wg.Wait()
			r.logf("20 RequestStream de 300 trozos, cada uno con su suma correcta")

			sw, _ := c.SendStream()
			_, err = sw.CloseAndReceive()
			if err := check(err != nil, "CloseAndReceive en un SendStream debió fallar"); err != nil {
				return err
			}
			r.logf("CloseAndReceive en un SendStream -> %v", err)
			return check(failures.Load() == 0, "%d respuestas incorrectas", failures.Load())
		},
	},
	{
		Name:    "ReceiveStream en los dos sentidos",
		Methods: []string{"Client.ReceiveStream", "Server.OnReceiveStream", "Client.OnReceiveStream", "StreamWriter.Write", "StreamReader.Read"},
		run: func(r *report) error {
			srv, c, peer, closeAll, err := setup()
			if err != nil {
				return err
			}
			defer closeAll()
			generate := func(conn *tcp.Client, m []byte, out *tcp.StreamWriter) error {
				n, _ := strconv.Atoi(string(m))
				for i := 0; i < n; i++ {
					if err := out.Write([]byte(strconv.Itoa(i))); err != nil {
						return err
					}
				}
				return nil
			}
			srv.OnReceiveStream(generate)
			c.OnReceiveStream(generate)
			p := peer()

			var failures atomic.Int32
			var wg sync.WaitGroup
			for k := 0; k < 10; k++ {
				for _, from := range []*tcp.Client{c, p} { // los dos lados abren streams con los mismos IDs
					wg.Add(1)
					go func(from *tcp.Client) {
						defer wg.Done()
						in, err := from.ReceiveStream([]byte("500"))
						if err != nil {
							failures.Add(1)
							return
						}
						got, err := readAll(in)
						if err != nil || len(got) != 500 || got[499] != "499" {
							failures.Add(1)
						}
					}(from)
				}
			}
			wg.Wait()
			r.logf("20 ReceiveStream de 500 trozos a la vez: 10 del cliente al servidor y 10 del servidor al cliente")
			return check(failures.Load() == 0, "%d streams incorrectos", failures.Load())
		},
	},
	{
		Name:    "Streams del servidor al cliente",
		Methods: []string{"Client.OnSendStream", "Client.OnRequestStream", "Client.SendStream", "Client.RequestStream", "StreamWriter.Write", "StreamWriter.Close", "StreamWriter.CloseAndReceive", "StreamReader.Read"},
		run: func(r *report) error {
			_, c, peer, closeAll, err := setup()
			if err != nil {
				return err
			}
			defer closeAll()

			received := make(chan []string, 1)
			c.OnSendStream(func(conn *tcp.Client, in *tcp.StreamReader) {
				got, _ := readAll(in)
				received <- got
			})
			c.OnRequestStream(func(conn *tcp.Client, in *tcp.StreamReader) ([]byte, error) {
				got, err := readAll(in)
				if err != nil {
					return nil, err
				}
				return []byte(strings.Join(got, "")), nil
			})
			p := peer() // la conexión del lado del servidor

			w, err := p.SendStream()
			if err != nil {
				return err
			}
			for _, part := range []string{"a", "b", "c"} {
				w.Write([]byte(part))
			}
			w.Close()
			got := <-received
			if err := check(strings.Join(got, "") == "abc", "SendStream servidor→cliente: %v", got); err != nil {
				return err
			}
			r.logf("SendStream servidor→cliente -> el cliente recibió %v", got)

			rs, err := p.RequestStream()
			if err != nil {
				return err
			}
			for _, part := range []string{"x", "y", "z"} {
				rs.Write([]byte(part))
			}
			resp, err := rs.CloseAndReceive()
			r.logf("RequestStream servidor→cliente -> el cliente respondió %q", resp)
			return check(err == nil && string(resp) == "xyz", "RequestStream servidor→cliente: %q %v", resp, err)
		},
	},
	{
		Name:    "Streams: errores y cancelación",
		Methods: []string{"StreamWriter.CloseWithError", "StreamReader.Close", "Server.OnSendStream", "Server.OnRequestStream", "Server.OnReceiveStream", "Client.SendStream", "Client.RequestStream", "Client.ReceiveStream"},
		run: func(r *report) error {
			srv, c, _, closeAll, err := setup()
			if err != nil {
				return err
			}
			defer closeAll()

			// Sin handlers.
			rs, _ := c.RequestStream()
			_, err = rs.CloseAndReceive()
			if err := check(err != nil, "RequestStream sin handler debió fallar"); err != nil {
				return err
			}
			r.logf("RequestStream sin handler -> %v", err)
			in, _ := c.ReceiveStream(nil)
			_, err = in.Read()
			if err := check(err != nil, "ReceiveStream sin handler debió fallar"); err != nil {
				return err
			}
			r.logf("ReceiveStream sin handler -> %v", err)

			// CloseWithError llega al lector.
			received := make(chan error, 1)
			srv.OnSendStream(func(conn *tcp.Client, in *tcp.StreamReader) {
				_, err := readAll(in)
				received <- err
			})
			w, _ := c.SendStream()
			w.Write([]byte("1"))
			w.CloseWithError(errors.New("el productor falló"))
			err = <-received
			if err := check(err != nil && err.Error() == "el productor falló", "CloseWithError: %v", err); err != nil {
				return err
			}
			r.logf("CloseWithError -> el lector recibe %q", err)

			// Error y panic del handler de ReceiveStream.
			srv.OnReceiveStream(func(conn *tcp.Client, m []byte, out *tcp.StreamWriter) error {
				out.Write([]byte("a"))
				if string(m) == "panic" {
					panic("boom")
				}
				return errors.New("el generador falló")
			})
			for _, m := range []string{"error", "panic"} {
				in, _ := c.ReceiveStream([]byte(m))
				got, err := readAll(in)
				if err := check(len(got) == 1 && err != nil, "%s: %v %v", m, got, err); err != nil {
					return err
				}
				r.logf("ReceiveStream con %s en el handler -> leyó %v y luego %v", m, got, err)
			}

			// Panic en OnRequestStream.
			srv.OnRequestStream(func(conn *tcp.Client, in *tcp.StreamReader) ([]byte, error) { panic("boom") })
			rs, _ = c.RequestStream()
			_, err = rs.CloseAndReceive()
			if err := check(err != nil, "panic en OnRequestStream debió llegar como error"); err != nil {
				return err
			}
			r.logf("panic en OnRequestStream -> %v", err)

			// El lector deja de leer: el escritor se entera.
			writerErr := make(chan error, 1)
			srv.OnReceiveStream(func(conn *tcp.Client, m []byte, out *tcp.StreamWriter) error {
				for {
					if err := out.Write([]byte("z")); err != nil {
						writerErr <- err
						return err
					}
				}
			})
			in, _ = c.ReceiveStream(nil)
			in.Read()
			in.Read()
			in.Close()
			select {
			case err = <-writerErr:
				r.logf("StreamReader.Close -> el escritor recibe %v", err)
			case <-time.After(3 * time.Second):
				return errors.New("el escritor no se enteró de la cancelación")
			}

			// La conexión se cae a mitad del stream.
			srv.OnReceiveStream(func(conn *tcp.Client, m []byte, out *tcp.StreamWriter) error {
				out.Write([]byte("1"))
				time.Sleep(50 * time.Millisecond)
				conn.Close()
				return nil
			})
			in, _ = c.ReceiveStream(nil)
			got, err := readAll(in)
			r.logf("conexión cerrada a mitad del stream -> leyó %v y luego %v", got, err)
			return check(err != nil && err != io.EOF, "debió fallar con error de conexión")
		},
	},
	{
		Name:    "Hub: creación, servidores y baja (HUB-01, 02, 03)",
		Methods: []string{"NewHub", "Hub.ID", "Hub.Name", "Hub.LoadServer", "Hub.DownServer", "Hub.Servers", "Hub.Metrics"},
		run: func(r *report) error {
			before := runtime.NumGoroutine()
			h := tcp.NewHub("", "nodo")
			time.Sleep(20 * time.Millisecond)
			if err := check(runtime.NumGoroutine() <= before, "NewHub arrancó goroutines (%d -> %d)", before, runtime.NumGoroutine()); err != nil {
				return err
			}
			if err := check(len(h.ID()) == 36 && h.Name() == "nodo", "identidad: %q %q", h.ID(), h.Name()); err != nil {
				return err
			}
			if err := check(tcp.NewHub("fijo", "").ID() == "fijo", "id dado no se respetó"); err != nil {
				return err
			}
			r.logf("NewHub sin id -> %s; no arranca goroutines ni escucha", h.ID())

			srv, err := h.LoadServer(0)
			if err != nil {
				return err
			}
			port := portOf(srv.Addr())
			if err := check(srv.ID() == h.ID() && srv.Name() == "nodo", "el servidor no tiene la identidad del Hub"); err != nil {
				return err
			}
			_, err = h.LoadServer(port)
			if err := check(hasErr(err, "already loaded"), "LoadServer repetido -> %v", err); err != nil {
				return err
			}
			srv2, err := h.LoadServer(0)
			if err != nil {
				return err
			}
			if err := check(len(h.Servers()) == 2, "Servers = %d", len(h.Servers())); err != nil {
				return err
			}
			r.logf("LoadServer(0) en :%d y :%s; repetido -> %v", port, srv2.Addr(), "MSG_HUB_SERVER_EXISTS")

			c, _ := tcp.NewClient("", "")
			if err := c.ConnectTo(fmt.Sprintf("127.0.0.1:%d", port)); err != nil {
				return err
			}
			if err := h.DownServer(port); err != nil {
				return err
			}
			if err := waitFor("cliente cerrado", func() bool { return c.ServerID() == "" || c.Send([]byte("x")) != nil }); err != nil {
				return err
			}
			err = h.DownServer(port)
			_, merr := h.Metrics(port)
			if err := check(hasErr(err, "no server loaded") && hasErr(merr, "no server loaded"), "tras DownServer: %v / %v", err, merr); err != nil {
				return err
			}
			if err := check(len(h.Servers()) == 1, "Servers tras la baja = %d", len(h.Servers())); err != nil {
				return err
			}
			r.logf("DownServer cierra las conexiones; repetido y Metrics -> MSG_HUB_SERVER_NOT_FOUND")
			return h.Close()
		},
	},
	{
		Name:    "Hub: mensajes tipados, orden y eventos (HUB-04, 05, 06, 11)",
		Methods: []string{"Hub.ConnectTo", "Hub.Send", "Hub.SendTo", "Hub.OnMessage", "Hub.OnConnect"},
		run: func(r *report) error {
			a, b, port, addr, err := hubPair()
			if err != nil {
				return err
			}
			defer a.Close()
			defer b.Close()

			var mu sync.Mutex
			var got []*tcp.Message
			var connSide, connName string
			a.OnConnect(func(srv *tcp.Server, conn *tcp.Client) {
				mu.Lock()
				defer mu.Unlock()
				if srv != nil {
					connSide, connName = "server", conn.Name()
				}
			})
			var bSawServer string
			b.OnConnect(func(srv *tcp.Server, conn *tcp.Client) {
				if srv == nil {
					bSawServer = conn.ServerName()
				}
			})
			a.OnMessage(func(srv *tcp.Server, conn *tcp.Client, m *tcp.Message) {
				mu.Lock()
				got = append(got, m)
				mu.Unlock()
			})
			var back atomic.Pointer[tcp.Message]
			b.OnMessage(func(srv *tcp.Server, conn *tcp.Client, m *tcp.Message) {
				if srv == nil {
					back.Store(m)
				}
			})

			if _, err := b.ConnectTo(addr); err != nil {
				return err
			}
			if err := check(bSawServer == "nodo-a", "OnConnect del cliente: servidor %q", bSawServer); err != nil {
				return err
			}

			const n = 500
			for i := 1; i <= n; i++ {
				if err := b.Send(addr, &tcp.Message{Type: tcp.TypeInt, Message: i, Header: map[string]any{"i": i}, From: map[string]any{"id": "falso"}}); err != nil {
					return err
				}
			}
			if err := waitFor("500 mensajes", func() bool { mu.Lock(); defer mu.Unlock(); return len(got) == n }); err != nil {
				return err
			}
			mu.Lock()
			for i, m := range got {
				v, ok := m.Message.(int64)
				if !ok || v != int64(i+1) {
					mu.Unlock()
					return fmt.Errorf("mensaje %d fuera de orden o mal tipado: %#v", i+1, m.Message)
				}
				if m.From.Str("id") != "hub-b" || m.From.Str("name") != "nodo-b" {
					mu.Unlock()
					return fmt.Errorf("From no lo puso el Hub: %v", m.From)
				}
			}
			side, name := connSide, connName
			got = got[:0]
			mu.Unlock()
			r.logf("%d TypeInt en orden, como int64, con From del Hub (el From del llamador se ignora)", n)
			if err := check(side == "server" && name == "nodo-b", "OnConnect del servidor: %s %q", side, name); err != nil {
				return err
			}
			r.logf("OnConnect: servidor ve al cliente %q; cliente ve al servidor %q", name, "nodo-a")

			for _, m := range []*tcp.Message{
				{Type: tcp.TypeFloat, Message: float32(1.5)},
				{Type: tcp.TypeText, Message: "hola"},
				{Type: tcp.TypeJson, Message: map[string]any{"k": 1}},
			} {
				if err := b.Send(addr, m); err != nil {
					return err
				}
			}
			if err := waitFor("3 tipos", func() bool { mu.Lock(); defer mu.Unlock(); return len(got) == 3 }); err != nil {
				return err
			}
			mu.Lock()
			f, fok := got[0].Message.(float64)
			t, tok := got[1].Message.(string)
			j, jok := got[2].Message.(map[string]any)
			_ = j
			mu.Unlock()
			if err := check(fok && f == 1.5 && tok && t == "hola" && (jok || got[2].Type == tcp.TypeJson), "conversiones: %v %v %#v", f, t, got[2].Message); err != nil {
				return err
			}
			r.logf("TypeFloat -> float64, TypeText -> string, TypeJson -> et.Json")

			for _, bad := range []*tcp.Message{
				{Type: 9, Message: 1},
				{Type: tcp.TypeBeat},
				{Type: tcp.TypeInt, Message: "x"},
				{Type: tcp.TypeText, Message: 3},
				nil,
			} {
				if err := b.Send(addr, bad); !hasErr(err, "invalid message") {
					return fmt.Errorf("mensaje inválido %+v aceptado: %v", bad, err)
				}
			}
			r.logf("tipo fuera de rango, TypeBeat, valor que no corresponde y nil -> MSG_HUB_INVALID_MESSAGE")

			if err := a.SendTo(port, "hub-b", &tcp.Message{Type: tcp.TypeText, Message: "de vuelta"}); err != nil {
				return err
			}
			if err := waitFor("SendTo", func() bool { return back.Load() != nil }); err != nil {
				return err
			}
			if err := check(back.Load().Message == "de vuelta" && back.Load().From.Str("name") == "nodo-a", "SendTo: %+v", back.Load()); err != nil {
				return err
			}
			err1 := a.SendTo(port, "otro", &tcp.Message{Type: tcp.TypeText, Message: "x"})
			err2 := b.Send("127.0.0.1:1", &tcp.Message{Type: tcp.TypeText, Message: "x"})
			err3 := a.SendTo(1, "hub-b", &tcp.Message{Type: tcp.TypeText, Message: "x"})
			r.logf("SendTo servidor -> cliente llega con srv nil; id desconocido -> %v; addr desconocido -> %v; puerto -> %v", err1, err2, err3)
			return check(hasErr(err1, "client not found") && hasErr(err2, "client not found") && hasErr(err3, "no server loaded"), "errores de destino")
		},
	},
	{
		Name:    "Hub: métricas y ToJson (HUB-08)",
		Methods: []string{"Hub.Metrics", "Hub.ToJson"},
		run: func(r *report) error {
			a, b, port, addr, err := hubPair()
			if err != nil {
				return err
			}
			defer a.Close()
			defer b.Close()
			if _, err := b.ConnectTo(addr); err != nil {
				return err
			}
			if err := waitFor("conexión en métricas", func() bool { m, _ := a.Metrics(port); return m.Int("count") == 1 }); err != nil {
				return err
			}
			for i := 0; i < 10; i++ {
				b.Send(addr, &tcp.Message{Type: tcp.TypeText, Message: "hola"})
			}
			a.SendTo(port, "hub-b", &tcp.Message{Type: tcp.TypeInt, Message: 7})
			if err := waitFor("10 entrantes", func() bool {
				m, _ := a.Metrics(port)
				cs := m.ArrayJson("clients")
				return len(cs) == 1 && cs[0].Int("messages_in") == 10
			}); err != nil {
				return err
			}
			m, _ := a.Metrics(port)
			c := m.ArrayJson("clients")[0]
			r.logf("Metrics: %s", c.ToString())
			if err := check(c.Str("name") == "nodo-b" && c.Int("messages_out") == 1 && c.Int("bytes_in") > 0 && c.Int("bytes_out") > 0 && c["connected_at"] != nil && c["last_message"] != nil, "métricas: %v", c); err != nil {
				return err
			}

			js := a.ToJson()
			if err := check(js.Str("id") == "hub-a" && len(js.ArrayJson("servers")) == 1, "ToJson: %v", js); err != nil {
				return err
			}
			if err := check(len(b.ToJson().ArrayJson("clients")) == 1, "ToJson de b sin clientes"); err != nil {
				return err
			}

			b.Disconnect(addr)
			if err := waitFor("sale de las métricas", func() bool { m, _ := a.Metrics(port); return m.Int("count") == 0 }); err != nil {
				return err
			}
			r.logf("al cerrarse, la conexión sale de las métricas")
			return nil
		},
	},
	{
		Name:    "Hub: clientes, caída, reconexión y Disconnect (HUB-07, 09)",
		Methods: []string{"Hub.ConnectTo", "Hub.Disconnect", "Hub.Clients", "Hub.OnDisconnect", "Hub.Status"},
		run: func(r *report) error {
			a, b, port, addr, err := hubPair()
			if err != nil {
				return err
			}
			defer a.Close()
			defer b.Close()

			var mu sync.Mutex
			var events []string
			a.OnDisconnect(func(srv *tcp.Server, conn *tcp.Client, err error) {
				mu.Lock()
				events = append(events, fmt.Sprintf("a srv=%v %s", srv != nil, conn.Name()))
				mu.Unlock()
			})
			b.OnDisconnect(func(srv *tcp.Server, conn *tcp.Client, err error) {
				mu.Lock()
				events = append(events, fmt.Sprintf("b srv=%v", srv != nil))
				mu.Unlock()
			})

			c1, err := b.ConnectTo(addr)
			if err != nil {
				return err
			}
			_, err = b.ConnectTo(addr)
			if err := check(hasErr(err, "already connected") && len(b.Clients()) == 1, "ConnectTo repetido -> %v", err); err != nil {
				return err
			}
			_, err = b.ConnectTo("127.0.0.1:1")
			if err := check(err != nil && len(b.Clients()) == 1, "una dirección que no conecta no debe quedar registrada"); err != nil {
				return err
			}

			if err := a.DownServer(port); err != nil {
				return err
			}
			if err := waitFor("down", func() bool { st := b.Status(); return len(st) == 1 && st[0].Str("state") == "down" }); err != nil {
				return err
			}
			err = b.Send(addr, &tcp.Message{Type: tcp.TypeText, Message: "x"})
			if err := check(hasErr(err, "not connected"), "Send a un cliente caído -> %v", err); err != nil {
				return err
			}
			r.logf("el servidor cae -> el cliente queda registrado en down; Send -> %v", err)

			if _, err := a.LoadServer(port); err != nil {
				return err
			}
			c2, err := b.ConnectTo(addr)
			if err != nil {
				return err
			}
			if err := check(c1 == c2 && b.Status()[0].Str("state") == "up", "la reconexión debe reusar el mismo Client"); err != nil {
				return err
			}
			r.logf("ConnectTo sobre un cliente caído reconecta el mismo *Client")

			if err := b.Disconnect(addr); err != nil {
				return err
			}
			err = b.Disconnect(addr)
			if err := check(hasErr(err, "client not found") && len(b.Clients()) == 0 && len(b.Status()) == 0, "Disconnect repetido -> %v", err); err != nil {
				return err
			}
			if err := waitFor("4 OnDisconnect", func() bool { mu.Lock(); defer mu.Unlock(); return len(events) == 4 }); err != nil {
				return err
			}
			mu.Lock()
			r.logf("OnDisconnect, una vez por conexión y lado: %v", events)
			mu.Unlock()
			return nil
		},
	},
	{
		Name:    "Hub: heartbeat, Status y timeouts compartidos (HUB-10, 12)",
		Methods: []string{"Hub.Heartbeat", "Hub.Status", "Hub.SetTimeoutClient", "Hub.SetTimeoutServer"},
		run: func(r *report) error {
			a, b, port, addr, err := hubPair()
			if err != nil {
				return err
			}
			defer a.Close()
			defer b.Close()

			if err := check(hasErr(b.SetTimeoutClient(0), "timeout") && hasErr(a.SetTimeoutServer(-1), "timeout"), "un timeout <= 0 debió fallar"); err != nil {
				return err
			}
			if err := a.SetTimeoutServer(2 * time.Second); err != nil {
				return err
			}

			if _, err := b.ConnectTo(addr); err != nil {
				return err
			}
			b.Heartbeat(20 * time.Millisecond)
			if err := waitFor("latido", func() bool { st := b.Status(); return len(st) == 1 && st[0]["last_beat"] != nil }); err != nil {
				return err
			}
			st := b.Status()[0]
			m, _ := a.Metrics(port)
			if err := check(st.Int("failures") == 0 && m.ArrayJson("clients")[0]["last_beat"] != nil, "estado: %v / %v", st, m); err != nil {
				return err
			}
			r.logf("Status: %s", st.ToString())

			// Un servidor que no contesta los latidos a tiempo (un tcp.Server simple que tarda)
			slow, _ := tcp.NewServer("lento", "lento")
			slow.OnRequest(func(conn *tcp.Client, m []byte) ([]byte, error) {
				time.Sleep(300 * time.Millisecond)
				return []byte("{}"), nil
			})
			if err := slow.Listen("127.0.0.1:0"); err != nil {
				return err
			}
			defer slow.Close()

			// El timeout se comparte: el cliente ya creado también queda en 60ms
			if err := b.SetTimeoutClient(60 * time.Millisecond); err != nil {
				return err
			}
			reason := make(chan error, 1)
			b.OnDisconnect(func(srv *tcp.Server, conn *tcp.Client, err error) {
				if conn.ServerName() == "lento" {
					reason <- err
				}
			})
			if _, err := b.ConnectTo(slow.Addr()); err != nil {
				return err
			}
			select {
			case err := <-reason:
				if err := check(hasErr(err, "heartbeat"), "motivo: %v", err); err != nil {
					return err
				}
				r.logf("3 latidos sin respuesta -> cierre con %v", err)
			case <-time.After(3 * time.Second):
				return errors.New("el heartbeat no cerró la conexión que no contesta")
			}
			for _, s := range b.Status() {
				if s.Str("addr") == slow.Addr() && s.Str("state") != "down" {
					return fmt.Errorf("el cliente que no contesta debería estar down: %v", s)
				}
			}

			b.Heartbeat(0)
			time.Sleep(50 * time.Millisecond)
			beats := func() map[string]any {
				out := map[string]any{}
				for _, s := range b.Status() {
					out[s.Str("addr")] = s["last_beat"]
				}
				return out
			}
			before := beats()
			time.Sleep(100 * time.Millisecond)
			after := beats()
			r.logf("Heartbeat(0) lo detiene")
			return check(fmt.Sprint(before) == fmt.Sprint(after), "los latidos siguieron tras Heartbeat(0): %v -> %v", before, after)
		},
	},
	{
		Name:    "Hub: mensajes ilegibles, panics y Close (HUB-13, 14)",
		Methods: []string{"Hub.Close", "Hub.OnMessage", "Hub.Servers", "Hub.Clients"},
		run: func(r *report) error {
			a, b, port, addr, err := hubPair()
			if err != nil {
				return err
			}

			var second atomic.Int32
			a.OnMessage(func(srv *tcp.Server, conn *tcp.Client, m *tcp.Message) { panic("falla del usuario") })
			a.OnMessage(func(srv *tcp.Server, conn *tcp.Client, m *tcp.Message) { second.Add(1) })
			reason := make(chan error, 4)
			a.OnDisconnect(func(srv *tcp.Server, conn *tcp.Client, err error) { reason <- err })

			if _, err := b.ConnectTo(addr); err != nil {
				return err
			}
			b.Send(addr, &tcp.Message{Type: tcp.TypeText, Message: "1"})
			b.Send(addr, &tcp.Message{Type: tcp.TypeText, Message: "2"})
			if err := waitFor("2 mensajes pese al panic", func() bool { return second.Load() == 2 }); err != nil {
				return err
			}
			r.logf("un panic en OnMessage se registra; la siguiente función y el siguiente mensaje siguen")

			// Un cliente tcp simple envía tramas a mano: seq 1 con tipo inválido se salta, seq 2 llega
			raw, _ := tcp.NewClient("raw", "raw")
			if err := raw.ConnectTo(addr); err != nil {
				return err
			}
			before := second.Load()
			raw.Send([]byte(`{"seq":2,"from":{},"header":{},"type":3,"message":"ok"}`))
			raw.Send([]byte(`{"seq":1,"from":{},"header":{},"type":1,"message":"no es int"}`))
			if err := waitFor("seq 2 tras saltar seq 1", func() bool { return second.Load() == before+1 }); err != nil {
				return err
			}
			r.logf("seq 1 inválido se descarta y cuenta; seq 2 (llegado antes) se entrega")

			raw.Send([]byte(`no es json`))
			select {
			case err := <-reason:
				if err := check(hasErr(err, "invalid message"), "motivo: %v", err); err != nil {
					return err
				}
				r.logf("Data sin seq legible -> el Hub cierra la conexión con %v", err)
			case <-time.After(3 * time.Second):
				return errors.New("el Hub no cerró la conexión con datos ilegibles")
			}

			// Un Request que no es latido (JSON cualquiera, type 0 por omisión) llega a las
			// funciones OnRequest del usuario sobre el servidor del Hub
			a.Servers()[0].OnRequest(func(conn *tcp.Client, m []byte) ([]byte, error) { return []byte("eco:" + string(m)), nil })
			var hubClient *tcp.Client
			for _, c := range b.Clients() {
				hubClient = c
			}
			resp, err := hubClient.Request([]byte(`{"op":"forward"}`))
			if err := check(err == nil && string(resp) == `eco:{"op":"forward"}`, "un Request que no es latido no llegó al usuario: %q %v", resp, err); err != nil {
				return err
			}
			r.logf("un Request con JSON que no es latido (sin from.id) llega al OnRequest del usuario")

			if err := a.Close(); err != nil {
				return err
			}
			if err := b.Close(); err != nil {
				return err
			}
			if err := check(len(a.Servers()) == 0 && len(b.Clients()) == 0, "Close no vació el Hub"); err != nil {
				return err
			}
			srv, err := a.LoadServer(port)
			if err != nil {
				return fmt.Errorf("el Hub no se pudo reusar tras Close: %v", err)
			}
			r.logf("Close vacía el Hub y se puede volver a usar (LoadServer en %s)", srv.Addr())
			return a.Close()
		},
	},
}

// ---- Seguridad ----

/**
* secureDial: Servidor con srvSec en un puerto libre y un cliente con cliSec que marca
* a addr (la del servidor si addr es ""). Retorna el servidor, el error de ConnectTo y
* la conexión del lado del servidor (nil si no llegó).
* @param srvSec, cliSec tcp.Security, addr string
* @return *tcp.Server, *tcp.Client, error, func() *tcp.Client
**/
func secureDial(srvSec, cliSec tcp.Security, addr string) (*tcp.Server, *tcp.Client, error, func() *tcp.Client) {
	srv, _ := tcp.NewServer("srv-sec", "servidor")
	srv.SetSecurity(srvSec)
	var peer atomic.Pointer[tcp.Client]
	srv.OnConnect(func(conn *tcp.Client) { peer.Store(conn) })
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		return srv, nil, err, nil
	}
	if addr == "" {
		addr = srv.Addr()
	}
	c, _ := tcp.NewClient("cli-sec", "cliente")
	c.SetSecurity(cliSec)
	err := c.ConnectTo(addr)
	getPeer := func() *tcp.Client {
		waitFor("conexión en el servidor", func() bool { return peer.Load() != nil })
		return peer.Load()
	}
	return srv, c, err, getPeer
}

/**
* writeCA: Escribe en dir una CA y un certificado para 127.0.0.1 firmado por ella.
* @param dir string
* @return cert, key, ca string, err error
**/
func writeCA(dir string) (string, string, string, error) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", "", err
	}
	caTpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "josefina-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	caDer, err := x509.CreateCertificate(rand.Reader, caTpl, caTpl, &caKey.PublicKey, caKey)
	if err != nil {
		return "", "", "", err
	}
	caCert, _ := x509.ParseCertificate(caDer)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", "", err
	}
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "nodo"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return "", "", "", err
	}
	keyDer, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", "", "", err
	}

	files := map[string]*pem.Block{
		"ca.pem":   {Type: "CERTIFICATE", Bytes: caDer},
		"cert.pem": {Type: "CERTIFICATE", Bytes: der},
		"key.pem":  {Type: "EC PRIVATE KEY", Bytes: keyDer},
	}
	for name, block := range files {
		if err := os.WriteFile(filepath.Join(dir, name), pem.EncodeToMemory(block), 0o600); err != nil {
			return "", "", "", err
		}
	}
	return filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"), filepath.Join(dir, "ca.pem"), nil
}

// ---- Rotación del secreto (§7.4) ----

var rotationCases = []testCase{
	{
		Name:    "Seguridad: rotación del secreto, dos secretos durante una ventana (§7.4)",
		Methods: []string{"Client.SetSecurity", "Server.SetSecurity", "Client.ToJson"},
		run: func(r *report) error {
			both := tcp.Security{Secret: "nuevo", Previous: []string{"viejo"}}
			old := tcp.Security{Secret: "viejo"}
			fresh := tcp.Security{Secret: "nuevo"}
			// Mientras un lado acepta los dos secretos, el otro entra con cualquiera
			for _, tc := range []struct {
				name     string
				srv, cli tcp.Security
			}{
				{"servidor con los dos, cliente con el viejo", both, old},
				{"servidor con el viejo, cliente con los dos", old, both},
				{"servidor con los dos, cliente con el nuevo", both, fresh},
				{"servidor con el nuevo, cliente con los dos", fresh, both},
				{"los dos con los dos", both, both},
			} {
				srv, c, err, peer := secureDial(tc.srv, tc.cli, "")
				if err := check(err == nil, "%s: ConnectTo retornó %v", tc.name, err); err != nil {
					srv.Close()
					return err
				}
				cj, pj := c.ToJson(), peer().ToJson()
				if err := check(cj["auth"] == true && pj["auth"] == true, "%s: auth cliente %v, servidor %v", tc.name, cj["auth"], pj["auth"]); err != nil {
					c.Close()
					srv.Close()
					return err
				}
				r.logf("%s: conecta, auth=true en los dos lados", tc.name)
				c.Close()
				srv.Close()
			}

			// Sin un secreto en común no entra, aunque cada lado tenga dos
			for _, tc := range []struct {
				name     string
				srv, cli tcp.Security
			}{
				{"servidor solo con el nuevo, cliente solo con el viejo", fresh, old},
				{"servidor solo con el viejo, cliente solo con el nuevo", old, fresh},
				{"dos secretos en cada lado, ninguno en común", tcp.Security{Secret: "a", Previous: []string{"b"}}, tcp.Security{Secret: "c", Previous: []string{"d"}}},
			} {
				srv, c, err, _ := secureDial(tc.srv, tc.cli, "")
				if err := check(hasErr(err, "authentication failed"), "%s: ConnectTo retornó %v", tc.name, err); err != nil {
					c.Close()
					srv.Close()
					return err
				}
				time.Sleep(50 * time.Millisecond)
				if err := check(srv.Count() == 0, "%s: el servidor registró la conexión (%d)", tc.name, srv.Count()); err != nil {
					c.Close()
					srv.Close()
					return err
				}
				r.logf("%s -> %v", tc.name, err)
				c.Close()
				srv.Close()
			}
			return nil
		},
	},
	{
		Name:    "Seguridad: el certificado y la CA se recargan sin reiniciar (§7.4)",
		Methods: []string{"NewTLS", "Server.SetSecurity", "Client.SetSecurity"},
		run: func(r *report) error {
			dir, err := os.MkdirTemp("", "tcp-reload")
			if err != nil {
				return err
			}
			defer os.RemoveAll(dir)
			cert, key, ca, err := writeCA(dir)
			if err != nil {
				return err
			}
			// The CA and certificate a client keeps from before the change
			oldDir := filepath.Join(dir, "old")
			os.Mkdir(oldDir, 0o700)
			for _, name := range []string{"ca.pem", "cert.pem", "key.pem"} {
				data, _ := os.ReadFile(filepath.Join(dir, name))
				os.WriteFile(filepath.Join(oldDir, name), data, 0o600)
			}

			srvTLS, cliTLS, err := tcp.NewTLS(cert, key, ca)
			if err != nil {
				return err
			}
			srv, _ := tcp.NewServer("srv-reload", "servidor")
			srv.SetSecurity(tcp.Security{ServerTLS: srvTLS})
			if err := srv.Listen("127.0.0.1:0"); err != nil {
				return err
			}
			defer srv.Close()
			// served: the issuer of the certificate the server presents now
			served := func() (string, error) {
				conn, err := tls.Dial("tcp", srv.Addr(), &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS13, GetClientCertificate: cliTLS.GetClientCertificate})
				if err != nil {
					return "", err
				}
				defer conn.Close()
				state := conn.ConnectionState()
				return fmt.Sprintf("%x", state.PeerCertificates[0].AuthorityKeyId), nil
			}
			dial := func(cfg *tls.Config) error {
				c, _ := tcp.NewClient("cli-reload", "cliente")
				c.SetSecurity(tcp.Security{ClientTLS: cfg})
				err := c.ConnectTo(srv.Addr())
				c.Close()
				return err
			}

			if err := check(dial(cliTLS) == nil, "conectar con la CA del arranque"); err != nil {
				return err
			}
			before, err := served()
			if err != nil {
				return err
			}

			// New CA and certificate written over the same files
			time.Sleep(20 * time.Millisecond)
			if _, _, _, err := writeCA(dir); err != nil {
				return err
			}
			if err := check(dial(cliTLS) == nil, "conectar después de cambiar los archivos"); err != nil {
				return err
			}
			after, err := served()
			if err != nil {
				return err
			}
			if err := check(after != before, "el servidor sigue presentando el certificado de la CA vieja"); err != nil {
				return err
			}
			_, oldCli, err := tcp.NewTLS(filepath.Join(oldDir, "cert.pem"), filepath.Join(oldDir, "key.pem"), filepath.Join(oldDir, "ca.pem"))
			if err != nil {
				return err
			}
			if err := check(dial(oldCli) != nil, "un cliente con la CA vieja todavía entra"); err != nil {
				return err
			}

			// A broken file keeps the last valid certificate
			os.WriteFile(cert, []byte("no es un certificado"), 0o600)
			if err := check(dial(cliTLS) == nil, "con un archivo roto debe seguir el último certificado válido"); err != nil {
				return err
			}
			r.logf("TLS mutuo desde archivos: conecta; se reescriben CA, certificado y llave: la conexión nueva usa los nuevos (otra CA emisora) sin reiniciar el servidor, y un cliente con la CA vieja ya no entra")
			r.logf("un certificado roto en disco no corta nada: sigue el último válido")
			return nil
		},
	},
}

/**
* tlsRelay: Intermediario que termina TLS en los dos lados (con su propio certificado)
* y copia los bytes hacia target. Retorna su dirección y cómo cerrarlo.
* @param target string
* @return string, func(), error
**/
func tlsRelay(target string) (string, func(), error) {
	srvTLS, cliTLS, err := tcp.NewTLS("", "", "")
	if err != nil {
		return "", nil, err
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", srvTLS)
	if err != nil {
		return "", nil, err
	}
	go func() {
		for {
			in, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				out, err := tls.Dial("tcp", target, cliTLS)
				if err != nil {
					in.Close()
					return
				}
				go func() { io.Copy(out, in); out.Close() }()
				io.Copy(in, out)
				in.Close()
			}()
		}
	}()
	return ln.Addr().String(), func() { ln.Close() }, nil
}

var securityCases = []testCase{
	{
		Name:    "Seguridad: secreto en el saludo (§7.4)",
		Methods: []string{"Client.SetSecurity", "Server.SetSecurity", "Client.ToJson"},
		run: func(r *report) error {
			// Mismo secreto: conecta, los dos lados quedan autenticados y el tráfico pasa
			srv, c, err, peer := secureDial(tcp.Security{Secret: "s3cr3t"}, tcp.Security{Secret: "s3cr3t"}, "")
			if err != nil {
				srv.Close()
				return fmt.Errorf("con el mismo secreto no conectó: %v", err)
			}
			got := make(chan string, 1)
			srv.OnSend(func(conn *tcp.Client, m []byte) { got <- string(m) })
			c.Send([]byte("hola"))
			select {
			case m := <-got:
				if err := check(m == "hola", "mensaje: %q", m); err != nil {
					return err
				}
			case <-time.After(3 * time.Second):
				return errors.New("no llegó el mensaje por la conexión autenticada")
			}
			cj, pj := c.ToJson(), peer().ToJson()
			if err := check(cj["auth"] == true && pj["auth"] == true && cj["tls"] == false, "ToJson cliente %v, servidor %v", cj, pj); err != nil {
				return err
			}
			if err := check(pj["id"] == "cli-sec", "id del cliente en el servidor: %v", pj["id"]); err != nil {
				return err
			}
			r.logf("mismo secreto: conecta, auth=true en los dos lados, el mensaje pasa")
			c.Close()
			srv.Close()

			// Rechazos: secreto distinto, cliente sin secreto, servidor sin secreto
			for _, tc := range []struct {
				name     string
				srv, cli string
				want     string
			}{
				{"secreto distinto", "s3cr3t", "otro", "authentication failed"},
				{"cliente sin secreto", "s3cr3t", "", "authentication required"},
				{"servidor sin secreto", "", "s3cr3t", "authentication failed"},
			} {
				srv, c, err, _ := secureDial(tcp.Security{Secret: tc.srv}, tcp.Security{Secret: tc.cli}, "")
				if err := check(hasErr(err, tc.want), "%s: ConnectTo retornó %v", tc.name, err); err != nil {
					srv.Close()
					return err
				}
				if tc.srv != "" {
					time.Sleep(50 * time.Millisecond)
					if err := check(srv.Count() == 0, "%s: el servidor registró la conexión (%d)", tc.name, srv.Count()); err != nil {
						srv.Close()
						return err
					}
				}
				r.logf("%s -> %v", tc.name, err)
				c.Close()
				srv.Close()
			}
			return nil
		},
	},
	{
		Name:    "Seguridad: TLS, TLS mutuo y canal ligado (§7.4)",
		Methods: []string{"NewTLS", "Client.SetSecurity", "Server.SetSecurity", "Client.ToJson"},
		run: func(r *report) error {
			srvTLS, cliTLS, err := tcp.NewTLS("", "", "")
			if err != nil {
				return err
			}
			sec := tcp.Security{Secret: "s3cr3t", ServerTLS: srvTLS, ClientTLS: cliTLS}
			srv, c, err, peer := secureDial(sec, sec, "")
			if err != nil {
				srv.Close()
				return fmt.Errorf("TLS autofirmado + secreto no conectó: %v", err)
			}
			resp, err := func() ([]byte, error) {
				srv.OnRequest(func(conn *tcp.Client, m []byte) ([]byte, error) { return append([]byte("eco:"), m...), nil })
				return c.Request([]byte("x"))
			}()
			if err := check(err == nil && string(resp) == "eco:x", "Request por TLS: %q %v", resp, err); err != nil {
				return err
			}
			cj, pj := c.ToJson(), peer().ToJson()
			if err := check(cj["tls"] == true && pj["tls"] == true && cj["auth"] == true, "ToJson cliente %v, servidor %v", cj, pj); err != nil {
				return err
			}
			r.logf("TLS 1.3 autofirmado + secreto: conecta, tls=true y auth=true, Request pasa")

			// Un cliente en TCP plano contra el servidor TLS no conecta
			plain, _ := tcp.NewClient("", "")
			plain.SetTimeout(time.Second)
			plain.SetSecurity(tcp.Security{Secret: "s3cr3t"})
			err = plain.ConnectTo(srv.Addr())
			if err := check(err != nil, "un cliente sin TLS conectó a un servidor TLS"); err != nil {
				return err
			}
			r.logf("cliente sin TLS contra servidor TLS -> %v", err)

			// Intermediario que termina TLS en los dos lados: las pruebas van ligadas a
			// cada sesión TLS, así que con el secreto correcto igual falla
			relay, stop, err := tlsRelay(srv.Addr())
			if err != nil {
				return err
			}
			mitm, _ := tcp.NewClient("", "")
			mitm.SetSecurity(sec)
			err = mitm.ConnectTo(relay)
			stop()
			if err := check(hasErr(err, "authentication failed"), "a través del intermediario: %v", err); err != nil {
				return err
			}
			r.logf("intermediario TLS que reenvía el saludo -> %v", err)

			// Sin secreto, el mismo intermediario pasa: es lo que cubre el secreto
			srvNo, cNo, err, _ := secureDial(tcp.Security{ServerTLS: srvTLS}, tcp.Security{ClientTLS: cliTLS}, "")
			if err != nil {
				srvNo.Close()
				return fmt.Errorf("TLS sin secreto no conectó: %v", err)
			}
			cNo.Close()
			relay, stop, _ = tlsRelay(srvNo.Addr())
			open, _ := tcp.NewClient("", "")
			open.SetSecurity(tcp.Security{ClientTLS: cliTLS})
			err = open.ConnectTo(relay)
			stop()
			open.Close()
			srvNo.Close()
			if err := check(err == nil, "sin secreto el intermediario debía pasar: %v", err); err != nil {
				return err
			}
			r.logf("contraprueba: sin secreto (TLS sin CA) el intermediario pasa")
			c.Close()
			srv.Close()

			// TLS mutuo con CA
			dir, err := os.MkdirTemp("", "tcp-ca-")
			if err != nil {
				return err
			}
			defer os.RemoveAll(dir)
			certFile, keyFile, caFile, err := writeCA(dir)
			if err != nil {
				return err
			}
			mSrv, mCli, err := tcp.NewTLS(certFile, keyFile, caFile)
			if err != nil {
				return err
			}
			srv, c, err, _ = secureDial(tcp.Security{ServerTLS: mSrv}, tcp.Security{ClientTLS: mCli}, "")
			if err != nil {
				srv.Close()
				return fmt.Errorf("TLS mutuo con CA no conectó: %v", err)
			}
			c.Close()
			r.logf("TLS mutuo con CA (certificado para 127.0.0.1): conecta")

			// Un cliente con un certificado que no firmó la CA no conecta
			stranger, _ := tcp.NewClient("", "")
			stranger.SetSecurity(tcp.Security{ClientTLS: cliTLS})
			err = stranger.ConnectTo(srv.Addr())
			srv.Close()
			if err := check(err != nil, "un cliente sin certificado de la CA conectó"); err != nil {
				return err
			}
			r.logf("cliente con certificado ajeno a la CA -> %v", err)

			_, _, err = tcp.NewTLS(certFile, "", "")
			if err := check(hasErr(err, "certificate and the key"), "NewTLS con solo el certificado: %v", err); err != nil {
				return err
			}
			r.logf("NewTLS con certificado sin llave -> %v", err)
			return nil
		},
	},
	{
		Name:    "Seguridad: Hub (§7.4, §14)",
		Methods: []string{"Hub.SetSecurity", "Hub.ConnectTo", "Hub.Send", "Hub.Clients"},
		run: func(r *report) error {
			srvTLS, cliTLS, err := tcp.NewTLS("", "", "")
			if err != nil {
				return err
			}
			sec := tcp.Security{Secret: "cluster", ServerTLS: srvTLS, ClientTLS: cliTLS}
			a, b, _, addr, err := hubPair()
			if err != nil {
				return err
			}
			defer a.Close()
			defer b.Close()
			// La seguridad aplica también al servidor que ya estaba cargado
			a.SetSecurity(sec)
			b.SetSecurity(sec)

			got := make(chan string, 1)
			a.OnMessage(func(srv *tcp.Server, conn *tcp.Client, m *tcp.Message) { got <- fmt.Sprint(m.Message) })
			if _, err := b.ConnectTo(addr); err != nil {
				return fmt.Errorf("hubs con la misma seguridad no conectaron: %v", err)
			}
			if err := b.Send(addr, &tcp.Message{Type: tcp.TypeText, Message: "replica"}); err != nil {
				return err
			}
			select {
			case m := <-got:
				if err := check(m == "replica", "mensaje: %q", m); err != nil {
					return err
				}
			case <-time.After(3 * time.Second):
				return errors.New("no llegó el mensaje entre hubs seguros")
			}
			for _, c := range b.Clients() {
				j := c.ToJson()
				if err := check(j["tls"] == true && j["auth"] == true, "cliente del hub: %v", j); err != nil {
					return err
				}
			}
			r.logf("dos hubs con secreto + TLS: conectan, el mensaje llega, tls=true y auth=true")

			intruder := tcp.NewHub("intruso", "intruso")
			defer intruder.Close()
			intruder.SetSecurity(tcp.Security{Secret: "otro", ClientTLS: cliTLS})
			_, err = intruder.ConnectTo(addr)
			if err := check(hasErr(err, "authentication failed"), "hub con otro secreto: %v", err); err != nil {
				return err
			}
			r.logf("hub con otro secreto -> %v", err)
			return nil
		},
	},
}

// ---- Ejecución y resultados ----

func main() {
	cases = append(cases, securityCases...)
	cases = append(cases, rotationCases...)
	results := make([]caseResult, 0, len(cases))
	covered := map[string]bool{}
	failed := 0
	started := time.Now()

	for i, tc := range cases {
		r := &report{}
		start := time.Now()
		err := runCase(tc, r)
		res := caseResult{
			Name:     tc.Name,
			Methods:  tc.Methods,
			Ok:       err == nil,
			Duration: time.Since(start).Round(time.Millisecond).String(),
			Details:  r.details,
		}
		if err != nil {
			res.Error = err.Error()
			failed++
		}
		for _, m := range tc.Methods {
			covered[m] = true
		}
		results = append(results, res)

		status := "PASS"
		if !res.Ok {
			status = "FAIL"
		}
		fmt.Printf("%2d. [%s] %-55s %8s\n", i+1, status, tc.Name, res.Duration)
		if !res.Ok {
			fmt.Printf("      %s\n", res.Error)
		}
	}

	var missing []string
	for _, m := range publicAPI {
		if !covered[m] {
			missing = append(missing, m)
		}
	}

	fmt.Printf("\n%d/%d casos correctos, %d/%d métodos públicos probados, en %v\n",
		len(cases)-failed, len(cases), len(publicAPI)-len(missing), len(publicAPI), time.Since(started).Round(time.Millisecond))
	if len(missing) > 0 {
		fmt.Println("Sin probar:", strings.Join(missing, ", "))
	}

	if err := writeResults(results, covered, missing, time.Since(started)); err != nil {
		fmt.Println("no se pudieron escribir los resultados:", err)
		os.Exit(1)
	}
	fmt.Println("Resultados en", resultDir)

	if failed > 0 || len(missing) > 0 {
		os.Exit(1)
	}
}

/**
* runCase: Ejecuta un caso convirtiendo un panic en error.
* @param tc testCase, r *report
* @return error
**/
func runCase(tc testCase, r *report) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	return tc.run(r)
}

/**
* writeResults: Escribe result.md (legible) y result.json (para herramientas).
* @param results []caseResult, covered map[string]bool, missing []string, total time.Duration
* @return error
**/
func writeResults(results []caseResult, covered map[string]bool, missing []string, total time.Duration) error {
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		return err
	}

	passed := 0
	for _, res := range results {
		if res.Ok {
			passed++
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Resultados: `tcp`\n\n")
	fmt.Fprintf(&b, "- Fecha: %s\n", time.Now().Format("2006-01-02 15:04:05 MST"))
	fmt.Fprintf(&b, "- Go: %s, %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&b, "- Casos: **%d/%d correctos**\n", passed, len(results))
	fmt.Fprintf(&b, "- Métodos públicos probados: **%d/%d**\n", len(publicAPI)-len(missing), len(publicAPI))
	fmt.Fprintf(&b, "- Duración total: %v\n\n", total.Round(time.Millisecond))
	fmt.Fprintf(&b, "Generado con `go run ./test/tcp` (desde la raíz del repo).\n\n")

	fmt.Fprintf(&b, "## Casos\n\n| # | Caso | Resultado | Duración |\n|---|---|---|---|\n")
	for i, res := range results {
		status := "✅ PASS"
		if !res.Ok {
			status = "❌ FAIL"
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %s |\n", i+1, res.Name, status, res.Duration)
	}

	fmt.Fprintf(&b, "\n## Detalle\n")
	for i, res := range results {
		fmt.Fprintf(&b, "\n### %d. %s\n\n", i+1, res.Name)
		fmt.Fprintf(&b, "Métodos: %s\n\n", "`"+strings.Join(res.Methods, "`, `")+"`")
		if !res.Ok {
			fmt.Fprintf(&b, "**Error:** %s\n\n", res.Error)
		}
		for _, d := range res.Details {
			fmt.Fprintf(&b, "- %s\n", d)
		}
	}

	fmt.Fprintf(&b, "\n## Cobertura de métodos públicos\n\n| Método | Probado en |\n|---|---|\n")
	for _, m := range publicAPI {
		var in []string
		for i, res := range results {
			for _, rm := range res.Methods {
				if rm == m {
					in = append(in, strconv.Itoa(i+1))
					break
				}
			}
		}
		where := "❌ sin probar"
		if covered[m] {
			where = strings.Join(in, ", ")
		}
		fmt.Fprintf(&b, "| `%s` | %s |\n", m, where)
	}

	if err := os.WriteFile(filepath.Join(resultDir, "result.md"), []byte(b.String()), 0o644); err != nil {
		return err
	}

	sort.Strings(missing)
	data, err := json.MarshalIndent(map[string]any{
		"date":     time.Now().Format(time.RFC3339),
		"go":       runtime.Version(),
		"passed":   passed,
		"total":    len(results),
		"duration": total.Round(time.Millisecond).String(),
		"methods":  len(publicAPI),
		"missing":  missing,
		"cases":    results,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(resultDir, "result.json"), data, 0o644)
}
