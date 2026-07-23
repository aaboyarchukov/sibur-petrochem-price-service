package http

import (
	"encoding/json"
	"errors"

	"sibur-petrochem-price-service/internal/domain"
	"sibur-petrochem-price-service/internal/service/calculations"

	nethttp "net/http"
	api "sibur-petrochem-price-service/internal/generated/api"
)

// NewRouter — собранный http.Handler: strict-server + маршруты контракта под /api/v1.
func NewRouter(calcs *calculations.Service, sources SourcesProvider) nethttp.Handler {
	strict := api.NewStrictHandlerWithOptions(New(calcs, sources), nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  requestErrorHandler,
		ResponseErrorHandlerFunc: responseErrorHandler,
	})

	mux := nethttp.NewServeMux()
	// presence — вне strict-сервера (см. PresenceHub); в контракте исключён из кодогена
	mux.Handle("GET /api/v1/presence/events", NewPresenceHub())

	return api.HandlerWithOptions(strict, api.StdHTTPServerOptions{
		BaseURL:          "/api/v1",
		BaseRouter:       mux,
		ErrorHandlerFunc: requestErrorHandler,
	})
}

// requestErrorHandler — ошибки разбора запроса (невалидное тело/параметры).
func requestErrorHandler(w nethttp.ResponseWriter, _ *nethttp.Request, err error) {
	writeError(w, nethttp.StatusBadRequest, "bad_request", err.Error())
}

// responseErrorHandler — ошибки хендлеров, не покрытые типизированными ответами.
func responseErrorHandler(w nethttp.ResponseWriter, _ *nethttp.Request, err error) {
	status := nethttp.StatusInternalServerError
	code := "internal_error"

	switch {
	case errors.Is(err, domain.ErrNotImplemented):
		status = nethttp.StatusNotImplemented
		code = "not_implemented"
	case errors.Is(err, domain.ErrCalculationNotFound), errors.Is(err, domain.ErrRowNotFound),
		errors.Is(err, domain.ErrSourceNotFound):
		status = nethttp.StatusNotFound
		code = "not_found"
	}

	writeError(w, status, code, err.Error())
}

func writeError(w nethttp.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiError(code, message))
}
