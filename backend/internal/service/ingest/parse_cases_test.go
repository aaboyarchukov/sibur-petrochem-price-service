package ingest

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Arrange ---

func ArrangeSspHeaders(t *testing.T, state State) State {
	t.Helper()
	state.Given.Headers = append([]string(nil), sspColumns...)

	return state
}

func ArrangeFormulaHeaders(t *testing.T, state State) State {
	t.Helper()
	state.Given.Headers = append([]string(nil), formulaColumns...)

	return state
}

func withoutHeader(name string) func(*testing.T, State) State {
	return func(t *testing.T, state State) State {
		t.Helper()
		kept := make([]string, 0, len(state.Given.Headers))
		for _, header := range state.Given.Headers {
			if header != name {
				kept = append(kept, header)
			}
		}
		state.Given.Headers = kept

		return state
	}
}

func expectIssues(substrs ...string) func(*testing.T, State) State {
	return func(t *testing.T, state State) State {
		t.Helper()
		state.Expect.IssueSubstrs = substrs

		return state
	}
}

// sspRow — валидная строка ssp; period и forecast задаются аргументами.
func sspRow(state State, rowID int64, period, forecast any) []any {
	company := state.Faker.Company()

	return []any{
		rowID, period, "354065", company.Name(), int64(226814), "Полипропилен " + company.Suffix(),
		"Formula", "RUB", forecast, "Russia", "Region", "Россия", "CL-10001", "Клиент 001",
	}
}

// formulaRow — валидная строка каталога формул.
func formulaRow(state State, id, inactive string, material any) []any {
	return []any{
		id, "( CFR - DISCOUNT ) * K", "", "91442",
		"2026-01-01 00:00:00", "2026-12-31 00:00:00", "2025-12-17 00:00:00",
		inactive, "2", "CNY", material, "CL-10395", "Клиент " + state.Faker.Person().FirstName(),
	}
}

// --- Assert ---

func AssertNoIssues(t *testing.T, state State) {
	t.Helper()
	require.Empty(t, state.Response.Issues)
}

func AssertIssues(t *testing.T, state State) {
	t.Helper()
	require.NotEmpty(t, state.Response.Issues)

	joined := make([]string, 0, len(state.Response.Issues))
	for _, issue := range state.Response.Issues {
		joined = append(joined, issue.String())
	}
	all := strings.Join(joined, "\n")
	for _, substr := range state.Expect.IssueSubstrs {
		assert.Contains(t, all, substr)
	}
	assert.Empty(t, state.Response.Ssp)
	assert.Empty(t, state.Response.Formulas)
}

// --- Tests ---

func TestParseSspValidFile(t *testing.T) {
	tc := newTestContext(t)

	tc.Given(
		ArrangeSspHeaders,
		func(t *testing.T, state State) State {
			t.Helper()
			state.Given.Cells = append(state.Given.Cells,
				sspRow(state, 1001, "2026-06-01 00:00:00", "1572.5"),
				sspRow(state, 1002, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), nil),
			)

			return state
		},
	).Then(
		AssertNoIssues,
		func(t *testing.T, state State) {
			t.Helper()
			require.Len(t, state.Response.Ssp, 2)
			first, second := state.Response.Ssp[0], state.Response.Ssp[1]
			assert.Equal(t, int64(1001), first.RowID)
			assert.Equal(t, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), first.Period)
			require.NotNil(t, first.Forecast)
			assert.InDelta(t, 1572.5, *first.Forecast, 1e-9)
			// excel-дата и строковая дата дают один и тот же период
			assert.Equal(t, first.Period, second.Period)
			assert.Nil(t, second.Forecast)
		},
	)

	ssp, issues := tc.SUT.Ssp(buildXlsx(t, tc.State.Given.Headers, tc.State.Given.Cells))
	tc.State.Response = Responses{Ssp: ssp, Issues: issues}
}

func TestParseSspMissingColumn(t *testing.T) {
	tc := newTestContext(t)

	tc.Given(
		ArrangeSspHeaders,
		withoutHeader("period"),
		expectIssues("period", "обязательная колонка отсутствует"),
	).Then(AssertIssues)

	ssp, issues := tc.SUT.Ssp(buildXlsx(t, tc.State.Given.Headers, tc.State.Given.Cells))
	tc.State.Response = Responses{Ssp: ssp, Issues: issues}
}

func TestParseSspBrokenRows(t *testing.T) {
	tc := newTestContext(t)

	tc.Given(
		ArrangeSspHeaders,
		func(t *testing.T, state State) State {
			t.Helper()
			state.Given.Cells = append(state.Given.Cells,
				sspRow(state, 1001, "2026-06-01", "не число"),
				sspRow(state, 1002, "кривая дата", "10"),
			)

			return state
		},
		expectIssues("строка 2", "forecast", "строка 3", "period", "некорректная дата"),
	).Then(AssertIssues)

	ssp, issues := tc.SUT.Ssp(buildXlsx(t, tc.State.Given.Headers, tc.State.Given.Cells))
	tc.State.Response = Responses{Ssp: ssp, Issues: issues}
}

func TestParseSspDuplicateRow(t *testing.T) {
	tc := newTestContext(t)

	tc.Given(
		ArrangeSspHeaders,
		func(t *testing.T, state State) State {
			t.Helper()
			state.Given.Cells = append(state.Given.Cells,
				sspRow(state, 1001, "2026-06-01", "10"),
				sspRow(state, 1001, "2026-06-01", "20"),
			)

			return state
		},
		expectIssues("строка 3", "дубль (row_id, period)"),
	).Then(AssertIssues)

	ssp, issues := tc.SUT.Ssp(buildXlsx(t, tc.State.Given.Headers, tc.State.Given.Cells))
	tc.State.Response = Responses{Ssp: ssp, Issues: issues}
}

func TestParseSspNotXlsx(t *testing.T) {
	tc := newTestContext(t)

	tc.Given(expectIssues("не читается как .xlsx")).Then(AssertIssues)

	ssp, issues := tc.SUT.Ssp(strings.NewReader("это не xlsx"))
	tc.State.Response = Responses{Ssp: ssp, Issues: issues}
}

func TestParseFormulasValidFile(t *testing.T) {
	tc := newTestContext(t)

	tc.Given(
		ArrangeFormulaHeaders,
		func(t *testing.T, state State) State {
			t.Helper()
			state.Given.Cells = append(state.Given.Cells,
				formulaRow(state, "Z430000113", "", int64(1353711)),
				formulaRow(state, "Z430000129", "X", nil),
			)

			return state
		},
	).Then(
		AssertNoIssues,
		func(t *testing.T, state State) {
			t.Helper()
			require.Len(t, state.Response.Formulas, 2)
			active, inactive := state.Response.Formulas[0], state.Response.Formulas[1]
			assert.False(t, active.Inactive)
			require.NotNil(t, active.MaterialID)
			assert.Equal(t, int64(1353711), *active.MaterialID)
			assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), active.ValidFrom)
			assert.Equal(t, "CNY", active.DocCurrency)
			assert.True(t, inactive.Inactive)
			assert.Nil(t, inactive.MaterialID)
		},
	)

	formulas, issues := tc.SUT.Formulas(buildXlsx(t, tc.State.Given.Headers, tc.State.Given.Cells))
	tc.State.Response = Responses{Formulas: formulas, Issues: issues}
}

func TestParseFormulasBrokenRows(t *testing.T) {
	tc := newTestContext(t)

	tc.Given(
		ArrangeFormulaHeaders,
		func(t *testing.T, state State) State {
			t.Helper()
			broken := formulaRow(state, "Z430000113", "да", int64(1))
			noDate := formulaRow(state, "Z430000114", "", nil)
			noDate[4] = "когда-нибудь"
			state.Given.Cells = append(state.Given.Cells, broken, noDate)

			return state
		},
		expectIssues("Неактивна", "ожидается пусто либо X", "Действительно с", "некорректная дата"),
	).Then(AssertIssues)

	formulas, issues := tc.SUT.Formulas(buildXlsx(t, tc.State.Given.Headers, tc.State.Given.Cells))
	tc.State.Response = Responses{Formulas: formulas, Issues: issues}
}

func TestParseFormulasEmptyRequired(t *testing.T) {
	tc := newTestContext(t)

	tc.Given(
		ArrangeFormulaHeaders,
		func(t *testing.T, state State) State {
			t.Helper()
			state.Given.Cells = append(state.Given.Cells, formulaRow(state, "", "", nil))

			return state
		},
		expectIssues("Ключ формулы", "пустое обязательное значение"),
	).Then(AssertIssues)

	formulas, issues := tc.SUT.Formulas(buildXlsx(t, tc.State.Given.Headers, tc.State.Given.Cells))
	tc.State.Response = Responses{Formulas: formulas, Issues: issues}
}
