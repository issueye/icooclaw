// Package builtin provides built-in tools for icooclaw.
package builtin

import (
	"os"

	"icooclaw/pkg/tools"
	"icooclaw/pkg/tools/builtin/file"
	"icooclaw/pkg/tools/builtin/web"
)

// RegisterBuiltinTools registers all built-in tools.
func RegisterBuiltinTools(registry *tools.Registry) {
	registry.Register(web.NewHTTPTool())
	registry.Register(web.NewWebSearchTool())
	registry.Register(NewDateTimeTool())

	// 文件系统工具
	// 使用环境变量或默认工作目录
	workDir := os.Getenv("ICOOCALW_WORKSPACE")
	if workDir == "" {
		workDir = "./workspace"
	}

	// 注册综合文件系统工具
	registry.Register(file.NewFilesystemTool(workDir))

	// 注册独立的文件操作工具
	registry.Register(file.NewReadFileTool(workDir))
	registry.Register(file.NewWriteFileTool(workDir))
	registry.Register(file.NewListDirTool(workDir))
	registry.Register(file.NewCopyFileTool(workDir))
}
