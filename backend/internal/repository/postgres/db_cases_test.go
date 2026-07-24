package postgres

// Интеграционные тесты репозитория: реальный PostgreSQL (testcontainers через pgrx),
// каждому тесту — собственная БД с полным набором миграций (схема + seed).

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sibur-petrochem-price-service/internal/domain"
)

func TestIntegration_LoadSources(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("loads seeded sources with counts consistent to rows", func(t *testing.T) {
			tc := newIntegrationContext(t)

			tc.Given().When(
				ActNoop,
			).Then(
				AssertSeedSourcesLoaded,
			)

			sources, err := tc.SUT.LoadSources(context.Background())
			require.NoError(t, err)
			counts, err := tc.SUT.SourceCounts(context.Background())
			require.NoError(t, err)

			tc.State.Response.Sources = sources
			tc.State.Response.Counts = counts
		})

		t.Run("facets are distinct, sorted and bounded by seed ssp", func(t *testing.T) {
			tc := newIntegrationContext(t)
			tc.Given().When(ActNoop)

			facets, err := tc.SUT.SourceFacets(context.Background())
			require.NoError(t, err)

			require.NotEmpty(t, facets.Products)
			require.NotEmpty(t, facets.Clients)
			assert.True(t, sort.SliceIsSorted(facets.Products, func(i, j int) bool {
				return facets.Products[i].Name < facets.Products[j].Name
			}), "продукты отсортированы по имени")
			assert.True(t, sort.SliceIsSorted(facets.Clients, func(i, j int) bool {
				return facets.Clients[i].Name < facets.Clients[j].Name
			}), "клиенты отсортированы по имени")

			seenProduct := map[int64]struct{}{}
			for _, product := range facets.Products {
				_, dup := seenProduct[product.ID]
				require.False(t, dup, "продукт %d не должен дублироваться", product.ID)
				seenProduct[product.ID] = struct{}{}
			}

			assert.Regexp(t, `^\d{4}-\d{2}$`, facets.PeriodMin)
			assert.Regexp(t, `^\d{4}-\d{2}$`, facets.PeriodMax)
			assert.LessOrEqual(t, facets.PeriodMin, facets.PeriodMax)
		})
	})
}

func TestIntegration_ReplaceSsp(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("replaces seeded demand rows in single transaction", func(t *testing.T) {
			tc := newIntegrationContext(t)

			tc.Given(
				ArrangeDomainSspRows,
			).When(
				ActNoop,
			).Then(
				AssertSspReplaced,
			)

			require.NoError(t, tc.SUT.ReplaceSsp(context.Background(), tc.State.Given.DomainSsp))
			tc.State.Response.Sources, tc.State.Response.ReplaceErr = tc.SUT.LoadSources(context.Background())
			require.NoError(t, tc.State.Response.ReplaceErr)

			counts, err := tc.SUT.SourceCounts(context.Background())
			require.NoError(t, err)
			tc.State.Response.Counts = counts
		})

		t.Run("keeps previous rows when copy violates unique key", func(t *testing.T) {
			tc := newIntegrationContext(t)

			tc.Given(
				ArrangeDomainSspRows,
				ArrangeDuplicateSspRow,
			).When(
				ActNoop,
			).Then(
				AssertSspReplaceRolledBack,
			)

			seedCounts, err := tc.SUT.SourceCounts(context.Background())
			require.NoError(t, err)
			tc.State.Given.SeedCounts = seedCounts

			tc.State.Response.ReplaceErr = tc.SUT.ReplaceSsp(context.Background(), tc.State.Given.DomainSsp)

			counts, err := tc.SUT.SourceCounts(context.Background())
			require.NoError(t, err)
			tc.State.Response.Counts = counts
		})
	})
}

func TestIntegration_ReplaceFormulas(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("replaces seeded formula catalog in single transaction", func(t *testing.T) {
			tc := newIntegrationContext(t)

			tc.Given(
				ArrangeDomainFormulas,
			).When(
				ActNoop,
			).Then(
				AssertFormulasReplaced,
			)

			require.NoError(t, tc.SUT.ReplaceFormulas(context.Background(), tc.State.Given.DomainFormulas))
			tc.State.Response.Sources, tc.State.Response.ReplaceErr = tc.SUT.LoadSources(context.Background())
			require.NoError(t, tc.State.Response.ReplaceErr)

			counts, err := tc.SUT.SourceCounts(context.Background())
			require.NoError(t, err)
			tc.State.Response.Counts = counts
		})
	})
}

// ── arrange ──

func ArrangeDomainSspRows(t *testing.T, state State) State {
	t.Helper()

	const rowsCount = 3

	state.Given.DomainSsp = make([]domain.SspRow, 0, rowsCount)
	for i := range rowsCount {
		forecast := state.Faker.Float64(2, 10, 5000)
		state.Given.DomainSsp = append(state.Given.DomainSsp, domain.SspRow{
			RowID:        int64(1_000_000 + i),
			Period:       time.Date(2026, time.Month(i+1), 1, 0, 0, 0, 0, time.UTC),
			CustomerASV:  state.Faker.Company().Name(),
			Customer:     state.Faker.Company().Name(),
			MaterialID:   int64(state.Faker.IntBetween(100_000, 999_999)),
			MaterialName: state.Faker.Lorem().Sentence(3),
			Contract:     "Formula",
			Currency:     "RUB",
			Forecast:     &forecast,
			Country:      "", // NULL в БД: проверяем round-trip пустых optional-полей
			Region:       state.Faker.Lorem().Word(),
			Market:       state.Faker.Lorem().Word(),
			ClientID:     "CL-" + state.Faker.RandomStringWithLength(5),
			ClientName:   state.Faker.Company().Name(),
		})
	}

	return state
}

// ArrangeDuplicateSspRow — дубль по (row_id, period): copyfrom обязан упасть, транзакция — откатиться.
func ArrangeDuplicateSspRow(t *testing.T, state State) State {
	t.Helper()

	require.NotEmpty(t, state.Given.DomainSsp)
	state.Given.DomainSsp = append(state.Given.DomainSsp, state.Given.DomainSsp[0])

	return state
}

func ArrangeDomainFormulas(t *testing.T, state State) State {
	t.Helper()

	material := int64(state.Faker.IntBetween(100_000, 999_999))
	state.Given.DomainFormulas = []domain.Formula{
		{
			FormulaID:      "Z" + state.Faker.RandomStringWithLength(9),
			Text:           "( QUOTE - L ) * D",
			MaterialGroupM: "MT" + state.Faker.RandomStringWithLength(8),
			ValidFrom:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			ValidTo:        time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
			CreatedAt:      time.Date(2025, 12, 17, 10, 30, 0, 0, time.UTC),
			Inactive:       false,
			PriceType:      "A1",
			DocCurrency:    "RUB",
			ClientID:       "CL-" + state.Faker.RandomStringWithLength(5),
			ClientName:     state.Faker.Company().Name(),
		},
		{
			FormulaID:   "Z" + state.Faker.RandomStringWithLength(9),
			Text:        "QUOTE * K",
			ValidFrom:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			ValidTo:     time.Date(2027, 4, 30, 0, 0, 0, 0, time.UTC),
			CreatedAt:   time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
			Inactive:    true,
			DocCurrency: "CNY",
			MaterialID:  &material,
			ClientID:    "CL-" + state.Faker.RandomStringWithLength(5),
			ClientName:  state.Faker.Company().Name(),
		},
	}

	return state
}

// ── act (no-op: SUT-вызов в теле теста после цепочки) ──

func ActNoop(t *testing.T, deps Deps, state State) State {
	t.Helper()

	return state
}

// ── assert ──

func AssertSeedSourcesLoaded(t *testing.T, state State) {
	t.Helper()

	sources, counts := state.Response.Sources, state.Response.Counts

	require.NotEmpty(t, sources.Ssp)
	require.NotEmpty(t, sources.Formulas)
	require.NotEmpty(t, sources.Components)
	require.NotEmpty(t, sources.Quotes)

	assert.Equal(t, counts["ssp"], int64(len(sources.Ssp)))
	assert.Equal(t, counts["formulas"], int64(len(sources.Formulas)))
	assert.Equal(t, counts["formula_components"], int64(len(sources.Components)))
	assert.Equal(t, counts["term_types"], int64(len(sources.TermTypes)))
	assert.Equal(t, counts["quotes"], int64(len(sources.Quotes)))
	assert.Equal(t, counts["quote_mapping"], int64(len(sources.QuoteMapping)))
	assert.Equal(t, counts["currency_rates"], int64(len(sources.CurrencyRates)))
	assert.Equal(t, counts["material_groups"], int64(len(sources.MaterialGroups)))
}

func AssertSspReplaced(t *testing.T, state State) {
	t.Helper()

	require.Len(t, state.Response.Sources.Ssp, len(state.Given.DomainSsp))
	assert.Equal(t, int64(len(state.Given.DomainSsp)), state.Response.Counts["ssp"])

	byKey := make(map[string]domain.SspRow, len(state.Response.Sources.Ssp))
	for _, row := range state.Response.Sources.Ssp {
		byKey[row.DemandKey()] = row
	}

	for _, given := range state.Given.DomainSsp {
		got, ok := byKey[given.DemandKey()]
		require.True(t, ok, "row %s not found after replace", given.DemandKey())

		assert.Equal(t, given.MaterialID, got.MaterialID)
		assert.Equal(t, given.Contract, got.Contract)
		assert.Equal(t, given.ClientID, got.ClientID)
		assert.Empty(t, got.Country)
		assert.Equal(t, given.Region, got.Region)
		require.NotNil(t, got.Forecast)
		assert.InEpsilon(t, *given.Forecast, *got.Forecast, 1e-9)
	}
}

func AssertSspReplaceRolledBack(t *testing.T, state State) {
	t.Helper()

	require.Error(t, state.Response.ReplaceErr)
	assert.Equal(t, state.Given.SeedCounts["ssp"], state.Response.Counts["ssp"])
}

func AssertFormulasReplaced(t *testing.T, state State) {
	t.Helper()

	require.Len(t, state.Response.Sources.Formulas, len(state.Given.DomainFormulas))
	assert.Equal(t, int64(len(state.Given.DomainFormulas)), state.Response.Counts["formulas"])

	byID := make(map[string]domain.Formula, len(state.Response.Sources.Formulas))
	for _, formula := range state.Response.Sources.Formulas {
		byID[formula.FormulaID] = formula
	}

	withGroup, ok := byID[state.Given.DomainFormulas[0].FormulaID]
	require.True(t, ok)
	assert.Equal(t, state.Given.DomainFormulas[0].MaterialGroupM, withGroup.MaterialGroupM)
	assert.Nil(t, withGroup.MaterialID)
	assert.False(t, withGroup.Inactive)
	assert.Equal(t, state.Given.DomainFormulas[0].CreatedAt, withGroup.CreatedAt)

	withMaterial, ok := byID[state.Given.DomainFormulas[1].FormulaID]
	require.True(t, ok)
	assert.Empty(t, withMaterial.MaterialGroupM)
	require.NotNil(t, withMaterial.MaterialID)
	assert.Equal(t, *state.Given.DomainFormulas[1].MaterialID, *withMaterial.MaterialID)
	assert.True(t, withMaterial.Inactive)
}
