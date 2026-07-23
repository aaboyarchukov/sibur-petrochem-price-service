package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"sibur-petrochem-price-service/internal/domain"

	sqlc_gen "sibur-petrochem-price-service/internal/generated/sqlc"
)

// ErrTransactionsUnsupported — подключение не умеет транзакции (например, замокан DBTX).
var ErrTransactionsUnsupported = errors.New("db connection does not support transactions")

// txStarter — способность соединения открыть транзакцию (*pgxpool.Pool, *pgx.Conn).
type txStarter interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// ReplaceSsp — полное замещение прогноза спроса: delete + copyfrom в одной транзакции.
func (r *Repository) ReplaceSsp(ctx context.Context, rows []domain.SspRow) error {
	return r.replace(ctx, func(qtx *sqlc_gen.Queries) error {
		if err := qtx.DeleteSsp(ctx); err != nil {
			return fmt.Errorf("delete ssp: %w", err)
		}
		if _, err := qtx.CopySsp(ctx, toSspParams(rows)); err != nil {
			return fmt.Errorf("copy ssp: %w", err)
		}

		return nil
	})
}

// ReplaceFormulas — полное замещение каталога формул: delete + copyfrom в одной транзакции.
func (r *Repository) ReplaceFormulas(ctx context.Context, rows []domain.Formula) error {
	return r.replace(ctx, func(qtx *sqlc_gen.Queries) error {
		if err := qtx.DeleteFormulas(ctx); err != nil {
			return fmt.Errorf("delete formulas: %w", err)
		}
		if _, err := qtx.CopyFormulas(ctx, toFormulaParams(rows)); err != nil {
			return fmt.Errorf("copy formulas: %w", err)
		}

		return nil
	})
}

func (r *Repository) replace(ctx context.Context, apply func(qtx *sqlc_gen.Queries) error) error {
	starter, ok := r.db.(txStarter)
	if !ok {
		return ErrTransactionsUnsupported
	}

	tx, err := starter.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := apply(r.queries.WithTx(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func toSspParams(rows []domain.SspRow) []sqlc_gen.CopySspParams {
	out := make([]sqlc_gen.CopySspParams, 0, len(rows))
	for _, row := range rows {
		out = append(out, sqlc_gen.CopySspParams{
			RowID:       row.RowID,
			Period:      pgtype.Date{Time: row.Period, Valid: true},
			CustomerAsv: optional(row.CustomerASV),
			Customer:    optional(row.Customer),
			MtrNsiCode:  row.MaterialID,
			MtrNsiName:  row.MaterialName,
			Contract:    row.Contract,
			Currency:    row.Currency,
			Forecast:    row.Forecast,
			Country:     optional(row.Country),
			Region:      optional(row.Region),
			Market:      optional(row.Market),
			ClientID:    row.ClientID,
			ClientName:  row.ClientName,
		})
	}

	return out
}

func toFormulaParams(rows []domain.Formula) []sqlc_gen.CopyFormulasParams {
	out := make([]sqlc_gen.CopyFormulasParams, 0, len(rows))
	for _, row := range rows {
		out = append(out, sqlc_gen.CopyFormulasParams{
			FormulaID:       row.FormulaID,
			FormulaText:     row.Text,
			MaterialGroupM:  optional(row.MaterialGroupM),
			BusinessPartner: optional(row.BusinessPartner),
			ValidFrom:       pgtype.Date{Time: row.ValidFrom, Valid: true},
			ValidTo:         pgtype.Date{Time: row.ValidTo, Valid: true},
			CreatedAt:       pgtype.Timestamp{Time: row.CreatedAt, Valid: true},
			Inactive:        row.Inactive,
			PriceType:       optional(row.PriceType),
			DocCurrency:     row.DocCurrency,
			MaterialID:      row.MaterialID,
			ClientID:        row.ClientID,
			ClientName:      row.ClientName,
		})
	}

	return out
}

// optional — пустая строка в домене означает отсутствие значения (NULL).
func optional(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}
