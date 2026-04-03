package commander

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/captain-duster/anwin-agent/internal/client"
)

const pollInterval = 3 * time.Second
const maxOutputBytes = 50000

type Commander struct {
	root        string
	anwinClient *client.AnwinClient
}

func New(root string, anwinClient *client.AnwinClient) *Commander {
	return &Commander{
		root:        root,
		anwinClient: anwinClient,
	}
}

func (c *Commander) Start() {
	fmt.Printf("  [%s] Command channel active — polling every %s\n", ts(), pollInterval)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		commands, err := c.anwinClient.PollCommands()
		if err != nil {
			continue
		}

		for _, cmd := range commands {
			c.execute(cmd)
		}
	}
}

func (c *Commander) execute(cmd client.AgentCommand) {
	fmt.Printf("  [%s] ⚡ Command received: %s", ts(), cmd.CommandType)
	if cmd.FilePath != "" {
		fmt.Printf(" → %s", cmd.FilePath)
	}
	if cmd.ShellCommand != "" {
		fmt.Printf(" → %s", truncateDisplay(cmd.ShellCommand, 80))
	}
	fmt.Println()

	var status string
	var stdout, stderr, errMsg string
	var exitCode int

	switch cmd.CommandType {
	case "WRITE_FILE":
		errMsg = c.writeFile(cmd)
	case "DELETE_FILE":
		errMsg = c.deleteFile(cmd)
	case "PATCH_FILE":
		errMsg = c.patchFile(cmd)
	case "EXECUTE_SHELL":
		stdout, stderr, exitCode, errMsg = c.executeShell(cmd)
	default:
		errMsg = "unknown command type: " + cmd.CommandType
	}

	if errMsg == "" {
		status = "COMPLETED"
		fmt.Printf("  [%s] ✓ Command completed: %s\n", ts(), cmd.ID)
	} else {
		status = "FAILED"
		fmt.Printf("  [%s] ✗ Command failed: %s — %s\n", ts(), cmd.ID, errMsg)
	}

	_ = c.anwinClient.ReportCommandResult(cmd.ID, status, stdout, stderr, exitCode, errMsg)
}

func (c *Commander) writeFile(cmd client.AgentCommand) string {
	if cmd.FilePath == "" {
		return "filePath is required"
	}

	safePath, err := c.safePath(cmd.FilePath)
	if err != nil {
		return err.Error()
	}

	dir := filepath.Dir(safePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("cannot create directory: %s", err.Error())
	}

	if err := os.WriteFile(safePath, []byte(cmd.Content), 0644); err != nil {
		return fmt.Sprintf("cannot write file: %s", err.Error())
	}

	return ""
}

func (c *Commander) deleteFile(cmd client.AgentCommand) string {
	if cmd.FilePath == "" {
		return "filePath is required"
	}

	safePath, err := c.safePath(cmd.FilePath)
	if err != nil {
		return err.Error()
	}

	if _, statErr := os.Stat(safePath); os.IsNotExist(statErr) {
		return ""
	}

	if err := os.Remove(safePath); err != nil {
		return fmt.Sprintf("cannot delete file: %s", err.Error())
	}

	return ""
}

func (c *Commander) patchFile(cmd client.AgentCommand) string {
	if cmd.FilePath == "" {
		return "filePath is required"
	}

	safePath, err := c.safePath(cmd.FilePath)
	if err != nil {
		return err.Error()
	}

	if _, statErr := os.Stat(safePath); os.IsNotExist(statErr) {
		return fmt.Sprintf("file does not exist: %s", cmd.FilePath)
	}

	if err := os.WriteFile(safePath, []byte(cmd.Content), 0644); err != nil {
		return fmt.Sprintf("cannot patch file: %s", err.Error())
	}

	return ""
}

func (c *Commander) executeShell(cmd client.AgentCommand) (string, string, int, string) {
	if cmd.ShellCommand == "" {
		return "", "", 1, "shellCommand is required"
	}

	timeout := cmd.TimeoutSeconds
	if timeout <= 0 {
		timeout = 300
	}
	if timeout > 1800 {
		timeout = 1800
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	workDir := c.root
	if cmd.WorkingDir != "" {
		sub, err := c.safePath(cmd.WorkingDir)
		if err != nil {
			return "", "", 1, err.Error()
		}
		info, statErr := os.Stat(sub)
		if statErr != nil || !info.IsDir() {
			return "", "", 1, fmt.Sprintf("working directory not found: %s", cmd.WorkingDir)
		}
		workDir = sub
	}

	var shell string
	var shellFlag string

	if runtime.GOOS == "windows" {
		shell = "cmd"
		shellFlag = "/C"
	} else {
		shell = "/bin/sh"
		shellFlag = "-c"
	}

	proc := exec.CommandContext(ctx, shell, shellFlag, cmd.ShellCommand)
	proc.Dir = workDir

	proc.Env = append(os.Environ(),
		"ANWIN_PROJECT_DIR="+c.root,
		"ANWIN_AGENT=true",
	)

	var stdoutBuf, stderrBuf strings.Builder
	proc.Stdout = &stdoutBuf
	proc.Stderr = &stderrBuf

	err := proc.Run()

	stdout := truncateOutput(stdoutBuf.String(), maxOutputBytes)
	stderr := truncateOutput(stderrBuf.String(), maxOutputBytes)

	if ctx.Err() == context.DeadlineExceeded {
		return stdout, stderr, 124, fmt.Sprintf("command timed out after %d seconds", timeout)
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return stdout, stderr, 1, fmt.Sprintf("exec error: %s", err.Error())
		}
	}

	return stdout, stderr, exitCode, ""
}

func (c *Commander) safePath(relative string) (string, error) {
	cleaned := filepath.Clean(relative)
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("absolute paths not allowed: %s", relative)
	}

	full := filepath.Join(c.root, cleaned)

	resolved, err := filepath.Abs(full)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path: %s", err.Error())
	}

	rootAbs, _ := filepath.Abs(c.root)
	if !strings.HasPrefix(resolved, rootAbs+string(filepath.Separator)) && resolved != rootAbs {
		return "", fmt.Errorf("path traversal detected: %s", relative)
	}

	return resolved, nil
}

func truncateOutput(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n... [truncated]"
}

func truncateDisplay(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func ts() string {
	return time.Now().Format("15:04:05")
}
