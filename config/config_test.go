package config_test

import (
	"gitlab.com/yakshaving.art/git-pull-mirror/config"
	"testing"
)

func TestLoadingValidConfiguration(t *testing.T) {
	c, err := config.LoadConfiguration("test-fixtures/valid-config.yml")
	if err != nil {
		t.Fatalf("Failed to load valid configuration: %s", err)
	}
	if len(c.Repostitories) != 2 {
		t.Fatalf("Invalid configuration repositories length: expected 2, got %d", len(c.Repostitories))
	}
	assertEquals(t, "https://github.com/yakshaving-art/git-pull-mirror.git", c.Repostitories[0].Origin)
	assertEquals(t, "git@gitlab.com:yakshaving.art/git-pull-mirror.git", c.Repostitories[0].Target)
	assertEquals(t, "https://user:password@github.com/group/user", c.Repostitories[1].Origin)
	assertEquals(t, "git@gitlab.com:other-group/other-user", c.Repostitories[1].Target)
}

func TestLoadingEmptyConfiguration(t *testing.T) {
	c, err := config.LoadConfiguration("test-fixtures/empty-config.yml")
	if err != nil {
		t.Fatalf("Failed to load valid configuration: %s", err)
	}
	if len(c.Repostitories) != 0 {
		t.Fatalf("Invalid configuration repositories length: expected 0, got %d", len(c.Repostitories))
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
			"failed to parse configuration file test-fixtures/unmarshable-config.yml: yaml: line 2: mapping values are not allowed in this context",
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

func assertEquals(t *testing.T, expected, got string) {
	if expected != got {
		t.Fatalf("Expected %s, got %s", expected, got)
	}
}
