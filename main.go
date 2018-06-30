package main

import (
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	// yaml "gopkg.in/yaml.v2"
	"flag"
)

var (
	address             = flag.String("listen.address", "localhost:9092", "address in which to listen for webhooks")
	configFile          = flag.String("config.file", "mirrors.yaml", "configuration file")
	skipWebhookCreation = flag.Bool("skip.webhooks.creation", false, "don't create webhooks after loading the configuration")
	repoPath            = flag.String("repostories.path", ".", "local path in which to store cloned repositories")
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
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
