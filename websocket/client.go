package websocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/msg"
	gws "github.com/gorilla/websocket"
)

const (
	requestTimeout = 10 * time.Second // espera de connected, ack y error
	inboxSize      = 256              // mensajes recibidos en espera de sus funciones
)

/**
* Message: Mensaje publicado en un canal, tal como lo recibe el cliente.
**/
type Message struct {
	Channel string          `json:"channel"`
	Queue   string          `json:"queue,omitempty"` // grupo, si llegó por una suscripción queue
	From    string          `json:"from"`            // id del cliente que publicó; "" si fue el servidor
	Data    json.RawMessage `json:"data"`
}

/**
* serverFrame: Cualquier mensaje del servidor al cliente.
**/
type serverFrame struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	Message   string          `json:"message"`
	Delivered *int            `json:"delivered"`
	Channel   string          `json:"channel"`
	Queue     string          `json:"queue"`
	From      string          `json:"from"`
	Data      json.RawMessage `json:"data"`
}

/**
* session: Una conexión del cliente con el servidor. Cada connect crea una nueva.
**/
type session struct {
	conn      *gws.Conn
	writeMu   sync.Mutex // Gorilla admite un solo escritor a la vez
	pendingMu sync.Mutex
	pending   map[string]chan serverFrame // operaciones esperando ack/error, por id
	welcome   chan string                 // recibe el id del mensaje connected
	inbox     chan serverFrame            // message y direct, en orden, para dispatch
	done      chan struct{}               // se cierra al terminar la lectura
	err       error                       // motivo; se lee después de <-done
	endOnce   sync.Once
}

/**
* Client: Cliente que se conecta a un Server.
**/
type Client struct {
	mu            sync.Mutex
	sess          *session                // conexión actual; nil si no hay
	clientID      string                  // asignado por el servidor en connected
	subs          map[string]Subscription // confirmadas por el servidor
	seq           atomic.Int64
	hsMu          sync.RWMutex
	connectFns    []func(c *Client)
	disconnectFns []func(c *Client, err error)
	messageFns    []func(c *Client, m Message)
	directFns     []func(c *Client, data json.RawMessage)
}

/**
* newClient: Crea un cliente sin conectar.
* @return *Client, error
**/
func newClient() (*Client, error) {
	return &Client{subs: make(map[string]Subscription)}, nil
}

/**
* current: Retorna la sesión actual.
* @return *session, error
**/
func (s *Client) current() (*session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sess == nil {
		return nil, errors.New(msg.MSG_WS_NOT_CONNECTED)
	}
	return s.sess, nil
}

/**
* connect: Se conecta a url (ws:// o wss://) y espera a que el servidor acepte la
* conexión. Si OnConnect del servidor la rechaza, retorna MSG_WS_REJECTED con el motivo.
* Al aceptarse llama a las funciones de OnConnect del cliente antes de retornar.
* @param url string
* @return error
**/
func (s *Client) connect(url string) error {
	if _, err := s.current(); err == nil {
		return errors.New(msg.MSG_WS_ALREADY_CONNECTED)
	}

	dialer := gws.Dialer{HandshakeTimeout: requestTimeout}
	conn, resp, err := dialer.Dial(url, nil)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("%w (HTTP %d)", err, resp.StatusCode)
		}
		return err
	}

	sess := &session{
		conn:    conn,
		pending: make(map[string]chan serverFrame),
		welcome: make(chan string, 1),
		inbox:   make(chan serverFrame, inboxSize),
		done:    make(chan struct{}),
	}
	s.mu.Lock()
	if s.sess != nil {
		s.mu.Unlock()
		conn.Close()
		return errors.New(msg.MSG_WS_ALREADY_CONNECTED)
	}
	s.sess = sess
	s.mu.Unlock()

	dispatched := make(chan struct{})
	go s.dispatch(sess, dispatched)
	accepted := make(chan bool, 1)
	go s.readLoop(sess, dispatched, accepted)

	failed := func() {
		accepted <- false
		s.mu.Lock()
		if s.sess == sess {
			s.sess = nil
		}
		s.mu.Unlock()
	}

	select {
	case id := <-sess.welcome:
		s.mu.Lock()
		s.clientID = id
		s.mu.Unlock()
		accepted <- true
	case <-sess.done:
		failed()
		return sess.err
	case <-time.After(requestTimeout):
		sess.end(errors.New(msg.MSG_WS_TIMEOUT))
		failed()
		return errors.New(msg.MSG_WS_TIMEOUT)
	}

	s.hsMu.RLock()
	fns := s.connectFns
	s.hsMu.RUnlock()
	for _, fn := range fns {
		call(func() error { fn(s); return nil })
	}
	return nil
}

/**
* end: Termina la sesión (solo la primera vez tiene efecto).
* @param err error
**/
func (sess *session) end(err error) {
	sess.endOnce.Do(func() {
		var ce *gws.CloseError
		switch {
		case errors.As(err, &ce) && ce.Code == gws.ClosePolicyViolation:
			sess.err = fmt.Errorf(msg.MSG_WS_REJECTED, ce.Text)
		default:
			sess.err = closeReason(err)
		}
		close(sess.done)
		sess.conn.Close()
	})
}

/**
* readLoop: Lee los mensajes del servidor: ack/error a quien los espera, connected a
* connect, message/direct a dispatch. Al terminar limpia el cliente y, si la conexión
* fue aceptada, llama a OnDisconnect después de entregar los mensajes pendientes.
* @param sess *session, dispatched chan struct{}, accepted chan bool
**/
func (s *Client) readLoop(sess *session, dispatched chan struct{}, accepted chan bool) {
	for {
		_, data, err := sess.conn.ReadMessage()
		if err != nil {
			sess.end(err)
			break
		}

		var f serverFrame
		if err := json.Unmarshal(data, &f); err != nil {
			continue
		}
		switch f.Type {
		case typeAck, typeError:
			sess.pendingMu.Lock()
			ch, ok := sess.pending[f.ID]
			delete(sess.pending, f.ID)
			sess.pendingMu.Unlock()
			if ok {
				ch <- f
			}
		case typeConnected:
			select {
			case sess.welcome <- f.ID:
			default:
			}
		case typeMessage, typeDirect:
			select {
			case sess.inbox <- f:
			case <-sess.done:
			}
		}
	}

	close(sess.inbox)
	<-dispatched

	s.mu.Lock()
	if s.sess == sess {
		s.sess = nil
		s.subs = make(map[string]Subscription)
	}
	s.mu.Unlock()

	if <-accepted {
		s.hsMu.RLock()
		fns := s.disconnectFns
		s.hsMu.RUnlock()
		for _, fn := range fns {
			call(func() error { fn(s, sess.err); return nil })
		}
	}
}

/**
* dispatch: Entrega en orden los mensajes recibidos a OnMessage y OnDirect, en su
* propia goroutine para no frenar la lectura de acks.
* @param sess *session, dispatched chan struct{}
**/
func (s *Client) dispatch(sess *session, dispatched chan struct{}) {
	defer close(dispatched)
	for f := range sess.inbox {
		s.hsMu.RLock()
		messageFns, directFns := s.messageFns, s.directFns
		s.hsMu.RUnlock()

		if f.Type == typeDirect {
			for _, fn := range directFns {
				call(func() error { fn(s, f.Data); return nil })
			}
			continue
		}
		m := Message{Channel: f.Channel, Queue: f.Queue, From: f.From, Data: f.Data}
		for _, fn := range messageFns {
			call(func() error { fn(s, m); return nil })
		}
	}
}

/**
* request: Envía una operación al servidor y espera su ack o error.
* @param in inbound
* @return serverFrame, *session (la sesión usada), error
**/
func (s *Client) request(in inbound) (serverFrame, *session, error) {
	sess, err := s.current()
	if err != nil {
		return serverFrame{}, nil, err
	}

	in.ID = strconv.FormatInt(s.seq.Add(1), 10)
	ch := make(chan serverFrame, 1)
	sess.pendingMu.Lock()
	sess.pending[in.ID] = ch
	sess.pendingMu.Unlock()
	defer func() {
		sess.pendingMu.Lock()
		delete(sess.pending, in.ID)
		sess.pendingMu.Unlock()
	}()

	sess.writeMu.Lock()
	sess.conn.SetWriteDeadline(time.Now().Add(writeWait))
	err = sess.conn.WriteJSON(in)
	sess.writeMu.Unlock()
	if err != nil {
		return serverFrame{}, sess, err
	}

	select {
	case f := <-ch:
		if f.Type == typeError {
			return f, sess, errors.New(f.Message)
		}
		return f, sess, nil
	case <-sess.done:
		return serverFrame{}, sess, sess.err
	case <-time.After(requestTimeout):
		return serverFrame{}, sess, errors.New(msg.MSG_WS_TIMEOUT)
	}
}

/**
* track: Guarda sub como confirmada si la sesión sigue siendo la actual.
* @param sess *session, sub Subscription, add bool
**/
func (s *Client) track(sess *session, sub Subscription, add bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sess != sess {
		return
	}
	if add {
		s.subs[sub.Channel] = sub
	} else {
		delete(s.subs, sub.Channel)
	}
}

/**
* subscribe: Se suscribe al canal, tipo topic.
* @param channel string
* @return error
**/
func (s *Client) subscribe(channel string) error {
	_, sess, err := s.request(inbound{Type: typeSubscribe, Channel: channel})
	if err != nil {
		return err
	}
	s.track(sess, Subscription{Channel: channel, Type: TypeTopic}, true)
	return nil
}

/**
* subscribeQueue: Se suscribe al canal, tipo queue, en el grupo queue.
* @param channel, queue string
* @return error
**/
func (s *Client) subscribeQueue(channel, queue string) error {
	_, sess, err := s.request(inbound{Type: typeQueue, Channel: channel, Queue: queue})
	if err != nil {
		return err
	}
	s.track(sess, Subscription{Channel: channel, Queue: queue, Type: TypeQueue}, true)
	return nil
}

/**
* unsubscribe: Quita la suscripción al canal.
* @param channel string
* @return error
**/
func (s *Client) unsubscribe(channel string) error {
	_, sess, err := s.request(inbound{Type: typeUnsubscribe, Channel: channel})
	if err != nil {
		return err
	}
	s.track(sess, Subscription{Channel: channel}, false)
	return nil
}

/**
* publish: Publica data (codificada en JSON) en el canal; retorna cuántos la recibieron.
* @param channel string, data any
* @return int, error
**/
func (s *Client) publish(channel string, data any) (int, error) {
	raw, err := toRaw(data)
	if err != nil {
		return 0, err
	}
	f, _, err := s.request(inbound{Type: typePublish, Channel: channel, Data: raw})
	if err != nil {
		return 0, err
	}
	if f.Delivered == nil {
		return 0, nil
	}
	return *f.Delivered, nil
}

/**
* subscriptions: Retorna las suscripciones confirmadas, ordenadas por canal.
* @return []Subscription
**/
func (s *Client) subscriptions() []Subscription {
	s.mu.Lock()
	result := make([]Subscription, 0, len(s.subs))
	for _, sub := range s.subs {
		result = append(result, sub)
	}
	s.mu.Unlock()
	sort.Slice(result, func(i, j int) bool { return result[i].Channel < result[j].Channel })
	return result
}

/**
* id: Retorna el id que el servidor asignó en la última conexión aceptada.
* @return string
**/
func (s *Client) id() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clientID
}

/**
* close: Cierra la conexión con un cierre normal; OnDisconnect llega después.
* @return error
**/
func (s *Client) close() error {
	sess, err := s.current()
	if err != nil {
		return nil
	}
	sess.conn.WriteControl(gws.CloseMessage, gws.FormatCloseMessage(gws.CloseNormalClosure, ""), time.Now().Add(writeWait))
	sess.end(nil)
	<-sess.done
	return nil
}

/**
* onConnect: Agrega fn a las funciones que se llaman cuando el servidor acepta la conexión.
* @param fn func(c *Client)
**/
func (s *Client) onConnect(fn func(c *Client)) {
	s.hsMu.Lock()
	defer s.hsMu.Unlock()
	s.connectFns = append(s.connectFns, fn)
}

/**
* onDisconnect: Agrega fn a las funciones que se llaman cuando se cierra una conexión aceptada.
* @param fn func(c *Client, err error)
**/
func (s *Client) onDisconnect(fn func(c *Client, err error)) {
	s.hsMu.Lock()
	defer s.hsMu.Unlock()
	s.disconnectFns = append(s.disconnectFns, fn)
}

/**
* onMessage: Agrega fn a las funciones que reciben los mensajes de los canales suscritos.
* @param fn func(c *Client, m Message)
**/
func (s *Client) onMessage(fn func(c *Client, m Message)) {
	s.hsMu.Lock()
	defer s.hsMu.Unlock()
	s.messageFns = append(s.messageFns, fn)
}

/**
* onDirect: Agrega fn a las funciones que reciben los mensajes directos (Conn.Send del servidor).
* @param fn func(c *Client, data json.RawMessage)
**/
func (s *Client) onDirect(fn func(c *Client, data json.RawMessage)) {
	s.hsMu.Lock()
	defer s.hsMu.Unlock()
	s.directFns = append(s.directFns, fn)
}
