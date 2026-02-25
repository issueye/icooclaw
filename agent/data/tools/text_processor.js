/**
 * 文本处理工具示例
 * 支持多种文本操作
 */

var tool = {
    name: "text_processor",
    description: "处理文本：支持大小写转换、字数统计、反转等操作",
    parameters: {
        type: "object",
        properties: {
            action: {
                type: "string",
                enum: ["uppercase", "lowercase", "reverse", "count", "trim"],
                description: "要执行的操作：uppercase(大写)、lowercase(小写)、reverse(反转)、count(统计)、trim(去除空白)"
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
            result = JSON.stringify({
                characters: text.length,
                words: text.trim().split(/\s+/).filter(function(w) { return w.length > 0; }).length,
                lines: text.split("\n").length
            });
            break;
        case "trim":
            result = text.trim();
            break;
        default:
            return JSON.stringify({ error: "未知操作: " + action });
    }
    
    if (typeof result === "object") {
        return result;
    }
    return JSON.stringify({
        action: action,
        original_length: text.length,
        result: result
    });
}