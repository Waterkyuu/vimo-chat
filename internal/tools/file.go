package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// File Read Tool
type FileReadTool struct{}

type FileReadInput struct {
	Path   string `json:"path"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

func (t *FileReadTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "file_read",
		Desc: `Read the contents of a file at the given path.
		Returns the file content with line numbers.
		Supports reading a specific range of lines using offset and limit parameters.
		Use this to inspect source code, configuration files, logs, or any text file.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "The absolute path to the file to read",
				Required: true,
			},
			"offset": {
				Type:     schema.Integer,
				Desc:     "The line number to start reading from (1-indexed). Defaults to 1",
				Required: false,
			},
			"limit": {
				Type:     schema.Integer,
				Desc:     "Maximum number of lines to return. Defaults to 2000",
				Required: false,
			},
		}),
	}, nil
}

func (t *FileReadTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args FileReadInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse file_read arguments: %w", err)
	}

	if args.Path == "" {
		return "Error: path is required", nil
	}

	data, err := os.ReadFile(args.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file '%s': %w", args.Path, err)
	}

	lines := strings.Split(string(data), "\n")

	offset := args.Offset
	if offset < 1 {
		offset = 1
	}
	if offset > len(lines) {
		offset = len(lines)
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 2000
	}

	end := offset + limit
	if end > len(lines)+1 {
		end = len(lines) + 1
	}

	var sb strings.Builder
	for i := offset - 1; i < end-1; i++ {
		fmt.Fprintf(&sb, "%d: %s\n", i+1, lines[i])
	}

	result := sb.String()
	if result == "" {
		return "(empty file)", nil
	}
	return result, nil
}

var _ tool.InvokableTool = &FileReadTool{}

// File Write Tool
type FileWriteTool struct{}

type FileWriteInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Append  bool   `json:"append"`
}

func (t *FileWriteTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "file_write",
		Desc: `Write content to a file at the given path.
		Creates the file if it does not exist; overwrites the file if it already exists.
		Set append=true to append content to the end of an existing file instead of overwriting.
		Parent directories will be created automatically if they do not exist.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "The absolute path to the file to write",
				Required: true,
			},
			"content": {
				Type:     schema.String,
				Desc:     "The content to write to the file",
				Required: true,
			},
			"append": {
				Type:     schema.Boolean,
				Desc:     "If true, append to the file instead of overwriting. Defaults to false",
				Required: false,
			},
		}),
	}, nil
}

func (t *FileWriteTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args FileWriteInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse file_write arguments: %w", err)
	}

	if args.Path == "" {
		return "Error: path is required", nil
	}

	dir := filepath.Dir(args.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create parent directories for '%s': %w", args.Path, err)
	}

	var flag int
	if args.Append {
		flag = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	} else {
		flag = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}

	f, err := os.OpenFile(args.Path, flag, 0o644)
	if err != nil {
		return "", fmt.Errorf("failed to open file '%s': %w", args.Path, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := f.WriteString(args.Content); err != nil {
		return "", fmt.Errorf("failed to write to file '%s': %w", args.Path, err)
	}

	action := "written"
	if args.Append {
		action = "appended"
	}

	return fmt.Sprintf("Successfully %s %d bytes to '%s'", action, len(args.Content), args.Path), nil
}

var _ tool.InvokableTool = &FileWriteTool{}

// --- List Directory ---

type ListDirTool struct{}

type ListDirInput struct {
	Path string `json:"path"`
}

func (t *ListDirTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "list_dir",
		Desc: `List the contents of a directory.
		Returns one entry per line, with a trailing '/' for subdirectories.
		Use this to explore the project structure and find files.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "The absolute path to the directory to list. Defaults to the current working directory",
				Required: true,
			},
		}),
	}, nil
}

func (t *ListDirTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args ListDirInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse list_dir arguments: %w", err)
	}

	dir := args.Path
	if dir == "" {
		dir = "."
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory '%s': %w", dir, err)
	}

	var sb strings.Builder
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		info, err := entry.Info()
		if err == nil {
			fmt.Fprintf(&sb, "%s\t%s\n", name, humanSize(info.Size()))
		} else {
			var fi fs.FileInfo
			_ = fi
			sb.WriteString(name + "\n")
		}
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		return "(empty directory)", nil
	}
	return result, nil
}

var _ tool.InvokableTool = &ListDirTool{}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
