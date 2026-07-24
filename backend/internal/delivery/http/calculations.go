package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"sibur-petrochem-price-service/internal/domain"
	"sibur-petrochem-price-service/internal/service/calculations"

	api "sibur-petrochem-price-service/internal/generated/api"
)

func (h *Handler) CreateCalculation(ctx context.Context, request api.CreateCalculationRequestObject) (api.CreateCalculationResponseObject, error) {
	if request.Body == nil {
		return api.CreateCalculation400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse(apiError("bad_request", "пустое тело запроса")),
		}, nil
	}

	if !h.sourcesReady() {
		return api.CreateCalculation409JSONResponse(apiError("sources_not_loaded",
			"источники не загружены: загрузите ssp.xlsx и formulas.xlsx или активируйте демо-набор")), nil
	}

	info, err := h.calcs.Create(ctx, calcParams(request.Body))
	if err != nil {
		if errors.Is(err, domain.ErrSourcesNotLoaded) {
			return api.CreateCalculation409JSONResponse(apiError("sources_not_loaded", err.Error())), nil
		}

		return nil, err
	}

	return api.CreateCalculation202JSONResponse(mapInfo(info)), nil
}

// calcParams — тело запроса в параметры расчёта: nullable-границы диапазона и
// необязательные списки продуктов/клиентов (пустые = все).
func calcParams(body *api.CreateCalculationRequest) calculations.CalcParams {
	params := calculations.CalcParams{}
	if from, err := body.PeriodFrom.Get(); err == nil {
		params.PeriodFrom = &from
	}
	if to, err := body.PeriodTo.Get(); err == nil {
		params.PeriodTo = &to
	}
	if body.ProductIds != nil {
		params.ProductIDs = *body.ProductIds
	}
	if body.ClientIds != nil {
		params.ClientIDs = *body.ClientIds
	}

	return params
}

func (h *Handler) GetCalculation(_ context.Context, request api.GetCalculationRequestObject) (api.GetCalculationResponseObject, error) {
	info, err := h.calcs.Get(request.CalculationID.String())
	if err != nil {
		if errors.Is(err, domain.ErrCalculationNotFound) {
			return api.GetCalculation404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse(apiError("not_found", err.Error())),
			}, nil
		}

		return nil, err
	}

	return api.GetCalculation200JSONResponse(mapInfo(info)), nil
}

// StreamCalculationProgress — SSE-поток прогресса. Расчёт синхронный:
// подписчик получает snapshot-события (включая финальное done), после чего поток закрывается.
func (h *Handler) StreamCalculationProgress(_ context.Context, request api.StreamCalculationProgressRequestObject) (api.StreamCalculationProgressResponseObject, error) {
	events, unsubscribe, err := h.calcs.Subscribe(request.CalculationID.String())
	if err != nil {
		if errors.Is(err, domain.ErrCalculationNotFound) {
			return api.StreamCalculationProgress404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse(apiError("not_found", err.Error())),
			}, nil
		}

		return nil, err
	}
	defer unsubscribe()

	var buffer bytes.Buffer
	for event := range events {
		payload, err := json.Marshal(api.CalculationProgressEvent{
			Status:        api.CalculationStatus(event.Status),
			ProcessedRows: event.Processed,
			TotalRows:     event.Total,
			Percent:       event.Percent,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal progress event: %w", err)
		}
		// bytes.Buffer.Write* всегда возвращают nil error
		_, _ = buffer.WriteString("data: ")
		_, _ = buffer.Write(payload)
		_, _ = buffer.WriteString("\n\n")
	}

	return api.StreamCalculationProgress200TextEventStreamResponse{
		Body:          &buffer,
		ContentLength: int64(buffer.Len()),
	}, nil
}

func (h *Handler) GetCalculationKPI(_ context.Context, request api.GetCalculationKPIRequestObject) (api.GetCalculationKPIResponseObject, error) {
	kpi, err := h.calcs.Kpi(request.CalculationID.String())
	if err != nil {
		if errors.Is(err, domain.ErrCalculationNotFound) {
			return api.GetCalculationKPI404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse(apiError("not_found", err.Error())),
			}, nil
		}

		return nil, err
	}

	return api.GetCalculationKPI200JSONResponse(mapKpi(kpi)), nil
}
