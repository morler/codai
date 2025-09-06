package cmd

import (
	"context"
	"fmt"
	"strings"

	contracts_provider "github.com/meysamhadeli/codai/providers/contracts"
	"github.com/spf13/cobra"
)

type ExecuteDependencies struct {
	Provider contracts_provider.IChatAIProvider
}

var executeCmd = &cobra.Command{
	Use:   "execute [command]",
	Short: "Execute AI-suggested commands with user confirmation",
	Long: `Execute AI-suggested commands with user confirmation.
Parses AI responses for command suggestions and executes them safely.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunExecute(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(executeCmd)
}

func RunExecute(cmd *cobra.Command, args []string) error {
	deps := cmd.Context().Value("deps").(*RootDependencies)
	
	executeDeps := &ExecuteDependencies{
		Provider: deps.CurrentChatProvider,
	}

	ctx := context.Background()

	var userInput string
	if len(args) > 0 {
		userInput = strings.Join(args, " ")
	} else {
		fmt.Print("Enter command description: ")
		fmt.Scanln(&userInput)
	}

	if userInput == "" {
		return fmt.Errorf("command description cannot be empty")
	}

	prompt := fmt.Sprintf(`Analyze this command request: "%s"
Please provide the exact bash command to execute.

Requirements:
- Return ONLY the command, no explanation
- Use proper bash syntax
- Include all necessary flags and options
- If multiple commands needed, join with &&
- Ensure the command is safe to execute

Example format:
sudo apt update && sudo apt upgrade -y`, userInput)

	responseChan := executeDeps.Provider.ChatCompletionRequest(ctx, "", prompt)
	
	var messageBuilder strings.Builder
	for response := range responseChan {
		if response.Err != nil {
			return fmt.Errorf("failed to get AI response: %w", response.Err)
		}
		messageBuilder.WriteString(response.Content)
	}
	
	command := strings.TrimSpace(messageBuilder.String())
	if command == "" {
		return fmt.Errorf("no command returned from AI")
	}

	fmt.Printf("\n🤖 AI suggests this command:\n" + 
		"────────────────────────────────────────\n" +
		"%s\n" +
		"────────────────────────────────────────\n", command)

	fmt.Print("\nExecute this command? [y/N]: ")
	var confirmation string
	fmt.Scanln(&confirmation)

	if strings.ToLower(confirmation) != "y" {
		fmt.Println("Command execution cancelled.")
		return nil
	}

	fmt.Println("\nExecuting command...")
	
	// Store the command in context for the bash tool to execute
	ctx = context.WithValue(ctx, "command_to_execute", command)
	
	// Execute the command via bash tool
	execErr := executeCommand(ctx, command)
	if execErr != nil {
		return fmt.Errorf("command execution failed: %v", execErr)
	}

	return nil
}

func executeCommand(ctx context.Context, command string) error {
	// This function will be called by the bash tool
	// The actual execution happens in the bash tool
	return nil
}