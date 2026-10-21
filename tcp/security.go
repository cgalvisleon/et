package tcp

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"sync"
	"time"

	"github.com/cgalvisleon/et/msg"
)

const (
	nonceSize     = 32                      // bytes de cada nonce del saludo
	exporterLabel = "EXPORTER-josefina-tcp" // etiqueta del canal TLS en las pruebas (RFC 5705)
	proofServer   = "josefina-tcp/server"   // dominio de la prueba que envía el servidor
	proofClient   = "josefina-tcp/client"   // dominio de la prueba que envía el cliente
	kidDomain     = "josefina-tcp/kid"      // dominio del identificador de un secreto
)

/**
* Security: Seguridad de las conexiones (spec §7.4). Los campos vacíos la apagan.
* Secret: secreto compartido para la autenticación mutua del saludo.
* ServerTLS: TLS del lado que escucha. ClientTLS: TLS del lado que marca.
**/
type Security struct {
	Secret    string
	Previous  []string // secretos anteriores que todavía se aceptan (rotación, §7.4)
	ServerTLS *tls.Config
	ClientTLS *tls.Config
}

/**
* secrets: El secreto vigente seguido de los anteriores, sin vacíos ni repetidos
* (vacío = saludo sin prueba).
* @return []string
**/
func (s Security) secrets() []string {
	result := []string{}
	for _, secret := range append([]string{s.Secret}, s.Previous...) {
		if secret == "" {
			continue
		}
		seen := false
		for _, have := range result {
			if have == secret {
				seen = true
				break
			}
		}
		if !seen {
			result = append(result, secret)
		}
	}
	return result
}

/**
* kid: Identificador público de un secreto (los primeros 8 bytes de
* HMAC-SHA256(secret, dominio), en hexadecimal). El cliente lo envía en el saludo
* para que el servidor pruebe con el secreto que los dos conocen.
* @param secret string
* @return string
**/
func kid(secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(kidDomain))
	return hex.EncodeToString(mac.Sum(nil)[:8])
}

/**
* pickSecret: El primero de secrets cuyo kid está en kids; sin kids (un cliente
* anterior a la rotación), el vigente.
* @param secrets, kids []string
* @return string, bool
**/
func pickSecret(secrets, kids []string) (string, bool) {
	if len(kids) == 0 {
		return secrets[0], true
	}
	for _, secret := range secrets {
		id := kid(secret)
		for _, have := range kids {
			if have == id {
				return secret, true
			}
		}
	}
	return "", false
}

/**
* security: Security que se puede cambiar con conexiones en uso; aplica a las que se
* abran después. El Hub comparte el mismo puntero con sus servidores y clientes.
**/
type security struct {
	mu  sync.RWMutex
	cfg Security
}

/**
* get: Retorna la configuración actual.
* @return Security
**/
func (s *security) get() Security {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

/**
* set: Reemplaza la configuración.
* @param cfg Security
**/
func (s *security) set(cfg Security) {
	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
}

/**
* newNonce: Un nonce aleatorio para el saludo.
* @return []byte, error
**/
func newNonce() ([]byte, error) {
	result := make([]byte, nonceSize)
	if _, err := rand.Read(result); err != nil {
		return nil, err
	}
	return result, nil
}

/**
* channelBinding: Material exportado de la sesión TLS de conn (vacío sin TLS). Entra
* en las pruebas del saludo, así una prueba no sirve en otra conexión TLS (un
* intermediario que termina TLS en los dos lados tendría dos sesiones distintas).
* @param conn net.Conn
* @return []byte, error
**/
func channelBinding(conn net.Conn) ([]byte, error) {
	tc, ok := conn.(*tls.Conn)
	if !ok {
		return nil, nil
	}
	state := tc.ConnectionState()
	return state.ExportKeyingMaterial(exporterLabel, nil, 32)
}

/**
* proof: HMAC-SHA256 con secret sobre el dominio, los dos nonces, el canal TLS y las
* dos identidades, cada parte con su largo delante.
* @param secret, domain string, clientNonce, serverNonce, binding []byte, clientID, serverID string
* @return []byte
**/
func proof(secret, domain string, clientNonce, serverNonce, binding []byte, clientID, serverID string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	for _, part := range [][]byte{[]byte(domain), clientNonce, serverNonce, binding, []byte(clientID), []byte(serverID)} {
		var size [4]byte
		binary.BigEndian.PutUint32(size[:], uint32(len(part)))
		mac.Write(size[:])
		mac.Write(part)
	}
	return mac.Sum(nil)
}

/**
* validProof: Si got es la prueba esperada (comparación en tiempo constante).
* @param got, want []byte
* @return bool
**/
func validProof(got, want []byte) bool {
	return len(got) > 0 && hmac.Equal(got, want)
}

/**
* tlsHandshake: Hace el handshake TLS de conn antes del timeout.
* @param conn *tls.Conn, timeout time.Duration
* @return error
**/
func tlsHandshake(conn *tls.Conn, timeout time.Duration) error {
	conn.SetDeadline(time.Now().Add(timeout))
	defer conn.SetDeadline(time.Time{})
	if err := conn.Handshake(); err != nil {
		return fmt.Errorf(msg.MSG_TCP_HANDSHAKE, err)
	}
	return nil
}

/**
* clientTLS: La configuración de cliente para marcar a addr: si verifica el
* certificado y no trae ServerName, usa el host de addr.
* @param cfg *tls.Config, addr string
* @return *tls.Config
**/
func clientTLS(cfg *tls.Config, addr string) *tls.Config {
	if cfg.ServerName != "" || cfg.InsecureSkipVerify {
		return cfg
	}
	result := cfg.Clone()
	if host, _, err := net.SplitHostPort(addr); err == nil {
		result.ServerName = host
	}
	return result
}

/**
* selfSigned: Un certificado ECDSA P-256 autofirmado, solo en memoria.
* @param name string
* @return tls.Certificate, error
**/
func selfSigned(name string) (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	if err != nil {
		return tls.Certificate{}, err
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: name},
		DNSNames:     []string{name},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

/**
* newTLS: Configuraciones TLS 1.3 de servidor y de cliente.
* Sin certFile/keyFile se usa un certificado autofirmado en memoria. Sin caFile no se
* verifica el certificado del otro lado (la autenticación queda en el secreto del
* saludo, ligado al canal); con caFile, los dos lados presentan certificado y se
* verifican contra esa CA (TLS mutuo).
* @param certFile, keyFile, caFile string
* @return *tls.Config, *tls.Config, error
**/
func newTLS(certFile, keyFile, caFile string) (*tls.Config, *tls.Config, error) {
	var cert tls.Certificate
	var err error
	switch {
	case certFile != "" && keyFile != "":
		cert, err = tls.LoadX509KeyPair(certFile, keyFile)
	case certFile == "" && keyFile == "":
		cert, err = selfSigned("josefina")
	default:
		err = errors.New(msg.MSG_TCP_TLS_KEYPAIR)
	}
	if err != nil {
		return nil, nil, err
	}

	server := &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}}
	client := &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}}
	if caFile == "" {
		client.InsecureSkipVerify = true
		return server, client, nil
	}

	data, err := os.ReadFile(caFile)
	if err != nil {
		return nil, nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, nil, fmt.Errorf(msg.MSG_TCP_TLS_CA, caFile)
	}
	server.ClientCAs = pool
	server.ClientAuth = tls.RequireAndVerifyClientCert
	client.RootCAs = pool
	return server, client, nil
}

/**
* clientHello: Saludo del lado que marca (§7.0, §7.4). Con secretos envía un nonce y
* los kid de sus secretos, verifica la prueba del servidor con el secreto que
* corresponda y responde con la suya en un tercer msgHello.
* @param l *link, secrets []string, id, name string
* @return hello, bool, error (el saludo del servidor, si quedó autenticado)
**/
func clientHello(l *link, secrets []string, id, name string) (hello, bool, error) {
	var nonce []byte
	var kids []string
	if len(secrets) > 0 {
		var err error
		if nonce, err = newNonce(); err != nil {
			return hello{}, false, err
		}
		for _, secret := range secrets {
			kids = append(kids, kid(secret))
		}
	}
	if err := l.sendHello(hello{ID: id, Name: name, Nonce: nonce, Kids: kids}); err != nil {
		return hello{}, false, err
	}

	server, err := l.readHello()
	if err != nil {
		return hello{}, false, err
	}
	if server.Error != "" {
		return hello{}, false, fmt.Errorf(msg.MSG_TCP_HANDSHAKE, server.Error)
	}
	if len(secrets) == 0 {
		return server, false, nil
	}

	binding, err := channelBinding(l.conn)
	if err != nil {
		return hello{}, false, fmt.Errorf(msg.MSG_TCP_HANDSHAKE, err)
	}
	// El servidor probó con uno de los secretos (un servidor anterior a la
	// rotación, con el suyo): vale el primero que verifique
	secret := ""
	if len(server.Nonce) >= nonceSize {
		for _, candidate := range secrets {
			want := proof(candidate, proofServer, nonce, server.Nonce, binding, id, server.ID)
			if validProof(server.Proof, want) {
				secret = candidate
				break
			}
		}
	}
	if secret == "" {
		l.sendHello(hello{Error: msg.MSG_TCP_AUTH_FAILED})
		return hello{}, false, fmt.Errorf(msg.MSG_TCP_HANDSHAKE, msg.MSG_TCP_AUTH_FAILED)
	}
	mine := proof(secret, proofClient, nonce, server.Nonce, binding, id, server.ID)
	if err := l.sendHello(hello{Proof: mine}); err != nil {
		return hello{}, false, err
	}
	return server, true, nil
}

/**
* serverHello: Saludo del lado que acepta (§7.0, §7.4). Con secretos exige el nonce
* del cliente, elige el secreto que los dos conocen (por los kid del cliente),
* responde con su nonce y su prueba, y verifica la prueba del cliente.
* @param l *link, secrets []string, id, name string
* @return hello, bool, error (el saludo del cliente, si quedó autenticado)
**/
func serverHello(l *link, secrets []string, id, name string) (hello, bool, error) {
	client, err := l.readHello()
	if err != nil {
		return hello{}, false, err
	}
	if len(secrets) == 0 {
		return client, false, l.sendHello(hello{ID: id, Name: name})
	}
	if len(client.Nonce) < nonceSize {
		l.sendHello(hello{Error: msg.MSG_TCP_AUTH_REQUIRED})
		return hello{}, false, fmt.Errorf(msg.MSG_TCP_HANDSHAKE, msg.MSG_TCP_AUTH_REQUIRED)
	}
	secret, ok := pickSecret(secrets, client.Kids)
	if !ok {
		l.sendHello(hello{Error: msg.MSG_TCP_AUTH_FAILED})
		return hello{}, false, fmt.Errorf(msg.MSG_TCP_HANDSHAKE, msg.MSG_TCP_AUTH_FAILED)
	}

	nonce, err := newNonce()
	if err != nil {
		return hello{}, false, err
	}
	binding, err := channelBinding(l.conn)
	if err != nil {
		return hello{}, false, fmt.Errorf(msg.MSG_TCP_HANDSHAKE, err)
	}
	mine := proof(secret, proofServer, client.Nonce, nonce, binding, client.ID, id)
	if err := l.sendHello(hello{ID: id, Name: name, Nonce: nonce, Proof: mine}); err != nil {
		return hello{}, false, err
	}

	answer, err := l.readHello()
	if err != nil {
		return hello{}, false, err
	}
	want := proof(secret, proofClient, client.Nonce, nonce, binding, client.ID, id)
	if !validProof(answer.Proof, want) {
		return hello{}, false, fmt.Errorf(msg.MSG_TCP_HANDSHAKE, msg.MSG_TCP_AUTH_FAILED)
	}
	return client, true, nil
}

/**
* setSecurity: Cambia la seguridad de las conexiones que abra el cliente.
* @param cfg Security
**/
func (s *Client) setSecurity(cfg Security) {
	s.sec.set(cfg)
}

/**
* setSecurity: Cambia la seguridad de las conexiones que acepte el servidor.
* @param cfg Security
**/
func (s *Server) setSecurity(cfg Security) {
	s.sec.set(cfg)
}

/**
* setSecurity: Cambia la seguridad de todos los servidores y clientes del Hub.
* @param cfg Security
**/
func (h *Hub) setSecurity(cfg Security) {
	h.sec.set(cfg)
}
