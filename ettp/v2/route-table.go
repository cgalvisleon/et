package ettp

import (
	"net/http"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/middleware"
)

/**
* initRouteTable
**/
func (s *Server) initRouteTable() error {
	s.Public(GET, "/version", s.getVersion, "Apigateway")
	s.Public(GET, "/test/{id}/{test}", s.getVersion, "Apigateway")
	// Develop Token
	production := config.App.Production
	if !production {
		s.Public(GET, "/develop/token", s.handlerDevToken, "Apigateway")
	}

	return s.Save()
}

/**
* getVersion
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) getVersion(w http.ResponseWriter, r *http.Request) {
	metric := middleware.GetMetrics(r)

	result := et.Json{
		"created_at": s.CreatedAt,
		"version":    s.Version,
		"service":    s.Name,
		"host":       s.Host,
		"company":    config.App.Company,
		"web":        config.App.Web,
		"help":       config.App.Help,
	}

	metric.JSON(w, r, http.StatusOK, result)
}

/**
* handlerDevToken
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) handlerDevToken(w http.ResponseWriter, r *http.Request) {
	metric := middleware.GetMetrics(r)

	developToken := func() string {
		production := config.App.Production
		if production {
			return ""
		}

		device := "DevelopToken"
		duration := time.Hour * 2
		token, err := claim.NewToken(device, device, device, device, device, duration)
		if err != nil {
			console.Alert(err)
			return ""
		}

		_, err = claim.ValidToken(token)
		if err != nil {
			console.Alertf("GetFromToken:%s", err.Error())
			return ""
		}

		return token
	}
	token := developToken()

	metric.JSON(w, r, http.StatusOK, et.Json{
		"token": token,
	})
}
