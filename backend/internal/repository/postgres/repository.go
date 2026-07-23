// Package postgres — чтение источников расчёта из PostgreSQL (sqlc).
package postgres

import (
	"context"
	"fmt"

	"sibur-petrochem-price-service/internal/domain"

	sqlc_gen "sibur-petrochem-price-service/internal/generated/sqlc"
)

// Repository — доступ к источникам данных расчёта.
type Repository struct {
	db      sqlc_gen.DBTX
	queries *sqlc_gen.Queries
}

func New(db sqlc_gen.DBTX) *Repository {
	if db == nil {
		return &Repository{}
	}

	return &Repository{db: db, queries: sqlc_gen.New(db)}
}

// LoadSources — загрузка всех источников в память для прогона расчёта.
func (r *Repository) LoadSources(ctx context.Context) (domain.Sources, error) {
	ssp, err := r.queries.ListSsp(ctx)
	if err != nil {
		return domain.Sources{}, fmt.Errorf("list ssp: %w", err)
	}

	formulas, err := r.queries.ListFormulas(ctx)
	if err != nil {
		return domain.Sources{}, fmt.Errorf("list formulas: %w", err)
	}

	components, err := r.queries.ListFormulaComponents(ctx)
	if err != nil {
		return domain.Sources{}, fmt.Errorf("list formula components: %w", err)
	}

	termTypes, err := r.queries.ListTermTypes(ctx)
	if err != nil {
		return domain.Sources{}, fmt.Errorf("list term types: %w", err)
	}

	quotes, err := r.queries.ListQuotes(ctx)
	if err != nil {
		return domain.Sources{}, fmt.Errorf("list quotes: %w", err)
	}

	mapping, err := r.queries.ListQuoteMapping(ctx)
	if err != nil {
		return domain.Sources{}, fmt.Errorf("list quote mapping: %w", err)
	}

	rates, err := r.queries.ListCurrencyRates(ctx)
	if err != nil {
		return domain.Sources{}, fmt.Errorf("list currency rates: %w", err)
	}

	groups, err := r.queries.ListMaterialGroups(ctx)
	if err != nil {
		return domain.Sources{}, fmt.Errorf("list material groups: %w", err)
	}

	return domain.Sources{
		Ssp:            mapSsp(ssp),
		Formulas:       mapFormulas(formulas),
		Components:     mapComponents(components),
		TermTypes:      mapTermTypes(termTypes),
		Quotes:         mapQuotes(quotes),
		QuoteMapping:   mapQuoteMapping(mapping),
		CurrencyRates:  mapCurrencyRates(rates),
		MaterialGroups: mapMaterialGroups(groups),
	}, nil
}

// SourceCounts — количество строк в каждом источнике (для GET /sources).
func (r *Repository) SourceCounts(ctx context.Context) (map[string]int64, error) {
	counts, err := r.queries.CountSourceRows(ctx)
	if err != nil {
		return nil, fmt.Errorf("count source rows: %w", err)
	}

	return map[string]int64{
		"ssp":                counts.Ssp,
		"formulas":           counts.Formulas,
		"formula_components": counts.FormulaComponents,
		"term_types":         counts.TermTypes,
		"quotes":             counts.Quotes,
		"quote_mapping":      counts.QuoteMapping,
		"currency_rates":     counts.CurrencyRates,
		"material_groups":    counts.MaterialGroups,
	}, nil
}
