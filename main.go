package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
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

func main() {
	setupLogger()

	args := parseArgs()

	if args.ShowVersion {
		fmt.Printf("Version: %s Commit: %s Date: %s\n", version.Version, version.Commit, version.Date)
		os.Exit(0)
	}

	if args.Debug {
		toggleDebugLogLevel()
	}

	if err := args.Check(); err != nil {
		logrus.Fatalf("Cannot start, arguments are invalid: %s", err)
		os.Exit(1)
	}

	c, err := config.LoadConfiguration(args.ConfigFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %s", err)
	}

	client, err := createClient(args)
	if err != nil {
		logrus.Fatalf("Failed to create GitHub Webhooks client: %s", err)
	}

	if args.DryRun {
		os.Exit(0)
	}

	s := server.New(client, server.WebHooksServerOptions{
		GitTimeoutSeconds: args.TimeoutSeconds,
		RepositoriesPath:  args.RepositoriesPath,
		SSHPrivateKey:     args.SSHKey,
		Concurrency:       args.Concurrency,
	})

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGUSR1, syscall.SIGUSR2)

	ready := make(chan interface{})
	go s.Run(args.Address, c, ready)

	<-ready

	for sig := range signalCh {
		switch sig {
		case syscall.SIGHUP:
			logrus.Info("Reloading the configuration")
			c, err := config.LoadConfiguration(args.ConfigFile)
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

func parseArgs() config.Arguments {
	args := config.Arguments{}
	flag.StringVar(&args.Address, "listen.address", ":9092", "address in which to listen for webhooks")
	flag.StringVar(&args.ConfigFile, "config.file", "mirrors.yml", "configuration file")
	flag.StringVar(&args.CallbackURL, "callback.url", os.Getenv("CALLBACK_URL"), "callback url to report to github for webhooks, must include schema and domain")
	flag.BoolVar(&args.Debug, "debug", false, "enable debugging log level")
	flag.BoolVar(&args.DryRun, "dryrun", false, "execute configuration loading, don't actually do anything")
	flag.StringVar(&args.GithubUser, "github.user", os.Getenv("GITHUB_USER"), "github username, used to configure the webhooks through the API")
	flag.StringVar(&args.GithubToken, "github.token", os.Getenv("GITHUB_TOKEN"), "github token, used as the password to configure the webhooks through the API")
	flag.StringVar(&args.GithubURL, "github.url", "https://api.github.com/hub", "github api url to register webhooks")

	// flag.StringVar(&args.GitlabUser, "gitlab.user", os.Getenv("GITLAB_USER"), "gitlab username, used to configure the webhooks through the API")
	// flag.StringVar(&args.GitlabToken, "gitlab.token", os.Getenv("GITLAB_TOKEN"), "gitlab token, used as the password to configure the webhooks through the API")
	// flag.StringVar(&args.GitlabURL, "gitlab.url", "", "gitlab api url to register webhooks")

	flag.StringVar(&args.WebhooksTarget, "webhooks.target", "github", "used to define different kinds of webhooks clients, GitHub by default")
	flag.StringVar(&args.RepositoriesPath, "repositories.path", ".", "local path in which to store cloned repositories")
	flag.StringVar(&args.SSHKey, "sshkey", os.Getenv("SSH_KEY"), "ssh key to use to identify to remotes")
	flag.Uint64Var(&args.TimeoutSeconds, "git.timeout.seconds", 60, "git operations timeout in seconds")

	flag.BoolVar(&args.ShowVersion, "version", false, "print the version and exit")

	flag.IntVar(&args.Concurrency, "concurrency", 4, "how many background tasks to execute concurrently")

	flag.Parse()

	return args
}

func createClient(args config.Arguments) (webhooks.Client, error) {
	// TODO: based on webhooksTarget we should create a github client or something else
	return github.New(github.ClientOpts{
		User:        args.GithubUser,
		Token:       args.GithubToken,
		GitHubURL:   args.GithubURL,
		CallbackURL: args.CallbackURL,
	})
}

func setupLogger() {
	logrus.AddHook(filename.NewHook())
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}
