package llm

import "vimo-chat/internal/config"

// miniMaxProvider configures the MiniMax API.
//
//	International: https://api.minimax.io/v1
//	China:         https://api.minimaxi.com/v1
func miniMaxProvider() ProviderInfo {
	return ProviderInfo{
		ID:      config.ProviderMiniMax,
		Name:    "MiniMax",
		BaseURL: "https://api.minimax.io/v1",
	}
}
