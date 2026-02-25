package script

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Run(t *testing.T) {
	engine := NewEngine(nil, nil)

	// 测试基本执行
	result, err := engine.Run("1 + 1")
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.ToInteger())

	// 测试函数定义
	_, err = engine.Run("function add(a, b) { return a + b; }")
	require.NoError(t, err)

	// 测试函数调用
	result, err = engine.Call("add", 3, 4)
	require.NoError(t, err)
	assert.Equal(t, int64(7), result.ToInteger())
}

func TestEngine_Console(t *testing.T) {
	engine := NewEngine(nil, nil)

	// 测试 console.log
	_, err := engine.Run(`console.log("Hello", "World")`)
	require.NoError(t, err)

	// 测试 console.info
	_, err = engine.Run(`console.info("Info message")`)
	require.NoError(t, err)

	// 测试 console.error
	_, err = engine.Run(`console.error("Error message")`)
	require.NoError(t, err)
}

func TestEngine_FS_ReadFile(t *testing.T) {
	engine := NewEngine(&Config{
		Workspace:     ".",
		AllowFileRead: true,
	}, nil)

	// 测试读取文件
	_, err := engine.Run(`
		var content = fs.readFile("engine.go");
		console.log("File length:", content.length);
	`)
	require.NoError(t, err)

	// 测试文件不存在
	_, err = engine.Run(`
		var content = fs.readFile("nonexistent_file.txt");
	`)
	assert.Error(t, err)
}

func TestEngine_FS_WriteFile(t *testing.T) {
	engine := NewEngine(&Config{
		Workspace:      t.TempDir(),
		AllowFileWrite: true,
	}, nil)

	// 测试写入文件
	_, err := engine.Run(`
		fs.writeFile("test.txt", "Hello, World!");
		console.log("File written");
	`)
	require.NoError(t, err)

	// 验证文件存在
	result, err := engine.Run(`fs.exists("test.txt")`)
	require.NoError(t, err)
	assert.True(t, result.ToBoolean())
}

func TestEngine_FS_Disabled(t *testing.T) {
	engine := NewEngine(&Config{
		Workspace:     ".",
		AllowFileRead: false,
	}, nil)

	// 测试禁用文件读取
	_, err := engine.Run(`fs.readFile("engine.go")`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")

	// 测试禁用文件写入
	_, err = engine.Run(`fs.writeFile("test.txt", "content")`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestEngine_FS_Dir(t *testing.T) {
	tmpDir := t.TempDir()
	engine := NewEngine(&Config{
		Workspace:       tmpDir,
		AllowFileWrite:  true,
		AllowFileRead:   true,
		AllowFileDelete: true,
	}, nil)

	// 创建目录
	_, err := engine.Run(`fs.mkdir("subdir")`)
	require.NoError(t, err)

	// 检查是否是目录
	result, err := engine.Run(`fs.isDir("subdir")`)
	require.NoError(t, err)
	assert.True(t, result.ToBoolean())

	// 列出目录
	_, err = engine.Run(`
		var files = fs.listDir(".");
		console.log("Files:", files.length);
	`)
	require.NoError(t, err)

	// 删除目录
	_, err = engine.Run(`fs.rmdir("subdir")`)
	require.NoError(t, err)
}

func TestEngine_HTTP_Get(t *testing.T) {
	engine := NewEngine(&Config{
		AllowNetwork: true,
		HTTPTimeout:  5,
	}, nil)

	// 测试 HTTP GET（使用示例 URL）
	_, err := engine.Run(`
		var response = http.get("https://httpbin.org/get");
		console.log("Status:", response.status_code);
		console.log("OK:", response.ok);
	`)
	// 可能因网络问题失败，所以不强制要求成功
	if err != nil {
		t.Logf("HTTP request failed (expected in test environment): %v", err)
	}
}

func TestEngine_HTTP_Disabled(t *testing.T) {
	engine := NewEngine(&Config{
		AllowNetwork: false,
	}, nil)

	// 测试禁用网络访问
	_, err := engine.Run(`http.get("https://example.com")`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestEngine_HTTP_Post(t *testing.T) {
	engine := NewEngine(&Config{
		AllowNetwork: true,
		HTTPTimeout:  5,
	}, nil)

	// 测试 HTTP POST
	_, err := engine.Run(`
		var response = http.postJSON("https://httpbin.org/post", { name: "test", value: 123 });
		console.log("Status:", response.status_code);
	`)
	if err != nil {
		t.Logf("HTTP request failed (expected in test environment): %v", err)
	}
}

func TestEngine_Shell_Exec(t *testing.T) {
	engine := NewEngine(&Config{
		AllowExec:   true,
		ExecTimeout: 5,
	}, nil)

	// 测试命令执行
	_, err := engine.Run(`
		var result = shell.exec("echo Hello");
		console.log("Stdout:", result.stdout);
		console.log("Exit code:", result.exit_code);
		console.log("Success:", result.success);
	`)
	require.NoError(t, err)

	// 测试命令执行结果
	result, err := engine.Run(`
		(function() {
			var r = shell.exec("echo test");
			return r.stdout.trim();
		})()
	`)
	require.NoError(t, err)
	assert.Contains(t, result.String(), "test")
}

func TestEngine_Shell_Disabled(t *testing.T) {
	engine := NewEngine(&Config{
		AllowExec: false,
	}, nil)

	// 测试禁用命令执行
	_, err := engine.Run(`shell.exec("echo test")`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestEngine_Utils(t *testing.T) {
	engine := NewEngine(nil, nil)

	// 测试时间函数
	_, err := engine.Run(`
		var now = utils.now();
		console.log("Now:", now);
		
		var ts = utils.timestamp();
		console.log("Timestamp:", ts);
		
		var formatted = utils.formatTime(ts, "2006-01-02");
		console.log("Formatted:", formatted);
	`)
	require.NoError(t, err)

	// 测试 UUID 生成
	result, err := engine.Run(`utils.uuid()`)
	require.NoError(t, err)
	assert.NotEmpty(t, result.String())

	// 测试环境变量
	_, err = engine.Run(`
		var path = utils.env("PATH");
		console.log("PATH:", path ? "set" : "not set");
		
		var custom = utils.envOr("NONEXISTENT_VAR", "default_value");
		console.log("Custom:", custom);
	`)
	require.NoError(t, err)
}

func TestEngine_JSON(t *testing.T) {
	engine := NewEngine(nil, nil)

	// 测试 JSON 操作
	_, err := engine.Run(`
		var obj = { name: "test", value: 123, nested: { key: "value" } };
		var str = JSON.stringify(obj);
		console.log("JSON:", str);
		
		var parsed = JSON.parse(str);
		console.log("Parsed name:", parsed.name);
		
		var pretty = JSON.pretty(obj);
		console.log("Pretty:", pretty.length);
	`)
	require.NoError(t, err)
}

func TestEngine_Base64(t *testing.T) {
	engine := NewEngine(nil, nil)

	// 测试 Base64 编码/解码
	result, err := engine.Run(`
		var encoded = Base64.encode("Hello, World!");
		console.log("Encoded:", encoded);
		var decoded = Base64.decode(encoded);
		decoded;
	`)
	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result.String())
}

func TestEngine_Context(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	engine := NewEngineWithContext(ctx, &Config{
		AllowExec: true,
	}, nil)

	// 测试上下文超时
	_, err := engine.Run(`
		utils.sleep(200);
	`)
	assert.Error(t, err) // 应该超时
}

func TestEngine_ComplexScript(t *testing.T) {
	engine := NewEngine(&Config{
		Workspace:      t.TempDir(),
		AllowFileRead:  true,
		AllowFileWrite: true,
	}, nil)

	// 测试复杂脚本
	script := `
		// 定义数据处理函数
		function processData(data) {
			return {
				original: data,
				length: data.length,
				upper: data.toUpperCase(),
				lower: data.toLowerCase(),
				timestamp: utils.timestamp()
			};
		}

		// 处理数据
		var result = processData("Hello World");
		
		// 写入文件
		fs.writeFile("output.json", JSON.pretty(result));
		
		// 读取文件
		var content = fs.readFile("output.json");
		
		// 返回结果
		content.length;
	`

	result, err := engine.Run(script)
	require.NoError(t, err)
	assert.True(t, result.ToInteger() > 0)
}

func TestEngine_GlobalVariables(t *testing.T) {
	engine := NewEngine(nil, nil)

	// 设置全局变量
	err := engine.SetGlobal("myGlobal", "Hello")
	require.NoError(t, err)

	// 使用全局变量
	result, err := engine.Run(`myGlobal + " World"`)
	require.NoError(t, err)
	assert.Equal(t, "Hello World", result.String())

	// 获取全局变量
	val := engine.GetGlobal("myGlobal")
	assert.Equal(t, "Hello", val.String())
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, ".", config.Workspace)
	assert.True(t, config.AllowFileRead)
	assert.False(t, config.AllowFileWrite)
	assert.False(t, config.AllowFileDelete)
	assert.False(t, config.AllowExec)
	assert.True(t, config.AllowNetwork)
	assert.Equal(t, 30, config.ExecTimeout)
	assert.Equal(t, 30, config.HTTPTimeout)
}
