package http

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sibur-petrochem-price-service/internal/domain"

	nethttp "net/http"
)

// Датасет (см. demoSources): период 2026-06, строки
// 1001 calculated, 1002 calculated USD, 1003 SPOT, 1004 no_formula, 1005 conflict.

func demoSources(t *testing.T, state State) domain.Sources {
	t.Helper()

	quote, factor := 1000.0, 1.5
	twinQuote, twinFactor := 900.0, 1.2

	makeRow := func(rowID, material int64, client, currency string, forecast float64) domain.SspRow {
		return domain.SspRow{
			RowID: rowID, Period: date(t, "2026-06-01"),
			MaterialID: material, MaterialName: "Материал " + client,
			Contract: "Formula", Currency: currency, Forecast: ptr(forecast),
			ClientID: client, ClientName: "Клиент " + client,
		}
	}
	makeFormula := func(id, client string, material int64, docCurrency string) domain.Formula {
		return domain.Formula{
			FormulaID: id, Text: "Q * K",
			ValidFrom: date(t, "2026-01-01"), ValidTo: date(t, "2026-12-31"),
			CreatedAt: date(t, "2026-01-10"), DocCurrency: docCurrency,
			MaterialID: ptr(material), ClientID: client,
		}
	}
	makeComponents := func(formulaID string, q, k float64) []domain.FormulaComponent {
		return []domain.FormulaComponent{
			{
				FormulaID: formulaID, TermNo: 1, VarName: "Q", TypeCode: "H", Value: &q,
				ValidFrom: date(t, "2026-01-01"), ValidTo: date(t, "2026-12-31"),
			},
			{
				FormulaID: formulaID, TermNo: 2, VarName: "K", TypeCode: "A", Value: &k,
				ValidFrom: date(t, "2026-01-01"), ValidTo: date(t, "2026-12-31"),
			},
		}
	}

	spot := makeRow(1003, 300, "CL-3", "RUB", 7)
	spot.Contract = "SPOT"

	components := makeComponents("ZA00000001", quote, factor)
	components = append(components, makeComponents("ZB00000001", 100, 1)...)
	components = append(components, makeComponents("ZC00000001", quote, factor)...)
	components = append(components, makeComponents("ZC00000002", twinQuote, twinFactor)...)

	return domain.Sources{
		Ssp: []domain.SspRow{
			makeRow(1001, 100, "CL-1", "RUB", 10),
			makeRow(1002, 200, "CL-2", "USD", 2),
			spot,
			makeRow(1004, 999, "CL-4", "RUB", 5),
			makeRow(1005, 500, "CL-5", "RUB", 1),
		},
		Formulas: []domain.Formula{
			makeFormula("ZA00000001", "CL-1", 100, "RUB"),
			makeFormula("ZB00000001", "CL-2", 200, "USD"),
			makeFormula("ZC00000001", "CL-5", 500, "RUB"),
			makeFormula("ZC00000002", "CL-5", 500, "RUB"),
		},
		Components: components,
		CurrencyRates: []domain.CurrencyRate{
			{Currency: "USD", CalDay: date(t, "2026-06-01"), VersionType: "Факт", Value: 90},
		},
	}
}

func ArrangeDemo(t *testing.T, state State) State {
	t.Helper()

	state.Given.Sources = demoSources(t, state)
	state.Given.Period = "2026-06"

	return state
}

func ActLoadSources(t *testing.T, deps Deps, state State) State {
	t.Helper()

	deps.Sources.src = state.Given.Sources
	deps.Sources.counts = map[string]int64{
		"ssp": 5, "formulas": 4, "formula_components": 8, "term_types": 11,
		"quotes": 0, "quote_mapping": 0, "currency_rates": 1, "material_groups": 0,
	}

	return state
}

func AssertCode(code int) func(*testing.T, State) {
	return func(t *testing.T, state State) {
		t.Helper()

		assert.Equal(t, code, state.Response.Code, "body: %s", string(state.Response.Raw))
	}
}

func TestSources(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("lists eight sources with row counts", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusOK),
				AssertSourcesList,
			)

			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodGet, "/api/v1/sources", nil,
			)
		})

		t.Run("previews ssp source", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusOK),
				AssertPreview,
			)

			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodGet, "/api/v1/sources/ssp/preview?limit=3", nil,
			)
		})

		t.Run("returns facets from ssp", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusOK),
				AssertFacets,
			)

			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodGet, "/api/v1/sources/facets", nil,
			)
		})
	})
}

func AssertFacets(t *testing.T, state State) {
	t.Helper()

	products, ok := state.Response.Body["products"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, products)

	clients, ok := state.Response.Body["clients"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, clients)

	assert.Equal(t, "2026-06", state.Response.Body["period_min"])
	assert.Equal(t, "2026-06", state.Response.Body["period_max"])
}

func TestCalculationsEndpoints(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("creates calculation and returns it by id", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusOK),
				AssertCalculationBody,
			)

			id := createCalculation(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodGet, "/api/v1/calculations/"+id, nil,
			)
		})

		t.Run("streams progress as sse with done event", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusOK),
				AssertSseDone,
			)

			id := createCalculation(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodGet, "/api/v1/calculations/"+id+"/events", nil,
			)
		})

		t.Run("returns five canonical kpi", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusOK),
				AssertKpiBody,
			)

			id := createCalculation(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodGet, "/api/v1/calculations/"+id+"/kpi", nil,
			)
		})

		t.Run("exports xlsx file", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusOK),
				AssertXlsxBody,
			)

			id := createCalculation(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodGet, "/api/v1/calculations/"+id+"/export", nil,
			)
		})
	})

	t.Run("should be able to not be able", func(t *testing.T) {
		t.Run("unknown period yields conflict", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusConflict),
			)

			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodPost, "/api/v1/calculations", calcBody("2031-01"),
			)
		})

		t.Run("unknown calculation yields not found", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusNotFound),
			)

			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodGet, "/api/v1/calculations/2ec3e716-6a2a-4f0e-9dd0-000000000000", nil,
			)
		})
	})
}

func TestRowsEndpoints(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("lists rows with status counts and filter", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusOK),
				AssertRowsBody,
			)

			id := createCalculation(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodGet, "/api/v1/calculations/"+id+"/rows?status=calculated", nil,
			)
		})

		t.Run("returns row details with applied formula", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusOK),
				AssertDetailsBody,
			)

			id := createCalculation(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodGet, "/api/v1/calculations/"+id+"/rows/1001", nil,
			)
		})

		t.Run("sets and resets manual price", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusOK),
				AssertManualCleared,
			)

			id := createCalculation(t, tc.SUT, tc.State.Given.Period)

			code, body, _ := do(t, tc.SUT, nethttp.MethodPut,
				"/api/v1/calculations/"+id+"/rows/1004/manual-price", map[string]any{"price": 120000})
			require.Equal(t, nethttp.StatusOK, code)
			row, ok := body["row"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, "manual", row["status"])

			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodDelete, "/api/v1/calculations/"+id+"/rows/1004/manual-price", nil,
			)
		})

		t.Run("selects alternative formula", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusOK),
				AssertSelectedFormula,
			)

			id := createCalculation(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodPut, "/api/v1/calculations/"+id+"/rows/1005/formula",
				map[string]any{"formula_id": "ZC00000002"},
			)
		})
	})

	t.Run("should be able to not be able", func(t *testing.T) {
		t.Run("unknown row yields not found", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusNotFound),
			)

			id := createCalculation(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodGet, "/api/v1/calculations/"+id+"/rows/777777", nil,
			)
		})

		t.Run("non-positive manual price yields unprocessable entity", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusUnprocessableEntity),
			)

			id := createCalculation(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodPut,
				"/api/v1/calculations/"+id+"/rows/1001/manual-price", map[string]any{"price": -5},
			)
		})

		t.Run("foreign formula yields unprocessable entity", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusUnprocessableEntity),
			)

			id := createCalculation(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodPut, "/api/v1/calculations/"+id+"/rows/1005/formula",
				map[string]any{"formula_id": "Z_UNKNOWN"},
			)
		})
	})
}

func TestFormulasAndConsolidated(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("parses formula expression", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusOK),
				AssertParsedFormula,
			)

			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodPost, "/api/v1/formulas/parse",
				map[string]any{"formula_text": "IF ( Q < SPOT , Q , SPOT )"},
			)
		})

		t.Run("submits part and returns consolidated document", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusOK),
				AssertConsolidatedBody,
			)

			id := createCalculation(t, tc.SUT, tc.State.Given.Period)

			code, body, _ := do(t, tc.SUT, nethttp.MethodPost, "/api/v1/calculations/"+id+"/submission", nil)
			require.Equal(t, nethttp.StatusOK, code)
			require.Equal(t, "joined", body["status"])

			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodGet, "/api/v1/consolidated/all", nil,
			)
		})
	})

	t.Run("should be able to not be able", func(t *testing.T) {
		t.Run("double submission yields conflict", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(ArrangeDemo).When(ActLoadSources).Then(
				AssertCode(nethttp.StatusConflict),
			)

			id := createCalculation(t, tc.SUT, tc.State.Given.Period)
			_, _, _ = do(t, tc.SUT, nethttp.MethodPost, "/api/v1/calculations/"+id+"/submission", nil)

			tc.State.Response.Code, tc.State.Response.Body, tc.State.Response.Raw = do(
				t, tc.SUT, nethttp.MethodPost, "/api/v1/calculations/"+id+"/submission", nil,
			)
		})
	})
}

// ── assert ──

func AssertSourcesList(t *testing.T, state State) {
	t.Helper()

	items, ok := state.Response.Body["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 8)

	first, ok := items[0].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, first["key"])
	assert.NotEmpty(t, first["name"])
	assert.Equal(t, "loaded", first["status"])
}

func AssertPreview(t *testing.T, state State) {
	t.Helper()

	columns, ok := state.Response.Body["columns"].([]any)
	require.True(t, ok)
	assert.NotEmpty(t, columns)

	rows, ok := state.Response.Body["rows"].([]any)
	require.True(t, ok)
	assert.LessOrEqual(t, len(rows), 3)
	assert.NotEmpty(t, rows)
}

func AssertCalculationBody(t *testing.T, state State) {
	t.Helper()

	assert.Equal(t, "done", state.Response.Body["status"])
	assert.Equal(t, "2026-06 — 2026-06", state.Response.Body["period"])

	progress, ok := state.Response.Body["progress"].(map[string]any)
	require.True(t, ok)
	assert.InDelta(t, 100, progress["percent"], 0.1)
}

func AssertSseDone(t *testing.T, state State) {
	t.Helper()

	payload := string(state.Response.Raw)
	assert.True(t, strings.HasPrefix(payload, "data: "), "SSE-формат: %q", payload)
	assert.Contains(t, payload, `"status":"done"`)
}

func AssertKpiBody(t *testing.T, state State) {
	t.Helper()

	for _, field := range []string{
		"formula_coverage_pct", "formulas_ok_pct", "calc_error_rows", "control_sum_mln", "unclassified_error_pct",
	} {
		assert.Contains(t, state.Response.Body, field)
	}
}

func AssertXlsxBody(t *testing.T, state State) {
	t.Helper()

	assert.NotEmpty(t, state.Response.Raw)
	// xlsx — это zip: сигнатура PK
	assert.True(t, len(state.Response.Raw) > 2 && state.Response.Raw[0] == 'P' && state.Response.Raw[1] == 'K')
}

func AssertRowsBody(t *testing.T, state State) {
	t.Helper()

	items, ok := state.Response.Body["items"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, items)
	for _, item := range items {
		row, ok := item.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "calculated", row["status"])
	}

	counts, ok := state.Response.Body["status_counts"].(map[string]any)
	require.True(t, ok)
	assert.InDelta(t, 2, counts["calculated"], 0.1)
	assert.InDelta(t, 1, counts["formula_conflict"], 0.1)
}

func AssertDetailsBody(t *testing.T, state State) {
	t.Helper()

	applied, ok := state.Response.Body["applied_formula"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ZA00000001", applied["formula_id"])

	components, ok := state.Response.Body["components"].([]any)
	require.True(t, ok)
	assert.Len(t, components, 2)

	alternatives, ok := state.Response.Body["alternatives"].([]any)
	require.True(t, ok)
	assert.NotEmpty(t, alternatives)
}

func AssertManualCleared(t *testing.T, state State) {
	t.Helper()

	row, ok := state.Response.Body["row"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "no_formula", row["status"])
	assert.Nil(t, row["final_price"])
}

func AssertSelectedFormula(t *testing.T, state State) {
	t.Helper()

	applied, ok := state.Response.Body["applied_formula"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ZC00000002", applied["formula_id"])

	row, ok := state.Response.Body["row"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, row["requires_review"])
}

func AssertParsedFormula(t *testing.T, state State) {
	t.Helper()

	assert.Equal(t, true, state.Response.Body["valid"])

	variables, ok := state.Response.Body["variables"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"Q", "SPOT"}, variables)

	functions, ok := state.Response.Body["functions"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"IF"}, functions)
}

func AssertConsolidatedBody(t *testing.T, state State) {
	t.Helper()

	parts, ok := state.Response.Body["parts"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, parts)
	first, ok := parts[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "joined", first["status"])

	rows, ok := state.Response.Body["rows"].([]any)
	require.True(t, ok)
	assert.Len(t, rows, 5)

	require.Contains(t, state.Response.Body, "kpi")
}
