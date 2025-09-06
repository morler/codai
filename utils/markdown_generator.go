package utils

import (
	"bytes"
	"context"
	"fmt"
	"github.com/alecthomas/chroma/v2/quick"
	"os"
	"strings"
)

var isCodeBlock = false

// RenderAndPrintMarkdown handles the rendering of markdown content,
func RenderAndPrintMarkdown(line string, language string, theme string) error {

	if strings.HasPrefix(line, "```") {
		isCodeBlock = !isCodeBlock
	}
	// Process the line based on its prefix
	if strings.HasPrefix(line, "+") && isCodeBlock {
		line = "\x1b[92m" + line + "\x1b[0m"
		fmt.Print(line)
	} else if strings.HasPrefix(line, "-") && isCodeBlock {
		line = "\x1b[91m" + line + "\x1b[0m"
		fmt.Print(line)
	} else {
		// Render the processed line
		err := quick.Highlight(os.Stdout, line, language, "terminal256", theme)
		if err != nil {
			return err
		}
	}

	return nil
}

// RenderAndPrintMarkdownWithContext handles the rendering of markdown content with cancellation support
func RenderAndPrintMarkdownWithContext(ctx context.Context, content string, language string, theme string) error {
	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		// Check for context cancellation before each line
		select {
		case <-ctx.Done():
			fmt.Printf("\n\n🔄 Output interrupted...\n")
			return ctx.Err()
		default:
		}
		
		// Process the line based on its prefix
		if strings.HasPrefix(line, "```") {
			isCodeBlock = !isCodeBlock
		}
		
		if strings.HasPrefix(line, "+") && isCodeBlock {
			coloredLine := "\x1b[92m" + line + "\x1b[0m\n"
			fmt.Print(coloredLine)
		} else if strings.HasPrefix(line, "-") && isCodeBlock {
			coloredLine := "\x1b[91m" + line + "\x1b[0m\n"
			fmt.Print(coloredLine)
		} else {
			// Use a buffer to capture the highlight output
			var buf bytes.Buffer
			if err := quick.Highlight(&buf, line+"\n", language, "terminal256", theme); err != nil {
				return err
			}
			fmt.Print(buf.String())
		}
		
		// Check for cancellation more frequently for responsive interruption
		if i%5 == 0 {
			select {
			case <-ctx.Done():
				fmt.Printf("\n\n🔄 Output interrupted...\n")
				return ctx.Err()
			default:
			}
		}
	}
	
	return nil
}
