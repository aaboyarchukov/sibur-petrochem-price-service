package pricing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sibur-petrochem-price-service/internal/domain"
)

func TestResolveComponents(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("takes latest valid_from on duplicate var and stretches valid_to to horizon", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDuplicateVarComponents,
			).When(
				ActResolveComponents,
			).Then(
				AssertNoResolveErrors,
				AssertDuplicateResolved,
			)
		})

		t.Run("treats empty constant as zero with warning", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeEmptyConstant,
			).When(
				ActResolveComponents,
			).Then(
				AssertNoResolveErrors,
				AssertEmptyConstantZero,
			)
		})
	})

	t.Run("should be able to not be able", func(t *testing.T) {
		t.Run("reports error for price list type 7", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangePriceListComponent,
			).When(
				ActResolveComponents,
			).Then(
				AssertResolveErrorContains,
			)
		})
	})
}

func TestResolveQuote(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("prefers fact version on equal date gap", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeQuoteVersionCascade,
			).When(
				ActResolveQuote,
			).Then(
				AssertNoError,
				AssertQuoteChoice,
			)
		})

		t.Run("collects quotes from multiple mapped codes and prefers past on tie", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeQuoteMultiCode,
			).When(
				ActResolveQuote,
			).Then(
				AssertNoError,
				AssertQuoteChoice,
			)
		})

		t.Run("breaks full tie by latest tech_load_ts", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeQuoteLoadTsTie,
			).When(
				ActResolveQuote,
			).Then(
				AssertNoError,
				AssertQuoteChoice,
			)
		})
	})

	t.Run("should be able to not be able", func(t *testing.T) {
		t.Run("fails when mapping is missing", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeQuoteWithoutMapping,
			).When(
				ActResolveQuote,
			).Then(
				AssertErrorContains,
			)
		})

		t.Run("fails when mapped codes have no quotes", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeQuoteWithoutValues,
			).When(
				ActResolveQuote,
			).Then(
				AssertErrorContains,
			)
		})
	})
}

func TestResolveRate(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("returns identity for RUB", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeRubRate,
			).When(
				ActResolveRate,
			).Then(
				AssertNoError,
				AssertRateValue,
			)
		})

		t.Run("chooses nearest calday with version cascade", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeUsdRateCascade,
			).When(
				ActResolveRate,
			).Then(
				AssertNoError,
				AssertRateValue,
			)
		})
	})

	t.Run("should be able to not be able", func(t *testing.T) {
		t.Run("fails when currency has no rates", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeMissingRate,
			).When(
				ActResolveRate,
			).Then(
				AssertErrorContains,
			)
		})
	})
}

func TestResolveCurrencyComponent(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("resolves cross rate via ZF technical id", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeCrossRateComponent,
			).When(
				ActResolveCurrencyComponent,
			).Then(
				AssertNoError,
				AssertRateValue,
			)
		})

		t.Run("prefers explicit currency over technical id", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeExplicitCurrencyComponent,
			).When(
				ActResolveCurrencyComponent,
			).Then(
				AssertNoError,
				AssertRateValue,
			)
		})
	})

	t.Run("should be able to not be able", func(t *testing.T) {
		t.Run("fails on unknown currency term", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeUnknownCurrencyTerm,
			).When(
				ActResolveCurrencyComponent,
			).Then(
				AssertErrorContains,
			)
		})
	})
}

// ── arrange: components ──

func ArrangeDuplicateVarComponents(t *testing.T, state State) State {
	t.Helper()

	state.Given.FormulaID = "Z" + state.Faker.RandomStringWithLength(9)
	state.Given.Period = date(t, "2026-06-01")
	state.Given.Horizon = date(t, "2027-03-01")
	oldValue := 100.0
	newValue := 200.0
	state.Given.Sources = domain.Sources{
		Components: []domain.FormulaComponent{
			{
				FormulaID: state.Given.FormulaID,
				TermNo:    1,
				VarName:   "D",
				ValidFrom: date(t, "2026-01-01"),
				// valid_to в прошлом — должен растянуться до горизонта и попасть в расчёт
				ValidTo:  date(t, "2026-02-01"),
				TypeCode: "H",
				Value:    &oldValue,
			},
			{
				FormulaID: state.Given.FormulaID,
				TermNo:    2,
				VarName:   "D",
				ValidFrom: date(t, "2026-05-01"),
				ValidTo:   date(t, "2026-12-31"),
				TypeCode:  "H",
				Value:     &newValue,
			},
		},
	}
	state.Expect.Value = newValue

	return state
}

func ArrangeEmptyConstant(t *testing.T, state State) State {
	t.Helper()

	state.Given.FormulaID = "Z" + state.Faker.RandomStringWithLength(9)
	state.Given.Period = date(t, "2026-06-01")
	state.Given.Horizon = date(t, "2027-03-01")
	state.Given.Sources = domain.Sources{
		Components: []domain.FormulaComponent{
			{
				FormulaID: state.Given.FormulaID,
				TermNo:    1,
				VarName:   "L",
				ValidFrom: date(t, "2026-01-01"),
				ValidTo:   date(t, "2026-12-31"),
				TypeCode:  "B",
				Value:     nil,
			},
		},
	}
	state.Expect.Value = 0
	state.Expect.Warning = "Пустой фиксированный терм"

	return state
}

func ArrangePriceListComponent(t *testing.T, state State) State {
	t.Helper()

	state.Given.FormulaID = "Z" + state.Faker.RandomStringWithLength(9)
	state.Given.Period = date(t, "2026-06-01")
	state.Given.Horizon = date(t, "2027-03-01")
	state.Given.Sources = domain.Sources{
		Components: []domain.FormulaComponent{
			{
				FormulaID: state.Given.FormulaID,
				TermNo:    1,
				VarName:   "CONDROW",
				ValidFrom: date(t, "2026-01-01"),
				ValidTo:   date(t, "2026-12-31"),
				TypeCode:  "7",
			},
		},
	}
	state.Expect.ErrSubstr = "Тип 7"

	return state
}

// ── arrange: quotes ──

func quoteComponent(state State, quoteName string) domain.FormulaComponent {
	return domain.FormulaComponent{
		FormulaID: "Z" + state.Faker.RandomStringWithLength(9),
		VarName:   "QUOTE",
		TypeCode:  "1",
		QuoteName: quoteName,
	}
}

func ArrangeQuoteVersionCascade(t *testing.T, state State) State {
	t.Helper()

	const code = int64(9001)
	state.Given.Period = date(t, "2026-06-01")
	state.Given.Component = quoteComponent(state, "PP CPT Moscow")
	state.Given.Sources = domain.Sources{
		QuoteMapping: []domain.QuoteMapping{
			{QuoteName: "PP CPT Moscow", LakeID: ptr(code), Currency: "RUB"},
		},
		Quotes: []domain.Quote{
			// одинаковый разрыв в днях — Факт (rank 0) должен победить ППР
			{QuoteCode: code, QuoteDate: date(t, "2026-05-01"), QuoteType: "ППР", Value: 500},
			{QuoteCode: code, QuoteDate: date(t, "2026-05-01"), QuoteType: "Факт", Value: 700},
		},
	}
	state.Expect.Value = 700
	state.Expect.VersionType = "Факт"
	state.Expect.QuoteCode = code

	return state
}

func ArrangeQuoteMultiCode(t *testing.T, state State) State {
	t.Helper()

	first, second := int64(9001), int64(9002)
	state.Given.Period = date(t, "2026-06-15")
	state.Given.Component = quoteComponent(state, "PE CFR Turkey")
	state.Given.Sources = domain.Sources{
		QuoteMapping: []domain.QuoteMapping{
			{QuoteName: "PE CFR Turkey", LakeID: ptr(first), Currency: "USD"},
			{QuoteName: "PE CFR Turkey", LakeID: ptr(second), Currency: "USD"},
		},
		Quotes: []domain.Quote{
			// равный gap (5 дней) и rank — прошлое (10.06) раньше будущего (20.06)
			{QuoteCode: first, QuoteDate: date(t, "2026-06-20"), QuoteType: "Факт", Value: 111},
			{QuoteCode: second, QuoteDate: date(t, "2026-06-10"), QuoteType: "Факт", Value: 222},
		},
	}
	state.Expect.Value = 222
	state.Expect.VersionType = "Факт"
	state.Expect.QuoteCode = second

	return state
}

func ArrangeQuoteLoadTsTie(t *testing.T, state State) State {
	t.Helper()

	const code = int64(9003)
	state.Given.Period = date(t, "2026-06-01")
	state.Given.Component = quoteComponent(state, "Styrene FOB")
	older := date(t, "2026-05-20")
	newer := date(t, "2026-05-25")
	state.Given.Sources = domain.Sources{
		QuoteMapping: []domain.QuoteMapping{
			{QuoteName: "Styrene FOB", LakeID: ptr(code), Currency: "USD"},
		},
		Quotes: []domain.Quote{
			// полный tie по дате/версии — берётся максимальный tech_load_ts
			{QuoteCode: code, QuoteDate: date(t, "2026-05-30"), QuoteType: "ОФ", Value: 10, TechLoadTS: &older},
			{QuoteCode: code, QuoteDate: date(t, "2026-05-30"), QuoteType: "ОФ", Value: 20, TechLoadTS: &newer},
		},
	}
	state.Expect.Value = 20
	state.Expect.VersionType = "ОФ"
	state.Expect.QuoteCode = code

	return state
}

func ArrangeQuoteWithoutMapping(t *testing.T, state State) State {
	t.Helper()

	state.Given.Period = date(t, "2026-06-01")
	state.Given.Component = quoteComponent(state, "UNMAPPED QUOTE")
	state.Given.Sources = domain.Sources{}
	state.Expect.ErrSubstr = "Нет маппинга котировки"

	return state
}

func ArrangeQuoteWithoutValues(t *testing.T, state State) State {
	t.Helper()

	state.Given.Period = date(t, "2026-06-01")
	state.Given.Component = quoteComponent(state, "MAPPED EMPTY")
	state.Given.Sources = domain.Sources{
		QuoteMapping: []domain.QuoteMapping{
			{QuoteName: "MAPPED EMPTY", LakeID: ptr(int64(777)), Currency: "USD"},
		},
	}
	state.Expect.ErrSubstr = "Нет значений котировки"

	return state
}

// ── arrange: rates ──

func ArrangeRubRate(t *testing.T, state State) State {
	t.Helper()

	state.Given.Period = date(t, "2026-06-01")
	state.Given.Currency = "RUB"
	state.Given.Sources = domain.Sources{}
	state.Expect.Value = 1

	return state
}

func ArrangeUsdRateCascade(t *testing.T, state State) State {
	t.Helper()

	state.Given.Period = date(t, "2026-06-15")
	state.Given.Currency = "USD"
	state.Given.Sources = domain.Sources{
		CurrencyRates: []domain.CurrencyRate{
			// дальше по дате — не берётся, несмотря на Факт
			{Currency: "USD", CalDay: date(t, "2026-06-01"), VersionType: "Факт", Value: 88.0},
			// ближайший день: ОФ побеждает ППР
			{Currency: "USD", CalDay: date(t, "2026-06-14"), VersionType: "ППР", Value: 90.5},
			{Currency: "USD", CalDay: date(t, "2026-06-14"), VersionType: "ОФ", Value: 89.42},
		},
	}
	state.Expect.Value = 89.42

	return state
}

func ArrangeMissingRate(t *testing.T, state State) State {
	t.Helper()

	state.Given.Period = date(t, "2026-06-01")
	state.Given.Currency = "JPY"
	state.Given.Sources = domain.Sources{}
	state.Expect.ErrSubstr = "Нет курса валюты"

	return state
}

// ── arrange: currency components ──

func ArrangeCrossRateComponent(t *testing.T, state State) State {
	t.Helper()

	state.Given.Period = date(t, "2026-06-01")
	state.Given.Component = domain.FormulaComponent{
		VarName:   "CUR_$_¥_PBC",
		TypeCode:  "5",
		QuoteName: "ZF0000000000000012", // кросс USD/CNY
	}
	state.Given.Sources = domain.Sources{
		CurrencyRates: []domain.CurrencyRate{
			{Currency: "USD", CalDay: date(t, "2026-06-01"), VersionType: "Факт", Value: 90.0},
			{Currency: "CNY", CalDay: date(t, "2026-06-01"), VersionType: "Факт", Value: 12.5},
		},
	}
	// CNY за 1 USD = 90 / 12.5
	state.Expect.Value = 7.2

	return state
}

func ArrangeExplicitCurrencyComponent(t *testing.T, state State) State {
	t.Helper()

	state.Given.Period = date(t, "2026-06-01")
	state.Given.Component = domain.FormulaComponent{
		VarName:   "EUR_RATE",
		TypeCode:  "5",
		Currency:  "EUR",
		QuoteName: "ZF0000000000000002", // явная валюта приоритетнее технического ID (USD)
	}
	state.Given.Sources = domain.Sources{
		CurrencyRates: []domain.CurrencyRate{
			{Currency: "EUR", CalDay: date(t, "2026-06-01"), VersionType: "Факт", Value: 98.7},
			{Currency: "USD", CalDay: date(t, "2026-06-01"), VersionType: "Факт", Value: 90.0},
		},
	}
	state.Expect.Value = 98.7

	return state
}

func ArrangeUnknownCurrencyTerm(t *testing.T, state State) State {
	t.Helper()

	state.Given.Period = date(t, "2026-06-01")
	state.Given.Component = domain.FormulaComponent{
		VarName:   "MYSTERY",
		TypeCode:  "5",
		QuoteName: "ZF9999999999999999",
	}
	state.Given.Sources = domain.Sources{}
	state.Expect.ErrSubstr = "Неизвестный валютный терм"

	return state
}

// ── act ──

func ActResolveComponents(t *testing.T, deps Deps, state State) State {
	t.Helper()

	idx := buildIndexes(state.Given.Sources, state.Given.Horizon)
	state.Response.Values, state.Response.Details, state.Response.Errors = resolveComponents(
		state.Given.FormulaID, state.Given.Period, idx,
	)

	return state
}

func ActResolveQuote(t *testing.T, deps Deps, state State) State {
	t.Helper()

	idx := buildIndexes(state.Given.Sources, state.Given.Horizon)
	value, meta, err := resolveQuote(state.Given.Component, state.Given.Period, idx)
	state.Response.Value, state.Response.Meta, state.Response.Err = value, meta, err

	return state
}

func ActResolveRate(t *testing.T, deps Deps, state State) State {
	t.Helper()

	idx := buildIndexes(state.Given.Sources, state.Given.Horizon)
	value, _, err := resolveRate(state.Given.Currency, state.Given.Period, idx)
	state.Response.Value, state.Response.Err = value, err

	return state
}

func ActResolveCurrencyComponent(t *testing.T, deps Deps, state State) State {
	t.Helper()

	idx := buildIndexes(state.Given.Sources, state.Given.Horizon)
	value, _, err := resolveCurrencyComponent(state.Given.Component, state.Given.Period, idx)
	state.Response.Value, state.Response.Err = value, err

	return state
}

// ── assert ──

func AssertNoError(t *testing.T, state State) {
	t.Helper()

	require.NoError(t, state.Response.Err)
}

func AssertErrorContains(t *testing.T, state State) {
	t.Helper()

	require.Error(t, state.Response.Err)
	assert.Contains(t, state.Response.Err.Error(), state.Expect.ErrSubstr)
}

func AssertNoResolveErrors(t *testing.T, state State) {
	t.Helper()

	assert.Empty(t, state.Response.Errors)
}

func AssertResolveErrorContains(t *testing.T, state State) {
	t.Helper()

	require.NotEmpty(t, state.Response.Errors)
	assert.Contains(t, state.Response.Errors[0], state.Expect.ErrSubstr)

	require.Len(t, state.Response.Details, 1)
	assert.NotEmpty(t, state.Response.Details[0].Error)
	assert.Nil(t, state.Response.Details[0].Value)
}

func AssertDuplicateResolved(t *testing.T, state State) {
	t.Helper()

	require.Contains(t, state.Response.Values, "D")
	assert.InDelta(t, state.Expect.Value, state.Response.Values["D"], 1e-9)
	require.Len(t, state.Response.Details, 1)
}

func AssertEmptyConstantZero(t *testing.T, state State) {
	t.Helper()

	require.Contains(t, state.Response.Values, "L")
	assert.InDelta(t, 0, state.Response.Values["L"], 1e-9)
	require.Len(t, state.Response.Details, 1)
	assert.Contains(t, state.Response.Details[0].Warning, state.Expect.Warning)
}

func AssertQuoteChoice(t *testing.T, state State) {
	t.Helper()

	assert.InDelta(t, state.Expect.Value, state.Response.Value, 1e-9)
	assert.Equal(t, state.Expect.VersionType, state.Response.Meta.VersionType)
	require.NotNil(t, state.Response.Meta.QuoteCode)
	assert.Equal(t, state.Expect.QuoteCode, *state.Response.Meta.QuoteCode)
}

func AssertRateValue(t *testing.T, state State) {
	t.Helper()

	assert.InDelta(t, state.Expect.Value, state.Response.Value, 1e-9)
}
