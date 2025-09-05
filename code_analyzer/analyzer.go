package code_analyzer

import (
	"encoding/json"
	"fmt"
	"github.com/meysamhadeli/codai/code_analyzer/contracts"
	"github.com/meysamhadeli/codai/code_analyzer/models"
	"github.com/meysamhadeli/codai/embed_data"
	"github.com/meysamhadeli/codai/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CodeAnalyzer handles the analysis of project files.
type CodeAnalyzer struct {
	Cwd          string
	cacheManager *CacheManager
}

func (analyzer *CodeAnalyzer) GeneratePrompt(codes []string, history []string, userInput string, requestedContext string) (string, string) {

	promptTemplate := string(embed_data.SummarizeFullContextPrompt)

	// Combine the relevant code into a single string
	code := strings.Join(codes, "\n---------\n\n")

	prompt := fmt.Sprintf("%s\n\n______\n%s\n\n______\n", fmt.Sprintf("## Here is the summary of context of project\n\n%s", code), fmt.Sprintf("## Here is the general template prompt for using AI\n\n%s", promptTemplate))
	userInputPrompt := fmt.Sprintf("## Here is user request\n%s", userInput)

	if requestedContext != "" {
		prompt = prompt + fmt.Sprintf("## Here are the requsted full context files for using in your task\n\n%s______\n", requestedContext)
	}

	historyPrompt := "## Here is the history of chats\n\n" + strings.Join(history, "\n---------\n\n")
	finalPrompt := fmt.Sprintf("%s\n\n______\n\n%s", historyPrompt, prompt)

	return finalPrompt, userInputPrompt
}

// NewCodeAnalyzer initializes a new CodeAnalyzer.
func NewCodeAnalyzer(cwd string) contracts.ICodeAnalyzer {
	// Initialize cache manager
	cacheManager, err := NewCacheManager("")
	if err != nil {
		// Fallback to no caching if cache initialization fails
		log.Printf("Warning: Failed to initialize cache manager: %v", err)
		cacheManager = nil
	}

	return &CodeAnalyzer{
		Cwd:          cwd,
		cacheManager: cacheManager,
	}
}

func (analyzer *CodeAnalyzer) GetProjectFiles(rootDir string) (*models.FullContextData, error) {
	// Check cache first if cache manager is available
	if analyzer.cacheManager != nil {
		// Generate cache key based on root directory
		projectCacheKey := fmt.Sprintf("%s_project_scan", rootDir)
		if cachedData, found := analyzer.cacheManager.GetConfigCache(projectCacheKey); found {
			return cachedData, nil
		}
	}

	var result models.FullContextData

	// Retrieve the ignore patterns from .gitignore, if it exists
	gitIgnorePatterns, err := utils.GetGitignorePatterns(rootDir)
	if err != nil {
		return nil, err
	}

	// Walk the directory tree and find all files
	err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(rootDir, path)
		relativePath = strings.ReplaceAll(relativePath, "\\", "/")

		// Check if the current directory or file should be skipped based on default ignore patterns
		if utils.IsDefaultIgnored(relativePath) {
			// Skip the directory or file
			if d.IsDir() {
				// If it's a directory, skip the whole directory
				return filepath.SkipDir
			}
			// If it's a file, just skip the file
			return nil
		}

		// Ensure that the current entry is a file, not a directory
		if !d.IsDir() {

			// Check file size
			fileInfo, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("failed to get file info: %s, error: %w", relativePath, err)
			}
			// Skip files over 100 KB (100 * 1024 bytes)
			if fileInfo.Size() > 100*1024 {
				return nil // Skip this file
			}

			if utils.IsGitIgnored(relativePath, gitIgnorePatterns) {
				// Debugging: Print the ignored file
				return nil // Skip this file
			}

			// Try to get cached file content first
			var content []byte
			if analyzer.cacheManager != nil {
				if cachedContent, found := analyzer.cacheManager.GetFileContentCache(path); found {
					content = cachedContent
				}
			}

			// Read file content if not cached
			if content == nil {
				content, err = ioutil.ReadFile(path)
				if err != nil {
					return fmt.Errorf("failed to read file: %s, error: %w", relativePath, err)
				}

				// Cache the file content if cache manager is available
				if analyzer.cacheManager != nil {
					analyzer.cacheManager.SetFileContentCache(path, content)
				}
			}

			// Try to get cached tree-sitter results
			var codeParts []string
			if analyzer.cacheManager != nil {
				if cachedParts, found := analyzer.cacheManager.GetTreeSitterCache(path); found {
					codeParts = cachedParts
				}
			}

			// Process file if not cached
			if codeParts == nil {
				codeParts = analyzer.ProcessFile(relativePath, content)

				// Cache the tree-sitter results if cache manager is available
				if analyzer.cacheManager != nil {
					analyzer.cacheManager.SetTreeSitterCache(path, codeParts)
				}
			}

			// Append the file data to the result
			result.FileData = append(result.FileData, models.FileData{RelativePath: relativePath, Code: string(content), TreeSitterCode: strings.Join(codeParts, "\n")})

			result.RawCodes = append(result.RawCodes, fmt.Sprintf("**File: %s**\n\n%s", relativePath, strings.Join(codeParts, "\n")))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Cache the complete project scan results
	if analyzer.cacheManager != nil {
		projectCacheKey := fmt.Sprintf("%s_project_scan", rootDir)
		analyzer.cacheManager.SetConfigCache(projectCacheKey, &result)
	}

	return &result, nil
}


// GetProjectFilesIncremental performs incremental scanning of project files
// Returns only files that have been added, modified, or deleted since the last scan
func (analyzer *CodeAnalyzer) GetProjectFilesIncremental(rootDir string) (*models.FullContextData, bool, error) {
	if analyzer.cacheManager == nil {
		// Fallback to full scan if cache is not available
		fullResult, err := analyzer.GetProjectFiles(rootDir)
		return fullResult, false, err
	}

	// Load previous snapshot
	snapshotKey := fmt.Sprintf("%s_snapshot", rootDir)
	prevSnapshot := analyzer.loadProjectSnapshot(snapshotKey)

	// Scan current file states
	currentSnapshot, err := analyzer.createProjectSnapshot(rootDir)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create current snapshot: %w", err)
	}

	// If no previous snapshot exists, perform full scan and save snapshot
	if prevSnapshot == nil {
		fullResult, err := analyzer.GetProjectFiles(rootDir)
		if err != nil {
			return nil, false, err
		}

		// Save current snapshot for next incremental scan
		analyzer.saveProjectSnapshot(snapshotKey, currentSnapshot)
		return fullResult, false, nil
	}

	// Compare snapshots and identify changes
	changedFiles, deletedFiles := analyzer.compareSnapshots(prevSnapshot, currentSnapshot)
	

	// If no changes, return cached full result
	if len(changedFiles) == 0 && len(deletedFiles) == 0 {
		projectCacheKey := fmt.Sprintf("%s_project_scan", rootDir)
		if cachedData, found := analyzer.cacheManager.GetConfigCache(projectCacheKey); found {
			return cachedData, true, nil
		}
		// If no cache available, fallback to full scan but mark as incremental since we detected no changes
		fullResult, err := analyzer.GetProjectFiles(rootDir)
		return fullResult, true, err
	}

	// Process changed files incrementally
	incrementalResult, err := analyzer.processIncrementalChanges(rootDir, changedFiles, deletedFiles, prevSnapshot)
	if err != nil {
		return nil, false, fmt.Errorf("failed to process incremental changes: %w", err)
	}

	// Save updated snapshot
	analyzer.saveProjectSnapshot(snapshotKey, currentSnapshot)

	// Cache the updated full result
	projectCacheKey := fmt.Sprintf("%s_project_scan", rootDir)
	analyzer.cacheManager.SetConfigCache(projectCacheKey, incrementalResult)

	return incrementalResult, true, nil
}

// loadProjectSnapshot loads the previous project snapshot from cache
func (analyzer *CodeAnalyzer) loadProjectSnapshot(snapshotKey string) *models.ProjectSnapshot {
	if analyzer.cacheManager == nil {
		return nil
	}

	snapshot, found := analyzer.cacheManager.GetProjectSnapshot(snapshotKey)
	if !found {
		return nil
	}

	return snapshot
}

// saveProjectSnapshot saves the current project snapshot to cache
func (analyzer *CodeAnalyzer) saveProjectSnapshot(snapshotKey string, snapshot *models.ProjectSnapshot) {
	if analyzer.cacheManager != nil {
		analyzer.cacheManager.SetProjectSnapshot(snapshotKey, snapshot)
	}
}

// createProjectSnapshot creates a snapshot of current project state
func (analyzer *CodeAnalyzer) createProjectSnapshot(rootDir string) (*models.ProjectSnapshot, error) {
	snapshot := &models.ProjectSnapshot{
		RootDir:   rootDir,
		Timestamp: time.Now(),
		Files:     make(map[string]models.FileSnapshot),
	}

	// Retrieve gitignore patterns
	gitIgnorePatterns, err := utils.GetGitignorePatterns(rootDir)
	if err != nil {
		return nil, err
	}

	// Walk directory and create file snapshots
	err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		relativePath = strings.ReplaceAll(relativePath, "\\", "/")

		// Skip ignored directories and files
		if utils.IsDefaultIgnored(relativePath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Process only files
		if !d.IsDir() {
			fileInfo, err := os.Stat(path)
			if err != nil {
				return err
			}

			// Skip large files (>100KB)
			if fileInfo.Size() > 100*1024 {
				return nil
			}

			// Skip gitignored files
			if utils.IsGitIgnored(relativePath, gitIgnorePatterns) {
				return nil
			}

			// Create file snapshot
			fileSnapshot := models.FileSnapshot{
				RelativePath: relativePath,
				ModTime:      fileInfo.ModTime(),
				Size:         fileInfo.Size(),
				Hash:         fmt.Sprintf("%d_%d", fileInfo.ModTime().Unix(), fileInfo.Size()),
			}

			snapshot.Files[relativePath] = fileSnapshot
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// compareSnapshots compares two snapshots and returns changed and deleted files
func (analyzer *CodeAnalyzer) compareSnapshots(prevSnapshot, currentSnapshot *models.ProjectSnapshot) ([]string, []string) {
	var changedFiles []string
	var deletedFiles []string

	// Find changed and new files
	for relativePath, currentFile := range currentSnapshot.Files {
		if prevFile, exists := prevSnapshot.Files[relativePath]; exists {
			// Check if file has changed
			if prevFile.Hash != currentFile.Hash {
				changedFiles = append(changedFiles, relativePath)
			}
		} else {
			// New file
			changedFiles = append(changedFiles, relativePath)
		}
	}

	// Find deleted files
	for relativePath := range prevSnapshot.Files {
		if _, exists := currentSnapshot.Files[relativePath]; !exists {
			deletedFiles = append(deletedFiles, relativePath)
		}
	}

	return changedFiles, deletedFiles
}

// processIncrementalChanges processes only the changed files and updates the full result
func (analyzer *CodeAnalyzer) processIncrementalChanges(rootDir string, changedFiles, deletedFiles []string, prevSnapshot *models.ProjectSnapshot) (*models.FullContextData, error) {
	// For simplicity and reliability, let's take a different approach:
	// 1. Start with a fresh scan but only process files efficiently using cache
	// 2. This ensures we always have a complete and consistent result
	
	result := &models.FullContextData{
		FileData: make([]models.FileData, 0),
		RawCodes: make([]string, 0),
	}

	// Get current project snapshot to know all current files
	currentSnapshot, err := analyzer.createProjectSnapshot(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create current snapshot for incremental processing: %w", err)
	}

	// Process all current files (changed files will read from disk, unchanged files from cache)
	for relativePath := range currentSnapshot.Files {
		filePath := filepath.Join(rootDir, relativePath)

		// Try to get cached file content first (for unchanged files)
		var content []byte
		var codeParts []string

		// Check if this file changed
		isChanged := false
		for _, changedFile := range changedFiles {
			if changedFile == relativePath {
				isChanged = true
				break
			}
		}

		if isChanged {
			// File changed - read fresh content and process
			content, err = ioutil.ReadFile(filePath)
			if err != nil {
				continue // Skip files that can't be read
			}

			// Cache the updated file content
			analyzer.cacheManager.SetFileContentCache(filePath, content)

			// Process with tree-sitter
			codeParts = analyzer.ProcessFile(relativePath, content)

			// Cache tree-sitter results
			analyzer.cacheManager.SetTreeSitterCache(filePath, codeParts)
		} else {
			// File unchanged - try to use cache
			if cachedContent, found := analyzer.cacheManager.GetFileContentCache(filePath); found {
				content = cachedContent
			} else {
				// Cache miss - read from disk
				content, err = ioutil.ReadFile(filePath)
				if err != nil {
					continue
				}
				analyzer.cacheManager.SetFileContentCache(filePath, content)
			}

			// Try cached tree-sitter results
			if cachedParts, found := analyzer.cacheManager.GetTreeSitterCache(filePath); found {
				codeParts = cachedParts
			} else {
				// Cache miss - process with tree-sitter
				codeParts = analyzer.ProcessFile(relativePath, content)
				analyzer.cacheManager.SetTreeSitterCache(filePath, codeParts)
			}
		}

		// Add to result
		fileData := models.FileData{
			RelativePath:   relativePath,
			Code:          string(content),
			TreeSitterCode: strings.Join(codeParts, "\n"),
		}

		result.FileData = append(result.FileData, fileData)
		result.RawCodes = append(result.RawCodes, fmt.Sprintf("**File: %s**\n\n%s", relativePath, strings.Join(codeParts, "\n")))
	}

	return result, nil
}

// ProcessFile processes a single file using Tree-sitter for syntax analysis (for .cs files).
func (analyzer *CodeAnalyzer) ProcessFile(filePath string, sourceCode []byte) []string {
	var elements []string

	var parser *sitter.Parser
	var lang *sitter.Language
	var query []byte

	language := utils.GetSupportedLanguage(filePath)
	parser = sitter.NewParser()

	// Determine the parser and language to use
	switch language {
	case "csharp":
		parser.SetLanguage(csharp.GetLanguage())
		lang = csharp.GetLanguage()
		query = embed_data.CSharpQuery
	case "go":
		parser.SetLanguage(golang.GetLanguage())
		lang = golang.GetLanguage()
		query = embed_data.GoQuery
	case "python":
		parser.SetLanguage(python.GetLanguage())
		lang = python.GetLanguage()
		query = embed_data.PythonQuery
	case "java":
		parser.SetLanguage(java.GetLanguage())
		lang = java.GetLanguage()
		query = embed_data.JavaQuery
	case "javascript":
		parser.SetLanguage(javascript.GetLanguage())
		lang = javascript.GetLanguage()
		query = embed_data.JavascriptQuery
	case "typescript":
		parser.SetLanguage(typescript.GetLanguage())
		lang = typescript.GetLanguage()
		query = embed_data.TypescriptQuery
	case "rust":
		// Rust support pending tree-sitter bindings availability
		// For now, process as plain text with basic structure analysis
		elements = append(elements, filePath)
		elements = append(elements, analyzer.extractRustStructure(string(sourceCode)))
		return elements
	case "zig":
		// Zig support pending tree-sitter bindings availability  
		// For now, process as plain text with basic structure analysis
		elements = append(elements, filePath)
		elements = append(elements, analyzer.extractZigStructure(string(sourceCode)))
		return elements
	default:
		// If the language doesn't match, process the original source code directly
		elements = append(elements, filePath)

		lines := strings.Split(string(sourceCode), "\n")
		// Get the first line
		elements = append(elements, lines[0]) // Adding First line from the array

		return elements
	}

	// Parse the source code
	tree := parser.Parse(nil, sourceCode)

	// Parse JSON data into a map
	queries := make(map[string]string)
	err := json.Unmarshal(query, &queries)
	if err != nil {
		log.Fatalf("failed to parse JSON: %v", err)
	}

	// Execute each query and capture results
	for tag, queryStr := range queries {
		query, err := sitter.NewQuery([]byte(queryStr), lang) // Use the appropriate language
		if err != nil {
			log.Fatalf("failed to compile query: %v", err)
		}

		cursor := sitter.NewQueryCursor()
		cursor.Exec(query, tree.RootNode())

		// Collect the results of the query
		for {
			match, ok := cursor.NextMatch()
			if !ok {
				break
			}

			for _, cap := range match.Captures {
				element := cap.Node.Content(sourceCode)
				// Tag the element with its type (e.g., namespace, class, method, interface)
				taggedElement := fmt.Sprintf("%s: %s", tag, element)
				elements = append(elements, taggedElement)
			}
		}
	}

	return elements
}

func (analyzer *CodeAnalyzer) TryGetInCompletedCodeBlocK(relativePaths string) (string, error) {
	var codes []string

	// Simplified regex to capture only the array of files
	re := regexp.MustCompile(`\[.*?\]`)
	match := re.FindString(relativePaths)

	if match == "" {
		return "", fmt.Errorf("no file paths found in input")
	}

	// Parse the match into a slice of strings
	var filePaths []string
	err := json.Unmarshal([]byte(match), &filePaths)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// Loop through each relative path and read the file content
	for _, relativePath := range filePaths {
		content, err := os.ReadFile(relativePath)
		if err != nil {
			continue
		}

		codes = append(codes, fmt.Sprintf("**File: %s**\n\n%s", relativePath, content))
	}

	if len(codes) == 0 {
		return "", fmt.Errorf("no valid files read")
	}

	requestedContext := strings.Join(codes, "\n---------\n\n")

	return requestedContext, nil
}

func (analyzer *CodeAnalyzer) ExtractCodeChanges(diff string) []models.CodeChange {
	filePathPattern := regexp.MustCompile("(?i)(?:\\d+\\.\\s*|File:\\s*)[`']?([^\\s*`']+?\\.[a-zA-Z0-9]+)[`']?\\b")

	lines := strings.Split(diff, "\n")
	var fileChanges []models.CodeChange

	var currentFilePath string
	var currentCodeBlock []string
	var insideCodeBlock bool
	var isTxtFile bool

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Detect a new file path
		if !insideCodeBlock && filePathPattern.MatchString(trimmedLine) {
			// Add the previous file's change if there was one
			if currentFilePath != "" && len(currentCodeBlock) > 0 {
				fileChanges = append(fileChanges, models.CodeChange{
					RelativePath: currentFilePath,
					Code:         strings.Join(currentCodeBlock, "\n"),
				})
				currentCodeBlock = nil
			}

			// Capture the new file path
			matches := filePathPattern.FindStringSubmatch(trimmedLine)
			currentFilePath = matches[1]
			isTxtFile = strings.HasSuffix(currentFilePath, ".md") || strings.HasSuffix(currentFilePath, ".txt")
			continue
		}

		// Start of a code block
		if !isTxtFile && strings.HasPrefix(trimmedLine, "```") {

			if !insideCodeBlock {
				// Start a code block only if a file path is defined
				if currentFilePath != "" {
					insideCodeBlock = true
				}
				continue
			} else {
				// End the code block
				insideCodeBlock = false
				if currentFilePath != "" && len(currentCodeBlock) > 0 {
					fileChanges = append(fileChanges, models.CodeChange{
						RelativePath: currentFilePath,
						Code:         strings.Join(currentCodeBlock, "\n"),
					})
					currentCodeBlock = nil
					currentFilePath = ""
				}
				continue
			}
		}

		if isTxtFile {
			currentCodeBlock = append(currentCodeBlock, line)
		}

		// Collect lines inside a code block
		if insideCodeBlock {
			currentCodeBlock = append(currentCodeBlock, line)
		}
	}

	if isTxtFile {
		// Ensure there are lines to process
		if len(currentCodeBlock) > 2 {
			// Check if the first line contains "```"
			if strings.Contains(currentCodeBlock[0], "```") {
				currentCodeBlock = currentCodeBlock[1:] // Remove the first line
			}
			// Check if the last line contains "```"
			if strings.Contains(currentCodeBlock[len(currentCodeBlock)-1], "```") {
				currentCodeBlock = currentCodeBlock[:len(currentCodeBlock)-1] // Remove the last line
			}
		}
	}

	// Add the last collected code block if any
	if currentFilePath != "" && len(currentCodeBlock) > 0 {
		fileChanges = append(fileChanges, models.CodeChange{
			RelativePath: currentFilePath,
			Code:         strings.Join(currentCodeBlock, "\n"),
			IsTxtFile:    isTxtFile,
		})
	}

	return fileChanges
}

func (analyzer *CodeAnalyzer) ApplyChanges(relativePath, diff string) error {
	// Ensure the directory structure exists
	dir := filepath.Dir(relativePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Process the diff content: handle additions and deletions
	diffLines := strings.Split(diff, "\n")
	var updatedContent []string

	for _, line := range diffLines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "-") {
			// Ignore lines that start with "-", effectively deleting them
			continue
		} else if strings.HasPrefix(trimmedLine, "+") {
			// Add lines that start with "+", but remove the "+" symbol
			updatedContent = append(updatedContent, strings.ReplaceAll(trimmedLine, "+", " "))
		} else {
			// Keep all other lines as they are
			updatedContent = append(updatedContent, line)
		}
	}

	// Handle deletion if code is empty
	if strings.TrimSpace(strings.Join(updatedContent, "\n")) == "" {
		// Check if file exists, then delete if it does
		if err := os.Remove(relativePath); err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("File %s does not exist, so no deletion necessary.\n", relativePath)
			} else {
				return fmt.Errorf("failed to delete file: %w", err)
			}
		}

		// After file deletion, check if the directory is empty and delete it if so
		if err := removeEmptyDirectoryIfNeeded(dir); err != nil {
			return err
		}
	} else {
		// Write the updated content to the file
		if err := ioutil.WriteFile(relativePath, []byte(strings.Join(updatedContent, "\n")), 0644); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
	}

	return nil
}

// removeEmptyDirectoryIfNeeded checks if a directory is empty, and if so, deletes it
func removeEmptyDirectoryIfNeeded(dir string) error {
	// Check if the directory is empty
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	// If the directory is empty, remove it
	if len(entries) == 0 {
		if err := os.Remove(dir); err != nil {
			return fmt.Errorf("failed to delete empty directory %s: %w", dir, err)
		}
	}
	return nil
}

// extractRustStructure extracts basic Rust code structure using regex patterns
func (analyzer *CodeAnalyzer) extractRustStructure(sourceCode string) string {
	var elements []string
	lines := strings.Split(sourceCode, "\n")
	
	// Rust patterns
	fnRegex := regexp.MustCompile(`^\s*(?:pub\s+)?fn\s+(\w+)`)
	structRegex := regexp.MustCompile(`^\s*(?:pub\s+)?struct\s+(\w+)`)
	enumRegex := regexp.MustCompile(`^\s*(?:pub\s+)?enum\s+(\w+)`)
	traitRegex := regexp.MustCompile(`^\s*(?:pub\s+)?trait\s+(\w+)`)
	implRegex := regexp.MustCompile(`^\s*impl(?:\s*<[^>]*>)?\s+(?:\w+\s+for\s+)?(\w+)`)
	modRegex := regexp.MustCompile(`^\s*(?:pub\s+)?mod\s+(\w+)`)
	constRegex := regexp.MustCompile(`^\s*(?:pub\s+)?const\s+(\w+)`)
	staticRegex := regexp.MustCompile(`^\s*(?:pub\s+)?static\s+(\w+)`)
	
	for _, line := range lines {
		if matches := fnRegex.FindStringSubmatch(line); matches != nil {
			elements = append(elements, fmt.Sprintf("function: %s", matches[1]))
		} else if matches := structRegex.FindStringSubmatch(line); matches != nil {
			elements = append(elements, fmt.Sprintf("struct: %s", matches[1]))
		} else if matches := enumRegex.FindStringSubmatch(line); matches != nil {
			elements = append(elements, fmt.Sprintf("enum: %s", matches[1]))
		} else if matches := traitRegex.FindStringSubmatch(line); matches != nil {
			elements = append(elements, fmt.Sprintf("trait: %s", matches[1]))
		} else if matches := implRegex.FindStringSubmatch(line); matches != nil {
			elements = append(elements, fmt.Sprintf("impl: %s", matches[1]))
		} else if matches := modRegex.FindStringSubmatch(line); matches != nil {
			elements = append(elements, fmt.Sprintf("mod: %s", matches[1]))
		} else if matches := constRegex.FindStringSubmatch(line); matches != nil {
			elements = append(elements, fmt.Sprintf("const: %s", matches[1]))
		} else if matches := staticRegex.FindStringSubmatch(line); matches != nil {
			elements = append(elements, fmt.Sprintf("static: %s", matches[1]))
		}
	}
	
	return strings.Join(elements, "\n")
}

// extractZigStructure extracts basic Zig code structure using regex patterns
func (analyzer *CodeAnalyzer) extractZigStructure(sourceCode string) string {
	var elements []string
	lines := strings.Split(sourceCode, "\n")
	
	// Zig patterns
	fnRegex := regexp.MustCompile(`^\s*(?:pub\s+)?fn\s+(\w+)`)
	constRegex := regexp.MustCompile(`^\s*(?:pub\s+)?const\s+(\w+)`)
	varRegex := regexp.MustCompile(`^\s*(?:pub\s+)?var\s+(\w+)`)
	structRegex := regexp.MustCompile(`^\s*(?:pub\s+)?const\s+(\w+)\s*=\s*struct`)
	enumRegex := regexp.MustCompile(`^\s*(?:pub\s+)?const\s+(\w+)\s*=\s*enum`)
	unionRegex := regexp.MustCompile(`^\s*(?:pub\s+)?const\s+(\w+)\s*=\s*union`)
	testRegex := regexp.MustCompile(`^\s*test\s+"([^"]+)"`)
	
	for _, line := range lines {
		if matches := testRegex.FindStringSubmatch(line); matches != nil {
			elements = append(elements, fmt.Sprintf("test: %s", matches[1]))
		} else if matches := structRegex.FindStringSubmatch(line); matches != nil {
			elements = append(elements, fmt.Sprintf("struct: %s", matches[1]))
		} else if matches := enumRegex.FindStringSubmatch(line); matches != nil {
			elements = append(elements, fmt.Sprintf("enum: %s", matches[1]))
		} else if matches := unionRegex.FindStringSubmatch(line); matches != nil {
			elements = append(elements, fmt.Sprintf("union: %s", matches[1]))
		} else if matches := fnRegex.FindStringSubmatch(line); matches != nil {
			elements = append(elements, fmt.Sprintf("function: %s", matches[1]))
		} else if matches := constRegex.FindStringSubmatch(line); matches != nil {
			elements = append(elements, fmt.Sprintf("const: %s", matches[1]))
		} else if matches := varRegex.FindStringSubmatch(line); matches != nil {
			elements = append(elements, fmt.Sprintf("var: %s", matches[1]))
		}
	}
	
	return strings.Join(elements, "\n")
}
