package daemon

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/go-chi/chi/v5"
)

// Systemd: the default daemon, a minimal HTTP server that answers at /.
type Systemd struct {
	serviceName string
	port        int
	server      *http.Server
	isRunning   bool
	wg          sync.WaitGroup
}

/**
* New: A stopped Systemd daemon.
* @return RepositoryCMD
**/
func New() RepositoryCMD {
	return &Systemd{
		serviceName: "systemd",
		isRunning:   false,
		wg:          sync.WaitGroup{},
	}
}

/**
* Version: Prints and returns the version.
* @return string
**/
func (s *Systemd) Version() string {
	result := "Version: 1.0.0"
	println(result)

	return result
}

/**
* Help: Prints the usage when key is empty.
* @param key string
**/
func (s *Systemd) Help(key string) {
	if key == "" {
		println("Uso: daemon [opciones]")
		println("Opciones:")
		println("  --h, --help     Mostrar esta ayuda")
		println("  --v, --version  Mostrar la versión")
		println("  --s, --status   Mostrar el estado del servicio")
		println("  --r, --restart  Reiniciar el servicio")
		println("  --up   				 Actualizar el servicio")
		println("  --down   			 Detener el servicio")
		println("  --start   			 Iniciar el servicio")
	}
}

/**
* SetConfig: Reads the port from PORT (1377 by default); an empty cfg only prints a message.
* @param cfg string
**/
func (s *Systemd) SetConfig(cfg string) {
	if cfg == "" {
		println(MSG_CONFIG_REQUIRED)
		return
	}

	s.port = envar.GetInt("PORT", 1377)
}

/**
* Status: The daemon status (empty for now).
* @return et.Json
**/
func (s *Systemd) Status() et.Json {
	return et.Json{}
}

/**
* Start: Starts the HTTP server in the background and stops it on SIGINT/SIGTERM; returns at once.
* @return et.Item
**/
func (s *Systemd) Start() et.Item {
	if s.isRunning {
		return et.Item{
			Ok:     true,
			Result: et.Json{"message": MSG_SERVICE_RUNNING},
		}
	}

	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	s.server = &http.Server{
		Addr:    fmt.Sprintf(`:%d`, s.port),
		Handler: r,
	}

	s.wg.Add(1)
	s.isRunning = true
	go func() {
		defer s.wg.Done()
		logs.Logf(s.serviceName, `Iniciando el servidor en http://localhost:%d`, s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.isRunning = false
			logs.Alert(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		s.Stop()
	}()

	return et.Item{
		Ok:     true,
		Result: et.Json{"message": MSG_SERVICE_RUNNING},
	}
}

/**
* Stop: Shuts the HTTP server down if it is running.
* @return et.Item
**/
func (s *Systemd) Stop() et.Item {
	if !s.isRunning {
		return et.Item{
			Ok:     true,
			Result: et.Json{"message": MSG_SERVICE_CLOSED},
		}
	}

	fmt.Println("Deteniendo el servidor...")
	if err := s.server.Shutdown(context.Background()); err != nil {
		log.Fatal(err)
	}
	s.wg.Wait()
	s.isRunning = false
	logs.Log("Servidor detenido.")

	return et.Item{
		Ok:     true,
		Result: et.Json{"message": MSG_SERVICE_CLOSED},
	}
}

/**
* Restart: Not implemented; returns an empty item.
* @return et.Item
**/
func (s *Systemd) Restart() et.Item {
	return et.Item{}
}

func init() {
	Registry("systemd", New())
}
