package ingest

import (
	"io"
	"strconv"
	"strings"
	"time"

	"sibur-petrochem-price-service/internal/domain"
)

// Колонки выгрузки formulas.xlsx (русские заголовки SAP) — формат
// documents/Специфика_столбцов_листов_экселя.md.
const (
	colFormulaID   = "Ключ формулы"
	colFormulaText = "Формула"
	colGroupM      = "Подгруппа материалов"
	colPartner     = "Деловой партнер"
	colValidFrom   = `"Действительно с" - метка врем. объекта`
	colValidTo     = `Метка времени "Действительно по"`
	colCreatedAt   = "Дата создания"
	colInactive    = "Неактивна"
	colPriceType   = "Тип цены"
	colDocCurrency = "ВалютаДокумента"
	colMaterial    = "Материал"
	colClientID    = "client_id"
	colClientName  = "client_name"
)

var formulaColumns = []string{
	colFormulaID, colFormulaText, colGroupM, colPartner, colValidFrom, colValidTo,
	colCreatedAt, colInactive, colPriceType, colDocCurrency, colMaterial, colClientID, colClientName,
}

// ParseFormulas — каталог формул из .xlsx. Любая проблема (вплоть до нечитаемого
// файла) возвращается списком issues — файл отклоняется целиком.
func ParseFormulas(r io.Reader) (rows []domain.Formula, issues []Issue) {
	src, issues := openSheet(r, formulaColumns)
	if len(issues) > 0 {
		return nil, issues
	}

	rows = make([]domain.Formula, 0, len(src.rows))
	for i, raw := range src.rows {
		row, rowIssues := parseFormulaRow(src, raw, i+headerRows+1)
		if len(rowIssues) > 0 {
			issues = append(issues, rowIssues...)
			continue
		}
		rows = append(rows, row)
	}
	if len(issues) > 0 {
		return nil, issues
	}

	return rows, nil
}

func parseFormulaRow(src sheet, raw []string, ref int) (row domain.Formula, issues []Issue) {
	dates, issues := parseFormulaDates(src, raw, ref)

	inactive, ok := parseInactive(src.cell(raw, colInactive))
	if !ok {
		issues = append(issues, Issue{Row: ref, Column: colInactive, Reason: "ожидается пусто либо X"})
	}

	var materialID *int64
	if cell := src.cell(raw, colMaterial); cell != "" {
		value, parseErr := strconv.ParseInt(cell, 10, 64)
		if parseErr != nil {
			issues = append(issues, Issue{Row: ref, Column: colMaterial, Reason: "ожидается целое число"})
		} else {
			materialID = &value
		}
	}

	for _, column := range []string{colFormulaID, colFormulaText, colDocCurrency, colClientID, colClientName} {
		if src.cell(raw, column) == "" {
			issues = append(issues, Issue{Row: ref, Column: column, Reason: "пустое обязательное значение"})
		}
	}
	if len(issues) > 0 {
		return domain.Formula{}, issues
	}

	return domain.Formula{
		FormulaID:       src.cell(raw, colFormulaID),
		Text:            src.cell(raw, colFormulaText),
		MaterialGroupM:  src.cell(raw, colGroupM),
		BusinessPartner: src.cell(raw, colPartner),
		ValidFrom:       dates.validFrom,
		ValidTo:         dates.validTo,
		CreatedAt:       dates.createdAt,
		Inactive:        inactive,
		PriceType:       src.cell(raw, colPriceType),
		DocCurrency:     src.cell(raw, colDocCurrency),
		MaterialID:      materialID,
		ClientID:        src.cell(raw, colClientID),
		ClientName:      src.cell(raw, colClientName),
	}, nil
}

// formulaDates — три обязательные даты строки формулы.
type formulaDates struct {
	validFrom time.Time
	validTo   time.Time
	createdAt time.Time
}

func parseFormulaDates(src sheet, raw []string, ref int) (dates formulaDates, issues []Issue) {
	fields := []struct {
		column string
		dst    *time.Time
	}{
		{colValidFrom, &dates.validFrom},
		{colValidTo, &dates.validTo},
		{colCreatedAt, &dates.createdAt},
	}
	for _, field := range fields {
		parsed, ok := parseDate(src.cell(raw, field.column))
		if !ok {
			issues = append(issues, Issue{Row: ref, Column: field.column, Reason: "некорректная дата"})
			continue
		}
		*field.dst = parsed
	}

	return dates, issues
}

// parseInactive — признак «Неактивна»: пусто = активна, X = неактивна.
func parseInactive(cell string) (inactive, ok bool) {
	switch strings.ToUpper(cell) {
	case "":
		return false, true
	case "X":
		return true, true
	default:
		return false, false
	}
}
