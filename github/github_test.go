package github_test

import (
	"gitlab.com/yakshaving.art/git-pull-mirror/github"
	"gitlab.com/yakshaving.art/git-pull-mirror/url"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterWebhooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEquals(t, r.Method, "PATCH")
		assertEquals(t, r.URL.Path, "/")
		assertEquals(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")

		username, token, _ := r.BasicAuth()
		assertEquals(t, username, "myuser")
		assertEquals(t, token, "mytoken")

		must(t, r.ParseForm())

		assertEquals(t, r.FormValue("hub.mode"), "subscribe")
		assertEquals(t, r.FormValue("hub.topic"), "https://mygithosing/mygroup/myproject/events/push")
		assertEquals(t, r.FormValue("hub.callback"), "http://myhostname/mypath")
	}))

	client, err := github.New(github.ClientOpts{
		CallbackURL: "http://myhostname/mypath",
		GitHubURL:   server.URL,
		Token:       "mytoken",
		User:        "myuser",
	})
	if err != nil {
		t.Fatalf("Failed to create github client: %s", err)
	}

	u, _ := url.Parse("http://mygithosing/mygroup/myproject")
	must(t, client.RegisterWebhook(u))
}

func must(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Got error %s", err)
	}
}

func assertEquals(t *testing.T, expected, actual string) {
	if expected != actual {
		t.Fatalf("%s != %s", expected, actual)
	}
}
