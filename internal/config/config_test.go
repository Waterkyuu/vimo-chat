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

func TestLoadWithMCPServers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	content := `{
		"active_provider": "openai",
		"providers": {},
		"mcp_servers": {
			"fs": {
				"transport": "stdio",
				"command": "npx",
				"args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
				"env": {"FOO": "bar"}
			},
			"remote": {
				"transport": "sse",
				"url": "http://localhost:8080/sse",
				"headers": {"Authorization": "Bearer token123"}
			},
			"v2": {
				"transport": "streamable_http",
				"url": "http://localhost:9090/mcp"
			}
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.MCPServers) != 3 {
		t.Fatalf("MCPServers count = %d, want 3", len(cfg.MCPServers))
	}

	fs := cfg.MCPServers["fs"]
	if fs.Transport != TransportStdio {
		t.Fatalf("fs transport = %q, want %q", fs.Transport, TransportStdio)
	}
	if fs.Command != "npx" {
		t.Fatalf("fs command = %q, want %q", fs.Command, "npx")
	}
	if len(fs.Args) != 3 || fs.Args[0] != "-y" {
		t.Fatalf("fs args = %v, unexpected", fs.Args)
	}
	if fs.Env["FOO"] != "bar" {
		t.Fatalf("fs env FOO = %q, want %q", fs.Env["FOO"], "bar")
	}

	remote := cfg.MCPServers["remote"]
	if remote.Transport != TransportSSE {
		t.Fatalf("remote transport = %q, want %q", remote.Transport, TransportSSE)
	}
	if remote.URL != "http://localhost:8080/sse" {
		t.Fatalf("remote url = %q, want http://localhost:8080/sse", remote.URL)
	}
	if remote.Headers["Authorization"] != "Bearer token123" {
		t.Fatalf("remote auth header missing")
	}

	v2 := cfg.MCPServers["v2"]
	if v2.Transport != TransportStreamableHTTP {
		t.Fatalf("v2 transport = %q, want %q", v2.Transport, TransportStreamableHTTP)
	}
	if v2.URL != "http://localhost:9090/mcp" {
		t.Fatalf("v2 url = %q, want http://localhost:9090/mcp", v2.URL)
	}
}

func TestSaveAndLoadMCPServersRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := Default()
	cfg.MCPServers = map[string]MCPServerConfig{
		"test": {
			Transport: TransportStdio,
			Command:   "my-mcp-server",
			Args:      []string{"--port", "3000"},
			Env:       map[string]string{"KEY": "val"},
		},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	s := loaded.MCPServers["test"]
	if s.Transport != TransportStdio {
		t.Fatalf("transport = %q, want %q", s.Transport, TransportStdio)
	}
	if s.Command != "my-mcp-server" {
		t.Fatalf("command = %q, want %q", s.Command, "my-mcp-server")
	}
	if len(s.Args) != 2 || s.Args[0] != "--port" {
		t.Fatalf("args = %v, unexpected", s.Args)
	}
	if s.Env["KEY"] != "val" {
		t.Fatalf("env KEY = %q, want %q", s.Env["KEY"], "val")
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
