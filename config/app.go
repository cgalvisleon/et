package config

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
)

type app struct {
	*et.Json
	Name       string `json:"name"`
	Version    string `json:"version"`
	Company    string `json:"company"`
	Web        string `json:"web"`
	Help       string `json:"help"`
	Production bool   `json:"production"`
	Host       string `json:"host"`
	PathApi    string `json:"path_api"`
	PathApp    string `json:"path_app"`
	Port       int    `json:"port"`
	Stage      string `json:"stage"`
	Debug      bool   `json:"debug"`
}

var App *app

func init() {
	App = newApp()
}

/**
* Valid
* @return error
**/
func (s *app) Valid() error {
	if s.Name == "" {
		return fmt.Errorf(msg.ERR_ENV_REQUIRED, "NAME")
	}

	if s.Version == "" {
		return fmt.Errorf(msg.ERR_ENV_REQUIRED, "VERSION")
	}

	if s.Company == "" {
		return fmt.Errorf(msg.ERR_ENV_REQUIRED, "COMPANY")
	}

	if s.Web == "" {
		return fmt.Errorf(msg.ERR_ENV_REQUIRED, "WEB")
	}

	if s.Help == "" {
		return fmt.Errorf(msg.ERR_ENV_REQUIRED, "HELP")
	}

	if s.Host == "" {
		return fmt.Errorf(msg.ERR_ENV_REQUIRED, "HOST")
	}

	if s.Port == 0 {
		return fmt.Errorf(msg.ERR_ENV_REQUIRED, "PORT")
	}

	if s.Stage == "" {
		return fmt.Errorf(msg.ERR_ENV_REQUIRED, "STAGE")
	}

	return nil
}

/**
* ToJson
* @return et.Json
**/
func (s *app) ToJson() et.Json {
	result := *s.Json
	result.Set("name", s.Name)
	result.Set("version", s.Version)
	result.Set("company", s.Company)
	result.Set("web", s.Web)
	result.Set("help", s.Help)
	result.Set("production", s.Production)
	result.Set("host", s.Host)
	result.Set("path_api", s.PathApi)
	result.Set("path_app", s.PathApp)
	result.Set("port", s.Port)
	result.Set("stage", s.Stage)
	result.Set("debug", s.Debug)
	return result
}

/**
* Set
* @param key string, val interface{}
* @return void
**/
func (s *app) Set(key string, val interface{}) {
	key = strings.ToUpper(key)
	s.Json.Set(key, val)
	Set(key, val)
}

/**
* SetStage
* @param stage string
* @return string
**/
func (s *app) SetStage(stage string) string {
	s.Stage = stage
	Set("STAGE", stage)

	return s.Stage
}

/**
* SetName
* @param name string
* @return string
**/
func (s *app) SetName(name string) string {
	s.Name = name
	Set("NAME", name)

	return s.Name
}

/**
* SetVersion
* @param version string
* @return string
**/
func (s *app) SetVersion(version string) string {
	s.Version = version
	Set("VERSION", version)

	return s.Version
}

/**
* SetCompany
* @param company string
* @return string
**/
func (s *app) SetCompany(company string) string {
	s.Company = company
	Set("COMPANY", company)

	return s.Company
}

/**
* SetWeb
* @param web string
* @return string
**/
func (s *app) SetWeb(web string) string {
	s.Web = web
	Set("WEB", web)

	return s.Web
}

/**
* SetHelp
* @param help string
* @return string
**/
func (s *app) SetHelp(help string) string {
	s.Help = help
	Set("HELP", help)

	return s.Help
}

/**
* SetHost
* @param host string
* @return string
**/
func (s *app) SetHost(host string) string {
	s.Host = host
	Set("HOST", host)

	return s.Host
}

/**
* SetPathApi
* @param pathApi string
* @return string
**/
func (s *app) SetPathApi(pathApi string) string {
	s.PathApi = pathApi
	Set("PATH_API", pathApi)

	return s.PathApi
}

/**
* SetPathApp
* @param pathApp string
* @return string
**/
func (s *app) SetPathApp(pathApp string) string {
	s.PathApp = pathApp
	Set("PATH_APP", pathApp)

	return s.PathApp
}

/**
* SetProduction
* @param production bool
* @return bool
**/
func (s *app) SetProduction(production bool) bool {
	s.Production = production
	Set("PRODUCTION", production)

	return s.Production
}

/**
* SetPort
* @param port int
* @return int
**/
func (s *app) SetPort(port int) int {
	s.Port = port
	Set("PORT", port)

	return s.Port
}

/**
* load
**/
func (s *app) load() {
	s.Name = GetStr("NAME", "et")
	s.Version = GetStr("VERSION", "0.0.1")
	s.Company = GetStr("COMPANY", "et")
	s.Web = GetStr("WEB", "https://et.com")
	s.Help = GetStr("HELP", "https://et.com/help")
	s.Host = GetStr("HOST", "localhost")
	s.PathApi = GetStr("PATH_API", "/api")
	s.PathApp = GetStr("PATH_APP", "/app")
	s.Production = GetBool("PRODUCTION", false)
	s.Port = GetInt("PORT", 3300)
	s.Stage = GetStr("STAGE", "local")
	s.Debug = GetBool("DEBUG", false)
}

/**
* newApp
* @return *app
**/
func newApp() *app {
	result := &app{}
	result.load()

	return result
}
