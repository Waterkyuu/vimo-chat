package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Provider string

type Transport string

const (
	ProviderOpenAI   Provider = "openai"
	ProviderZAI      Provider = "zai"
	ProviderDeepseek Provider = "deepseek"

	TransportStdio          Transport = "stdio"
	TransportSSE            Transport = "sse"
	TransportStreamableHTTP Transport = "streamable_http"

	defaultConfigPerm = 0o600
)

type MCPServerConfig struct {
	Transport Transport         `json:"transport"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	URL       string            `json:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

type ProviderConfig struct {
	APIKey string `json:"api_key"`
	Model  string `json:"model"`
}

type Config struct {
	ActiveProvider Provider                    `json:"active_provider"`
	Providers      map[Provider]ProviderConfig `json:"providers"`
	MCPServers     map[string]MCPServerConfig  `json:"mcp_servers,omitempty"`
}

func Default() Config {
	return Config{
		ActiveProvider: ProviderOpenAI,
		Providers: map[Provider]ProviderConfig{
			ProviderOpenAI:   {Model: "gpt-5.4"},
			ProviderZAI:      {Model: "glm-5.1"},
			ProviderDeepseek: {Model: "deepseek-v4-pro"},
		},
	}
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vimo", "config.json"), nil
}

func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path) // #nosec G304 -- config path is selected by the caller or os.UserConfigDir.
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), err
	}

	cfg.mergeDefaults()
	return cfg, nil
}

func Save(path string, cfg Config) error {
	cfg.mergeDefaults()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), defaultConfigPerm)
}

func (c Config) Provider(provider Provider) ProviderConfig {
	if c.Providers == nil {
		return Default().Providers[provider]
	}
	if cfg, ok := c.Providers[provider]; ok {
		return cfg
	}
	return Default().Providers[provider]
}

func (c *Config) SetProviderConfig(provider Provider, providerConfig ProviderConfig) {
	if c.Providers == nil {
		c.Providers = map[Provider]ProviderConfig{}
	}
	c.Providers[provider] = providerConfig
}

func (c *Config) mergeDefaults() {
	defaults := Default()
	if !isKnownProvider(c.ActiveProvider) {
		c.ActiveProvider = defaults.ActiveProvider
	}
	if c.Providers == nil {
		c.Providers = map[Provider]ProviderConfig{}
	}
	for provider, defaultProviderConfig := range defaults.Providers {
		current := c.Providers[provider]
		if current.Model == "" {
			current.Model = defaultProviderConfig.Model
		}
		c.Providers[provider] = current
	}
}

func isKnownProvider(provider Provider) bool {
	switch provider {
	case ProviderOpenAI, ProviderZAI, ProviderDeepseek:
		return true
	default:
		return false
	}
}
