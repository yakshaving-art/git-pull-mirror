package webhooks

import (
	"gitlab.com/yakshaving.art/git-pull-mirror/url"
)

// HookPayload is the payload that is pushed by a hook
type HookPayload interface {
	GetRepository() string
}

// Client is a Webhooks client
type Client interface {
	RegisterWebhook(url.GitURL) error
	ParseHookPayload(payload string) (HookPayload, error)
	GetCallbackURL() string
}
