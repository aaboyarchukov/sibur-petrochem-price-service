package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"sibur-petrochem-price-service/internal/domain"

	sqlc_gen "sibur-petrochem-price-service/internal/generated/sqlc"
)

// Маппинг sqlc-строк в доменные модели. Nullable-указатели переводятся
// в пустые значения там, где домен трактует отсутствие как «пусто».

func mapSsp(rows []sqlc_gen.ListSspRow) []domain.SspRow {
	out := make([]domain.SspRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.SspRow{
			RowID:        row.RowID,
			Period:       row.Period.Time,
			CustomerASV:  deref(row.CustomerAsv),
			Customer:     deref(row.Customer),
			MaterialID:   row.MtrNsiCode,
			MaterialName: row.MtrNsiName,
			Contract:     row.Contract,
			Currency:     row.Currency,
			Forecast:     row.Forecast,
			Country:      deref(row.Country),
			Region:       deref(row.Region),
			Market:       deref(row.Market),
			ClientID:     row.ClientID,
			ClientName:   row.ClientName,
		})
	}

	return out
}

func mapFormulas(rows []sqlc_gen.ListFormulasRow) []domain.Formula {
	out := make([]domain.Formula, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Formula{
			FormulaID:       row.FormulaID,
			Text:            row.FormulaText,
			MaterialGroupM:  deref(row.MaterialGroupM),
			BusinessPartner: deref(row.BusinessPartner),
			ValidFrom:       row.ValidFrom.Time,
			ValidTo:         row.ValidTo.Time,
			CreatedAt:       row.CreatedAt.Time,
			Inactive:        row.Inactive,
			PriceType:       deref(row.PriceType),
			DocCurrency:     row.DocCurrency,
			MaterialID:      row.MaterialID,
			ClientID:        row.ClientID,
			ClientName:      row.ClientName,
		})
	}

	return out
}

func mapComponents(rows []sqlc_gen.ListFormulaComponentsRow) []domain.FormulaComponent {
	out := make([]domain.FormulaComponent, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.FormulaComponent{
			FormulaID: row.FormulaID,
			TermNo:    int(row.TermNo),
			VarName:   row.VarName,
			ValidFrom: row.ValidFrom.Time,
			ValidTo:   row.ValidTo.Time,
			TypeCode:  row.TypeCode,
			Value:     row.Value,
			Currency:  deref(row.Currency),
			QuoteName: deref(row.QuoteName),
		})
	}

	return out
}

func mapTermTypes(rows []sqlc_gen.TermType) []domain.TermType {
	out := make([]domain.TermType, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.TermType{
			TypeCode:  row.TypeCode,
			TypeLabel: row.TypeLabel,
			Category:  row.Category,
		})
	}

	return out
}

func mapQuotes(rows []sqlc_gen.ListQuotesRow) []domain.Quote {
	out := make([]domain.Quote, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Quote{
			QuoteType:  row.QuoteType,
			QuoteName:  row.QuoteName,
			QuoteCode:  row.QuoteCode,
			QuoteDate:  row.QuoteDate.Time,
			Currency:   row.QuoteCurrency,
			Value:      row.QuoteVal,
			TechLoadTS: tsPtr(row.TechLoadTs),
		})
	}

	return out
}

func mapQuoteMapping(rows []sqlc_gen.ListQuoteMappingRow) []domain.QuoteMapping {
	out := make([]domain.QuoteMapping, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.QuoteMapping{
			QuoteName: row.QuoteName,
			LakeID:    row.LakeID,
			Currency:  row.QuoteCurrency,
		})
	}

	return out
}

func mapCurrencyRates(rows []sqlc_gen.ListCurrencyRatesRow) []domain.CurrencyRate {
	out := make([]domain.CurrencyRate, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.CurrencyRate{
			Currency:    row.CurrencyName,
			CalDay:      row.Calday.Time,
			VersionType: row.VersionType,
			Value:       row.CurrencyValue,
		})
	}

	return out
}

func mapMaterialGroups(rows []sqlc_gen.ListMaterialGroupsRow) []domain.MaterialGroup {
	out := make([]domain.MaterialGroup, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.MaterialGroup{
			GroupM:       row.GroupM,
			MaterialID:   row.MaterialID,
			MaterialName: row.MaterialName,
			ValidFrom:    row.ValidFrom.Time,
			ValidTo:      row.ValidTo.Time,
		})
	}

	return out
}

func deref(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func tsPtr(value pgtype.Timestamp) *time.Time {
	if !value.Valid {
		return nil
	}

	parsed := value.Time

	return &parsed
}
