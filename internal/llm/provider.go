package llm

import "vimo-chat/internal/config"

type ProviderInfo struct {
	ID      config.Provider
	Name    string
	BaseURL string
}

func providerInfo(provider config.Provider) (ProviderInfo, bool) {
	switch provider {
	case config.ProviderOpenAI:
		return openAIProvider(), true
	case config.ProviderZAI:
		return zaiProvider(), true
	case config.ProviderZAICodingPlan:
		return zaiCodingPlanProvider(), true
	case config.ProviderDeepseek:
		return deepseekProvider(), true
	case config.ProviderKimi:
		return kimiProvider(), true
	case config.ProviderMiniMax:
		return miniMaxProvider(), true
	default:
		return ProviderInfo{}, false
	}
}
