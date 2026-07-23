package calculations

import (
	"context"
	"testing"
	"time"

	"github.com/godepo/groat"
	"github.com/jaswdr/faker/v2"

	"sibur-petrochem-price-service/internal/domain"
	"sibur-petrochem-price-service/internal/service/pricing"
)

type (
	fakeLoader struct {
		src domain.Sources
		err error
	}

	Givens struct {
		Sources domain.Sources
		Period  string
		Price   float64
		Formula string
	}

	Expects struct {
		Value     float64
		Status    domain.Status
		ErrTarget error
		FormulaID string
		Count     int
	}

	Responses struct {
		Info    Info
		Kpi     Kpi
		Page    RowsPage
		Details Details
		Doc     Consolidated
		Part    Part
		Err     error
	}

	Deps struct {
		Loader *fakeLoader
	}

	State struct {
		Given    Givens
		Faker    faker.Faker
		Response Responses
		Expect   Expects
	}
)

func (f *fakeLoader) LoadSources(_ context.Context) (domain.Sources, error) {
	return f.src, f.err
}

func newTestContext(t *testing.T) *groat.Case[Deps, State, *Service] {
	t.Helper()

	tc := groat.New[Deps, State, *Service](t, func(t *testing.T, deps Deps) *Service {
		t.Helper()

		return New(deps.Loader, pricing.NewEngine())
	}, func(t *testing.T, deps Deps) Deps {
		t.Helper()

		deps.Loader = &fakeLoader{}

		return deps
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
