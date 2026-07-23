// Package http реализует HTTP-слой приложения поверх strict-server,
// сгенерированного oapi-codegen из api/openapi.yaml.
//
// Хендлеры разбиты по доменным областям (файлы sources.go, calculations.go,
// rows.go, formulas.go, consolidated.go); Handler реализует полный
// api.StrictServerInterface. Ручка не может вернуть статус вне контракта.
package http

import (
	"context"
	"sync"
	"time"

	"sibur-petrochem-price-service/internal/domain"
	"sibur-petrochem-price-service/internal/service/calculations"
	"sibur-petrochem-price-service/internal/service/pricing/expr"

	api "sibur-petrochem-price-service/internal/generated/api"
)

// SourcesProvider — доступ к источникам данных (repository/postgres).
type SourcesProvider interface {
	LoadSources(ctx context.Context) (domain.Sources, error)
	SourceCounts(ctx context.Context) (map[string]int64, error)
	ReplaceSsp(ctx context.Context, rows []domain.SspRow) error
	ReplaceFormulas(ctx context.Context, rows []domain.Formula) error
}

// Handler — корневой обработчик, реализует api.StrictServerInterface.
type Handler struct {
	calcs   *calculations.Service
	sources SourcesProvider
	eval    expr.Evaluator

	// метки времени успешных загрузок .xlsx; in-memory (MVP: рестарт = сброс)
	uploadedMu sync.Mutex
	uploaded   map[api.SourceKey]time.Time
}

func New(calcs *calculations.Service, sources SourcesProvider) *Handler {
	return &Handler{
		calcs:    calcs,
		sources:  sources,
		eval:     expr.New(),
		uploaded: map[api.SourceKey]time.Time{},
	}
}

func (h *Handler) markUploaded(key api.SourceKey) {
	h.uploadedMu.Lock()
	defer h.uploadedMu.Unlock()
	h.uploaded[key] = time.Now()
}

func (h *Handler) uploadTime(key api.SourceKey) (ts time.Time, ok bool) {
	h.uploadedMu.Lock()
	defer h.uploadedMu.Unlock()
	ts, ok = h.uploaded[key]

	return ts, ok
}

// sourcesReady — пользовательские источники загружены (файлами либо демо-набором).
func (h *Handler) sourcesReady() bool {
	_, sspOK := h.uploadTime(sourceKeySsp)
	_, formulasOK := h.uploadTime(sourceKeyFormulas)

	return sspOK && formulasOK
}

var _ api.StrictServerInterface = (*Handler)(nil)

// formulaVariables — переменные выражения применённой формулы (для applied_formula).
func (h *Handler) formulaVariables(details calculations.Details) []string {
	if details.Applied == nil {
		return nil
	}

	return h.eval.Analyze(details.Applied.FormulaText).Variables
}
