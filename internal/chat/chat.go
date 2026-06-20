package chat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"vimo-chat/internal/instructions"
	"vimo-chat/internal/memory"
	"vimo-chat/internal/skill"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type ChatService struct {
	model     *openai.ChatModel
	sysPrompt string
	tools     []tool.BaseTool
	memSvc    *memory.Service
}

func NewChatService(model *openai.ChatModel, tools []tool.BaseTool, memSvc *memory.Service) *ChatService {
	return &ChatService{
		model:     model,
		sysPrompt: skill.BuildSysPrompt(),
		tools:     tools,
		memSvc:    memSvc,
	}
}

// No Stream
func (cs *ChatService) Generate(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	reActAgent, newMessages, err := cs.buildAgent(ctx, messages)
	if err != nil {
		return nil, err
	}

	reply, err := reActAgent.Generate(ctx, newMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	return reply, nil
}

// Stream without middle callback
func (cs *ChatService) Stream(
	ctx context.Context,
	messages []*schema.Message,
) (*schema.StreamReader[*schema.Message], error) {

	reActAgent, newMessages, err := cs.buildAgent(ctx, messages)
	if err != nil {
		return nil, err
	}

	reply, err := reActAgent.Stream(ctx, newMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to stream content: %w", err)
	}

	return reply, nil

}

// Stream with middle callback
func (cs *ChatService) StreamEvents(ctx context.Context, messages []*schema.Message) (<-chan Event, error) {
	events := make(chan Event, 32)

	reActAgent, newMessages, err := cs.buildAgent(ctx, messages)

	if err != nil {
		close(events)
		return nil, err
	}

	callback := react.BuildAgentCallback(
		createModelCallbackHandler(),
		createToolCallbackHandler(events),
	)

	stream, err := reActAgent.Stream(
		ctx,
		newMessages,
		agent.WithComposeOptions(compose.WithCallbacks(callback)),
	)

	if err != nil {
		close(events)
		return nil, fmt.Errorf("failed to stream content: %w", err)
	}

	go func() {
		defer close(events)
		defer stream.Close()

		for {
			msg, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				events <- Event{Type: EventDone}
				return
			}

			if err != nil {
				events <- Event{
					Type: EventError,
					Err:  err,
				}
				return
			}

			if msg.Content == "" {
				continue
			}

			events <- Event{
				Type:    EventAssistantDelta,
				Content: msg.Content,
			}
		}
	}()

	return events, nil
}

func (cs *ChatService) buildAgent(
	ctx context.Context,
	messages []*schema.Message,
) (*react.Agent, []*schema.Message, error) {

	toolsNode := compose.ToolsNodeConfig{
		Tools: cs.tools,
	}

	reActAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel:      cs.model,
		ToolsConfig:           toolsNode,
		MaxStep:               30,
		StreamToolCallChecker: streamToolCallChecker,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create react agent: %w", err)
	}

	sysContent := cs.sysPrompt

	// Inject project/global rule files (AGENTS.md and aliases) so the agent
	// follows repository-specific conventions.
	if rules := instructions.Load(); rules != "" {
		sysContent += "\n\n" + rules
	}

	// If there is a memory service, spell the memory into the system prompt words
	if cs.memSvc != nil {
		memCtx, err := cs.memSvc.GetContextForPrompt(ctx)
		if err != nil {
			log.Printf("failed to get memory context: %v", err)
		} else if memCtx != "" {
			sysContent += "\n\n" + memCtx
		}
	}

	sysMessages := []*schema.Message{
		{
			Role:    schema.System,
			Content: sysContent,
		},
	}

	newMessages := append(sysMessages, messages...)
	return reActAgent, newMessages, nil
}

func streamToolCallChecker(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
	defer sr.Close()
	for {
		msg, err := sr.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// finish
				break
			}

			return false, err
		}

		if len(msg.ToolCalls) > 0 {
			return true, nil
		}
	}
	return false, nil
}
