package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCalculatorTool tests for CalculatorTool
func TestCalculatorTool_NewCalculatorTool(t *testing.T) {
	tool := NewCalculatorTool()
	require.NotNil(t, tool)
	assert.Equal(t, "calculator", tool.Name())
	assert.Contains(t, tool.Description(), "数学计算")
}

func TestCalculatorTool_Execute_SimpleArithmetic(t *testing.T) {
	tool := NewCalculatorTool()

	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{"Addition", "2 + 2", false},
		{"Subtraction", "10 - 3", false},
		{"Multiplication", "3 * 4", false},
		{"Division", "15 / 3", false},
		{"Modulo", "17 % 5", false},
		{"Negative", "-5 + 3", false},
		{"Decimal", "3.14 + 2.86", false},
		{"Parentheses", "(2 + 3) * 4", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), map[string]interface{}{
				"expression": tt.expression,
			})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Contains(t, result, `"result":`)
			}
		})
	}
}

func TestCalculatorTool_Execute_Power(t *testing.T) {
	tool := NewCalculatorTool()

	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{"Power 2**3", "2 ** 3", false},
		{"Power 10**2", "10 ** 2", false},
		{"Power sqrt", "sqrt(16)", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), map[string]interface{}{
				"expression": tt.expression,
			})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Contains(t, result, `"result":`)
			}
		})
	}
}

func TestCalculatorTool_Execute_MathFunctions(t *testing.T) {
	tool := NewCalculatorTool()

	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{"Abs", "abs(-5)", false},
		{"Floor", "floor(3.7)", false},
		{"Ceil", "ceil(3.2)", false},
		{"Round", "round(3.5)", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), map[string]interface{}{
				"expression": tt.expression,
			})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Contains(t, result, `"result":`)
			}
		})
	}
}

func TestCalculatorTool_Execute_Constants(t *testing.T) {
	tool := NewCalculatorTool()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"expression": "pi() * 2",
	})

	require.NoError(t, err)
	assert.Contains(t, result, `"result":`)
}

func TestCalculatorTool_Execute_InvalidExpression(t *testing.T) {
	tool := NewCalculatorTool()

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"expression": "",
	})

	assert.Error(t, err)
}

func TestCalculatorTool_Execute_DivisionByZero(t *testing.T) {
	tool := NewCalculatorTool()

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"expression": "1 / 0",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "division by zero")
}
