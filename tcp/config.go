package tcp

import (
	"github.com/cgalvisleon/et/file"
)

type Config struct {
	Nodes []string `json:"nodes"`
}

/**
* getConfig: Returns the config
* @return *Config, error
**/
func getConfig(path string) (*Config, error) {
	var result *Config
	err := file.Read(path, &result)
	if err != nil {
		return nil, err
	}

	if result == nil {
		result = &Config{
			Nodes: []string{},
		}

		file.Write(path, result)
	}

	return result, nil
}
