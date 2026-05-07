package tcp

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"time"
)

// ---------------------------------------------------------------------------
// TLS config helpers
// ---------------------------------------------------------------------------

/**
* NewTLSFromFiles: Creates a server *tls.Config from PEM certificate and key files.
* @param certFile string
* @param keyFile string
* @return *tls.Config, error
**/
func NewTLSFromFiles(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

/**
* NewMTLSConfig: Creates a mutual TLS *tls.Config requiring client certificate verification.
* @param certFile string
* @param keyFile string
* @param caFile string
* @return *tls.Config, error
**/
func NewMTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("failed to parse CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

/**
* NewClientTLSConfig: Creates a client *tls.Config that trusts the given CA file.
* @param caFile string
* @return *tls.Config, error
**/
func NewClientTLSConfig(caFile string) (*tls.Config, error) {
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("failed to parse CA certificate")
	}

	return &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}, nil
}

/**
* NewSelfSignedTLS: Generates an in-memory self-signed TLS config for development.
* @return *tls.Config, error
**/
func NewSelfSignedTLS() (*tls.Config, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"et/tcp"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// ---------------------------------------------------------------------------
// TLSNode — decorator over *Node
// ---------------------------------------------------------------------------

type TLSNode struct {
	*Node
}

/**
* NewTLSNode: Creates a TLS-wrapped Node without modifying the original Node.
* @param port int
* @param cfg *tls.Config
* @return *TLSNode
**/
func NewTLSNode(port int, cfg *tls.Config) *TLSNode {
	node := NewNode(port)
	node.tlsConfig = cfg
	return &TLSNode{Node: node}
}

// ---------------------------------------------------------------------------
// TLSClient — decorator over *Client
// ---------------------------------------------------------------------------

type TLSClient struct {
	*Client
}

/**
* NewTLSClient: Creates a TLS-wrapped Client without modifying the original Client.
* @param addr string
* @param cfg *tls.Config
* @return *TLSClient
**/
func NewTLSClient(addr string, cfg *tls.Config) *TLSClient {
	client := NewClient(addr)
	client.tlsConfig = cfg
	return &TLSClient{Client: client}
}
