package pricing

import (
	"testing"
	"time"

	"github.com/godepo/groat"
	"github.com/jaswdr/faker/v2"

	"sibur-petrochem-price-service/internal/domain"
)

type (
	Givens struct {
		Sources   domain.Sources
		Period    time.Time
		Horizon   time.Time
		Component domain.FormulaComponent
		Currency  string
		FormulaID string
	}

	Expects struct {
		Value       float64
		VersionType string
		QuoteCode   int64
		ErrSubstr   string
		Warning     string
	}

	Responses struct {
		Value   float64
		Meta    domain.ComponentValue
		Values  map[string]float64
		Details []domain.ComponentValue
		Errors  []string
		Err     error

		Result []domain.CandidateResult
		Rows   []domain.CalcRow
	}

	Deps struct{}

	State struct {
		Given    Givens
		Faker    faker.Faker
		Response Responses
		Expect   Expects
	}
)

func newTestContext(t *testing.T) *groat.Case[Deps, State, *Engine] {
	t.Helper()

	tc := groat.New[Deps, State, *Engine](t, func(t *testing.T, deps Deps) *Engine {
		t.Helper()

		return NewEngine()
	})

	tc.Go()

	tc.State.Faker = faker.New()

	return tc
}

func date(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("bad date literal %q: %v", value, err)
	}

	return parsed
}

func ptr[T any](value T) *T {
	return &value
}
