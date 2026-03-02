package mcp

import (
	"log/slog"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cfg := ServerConfig{
		Name:      "test-server",
		Command:  "echo",
		Args:     []string{"hello"},
		Env:      map[string]string{"TEST": "value"},
		Transport: "stdio",
		Timeout:  30 * time.Second,
	}

	client := NewClient("test", cfg, nil)

	if client == nil {
		t.Error("NewClient returned nil")
	}

	if client.Name() != "test" {
		t.Errorf("Expected name 'test', got '%s'", client.Name())
	}

	if client.serverCfg.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.serverCfg.Timeout)
	}
}

func TestClient_IsConnected(t *testing.T) {
	cfg := ServerConfig{
		Name:      "test",
		Transport: "stdio",
	}

	client := NewClient("test", cfg, nil)

	// 初始状态应该是未连接
	if client.IsConnected() {
		t.Error("Expected client to be not connected initially")
	}
}

func TestClient_Name(t *testing.T) {
	cfg := ServerConfig{
		Name:      "my-server",
		Transport: "stdio",
	}

	client := NewClient("custom-name", cfg, nil)

	if client.name != "custom-name" {
		t.Errorf("Expected name 'custom-name', got '%s'", client.name)
	}

	if client.Name() != "custom-name" {
		t.Errorf("Expected Name() 'custom-name', got '%s'", client.Name())
	}
}

func TestClient_Tools(t *testing.T) {
	cfg := ServerConfig{
		Name:      "test",
		Transport: "stdio",
	}

	client := NewClient("test", cfg, nil)

	// 初始状态可以是nil或空切片
	tools := client.Tools()
	if len(tools) != 0 {
		t.Errorf("Expected 0 tools, got %d", len(tools))
	}
}

func TestClient_Resources(t *testing.T) {
	cfg := ServerConfig{
		Name:      "test",
		Transport: "stdio",
	}

	client := NewClient("test", cfg, nil)

	// 初始状态可以是nil或空切片
	resources := client.Resources()
	if len(resources) != 0 {
		t.Errorf("Expected 0 resources, got %d", len(resources))
	}
}

func TestClient_Prompts(t *testing.T) {
	cfg := ServerConfig{
		Name:      "test",
		Transport: "stdio",
	}

	client := NewClient("test", cfg, nil)

	// 初始状态可以是nil或空切片
	prompts := client.Prompts()
	if len(prompts) != 0 {
		t.Errorf("Expected 0 prompts, got %d", len(prompts))
	}
}

func TestClient_Disconnect(t *testing.T) {
	cfg := ServerConfig{
		Name:      "test",
		Transport: "stdio",
	}

	client := NewClient("test", cfg, nil)

	// 断开未连接的客户端不应该出错
	err := client.Disconnect()
	if err != nil {
		t.Errorf("Disconnect on unconnected client returned error: %v", err)
	}

	// 再次断开也不应该出错
	err = client.Disconnect()
	if err != nil {
		t.Errorf("Disconnect again returned error: %v", err)
	}
}

func TestNewClientManager(t *testing.T) {
	var logger *slog.Logger // nil logger
	mgr := NewClientManager(logger)

	if mgr == nil {
		t.Error("NewClientManager returned nil")
	}

	if mgr.clients == nil {
		t.Error("Expected clients map to be initialized")
	}

	if mgr.Count() != 0 {
		t.Errorf("Expected 0 clients, got %d", mgr.Count())
	}
}

func TestClientManager_AddClient(t *testing.T) {
	mgr := NewClientManager(nil)

	cfg := ServerConfig{
		Name:      "test-server",
		Transport: "stdio",
	}

	client := mgr.AddClient("test", cfg)

	if client == nil {
		t.Error("AddClient returned nil")
	}

	if mgr.Count() != 1 {
		t.Errorf("Expected 1 client, got %d", mgr.Count())
	}

	// 添加同名客户端会覆盖
	client2 := mgr.AddClient("test", cfg)
	if client2 == nil {
		t.Error("AddClient returned nil for duplicate")
	}

	if mgr.Count() != 1 {
		t.Errorf("Expected still 1 client after duplicate add, got %d", mgr.Count())
	}
}

func TestClientManager_GetClient(t *testing.T) {
	mgr := NewClientManager(nil)

	cfg := ServerConfig{
		Name:      "test",
		Transport: "stdio",
	}

	// 获取不存在的客户端
	client, ok := mgr.GetClient("not-exist")
	if ok {
		t.Error("Expected false for non-existent client")
	}
	if client != nil {
		t.Error("Expected nil client for non-existent")
	}

	// 添加并获取
	mgr.AddClient("test", cfg)
	client, ok = mgr.GetClient("test")
	if !ok {
		t.Error("Expected true for existing client")
	}
	if client == nil {
		t.Error("Expected non-nil client")
	}
}

func TestClientManager_RemoveClient(t *testing.T) {
	mgr := NewClientManager(nil)

	cfg := ServerConfig{
		Name:      "test",
		Transport: "stdio",
	}

	// 移除不存在的客户端
	err := mgr.RemoveClient("not-exist")
	if err == nil {
		t.Error("Expected error when removing non-existent client")
	}

	// 添加并移除
	mgr.AddClient("test", cfg)
	err = mgr.RemoveClient("test")
	if err != nil {
		t.Errorf("RemoveClient returned error: %v", err)
	}

	if mgr.Count() != 0 {
		t.Errorf("Expected 0 clients after remove, got %d", mgr.Count())
	}
}

func TestClientManager_ListClients(t *testing.T) {
	mgr := NewClientManager(nil)

	if len(mgr.ListClients()) != 0 {
		t.Error("Expected empty list initially")
	}

	cfg := ServerConfig{
		Name:      "test",
		Transport: "stdio",
	}

	mgr.AddClient("client1", cfg)
	mgr.AddClient("client2", cfg)

	clients := mgr.ListClients()
	if len(clients) != 2 {
		t.Errorf("Expected 2 clients, got %d", len(clients))
	}
}

func TestClientManager_DisconnectAll(t *testing.T) {
	mgr := NewClientManager(nil)

	// 断开空管理器
	err := mgr.DisconnectAll()
	if err != nil {
		t.Errorf("DisconnectAll on empty manager returned error: %v", err)
	}
}

func TestClientManager_Close(t *testing.T) {
	mgr := NewClientManager(nil)

	err := mgr.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestClientManager_GetAllTools(t *testing.T) {
	mgr := NewClientManager(nil)

	tools := mgr.GetAllTools()
	if tools == nil {
		t.Error("Expected non-nil tools map")
	}
	if len(tools) != 0 {
		t.Errorf("Expected empty tools map, got %d", len(tools))
	}
}

func TestServerConfig_Defaults(t *testing.T) {
	cfg := ServerConfig{
		Name:      "test",
		Transport: "stdio",
	}

	client := NewClient("test", cfg, nil)

	// 如果没有设置Timeout，应该使用默认值
	if client.serverCfg.Timeout == 0 {
		t.Error("Expected default timeout to be set")
	}
}

func TestNewClient_WithNilLogger(t *testing.T) {
	cfg := ServerConfig{
		Name:      "test",
		Transport: "stdio",
	}

	// 使用nil logger不应该panic
	client := NewClient("test", cfg, nil)
	if client == nil {
		t.Error("NewClient returned nil with nil logger")
	}

	if client.logger == nil {
		t.Error("Expected logger to be initialized with default")
	}
}
