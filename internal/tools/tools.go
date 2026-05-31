package tools

import (
	"context"
	"log"

	"vimo-chat/internal/mcp"

	"github.com/cloudwego/eino-ext/components/tool/sequentialthinking"
	"github.com/cloudwego/eino/components/tool"
)

func LoadLocalTools() []tool.BaseTool {

	localToolsList := []tool.BaseTool{
		&ShellTool{},
		&FileReadTool{},
		&FileWriteTool{},
		&ListDirTool{},
		&GrepTool{},
		&GitStatusTool{},
		&GitDiffTool{},
		&GitLogTool{},
		&LoadSkillTool{},
	}
	thinkTool, err := sequentialthinking.NewTool()
	if err != nil {
		log.Printf("failed to create sequential thinking tool: %v", err)
	}

	if thinkTool != nil {
		localToolsList = append(localToolsList, thinkTool)
	}

	return localToolsList
}

func LoadMCPTools(ctx context.Context, mcpManager *mcp.Manager) []tool.BaseTool {
	if mcpManager == nil {
		return nil
	}
	return mcpManager.LoadTools(ctx)
}

func LoadTools(ctx context.Context, mcpManager *mcp.Manager) []tool.BaseTool {
	tools := LoadLocalTools()
	tools = append(tools, LoadMCPTools(ctx, mcpManager)...)
	return tools
}
