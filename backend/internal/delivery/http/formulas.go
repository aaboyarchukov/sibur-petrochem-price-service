package http

import (
	"context"

	"github.com/oapi-codegen/nullable"

	api "sibur-petrochem-price-service/internal/generated/api"
)

func (h *Handler) ParseFormula(_ context.Context, request api.ParseFormulaRequestObject) (api.ParseFormulaResponseObject, error) {
	if request.Body == nil {
		return api.ParseFormula400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse(apiError("bad_request", "пустое тело запроса")),
		}, nil
	}

	analysis := h.eval.Analyze(request.Body.FormulaText)

	out := api.ParsedFormula{
		Valid:     analysis.Valid,
		Variables: analysis.Variables,
		Functions: make([]api.ParsedFormulaFunctions, 0, len(analysis.Functions)),
	}
	if out.Variables == nil {
		out.Variables = []string{}
	}
	for _, function := range analysis.Functions {
		out.Functions = append(out.Functions, api.ParsedFormulaFunctions(function))
	}

	parseErrors := make([]struct {
		Message  string                 `json:"message"`
		Position nullable.Nullable[int] `json:"position,omitempty"`
	}, 0, len(analysis.Errors))
	for _, parseError := range analysis.Errors {
		item := struct {
			Message  string                 `json:"message"`
			Position nullable.Nullable[int] `json:"position,omitempty"`
		}{Message: parseError.Message, Position: nullableOf(parseError.Position)}
		parseErrors = append(parseErrors, item)
	}
	out.Errors = &parseErrors

	return api.ParseFormula200JSONResponse(out), nil
}
