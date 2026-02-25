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
  }
}
```

参数说明：
- `name` - 工具名称（必需）：只能包含字母、数字和下划线，不能以数字开头，2-50 字符
- `description` - 工具描述（必需）
- `code` - JavaScript 执行函数代码（必需）：必须定义 `execute(params)` 函数
- `parameters` - 参数定义（可选，JSON Schema 格式）
- `overwrite` - 是否覆盖已存在的工具（可选，默认 false）

**安全性限制**：
- 代码不能使用 `require()`、`import`、`eval()`、`Function()` 等
- 不能访问 `process`、`fs`、`http`、`net` 等模块
- 不能使用 `fetch()`、`XMLHttpRequest`、`WebSocket` 等网络 API

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

## 工具脚本格式

每个 JavaScript 工具文件需要定义以下内容：

```javascript
// 1. 定义工具元数据
var tool = {
    name: "tool_name",           // 工具名称（必需）
    description: "工具描述",      // 工具描述（必需）
    parameters: {                // 参数定义（OpenAI 格式）
        type: "object",
        properties: {
            param1: {
                type: "string",
                description: "参数1描述"
            },
            param2: {
                type: "number",
                description: "参数2描述"
            }
        },
        required: ["param1"]
    }
};

// 2. 定义执行函数
function execute(params) {
    // params 是传入的参数对象
    var result = process(params);
    
    // 返回字符串结果（推荐使用 JSON 字符串）
    return JSON.stringify({
        success: true,
        data: result
    });
}
```

## 示例工具

### 1. 计算器工具

```javascript
var tool = {
    name: "calculator",
    description: "执行基本数学计算",
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
    
    // 验证表达式安全性
    var allowedChars = /^[0-9+\-*/().%\s]+$/;
    if (!allowedChars.test(expression)) {
        return JSON.stringify({
            error: "表达式包含不允许的字符"
        });
    }
    
    try {
        var result = Function('"use strict"; return (' + expression + ')')();
        return JSON.stringify({
            expression: expression,
            result: result
        });
    } catch (e) {
        return JSON.stringify({
            error: "计算错误: " + e.message
        });
    }
}
```

### 2. 文本处理工具

```javascript
var tool = {
    name: "text_processor",
    description: "处理文本：支持大小写转换、字数统计等",
    parameters: {
        type: "object",
        properties: {
            action: {
                type: "string",
                enum: ["uppercase", "lowercase", "reverse", "count"],
                description: "操作类型"
            },
            text: {
                type: "string",
                description: "要处理的文本"
            }
        },
        required: ["action", "text"]
    }
};

function execute(params) {
    var action = params.action;
    var text = params.text;
    var result;
    
    switch (action) {
        case "uppercase":
            result = text.toUpperCase();
            break;
        case "lowercase":
            result = text.toLowerCase();
            break;
        case "reverse":
            result = text.split("").reverse().join("");
            break;
        case "count":
            return JSON.stringify({
                characters: text.length,
                words: text.trim().split(/\s+/).length,
                lines: text.split("\n").length
            });
        default:
            return JSON.stringify({ error: "未知操作: " + action });
    }
    
    return JSON.stringify({
        action: action,
        result: result
    });
}
```

### 3. 数据格式转换工具

```javascript
var tool = {
    name: "format_converter",
    description: "在 JSON 和 YAML 格式之间转换",
    parameters: {
        type: "object",
        properties: {
            from: {
                type: "string",
                enum: ["json", "yaml"],
                description: "源格式"
            },
            to: {
                type: "string",
                enum: ["json", "yaml"],
                description: "目标格式"
            },
            data: {
                type: "string",
                description: "要转换的数据"
            }
        },
        required: ["from", "to", "data"]
    }
};

function execute(params) {
    // 实现格式转换逻辑
    // 注意：goja 不支持所有 ES6+ 特性
    return JSON.stringify({
        converted: params.data
    });
}
```

## 可用 API

在 JavaScript 脚本中，可以使用以下内置对象：

### console

```javascript
console.log("信息日志");
console.info("信息日志");
console.debug("调试日志");
console.warn("警告日志");
console.error("错误日志");
console.table({ key: "value" }); // 表格形式输出
```

### JSON

```javascript
var obj = JSON.parse('{"key": "value"}');
var str = JSON.stringify(obj);
```

### 标准库

- `Math` - 数学运算
- `Date` - 日期处理
- `String` 方法
- `Array` 方法
- `RegExp` - 正则表达式

## 使用方式

1. 在工作区目录下创建 `tools` 目录
2. 将 `.js` 或 `.tool` 文件放入该目录
3. 重启 Agent 或调用工具加载接口

目录结构：
```
workspace/
├── tools/
│   ├── calculator.js
│   ├── text_processor.js
│   └── custom_tool.tool
└── data/
```

## 注意事项

1. **安全性**：JavaScript 在沙箱环境中执行，无法访问文件系统或网络
2. **性能**：脚本执行有超时限制（默认 30 秒）
3. **兼容性**：使用 ES5 语法，goja 不完全支持 ES6+ 特性
4. **返回值**：execute 函数应返回字符串，推荐返回 JSON 格式
5. **错误处理**：使用 try-catch 捕获可能的错误

## 配置选项

可在配置文件中设置：

```toml
[tools]
enabled = true
workspace = "./workspace"
exec_timeout = 30
```

## 运行时限制

- 最大内存：10MB（可配置）
- 执行超时：30秒（可配置）
- 调用栈深度：100层