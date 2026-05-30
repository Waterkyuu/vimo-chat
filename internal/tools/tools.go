package tools

import (
	"context"
	"log"

	"github.com/cloudwego/eino-ext/components/tool/sequentialthinking"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
)

// LoadLocalTools loads all local tools for the agent.
func LoadLocalTools(ctx context.Context) []tool.BaseTool {

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

// LoadMCPTools loads MCP tools from the given MCP client.
func LoadMCPTools(ctx context.Context, mcpClient *client.Client) []tool.BaseTool {
	// Note: To use MCP tools, you need to install the eino-ext MCP package:
	// go get github.com/cloudwego/eino-ext/components/tool/mcp
	//
	// Then use it as:
	// mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	// mcpTools, err := mcpp.GetTools(ctx, &mcpp.Config{Cli: mcpClient})

	log.Printf("MCP tools loading is not yet implemented. Please install eino-ext MCP package first.")
	return nil
}
