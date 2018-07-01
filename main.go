package main

import (
	"flag"
	"github.com/sirupsen/logrus"
	"gitlab.com/yakshaving.art/git-pull-mirror/config"
	"gitlab.com/yakshaving.art/git-pull-mirror/github"
	"gitlab.com/yakshaving.art/git-pull-mirror/server"
	"os"
	"os/signal"
	"syscall"
)

var (
	address        = flag.String("listen.address", "localhost:9092", "address in which to listen for webhooks")
	configFile     = flag.String("config.file", "mirrors.yml", "configuration file")
	callbackURL    = flag.String("callback.url", os.Getenv("CALLBACK_URL"), "callback url to report to github for webhooks, must include schema and domain")
	debug          = flag.Bool("debug", false, "enable debugging log level")
	dryrun         = flag.Bool("dryrun", false, "execute configuration loading, don't actually do anything")
	githubUser     = flag.String("github.user", os.Getenv("GITHUB_USER"), "github username, used to configure the webhooks through the API")
	githubToken    = flag.String("github.token", os.Getenv("GITHUB_TOKEN"), "github token, used as the password to configure the webhooks through the API")
	githubURL      = flag.String("github.url", "https://api.github.com/hub", "api url to register webhooks")
	repoPath       = flag.String("repositories.path", ".", "local path in which to store cloned repositories")
	sshkey         = flag.String("sshkey", os.Getenv("SSH_KEY"), "ssh key to use to identify to remotes")
	timeoutSeconds = flag.Int("git.timeout.seconds", 60, "git operations timeout in seconds")
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	flag.Parse()

	if *debug {
		toggleDebugLogLevel()
	}

	if _, err := os.Stat(*repoPath); err != nil {
		logrus.Fatalf("failed to stat local repositories path %s: %s", *repoPath, err)
	}

	c, err := config.LoadConfiguration(*configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %s", err)
	}

	s := server.New(server.WebHooksServerOptions{
		GitTimeoutSeconds: *timeoutSeconds,
		RepositoriesPath:  *repoPath,
		SSHPrivateKey:     *sshkey,
		GitHubClientOpts: github.ClientOpts{
			User:        *githubUser,
			Token:       *githubToken,
			GitHubURL:   *githubURL,
			CallbackURL: *callbackURL,
		},
	})

	if *dryrun {
		os.Exit(0)
	}

	if err := s.Configure(c); err != nil {
		logrus.Fatalf("Failed to configure webhooks server: %s", err)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGUSR1)

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
