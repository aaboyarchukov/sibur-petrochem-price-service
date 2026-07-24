package http

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	nethttp "net/http"
)

// Колонки пользовательских выгрузок — контракт формата .xlsx (см. service/ingest).
var (
	sspUploadHeaders = []string{
		"row_id", "period", "customer_asv", "customer", "mtr_nsi_code", "mtr_nsi_name",
		"contract", "currency", "forecast", "country", "region", "market", "client_id", "client_name",
	}
	formulaUploadHeaders = []string{
		"Ключ формулы", "Формула", "Подгруппа материалов", "Деловой партнер",
		`"Действительно с" - метка врем. объекта`, `Метка времени "Действительно по"`,
		"Дата создания", "Неактивна", "Тип цены", "ВалютаДокумента", "Материал", "client_id", "client_name",
	}
)

// buildSourceXlsx — книга с одним листом: заголовки + строки.
func buildSourceXlsx(t *testing.T, headers []string, rows [][]any) io.Reader {
	t.Helper()

	book := excelize.NewFile()
	sheet := book.GetSheetName(0)

	headerCells := make([]any, 0, len(headers))
	for _, header := range headers {
		headerCells = append(headerCells, header)
	}
	require.NoError(t, book.SetSheetRow(sheet, "A1", &headerCells))

	for i, row := range rows {
		axis, err := excelize.CoordinatesToCellName(1, i+2)
		require.NoError(t, err)
		require.NoError(t, book.SetSheetRow(sheet, axis, &row))
	}

	buffer, err := book.WriteToBuffer()
	require.NoError(t, err)

	return bytes.NewReader(buffer.Bytes())
}

// uploadFile — POST /sources/{key}/file с multipart-частью file.
func uploadFile(t *testing.T, sut nethttp.Handler, key string, file io.Reader) (int, map[string]any) {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", key+".xlsx")
	require.NoError(t, err)
	_, err = io.Copy(part, file)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(nethttp.MethodPost, "/api/v1/sources/"+key+"/file", &buf)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	sut.ServeHTTP(recorder, request)

	parsed := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &parsed)

	return recorder.Code, parsed
}

func sspUploadRow(rowID int64, period, client string) []any {
	return []any{
		rowID, period, "354065", "ООО Тест", int64(226814), "Полипропилен PP",
		"Formula", "RUB", "100.5", "Russia", "Region", "Россия", client, "Клиент " + client,
	}
}

func TestUploadSspReplacesData(t *testing.T) {
	tc := newTestContext(t)

	tc.Given(func(t *testing.T, state State) State {
		t.Helper()
		state.Given.Sources = demoSources(t, state)

		return state
	}).When(func(t *testing.T, deps Deps, state State) State {
		t.Helper()
		deps.Sources.src = state.Given.Sources
		deps.Sources.counts["ssp"] = int64(len(state.Given.Sources.Ssp))

		return state
	}).Then(func(t *testing.T, state State) {
		t.Helper()
		require.Equal(t, nethttp.StatusOK, state.Response.Code)
		assert.EqualValues(t, 2, state.Response.Body["row_count"])
		assert.Equal(t, "loaded", state.Response.Body["status"])
		assert.NotEmpty(t, state.Response.Body["uploaded_at"])
	})

	file := buildSourceXlsx(t, sspUploadHeaders, [][]any{
		sspUploadRow(9001, "2026-07-01 00:00:00", "CL-9001"),
		sspUploadRow(9002, "2026-07-01 00:00:00", "CL-9002"),
	})
	code, body := uploadFile(t, tc.SUT, "ssp", file)
	tc.State.Response = Responses{Code: code, Body: body}

	// замещение видно в превью источника
	previewCode, preview, _ := do(t, tc.SUT, nethttp.MethodGet, "/api/v1/sources/ssp/preview", nil)
	require.Equal(t, nethttp.StatusOK, previewCode)
	assert.EqualValues(t, 2, preview["total_rows"])

	// расчёт за период файла использует загруженные строки
	calcID := createCalculation(t, tc.SUT, "2026-07")
	rowsCode, rowsBody, _ := do(t, tc.SUT, nethttp.MethodGet, "/api/v1/calculations/"+calcID+"/rows?limit=50", nil)
	require.Equal(t, nethttp.StatusOK, rowsCode)
	items, ok := rowsBody["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 2)
}

func TestUploadSspRejectsBrokenFile(t *testing.T) {
	tc := newTestContext(t)

	tc.Given(func(t *testing.T, state State) State {
		t.Helper()
		state.Given.Sources = demoSources(t, state)

		return state
	}).When(func(t *testing.T, deps Deps, state State) State {
		t.Helper()
		deps.Sources.src = state.Given.Sources
		deps.Sources.counts["ssp"] = int64(len(state.Given.Sources.Ssp))

		return state
	}).Then(func(t *testing.T, state State) {
		t.Helper()
		require.Equal(t, nethttp.StatusBadRequest, state.Response.Code)
		message, _ := state.Response.Body["message"].(string)
		assert.Contains(t, message, "period")
	})

	headersWithoutPeriod := make([]string, 0, len(sspUploadHeaders)-1)
	for _, header := range sspUploadHeaders {
		if header != "period" {
			headersWithoutPeriod = append(headersWithoutPeriod, header)
		}
	}
	file := buildSourceXlsx(t, headersWithoutPeriod, [][]any{sspUploadRow(9001, "2026-07-01", "CL-9001")})
	code, body := uploadFile(t, tc.SUT, "ssp", file)
	tc.State.Response = Responses{Code: code, Body: body}

	// данные источника не изменены
	previewCode, preview, _ := do(t, tc.SUT, nethttp.MethodGet, "/api/v1/sources/ssp/preview", nil)
	require.Equal(t, nethttp.StatusOK, previewCode)
	assert.EqualValues(t, len(tc.State.Given.Sources.Ssp), preview["total_rows"])
}

func TestUploadFormulasReplacesData(t *testing.T) {
	tc := newTestContext(t)

	tc.Then(func(t *testing.T, state State) {
		t.Helper()
		require.Equal(t, nethttp.StatusOK, state.Response.Code)
		assert.EqualValues(t, 1, state.Response.Body["row_count"])
	})

	file := buildSourceXlsx(t, formulaUploadHeaders, [][]any{{
		"Z430000113", "( CFR - DISCOUNT ) * K", "", "91442",
		"2026-01-01 00:00:00", "2026-12-31 00:00:00", "2025-12-17 00:00:00",
		"", "2", "CNY", int64(1353711), "CL-10395", "Клиент 395",
	}})
	code, body := uploadFile(t, tc.SUT, "formulas", file)
	tc.State.Response = Responses{Code: code, Body: body}
}

func TestUploadReferenceSourceKeepsSeedBehavior(t *testing.T) {
	tc := newTestContext(t)

	tc.When(func(t *testing.T, deps Deps, state State) State {
		t.Helper()
		deps.Sources.counts["quotes"] = 15

		return state
	}).Then(func(t *testing.T, state State) {
		t.Helper()
		require.Equal(t, nethttp.StatusOK, state.Response.Code)
		assert.EqualValues(t, 15, state.Response.Body["row_count"])
		assert.Nil(t, state.Response.Body["uploaded_at"])
	})

	code, body := uploadFile(t, tc.SUT, "quotes", strings.NewReader("любой файл — содержимое игнорируется"))
	tc.State.Response = Responses{Code: code, Body: body}
}

func TestCreateCalculationWithoutSources(t *testing.T) {
	tc := newTestContext(t)

	tc.Then(func(t *testing.T, state State) {
		t.Helper()
		require.Equal(t, nethttp.StatusConflict, state.Response.Code)
		assert.Equal(t, "sources_not_loaded", state.Response.Body["code"])
	})

	// ни файлы, ни демо-набор не активированы — расчёт запрещён
	code, body, _ := do(t, tc.SUT, nethttp.MethodPost, "/api/v1/calculations", calcBody("2026-06"))
	tc.State.Response = Responses{Code: code, Body: body}
}

func TestCreateCalculationAfterDemoActivation(t *testing.T) {
	tc := newTestContext(t)

	tc.Given(func(t *testing.T, state State) State {
		t.Helper()
		state.Given.Sources = demoSources(t, state)

		return state
	}).When(func(t *testing.T, deps Deps, state State) State {
		t.Helper()
		deps.Sources.src = state.Given.Sources

		return state
	}).Then(func(t *testing.T, state State) {
		t.Helper()
		require.Equal(t, nethttp.StatusAccepted, state.Response.Code)
	})

	demoCode, demoBody, _ := do(t, tc.SUT, nethttp.MethodPost, "/api/v1/sources/demo", nil)
	require.Equal(t, nethttp.StatusOK, demoCode)
	items, ok := demoBody["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 8)
	// демо помечает пользовательские источники загруженными
	first, ok := items[0].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, first["uploaded_at"])

	code, body, _ := do(t, tc.SUT, nethttp.MethodPost, "/api/v1/calculations", calcBody("2026-06"))
	tc.State.Response = Responses{Code: code, Body: body}
}

func TestResetSourcesReturnsInitialState(t *testing.T) {
	tc := newTestContext(t)

	tc.Given(func(t *testing.T, state State) State {
		t.Helper()
		state.Given.Sources = demoSources(t, state)

		return state
	}).When(func(t *testing.T, deps Deps, state State) State {
		t.Helper()
		deps.Sources.src = state.Given.Sources

		return state
	}).Then(func(t *testing.T, state State) {
		t.Helper()
		// после сброса расчёт снова запрещён
		require.Equal(t, nethttp.StatusConflict, state.Response.Code)
		assert.Equal(t, "sources_not_loaded", state.Response.Body["code"])
	})

	// полный цикл: демо + расчёт → reset → отметки сняты, расчёт запрещён
	calcID := createCalculation(t, tc.SUT, "2026-06")

	resetCode, resetBody, _ := do(t, tc.SUT, nethttp.MethodPost, "/api/v1/sources/reset", nil)
	require.Equal(t, nethttp.StatusOK, resetCode)
	items, ok := resetBody["items"].([]any)
	require.True(t, ok)
	first, ok := items[0].(map[string]any)
	require.True(t, ok)
	assert.Nil(t, first["uploaded_at"], "отметка загрузки должна быть снята")

	// расчёты сессии удалены
	getCode, _, _ := do(t, tc.SUT, nethttp.MethodGet, "/api/v1/calculations/"+calcID, nil)
	require.Equal(t, nethttp.StatusNotFound, getCode)

	code, body, _ := do(t, tc.SUT, nethttp.MethodPost, "/api/v1/calculations", calcBody("2026-06"))
	tc.State.Response = Responses{Code: code, Body: body}
}

func TestUploadUnknownSource(t *testing.T) {
	tc := newTestContext(t)

	tc.Then(func(t *testing.T, state State) {
		t.Helper()
		require.Equal(t, nethttp.StatusBadRequest, state.Response.Code)
	})

	code, body := uploadFile(t, tc.SUT, "unknown", strings.NewReader("x"))
	tc.State.Response = Responses{Code: code, Body: body}
}
