package calculations

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"sibur-petrochem-price-service/internal/domain"
)

// Rows — таблица строк с фильтром, поиском, сортировкой и пагинацией.
func (s *Service) Rows(id string, query RowsQuery) (RowsPage, error) {
	calc, err := s.find(id)
	if err != nil {
		return RowsPage{}, err
	}

	calc.mu.RLock()
	defer calc.mu.RUnlock()

	rows := calc.effectiveRows()

	counts := make(map[domain.Status]int, len(rows))
	for _, row := range rows {
		counts[row.Status]++
	}

	filtered := filterRows(rows, query)
	sortRows(filtered, query)

	total := len(filtered)
	page := paginate(filtered, query.Offset, query.Limit)

	return RowsPage{Items: page, Total: total, StatusCounts: counts}, nil
}

// Details — полная расшифровка строки с учётом ручных правок и выбора формулы.
func (s *Service) Details(id, rowID string) (Details, error) {
	calc, err := s.find(id)
	if err != nil {
		return Details{}, err
	}

	calc.mu.RLock()
	defer calc.mu.RUnlock()

	return calc.details(rowID)
}

// SetManualPrice — ручная цена строки (строго больше нуля).
func (s *Service) SetManualPrice(id, rowID string, price float64) (Details, error) {
	calc, err := s.find(id)
	if err != nil {
		return Details{}, err
	}

	if price <= 0 {
		return Details{}, fmt.Errorf("%w: цена должна быть больше нуля", domain.ErrInvalidPrice)
	}

	calc.mu.Lock()
	defer calc.mu.Unlock()

	row, err := calc.findRow(rowID)
	if err != nil {
		return Details{}, err
	}
	calc.manual[row.DemandKey] = price

	return calc.details(rowID)
}

// ResetManualPrice — сброс ручной правки, возврат расчётной цены.
func (s *Service) ResetManualPrice(id, rowID string) (Details, error) {
	calc, err := s.find(id)
	if err != nil {
		return Details{}, err
	}

	calc.mu.Lock()
	defer calc.mu.Unlock()

	row, err := calc.findRow(rowID)
	if err != nil {
		return Details{}, err
	}
	delete(calc.manual, row.DemandKey)

	return calc.details(rowID)
}

// SelectFormula — выбор формулы из списка подходящих кандидатов строки.
func (s *Service) SelectFormula(id, rowID, formulaID string) (Details, error) {
	calc, err := s.find(id)
	if err != nil {
		return Details{}, err
	}

	calc.mu.Lock()
	defer calc.mu.Unlock()

	row, err := calc.findRow(rowID)
	if err != nil {
		return Details{}, err
	}

	found := false
	for _, candidate := range calc.result.Candidates(row.DemandKey) {
		if candidate.FormulaID == formulaID {
			found = true

			break
		}
	}
	if !found {
		return Details{}, fmt.Errorf("%w: %s", domain.ErrFormulaNotAllowed, formulaID)
	}

	calc.selected[row.DemandKey] = formulaID

	return calc.details(rowID)
}

// ExportRows — все строки расчёта с учётом правок (для выгрузки).
func (s *Service) ExportRows(id string) ([]domain.CalcRow, error) {
	calc, err := s.find(id)
	if err != nil {
		return nil, err
	}

	calc.mu.RLock()
	defer calc.mu.RUnlock()

	return calc.effectiveRows(), nil
}

// ── внутреннее состояние расчёта ──

func (c *calculation) effectiveRows() []domain.CalcRow {
	rows := make([]domain.CalcRow, 0, len(c.result.Rows))
	for _, row := range c.result.Rows {
		rows = append(rows, c.effectiveRow(row))
	}

	return rows
}

// effectiveRow — строка с применёнными мутациями (ручная цена, выбор формулы).
func (c *calculation) effectiveRow(row domain.CalcRow) domain.CalcRow {
	if price, ok := c.manual[row.DemandKey]; ok {
		manualPrice := price
		row.Status = domain.StatusManual
		row.Price = &manualPrice
		row.RequiresReview = false

		return row
	}

	formulaID, ok := c.selected[row.DemandKey]
	if !ok {
		return row
	}

	for _, candidate := range c.result.Candidates(row.DemandKey) {
		if candidate.FormulaID != formulaID {
			continue
		}
		row.Status = candidate.Status
		if candidate.Status == domain.StatusCalculated && !candidate.IsFormulaActual {
			row.Status = domain.StatusCalculatedExpired
		}
		row.Price = candidate.Price
		row.RequiresReview = false
		row.Warning = candidate.Warning
		row.Error = candidate.Error

		break
	}

	return row
}

func (c *calculation) findRow(rowID string) (domain.CalcRow, error) {
	parsed, err := strconv.ParseInt(rowID, 10, 64)
	if err != nil {
		return domain.CalcRow{}, fmt.Errorf("%w: %s", domain.ErrRowNotFound, rowID)
	}

	for _, row := range c.result.Rows {
		if row.RowID == parsed {
			return row, nil
		}
	}

	return domain.CalcRow{}, fmt.Errorf("%w: %s", domain.ErrRowNotFound, rowID)
}

func (c *calculation) details(rowID string) (Details, error) {
	baseRow, err := c.findRow(rowID)
	if err != nil {
		return Details{}, err
	}

	row := c.effectiveRow(baseRow)
	key := row.DemandKey
	alternatives := c.result.Candidates(key)

	appliedID := ""
	userSelected := false
	if selectedID, ok := c.selected[key]; ok {
		appliedID = selectedID
		userSelected = true
	} else {
		for _, candidate := range alternatives {
			if candidate.IsSelected {
				appliedID = candidate.FormulaID
			}
		}
	}

	details := Details{Row: row, EqualPriorityCount: row.EqualPriorityCount}

	for i := range alternatives {
		isApplied := alternatives[i].FormulaID == appliedID
		alternatives[i].IsSelected = isApplied
		if !isApplied {
			continue
		}
		applied := alternatives[i]
		if userSelected {
			applied.SelectionReason = domain.SelectionUserSelected
		}
		details.Applied = &applied
		details.Components = applied.Components
		details.PriceFormulaCurrency = applied.PriceFormulaCurrency
		details.Conversion = applied.Conversion
	}
	details.Alternatives = alternatives

	if price, ok := c.manual[key]; ok {
		manualPrice := price
		details.ManualPrice = &manualPrice
	}

	return details, nil
}

// ── фильтры и сортировка ──

func filterRows(rows []domain.CalcRow, query RowsQuery) []domain.CalcRow {
	needle := strings.ToLower(strings.TrimSpace(query.Query))
	out := make([]domain.CalcRow, 0, len(rows))
	for _, row := range rows {
		if query.Status != nil && row.Status != *query.Status {
			continue
		}
		if needle != "" && !rowMatches(row, needle) {
			continue
		}
		out = append(out, row)
	}

	return out
}

func rowMatches(row domain.CalcRow, needle string) bool {
	haystack := strings.ToLower(
		row.ClientName + " " + row.MaterialName + " " + strconv.FormatInt(row.RowID, 10),
	)

	return strings.Contains(haystack, needle)
}

func sortRows(rows []domain.CalcRow, query RowsQuery) {
	desc := query.Order == "desc"
	sort.SliceStable(rows, func(i, j int) bool {
		less := compareRows(rows[i], rows[j], query.Sort)
		if desc {
			return !less && !rowsEqualBy(rows[i], rows[j], query.Sort)
		}

		return less
	})
}

func compareRows(a, b domain.CalcRow, key string) bool {
	switch key {
	case "client":
		return a.ClientName < b.ClientName
	case "material":
		return a.MaterialName < b.MaterialName
	case "volume":
		return floatOrMinus(a.Forecast) < floatOrMinus(b.Forecast)
	case "price":
		return floatOrMinus(a.Price) < floatOrMinus(b.Price)
	default:
		return a.RowID < b.RowID
	}
}

func rowsEqualBy(a, b domain.CalcRow, key string) bool {
	return !compareRows(a, b, key) && !compareRows(b, a, key)
}

func floatOrMinus(value *float64) float64 {
	if value == nil {
		return -1
	}

	return *value
}

func paginate(rows []domain.CalcRow, offset, limit int) []domain.CalcRow {
	if offset >= len(rows) {
		return nil
	}
	end := len(rows)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}

	return rows[offset:end]
}
