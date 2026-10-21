package websocket

import (
	"encoding/json"
	"net/http"
)

// ---- Servidor ----

/**
* NewServer: Crea un servidor WebSocket que todavía no escucha.
* @return *Server, error
**/
func NewServer() (*Server, error) {
	return newServer()
}

/**
* ServeHTTP: Atiende una conexión WebSocket; bloquea hasta que termina. Permite montar
* el servidor en cualquier router como http.Handler.
* @param w http.ResponseWriter, r *http.Request
**/
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.serveHTTP(w, r)
}

/**
* Listen: Abre un servidor HTTP propio en addr que atiende todas las rutas como WebSocket.
* @param addr string
* @return error
**/
func (s *Server) Listen(addr string) error {
	return s.listen(addr)
}

/**
* Addr: Retorna la dirección de Listen, o "" si no escucha.
* @return string
**/
func (s *Server) Addr() string {
	return s.addr()
}

/**
* Close: Deja de aceptar, cierra todos los clientes y espera a que terminen sus eventos.
* Es definitivo.
* @return error
**/
func (s *Server) Close() error {
	return s.close()
}

/**
* Count: Retorna cuántos clientes están conectados.
* @return int
**/
func (s *Server) Count() int {
	return s.count()
}

/**
* Publish: Publica data (codificada en JSON) en el canal: la reciben todos los
* suscriptores topic y un miembro de cada grupo queue. Retorna cuántos la recibieron.
* @param channel string, data any
* @return int, error
**/
func (s *Server) Publish(channel string, data any) (int, error) {
	return s.publish(channel, data)
}

// ---- Eventos ----

/**
* OnConnect: Agrega fn a las funciones que se llaman al conectarse un cliente. Si fn
* retorna error, la conexión se rechaza con un cierre 1008 y ese motivo.
* @param fn func(c *Conn) error
**/
func (s *Server) OnConnect(fn func(c *Conn) error) {
	s.onConnect(fn)
}

/**
* OnDisconnect: Agrega fn a las funciones que se llaman al desconectarse un cliente
* aceptado; err es el motivo.
* @param fn func(c *Conn, err error)
**/
func (s *Server) OnDisconnect(fn func(c *Conn, err error)) {
	s.onDisconnect(fn)
}

/**
* OnSubscribe: Agrega fn a las funciones que se llaman antes de cada suscripción (topic
* o queue). Si fn retorna error, la suscripción no se hace.
* @param fn func(c *Conn, sub Subscription) error
**/
func (s *Server) OnSubscribe(fn func(c *Conn, sub Subscription) error) {
	s.onSubscribe(fn)
}

/**
* OnUnsubscribe: Agrega fn a las funciones que se llaman después de cada desuscripción,
* también las que ocurren al desconectarse.
* @param fn func(c *Conn, sub Subscription)
**/
func (s *Server) OnUnsubscribe(fn func(c *Conn, sub Subscription)) {
	s.onUnsubscribe(fn)
}

// ---- Conexión (un cliente visto desde el servidor) ----

/**
* ID: Retorna el id del cliente (ULID).
* @return string
**/
func (s *Conn) ID() string {
	return s.id()
}

/**
* Request: Retorna la solicitud HTTP con la que se conectó (encabezados, query, etc.).
* @return *http.Request
**/
func (s *Conn) Request() *http.Request {
	return s.request()
}

/**
* Send: Envía data (codificada en JSON) solo a este cliente, como mensaje tipo direct.
* @param data any
* @return error
**/
func (s *Conn) Send(data any) error {
	return s.send(data)
}

/**
* Subscribe: Suscribe al cliente al canal, tipo topic (recibe todos los mensajes).
* @param channel string
* @return error
**/
func (s *Conn) Subscribe(channel string) error {
	return s.subscribe(channel)
}

/**
* SubscribeQueue: Suscribe al cliente al canal, tipo queue, en el grupo queue (el grupo
* recibe cada mensaje una vez, por turnos).
* @param channel, queue string
* @return error
**/
func (s *Conn) SubscribeQueue(channel, queue string) error {
	return s.subscribeQueue(channel, queue)
}

/**
* Unsubscribe: Quita la suscripción del cliente al canal.
* @param channel string
* @return error
**/
func (s *Conn) Unsubscribe(channel string) error {
	return s.unsubscribe(channel)
}

/**
* Subscriptions: Retorna las suscripciones del cliente, ordenadas por canal.
* @return []Subscription
**/
func (s *Conn) Subscriptions() []Subscription {
	return s.subscriptions()
}

/**
* Close: Cierra la conexión del cliente con un cierre normal.
* @return error
**/
func (s *Conn) Close() error {
	return s.close()
}

// ---- Cliente ----

/**
* NewClient: Crea un cliente sin conectar.
* @return *Client, error
**/
func NewClient() (*Client, error) {
	return newClient()
}

/**
* Connect: Se conecta a url (ws:// o wss://) y espera a que el servidor acepte la
* conexión; si su OnConnect la rechaza, retorna MSG_WS_REJECTED con el motivo.
* @param url string
* @return error
**/
func (s *Client) Connect(url string) error {
	return s.connect(url)
}

/**
* Close: Cierra la conexión con un cierre normal.
* @return error
**/
func (s *Client) Close() error {
	return s.close()
}

/**
* ID: Retorna el id que el servidor asignó en la última conexión aceptada.
* @return string
**/
func (s *Client) ID() string {
	return s.id()
}

/**
* Subscribe: Se suscribe al canal, tipo topic; espera la confirmación del servidor.
* @param channel string
* @return error
**/
func (s *Client) Subscribe(channel string) error {
	return s.subscribe(channel)
}

/**
* SubscribeQueue: Se suscribe al canal, tipo queue, en el grupo queue; espera la confirmación.
* @param channel, queue string
* @return error
**/
func (s *Client) SubscribeQueue(channel, queue string) error {
	return s.subscribeQueue(channel, queue)
}

/**
* Unsubscribe: Quita la suscripción al canal; espera la confirmación.
* @param channel string
* @return error
**/
func (s *Client) Unsubscribe(channel string) error {
	return s.unsubscribe(channel)
}

/**
* Publish: Publica data (codificada en JSON) en el canal; retorna cuántos la recibieron.
* @param channel string, data any
* @return int, error
**/
func (s *Client) Publish(channel string, data any) (int, error) {
	return s.publish(channel, data)
}

/**
* Subscriptions: Retorna las suscripciones confirmadas por el servidor, ordenadas por canal.
* @return []Subscription
**/
func (s *Client) Subscriptions() []Subscription {
	return s.subscriptions()
}

// ---- Eventos del cliente ----

/**
* OnConnect: Agrega fn a las funciones que se llaman cuando el servidor acepta la
* conexión, antes de que Connect retorne.
* @param fn func(c *Client)
**/
func (s *Client) OnConnect(fn func(c *Client)) {
	s.onConnect(fn)
}

/**
* OnDisconnect: Agrega fn a las funciones que se llaman cuando se cierra una conexión
* aceptada, la cierre quien la cierre; desde aquí se puede volver a llamar a Connect.
* @param fn func(c *Client, err error)
**/
func (s *Client) OnDisconnect(fn func(c *Client, err error)) {
	s.onDisconnect(fn)
}

/**
* OnMessage: Agrega fn a las funciones que reciben, en orden, los mensajes de los canales suscritos.
* @param fn func(c *Client, m Message)
**/
func (s *Client) OnMessage(fn func(c *Client, m Message)) {
	s.onMessage(fn)
}

/**
* OnDirect: Agrega fn a las funciones que reciben los mensajes directos del servidor (Conn.Send).
* @param fn func(c *Client, data json.RawMessage)
**/
func (s *Client) OnDirect(fn func(c *Client, data json.RawMessage)) {
	s.onDirect(fn)
}
