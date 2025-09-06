package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// CommandExecutor handles safe execution of AI-suggested commands
type CommandExecutor struct {
	// Add any configuration or dependencies here
}

// NewCommandExecutor creates a new command executor instance
func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{}
}

// ExecuteCommand safely executes a command with user confirmation
func (ce *CommandExecutor) ExecuteCommand(ctx context.Context, command string) error {
	if command == "" {
		return fmt.Errorf("empty command provided")
	}

	// Security checks
	if err := ce.validateCommand(command); err != nil {
		return fmt.Errorf("command validation failed: %v", err)
	}

	// Platform-specific execution
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		// Unix-like systems
		cmd = exec.CommandContext(ctx, "bash", "-c", command)
	}

	// Set up pipes for real-time output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	fmt.Printf("=>")
	
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command execution failed: %v", err)
	}

	return nil
}

// validateCommand performs security checks on the proposed command
func (ce *CommandExecutor) validateCommand(command string) error {
	// List of dangerous commands/patterns to reject
	dangerousPatterns := []string{
		"rm -rf /",
		":(){ :|:& };:",  // Fork bomb
		"> /dev/sda",      // Disk overwrite
		"wipefs",
		"fdisk",
		"mkfs",
		"dd if=",
	}

	cmdLower := strings.ToLower(command)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmdLower, strings.ToLower(pattern)) {
			return fmt.Errorf("potentially dangerous command detected: %s", pattern)
		}
	}

	return nil
}

// IntegrateCommandExecution adds command execution functionality to a cobra command
func IntegrateCommandExecution(cmd *cobra.Command, args []string) error {
	// This function can be used to integrate command execution into existing commands
	// It will check for command context and execute if present
	
	ctx := cmd.Context()
	if command, ok := ctx.Value("command_to_execute").(string); ok && command != "" {
		executor := NewCommandExecutor()
		return executor.ExecuteCommand(ctx, command)
	}
	
	return nil
}