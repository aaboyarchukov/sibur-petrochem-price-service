package http

import (
	"context"
	"errors"

	"sibur-petrochem-price-service/internal/domain"
	"sibur-petrochem-price-service/internal/service/calculations"

	api "sibur-petrochem-price-service/internal/generated/api"
)

func (h *Handler) ListCalculationRows(_ context.Context, request api.ListCalculationRowsRequestObject) (api.ListCalculationRowsResponseObject, error) {
	const defaultLimit = 100

	query := calculations.RowsQuery{Limit: defaultLimit}
	params := request.Params
	if params.Status != nil {
		if status, ok := statusFromAPI[*params.Status]; ok {
			query.Status = &status
		}
	}
	if params.Query != nil {
		query.Query = *params.Query
	}
	if params.Sort != nil {
		query.Sort = string(*params.Sort)
	}
	if params.Order != nil {
		query.Order = string(*params.Order)
	}
	if params.Limit != nil {
		query.Limit = *params.Limit
	}
	if params.Offset != nil {
		query.Offset = *params.Offset
	}
	if params.OnlyFormulaErrors != nil {
		query.OnlyFormulaErrors = *params.OnlyFormulaErrors
	}

	page, err := h.calcs.Rows(request.CalculationID.String(), query)
	if err != nil {
		if errors.Is(err, domain.ErrCalculationNotFound) {
			return api.ListCalculationRows404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse(apiError("not_found", err.Error())),
			}, nil
		}

		return nil, err
	}

	out := api.RowsPage{
		Items:        make([]api.CalculationRow, 0, len(page.Items)),
		Total:        page.Total,
		StatusCounts: make(map[string]int, len(page.StatusCounts)),
	}
	for _, row := range page.Items {
		out.Items = append(out.Items, mapRow(row))
	}
	for status, count := range page.StatusCounts {
		out.StatusCounts[string(statusToAPI[status])] = count
	}

	return api.ListCalculationRows200JSONResponse(out), nil
}

func (h *Handler) GetRowDetails(_ context.Context, request api.GetRowDetailsRequestObject) (api.GetRowDetailsResponseObject, error) {
	details, err := h.calcs.Details(request.CalculationID.String(), request.RowID)
	if err != nil {
		if errors.Is(err, domain.ErrCalculationNotFound) || errors.Is(err, domain.ErrRowNotFound) {
			return api.GetRowDetails404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse(apiError("not_found", err.Error())),
			}, nil
		}

		return nil, err
	}

	return api.GetRowDetails200JSONResponse(mapDetails(details, h.formulaVariables(details))), nil
}

func (h *Handler) SetManualPrice(_ context.Context, request api.SetManualPriceRequestObject) (api.SetManualPriceResponseObject, error) {
	if request.Body == nil {
		return api.SetManualPrice400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse(apiError("bad_request", "пустое тело запроса")),
		}, nil
	}

	details, err := h.calcs.SetManualPrice(request.CalculationID.String(), request.RowID, request.Body.Price)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidPrice):
			return api.SetManualPrice422JSONResponse(apiError("invalid_price", err.Error())), nil
		case errors.Is(err, domain.ErrCalculationNotFound), errors.Is(err, domain.ErrRowNotFound):
			return api.SetManualPrice404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse(apiError("not_found", err.Error())),
			}, nil
		}

		return nil, err
	}

	return api.SetManualPrice200JSONResponse(mapDetails(details, h.formulaVariables(details))), nil
}

func (h *Handler) ResetManualPrice(_ context.Context, request api.ResetManualPriceRequestObject) (api.ResetManualPriceResponseObject, error) {
	details, err := h.calcs.ResetManualPrice(request.CalculationID.String(), request.RowID)
	if err != nil {
		if errors.Is(err, domain.ErrCalculationNotFound) || errors.Is(err, domain.ErrRowNotFound) {
			return api.ResetManualPrice404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse(apiError("not_found", err.Error())),
			}, nil
		}

		return nil, err
	}

	return api.ResetManualPrice200JSONResponse(mapDetails(details, h.formulaVariables(details))), nil
}

func (h *Handler) SelectRowFormula(_ context.Context, request api.SelectRowFormulaRequestObject) (api.SelectRowFormulaResponseObject, error) {
	if request.Body == nil {
		return api.SelectRowFormula422JSONResponse(apiError("bad_request", "пустое тело запроса")), nil
	}

	details, err := h.calcs.SelectFormula(request.CalculationID.String(), request.RowID, request.Body.FormulaID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrFormulaNotAllowed):
			return api.SelectRowFormula422JSONResponse(apiError("formula_not_allowed", err.Error())), nil
		case errors.Is(err, domain.ErrCalculationNotFound), errors.Is(err, domain.ErrRowNotFound):
			return api.SelectRowFormula404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse(apiError("not_found", err.Error())),
			}, nil
		}

		return nil, err
	}

	return api.SelectRowFormula200JSONResponse(mapDetails(details, h.formulaVariables(details))), nil
}
