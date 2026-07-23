package ingest

import (
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// sheet — первый лист книги: индексы колонок по нормализованным заголовкам + строки данных.
type sheet struct {
	columns map[string]int
	rows    [][]string
}

// headerRows — строка заголовков занимает первую строку листа; данные начинаются со второй.
const headerRows = 1

// openSheet читает первый лист .xlsx и проверяет наличие обязательных колонок.
// Все проблемы (включая нечитаемый файл) возвращаются списком issues.
func openSheet(r io.Reader, required []string) (sheet, []Issue) {
	book, err := excelize.OpenReader(r)
	if err != nil {
		return sheet{}, []Issue{{Reason: "файл не читается как .xlsx"}}
	}
	defer func() { _ = book.Close() }()

	rows, err := book.GetRows(book.GetSheetName(0), excelize.Options{RawCellValue: true})
	if err != nil {
		return sheet{}, []Issue{{Reason: "первый лист книги не читается"}}
	}
	if len(rows) == 0 {
		return sheet{}, []Issue{{Reason: "лист пуст — нет строки заголовков"}}
	}

	columns := make(map[string]int, len(rows[0]))
	for idx, header := range rows[0] {
		name := normalizeHeader(header)
		if _, exists := columns[name]; !exists {
			columns[name] = idx
		}
	}

	var issues []Issue
	for _, name := range required {
		if _, ok := columns[name]; !ok {
			issues = append(issues, Issue{Row: headerRows, Column: name, Reason: "обязательная колонка отсутствует"})
		}
	}
	if len(issues) > 0 {
		return sheet{}, issues
	}

	return sheet{columns: columns, rows: rows[headerRows:]}, nil
}

// cell — значение колонки в строке; пустая строка, если ячейка отсутствует.
func (s sheet) cell(row []string, column string) string {
	idx, ok := s.columns[column]
	if !ok || idx >= len(row) {
		return ""
	}

	return strings.TrimSpace(row[idx])
}

// normalizeHeader — trim, NBSP → пробел, схлопывание повторных пробелов.
func normalizeHeader(header string) string {
	const nbsp = " "

	return strings.Join(strings.Fields(strings.ReplaceAll(header, nbsp, " ")), " ")
}

// parseDate — дата из строкового представления либо excel-серийного номера.
func parseDate(raw string) (time.Time, bool) {
	layouts := []string{"2006-01-02 15:04:05", "2006-01-02", "02.01.2006 15:04:05", "02.01.2006"}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed, true
		}
	}

	serial, err := strconv.ParseFloat(raw, 64)
	if err != nil || serial <= 0 {
		return time.Time{}, false
	}
	parsed, err := excelize.ExcelDateToTime(serial, false)
	if err != nil {
		return time.Time{}, false
	}

	return parsed, true
}
