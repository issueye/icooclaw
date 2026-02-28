动态创建新的 JavaScript 工具。使用此工具可以在运行时创建新的工具并立即使用。

**重要限制：**
- 不支持 `async` 和 `await` 关键字（JavaScript 引擎限制）
- HTTP 请求使用同步方式：`const response = http.get(url);`（不要使用 `await`）
- 所有工具函数必须是同步的：`function execute(params) { ... }`（不要使用 `async function`）

**示例：**
```javascript
// 正确 - 同步方式
function execute(params) {
    const response = http.get("https://api.example.com/data");
    const data = response.body;
    return JSON.stringify(data);
}

// 错误 - 不支持 async/await
async function execute(params) {
    const response = await http.get("https://api.example.com.com/data");
    return response.data;
}
```

## 工具参数

创建工具时支持以下参数：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 工具名称，只能包含字母、数字和下划线，如 `my_calculator` |
| description | string | 是 | 工具功能描述，清晰说明工具的用途 |
| code | string | 是 | JavaScript 执行函数代码，必须定义 `execute(params)` 函数 |
| parameters | object | 否 | 工具参数定义（JSON Schema 格式），默认 `{"type": "object", "properties": {}}` |
| permissions | object | 否 | 工具权限配置，控制脚本可以访问的功能 |
| overwrite | boolean | 否 | 是否覆盖已存在的同名工具，默认 `false` |

## 权限配置

`permissions` 对象支持以下属性：

| 属性 | 类型 | 说明 |
|------|------|------|
| fileRead | boolean | 允许读取文件，可使用 `fs.readFile`、`fs.exists` 等 |
| fileWrite | boolean | 允许写入文件，可使用 `fs.writeFile`、`fs.appendFile` 等 |
| fileDelete | boolean | 允许删除文件，可使用 `fs.deleteFile`、`fs.rmdir` 等 |
| network | boolean | 允许网络访问，可使用 `http.get`、`http.post` 等 |
| exec | boolean | 允许执行命令，可使用 `shell.exec` 等 |

## 参数定义（JSON Schema）

`parameters` 使用 JSON Schema 格式定义工具参数。例如：

```json
{
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "description": "要问候的名字"
    },
    "age": {
      "type": "number",
      "description": "年龄"
    }
  },
  "required": ["name"]
}
```

## 完整示例

### 示例 1：带参数的问候工具

```json
{
  "name": "greeting",
  "description": "生成个性化的问候消息",
  "parameters": {
    "type": "object",
    "properties": {
      "name": {
        "type": "string",
        "description": "要问候的名字"
      },
      "timeOfDay": {
        "type": "string",
        "description": "时段（morning/afternoon/evening）",
        "enum": ["morning", "afternoon", "evening"]
      }
    },
    "required": ["name"]
  },
  "code": "function execute(params) {\n    var name = params.name || 'World';\n    var timeMsg = '你好';\n    if (params.timeOfDay === 'morning') {\n        timeMsg = '早上好';\n    } else if (params.timeOfDay === 'afternoon') {\n        timeMsg = '下午好';\n    } else if (params.timeOfDay === 'evening') {\n        timeMsg = '晚上好';\n    }\n    return timeMsg + '，' + name + '！';\n}"
}
```

### 示例 2：文件处理工具

```json
{
  "name": "file_uppercase",
  "description": "将文件内容转换为大写",
  "parameters": {
    "type": "object",
    "properties": {
      "inputPath": {
        "type": "string",
        "description": "输入文件路径"
      },
      "outputPath": {
        "type": "string",
        "description": "输出文件路径"
      }
    },
    "required": ["inputPath", "outputPath"]
  },
  "code": "function execute(params) {\n    var content = fs.readFile(params.inputPath);\n    var upperContent = content.toUpperCase();\n    fs.writeFile(params.outputPath, upperContent);\n    return '文件已转换为大写并保存到: ' + params.outputPath;\n}",
  "permissions": {
    "fileRead": true,
    "fileWrite": true
  }
}
```

### 示例 3：API 调用工具

```json
{
  "name": "fetch_weather",
  "description": "获取指定城市的天气信息",
  "parameters": {
    "type": "object",
    "properties": {
      "city": {
        "type": "string",
        "description": "城市名称"
      }
    },
    "required": ["city"]
  },
  "code": "function execute(params) {\n    var url = 'https://api.example.com/weather?city=' + encodeURIComponent(params.city);\n    var response = http.get(url);\n    var data = JSON.parse(response.body);\n    return '城市: ' + data.city + '\\n天气: ' + data.weather + '\\n温度: ' + data.temp + '°C';\n}",
  "permissions": {
    "network": true
  }
}
```

### 示例 4：计算器工具

```json
{
  "name": "calculator",
  "description": "执行数学计算",
  "parameters": {
    "type": "object",
    "properties": {
      "operation": {
        "type": "string",
        "description": "运算类型（add/subtract/multiply/divide）",
        "enum": ["add", "subtract", "multiply", "divide"]
      },
      "a": {
        "type": "number",
        "description": "第一个数字"
      },
      "b": {
        "type": "number",
        "description": "第二个数字"
      }
    },
    "required": ["operation", "a", "b"]
  },
  "code": "function execute(params) {\n    var result;\n    switch(params.operation) {\n        case 'add':\n            result = params.a + params.b;\n            break;\n        case 'subtract':\n            result = params.a - params.b;\n            break;\n        case 'multiply':\n            result = params.a * params.b;\n            break;\n        case 'divide':\n            if (params.b === 0) {\n                return '错误: 除数不能为零';\n            }\n            result = params.a / params.b;\n            break;\n    }\n    return params.a + ' ' + params.operation + ' ' + params.b + ' = ' + result;\n}"
}
```

## 内置对象

脚本中可以使用以下内置对象：

### console - 控制台输出
- console.log(...args) - 输出信息
- console.info(...args) - 输出信息
- console.debug(...args) - 输出调试信息
- console.warn(...args) - 输出警告
- console.error(...args) - 输出错误
- console.table(data) - 表格形式输出

### JSON - JSON 操作
- JSON.stringify(obj) - 对象转字符串
- JSON.parse(str) - 字符串转对象
- JSON.pretty(obj) - 格式化输出

### Base64 - 编码解码
- Base64.encode(str) - Base64 编码
- Base64.decode(str) - Base64 解码

### crypto - 加密工具（已内置）
- crypto.md5(data) - MD5 哈希
- crypto.sha1(data) - SHA1 哈希
- crypto.sha256(data) - SHA256 哈希
- crypto.hmacMD5(data, key) - HMAC-MD5
- crypto.hmacSHA1(data, key) - HMAC-SHA1
- crypto.hmacSHA256(data, key) - HMAC-SHA256
- crypto.aesEncrypt(plaintext, key) - AES 加密
- crypto.aesDecrypt(ciphertext, key) - AES 解密
- crypto.base64Encode(data) / crypto.base64Decode(encoded) - Base64 编解码
- crypto.hexEncode(data) / crypto.hexDecode(encoded) - Hex 编解码

### fs - 文件系统（需要设置 permissions.fileRead/fileWrite/fileDelete）
- fs.readFile(path) - 读取文件内容
- fs.writeFile(path, content) - 写入文件
- fs.appendFile(path, content) - 追加内容
- fs.exists(path) - 检查文件是否存在
- fs.deleteFile(path) - 删除文件
- fs.mkdir(path) - 创建目录
- fs.rmdir(path) - 删除目录
- fs.listDir(path) - 列出目录内容
- fs.copyFile(src, dst) - 复制文件
- fs.moveFile(src, dst) - 移动文件

### http - HTTP 客户端（需要设置 permissions.network）
- http.get(url) - GET 请求
- http.post(url, body) - POST 请求
- http.postJSON(url, body) - POST JSON 请求
- http.request(method, url, body, headers) - 自定义请求
- http.download(url, savePath) - 下载文件

### shell - 命令执行（需要设置 permissions.exec）
- shell.exec(command) - 执行命令
- shell.execWithTimeout(command, timeout) - 带超时执行
- shell.execInDir(command, workDir) - 在指定目录执行

### utils - 工具函数
- utils.now() - 当前时间字符串
- utils.timestamp() - 当前时间戳
- utils.formatTime(ts, layout) - 格式化时间
- utils.sleep(ms) - 休眠毫秒
- utils.env(key) - 获取环境变量
- utils.cwd() - 当前工作目录

## 其他工具

配套提供以下管理工具：

- **list_tools** - 列出所有可用的工具
- **delete_tool** - 删除动态创建的 JavaScript 工具
