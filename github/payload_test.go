package github

import (
	"io/ioutil"
	"testing"
)

func TestParsingPayload(t *testing.T) {
	client, err := New(ClientOpts{
		CallbackURL: "http://myhostname/mypath",
		GitHubURL:   "http://localhost",
		Token:       "mytoken",
		User:        "myuser",
	})
	if err != nil {
		t.Fatalf("Failed to create github client: %s", err)
	}

	payload, err := ioutil.ReadFile("test-fixtures/payload.json")
	if err != nil {
		t.Fatalf("Failed to read fixture file: %s", err)
	}

	hook, err := client.ParseHookPayload(string(payload))
	if err != nil {
		t.Fatalf("Failed to parse payload: %s", err)
	}

	if hook.GetRepository() != "pcarranza/testing-webhooks" {
		t.Fatalf("unexpected full name, expected %s, got %s", "pcarranza/testing-webhooks", hook.GetRepository())
	}
}

func TestParsingInvalidPayloadFails(t *testing.T) {
	client, err := New(ClientOpts{
		CallbackURL: "http://myhostname/mypath",
		GitHubURL:   "http://localhost",
		Token:       "mytoken",
		User:        "myuser",
	})
	if err != nil {
		t.Fatalf("Failed to create github client: %s", err)
	}

	_, err = client.ParseHookPayload("invalid")
	if err == nil {
		t.Fatalf("Should have failed to parse payload")
	}
}
