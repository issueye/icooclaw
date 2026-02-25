# JavaScript 工具系统

Agent 支持通过 JavaScript 脚本动态添加工具。有两种方式：
1. 在工作区的 `tools` 目录下放置 `.js` 或 `.tool` 文件（启动时自动加载）
2. 使用 `create_tool` 工具在运行时动态创建

## 动态工具管理

### 创建工具 (create_tool)

使用 `create_tool` 工具可以在运行时动态创建新的 JavaScript 工具：

```json
{
  "name": "my_tool",
  "description": "我的自定义工具",
  "code": "function execute(params) {\n    return JSON.stringify({result: params.input});\n}",
  "parameters": {
    "type": "object",
    "properties": {
      "input": {
        "type": "string",
        "description": "输入文本"
      }
    },
    "required": ["input"]
  },
  "permissions": {
    "fileRead": true,
    "network": true
  }
}
```

参数说明：
- `name` - 工具名称（必需）：只能包含字母、数字和下划线，不能以数字开头，2-50 字符
- `description` - 工具描述（必需）
- `code` - JavaScript 执行函数代码（必需）：必须定义 `execute(params)` 函数
- `parameters` - 参数定义（可选，JSON Schema 格式）
- `permissions` - 权限配置（可选，控制脚本可以访问的功能）
- `overwrite` - 是否覆盖已存在的工具（可选，默认 false）

### 权限配置

| 权限 | 说明 | 可用功能 |
|------|------|----------|
| `fileRead` | 读取文件 | `fs.readFile`, `fs.exists`, `fs.listDir`, `fs.isDir`, `fs.isFile`, `fs.getInfo` |
| `fileWrite` | 写入文件 | `fs.writeFile`, `fs.appendFile`, `fs.mkdir`, `fs.copyFile`, `fs.moveFile` |
| `fileDelete` | 删除文件 | `fs.deleteFile`, `fs.rmdir` |
| `network` | 网络访问 | `http.get`, `http.post`, `http.postJSON`, `http.request`, `http.download` |
| `exec` | 执行命令 | `shell.exec`, `shell.execWithTimeout`, `shell.execInDir`, `shell.pipe` |

### 列出工具 (list_tools)

使用 `list_tools` 工具列出所有可用工具：

```json
{}
```

返回结果包含：
- `builtin_tools` - 内置工具列表
- `dynamic_tools` - 动态创建的工具列表
- `create_example` - 创建工具示例

### 删除工具 (delete_tool)

使用 `delete_tool` 工具删除动态创建的工具：

```json
{
  "name": "my_tool"
}
```

注意：只能删除通过 `create_tool` 创建的工具，不能删除内置工具。

## 内置对象

### console - 控制台输出

```javascript
console.log("消息");      // 信息日志
console.info("信息");    // 信息日志
console.debug("调试");   // 调试日志
console.warn("警告");    // 警告日志
console.error("错误");   // 错误日志
console.table({key: "value"});  // 表格输出
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

### fs - 文件系统（需要权限）

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

// 检查是否是目录/文件
if (fs.isDir("path")) { }
if (fs.isFile("path")) { }

// 列出目录内容
var files = fs.listDir(".");
// files: [{name, is_dir, size, modified}, ...]

// 创建/删除目录
fs.mkdir("newdir");
fs.rmdir("dir");

// 删除文件
fs.deleteFile("file.txt");

// 复制/移动文件
fs.copyFile("src.txt", "dst.txt");
fs.moveFile("old.txt", "new.txt");

// 获取文件信息
var info = fs.getInfo("file.txt");
// info: {name, size, is_dir, modified, mode}
```

### http - HTTP 客户端（需要权限）

```javascript
// GET 请求
var response = http.get("https://api.example.com/data");
console.log("状态码:", response.status_code);
console.log("响应体:", response.body);
console.log("是否成功:", response.ok);

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

### shell - 命令执行（需要权限）

```javascript
// 执行命令
var result = shell.exec("ls -la");
console.log("输出:", result.stdout);
console.log("错误:", result.stderr);
console.log("退出码:", result.exit_code);
console.log("是否成功:", result.success);
console.log("执行时间:", result.duration);
console.log("是否超时:", result.timed_out);

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

## 配置

在 `config.toml` 中配置 JavaScript 工具：

```toml
# JavaScript 工具配置
[tools.js]
enabled = true                    # 是否启用 JS 工具
tools_dir = "tools"              # JS 工具目录（相对于 workspace）
max_memory = 10485760           # 最大内存（字节），默认 10MB
timeout = 30                     # 执行超时（秒）

# JS 工具默认权限
[tools.js.permissions]
file_read = false               # 允许读取文件
file_write = false              # 允许写入文件
file_delete = false             # 允许删除文件
network = false                 # 允许网络访问
exec = false                    # 允许执行命令
http_timeout = 30              # HTTP 请求超时（秒）
exec_timeout = 30               # 命令执行超时（秒）
# allowed_domains = ["api.example.com"]  # 允许的域名白名单
```

## 示例

### 1. 简单计算器

```javascript
var tool = {
    name: "calculator",
    description: "执行数学计算",
    parameters: {
        type: "object",
        properties: {
            expression: {
                type: "string",
                description: "数学表达式，如: 2 + 3 * 4"
            }
        },
        required: ["expression"]
    }
};

function execute(params) {
    var expression = params.expression;
    var result = Function('"use strict"; return (' + expression + ')')();
    return JSON.stringify({
        expression: expression,
        result: result
    });
}
```

### 2. API 数据获取（需要网络权限）

```javascript
var tool = {
    name: "api_fetcher",
    description: "从API获取数据并保存到文件",
    parameters: {
        type: "object",
        properties: {
            url: { type: "string", description: "API URL" },
            file: { type: "string", description: "保存文件名" }
        },
        required: ["url", "file"]
    },
    permissions: {
        network: true,
        fileWrite: true
    }
};

function execute(params) {
    var response = http.get(params.url);
    if (response.ok) {
        fs.writeFile(params.file, response.body);
        return JSON.stringify({ success: true, file: params.file });
    }
    return JSON.stringify({ success: false, error: response.status });
}
```

### 3. 文件处理（需要文件权限）

```javascript
var tool = {
    name: "file_processor",
    description: "处理文件内容",
    parameters: {
        type: "object",
        properties: {
            input: { type: "string", description: "输入文件" },
            output: { type: "string", description: "输出文件" },
            operation: { type: "string", description: "操作类型" }
        },
        required: ["input", "output", "operation"]
    },
    permissions: {
        fileRead: true,
        fileWrite: true
    }
};

function execute(params) {
    var content = fs.readFile(params.input);
    var result;
    
    switch (params.operation) {
        case "uppercase":
            result = content.toUpperCase();
            break;
        case "lowercase":
            result = content.toLowerCase();
            break;
        default:
            return JSON.stringify({ error: "Unknown operation" });
    }
    
    fs.writeFile(params.output, result);
    return JSON.stringify({ success: true, output: params.output });
}
```

### 4. 命令执行（需要 exec 权限）

```javascript
var tool = {
    name: "run_command",
    description: "执行 shell 命令",
    parameters: {
        type: "object",
        properties: {
            command: { type: "string", description: "要执行的命令" },
            timeout: { type: "integer", description: "超时时间（秒）" }
        },
        required: ["command"]
    },
    permissions: {
        exec: true
    }
};

function execute(params) {
    var timeout = params.timeout || 30;
    var result = shell.execWithTimeout(params.command, timeout);
    
    return JSON.stringify({
        success: result.success,
        stdout: result.stdout,
        stderr: result.stderr,
        exitCode: result.exit_code,
        duration: result.duration
    });
}
```

## 安全最佳实践

1. **最小权限原则**：只请求必要的权限
2. **输入验证**：验证所有输入参数
3. **错误处理**：正确处理和返回错误信息
4. **资源限制**：设置合理的超时和内存限制
5. **日志记录**：使用 console 记录重要操作
6. **敏感信息**：不要在代码中硬编码敏感信息