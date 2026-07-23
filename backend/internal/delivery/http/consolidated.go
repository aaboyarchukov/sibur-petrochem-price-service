package http

import (
	"context"
	"errors"

	"sibur-petrochem-price-service/internal/domain"

	api "sibur-petrochem-price-service/internal/generated/api"
)

func (h *Handler) SubmitCalculationPart(_ context.Context, request api.SubmitCalculationPartRequestObject) (api.SubmitCalculationPartResponseObject, error) {
	part, err := h.calcs.Submit(request.CalculationID.String())
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrAlreadySubmitted):
			return api.SubmitCalculationPart409JSONResponse(apiError("already_submitted", err.Error())), nil
		case errors.Is(err, domain.ErrCalculationNotFound):
			return api.SubmitCalculationPart404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse(apiError("not_found", err.Error())),
			}, nil
		}

		return nil, err
	}

	return api.SubmitCalculationPart200JSONResponse(mapPart(part)), nil
}

func (h *Handler) GetConsolidatedDocument(_ context.Context, request api.GetConsolidatedDocumentRequestObject) (api.GetConsolidatedDocumentResponseObject, error) {
	doc, err := h.calcs.Consolidated(request.Period)
	if err != nil {
		return nil, err
	}

	out := api.ConsolidatedDocument{
		Period:    doc.Period,
		Parts:     make([]api.ConsolidatedPart, 0, len(doc.Parts)),
		KPI:       mapKpi(doc.Kpi),
		Rows:      make([]api.ConsolidatedRow, 0, len(doc.Rows)),
		TotalRows: doc.TotalRows,
	}
	for _, part := range doc.Parts {
		out.Parts = append(out.Parts, mapPart(part))
	}

	rows := paginateConsolidated(doc.Rows, request.Params.Offset, request.Params.Limit)
	for _, row := range rows {
		isDraft := row.IsDraft
		out.Rows = append(out.Rows, api.ConsolidatedRow{
			AnalystName: row.AnalystName,
			IsDraft:     &isDraft,
			Row:         mapRow(row.Row),
		})
	}

	return api.GetConsolidatedDocument200JSONResponse(out), nil
}

func paginateConsolidated[T any](rows []T, offsetParam, limitParam *int) []T {
	offset := 0
	if offsetParam != nil {
		offset = *offsetParam
	}
	if offset >= len(rows) {
		return nil
	}

	end := len(rows)
	if limitParam != nil && *limitParam > 0 && offset+*limitParam < end {
		end = offset + *limitParam
	}

	return rows[offset:end]
}
