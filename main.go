package main

import (
	"flag"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"gitlab.com/yakshaving.art/git-pull-mirror/config"
	"gitlab.com/yakshaving.art/git-pull-mirror/github"
	_ "gitlab.com/yakshaving.art/git-pull-mirror/metrics"
	"gitlab.com/yakshaving.art/git-pull-mirror/server"
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

	checkArgs()

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

// checkArgs ensures that all the arguments make some sense before trying to do anything at all.
func checkArgs() {
	if strings.TrimSpace(*configFile) == "" {
		logrus.Fatalf("Config file is mandatory, please set it through the -config.file argument")
	}
	if strings.TrimSpace(*callbackURL) == "" {
		logrus.Fatalf("Callback URL is mandatory, please set it through the environment CALLBACK_URL variable or with -callback.url")
	}
	if u, err := url.ParseRequestURI(*callbackURL); err != nil {
		logrus.Fatalf("Invalid callback URL '%s': %s", *callbackURL, err)
	} else {
		if strings.TrimSpace(u.Scheme) == "" || strings.TrimSpace(u.Path) == "" || strings.TrimSpace(u.Host) == "" {
			logrus.Fatalf("Invalid callback URL '%s', it should include a path", *callbackURL)
		}
	}
	if len(strings.TrimSpace(*githubUser)) == 0 {
		logrus.Fatalf("GitHub user is mandatory, please set it through the environment GITHUB_USER variable or with -github.user")
	}
	if len(strings.TrimSpace(*githubToken)) == 0 {
		logrus.Fatalf("GitToken user is mandatory, please set it through the environment GITHUB_TOKEN variable or with -github.token")
	}
	if _, err := url.Parse(*githubURL); err != nil {
		logrus.Fatalf("Invalid github URL '%s': %s", *githubURL, err)
	}

	if _, err := os.Stat(*repoPath); err != nil {
		logrus.Fatalf("Repositories path is not accessible: %s", err)
	}
	if strings.TrimSpace(*sshkey) != "" {
		if _, err := os.Stat(*sshkey); err != nil {
			logrus.Fatalf("SSH Key %s is not accessible", err)
		}
	}
	if *timeoutSeconds <= 0 {
		logrus.Fatalf("Invalid timeout seconds %d, it should be 1 or higher", *timeoutSeconds)
	}
}
