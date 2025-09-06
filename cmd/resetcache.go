package cmd

import (
	"bufio"
	"fmt"
	"github.com/meysamhadeli/codai/constants/lipgloss"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

// resetCacheCmd represents the reset-cache command
var resetCacheCmd = &cobra.Command{
	Use:   "reset-cache",
	Short: "Reset the project cache for Codai",
	Long: `The 'reset-cache' command removes all cached files in the project '.cache' directory.
This includes file content cache, tree-sitter parsing results, project snapshots, and configuration cache.
Use this command to clear corrupted cache or when experiencing cache-related issues.`,
	Run: func(cmd *cobra.Command, args []string) {
		var force bool
		var stats bool

		// Parse flags
		force, _ = cmd.Flags().GetBool("force")
		stats, _ = cmd.Flags().GetBool("stats")

		// Handle reset-cache command
		handleResetCacheCommand(force, stats, cmd)
	},
}

func init() {
	// Define command-specific flags
	resetCacheCmd.Flags().BoolP("force", "f", false, "Force cache reset without confirmation")
	resetCacheCmd.Flags().BoolP("stats", "s", false, "Show cache statistics before reset")

	// Add the reset-cache command to the root command
	rootCmd.AddCommand(resetCacheCmd)
}

func handleResetCacheCommand(force bool, showStats bool, cmd *cobra.Command) {
	// Initialize the analyzer with cache enabled
	rootDependencies := handleRootCommand(cmd)
	if rootDependencies == nil {
		return
	}

	// Show cache statistics if requested
	if showStats && rootDependencies.Analyzer != nil {
		fmt.Println(lipgloss.Info.Render("Cache Statistics:"))
		if cacheStats, err := rootDependencies.Analyzer.GetCacheStats(); err == nil {
			if enabled, ok := cacheStats["cache_enabled"].(bool); !ok || !enabled {
				fmt.Println("  Cache is disabled")
				return
			}
			
			if dir, ok := cacheStats["cache_dir"].(string); ok {
				fmt.Printf("  Cache Directory: %s\n", dir)
			}
			if files, ok := cacheStats["cache_files"].(int); ok {
				fmt.Printf("  Cached Files: %d\n", files)
			}
			if size, ok := cacheStats["total_size"].(int64); ok {
				fmt.Printf("  Total Size: %.2f MB\n", float64(size)/(1024*1024))
			}
			if hitRate, ok := cacheStats["hit_rate"].(float64); ok {
				fmt.Printf("  Hit Rate: %.1f%%\n", hitRate)
			}
		} else {
			fmt.Println(lipgloss.Yellow.Render(fmt.Sprintf("Warning: Could not show statistics: %v", err)))
		}
		
		// Only show stats, skip the actual reset
		return
	}

	// Confirm reset for full cache reset (if not forced)
	if !force {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Are you sure you want to reset the entire project cache? (y/N): ")
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println(lipgloss.Yellow.Render("Cache reset cancelled."))
			return
		}
	}

	// Reset the cache
	spinner := pterm.DefaultSpinner.WithStyle(pterm.NewStyle(pterm.FgCyan)).
		WithSequence("⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏").
		WithDelay(100).WithRemoveWhenDone(true)

	spinnerInstance, _ := spinner.Start("Resetting project cache...")

	// Clear cache using the analyzer's cache manager
	if rootDependencies.Analyzer == nil {
		spinnerInstance.Stop()
		fmt.Print("\r")
		fmt.Println(lipgloss.Yellow.Render("Cache is disabled. No cache to reset."))
		return
	}

	err := rootDependencies.Analyzer.ClearCache()
	if err != nil {
		spinnerInstance.Stop()
		fmt.Print("\r")
		fmt.Println(lipgloss.Red.Render(fmt.Sprintf("Error resetting cache: %v", err)))
		return
	}

	spinnerInstance.Stop()
	fmt.Print("\r")
	fmt.Println(lipgloss.Green.Render("✓ Project cache has been successfully reset!"))
}