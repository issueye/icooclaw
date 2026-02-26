package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// roleCmd 角色设定命令
var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "角色设定管理",
	Long: `管理 AI 角色设定。

子命令:
  set <prompt>     设置角色提示词
  get              获取当前角色提示词
  clear            清除角色提示词`,
}

// roleSetCmd 设置角色提示词
var roleSetCmd = &cobra.Command{
	Use:   "set <prompt>",
	Short: "设置角色提示词",
	Long: `设置 AI 的角色提示词，这个设定会被保存并在后续对话中保持。

示例:
  icooclaw role set "你是一个友好的助手，喜欢用中文回答问题"`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rolePrompt := args[0]
		setRolePrompt(rolePrompt)
	},
}

// roleGetCmd 获取角色提示词
var roleGetCmd = &cobra.Command{
	Use:   "get",
	Short: "获取当前角色提示词",
	Long:  "获取当前设定的角色提示词",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		getRolePrompt()
	},
}

// roleClearCmd 清除角色提示词
var roleClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "清除角色提示词",
	Long:  "清除当前设定的角色提示词",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clearRolePrompt()
	},
}

func init() {
	roleCmd.AddCommand(roleSetCmd)
	roleCmd.AddCommand(roleGetCmd)
	roleCmd.AddCommand(roleClearCmd)
	rootCmd.AddCommand(roleCmd)
}

func setRolePrompt(prompt string) {
	// 为 CLI 创建一个虚拟会话
	session, err := agentInstance.Storage().GetOrCreateSession("cli", "cli-session", "cli-user")
	if err != nil {
		fmt.Printf("错误: 获取会话失败: %v\n", err)
		return
	}

	err = agentInstance.SetSessionRolePrompt(session.ID, prompt)
	if err != nil {
		fmt.Printf("错误: 设置角色提示词失败: %v\n", err)
		return
	}

	fmt.Println("角色提示词已设置:")
	fmt.Println(prompt)
}

func getRolePrompt() {
	// 为 CLI 创建一个虚拟会话
	session, err := agentInstance.Storage().GetOrCreateSession("cli", "cli-session", "cli-user")
	if err != nil {
		fmt.Printf("错误: 获取会话失败: %v\n", err)
		return
	}

	rolePrompt, err := agentInstance.GetSessionRolePrompt(session.ID)
	if err != nil {
		fmt.Printf("错误: 获取角色提示词失败: %v\n", err)
		return
	}

	if rolePrompt == "" {
		fmt.Println("当前未设定角色提示词")
		return
	}

	fmt.Println("当前角色提示词:")
	fmt.Println(rolePrompt)
}

func clearRolePrompt() {
	// 为 CLI 创建一个虚拟会话
	session, err := agentInstance.Storage().GetOrCreateSession("cli", "cli-session", "cli-user")
	if err != nil {
		fmt.Printf("错误: 获取会话失败: %v\n", err)
		return
	}

	err = agentInstance.SetSessionRolePrompt(session.ID, "")
	if err != nil {
		fmt.Printf("错误: 清除角色提示词失败: %v\n", err)
		return
	}

	fmt.Println("角色提示词已清除")
}
