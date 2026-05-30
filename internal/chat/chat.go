package llm

import (
	"github.com/cloudwego/eino-ext/components/model/openai"
)

// ChatModelProvider defines the interface for getting chat model
type ChatModelProvider interface {
	GetChatModel() *openai.ChatModel
}

func Chat() {

}
