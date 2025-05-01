package config

import (
	"fmt"

	"github.com/cgalvisleon/et/msg"
)

type app struct {
	Name       string
	Version    string
	Company    string
	Web        string
	Help       string
	Production bool
	Host       string
	PathUrl    string
	Port       int
	Stage      string
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
* SetPathUrl
* @param pathUrl string
* @return string
**/
func (s *app) SetPathUrl(pathUrl string) string {
	s.PathUrl = pathUrl
	Set("PATH_URL", pathUrl)

	return s.PathUrl
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
* Reload
**/
func (s *app) Reload() {
	s.Name = String("NAME", "et")
	s.Version = String("VERSION", "0.0.1")
	s.Company = String("COMPANY", "et")
	s.Web = String("WEB", "https://et.com")
	s.Help = String("HELP", "https://et.com/help")
	s.Host = String("HOST", "localhost")
	s.PathUrl = String("PATH_URL", "")
	s.Production = Bool("PRODUCTION", false)
	s.Port = Int("PORT", 3300)
	s.Stage = String("STAGE", "local")
}

/**
* newApp
* @return *app
**/
func newApp() *app {
	result := &app{}
	result.Reload()

	return result
}
