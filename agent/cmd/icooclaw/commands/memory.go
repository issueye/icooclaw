package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// memoryCmd 记忆管理命令
var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "记忆管理",
	Long: `管理用户记忆和偏好设置。

子命令:
  get              获取记忆文件内容
  set-preference  设置用户偏好
  set-fact        设置重要事实
  set-knowledge   设置学到的知识`,
}

// memoryGetCmd 获取记忆文件内容
var memoryGetCmd = &cobra.Command{
	Use:   "get",
	Short: "获取记忆文件内容",
	Long:  "获取 memory/MEMORY.md 文件的完整内容",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		getMemoryFile()
	},
}

// memorySetPrefCmd 设置用户偏好
var memorySetPrefCmd = &cobra.Command{
	Use:   "set-preference <content>",
	Short: "设置用户偏好",
	Long: `设置用户偏好和设置。这些信息会在对话中被 AI 使用。

示例:
  icooclaw memory set-preference "我喜欢用中文交流，称呼我为张三"`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		content := args[0]
		setMemoryContent("用户偏好", content)
	},
}

// memorySetFactCmd 设置重要事实
var memorySetFactCmd = &cobra.Command{
	Use:   "set-fact <content>",
	Short: "设置重要事实",
	Long: `设置重要事实和信息。这些信息会在对话中被 AI 记住。

示例:
  icooclaw memory set-fact "我是一名软件工程师，主要使用 Go 语言"`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		content := args[0]
		setMemoryContent("重要事实", content)
	},
}

// memorySetKnowledgeCmd 设置学到的知识
var memorySetKnowledgeCmd = &cobra.Command{
	Use:   "set-knowledge <content>",
	Short: "设置学到的知识",
	Long: `设置从对话中学习到的知识。

示例:
  icooclaw memory set-knowledge "用户经常问关于 Go 语言的问题"`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		content := args[0]
		setMemoryContent("学到的知识", content)
	},
}

func init() {
	memoryCmd.AddCommand(memoryGetCmd)
	memoryCmd.AddCommand(memorySetPrefCmd)
	memoryCmd.AddCommand(memorySetFactCmd)
	memoryCmd.AddCommand(memorySetKnowledgeCmd)
	rootCmd.AddCommand(memoryCmd)
}

func getMemoryFile() {
	content, err := agentInstance.GetMemoryFile()
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Println("=== memory/MEMORY.md 内容 ===")
	fmt.Println(content)
}

func setMemoryContent(section, content string) {
	err := agentInstance.UpdateMemoryFile(section, content)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Printf("%s 已更新\n", section)
	fmt.Println("内容:", content)
}
