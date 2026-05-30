package llm

import (
	"vimo-chat/internal/config"
)

func deepseekProvider() ProviderInfo {
	return ProviderInfo{
		ID:      config.ProviderDeepseek,
		Name:    "Deepseek",
		BaseURL: "https://api.deepseek.com",
	}
}
