package pricing

import (
	"time"

	"sibur-petrochem-price-service/internal/domain"
)

// componentWindow — компонент с растянутым до горизонта ssp сроком действия
// (как effective_component_valid_to в эталоне).
type componentWindow struct {
	domain.FormulaComponent
	effectiveValidTo time.Time
}

// indexes — предрассчитанные индексы источников для быстрого разрешения компонентов.
type indexes struct {
	quoteNameToCodes    map[string][]int64
	quotesByCode        map[int64][]domain.Quote
	ratesByCurrency     map[string][]domain.CurrencyRate
	componentsByFormula map[string][]componentWindow
	typeLabels          map[string]string
}

func buildIndexes(src domain.Sources, horizon time.Time) *indexes {
	idx := &indexes{
		quoteNameToCodes:    make(map[string][]int64, len(src.QuoteMapping)),
		quotesByCode:        make(map[int64][]domain.Quote, len(src.Quotes)),
		ratesByCurrency:     make(map[string][]domain.CurrencyRate, len(src.CurrencyRates)),
		componentsByFormula: make(map[string][]componentWindow, len(src.Components)),
		typeLabels:          make(map[string]string, len(src.TermTypes)),
	}

	seenCodes := make(map[string]map[int64]bool, len(src.QuoteMapping))
	for _, mapping := range src.QuoteMapping {
		if mapping.LakeID == nil {
			continue
		}
		if seenCodes[mapping.QuoteName] == nil {
			seenCodes[mapping.QuoteName] = map[int64]bool{}
		}
		if seenCodes[mapping.QuoteName][*mapping.LakeID] {
			continue
		}
		seenCodes[mapping.QuoteName][*mapping.LakeID] = true
		idx.quoteNameToCodes[mapping.QuoteName] = append(idx.quoteNameToCodes[mapping.QuoteName], *mapping.LakeID)
	}

	for _, quote := range src.Quotes {
		idx.quotesByCode[quote.QuoteCode] = append(idx.quotesByCode[quote.QuoteCode], quote)
	}

	for _, rate := range src.CurrencyRates {
		idx.ratesByCurrency[rate.Currency] = append(idx.ratesByCurrency[rate.Currency], rate)
	}

	for _, component := range src.Components {
		effective := component.ValidTo
		if effective.Before(horizon) {
			effective = horizon
		}
		idx.componentsByFormula[component.FormulaID] = append(
			idx.componentsByFormula[component.FormulaID],
			componentWindow{FormulaComponent: component, effectiveValidTo: effective},
		)
	}

	for _, termType := range src.TermTypes {
		idx.typeLabels[termType.TypeCode] = termType.TypeLabel
	}

	return idx
}
