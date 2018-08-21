package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "gitlab.com/yakshaving.art/git-pull-mirror/metrics"

	"github.com/onrik/logrus/filename"
	"github.com/sirupsen/logrus"

	"gitlab.com/yakshaving.art/git-pull-mirror/config"
	"gitlab.com/yakshaving.art/git-pull-mirror/github"
	"gitlab.com/yakshaving.art/git-pull-mirror/server"
	"gitlab.com/yakshaving.art/git-pull-mirror/version"
	"gitlab.com/yakshaving.art/git-pull-mirror/webhooks"
)

var (
	address     = flag.String("listen.address", ":9092", "address in which to listen for webhooks")
	configFile  = flag.String("config.file", "mirrors.yml", "configuration file")
	callbackURL = flag.String("callback.url", os.Getenv("CALLBACK_URL"), "callback url to report to github for webhooks, must include schema and domain")
	debug       = flag.Bool("debug", false, "enable debugging log level")
	dryrun      = flag.Bool("dryrun", false, "execute configuration loading, don't actually do anything")
	githubUser  = flag.String("github.user", os.Getenv("GITHUB_USER"), "github username, used to configure the webhooks through the API")
	githubToken = flag.String("github.token", os.Getenv("GITHUB_TOKEN"), "github token, used as the password to configure the webhooks through the API")
	githubURL   = flag.String("github.url", "https://api.github.com/hub", "github api url to register webhooks")
	// gitlabUser       = flag.String("gitlab.user", os.Getenv("GITLAB_USER"), "gitlab username, used to configure the webhooks through the API")
	// gitlabToken      = flag.String("gitlab.token", os.Getenv("GITLAB_TOKEN"), "gitlab token, used as the password to configure the webhooks through the API")
	// gitlabURL        = flag.String("gitlab.url", "", "gitlab api url to register webhooks")
	// webhooksTarget = flag.String("webhooks.target", "github", "Used to define different kinds of webhooks clients, GitHub by default")
	repoPath         = flag.String("repositories.path", ".", "local path in which to store cloned repositories")
	skipRegistration = flag.Bool("skip.webhooks.registration", false, "don't register webhooks")
	sshkey           = flag.String("sshkey", os.Getenv("SSH_KEY"), "ssh key to use to identify to remotes")
	timeoutSeconds   = flag.Int("git.timeout.seconds", 60, "git operations timeout in seconds")

	showVersion = flag.Bool("version", false, "print the version and exit")
)

func main() {
	logrus.AddHook(filename.NewHook())
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	flag.Parse()

	if *showVersion {
		fmt.Printf("Version: %s Commit: %s Date: %s\n", version.Version, version.Commit, version.Date)
		os.Exit(0)
	}

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

	client, err := createClient()
	if err != nil {
		logrus.Fatalf("Failed to create GitHub Webhooks client: %s", err)
	}

	s := server.New(client, server.WebHooksServerOptions{
		GitTimeoutSeconds:        *timeoutSeconds,
		RepositoriesPath:         *repoPath,
		SSHPrivateKey:            *sshkey,
		SkipWebhooksRegistration: *skipRegistration,
	})

	if err := s.Validate(); err != nil {
		logrus.Fatalf("Webhooks server failed to validate the configuration: %s", err)
	}

	if *dryrun {
		os.Exit(0)
	}

	if err := s.Configure(c); err != nil {
		logrus.Fatalf("Failed to configure webhooks server: %s", err)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGUSR1, syscall.SIGUSR2)

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

		case syscall.SIGUSR2:
			logrus.Info("Received USR2, forcing an update in all the repositories")
			s.UpdateAll()

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

func createClient() (webhooks.Client, error) {
	// TODO: based on webhooksTarget we should create a github client or something else
	return github.New(github.ClientOpts{
		User:        *githubUser,
		Token:       *githubToken,
		GitHubURL:   *githubURL,
		CallbackURL: *callbackURL,
	})
}
