package websocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	gws "github.com/gorilla/websocket"
)

/**
* Conn: Un cliente conectado, visto desde el servidor.
**/
type Conn struct {
	clientID  string
	server    *Server
	conn      *gws.Conn
	req       *http.Request
	out       chan []byte   // mensajes en espera de envío; los escribe writePump
	done      chan struct{} // se cierra al terminar la conexión
	killOnce  sync.Once
	closeErr  error                   // motivo; se lee después de <-done
	subMu     sync.Mutex              // protege subs; una operación de suscripción a la vez
	subs      map[string]Subscription // por canal; nil cuando la conexión terminó
	connected atomic.Bool             // OnConnect lo aceptó
}

/**
* newConn: Crea la conexión de un cliente recién aceptado.
* @param server *Server, conn *gws.Conn, r *http.Request
* @return *Conn
**/
func newConn(server *Server, conn *gws.Conn, r *http.Request) *Conn {
	return &Conn{
		clientID: reg.ULID(),
		server:   server,
		conn:     conn,
		req:      r,
		out:      make(chan []byte, sendBuffer),
		done:     make(chan struct{}),
		subs:     make(map[string]Subscription),
	}
}

/**
* id: Retorna el id del cliente.
* @return string
**/
func (s *Conn) id() string {
	return s.clientID
}

/**
* request: Retorna la solicitud HTTP con la que se conectó (encabezados, query, etc.).
* @return *http.Request
**/
func (s *Conn) request() *http.Request {
	return s.req
}

/**
* enqueue: Deja b en el buffer de salida sin bloquear; false si está lleno o cerrado.
* @param b []byte
* @return bool
**/
func (s *Conn) enqueue(b []byte) bool {
	select {
	case <-s.done:
		return false
	default:
	}
	select {
	case s.out <- b:
		return true
	default:
		return false
	}
}

/**
* send: Envía data solo a este cliente como mensaje tipo direct.
* @param data any
* @return error
**/
func (s *Conn) send(data any) error {
	raw, err := toRaw(data)
	if err != nil {
		return err
	}
	b, err := json.Marshal(direct{Type: typeDirect, Data: raw})
	if err != nil {
		return err
	}
	if s.enqueue(b) {
		return nil
	}
	select {
	case <-s.done:
		return errors.New(msg.MSG_WS_CLOSED)
	default:
		return errors.New(msg.MSG_WS_SLOW_CONSUMER)
	}
}

/**
* reply: Envía un ack o un error; si el buffer está lleno, desconecta al cliente.
* @param r reply
**/
func (s *Conn) reply(r reply) {
	b, _ := json.Marshal(r)
	if !s.enqueue(b) {
		s.kill(errors.New(msg.MSG_WS_SLOW_CONSUMER))
	}
}

/**
* replyErr: Responde ack si err es nil, o error con su texto.
* @param id string, err error
**/
func (s *Conn) replyErr(id string, err error) {
	if err != nil {
		s.reply(reply{Type: typeError, ID: id, Message: err.Error()})
		return
	}
	s.reply(reply{Type: typeAck, ID: id})
}

/**
* checkLocked: Revisa si sub se puede agregar (requiere subMu). done indica que no hay
* nada más que hacer: ya existe igual (err nil) o hay un conflicto (err).
* @param sub Subscription
* @return bool, error
**/
func (s *Conn) checkLocked(sub Subscription) (bool, error) {
	if s.subs == nil {
		return true, errors.New(msg.MSG_WS_CLOSED)
	}
	if current, ok := s.subs[sub.Channel]; ok {
		if current == sub {
			return true, nil
		}
		return true, fmt.Errorf(msg.MSG_WS_ALREADY_SUBSCRIBED, sub.Channel)
	}
	return false, nil
}

/**
* addSubscription: Suscribe al cliente: valida, llama a onSubscribe sin locks tomados,
* vuelve a validar y registra.
* @param sub Subscription
* @return error
**/
func (s *Conn) addSubscription(sub Subscription) error {
	if sub.Channel == "" {
		return errors.New(msg.MSG_WS_CHANNEL_REQUIRED)
	}

	s.subMu.Lock()
	done, err := s.checkLocked(sub)
	s.subMu.Unlock()
	if done {
		return err
	}

	if err := s.server.hs.subscribe(s, sub); err != nil {
		return err
	}

	s.subMu.Lock()
	defer s.subMu.Unlock()
	if done, err := s.checkLocked(sub); done {
		return err
	}
	s.subs[sub.Channel] = sub
	s.server.addSub(s, sub)
	return nil
}

/**
* subscribe: Suscribe al cliente al canal, tipo topic.
* @param channel string
* @return error
**/
func (s *Conn) subscribe(channel string) error {
	return s.addSubscription(Subscription{Channel: channel, Type: TypeTopic})
}

/**
* subscribeQueue: Suscribe al cliente al canal, tipo queue, en el grupo queue.
* @param channel, queue string
* @return error
**/
func (s *Conn) subscribeQueue(channel, queue string) error {
	return s.addSubscription(Subscription{Channel: channel, Queue: queue, Type: TypeQueue})
}

/**
* unsubscribe: Quita la suscripción al canal y llama a onUnsubscribe.
* @param channel string
* @return error
**/
func (s *Conn) unsubscribe(channel string) error {
	if channel == "" {
		return errors.New(msg.MSG_WS_CHANNEL_REQUIRED)
	}

	s.subMu.Lock()
	if s.subs == nil {
		s.subMu.Unlock()
		return errors.New(msg.MSG_WS_CLOSED)
	}
	sub, ok := s.subs[channel]
	if !ok {
		s.subMu.Unlock()
		return fmt.Errorf(msg.MSG_WS_NOT_SUBSCRIBED, channel)
	}
	delete(s.subs, channel)
	s.server.removeSub(s, sub)
	s.subMu.Unlock()

	s.server.hs.unsubscribe(s, sub)
	return nil
}

/**
* subscriptions: Retorna las suscripciones actuales, ordenadas por canal.
* @return []Subscription
**/
func (s *Conn) subscriptions() []Subscription {
	s.subMu.Lock()
	result := make([]Subscription, 0, len(s.subs))
	for _, sub := range s.subs {
		result = append(result, sub)
	}
	s.subMu.Unlock()

	sort.Slice(result, func(i, j int) bool { return result[i].Channel < result[j].Channel })
	return result
}

/**
* close: Cierra la conexión desde el servidor con un cierre normal.
* @return error
**/
func (s *Conn) close() error {
	s.conn.WriteControl(gws.CloseMessage, gws.FormatCloseMessage(gws.CloseNormalClosure, ""), time.Now().Add(writeWait))
	s.kill(nil)
	return nil
}

/**
* reject: Cierra la conexión con un cierre 1008 (policy violation) y el motivo.
* @param err error
**/
func (s *Conn) reject(err error) {
	reason := err.Error()
	if len(reason) > maxCloseReason {
		reason = reason[:maxCloseReason]
	}
	s.conn.WriteControl(gws.CloseMessage, gws.FormatCloseMessage(gws.ClosePolicyViolation, reason), time.Now().Add(writeWait))
	s.kill(err)
}

/**
* kill: Termina la conexión (solo la primera vez tiene efecto). Es barato y se puede
* llamar desde cualquier goroutine; la limpieza la hace finish en la goroutine de la conexión.
* @param err error
**/
func (s *Conn) kill(err error) {
	s.killOnce.Do(func() {
		s.closeErr = closeReason(err)
		close(s.done)
		s.conn.Close()
	})
}

/**
* closeReason: Motivo de cierre: MSG_WS_CLOSED en un cierre normal, o con la causa.
* @param err error
* @return error
**/
func closeReason(err error) error {
	if err == nil || errors.Is(err, net.ErrClosed) ||
		gws.IsCloseError(err, gws.CloseNormalClosure, gws.CloseGoingAway, gws.CloseNoStatusReceived) {
		return errors.New(msg.MSG_WS_CLOSED)
	}
	return fmt.Errorf("%s: %w", msg.MSG_WS_CLOSED, err)
}

/**
* finish: Quita las suscripciones (llamando a onUnsubscribe) y, si estaba conectado,
* llama a onDisconnect. Corre una vez, en la goroutine de la conexión, tras kill.
**/
func (s *Conn) finish() {
	s.subMu.Lock()
	subs := make([]Subscription, 0, len(s.subs))
	for _, sub := range s.subs {
		subs = append(subs, sub)
		s.server.removeSub(s, sub)
	}
	s.subs = nil
	s.subMu.Unlock()

	s.server.removeConn(s)

	sort.Slice(subs, func(i, j int) bool { return subs[i].Channel < subs[j].Channel })
	for _, sub := range subs {
		s.server.hs.unsubscribe(s, sub)
	}

	if s.connected.Load() {
		s.server.hs.disconnect(s, s.closeErr)
	}
}

/**
* writePump: Escribe en orden los mensajes del buffer y envía ping periódicamente.
* Es el único que escribe mensajes de datos en la conexión.
**/
func (s *Conn) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case b := <-s.out:
			s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := s.conn.WriteMessage(gws.TextMessage, b); err != nil {
				s.kill(err)
				return
			}
		case <-ticker.C:
			s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := s.conn.WriteMessage(gws.PingMessage, nil); err != nil {
				s.kill(err)
				return
			}
		case <-s.done:
			return
		}
	}
}

/**
* readPump: Lee y atiende los mensajes del cliente, uno a la vez, hasta que la conexión termina.
**/
func (s *Conn) readPump() {
	s.conn.SetReadLimit(maxMessageSize)
	s.conn.SetReadDeadline(time.Now().Add(pongWait))
	s.conn.SetPongHandler(func(string) error {
		return s.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		kind, data, err := s.conn.ReadMessage()
		if err != nil {
			s.kill(err)
			return
		}
		s.conn.SetReadDeadline(time.Now().Add(pongWait))

		if kind != gws.TextMessage {
			s.reply(reply{Type: typeError, Message: msg.MSG_WS_INVALID_MESSAGE})
			continue
		}
		s.handle(data)
	}
}

/**
* handle: Atiende un mensaje del cliente y responde ack o error.
* @param data []byte
**/
func (s *Conn) handle(data []byte) {
	var in inbound
	if err := json.Unmarshal(data, &in); err != nil {
		s.reply(reply{Type: typeError, Message: msg.MSG_WS_INVALID_MESSAGE})
		return
	}

	switch in.Type {
	case typeSubscribe:
		s.replyErr(in.ID, s.subscribe(in.Channel))
	case typeQueue:
		s.replyErr(in.ID, s.subscribeQueue(in.Channel, in.Queue))
	case typeUnsubscribe:
		s.replyErr(in.ID, s.unsubscribe(in.Channel))
	case typePublish:
		delivered, err := s.server.publishRaw(in.Channel, in.Data, s.clientID)
		if err != nil {
			s.replyErr(in.ID, err)
			return
		}
		s.reply(reply{Type: typeAck, ID: in.ID, Delivered: &delivered})
	default:
		s.reply(reply{Type: typeError, ID: in.ID, Message: fmt.Sprintf(msg.MSG_WS_UNKNOWN_TYPE, in.Type)})
	}
}
