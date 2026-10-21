package websocket

import (
	"sync"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
)

/**
* handlers: Funciones de evento del servidor. Se agregan y se llaman en orden, sin
* locks del servidor tomados.
**/
type handlers struct {
	mu             sync.RWMutex                            // protege las listas
	connectFns     []func(c *Conn) error                   // pueden rechazar la conexión
	disconnectFns  []func(c *Conn, err error)              // al desconectarse
	subscribeFns   []func(c *Conn, sub Subscription) error // pueden rechazar la suscripción
	unsubscribeFns []func(c *Conn, sub Subscription)       // después de desuscribir
}

/**
* call: Llama a fn convirtiendo un panic en error (y registrándolo en el log).
* @param fn func() error
* @return error
**/
func call(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = logs.Errorf(msg.MSG_WS_HANDLER_PANIC, r)
		}
	}()
	return fn()
}

/**
* onConnect: Agrega fn a las funciones que se llaman al conectarse un cliente.
* @param fn func(c *Conn) error
**/
func (s *handlers) onConnect(fn func(c *Conn) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connectFns = append(s.connectFns, fn)
}

/**
* onDisconnect: Agrega fn a las funciones que se llaman al desconectarse un cliente.
* @param fn func(c *Conn, err error)
**/
func (s *handlers) onDisconnect(fn func(c *Conn, err error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.disconnectFns = append(s.disconnectFns, fn)
}

/**
* onSubscribe: Agrega fn a las funciones que se llaman antes de cada suscripción.
* @param fn func(c *Conn, sub Subscription) error
**/
func (s *handlers) onSubscribe(fn func(c *Conn, sub Subscription) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribeFns = append(s.subscribeFns, fn)
}

/**
* onUnsubscribe: Agrega fn a las funciones que se llaman después de cada desuscripción.
* @param fn func(c *Conn, sub Subscription)
**/
func (s *handlers) onUnsubscribe(fn func(c *Conn, sub Subscription)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unsubscribeFns = append(s.unsubscribeFns, fn)
}

/**
* connect: Llama a las funciones de onConnect; la primera que retorna error (o hace
* panic) rechaza la conexión.
* @param c *Conn
* @return error
**/
func (s *handlers) connect(c *Conn) error {
	s.mu.RLock()
	fns := s.connectFns
	s.mu.RUnlock()

	for _, fn := range fns {
		if err := call(func() error { return fn(c) }); err != nil {
			return err
		}
	}
	return nil
}

/**
* disconnect: Llama a las funciones de onDisconnect; un panic se registra y se sigue.
* @param c *Conn, err error
**/
func (s *handlers) disconnect(c *Conn, err error) {
	s.mu.RLock()
	fns := s.disconnectFns
	s.mu.RUnlock()

	for _, fn := range fns {
		call(func() error { fn(c, err); return nil })
	}
}

/**
* subscribe: Llama a las funciones de onSubscribe; la primera que retorna error (o
* hace panic) rechaza la suscripción.
* @param c *Conn, sub Subscription
* @return error
**/
func (s *handlers) subscribe(c *Conn, sub Subscription) error {
	s.mu.RLock()
	fns := s.subscribeFns
	s.mu.RUnlock()

	for _, fn := range fns {
		if err := call(func() error { return fn(c, sub) }); err != nil {
			return err
		}
	}
	return nil
}

/**
* unsubscribe: Llama a las funciones de onUnsubscribe; un panic se registra y se sigue.
* @param c *Conn, sub Subscription
**/
func (s *handlers) unsubscribe(c *Conn, sub Subscription) {
	s.mu.RLock()
	fns := s.unsubscribeFns
	s.mu.RUnlock()

	for _, fn := range fns {
		call(func() error { fn(c, sub); return nil })
	}
}
