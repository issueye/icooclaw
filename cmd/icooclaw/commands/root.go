package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/icooclaw/icooclaw/internal/agent"
	"github.com/icooclaw/icooclaw/internal/agent/tools"
	"github.com/icooclaw/icooclaw/internal/bus"
	"github.com/icooclaw/icooclaw/internal/channel"
	"github.com/icooclaw/icooclaw/internal/config"
	"github.com/icooclaw/icooclaw/internal/provider"
	"github.com/icooclaw/icooclaw/internal/scheduler"
	"github.com/icooclaw/icooclaw/internal/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// Global variables for initialized components
var (
	cfg            *config.Config
	logger         *slog.Logger
	db             *gorm.DB
	messageBus     *bus.MessageBus
	providerReg    *provider.Registry
	channelManager *channel.Manager
	agentInstance  *agent.Agent
	schedulerInst  *scheduler.Scheduler
	toolRegistry   *tools.Registry
)

// RootCmd represents the root command
var rootCmd = &cobra.Command{
	Use:   "icooclaw",
	Short: "icooclaw - AI Agent CLI",
	Long: `icooclaw is an AI agent CLI tool that provides various commands
for interacting with AI agents, managing scheduled tasks, and more.

Examples:
  icooclaw serve        # Start the web server (WebSocket/Webhook)
  icooclaw chat         # Start interactive chat (REPL)
  icooclaw run "hello"  # Run a single message
  icooclaw cron list    # List scheduled tasks
  icooclaw config get   # Get configuration`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip initialization for help, version, and completion commands
		if cmd.Name() == "help" || cmd.Name() == "version" || cmd.Name() == "completion" {
			return nil
		}
		return initComponents()
	},
}

// initComponents initializes all required components
func initComponents() error {
	var err error

	// 1. Initialize config
	cfg, err = config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 2. Initialize logger
	logger = config.InitLogger(cfg.Log.Level, cfg.Log.Format)
	slog.SetDefault(logger)

	// 3. Initialize database
	db, err = storage.InitDB(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	logger.Info("Database initialized", "path", cfg.Database.Path)

	// 4. Initialize message bus
	messageBus = bus.NewMessageBus()
	messageBus.SetLogger(logger)
	logger.Info("Message bus initialized")

	// 5. Initialize provider registry
	providerReg = provider.NewRegistry()
	if err := providerReg.RegisterFromConfig(cfg.Providers); err != nil {
		return fmt.Errorf("failed to register providers: %w", err)
	}
	logger.Info("Providers registered", "count", providerReg.Count())

	// 6. Initialize channel manager
	channelManager = channel.NewManager(messageBus, cfg.Channels, db, logger)
	logger.Info("Channels registered", "count", channelManager.Count())

	// 7. Create default agent
	agentConfig := cfg.Agents.Defaults
	defaultProviderName := cfg.Agents.DefaultProvider
	if defaultProviderName == "" {
		providers := providerReg.List()
		if len(providers) > 0 {
			defaultProviderName = providers[0]
		}
	}

	defaultProvider, err := providerReg.Get(defaultProviderName)
	if err != nil {
		return fmt.Errorf("failed to get default provider: %w", err)
	}

	agentInstance = agent.NewAgent(
		agentConfig.Name,
		defaultProvider,
		storage.NewStorage(db),
		agentConfig,
		logger,
	)

	// 8. Initialize tools
	toolRegistry = tools.InitTools(cfg, logger, channelManager)
	agentInstance.SetTools(toolRegistry)

	// 9. Initialize scheduler
	schedulerInst = scheduler.NewScheduler(messageBus, storage.NewStorage(db), cfg, logger)

	return nil
}

// getContext creates a cancellable context
func getContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// handleSignals sets up signal handling
func handleSignals(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt signal. Shutting down...")
		cancel()
	}()
}

// cleanup performs cleanup operations
func cleanup() {
	logger.Info("Shutting down...")

	// Stop WebSocket channel
	if channelManager != nil {
		if wsCh, err := channelManager.Get("websocket"); err == nil {
			wsCh.Stop()
		}
	}

	// Stop scheduler
	if schedulerInst != nil && schedulerInst.IsRunning() {
		schedulerInst.Stop()
	}

	logger.Info("Shutdown complete")
}

// printProviders prints available providers
func printProviders() {
	providers := providerReg.List()
	fmt.Println("Available providers:")
	for _, name := range providers {
		p, _ := providerReg.Get(name)
		fmt.Printf("  - %s: %s\n", name, p.GetDefaultModel())
	}
}

func Execute() error {
	// Add persistent flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "Config file path")
	rootCmd.PersistentFlags().StringP("log-level", "l", "", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")

	// Bind flags to viper
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	return rootCmd.Execute()
}
