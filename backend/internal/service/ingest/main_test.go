package ingest

import (
	"bytes"
	"io"
	"testing"

	"github.com/godepo/groat"
	"github.com/jaswdr/faker/v2"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"sibur-petrochem-price-service/internal/domain"
)

type (
	Givens struct {
		Headers []string
		Cells   [][]any
	}

	Responses struct {
		Ssp      []domain.SspRow
		Formulas []domain.Formula
		Issues   []Issue
	}

	Expects struct {
		RowCount     int
		IssueSubstrs []string
	}

	Deps struct{}

	State struct {
		Given    Givens
		Faker    faker.Faker
		Response Responses
		Expect   Expects
	}
)

// parsers — SUT: фасад над функциями пакета для groat-контекста.
type parsers struct{}

func (parsers) Ssp(r io.Reader) ([]domain.SspRow, []Issue)       { return ParseSsp(r) }
func (parsers) Formulas(r io.Reader) ([]domain.Formula, []Issue) { return ParseFormulas(r) }

func newTestContext(t *testing.T) *groat.Case[Deps, State, parsers] {
	t.Helper()

	tc := groat.New[Deps, State, parsers](t, func(t *testing.T, _ Deps) parsers {
		t.Helper()

		return parsers{}
	})

	tc.Go()

	tc.State.Faker = faker.New()

	return tc
}

// buildXlsx — книга с одним листом: заголовки + строки данных.
func buildXlsx(t *testing.T, headers []string, cells [][]any) io.Reader {
	t.Helper()

	book := excelize.NewFile()
	sheet := book.GetSheetName(0)

	headerCells := make([]any, 0, len(headers))
	for _, header := range headers {
		headerCells = append(headerCells, header)
	}
	require.NoError(t, book.SetSheetRow(sheet, "A1", &headerCells))

	for i, row := range cells {
		axis, err := excelize.CoordinatesToCellName(1, i+2)
		require.NoError(t, err)
		require.NoError(t, book.SetSheetRow(sheet, axis, &row))
	}

	buffer, err := book.WriteToBuffer()
	require.NoError(t, err)

	return bytes.NewReader(buffer.Bytes())
}
