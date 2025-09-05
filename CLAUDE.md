# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 🚀 Development Commands

**Build the project:**
```bash
go build -v ./...
```

**Run tests:**
```bash
go test -v ./...
```

**Run specific test:**
```bash
go test -v ./code_analyzer -run TestGeneratePrompt
```

**For MSYS2/Windows users:**
```bash
# Set environment variables to avoid linker issues
export TMPDIR="C:/temp" && export TEMP="C:/temp" && export TMP="C:/temp"
mkdir -p /c/temp

# Then run tests normally
go test -v ./code_analyzer
```

**Install globally:**
```bash
go install github.com/meysamhadeli/codai@latest
```

**Run codai:**
```bash
codai code
```

## 🏗️ Architecture Overview

Codai is a Go-based AI coding assistant CLI tool that provides intelligent code suggestions, refactoring, and code reviews using multiple LLM providers.

### Core Components

**1. Command Structure (cmd/)**
- `root.go`: Main CLI entry point with Cobra integration
- `code.go`: Primary "code" subcommand implementation
- Uses dependency injection pattern with `RootDependencies` struct

**2. AI Providers (providers/)**
- Factory pattern for multiple LLM providers (OpenAI, Anthropic, Azure, Ollama, etc.)
- Each provider implements `IChatAIProvider` interface
- Configurable via YAML/JSON config files or environment variables

**3. Code Analysis (code_analyzer/)**
- Tree-sitter integration for syntax-aware code parsing
- Supports multiple languages: Go, Python, Java, JavaScript, TypeScript, C#
- File change extraction and application system
- Project context summarization

**4. Configuration Management (config/)**
- Viper-based configuration with multiple sources (file, env, flags)
- Default provider: OpenAI GPT-4o
- Theme support using Chroma syntax highlighting

**5. Session Management**
- `chat_history/`: Maintains conversational context per session
- `token_management/`: Tracks token usage for each request

### Key Interfaces
- `IChatAIProvider`: AI provider contract for chat completions
- `ICodeAnalyzer`: Code analysis and file manipulation
- `IChatHistory`: Conversation history persistence
- `ITokenManagement`: Token usage tracking

### Data Flow
1. User runs `codai code` in project directory
2. Code analyzer scans project files using tree-sitter
3. Context is summarized and sent to configured AI provider
4. AI responses are parsed for code changes using pattern matching
5. Changes are applied to files with user confirmation
6. Conversation history and token usage are tracked

## 🔧 Configuration

**Environment Variables:**
```bash
export API_KEY="your_api_key"
export PROVIDER="openai"
export MODEL="gpt-4o"
```

**Config File (codai-config.yml):**
```yaml
ai_provider_config:
  provider: "azure-openai"
  base_url: "https://your-endpoint.openai.azure.com"
  model: "gpt-4o"
  api_version: "2024-04-01-preview"
  temperature: 0.2
theme: "dracula"
```

## 🧪 Testing

Tests use testify framework with sequential execution pattern:
- Tests run in specific order via `TestRunInSequence`
- Temporary directories created for each test
- Extensive testing of code change extraction patterns

**Test Structure:**
```bash
code_analyzer/analyzer_test.go  # Main test suite
code_analyzer/cache_test.go     # Cache functionality tests
```

## ⚡ Performance & Caching

### File Caching System
Codai implements an intelligent caching system to improve performance for repeated operations:

**Cache Types:**
- **File Content Cache**: Caches file content based on modification time
- **Tree-sitter Parse Cache**: Caches syntax parsing results  
- **Configuration Cache**: Caches project configuration data
- **Gitignore Pattern Cache**: Caches ignore pattern matching

**Performance Benefits:**
- **Real-world usage**: ~13% performance improvement in typical scenarios
- **Large projects**: More significant gains with complex file structures
- **Repeated scans**: Major time savings for unchanged files

**Cache Location:**
```
~/.codai/cache/  # Default cache directory
```

**Cache Features:**
- Automatic invalidation based on file modification time
- Thread-safe concurrent access
- gob encoding for type-safe serialization
- Configurable cleanup and statistics

## 📦 Dependencies

- **Cobra**: CLI framework
- **Viper**: Configuration management
- **Tree-sitter**: Syntax parsing
- **Chroma**: Syntax highlighting
- **Pterm**: Terminal UI components

## 🎯 Supported AI Providers

- OpenAI
- Azure OpenAI
- Anthropic
- Gemini
- Mistral
- Grok
- Qwen
- DeepSeek
- OpenRouter
- Ollama (local models)

## 🔍 Code Analysis Features

- Multi-language support via tree-sitter
- File pattern matching for change extraction
- Context-aware prompt generation
- Session-based conversation history
- Token usage tracking
- Syntax-aware code processing

## ⚠️ Known Issues

- Windows build may encounter file permission issues during testing
- Some tree-sitter language bindings may have platform-specific limitations
- Complex code change patterns may require manual verification