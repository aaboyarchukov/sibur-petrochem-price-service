package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/godepo/groat"
	"github.com/jaswdr/faker/v2"
	"github.com/stretchr/testify/require"

	"sibur-petrochem-price-service/internal/domain"
	"sibur-petrochem-price-service/internal/service/calculations"
	"sibur-petrochem-price-service/internal/service/pricing"

	nethttp "net/http"
)

type (
	fakeSources struct {
		src    domain.Sources
		counts map[string]int64
	}

	Givens struct {
		Sources domain.Sources
		Period  string
		Body    map[string]any
	}

	Expects struct {
		Code      int
		Substr    string
		FormulaID string
	}

	Responses struct {
		Code int
		Body map[string]any
		Raw  []byte
	}

	Deps struct {
		Sources *fakeSources
	}

	State struct {
		Given    Givens
		Faker    faker.Faker
		Response Responses
		Expect   Expects
	}
)

func (f *fakeSources) LoadSources(_ context.Context) (domain.Sources, error) {
	return f.src, nil
}

func (f *fakeSources) SourceCounts(_ context.Context) (map[string]int64, error) {
	return f.counts, nil
}

// SourceFacets — уникальные продукты/клиенты и границы горизонта из fake-ssp.
func (f *fakeSources) SourceFacets(_ context.Context) (domain.SourceFacets, error) {
	products := map[int64]string{}
	clients := map[string]string{}
	var min, max string
	for _, row := range f.src.Ssp {
		products[row.MaterialID] = row.MaterialName
		clients[row.ClientID] = row.ClientName
		month := row.Period.Format("2006-01")
		if min == "" || month < min {
			min = month
		}
		if month > max {
			max = month
		}
	}

	facets := domain.SourceFacets{PeriodMin: min, PeriodMax: max}
	for id, name := range products {
		facets.Products = append(facets.Products, domain.ProductFacet{ID: id, Name: name})
	}
	for id, name := range clients {
		facets.Clients = append(facets.Clients, domain.ClientFacet{ID: id, Name: name})
	}

	return facets, nil
}

func (f *fakeSources) ReplaceSsp(_ context.Context, rows []domain.SspRow) error {
	f.src.Ssp = rows
	f.counts["ssp"] = int64(len(rows))

	return nil
}

func (f *fakeSources) ReplaceFormulas(_ context.Context, rows []domain.Formula) error {
	f.src.Formulas = rows
	f.counts["formulas"] = int64(len(rows))

	return nil
}

func newTestContext(t *testing.T) *groat.Case[Deps, State, nethttp.Handler] {
	t.Helper()

	tc := groat.New[Deps, State, nethttp.Handler](t, func(t *testing.T, deps Deps) nethttp.Handler {
		t.Helper()

		service := calculations.New(deps.Sources, pricing.NewEngine())

		return NewRouter(service, deps.Sources)
	}, func(t *testing.T, deps Deps) Deps {
		t.Helper()

		deps.Sources = &fakeSources{counts: map[string]int64{}}

		return deps
	})

	tc.Go()

	tc.State.Faker = faker.New()

	return tc
}

// do — выполнить запрос к SUT и разобрать JSON-ответ в state.
func do(t *testing.T, sut nethttp.Handler, method, path string, body any) (int, map[string]any, []byte) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(encoded)
	}

	request := httptest.NewRequest(method, path, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	recorder := httptest.NewRecorder()
	sut.ServeHTTP(recorder, request)

	raw := recorder.Body.Bytes()
	parsed := map[string]any{}
	_ = json.Unmarshal(raw, &parsed)

	return recorder.Code, parsed, raw
}

// createCalculation — хелпер: активировать источники (демо) и создать расчёт через API.
func createCalculation(t *testing.T, sut nethttp.Handler, period string) string {
	t.Helper()

	demoCode, _, _ := do(t, sut, nethttp.MethodPost, "/api/v1/sources/demo", nil)
	require.Equal(t, nethttp.StatusOK, demoCode)

	code, body, _ := do(t, sut, nethttp.MethodPost, "/api/v1/calculations", calcBody(period))
	require.Equal(t, nethttp.StatusAccepted, code)
	id, ok := body["id"].(string)
	require.True(t, ok, "в ответе должен быть id расчёта")

	return id
}

// calcBody — тело POST /calculations из строки периода: пусто/"all" → весь горизонт,
// "YYYY-MM" → диапазон из одного месяца.
func calcBody(period string) map[string]any {
	if period == "" || period == "all" {
		return map[string]any{}
	}

	return map[string]any{"period_from": period, "period_to": period}
}

func date(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse("2006-01-02", value)
	require.NoError(t, err)

	return parsed
}

func ptr[T any](value T) *T {
	return &value
}
