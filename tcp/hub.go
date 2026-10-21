package tcp

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
)

// beatMaxFailures: latidos seguidos sin respuesta que cierran la conexión (§14.7).
const beatMaxFailures = 3

/**
* connState: Estado del Hub por conexión viva (aceptada por un servidor del Hub o
* abierta con ConnectTo): orden de los mensajes y métricas.
**/
type connState struct {
	srv         *Server // nil: la abrió el Hub con ConnectTo
	sendMu      sync.Mutex
	sendSeq     uint64 // último seq enviado
	recvMu      sync.Mutex
	nextSeq     uint64              // siguiente seq que se entrega
	pending     map[uint64]*Message // llegados adelantados; nil = inválido, se salta
	delivering  bool                // hay una goroutine entregando
	closed      bool
	connectedAt time.Time
	messagesIn  atomic.Int64
	messagesOut atomic.Int64
	bytesIn     atomic.Int64
	bytesOut    atomic.Int64
	timesMu     sync.Mutex
	lastMessage time.Time
	lastBeat    time.Time
}

/**
* clientInfo: Estado de un cliente registrado con ConnectTo (por dirección), que
* sobrevive a las reconexiones: lo que informa Status.
**/
type clientInfo struct {
	mu          sync.Mutex
	connectedAt time.Time
	lastBeat    time.Time
	latency     time.Duration
	failures    int
	beating     bool
}

/**
* Hub: Agrupa varios servidores y varios clientes de este paquete para llevar
* mensajes tipados entre nodos (spec §14). Los dos extremos de una conexión del Hub
* son Hubs.
**/
type Hub struct {
	mu            sync.RWMutex           // protege los mapas
	hubID         string                 // fijo desde NewHub
	hubName       string                 // nombre del Hub
	clientMap     map[string]*Client     // clave: addr tal como se pasó a ConnectTo
	infoMap       map[string]*clientInfo // clave: addr
	connecting    map[string]bool        // addr con un ConnectTo en curso
	serverMap     map[string]*Server     // clave: puerto en decimal
	conns         map[*Client]*connState
	timeoutServer *duration // el mismo puntero en todos los servidores del Hub
	timeoutClient *duration // el mismo puntero en todos los clientes del Hub
	sec           *security // el mismo puntero en todos los servidores y clientes del Hub
	fnMu          sync.RWMutex
	messageFns    []func(srv *Server, conn *Client, msg *Message)
	connectFns    []func(srv *Server, conn *Client)
	disconnectFns []func(srv *Server, conn *Client, err error)
	beatMu        sync.Mutex
	beatStop      chan struct{} // nil: heartbeat apagado
}

/**
* newHub: Crea el Hub; no escucha, no conecta ni arranca goroutines. Con id vacío se
* genera un UUID.
* @param id, name string
* @return *Hub
**/
func newHub(id, name string) *Hub {
	return &Hub{
		hubID:         reg.GetUUID(id),
		hubName:       name,
		clientMap:     make(map[string]*Client),
		infoMap:       make(map[string]*clientInfo),
		connecting:    make(map[string]bool),
		serverMap:     make(map[string]*Server),
		conns:         make(map[*Client]*connState),
		timeoutServer: newDuration(defaultTimeout),
		timeoutClient: newDuration(defaultTimeout),
		sec:           &security{},
	}
}

/**
* id: Identificador del Hub
* @return string
**/
func (h *Hub) id() string {
	return h.hubID
}

/**
* name: Nombre del Hub
* @return string
**/
func (h *Hub) name() string {
	return h.hubName
}

/**
* from: El From que el Hub pone en lo que envía
* @return et.Json
**/
func (h *Hub) from() et.Json {
	return et.Json{"id": h.hubID, "name": h.hubName}
}

// ---- Eventos ----

/**
* onMessage: Agrega fn a las funciones que reciben cada Message
* @param fn func(srv *Server, conn *Client, msg *Message)
**/
func (h *Hub) onMessage(fn func(srv *Server, conn *Client, msg *Message)) {
	h.fnMu.Lock()
	defer h.fnMu.Unlock()
	h.messageFns = append(h.messageFns, fn)
}

/**
* onConnect: Agrega fn a las funciones que se llaman al abrir una conexión
* @param fn func(srv *Server, conn *Client)
**/
func (h *Hub) onConnect(fn func(srv *Server, conn *Client)) {
	h.fnMu.Lock()
	defer h.fnMu.Unlock()
	h.connectFns = append(h.connectFns, fn)
}

/**
* onDisconnect: Agrega fn a las funciones que se llaman al cerrarse una conexión
* @param fn func(srv *Server, conn *Client, err error)
**/
func (h *Hub) onDisconnect(fn func(srv *Server, conn *Client, err error)) {
	h.fnMu.Lock()
	defer h.fnMu.Unlock()
	h.disconnectFns = append(h.disconnectFns, fn)
}

/**
* fireMessage: Llama a las funciones de onMessage; un panic se registra y se sigue
* @param srv *Server, conn *Client, m *Message
**/
func (h *Hub) fireMessage(srv *Server, conn *Client, m *Message) {
	h.fnMu.RLock()
	fns := h.messageFns
	h.fnMu.RUnlock()
	for _, fn := range fns {
		safeCall(func() { fn(srv, conn, m) })
	}
}

/**
* fireConnect: Llama a las funciones de onConnect
* @param srv *Server, conn *Client
**/
func (h *Hub) fireConnect(srv *Server, conn *Client) {
	h.fnMu.RLock()
	fns := h.connectFns
	h.fnMu.RUnlock()
	for _, fn := range fns {
		safeCall(func() { fn(srv, conn) })
	}
}

/**
* fireDisconnect: Llama a las funciones de onDisconnect
* @param srv *Server, conn *Client, err error
**/
func (h *Hub) fireDisconnect(srv *Server, conn *Client, err error) {
	h.fnMu.RLock()
	fns := h.disconnectFns
	h.fnMu.RUnlock()
	for _, fn := range fns {
		safeCall(func() { fn(srv, conn, err) })
	}
}

// ---- Estado por conexión ----

/**
* state: El estado de conn; lo crea si conn sigue conectada (un mensaje puede llegar
* antes que onConnect). nil si la conexión ya se cerró.
* @param srv *Server, conn *Client
* @return *connState
**/
func (h *Hub) state(srv *Server, conn *Client) *connState {
	h.mu.RLock()
	st := h.conns[conn]
	h.mu.RUnlock()
	if st != nil {
		return st
	}

	if _, err := conn.current(); err != nil {
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if st = h.conns[conn]; st == nil {
		st = &connState{srv: srv, nextSeq: 1, pending: make(map[uint64]*Message), connectedAt: time.Now()}
		h.conns[conn] = st
	}
	return st
}

/**
* removeState: Saca la conexión del Hub (y de las métricas) y descarta lo pendiente
* @param conn *Client
**/
func (h *Hub) removeState(conn *Client) {
	h.mu.Lock()
	st := h.conns[conn]
	delete(h.conns, conn)
	h.mu.Unlock()
	if st == nil {
		return
	}

	st.recvMu.Lock()
	st.closed = true
	st.pending = make(map[uint64]*Message)
	st.recvMu.Unlock()
}

/**
* receive: Atiende un msgSend: lo decodifica y lo entrega a onMessage en orden de seq,
* de a uno por conexión (§14.3).
* @param srv *Server, conn *Client, data []byte
**/
func (h *Hub) receive(srv *Server, conn *Client, data []byte) {
	st := h.state(srv, conn)
	if st == nil {
		return
	}

	w, ok := decodeWire(data)
	if !ok || w.Seq == 0 {
		logs.Alertm(msg.MSG_HUB_INVALID_MESSAGE)
		conn.closeWith(errInvalidMessage())
		return
	}

	m, err := toMessage(w)
	if err != nil || m.Type == TypeBeat {
		logs.Alertf("%s: seq %d", msg.MSG_HUB_INVALID_MESSAGE, w.Seq)
		m = nil
	}

	st.messagesIn.Add(1)
	st.bytesIn.Add(int64(len(data)))
	st.timesMu.Lock()
	st.lastMessage = time.Now()
	st.timesMu.Unlock()

	st.recvMu.Lock()
	if _, dup := st.pending[w.Seq]; st.closed || w.Seq < st.nextSeq || dup {
		st.recvMu.Unlock()
		return
	}
	st.pending[w.Seq] = m
	if st.delivering {
		st.recvMu.Unlock()
		return
	}

	st.delivering = true
	for {
		next, ok := st.pending[st.nextSeq]
		if !ok || st.closed {
			break
		}
		delete(st.pending, st.nextSeq)
		st.nextSeq++
		st.recvMu.Unlock()
		if next != nil {
			h.fireMessage(srv, conn, next)
		}
		st.recvMu.Lock()
	}
	st.delivering = false
	st.recvMu.Unlock()
}

/**
* answerBeat: Responde los latidos (con o sin heartbeat encendido); cualquier otro
* request se cede a las demás funciones de onRequest.
* @param srv *Server, conn *Client, data []byte
* @return []byte, error
**/
func (h *Hub) answerBeat(srv *Server, conn *Client, data []byte) ([]byte, error) {
	w, ok := decodeWire(data)
	if !ok || !isBeat(w) {
		return nil, nil
	}

	if st := h.state(srv, conn); st != nil {
		st.timesMu.Lock()
		st.lastBeat = time.Now()
		st.timesMu.Unlock()
	}
	return encodeBeat(h.from()), nil
}

/**
* sendOn: Envía m por conn con el siguiente seq; asignar el seq y escribir ocurren
* bajo el mismo lock. Si la escritura falla, cierra la conexión: un seq a medio
* escribir dejaría al receptor esperando.
* @param conn *Client, st *connState, m *Message
* @return error
**/
func (h *Hub) sendOn(conn *Client, st *connState, m *Message) error {
	st.sendMu.Lock()
	defer st.sendMu.Unlock()

	data, err := encodeMessage(m, st.sendSeq+1, h.from())
	if err != nil {
		return err
	}

	if err := conn.send(data); err != nil {
		// Sin conexión o mensaje demasiado grande: no se escribió nada
		if err.Error() != msg.MSG_TCP_NOT_CONNECTED && err.Error() != msg.MSG_DATA_TOO_LARGE {
			conn.closeWith(err)
		}
		return err
	}

	st.sendSeq++
	st.messagesOut.Add(1)
	st.bytesOut.Add(int64(len(data)))
	return nil
}

// ---- Servidores ----

/**
* loadServer: Crea un servidor con la identidad y el timeout del Hub, escucha en
* :port y lo registra. port 0: el sistema elige y la clave es el puerto real.
* @param port int
* @return *Server, error
**/
func (h *Hub) loadServer(port int) (*Server, error) {
	key := strconv.Itoa(port)
	h.mu.RLock()
	_, exists := h.serverMap[key]
	h.mu.RUnlock()
	if port != 0 && exists {
		return nil, fmt.Errorf(msg.MSG_HUB_SERVER_EXISTS, key)
	}

	srv, err := newServer(h.hubID, h.hubName)
	if err != nil {
		return nil, err
	}
	srv.timeout = h.timeoutServer
	srv.sec = h.sec
	srv.onConnect(func(conn *Client) {
		h.state(srv, conn)
		h.fireConnect(srv, conn)
	})
	srv.onDisconnect(func(conn *Client, err error) {
		h.fireDisconnect(srv, conn, err)
		h.removeState(conn)
	})
	srv.onSend(func(conn *Client, data []byte) { h.receive(srv, conn, data) })
	srv.onRequest(func(conn *Client, data []byte) ([]byte, error) { return h.answerBeat(srv, conn, data) })

	if err := srv.listen(":" + key); err != nil {
		return nil, err
	}

	if port == 0 {
		if _, p, err := net.SplitHostPort(srv.addr()); err == nil {
			key = p
		}
	}

	h.mu.Lock()
	if _, exists := h.serverMap[key]; exists {
		h.mu.Unlock()
		srv.close()
		return nil, fmt.Errorf(msg.MSG_HUB_SERVER_EXISTS, key)
	}
	h.serverMap[key] = srv
	h.mu.Unlock()
	return srv, nil
}

/**
* getServer: El servidor cargado en port
* @param port int
* @return *Server, error
**/
func (h *Hub) getServer(port int) (*Server, error) {
	key := strconv.Itoa(port)
	h.mu.RLock()
	srv, exists := h.serverMap[key]
	h.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf(msg.MSG_HUB_SERVER_NOT_FOUND, key)
	}
	return srv, nil
}

/**
* downServer: Cierra el servidor (espera sus OnDisconnect) y lo quita del Hub
* @param port int
* @return error
**/
func (h *Hub) downServer(port int) error {
	srv, err := h.getServer(port)
	if err != nil {
		return err
	}

	err = srv.close()
	key := strconv.Itoa(port)
	h.mu.Lock()
	if h.serverMap[key] == srv {
		delete(h.serverMap, key)
	}
	h.mu.Unlock()
	return err
}

/**
* servers: Los servidores cargados
* @return []*Server
**/
func (h *Hub) servers() []*Server {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]*Server, 0, len(h.serverMap))
	for _, srv := range h.serverMap {
		result = append(result, srv)
	}
	return result
}

/**
* timeOrNil: t en RFC3339Nano, o nil si es cero
* @param t time.Time
* @return any
**/
func timeOrNil(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Format(time.RFC3339Nano)
}

/**
* metrics: Clientes conectados al servidor de port y sus métricas (§14.8)
* @param port int
* @return et.Json, error
**/
func (h *Hub) metrics(port int) (et.Json, error) {
	srv, err := h.getServer(port)
	if err != nil {
		return nil, err
	}

	h.mu.RLock()
	clients := []et.Json{}
	for conn, st := range h.conns {
		if st.srv != srv {
			continue
		}
		st.timesMu.Lock()
		lastMessage, lastBeat := st.lastMessage, st.lastBeat
		st.timesMu.Unlock()
		clients = append(clients, et.Json{
			"id":           conn.id(),
			"name":         conn.name(),
			"addr":         conn.to(),
			"connected_at": timeOrNil(st.connectedAt),
			"messages_in":  st.messagesIn.Load(),
			"messages_out": st.messagesOut.Load(),
			"bytes_in":     st.bytesIn.Load(),
			"bytes_out":    st.bytesOut.Load(),
			"last_message": timeOrNil(lastMessage),
			"last_beat":    timeOrNil(lastBeat),
		})
	}
	h.mu.RUnlock()

	return et.Json{
		"port":    port,
		"addr":    srv.addr(),
		"count":   len(clients),
		"clients": clients,
	}, nil
}

// ---- Clientes ----

/**
* newHubClient: Un cliente con la identidad, el timeout y las funciones del Hub
* @param info *clientInfo
* @return *Client
**/
func (h *Hub) newHubClient(info *clientInfo) *Client {
	c, _ := newClient(h.hubID, h.hubName)
	c.timeout = h.timeoutClient
	c.sec = h.sec
	c.onConnect(func(conn *Client) {
		info.mu.Lock()
		info.connectedAt = time.Now()
		info.failures = 0
		info.mu.Unlock()
		h.state(nil, conn)
		h.fireConnect(nil, conn)
	})
	c.onDisconnect(func(conn *Client, err error) {
		h.fireDisconnect(nil, conn, err)
		h.removeState(conn)
	})
	c.onSend(func(conn *Client, data []byte) { h.receive(nil, conn, data) })
	c.onRequest(func(conn *Client, data []byte) ([]byte, error) { return h.answerBeat(nil, conn, data) })
	return c
}

/**
* connectTo: Abre (o reabre, si quedó caído) el cliente del Hub hacia addr. Queda
* registrado antes de conectar, así OnConnect ya puede usar Send(addr); si un addr
* nuevo no conecta, se quita.
* @param addr string
* @return *Client, error
**/
func (h *Hub) connectTo(addr string) (*Client, error) {
	h.mu.Lock()
	c, exists := h.clientMap[addr]
	if h.connecting[addr] {
		h.mu.Unlock()
		return nil, errors.New(msg.MSG_TCP_ALREADY_CONNECTED)
	}
	if exists {
		if _, err := c.current(); err == nil {
			h.mu.Unlock()
			return nil, errors.New(msg.MSG_TCP_ALREADY_CONNECTED)
		}
	} else {
		info := &clientInfo{}
		c = h.newHubClient(info)
		h.clientMap[addr] = c
		h.infoMap[addr] = info
	}
	h.connecting[addr] = true
	h.mu.Unlock()

	err := c.connectTo(addr)

	h.mu.Lock()
	delete(h.connecting, addr)
	if err != nil && !exists && h.clientMap[addr] == c {
		delete(h.clientMap, addr)
		delete(h.infoMap, addr)
	}
	h.mu.Unlock()

	if err != nil {
		return nil, err
	}
	return c, nil
}

/**
* disconnect: Cierra el cliente de addr y lo quita del Hub
* @param addr string
* @return error
**/
func (h *Hub) disconnect(addr string) error {
	h.mu.Lock()
	c, exists := h.clientMap[addr]
	delete(h.clientMap, addr)
	delete(h.infoMap, addr)
	h.mu.Unlock()
	if !exists {
		return fmt.Errorf(msg.MSG_HUB_CLIENT_NOT_FOUND, addr)
	}
	return ignoreClosed(c.close())
}

/**
* ignoreClosed: nil si err solo dice que la conexión ya estaba cerrada (el otro lado
* la cerró mientras se cerraba esta): para el Hub no es un error.
* @param err error
* @return error
**/
func ignoreClosed(err error) error {
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

/**
* clients: Los clientes registrados, conectados o no
* @return []*Client
**/
func (h *Hub) clients() []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]*Client, 0, len(h.clientMap))
	for _, c := range h.clientMap {
		result = append(result, c)
	}
	return result
}

// ---- Envío ----

/**
* send: Envía msg por el cliente registrado con addr
* @param addr string, m *Message
* @return error
**/
func (h *Hub) send(addr string, m *Message) error {
	h.mu.RLock()
	c, exists := h.clientMap[addr]
	h.mu.RUnlock()
	if !exists {
		return fmt.Errorf(msg.MSG_HUB_CLIENT_NOT_FOUND, addr)
	}

	if _, err := encodeMessage(m, 1, nil); err != nil {
		return err
	}

	st := h.state(nil, c)
	if st == nil {
		return errors.New(msg.MSG_TCP_NOT_CONNECTED)
	}
	return h.sendOn(c, st, m)
}

/**
* sendTo: Envía msg a cada conexión del servidor de port cuyo ID es clientID
* @param port int, clientID string, m *Message
* @return error (el primero)
**/
func (h *Hub) sendTo(port int, clientID string, m *Message) error {
	srv, err := h.getServer(port)
	if err != nil {
		return err
	}

	if _, err := encodeMessage(m, 1, nil); err != nil {
		return err
	}

	type target struct {
		conn *Client
		st   *connState
	}
	h.mu.RLock()
	targets := []target{}
	for conn, st := range h.conns {
		if st.srv == srv && conn.id() == clientID {
			targets = append(targets, target{conn, st})
		}
	}
	h.mu.RUnlock()

	if len(targets) == 0 {
		return fmt.Errorf(msg.MSG_HUB_CLIENT_NOT_FOUND, clientID)
	}

	var result error
	for _, t := range targets {
		if err := h.sendOn(t.conn, t.st, m); err != nil && result == nil {
			result = err
		}
	}
	return result
}

// ---- Heartbeat ----

/**
* heartbeat: Arranca o cambia el latido de todos los clientes del Hub; interval <= 0
* lo detiene.
* @param interval time.Duration
**/
func (h *Hub) heartbeat(interval time.Duration) {
	h.beatMu.Lock()
	defer h.beatMu.Unlock()

	if h.beatStop != nil {
		close(h.beatStop)
		h.beatStop = nil
	}
	if interval <= 0 {
		return
	}

	stop := make(chan struct{})
	h.beatStop = stop
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				h.beatAll()
			}
		}
	}()
}

/**
* beatAll: Un latido por cada cliente conectado que no tenga uno en curso
**/
func (h *Hub) beatAll() {
	type target struct {
		c    *Client
		info *clientInfo
	}
	h.mu.RLock()
	targets := make([]target, 0, len(h.clientMap))
	for addr, c := range h.clientMap {
		targets = append(targets, target{c, h.infoMap[addr]})
	}
	h.mu.RUnlock()

	for _, t := range targets {
		if t.info == nil {
			continue
		}
		if _, err := t.c.current(); err != nil {
			continue
		}
		t.info.mu.Lock()
		if t.info.beating {
			t.info.mu.Unlock()
			continue
		}
		t.info.beating = true
		t.info.mu.Unlock()
		go h.beat(t.c, t.info)
	}
}

/**
* beat: Envía un latido y espera la respuesta hasta timeoutClient; con
* beatMaxFailures fallas seguidas cierra la conexión con MSG_HUB_HEARTBEAT.
* @param c *Client, info *clientInfo
**/
func (h *Hub) beat(c *Client, info *clientInfo) {
	start := time.Now()
	data, err := c.request(encodeBeat(h.from()))
	if err == nil {
		if w, ok := decodeWire(data); !ok || !isBeat(w) {
			err = errInvalidMessage()
		}
	}

	info.mu.Lock()
	info.beating = false
	if err == nil {
		info.failures = 0
		info.lastBeat = time.Now()
		info.latency = time.Since(start)
		info.mu.Unlock()
		return
	}
	info.failures++
	failures := info.failures
	info.mu.Unlock()

	if failures >= beatMaxFailures {
		c.closeWith(errors.New(msg.MSG_HUB_HEARTBEAT))
	}
}

/**
* status: Estado de cada cliente registrado (§14.7)
* @return []et.Json
**/
func (h *Hub) status() []et.Json {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]et.Json, 0, len(h.clientMap))
	for addr, c := range h.clientMap {
		state := "up"
		if _, err := c.current(); err != nil {
			state = "down"
		}
		item := et.Json{
			"addr":   addr,
			"server": et.Json{"id": c.serverID(), "name": c.serverName()},
			"state":  state,
		}
		if info := h.infoMap[addr]; info != nil {
			info.mu.Lock()
			item["connected_at"] = timeOrNil(info.connectedAt)
			item["last_beat"] = timeOrNil(info.lastBeat)
			item["latency_ms"] = float64(info.latency.Microseconds()) / 1000
			item["failures"] = info.failures
			info.mu.Unlock()
		}
		result = append(result, item)
	}
	return result
}

// ---- Timeouts, JSON y cierre ----

/**
* setTimeoutServer: Cambia el timeout de todos los servidores del Hub (y de los que se creen)
* @param d time.Duration
* @return error
**/
func (h *Hub) setTimeoutServer(d time.Duration) error {
	return h.timeoutServer.set(d)
}

/**
* setTimeoutClient: Cambia el timeout de todos los clientes del Hub (y de los que se creen)
* @param d time.Duration
* @return error
**/
func (h *Hub) setTimeoutClient(d time.Duration) error {
	return h.timeoutClient.set(d)
}

/**
* toJson: {"id","name","servers":[Metrics de cada uno],"clients":Status()}
* @return et.Json
**/
func (h *Hub) toJson() et.Json {
	servers := []et.Json{}
	for _, key := range h.serverKeys() {
		port, _ := strconv.Atoi(key)
		if metrics, err := h.metrics(port); err == nil {
			servers = append(servers, metrics)
		}
	}

	return et.Json{
		"id":      h.hubID,
		"name":    h.hubName,
		"servers": servers,
		"clients": h.status(),
	}
}

/**
* serverKeys: Los puertos cargados
* @return []string
**/
func (h *Hub) serverKeys() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]string, 0, len(h.serverMap))
	for key := range h.serverMap {
		result = append(result, key)
	}
	return result
}

/**
* close: Detiene el heartbeat, cierra todos los clientes y da de baja todos los
* servidores. El Hub se puede volver a usar.
* @return error (el primero)
**/
func (h *Hub) close() error {
	h.heartbeat(0)

	h.mu.Lock()
	clients := h.clientMap
	servers := h.serverMap
	h.clientMap = make(map[string]*Client)
	h.infoMap = make(map[string]*clientInfo)
	h.serverMap = make(map[string]*Server)
	h.mu.Unlock()

	var result error
	for _, c := range clients {
		if err := ignoreClosed(c.close()); err != nil && result == nil {
			result = err
		}
	}
	for _, srv := range servers {
		if err := ignoreClosed(srv.close()); err != nil && result == nil {
			result = err
		}
	}
	return result
}
