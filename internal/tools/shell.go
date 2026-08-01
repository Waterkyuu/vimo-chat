package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"runtime"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type ShellTool struct{}

type ShellInput struct {
	Command string `json:"command"`
	WorkDir string `json:"workdir"`
	Timeout int    `json:"timeout"`
}

func (t *ShellTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "shell",
		Desc: `Execute a shell command and return its output.
		Use this to run git, npm, go, docker, or any other CLI commands.
		Automatically uses cmd on Windows and sh on Unix-like systems.
		Supports specifying a working directory and timeout.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command": {
				Type:     schema.String,
				Desc:     "The shell command to execute",
				Required: true,
			},
			"workdir": {
				Type:     schema.String,
				Desc:     "The working directory for the command. Defaults to the current working directory",
				Required: false,
			},
			"timeout": {
				Type:     schema.Integer,
				Desc:     "Timeout in milliseconds. Default is 120000 (2 minutes). Maximum is 600000 (10 minutes)",
				Required: false,
			},
		}),
	}, nil
}

func (t *ShellTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args ShellInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse shell arguments: %w", err)
	}

	if args.Command == "" {
		return "Error: command is required", nil
	}

	timeout := time.Duration(args.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	if timeout > 600*time.Second {
		timeout = 600 * time.Second
	}

	workDir := args.WorkDir
	if workDir == "" {
		if dir, err := os.Getwd(); err == nil {
			workDir = dir
		}
	}

	var cmd *exec.Cmd
	shellCmd := formatShellCommand(args.Command)

	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", shellCmd)
	} else {
		shell := detectShell()
		cmd = exec.CommandContext(ctx, shell, "-c", shellCmd)
	}

	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	var runErr error
	select {
	case err := <-done:
		runErr = err
	case <-time.After(timeout):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return fmt.Sprintf("Command timed out after %v\n\nSTDOUT:\n%s\nSTDERR:\n%s",
			timeout, stdout.String(), stderr.String()), nil
	}

	out := stdout.String()
	errOut := stderr.String()

	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	var sb strings.Builder
	if exitCode != 0 {
		fmt.Fprintf(&sb, "Exit code: %d\n", exitCode)
	}
	if out != "" {
		sb.WriteString(out)
	}
	if errOut != "" {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("STDERR:\n")
		sb.WriteString(errOut)
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		if exitCode == 0 {
			return "(command completed with no output)", nil
		}
		return fmt.Sprintf("Exit code: %d", exitCode), nil
	}

	return result, nil
}

func formatShellCommand(command string) string {
	return strings.TrimSpace(command)
}

func detectShell() string {
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh
	}
	for _, candidate := range []string{"/bin/bash", "/bin/zsh", "/bin/sh"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "/bin/sh"
}

var _ tool.InvokableTool = &ShellTool{}
