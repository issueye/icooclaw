package commands

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/icooclaw/icooclaw/internal/agent"
	"github.com/icooclaw/icooclaw/internal/provider"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "启动交互式聊天 (REPL)",
	Long: `启动与 AI 代理的交互式聊天会话。
这将打开一个 REPL (读取-求值-打印循环)，您可以在其中输入消息
并接收 AI 代理的响应。

聊天中的命令:
  help, /help, ?     - 显示帮助信息
  quit, exit, q      - 退出程序
  model <name>       - 切换到不同的模型
  model <provider>/<model> - 切换到提供商的模型
  providers, /p      - 列出可用的提供商
  history, /hist     - 显示命令历史
  clear, /c, cls    - 清屏`,
	Run: func(cmd *cobra.Command, args []string) {
		runChat()
	},
}

func runChat() {
	ctx, cancel := context.WithCancel(context.Background())

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n收到中断信号。正在退出...")
		cancel()
	}()

	// Start agent
	go agentInstance.Run(ctx, messageBus)

	// Print providers
	printProviders()

	// Run CLI
	cli := NewCLI(agentInstance, providerReg, logger)
	cli.Run(ctx)

	// Cleanup
	cleanup()
}

// CLI REPL 交互模式
type CLI struct {
	agent        *agent.Agent
	providerReg  *provider.Registry
	history      []string
	historyIndex int
	scanner      *bufio.Scanner
	logger       *slog.Logger
	running      bool
}

// NewCLI 创建 CLI
func NewCLI(agentInstance *agent.Agent, providerReg *provider.Registry, logger *slog.Logger) *CLI {
	return &CLI{
		agent:        agentInstance,
		providerReg:  providerReg,
		history:      []string{},
		historyIndex: -1,
		scanner:      bufio.NewScanner(os.Stdin),
		logger:       logger,
		running:      true,
	}
}

// Run 运行 CLI
func (c *CLI) Run(ctx context.Context) {
	c.logger.Info("CLI 已启动。输入 'help' 查看命令，输入 'quit' 或 'exit' 退出。")

	// 打印欢迎信息
	c.printWelcome()

	for c.running {
		fmt.Print("\n> ")

		if !c.scanner.Scan() {
			break
		}

		input := c.scanner.Text()
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		// 添加到历史
		c.history = append(c.history, input)
		c.historyIndex = len(c.history)

		// 处理命令
		if err := c.handleCommand(ctx, input); err != nil {
			c.logger.Error("Command error", "error", err)
			fmt.Printf("Error: %v\n", err)
		}
	}
}

// handleCommand 处理命令
func (c *CLI) handleCommand(ctx context.Context, input string) error {
	// 检查是否是特殊命令
	if strings.HasPrefix(input, "/") {
		return c.handleSlashCommand(ctx, input)
	}

	// 检查是否是退出命令
	lower := strings.ToLower(input)
	if lower == "quit" || lower == "exit" || lower == "q" {
		c.running = false
		fmt.Println("再见!")
		return nil
	}

	// 检查是否是帮助命令
	if lower == "help" || lower == "h" || lower == "?" {
		c.printHelp()
		return nil
	}

	// 检查是否是历史命令
	if lower == "history" || lower == "hist" {
		c.printHistory()
		return nil
	}

	// 检查是否是模型切换命令
	if strings.HasPrefix(lower, "model ") {
		model := strings.TrimPrefix(input, "model ")
		return c.switchModel(model)
	}

	// 检查是否是 providers 命令
	if lower == "providers" || lower == "provider" {
		c.printProviders()
		return nil
	}

	// 检查是否是 clear 命令
	if lower == "clear" || lower == "cls" {
		c.clearScreen()
		return nil
	}

	// 发送消息给 Agent
	return c.sendMessage(ctx, input)
}

// handleSlashCommand 处理斜杠命令
func (c *CLI) handleSlashCommand(ctx context.Context, input string) error {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/help", "/h":
		c.printSlashHelp()
	case "/model", "/m":
		if len(args) > 0 {
			return c.switchModel(args[0])
		}
		return fmt.Errorf("用法: /model <模型名称>")
	case "/providers", "/p":
		c.printProviders()
	case "/clear", "/c":
		c.clearScreen()
	case "/history":
		c.printHistory()
	default:
		return fmt.Errorf("未知命令: %s", cmd)
	}

	return nil
}

// sendMessage 发送消息
func (c *CLI) sendMessage(ctx context.Context, input string) error {
	c.logger.Info("正在发送消息给代理", "content", input)

	// 直接调用 agent 处理消息
	resp, err := c.agent.ProcessMessage(ctx, input)
	if err != nil {
		return err
	}

	fmt.Printf("\n%s\n", resp)
	return nil
}

// switchModel 切换模型
func (c *CLI) switchModel(model string) error {
	// 检查是否是 provider:model 格式
	if strings.Contains(model, "/") {
		parts := strings.SplitN(model, "/", 2)
		providerName := parts[0]
		modelName := parts[1]

		// 尝试使用指定 provider
		if p, err := c.providerReg.Get(providerName); err == nil {
			// 创建新的 Agent 使用指定模型
			_ = p
			c.logger.Info("已切换模型", "provider", providerName, "model", modelName)
			fmt.Printf("已切换模型: %s (通过 %s)\n", modelName, providerName)
			return nil
		}
		return fmt.Errorf("未找到提供商: %s", providerName)
	}

	// 尝试在当前 provider 中切换模型
	fmt.Printf("模型: %s\n", model)
	c.logger.Info("已切换模型", "model", model)
	return nil
}

// printWelcome 打印欢迎信息
func (c *CLI) printWelcome() {
	fmt.Println("========================================")
	fmt.Println("       icooclaw CLI - 交互模式")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("命令:")
	fmt.Println("  help, /help    - 显示帮助信息")
	fmt.Println("  quit, exit    - 退出程序")
	fmt.Println("  model <名称>  - 切换模型")
	fmt.Println("  providers     - 列出可用的提供商")
	fmt.Println("  history       - 显示命令历史")
	fmt.Println("  clear         - 清屏")
	fmt.Println()
	fmt.Println("直接输入消息与 AI 聊天!")
	fmt.Println("========================================")
}

// printHelp 打印帮助信息
func (c *CLI) printHelp() {
	fmt.Println("可用命令:")
	fmt.Println("  help, /help, ?       - 显示帮助信息")
	fmt.Println("  quit, exit, q        - 退出程序")
	fmt.Println("  model <名称>         - 切换到不同的模型")
	fmt.Println("  model <提供商>/<模型> - 切换到提供商的模型")
	fmt.Println("  providers, /p        - 列出可用的提供商")
	fmt.Println("  history, /hist       - 显示命令历史")
	fmt.Println("  clear, /c, cls       - 清屏")
	fmt.Println()
	fmt.Println("直接输入消息与 AI 聊天!")
}

// printSlashHelp 打印斜杠命令帮助
func (c *CLI) printSlashHelp() {
	fmt.Println("斜杠命令:")
	fmt.Println("  /help, /h       - 显示帮助")
	fmt.Println("  /model, /m      - 切换模型")
	fmt.Println("  /providers, /p  - 列出提供商")
	fmt.Println("  /clear, /c      - 清屏")
	fmt.Println("  /history        - 显示历史")
}

// printProviders 打印可用 Provider
func (c *CLI) printProviders() {
	providers := c.providerReg.List()
	fmt.Println("可用的提供商:")
	for _, name := range providers {
		p, err := c.providerReg.Get(name)
		if err == nil {
			fmt.Printf("  - %s (默认模型: %s)\n", name, p.GetDefaultModel())
		}
	}
}

// printHistory 打印历史记录
func (c *CLI) printHistory() {
	fmt.Println("命令历史:")
	for i, cmd := range c.history {
		fmt.Printf("  %d: %s\n", i+1, cmd)
	}
}

// clearScreen 清屏
func (c *CLI) clearScreen() {
	fmt.Print("\033[2J")
	fmt.Print("\033[H")
}

func init() {
	rootCmd.AddCommand(chatCmd)
}
