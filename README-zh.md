# Codai

> 🤖 **终端中的AI编程助手** | [English Version](./README.md)

![](./assets/codai-demo.gif)

## ✨ 特性

### 🤖 AI提供商支持
- **10+AI提供商**: OpenAI、Anthropic、Gemini、Grok、DeepSeek、通义千问、Mistral、Azure OpenAI、Ollama等
- **灵活配置**: 轻松切换不同的提供商和模型
- **本地AI支持**: 通过Ollama运行本地模型

### 🧠 智能代码分析
- **上下文感知**: 理解整个项目结构和依赖关系
- **Tree-sitter集成**: 高级语法解析，实现精准代码理解
- **多语言支持**: C#、Go、Python、Java、JavaScript、TypeScript、Rust、Zig等

### 🚀 性能与效率
- **智能缓存系统**: 重复操作性能提升13%+
- **会话管理**: 跨会话维护对话和代码上下文
- **Token跟踪**: 监控和优化AI请求成本

### 🔧 开发工作流
- **代码生成**: 添加新功能、函数和测试用例
- **重构代码**: 改善代码结构和效率
- **Bug修复**: 智能错误检测和解决方案建议
- **代码审查**: AI驱动的代码质量分析和优化
- **文档生成**: 生成全面的项目文档
- **多文件操作**: 同时修改多个文件

### ⚙️ 配置与定制
- **灵活配置**: YAML配置文件、环境变量或CLI参数
- **主题支持**: 通过Chroma支持多种语法高亮主题
- **自定义忽略**: `.codai-gitignore`实现细粒度文件过滤

## 🚀 快速开始

要全局安装`codai`，可以使用以下命令：

```bash
go install github.com/meysamhadeli/codai@latest
```

### ⚙️ 零配置开始

**只需提供API密钥，即可开始使用！**
```bash
export API_KEY="your_api_key"
```

> [!IMPORTANT]
> Codai默认使用**OpenAI**作为模型，你可以通过`--provider`子命令选择合适的模型，使用`--model`子命令为每个提供商选择合适的模型。
> *   [OpenAI](https://platform.openai.com/docs/api-reference/introduction)
> *   [Ollama](https://github.com/ollama/ollama/blob/main/docs/api.md)
> *   [Azure OpenAI](https://learn.microsoft.com/en-us/azure/ai-services/openai/reference)
> *   [Anthropic](https://docs.anthropic.com/en/api/getting-started)
> *   [Gemini](https://ai.google.dev/docs)
> *   [Mistral](https://docs.mistral.ai/)
> *   [Grok](https://docs.x.ai/docs)
> *   [通义千问](https://help.aliyun.com/zh/dashscope/developer-reference/overview)
> *   [DeepSeek](https://platform.deepseek.com/docs)
> *   [OpenRouter](https://openrouter.ai/docs/quick-start)

### 🔧 高级配置

对于更高级的配置，在`工作目录的根目录`添加`codai-config.yml`文件，或使用`环境变量`全局设置以下配置。

根据你的`AI提供商`，`codai-config`文件应该如下例所示：

**codai-config.yml**
```yml
ai_provider_config:
  provider: "azure-openai"
  base_url: "https://test.openai.azure.com"
  model: "gpt-4o"
  api_version: "2024-04-01-preview"     #（可选，如果你的AI提供商如'AzureOpenai'或'Anthropic'有聊天API版本）
  temperature: 0.2     #（可选，如果你想使用'Temperature'）
  reasoning_effort: "low"     #（可选，如果你想使用'Reasoning'）
theme: "dracula"
```

如果你希望自定义配置，可以创建自己的`codai-config.yml`文件并将其放置在要使用codai分析的`每个项目`的`根目录`中。如果`没有提供配置`文件，codai将使用`默认设置`。

你也可以通过以下CLI命令从任何目录指定配置文件：
```bash
codai code --config ./codai-config.yml
```

此外，你可以直接在命令行中传递配置选项。例如：
```bash
codai code --provider openapi --temperature 0.8 --api_key test-key
```
这种灵活性允许你随时自定义codai的配置。

**.codai-gitignore**

另外，你可以在`工作目录的根目录`使用`.codai-gitignore`，codai将忽略我们在`.codai-gitignore`中指定的文件。

> [!NOTE]
> 我们使用[Chroma](https://github.com/alecthomas/chroma)来设置`文本`和`代码块`的`样式`，你可以在[Chroma样式画廊](https://xyproto.github.io/splash/docs/)中找到更多主题，并在`codai`中将其用作`主题`。

## ▶️ 如何运行

要使用`codai`作为你的代码助手，导航到想要应用codai的目录并运行以下命令：

```bash
codai code
```
此命令将启动codai助手来帮助你处理编程任务，同时理解你代码的上下文。

## ⚡ 性能与缓存

### 智能文件缓存系统

Codai实现了一个复杂的缓存系统，显著提高了重复操作的性能：

**缓存类型：**
- **文件内容缓存**：基于修改时间和文件大小缓存文件内容
- **Tree-sitter解析缓存**：缓存语法解析结果以避免重新计算
- **配置缓存**：缓存项目配置数据以加快启动速度
- **Gitignore模式缓存**：缓存忽略模式匹配结果

**性能优势：**
- **实际改善**：在典型使用场景中测得13%的性能提升
- **大型项目**：对于具有许多文件的复杂代码库有更显著的改善
- **重复扫描**：多次分析未更改文件时节省大量时间
- **启动优化**：通过配置缓存实现更快的项目初始化

**技术特性：**
- **自动失效**：当源文件被修改时，缓存条目自动过期
- **线程安全**：通过读写锁保护并发访问
- **类型安全序列化**：使用Go的原生`gob`编码实现可靠的数据持久化
- **智能清理**：可配置的缓存过期和自动清理机制

**缓存位置：**
```
~/.codai/cache/  # 默认缓存目录
```

缓存对用户完全透明，无需手动管理。所有缓存操作基于文件修改时间和校验和自动进行。

## 🗺️ 规划

🌀 这个项目正在进行中；随着时间推移将添加新功能。🌀

我会尽力在此应用的[Issues](https://github.com/meysamhadeli/codai/issues)部分添加新功能。

## 🌟 支持

如果你喜欢我的工作，欢迎：

- ⭐ 给这个仓库点星。我们将一起开心 :)

非常感谢你的支持！

## 🤝 贡献

感谢所有[贡献者](https://github.com/meysamhadeli/codai/graphs/contributors)，你们很棒，没有你们这一切都不可能！目标是建立一个分类的、社区驱动的知名资源集合。

请遵循此[贡献指南](./CONTRIBUTION.md)来提交Pull Request或创建Issue。

## 📄 许可证

本项目使用Apache 2.0许可证。详情请参见[LICENSE](./LICENSE)文件。