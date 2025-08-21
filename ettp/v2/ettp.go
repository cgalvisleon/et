package ettp

/**
* New
* @param port int, config *Config
* @return *Server
**/
func New(port int, config *Config) *Server {
	result := NewServer(port, config)

	return result
}
