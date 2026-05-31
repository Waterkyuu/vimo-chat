package chat

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	callbacksHelper "github.com/cloudwego/eino/utils/callbacks"
)

// Create a tool callback handler to obtain intermediate tool call results.
func createToolCallbackHandler(events chan<- Event) *callbacksHelper.ToolCallbackHandler {
	return &callbacksHelper.ToolCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *tool.CallbackInput) context.Context {
			events <- Event{
				Type:       EventToolStart,
				ToolCallID: compose.GetToolCallID(ctx),
				ToolName:   info.Name,
				Arguments:  input.ArgumentsInJSON,
			}

			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *tool.CallbackOutput) context.Context {
			events <- Event{
				Type:       EventToolResult,
				ToolCallID: compose.GetToolCallID(ctx),
				ToolName:   info.Name,
				Content:    output.Response,
			}

			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			events <- Event{
				Type:     EventError,
				ToolName: info.Name,
				Err:      err,
			}

			return ctx
		},
	}
}

// Create a model callback handler to obtain intermediate model generation events.
func createModelCallbackHandler() *callbacksHelper.ModelCallbackHandler {
	return &callbacksHelper.ModelCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *model.CallbackInput) context.Context {
			fmt.Printf("\n==========[Model Generation Started]==========\n")
			fmt.Printf("Input Message Count: %d\n", len(input.Messages))
			fmt.Printf("==============================================\n")
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
			fmt.Printf("\n==========[Model Generation Completed]==========\n")
			if output.TokenUsage != nil {
				fmt.Printf("Token Usage:\n")
				fmt.Printf("  - Prompt Tokens: %d\n", output.TokenUsage.PromptTokens)
				fmt.Printf("  - Completion Tokens: %d\n", output.TokenUsage.CompletionTokens)
				fmt.Printf("  - Total Tokens: %d\n", output.TokenUsage.TotalTokens)
			}
			fmt.Printf("================================================\n")
			return ctx
		},
	}
}
