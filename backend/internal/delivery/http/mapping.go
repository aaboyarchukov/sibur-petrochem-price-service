package http

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/oapi-codegen/nullable"

	"sibur-petrochem-price-service/internal/domain"
	"sibur-petrochem-price-service/internal/service/calculations"

	openapi_types "github.com/oapi-codegen/runtime/types"
	api "sibur-petrochem-price-service/internal/generated/api"
)

// Маппинг доменных моделей в контрактные (api/openapi.yaml).

var statusToAPI = map[domain.Status]api.RowStatus{
	domain.StatusCalculated:        "calculated",
	domain.StatusCalculatedExpired: "calculated_expired",
	domain.StatusFormulaConflict:   "formula_conflict",
	domain.StatusComponentError:    "component_error",
	domain.StatusInvalidFormula:    "invalid_formula",
	domain.StatusFormulaNotFound:   "no_formula",
	domain.StatusSpotNotCalculated: "spot_not_calculated",
	domain.StatusManual:            "manual",
}

var statusFromAPI = map[api.RowStatus]domain.Status{
	"calculated":          domain.StatusCalculated,
	"calculated_expired":  domain.StatusCalculatedExpired,
	"formula_conflict":    domain.StatusFormulaConflict,
	"component_error":     domain.StatusComponentError,
	"invalid_formula":     domain.StatusInvalidFormula,
	"no_formula":          domain.StatusFormulaNotFound,
	"spot_not_calculated": domain.StatusSpotNotCalculated,
	"manual":              domain.StatusManual,
}

var componentTypeToAPI = map[string]api.ComponentType{
	"1": "quote",
	"5": "currency_rate",
	"H": "constant",
	"A": "markup",
	"B": "logistics",
	"C": "quote_correction",
	"D": "discount",
	"E": "other",
	"7": "price_list",
	"0": "grouping",
	"6": "grouping",
}

var selectionReasonToAPI = map[domain.SelectionReason]api.SelectionReason{
	domain.SelectionActualSuccessful: "actual_successful",
	domain.SelectionLatestExpired:    "latest_expired_successful",
	domain.SelectionTieBreak:         "technical_tie_break",
	domain.SelectionNoSuccessful:     "no_successful",
	domain.SelectionUserSelected:     "user_selected",
}

func nullableOf[T any](value *T) nullable.Nullable[T] {
	if value == nil {
		return nullable.NewNullNullable[T]()
	}

	return nullable.NewNullableWithValue(*value)
}

func nullableString(value string) nullable.Nullable[string] {
	if value == "" {
		return nullable.NewNullNullable[string]()
	}

	return nullable.NewNullableWithValue(value)
}

func apiDate(value time.Time) *openapi_types.Date {
	return &openapi_types.Date{Time: value}
}

func nullableDate(value *time.Time) nullable.Nullable[openapi_types.Date] {
	if value == nil {
		return nullable.NewNullNullable[openapi_types.Date]()
	}

	return nullable.NewNullableWithValue(openapi_types.Date{Time: *value})
}

func mapInfo(info calculations.Info) api.Calculation {
	id, _ := uuid.Parse(info.ID)

	out := api.Calculation{
		ID:     id,
		Period: info.Period,
		Status: api.CalculationStatus(info.Status),
		Progress: api.CalculationProgress{
			ProcessedRows: info.Processed,
			TotalRows:     info.Total,
			Percent:       progressPercent(info.Processed, info.Total),
		},
		CreatedAt:  info.CreatedAt,
		FinishedAt: nullableOf(info.FinishedAt),
	}

	return out
}

func progressPercent(processed, total int) int {
	const full = 100
	if total == 0 {
		return 0
	}

	return processed * full / total
}

func mapRow(row domain.CalcRow) api.CalculationRow {
	out := api.CalculationRow{
		RowID:          strconv.FormatInt(row.RowID, 10),
		Period:         row.Period.Format("2006-01"),
		ClientID:       row.ClientID,
		ClientName:     row.ClientName,
		MaterialID:     strconv.FormatInt(row.MaterialID, 10),
		MaterialName:   row.MaterialName,
		DealType:       api.DealType(row.Contract),
		Currency:       row.Currency,
		Volume:         nullableOf(row.Forecast),
		Status:         statusToAPI[row.Status],
		FinalPrice:     nullableOf(row.Price),
		CandidateCount: &row.CandidateCount,
		RequiresReview: &row.RequiresReview,
		Warning:        nullableString(row.Warning),
		Error:          nullableString(row.Error),
	}
	if row.MaterialGroupM != "" {
		out.MaterialGroupM = nullable.NewNullableWithValue(row.MaterialGroupM)
	}

	return out
}

func mapKpi(kpi calculations.Kpi) api.KPI {
	return api.KPI{
		FormulaCoveragePct:   kpi.FormulaCoveragePct,
		FormulasOkPct:        kpi.FormulasOkPct,
		CalcErrorRows:        kpi.CalcErrorRows,
		ControlSumMln:        kpi.ControlSumMln,
		UnclassifiedErrorPct: kpi.UnclassifiedErrorPct,
	}
}

func mapComponent(component domain.ComponentValue) api.FormulaComponent {
	status := api.FormulaComponentStatus("ok")
	if component.Error != "" {
		status = "error"
	}

	componentType, ok := componentTypeToAPI[component.TypeCode]
	if !ok {
		componentType = "other"
	}

	out := api.FormulaComponent{
		VarName:     component.VarName,
		Type:        componentType,
		Status:      status,
		Value:       nullableOf(component.Value),
		QuoteName:   nullableString(component.QuoteName),
		QuoteCode:   nullableOf(component.QuoteCode),
		ValueDate:   nullableDate(component.SourceDate),
		VersionType: nullableString(component.VersionType),
		DateGapDays: nullableOf(component.DateGapDays),
		Warning:     nullableString(component.Warning),
		Error:       nullableString(component.Error),
	}
	if component.Source != "" {
		out.Source = &component.Source
	}
	if component.TypeLabel != "" {
		out.TypeLabel = &component.TypeLabel
	}

	return out
}

func mapCandidate(candidate domain.CandidateResult) api.AlternativeFormula {
	scope := api.MatchScope(candidate.MatchScope)
	candidateStatus := statusToAPI[candidate.Status]
	createdOn := candidate.CreatedAt

	out := api.AlternativeFormula{
		FormulaID:            candidate.FormulaID,
		FormulaText:          candidate.FormulaText,
		MatchScope:           &scope,
		ValidFrom:            apiDate(candidate.ValidFrom),
		ValidTo:              apiDate(candidate.ValidTo),
		CreatedOn:            nullableDate(&createdOn),
		IsActual:             &candidate.IsFormulaActual,
		Price:                nullableOf(candidate.Price),
		FormulaCurrency:      &candidate.FormulaCurrency,
		PriceFormulaCurrency: nullableOf(candidate.PriceFormulaCurrency),
		Status:               &candidateStatus,
		EqualPriorityCount:   &candidate.EqualPriorityCount,
		Warning:              nullableString(candidate.Warning),
		CalcError:            nullableString(candidate.Error),
		IsSelected:           candidate.IsSelected,
	}
	if reason, ok := selectionReasonToAPI[candidate.SelectionReason]; ok && candidate.IsSelected {
		out.SelectionReason = &reason
	}

	return out
}

func mapApplied(candidate domain.CandidateResult, variables []string) api.AppliedFormula {
	scope := api.MatchScope(candidate.MatchScope)
	createdOn := candidate.CreatedAt
	isExtended := candidate.Status == domain.StatusCalculated && !candidate.IsFormulaActual

	out := api.AppliedFormula{
		FormulaID:       candidate.FormulaID,
		FormulaText:     candidate.FormulaText,
		Variables:       &variables,
		FormulaCurrency: &candidate.FormulaCurrency,
		MatchScope:      &scope,
		ValidFrom:       apiDate(candidate.ValidFrom),
		ValidTo:         apiDate(candidate.ValidTo),
		CreatedOn:       nullableDate(&createdOn),
		IsActual:        &candidate.IsFormulaActual,
		IsExtended:      &isExtended,
	}
	if reason, ok := selectionReasonToAPI[candidate.SelectionReason]; ok {
		out.SelectionReason = &reason
	}

	return out
}

func mapConversion(conversion *domain.Conversion) *api.CurrencyConversion {
	if conversion == nil {
		return nil
	}

	return &api.CurrencyConversion{
		FromCurrency: conversion.FromCurrency,
		ToCurrency:   conversion.ToCurrency,
		FromRate:     nullableOf(conversion.FromRate),
		ToRate:       nullableOf(conversion.ToRate),
		RateDate:     nullableDate(conversion.RateDate),
		VersionType:  nullableString(conversion.VersionType),
	}
}

func mapDetails(details calculations.Details, variables []string) api.RowDetails {
	out := api.RowDetails{
		Row:                  mapRow(details.Row),
		Components:           make([]api.FormulaComponent, 0, len(details.Components)),
		Alternatives:         make([]api.AlternativeFormula, 0, len(details.Alternatives)),
		PriceFormulaCurrency: nullableOf(details.PriceFormulaCurrency),
		Conversion:           mapConversion(details.Conversion),
		EqualPriorityCount:   &details.EqualPriorityCount,
		ManualPrice:          nullableOf(details.ManualPrice),
	}

	if details.Applied != nil {
		applied := mapApplied(*details.Applied, variables)
		out.AppliedFormula = &applied
	}
	for _, component := range details.Components {
		out.Components = append(out.Components, mapComponent(component))
	}
	for _, candidate := range details.Alternatives {
		out.Alternatives = append(out.Alternatives, mapCandidate(candidate))
	}

	return out
}

func mapPart(part calculations.Part) api.ConsolidatedPart {
	id, _ := uuid.Parse(part.CalculationID)

	return api.ConsolidatedPart{
		CalculationID: id,
		AnalystName:   part.AnalystName,
		PartName:      nullableString(part.PartName),
		Status:        api.PartStatus(part.Status),
		RowCount:      part.RowCount,
		PricedPct:     &part.PricedPct,
		SubmittedAt:   nullableOf(part.SubmittedAt),
	}
}

func apiError(code, message string) api.Error {
	return api.Error{Code: code, Message: message}
}
