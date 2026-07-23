package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Контрольные значения посчитаны эталонным evaluate_formula (pricing_pipeline_fixed.py)
// на реальных выражениях из formulas.csv — фиксируют эквивалентность Go-порта.
func TestEvaluator_ReferenceEquivalence(t *testing.T) {
	t.Run("should be able to be able", func(t *testing.T) {
		tests := map[string]struct {
			expression string
			vars       map[string]float64
			want       float64
		}{
			"cny formula with vat and currency symbol variable": {
				expression: "( ( CFR_CHINA - DISCOUNT ) * DUTY * CUR_$_¥_PBC * VAT + PREMIUM ) / VAT",
				vars: map[string]float64{
					"CFR_CHINA": 980.5, "DISCOUNT": 45.2, "DUTY": 1.065,
					"CUR_$_¥_PBC": 7.1832, "VAT": 1.13, "PREMIUM": 120.0,
				},
				want: 7261.340702665486,
			},
			"digit-leading variables": {
				expression: "( SCI / 1_13 / 1_02 - 1_1 ) * D",
				vars: map[string]float64{
					"SCI": 1130.77, "1_13": 1.13, "1_02": 1.02, "1_1": 1.1, "D": 0.9235,
				},
				want: 904.9932555006073,
			},
			"if with false condition returns spot": {
				expression: "IF ( ( ( CPT_MOSCOW_109 - L ) / H1 ) * D < SPOT , ( ( CPT_MOSCOW_109 - L ) / H1 ) * D , SPOT )",
				vars: map[string]float64{
					"CPT_MOSCOW_109": 101250.0, "L": 3650.0, "H1": 0.8906, "D": 0.7154, "SPOT": 98728.12,
				},
				want: 78400.0,
			},
			"rnd_x half-up to two digits": {
				expression: "RND_X ( ( QUOTE + MARGIN ) * RATE , 2 )",
				vars:       map[string]float64{"QUOTE": 755.125, "MARGIN": 33.333, "RATE": 89.4215},
				want:       70505.1,
			},
			"min max with python mod precedence": {
				expression: "MAX ( MIN ( A , B ) , C ) - - D ** 2 % 7",
				vars:       map[string]float64{"A": 15.5, "B": 22.1, "C": 18.75, "D": 3.0},
				want:       13.75,
			},
		}

		sut := New()
		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				got, err := sut.Evaluate(tc.expression, tc.vars)

				require.NoError(t, err)
				assert.InDelta(t, tc.want, got, 1e-9)
			})
		}
	})
}
