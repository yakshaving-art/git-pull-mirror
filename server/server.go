package server

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
	"gitlab.com/yakshaving.art/git-pull-mirror/config"
)

// WebHooksServer is the server that will listen for webhooks calls and handle them
type WebHooksServer struct {
	wg *sync.WaitGroup

	opts         WebHooksServerOptions
	repositories config.Config
	running      bool
}

// WebHooksServerOptions holds server configuration options
type WebHooksServerOptions struct {
	GitTimeoutSeconds int
	RepositoriesPath  string
}

// New returns a new unconfigured webhooks server
func New(opts WebHooksServerOptions) *WebHooksServer {
	return &WebHooksServer{
		wg:   &sync.WaitGroup{},
		opts: opts,
	}
}

// Configure loads the configuration on the server and sets it. Can fail if any
// part of the configuration fails to be executed, for example: if an origin git
// repo is non existing.
func (ws *WebHooksServer) Configure(c config.Config) error {
	logrus.Debug("loading configuration")
	g := newClient(ws.opts.RepositoriesPath, ws.opts.GitTimeoutSeconds)

	errors := make(chan error)

	wg := &sync.WaitGroup{}
	for _, r := range c.Repostitories {
		wg.Add(1)
		go func(r config.RepositoryConfig) {
			defer wg.Done()

			_, err := g.CloneOrOpen(r.OriginURL, r.Target)
			if err != nil {
				errors <- fmt.Errorf("%s: %s", r.OriginURL, err)
			}
		}(r)
	}
	wg.Wait()

	close(errors)

	failed := false
	for err := range errors {
		failed = true
		logrus.Errorf("failed to clone or open repository %s", err)
	}

	if failed {
		return fmt.Errorf("failed to load configuration")
	}

	ws.repositories = c
	logrus.Infof("configuration loaded successfully")
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

		logrus.Infof("URI: %s", r.RequestURI)
		logrus.Infof("Form: %#v", r.Form)

		w.WriteHeader(http.StatusOK)
	})

	logrus.Infof("Listening on %s", address)
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
