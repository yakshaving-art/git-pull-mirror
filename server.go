package main

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"sync"
)

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
