package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"

	"vimo-chat/internal/config"
)

type ChatModelProvider interface {
	GetChatModel() *openai.ChatModel
}

// type chatModelProvider struct {
// 	chatModel *openai.ChatModel
// }

func NewChatModelProvider(ctx context.Context, cfg config.Config) (*openai.ChatModel, error) {

	provider := cfg.ActiveProvider

	info, ok := providerInfo(provider)

	if !ok {
		return nil, fmt.Errorf("unsupported provider %s", provider)
	}

	providerConfig := cfg.Provider(provider)

	if providerConfig.APIKey == "" {
		return nil, fmt.Errorf("%s api Key is missing", info.Name)
	}

	if providerConfig.Model == "" {
		return nil, fmt.Errorf("%s model is missing", info.Name)
	}

	modelConfig := &openai.ChatModelConfig{
		APIKey:  providerConfig.APIKey,
		Model:   providerConfig.Model,
		BaseURL: info.BaseURL,
		Timeout: 60 * time.Second,
	}

	return openai.NewChatModel(ctx, modelConfig)

}

// // GetChatModel returns the chat model
// func (p *chatModelProvider) GetChatModel() *openai.ChatModel {
// 	return p.chatModel
// }
