package llm

import (
	"vimo-chat/internal/config"
)

func openAIProvider() ProviderInfo {
	return ProviderInfo{
		ID:      config.ProviderOpenAI,
		Name:    "OpenAI",
		BaseURL: "",
	}
}
