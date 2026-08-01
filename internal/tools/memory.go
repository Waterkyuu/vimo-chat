package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"vimo-chat/internal/memory"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ---- Memory save tool ----
type MemorySaveTool struct {
	svc *memory.Service
}

type MemorySaveInput struct {
	Content string `json:"content"`
}

func (t *MemorySaveTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "memory_save",
		Desc: `Save an important fact about the user for future conversations.
        Use this when the user explicitly asks to remember something, or when you identify a persistent preference, habit, or important fact.
        Each memory should be a single, concise sentence.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"content": {
				Type:     schema.String,
				Desc:     "The fact to remember. Should be a single concise sentence.",
				Required: true,
			},
		}),
	}, nil
}

func (t *MemorySaveTool) InvokableRun(
	ctx context.Context,
	argumentsInJSON string,
	opts ...tool.Option,
) (string, error) {
	var args MemorySaveInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("parse memory_save args: %w", err)
	}

	m, err := t.svc.Remember(ctx, args.Content)
	if err != nil {
		return "", fmt.Errorf("save memory: %w", err)
	}
	return fmt.Sprintf("Memory saved: %s (id: %s)", m.Content, m.ID), nil
}

var _ tool.InvokableTool = &MemorySaveTool{}

// ---- Memory delete tool ----
type MemoryDeleteTool struct {
	svc *memory.Service
}

type MemoryDeleteInput struct {
	ID string `json:"id"`
}

func (t *MemoryDeleteTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "memory_delete",
		Desc: `Delete a saved memory by its ID.
        Use this when the user asks to forget or remove a previously saved memory.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"id": {
				Type:     schema.String,
				Desc:     "The ID of the memory to delete",
				Required: true,
			},
		}),
	}, nil
}

func (t *MemoryDeleteTool) InvokableRun(
	ctx context.Context,
	argumentsInJSON string,
	opts ...tool.Option,
) (string, error) {
	var args MemoryDeleteInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("parse memory_delete args: %w", err)
	}

	if err := t.svc.Forget(ctx, args.ID); err != nil {
		return "", fmt.Errorf("delete memory: %w", err)
	}
	return fmt.Sprintf("Memory deleted: %s", args.ID), nil
}

var _ tool.InvokableTool = &MemoryDeleteTool{}

// ---- Memory list tool ----
type MemoryListTool struct {
	svc *memory.Service
}

type MemoryListInput struct {
	Query string `json:"query,omitempty"`
}

func (t *MemoryListTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "memory_list",
		Desc: `List all saved memories, or search memories by keyword.
        Use this when the user asks what you remember about them.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "Optional keyword to search memories. If empty, returns all memories.",
				Required: false,
			},
		}),
	}, nil
}

func (t *MemoryListTool) InvokableRun(
	ctx context.Context,
	argumentsInJSON string,
	opts ...tool.Option,
) (string, error) {
	var args MemoryListInput
	_ = json.Unmarshal([]byte(argumentsInJSON), &args)

	var memories []*memory.Memory
	var err error

	if args.Query != "" {
		memories, err = t.svc.Search(ctx, args.Query)
	} else {
		memories, err = t.svc.ListAll(ctx)
	}

	if err != nil {
		return "", fmt.Errorf("list memories: %w", err)
	}

	if len(memories) == 0 {
		return "No memories found.", nil
	}

	var sb strings.Builder
	for _, m := range memories {
		sb.WriteString(fmt.Sprintf("- [%s] %s (id: %s)\n", m.Type, m.Content, m.ID))
	}
	return sb.String(), nil
}

var _ tool.InvokableTool = &MemoryListTool{}
