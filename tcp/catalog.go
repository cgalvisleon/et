package tcp

import (
	"crypto/tls"
	"time"

	"github.com/cgalvisleon/et/et"
)

// ---- Cliente: conexión ----

/**
* NewClient: Crea un cliente sin conectar. Con id vacío se genera un UUID.
* @param id,name string
* @return *Client, error
**/
func NewClient(id, name string) (*Client, error) {
	return newClient(id, name)
}

/**
* ToJson: Retorna el cliente en formato JSON: id, name, to, server (id y name), tls y auth.
* @return et.Json
**/
func (s *Client) ToJson() et.Json {
	return s.toJson()
}

/**
* ID: Retorna el identificador del cliente. En una conexión del servidor es el que
* el cliente envió al saludar.
* @return string
**/
func (s *Client) ID() string {
	return s.id()
}

/**
* Name: Retorna el nombre del cliente. En una conexión del servidor es el que el
* cliente envió al saludar.
* @return string
**/
func (s *Client) Name() string {
	return s.name()
}

/**
* To: Retorna la dirección del otro extremo de la última conexión.
* @return string
**/
func (s *Client) To() string {
	return s.to()
}

/**
* ServerID: Retorna el identificador del servidor de la última conexión.
* @return string
**/
func (s *Client) ServerID() string {
	return s.serverID()
}

/**
* ServerName: Retorna el nombre del servidor de la última conexión.
* @return string
**/
func (s *Client) ServerName() string {
	return s.serverName()
}

/**
* ConnectTo: Abre la conexión con addr y empieza a leer mensajes en segundo plano.
* @param addr string
* @return error
**/
func (s *Client) ConnectTo(addr string) error {
	return s.connectTo(addr)
}

/**
* Close: Cierra la conexión; los Request en espera retornan MSG_TCP_CLOSED.
* @return error
**/
func (s *Client) Close() error {
	return s.close()
}

/**
* SetTimeout: Cambia el timeout de conexión, escritura y Request (por defecto 10s); aplica
* también a la conexión abierta.
* @param timeout time.Duration
* @return error
**/
func (s *Client) SetTimeout(timeout time.Duration) error {
	return s.setTimeout(timeout)
}

/**
* SetSecurity: Cambia la seguridad (TLS y secreto del saludo, spec §7.4) de las
* conexiones que se abran después; la conexión abierta sigue como estaba.
* @param cfg Security
**/
func (s *Client) SetSecurity(cfg Security) {
	s.setSecurity(cfg)
}

// ---- Cliente: envío ----

/**
* Send: Envía message sin esperar respuesta.
* @param message []byte
* @return error
**/
func (s *Client) Send(message []byte) error {
	return s.send(message)
}

/**
* Request: Envía message y espera su respuesta, hasta el timeout. Admite llamadas concurrentes.
* @param message []byte
* @return []byte, error
**/
func (s *Client) Request(message []byte) ([]byte, error) {
	return s.request(message)
}

// ---- Cliente: recepción ----

/**
* OnConnect: Agrega fn a las funciones que se llaman cada vez que ConnectTo abre la
* conexión, antes de que ConnectTo retorne.
* @param fn func(conn *Client)
**/
func (s *Client) OnConnect(fn func(conn *Client)) {
	s.onConnect(fn)
}

/**
* OnDisconnect: Agrega fn a las funciones que se llaman cuando la conexión se cierra, la
* cierre quien la cierre; err es el motivo.
* @param fn func(conn *Client, err error)
**/
func (s *Client) OnDisconnect(fn func(conn *Client, err error)) {
	s.onDisconnect(fn)
}

/**
* OnSend: Agrega fn a las funciones que reciben los mensajes tipo send, en orden; conn
* es la conexión por la que llegó.
* @param fn func(conn *Client, message []byte)
**/
func (s *Client) OnSend(fn func(conn *Client, message []byte)) {
	s.onSend(fn)
}

/**
* OnRequest: Agrega fn a las funciones que atienden los mensajes tipo request. Responde
* la primera que retorne una respuesta distinta de nil o un error; (nil, nil) cede a la siguiente.
* @param fn func(conn *Client, message []byte) ([]byte, error)
**/
func (s *Client) OnRequest(fn func(conn *Client, message []byte) ([]byte, error)) {
	s.onRequest(fn)
}

// ---- Cliente: streams ----

/**
* SendStream: Abre un stream para enviar datos por partes, sin esperar respuesta. Se
* escribe con Write y se termina con Close; el otro lado lo recibe en OnSendStream.
* @return *StreamWriter, error
**/
func (s *Client) SendStream() (*StreamWriter, error) {
	return s.sendStream()
}

/**
* RequestStream: Abre un stream para enviar datos por partes y esperar una respuesta.
* Se escribe con Write y se termina con CloseAndReceive, que retorna la respuesta; el
* otro lado lo atiende en OnRequestStream.
* @return *StreamWriter, error
**/
func (s *Client) RequestStream() (*StreamWriter, error) {
	return s.requestStream()
}

/**
* ReceiveStream: Envía message y recibe la respuesta por partes con Read, hasta io.EOF;
* el otro lado la escribe en OnReceiveStream.
* @param message []byte
* @return *StreamReader, error
**/
func (s *Client) ReceiveStream(message []byte) (*StreamReader, error) {
	return s.receiveStream(message)
}

/**
* OnSendStream: Define la función que recibe los SendStream; reemplaza a la anterior
* porque un stream solo se puede leer una vez. Al retornar, lo que no leyó se cancela.
* @param fn func(conn *Client, in *StreamReader)
**/
func (s *Client) OnSendStream(fn func(conn *Client, in *StreamReader)) {
	s.onSendStream(fn)
}

/**
* OnRequestStream: Define la función que atiende los RequestStream; lo que retorna es
* la respuesta. Reemplaza a la anterior.
* @param fn func(conn *Client, in *StreamReader) ([]byte, error)
**/
func (s *Client) OnRequestStream(fn func(conn *Client, in *StreamReader) ([]byte, error)) {
	s.onRequestStream(fn)
}

/**
* OnReceiveStream: Define la función que atiende los ReceiveStream: escribe la respuesta
* por partes en out. Si retorna nil el stream termina bien; si retorna un error, el
* remitente lo recibe al leer. Reemplaza a la anterior.
* @param fn func(conn *Client, message []byte, out *StreamWriter) error
**/
func (s *Client) OnReceiveStream(fn func(conn *Client, message []byte, out *StreamWriter) error) {
	s.onReceiveStream(fn)
}

// ---- Servidor ----

/**
* NewServer: Crea un servidor que todavía no escucha. Con id vacío se genera un UUID;
* el id no cambia aunque se vuelva a llamar a Listen.
* @param id, name string
* @return *Server, error
**/
func NewServer(id, name string) (*Server, error) {
	return newServer(id, name)
}

/**
* ID: Retorna el identificador del servidor.
* @return string
**/
func (s *Server) ID() string {
	return s.id()
}

/**
* Name: Retorna el nombre del servidor.
* @return string
**/
func (s *Server) Name() string {
	return s.name()
}

/**
* ToJson: Retorna el servidor en formato JSON: id, name, addr y peers.
* @return et.Json
**/
func (s *Server) ToJson() et.Json {
	return s.toJson()
}

/**
* SetName: Cambia el nombre del servidor.
* @param name string
**/
func (s *Server) SetName(name string) {
	s.setName(name)
}

/**
* Listen: Empieza a escuchar en addr y acepta conexiones en segundo plano.
* @param addr string
* @return error
**/
func (s *Server) Listen(addr string) error {
	return s.listen(addr)
}

/**
* Addr: Retorna la dirección en la que escucha, o "" si no está escuchando.
* @return string
**/
func (s *Server) Addr() string {
	return s.addr()
}

/**
* Close: Deja de escuchar, cierra todas las conexiones y espera a que terminen.
* @return error
**/
func (s *Server) Close() error {
	return s.close()
}

/**
* Count: Retorna el número de conexiones abiertas.
* @return int
**/
func (s *Server) Count() int {
	return s.count()
}

/**
* SetTimeout: Cambia el timeout de escritura y Request (por defecto 10s); aplica también
* a las conexiones abiertas.
* @param timeout time.Duration
* @return error
**/
func (s *Server) SetTimeout(timeout time.Duration) error {
	return s.setTimeout(timeout)
}

/**
* SetSecurity: Cambia la seguridad (TLS y secreto del saludo, spec §7.4) de las
* conexiones que se acepten después; las abiertas siguen como estaban.
* @param cfg Security
**/
func (s *Server) SetSecurity(cfg Security) {
	s.setSecurity(cfg)
}

/**
* OnConnect: Agrega fn a las funciones que se llaman con cada conexión aceptada. Con
* conn el servidor envía a ese cliente (conn.Send, conn.Request) mientras sigue recibiendo.
* @param fn func(conn *Client)
**/
func (s *Server) OnConnect(fn func(conn *Client)) {
	s.onConnect(fn)
}

/**
* OnDisconnect: Agrega fn a las funciones que se llaman cuando se cierra cualquier
* conexión, la cierre el cliente o el servidor; err es el motivo.
* @param fn func(conn *Client, err error)
**/
func (s *Server) OnDisconnect(fn func(conn *Client, err error)) {
	s.onDisconnect(fn)
}

/**
* OnSend: Agrega fn a las funciones que reciben los mensajes tipo send de cualquier
* conexión; conn es la conexión por la que llegó, para responderle o identificarla.
* @param fn func(conn *Client, message []byte)
**/
func (s *Server) OnSend(fn func(conn *Client, message []byte)) {
	s.onSend(fn)
}

/**
* OnRequest: Agrega fn a las funciones que atienden los mensajes tipo request de
* cualquier conexión, con la misma regla que Client.OnRequest.
* @param fn func(conn *Client, message []byte) ([]byte, error)
**/
func (s *Server) OnRequest(fn func(conn *Client, message []byte) ([]byte, error)) {
	s.onRequest(fn)
}

/**
* OnSendStream: Define la función que recibe los SendStream de cualquier conexión.
* @param fn func(conn *Client, in *StreamReader)
**/
func (s *Server) OnSendStream(fn func(conn *Client, in *StreamReader)) {
	s.onSendStream(fn)
}

/**
* OnRequestStream: Define la función que atiende los RequestStream de cualquier conexión.
* @param fn func(conn *Client, in *StreamReader) ([]byte, error)
**/
func (s *Server) OnRequestStream(fn func(conn *Client, in *StreamReader) ([]byte, error)) {
	s.onRequestStream(fn)
}

/**
* OnReceiveStream: Define la función que atiende los ReceiveStream de cualquier conexión.
* @param fn func(conn *Client, message []byte, out *StreamWriter) error
**/
func (s *Server) OnReceiveStream(fn func(conn *Client, message []byte, out *StreamWriter) error) {
	s.onReceiveStream(fn)
}

// ---- StreamWriter ----

/**
* Write: Envía un trozo. Si el lector tiene muchos trozos sin leer, espera a que lea
* (hasta el timeout). Si el lector dejó de leer, retorna su motivo.
* @param data []byte
* @return error
**/
func (s *StreamWriter) Write(data []byte) error {
	return s.write(data)
}

/**
* Close: Termina el stream con normalidad; el lector recibe io.EOF.
* @return error
**/
func (s *StreamWriter) Close() error {
	return s.close()
}

/**
* CloseWithError: Termina el stream con un error; el lector lo recibe en lugar de io.EOF.
* @param reason error
* @return error
**/
func (s *StreamWriter) CloseWithError(reason error) error {
	return s.closeWithError(reason)
}

/**
* CloseAndReceive: Solo para RequestStream: termina el stream y espera la respuesta.
* @return []byte, error
**/
func (s *StreamWriter) CloseAndReceive() ([]byte, error) {
	return s.closeAndReceive()
}

// ---- StreamReader ----

/**
* Read: Retorna el siguiente trozo, en orden; io.EOF al terminar el stream, o el error
* con que lo cerró el escritor. Espera cada trozo hasta el timeout.
* @return []byte, error
**/
func (s *StreamReader) Read() ([]byte, error) {
	return s.read()
}

/**
* Close: Deja de leer antes del final; el escritor recibe MSG_TCP_STREAM_CANCELED.
**/
func (s *StreamReader) Close() {
	s.close()
}

// ---- Hub ----

/**
* NewHub: Crea un Hub que todavía no escucha ni conecta (no arranca goroutines). Con id
* vacío se genera un UUID; id y name son la identidad de sus servidores y clientes.
* @param id, name string
* @return *Hub
**/
func NewHub(id, name string) *Hub {
	return newHub(id, name)
}

/**
* ID: Retorna el identificador del Hub.
* @return string
**/
func (h *Hub) ID() string {
	return h.id()
}

/**
* Name: Retorna el nombre del Hub.
* @return string
**/
func (h *Hub) Name() string {
	return h.name()
}

/**
* ToJson: Retorna id, name, las métricas de cada servidor y el estado de cada cliente.
* @return et.Json
**/
func (h *Hub) ToJson() et.Json {
	return h.toJson()
}

/**
* LoadServer: Crea un servidor del Hub, lo pone a escuchar en :port y lo registra.
* Con port 0 el sistema elige el puerto.
* @param port int
* @return *Server, error
**/
func (h *Hub) LoadServer(port int) (*Server, error) {
	return h.loadServer(port)
}

/**
* DownServer: Cierra el servidor de port (espera sus OnDisconnect) y lo quita del Hub.
* @param port int
* @return error
**/
func (h *Hub) DownServer(port int) error {
	return h.downServer(port)
}

/**
* Servers: Retorna los servidores cargados.
* @return []*Server
**/
func (h *Hub) Servers() []*Server {
	return h.servers()
}

/**
* Metrics: Retorna los clientes conectados al servidor de port y sus métricas.
* @param port int
* @return et.Json, error
**/
func (h *Hub) Metrics(port int) (et.Json, error) {
	return h.metrics(port)
}

/**
* ConnectTo: Abre un cliente del Hub hacia addr (o reconecta uno caído) y lo registra.
* @param addr string
* @return *Client, error
**/
func (h *Hub) ConnectTo(addr string) (*Client, error) {
	return h.connectTo(addr)
}

/**
* Disconnect: Cierra el cliente de addr y lo quita del Hub.
* @param addr string
* @return error
**/
func (h *Hub) Disconnect(addr string) error {
	return h.disconnect(addr)
}

/**
* Clients: Retorna los clientes registrados, conectados o no.
* @return []*Client
**/
func (h *Hub) Clients() []*Client {
	return h.clients()
}

/**
* Send: Envía msg por el cliente de addr (cliente → servidor). From lo pone el Hub.
* @param addr string, msg *Message
* @return error
**/
func (h *Hub) Send(addr string, msg *Message) error {
	return h.send(addr, msg)
}

/**
* SendTo: Envía msg a cada conexión del servidor de port cuyo ID es clientID
* (servidor → cliente). From lo pone el Hub.
* @param port int, clientID string, msg *Message
* @return error
**/
func (h *Hub) SendTo(port int, clientID string, msg *Message) error {
	return h.sendTo(port, clientID, msg)
}

/**
* OnMessage: Agrega fn a las funciones que reciben cada Message, en orden por conexión.
* srv es el servidor que lo recibió, o nil si llegó por un cliente del Hub.
* @param fn func(srv *Server, conn *Client, msg *Message)
**/
func (h *Hub) OnMessage(fn func(srv *Server, conn *Client, msg *Message)) {
	h.onMessage(fn)
}

/**
* OnConnect: Agrega fn a las funciones que se llaman al abrir una conexión del Hub.
* @param fn func(srv *Server, conn *Client)
**/
func (h *Hub) OnConnect(fn func(srv *Server, conn *Client)) {
	h.onConnect(fn)
}

/**
* OnDisconnect: Agrega fn a las funciones que se llaman, una vez por conexión, al cerrarse.
* @param fn func(srv *Server, conn *Client, err error)
**/
func (h *Hub) OnDisconnect(fn func(srv *Server, conn *Client, err error)) {
	h.onDisconnect(fn)
}

/**
* Heartbeat: Arranca o cambia el latido de todos los clientes del Hub; interval <= 0
* lo detiene. Con 3 latidos seguidos sin respuesta se cierra la conexión.
* @param interval time.Duration
**/
func (h *Hub) Heartbeat(interval time.Duration) {
	h.heartbeat(interval)
}

/**
* Status: Retorna el estado de cada cliente registrado (up/down, último latido, latencia).
* @return []et.Json
**/
func (h *Hub) Status() []et.Json {
	return h.status()
}

/**
* SetTimeoutServer: Cambia el timeout de todos los servidores del Hub.
* @param timeout time.Duration
* @return error
**/
func (h *Hub) SetTimeoutServer(timeout time.Duration) error {
	return h.setTimeoutServer(timeout)
}

/**
* SetTimeoutClient: Cambia el timeout de todos los clientes del Hub.
* @param timeout time.Duration
* @return error
**/
func (h *Hub) SetTimeoutClient(timeout time.Duration) error {
	return h.setTimeoutClient(timeout)
}

/**
* SetSecurity: Cambia la seguridad (TLS y secreto del saludo, spec §7.4) de todos los
* servidores y clientes del Hub, también los que se creen después; aplica a las
* conexiones que se abran o acepten desde ahora.
* @param cfg Security
**/
func (h *Hub) SetSecurity(cfg Security) {
	h.setSecurity(cfg)
}

/**
* Close: Detiene el heartbeat, cierra todos los clientes y da de baja todos los servidores.
* @return error
**/
func (h *Hub) Close() error {
	return h.close()
}

// ---- Seguridad ----

/**
* NewTLS: Configuraciones TLS 1.3 de servidor y de cliente para Security (spec §7.4).
* Sin certFile ni keyFile usa un certificado autofirmado en memoria; sin caFile no
* verifica el certificado del otro lado (la autenticación queda en el secreto); con
* caFile, TLS mutuo verificado contra esa CA.
* @param certFile, keyFile, caFile string
* @return *tls.Config, *tls.Config, error (servidor, cliente)
**/
func NewTLS(certFile, keyFile, caFile string) (*tls.Config, *tls.Config, error) {
	return newTLS(certFile, keyFile, caFile)
}
