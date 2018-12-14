package server

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/yakshaving.art/git-pull-mirror/config"
	"gitlab.com/yakshaving.art/git-pull-mirror/metrics"
	"gitlab.com/yakshaving.art/git-pull-mirror/webhooks"
)

// WebHooksServer is the server that will listen for webhooks calls and handle them
type WebHooksServer struct {
	wg   *sync.WaitGroup
	lock *sync.Mutex

	mux *http.ServeMux

	WebHooksClient webhooks.Client
	opts           WebHooksServerOptions
	config         config.Config
	repositories   map[string]Repository
	running        bool
	ready          bool
	callbackPath   string

	tasksCh chan pullTask
}

type pullTask struct {
	id   string
	repo Repository
}

// WebHooksServerOptions holds server configuration options
type WebHooksServerOptions struct {
	GitTimeoutSeconds uint64
	RepositoriesPath  string
	SSHPrivateKey     string
	Concurrency       int
}

// New returns a new unconfigured webhooks server
func New(client webhooks.Client, opts WebHooksServerOptions) *WebHooksServer {
	return &WebHooksServer{
		wg:             &sync.WaitGroup{},
		lock:           &sync.Mutex{},
		opts:           opts,
		WebHooksClient: client,
		tasksCh:        make(chan pullTask, opts.Concurrency),
	}
}

// Configure sets up the server
func (ws *WebHooksServer) Configure(c config.Config) error {
	logrus.Debug("loading configuration")

	g := newGitClient(ws.opts)

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
				metrics.RepoIsUp.WithLabelValues(r.OriginURL.ToPath()).Set(0)
				return
			}

			if err = repo.Fetch(); err != nil {
				errors <- fmt.Errorf("failed to fetch %s: %s", r.OriginURL, err)
				metrics.RepoIsUp.WithLabelValues(r.OriginURL.ToPath()).Set(0)
				return
			}

			if err = ws.WebHooksClient.RegisterWebhook(r.OriginURL); err != nil {
				// We're skipping these errors on purporse to allow the server to boot up even if webhooks fail
				logrus.Warnf("failed to register webhooks for %s: %s", r.OriginURL, err)
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

	ws.config = c
	ws.repositories = repositories
	ws.ready = true
	metrics.ServerIsUp.Set(1)

	metrics.LastSuccessfulConfigApply.Set(float64(time.Now().Unix()))

	logrus.Infof("configuration loaded successfully")
	return nil
}

// Run starts the execution of the server, forever
func (ws *WebHooksServer) Run(address string, c config.Config, ready chan interface{}) {
	logrus.Debugf("Booting up server")
	if err := ws.Configure(c); err != nil {
		logrus.Warnf("failed to configure server propertly: %s", err)
	}

	callback, err := url.ParseRequestURI(ws.WebHooksClient.GetCallbackURL())
	if err != nil {
		logrus.Fatalf("could not parse callback url %s: %s", ws.WebHooksClient.GetCallbackURL(), err)
	}
	ws.callbackPath = callback.Path

	// Launch as many worker goroutines as concurrency was declared
	for i := 0; i < ws.opts.Concurrency; i++ {
		go func() {
			for task := range ws.tasksCh {
				ws.updateRepository(task.id, task.repo)
			}
		}()
	}

	ws.mux = http.NewServeMux()
	ws.mux.HandleFunc(ws.callbackPath, ws.WebHookHandler)

	logrus.Infof("starting listener on %s", address)
	ws.running = true

	ready <- true
	if err := http.ListenAndServe(address, ws.mux); err != nil {
		logrus.Fatalf("failed to start http server: %s", err)
	}
}

// Shutdown performs a graceful shutdown of the webhooks server
func (ws *WebHooksServer) Shutdown() {
	ws.running = false

	// Wait for all the ongoing requests to finish
	ws.wg.Wait()

	// Close the channel so we don't leak goroutines
	close(ws.tasksCh)

	logrus.Infof("server stopped")
}

// WebHookHandler handles a webhook request
func (ws *WebHooksServer) WebHookHandler(w http.ResponseWriter, r *http.Request) {
	if !ws.running {
		http.Error(w, "server is shutting down", http.StatusServiceUnavailable)
		return
	}
	if !ws.ready {
		http.Error(w, "Server is not ready to receive requests", http.StatusServiceUnavailable)
		return
	}

	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("only POST is allowed"), http.StatusBadRequest)
		return
	}

	metrics.HooksReceivedTotal.Inc()

	id := uuid.NewUUID().String()
	logrus.Debugf("Received request %s from %s", id, r.RemoteAddr)

	ws.wg.Add(1)
	defer ws.wg.Done()

	if err := r.ParseForm(); err != nil {
		logrus.Debugf("Failed to parse form on request %s: %#v", id, r)
		http.Error(w, fmt.Sprintf("bad request: %s", err), http.StatusBadRequest)
		return
	}

	payload := r.FormValue("payload")
	if payload == "" {
		logrus.Debugf("No payload in form for request %s: %#v", id, r.Form)
		http.Error(w, "no payload in form", http.StatusBadRequest)
		return
	}

	client := ws.WebHooksClient
	hookPayload, err := client.ParseHookPayload(payload)
	if err != nil {
		logrus.Debugf("Failed to parse hook payload for request %s: %s - %s", id, err, payload)
		http.Error(w, fmt.Sprintf("bad request: %s", err), http.StatusBadRequest)
		return
	}

	ws.lock.Lock()
	defer ws.lock.Unlock()

	repo, ok := ws.repositories[hookPayload.GetRepository()]
	if !ok {
		http.Error(w, fmt.Sprintf("unknown repo %s", hookPayload.GetRepository()), http.StatusNotFound)
		return
	}

	metrics.HooksAcceptedTotal.WithLabelValues(hookPayload.GetRepository()).Inc()

	ws.wg.Add(1)
	ws.tasksCh <- pullTask{id: id, repo: repo}

	w.WriteHeader(http.StatusAccepted)
}

// UpdateAll triggers an update for all the repositories
func (ws *WebHooksServer) UpdateAll() {
	if !ws.ready {
		logrus.Warnf("Can't update all repos when the service is not ready")
		return
	}

	for _, repo := range ws.repositories {
		ws.wg.Add(1)
		ws.tasksCh <- pullTask{id: "USR2", repo: repo}
	}
}

func (ws *WebHooksServer) updateRepository(requestID string, repo Repository) {
	defer ws.wg.Done()

	startFetch := time.Now()
	if err := repo.Fetch(); err != nil {
		logrus.Errorf("failed to fetch repo %s for request %s: %s", repo.origin, requestID, err)
		metrics.HooksFailedTotal.WithLabelValues(repo.origin.ToPath()).Inc()
		metrics.RepoIsUp.WithLabelValues(repo.origin.ToPath()).Set(0)
		return
	}
	metrics.GitLatencySecondsTotal.WithLabelValues("fetch", repo.origin.ToPath()).Observe(((time.Now().Sub(startFetch)).Seconds()))
	metrics.HooksUpdatedTotal.WithLabelValues(repo.origin.ToPath()).Inc()
	metrics.RepoIsUp.WithLabelValues(repo.origin.ToPath()).Set(1)

	startPush := time.Now()
	if err := repo.Push(); err != nil {
		logrus.Errorf("failed to push repo %s to %s for request %s: %s", repo.origin, repo.target, requestID, err)
		metrics.HooksFailedTotal.WithLabelValues(repo.target.ToPath()).Inc()
		metrics.RepoIsUp.WithLabelValues(repo.target.ToPath()).Set(0)
		return
	}
	metrics.GitLatencySecondsTotal.WithLabelValues("push", repo.target.ToPath()).Observe(((time.Now().Sub(startPush)).Seconds()))
	metrics.HooksUpdatedTotal.WithLabelValues(repo.target.ToPath()).Inc()
	metrics.RepoIsUp.WithLabelValues(repo.target.ToPath()).Set(1)

	logrus.Debugf("repository %s pushed to %s for request %s", repo.origin, repo.target, requestID)
}
