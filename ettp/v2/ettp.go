package ettp

/**
* New
* @param name string, config *Config
* @return *Server
**/
func New(name string, config *Config) *Server {
	result := NewServer(name, config)

	return result
}
