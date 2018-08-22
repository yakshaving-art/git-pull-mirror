package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	httpurl "net/url"
	"os"
	"strings"
	"testing"

	"gitlab.com/yakshaving.art/git-pull-mirror/config"
	"gitlab.com/yakshaving.art/git-pull-mirror/github"
	"gitlab.com/yakshaving.art/git-pull-mirror/url"
	git "gopkg.in/src-d/go-git.v4"
)

func TestBuildingAServerAndConfigureWithEmptyConfigWorks(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "server_test")
	must(t, "could not create a temporary dir", err)

	defer os.RemoveAll(tmpDir)

	client, err := github.New(github.ClientOpts{
		CallbackURL: "http://myhostname/mypath",
		GitHubURL:   "http://localhost",
		Token:       "mytoken",
		User:        "myuser",
	})
	s := New(client, WebHooksServerOptions{
		GitTimeoutSeconds: 10,
		RepositoriesPath:  tmpDir,
	})
	originURL, err := url.Parse("https://github.com/yakshaving-art/git-pull-mirror.git")
	must(t, "could not parse origin url", err)

	targetURL := url.GitURL{
		Domain:    "gitlab.com",
		Name:      "git-pull-mirror",
		Owner:     "yakshaving.art",
		URI:       "file://" + tmpDir + "/target/gitlab.com/yakshaving.art/git-pull-mirror",
		Transport: "file",
	}

	_, err = git.PlainInit(tmpDir+"/gitlab.com/yakshaving.art/git-pull-mirror", true)
	must(t, "failed to plain init target repo", err)

	c := make(chan interface{})
	go func() {
		s.Run(":9092", config.Config{
			Repositories: []config.RepositoryConfig{
				{
					Origin: originURL.URI, OriginURL: originURL,
					Target: targetURL.URI, TargetURL: targetURL,
				},
			},
		}, c)
	}()
	<-c
	defer s.Shutdown()

	tt := []struct {
		name string
		test func(*testing.T)
	}{
		{
			"Correct webhook invocation",
			func(t *testing.T) {
				res, err := runWebhook(s.callbackPath, "yakshaving-art/git-pull-mirror")
				must(t, "failed to execute webhooks 1", err)

				if res.Status != "202 Accepted" {
					t.Fatalf("Unexpected status code %s", res.Status)
				}
			},
		},
		{
			"Invalid repo name",
			func(t *testing.T) {
				res, err := runWebhook(s.callbackPath, "yakshaving-art")
				must(t, "failed to execute webhooks 2", err)

				if res.Status != "404 Not Found" {
					t.Fatalf("Unexpected status code %s", res.Status)
				}
			},
		},
		{
			"Update all",
			func(t *testing.T) {
				s.UpdateAll()
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			tc.test(t)
		})
	}

}

func must(t *testing.T, desc string, err error) {
	if err != nil {
		t.Fatalf("%s, got error %s", desc, err)
	}
}

func runWebhook(path, fullname string) (*http.Response, error) {
	b, err := json.Marshal(github.HookPayload{
		Repository: github.Repository{
			FullName: fullname,
		},
		Hook: github.Hook{
			Events: []string{"push"},
		},
	})
	if err != nil {
		return nil, err
	}

	form := httpurl.Values{}
	form.Add("payload", string(b))

	serverURL := "http://localhost:9092" + path

	req, err := http.NewRequest("POST", serverURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}

	return http.DefaultClient.Do(req)
}
