package websocket

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/cgalvisleon/et/msg"
	gws "github.com/gorilla/websocket"
)

/**
* Server: Servidor WebSocket con canales topic y queue.
**/
type Server struct {
	hs         *handlers
	upgrader   gws.Upgrader
	mu         sync.Mutex          // protege clients, channels, closed, httpServer y listener
	clients    map[*Conn]struct{}  // conexiones vivas (también las que están en OnConnect)
	channels   map[string]*channel // canales con al menos un suscriptor
	connected  atomic.Int64        // clientes aceptados por OnConnect
	closed     bool                // Close ya se llamó
	httpServer *http.Server        // solo con Listen
	listener   net.Listener        // solo con Listen
	wg         sync.WaitGroup      // una por conexión en ServeHTTP
}

/**
* channel: Suscriptores de un canal.
**/
type channel struct {
	topic  map[*Conn]struct{}     // suscriptores topic
	queues map[string]*queueGroup // grupos queue, por nombre
}

/**
* queueGroup: Miembros de un grupo queue y el turno del siguiente.
**/
type queueGroup struct {
	members []*Conn
	next    int
}

/**
* newServer: Crea un servidor sin escuchar.
* @return *Server, error
**/
func newServer() (*Server, error) {
	return &Server{
		hs:       &handlers{},
		upgrader: gws.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024},
		clients:  make(map[*Conn]struct{}),
		channels: make(map[string]*channel),
	}, nil
}

/**
* serveHTTP: Atiende una conexión WebSocket: upgrade, OnConnect, lectura y limpieza.
* Bloquea hasta que la conexión termina.
* @param w http.ResponseWriter, r *http.Request
**/
func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		http.Error(w, msg.MSG_WS_SERVER_CLOSED, http.StatusServiceUnavailable)
		return
	}
	s.wg.Add(1)
	s.mu.Unlock()
	defer s.wg.Done()

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrade ya respondió el error HTTP
	}

	c := newConn(s, conn, r)
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		c.close()
		return
	}
	s.clients[c] = struct{}{}
	s.mu.Unlock()

	go c.writePump()

	if err := s.hs.connect(c); err != nil {
		c.reject(err)
		c.finish()
		return
	}
	c.connected.Store(true)
	s.connected.Add(1)
	c.reply(reply{Type: typeConnected, ID: c.clientID})

	c.readPump()
	c.finish()
}

/**
* removeConn: Olvida la conexión (y lo descuenta si estaba conectado).
* @param c *Conn
**/
func (s *Server) removeConn(c *Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clients[c]; !ok {
		return
	}
	delete(s.clients, c)
	if c.connected.Load() {
		s.connected.Add(-1)
	}
}

/**
* addSub: Registra la suscripción de c en su canal.
* @param c *Conn, sub Subscription
**/
func (s *Server) addSub(c *Conn, sub Subscription) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.channels[sub.Channel]
	if ch == nil {
		ch = &channel{topic: make(map[*Conn]struct{}), queues: make(map[string]*queueGroup)}
		s.channels[sub.Channel] = ch
	}

	if sub.Type == TypeTopic {
		ch.topic[c] = struct{}{}
		return
	}
	group := ch.queues[sub.Queue]
	if group == nil {
		group = &queueGroup{}
		ch.queues[sub.Queue] = group
	}
	group.members = append(group.members, c)
}

/**
* removeSub: Quita la suscripción de c de su canal; borra el grupo o el canal si quedan vacíos.
* @param c *Conn, sub Subscription
**/
func (s *Server) removeSub(c *Conn, sub Subscription) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.channels[sub.Channel]
	if ch == nil {
		return
	}

	if sub.Type == TypeTopic {
		delete(ch.topic, c)
	} else if group := ch.queues[sub.Queue]; group != nil {
		for i, member := range group.members {
			if member != c {
				continue
			}
			group.members = append(group.members[:i], group.members[i+1:]...)
			if i < group.next {
				group.next--
			}
			break
		}
		if len(group.members) == 0 {
			delete(ch.queues, sub.Queue)
		} else if group.next >= len(group.members) {
			group.next = 0
		}
	}

	if len(ch.topic) == 0 && len(ch.queues) == 0 {
		delete(s.channels, sub.Channel)
	}
}

/**
* publishRaw: Entrega data a los suscriptores del canal: a cada topic, y a un miembro
* por grupo queue (el siguiente en turno con espacio). Los suscriptores topic con el
* buffer lleno se desconectan. Retorna cuántos lo recibieron.
* @param channelName string, data json.RawMessage, from string
* @return int, error
**/
func (s *Server) publishRaw(channelName string, data json.RawMessage, from string) (int, error) {
	if channelName == "" {
		return 0, errors.New(msg.MSG_WS_CHANNEL_REQUIRED)
	}

	env := envelope{Type: typeMessage, Channel: channelName, From: from, Data: data}
	topicMsg, err := json.Marshal(env)
	if err != nil {
		return 0, err
	}

	var slow []*Conn
	delivered := 0

	s.mu.Lock()
	if ch := s.channels[channelName]; ch != nil {
		for c := range ch.topic {
			if c.enqueue(topicMsg) {
				delivered++
			} else {
				slow = append(slow, c)
			}
		}

		for name, group := range ch.queues {
			env.Queue = name
			queueMsg, err := json.Marshal(env)
			if err != nil {
				s.mu.Unlock()
				return delivered, err
			}
			n := len(group.members)
			for i := 0; i < n; i++ {
				idx := (group.next + i) % n
				if group.members[idx].enqueue(queueMsg) {
					group.next = (idx + 1) % n
					delivered++
					break
				}
			}
		}
	}
	s.mu.Unlock()

	for _, c := range slow {
		c.kill(errors.New(msg.MSG_WS_SLOW_CONSUMER))
	}
	return delivered, nil
}

/**
* publish: Publica data (codificada en JSON) en el canal desde el servidor.
* @param channelName string, data any
* @return int, error
**/
func (s *Server) publish(channelName string, data any) (int, error) {
	raw, err := toRaw(data)
	if err != nil {
		return 0, err
	}
	return s.publishRaw(channelName, raw, "")
}

/**
* listen: Abre un servidor HTTP propio en addr que atiende todas las rutas con serveHTTP.
* @param addr string
* @return error
**/
func (s *Server) listen(addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New(msg.MSG_WS_SERVER_CLOSED)
	}
	if s.httpServer != nil {
		return errors.New(msg.MSG_WS_ALREADY_LISTENING)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Handler:           http.HandlerFunc(s.serveHTTP),
		ReadHeaderTimeout: writeWait,
	}
	s.httpServer = srv
	s.listener = ln
	go srv.Serve(ln)
	return nil
}

/**
* addr: Retorna la dirección de Listen, o "" si no escucha.
* @return string
**/
func (s *Server) addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

/**
* count: Retorna cuántos clientes están conectados (aceptados por OnConnect).
* @return int
**/
func (s *Server) count() int {
	return int(s.connected.Load())
}

/**
* close: Deja de aceptar, cierra todos los clientes y espera a que terminen sus eventos.
* @return error
**/
func (s *Server) close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	srv := s.httpServer
	clients := make([]*Conn, 0, len(s.clients))
	for c := range s.clients {
		clients = append(clients, c)
	}
	s.mu.Unlock()

	var err error
	if srv != nil {
		err = srv.Close() // no toca las conexiones WebSocket (ya fueron tomadas del servidor HTTP)
	}
	for _, c := range clients {
		c.close()
	}

	s.wg.Wait()
	return err
}

/**
* onConnect: Agrega fn a las funciones que se llaman al conectarse un cliente.
* @param fn func(c *Conn) error
**/
func (s *Server) onConnect(fn func(c *Conn) error) {
	s.hs.onConnect(fn)
}

/**
* onDisconnect: Agrega fn a las funciones que se llaman al desconectarse un cliente.
* @param fn func(c *Conn, err error)
**/
func (s *Server) onDisconnect(fn func(c *Conn, err error)) {
	s.hs.onDisconnect(fn)
}

/**
* onSubscribe: Agrega fn a las funciones que se llaman antes de cada suscripción.
* @param fn func(c *Conn, sub Subscription) error
**/
func (s *Server) onSubscribe(fn func(c *Conn, sub Subscription) error) {
	s.hs.onSubscribe(fn)
}

/**
* onUnsubscribe: Agrega fn a las funciones que se llaman después de cada desuscripción.
* @param fn func(c *Conn, sub Subscription)
**/
func (s *Server) onUnsubscribe(fn func(c *Conn, sub Subscription)) {
	s.hs.onUnsubscribe(fn)
}
