// Package pricing — движок расчёта цен: порт эталонного алгоритма
// pricing_pipeline_fixed.py (подбор кандидатов, разрешение компонентов,
// котировки/курсы с каскадом Факт→ОФ→ППР, вычисление, выбор кандидата).
package pricing

import (
	"sibur-petrochem-price-service/internal/service/pricing/expr"
)

// Engine — движок расчёта цен.
type Engine struct {
	eval expr.Evaluator
}

func NewEngine() *Engine {
	return &Engine{eval: expr.New()}
}
