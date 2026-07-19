package config

import "testing"

func TestFromEnvUsesDefaults(t *testing.T) {
	t.Setenv("FLEX_HOST", "")
	t.Setenv("FLEX_PORT", "")

	config, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv() returned an error: %v", err)
	}
	if config.Address() != "0.0.0.0:8080" {
		t.Fatalf("unexpected address: %s", config.Address())
	}
}

func TestFromEnvRejectsInvalidPort(t *testing.T) {
	t.Setenv("FLEX_PORT", "70000")

	if _, err := FromEnv(); err == nil {
		t.Fatal("FromEnv() should reject an invalid port")
	}
}
