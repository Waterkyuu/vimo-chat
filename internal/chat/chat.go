package llm

import (
	"github.com/cloudwego/eino-ext/components/model/openai"
)

// ChatModelProvider defines the interface for getting chat model
type ChatModelProvider interface {
	GetChatModel() *openai.ChatModel
}

// chatModelProvider implements ChatModelProvider
type chatModelProvider struct {
	chatModel *openai.ChatModel
}

// // NewChatModelProvider creates a new ChatModelProvider
// func NewChatModelProvider(ctx context.Context) (ChatModelProvider, error) {
// 	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
// 		BaseURL: cfg.AI.DefaultUrl,
// 		APIKey:  cfg.AI.DefaultAPIKey,
// 		Model:   cfg.AI.DefaultModel,
// 	})

// 	if err != nil {
// 		return nil, err
// 	}

// 	return &chatModelProvider{chatModel: chatModel}, nil
// }

// // GetChatModel returns the chat model
// func (p *chatModelProvider) GetChatModel() *openai.ChatModel {
// 	return p.chatModel
// }