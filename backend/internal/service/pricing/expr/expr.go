// Package expr — разбор и вычисление выражений договорных формул.
// Порт safe-eval эталонного алгоритма (pricing_pipeline_fixed.py):
// операторы + - * / % **, унарный минус, сравнения, функции IF/RND_X/MIN/MAX,
// переменные с символами $ € ¥ ₱, имена могут начинаться с цифры.
package expr

import (
	"errors"
	"fmt"
	"math"

	"github.com/shopspring/decimal"
)

// ErrEvaluation — базовая ошибка вычисления выражения.
var ErrEvaluation = errors.New("formula evaluation error")

// Evaluator — вычислитель выражений формул.
type Evaluator struct{}

func New() Evaluator {
	return Evaluator{}
}

// AnalyzeResult — результат разбора выражения для POST /formulas/parse.
type AnalyzeResult struct {
	Valid     bool
	Variables []string
	Functions []string
	Errors    []ParseError
}

// ParseError — ошибка разбора с позицией (если известна).
type ParseError struct {
	Message  string
	Position *int
}

var allowedFunctions = map[string]bool{"IF": true, "RND_X": true, "MIN": true, "MAX": true}

// Analyze — разбор без вычисления: переменные в порядке появления, функции, ошибки.
func (Evaluator) Analyze(expression string) AnalyzeResult {
	root, parsed, err := parseExpression(expression)
	if err != nil {
		return analyzeError(err)
	}

	if name, pos, ok := findForbiddenCall(root); ok {
		position := pos

		return AnalyzeResult{
			Valid:     false,
			Variables: parsed.variables,
			Functions: parsed.functions,
			Errors:    []ParseError{{Message: "запрещённая функция: " + name, Position: &position}},
		}
	}

	return AnalyzeResult{
		Valid:     true,
		Variables: parsed.variables,
		Functions: parsed.functions,
		Errors:    []ParseError{},
	}
}

// Evaluate — вычисление выражения с подстановкой значений переменных.
func (Evaluator) Evaluate(expression string, variables map[string]float64) (float64, error) {
	root, _, err := parseExpression(expression)
	if err != nil {
		return 0, fmt.Errorf("%w: Синтаксическая ошибка формулы: %s", ErrEvaluation, err.Error())
	}

	result, err := evalNode(root, variables)
	if err != nil {
		return 0, err
	}

	if result.isBool || math.IsNaN(result.num) || math.IsInf(result.num, 0) {
		return 0, fmt.Errorf("%w: Формула вернула некорректное значение", ErrEvaluation)
	}

	return result.num, nil
}

func analyzeError(err error) AnalyzeResult {
	var syntax *syntaxError
	if errors.As(err, &syntax) {
		position := syntax.position

		return AnalyzeResult{
			Valid:  false,
			Errors: []ParseError{{Message: "синтаксическая ошибка: " + syntax.message, Position: &position}},
		}
	}

	return AnalyzeResult{
		Valid:  false,
		Errors: []ParseError{{Message: err.Error()}},
	}
}

func findForbiddenCall(root node) (name string, pos int, found bool) {
	stack := []node{root}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		switch typed := current.(type) {
		case callNode:
			if !allowedFunctions[typed.name] {
				return typed.name, typed.pos, true
			}
			stack = append(stack, typed.args...)
		case unaryNode:
			stack = append(stack, typed.operand)
		case binNode:
			stack = append(stack, typed.left, typed.right)
		case cmpNode:
			stack = append(stack, typed.first)
			stack = append(stack, typed.rest...)
		}
	}

	return "", 0, false
}

// value — результат вычисления узла: число либо булево (результат сравнения).
type value struct {
	num    float64
	b      bool
	isBool bool
}

func numValue(num float64) value { return value{num: num} }

func boolValue(b bool) value { return value{b: b, isBool: true} }

// asNumber — python-семантика: True == 1, False == 0 в арифметике.
func (v value) asNumber() float64 {
	if v.isBool {
		if v.b {
			return 1
		}

		return 0
	}

	return v.num
}

// truthy — python-семантика bool(): ненулевое число истинно.
func (v value) truthy() bool {
	if v.isBool {
		return v.b
	}

	return v.num != 0
}

func evalNode(root node, variables map[string]float64) (value, error) {
	switch typed := root.(type) {
	case numNode:
		// Alias-подстановка эталона заменяет и числовые токены, если так названа переменная.
		if val, ok := variables[typed.text]; ok {
			return numValue(val), nil
		}

		return numValue(typed.value), nil

	case varNode:
		val, ok := variables[typed.name]
		if !ok {
			return value{}, fmt.Errorf("%w: Не найдена переменная: %s", ErrEvaluation, typed.name)
		}

		return numValue(val), nil

	case unaryNode:
		return evalUnary(typed, variables)

	case binNode:
		return evalBinary(typed, variables)

	case cmpNode:
		return evalCompare(typed, variables)

	case callNode:
		return evalCall(typed, variables)

	default:
		return value{}, fmt.Errorf("%w: Запрещённый элемент формулы", ErrEvaluation)
	}
}

func evalUnary(n unaryNode, variables map[string]float64) (value, error) {
	operand, err := evalNode(n.operand, variables)
	if err != nil {
		return value{}, err
	}

	if n.op == "-" {
		return numValue(-operand.asNumber()), nil
	}

	return numValue(operand.asNumber()), nil
}

func evalBinary(n binNode, variables map[string]float64) (value, error) {
	left, err := evalNode(n.left, variables)
	if err != nil {
		return value{}, err
	}

	right, err := evalNode(n.right, variables)
	if err != nil {
		return value{}, err
	}

	a, b := left.asNumber(), right.asNumber()

	switch n.op {
	case "+":
		return numValue(a + b), nil
	case "-":
		return numValue(a - b), nil
	case "*":
		return numValue(a * b), nil
	case "/":
		if b == 0 {
			return value{}, fmt.Errorf("%w: деление на ноль", ErrEvaluation)
		}

		return numValue(a / b), nil
	case "%":
		if b == 0 {
			return value{}, fmt.Errorf("%w: деление на ноль", ErrEvaluation)
		}

		return numValue(pythonMod(a, b)), nil
	case "**":
		return numValue(math.Pow(a, b)), nil
	default:
		return value{}, fmt.Errorf("%w: Запрещённый элемент формулы: %s", ErrEvaluation, n.op)
	}
}

// pythonMod — остаток со знаком делителя, как в python (-7 %% 3 == 2).
func pythonMod(a, b float64) float64 {
	result := math.Mod(a, b)
	if result != 0 && (result < 0) != (b < 0) {
		result += b
	}

	return result
}

func evalCompare(n cmpNode, variables map[string]float64) (value, error) {
	left, err := evalNode(n.first, variables)
	if err != nil {
		return value{}, err
	}

	previous := left.asNumber()
	for i, op := range n.ops {
		right, err := evalNode(n.rest[i], variables)
		if err != nil {
			return value{}, err
		}

		current := right.asNumber()
		if !compare(op, previous, current) {
			return boolValue(false), nil
		}
		previous = current
	}

	return boolValue(true), nil
}

func compare(op string, a, b float64) bool {
	switch op {
	case "<":
		return a < b
	case "<=":
		return a <= b
	case ">":
		return a > b
	case ">=":
		return a >= b
	case "=", "==":
		return a == b
	case "<>", "!=":
		return a != b
	default:
		return false
	}
}

func evalCall(n callNode, variables map[string]float64) (value, error) {
	if !allowedFunctions[n.name] {
		return value{}, fmt.Errorf("%w: Запрещённая функция: %s", ErrEvaluation, n.name)
	}

	args := make([]value, 0, len(n.args))
	for _, argNode := range n.args {
		arg, err := evalNode(argNode, variables)
		if err != nil {
			return value{}, err
		}
		args = append(args, arg)
	}

	switch n.name {
	case "IF":
		return evalIf(args)
	case "RND_X":
		return evalRound(args)
	case "MIN", "MAX":
		return evalMinMax(n.name, args)
	default:
		return value{}, fmt.Errorf("%w: Запрещённая функция: %s", ErrEvaluation, n.name)
	}
}

func evalIf(args []value) (value, error) {
	const maxArgs = 3
	if len(args) < 2 || len(args) > maxArgs {
		return value{}, fmt.Errorf("%w: IF ожидает 2 или 3 аргумента", ErrEvaluation)
	}

	if args[0].truthy() {
		return args[1], nil
	}
	if len(args) == maxArgs {
		return args[2], nil
	}

	return numValue(0), nil
}

// evalRound — коммерческое округление ROUND_HALF_UP, как Decimal(str(v)) в эталоне.
func evalRound(args []value) (value, error) {
	const maxArgs = 2
	if len(args) < 1 || len(args) > maxArgs {
		return value{}, fmt.Errorf("%w: RND_X ожидает 1 или 2 аргумента", ErrEvaluation)
	}

	digits := 0
	if len(args) == maxArgs {
		digits = int(args[1].asNumber())
	}

	rounded := decimal.NewFromFloat(args[0].asNumber()).Round(int32(digits))
	result, _ := rounded.Float64()

	return numValue(result), nil
}

func evalMinMax(name string, args []value) (value, error) {
	if len(args) == 0 {
		return value{}, fmt.Errorf("%w: %s ожидает хотя бы один аргумент", ErrEvaluation, name)
	}

	result := args[0].asNumber()
	for _, arg := range args[1:] {
		current := arg.asNumber()
		if (name == "MIN" && current < result) || (name == "MAX" && current > result) {
			result = current
		}
	}

	return numValue(result), nil
}
