package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/cgalvisleon/et/websocket"
	gws "github.com/gorilla/websocket"
)

/**
* Verifica websocket contra su spec.md con clientes Gorilla reales y escribe
* los resultados en test/websocket/result/.
*
* Uso (desde la raíz del repo):
*   go run ./test/websocket          // o con -race: go run -race ./test/websocket
* Sale con código 1 si algún caso falla, si un requisito queda sin probar o si queda
* un método público sin probar.
**/

const (
	resultDir  = "test/websocket/result"
	packageDir = "websocket"
	wait       = 3 * time.Second
)

// requirements: requisitos de spec.md §2.
var requirements = []struct{ ID, Text string }{
	{"WS-01", "Usa gorilla/websocket"},
	{"WS-02", "Conectarse (http.Handler o Listen)"},
	{"WS-03", "Suscribirse tipo topic"},
	{"WS-04", "Suscribirse tipo queue"},
	{"WS-05", "Desuscribirse"},
	{"WS-06", "Publicar (servidor y cliente)"},
	{"WS-07", "OnConnect (puede rechazar)"},
	{"WS-08", "OnDisconnect"},
	{"WS-09", "OnSubscribe (puede rechazar)"},
	{"WS-10", "OnUnsubscribe"},
	{"WS-11", "Modelo catalog"},
	{"WS-12", "Concurrencia y consumidor lento"},
	{"WS-13", "Panic en eventos"},
	{"WS-14", "Cliente: NewClient y Connect devuelven un Client"},
	{"WS-15", "Cliente: suscribir, queue, desuscribir y publicar"},
	{"WS-16", "Cliente: recibir mensajes en orden"},
	{"WS-17", "Cliente: OnConnect, OnDisconnect y reconexión"},
}

// publicAPI: todos los métodos públicos de websocket (catalog.go).
var publicAPI = []string{
	"NewServer", "Server.ServeHTTP", "Server.Listen", "Server.Addr", "Server.Close", "Server.Count", "Server.Publish",
	"Server.OnConnect", "Server.OnDisconnect", "Server.OnSubscribe", "Server.OnUnsubscribe",
	"Conn.ID", "Conn.Request", "Conn.Send", "Conn.Subscribe", "Conn.SubscribeQueue",
	"Conn.Unsubscribe", "Conn.Subscriptions", "Conn.Close",
	"NewClient", "Client.Connect", "Client.Close", "Client.ID", "Client.Subscribe", "Client.SubscribeQueue",
	"Client.Unsubscribe", "Client.Publish", "Client.Subscriptions",
	"Client.OnConnect", "Client.OnDisconnect", "Client.OnMessage", "Client.OnDirect",
}

type testCase struct {
	Name         string
	Requirements []string
	Methods      []string
	run          func(r *report) error
}

type caseResult struct {
	Name         string   `json:"name"`
	Requirements []string `json:"requirements"`
	Methods      []string `json:"methods"`
	Ok           bool     `json:"ok"`
	Duration     string   `json:"duration"`
	Error        string   `json:"error,omitempty"`
	Details      []string `json:"details"`
}

type report struct {
	mu      sync.Mutex
	details []string
}

func (r *report) logf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.details = append(r.details, fmt.Sprintf(format, args...))
}

func check(ok bool, format string, args ...any) error {
	if ok {
		return nil
	}
	return fmt.Errorf(format, args...)
}

func waitFor(what string, cond func() bool) error {
	deadline := time.Now().Add(wait)
	for !cond() {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout esperando: %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil
}

// ---- Cliente de prueba (Gorilla) ----

type frame = map[string]any

type wsClient struct {
	conn     *gws.Conn
	writeMu  sync.Mutex
	in       chan frame
	closed   chan struct{}
	closeErr error
	backlog  []frame
	seq      atomic.Int64
	id       string // asignado por el servidor en connected
}

/**
* dial: Conecta un cliente de prueba y lee sus mensajes en segundo plano.
**/
func dial(url string) (*wsClient, *http.Response, error) {
	conn, resp, err := gws.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, resp, err
	}
	c := &wsClient{conn: conn, in: make(chan frame, 100000), closed: make(chan struct{})}
	defer c.awaitConnected()
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				c.closeErr = err
				close(c.closed)
				return
			}
			var f frame
			json.Unmarshal(data, &f)
			c.in <- f
		}
	}()
	return c, resp, nil
}

/**
* awaitConnected: Espera el mensaje connected (o el cierre, si el servidor rechazó);
* lo que llegue antes queda en backlog.
**/
func (c *wsClient) awaitConnected() {
	deadline := time.After(wait)
	for {
		select {
		case f := <-c.in:
			if f["type"] == "connected" {
				c.id = fmt.Sprint(f["id"])
				return
			}
			c.backlog = append(c.backlog, f)
		case <-c.closed:
			return
		case <-deadline:
			return
		}
	}
}

func (c *wsClient) send(f frame) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteJSON(f)
}

func (c *wsClient) next() (frame, error) {
	if len(c.backlog) > 0 {
		f := c.backlog[0]
		c.backlog = c.backlog[1:]
		return f, nil
	}
	select {
	case f := <-c.in:
		return f, nil
	case <-c.closed:
		select {
		case f := <-c.in:
			return f, nil
		default:
		}
		return nil, fmt.Errorf("conexión cerrada: %v", c.closeErr)
	case <-time.After(wait):
		return nil, errors.New("timeout esperando mensaje")
	}
}

/**
* call: Envía una operación con id y espera su ack o error; guarda lo demás en backlog.
**/
func (c *wsClient) call(f frame) (frame, error) {
	id := strconv.FormatInt(c.seq.Add(1), 10)
	f["id"] = id
	if err := c.send(f); err != nil {
		return nil, err
	}
	var skipped []frame
	defer func() { c.backlog = append(skipped, c.backlog...) }()
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		var got frame
		select {
		case got = <-c.in:
		case <-c.closed:
			return nil, fmt.Errorf("conexión cerrada: %v", c.closeErr)
		case <-time.After(wait):
			return nil, errors.New("timeout esperando respuesta")
		}
		if got["id"] == id {
			if got["type"] == "error" {
				return got, errors.New(fmt.Sprint(got["message"]))
			}
			return got, nil
		}
		skipped = append(skipped, got)
	}
	return nil, errors.New("timeout esperando respuesta")
}

/**
* messages: Espera n mensajes de tipo message/direct.
**/
func (c *wsClient) messages(n int) ([]frame, error) {
	var out []frame
	for len(out) < n {
		f, err := c.next()
		if err != nil {
			return out, err
		}
		if f["type"] == "message" || f["type"] == "direct" {
			out = append(out, f)
		}
	}
	return out, nil
}

/**
* quiet: Verifica que no llegue ningún mensaje de canal en d.
**/
func (c *wsClient) quiet(d time.Duration) error {
	for _, f := range c.backlog {
		if f["type"] == "message" {
			return fmt.Errorf("llegó un mensaje inesperado: %v", f)
		}
	}
	select {
	case f := <-c.in:
		return fmt.Errorf("llegó un mensaje inesperado: %v", f)
	case <-time.After(d):
		return nil
	}
}

func (c *wsClient) close() {
	c.writeMu.Lock()
	c.conn.WriteMessage(gws.CloseMessage, gws.FormatCloseMessage(gws.CloseNormalClosure, ""))
	c.writeMu.Unlock()
	select {
	case <-c.closed:
	case <-time.After(wait):
	}
	c.conn.Close()
}

/**
* setup: Servidor con Listen y su URL ws://.
**/
func setup() (*websocket.Server, string, error) {
	srv, err := websocket.NewServer()
	if err != nil {
		return nil, "", err
	}
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		return nil, "", err
	}
	return srv, "ws://" + srv.Addr() + "/", nil
}

func dialN(url string, n int) ([]*wsClient, error) {
	out := make([]*wsClient, n)
	for i := range out {
		c, _, err := dial(url)
		if err != nil {
			return nil, err
		}
		out[i] = c
	}
	return out, nil
}

// ---- Casos ----

var cases = []testCase{
	{
		Name:         "Conexión: Listen, ServeHTTP y Close",
		Requirements: []string{"WS-01", "WS-02"},
		Methods:      []string{"NewServer", "Server.Listen", "Server.Addr", "Server.Count", "Server.ServeHTTP", "Server.Close"},
		run: func(r *report) error {
			srv, url, err := setup()
			if err != nil {
				return err
			}
			r.logf("Listen en %s", srv.Addr())
			err = srv.Listen("127.0.0.1:0")
			if err := check(err != nil, "un segundo Listen debió fallar"); err != nil {
				return err
			}
			r.logf("segundo Listen -> %v", err)

			clients, err := dialN(url, 3)
			if err != nil {
				return err
			}
			if err := waitFor("Count=3", func() bool { return srv.Count() == 3 }); err != nil {
				return err
			}
			r.logf("3 clientes Gorilla conectados: Count=%d", srv.Count())

			// Montado como http.Handler en un router cualquiera.
			other, _ := websocket.NewServer()
			mux := http.NewServeMux()
			mux.Handle("/ws", other)
			hs := httptest.NewServer(mux)
			mounted, _, err := dial("ws" + strings.TrimPrefix(hs.URL, "http") + "/ws")
			if err != nil {
				return fmt.Errorf("montado en un mux: %w", err)
			}
			if err := waitFor("Count=1 montado", func() bool { return other.Count() == 1 }); err != nil {
				return err
			}
			r.logf("montado en http.ServeMux en /ws: conecta (Count=%d)", other.Count())

			if err := srv.Close(); err != nil {
				return err
			}
			if err := check(srv.Count() == 0, "Count después de Close = %d", srv.Count()); err != nil {
				return err
			}
			for _, c := range clients {
				<-c.closed
			}
			r.logf("Close cerró los 3 clientes; Count=0")

			other.Close()
			<-mounted.closed
			_, resp, err := dial("ws" + strings.TrimPrefix(hs.URL, "http") + "/ws")
			hs.Close()
			if err := check(err != nil && resp != nil && resp.StatusCode == http.StatusServiceUnavailable, "upgrade después de Close debió dar 503: %v", err); err != nil {
				return err
			}
			r.logf("upgrade después de Close -> HTTP %d", resp.StatusCode)
			return nil
		},
	},
	{
		Name:         "Suscripción topic",
		Requirements: []string{"WS-03", "WS-06"},
		Methods:      []string{"Server.Publish", "Server.OnConnect", "Conn.Subscriptions"},
		run: func(r *report) error {
			srv, url, err := setup()
			if err != nil {
				return err
			}
			defer srv.Close()
			var mu sync.Mutex
			var conns []*websocket.Conn
			srv.OnConnect(func(c *websocket.Conn) error {
				mu.Lock()
				conns = append(conns, c)
				mu.Unlock()
				return nil
			})

			clients, err := dialN(url, 3)
			if err != nil {
				return err
			}
			for _, c := range clients {
				if _, err := c.call(frame{"type": "subscribe", "channel": "news"}); err != nil {
					return err
				}
			}
			_, err = clients[0].call(frame{"type": "subscribe", "channel": "news"})
			if err := check(err == nil, "repetir la misma suscripción debió dar ack: %v", err); err != nil {
				return err
			}
			r.logf("repetir la misma suscripción -> ack")

			delivered, err := srv.Publish("news", map[string]any{"n": 1})
			if err := check(err == nil && delivered == 3, "Publish delivered=%d %v", delivered, err); err != nil {
				return err
			}
			for i, c := range clients {
				msgs, err := c.messages(1)
				if err != nil {
					return fmt.Errorf("cliente %d: %w", i, err)
				}
				m := msgs[0]
				if err := check(m["channel"] == "news" && m["from"] == "" && fmt.Sprint(m["data"]) == "map[n:1]", "mensaje: %v", m); err != nil {
					return err
				}
			}
			r.logf("Publish del servidor -> delivered=3; los 3 suscriptores topic lo recibieron (from=\"\")")

			mu.Lock()
			subs := conns[0].Subscriptions()
			mu.Unlock()
			r.logf("Subscriptions() = %+v", subs)
			return check(len(subs) == 1 && subs[0].Channel == "news" && subs[0].Type == websocket.TypeTopic, "Subscriptions: %+v", subs)
		},
	},
	{
		Name:         "Suscripción queue: reparto por turnos",
		Requirements: []string{"WS-04", "WS-06"},
		Methods:      []string{"Server.Publish"},
		run: func(r *report) error {
			srv, url, err := setup()
			if err != nil {
				return err
			}
			defer srv.Close()
			clients, err := dialN(url, 6)
			if err != nil {
				return err
			}
			// 3 en el grupo "a", 2 en el grupo "b", 1 topic.
			for i := 0; i < 3; i++ {
				if _, err := clients[i].call(frame{"type": "queue", "channel": "jobs", "queue": "a"}); err != nil {
					return err
				}
			}
			for i := 3; i < 5; i++ {
				if _, err := clients[i].call(frame{"type": "queue", "channel": "jobs", "queue": "b"}); err != nil {
					return err
				}
			}
			if _, err := clients[5].call(frame{"type": "subscribe", "channel": "jobs"}); err != nil {
				return err
			}

			const n = 30
			total := 0
			for i := 0; i < n; i++ {
				d, err := srv.Publish("jobs", i)
				if err != nil {
					return err
				}
				total += d
			}
			if err := check(total == n*3, "delivered total=%d, esperado %d (1 por grupo + 1 topic)", total, n*3); err != nil {
				return err
			}

			counts := make([]int, 6)
			seen := map[string]map[string]bool{"a": {}, "b": {}}
			for i, c := range clients {
				expected := map[int]int{0: 10, 1: 10, 2: 10, 3: 15, 4: 15, 5: 30}[i]
				msgs, err := c.messages(expected)
				if err != nil {
					return fmt.Errorf("cliente %d: %w", i, err)
				}
				counts[i] = len(msgs)
				if i < 5 {
					group := msgs[0]["queue"].(string)
					for _, m := range msgs {
						key := fmt.Sprint(m["data"])
						if seen[group][key] {
							return fmt.Errorf("grupo %s recibió %s dos veces", group, key)
						}
						seen[group][key] = true
					}
				}
			}
			for _, c := range clients {
				if err := c.quiet(100 * time.Millisecond); err != nil {
					return err
				}
			}
			r.logf("30 mensajes: grupo a (3 miembros) %v, grupo b (2 miembros) %v, topic %d", counts[0:3], counts[3:5], counts[5])
			r.logf("cada grupo recibió cada mensaje exactamente una vez; el topic los 30")
			return check(len(seen["a"]) == n && len(seen["b"]) == n, "cobertura de grupos: a=%d b=%d", len(seen["a"]), len(seen["b"]))
		},
	},
	{
		Name:         "Desuscribirse y reglas de suscripción",
		Requirements: []string{"WS-05", "WS-03", "WS-04"},
		Methods:      []string{"Server.Publish"},
		run: func(r *report) error {
			srv, url, err := setup()
			if err != nil {
				return err
			}
			defer srv.Close()
			c, _, err := dial(url)
			if err != nil {
				return err
			}
			if _, err := c.call(frame{"type": "subscribe", "channel": "x"}); err != nil {
				return err
			}
			_, err = c.call(frame{"type": "queue", "channel": "x", "queue": "g"})
			if err := check(err != nil, "queue sobre un topic del mismo canal debió fallar"); err != nil {
				return err
			}
			r.logf("queue sobre canal ya suscrito como topic -> %v", err)

			if _, err := c.call(frame{"type": "unsubscribe", "channel": "x"}); err != nil {
				return err
			}
			d, _ := srv.Publish("x", "hola")
			if err := check(d == 0, "después de unsubscribe delivered=%d", d); err != nil {
				return err
			}
			if err := c.quiet(150 * time.Millisecond); err != nil {
				return err
			}
			r.logf("después de unsubscribe no llega nada (delivered=0)")

			_, err = c.call(frame{"type": "unsubscribe", "channel": "x"})
			if err := check(err != nil, "unsubscribe de un canal no suscrito debió fallar"); err != nil {
				return err
			}
			r.logf("unsubscribe no suscrito -> %v", err)
			_, err = c.call(frame{"type": "subscribe", "channel": ""})
			if err := check(err != nil, "canal vacío debió fallar"); err != nil {
				return err
			}
			r.logf("canal vacío -> %v", err)

			if _, err := c.call(frame{"type": "queue", "channel": "x", "queue": "g"}); err != nil {
				return fmt.Errorf("tras desuscribirse, queue debió funcionar: %w", err)
			}
			r.logf("tras desuscribirse, se puede suscribir como queue")
			return nil
		},
	},
	{
		Name:         "Publicar desde un cliente y mensajes directos",
		Requirements: []string{"WS-06"},
		Methods:      []string{"Conn.ID", "Conn.Send", "Server.OnConnect"},
		run: func(r *report) error {
			srv, url, err := setup()
			if err != nil {
				return err
			}
			defer srv.Close()
			ids := make(chan string, 2)
			srv.OnConnect(func(c *websocket.Conn) error {
				ids <- c.ID()
				return c.Send(map[string]string{"hola": c.ID()})
			})
			a, _, err := dial(url)
			if err != nil {
				return err
			}
			aID := <-ids
			b, _, err := dial(url)
			if err != nil {
				return err
			}
			<-ids

			for _, c := range []*wsClient{a, b} {
				msgs, err := c.messages(1)
				if err != nil {
					return err
				}
				if err := check(msgs[0]["type"] == "direct", "esperado direct: %v", msgs[0]); err != nil {
					return err
				}
			}
			r.logf("Client.Send en OnConnect -> cada cliente recibió su mensaje direct")

			if _, err := b.call(frame{"type": "subscribe", "channel": "chat"}); err != nil {
				return err
			}
			ack, err := a.call(frame{"type": "publish", "channel": "chat", "data": map[string]any{"texto": "hola"}})
			if err != nil {
				return err
			}
			msgs, err := b.messages(1)
			if err != nil {
				return err
			}
			m := msgs[0]
			if err := check(m["from"] == aID && fmt.Sprint(m["data"]) == "map[texto:hola]" && fmt.Sprint(ack["delivered"]) == "1", "publish cliente: %v ack %v", m, ack); err != nil {
				return err
			}
			r.logf("publish de A -> B lo recibe con from=%s…; ack delivered=%v", aID[:8], ack["delivered"])

			ack, err = a.call(frame{"type": "publish", "channel": "vacio", "data": 1})
			if err := check(err == nil && fmt.Sprint(ack["delivered"]) == "0", "canal sin suscriptores: %v %v", ack, err); err != nil {
				return err
			}
			r.logf("publish en canal sin suscriptores -> ack delivered=0")
			return nil
		},
	},
	{
		Name:         "Protocolo: mensajes inválidos",
		Requirements: []string{"WS-02"},
		Methods:      nil,
		run: func(r *report) error {
			srv, url, err := setup()
			if err != nil {
				return err
			}
			defer srv.Close()
			c, _, err := dial(url)
			if err != nil {
				return err
			}
			c.writeMu.Lock()
			c.conn.WriteMessage(gws.TextMessage, []byte("{no es json"))
			c.conn.WriteMessage(gws.BinaryMessage, []byte{1, 2, 3})
			c.writeMu.Unlock()
			for i := 0; i < 2; i++ {
				f, err := c.next()
				if err != nil {
					return err
				}
				if err := check(f["type"] == "error", "esperado error: %v", f); err != nil {
					return err
				}
				r.logf("inválido -> error %q", f["message"])
			}
			_, err = c.call(frame{"type": "otro"})
			if err := check(err != nil, "tipo desconocido debió fallar"); err != nil {
				return err
			}
			r.logf("tipo desconocido -> %v", err)
			_, err = c.call(frame{"type": "subscribe", "channel": "sigue"})
			r.logf("la conexión sigue abierta: subscribe -> ack")
			return err
		},
	},
	{
		Name:         "OnConnect: autenticar y rechazar",
		Requirements: []string{"WS-07", "WS-13"},
		Methods:      []string{"Server.OnConnect", "Conn.Request", "Conn.Subscribe", "Conn.SubscribeQueue", "Server.Count"},
		run: func(r *report) error {
			srv, url, err := setup()
			if err != nil {
				return err
			}
			defer srv.Close()
			var disconnects atomic.Int32
			srv.OnDisconnect(func(c *websocket.Conn, err error) { disconnects.Add(1) })
			srv.OnConnect(func(c *websocket.Conn) error {
				if c.Request().URL.Query().Get("token") == "panic" {
					panic("token malo")
				}
				if c.Request().URL.Query().Get("token") != "secreto" {
					return errors.New("token inválido")
				}
				if err := c.Subscribe("bienvenida"); err != nil {
					return err
				}
				return c.SubscribeQueue("tareas", "trabajadores")
			})

			bad, _, err := dial(url + "?token=otro")
			if err != nil {
				return err
			}
			<-bad.closed
			var ce *gws.CloseError
			if err := check(errors.As(bad.closeErr, &ce) && ce.Code == gws.ClosePolicyViolation && ce.Text == "token inválido", "cierre esperado 1008: %v", bad.closeErr); err != nil {
				return err
			}
			r.logf("token inválido -> cierre %d %q", ce.Code, ce.Text)

			pan, _, err := dial(url + "?token=panic")
			if err != nil {
				return err
			}
			<-pan.closed
			if err := check(errors.As(pan.closeErr, &ce) && ce.Code == gws.ClosePolicyViolation, "panic en OnConnect debió rechazar: %v", pan.closeErr); err != nil {
				return err
			}
			r.logf("panic en OnConnect -> rechazada: %q", ce.Text)

			good, _, err := dial(url + "?token=secreto")
			if err != nil {
				return err
			}
			if err := waitFor("Count=1", func() bool { return srv.Count() == 1 }); err != nil {
				return err
			}
			d1, _ := srv.Publish("bienvenida", "hola")
			d2, _ := srv.Publish("tareas", "t1")
			if _, err := good.messages(2); err != nil || d1 != 1 || d2 != 1 {
				return fmt.Errorf("suscripciones hechas en OnConnect: %v %d %d", err, d1, d2)
			}
			r.logf("token válido -> conectado (Count=1) y suscrito desde OnConnect a un topic y a una queue")
			time.Sleep(50 * time.Millisecond)
			return check(disconnects.Load() == 0, "OnDisconnect no debe llamarse en conexiones rechazadas: %d", disconnects.Load())
		},
	},
	{
		Name:         "OnDisconnect: cierre del cliente, del servidor y Server.Close",
		Requirements: []string{"WS-08", "WS-10"},
		Methods:      []string{"Server.OnDisconnect", "Server.OnUnsubscribe", "Conn.Close", "Server.Close"},
		run: func(r *report) error {
			srv, url, err := setup()
			if err != nil {
				return err
			}
			var mu sync.Mutex
			var order []string
			var conns []*websocket.Conn
			srv.OnConnect(func(c *websocket.Conn) error {
				mu.Lock()
				conns = append(conns, c)
				mu.Unlock()
				return nil
			})
			srv.OnUnsubscribe(func(c *websocket.Conn, sub websocket.Subscription) {
				mu.Lock()
				order = append(order, "unsubscribe:"+sub.Channel)
				mu.Unlock()
			})
			srv.OnDisconnect(func(c *websocket.Conn, err error) {
				mu.Lock()
				order = append(order, "disconnect:"+err.Error())
				mu.Unlock()
			})
			snapshot := func() []string {
				mu.Lock()
				defer mu.Unlock()
				return append([]string(nil), order...)
			}

			a, _, err := dial(url)
			if err != nil {
				return err
			}
			a.call(frame{"type": "subscribe", "channel": "uno"})
			a.call(frame{"type": "queue", "channel": "dos"})
			a.close()
			if err := waitFor("disconnect de a", func() bool { return len(snapshot()) == 3 }); err != nil {
				return err
			}
			got := snapshot()
			r.logf("el cliente cierra -> %v", got)
			if err := check(got[0] == "unsubscribe:dos" && got[1] == "unsubscribe:uno" && strings.HasPrefix(got[2], "disconnect:"), "orden: %v", got); err != nil {
				return err
			}

			b, _, err := dial(url)
			if err != nil {
				return err
			}
			if err := waitFor("conexión b", func() bool { mu.Lock(); defer mu.Unlock(); return len(conns) == 2 }); err != nil {
				return err
			}
			mu.Lock()
			bConn := conns[1]
			mu.Unlock()
			bConn.Close()
			<-b.closed
			if err := waitFor("disconnect de b", func() bool { return len(snapshot()) == 4 }); err != nil {
				return err
			}
			r.logf("el servidor cierra al cliente (Client.Close) -> %s; el cliente recibió %v", snapshot()[3], b.closeErr)

			c, _, err := dial(url)
			if err != nil {
				return err
			}
			c.call(frame{"type": "subscribe", "channel": "tres"})
			srv.Close()
			got = snapshot()
			if err := check(len(got) == 6, "Server.Close debe esperar los eventos: %v", got); err != nil {
				return err
			}
			r.logf("Server.Close espera a OnUnsubscribe y OnDisconnect -> %v", got[4:])
			return nil
		},
	},
	{
		Name:         "OnSubscribe: rechazar y ampliar",
		Requirements: []string{"WS-09", "WS-10", "WS-13"},
		Methods:      []string{"Server.OnSubscribe", "Server.OnUnsubscribe", "Conn.Subscribe", "Conn.Unsubscribe", "Conn.Subscriptions"},
		run: func(r *report) error {
			srv, url, err := setup()
			if err != nil {
				return err
			}
			defer srv.Close()
			var seen sync.Map
			var unsubs atomic.Int32
			srv.OnSubscribe(func(c *websocket.Conn, sub websocket.Subscription) error {
				seen.Store(sub.Channel, sub)
				switch sub.Channel {
				case "privado":
					return errors.New("sin permiso para privado")
				case "explota":
					panic("boom")
				case "sala":
					// Ampliar: al entrar a una sala, suscribir también a su canal de presencia.
					return c.Subscribe("sala.presencia")
				}
				return nil
			})
			srv.OnUnsubscribe(func(c *websocket.Conn, sub websocket.Subscription) {
				unsubs.Add(1)
				if sub.Channel == "sala" {
					c.Unsubscribe("sala.presencia")
				}
				if sub.Channel == "sala.presencia" {
					panic("panic en OnUnsubscribe")
				}
			})
			var conn atomic.Pointer[websocket.Conn]
			srv.OnConnect(func(c *websocket.Conn) error { conn.Store(c); return nil })

			c, _, err := dial(url)
			if err != nil {
				return err
			}
			_, err = c.call(frame{"type": "subscribe", "channel": "privado"})
			if err := check(err != nil && err.Error() == "sin permiso para privado", "rechazo: %v", err); err != nil {
				return err
			}
			d, _ := srv.Publish("privado", 1)
			if err := check(d == 0, "suscripción rechazada no debe recibir (delivered=%d)", d); err != nil {
				return err
			}
			r.logf("OnSubscribe rechaza -> el cliente recibe %q y no queda suscrito", err)

			_, err = c.call(frame{"type": "subscribe", "channel": "explota"})
			if err := check(err != nil, "panic en OnSubscribe debió rechazar"); err != nil {
				return err
			}
			r.logf("panic en OnSubscribe -> %v", err)

			if _, err := c.call(frame{"type": "queue", "channel": "cola", "queue": "g"}); err != nil {
				return err
			}
			v, _ := seen.Load("cola")
			sub := v.(websocket.Subscription)
			if err := check(sub.Type == websocket.TypeQueue && sub.Queue == "g", "OnSubscribe recibió %+v", sub); err != nil {
				return err
			}
			r.logf("OnSubscribe recibe el tipo y el grupo: %+v", sub)

			if _, err := c.call(frame{"type": "subscribe", "channel": "sala"}); err != nil {
				return err
			}
			subs := conn.Load().Subscriptions()
			if err := check(len(subs) == 3, "sala debió agregar sala.presencia: %+v", subs); err != nil {
				return err
			}
			r.logf("OnSubscribe amplía: subscribe a sala -> también sala.presencia (Subscriptions=%d)", len(subs))

			if _, err := c.call(frame{"type": "unsubscribe", "channel": "sala"}); err != nil {
				return err
			}
			subs = conn.Load().Subscriptions()
			if err := check(len(subs) == 1 && subs[0].Channel == "cola", "después de unsubscribe sala: %+v", subs); err != nil {
				return err
			}
			r.logf("OnUnsubscribe amplía: unsubscribe de sala -> también sala.presencia (panic en OnUnsubscribe registrado, sin efecto)")
			return check(unsubs.Load() == 2, "OnUnsubscribe llamadas=%d", unsubs.Load())
		},
	},
	{
		Name:         "Concurrencia: 20 clientes y 5 publicadores",
		Requirements: []string{"WS-12", "WS-03", "WS-04", "WS-06"},
		Methods:      []string{"Server.Publish"},
		run: func(r *report) error {
			srv, url, err := setup()
			if err != nil {
				return err
			}
			defer srv.Close()
			const nc, np, per = 20, 5, 100
			clients, err := dialN(url, nc)
			if err != nil {
				return err
			}
			var wg sync.WaitGroup
			for i, c := range clients {
				wg.Add(1)
				go func(i int, c *wsClient) {
					defer wg.Done()
					if i < 10 {
						c.call(frame{"type": "subscribe", "channel": "carga"})
					} else {
						c.call(frame{"type": "queue", "channel": "carga", "queue": "q"})
					}
				}(i, c)
			}
			wg.Wait()

			start := time.Now()
			for p := 0; p < np; p++ {
				wg.Add(1)
				go func(p int) {
					defer wg.Done()
					for i := 0; i < per; i++ {
						srv.Publish("carga", map[string]int{"p": p, "i": i})
					}
				}(p)
			}
			wg.Wait()

			total := np * per
			var queueCount atomic.Int32
			var failures atomic.Int32
			for i, c := range clients {
				wg.Add(1)
				go func(i int, c *wsClient) {
					defer wg.Done()
					if i < 10 {
						msgs, err := c.messages(total)
						if err != nil {
							failures.Add(1)
							return
						}
						last := map[float64]float64{}
						for _, m := range msgs {
							d := m["data"].(map[string]any)
							p, n := d["p"].(float64), d["i"].(float64)
							if prev, ok := last[p]; ok && n != prev+1 {
								failures.Add(1) // orden por publicador
								return
							}
							last[p] = n
						}
						return
					}
					for {
						select {
						case f := <-c.in:
							if f["type"] == "message" {
								queueCount.Add(1)
							}
						case <-time.After(300 * time.Millisecond):
							return
						}
					}
				}(i, c)
			}
			wg.Wait()
			r.logf("%d publicaciones concurrentes en %v", total, time.Since(start).Round(time.Millisecond))
			r.logf("10 topic: cada uno recibió las %d en orden por publicador; queue (10 miembros): %d en total", total, queueCount.Load())
			return check(failures.Load() == 0 && int(queueCount.Load()) == total, "fallas=%d queue=%d", failures.Load(), queueCount.Load())
		},
	},
	{
		Name:         "Consumidor lento",
		Requirements: []string{"WS-12"},
		Methods:      []string{"Server.Publish", "Server.OnDisconnect"},
		run: func(r *report) error {
			srv, url, err := setup()
			if err != nil {
				return err
			}
			defer srv.Close()
			reason := make(chan error, 1)
			srv.OnDisconnect(func(c *websocket.Conn, err error) { reason <- err })

			// El lento se suscribe y luego deja de leer del socket.
			slow, _, err := gws.DefaultDialer.Dial(url, nil)
			if err != nil {
				return err
			}
			defer slow.Close()
			slow.ReadMessage() // connected
			slow.WriteJSON(frame{"type": "subscribe", "channel": "grande"})
			slow.ReadMessage() // ack
			fast, _, err := dial(url)
			if err != nil {
				return err
			}
			if _, err := fast.call(frame{"type": "subscribe", "channel": "grande"}); err != nil {
				return err
			}

			payload := strings.Repeat("x", 64*1024)
			start := time.Now()
			published := 0
			var disconnected error
			for i := 0; i < 2000 && disconnected == nil; i++ {
				srv.Publish("grande", payload)
				published++
				select {
				case disconnected = <-reason:
				default:
				}
			}
			elapsed := time.Since(start)
			if disconnected == nil {
				select {
				case disconnected = <-reason:
				case <-time.After(wait):
				}
			}
			if err := check(disconnected != nil && strings.Contains(disconnected.Error(), "slow") || disconnected != nil && strings.Contains(disconnected.Error(), "lento"), "el lento debió desconectarse: %v", disconnected); err != nil {
				return err
			}
			r.logf("tras %d mensajes de 64 KiB, el lector lento se desconectó: %v", published, disconnected)
			r.logf("Publish nunca se bloqueó (%v en total)", elapsed.Round(time.Millisecond))
			msgs, err := fast.messages(published)
			if err != nil {
				return fmt.Errorf("el cliente rápido: %w", err)
			}
			r.logf("el cliente rápido recibió los %d", len(msgs))
			return nil
		},
	},
	{
		Name:         "Cliente Go: NewClient, Connect y rechazo",
		Requirements: []string{"WS-14", "WS-17", "WS-07"},
		Methods:      []string{"NewClient", "Client.Connect", "Client.ID", "Client.Close", "Client.OnConnect", "Conn.ID"},
		run: func(r *report) error {
			srv, url, err := setup()
			if err != nil {
				return err
			}
			serverIDs := make(chan string, 4)
			srv.OnConnect(func(c *websocket.Conn) error {
				if c.Request().URL.Query().Get("token") != "ok" {
					return errors.New("token inválido")
				}
				serverIDs <- c.ID()
				return nil
			})

			c, err := websocket.NewClient()
			if err != nil {
				return err
			}
			var connected atomic.Int32
			c.OnConnect(func(c *websocket.Client) { connected.Add(1) })

			err = c.Connect(url + "?token=malo")
			if err := check(err != nil && strings.Contains(err.Error(), "token inválido"), "Connect rechazado debió retornar el motivo: %v", err); err != nil {
				return err
			}
			r.logf("Connect rechazado por OnConnect del servidor -> %v", err)

			if err := c.Connect(url + "?token=ok"); err != nil {
				return fmt.Errorf("Connect tras un rechazo: %w", err)
			}
			if err := check(connected.Load() == 1, "OnConnect del cliente debe correr antes de que Connect retorne"); err != nil {
				return err
			}
			serverID := <-serverIDs
			if err := check(c.ID() == serverID && c.ID() != "", "ID del cliente %q, en el servidor %q", c.ID(), serverID); err != nil {
				return err
			}
			r.logf("Connect aceptado: Client.ID()=%s… coincide con Conn.ID() del servidor; OnConnect del cliente llamado", c.ID()[:8])

			err = c.Connect(url + "?token=ok")
			if err := check(err != nil, "un segundo Connect debió fallar"); err != nil {
				return err
			}
			r.logf("segundo Connect -> %v", err)

			if err := c.Close(); err != nil {
				return err
			}
			_, err = c.Publish("x", 1)
			if err := check(err != nil, "Publish sin conexión debió fallar"); err != nil {
				return err
			}
			r.logf("Close; operación sin conexión -> %v", err)

			srv.Close()
			err = c.Connect(url + "?token=ok")
			r.logf("Connect a un servidor cerrado -> %v", err)
			return check(err != nil, "Connect a un servidor cerrado debió fallar")
		},
	},
	{
		Name:         "Cliente Go: suscribir, queue, publicar y recibir en orden",
		Requirements: []string{"WS-15", "WS-16", "WS-03", "WS-04", "WS-06", "WS-09"},
		Methods:      []string{"Client.Subscribe", "Client.SubscribeQueue", "Client.Unsubscribe", "Client.Publish", "Client.Subscriptions", "Client.OnMessage", "Client.OnDirect", "Conn.Send"},
		run: func(r *report) error {
			srv, url, err := setup()
			if err != nil {
				return err
			}
			defer srv.Close()
			srv.OnSubscribe(func(c *websocket.Conn, sub websocket.Subscription) error {
				if sub.Channel == "privado" {
					return errors.New("sin permiso")
				}
				return nil
			})
			srv.OnConnect(func(c *websocket.Conn) error { return c.Send("bienvenido") })

			type inbox struct {
				mu     sync.Mutex
				msgs   []websocket.Message
				direct []string
			}
			newClient := func(box *inbox) (*websocket.Client, error) {
				c, _ := websocket.NewClient()
				c.OnMessage(func(c *websocket.Client, m websocket.Message) {
					box.mu.Lock()
					box.msgs = append(box.msgs, m)
					box.mu.Unlock()
				})
				c.OnDirect(func(c *websocket.Client, data json.RawMessage) {
					box.mu.Lock()
					box.direct = append(box.direct, string(data))
					box.mu.Unlock()
				})
				return c, c.Connect(url)
			}
			count := func(box *inbox) int { box.mu.Lock(); defer box.mu.Unlock(); return len(box.msgs) }

			boxes := make([]*inbox, 4)
			clients := make([]*websocket.Client, 4)
			for i := range clients {
				boxes[i] = &inbox{}
				if clients[i], err = newClient(boxes[i]); err != nil {
					return err
				}
				defer clients[i].Close()
			}
			if err := waitFor("OnDirect", func() bool {
				boxes[0].mu.Lock()
				defer boxes[0].mu.Unlock()
				return len(boxes[0].direct) == 1
			}); err != nil {
				return err
			}
			r.logf("OnDirect recibió el Conn.Send del servidor: %s", boxes[0].direct[0])

			// 0 y 1: topic; 2 y 3: queue "w". El 0 publica.
			for i := 0; i < 2; i++ {
				if err := clients[i].Subscribe("eventos"); err != nil {
					return err
				}
			}
			for i := 2; i < 4; i++ {
				if err := clients[i].SubscribeQueue("eventos", "w"); err != nil {
					return err
				}
			}
			err = clients[0].Subscribe("privado")
			if err := check(err != nil && err.Error() == "sin permiso", "OnSubscribe del servidor debió rechazar: %v", err); err != nil {
				return err
			}
			r.logf("Subscribe rechazado por el servidor -> %v", err)
			subs := clients[2].Subscriptions()
			if err := check(len(subs) == 1 && subs[0].Type == websocket.TypeQueue && subs[0].Queue == "w", "Subscriptions: %+v", subs); err != nil {
				return err
			}

			const n = 500
			delivered := 0
			for i := 0; i < n; i++ {
				d, err := clients[0].Publish("eventos", i)
				if err != nil {
					return err
				}
				delivered += d
			}
			if err := check(delivered == n*3, "delivered=%d, esperado %d", delivered, n*3); err != nil {
				return err
			}
			if err := waitFor("mensajes", func() bool {
				return count(boxes[0]) == n && count(boxes[1]) == n && count(boxes[2])+count(boxes[3]) == n
			}); err != nil {
				return err
			}
			boxes[1].mu.Lock()
			for i, m := range boxes[1].msgs {
				if string(m.Data) != strconv.Itoa(i) || m.From != clients[0].ID() || m.Channel != "eventos" {
					boxes[1].mu.Unlock()
					return fmt.Errorf("mensaje %d fuera de orden o incorrecto: %+v", i, m)
				}
			}
			boxes[1].mu.Unlock()
			r.logf("%d Publish desde un Client (delivered=%d): los topic recibieron %d en orden con From=ID del publicador; la queue repartió %d/%d", n, delivered, n, count(boxes[2]), count(boxes[3]))

			if err := clients[1].Unsubscribe("eventos"); err != nil {
				return err
			}
			d, _ := clients[0].Publish("eventos", "tras unsubscribe")
			if err := check(d == 2, "después de Unsubscribe delivered=%d (esperado 2)", d); err != nil {
				return err
			}
			r.logf("Unsubscribe confirmado: la siguiente publicación llega a 2 (el publicador y la queue)")

			// Una función de OnMessage puede llamar a Subscribe sin bloquearse.
			sub := make(chan error, 1)
			clients[3].OnMessage(func(c *websocket.Client, m websocket.Message) {
				if m.Channel == "gatillo" {
					sub <- c.Subscribe("desde-handler")
				}
			})
			clients[3].Subscribe("gatillo")
			srv.Publish("gatillo", 1)
			select {
			case err := <-sub:
				if err != nil {
					return err
				}
			case <-time.After(wait):
				return errors.New("Subscribe dentro de OnMessage se bloqueó")
			}
			r.logf("Subscribe desde dentro de OnMessage -> ok (sin bloqueo)")
			return nil
		},
	},
	{
		Name:         "Cliente Go: desconexión y reconexión",
		Requirements: []string{"WS-17", "WS-08"},
		Methods:      []string{"Client.OnDisconnect", "Client.Connect", "Client.Subscriptions", "Conn.Close"},
		run: func(r *report) error {
			srv, url, err := setup()
			if err != nil {
				return err
			}
			var conns sync.Map
			srv.OnConnect(func(c *websocket.Conn) error { conns.Store(c.ID(), c); return nil })

			c, _ := websocket.NewClient()
			var disconnects atomic.Int32
			reasons := make(chan string, 4)
			c.OnDisconnect(func(c *websocket.Client, err error) {
				reasons <- err.Error()
				if disconnects.Add(1) == 1 {
					c.Connect(url) // reconectar
				}
			})
			if err := c.Connect(url); err != nil {
				return err
			}
			c.Subscribe("a")
			firstID := c.ID()

			v, _ := conns.Load(firstID)
			v.(*websocket.Conn).Close() // el servidor corta
			r.logf("el servidor cierra la conexión -> OnDisconnect: %s", <-reasons)
			if err := waitFor("reconexión", func() bool { return c.ID() != firstID && c.ID() != "" }); err != nil {
				return err
			}
			if err := check(len(c.Subscriptions()) == 0, "tras reconectar no debe quedar ninguna suscripción: %+v", c.Subscriptions()); err != nil {
				return err
			}
			if err := c.Subscribe("a"); err != nil {
				return fmt.Errorf("después de reconectar: %w", err)
			}
			r.logf("se reconectó desde OnDisconnect con otro ID; las suscripciones se limpian y se vuelven a hacer")

			srv.Close()
			r.logf("el servidor se apaga -> OnDisconnect: %s", <-reasons)
			return check(disconnects.Load() == 2, "OnDisconnect llamadas=%d", disconnects.Load())
		},
	},
	{
		Name:         "Modelo catalog (análisis del código)",
		Requirements: []string{"WS-11", "WS-01"},
		Methods:      nil,
		run: func(r *report) error {
			return checkCatalog(r)
		},
	},
}

/**
* checkCatalog: Verifica en el código de websocket que todo método o función
* pública esté en catalog.go y solo llame a su versión privada, y que se use Gorilla.
**/
func checkCatalog(r *report) error {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, packageDir, nil, 0)
	if err != nil {
		return err
	}
	pkg := pkgs["websocket"]
	if pkg == nil {
		return errors.New("no se encontró el paquete websocket")
	}

	gorilla := false
	var exported []string
	var problems []string
	for path, file := range pkg.Files {
		for _, imp := range file.Imports {
			if imp.Path.Value == `"github.com/gorilla/websocket"` {
				gorilla = true
			}
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !fn.Name.IsExported() {
				continue
			}
			name := fn.Name.Name
			if fn.Recv != nil {
				t := fn.Recv.List[0].Type
				if star, ok := t.(*ast.StarExpr); ok {
					t = star.X
				}
				name = t.(*ast.Ident).Name + "." + name
			}
			exported = append(exported, name)
			if filepath.Base(path) != "catalog.go" {
				problems = append(problems, name+" está fuera de catalog.go ("+filepath.Base(path)+")")
				continue
			}
			private := string(unicode.ToLower(rune(fn.Name.Name[0]))) + fn.Name.Name[1:]
			if strings.ToUpper(fn.Name.Name) == fn.Name.Name {
				private = strings.ToLower(fn.Name.Name) // ID -> id
			}
			if len(fn.Body.List) != 1 || !strings.Contains(callsIn(fn.Body.List[0]), private+"(") {
				problems = append(problems, name+" no es un envoltorio de "+private)
			}
		}
	}
	sort.Strings(exported)
	r.logf("usa github.com/gorilla/websocket: %v", gorilla)
	r.logf("%d métodos públicos, todos en catalog.go como envoltorios de su versión privada", len(exported))

	want := append([]string(nil), publicAPI...)
	sort.Strings(want)
	if strings.Join(want, ",") != strings.Join(exported, ",") {
		problems = append(problems, fmt.Sprintf("la lista publicAPI no coincide con el código: %v", exported))
	}
	if !gorilla {
		problems = append(problems, "no importa github.com/gorilla/websocket")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

/**
* callsIn: Retorna las funciones llamadas dentro de n, como "nombre(".
**/
func callsIn(n ast.Node) string {
	var calls []string
	ast.Inspect(n, func(x ast.Node) bool {
		if call, ok := x.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				calls = append(calls, sel.Sel.Name+"(")
			} else if id, ok := call.Fun.(*ast.Ident); ok {
				calls = append(calls, id.Name+"(")
			}
		}
		return true
	})
	return strings.Join(calls, " ")
}

// ---- Ejecución y resultados ----

func main() {
	if _, err := os.Stat(packageDir); err != nil {
		fmt.Println("ejecutar desde la raíz del repo: go run ./test/websocket")
		os.Exit(1)
	}

	results := make([]caseResult, 0, len(cases))
	coveredMethods := map[string]bool{}
	reqStatus := map[string]string{} // "" sin probar, "ok", "fail"
	failed := 0
	started := time.Now()

	for i, tc := range cases {
		r := &report{}
		start := time.Now()
		err := runCase(tc, r)
		res := caseResult{
			Name:         tc.Name,
			Requirements: tc.Requirements,
			Methods:      tc.Methods,
			Ok:           err == nil,
			Duration:     time.Since(start).Round(time.Millisecond).String(),
			Details:      r.details,
		}
		if err != nil {
			res.Error = err.Error()
			failed++
		}
		for _, m := range tc.Methods {
			coveredMethods[m] = true
		}
		for _, req := range tc.Requirements {
			if !res.Ok {
				reqStatus[req] = "fail"
			} else if reqStatus[req] == "" {
				reqStatus[req] = "ok"
			}
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

	var missingMethods, missingReqs []string
	for _, m := range publicAPI {
		if !coveredMethods[m] {
			missingMethods = append(missingMethods, m)
		}
	}
	reqOK := 0
	for _, req := range requirements {
		switch reqStatus[req.ID] {
		case "ok":
			reqOK++
		case "":
			missingReqs = append(missingReqs, req.ID)
		}
	}

	fmt.Printf("\n%d/%d casos correctos, %d/%d requisitos cumplidos, %d/%d métodos públicos probados, en %v\n",
		len(cases)-failed, len(cases), reqOK, len(requirements), len(publicAPI)-len(missingMethods), len(publicAPI), time.Since(started).Round(time.Millisecond))
	if len(missingMethods) > 0 {
		fmt.Println("Métodos sin probar:", strings.Join(missingMethods, ", "))
	}
	if len(missingReqs) > 0 {
		fmt.Println("Requisitos sin probar:", strings.Join(missingReqs, ", "))
	}

	if err := writeResults(results, reqStatus, missingMethods, time.Since(started)); err != nil {
		fmt.Println("no se pudieron escribir los resultados:", err)
		os.Exit(1)
	}
	fmt.Println("Resultados en", resultDir)

	if failed > 0 || len(missingMethods) > 0 || len(missingReqs) > 0 {
		os.Exit(1)
	}
}

func runCase(tc testCase, r *report) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	return tc.run(r)
}

func writeResults(results []caseResult, reqStatus map[string]string, missing []string, total time.Duration) error {
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		return err
	}
	passed := 0
	for _, res := range results {
		if res.Ok {
			passed++
		}
	}
	reqOK := 0
	for _, req := range requirements {
		if reqStatus[req.ID] == "ok" {
			reqOK++
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Resultados: `websocket`\n\n")
	fmt.Fprintf(&b, "- Fecha: %s\n", time.Now().Format("2006-01-02 15:04:05 MST"))
	fmt.Fprintf(&b, "- Go: %s, %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&b, "- Casos: **%d/%d correctos**\n", passed, len(results))
	fmt.Fprintf(&b, "- Requisitos de `spec.md`: **%d/%d cumplidos**\n", reqOK, len(requirements))
	fmt.Fprintf(&b, "- Métodos públicos probados: **%d/%d**\n", len(publicAPI)-len(missing), len(publicAPI))
	fmt.Fprintf(&b, "- Duración total: %v\n\n", total.Round(time.Millisecond))
	fmt.Fprintf(&b, "Generado con `go run ./test/websocket` (desde la raíz del repo).\n\n")

	fmt.Fprintf(&b, "## Requisitos\n\n| ID | Requisito | Estado | Casos |\n|---|---|---|---|\n")
	for _, req := range requirements {
		var in []string
		for i, res := range results {
			for _, id := range res.Requirements {
				if id == req.ID {
					in = append(in, strconv.Itoa(i+1))
				}
			}
		}
		status := map[string]string{"ok": "✅ cumple", "fail": "❌ falla", "": "⚠️ sin probar"}[reqStatus[req.ID]]
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", req.ID, req.Text, status, strings.Join(in, ", "))
	}

	fmt.Fprintf(&b, "\n## Casos\n\n| # | Caso | Requisitos | Resultado | Duración |\n|---|---|---|---|---|\n")
	for i, res := range results {
		status := "✅ PASS"
		if !res.Ok {
			status = "❌ FAIL"
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s |\n", i+1, res.Name, strings.Join(res.Requirements, ", "), status, res.Duration)
	}

	fmt.Fprintf(&b, "\n## Detalle\n")
	for i, res := range results {
		fmt.Fprintf(&b, "\n### %d. %s\n\n", i+1, res.Name)
		if len(res.Methods) > 0 {
			fmt.Fprintf(&b, "Métodos: `%s`\n\n", strings.Join(res.Methods, "`, `"))
		}
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
		where := strings.Join(in, ", ")
		if where == "" {
			where = "❌ sin probar"
		}
		fmt.Fprintf(&b, "| `%s` | %s |\n", m, where)
	}

	if err := os.WriteFile(filepath.Join(resultDir, "result.md"), []byte(b.String()), 0o644); err != nil {
		return err
	}
	data, err := json.MarshalIndent(map[string]any{
		"date":         time.Now().Format(time.RFC3339),
		"go":           runtime.Version(),
		"passed":       passed,
		"total":        len(results),
		"requirements": reqStatus,
		"missing":      missing,
		"duration":     total.Round(time.Millisecond).String(),
		"cases":        results,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(resultDir, "result.json"), data, 0o644)
}
