package main

import (
	"flag"
	"github.com/sirupsen/logrus"
	"gitlab.com/yakshaving.art/git-pull-mirror/config"
	"gitlab.com/yakshaving.art/git-pull-mirror/server"
	"os"
	"os/signal"
	"syscall"
)

var (
	address    = flag.String("listen.address", "localhost:9092", "address in which to listen for webhooks")
	configFile = flag.String("config.file", "mirrors.yml", "configuration file")
	dryrun     = flag.Bool("dryrun", false, "execute configuration loading, don't actually do anything")
	repoPath   = flag.String("repostories.path", ".", "local path in which to store cloned repositories")
	debug      = flag.Bool("debug", false, "enable debugging log level")
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	flag.Parse()

	if *debug {
		toggleDebugLogLevel()
	}

	c, err := config.LoadConfiguration(*configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %s", err)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGUSR1)

	s := server.New()
	s.Configure(c)
	go s.Run(*address)

	for sig := range signalCh {
		switch sig {
		case syscall.SIGHUP:
			logrus.Info("Reloading the configuration")
			c, err := config.LoadConfiguration(*configFile)
			if err != nil {
				logrus.Errorf("Failed to load configuration: %s", err)
				continue
			}
			s.Configure(c)

		case syscall.SIGUSR1:
			logrus.Info("toggling debug log level")
			toggleDebugLogLevel()

		case syscall.SIGINT:
			logrus.Info("Shutting down gracefully")
			s.Shutdown()
			os.Exit(0)
		}
	}
}

func toggleDebugLogLevel() {
	switch logrus.GetLevel() {
	case logrus.InfoLevel:
		logrus.SetLevel(logrus.DebugLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
}
