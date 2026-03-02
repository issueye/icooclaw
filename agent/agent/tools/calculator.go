package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// CalculatorTool 计算器工具
type CalculatorTool struct {
	baseTool *BaseTool
}

// NewCalculatorTool 创建计算器工具
func NewCalculatorTool() *CalculatorTool {
	tool := NewBaseTool(
		"calculator",
		"执行数学计算。支持基本运算符 (+, -, *, /, %), 幂运算 (**), 以及数学函数 (sin, cos, tan, sqrt, log, abs, round, floor, ceil)。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"expression": map[string]interface{}{
					"type":        "string",
					"description": "数学表达式，例如: 2 + 2, sqrt(16), sin(3.14159/2)",
				},
			},
			"required": []string{"expression"},
		},
		nil,
	)

	return &CalculatorTool{
		baseTool: tool,
	}
}

// Name 获取名称
func (t *CalculatorTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *CalculatorTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *CalculatorTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 执行计算
func (t *CalculatorTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	expression, ok := params["expression"].(string)
	if !ok || expression == "" {
		return "", fmt.Errorf("invalid or missing expression")
	}

	// 清理表达式
	expression = strings.TrimSpace(expression)

	// 解析并计算表达式
	result, err := t.evaluate(expression)
	if err != nil {
		return "", fmt.Errorf("calculation error: %w", err)
	}

	// 构建结果
	output := map[string]interface{}{
		"expression": expression,
		"result":     result,
		"type":       "number",
	}

	// 如果结果是整数，显示为整数
	if result == math.Floor(result) && math.Abs(result) < 1e15 {
		output["result"] = int64(result)
		output["formatted"] = fmt.Sprintf("%d", int64(result))
	} else {
		output["formatted"] = strconv.FormatFloat(result, 'f', -1, 64)
	}

	resultJSON, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// evaluate 评估数学表达式
func (t *CalculatorTool) evaluate(expr string) (float64, error) {
	expr = strings.TrimSpace(expr)

	// 处理函数
	expr, err := t.evaluateFunctions(expr)
	if err != nil {
		return 0, err
	}

	// 处理括号
	for {
		start := strings.LastIndex(expr, "(")
		if start == -1 {
			break
		}
		end := strings.Index(expr[start:], ")")
		if end == -1 {
			return 0, fmt.Errorf("mismatched parentheses")
		}

		subExpr := expr[start+1 : end]
		result, err := t.evaluate(subExpr)
		if err != nil {
			return 0, err
		}

		expr = expr[:start] + t.formatNumber(result) + expr[start+end+1:]
	}

	// 处理运算符
	return t.evaluateOperators(expr)
}

// evaluateFunctions 处理数学函数
func (t *CalculatorTool) evaluateFunctions(expr string) (string, error) {
	functions := map[string]func(float64) float64{
		"sin":   math.Sin,
		"cos":   math.Cos,
		"tan":   math.Tan,
		"asin":  math.Asin,
		"acos":  math.Acos,
		"atan":  math.Atan,
		"sqrt":  math.Sqrt,
		"cbrt":  math.Cbrt,
		"abs":   math.Abs,
		"floor": math.Floor,
		"ceil":  math.Ceil,
		"round": math.Round,
		"log":   math.Log,
		"log10": math.Log10,
		"log2":  math.Log2,
		"exp":   math.Exp,
		"pi":    func(x float64) float64 { return math.Pi },
		"e":     func(x float64) float64 { return math.E },
	}

	for name, fn := range functions {
		expr = strings.ReplaceAll(expr, name+"()", t.formatNumber(fn(0)))
	}

	// 处理带参数的函数
	funcsWithArgs := []string{"sin", "cos", "tan", "asin", "acos", "atan", "sqrt", "abs", "floor", "ceil", "round", "log", "log10", "log2", "exp"}

	for _, name := range funcsWithArgs {
		pattern := name + "("
		for {
			i := strings.Index(expr, pattern)
			if i == -1 {
				break
			}

			// 找到参数
			start := i + len(pattern)
			end := start
			depth := 1
			for end < len(expr) {
				if expr[end] == '(' {
					depth++
				} else if expr[end] == ')' {
					depth--
					if depth == 0 {
						break
					}
				}
				end++
			}

			if depth != 0 {
				return "", fmt.Errorf("mismatched parentheses in function %s", name)
			}

			argStr := expr[start:end]
			arg, err := t.evaluate(argStr)
			if err != nil {
				return "", err
			}

			fn, ok := functions[name]
			if !ok {
				return "", fmt.Errorf("unknown function: %s", name)
			}

			result := fn(arg)
			expr = expr[:i] + t.formatNumber(result) + expr[end+1:]
		}
	}

	return expr, nil
}

// evaluateOperators 处理运算符
func (t *CalculatorTool) evaluateOperators(expr string) (float64, error) {
	expr = strings.TrimSpace(expr)

	// 首先处理幂运算
	for {
		i := strings.Index(expr, "**")
		if i == -1 {
			break
		}

		left, err := t.getNumber(expr[:i])
		if err != nil {
			return 0, err
		}

		right, err := t.getNumber(expr[i+2:])
		if err != nil {
			return 0, err
		}

		result := math.Pow(left, right)
		expr = t.formatNumber(result)
	}

	// 处理乘除取余
	for {
		i := t.findOperator(expr, []string{"*", "/", "%"})
		if i == -1 {
			break
		}

		op := expr[i : i+1]
		left, err := t.getNumber(expr[:i])
		if err != nil {
			return 0, err
		}

		right, err := t.getNumber(expr[i+1:])
		if err != nil {
			return 0, err
		}

		var result float64
		switch op {
		case "*":
			result = left * right
		case "/":
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			result = left / right
		case "%":
			result = math.Mod(left, right)
		}

		expr = t.formatNumber(result)
	}

	// 处理加减
	for {
		i := t.findOperator(expr, []string{"+", "-"})
		if i == -1 {
			break
		}

		// 跳过开头的负号
		if i == 0 {
			expr = "0" + expr
			i = 1
		}

		op := expr[i : i+1]
		left, err := t.getNumber(expr[:i])
		if err != nil {
			return 0, err
		}

		right, err := t.getNumber(expr[i+1:])
		if err != nil {
			return 0, err
		}

		var result float64
		switch op {
		case "+":
			result = left + right
		case "-":
			result = left - right
		}

		expr = t.formatNumber(result)
	}

	// 解析最终数字
	return t.getNumber(expr)
}

// findOperator 找到运算符位置（从左到右优先级）
func (t *CalculatorTool) findOperator(expr string, operators []string) int {
	minIdx := -1
	minPri := len(operators) + 1

	// 处理负号开头
	start := 0
	if len(expr) > 0 && expr[0] == '-' {
		start = 1
	}

	for i := start; i < len(expr); i++ {
		for j, op := range operators {
			if strings.HasPrefix(expr[i:], op) && (minIdx == -1 || j < minPri) {
				minIdx = i
				minPri = j
			}
		}
	}

	return minIdx
}

// getNumber 从字符串获取数字
func (t *CalculatorTool) getNumber(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty expression")
	}
	return strconv.ParseFloat(s, 64)
}

// formatNumber 格式化数字
func (t *CalculatorTool) formatNumber(n float64) string {
	if n == math.Floor(n) && math.Abs(n) < 1e15 {
		return fmt.Sprintf("%d", int64(n))
	}
	return strconv.FormatFloat(n, 'f', -1, 64)
}

// ToDefinition 转换为工具定义
func (t *CalculatorTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}
