package llm

import "vimo-chat/internal/config"

// zaiCodingPlanProvider configures the GLM Coding Plan endpoint.
//
// The Coding Plan uses a dedicated endpoint that is separate from the general
// ZAI/GLM API and only accepts Coding Plan API keys: a key bound to the general
// endpoint will be rejected (HTTP 401) here, and vice versa.
//
//	International (Z.AI): https://api.z.ai/api/coding/paas/v4
//	China (bigmodel):     https://open.bigmodel.cn/api/coding/paas/v4
func zaiCodingPlanProvider() ProviderInfo {
	return ProviderInfo{
		ID:      config.ProviderZAICodingPlan,
		Name:    "ZAI Coding Plan",
		BaseURL: "https://api.z.ai/api/coding/paas/v4",
	}
}
