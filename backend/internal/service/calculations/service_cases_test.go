package calculations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sibur-petrochem-price-service/internal/domain"
)

func TestService_Create(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("runs calculation for requested period only", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
			).When(
				ActLoadSources,
			).Then(
				AssertNoError,
				AssertCalculationDone,
			)

			tc.State.Response.Info, tc.State.Response.Err = tc.SUT.Create(t.Context(), tc.State.Given.Period)
		})

		t.Run("subscriber after completion receives done event", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
			).When(
				ActLoadSources,
			).Then(
				AssertNoError,
			)

			info, err := tc.SUT.Create(t.Context(), tc.State.Given.Period)
			require.NoError(t, err)

			events, unsubscribe, err := tc.SUT.Subscribe(info.ID)
			require.NoError(t, err)
			defer unsubscribe()

			event, ok := <-events
			require.True(t, ok, "должно прийти событие-снапшот")
			assert.Equal(t, "done", event.Status)
			assert.Equal(t, 100, event.Percent)

			tc.State.Response.Err = nil
		})
	})

	t.Run("should be able to not be able", func(t *testing.T) {
		t.Run("fails on period without demand rows", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
				ArrangeUnknownPeriod,
			).When(
				ActLoadSources,
			).Then(
				AssertErrorIs(domain.ErrSourcesNotLoaded),
			)

			tc.State.Response.Info, tc.State.Response.Err = tc.SUT.Create(t.Context(), tc.State.Given.Period)
		})
	})
}

func TestService_Rows(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("filters by status and counts statuses", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
			).When(
				ActLoadSources,
			).Then(
				AssertNoError,
				AssertRowsAllStatus(domain.StatusCalculated),
				AssertStatusCountsComplete,
			)

			info := mustCreate(t, tc.SUT, tc.State.Given.Period)
			status := domain.StatusCalculated
			tc.State.Response.Page, tc.State.Response.Err = tc.SUT.Rows(info.ID, RowsQuery{Status: &status})
		})

		t.Run("searches by client name and paginates", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
			).When(
				ActLoadSources,
			).Then(
				AssertNoError,
				AssertSearchResult,
			)

			info := mustCreate(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Page, tc.State.Response.Err = tc.SUT.Rows(info.ID, RowsQuery{
				Query: "клиент cl-2", Limit: 10,
			})
		})

		t.Run("sorts by price descending", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
			).When(
				ActLoadSources,
			).Then(
				AssertNoError,
				AssertSortedByPriceDesc,
			)

			info := mustCreate(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Page, tc.State.Response.Err = tc.SUT.Rows(info.ID, RowsQuery{
				Sort: "price", Order: "desc",
			})
		})
	})
}

func TestService_Details(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("returns applied formula with components and alternatives", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
			).When(
				ActLoadSources,
			).Then(
				AssertNoError,
				AssertCalculatedDetails,
			)

			info := mustCreate(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Details, tc.State.Response.Err = tc.SUT.Details(info.ID, "1001")
		})
	})

	t.Run("should be able to not be able", func(t *testing.T) {
		t.Run("fails on unknown row", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
			).When(
				ActLoadSources,
			).Then(
				AssertErrorIs(domain.ErrRowNotFound),
			)

			info := mustCreate(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Details, tc.State.Response.Err = tc.SUT.Details(info.ID, "777777")
		})

		t.Run("fails on unknown calculation", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
			).When(
				ActLoadSources,
			).Then(
				AssertErrorIs(domain.ErrCalculationNotFound),
			)

			tc.State.Response.Details, tc.State.Response.Err = tc.SUT.Details("no-such-calc", "1001")
		})
	})
}

func TestService_Mutations(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("manual price makes row manual", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
				ArrangeManualPrice(120000),
			).When(
				ActLoadSources,
			).Then(
				AssertNoError,
				AssertRowStatusIs(domain.StatusManual),
				AssertManualPriceApplied,
			)

			info := mustCreate(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Details, tc.State.Response.Err = tc.SUT.SetManualPrice(
				info.ID, "1004", tc.State.Given.Price,
			)
		})

		t.Run("reset manual price restores original status", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
				ArrangeManualPrice(120000),
			).When(
				ActLoadSources,
			).Then(
				AssertNoError,
				AssertRowStatusIs(domain.StatusFormulaNotFound),
			)

			info := mustCreate(t, tc.SUT, tc.State.Given.Period)
			_, err := tc.SUT.SetManualPrice(info.ID, "1004", tc.State.Given.Price)
			require.NoError(t, err)

			tc.State.Response.Details, tc.State.Response.Err = tc.SUT.ResetManualPrice(info.ID, "1004")
		})

		t.Run("selecting formula switches price and clears review flag", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
				ArrangeSelectedFormula("ZC00000002"),
			).When(
				ActLoadSources,
			).Then(
				AssertNoError,
				AssertSelectedFormulaApplied,
			)

			info := mustCreate(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Details, tc.State.Response.Err = tc.SUT.SelectFormula(
				info.ID, "1005", tc.State.Given.Formula,
			)
		})
	})

	t.Run("should be able to not be able", func(t *testing.T) {
		t.Run("rejects non-positive manual price", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
				ArrangeManualPrice(0),
			).When(
				ActLoadSources,
			).Then(
				AssertErrorIs(domain.ErrInvalidPrice),
			)

			info := mustCreate(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Details, tc.State.Response.Err = tc.SUT.SetManualPrice(
				info.ID, "1001", tc.State.Given.Price,
			)
		})

		t.Run("rejects formula outside candidates", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
				ArrangeSelectedFormula("Z_NOT_A_CANDIDATE"),
			).When(
				ActLoadSources,
			).Then(
				AssertErrorIs(domain.ErrFormulaNotAllowed),
			)

			info := mustCreate(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Details, tc.State.Response.Err = tc.SUT.SelectFormula(
				info.ID, "1005", tc.State.Given.Formula,
			)
		})
	})
}

func TestService_Kpi(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("computes five canonical kpi", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
			).When(
				ActLoadSources,
			).Then(
				AssertNoError,
				AssertCanonicalKpi,
			)

			info := mustCreate(t, tc.SUT, tc.State.Given.Period)
			tc.State.Response.Kpi, tc.State.Response.Err = tc.SUT.Kpi(info.ID)
		})

		t.Run("manual price increases control sum", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
				ArrangeManualPrice(100000),
			).When(
				ActLoadSources,
			).Then(
				AssertNoError,
				AssertControlSumGrew,
			)

			info := mustCreate(t, tc.SUT, tc.State.Given.Period)
			_, err := tc.SUT.SetManualPrice(info.ID, "1004", tc.State.Given.Price)
			require.NoError(t, err)

			tc.State.Response.Kpi, tc.State.Response.Err = tc.SUT.Kpi(info.ID)
		})
	})
}

func TestService_Consolidated(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("submission joins part to consolidated document", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
			).When(
				ActLoadSources,
			).Then(
				AssertNoError,
				AssertPartJoined,
				AssertConsolidatedRows,
			)

			info := mustCreate(t, tc.SUT, tc.State.Given.Period)

			part, err := tc.SUT.Submit(info.ID)
			require.NoError(t, err)
			tc.State.Response.Part = part

			tc.State.Response.Doc, tc.State.Response.Err = tc.SUT.Consolidated(tc.State.Given.Period)
		})
	})

	t.Run("should be able to not be able", func(t *testing.T) {
		t.Run("second submission is rejected", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDemoSources,
			).When(
				ActLoadSources,
			).Then(
				AssertErrorIs(domain.ErrAlreadySubmitted),
			)

			info := mustCreate(t, tc.SUT, tc.State.Given.Period)
			_, err := tc.SUT.Submit(info.ID)
			require.NoError(t, err)

			_, tc.State.Response.Err = tc.SUT.Submit(info.ID)
		})
	})
}

func mustCreate(t *testing.T, sut *Service, period string) Info {
	t.Helper()

	info, err := sut.Create(t.Context(), period)
	require.NoError(t, err)

	return info
}

// ── датасет: 6 строк периода 2026-06 + 1 строка чужого периода ──
// 1001 calculated (RUB, 1000*1.5=1500, forecast 10)
// 1002 calculated (валюта строки USD, формула в USD: 100, forecast 2, курс 90)
// 1003 SPOT
// 1004 no_formula (forecast 5)
// 1005 conflict (две равноприоритетные, цены 1500 и 1080, forecast 1)
// 1006 component_error (котировка без маппинга)

func demoSspRow(t *testing.T, rowID int64, material int64, client, currency string, forecast float64) domain.SspRow {
	t.Helper()

	return domain.SspRow{
		RowID:        rowID,
		Period:       date(t, "2026-06-01"),
		MaterialID:   material,
		MaterialName: "Материал " + client,
		Contract:     "Formula",
		Currency:     currency,
		Forecast:     ptr(forecast),
		ClientID:     client,
		ClientName:   "Клиент " + client,
	}
}

func demoFormula(t *testing.T, id string, client string, material int64, docCurrency string) domain.Formula {
	t.Helper()

	return domain.Formula{
		FormulaID:   id,
		Text:        "Q * K",
		ValidFrom:   date(t, "2026-01-01"),
		ValidTo:     date(t, "2026-12-31"),
		CreatedAt:   date(t, "2026-01-10"),
		DocCurrency: docCurrency,
		MaterialID:  ptr(material),
		ClientID:    client,
	}
}

func demoComponents(t *testing.T, formulaID string, quote, factor float64) []domain.FormulaComponent {
	t.Helper()

	return []domain.FormulaComponent{
		{
			FormulaID: formulaID, TermNo: 1, VarName: "Q", TypeCode: "H", Value: &quote,
			ValidFrom: date(t, "2026-01-01"), ValidTo: date(t, "2026-12-31"),
		},
		{
			FormulaID: formulaID, TermNo: 2, VarName: "K", TypeCode: "A", Value: &factor,
			ValidFrom: date(t, "2026-01-01"), ValidTo: date(t, "2026-12-31"),
		},
	}
}

func ArrangeDemoSources(t *testing.T, state State) State {
	t.Helper()

	spot := demoSspRow(t, 1003, 300, "CL-3", "RUB", 7)
	spot.Contract = "SPOT"

	otherPeriod := demoSspRow(t, 2001, 100, "CL-1", "RUB", 3)
	otherPeriod.Period = date(t, "2026-07-01")

	quoteComponent := domain.FormulaComponent{
		FormulaID: "ZD00000001", TermNo: 1, VarName: "Q", TypeCode: "1", QuoteName: "NO MAPPING",
		ValidFrom: date(t, "2026-01-01"), ValidTo: date(t, "2026-12-31"),
	}

	components := demoComponents(t, "ZA00000001", 1000, 1.5)
	components = append(components, demoComponents(t, "ZB00000001", 100, 1)...)
	components = append(components, demoComponents(t, "ZC00000001", 1000, 1.5)...)
	components = append(components, demoComponents(t, "ZC00000002", 900, 1.2)...)
	components = append(components, quoteComponent)

	state.Given.Sources = domain.Sources{
		Ssp: []domain.SspRow{
			demoSspRow(t, 1001, 100, "CL-1", "RUB", 10),
			demoSspRow(t, 1002, 200, "CL-2", "USD", 2),
			spot,
			demoSspRow(t, 1004, 999, "CL-4", "RUB", 5),
			demoSspRow(t, 1005, 500, "CL-5", "RUB", 1),
			demoSspRow(t, 1006, 600, "CL-6", "RUB", 4),
			otherPeriod,
		},
		Formulas: []domain.Formula{
			demoFormula(t, "ZA00000001", "CL-1", 100, "RUB"),
			demoFormula(t, "ZB00000001", "CL-2", 200, "USD"),
			demoFormula(t, "ZC00000001", "CL-5", 500, "RUB"),
			demoFormula(t, "ZC00000002", "CL-5", 500, "RUB"),
			demoFormula(t, "ZD00000001", "CL-6", 600, "RUB"),
		},
		Components: components,
		CurrencyRates: []domain.CurrencyRate{
			{Currency: "USD", CalDay: date(t, "2026-06-01"), VersionType: "Факт", Value: 90},
		},
	}
	state.Given.Period = "2026-06"

	return state
}

func ArrangeUnknownPeriod(t *testing.T, state State) State {
	t.Helper()

	state.Given.Period = "2031-01"

	return state
}

func ArrangeManualPrice(price float64) func(*testing.T, State) State {
	return func(t *testing.T, state State) State {
		t.Helper()

		state.Given.Price = price

		return state
	}
}

func ArrangeSelectedFormula(formulaID string) func(*testing.T, State) State {
	return func(t *testing.T, state State) State {
		t.Helper()

		state.Given.Formula = formulaID

		return state
	}
}

// ── act ──

func ActLoadSources(t *testing.T, deps Deps, state State) State {
	t.Helper()

	deps.Loader.src = state.Given.Sources

	return state
}

// ── assert ──

func AssertNoError(t *testing.T, state State) {
	t.Helper()

	require.NoError(t, state.Response.Err)
}

func AssertErrorIs(target error) func(*testing.T, State) {
	return func(t *testing.T, state State) {
		t.Helper()

		require.Error(t, state.Response.Err)
		assert.ErrorIs(t, state.Response.Err, target)
	}
}

func AssertCalculationDone(t *testing.T, state State) {
	t.Helper()

	assert.Equal(t, "done", state.Response.Info.Status)
	assert.Equal(t, "2026-06", state.Response.Info.Period)
	// 6 строк периода; строка другого периода не входит
	assert.Equal(t, 6, state.Response.Info.Total)
	assert.NotEmpty(t, state.Response.Info.ID)
}

func AssertRowsAllStatus(status domain.Status) func(*testing.T, State) {
	return func(t *testing.T, state State) {
		t.Helper()

		require.NotEmpty(t, state.Response.Page.Items)
		for _, row := range state.Response.Page.Items {
			assert.Equal(t, status, row.Status)
		}
	}
}

func AssertStatusCountsComplete(t *testing.T, state State) {
	t.Helper()

	counts := state.Response.Page.StatusCounts
	assert.Equal(t, 2, counts[domain.StatusCalculated])
	assert.Equal(t, 1, counts[domain.StatusSpotNotCalculated])
	assert.Equal(t, 1, counts[domain.StatusFormulaNotFound])
	assert.Equal(t, 1, counts[domain.StatusFormulaConflict])
	assert.Equal(t, 1, counts[domain.StatusComponentError])
}

func AssertSearchResult(t *testing.T, state State) {
	t.Helper()

	require.Len(t, state.Response.Page.Items, 1)
	assert.Equal(t, "Клиент CL-2", state.Response.Page.Items[0].ClientName)
}

func AssertSortedByPriceDesc(t *testing.T, state State) {
	t.Helper()

	items := state.Response.Page.Items
	require.NotEmpty(t, items)
	var previous *float64
	for _, row := range items {
		if row.Price == nil {
			continue
		}
		if previous != nil {
			assert.LessOrEqual(t, *row.Price, *previous)
		}
		previous = row.Price
	}
}

func AssertCalculatedDetails(t *testing.T, state State) {
	t.Helper()

	details := state.Response.Details
	require.NotNil(t, details.Applied)
	assert.Equal(t, "ZA00000001", details.Applied.FormulaID)
	assert.Len(t, details.Components, 2)
	assert.Len(t, details.Alternatives, 1)
	require.NotNil(t, details.Row.Price)
	assert.InDelta(t, 1500, *details.Row.Price, 1e-9)
}

func AssertRowStatusIs(status domain.Status) func(*testing.T, State) {
	return func(t *testing.T, state State) {
		t.Helper()

		assert.Equal(t, status, state.Response.Details.Row.Status)
	}
}

func AssertManualPriceApplied(t *testing.T, state State) {
	t.Helper()

	require.NotNil(t, state.Response.Details.Row.Price)
	assert.InDelta(t, state.Given.Price, *state.Response.Details.Row.Price, 1e-9)
	require.NotNil(t, state.Response.Details.ManualPrice)
}

func AssertSelectedFormulaApplied(t *testing.T, state State) {
	t.Helper()

	details := state.Response.Details
	require.NotNil(t, details.Applied)
	assert.Equal(t, "ZC00000002", details.Applied.FormulaID)
	require.NotNil(t, details.Row.Price)
	assert.InDelta(t, 1080, *details.Row.Price, 1e-9)
	assert.False(t, details.Row.RequiresReview)
	assert.Equal(t, domain.SelectionUserSelected, details.Applied.SelectionReason)
}

func AssertCanonicalKpi(t *testing.T, state State) {
	t.Helper()

	kpi := state.Response.Kpi
	// 5 Formula-строк (SPOT исключён), с кандидатами 4 → 80%
	assert.Equal(t, 80, kpi.FormulaCoveragePct)
	// формулы: ZA,ZB,ZC1,ZC2 ok; ZD с ошибкой → 4/5 = 80%
	assert.Equal(t, 80, kpi.FormulasOkPct)
	// одна строка с формулой без цены (component_error)
	assert.Equal(t, 1, kpi.CalcErrorRows)
	// 1500*10*1 + 100*2*90 + 1500*1*1 = 34500 → 0.0345 млн
	assert.InDelta(t, 0.0345, kpi.ControlSumMln, 1e-9)
	// все ошибки классифицированы
	assert.Equal(t, 0, kpi.UnclassifiedErrorPct)
}

func AssertControlSumGrew(t *testing.T, state State) {
	t.Helper()

	// 34500 + 100000*5 = 534500 → 0.5345 млн
	assert.InDelta(t, 0.5345, state.Response.Kpi.ControlSumMln, 1e-9)
}

func AssertPartJoined(t *testing.T, state State) {
	t.Helper()

	assert.Equal(t, "joined", state.Response.Part.Status)
	require.NotEmpty(t, state.Response.Doc.Parts)
	assert.Equal(t, "joined", state.Response.Doc.Parts[0].Status)
}

func AssertConsolidatedRows(t *testing.T, state State) {
	t.Helper()

	assert.Equal(t, "2026-06", state.Response.Doc.Period)
	assert.Len(t, state.Response.Doc.Rows, 6)
	assert.Equal(t, 6, state.Response.Doc.TotalRows)
}
