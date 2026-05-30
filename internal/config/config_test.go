package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.ActiveProvider != ProviderOpenAI {
		t.Fatalf("ActiveProvider = %q, want %q", cfg.ActiveProvider, ProviderOpenAI)
	}
	defaults := Default()
	if got, want := cfg.Provider(ProviderOpenAI).Model, defaults.Provider(ProviderOpenAI).Model; got != want {
		t.Fatalf("OpenAI model = %q, want default", got)
	}
	if got, want := cfg.Provider(ProviderZAI).Model, defaults.Provider(ProviderZAI).Model; got != want {
		t.Fatalf("ZAI model = %q, want default", got)
	}
	if got, want := cfg.Provider(ProviderDeepseek).Model, defaults.Provider(ProviderDeepseek).Model; got != want {
		t.Fatalf("Deepseek model = %q, want default", got)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	cfg := Default()
	cfg.ActiveProvider = ProviderDeepseek
	cfg.Providers[ProviderOpenAI] = ProviderConfig{APIKey: "openai-key", Model: "gpt-4.1"}
	cfg.Providers[ProviderZAI] = ProviderConfig{APIKey: "zai-key", Model: "glm-4.5"}
	cfg.Providers[ProviderDeepseek] = ProviderConfig{APIKey: "deepseek-key", Model: "deepseek-reasoner"}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !containsAll(string(raw), "active_provider", "api_key", "deepseek-reasoner") {
		t.Fatalf("saved config does not use expected JSON shape:\n%s", raw)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.ActiveProvider != ProviderDeepseek {
		t.Fatalf("ActiveProvider = %q, want %q", loaded.ActiveProvider, ProviderDeepseek)
	}
	if got := loaded.Provider(ProviderOpenAI).APIKey; got != "openai-key" {
		t.Fatalf("OpenAI key = %q, want saved key", got)
	}
	if got := loaded.Provider(ProviderZAI).Model; got != "glm-4.5" {
		t.Fatalf("ZAI model = %q, want saved model", got)
	}
	if got := loaded.Provider(ProviderDeepseek).Model; got != "deepseek-reasoner" {
		t.Fatalf("Deepseek model = %q, want saved model", got)
	}
}

func TestLoadMergesMissingProviderDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(
		path,
		[]byte(`{"active_provider":"zai","providers":{"zai":{"api_key":"zai-key"}}}`),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got := cfg.Provider(ProviderZAI).APIKey; got != "zai-key" {
		t.Fatalf("ZAI key = %q, want saved key", got)
	}
	defaults := Default()
	if got, want := cfg.Provider(ProviderZAI).Model, defaults.Provider(ProviderZAI).Model; got != want {
		t.Fatalf("ZAI model = %q, want default merged", got)
	}
	if got, want := cfg.Provider(ProviderOpenAI).Model, defaults.Provider(ProviderOpenAI).Model; got != want {
		t.Fatalf("OpenAI model = %q, want default merged", got)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
