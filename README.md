# Codai

> 🤖 **AI-Powered Coding Assistant for Your Terminal** | [中文版本](./README-zh.md)

![](./assets/codai-demo.gif)


## ✨ Features

### 🤖 AI Provider Support
- **10+ AI Providers**: OpenAI, Anthropic, Gemini, Grok, DeepSeek, Qwen, Mistral, Azure OpenAI, Ollama, and more
- **Flexible Configuration**: Easy switching between providers and models
- **Local AI Support**: Run with local models via Ollama

### 🧠 Intelligent Code Analysis
- **Context-Aware**: Understands your entire project structure and dependencies
- **Tree-sitter Integration**: Advanced syntax parsing for precise code understanding
- **Multi-language Support**: C#, Go, Python, Java, JavaScript, TypeScript, Rust, Zig, and more

### 🚀 Performance & Efficiency
- **Intelligent Caching System**: 13%+ performance improvement for repeated operations
- **Session Management**: Maintains conversation and code context across sessions
- **Token Tracking**: Monitor and optimize AI request costs

### 🔧 Development Workflow
- **Code Generation**: Add new features, functions, and test cases
- **Refactoring**: Improve code structure and efficiency
- **Bug Fixing**: Intelligent bug detection and solution suggestions
- **Code Review**: AI-powered code quality analysis and optimization
- **Documentation**: Generate comprehensive project documentation
- **Multi-file Operations**: Modify multiple files simultaneously

### ⚙️ Configuration & Customization
- **Flexible Config**: YAML config files, environment variables, or CLI parameters
- **Theme Support**: Multiple syntax highlighting themes via Chroma
- **Custom Ignore**: `.codai-gitignore` for fine-grained file filtering

## 🚀 Get Started
To install `codai` globally, you can use the following command:

```bash
go install github.com/meysamhadeli/codai@latest
```

### ⚙️ Zero Setup

**Simply provide your API key, and it just works!**
```bash
export API_KEY="your_api_key"
```


> [!IMPORTANT]
> Codai use **OpenApi** as a default model and with subcommand `--provider` you can choose your appropriate model and use subcommand `--model` for choosing appropriate model of each provider.
> *   [OpenAI](https://platform.openai.com/docs/api-reference/introduction)
> *   [Ollama](https://github.com/ollama/ollama/blob/main/docs/api.md)
> *   [Azure OpenAI](https://learn.microsoft.com/en-us/azure/ai-services/openai/reference)
> *   [Anthropic](https://docs.anthropic.com/en/api/getting-started)
> *   [Gemini](https://ai.google.dev/docs)
> *   [Mistral](https://docs.mistral.ai/)
> *   [Grok](https://docs.x.ai/docs)
> *   [Qwen](https://help.aliyun.com/zh/dashscope/developer-reference/overview)
> *   [DeepSeek](https://platform.deepseek.com/docs)
> *   [OpenRouter](https://openrouter.ai/docs/quick-start)

### 🔧 Advance Configuration
For more advance configuration add a `codai-config.yml` file in the `root of your working directory` or using `environment variables` to set below configs `globally` as a configuration.

The `codai-config` file should be like following example base on your `AI provider`:

**codai-config.yml**
```yml
ai_provider_config:
  provider: "azure-openai"
  base_url: "https://test.openai.azure.com"
  model: "gpt-4o"
  api_version: "2024-04-01-preview"     #(Optional, If your AI provider like 'AzureOpenai' or 'Anthropic' has chat api version.)
  temperature: 0.2     #(Optional, If you want use 'Temperature'.)
  reasoning_effort: "low"     #(Optional, If you want use 'Reasoning'.) 
theme: "dracula"
```

If you wish to customize your configuration, you can create your own `codai-config.yml` file and place it in the `root directory` of `each project` you want to analyze with codai. If `no configuration` file is provided, codai will use the `default settings`.

You can also specify a configuration file from any directory by using the following CLI command:
```bash
codai code --config ./codai-config.yml
```
Additionally, you can pass configuration options directly in the command line. For example:
```bash
codai code --provider openapi --temperature 0.8 --api_key test-key
```
This flexibility allows you to customize config of codai on the fly.


**.codai-gitignore**

Also, you can use `.codai-gitignore` in the `root of your working directory,` and codai will ignore the files that we specify in our `.codai-gitignore`.
> [!NOTE]
> We used [Chroma](https://github.com/alecthomas/chroma) for `style` of our `text` and `code block`, and you can find more theme here in [Chroma Style Gallery](https://xyproto.github.io/splash/docs/) and use it as a `theme` in `codai`.

## ▶️ How to Run
To use `codai` as your code assistant, navigate to the directory where you want to apply codai and run the following command:

```bash
codai code
```
This command will initiate the codai assistant to help you with your coding tasks with understanding the context of your code.

## ⚡ Performance & Caching

### Intelligent File Caching System

Codai implements a sophisticated caching system that significantly improves performance for repeated operations:

**Cache Types:**
- **File Content Cache**: Caches file content based on modification time and file size
- **Tree-sitter Parse Cache**: Caches syntax parsing results to avoid recomputation
- **Configuration Cache**: Caches project configuration data for faster startup
- **Gitignore Pattern Cache**: Caches ignore pattern matching results

**Performance Benefits:**
- **Real-world improvement**: Measured 13% performance boost in typical usage scenarios
- **Large projects**: More significant gains for complex codebases with many files
- **Repeated scans**: Major time savings when analyzing unchanged files multiple times
- **Startup optimization**: Faster project initialization through configuration caching

**Technical Features:**
- **Automatic invalidation**: Cache entries automatically expire when source files are modified
- **Thread-safe**: Concurrent access protected by read-write locks
- **Type-safe serialization**: Uses Go's native `gob` encoding for reliable data persistence
- **Smart cleanup**: Configurable cache expiration and automatic cleanup mechanisms

**Cache Location:**
```
~/.codai/cache/  # Default cache directory
```

The cache is completely transparent to users and requires no manual management. All cache operations happen automatically based on file modification times and checksums.

## 🗺️ Plan
🌀 This project is a work in progress; new features will be added over time. 🌀

I will try to add new features in the [Issues](https://github.com/meysamhadeli/codai/issues) section of this app.

# 🌟 Support

If you like my work, feel free to:

- ⭐ this repository. And we will be happy together :)

Thanks a bunch for supporting me!

## 🤝 Contribution

Thanks to all [contributors](https://github.com/meysamhadeli/codai/graphs/contributors), you're awesome and this wouldn't be possible without you! The goal is to build a categorized, community-driven collection of very well-known resources.

Please follow this [contribution guideline](./CONTRIBUTION.md) to submit a pull request or create the issue.
