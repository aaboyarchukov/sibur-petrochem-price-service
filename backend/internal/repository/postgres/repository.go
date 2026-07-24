// Package postgres — чтение источников расчёта из PostgreSQL (sqlc).
package postgres

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgtype"

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

// SourceFacets — уникальные продукты, клиенты и границы горизонта из ssp
// для пикеров экрана параметров; продукты/клиенты отсортированы по имени.
func (r *Repository) SourceFacets(ctx context.Context) (domain.SourceFacets, error) {
	products, err := r.queries.ListSspProducts(ctx)
	if err != nil {
		return domain.SourceFacets{}, fmt.Errorf("list ssp products: %w", err)
	}

	clients, err := r.queries.ListSspClients(ctx)
	if err != nil {
		return domain.SourceFacets{}, fmt.Errorf("list ssp clients: %w", err)
	}

	bounds, err := r.queries.SspPeriodBounds(ctx)
	if err != nil {
		return domain.SourceFacets{}, fmt.Errorf("ssp period bounds: %w", err)
	}

	facets := domain.SourceFacets{
		Products:  make([]domain.ProductFacet, 0, len(products)),
		Clients:   make([]domain.ClientFacet, 0, len(clients)),
		PeriodMin: monthOrEmpty(bounds.PeriodMin),
		PeriodMax: monthOrEmpty(bounds.PeriodMax),
	}
	for _, product := range products {
		facets.Products = append(facets.Products, domain.ProductFacet{ID: product.MtrNsiCode, Name: product.MtrNsiName})
	}
	for _, client := range clients {
		facets.Clients = append(facets.Clients, domain.ClientFacet{ID: client.ClientID, Name: client.ClientName})
	}

	sort.SliceStable(facets.Products, func(i, j int) bool { return facets.Products[i].Name < facets.Products[j].Name })
	sort.SliceStable(facets.Clients, func(i, j int) bool { return facets.Clients[i].Name < facets.Clients[j].Name })

	return facets, nil
}

// monthOrEmpty — граница горизонта в формате YYYY-MM; пусто при отсутствии данных.
func monthOrEmpty(date pgtype.Date) string {
	if !date.Valid {
		return ""
	}

	return date.Time.Format("2006-01")
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
