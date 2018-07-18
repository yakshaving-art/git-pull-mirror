package server

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gitlab.com/yakshaving.art/git-pull-mirror/config"
	"gitlab.com/yakshaving.art/git-pull-mirror/github"
	"gitlab.com/yakshaving.art/git-pull-mirror/metrics"
)

// WebHooksServer is the server that will listen for webhooks calls and handle them
type WebHooksServer struct {
	wg   *sync.WaitGroup
	lock *sync.Mutex

	opts         WebHooksServerOptions
	config       config.Config
	repositories map[string]Repository
	running      bool
	callbackPath string
}

// WebHooksServerOptions holds server configuration options
type WebHooksServerOptions struct {
	GitTimeoutSeconds        int
	RepositoriesPath         string
	SSHPrivateKey            string
	SkipWebhooksRegistration bool
	GitHubClientOpts         github.ClientOpts
}

// New returns a new unconfigured webhooks server
func New(opts WebHooksServerOptions) *WebHooksServer {
	return &WebHooksServer{
		wg:   &sync.WaitGroup{},
		lock: &sync.Mutex{},
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

	g := newGitClient(ws.opts)
	gh := github.New(ws.opts.GitHubClientOpts)

	repositories := make(map[string]Repository, len(c.Repositories))
	errors := make(chan error, len(c.Repositories))

	wg := &sync.WaitGroup{}
	for _, r := range c.Repositories {
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

			if !ws.opts.SkipWebhooksRegistration {
				if err = gh.RegisterWebhook(r.OriginURL); err != nil {
					errors <- fmt.Errorf("failed to register webhooks for %s: %s", r.OriginURL, err)
					return
				}
			}

			repositories[r.OriginURL.ToKey()] = repo
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

	ws.lock.Lock()
	defer ws.lock.Unlock()

	ws.callbackPath = callback.Path
	ws.config = c
	ws.repositories = repositories

	metrics.LastSuccessfulConfigApply.Set(float64(time.Now().Unix()))

	logrus.Infof("configuration loaded successfully")
	return nil
}

// Run starts the execution of the server, forever
func (ws *WebHooksServer) Run(address string) {
	ws.running = true
	http.HandleFunc(ws.callbackPath, ws.WebHookHandler)

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

// WebHookHandler handles a webhook request
func (ws *WebHooksServer) WebHookHandler(w http.ResponseWriter, r *http.Request) {
	if !ws.running {
		http.Error(w, "server is shutting down", http.StatusServiceUnavailable)
		return
	}

	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("only POST is allowed"), http.StatusBadRequest)
		return
	}

	metrics.HooksReceivedTotal.Inc()

	ws.wg.Add(1)
	defer ws.wg.Done()

	if err := r.ParseForm(); err != nil {
		logrus.Debugf("Failed to parse form on request %#v", r)
		http.Error(w, fmt.Sprintf("bad request: %s", err), http.StatusBadRequest)
		return
	}

	payload := r.FormValue("payload")
	if payload == "" {
		logrus.Debugf("No payload in form: %#v", r.Form)
		http.Error(w, "no payload in form", http.StatusBadRequest)
		return
	}

	hookPayload, err := github.ParseHookPayload(payload)
	if err != nil {
		logrus.Debugf("Failed to parse hook payload: %s - %s", err, payload)
		http.Error(w, fmt.Sprintf("bad request: %s", err), http.StatusBadRequest)
		return
	}

	ws.lock.Lock()
	defer ws.lock.Unlock()

	repo, ok := ws.repositories[hookPayload.Repository.FullName]
	if !ok {
		http.Error(w, fmt.Sprintf("unknown repo %s", hookPayload.Repository.FullName), http.StatusNotFound)
		return
	}

	metrics.HooksAcceptedTotal.WithLabelValues(hookPayload.Repository.FullName).Inc()

	ws.wg.Add(1)
	go ws.updateRepository(repo)

	w.WriteHeader(http.StatusAccepted)
}

// UpdateAll triggers an update for all the repositories
func (ws *WebHooksServer) UpdateAll() {
	for _, repo := range ws.repositories {
		ws.wg.Add(1)
		go ws.updateRepository(repo)
	}
}

func (ws *WebHooksServer) updateRepository(repo Repository) {
	defer ws.wg.Done()

	startFetch := time.Now()
	if err := repo.Fetch(); err != nil {
		logrus.Errorf("failed to fetch repo %s: %s", repo.origin, err)
		metrics.HooksFailedTotal.WithLabelValues(repo.origin.ToPath()).Inc()
		return
	}
	metrics.GitLatencySecondsTotal.WithLabelValues("fetch", repo.origin.ToPath()).Observe(((time.Now().Sub(startFetch)).Seconds()))
	metrics.HooksUpdatedTotal.WithLabelValues(repo.origin.ToPath()).Inc()

	startPush := time.Now()
	if err := repo.Push(); err != nil {
		logrus.Errorf("failed to push repo %s to %s: %s", repo.origin, repo.target, err)
		metrics.HooksFailedTotal.WithLabelValues(repo.target.ToPath()).Inc()
		return
	}
	metrics.GitLatencySecondsTotal.WithLabelValues("push", repo.target.ToPath()).Observe(((time.Now().Sub(startPush)).Seconds()))
	metrics.HooksUpdatedTotal.WithLabelValues(repo.target.ToPath()).Inc()

	logrus.Debugf("updated repository %s in %s", repo.origin, repo.target)
}
