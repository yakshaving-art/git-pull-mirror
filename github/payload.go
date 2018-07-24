package github

import (
	"encoding/json"
	"fmt"
	"gitlab.com/yakshaving.art/git-pull-mirror/webhooks"
)

// Repository holds the repository information
type Repository struct {
	URL      string `json:"url"`
	FullName string `json:"full_name"`
}

// Hook holds the hook events
type Hook struct {
	Events []string `json:"events"`
}

// HookPayload hold the GitHub
type HookPayload struct {
	Repository Repository `json:"repository"`
	Hook       Hook       `json:"hook"`
}

// GetRepository implements webhook.HookPayload interface
func (h HookPayload) GetRepository() string {
	return h.Repository.FullName
}

// ParseHookPayload parses a payload string and returns the payload as a struct
func (c Client) ParseHookPayload(payload string) (webhooks.HookPayload, error) {
	var hookPayload HookPayload
	if err := json.Unmarshal([]byte(payload), &hookPayload); err != nil {
		return hookPayload, fmt.Errorf("could not parse hook payload: %s", err)
	}
	return hookPayload, nil
}
