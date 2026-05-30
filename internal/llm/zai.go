package llm

import "vimo-chat/internal/config"

func zaiProvider() ProviderInfo {
	return ProviderInfo{
		ID:      config.ProviderZAI,
		Name:    "ZAI",
		BaseURL: "https://open.bigmodel.cn/api/paas/v4",
	}
}
