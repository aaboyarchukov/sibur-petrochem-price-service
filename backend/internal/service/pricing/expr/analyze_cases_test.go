package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluator_Analyze(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("extracts variables in order of appearance and functions", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeAnalyzeValid,
			).Then(
				AssertAnalysisValid,
			)

			tc.State.Response.Analysis = tc.SUT.Analyze(tc.State.Given.Expression)
		})
	})

	t.Run("should be able to not be able", func(t *testing.T) {
		t.Run("reports syntax error with position", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeAnalyzeInvalid,
			).Then(
				AssertAnalysisInvalid,
			)

			tc.State.Response.Analysis = tc.SUT.Analyze(tc.State.Given.Expression)
		})

		t.Run("reports forbidden function as error", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeAnalyzeForbiddenFunction,
			).Then(
				AssertAnalysisInvalid,
			)

			tc.State.Response.Analysis = tc.SUT.Analyze(tc.State.Given.Expression)
		})
	})
}

// ── arrange ──

func ArrangeAnalyzeValid(t *testing.T, state State) State {
	t.Helper()

	state.Given.Expression = "IF ( ( ( CPT_MOSCOW_109 - L ) / H1 ) * D < SPOT , ( ( CPT_MOSCOW_109 - L ) / H1 ) * D , SPOT )"
	state.Expect.Variables = []string{"CPT_MOSCOW_109", "L", "H1", "D", "SPOT"}
	state.Expect.Functions = []string{"IF"}
	state.Expect.Valid = true

	return state
}

func ArrangeAnalyzeInvalid(t *testing.T, state State) State {
	t.Helper()

	state.Given.Expression = "( A - L ) / / H1"
	state.Expect.Valid = false

	return state
}

func ArrangeAnalyzeForbiddenFunction(t *testing.T, state State) State {
	t.Helper()

	state.Given.Expression = "EVIL ( A )"
	state.Expect.Valid = false

	return state
}

// ── assert ──

func AssertAnalysisValid(t *testing.T, state State) {
	t.Helper()

	analysis := state.Response.Analysis
	require.True(t, analysis.Valid)
	assert.Empty(t, analysis.Errors)
	assert.Equal(t, state.Expect.Variables, analysis.Variables)
	assert.Equal(t, state.Expect.Functions, analysis.Functions)
}

func AssertAnalysisInvalid(t *testing.T, state State) {
	t.Helper()

	analysis := state.Response.Analysis
	require.False(t, analysis.Valid)
	require.NotEmpty(t, analysis.Errors)
	assert.NotEmpty(t, analysis.Errors[0].Message)
}
