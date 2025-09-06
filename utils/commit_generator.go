package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/meysamhadeli/codai/providers/models"
)

// CommitMessageRequest represents the request for generating a commit message
type CommitMessageRequest struct {
	StagedDiff      string
	WorkingDir      string
	Branch          string
	RecentCommits   []string
	UserInput       string
	CommitStyle     string
}

// CommitMessageGenerator generates commit messages using AI
type CommitMessageGenerator struct {
	aiProvider interface {
		ChatCompletionRequest(ctx context.Context, userPrompt string, systemPrompt string) <-chan models.StreamResponse
	}
}

// NewCommitMessageGenerator creates a new commit message generator
func NewCommitMessageGenerator(aiProvider interface {
	ChatCompletionRequest(ctx context.Context, userPrompt string, systemPrompt string) <-chan models.StreamResponse
}) *CommitMessageGenerator {
	return &CommitMessageGenerator{aiProvider: aiProvider}
}

// GenerateCommitMessage generates a commit message using AI
func (g *CommitMessageGenerator) GenerateCommitMessage(ctx context.Context, request CommitMessageRequest) (string, error) {
	systemPrompt := g.createCommitSystemPrompt()
	userPrompt := createCommitUserPrompt(request)
	
	responseChan := g.aiProvider.ChatCompletionRequest(ctx, userPrompt, systemPrompt)
	
	var messageBuilder strings.Builder
	for response := range responseChan {
		if response.Err != nil {
			return "", fmt.Errorf("failed to generate commit message: %w", response.Err)
		}
		messageBuilder.WriteString(response.Content)
	}
	
	return messageBuilder.String(), nil
}

func (g *CommitMessageGenerator) createCommitSystemPrompt() string {
	return `You are a helpful AI assistant that generates concise, meaningful Git commit messages.

Please follow these guidelines for commit messages:
1. Keep the first line under 72 characters (the title)
2. Use present tense ("Add feature" not "Added feature") 
3. Use imperative mood ("Move cursor to..." not "Moves cursor to...")
4. Start with a capital letter
5. Don't end with a period
6. Be specific about what changed

Format the commit message as:
- First line: Brief summary (required)
- Second line: Leave blank
- Third+ lines: Detailed explanation if needed

Examples:
"Add user authentication middleware

- Implement JWT token validation
- Add refresh token mechanism  
- Update auth middleware tests"

"Fix memory leak in image processing

- Properly dispose of image resources
- Fix goroutine cleanup in resize function"

"Update dependencies to latest versions

- Update Go modules
- Fix breaking changes in image library"`
}

func createCommitUserPrompt(request CommitMessageRequest) string {
	var prompt strings.Builder
	
	prompt.WriteString("Please generate a commit message for the following changes:")
	
	if request.WorkingDir != "" {
		prompt.WriteString(fmt.Sprintf("\nWorking Directory: %s", request.WorkingDir))
	}
	
	if request.Branch != "" {
		prompt.WriteString(fmt.Sprintf("\nBranch: %s", request.Branch))
	}
	
	if len(request.RecentCommits) > 0 {
		prompt.WriteString("\n\n## Recent Commit History!")
		for i, commit := range request.RecentCommits {
			if i < 3 {
				prompt.WriteString(fmt.Sprintf("\n- %s", commit))
			}
		}
	}
	
	if request.StagedDiff != "" {
		prompt.WriteString("\n\n## Staged Changes:")
		prompt.WriteString(fmt.Sprintf("\n```diff\n%s\n```", request.StagedDiff))
	}
	
	if request.UserInput != "" {
		prompt.WriteString(fmt.Sprintf("\n\n## User Request:\n%s", request.UserInput))
	}
	
	if request.CommitStyle != "" {
		prompt.WriteString(fmt.Sprintf("\n\n## Preferred Style:\n%s", request.CommitStyle))
	}
	
	prompt.WriteString("\n\n## Instructions:")
	prompt.WriteString("\n- Generate a clear and concise commit message based on the changes shown above")
	prompt.WriteString("\n- Focus on the 'why' and 'what' the changes accomplish")
	prompt.WriteString("\n- Follow the commit message best practices and formatting guidelines")
	prompt.WriteString("\n- If the changes are small and obvious, keep the message brief")
	prompt.WriteString("\n- For larger changes, include a detailed description in the body")
	
	return prompt.String()
}