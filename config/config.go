package config

import (
	"fmt"
	"io/ioutil"

	"gitlab.com/yakshaving.art/git-pull-mirror/url"
	yaml "gopkg.in/yaml.v2"
)

// Config holds the configuration of the application
type Config struct {
	Repostitories []RepositoryConfig `yaml:"repositories"`
}

// RepositoryConfig holds the repository origin url, git origin parsing and
// target url
type RepositoryConfig struct {
	Origin    string `yaml:"origin"`
	OriginURL url.GitURL

	Target string `yaml:"target"`
}

// LoadConfiguration loads the file and parses the origin url, returns a
// configuration if everything checks up, an error in case of any failure.
func LoadConfiguration(filename string) (Config, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("failed reading configuration file %s: %s", filename, err)
	}

	c := Config{}
	if err = yaml.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("failed to parse configuration file %s: %s", filename, err)
	}
	for i, repo := range c.Repostitories {
		origin, err := url.Parse(repo.Origin)
		if err != nil {
			return c, fmt.Errorf("failed to parse origin url %s: %s", repo.Origin, err)
		}
		c.Repostitories[i].OriginURL = origin
	}

	return c, nil
}
