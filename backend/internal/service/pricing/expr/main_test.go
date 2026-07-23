package expr

import (
	"testing"

	"github.com/godepo/groat"
	"github.com/jaswdr/faker/v2"
)

type (
	Givens struct {
		Expression string
		Vars       map[string]float64
	}

	Expects struct {
		Value     float64
		ErrSubstr string
		Variables []string
		Functions []string
		Valid     bool
	}

	Responses struct {
		Value    float64
		Err      error
		Analysis AnalyzeResult
	}

	Deps struct{}

	State struct {
		Given    Givens
		Faker    faker.Faker
		Response Responses
		Expect   Expects
	}
)

func newTestContext(t *testing.T) *groat.Case[Deps, State, Evaluator] {
	t.Helper()

	tc := groat.New[Deps, State, Evaluator](t, func(t *testing.T, deps Deps) Evaluator {
		t.Helper()

		return New()
	})

	tc.Go()

	tc.State.Faker = faker.New()

	return tc
}
