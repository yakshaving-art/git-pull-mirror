package server

import (
	"encoding/json"
	"fmt"
)

// HookPayload hold the GitHub
type HookPayload struct {
	Repository struct {
		URL      string `json:"url"`
		FullName string `json:"full_name"`
	} `json:"repository"`
	Hook struct {
		Events []string `json:"events"`
	} `json:"hook"`
}

func parsePayload(payload string) (HookPayload, error) {
	var hookPayload HookPayload
	if err := json.Unmarshal([]byte(payload), &hookPayload); err != nil {
		return hookPayload, fmt.Errorf("could not parse hook payload: %s", err)
	}
	return hookPayload, nil
}
