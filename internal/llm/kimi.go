package llm

import "vimo-chat/internal/config"

// kimiProvider configures the Moonshot Kimi API.
//
//	International: https://api.moonshot.ai/v1
//	China:         https://api.moonshot.cn/v1
func kimiProvider() ProviderInfo {
	return ProviderInfo{
		ID:      config.ProviderKimi,
		Name:    "Kimi",
		BaseURL: "https://api.moonshot.ai/v1",
	}
}
