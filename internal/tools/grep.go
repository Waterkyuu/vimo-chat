package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type GrepTool struct{}

type GrepInput struct {
	Pattern     string `json:"pattern"`
	Path        string `json:"path"`
	Include     string `json:"include"`
	IgnoreCase  bool   `json:"ignore_case"`
	ShowLineNum bool   `json:"show_line_number"`
	Context     int    `json:"context"`
	MaxResults  int    `json:"max_results"`
}

func (t *GrepTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "grep",
		Desc: `Search file contents using regular expressions. Supports full regex syntax (e.g. "log.*Error", "func\\s+\\w+").
Returns file paths and line numbers with matching content.
Use this to find function definitions, variable usages, error messages, or any text pattern across the codebase.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"pattern": {
				Type:     schema.String,
				Desc:     "The regular expression pattern to search for in file contents",
				Required: true,
			},
			"path": {
				Type:     schema.String,
				Desc:     "The directory to search in. Defaults to the current working directory if not specified",
				Required: false,
			},
			"include": {
				Type:     schema.String,
				Desc:     "File pattern to include in the search (e.g. \"*.go\", \"*.{ts,tsx}\", \"*.py\")",
				Required: false,
			},
			"ignore_case": {
				Type:     schema.Boolean,
				Desc:     "Whether to perform a case-insensitive search. Defaults to false",
				Required: false,
			},
			"show_line_number": {
				Type:     schema.Boolean,
				Desc:     "Whether to include line numbers in the output. Defaults to true",
				Required: false,
			},
			"context": {
				Type:     schema.Integer,
				Desc:     "Number of lines of context to show before and after each match. Default is 0",
				Required: false,
			},
			"max_results": {
				Type:     schema.Integer,
				Desc:     "Maximum number of matching lines to return. Default is 200",
				Required: false,
			},
		}),
	}, nil
}

func (t *GrepTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args GrepInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse grep arguments: %w", err)
	}

	if args.Pattern == "" {
		return "Error: pattern is required", nil
	}

	if args.MaxResults <= 0 {
		args.MaxResults = 200
	}

	rgArgs := []string{
		"--max-count", fmt.Sprintf("%d", args.MaxResults),
	}

	if args.IgnoreCase {
		rgArgs = append(rgArgs, "-i")
	}

	if !args.ShowLineNum {
		rgArgs = append(rgArgs, "-N")
	} else {
		rgArgs = append(rgArgs, "-n")
	}

	if args.Context > 0 {
		rgArgs = append(rgArgs, "-C", fmt.Sprintf("%d", args.Context))
	}

	if args.Include != "" {
		rgArgs = append(rgArgs, "--glob", args.Include)
	}

	rgArgs = append(rgArgs, args.Pattern)

	searchPath := "."
	if args.Path != "" {
		searchPath = args.Path
	}
	rgArgs = append(rgArgs, searchPath)

	rgPath, err := exec.LookPath("rg")
	if err != nil {
		out, err := grepFallback(args)
		if err != nil {
			return "", fmt.Errorf("neither rg nor grep available: %w", err)
		}
		return out, nil
	}

	cmd := exec.CommandContext(ctx, rgPath, rgArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "No matches found.", nil
		}
		return "", fmt.Errorf("rg execution failed: %w\n%s", err, string(out))
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		return "No matches found.", nil
	}

	return result, nil
}

func grepFallback(args GrepInput) (string, error) {
	grepArgs := []string{"-n"}

	if args.IgnoreCase {
		grepArgs = append(grepArgs, "-i")
	}

	if args.Context > 0 {
		grepArgs = append(grepArgs, "-C", fmt.Sprintf("%d", args.Context))
	}

	if args.Include != "" {
		grepArgs = append(grepArgs, "--include", args.Include)
	}

	grepArgs = append(grepArgs, "-r", args.Pattern)

	searchPath := "."
	if args.Path != "" {
		searchPath = args.Path
	}
	grepArgs = append(grepArgs, searchPath)

	cmd := exec.Command("grep", grepArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "No matches found.", nil
		}
		return "", fmt.Errorf("grep execution failed: %w\n%s", err, string(out))
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		return "No matches found.", nil
	}

	lines := strings.Split(result, "\n")
	if args.MaxResults > 0 && len(lines) > args.MaxResults {
		lines = lines[:args.MaxResults]
		result = strings.Join(lines, "\n")
	}

	return result, nil
}

// Compile check
var _ tool.InvokableTool = &GrepTool{}
