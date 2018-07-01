package server

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/sirupsen/logrus"
	"gitlab.com/yakshaving.art/git-pull-mirror/config"
	"gitlab.com/yakshaving.art/git-pull-mirror/github"
)

// WebHooksServer is the server that will listen for webhooks calls and handle them
type WebHooksServer struct {
	wg *sync.WaitGroup

	opts         WebHooksServerOptions
	repositories config.Config
	running      bool
	callbackPath string
}

// WebHooksServerOptions holds server configuration options
type WebHooksServerOptions struct {
	GitTimeoutSeconds int
	RepositoriesPath  string
	SSHPrivateKey     string
	GitHubClientOpts  github.ClientOpts
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

	callback, err := url.Parse(ws.opts.GitHubClientOpts.CallbackURL)
	if err != nil {
		return fmt.Errorf("could not parse callback url %s: %s", ws.opts.GitHubClientOpts.CallbackURL, err)
	}
	ws.callbackPath = callback.Path

	g := newGitClient(ws.opts)
	gh := github.New(ws.opts.GitHubClientOpts)

	errors := make(chan error, len(c.Repostitories))

	wg := &sync.WaitGroup{}
	for _, r := range c.Repostitories {
		wg.Add(1)
		go func(r config.RepositoryConfig) {
			defer wg.Done()

			repo, err := g.CloneOrOpen(r.OriginURL, r.TargetURL)
			if err != nil {
				errors <- fmt.Errorf("failed to clone or open %s: %s", r.OriginURL, err)
				return
			}

			if err = repo.Fetch(); err != nil {
				errors <- fmt.Errorf("failed to fetch %s: %s", r.OriginURL, err)
				return
			}

			if err = gh.RegisterWebhook(r.OriginURL); err != nil {
				errors <- fmt.Errorf("failed to register webhooks for %s: %s", r.OriginURL, err)
				return
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
	http.HandleFunc(ws.callbackPath, func(w http.ResponseWriter, r *http.Request) {
		if !ws.running {
			http.Error(w, "server is shutting down", http.StatusServiceUnavailable)
			return
		}

		ws.wg.Add(1)
		defer ws.wg.Done()

		r.ParseForm()

		logrus.Infof("URI: %s", r.RequestURI)
		logrus.Infof("Form: %#v", r.Form)

		w.WriteHeader(http.StatusOK)
	})

	logrus.Infof("starting listener on %s", address)
	if err := http.ListenAndServe(address, nil); err != nil {
		logrus.Fatalf("failed to start http server: %s", err)
	}
}

// Shutdown performs a graceful shutdown of the webhooks server
func (ws *WebHooksServer) Shutdown() {
	ws.running = false

	// Wait for all the ongoing requests to finish
	ws.wg.Wait()

	logrus.Infof("server stopped")
}
