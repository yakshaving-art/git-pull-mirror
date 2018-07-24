package github

import (
	"fmt"
	"io/ioutil"
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
func New(opts ClientOpts) (Client, error) {
	c := Client{opts}
	if opts.User == "" {
		return c, fmt.Errorf("GitHub username is necessary for registering webhooks")
	}
	if opts.Token == "" {
		return c, fmt.Errorf("GitHub token is necessary for registering webhooks")
	}
	if opts.GitHubURL == "" {
		return c, fmt.Errorf("GitHub url is necessary for registering webhooks")
	}
	if opts.CallbackURL == "" {
		return c, fmt.Errorf("Callback url is necessary for registering webhooks")
	}
	return c, nil
}

// GetCallbackURL implements webhooks.Client interface
func (c Client) GetCallbackURL() string {
	return c.opts.CallbackURL
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
	if err != nil {
		return fmt.Errorf("webhook creation request failed hard: %s", err)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		logrus.Debugf("webhook for %s correctly registered", uri)
		return nil

	default:
		b, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()

		if err != nil {
			return fmt.Errorf("webhook creation request failed with status %d %s - failed to read body: %s", resp.StatusCode, resp.Status, err)
		}

		return fmt.Errorf("webhook creation request failed with status %d %s: %s", resp.StatusCode, resp.Status, string(b))
	}
}
