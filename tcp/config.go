package tcp

import (
	"github.com/cgalvisleon/et/file"
)

type Config struct {
	Nodes    []string `json:"nodes"`
	filePath string   `json:"-"`
}

/**
* getConfig: Returns the config
* @return *Config, error
**/
func getConfig() (*Config, error) {
	filePath := "./config.json"
	var result *Config
	err := file.Read(filePath, &result)
	if err != nil {
		return nil, err
	}

	if result == nil {
		result = &Config{
			Nodes: []string{},
		}

		file.Write(filePath, result)
	}

	result.filePath = filePath

	return result, nil
}
