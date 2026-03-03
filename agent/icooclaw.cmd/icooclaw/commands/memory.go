package commands

import (
	"github.com/spf13/cobra"
)

// memoryCmd 记忆管理命令
var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Memory management",
	Long: `Manage user memory and preferences.

Subcommands:
  get             Get memory file content
  set-preference  Set user preference
  set-fact        Set important fact
  set-knowledge   Set learned knowledge`,
}

// memoryGetCmd 获取记忆文件内容
var memoryGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get memory file content",
	Long:  "Get the full content of memory/MEMORY.md file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// if err := checkInitialized(); err != nil {
		// 	return fmt.Errorf("memory get: %w", err)
		// }
		// return getMemoryFile()
		return nil
	},
}

// memorySetPrefCmd 设置用户偏好
var memorySetPrefCmd = &cobra.Command{
	Use:   "set-preference <content>",
	Short: "Set user preference",
	Long: `Set user preferences and settings. This information will be used by AI in conversations.

Example:
  icooclaw memory set-preference "I prefer Chinese, call me Zhang San"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// if err := checkInitialized(); err != nil {
		// 	return fmt.Errorf("memory set-preference: %w", err)
		// }
		// content := args[0]
		// return setMemoryContent("用户偏好", content)
		return nil
	},
}

// memorySetFactCmd 设置重要事实
var memorySetFactCmd = &cobra.Command{
	Use:   "set-fact <content>",
	Short: "Set important fact",
	Long: `Set important facts and information. This information will be remembered by AI in conversations.

Example:
  icooclaw memory set-fact "I am a software engineer, mainly using Go"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// if err := checkInitialized(); err != nil {
		// 	return fmt.Errorf("memory set-fact: %w", err)
		// }
		// content := args[0]
		// return setMemoryContent("重要事实", content)
		return nil
	},
}

// memorySetKnowledgeCmd 设置学到的知识
var memorySetKnowledgeCmd = &cobra.Command{
	Use:   "set-knowledge <content>",
	Short: "Set learned knowledge",
	Long: `Set knowledge learned from conversations.

Example:
  icooclaw memory set-knowledge "User often asks about Go programming"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// if err := checkInitialized(); err != nil {
		// 	return fmt.Errorf("memory set-knowledge: %w", err)
		// }
		// content := args[0]
		// return setMemoryContent("学到的知识", content)
		return nil
	},
}

func init() {
	memoryCmd.AddCommand(memoryGetCmd)
	memoryCmd.AddCommand(memorySetPrefCmd)
	memoryCmd.AddCommand(memorySetFactCmd)
	memoryCmd.AddCommand(memorySetKnowledgeCmd)
	rootCmd.AddCommand(memoryCmd)
}

// // getMemoryFile 获取记忆文件内容
// func getMemoryFile() error {
// 	content, err := agentInstance.GetMemoryFile()
// 	if err != nil {
// 		return fmt.Errorf("get memory file: %w", err)
// 	}

// 	fmt.Println("=== memory/MEMORY.md Content ===")
// 	fmt.Println(content)
// 	return nil
// }

// // setMemoryContent 设置记忆内容
// func setMemoryContent(section, content string) error {
// 	if err := agentInstance.UpdateMemoryFile(section, content); err != nil {
// 		return fmt.Errorf("update memory: %w", err)
// 	}

// 	fmt.Printf("%s updated\n", section)
// 	fmt.Println("Content:", content)
// 	return nil
// }
