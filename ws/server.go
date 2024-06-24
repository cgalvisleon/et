package ws

type Conn struct {
	hub *Hub
}

var servws *Conn

// Load the Websocket Hub
func Server() (*Conn, error) {
	if servws != nil {
		return servws, nil
	}

	hub := NewHub()
	go hub.Run()

	servws = &Conn{
		hub: hub,
	}

	return servws, nil
}

// Close the Websocket Hub
func Close() error {
	return nil
}
