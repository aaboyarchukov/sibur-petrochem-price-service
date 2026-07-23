package domain

import (
	"strconv"
	"time"
)

// SspRow — строка прогноза спроса (одна = клиент+продукт+месяц+объём).
type SspRow struct {
	RowID        int64
	Period       time.Time
	CustomerASV  string
	Customer     string
	MaterialID   int64
	MaterialName string
	Contract     string // Formula | SPOT
	Currency     string
	Forecast     *float64
	Country      string
	Region       string
	Market       string
	ClientID     string
	ClientName   string
}

// DemandKey — уникальный ключ строки спроса: row_id повторяется по месяцам.
func (r SspRow) DemandKey() string {
	return strconv.FormatInt(r.RowID, 10) + "|" + r.Period.Format("2006-01-02")
}

// IsFormula — строка участвует в подборе формул (contract = Formula).
func (r SspRow) IsFormula() bool {
	return r.Contract == "Formula"
}

// Formula — договорная формула из каталога.
type Formula struct {
	FormulaID       string
	Text            string
	MaterialGroupM  string // пусто — формула задана прямо на материал
	BusinessPartner string
	ValidFrom       time.Time
	ValidTo         time.Time
	CreatedAt       time.Time
	Inactive        bool
	PriceType       string // SAP-классификатор; НЕ джойнится с ssp.Contract
	DocCurrency     string
	MaterialID      *int64 // nil — формула задана на группу M
	ClientID        string
	ClientName      string
}

// FormulaComponent — терм формулы из состава.
type FormulaComponent struct {
	FormulaID string
	TermNo    int
	VarName   string
	ValidFrom time.Time
	ValidTo   time.Time
	TypeCode  string // 1 котировка, 5 курс, H/A/B/C/D/E константы, 7 прайс-лист, 0/6 группировки
	Value     *float64
	Currency  string
	QuoteName string // тип 1 — SAP-имя котировки; тип 5 — технический ZF-идентификатор
}

// TermType — расшифровка типа терма.
type TermType struct {
	TypeCode  string
	TypeLabel string
	Category  string
}

// Quote — значение рыночной котировки.
type Quote struct {
	QuoteType  string // Факт | ОФ | ППР
	QuoteName  string
	QuoteCode  int64
	QuoteDate  time.Time
	Currency   string
	Value      float64
	TechLoadTS *time.Time
}

// QuoteMapping — связь SAP-имени котировки с ID в озере данных.
type QuoteMapping struct {
	QuoteName string
	LakeID    *int64
	Currency  string
}

// CurrencyRate — курс валюты к RUB на день.
type CurrencyRate struct {
	Currency    string
	CalDay      time.Time
	VersionType string // Факт | ОФ | ППР
	Value       float64
}

// MaterialGroup — вхождение материала в группу M с окном действия.
type MaterialGroup struct {
	GroupM       string
	MaterialID   int64
	MaterialName string
	ValidFrom    time.Time
	ValidTo      time.Time
}

// Sources — все источники, загруженные в память для расчёта.
type Sources struct {
	Ssp            []SspRow
	Formulas       []Formula
	Components     []FormulaComponent
	TermTypes      []TermType
	Quotes         []Quote
	QuoteMapping   []QuoteMapping
	CurrencyRates  []CurrencyRate
	MaterialGroups []MaterialGroup
}
