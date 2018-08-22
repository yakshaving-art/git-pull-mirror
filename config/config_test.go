package config_test

import (
	"fmt"
	"testing"

	"gitlab.com/yakshaving.art/git-pull-mirror/config"
)

func TestLoadingValidConfiguration(t *testing.T) {
	c, err := config.LoadConfiguration("test-fixtures/valid-config.yml")
	if err != nil {
		t.Fatalf("Failed to load valid configuration: %s", err)
	}
	if len(c.Repositories) != 2 {
		t.Fatalf("Invalid configuration repositories length: expected 2, got %d", len(c.Repositories))
	}
	assertEquals(t, "https://github.com/yakshaving-art/git-pull-mirror.git", c.Repositories[0].Origin)
	assertEquals(t, "git@gitlab.com:yakshaving.art/git-pull-mirror.git", c.Repositories[0].Target)
	assertEquals(t, "https://user:password@github.com/group/user", c.Repositories[1].Origin)
	assertEquals(t, "git@gitlab.com:other-group/other-user", c.Repositories[1].Target)
}

func TestLoadingEmptyConfiguration(t *testing.T) {
	c, err := config.LoadConfiguration("test-fixtures/empty-config.yml")
	if err != nil {
		t.Fatalf("Failed to load valid configuration: %s", err)
	}
	if len(c.Repositories) != 0 {
		t.Fatalf("Invalid configuration repositories length: expected 0, got %d", len(c.Repositories))
	}
}

func TestLoadingInvalidConfiguration(t *testing.T) {
	tt := []struct {
		name     string
		filename string
		err      string
	}{
		{
			"unmarshable config",
			"test-fixtures/unmarshable-config.yml",
			"failed to parse configuration file test-fixtures/unmarshable-config.yml: yaml: line 3: mapping values are not allowed in this context",
		},
		{
			"non existing file",
			"test-fixtures/non-existing-config.yml",
			"failed reading configuration file test-fixtures/non-existing-config.yml: open test-fixtures/non-existing-config.yml: no such file or directory",
		},
		{
			"invalid config",
			"test-fixtures/invalid-config.yml",
			"failed to parse origin url https://github.com/yakshaving-art: Invalid URL",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_, err := config.LoadConfiguration(tc.filename)
			if err == nil {
				t.Fatal("Invalid configuration loading should have failed but didn't")
			}
			assertEquals(t, tc.err, err.Error())
		})
	}
}

func TestArguments(t *testing.T) {
	tt := []struct {
		name string
		args config.Arguments
		err  string
	}{
		{
			"empty arguments",
			config.Arguments{},
			"Config file is mandatory, please set it through the -config.file argument",
		},
		{
			"without callback url ",
			config.Arguments{
				ConfigFile: "/tmp",
			},
			"Callback URL is mandatory, please set it through the environment CALLBACK_URL variable or with -callback.url",
		},
		{
			"with an invalid callback url",
			config.Arguments{
				ConfigFile:  "/tmp",
				CallbackURL: "invalidurl",
			},
			"Invalid callback URL 'invalidurl': parse invalidurl: invalid URI for request",
		},
		{
			"with a valid callback url without a path",
			config.Arguments{
				ConfigFile:  "/tmp",
				CallbackURL: "http://valid.com",
			},
			"Invalid callback URL 'http://valid.com', it should include a path",
		},
		{
			"without a github user",
			config.Arguments{
				ConfigFile:  "/tmp",
				CallbackURL: "http://valid.com/somepath",
			},
			"GitHub user is mandatory, please set it through the environment GITHUB_USER variable or with -github.user",
		},
		{
			"without a github token",
			config.Arguments{
				ConfigFile:  "/tmp",
				CallbackURL: "http://valid.com/somepath",
				GithubUser:  "pullbot",
			},
			"GitHubToken user is mandatory, please set it through the environment GITHUB_TOKEN variable or with -github.token",
		},
		{
			"without a github url",
			config.Arguments{
				ConfigFile:  "/tmp",
				CallbackURL: "http://valid.com/somepath",
				GithubUser:  "pullbot",
				GithubToken: "sometoken",
			},
			"Invalid GitHub URL '': parse : empty url",
		},
		{
			"with an invalid github url",
			config.Arguments{
				ConfigFile:  "/tmp",
				CallbackURL: "http://valid.com/somepath",
				GithubUser:  "pullbot",
				GithubToken: "sometoken",
				GithubURL:   "invalid",
			},
			"Invalid GitHub URL 'invalid': parse invalid: invalid URI for request",
		},
		{
			"without a repositories path",
			config.Arguments{
				ConfigFile:  "/tmp",
				CallbackURL: "http://valid.com/somepath",
				GithubUser:  "pullbot",
				GithubToken: "sometoken",
				GithubURL:   "https://api.github.com/hub",
			},
			"Repositories path is not accessible: stat : no such file or directory",
		},
		{
			"with a repositories file, not a folder",
			config.Arguments{
				ConfigFile:       "/tmp",
				CallbackURL:      "http://valid.com/somepath",
				GithubUser:       "pullbot",
				GithubToken:      "sometoken",
				GithubURL:        "https://api.github.com/hub",
				RepositoriesPath: "/etc/hosts",
			},
			"Repositories path folder /etc/hosts it not a folder",
		},
		{
			"with an invalid timeout",
			config.Arguments{
				ConfigFile:       "/tmp",
				CallbackURL:      "http://valid.com/somepath",
				GithubUser:       "pullbot",
				GithubToken:      "sometoken",
				GithubURL:        "https://api.github.com/hub",
				RepositoriesPath: "/tmp",
			},
			"Invalid timeout seconds 0, it should be 1 or higher",
		},
		{
			"with an invalid ssh key",
			config.Arguments{
				ConfigFile:       "/tmp",
				CallbackURL:      "http://valid.com/somepath",
				GithubUser:       "pullbot",
				GithubToken:      "sometoken",
				GithubURL:        "https://api.github.com/hub",
				RepositoriesPath: "/tmp",
				TimeoutSeconds:   1,
				SSHKey:           "/tmp/non-existing-file-hopefully",
			},
			"SSH Key stat /tmp/non-existing-file-hopefully: no such file or directory is not accessible",
		},
		{
			"without an invalid timeout",
			config.Arguments{
				ConfigFile:       "/tmp",
				CallbackURL:      "http://valid.com/somepath",
				GithubUser:       "pullbot",
				GithubToken:      "sometoken",
				GithubURL:        "https://api.github.com/hub",
				RepositoriesPath: "/tmp",
				TimeoutSeconds:   1,
			},
			"%!s(<nil>)",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.args.Check()
			assertEquals(t, tc.err, fmt.Sprintf("%s", err))
		})
	}

}

func assertEquals(t *testing.T, expected, got string) {
	if expected != got {
		t.Fatalf("Expected %s, got %s", expected, got)
	}
}
