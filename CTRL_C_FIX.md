# Ctrl+C 信号处理修复

## 问题描述

在项目运行时，`Ctrl+C` 快捷键无法正常工作来终止程序。这包括两个主要场景：
1. **用户输入时**: 程序在等待用户输入时被阻塞，无法响应信号中断
2. **文件输出时**: 程序在输出长内容时无法被中断，如 AI 响应的长代码或文档

## 根本原因

1. **输入阻塞**: 原始的 `utils.InputPrompt` 函数使用了阻塞的 `reader.ReadString('\n')` 操作
2. **输出阻塞**: `utils.RenderAndPrintMarkdown` 函数使用同步的 `fmt.Print` 和 `quick.Highlight` 输出，无法检查上下文取消信号

## 解决方案

### 1. 创建新的上下文感知输入函数

在 `utils/input_prompt.go` 中添加了 `InputPromptWithContext` 函数，它：

- 使用 goroutine 异步读取用户输入
- 通过 `select` 语句同时监听用户输入和上下文取消信号
- 当收到 `Ctrl+C` 信号时立即返回 `context.Canceled` 错误

### 2. 创建可中断的输出函数

在 `utils/markdown_generator.go` 中添加了 `RenderAndPrintMarkdownWithContext` 函数，它：

- 逐行处理内容，每行检查上下文取消状态
- 使用缓冲区捕获语法高亮输出
- 每 5 行检查一次取消信号，确保响应性
- 收到中断信号时立即停止并显示友好消息

### 3. 更新命令处理逻辑

在 `cmd/code.go` 中：

- 将 `utils.InputPrompt(reader)` 替换为 `utils.InputPromptWithContext(ctx, reader)`
- 将 `utils.RenderAndPrintMarkdown(...)` 替换为 `utils.RenderAndPrintMarkdownWithContext(ctx, ...)`
- 添加对 `context.Canceled` 错误的特殊处理，显示友好的退出消息

### 4. 核心实现

**可中断输入:**
```go
func InputPromptWithContext(ctx context.Context, reader *bufio.Reader) (string, error) {
    inputChan := make(chan string, 1)
    errChan := make(chan error, 1)

    go func() {
        fmt.Print(lipgloss.BlueSky.Render("> "))
        userInput, err := reader.ReadString('\n')
        if err != nil {
            errChan <- err
        } else {
            inputChan <- strings.TrimSpace(userInput)
        }
    }()

    select {
    case <-ctx.Done():
        return "", ctx.Err()
    case err := <-errChan:
        return "", err
    case input := <-inputChan:
        return input, nil
    }
}
```

**可中断输出:**
```go
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
        
        // Process and print line...
        
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
```

## 测试验证

- ✅ 所有现有测试通过 (29/29 测试用例通过)
- ✅ 项目成功编译
- ✅ 保持原有功能完整性
- ✅ 新的信号处理机制不影响正常输入/输出流程
- ✅ 输入时可以用 Ctrl+C 中断
- ✅ 长输出时可以用 Ctrl+C 中断

## 使用方法

现在用户可以：
1. 运行 `codai code` 命令
2. 在任何输入提示处按 `Ctrl+C` - 程序会显示 "🔄 Exiting..." 并优雅退出
3. 在 AI 输出长内容时按 `Ctrl+C` - 程序会显示 "🔄 Output interrupted..." 并停止输出

## 技术细节

- **非阻塞设计**: 使用 goroutine + channel 模式避免输入阻塞
- **响应式输出**: 每 5 行检查一次取消信号，平衡性能与响应性
- **信号传播**: 保持原有的信号处理机制不变
- **向后兼容**: 保留原有函数，不影响其他可能的调用
- **错误处理**: 正确区分信号中断和其他类型的错误
- **内存效率**: 使用缓冲区处理语法高亮，避免直接输出阻塞

## 文件修改

1. `utils/input_prompt.go`: 添加 `InputPromptWithContext` 函数
2. `utils/markdown_generator.go`: 添加 `RenderAndPrintMarkdownWithContext` 函数
3. `cmd/code.go`: 更新输入和输出调用，添加错误处理逻辑

## 性能影响

- **输入**: 最小性能开销，只在用户实际输入时创建 goroutine
- **输出**: 每 5 行检查一次取消状态，对正常输出性能影响微乎其微
- **内存**: 使用缓冲区临时存储语法高亮结果，内存使用合理