package calculations

import (
	"sibur-petrochem-price-service/internal/domain"
	"sibur-petrochem-price-service/internal/service/pricing"
)

const million = 1_000_000

// classifiedErrors — классифицированные ошибки канона KPI.
var classifiedErrors = map[domain.Status]bool{
	domain.StatusFormulaNotFound: true,
	domain.StatusComponentError:  true,
	domain.StatusInvalidFormula:  true,
}

// Kpi — пять канонических показателей расчёта (пересчитываются после мутаций).
func (s *Service) Kpi(id string) (Kpi, error) {
	calc, err := s.find(id)
	if err != nil {
		return Kpi{}, err
	}

	calc.mu.RLock()
	defer calc.mu.RUnlock()

	return computeKpi(calc.effectiveRows(), calc.result), nil
}

func computeKpi(rows []domain.CalcRow, result pricing.RunResult) Kpi {
	kpi := Kpi{
		FormulaCoveragePct:   coveragePct(rows),
		FormulasOkPct:        formulasOkPct(result),
		CalcErrorRows:        calcErrorRows(rows),
		ControlSumMln:        controlSumMln(rows, result),
		UnclassifiedErrorPct: unclassifiedErrorPct(rows),
	}

	return kpi
}

// coveragePct — % покрытия формулами: строки Formula с кандидатами среди всех Formula (SPOT исключён).
func coveragePct(rows []domain.CalcRow) int {
	total, covered := 0, 0
	for _, row := range rows {
		if row.Status == domain.StatusSpotNotCalculated || row.Contract != "Formula" {
			continue
		}
		total++
		if row.CandidateCount > 0 {
			covered++
		}
	}

	return percentOf(covered, total)
}

// formulasOkPct — % формул, у которых все кандидаты CALCULATED, среди всех найденных формул.
func formulasOkPct(result pricing.RunResult) int {
	okByFormula := map[string]bool{}
	result.EachCandidate(func(_ string, candidate domain.CandidateResult) {
		ok, seen := okByFormula[candidate.FormulaID]
		if !seen {
			ok = true
		}
		okByFormula[candidate.FormulaID] = ok && candidate.Status == domain.StatusCalculated
	})

	okCount := 0
	for _, ok := range okByFormula {
		if ok {
			okCount++
		}
	}

	return percentOf(okCount, len(okByFormula))
}

// calcErrorRows — строк с ошибкой расчёта: формула найдена, цены нет.
func calcErrorRows(rows []domain.CalcRow) int {
	count := 0
	for _, row := range rows {
		if row.Contract != "Formula" || row.CandidateCount == 0 || row.Price != nil {
			continue
		}
		if row.Status == domain.StatusComponentError || row.Status == domain.StatusInvalidFormula {
			count++
		}
	}

	return count
}

// controlSumMln — Σ(price × forecast × курс_к_RUB) / 1e6 по всем строкам с ценой.
func controlSumMln(rows []domain.CalcRow, result pricing.RunResult) float64 {
	sum := 0.0
	for _, row := range rows {
		if row.Price == nil || row.Forecast == nil {
			continue
		}
		rate := 1.0
		if row.Currency != "RUB" {
			resolved, err := result.RateToRUB(row.Currency, row.Period)
			if err != nil {
				continue
			}
			rate = resolved
		}
		sum += *row.Price * *row.Forecast * rate
	}

	return sum / million
}

// unclassifiedErrorPct — % неклассифицированных ошибок среди всех строк с ошибкой.
func unclassifiedErrorPct(rows []domain.CalcRow) int {
	total, unclassified := 0, 0
	for _, row := range rows {
		classified := classifiedErrors[row.Status]
		hasError := row.Error != "" && row.Price == nil &&
			row.Status != domain.StatusSpotNotCalculated && row.Status != domain.StatusManual
		if !classified && !hasError {
			continue
		}
		total++
		if !classified {
			unclassified++
		}
	}

	return percentOf(unclassified, total)
}
