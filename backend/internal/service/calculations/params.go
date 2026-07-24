package calculations

import (
	"sibur-petrochem-price-service/internal/domain"
)

// CalcParams — параметры запуска расчёта: диапазон месяцев и фильтры продукта/клиента.
// Границы диапазона (формат YYYY-MM) необязательны: nil — открыта. Пустой список
// продуктов/клиентов не сужает выборку. Внутри фильтра — OR, между продуктом и клиентом — AND.
type CalcParams struct {
	PeriodFrom *string
	PeriodTo   *string
	ProductIDs []int64
	ClientIDs  []string
}

// filterDemand — строки спроса, попавшие в пересечение всех заданных фильтров.
func filterDemand(rows []domain.SspRow, params CalcParams) []domain.SspRow {
	products := make(map[int64]struct{}, len(params.ProductIDs))
	for _, id := range params.ProductIDs {
		products[id] = struct{}{}
	}
	clients := make(map[string]struct{}, len(params.ClientIDs))
	for _, id := range params.ClientIDs {
		clients[id] = struct{}{}
	}

	filtered := make([]domain.SspRow, 0, len(rows))
	for _, row := range rows {
		if !params.matchesPeriod(row) {
			continue
		}
		if len(products) > 0 {
			if _, ok := products[row.MaterialID]; !ok {
				continue
			}
		}
		if len(clients) > 0 {
			if _, ok := clients[row.ClientID]; !ok {
				continue
			}
		}
		filtered = append(filtered, row)
	}

	return filtered
}

// matchesPeriod — месяц строки в диапазоне [PeriodFrom, PeriodTo] включительно (nil — граница открыта).
func (p CalcParams) matchesPeriod(row domain.SspRow) bool {
	month := row.Period.Format("2006-01")
	if p.PeriodFrom != nil && month < *p.PeriodFrom {
		return false
	}
	if p.PeriodTo != nil && month > *p.PeriodTo {
		return false
	}

	return true
}

// label — человекочитаемая метка периода расчёта для отображения.
func (p CalcParams) label() string {
	switch {
	case p.PeriodFrom == nil && p.PeriodTo == nil:
		return "весь период"
	case p.PeriodFrom != nil && p.PeriodTo != nil:
		return *p.PeriodFrom + " — " + *p.PeriodTo
	case p.PeriodFrom != nil:
		return "с " + *p.PeriodFrom
	default:
		return "по " + *p.PeriodTo
	}
}
