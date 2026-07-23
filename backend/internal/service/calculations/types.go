package calculations

import (
	"time"

	"sibur-petrochem-price-service/internal/domain"
)

// Info — состояние расчёта для API.
type Info struct {
	ID         string
	Period     string
	Status     string // pending | running | done | failed
	Processed  int
	Total      int
	CreatedAt  time.Time
	FinishedAt *time.Time
}

// ProgressEvent — событие SSE-потока прогресса.
type ProgressEvent struct {
	Status    string
	Processed int
	Total     int
	Percent   int
}

// RowsQuery — фильтры/сортировка/пагинация таблицы строк.
type RowsQuery struct {
	Status *domain.Status
	Query  string
	Sort   string // row_id | client | material | volume | price
	Order  string // asc | desc
	Limit  int
	Offset int
}

// RowsPage — страница строк + счётчики статусов до фильтрации.
type RowsPage struct {
	Items        []domain.CalcRow
	Total        int
	StatusCounts map[domain.Status]int
}

// Details — полная расшифровка строки.
type Details struct {
	Row                  domain.CalcRow
	Applied              *domain.CandidateResult
	Components           []domain.ComponentValue
	Alternatives         []domain.CandidateResult
	PriceFormulaCurrency *float64
	Conversion           *domain.Conversion
	EqualPriorityCount   int
	ManualPrice          *float64
}

// Kpi — пять канонических показателей (documents/кпэ_для_отображения.md).
type Kpi struct {
	FormulaCoveragePct   int
	FormulasOkPct        int
	CalcErrorRows        int
	ControlSumMln        float64
	UnclassifiedErrorPct int
}

// Part — участок аналитика в сводном документе.
type Part struct {
	CalculationID string
	AnalystName   string
	PartName      string
	Status        string // draft | review | joined
	RowCount      int
	PricedPct     int
	SubmittedAt   *time.Time
}

// ConsolidatedRow — строка сводного документа.
type ConsolidatedRow struct {
	AnalystName string
	IsDraft     bool
	Row         domain.CalcRow
}

// Consolidated — сводный документ за период.
type Consolidated struct {
	Period    string
	Parts     []Part
	Kpi       Kpi
	Rows      []ConsolidatedRow
	TotalRows int
}
