// Package script provides JavaScript scripting engine for icooclaw.
package script

import (
	"context"
	"testing"
)

func TestEngine_Run(t *testing.T) {
	cfg := DefaultConfig()
	engine := NewEngine(cfg, nil)

	tests := []struct {
		name    string
		script  string
		wantErr bool
	}{
		{
			name:    "simple expression",
			script:  "1 + 1",
			wantErr: false,
		},
		{
			name:    "console.log",
			script:  "console.log('hello')",
			wantErr: false,
		},
		{
			name:    "JSON operations",
			script:  "JSON.stringify({a: 1})",
			wantErr: false,
		},
		{
			name:    "Base64 encoding",
			script:  "Base64.encode('hello')",
			wantErr: false,
		},
		{
			name:    "crypto SHA256",
			script:  "crypto.sha256('hello')",
			wantErr: false,
		},
		{
			name:    "utils timestamp",
			script:  "utils.timestamp()",
			wantErr: false,
		},
		{
			name:    "syntax error",
			script:  "invalid javascript",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.Run(tt.script)
			if (err != nil) != tt.wantErr {
				t.Errorf("Engine.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngine_Call(t *testing.T) {
	cfg := DefaultConfig()
	engine := NewEngine(cfg, nil)

	_, err := engine.Run("function add(a, b) { return a + b; }")
	if err != nil {
		t.Fatalf("Failed to define function: %v", err)
	}

	result, err := engine.Call("add", 1, 2)
	if err != nil {
		t.Fatalf("Failed to call function: %v", err)
	}

	if result == nil || result.ToInteger() != 3 {
		t.Errorf("Expected 3, got %v", result)
	}
}

func TestEngine_SetGlobal(t *testing.T) {
	cfg := DefaultConfig()
	engine := NewEngine(cfg, nil)

	err := engine.SetGlobal("myVar", "hello")
	if err != nil {
		t.Fatalf("Failed to set global: %v", err)
	}

	result, err := engine.Run("myVar")
	if err != nil {
		t.Fatalf("Failed to access global: %v", err)
	}

	if result == nil || result.String() != "hello" {
		t.Errorf("Expected 'hello', got %v", result)
	}
}

func TestConsole(t *testing.T) {
	console := NewConsole(nil)

	if console.Name() != "console" {
		t.Errorf("Expected name 'console', got %s", console.Name())
	}

	obj := console.Object()
	if obj == nil {
		t.Error("Object should not be nil")
	}

	console.Log("test")
	console.Info("test")
	console.Debug("test")
	console.Warn("test")
	console.Error("test")
}

func TestUtils(t *testing.T) {
	utils := NewUtils()

	if utils.Name() != "utils" {
		t.Errorf("Expected name 'utils', got %s", utils.Name())
	}

	ts := utils.Timestamp()
	if ts <= 0 {
		t.Errorf("Expected positive timestamp, got %d", ts)
	}

	uuid := utils.UUID()
	if len(uuid) != 36 {
		t.Errorf("Expected UUID length 36, got %d", len(uuid))
	}
}

func TestCrypto(t *testing.T) {
	c := &crypto{}

	hash := c.SHA256("hello")
	if hash == "" {
		t.Error("SHA256 should not be empty")
	}

	encoded := c.Base64Encode("hello")
	if encoded == "" {
		t.Error("Base64Encode should not be empty")
	}

	decoded, err := c.Base64Decode(encoded)
	if err != nil {
		t.Errorf("Base64Decode failed: %v", err)
	}
	if decoded != "hello" {
		t.Errorf("Expected 'hello', got '%s'", decoded)
	}
}

func TestShellExec_Disabled(t *testing.T) {
	cfg := &Config{AllowExec: false}
	shell := NewShellExec(context.Background(), cfg, nil)

	_, err := shell.Exec("echo hello")
	if err == nil {
		t.Error("Expected error when shell execution is disabled")
	}
}

func TestHTTPClient_Disabled(t *testing.T) {
	cfg := &Config{AllowNetwork: false}
	httpClient := NewHTTPClient(cfg, nil)

	_, err := httpClient.Get("http://example.com", nil)
	if err == nil {
		t.Error("Expected error when network access is disabled")
	}
}

func TestFileSystem_Disabled(t *testing.T) {
	cfg := &Config{AllowFileRead: false, AllowFileWrite: false}
	fs := NewFileSystem(cfg, nil)

	_, err := fs.ReadFile("test.txt")
	if err == nil {
		t.Error("Expected error when file reading is disabled")
	}

	err = fs.WriteFile("test.txt", "content")
	if err == nil {
		t.Error("Expected error when file writing is disabled")
	}
}

func TestScriptTool(t *testing.T) {
	cfg := DefaultConfig()
	tool := NewScriptTool(cfg, nil)

	if tool.Name() != "script" {
		t.Errorf("Expected name 'script', got %s", tool.Name())
	}

	result := tool.Execute(context.Background(), map[string]any{"code": "1 + 1"})
	if !result.Success {
		t.Errorf("Expected success, got error: %v", result.Error)
	}
}