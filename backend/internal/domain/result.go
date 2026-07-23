package domain

import "time"

// Status — статус строки/кандидата в терминах эталонного алгоритма.
type Status string

const (
	StatusCalculated        Status = "CALCULATED"
	StatusCalculatedExpired Status = "CALCULATED_WITH_EXPIRED_FORMULA"
	StatusFormulaConflict   Status = "CALCULATED_WITH_FORMULA_CONFLICT"
	StatusComponentError    Status = "COMPONENT_ERROR"
	StatusInvalidFormula    Status = "INVALID_FORMULA"
	StatusFormulaNotFound   Status = "FORMULA_NOT_FOUND"
	StatusSpotNotCalculated Status = "SPOT_NOT_CALCULATED"
	// StatusManual — применена ручная цена (состояние UI-сценария, не эталона).
	StatusManual Status = "MANUAL"
)

// MatchScope — как формула сматчилась со строкой.
type MatchScope string

const (
	MatchScopeMaterial MatchScope = "material"
	MatchScopeGroupM   MatchScope = "group_m"
)

// SelectionReason — причина выбора кандидата.
type SelectionReason string

const (
	SelectionActualSuccessful SelectionReason = "ACTUAL_SUCCESSFUL_FORMULA"
	SelectionLatestExpired    SelectionReason = "LATEST_EXPIRED_SUCCESSFUL_FORMULA"
	SelectionNoSuccessful     SelectionReason = "NO_SUCCESSFUL_FORMULA"
	SelectionTieBreak         SelectionReason = "TECHNICAL_TIE_BREAK"
	SelectionUserSelected     SelectionReason = "USER_SELECTED"
)

// ComponentValue — разрешённое значение компонента формулы с расшифровкой источника.
type ComponentValue struct {
	VarName     string
	TypeCode    string
	TypeLabel   string
	Value       *float64
	Source      string
	QuoteName   string
	QuoteCode   *int64
	SourceDate  *time.Time
	VersionType string
	DateGapDays *int
	Warning     string
	Error       string
}

// CandidateResult — рассчитанный кандидат-формула для строки спроса.
type CandidateResult struct {
	DemandKey            string
	RowID                int64
	Period               time.Time
	FormulaID            string
	FormulaText          string
	FormulaCurrency      string
	DemandCurrency       string
	MatchScope           MatchScope
	CreatedAt            time.Time
	ValidFrom            time.Time
	ValidTo              time.Time
	IsFormulaActual      bool
	Status               Status
	PriceFormulaCurrency *float64
	Price                *float64
	Components           []ComponentValue
	Conversion           *Conversion
	Warning              string
	Error                string

	// Поля выбора (заполняются после select).
	IsSelected         bool
	SelectionReason    SelectionReason
	EqualPriorityCount int
	RequiresReview     bool
}

// Conversion — детали конвертации цены из валюты формулы в валюту строки.
type Conversion struct {
	FromCurrency string
	ToCurrency   string
	FromRate     *float64
	ToRate       *float64
	RateDate     *time.Time
	VersionType  string
}

// CalcRow — итоговая строка результата расчёта.
type CalcRow struct {
	DemandKey          string
	RowID              int64
	Period             time.Time
	ClientID           string
	ClientName         string
	MaterialID         int64
	MaterialName       string
	MaterialGroupM     string
	Contract           string
	Currency           string
	Forecast           *float64
	Status             Status
	Price              *float64
	CandidateCount     int
	EqualPriorityCount int
	RequiresReview     bool
	Warning            string
	Error              string
}
