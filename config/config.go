package config

import (
	"fmt"
	"io/ioutil"
	neturl "net/url"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"gitlab.com/yakshaving.art/git-pull-mirror/url"
	yaml "gopkg.in/yaml.v2"
)

// Config holds the configuration of the application
type Config struct {
	Repositories []RepositoryConfig `yaml:"repositories"`
}

// RepositoryConfig holds the repository origin url, git origin parsing and
// target url
type RepositoryConfig struct {
	Origin    string `yaml:"origin"`
	OriginURL url.GitURL

	Target    string `yaml:"target"`
	TargetURL url.GitURL
}

// Arguments parsed through user provided flags
type Arguments struct {
	Address     string
	ConfigFile  string
	CallbackURL string
	Debug       bool

	GithubUser  string
	GithubToken string
	GithubURL   string

	WebhooksTarget   string
	RepositoriesPath string
	SSHKey           string
	TimeoutSeconds   uint64

	DryRun      bool
	ShowVersion bool

	Concurrency int
}

// LoadConfiguration loads the file and parses the origin url, returns a
// configuration if everything checks up, an error in case of any failure.
func LoadConfiguration(filename string) (Config, error) {
	logrus.Debugf("reading configuration file %s", filename)
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("failed reading configuration file %s: %s", filename, err)
	}

	c := Config{}
	if err = yaml.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("failed to parse configuration file %s: %s", filename, err)
	}

	for i, repo := range c.Repositories {
		origin, err := url.Parse(repo.Origin)
		if err != nil {
			return c, fmt.Errorf("failed to parse origin url %s: %s", repo.Origin, err)
		}
		c.Repositories[i].OriginURL = origin

		target, err := url.Parse(repo.Target)
		if err != nil {
			return c, fmt.Errorf("failed to parse target url %s: %s", repo.Target, err)
		}
		c.Repositories[i].TargetURL = target
	}

	return c, nil
}

// Check runs the arguments structure through a validation. It returns an error if the arguments are invalid.
func (a Arguments) Check() error {
	if strings.TrimSpace(a.ConfigFile) == "" {
		return fmt.Errorf("Config file is mandatory, please set it through the -config.file argument")
	}
	if strings.TrimSpace(a.CallbackURL) == "" {
		return fmt.Errorf("Callback URL is mandatory, please set it through the environment CALLBACK_URL variable or with -callback.url")
	}

	u, err := neturl.ParseRequestURI(a.CallbackURL)
	if err != nil {
		return fmt.Errorf("Invalid callback URL '%s': %s", a.CallbackURL, err)
	}
	if strings.TrimSpace(u.Scheme) == "" || strings.TrimSpace(u.Path) == "" || strings.TrimSpace(u.Host) == "" {
		return fmt.Errorf("Invalid callback URL '%s', it should include a path", a.CallbackURL)
	}

	if len(strings.TrimSpace(a.GithubUser)) == 0 {
		return fmt.Errorf("GitHub user is mandatory, please set it through the environment GITHUB_USER variable or with -github.user")
	}
	if len(strings.TrimSpace(a.GithubToken)) == 0 {
		return fmt.Errorf("GitHubToken user is mandatory, please set it through the environment GITHUB_TOKEN variable or with -github.token")
	}

	u, err = neturl.ParseRequestURI(a.GithubURL)
	if err != nil {
		return fmt.Errorf("Invalid GitHub URL '%s': %s", a.GithubURL, err)
	}

	f, err := os.Stat(a.RepositoriesPath)
	if err != nil {
		return fmt.Errorf("Repositories path is not accessible: %s", err)
	}
	if !f.IsDir() {
		return fmt.Errorf("Repositories path folder %s it not a folder", a.RepositoriesPath)
	}

	if strings.TrimSpace(a.SSHKey) != "" {
		if _, err := os.Stat(a.SSHKey); err != nil {
			return fmt.Errorf("SSH Key %s is not accessible", err)
		}
	}
	if a.TimeoutSeconds <= 0 {
		return fmt.Errorf("Invalid timeout seconds %d, it should be 1 or higher", a.TimeoutSeconds)
	}

	if a.Concurrency <= 0 {
		return fmt.Errorf("Invalid concurrency %d, it has to be 1 or higher", a.Concurrency)
	}

	return nil
}
