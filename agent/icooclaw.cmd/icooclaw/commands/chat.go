package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"icooclaw.ai/agent"
	"icooclaw.ai/provider"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start interactive chat (REPL)",
	Long: `Start an interactive chat session with the AI agent.

This opens a REPL (Read-Eval-Print Loop) where you can input messages
and receive responses from the AI agent.

Chat commands:
  help, /help, ?           - Show help
  quit, exit, q            - Quit the program
  model <name>             - Switch model
  model <provider>/<model> - Switch to provider's model
  providers, /p            - List available providers
  history, /hist           - Show command history
  clear, /c, cls           - Clear screen`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// if err := checkInitialized(); err != nil {
		// 	return fmt.Errorf("chat: %w", err)
		// }
		return runChat()
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)
}

// runChat 运行聊天模式
func runChat() error {
	// ctx, cancel := context.WithCancel(context.Background())

	// 处理信号
	// sigCh := make(chan os.Signal, 1)
	// signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	// go func() {
	// 	<-sigCh
	// 	fmt.Println("\nReceived interrupt signal. Exiting...")
	// 	cancel()
	// }()

	// 启动 Agent
	// go agentInstance.Run(ctx, messageBus)

	// 打印可用的 Provider
	// printProviders()

	// 运行 CLI
	// cli := NewCLI(agentInstance, providerReg, logger)
	// cli.Run(ctx)

	// 清理
	// cleanup()
	return nil
}

// CLI REPL 交互模式
type CLI struct {
	agent       *agent.Agent
	providerReg *provider.Registry
	history     []string
	scanner     *bufio.Scanner
	logger      *slog.Logger
	running     bool
}

// NewCLI 创建 CLI
func NewCLI(agentInstance *agent.Agent, providerReg *provider.Registry, logger *slog.Logger) *CLI {
	return &CLI{
		agent:       agentInstance,
		providerReg: providerReg,
		history:     make([]string, 0),
		scanner:     bufio.NewScanner(os.Stdin),
		logger:      logger,
		running:     true,
	}
}

// Run 运行 CLI
func (c *CLI) Run(ctx context.Context) {
	c.logger.Info("CLI started. Type 'help' for commands, 'quit' or 'exit' to exit.")
	c.printWelcome()

	for c.running {
		fmt.Print("\n> ")

		if !c.scanner.Scan() {
			break
		}

		input := strings.TrimSpace(c.scanner.Text())
		if input == "" {
			continue
		}

		// 添加到历史
		c.history = append(c.history, input)

		// 处理命令
		if err := c.handleCommand(ctx, input); err != nil {
			c.logger.Error("Command error", "error", err)
			fmt.Printf("Error: %v\n", err)
		}
	}
}

// handleCommand 处理命令
func (c *CLI) handleCommand(ctx context.Context, input string) error {
	// 检查斜杠命令
	if strings.HasPrefix(input, "/") {
		return c.handleSlashCommand(ctx, input)
	}

	lower := strings.ToLower(input)

	// 检查内置命令
	switch {
	case lower == "quit" || lower == "exit" || lower == "q":
		c.running = false
		fmt.Println("Goodbye!")
		return nil

	case lower == "help" || lower == "h" || lower == "?":
		c.printHelp()
		return nil

	case lower == "history" || lower == "hist":
		c.printHistory()
		return nil

	case lower == "providers" || lower == "provider":
		c.printProviders()
		return nil

	case lower == "clear" || lower == "cls":
		c.clearScreen()
		return nil

	case strings.HasPrefix(lower, "model "):
		model := strings.TrimPrefix(input, "model ")
		return c.switchModel(model)
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
		if len(args) == 0 {
			return errors.New("usage: /model <model_name>")
		}
		return c.switchModel(args[0])
	case "/providers", "/p":
		c.printProviders()
	case "/clear", "/c":
		c.clearScreen()
	case "/history":
		c.printHistory()
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}

	return nil
}

// sendMessage 发送消息
func (c *CLI) sendMessage(ctx context.Context, input string) error {
	c.logger.Info("Sending message to agent", "content", truncateMessage(input, 50))

	resp, err := c.agent.ProcessMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("agent error: %w", err)
	}

	fmt.Printf("\n%s\n", resp)
	return nil
}

// switchModel 切换模型
func (c *CLI) switchModel(model string) error {
	// 格式: provider/model 或 model
	if strings.Contains(model, "/") {
		parts := strings.SplitN(model, "/", 2)
		providerName := parts[0]
		modelName := parts[1]

		p, err := c.providerReg.Get(providerName)
		if err != nil {
			return fmt.Errorf("provider not found: %s", providerName)
		}

		// 切换 Provider
		c.agent.SetProvider(p)
		fmt.Printf("✓ Switched to %s (model: %s)\n", providerName, modelName)
		return nil
	}

	// 仅切换模型名称（使用当前 provider）
	currentProvider := c.agent.Provider()
	if currentProvider == nil {
		return errors.New("no provider available")
	}

	// 创建新的 provider 实例（使用新模型）
	newProvider := provider.NewOpenAICompatibleProvider(
		currentProvider.GetName(),
		currentProvider.GetAPIKey(),
		currentProvider.GetAPIBase(),
		model,
	)
	c.agent.SetProvider(newProvider)
	fmt.Printf("✓ Model switched to: %s\n", model)
	return nil
}

// printWelcome 打印欢迎信息
func (c *CLI) printWelcome() {
	fmt.Println("========================================")
	fmt.Println("       icooclaw CLI - Interactive Mode")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  help, /help    - Show help")
	fmt.Println("  quit, exit     - Exit program")
	fmt.Println("  model <name>   - Switch model")
	fmt.Println("  providers      - List providers")
	fmt.Println("  history        - Show history")
	fmt.Println("  clear          - Clear screen")
	fmt.Println()
	fmt.Println("Type a message to chat with AI!")
	fmt.Println("========================================")
}

// printHelp 打印帮助信息
func (c *CLI) printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  help, /help, ?       - Show help")
	fmt.Println("  quit, exit, q        - Exit program")
	fmt.Println("  model <name>         - Switch model")
	fmt.Println("  model <provider>/<model> - Switch provider's model")
	fmt.Println("  providers, /p        - List providers")
	fmt.Println("  history, /hist       - Show history")
	fmt.Println("  clear, /c, cls       - Clear screen")
	fmt.Println()
	fmt.Println("Type a message to chat with AI!")
}

// printSlashHelp 打印斜杠命令帮助
func (c *CLI) printSlashHelp() {
	fmt.Println("Slash commands:")
	fmt.Println("  /help, /h       - Show help")
	fmt.Println("  /model, /m      - Switch model")
	fmt.Println("  /providers, /p  - List providers")
	fmt.Println("  /clear, /c      - Clear screen")
	fmt.Println("  /history        - Show history")
}

// printProviders 打印可用 Provider
func (c *CLI) printProviders() {
	providers := c.providerReg.List()
	fmt.Println("Available providers:")
	for _, name := range providers {
		p, err := c.providerReg.Get(name)
		if err != nil {
			continue
		}
		fmt.Printf("  - %s (default model: %s)\n", name, p.GetDefaultModel())
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
