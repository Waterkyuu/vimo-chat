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

// Git Status
type GitStatusTool struct{}

type GitStatusInput struct {
	WorkDir string `json:"workdir"`
}

func (t *GitStatusTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "git_status",
		Desc: `Run 'git status' to see the current state of a git repository.
		Shows current branch, staged, unstaged, and untracked files.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"workdir": {
				Type:     schema.String,
				Desc:     "The working directory of the git repository. Defaults to the current working directory",
				Required: false,
			},
		}),
	}, nil
}

func (t *GitStatusTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args GitStatusInput
	_ = json.Unmarshal([]byte(argumentsInJSON), &args)

	out, err := runGit(ctx, args.WorkDir, "status", "--short", "--branch")
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(out)
	if result == "" {
		return "Working tree clean", nil
	}
	return result, nil
}

var _ tool.InvokableTool = &GitStatusTool{}

// Git diff
type GitDiffTool struct{}

type GitDiffInput struct {
	WorkDir  string `json:"workdir"`
	Staged   bool   `json:"staged"`
	FileName string `json:"file_name"`
}

func (t *GitDiffTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "git_diff",
		Desc: `Run 'git diff' to see changes in the working tree or staging area.
		Shows what has been modified but not yet committed. Use staged=true to see staged changes.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"workdir": {
				Type:     schema.String,
				Desc:     "The working directory of the git repository. Defaults to the current working directory",
				Required: false,
			},
			"staged": {
				Type:     schema.Boolean,
				Desc:     "If true, show staged changes (--cached). Defaults to false",
				Required: false,
			},
			"file_name": {
				Type:     schema.String,
				Desc:     "Optional specific file to diff. If omitted, shows all changes",
				Required: false,
			},
		}),
	}, nil
}

func (t *GitDiffTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args GitDiffInput
	_ = json.Unmarshal([]byte(argumentsInJSON), &args)

	gitArgs := []string{"diff"}
	if args.Staged {
		gitArgs = append(gitArgs, "--cached")
	}
	if args.FileName != "" {
		gitArgs = append(gitArgs, "--", args.FileName)
	}

	out, err := runGit(ctx, args.WorkDir, gitArgs...)
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(out)
	if result == "" {
		return "No changes detected.", nil
	}
	return result, nil
}

var _ tool.InvokableTool = &GitDiffTool{}

// Git log
type GitLogTool struct{}

type GitLogInput struct {
	WorkDir string `json:"workdir"`
	Count   int    `json:"count"`
	Oneline bool   `json:"oneline"`
	Branch  string `json:"branch"`
}

func (t *GitLogTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "git_log",
		Desc: `Run 'git log' to view commit history.
		Shows recent commits with author, date, and message.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"workdir": {
				Type:     schema.String,
				Desc:     "The working directory of the git repository. Defaults to the current working directory",
				Required: false,
			},
			"count": {
				Type:     schema.Integer,
				Desc:     "Number of commits to show. Default is 10",
				Required: false,
			},
			"oneline": {
				Type:     schema.Boolean,
				Desc:     "If true, show each commit on a single line. Defaults to false",
				Required: false,
			},
			"branch": {
				Type:     schema.String,
				Desc:     "Optional branch name to show logs for. Defaults to current branch",
				Required: false,
			},
		}),
	}, nil
}

func (t *GitLogTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args GitLogInput
	_ = json.Unmarshal([]byte(argumentsInJSON), &args)

	count := args.Count
	if count <= 0 {
		count = 10
	}

	gitArgs := []string{"log", fmt.Sprintf("-%d", count)}
	if args.Oneline {
		gitArgs = append(gitArgs, "--oneline")
	} else {
		gitArgs = append(gitArgs, "--format=%h %an <%ae> %s (%cr)")
	}
	if args.Branch != "" {
		gitArgs = append(gitArgs, args.Branch)
	}

	out, err := runGit(ctx, args.WorkDir, gitArgs...)
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(out)
	if result == "" {
		return "No commits found.", nil
	}
	return result, nil
}

var _ tool.InvokableTool = &GitLogTool{}

// helper
func runGit(ctx context.Context, workDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if workDir != "" {
		cmd.Dir = workDir
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w\n%s", strings.Join(args, " "), err, string(out))
	}
	return string(out), nil
}
