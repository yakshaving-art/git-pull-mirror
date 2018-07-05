package server

import (
	"gitlab.com/yakshaving.art/git-pull-mirror/config"
	"gitlab.com/yakshaving.art/git-pull-mirror/github"
	"gitlab.com/yakshaving.art/git-pull-mirror/url"
	"testing"
)

func TestBuildingAServerAndConfigureWithEmptyConfigWorks(t *testing.T) {
	s := New(WebHooksServerOptions{
		GitTimeoutSeconds:        60,
		RepositoriesPath:         "/tmp",
		SkipWebhooksRegistration: true,
		GitHubClientOpts: github.ClientOpts{
			CallbackURL: "https://example.com/",
			Token:       "xxx",
			User:        "user",
		},
	})
	originURL, err := url.Parse("https://github.com/yakshaving-art/git-pull-mirror.git")
	must(t, err)

	targetURL, err := url.Parse("https://gitlab.com/yakshaving.art/git-pull-mirror.git")
	must(t, err)

	if err := s.Configure(config.Config{
		Repositories: []config.RepositoryConfig{
			{
				Origin: originURL.URI, OriginURL: originURL,
				Target: targetURL.URI, TargetURL: targetURL,
			},
		},
	}); err != nil {
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

func must(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Got error %s", err)
	}
}
