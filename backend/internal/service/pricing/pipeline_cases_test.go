package pricing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sibur-petrochem-price-service/internal/domain"
)

func TestEngine_Run_Matching(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("matches formula directly by material and via group m", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDirectAndGroupMatch,
			).Then(
				AssertRowStatus(domain.StatusCalculated),
				AssertCandidateScopes,
			)

			result := tc.SUT.Run(tc.State.Given.Sources, nil)
			tc.State.Response.Rows = result.Rows
			tc.State.Response.Result = result.Candidates(tc.State.Given.Sources.Ssp[0].DemandKey())
		})
	})

	t.Run("should be able to not be able", func(t *testing.T) {
		t.Run("spot row is not calculated", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeSpotRow,
			).Then(
				AssertRowStatus(domain.StatusSpotNotCalculated),
				AssertNoCandidates,
			)

			result := tc.SUT.Run(tc.State.Given.Sources, nil)
			tc.State.Response.Rows = result.Rows
			tc.State.Response.Result = result.Candidates(tc.State.Given.Sources.Ssp[0].DemandKey())
		})

		t.Run("inactive and future formulas are excluded", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeInactiveAndFutureFormulas,
			).Then(
				AssertRowStatus(domain.StatusFormulaNotFound),
				AssertNoCandidates,
			)

			result := tc.SUT.Run(tc.State.Given.Sources, nil)
			tc.State.Response.Rows = result.Rows
			tc.State.Response.Result = result.Candidates(tc.State.Given.Sources.Ssp[0].DemandKey())
		})

		t.Run("group match respects group validity window", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeGroupOutsideWindow,
			).Then(
				AssertRowStatus(domain.StatusFormulaNotFound),
				AssertNoCandidates,
			)

			result := tc.SUT.Run(tc.State.Given.Sources, nil)
			tc.State.Response.Rows = result.Rows
			tc.State.Response.Result = result.Candidates(tc.State.Given.Sources.Ssp[0].DemandKey())
		})
	})
}

func TestEngine_Run_Selection(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("actual successful beats expired successful", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeActualAndExpiredFormulas,
			).Then(
				AssertRowStatus(domain.StatusCalculated),
				AssertSelectedFormula,
			)

			result := tc.SUT.Run(tc.State.Given.Sources, nil)
			tc.State.Response.Rows = result.Rows
			tc.State.Response.Result = result.Candidates(tc.State.Given.Sources.Ssp[0].DemandKey())
		})

		t.Run("only expired successful becomes calculated with expired formula", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeOnlyExpiredFormula,
			).Then(
				AssertRowStatus(domain.StatusCalculatedExpired),
				AssertRowWarningContains("Использована неактуальная формула"),
			)

			result := tc.SUT.Run(tc.State.Given.Sources, nil)
			tc.State.Response.Rows = result.Rows
		})

		t.Run("newer created_at wins between actual candidates", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeNewerAndOlderFormulas,
			).Then(
				AssertRowStatus(domain.StatusCalculated),
				AssertSelectedFormula,
			)

			result := tc.SUT.Run(tc.State.Given.Sources, nil)
			tc.State.Response.Rows = result.Rows
			tc.State.Response.Result = result.Candidates(tc.State.Given.Sources.Ssp[0].DemandKey())
		})

		t.Run("equal priority candidates become conflict with min formula id", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeEqualPriorityFormulas,
			).Then(
				AssertRowStatus(domain.StatusFormulaConflict),
				AssertConflictSelection,
			)

			result := tc.SUT.Run(tc.State.Given.Sources, nil)
			tc.State.Response.Rows = result.Rows
			tc.State.Response.Result = result.Candidates(tc.State.Given.Sources.Ssp[0].DemandKey())
		})
	})
}

func TestEngine_Run_StatusesAndConversion(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("converts price from document currency to row currency", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeCurrencyConversionRow,
			).Then(
				AssertRowStatus(domain.StatusCalculated),
				AssertConvertedPrice,
			)

			result := tc.SUT.Run(tc.State.Given.Sources, nil)
			tc.State.Response.Rows = result.Rows
			tc.State.Response.Result = result.Candidates(tc.State.Given.Sources.Ssp[0].DemandKey())
		})

		t.Run("one broken row does not stop the others", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeBrokenAndHealthyRows,
			).Then(
				AssertMixedStatuses,
			)

			result := tc.SUT.Run(tc.State.Given.Sources, nil)
			tc.State.Response.Rows = result.Rows
		})
	})

	t.Run("should be able to not be able", func(t *testing.T) {
		t.Run("missing quote mapping yields component error", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeComponentErrorRow,
			).Then(
				AssertRowStatus(domain.StatusComponentError),
				AssertRowHasError,
			)

			result := tc.SUT.Run(tc.State.Given.Sources, nil)
			tc.State.Response.Rows = result.Rows
		})

		t.Run("broken expression yields invalid formula", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeInvalidFormulaRow,
			).Then(
				AssertRowStatus(domain.StatusInvalidFormula),
				AssertRowHasError,
			)

			result := tc.SUT.Run(tc.State.Given.Sources, nil)
			tc.State.Response.Rows = result.Rows
		})
	})
}

// ── строители данных ──

// baseSources — минимальный набор: одна Formula-строка ssp + формула с константами.
func baseSources(t *testing.T, state State) domain.Sources {
	t.Helper()

	quote := 1000.0
	multiplier := 1.5

	return domain.Sources{
		Ssp: []domain.SspRow{{
			RowID:        int64(state.Faker.IntBetween(1_000_000, 2_000_000)),
			Period:       date(t, "2026-06-01"),
			MaterialID:   226814,
			MaterialName: state.Faker.Lorem().Sentence(2),
			Contract:     "Formula",
			Currency:     "RUB",
			Forecast:     ptr(state.Faker.Float64(1, 10, 900)),
			ClientID:     "CL-10328",
			ClientName:   state.Faker.Company().Name(),
		}},
		Formulas: []domain.Formula{{
			FormulaID:   "Z900000001",
			Text:        "Q * K",
			ValidFrom:   date(t, "2026-01-01"),
			ValidTo:     date(t, "2026-12-31"),
			CreatedAt:   date(t, "2026-01-10"),
			DocCurrency: "RUB",
			MaterialID:  ptr(int64(226814)),
			ClientID:    "CL-10328",
		}},
		Components: []domain.FormulaComponent{
			{
				FormulaID: "Z900000001", TermNo: 1, VarName: "Q", TypeCode: "H", Value: &quote,
				ValidFrom: date(t, "2026-01-01"), ValidTo: date(t, "2026-12-31"),
			},
			{
				FormulaID: "Z900000001", TermNo: 2, VarName: "K", TypeCode: "A", Value: &multiplier,
				ValidFrom: date(t, "2026-01-01"), ValidTo: date(t, "2026-12-31"),
			},
		},
	}
}

func ArrangeDirectAndGroupMatch(t *testing.T, state State) State {
	t.Helper()

	src := baseSources(t, state)
	// вторая формула на группу M, содержащую материал строки
	groupFormula := src.Formulas[0]
	groupFormula.FormulaID = "Z900000002"
	groupFormula.MaterialID = nil
	groupFormula.MaterialGroupM = "MT00000081"
	groupFormula.CreatedAt = date(t, "2026-01-05")
	src.Formulas = append(src.Formulas, groupFormula)
	src.Components = append(src.Components,
		domain.FormulaComponent{
			FormulaID: "Z900000002", TermNo: 1, VarName: "Q", TypeCode: "H", Value: ptr(900.0),
			ValidFrom: date(t, "2026-01-01"), ValidTo: date(t, "2026-12-31"),
		},
		domain.FormulaComponent{
			FormulaID: "Z900000002", TermNo: 2, VarName: "K", TypeCode: "A", Value: ptr(1.2),
			ValidFrom: date(t, "2026-01-01"), ValidTo: date(t, "2026-12-31"),
		},
	)
	src.MaterialGroups = []domain.MaterialGroup{{
		GroupM: "MT00000081", MaterialID: 226814,
		ValidFrom: date(t, "1900-01-01"), ValidTo: date(t, "9999-12-31"),
	}}
	state.Given.Sources = src
	// прямой матч приоритетнее группы
	state.Expect.ErrSubstr = "Z900000001"

	return state
}

func ArrangeSpotRow(t *testing.T, state State) State {
	t.Helper()

	src := baseSources(t, state)
	src.Ssp[0].Contract = "SPOT"
	state.Given.Sources = src

	return state
}

func ArrangeInactiveAndFutureFormulas(t *testing.T, state State) State {
	t.Helper()

	src := baseSources(t, state)
	src.Formulas[0].Inactive = true
	future := src.Formulas[0]
	future.FormulaID = "Z900000003"
	future.Inactive = false
	future.ValidFrom = date(t, "2026-07-01") // начнёт действовать после периода
	src.Formulas = append(src.Formulas, future)
	state.Given.Sources = src

	return state
}

func ArrangeGroupOutsideWindow(t *testing.T, state State) State {
	t.Helper()

	src := baseSources(t, state)
	src.Formulas[0].MaterialID = nil
	src.Formulas[0].MaterialGroupM = "MT00000081"
	src.MaterialGroups = []domain.MaterialGroup{{
		GroupM: "MT00000081", MaterialID: 226814,
		// окно группы закончилось до периода строки
		ValidFrom: date(t, "1900-01-01"), ValidTo: date(t, "2026-01-31"),
	}}
	state.Given.Sources = src

	return state
}

func ArrangeActualAndExpiredFormulas(t *testing.T, state State) State {
	t.Helper()

	src := baseSources(t, state)
	expired := src.Formulas[0]
	expired.FormulaID = "Z900000000" // меньший id — но просрочен, победить не должен
	expired.ValidTo = date(t, "2026-05-31")
	src.Formulas = append(src.Formulas, expired)
	src.Components = append(src.Components, cloneComponents(src.Components[:2], "Z900000000")...)
	state.Given.Sources = src
	state.Expect.ErrSubstr = "Z900000001"

	return state
}

func ArrangeOnlyExpiredFormula(t *testing.T, state State) State {
	t.Helper()

	src := baseSources(t, state)
	src.Formulas[0].ValidTo = date(t, "2026-05-31")
	state.Given.Sources = src

	return state
}

func ArrangeNewerAndOlderFormulas(t *testing.T, state State) State {
	t.Helper()

	src := baseSources(t, state)
	newer := src.Formulas[0]
	newer.FormulaID = "Z900000009"
	newer.CreatedAt = date(t, "2026-03-01") // новее — должна победить
	src.Formulas = append(src.Formulas, newer)
	src.Components = append(src.Components, cloneComponents(src.Components[:2], "Z900000009")...)
	state.Given.Sources = src
	state.Expect.ErrSubstr = "Z900000009"

	return state
}

func ArrangeEqualPriorityFormulas(t *testing.T, state State) State {
	t.Helper()

	src := baseSources(t, state)
	twin := src.Formulas[0]
	twin.FormulaID = "Z900000005" // те же created_at/valid_from/scope → равный приоритет
	src.Formulas = append(src.Formulas, twin)
	src.Components = append(src.Components, cloneComponents(src.Components[:2], "Z900000005")...)
	state.Given.Sources = src
	// tie-break: минимальный formula_id
	state.Expect.ErrSubstr = "Z900000001"

	return state
}

func ArrangeCurrencyConversionRow(t *testing.T, state State) State {
	t.Helper()

	src := baseSources(t, state)
	src.Formulas[0].DocCurrency = "USD"
	src.CurrencyRates = []domain.CurrencyRate{
		{Currency: "USD", CalDay: date(t, "2026-06-01"), VersionType: "Факт", Value: 90.0},
	}
	state.Given.Sources = src
	// 1000 * 1.5 = 1500 USD → 1500 * 90 / 1 RUB
	state.Expect.Value = 135000

	return state
}

func ArrangeComponentErrorRow(t *testing.T, state State) State {
	t.Helper()

	src := baseSources(t, state)
	// компонент-котировка без маппинга
	src.Components[0] = domain.FormulaComponent{
		FormulaID: "Z900000001", TermNo: 1, VarName: "Q",
		ValidFrom: date(t, "2026-01-01"), ValidTo: date(t, "2026-12-31"),
		TypeCode: "1", QuoteName: "NO SUCH QUOTE",
	}
	state.Given.Sources = src

	return state
}

func ArrangeInvalidFormulaRow(t *testing.T, state State) State {
	t.Helper()

	src := baseSources(t, state)
	src.Formulas[0].Text = "Q * / K"
	state.Given.Sources = src

	return state
}

func ArrangeBrokenAndHealthyRows(t *testing.T, state State) State {
	t.Helper()

	src := baseSources(t, state)
	broken := src.Ssp[0]
	broken.RowID = src.Ssp[0].RowID + 1
	broken.MaterialID = 999999 // формулы нет → FORMULA_NOT_FOUND
	src.Ssp = append(src.Ssp, broken)
	state.Given.Sources = src

	return state
}

func cloneComponents(components []domain.FormulaComponent, formulaID string) []domain.FormulaComponent {
	out := make([]domain.FormulaComponent, 0, len(components))
	for _, component := range components {
		clone := component
		clone.FormulaID = formulaID
		out = append(out, clone)
	}

	return out
}

// ── assert ──

func AssertRowStatus(status domain.Status) func(*testing.T, State) {
	return func(t *testing.T, state State) {
		t.Helper()

		require.NotEmpty(t, state.Response.Rows)
		assert.Equal(t, status, state.Response.Rows[0].Status)
	}
}

func AssertNoCandidates(t *testing.T, state State) {
	t.Helper()

	assert.Empty(t, state.Response.Result)
	require.NotEmpty(t, state.Response.Rows)
	assert.Zero(t, state.Response.Rows[0].CandidateCount)
	assert.Nil(t, state.Response.Rows[0].Price)
}

func AssertCandidateScopes(t *testing.T, state State) {
	t.Helper()

	require.Len(t, state.Response.Result, 2)

	scopes := map[domain.MatchScope]bool{}
	for _, candidate := range state.Response.Result {
		scopes[candidate.MatchScope] = true
	}
	assert.True(t, scopes[domain.MatchScopeMaterial])
	assert.True(t, scopes[domain.MatchScopeGroupM])

	// применён прямой матч (scope_priority material < group)
	AssertSelectedFormula(t, state)
}

func AssertSelectedFormula(t *testing.T, state State) {
	t.Helper()

	var selected *domain.CandidateResult
	for i := range state.Response.Result {
		if state.Response.Result[i].IsSelected {
			selected = &state.Response.Result[i]
		}
	}
	require.NotNil(t, selected, "должен быть выбранный кандидат")
	assert.Equal(t, state.Expect.ErrSubstr, selected.FormulaID)
}

func AssertConflictSelection(t *testing.T, state State) {
	t.Helper()

	require.NotEmpty(t, state.Response.Rows)
	row := state.Response.Rows[0]
	assert.True(t, row.RequiresReview)
	assert.Equal(t, 2, row.EqualPriorityCount)

	AssertSelectedFormula(t, state)
}

func AssertRowWarningContains(substr string) func(*testing.T, State) {
	return func(t *testing.T, state State) {
		t.Helper()

		require.NotEmpty(t, state.Response.Rows)
		assert.Contains(t, state.Response.Rows[0].Warning, substr)
	}
}

func AssertRowHasError(t *testing.T, state State) {
	t.Helper()

	require.NotEmpty(t, state.Response.Rows)
	assert.NotEmpty(t, state.Response.Rows[0].Error)
	assert.Nil(t, state.Response.Rows[0].Price)
}

func AssertConvertedPrice(t *testing.T, state State) {
	t.Helper()

	require.NotEmpty(t, state.Response.Rows)
	row := state.Response.Rows[0]
	require.NotNil(t, row.Price)
	assert.InDelta(t, state.Expect.Value, *row.Price, 1e-6)

	var selected *domain.CandidateResult
	for i := range state.Response.Result {
		if state.Response.Result[i].IsSelected {
			selected = &state.Response.Result[i]
		}
	}
	require.NotNil(t, selected)
	require.NotNil(t, selected.PriceFormulaCurrency)
	assert.InDelta(t, 1500, *selected.PriceFormulaCurrency, 1e-6)
	require.NotNil(t, selected.Conversion)
	assert.Equal(t, "USD", selected.Conversion.FromCurrency)
	assert.Equal(t, "RUB", selected.Conversion.ToCurrency)
}

func AssertMixedStatuses(t *testing.T, state State) {
	t.Helper()

	require.Len(t, state.Response.Rows, 2)

	statuses := map[domain.Status]bool{}
	for _, row := range state.Response.Rows {
		statuses[row.Status] = true
	}
	assert.True(t, statuses[domain.StatusCalculated], "здоровая строка посчитана")
	assert.True(t, statuses[domain.StatusFormulaNotFound], "проблемная строка помечена")
}
