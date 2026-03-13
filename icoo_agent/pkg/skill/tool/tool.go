package tool

import (
	"context"
	"fmt"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"icooclaw/pkg/utils"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type InstallTool struct {
	workspace string
	store     *storage.SkillStorage
}

func NewInstallTool(workspace string, store *storage.SkillStorage) *InstallTool {
	return &InstallTool{
		workspace: workspace,
		store:     store,
	}
}
func (t *InstallTool) Name() string {
	return "skill_install"
}

// Description 获取工具描述
func (t *InstallTool) Description() string {
	return "安装技能"
}

// Parameters 获取工具参数
func (t *InstallTool) Parameters() map[string]any {
	return map[string]any{
		"name": map[string]any{
			"type":        "string",
			"description": "技能名称",
			"required":    true,
		},
		"version": map[string]any{
			"type":        "string",
			"description": "技能版本",
			"required":    true,
		},
		"source": map[string]any{
			"type":        "string",
			"description": "技能来源，例如：github.com/icooclaw/icoo_agent",
			"required":    true,
		},
	}
}

// Execute 执行工具
func (t *InstallTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	name := args["name"].(string)
	// version := args["version"].(string)
	source := args["source"].(string)

	if name == "" {
		return tools.ErrorResult("需要提供 name 参数")
	}

	// 安装技能
	if err := t.InstallFromGitHub(ctx, source); err != nil {
		return tools.ErrorResult(fmt.Sprintf("安装技能 %s 失败: %s", name, err.Error()))
	}

	// 保存技能
	saveData := &storage.Skill{
		Name:        name,
		Description: "",
		Version:     "",
		Path:        filepath.Join(t.workspace, consts.SKILL_DIR, name),
	}
	if err := t.store.SaveSkill(saveData); err != nil {
		return tools.ErrorResult(fmt.Sprintf("保存技能 %s 失败: %w", name, err.Error()))
	}

	return tools.SuccessResult("安装成功")
}

// InstallFromGitHub 从 GitHub 安装技能。
func (t *InstallTool) InstallFromGitHub(ctx context.Context, repo string) error {
	skillDir := filepath.Join(t.workspace, consts.SKILL_DIR, filepath.Base(repo))

	if _, err := os.Stat(skillDir); err == nil {
		return fmt.Errorf("skill '%s' already exists", filepath.Base(repo))
	}

	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/SKILL.md", repo)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %s", err.Error())
	}

	// 请求技能文件
	resp, err := utils.DoRequestWithRetry(client, req)
	if err != nil {
		return fmt.Errorf("failed to fetch skill: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to fetch skill: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")

	// 写文件
	if err := utils.WriteFileAtomic(skillPath, body, 0o600); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	return nil
}
