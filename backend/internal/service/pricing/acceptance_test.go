package pricing

// Приёмочный тест: прогон движка на полном демо-наборе documents/*.csv
// и сверка с эталонным прогоном pricing_pipeline_fixed.py
// (fixture testdata/reference_results.csv: demand_key,status,price).

import (
	"encoding/csv"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sibur-petrochem-price-service/internal/domain"
)

func TestEngine_Run_ReferenceAcceptance(t *testing.T) {
	src := loadDemoSources(t)
	reference := loadReferenceResults(t)

	result := NewEngine().Run(src, nil)

	require.Len(t, result.Rows, len(reference))

	statusCounts := map[domain.Status]int{}
	mismatchedStatuses := 0
	mismatchedPrices := 0
	for _, row := range result.Rows {
		statusCounts[row.Status]++

		want, ok := reference[row.DemandKey]
		require.True(t, ok, "нет эталонной строки для %s", row.DemandKey)

		if string(row.Status) != want.status {
			mismatchedStatuses++
			if mismatchedStatuses <= 5 {
				t.Logf("status mismatch %s: got %s want %s", row.DemandKey, row.Status, want.status)
			}

			continue
		}

		if !priceEqual(row.Price, want.price) {
			mismatchedPrices++
			if mismatchedPrices <= 5 {
				t.Logf("price mismatch %s: got %v want %v", row.DemandKey, floatOrNil(row.Price), floatOrNil(want.price))
			}
		}
	}

	assert.Zero(t, mismatchedStatuses, "строк с расхождением статуса")
	assert.Zero(t, mismatchedPrices, "строк с расхождением цены")

	// Контрольное распределение эталонного прогона.
	assert.Equal(t, 1397, statusCounts[domain.StatusCalculated])
	assert.Equal(t, 1901, statusCounts[domain.StatusComponentError])
	assert.Equal(t, 478, statusCounts[domain.StatusFormulaNotFound])
	assert.Equal(t, 163, statusCounts[domain.StatusSpotNotCalculated])
	assert.Equal(t, 274, statusCounts[domain.StatusCalculatedExpired])
	assert.Equal(t, 65, statusCounts[domain.StatusFormulaConflict])
}

func floatOrNil(value *float64) any {
	if value == nil {
		return nil
	}

	return *value
}

func priceEqual(got *float64, want *float64) bool {
	if (got == nil) != (want == nil) {
		return false
	}
	if got == nil {
		return true
	}

	const relTolerance = 1e-6

	return math.Abs(*got-*want) <= relTolerance*math.Max(1, math.Abs(*want))
}

type referenceRow struct {
	status string
	price  *float64
}

func loadReferenceResults(t *testing.T) map[string]referenceRow {
	t.Helper()

	records := readCSV(t, filepath.Join("testdata", "reference_results.csv"))
	out := make(map[string]referenceRow, len(records.rows))
	for _, row := range records.rows {
		out[records.get(row, "demand_key")] = referenceRow{
			status: records.get(row, "status"),
			price:  parseFloatPtr(records.get(row, "price")),
		}
	}

	return out
}

// ── загрузка documents/*.csv (семантика normalize_sources эталона) ──

func loadDemoSources(t *testing.T) domain.Sources {
	t.Helper()

	dir := filepath.Join("..", "..", "..", "..", "documents")

	return domain.Sources{
		Ssp:            loadSsp(t, filepath.Join(dir, "ssp.csv")),
		Formulas:       loadFormulas(t, filepath.Join(dir, "formulas.csv")),
		Components:     loadComponents(t, filepath.Join(dir, "formula_components.csv")),
		TermTypes:      loadTermTypes(t, filepath.Join(dir, "term_types.csv")),
		Quotes:         loadQuotes(t, filepath.Join(dir, "quotes.csv")),
		QuoteMapping:   loadQuoteMapping(t, filepath.Join(dir, "quote_mapping.csv")),
		CurrencyRates:  loadCurrencyRates(t, filepath.Join(dir, "currency_rates.csv")),
		MaterialGroups: loadMaterialGroups(t, filepath.Join(dir, "material_groups.csv")),
	}
}

type csvTable struct {
	header map[string]int
	rows   [][]string
}

func (tab csvTable) get(row []string, column string) string {
	index, ok := tab.header[column]
	if !ok {
		return ""
	}

	return strings.TrimSpace(row[index])
}

func readCSV(t *testing.T, path string) csvTable {
	t.Helper()

	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	require.NoError(t, err)
	require.NotEmpty(t, records)

	header := make(map[string]int, len(records[0]))
	for i, name := range records[0] {
		header[strings.TrimSpace(name)] = i
	}

	return csvTable{header: header, rows: records[1:]}
}

// parseDate — дата с нормализацией к полуночи (как .dt.normalize() в эталоне).
func parseDate(t *testing.T, value string) time.Time {
	t.Helper()

	layouts := []string{"2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Truncate(24 * time.Hour)
		}
	}
	t.Fatalf("не разобрана дата: %q", value)

	return time.Time{}
}

func parseTimestamp(t *testing.T, value string) time.Time {
	t.Helper()

	layouts := []string{"2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	t.Fatalf("не разобран timestamp: %q", value)

	return time.Time{}
}

func parseFloatPtr(value string) *float64 {
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}

	return &parsed
}

func parseIntPtr(value string) *int64 {
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	whole := int64(parsed)

	return &whole
}

func loadSsp(t *testing.T, path string) []domain.SspRow {
	t.Helper()

	tab := readCSV(t, path)
	out := make([]domain.SspRow, 0, len(tab.rows))
	for _, row := range tab.rows {
		rowID, err := strconv.ParseInt(tab.get(row, "row_id"), 10, 64)
		require.NoError(t, err)
		materialID, err := strconv.ParseInt(tab.get(row, "mtr_nsi_code"), 10, 64)
		require.NoError(t, err)

		out = append(out, domain.SspRow{
			RowID:        rowID,
			Period:       parseDate(t, tab.get(row, "period")),
			MaterialID:   materialID,
			MaterialName: tab.get(row, "mtr_nsi_name"),
			Contract:     tab.get(row, "contract"),
			Currency:     tab.get(row, "currency"),
			Forecast:     parseFloatPtr(tab.get(row, "forecast")),
			ClientID:     tab.get(row, "client_id"),
			ClientName:   tab.get(row, "client_name"),
		})
	}

	return out
}

func loadFormulas(t *testing.T, path string) []domain.Formula {
	t.Helper()

	tab := readCSV(t, path)
	out := make([]domain.Formula, 0, len(tab.rows))
	for _, row := range tab.rows {
		out = append(out, domain.Formula{
			FormulaID:      tab.get(row, "Ключ формулы"),
			Text:           tab.get(row, "Формула"),
			MaterialGroupM: tab.get(row, "Подгруппа материалов"),
			ValidFrom:      parseDate(t, tab.get(row, `"Действительно с" - метка врем. объекта`)),
			ValidTo:        parseDate(t, tab.get(row, `Метка времени "Действительно по"`)),
			CreatedAt:      parseTimestamp(t, tab.get(row, "Дата создания")),
			Inactive:       tab.get(row, "Неактивна") == "X",
			DocCurrency:    tab.get(row, "ВалютаДокумента"),
			MaterialID:     parseIntPtr(tab.get(row, "Материал")),
			ClientID:       tab.get(row, "client_id"),
			ClientName:     tab.get(row, "client_name"),
		})
	}

	return out
}

func loadComponents(t *testing.T, path string) []domain.FormulaComponent {
	t.Helper()

	tab := readCSV(t, path)
	out := make([]domain.FormulaComponent, 0, len(tab.rows))
	for _, row := range tab.rows {
		termNo := 0
		if parsed := parseIntPtr(tab.get(row, "Номер терма в формуле")); parsed != nil {
			termNo = int(*parsed)
		}

		out = append(out, domain.FormulaComponent{
			FormulaID: tab.get(row, "Формула"),
			TermNo:    termNo,
			VarName:   tab.get(row, "Имя переменной"),
			ValidFrom: parseDate(t, tab.get(row, "Действительно с")),
			ValidTo:   parseDate(t, tab.get(row, "Действительно по")),
			TypeCode:  tab.get(row, "Тип фиксированного терма"),
			Value:     parseFloatPtr(tab.get(row, "Значение")),
			Currency:  tab.get(row, "Валюта"),
			QuoteName: tab.get(row, "Имя котировки"),
		})
	}

	return out
}

func loadTermTypes(t *testing.T, path string) []domain.TermType {
	t.Helper()

	tab := readCSV(t, path)
	out := make([]domain.TermType, 0, len(tab.rows))
	for _, row := range tab.rows {
		out = append(out, domain.TermType{
			TypeCode:  tab.get(row, "type_code"),
			TypeLabel: tab.get(row, "type_label"),
			Category:  tab.get(row, "category"),
		})
	}

	return out
}

func loadQuotes(t *testing.T, path string) []domain.Quote {
	t.Helper()

	tab := readCSV(t, path)
	out := make([]domain.Quote, 0, len(tab.rows))
	for _, row := range tab.rows {
		code, err := strconv.ParseInt(tab.get(row, "quote_code"), 10, 64)
		require.NoError(t, err)
		value, err := strconv.ParseFloat(tab.get(row, "quote_val"), 64)
		require.NoError(t, err)

		quote := domain.Quote{
			QuoteType: tab.get(row, "quote_type"),
			QuoteName: tab.get(row, "quote_name"),
			QuoteCode: code,
			QuoteDate: parseDate(t, tab.get(row, "quote_date")),
			Currency:  tab.get(row, "quote_currency"),
			Value:     value,
		}
		if raw := tab.get(row, "tech_load_ts"); raw != "" {
			loadTS := parseTimestamp(t, raw)
			quote.TechLoadTS = &loadTS
		}
		out = append(out, quote)
	}

	return out
}

func loadQuoteMapping(t *testing.T, path string) []domain.QuoteMapping {
	t.Helper()

	tab := readCSV(t, path)
	out := make([]domain.QuoteMapping, 0, len(tab.rows))
	for _, row := range tab.rows {
		out = append(out, domain.QuoteMapping{
			QuoteName: tab.get(row, "Имя котировки"),
			LakeID:    parseIntPtr(tab.get(row, "ID в озере данных")),
			Currency:  tab.get(row, "Валюта котировки"),
		})
	}

	return out
}

func loadCurrencyRates(t *testing.T, path string) []domain.CurrencyRate {
	t.Helper()

	tab := readCSV(t, path)
	out := make([]domain.CurrencyRate, 0, len(tab.rows))
	for _, row := range tab.rows {
		value, err := strconv.ParseFloat(tab.get(row, "currency_value"), 64)
		require.NoError(t, err)

		out = append(out, domain.CurrencyRate{
			Currency:    strings.ToUpper(tab.get(row, "currency_name")),
			CalDay:      parseDate(t, tab.get(row, "calday")),
			VersionType: tab.get(row, "version_type"),
			Value:       value,
		})
	}

	return out
}

func loadMaterialGroups(t *testing.T, path string) []domain.MaterialGroup {
	t.Helper()

	minDate := time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)
	maxDate := time.Date(9999, time.December, 31, 0, 0, 0, 0, time.UTC)

	tab := readCSV(t, path)
	out := make([]domain.MaterialGroup, 0, len(tab.rows))
	for _, row := range tab.rows {
		materialID, err := strconv.ParseInt(tab.get(row, "code_nsi"), 10, 64)
		require.NoError(t, err)

		group := domain.MaterialGroup{
			GroupM:     tab.get(row, "hname"),
			MaterialID: materialID,
			ValidFrom:  minDate,
			ValidTo:    maxDate,
		}
		if raw := tab.get(row, "datuv"); raw != "" {
			group.ValidFrom = parseDate(t, raw)
		}
		if raw := tab.get(row, "datub"); raw != "" {
			group.ValidTo = parseDate(t, raw)
		}
		out = append(out, group)
	}

	return out
}
