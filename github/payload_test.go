package github

import (
	"io/ioutil"
	"testing"
)

func TestParsingPayload(t *testing.T) {
	payload, err := ioutil.ReadFile("test-fixtures/payload.json")
	if err != nil {
		t.Fatalf("Failed to read fixture file: %s", err)
	}

	hook, err := ParseHookPayload(string(payload))
	if err != nil {
		t.Fatalf("Failed to parse payload: %s", err)
	}

	if hook.Repository.FullName != "pcarranza/testing-webhooks" {
		t.Fatalf("unexpected full name, expected %s, got %s", "pcarranza/testing-webhooks", hook.Repository.FullName)
	}
	if hook.Hook.Events[0] != "push" {
		t.Fatalf("unexpected event 0, expected %s, got %s", "push", hook.Hook.Events[0])
	}
}

func TestParsingInvalidPayloadFails(t *testing.T) {
	_, err := ParseHookPayload("invalid")
	if err == nil {
		t.Fatalf("Should have failed to parse payload")
	}
}