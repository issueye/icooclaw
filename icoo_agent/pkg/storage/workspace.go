package storage

import (
	"os"
	"path/filepath"
	"strings"
)

// WorkspaceStorage 工作空间存储
type WorkspaceStorage struct {
	workspace string
}

// NewWorkspaceStorage 创建工作空间存储
func NewWorkspaceStorage(workspace string) *WorkspaceStorage {
	return &WorkspaceStorage{workspace: workspace}
}

// GetWorkspace 获取工作空间
func (s *WorkspaceStorage) GetWorkspace() string {
	return s.workspace
}

// SetWorkspace 设置工作空间
func (s *WorkspaceStorage) SetWorkspace(workspace string) {
	s.workspace = workspace
}

// AGENTS 工作空间下的智能体
func (s *WorkspaceStorage) LoadAgent(name string) (string, error) {
	path := filepath.Join(s.workspace, "agents", name+".md")

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// SOUL 人设配置文件
func (s *WorkspaceStorage) LoadSOUL() (string, error) {
	path := filepath.Join(s.workspace, "SOUL.md")

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// USER 用户配置文件
func (s *WorkspaceStorage) LoadUSER() (string, error) {
	path := filepath.Join(s.workspace, "USER.md")

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (s *WorkspaceStorage) LoadWorkspace() (string, error) {
	agentPrompt, err := s.LoadAgent("AGENTS")
	if err != nil {
		return "", err
	}

	soulPrompt, err := s.LoadSOUL()
	if err != nil {
		return "", err
	}

	userPrompt, err := s.LoadUSER()
	if err != nil {
		return "", err
	}

	// 合并智能体、人设和用户配置
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString(agentPrompt)
	sb.WriteString("\n")
	sb.WriteString(soulPrompt)
	sb.WriteString("\n")
	sb.WriteString(userPrompt)
	sb.WriteString("\n")

	return sb.String(), nil
}
