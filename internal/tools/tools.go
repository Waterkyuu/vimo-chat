package tools

import (
	"context"
	"log"

	"vimo-chat/internal/mcp"
	"vimo-chat/internal/memory"

	"github.com/cloudwego/eino-ext/components/tool/sequentialthinking"
	"github.com/cloudwego/eino/components/tool"
)

func LoadLocalTools(memSvc *memory.Service) []tool.BaseTool {

	// Get memory tools
	memoryTools := loadMemoryTools(memSvc)

	// Get other local tools
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

	localToolsList = append(localToolsList, memoryTools...)

	thinkTool, err := sequentialthinking.NewTool()
	if err != nil {
		log.Printf("failed to create sequential thinking tool: %v", err)
	}

	if thinkTool != nil {
		localToolsList = append(localToolsList, thinkTool)
	}

	return localToolsList
}

// Load MCP tools
func LoadMCPTools(ctx context.Context, mcpManager *mcp.Manager) []tool.BaseTool {
	if mcpManager == nil {
		return nil
	}
	return mcpManager.LoadTools(ctx)
}

func loadMemoryTools(memSvc *memory.Service) []tool.BaseTool {
	if memSvc == nil {
		return nil
	}

	return []tool.BaseTool{
		&MemorySaveTool{svc: memSvc},
		&MemoryDeleteTool{svc: memSvc},
		&MemoryListTool{svc: memSvc},
	}
}

func LoadTools(ctx context.Context, memSvc *memory.Service, mcpManager *mcp.Manager) []tool.BaseTool {
	tools := LoadLocalTools(memSvc)
	tools = append(tools, LoadMCPTools(ctx, mcpManager)...)
	return tools
}
