/**
 * 计算器工具示例
 * 支持基本数学运算
 */

var tool = {
    name: "calculator_example",
    description: "执行基本数学计算，如加减乘除",
    parameters: {
        type: "object",
        properties: {
            expression: {
                type: "string",
                description: "要计算的数学表达式，例如: '2 + 3 * 4'"
            }
        },
        required: ["expression"]
    }
};

function execute(params) {
    var expression = params.expression;
    
    // 安全的数学运算（仅支持基本运算）
    var allowedChars = /^[0-9+\-*/().%\s]+$/;
    if (!allowedChars.test(expression)) {
        return JSON.stringify({
            error: "表达式包含不允许的字符",
            expression: expression
        });
    }
    
    try {
        // 使用 Function 构造函数进行安全计算
        var result = Function('"use strict"; return (' + expression + ')')();
        return JSON.stringify({
            expression: expression,
            result: result
        });
    } catch (e) {
        return JSON.stringify({
            error: "计算错误: " + e.message,
            expression: expression
        });
    }
}