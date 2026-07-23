package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/xuri/excelize/v2"

	"sibur-petrochem-price-service/internal/domain"

	api "sibur-petrochem-price-service/internal/generated/api"
)

var exportColumns = []string{
	"row_id", "period", "client_id", "client_name", "material_id", "material_name",
	"contract", "currency", "forecast", "status", "price", "warning", "error",
}

func (h *Handler) ExportCalculation(_ context.Context, request api.ExportCalculationRequestObject) (api.ExportCalculationResponseObject, error) {
	rows, err := h.calcs.ExportRows(request.CalculationID.String())
	if err != nil {
		if errors.Is(err, domain.ErrCalculationNotFound) {
			return api.ExportCalculation404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse(apiError("not_found", err.Error())),
			}, nil
		}

		return nil, err
	}

	payload, err := buildWorkbook("prices", exportColumns, rowsToCells(rows, nil))
	if err != nil {
		return nil, err
	}

	return api.ExportCalculation200ApplicationVndOpenxmlformatsOfficedocumentSpreadsheetmlSheetResponse{
		Body:          bytes.NewReader(payload),
		ContentLength: int64(len(payload)),
	}, nil
}

func (h *Handler) ExportConsolidatedDocument(_ context.Context, request api.ExportConsolidatedDocumentRequestObject) (api.ExportConsolidatedDocumentResponseObject, error) {
	doc, err := h.calcs.Consolidated(request.Period)
	if err != nil {
		return nil, err
	}

	// В выгрузку входят только присоединённые участки.
	var rows []domain.CalcRow
	var analysts []string
	for _, row := range doc.Rows {
		if row.IsDraft {
			continue
		}
		rows = append(rows, row.Row)
		analysts = append(analysts, row.AnalystName)
	}

	columns := append([]string{"analyst"}, exportColumns...)
	payload, err := buildWorkbook("consolidated", columns, rowsToCells(rows, analysts))
	if err != nil {
		return nil, err
	}

	return api.ExportConsolidatedDocument200ApplicationVndOpenxmlformatsOfficedocumentSpreadsheetmlSheetResponse{
		Body:          bytes.NewReader(payload),
		ContentLength: int64(len(payload)),
	}, nil
}

// rowsToCells — строки расчёта в ячейки листа; analysts добавляет первый столбец.
func rowsToCells(rows []domain.CalcRow, analysts []string) [][]any {
	out := make([][]any, 0, len(rows))
	for i, row := range rows {
		cells := []any{
			strconv.FormatInt(row.RowID, 10), row.Period.Format("2006-01"),
			row.ClientID, row.ClientName,
			strconv.FormatInt(row.MaterialID, 10), row.MaterialName,
			row.Contract, row.Currency,
			floatCell(row.Forecast), string(row.Status), floatCell(row.Price),
			row.Warning, row.Error,
		}
		if analysts != nil {
			cells = append([]any{analysts[i]}, cells...)
		}
		out = append(out, cells)
	}

	return out
}

func floatCell(value *float64) any {
	if value == nil {
		return ""
	}

	return *value
}

func buildWorkbook(sheet string, columns []string, rows [][]any) ([]byte, error) {
	book := excelize.NewFile()
	defer func() { _ = book.Close() }()

	const defaultSheet = "Sheet1"
	if err := book.SetSheetName(defaultSheet, sheet); err != nil {
		return nil, fmt.Errorf("rename sheet: %w", err)
	}

	header := make([]any, 0, len(columns))
	for _, column := range columns {
		header = append(header, column)
	}
	if err := book.SetSheetRow(sheet, "A1", &header); err != nil {
		return nil, fmt.Errorf("write header: %w", err)
	}

	const dataStartRow = 2 // строки данных начинаются после заголовка
	for i, cells := range rows {
		cell, err := excelize.CoordinatesToCellName(1, i+dataStartRow)
		if err != nil {
			return nil, fmt.Errorf("cell name: %w", err)
		}
		if err := book.SetSheetRow(sheet, cell, &cells); err != nil {
			return nil, fmt.Errorf("write row: %w", err)
		}
	}

	var buffer bytes.Buffer
	if err := book.Write(&buffer); err != nil {
		return nil, fmt.Errorf("write workbook: %w", err)
	}

	return buffer.Bytes(), nil
}
