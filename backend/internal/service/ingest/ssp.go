package ingest

import (
	"io"
	"strconv"

	"sibur-petrochem-price-service/internal/domain"
)

// Колонки выгрузки ssp.xlsx — формат documents/Специфика_столбцов_листов_экселя.md.
var sspColumns = []string{
	"row_id", "period", "customer_asv", "customer", "mtr_nsi_code", "mtr_nsi_name",
	"contract", "currency", "forecast", "country", "region", "market", "client_id", "client_name",
}

// ParseSsp — прогноз спроса из .xlsx. Любая проблема (вплоть до нечитаемого файла)
// возвращается списком issues — файл отклоняется целиком.
func ParseSsp(r io.Reader) (rows []domain.SspRow, issues []Issue) {
	src, issues := openSheet(r, sspColumns)
	if len(issues) > 0 {
		return nil, issues
	}

	rows = make([]domain.SspRow, 0, len(src.rows))
	seen := make(map[string]int, len(src.rows))
	for i, raw := range src.rows {
		ref := i + headerRows + 1
		row, rowIssues := parseSspRow(src, raw, ref)
		if len(rowIssues) > 0 {
			issues = append(issues, rowIssues...)
			continue
		}

		key := row.DemandKey()
		if firstRef, dup := seen[key]; dup {
			issues = append(issues, Issue{Row: ref, Reason: "дубль (row_id, period) — уже встречался в строке " + strconv.Itoa(firstRef)})
			continue
		}
		seen[key] = ref
		rows = append(rows, row)
	}
	if len(issues) > 0 {
		return nil, issues
	}

	return rows, nil
}

func parseSspRow(src sheet, raw []string, ref int) (row domain.SspRow, issues []Issue) {
	rowID, ok := parseRequiredInt(src.cell(raw, "row_id"))
	if !ok {
		issues = append(issues, Issue{Row: ref, Column: "row_id", Reason: "ожидается целое число"})
	}

	period, ok := parseDate(src.cell(raw, "period"))
	if !ok {
		issues = append(issues, Issue{Row: ref, Column: "period", Reason: "некорректная дата"})
	}

	materialID, ok := parseRequiredInt(src.cell(raw, "mtr_nsi_code"))
	if !ok {
		issues = append(issues, Issue{Row: ref, Column: "mtr_nsi_code", Reason: "ожидается целое число"})
	}

	var forecast *float64
	if cell := src.cell(raw, "forecast"); cell != "" {
		value, parseErr := strconv.ParseFloat(cell, 64)
		if parseErr != nil {
			issues = append(issues, Issue{Row: ref, Column: "forecast", Reason: "ожидается число"})
		} else {
			forecast = &value
		}
	}

	for _, column := range []string{"mtr_nsi_name", "contract", "currency", "client_id", "client_name"} {
		if src.cell(raw, column) == "" {
			issues = append(issues, Issue{Row: ref, Column: column, Reason: "пустое обязательное значение"})
		}
	}
	if len(issues) > 0 {
		return domain.SspRow{}, issues
	}

	return domain.SspRow{
		RowID:        rowID,
		Period:       period,
		CustomerASV:  src.cell(raw, "customer_asv"),
		Customer:     src.cell(raw, "customer"),
		MaterialID:   materialID,
		MaterialName: src.cell(raw, "mtr_nsi_name"),
		Contract:     src.cell(raw, "contract"),
		Currency:     src.cell(raw, "currency"),
		Forecast:     forecast,
		Country:      src.cell(raw, "country"),
		Region:       src.cell(raw, "region"),
		Market:       src.cell(raw, "market"),
		ClientID:     src.cell(raw, "client_id"),
		ClientName:   src.cell(raw, "client_name"),
	}, nil
}

func parseRequiredInt(cell string) (value int64, ok bool) {
	parsed, err := strconv.ParseInt(cell, 10, 64)
	if cell == "" || err != nil {
		return 0, false
	}

	return parsed, true
}
