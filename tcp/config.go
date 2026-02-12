package tcp

import (
	"github.com/cgalvisleon/et/file"
)

type Config struct {
	TCP []string `json:"tcp"`
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
			TCP: []string{},
		}

		err = file.Write(filePath, result)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	return result, nil
}

/**
* GetNodes: Returns the nodes
* @return []string, error
**/
func GetNodes() ([]string, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}

	return config.TCP, nil
}
