package utils

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GitOperations handles git-related operations
type GitOperations struct {
	workingDir string
}

// NewGitOperations creates a new GitOperations instance
func NewGitOperations(workingDir string) *GitOperations {
	return &GitOperations{workingDir: workingDir}
}

// CheckGitRepo checks if the current directory is a git repository
func (g *GitOperations) CheckGitRepo() error {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = g.workingDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not a git repository")
	}
	return nil
}

// GetGitStatus returns the current git status
func (g *GitOperations) GetGitStatus() (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = g.workingDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}
	return string(output), nil
}

// AddFiles adds all modified files to staging
func (g *GitOperations) AddFiles() error {
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = g.workingDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add files to git: %w", err)
	}
	return nil
}

// Commit creates a git commit with the given message
func (g *GitOperations) Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = g.workingDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}
	return nil
}

// GetGitDiff returns the diff of staged changes
func (g *GitOperations) GetGitDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--unified=3")
	cmd.Dir = g.workingDir
	output, err := cmd.Output()
	if err != nil {
		// Check if there's no diff (empty)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("failed to get git diff: %w", err)
	}
	return string(output), nil
}

// GetRecentCommits returns recent commit messages
func (g *GitOperations) GetRecentCommits(limit int) ([]string, error) {
	cmd := exec.Command("git", "log", fmt.Sprintf("--max-count=%d", limit), "--pretty=format:%H|%s|%an|%ai")
	cmd.Dir = g.workingDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get recent commits: %w", err)
	}
	
	var commits []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line != "" {
			commits = append(commits, line)
		}
	}
	return commits, nil
}

// GetBranchName returns the current branch name
func (g *GitOperations) GetBranchName() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = g.workingDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get branch name: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// HasUncommittedChanges checks if there are uncommitted changes
func (g *GitOperations) HasUncommittedChanges() (bool, error) {
	status, err := g.GetGitStatus()
	if err != nil {
		return false, err
	}
	return status != "", nil
}

// HasStagedChanges checks if there are staged changes ready to commit
func (g *GitOperations) HasStagedChanges() (bool, error) {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = g.workingDir
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Exit code 1 means there are staged changes
			return true, nil
		}
		return false, fmt.Errorf("failed to check staged changes: %w", err)
	}
	return false, nil // Exit code 0 means no staged changes
}

// GenerateCommitRequest creates a request for AI to generate commit message
func (g *GitOperations) GenerateCommitRequest(ctx context.Context, recentCommits []string, stagedDiff string) string {
	var commitMetadataBuilder strings.Builder
	commitMetadataBuilder.WriteString(fmt.Sprintf("Time: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	commitMetadataBuilder.WriteString(fmt.Sprintf("Working Directory: %s\n", g.workingDir))
	
	branch, _ := g.GetBranchName()
	commitMetadataBuilder.WriteString(fmt.Sprintf("Branch: %s\n", branch))
	
	if len(recentCommits) > 0 {
		commitMetadataBuilder.WriteString("\n## Recent Commit History:\n")
		for i, commit := range recentCommits {
			if i < 3 { // Show only last 3 commits
				parts := strings.Split(commit, "|")
				if len(parts) >= 2 {
					commitMetadataBuilder.WriteString(fmt.Sprintf("- %s: %s\n", parts[1][:7], parts[2]))
				}
			}
		}
	}
	
	if stagedDiff != "" {
		commitMetadataBuilder.WriteString("\n## Staged Changes:\n")
		commitMetadataBuilder.WriteString("```diff\n")
		// Limit diff to avoid overwhelming the AI
		diffLines := strings.Split(stagedDiff, "\n")
		if len(diffLines) > 100 {
			commitMetadataBuilder.WriteString(strings.Join(diffLines[:100], "\n"))
			commitMetadataBuilder.WriteString(fmt.Sprintf("\n... (truncated %d more lines)\n", len(diffLines)-100))
		} else {
			commitMetadataBuilder.WriteString(stagedDiff)
		}
		commitMetadataBuilder.WriteString("\n```\n")
	}
	
	commitMetadataBuilder.WriteString("\n## User Request:\n")
	commitMetadataBuilder.WriteString("")
	
	return commitMetadataBuilder.String()
}