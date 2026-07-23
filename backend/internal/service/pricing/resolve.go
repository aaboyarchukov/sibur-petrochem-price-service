package pricing

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"sibur-petrochem-price-service/internal/domain"
)

// Валюты расчёта: рубль — базовая, остальные конвертируются курсом.
const (
	currencyRUB = "RUB"
	currencyUSD = "USD"
	currencyEUR = "EUR"
	currencyCNY = "CNY"
)

// Типы версий котировок и курсов (каскад Факт → ОФ → ППР).
const (
	versionFact = "Факт"
	versionOF   = "ОФ"
	versionPPR  = "ППР"
)

// Константные типы термов (значение берётся из состава формулы).
var constantTypes = map[string]bool{"H": true, "A": true, "B": true, "C": true, "D": true, "E": true}

// currencyTermMap — технические ZF-идентификаторы валютных термов (как в эталоне).
var currencyTermMap = map[string][]string{
	"ZF0000000000000002": {"single", currencyUSD},
	"ZF0000000000000003": {"single", currencyEUR},
	"ZF0000000000000024": {"single", currencyCNY},
	"ZF0000000000000012": {"cross", currencyUSD, currencyCNY}, // CNY за 1 USD
}

func versionRank(versionType string) int {
	const (
		rankFact    = 0
		rankOF      = 1
		rankPPR     = 2
		rankUnknown = 99
	)
	switch versionType {
	case versionFact:
		return rankFact
	case versionOF:
		return rankOF
	case versionPPR:
		return rankPPR
	default:
		return rankUnknown
	}
}

// rateInfo — метаданные выбранного курса/котировки для расшифровки.
type rateInfo struct {
	Date        time.Time
	VersionType string
	GapDays     int
}

const hoursPerDay = 24

func gapDays(a, b time.Time) int {
	return int(math.Abs(a.Sub(b).Hours()) / hoursPerDay)
}

// versionedItem — кандидат выбора «ближайшей версии» (каскад эталона):
// min |date−period| → version_rank (Факт→ОФ→ППР) → прошлое раньше будущего → max tech_load_ts.
type versionedItem struct {
	index   int
	gap     int
	rank    int
	future  int
	loadTS  time.Time
	hasLoad bool
}

func chooseNearest(items []versionedItem) int {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.gap != b.gap {
			return a.gap < b.gap
		}
		if a.rank != b.rank {
			return a.rank < b.rank
		}
		if a.future != b.future {
			return a.future < b.future
		}
		// tech_load_ts по убыванию; отсутствующий — в конец (как NaT в pandas)
		if a.hasLoad != b.hasLoad {
			return a.hasLoad
		}

		return a.loadTS.After(b.loadTS)
	})

	return items[0].index
}

func resolveQuote(component domain.FormulaComponent, period time.Time, idx *indexes) (float64, domain.ComponentValue, error) {
	name := strings.TrimSpace(component.QuoteName)

	codes := idx.quoteNameToCodes[name]
	if len(codes) == 0 {
		return 0, domain.ComponentValue{}, fmt.Errorf("%w: Нет маппинга котировки: %s", domain.ErrComponent, name)
	}

	var pool []domain.Quote
	for _, code := range codes {
		pool = append(pool, idx.quotesByCode[code]...)
	}
	if len(pool) == 0 {
		return 0, domain.ComponentValue{}, fmt.Errorf(
			"%w: Нет значений котировки: %s; quote_code=%v", domain.ErrComponent, name, codes,
		)
	}

	items := make([]versionedItem, 0, len(pool))
	for i, quote := range pool {
		item := versionedItem{
			index: i,
			gap:   gapDays(quote.QuoteDate, period),
			rank:  versionRank(quote.QuoteType),
		}
		if quote.QuoteDate.After(period) {
			item.future = 1
		}
		if quote.TechLoadTS != nil {
			item.loadTS = *quote.TechLoadTS
			item.hasLoad = true
		}
		items = append(items, item)
	}

	chosen := pool[chooseNearest(items)]
	sourceDate := chosen.QuoteDate
	gap := gapDays(chosen.QuoteDate, period)

	return chosen.Value, domain.ComponentValue{
		Source:      "quotes.csv",
		QuoteName:   name,
		QuoteCode:   &chosen.QuoteCode,
		SourceDate:  &sourceDate,
		VersionType: chosen.QuoteType,
		DateGapDays: &gap,
	}, nil
}

func resolveRate(currency string, period time.Time, idx *indexes) (float64, rateInfo, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == currencyRUB {
		return 1, rateInfo{Date: period, VersionType: "identity", GapDays: 0}, nil
	}

	pool := idx.ratesByCurrency[currency]
	if len(pool) == 0 {
		return 0, rateInfo{}, fmt.Errorf("%w: Нет курса валюты: %s", domain.ErrComponent, currency)
	}

	items := make([]versionedItem, 0, len(pool))
	for i, rate := range pool {
		item := versionedItem{
			index: i,
			gap:   gapDays(rate.CalDay, period),
			rank:  versionRank(rate.VersionType),
		}
		if rate.CalDay.After(period) {
			item.future = 1
		}
		items = append(items, item)
	}

	chosen := pool[chooseNearest(items)]

	return chosen.Value, rateInfo{
		Date:        chosen.CalDay,
		VersionType: chosen.VersionType,
		GapDays:     gapDays(chosen.CalDay, period),
	}, nil
}

func resolveCurrencyComponent(component domain.FormulaComponent, period time.Time, idx *indexes) (float64, rateInfo, error) {
	technicalID := strings.TrimSpace(component.QuoteName)
	variable := strings.ToUpper(strings.TrimSpace(component.VarName))
	explicit := strings.ToUpper(strings.TrimSpace(component.Currency))

	if explicit != "" && (explicit == currencyRUB || len(idx.ratesByCurrency[explicit]) > 0) {
		return resolveRate(explicit, period, idx)
	}

	if mapping, ok := currencyTermMap[technicalID]; ok {
		return resolveTechnicalTerm(mapping, period, idx)
	}

	if variable == currencyUSD || variable == currencyEUR || variable == currencyCNY || variable == currencyRUB {
		return resolveRate(variable, period, idx)
	}

	return 0, rateInfo{}, fmt.Errorf(
		"%w: Неизвестный валютный терм: var=%s; technical_id=%s", domain.ErrComponent, component.VarName, technicalID,
	)
}

// resolveTechnicalTerm — курс по ZF-идентификатору: одиночный либо кросс (num/den).
func resolveTechnicalTerm(mapping []string, period time.Time, idx *indexes) (float64, rateInfo, error) {
	if mapping[0] == "single" {
		return resolveRate(mapping[1], period, idx)
	}

	numerator, numeratorInfo, err := resolveRate(mapping[1], period, idx)
	if err != nil {
		return 0, rateInfo{}, err
	}

	denominator, _, err := resolveRate(mapping[2], period, idx)
	if err != nil {
		return 0, rateInfo{}, err
	}

	return numerator / denominator, numeratorInfo, nil
}

// resolveComponents — активные на период компоненты формулы со значениями.
// Для дублей var_name берётся компонент с самым поздним valid_from.
func resolveComponents(formulaID string, period time.Time, idx *indexes) (values map[string]float64, details []domain.ComponentValue, errorsFound []string) {
	frame, ok := idx.componentsByFormula[formulaID]
	if !ok {
		return map[string]float64{}, nil, []string{"Нет компонентов для формулы " + formulaID}
	}

	active := make([]componentWindow, 0, len(frame))
	for _, component := range frame {
		if !component.ValidFrom.After(period) && !component.effectiveValidTo.Before(period) {
			active = append(active, component)
		}
	}

	sort.SliceStable(active, func(i, j int) bool {
		if active[i].VarName != active[j].VarName {
			return active[i].VarName < active[j].VarName
		}
		if !active[i].ValidFrom.Equal(active[j].ValidFrom) {
			return active[i].ValidFrom.After(active[j].ValidFrom)
		}

		return active[i].TermNo > active[j].TermNo
	})

	values = make(map[string]float64, len(active))
	details = make([]domain.ComponentValue, 0, len(active))

	seen := make(map[string]bool, len(active))
	for _, component := range active {
		if seen[component.VarName] {
			continue
		}
		seen[component.VarName] = true

		detail := resolveComponent(component, period, idx)
		if detail.Error != "" {
			errorsFound = append(errorsFound, component.VarName+": "+detail.Error)
		} else if detail.Value != nil {
			values[component.VarName] = *detail.Value
		}
		details = append(details, detail)
	}

	return values, details, errorsFound
}

func resolveComponent(component componentWindow, period time.Time, idx *indexes) domain.ComponentValue {
	detail := domain.ComponentValue{
		VarName:   component.VarName,
		TypeCode:  component.TypeCode,
		TypeLabel: idx.typeLabels[component.TypeCode],
	}

	switch {
	case constantTypes[component.TypeCode]:
		detail.Source = "formula_components.csv"
		if component.Value == nil {
			zero := 0.0
			detail.Value = &zero
			detail.Warning = "Пустой фиксированный терм принят равным 0"

			return detail
		}
		detail.Value = component.Value

		return detail

	case component.TypeCode == "1":
		value, meta, err := resolveQuote(component.FormulaComponent, period, idx)
		if err != nil {
			detail.Error = componentErrorText(err)

			return detail
		}
		meta.VarName, meta.TypeCode, meta.TypeLabel = detail.VarName, detail.TypeCode, detail.TypeLabel
		meta.Value = &value

		return meta

	case component.TypeCode == "5":
		value, info, err := resolveCurrencyComponent(component.FormulaComponent, period, idx)
		if err != nil {
			detail.Error = componentErrorText(err)

			return detail
		}
		sourceDate := info.Date
		gap := info.GapDays
		detail.Source = "currency_rates.csv"
		detail.Value = &value
		detail.SourceDate = &sourceDate
		detail.VersionType = info.VersionType
		detail.DateGapDays = &gap

		return detail

	case component.TypeCode == "7":
		detail.Error = "Тип 7 требует отдельного источника Pricing/SAP, которого нет в наборе"

		return detail

	default:
		detail.Error = "Неподдерживаемый тип компонента: " + component.TypeCode

		return detail
	}
}

// componentErrorText — текст ошибки без префикса sentinel-ошибки.
func componentErrorText(err error) string {
	text := err.Error()

	return strings.TrimPrefix(text, domain.ErrComponent.Error()+": ")
}
