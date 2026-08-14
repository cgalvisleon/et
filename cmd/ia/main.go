package main

import (
	"net/http"
	"os"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/ettp/v2"
	"github.com/cgalvisleon/et/ia"
	"github.com/cgalvisleon/et/logs"
)

// engine is the ia.Engine shared by every HTTP handler in this binary.
var engine *ia.Engine

func main() {
	engine = mustEngine()

	timeout := envar.GetFloat("TIMEOUT", 100)
	srv, err := ettp.New("Ia", &ettp.Config{
		Port:         envar.GetInt("PORT", 4070),
		RpcPort:      envar.GetInt("RPC_PORT", 4700),
		Parent:       "/api",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
		Timeout:      time.Duration(timeout) * time.Second,
		IsTLS:        false,
		Debug:        true,
	})
	if err != nil {
		logs.Panic(err)
	}

	registerRoutes(srv)

	srv.Start()
}

/**
* registerRoutes: wires the conversational endpoint plus the knowledge-base support
* endpoints (Learn/Revise/Verify/Unload/IsLoaded) onto srv.
* @param srv *ettp.Server
**/
func registerRoutes(srv *ettp.Server) {
	routes := []struct {
		method  string
		path    string
		handler func(w http.ResponseWriter, r *http.Request)
	}{
		{ettp.POST, "/conversation", httpConversation},
		{ettp.POST, "/learn", httpLearn},
		{ettp.PUT, "/revise/{factId}", httpRevise},
		{ettp.POST, "/verify", httpVerify},
		{ettp.DELETE, "/unload/{kbId}", httpUnload},
		{ettp.GET, "/loaded/{kbId}", httpIsLoaded},
	}

	for _, route := range routes {
		if _, err := srv.Public(route.method, route.path, route.handler, srv.Name); err != nil {
			logs.Panic(err)
		}
	}
}

/**
* mustEngine: builds the ia.Engine used by every handler. The classifier's model is
* loaded from IA_MODEL_PATH when it exists, or trained on the bundled sample dataset
* otherwise (and saved to IA_MODEL_PATH for next time, if set). Knowledge bases are
* kept in memory only (no Store), evicted after IA_IDLE_TTL_MINUTES of inactivity.
* @return *ia.Engine
**/
func mustEngine() *ia.Engine {
	model, err := loadOrTrainModel()
	if err != nil {
		logs.Panic(err)
	}

	classifier, err := ia.NewClassifier(model, nil)
	if err != nil {
		logs.Panic(err)
	}

	idleMinutes := envar.GetInt("IA_IDLE_TTL_MINUTES", 60)
	result, err := ia.New(nil, classifier, time.Duration(idleMinutes)*time.Minute)
	if err != nil {
		logs.Panic(err)
	}

	return result
}

/**
* loadOrTrainModel: reads a previously trained model from IA_MODEL_PATH when set and
* present on disk; otherwise trains one from ia.SampleDataset and, if IA_MODEL_PATH is
* set, persists it there for subsequent runs.
* @return *ia.Model, error
**/
func loadOrTrainModel() (*ia.Model, error) {
	path := envar.GetStr("IA_MODEL_PATH", "")
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			return ia.LoadModel(path)
		}
	}

	examples, err := ia.SampleDataset()
	if err != nil {
		return nil, err
	}

	model, metrics, err := ia.TrainFromExamples(examples, nil, 500, 0.3, 0.8, time.Now().UnixNano())
	if err != nil {
		return nil, err
	}
	logs.Infof("modelo entrenado con el dataset de ejemplo (accuracy=%.2f, f1=%.2f)", metrics.Accuracy, metrics.F1)

	if path != "" {
		if err := model.Save(path); err != nil {
			logs.Alert(err)
		}
	}

	return model, nil
}
