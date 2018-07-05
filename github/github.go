package github

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	giturl "gitlab.com/yakshaving.art/git-pull-mirror/url"
)

// Client is a GitHub client
type Client struct {
	opts ClientOpts
}

// ClientOpts is used to store all the options
type ClientOpts struct {
	User        string
	Token       string
	GitHubURL   string
	CallbackURL string
}

// New creates a new Client
func New(opts ClientOpts) Client {
	return Client{
		opts: opts,
	}
}

// RegisterWebhook registers a new webhook
func (c Client) RegisterWebhook(uri giturl.GitURL) error {
	logrus.Debugf("registering webhook for %s", uri)

	form := url.Values{}
	form.Add("hub.mode", "subscribe")
	form.Add("hub.topic", fmt.Sprintf("https://%s/events/push", uri))
	form.Add("hub.callback", c.opts.CallbackURL)

	req, err := http.NewRequest("POST", c.opts.GitHubURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("could not create request for webhook: %s", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.opts.User, c.opts.Token)

	resp, err := http.DefaultClient.Do(req)
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		logrus.Debugf("webhook for %s correctly registered", uri)
		return nil
	default:
		return fmt.Errorf("webhook creation request failed with status %d: %s - %s", resp.StatusCode, resp.Status, err)
	}
}
