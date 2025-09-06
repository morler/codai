# Ctrl+C 信号处理修复

## 问题描述

在项目运行时，`Ctrl+C` 快捷键无法正常工作来终止程序。这是因为程序在等待用户输入时被阻塞，无法响应信号中断。

## 根本原因

原始的 `utils.InputPrompt` 函数使用了阻塞的 `reader.ReadString('\n')` 操作。当程序在等待用户输入时，尽管已经设置了信号处理机制，但程序被阻塞在输入读取上，无法检查 `ctx.Done()` 信号。

## 解决方案

### 1. 创建新的上下文感知输入函数

在 `utils/input_prompt.go` 中添加了 `InputPromptWithContext` 函数，它：

- 使用 goroutine 异步读取用户输入
- 通过 `select` 语句同时监听用户输入和上下文取消信号
- 当收到 `Ctrl+C` 信号时立即返回 `context.Canceled` 错误

### 2. 更新命令处理逻辑

在 `cmd/code.go` 中：

- 将 `utils.InputPrompt(reader)` 替换为 `utils.InputPromptWithContext(ctx, reader)`
- 添加对 `context.Canceled` 错误的特殊处理，显示友好的退出消息

### 3. 核心实现

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

## 测试验证

- ✅ 所有现有测试通过
- ✅ 项目成功编译
- ✅ 保持原有功能完整性
- ✅ 新的信号处理机制不影响正常输入流程

## 使用方法

现在用户可以：
1. 运行 `codai code` 命令
2. 在任何输入提示处按 `Ctrl+C`
3. 程序会显示 "🔄 Exiting..." 并优雅退出

## 技术细节

- **非阻塞设计**: 使用 goroutine + channel 模式避免输入阻塞
- **信号传播**: 保持原有的信号处理机制不变
- **向后兼容**: 保留原有的 `InputPrompt` 函数，不影响其他可能的调用
- **错误处理**: 正确区分信号中断和其他类型的错误

## 文件修改

1. `utils/input_prompt.go`: 添加 `InputPromptWithContext` 函数
2. `cmd/code.go`: 更新输入调用和错误处理逻辑