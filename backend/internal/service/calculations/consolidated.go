package calculations

import (
	"fmt"
	"time"

	"sibur-petrochem-price-service/internal/domain"
)

// Submit — присоединение участка аналитика к сводному документу.
func (s *Service) Submit(id string) (Part, error) {
	calc, err := s.find(id)
	if err != nil {
		return Part{}, err
	}

	calc.mu.Lock()
	defer calc.mu.Unlock()

	if calc.submitted {
		return Part{}, fmt.Errorf("%w: %s", domain.ErrAlreadySubmitted, id)
	}
	calc.submitted = true
	calc.submittedAt = time.Now().UTC()

	return calc.part(), nil
}

// Consolidated — сводный документ за период: участки, канонические KPI, строки.
func (s *Service) Consolidated(period string) (Consolidated, error) {
	calcs := s.byPeriod(period)

	doc := Consolidated{Period: period}
	var allRows []domain.CalcRow
	for _, calc := range calcs {
		calc.mu.RLock()
		doc.Parts = append(doc.Parts, calc.part())
		rows := calc.effectiveRows()
		for _, row := range rows {
			doc.Rows = append(doc.Rows, ConsolidatedRow{
				AnalystName: AnalystName,
				IsDraft:     !calc.submitted,
				Row:         row,
			})
		}
		allRows = append(allRows, rows...)
		if len(calcs) > 0 {
			doc.Kpi = computeKpi(allRows, calc.result)
		}
		calc.mu.RUnlock()
	}
	doc.TotalRows = len(doc.Rows)

	return doc, nil
}

// part — участок текущего расчёта (вызывается под мьютексом расчёта).
func (c *calculation) part() Part {
	rows := c.effectiveRows()
	priced := 0
	for _, row := range rows {
		if row.Price != nil {
			priced++
		}
	}

	status := "draft"
	var submittedAt *time.Time
	if c.submitted {
		status = "joined"
		at := c.submittedAt
		submittedAt = &at
	}

	return Part{
		CalculationID: c.id,
		AnalystName:   AnalystName,
		PartName:      partName,
		Status:        status,
		RowCount:      len(rows),
		PricedPct:     percentOf(priced, len(rows)),
		SubmittedAt:   submittedAt,
	}
}
