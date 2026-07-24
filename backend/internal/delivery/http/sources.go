package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strconv"
	"strings"

	"github.com/oapi-codegen/nullable"

	"sibur-petrochem-price-service/internal/domain"
	"sibur-petrochem-price-service/internal/service/ingest"

	api "sibur-petrochem-price-service/internal/generated/api"
)

// Пользовательские источники: только они принимают реальный ingest .xlsx.
const (
	sourceKeySsp      api.SourceKey = "ssp"
	sourceKeyFormulas api.SourceKey = "formulas"
)

// sourceDef — статическое описание источника (ключ, название, файл, вид).
type sourceDef struct {
	key      api.SourceKey
	name     string
	fileName string
	kind     api.SourceKind
}

// Порядок фиксирован: 2 пользовательских + 6 справочников.
var sourceDefs = []sourceDef{
	{key: "ssp", name: "Прогноз спроса", fileName: "ssp.xlsx", kind: "uploaded"},
	{key: "formulas", name: "Каталог формул", fileName: "formulas.xlsx", kind: "uploaded"},
	{key: "formula_components", name: "Компоненты формул", fileName: "formula_components.xlsx", kind: "reference"},
	{key: "term_types", name: "Типы термов", fileName: "term_types.xlsx", kind: "reference"},
	{key: "quotes", name: "Котировки", fileName: "quotes.xlsx", kind: "reference"},
	{key: "quote_mapping", name: "Маппинг котировок", fileName: "quote_mapping.xlsx", kind: "reference"},
	{key: "currency_rates", name: "Курсы валют", fileName: "currency_rates.xlsx", kind: "reference"},
	{key: "material_groups", name: "Группы материалов", fileName: "material_groups.xlsx", kind: "reference"},
}

func (h *Handler) buildSource(def sourceDef, rowCount int64) api.Source {
	fileName := def.fileName
	count := int(rowCount)
	status := api.SourceStatus("loaded")
	if count == 0 {
		status = "missing"
	}

	out := api.Source{
		Key:      def.key,
		Name:     def.name,
		FileName: &fileName,
		Kind:     def.kind,
		Status:   status,
		RowCount: nullable.NewNullableWithValue(count),
		Issues:   &[]string{},
	}
	if ts, ok := h.uploadTime(def.key); ok {
		out.UploadedAt = nullable.NewNullableWithValue(ts)
	}

	return out
}

func (h *Handler) ListSources(ctx context.Context, _ api.ListSourcesRequestObject) (api.ListSourcesResponseObject, error) {
	items, err := h.sourceItems(ctx)
	if err != nil {
		return nil, err
	}

	return api.ListSources200JSONResponse{Items: items}, nil
}

// GetSourceFacets — продукты, клиенты и границы горизонта из ssp для пикеров параметров.
func (h *Handler) GetSourceFacets(ctx context.Context, _ api.GetSourceFacetsRequestObject) (api.GetSourceFacetsResponseObject, error) {
	facets, err := h.sources.SourceFacets(ctx)
	if err != nil {
		return nil, err
	}

	return api.GetSourceFacets200JSONResponse(mapFacets(facets)), nil
}

// ResetSources — начальное состояние: отметки загрузки снимаются, расчёты
// сессии удаляются. Данные таблиц остаются — повторная загрузка или демо
// снова разрешают расчёт.
func (h *Handler) ResetSources(ctx context.Context, _ api.ResetSourcesRequestObject) (api.ResetSourcesResponseObject, error) {
	h.resetUploaded()
	h.calcs.Reset()

	items, err := h.sourceItems(ctx)
	if err != nil {
		return nil, err
	}

	return api.ResetSources200JSONResponse{Items: items}, nil
}

// LoadDemoSources — активация демо-набора: seed-данные помечаются загруженными,
// расчёт разрешается без передачи файлов.
func (h *Handler) LoadDemoSources(ctx context.Context, _ api.LoadDemoSourcesRequestObject) (api.LoadDemoSourcesResponseObject, error) {
	h.markUploaded(sourceKeySsp)
	h.markUploaded(sourceKeyFormulas)

	items, err := h.sourceItems(ctx)
	if err != nil {
		return nil, err
	}

	return api.LoadDemoSources200JSONResponse{Items: items}, nil
}

func (h *Handler) sourceItems(ctx context.Context) ([]api.Source, error) {
	counts, err := h.sources.SourceCounts(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]api.Source, 0, len(sourceDefs))
	for _, def := range sourceDefs {
		items = append(items, h.buildSource(def, counts[string(def.key)]))
	}

	return items, nil
}

// UploadSourceFile — приём файла источника. Для ssp/formulas выполняется полный
// ingest .xlsx (парсинг + замещение таблицы); остальные источники — из seed-миграций,
// файл принимается без обработки.
func (h *Handler) UploadSourceFile(ctx context.Context, request api.UploadSourceFileRequestObject) (api.UploadSourceFileResponseObject, error) {
	def, found := sourceDefByKey(api.SourceKey(request.SourceKey))
	if !found {
		return upload400("unknown_source", "неизвестный ключ источника"), nil
	}

	if def.key == sourceKeySsp || def.key == sourceKeyFormulas {
		failure, err := h.ingestUpload(ctx, def.key, request.Body)
		if failure != nil || err != nil {
			return failure, err
		}
		h.markUploaded(def.key)
	}

	counts, err := h.sources.SourceCounts(ctx)
	if err != nil {
		return nil, err
	}

	return api.UploadSourceFile200JSONResponse(h.buildSource(def, counts[string(def.key)])), nil
}

// ingestUpload — чтение файла из multipart, парсинг и замещение данных источника.
// Непустой failure — готовый ответ 400; (nil, nil) — успех.
func (h *Handler) ingestUpload(
	ctx context.Context,
	key api.SourceKey,
	body *multipart.Reader,
) (failure api.UploadSourceFileResponseObject, err error) {
	data, readErr := readFilePart(body)
	if readErr != nil {
		return upload400("invalid_file", readErr.Error()), nil
	}

	issues, replaceErr := h.replaceSource(ctx, key, data)
	if replaceErr != nil {
		return nil, replaceErr
	}
	if len(issues) > 0 {
		return upload400("validation_failed", formatIssues(issues)), nil
	}

	return nil, nil
}

// replaceSource — парсинг по ключу источника и замещение таблицы при чистом файле.
func (h *Handler) replaceSource(ctx context.Context, key api.SourceKey, data []byte) ([]ingest.Issue, error) {
	if key == sourceKeySsp {
		rows, issues := ingest.ParseSsp(bytes.NewReader(data))
		if len(issues) > 0 {
			return issues, nil
		}

		return nil, h.sources.ReplaceSsp(ctx, rows)
	}

	rows, issues := ingest.ParseFormulas(bytes.NewReader(data))
	if len(issues) > 0 {
		return issues, nil
	}

	return nil, h.sources.ReplaceFormulas(ctx, rows)
}

// readFilePart — часть file из multipart-тела с лимитом размера.
func readFilePart(body *multipart.Reader) ([]byte, error) {
	const maxUploadBytes = 20 << 20

	if body == nil {
		return nil, errors.New("пустое тело запроса")
	}

	for {
		part, err := body.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("чтение multipart: %w", err)
		}
		if part.FormName() != "file" {
			continue
		}

		data, err := io.ReadAll(io.LimitReader(part, maxUploadBytes+1))
		if err != nil {
			return nil, fmt.Errorf("чтение файла: %w", err)
		}
		if len(data) > maxUploadBytes {
			return nil, errors.New("файл больше 20 МБ")
		}
		if len(data) == 0 {
			return nil, errors.New("пустой файл")
		}

		return data, nil
	}

	return nil, errors.New("в запросе нет части file")
}

// formatIssues — первые 20 проблем в одну строку + счётчик остальных.
func formatIssues(issues []ingest.Issue) string {
	const maxShown = 20

	shown := issues
	if len(shown) > maxShown {
		shown = shown[:maxShown]
	}

	parts := make([]string, 0, len(shown))
	for _, issue := range shown {
		parts = append(parts, issue.String())
	}

	out := "файл отклонён: " + strings.Join(parts, "; ")
	if extra := len(issues) - maxShown; extra > 0 {
		out += "; и ещё " + strconv.Itoa(extra) + " проблем"
	}

	return out
}

func sourceDefByKey(key api.SourceKey) (def sourceDef, found bool) {
	for _, candidate := range sourceDefs {
		if candidate.key == key {
			return candidate, true
		}
	}

	return sourceDef{}, false
}

func upload400(code, message string) api.UploadSourceFile400JSONResponse {
	return api.UploadSourceFile400JSONResponse{
		BadRequestJSONResponse: api.BadRequestJSONResponse(apiError(code, message)),
	}
}

func (h *Handler) PreviewSource(ctx context.Context, request api.PreviewSourceRequestObject) (api.PreviewSourceResponseObject, error) {
	const defaultLimit = 5
	limit := defaultLimit
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	src, err := h.sources.LoadSources(ctx)
	if err != nil {
		return nil, err
	}

	preview, ok := buildPreview(src, api.SourceKey(request.SourceKey), limit)
	if !ok {
		return api.PreviewSource404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse(apiError("not_found", "источник не найден")),
		}, nil
	}

	return api.PreviewSource200JSONResponse(preview), nil
}

// buildPreview — первые строки источника в форме columns + rows.
func buildPreview(src domain.Sources, key api.SourceKey, limit int) (api.SourcePreview, bool) {
	//nolint:exhaustive // ветки перечислены строковыми литералами ключей; default закрывает остальное
	switch key {
	case "ssp":
		return previewSsp(src.Ssp, limit), true
	case "formulas":
		return previewFormulas(src.Formulas, limit), true
	case "formula_components":
		return previewComponents(src.Components, limit), true
	case "term_types":
		return previewTermTypes(src.TermTypes, limit), true
	case "quotes":
		return previewQuotes(src.Quotes, limit), true
	case "quote_mapping":
		return previewQuoteMapping(src.QuoteMapping, limit), true
	case "currency_rates":
		return previewRates(src.CurrencyRates, limit), true
	case "material_groups":
		return previewGroups(src.MaterialGroups, limit), true
	default:
		return api.SourcePreview{}, false
	}
}

func previewSsp(rows []domain.SspRow, limit int) api.SourcePreview {
	out := api.SourcePreview{
		Columns:   []string{"row_id", "period", "client_name", "mtr_nsi_name", "forecast", "contract"},
		Rows:      make([][]string, 0, limit),
		TotalRows: len(rows),
	}
	for _, row := range rows[:min(limit, len(rows))] {
		forecast := ""
		if row.Forecast != nil {
			forecast = strconv.FormatFloat(*row.Forecast, 'f', -1, 64)
		}
		out.Rows = append(out.Rows, []string{
			strconv.FormatInt(row.RowID, 10), row.Period.Format("2006-01"),
			row.ClientName, row.MaterialName, forecast, row.Contract,
		})
	}

	return out
}

func previewFormulas(rows []domain.Formula, limit int) api.SourcePreview {
	out := api.SourcePreview{
		Columns:   []string{"formula_id", "formula_text", "client_id", "valid_from", "valid_to", "doc_currency"},
		Rows:      make([][]string, 0, limit),
		TotalRows: len(rows),
	}
	for _, row := range rows[:min(limit, len(rows))] {
		out.Rows = append(out.Rows, []string{
			row.FormulaID, row.Text, row.ClientID,
			row.ValidFrom.Format("2006-01-02"), row.ValidTo.Format("2006-01-02"), row.DocCurrency,
		})
	}

	return out
}

func previewComponents(rows []domain.FormulaComponent, limit int) api.SourcePreview {
	out := api.SourcePreview{
		Columns:   []string{"formula_id", "var_name", "type_code", "value", "quote_name"},
		Rows:      make([][]string, 0, limit),
		TotalRows: len(rows),
	}
	for _, row := range rows[:min(limit, len(rows))] {
		value := ""
		if row.Value != nil {
			value = strconv.FormatFloat(*row.Value, 'f', -1, 64)
		}
		out.Rows = append(out.Rows, []string{row.FormulaID, row.VarName, row.TypeCode, value, row.QuoteName})
	}

	return out
}

func previewTermTypes(rows []domain.TermType, limit int) api.SourcePreview {
	out := api.SourcePreview{
		Columns:   []string{"type_code", "type_label", "category"},
		Rows:      make([][]string, 0, limit),
		TotalRows: len(rows),
	}
	for _, row := range rows[:min(limit, len(rows))] {
		out.Rows = append(out.Rows, []string{row.TypeCode, row.TypeLabel, row.Category})
	}

	return out
}

func previewQuotes(rows []domain.Quote, limit int) api.SourcePreview {
	out := api.SourcePreview{
		Columns:   []string{"quote_code", "quote_name", "quote_date", "quote_type", "quote_val"},
		Rows:      make([][]string, 0, limit),
		TotalRows: len(rows),
	}
	for _, row := range rows[:min(limit, len(rows))] {
		out.Rows = append(out.Rows, []string{
			strconv.FormatInt(row.QuoteCode, 10), row.QuoteName,
			row.QuoteDate.Format("2006-01-02"), row.QuoteType,
			strconv.FormatFloat(row.Value, 'f', -1, 64),
		})
	}

	return out
}

func previewQuoteMapping(rows []domain.QuoteMapping, limit int) api.SourcePreview {
	out := api.SourcePreview{
		Columns:   []string{"quote_name", "lake_id", "quote_currency"},
		Rows:      make([][]string, 0, limit),
		TotalRows: len(rows),
	}
	for _, row := range rows[:min(limit, len(rows))] {
		lakeID := ""
		if row.LakeID != nil {
			lakeID = strconv.FormatInt(*row.LakeID, 10)
		}
		out.Rows = append(out.Rows, []string{row.QuoteName, lakeID, row.Currency})
	}

	return out
}

func previewRates(rows []domain.CurrencyRate, limit int) api.SourcePreview {
	out := api.SourcePreview{
		Columns:   []string{"currency_name", "calday", "version_type", "currency_value"},
		Rows:      make([][]string, 0, limit),
		TotalRows: len(rows),
	}
	for _, row := range rows[:min(limit, len(rows))] {
		out.Rows = append(out.Rows, []string{
			row.Currency, row.CalDay.Format("2006-01-02"), row.VersionType,
			strconv.FormatFloat(row.Value, 'f', -1, 64),
		})
	}

	return out
}

func previewGroups(rows []domain.MaterialGroup, limit int) api.SourcePreview {
	out := api.SourcePreview{
		Columns:   []string{"group_m", "material_id", "material_name"},
		Rows:      make([][]string, 0, limit),
		TotalRows: len(rows),
	}
	for _, row := range rows[:min(limit, len(rows))] {
		out.Rows = append(out.Rows, []string{row.GroupM, strconv.FormatInt(row.MaterialID, 10), row.MaterialName})
	}

	return out
}
