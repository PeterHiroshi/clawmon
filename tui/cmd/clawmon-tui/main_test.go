package main

import (
	"testing"
)

func TestVersionVariableExists(t *testing.T) {
	// version should be set (default is "dev" at compile time)
	if version == "" {
		t.Error("version variable should not be empty")
	}
}

func TestResolveDaemonURLDefault(t *testing.T) {
	// When no CLI URL is provided and no config file, should return default
	url := resolveDaemonURL("")
	if url == "" {
		t.Error("resolveDaemonURL should return a non-empty default URL")
	}
}

func TestResolveDaemonURLExplicit(t *testing.T) {
	url := resolveDaemonURL("http://localhost:1234")
	if url != "http://localhost:1234" {
		t.Errorf("expected explicit URL, got %s", url)
	}
}
