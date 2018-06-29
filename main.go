package main

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	// yaml "gopkg.in/yaml.v2"
	"flag"
)

var (
	address             = flag.String("listen.address", "localhost:9092", "address in which to listen for webhooks")
	configFile          = flag.String("conf", "mirrors.yaml", "configuration file")
	skipWebhookCreation = flag.Bool("skip.webhooks.creation", false, "don't create webhooks after loading the configuration")
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{})
}

func main() {
	flag.Parse()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGUSR1)

	loadConfiguration(*configFile)
	createWebHooks(*skipWebhookCreation)

	server := NewWebHooksServer()
	go server.Run(*address)

	for sig := range signalCh {
		switch sig {
		case syscall.SIGHUP:
			logrus.Info("Reloading the configuration")
			loadConfiguration(*configFile)
			createWebHooks(*skipWebhookCreation)

		case syscall.SIGUSR1:
			logrus.Infof("Printing running configuration")

		case syscall.SIGINT:
			logrus.Info("Shutting down gracefully")
			server.Shutdown()
			os.Exit(0)
		}
	}
}

func loadConfiguration(confgFile string) {

}

func createWebHooks(skipWebhookCreation bool) {
	if skipWebhookCreation {
		logrus.Infof("Skipping creationg of webhooks")
		return
	}

}

type WebHooksServer struct {
	wg      *sync.WaitGroup
	running bool
}

func NewWebHooksServer() *WebHooksServer {
	return &WebHooksServer{
		wg: &sync.WaitGroup{},
	}
}

func (ws WebHooksServer) Run(address string) {
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

func (ws WebHooksServer) Shutdown() {
	ws.running = false
	// Wait for all the ongoing requests to finish
	ws.wg.Wait()
	logrus.Infof("Server stopped")
}
