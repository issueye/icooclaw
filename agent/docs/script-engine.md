# 脚本引擎

Agent 提供了一个基于 [goja](https://github.com/dop251/goja) 的 JavaScript 脚本引擎，支持文件操作、网络访问、命令执行等功能。

## 基本用法

```go
package main

import (
    "log/slog"
    
    "github.com/icooclaw/icooclaw/internal/script"
)

func main() {
    // 创建引擎配置
    config := &script.Config{
        Workspace:       "./workspace",  // 工作目录
        AllowFileRead:   true,           // 允许读取文件
        AllowFileWrite:  true,           // 允许写入文件
        AllowFileDelete: false,          // 禁止删除文件
        AllowExec:       false,          // 禁止执行命令
        AllowNetwork:    true,           // 允许网络访问
        HTTPTimeout:     30,             // HTTP 超时（秒）
        ExecTimeout:     30,             // 命令执行超时（秒）
    }
    
    // 创建脚本引擎
    engine := script.NewEngine(config, slog.Default())
    
    // 执行脚本
    result, err := engine.Run(`
        var data = fs.readFile("config.json");
        var config = JSON.parse(data);
        console.log("Config:", config.name);
    `)
}
```

## 内置对象

### console - 控制台

```javascript
console.log("消息");      // 信息日志
console.info("信息");    // 信息日志
console.debug("调试");   // 调试日志
console.warn("警告");    // 警告日志
console.error("错误");   // 错误日志
console.table({key: "value"});  // 表格输出
```

### fs - 文件系统

```javascript
// 读取文件
var content = fs.readFile("file.txt");

// 写入文件
fs.writeFile("output.txt", "Hello World");

// 追加内容
fs.appendFile("log.txt", "新的一行\n");

// 检查文件是否存在
if (fs.exists("file.txt")) {
    console.log("文件存在");
}

// 检查是否是目录
if (fs.isDir("path")) { }

// 检查是否是文件
if (fs.isFile("path")) { }

// 列出目录内容
var files = fs.listDir(".");
// files: [{name, is_dir, size, modified}, ...]

// 创建目录
fs.mkdir("newdir");

// 递归创建目录
fs.mkdirAll("a/b/c");

// 删除文件
fs.deleteFile("file.txt");

// 删除目录
fs.rmdir("dir");

// 复制文件
fs.copyFile("src.txt", "dst.txt");

// 移动文件
fs.moveFile("old.txt", "new.txt");

// 获取文件信息
var info = fs.getInfo("file.txt");
// info: {name, size, is_dir, modified, mode}
```

### http - HTTP 客户端

```javascript
// GET 请求
var response = http.get("https://api.example.com/data");
console.log("状态码:", response.status_code);
console.log("响应体:", response.body);

// POST 请求
var response = http.post("https://api.example.com/create", {name: "test"});

// POST JSON 请求
var response = http.postJSON("https://api.example.com/api", {
    name: "test",
    value: 123
});

// 自定义请求
var response = http.request("POST", "https://api.example.com/api", 
    {key: "value"},  // body
    {"Authorization": "Bearer token"}  // headers
);

// 下载文件
http.download("https://example.com/file.zip", "downloads/file.zip");
```

### shell - 命令执行

```javascript
// 执行命令
var result = shell.exec("ls -la");
console.log("输出:", result.stdout);
console.log("错误:", result.stderr);
console.log("退出码:", result.exit_code);
console.log("是否成功:", result.success);
console.log("执行时间:", result.duration);

// 带超时执行
var result = shell.execWithTimeout("long-running-command", 60);  // 60秒超时

// 在指定目录执行
var result = shell.execInDir("npm install", "/path/to/project");

// 管道命令
var result = shell.pipe(["cat file.txt", "grep keyword", "wc -l"]);
```

### utils - 工具函数

```javascript
// 获取当前时间
var now = utils.now();  // "2024-01-01T12:00:00+08:00"

// 获取时间戳
var ts = utils.timestamp();  // 1704096000

// 格式化时间
var formatted = utils.formatTime(ts, "2006-01-02");  // "2024-01-01"

// 解析时间
var timestamp = utils.parseTime("2024-01-01T00:00:00Z");

// 休眠（毫秒）
utils.sleep(1000);  // 休眠1秒

// 获取环境变量
var path = utils.env("PATH");
var custom = utils.envOr("MY_VAR", "default_value");

// 获取当前工作目录
var cwd = utils.cwd();

// 获取主机名
var hostname = utils.hostname();

// 生成 UUID
var uuid = utils.uuid();
```

### JSON - JSON 操作

```javascript
// 字符串化
var json = JSON.stringify({name: "test", value: 123});

// 格式化输出
var pretty = JSON.pretty({name: "test", value: 123});
// 输出:
// {
//   "name": "test",
//   "value": 123
// }

// 解析
var obj = JSON.parse('{"name": "test"}');
```

### Base64 - 编码解码

```javascript
// 编码
var encoded = Base64.encode("Hello World");
// "SGVsbG8gV29ybGQ="

// 解码
var decoded = Base64.decode("SGVsbG8gV29ybGQ=");
// "Hello World"
```

## 安全配置

```go
config := &script.Config{
    // 工作目录（文件操作的基础路径）
    Workspace: "./workspace",
    
    // 文件权限
    AllowFileRead:   true,   // 允许读取文件
    AllowFileWrite:  false,  // 禁止写入文件
    AllowFileDelete: false,  // 禁止删除文件
    
    // 命令执行权限
    AllowExec: false,  // 禁止执行 shell 命令
    
    // 网络权限
    AllowNetwork:   true,  // 允许网络访问
    AllowedDomains: []string{"api.example.com"},  // 域名白名单（空=允许所有）
    
    // 超时设置
    ExecTimeout: 30,   // 命令执行超时（秒）
    HTTPTimeout: 30,   // HTTP 请求超时（秒）
    MaxMemory:   10 * 1024 * 1024,  // 最大内存（字节）
}
```

## 动态创建脚本工具

在 JS 工具脚本中，可以指定需要增强功能：

```javascript
// data/tools/enhanced_tool.js
var tool = {
    name: "enhanced_tool",
    description: "使用增强功能的工具",
    parameters: {
        type: "object",
        properties: {
            url: {
                type: "string",
                description: "要获取的URL"
            },
            file: {
                type: "string",
                description: "保存的文件名"
            }
        },
        required: ["url", "file"]
    },
    // 需要的功能权限
    permissions: {
        network: true,    // 需要网络访问
        fileWrite: true   // 需要文件写入
    }
};

function execute(params) {
    // 下载文件
    var response = http.get(params.url);
    
    // 保存到文件
    fs.writeFile(params.file, response.body);
    
    return JSON.stringify({
        success: true,
        file: params.file,
        size: response.body.length
    });
}
```

## 使用上下文

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

engine := script.NewEngineWithContext(ctx, config, logger)

// 脚本执行会在上下文取消时中断
result, err := engine.Run("utils.sleep(60000);")
// 如果上下文超时，将返回错误
```

## 最佳实践

1. **权限最小化**：只开启必要的权限
2. **使用白名单**：限制网络访问的域名
3. **设置超时**：防止脚本长时间运行
4. **错误处理**：检查返回的错误和结果
5. **日志记录**：使用 console 记录重要信息