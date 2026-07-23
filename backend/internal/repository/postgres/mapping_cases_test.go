package postgres

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqlc_gen "sibur-petrochem-price-service/internal/generated/sqlc"
)

func TestMapping_Ssp(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("maps ssp rows to domain with demand key", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeSspRows,
			).When(
				ActMapSsp,
			).Then(
				AssertSspMapped,
			)

			tc.State.Response.Ssp = mapSsp(tc.State.Given.SspRows)
		})
	})
}

func TestMapping_Formulas(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("maps formulas with nullable group and material", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeFormulaRows,
			).When(
				ActMapFormulas,
			).Then(
				AssertFormulasMapped,
			)

			tc.State.Response.Formulas = mapFormulas(tc.State.Given.FormulaRows)
		})
	})
}

func TestMapping_Components(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("maps components with nullable value and quote name", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeComponentRows,
			).When(
				ActMapComponents,
			).Then(
				AssertComponentsMapped,
			)

			tc.State.Response.Components = mapComponents(tc.State.Given.ComponentRows)
		})
	})
}

func TestMapping_Quotes(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("maps quotes with optional tech load ts", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeQuoteRows,
			).When(
				ActMapQuotes,
			).Then(
				AssertQuotesMapped,
			)

			tc.State.Response.Quotes = mapQuotes(tc.State.Given.QuoteRows)
		})
	})
}

func TestMapping_Reference(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("maps quote mapping, rates and material groups", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeMappingRows,
				ArrangeRateRows,
				ArrangeGroupRows,
			).When(
				ActMapReference,
			).Then(
				AssertReferenceMapped,
			)

			tc.State.Response.Mapping = mapQuoteMapping(tc.State.Given.MappingRows)
			tc.State.Response.Rates = mapCurrencyRates(tc.State.Given.RateRows)
			tc.State.Response.Groups = mapMaterialGroups(tc.State.Given.GroupRows)
		})
	})
}

// ── arrange ──

func pgDate(t *testing.T, value string) pgtype.Date {
	t.Helper()

	parsed, err := time.Parse("2006-01-02", value)
	require.NoError(t, err)

	return pgtype.Date{Time: parsed, Valid: true}
}

func pgTs(t *testing.T, value string) pgtype.Timestamp {
	t.Helper()

	parsed, err := time.Parse("2006-01-02 15:04:05", value)
	require.NoError(t, err)

	return pgtype.Timestamp{Time: parsed, Valid: true}
}

func ArrangeSspRows(t *testing.T, state State) State {
	t.Helper()

	forecast := state.Faker.Float64(2, 10, 5000)
	state.Given.SspRows = []sqlc_gen.ListSspRow{
		{
			RowID:      int64(state.Faker.IntBetween(1_000_000, 2_000_000)),
			Period:     pgDate(t, "2026-06-01"),
			MtrNsiCode: int64(state.Faker.IntBetween(100_000, 999_999)),
			MtrNsiName: state.Faker.Lorem().Sentence(3),
			Contract:   "Formula",
			Currency:   "RUB",
			Forecast:   &forecast,
			ClientID:   "CL-" + state.Faker.RandomStringWithLength(5),
			ClientName: state.Faker.Company().Name(),
		},
		{
			RowID:      int64(state.Faker.IntBetween(2_000_001, 3_000_000)),
			Period:     pgDate(t, "2026-07-01"),
			MtrNsiCode: int64(state.Faker.IntBetween(100_000, 999_999)),
			MtrNsiName: state.Faker.Lorem().Sentence(3),
			Contract:   "SPOT",
			Currency:   "USD",
			Forecast:   nil,
			ClientID:   "CL-" + state.Faker.RandomStringWithLength(5),
			ClientName: state.Faker.Company().Name(),
		},
	}

	return state
}

func ArrangeFormulaRows(t *testing.T, state State) State {
	t.Helper()

	group := "MT" + state.Faker.RandomStringWithLength(8)
	material := int64(state.Faker.IntBetween(100_000, 999_999))
	state.Given.FormulaRows = []sqlc_gen.ListFormulasRow{
		{
			FormulaID:      "Z" + state.Faker.RandomStringWithLength(9),
			FormulaText:    "( QUOTE - L ) * D",
			MaterialGroupM: &group,
			ValidFrom:      pgDate(t, "2026-01-01"),
			ValidTo:        pgDate(t, "2026-12-31"),
			CreatedAt:      pgTs(t, "2025-12-17 10:30:00"),
			Inactive:       false,
			DocCurrency:    "RUB",
			MaterialID:     nil,
			ClientID:       "CL-" + state.Faker.RandomStringWithLength(5),
			ClientName:     state.Faker.Company().Name(),
		},
		{
			FormulaID:      "Z" + state.Faker.RandomStringWithLength(9),
			FormulaText:    "QUOTE * K",
			MaterialGroupM: nil,
			ValidFrom:      pgDate(t, "2026-02-01"),
			ValidTo:        pgDate(t, "2027-04-30"),
			CreatedAt:      pgTs(t, "2026-03-03 00:00:00"),
			Inactive:       true,
			DocCurrency:    "CNY",
			MaterialID:     &material,
			ClientID:       "CL-" + state.Faker.RandomStringWithLength(5),
			ClientName:     state.Faker.Company().Name(),
		},
	}

	return state
}

func ArrangeComponentRows(t *testing.T, state State) State {
	t.Helper()

	value := state.Faker.Float64(4, 0, 100)
	quoteName := state.Faker.Lorem().Word()
	currency := "USD"
	state.Given.ComponentRows = []sqlc_gen.ListFormulaComponentsRow{
		{
			FormulaID: "Z" + state.Faker.RandomStringWithLength(9),
			TermNo:    1,
			VarName:   "QUOTE",
			ValidFrom: pgTs(t, "2026-01-01 00:00:00"),
			ValidTo:   pgTs(t, "2026-12-31 00:00:00"),
			TypeCode:  "1",
			Value:     nil,
			Currency:  nil,
			QuoteName: &quoteName,
		},
		{
			FormulaID: "Z" + state.Faker.RandomStringWithLength(9),
			TermNo:    2,
			VarName:   "D",
			ValidFrom: pgTs(t, "2026-01-01 00:00:00"),
			ValidTo:   pgTs(t, "2026-12-31 00:00:00"),
			TypeCode:  "H",
			Value:     &value,
			Currency:  &currency,
			QuoteName: nil,
		},
	}

	return state
}

func ArrangeQuoteRows(t *testing.T, state State) State {
	t.Helper()

	loadTS := pgTs(t, "2026-06-10 12:00:00")
	state.Given.QuoteRows = []sqlc_gen.ListQuotesRow{
		{
			QuoteType:     "Факт",
			QuoteName:     state.Faker.Lorem().Word(),
			TechQuoteName: state.Faker.Lorem().Word(),
			QuoteCode:     int64(state.Faker.IntBetween(1_000, 9_999)),
			QuoteDate:     pgDate(t, "2026-05-01"),
			QuoteCurrency: "USD",
			QuoteVal:      state.Faker.Float64(2, 100, 2000),
			TechLoadTs:    loadTS,
		},
		{
			QuoteType:     "ППР",
			QuoteName:     state.Faker.Lorem().Word(),
			TechQuoteName: state.Faker.Lorem().Word(),
			QuoteCode:     int64(state.Faker.IntBetween(1_000, 9_999)),
			QuoteDate:     pgDate(t, "2026-06-01"),
			QuoteCurrency: "RUB",
			QuoteVal:      state.Faker.Float64(2, 100, 2000),
			TechLoadTs:    pgtype.Timestamp{},
		},
	}

	return state
}

func ArrangeMappingRows(t *testing.T, state State) State {
	t.Helper()

	lakeID := int64(state.Faker.IntBetween(1_000, 9_999))
	state.Given.MappingRows = []sqlc_gen.ListQuoteMappingRow{
		{QuoteName: state.Faker.Lorem().Word(), LakeID: &lakeID, QuoteCurrency: "USD"},
		{QuoteName: state.Faker.Lorem().Word(), LakeID: nil, QuoteCurrency: "EUR"},
	}

	return state
}

func ArrangeRateRows(t *testing.T, state State) State {
	t.Helper()

	state.Given.RateRows = []sqlc_gen.ListCurrencyRatesRow{
		{
			CurrencyName:  "USD",
			Calday:        pgDate(t, "2026-06-15"),
			VersionType:   "Факт",
			CurrencyValue: state.Faker.Float64(2, 50, 150),
		},
	}

	return state
}

func ArrangeGroupRows(t *testing.T, state State) State {
	t.Helper()

	state.Given.GroupRows = []sqlc_gen.ListMaterialGroupsRow{
		{
			GroupM:       "MT" + state.Faker.RandomStringWithLength(8),
			MaterialID:   int64(state.Faker.IntBetween(100_000, 999_999)),
			MaterialName: state.Faker.Lorem().Sentence(2),
			ValidFrom:    pgDate(t, "1900-01-01"),
			ValidTo:      pgDate(t, "9999-12-31"),
		},
	}

	return state
}

// ── act (no-op: SUT-вызов в теле теста, deps не нужны) ──

func ActMapSsp(t *testing.T, deps Deps, state State) State {
	t.Helper()

	return state
}

func ActMapFormulas(t *testing.T, deps Deps, state State) State {
	t.Helper()

	return state
}

func ActMapComponents(t *testing.T, deps Deps, state State) State {
	t.Helper()

	return state
}

func ActMapQuotes(t *testing.T, deps Deps, state State) State {
	t.Helper()

	return state
}

func ActMapReference(t *testing.T, deps Deps, state State) State {
	t.Helper()

	return state
}

// ── assert ──

func AssertSspMapped(t *testing.T, state State) {
	t.Helper()

	require.Len(t, state.Response.Ssp, len(state.Given.SspRows))

	first, second := state.Response.Ssp[0], state.Response.Ssp[1]
	givenFirst := state.Given.SspRows[0]

	assert.Equal(t, givenFirst.RowID, first.RowID)
	assert.Equal(t, givenFirst.MtrNsiCode, first.MaterialID)
	assert.Equal(t, givenFirst.Period.Time, first.Period)
	assert.InEpsilon(t, *givenFirst.Forecast, *first.Forecast, 1e-9)
	assert.True(t, first.IsFormula())
	assert.Contains(t, first.DemandKey(), "|2026-06-01")

	assert.False(t, second.IsFormula())
	assert.Nil(t, second.Forecast)
}

func AssertFormulasMapped(t *testing.T, state State) {
	t.Helper()

	require.Len(t, state.Response.Formulas, len(state.Given.FormulaRows))

	withGroup, withMaterial := state.Response.Formulas[0], state.Response.Formulas[1]

	assert.Equal(t, *state.Given.FormulaRows[0].MaterialGroupM, withGroup.MaterialGroupM)
	assert.Nil(t, withGroup.MaterialID)
	assert.False(t, withGroup.Inactive)
	assert.Equal(t, state.Given.FormulaRows[0].CreatedAt.Time, withGroup.CreatedAt)

	assert.Empty(t, withMaterial.MaterialGroupM)
	require.NotNil(t, withMaterial.MaterialID)
	assert.Equal(t, *state.Given.FormulaRows[1].MaterialID, *withMaterial.MaterialID)
	assert.True(t, withMaterial.Inactive)
}

func AssertComponentsMapped(t *testing.T, state State) {
	t.Helper()

	require.Len(t, state.Response.Components, len(state.Given.ComponentRows))

	quote, constant := state.Response.Components[0], state.Response.Components[1]

	assert.Equal(t, "1", quote.TypeCode)
	assert.Nil(t, quote.Value)
	assert.Equal(t, *state.Given.ComponentRows[0].QuoteName, quote.QuoteName)
	assert.Empty(t, quote.Currency)

	assert.Equal(t, "H", constant.TypeCode)
	require.NotNil(t, constant.Value)
	assert.InEpsilon(t, *state.Given.ComponentRows[1].Value, *constant.Value, 1e-9)
	assert.Equal(t, "USD", constant.Currency)
	assert.Empty(t, constant.QuoteName)
}

func AssertQuotesMapped(t *testing.T, state State) {
	t.Helper()

	require.Len(t, state.Response.Quotes, len(state.Given.QuoteRows))

	withTS, withoutTS := state.Response.Quotes[0], state.Response.Quotes[1]

	assert.Equal(t, state.Given.QuoteRows[0].QuoteCode, withTS.QuoteCode)
	require.NotNil(t, withTS.TechLoadTS)
	assert.Equal(t, state.Given.QuoteRows[0].TechLoadTs.Time, *withTS.TechLoadTS)

	assert.Nil(t, withoutTS.TechLoadTS)
}

func AssertReferenceMapped(t *testing.T, state State) {
	t.Helper()

	require.Len(t, state.Response.Mapping, 2)
	require.NotNil(t, state.Response.Mapping[0].LakeID)
	assert.Nil(t, state.Response.Mapping[1].LakeID)

	require.Len(t, state.Response.Rates, 1)
	assert.Equal(t, "USD", state.Response.Rates[0].Currency)
	assert.InEpsilon(t, state.Given.RateRows[0].CurrencyValue, state.Response.Rates[0].Value, 1e-9)

	require.Len(t, state.Response.Groups, 1)
	assert.Equal(t, state.Given.GroupRows[0].MaterialID, state.Response.Groups[0].MaterialID)
}
