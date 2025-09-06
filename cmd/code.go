package cmd

import (
	"bufio"
	"context"
	"fmt"
	"github.com/meysamhadeli/codai/code_analyzer/models"
	"github.com/meysamhadeli/codai/constants/lipgloss"
	"github.com/meysamhadeli/codai/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// CodeCmd: codai code
var codeCmd = &cobra.Command{
	Use:   "code",
	Short: "Run the AI-powered code assistant for various coding tasks within a session.",
	Long: `The 'code' subcommand allows users to leverage a session-based AI assistant for a range of coding tasks. 
This assistant can suggest new code, refactor existing code, review code for improvements, and even propose new features 
based on the current project context. Each interaction is part of a session, allowing for continuous context and 
improved responses throughout the user experience.`,
	Run: func(cmd *cobra.Command, args []string) {
		rootDependencies := handleRootCommand(cmd)
		handleCodeCommand(rootDependencies)
	},
}

func handleCodeCommand(rootDependencies *RootDependencies) {

	// Create a context with cancel function
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var requestedContext string
	var fullContext *models.FullContextData

	spinner := pterm.DefaultSpinner.WithStyle(pterm.NewStyle(pterm.FgLightBlue)).WithSequence("⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏").WithDelay(100).WithRemoveWhenDone(true)

	go utils.GracefulShutdown(ctx, cancel, func() {

		rootDependencies.ChatHistory.ClearHistory()
		rootDependencies.TokenManagement.ClearToken()
	})

	reader := bufio.NewReader(os.Stdin)

	codeOptionsBox := lipgloss.BoxStyle.Render("/help  Help for code subcommand")
	fmt.Println(codeOptionsBox)

	spinnerLoadContext, _ := spinner.Start("Loading Context...")

	// Get all data files from the root directory using configured display mode
	fullContext, err := rootDependencies.Analyzer.GetProjectFilesWithDisplayMode(rootDependencies.Cwd, rootDependencies.Config.FileDisplayMode)

	if err != nil {
		spinnerLoadContext.Stop()
		fmt.Print("\r")
		fmt.Println(lipgloss.Red.Render(fmt.Sprintf("%v", err)))
	}

	spinnerLoadContext.Stop()
	fmt.Print("\r")

	// Launch the user input handler in a goroutine
startLoop: // Label for the start loop
	for {
		select {
		case <-ctx.Done():
			// Wait for GracefulShutdown to complete
			return

		default:
			displayTokens := func() {
				rootDependencies.TokenManagement.DisplayTokens(rootDependencies.Config.AIProviderConfig.Provider, rootDependencies.Config.AIProviderConfig.Model)
			}

			// Get user input with context cancellation support
			userInput, err := utils.InputPromptWithContext(ctx, reader)

			if err != nil {
				// Check if the error is due to context cancellation (Ctrl+C)
				if err == context.Canceled {
					fmt.Println(lipgloss.Yellow.Render("\n🔄 Exiting..."))
					return
				}
				fmt.Println(lipgloss.Red.Render(fmt.Sprintf("%v", err)))
				continue
			}

			if userInput == "" {
				fmt.Print("\r")
				continue
			}

			// Configure help code subcommand
			isHelpSubcommands, exit := findCodeSubCommand(userInput, rootDependencies)

			if isHelpSubcommands {
				continue
			}

			if exit {
				return
			}

			var aiResponseBuilder strings.Builder

			chatRequestOperation := func() error {

				finalPrompt, userInputPrompt := rootDependencies.Analyzer.GeneratePrompt(fullContext.RawCodes, rootDependencies.ChatHistory.GetHistory(), userInput, requestedContext)

				// 启动AI思考动画
				aiSpinner := pterm.DefaultSpinner.
					WithStyle(pterm.NewStyle(pterm.FgCyan)).
					WithSequence("🤔", "🧠", "💭", "✨", "🚀", "💡").
					WithDelay(1000).
					WithRemoveWhenDone(true)
				
				// 根据不同provider显示不同的动画文案
				var spinnerText string
				switch rootDependencies.Config.AIProviderConfig.Provider {
				case "anthropic":
					spinnerText = "Claude is analyzing your code..."
				case "openai":
					spinnerText = "ChatGPT is processing your request..."
				case "azure-openai":
					spinnerText = "Azure OpenAI is processing..."
				case "gemini":
					spinnerText = "Gemini is thinking..."
				case "ollama":
					spinnerText = "Local AI is working..."
				case "deepseek":
					spinnerText = "DeepSeek is analyzing..."
				case "grok":
					spinnerText = "Grok is processing..."
				case "mistral":
					spinnerText = "Mistral is thinking..."
				case "qwen":
					spinnerText = "Qwen is working..."
				case "openrouter":
					spinnerText = "OpenRouter AI is processing..."
				default:
					spinnerText = "AI is thinking..."
				}
				
				spinnerAI, _ := aiSpinner.Start(spinnerText)

				// Step 7: Send the relevant code and user input to the AI API
				responseChan := rootDependencies.CurrentChatProvider.ChatCompletionRequest(ctx, userInputPrompt, finalPrompt)

				// Iterate over response channel to handle streamed data or errors.
				firstResponse := true
				for response := range responseChan {
					if response.Err != nil {
						spinnerAI.Stop()
						return response.Err
					}

					if response.Done {
						if firstResponse {
							spinnerAI.Stop()
						}
						rootDependencies.ChatHistory.AddToHistory(userInput, aiResponseBuilder.String())
						return nil
					}

					// 收到第一个响应内容时停止spinner并开始显示内容
					if firstResponse && response.Content != "" {
						spinnerAI.Stop()
						fmt.Print("\n") // 为输出内容留出空间
						firstResponse = false
					}

					aiResponseBuilder.WriteString(response.Content)

					language := utils.DetectLanguageFromCodeBlock(response.Content)
					if err := utils.RenderAndPrintMarkdownWithContext(ctx, response.Content, language, rootDependencies.Config.Theme); err != nil {
						// Check if it was cancelled by user
						if err == context.Canceled {
							return fmt.Errorf("Output cancelled by user")
						}
						return fmt.Errorf("Error rendering markdown: %v", err)
					}
				}

				return nil
			}

			// First, execute the AI request
			if err := chatRequestOperation(); err != nil {
				fmt.Println(lipgloss.Red.Render(fmt.Sprintf("%v", err)))
				displayTokens()
				continue startLoop
			}

			// After AI response is complete, try to get full block code if block codes is summarized and incomplete
			requestedContext, err = rootDependencies.Analyzer.TryGetInCompletedCodeBlocK(aiResponseBuilder.String())

			if requestedContext != "" && err == nil {
				fmt.Print("\n")
				fmt.Println(lipgloss.Green.Render("🔄 Auto-accepting additional context for complete code blocks..."))

				// Reset the builder for second request
				aiResponseBuilder.Reset()

				if err := chatRequestOperation(); err != nil {
					fmt.Println(lipgloss.Red.Render(fmt.Sprintf("%v", err)))
					displayTokens()
					continue
				}
			}

			// Extract code from AI response and structure this code to apply to git
			changes := rootDependencies.Analyzer.ExtractCodeChanges(aiResponseBuilder.String())

			if changes == nil {
				fmt.Println()
				displayTokens()
				continue
			}

			fmt.Print("\n")

			// Try to apply changes
			for _, change := range changes {

				// Prompt the user to accept or reject the changes
				promptAccepted, err := utils.ConfirmPrompt(change.RelativePath, reader)
				if err != nil {
					fmt.Println(lipgloss.Red.Render(fmt.Sprintf("Error getting user prompt: %v", err)))
					continue
				}

				if promptAccepted {
					err := rootDependencies.Analyzer.ApplyChanges(change.RelativePath, change.Code)
					if err != nil {
						fmt.Println(lipgloss.Red.Render(fmt.Sprintf("Error applying changes: %v", err)))
						continue
					}
					fmt.Println(lipgloss.Green.Render("✔️ Changes accepted!"))

					fmt.Print("\r")

				} else {
					fmt.Println(lipgloss.Red.Render("❌ Changes rejected."))
				}
			}

			displayTokens()
		}
	}
}

func findCodeSubCommand(command string, rootDependencies *RootDependencies) (bool, bool) {
	switch command {
	case "/help":
		helps := "/clear  Clear screen\n/exit  Exit from codai\n/token  Token information\n/live-token  Session token stats with details\n/clear-token  Clear token from session\n/clear-history  Clear history of chat from session\n/display-mode  Show current file display mode\n/set-display-mode <mode>  Set file display mode (info/relevant/full)"
		styledHelps := lipgloss.BoxStyle.Render(helps)
		fmt.Println(styledHelps)
		return true, false
	case "/clear":
		fmt.Print("\033[2J\033[H")
		return true, false
	case "/exit":
		return false, true
	case "/token":
		rootDependencies.TokenManagement.DisplayTokens(
			rootDependencies.Config.AIProviderConfig.Provider,
			rootDependencies.Config.AIProviderConfig.Model,
		)
		return true, false
	case "/live-token":
		// 显示实时token统计信息 
		total, input, output := rootDependencies.TokenManagement.GetCurrentTokenUsage()
		cost := rootDependencies.TokenManagement.CalculateCost(
			rootDependencies.Config.AIProviderConfig.Provider,
			rootDependencies.Config.AIProviderConfig.Model,
			input, output,
		)
		fmt.Printf("📊 Session Token Stats:\n")
		fmt.Printf("   Total: %d tokens (Input: %d, Output: %d)\n", total, input, output)
		fmt.Printf("   Cost: $%.6f\n", cost)
		fmt.Printf("   Model: %s\n", rootDependencies.Config.AIProviderConfig.Model)
		return true, false
	case "/clear-token":
		rootDependencies.TokenManagement.ClearToken()
		return true, false
	case "/clear-history":
		rootDependencies.ChatHistory.ClearHistory()
		return true, false
	case "/display-mode":
		fmt.Printf("Current file display mode: %s\n", rootDependencies.Config.FileDisplayMode)
		fmt.Println("Available modes:")
		fmt.Println("  info     - Show only file directory, name, and line count")
		fmt.Println("  relevant - Show relevant code parts (parsed or first 50 lines)")
		fmt.Println("  full     - Show complete file content")
		return true, false
	default:
		// Handle set-display-mode command
		if strings.HasPrefix(command, "/set-display-mode ") {
			parts := strings.Split(command, " ")
			if len(parts) >= 2 {
				mode := strings.TrimSpace(parts[1])
				if mode == "info" || mode == "relevant" || mode == "full" {
					rootDependencies.Config.FileDisplayMode = mode
					fmt.Printf("File display mode set to: %s\n", mode)
					fmt.Println("Note: Changes will take effect for new context loading.")
				} else {
					fmt.Println("Invalid display mode. Use 'info', 'relevant', or 'full'.")
				}
			} else {
				fmt.Println("Usage: /set-display-mode <mode>")
				fmt.Println("Available modes: info, relevant, full")
			}
			return true, false
		}
		return false, false
	}
}
