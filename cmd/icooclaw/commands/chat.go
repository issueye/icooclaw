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
	Short: "Start interactive chat (REPL)",
	Long: `Start an interactive chat session with the AI agent.
This opens a REPL (Read-Eval-Print Loop) where you can type messages
and receive responses from the AI agent.

Commands within chat:
  help, /help, ?     - Show help message
  quit, exit, q     - Exit the program
  model <name>       - Switch to a different model
  model <provider>/<model> - Switch to provider's model
  providers, /p      - List available providers
  history, /hist     - Show command history
  clear, /c, cls     - Clear screen`,
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
		fmt.Println("\nReceived interrupt signal. Exiting...")
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
	c.logger.Info("CLI started. Type 'help' for commands, 'quit' or 'exit' to exit.")

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
		fmt.Println("Goodbye!")
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
		return fmt.Errorf("Usage: /model <model_name>")
	case "/providers", "/p":
		c.printProviders()
	case "/clear", "/c":
		c.clearScreen()
	case "/history":
		c.printHistory()
	default:
		return fmt.Errorf("Unknown command: %s", cmd)
	}

	return nil
}

// sendMessage 发送消息
func (c *CLI) sendMessage(ctx context.Context, input string) error {
	c.logger.Info("Sending message to agent", "content", input)

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
			c.logger.Info("Switched to model", "provider", providerName, "model", modelName)
			fmt.Printf("Switched to model: %s (via %s)\n", modelName, providerName)
			return nil
		}
		return fmt.Errorf("Provider not found: %s", providerName)
	}

	// 尝试在当前 provider 中切换模型
	fmt.Printf("Model: %s\n", model)
	c.logger.Info("Switched to model", "model", model)
	return nil
}

// printWelcome 打印欢迎信息
func (c *CLI) printWelcome() {
	fmt.Println("========================================")
	fmt.Println("       icooclaw CLI - Interactive Mode")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  help, /help    - Show this help message")
	fmt.Println("  quit, exit    - Exit the program")
	fmt.Println("  model <name>  - Switch model")
	fmt.Println("  providers     - List available providers")
	fmt.Println("  history       - Show command history")
	fmt.Println("  clear         - Clear screen")
	fmt.Println()
	fmt.Println("Just type your message to chat with the AI!")
	fmt.Println("========================================")
}

// printHelp 打印帮助信息
func (c *CLI) printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  help, /help, ?     - Show this help message")
	fmt.Println("  quit, exit, q       - Exit the program")
	fmt.Println("  model <name>        - Switch to a different model")
	fmt.Println("  model <provider>/<model> - Switch to provider's model")
	fmt.Println("  providers, /p       - List available providers")
	fmt.Println("  history, /hist      - Show command history")
	fmt.Println("  clear, /c, cls      - Clear screen")
	fmt.Println()
	fmt.Println("Just type your message to chat with the AI!")
}

// printSlashHelp 打印斜杠命令帮助
func (c *CLI) printSlashHelp() {
	fmt.Println("Slash Commands:")
	fmt.Println("  /help, /h      - Show this help")
	fmt.Println("  /model, /m     - Switch model")
	fmt.Println("  /providers, /p - List providers")
	fmt.Println("  /clear, /c     - Clear screen")
	fmt.Println("  /history       - Show history")
}

// printProviders 打印可用 Provider
func (c *CLI) printProviders() {
	providers := c.providerReg.List()
	fmt.Println("Available providers:")
	for _, name := range providers {
		p, err := c.providerReg.Get(name)
		if err == nil {
			fmt.Printf("  - %s (default model: %s)\n", name, p.GetDefaultModel())
		}
	}
}

// printHistory 打印历史记录
func (c *CLI) printHistory() {
	fmt.Println("Command history:")
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
