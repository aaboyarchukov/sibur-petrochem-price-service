package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluator_Evaluate(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		t.Run("evaluates reference IF expression", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeReferenceIfExpression,
			).Then(
				AssertNoError,
				AssertValue,
			)

			tc.State.Response.Value, tc.State.Response.Err = tc.SUT.Evaluate(
				tc.State.Given.Expression, tc.State.Given.Vars,
			)
		})

		t.Run("evaluates arithmetic with power, mod and unary minus", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeArithmeticExpression,
			).Then(
				AssertNoError,
				AssertValue,
			)

			tc.State.Response.Value, tc.State.Response.Err = tc.SUT.Evaluate(
				tc.State.Given.Expression, tc.State.Given.Vars,
			)
		})

		t.Run("rounds half-up via RND_X like reference", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeRoundingExpression,
			).Then(
				AssertNoError,
				AssertValue,
			)

			tc.State.Response.Value, tc.State.Response.Err = tc.SUT.Evaluate(
				tc.State.Given.Expression, tc.State.Given.Vars,
			)
		})

		t.Run("supports currency symbols and digit-leading variable names", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeExoticVariableNames,
			).Then(
				AssertNoError,
				AssertValue,
			)

			tc.State.Response.Value, tc.State.Response.Err = tc.SUT.Evaluate(
				tc.State.Given.Expression, tc.State.Given.Vars,
			)
		})

		t.Run("evaluates MIN and MAX", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeMinMaxExpression,
			).Then(
				AssertNoError,
				AssertValue,
			)

			tc.State.Response.Value, tc.State.Response.Err = tc.SUT.Evaluate(
				tc.State.Given.Expression, tc.State.Given.Vars,
			)
		})
	})

	t.Run("should be able to not be able", func(t *testing.T) {
		t.Run("fails on unknown variable", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeUnknownVariable,
			).Then(
				AssertEvalError,
			)

			tc.State.Response.Value, tc.State.Response.Err = tc.SUT.Evaluate(
				tc.State.Given.Expression, tc.State.Given.Vars,
			)
		})

		t.Run("fails on forbidden function", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeForbiddenFunction,
			).Then(
				AssertEvalError,
			)

			tc.State.Response.Value, tc.State.Response.Err = tc.SUT.Evaluate(
				tc.State.Given.Expression, tc.State.Given.Vars,
			)
		})

		t.Run("fails on syntax error", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeSyntaxError,
			).Then(
				AssertEvalError,
			)

			tc.State.Response.Value, tc.State.Response.Err = tc.SUT.Evaluate(
				tc.State.Given.Expression, tc.State.Given.Vars,
			)
		})

		t.Run("fails when result is boolean", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeBooleanResult,
			).Then(
				AssertEvalError,
			)

			tc.State.Response.Value, tc.State.Response.Err = tc.SUT.Evaluate(
				tc.State.Given.Expression, tc.State.Given.Vars,
			)
		})

		t.Run("fails on division by zero", func(t *testing.T) {
			tc := newTestContext(t)

			tc.Given(
				ArrangeDivisionByZero,
			).Then(
				AssertEvalError,
			)

			tc.State.Response.Value, tc.State.Response.Err = tc.SUT.Evaluate(
				tc.State.Given.Expression, tc.State.Given.Vars,
			)
		})
	})
}

// ── arrange ──

func ArrangeReferenceIfExpression(t *testing.T, state State) State {
	t.Helper()

	// Эталонная формула из formulas.csv: условие ложно → берётся SPOT.
	state.Given.Expression = "IF ( ( ( CPT_MOSCOW_109 - L ) / H1 ) * D < SPOT , ( ( CPT_MOSCOW_109 - L ) / H1 ) * D , SPOT )"
	state.Given.Vars = map[string]float64{
		"CPT_MOSCOW_109": 100000,
		"L":              3650,
		"H1":             0.8906,
		"D":              0.7154,
		"SPOT":           50000,
	}
	// (100000-3650)/0.8906*0.7154 = 77398.35... > 50000 → SPOT
	state.Expect.Value = 50000

	return state
}

func ArrangeArithmeticExpression(t *testing.T, state State) State {
	t.Helper()

	state.Given.Expression = "- A ** 2 + 7 % 3 * ( B - 1 )"
	state.Given.Vars = map[string]float64{"A": 3, "B": 4}
	// как в python: -(3**2) + (7%3)*(4-1) = -9 + 3 = -6
	state.Expect.Value = -6

	return state
}

func ArrangeRoundingExpression(t *testing.T, state State) State {
	t.Helper()

	// ROUND_HALF_UP: 2.5 -> 3; 1.005 с двумя знаками -> 1.01 (как Decimal(str(v)))
	state.Given.Expression = "RND_X ( 2.5 ) + RND_X ( 1.005 , 2 )"
	state.Given.Vars = map[string]float64{}
	state.Expect.Value = 3 + 1.01

	return state
}

func ArrangeExoticVariableNames(t *testing.T, state State) State {
	t.Helper()

	state.Given.Expression = "( SCI / 1_13 / 1_02 - 1_1 ) * D + CUR_$_¥_PBC"
	state.Given.Vars = map[string]float64{
		"SCI":         1130,
		"1_13":        1.13,
		"1_02":        1.02,
		"1_1":         1.1,
		"D":           2,
		"CUR_$_¥_PBC": 7.2,
	}
	// (1130/1.13/1.02 - 1.1)*2 + 7.2
	state.Expect.Value = (1130/1.13/1.02-1.1)*2 + 7.2

	return state
}

func ArrangeMinMaxExpression(t *testing.T, state State) State {
	t.Helper()

	state.Given.Expression = "MIN ( A , B , 10 ) + MAX ( A , B )"
	state.Given.Vars = map[string]float64{"A": 25, "B": 4}
	state.Expect.Value = 4 + 25

	return state
}

func ArrangeUnknownVariable(t *testing.T, state State) State {
	t.Helper()

	state.Given.Expression = "KNOWN + UNKNOWN_VAR"
	state.Given.Vars = map[string]float64{"KNOWN": state.Faker.Float64(2, 1, 100)}
	state.Expect.ErrSubstr = "UNKNOWN_VAR"

	return state
}

func ArrangeForbiddenFunction(t *testing.T, state State) State {
	t.Helper()

	state.Given.Expression = "EVIL ( 1 )"
	state.Given.Vars = map[string]float64{}
	state.Expect.ErrSubstr = "функция"

	return state
}

func ArrangeSyntaxError(t *testing.T, state State) State {
	t.Helper()

	state.Given.Expression = "( A - L ) / / H1"
	state.Given.Vars = map[string]float64{"A": 1, "L": 1, "H1": 1}
	state.Expect.ErrSubstr = "Синтаксическая ошибка"

	return state
}

func ArrangeBooleanResult(t *testing.T, state State) State {
	t.Helper()

	state.Given.Expression = "A < B"
	state.Given.Vars = map[string]float64{"A": 1, "B": 2}
	state.Expect.ErrSubstr = "некорректное значение"

	return state
}

func ArrangeDivisionByZero(t *testing.T, state State) State {
	t.Helper()

	state.Given.Expression = "A / ( B - B )"
	state.Given.Vars = map[string]float64{"A": 1, "B": state.Faker.Float64(2, 1, 100)}
	state.Expect.ErrSubstr = "делени"

	return state
}

// ── assert ──

func AssertNoError(t *testing.T, state State) {
	t.Helper()

	require.NoError(t, state.Response.Err)
}

func AssertValue(t *testing.T, state State) {
	t.Helper()

	assert.InDelta(t, state.Expect.Value, state.Response.Value, 1e-9)
}

func AssertEvalError(t *testing.T, state State) {
	t.Helper()

	require.Error(t, state.Response.Err)
	assert.Contains(t, state.Response.Err.Error(), state.Expect.ErrSubstr)
}
