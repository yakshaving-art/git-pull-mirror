package server

import (
	"github.com/sirupsen/logrus"
	"gitlab.com/yakshaving.art/git-pull-mirror/config"
	"net/http"
	"sync"
)

// WebHooksServer is the server that will listen for webhooks calls and handle them
type WebHooksServer struct {
	wg *sync.WaitGroup

	config  config.Config
	running bool
}

// New returns a new unconfigured webhooks server
func New() *WebHooksServer {
	return &WebHooksServer{
		wg: &sync.WaitGroup{},
	}
}

// Configure loads the configuration on the server and sets it. Can fail if any
// part of the configuration fails to be executed, for example: if an origin git
// repo is non existing.
func (ws *WebHooksServer) Configure(c config.Config) error {
	return nil
}

// Run starts the execution of the server, forever
func (ws *WebHooksServer) Run(address string) {
	ws.running = true
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !ws.running {
			http.Error(w, "Server is stopping", http.StatusServiceUnavailable)
			return
		}

		ws.wg.Add(1)
		defer ws.wg.Done()

		r.ParseForm()

		logrus.Infof("URI: %s\n", r.RequestURI)
		logrus.Infof("Form: %#v", r.Form)

		w.WriteHeader(http.StatusOK)
	})

	logrus.Infof("Listening on %s\n", address)
	if err := http.ListenAndServe(address, nil); err != nil {
		logrus.Fatalf("Failed to start http server: %s", err)
	}
}

// Shutdown performs a graceful shutdown of the webhooks server
func (ws *WebHooksServer) Shutdown() {
	ws.running = false
	// Wait for all the ongoing requests to finish
	ws.wg.Wait()
	logrus.Infof("Server stopped")
}
