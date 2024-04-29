package ws

var (
	conn *Conn
)

type Conn struct {
	hub *Hub
}

// Load the Websocket Hub
func Load() (*Conn, error) {
	if conn != nil {
		return conn, nil
	}

	hub := connect()
	conn = &Conn{
		hub: hub,
	}

	return conn, nil
}

// Close the Websocket Hub
func Close() error {
	return nil
}
