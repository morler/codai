package utils

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/meysamhadeli/codai/constants/lipgloss"
)

// InputPrompt prompts the user to enter their request for code assistance in a charming way
func InputPrompt(reader *bufio.Reader) (string, error) {

	// Beautifully styled prompt message
	fmt.Print(lipgloss.BlueSky.Render("> "))

	// Read user input
	userInput, err := reader.ReadString('\n')
	if userInput == "" {
		return "", nil
	}

	if err != nil {
		if err == io.EOF {
			return "", nil
		}
		return "", fmt.Errorf(lipgloss.Red.Render("🚫 Error reading input: "))
	}

	return strings.TrimSpace(userInput), nil
}

// InputPromptWithContext prompts the user with context cancellation support
func InputPromptWithContext(ctx context.Context, reader *bufio.Reader) (string, error) {
	// Create channels for input and errors
	inputChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Start a goroutine to read input
	go func() {
		// Beautifully styled prompt message
		fmt.Print(lipgloss.BlueSky.Render("> "))
		
		userInput, err := reader.ReadString('\n')
		
		if err != nil {
			if err == io.EOF {
				errChan <- nil
			} else {
				errChan <- fmt.Errorf(lipgloss.Red.Render("🚫 Error reading input: "))
			}
			return
		}
		
		if userInput == "" {
			inputChan <- ""
		} else {
			inputChan <- strings.TrimSpace(userInput)
		}
	}()

	// Wait for either input or context cancellation
	select {
	case <-ctx.Done():
		fmt.Println() // Print newline for clean exit
		return "", ctx.Err()
	case err := <-errChan:
		return "", err
	case input := <-inputChan:
		return input, nil
	}
}
