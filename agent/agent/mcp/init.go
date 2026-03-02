// Package mcp 提供了 MCP (Model Context Protocol) 客户端和工具集成
//
// 主要功能:
//   - MCP 客户端: 支持 stdio 和 HTTP/SSE 传输
//   - MCP 工具适配: 将 MCP 工具转换为 Agent 可用工具
//   - MCP 资源管理: 管理和访问 MCP 资源
//   - MCP 提示管理: 管理和使用 MCP 提示
//   - MCP 管理器: 统一管理所有 MCP 功能
//
// 使用示例:
//
//	cfg, err := config.Load()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	mgr := mcp.NewManager(cfg.MCP, logger)
//	if err := mcp.Init(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//
//	// 获取工具注册表并注册到 Agent
//	registry := mgr.GetToolsManager().GetRegistry()
//	for _, tool := range registry.List() {
//	    agent.RegisterTool(tool)
//	}
package mcp
