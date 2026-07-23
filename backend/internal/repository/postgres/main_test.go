package postgres

import (
	"testing"

	"github.com/godepo/groat"
	"github.com/jaswdr/faker/v2"

	"sibur-petrochem-price-service/internal/domain"

	sqlc_gen "sibur-petrochem-price-service/internal/generated/sqlc"
)

type (
	Givens struct {
		SspRows       []sqlc_gen.ListSspRow
		FormulaRows   []sqlc_gen.ListFormulasRow
		ComponentRows []sqlc_gen.ListFormulaComponentsRow
		QuoteRows     []sqlc_gen.ListQuotesRow
		MappingRows   []sqlc_gen.ListQuoteMappingRow
		RateRows      []sqlc_gen.ListCurrencyRatesRow
		GroupRows     []sqlc_gen.ListMaterialGroupsRow
	}

	Responses struct {
		Ssp        []domain.SspRow
		Formulas   []domain.Formula
		Components []domain.FormulaComponent
		Quotes     []domain.Quote
		Mapping    []domain.QuoteMapping
		Rates      []domain.CurrencyRate
		Groups     []domain.MaterialGroup
	}

	Deps struct{}

	State struct {
		Given    Givens
		Faker    faker.Faker
		Response Responses
	}
)

func newTestContext(t *testing.T) *groat.Case[Deps, State, *Repository] {
	t.Helper()

	tc := groat.New[Deps, State, *Repository](t, func(t *testing.T, deps Deps) *Repository {
		t.Helper()

		return New(nil)
	})

	tc.Go()

	tc.State.Faker = faker.New()

	return tc
}
