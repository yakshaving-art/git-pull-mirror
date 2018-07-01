package server

import (
	"gitlab.com/yakshaving.art/git-pull-mirror/config"
	"gitlab.com/yakshaving.art/git-pull-mirror/github"
	"testing"
)

func TestBuildingAServerAndConfigureWithEmptyConfigWorks(t *testing.T) {
	s := New(WebHooksServerOptions{
		GitTimeoutSeconds: 60,
		RepositoriesPath:  "/tmp",
		GitHubClientOpts: github.ClientOpts{
			CallbackURL: "https://example.com/",
			Token:       "xxx",
			User:        "user",
		},
	})
	if err := s.Configure(config.Config{}); err != nil {
		t.Fatalf("Failed to configure server: %s", err)
	}

	c := make(chan bool)
	go func() {
		c <- true
		s.Run("localhost:9092")
	}()
	<-c

	s.Shutdown()
}
