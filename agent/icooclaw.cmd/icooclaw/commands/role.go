package commands

import (
	"github.com/spf13/cobra"
)

// roleCmd 角色设定命令
var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "Role management",
	Long: `Manage AI role settings.

Subcommands:
  set <prompt>   Set role prompt
  get            Get current role prompt
  clear          Clear role prompt`,
}

// roleSetCmd 设置角色提示词
var roleSetCmd = &cobra.Command{
	Use:   "set <prompt>",
	Short: "Set role prompt",
	Long: `Set AI role prompt. This setting will be saved and persist in subsequent conversations.

Example:
  icooclaw role set "You are a friendly assistant who likes to answer questions in Chinese"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// if err := checkInitialized(); err != nil {
		// 	return fmt.Errorf("role set: %w", err)
		// }
		// rolePrompt := args[0]
		// return setRolePrompt(rolePrompt)
		return nil
	},
}

// roleGetCmd 获取角色提示词
var roleGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get current role prompt",
	Long:  "Get the currently set role prompt",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// if err := checkInitialized(); err != nil {
		// 	return fmt.Errorf("role get: %w", err)
		// }
		// return getRolePrompt()
		return nil
	},
}

// roleClearCmd 清除角色提示词
var roleClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear role prompt",
	Long:  "Clear the currently set role prompt",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// if err := checkInitialized(); err != nil {
		// 	return fmt.Errorf("role clear: %w", err)
		// }
		// return clearRolePrompt()
		return nil
	},
}

func init() {
	roleCmd.AddCommand(roleSetCmd)
	roleCmd.AddCommand(roleGetCmd)
	roleCmd.AddCommand(roleClearCmd)
	rootCmd.AddCommand(roleCmd)
}

// // setRolePrompt 设置角色提示词
// func setRolePrompt(prompt string) error {
// 	// 为 CLI 创建虚拟会话
// 	session, err := agentInstance.Storage().GetOrCreateSession("cli", "cli-session", "cli-user")
// 	if err != nil {
// 		return fmt.Errorf("get session: %w", err)
// 	}

// 	if err := agentInstance.SetSessionRolePrompt(session.ID, prompt); err != nil {
// 		return fmt.Errorf("set role prompt: %w", err)
// 	}

// 	fmt.Println("Role prompt set:")
// 	fmt.Println(prompt)
// 	return nil
// }

// // getRolePrompt 获取角色提示词
// func getRolePrompt() error {
// 	// 为 CLI 创建虚拟会话
// 	session, err := agentInstance.Storage().GetOrCreateSession("cli", "cli-session", "cli-user")
// 	if err != nil {
// 		return fmt.Errorf("get session: %w", err)
// 	}

// 	rolePrompt, err := agentInstance.GetSessionRolePrompt(session.ID)
// 	if err != nil {
// 		return fmt.Errorf("get role prompt: %w", err)
// 	}

// 	if rolePrompt == "" {
// 		fmt.Println("No role prompt set")
// 		return nil
// 	}

// 	fmt.Println("Current role prompt:")
// 	fmt.Println(rolePrompt)
// 	return nil
// }

// // clearRolePrompt 清除角色提示词
// func clearRolePrompt() error {
// 	// 为 CLI 创建虚拟会话
// 	session, err := agentInstance.Storage().GetOrCreateSession("cli", "cli-session", "cli-user")
// 	if err != nil {
// 		return fmt.Errorf("get session: %w", err)
// 	}

// 	if err := agentInstance.SetSessionRolePrompt(session.ID, ""); err != nil {
// 		return fmt.Errorf("clear role prompt: %w", err)
// 	}

// 	fmt.Println("Role prompt cleared")
// 	return nil
// }
