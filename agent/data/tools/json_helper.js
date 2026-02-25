/**
 * JSON 数据处理工具
 * 支持 JSON 格式化、提取、转换等操作
 */

var tool = {
    name: "json_helper",
    description: "JSON 数据处理工具：格式化、提取路径值、合并对象等",
    parameters: {
        type: "object",
        properties: {
            action: {
                type: "string",
                enum: ["format", "minify", "get", "merge", "validate"],
                description: "操作类型: format(格式化)、minify(压缩)、get(提取路径值)、merge(合并对象)、validate(验证)"
            },
            data: {
                type: "string",
                description: "JSON 数据字符串"
            },
            path: {
                type: "string",
                description: "要提取的路径，如 'user.name' 或 'items[0].id' (仅用于 get 操作)"
            },
            data2: {
                type: "string",
                description: "第二个 JSON 数据 (仅用于 merge 操作)"
            },
            indent: {
                type: "integer",
                description: "缩进空格数 (仅用于 format 操作，默认 2)"
            }
        },
        required: ["action", "data"]
    }
};

function execute(params) {
    var action = params.action;
    var dataStr = params.data;
    
    try {
        var data = JSON.parse(dataStr);
        
        switch (action) {
            case "format":
                var indent = params.indent || 2;
                return JSON.stringify({
                    success: true,
                    result: JSON.stringify(data, null, indent)
                });
                
            case "minify":
                return JSON.stringify({
                    success: true,
                    result: JSON.stringify(data)
                });
                
            case "get":
                var path = params.path;
                if (!path) {
                    return JSON.stringify({
                        success: false,
                        error: "path parameter is required for 'get' action"
                    });
                }
                var value = getByPath(data, path);
                return JSON.stringify({
                    success: true,
                    path: path,
                    value: value
                });
                
            case "merge":
                var data2Str = params.data2;
                if (!data2Str) {
                    return JSON.stringify({
                        success: false,
                        error: "data2 parameter is required for 'merge' action"
                    });
                }
                var data2 = JSON.parse(data2Str);
                var merged = deepMerge(data, data2);
                return JSON.stringify({
                    success: true,
                    result: merged
                });
                
            case "validate":
                return JSON.stringify({
                    success: true,
                    valid: true,
                    type: getDataType(data)
                });
                
            default:
                return JSON.stringify({
                    success: false,
                    error: "Unknown action: " + action
                });
        }
    } catch (e) {
        return JSON.stringify({
            success: false,
            error: "JSON parse error: " + e.message
        });
    }
}

// 根据路径获取值
function getByPath(obj, path) {
    var parts = path.split(/[\.\[\]]+/).filter(function(p) { return p !== ""; });
    var current = obj;
    
    for (var i = 0; i < parts.length; i++) {
        var part = parts[i];
        if (current === null || current === undefined) {
            return undefined;
        }
        if (Array.isArray(current)) {
            var index = parseInt(part);
            if (isNaN(index)) {
                return undefined;
            }
            current = current[index];
        } else if (typeof current === "object") {
            current = current[part];
        } else {
            return undefined;
        }
    }
    
    return current;
}

// 深度合并对象
function deepMerge(target, source) {
    var result = {};
    
    // 复制 target
    for (var key in target) {
        if (target.hasOwnProperty(key)) {
            if (typeof target[key] === "object" && target[key] !== null && !Array.isArray(target[key])) {
                result[key] = deepMerge(target[key], {});
            } else {
                result[key] = target[key];
            }
        }
    }
    
    // 合并 source
    for (var key in source) {
        if (source.hasOwnProperty(key)) {
            if (typeof source[key] === "object" && source[key] !== null && !Array.isArray(source[key])) {
                if (typeof result[key] === "object" && result[key] !== null) {
                    result[key] = deepMerge(result[key], source[key]);
                } else {
                    result[key] = deepMerge({}, source[key]);
                }
            } else {
                result[key] = source[key];
            }
        }
    }
    
    return result;
}

// 获取数据类型
function getDataType(data) {
    if (data === null) return "null";
    if (Array.isArray(data)) return "array";
    return typeof data;
}