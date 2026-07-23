package pricing

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"sibur-petrochem-price-service/internal/domain"
	"sibur-petrochem-price-service/internal/service/pricing/expr"
)

const conflictMessage = "Найдено несколько равноприоритетных формул. " +
	"Автоматически выбрана формула с минимальным formula_id. " +
	"Нужно выбрать формулу"

// RunResult — результат прогона движка: строки + все кандидаты по ключу спроса.
type RunResult struct {
	Rows            []domain.CalcRow
	candidatesByKey map[string][]*domain.CandidateResult
	Horizon         time.Time
	idx             *indexes
}

// RateToRUB — курс валюты к RUB на дату (для контрольной суммы KPI).
func (r RunResult) RateToRUB(currency string, period time.Time) (float64, error) {
	value, _, err := resolveRate(currency, period, r.idx)

	return value, err
}

func (r RunResult) Candidates(demandKey string) []domain.CandidateResult {
	group := r.candidatesByKey[demandKey]
	out := make([]domain.CandidateResult, 0, len(group))
	for _, candidate := range group {
		out = append(out, *candidate)
	}

	return out
}

// EachCandidate — обход всех кандидатов (для KPI «% формул без ошибок»).
func (r RunResult) EachCandidate(fn func(demandKey string, candidate domain.CandidateResult)) {
	for key, group := range r.candidatesByKey {
		for _, candidate := range group {
			fn(key, *candidate)
		}
	}
}

// candidateMatch — сматченная пара «строка спроса × формула».
type candidateMatch struct {
	row     domain.SspRow
	formula domain.Formula
	scope   domain.MatchScope
}

// Run — полный прогон: подбор кандидатов → расчёт каждого → выбор → строки.
// Ошибка одной строки не прерывает расчёт остальных.
func (e *Engine) Run(src domain.Sources, onProgress func(processed, total int)) RunResult {
	horizon := maxPeriod(src.Ssp)
	idx := buildIndexes(src, horizon)
	matches := buildMatches(src)

	byKey := make(map[string][]*domain.CandidateResult, len(matches))
	total := len(src.Ssp)
	for i, row := range src.Ssp {
		key := row.DemandKey()
		for _, match := range matches[key] {
			byKey[key] = append(byKey[key], e.calculateCandidate(match, idx))
		}
		if onProgress != nil {
			onProgress(i+1, total)
		}
	}

	outcomes := make(map[string]selectionOutcome, len(byKey))
	for key, group := range byKey {
		outcomes[key] = selectForKey(group)
	}

	return RunResult{
		Rows:            buildRows(src.Ssp, byKey, outcomes),
		candidatesByKey: byKey,
		Horizon:         horizon,
		idx:             idx,
	}
}

func maxPeriod(rows []domain.SspRow) time.Time {
	var horizon time.Time
	for _, row := range rows {
		if row.Period.After(horizon) {
			horizon = row.Period
		}
	}

	return horizon
}

// buildMatches — подбор кандидатов: только строки Formula, формулы не inactive,
// матч напрямую по материалу или через группу M (с проверкой окна группы),
// будущие формулы отсекаются, просроченные участвуют.
func buildMatches(src domain.Sources) map[string][]candidateMatch {
	formulasByClient := make(map[string][]domain.Formula, len(src.Formulas))
	for _, formula := range src.Formulas {
		if formula.Inactive {
			continue
		}
		formulasByClient[formula.ClientID] = append(formulasByClient[formula.ClientID], formula)
	}

	groupEntries := make(map[string][]domain.MaterialGroup, len(src.MaterialGroups))
	for _, group := range src.MaterialGroups {
		groupEntries[group.GroupM] = append(groupEntries[group.GroupM], group)
	}

	matches := make(map[string][]candidateMatch, len(src.Ssp))
	for _, row := range src.Ssp {
		if !row.IsFormula() {
			continue
		}
		matches[row.DemandKey()] = matchRow(row, formulasByClient[row.ClientID], groupEntries)
	}

	return matches
}

// matchRow — кандидаты одной строки: прямой матч первым (scope material побеждает при дубле).
func matchRow(row domain.SspRow, formulas []domain.Formula, groupEntries map[string][]domain.MaterialGroup) []candidateMatch {
	var out []candidateMatch
	seen := map[string]bool{}

	for _, formula := range formulas {
		if formula.MaterialID == nil || *formula.MaterialID != row.MaterialID {
			continue
		}
		if row.Period.Before(formula.ValidFrom) || seen[formula.FormulaID] {
			continue
		}
		seen[formula.FormulaID] = true
		out = append(out, candidateMatch{row: row, formula: formula, scope: domain.MatchScopeMaterial})
	}

	// Матч через группу M с проверкой окна действия вхождения в группу.
	for _, formula := range formulas {
		if formula.MaterialGroupM == "" || row.Period.Before(formula.ValidFrom) || seen[formula.FormulaID] {
			continue
		}
		if !groupContains(groupEntries[formula.MaterialGroupM], row) {
			continue
		}
		seen[formula.FormulaID] = true
		out = append(out, candidateMatch{row: row, formula: formula, scope: domain.MatchScopeGroupM})
	}

	return out
}

func groupContains(entries []domain.MaterialGroup, row domain.SspRow) bool {
	for _, group := range entries {
		if group.MaterialID != row.MaterialID {
			continue
		}
		if row.Period.Before(group.ValidFrom) || row.Period.After(group.ValidTo) {
			continue
		}

		return true
	}

	return false
}

func (e *Engine) calculateCandidate(match candidateMatch, idx *indexes) *domain.CandidateResult {
	period := match.row.Period
	isActual := !period.Before(match.formula.ValidFrom) && !period.After(match.formula.ValidTo)

	candidate := &domain.CandidateResult{
		DemandKey:       match.row.DemandKey(),
		RowID:           match.row.RowID,
		Period:          period,
		FormulaID:       match.formula.FormulaID,
		FormulaText:     match.formula.Text,
		FormulaCurrency: match.formula.DocCurrency,
		DemandCurrency:  match.row.Currency,
		MatchScope:      match.scope,
		CreatedAt:       match.formula.CreatedAt,
		ValidFrom:       match.formula.ValidFrom,
		ValidTo:         match.formula.ValidTo,
		IsFormulaActual: isActual,
	}
	if !isActual {
		candidate.Warning = "Формула неактуальна на дату расчёта"
	}

	values, details, componentErrors := resolveComponents(match.formula.FormulaID, period, idx)
	candidate.Components = details

	if len(componentErrors) > 0 {
		candidate.Status = domain.StatusComponentError
		candidate.Error = strings.Join(componentErrors, " | ")

		return candidate
	}

	priceFormulaCurrency, err := e.eval.Evaluate(match.formula.Text, values)
	if err != nil {
		candidate.Status = domain.StatusInvalidFormula
		candidate.Error = evalErrorText(err)

		return candidate
	}

	price, conversion, err := convertCurrency(priceFormulaCurrency, match.formula.DocCurrency, match.row.Currency, period, idx)
	if err != nil {
		// как в эталоне: ошибка конвертации попадает в общий try расчёта
		candidate.Status = domain.StatusInvalidFormula
		candidate.Error = componentErrorText(err)

		return candidate
	}

	candidate.Status = domain.StatusCalculated
	candidate.PriceFormulaCurrency = &priceFormulaCurrency
	candidate.Price = &price
	candidate.Conversion = conversion

	return candidate
}

func convertCurrency(value float64, from, to string, period time.Time, idx *indexes) (float64, *domain.Conversion, error) {
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))
	if from == to {
		return value, nil, nil
	}

	fromRate, fromInfo, err := resolveRate(from, period, idx)
	if err != nil {
		return 0, nil, err
	}

	toRate, _, err := resolveRate(to, period, idx)
	if err != nil {
		return 0, nil, err
	}

	rateDate := fromInfo.Date

	return value * fromRate / toRate, &domain.Conversion{
		FromCurrency: from,
		ToCurrency:   to,
		FromRate:     &fromRate,
		ToRate:       &toRate,
		RateDate:     &rateDate,
		VersionType:  fromInfo.VersionType,
	}, nil
}

func evalErrorText(err error) string {
	return strings.TrimPrefix(err.Error(), expr.ErrEvaluation.Error()+": ")
}

// selectionOutcome — итог выбора кандидата для строки спроса.
type selectionOutcome struct {
	top            *domain.CandidateResult
	finalStatus    domain.Status
	reason         domain.SelectionReason
	warning        string
	errText        string
	equalCount     int
	requiresReview bool
}

func scopePriority(scope domain.MatchScope) int {
	if scope == domain.MatchScopeMaterial {
		return 0
	}

	return 1
}

// actualLess — порядок выбора среди актуальных: scope, created_at desc, valid_from desc, formula_id asc.
func actualLess(a, b *domain.CandidateResult) bool {
	if scopePriority(a.MatchScope) != scopePriority(b.MatchScope) {
		return scopePriority(a.MatchScope) < scopePriority(b.MatchScope)
	}
	if !a.CreatedAt.Equal(b.CreatedAt) {
		return a.CreatedAt.After(b.CreatedAt)
	}
	if !a.ValidFrom.Equal(b.ValidFrom) {
		return a.ValidFrom.After(b.ValidFrom)
	}

	return a.FormulaID < b.FormulaID
}

// expiredLess — порядок среди просроченных: сперва максимальный valid_to.
func expiredLess(a, b *domain.CandidateResult) bool {
	if !a.ValidTo.Equal(b.ValidTo) {
		return a.ValidTo.After(b.ValidTo)
	}

	return actualLess(a, b)
}

func sortedBy(group []*domain.CandidateResult, less func(a, b *domain.CandidateResult) bool) []*domain.CandidateResult {
	pool := make([]*domain.CandidateResult, len(group))
	copy(pool, group)
	sort.SliceStable(pool, func(i, j int) bool { return less(pool[i], pool[j]) })

	return pool
}

// selectForKey — выбор применяемого кандидата после расчёта всех (порт select_candidate_results).
func selectForKey(group []*domain.CandidateResult) selectionOutcome {
	successful := filterCandidates(group, func(c *domain.CandidateResult) bool {
		return c.Status == domain.StatusCalculated
	})
	actualSuccessful := filterCandidates(successful, func(c *domain.CandidateResult) bool {
		return c.IsFormulaActual
	})

	outcome := pickTop(group, successful, actualSuccessful)

	outcome.errText = outcome.top.Error
	outcome.requiresReview = outcome.equalCount > 1 && outcome.top.Status == domain.StatusCalculated
	if outcome.requiresReview {
		outcome.finalStatus = domain.StatusFormulaConflict
		outcome.reason = domain.SelectionTieBreak
		outcome.errText = conflictMessage
		outcome.warning = combineMessages(outcome.warning, conflictMessage)
	}

	outcome.top.IsSelected = true
	outcome.top.SelectionReason = outcome.reason
	outcome.top.EqualPriorityCount = outcome.equalCount
	outcome.top.RequiresReview = outcome.requiresReview

	return outcome
}

func pickTop(group, successful, actualSuccessful []*domain.CandidateResult) selectionOutcome {
	if len(actualSuccessful) > 0 {
		pool := sortedBy(actualSuccessful, actualLess)
		top := pool[0]
		equal := filterCandidates(pool, func(c *domain.CandidateResult) bool {
			return scopePriority(c.MatchScope) == scopePriority(top.MatchScope) &&
				c.CreatedAt.Equal(top.CreatedAt) && c.ValidFrom.Equal(top.ValidFrom)
		})

		return selectionOutcome{
			top:         top,
			finalStatus: domain.StatusCalculated,
			reason:      domain.SelectionActualSuccessful,
			warning:     top.Warning,
			equalCount:  len(equal),
		}
	}

	expiredSuccessful := filterCandidates(successful, func(c *domain.CandidateResult) bool {
		return !c.IsFormulaActual
	})
	if len(expiredSuccessful) > 0 {
		pool := sortedBy(expiredSuccessful, expiredLess)
		top := pool[0]
		equal := filterCandidates(pool, func(c *domain.CandidateResult) bool {
			return c.ValidTo.Equal(top.ValidTo) &&
				scopePriority(c.MatchScope) == scopePriority(top.MatchScope) &&
				c.CreatedAt.Equal(top.CreatedAt) && c.ValidFrom.Equal(top.ValidFrom)
		})

		const dateLayout = "02.01.2006"
		warning := fmt.Sprintf(
			"Использована неактуальная формула: срок действия закончился %s, дата расчёта — %s.",
			top.ValidTo.Format(dateLayout), top.Period.Format(dateLayout),
		)

		return selectionOutcome{
			top:         top,
			finalStatus: domain.StatusCalculatedExpired,
			reason:      domain.SelectionLatestExpired,
			warning:     warning,
			equalCount:  len(equal),
		}
	}

	// Ни одна не рассчиталась: ошибка наиболее приоритетного кандидата, цена пустая.
	actualErrors := filterCandidates(group, func(c *domain.CandidateResult) bool {
		return c.IsFormulaActual
	})
	pool := sortedBy(group, expiredLess)
	if len(actualErrors) > 0 {
		pool = sortedBy(actualErrors, actualLess)
	}
	top := pool[0]

	return selectionOutcome{
		top:         top,
		finalStatus: top.Status,
		reason:      domain.SelectionNoSuccessful,
		warning:     top.Warning,
		equalCount:  1,
	}
}

func filterCandidates(group []*domain.CandidateResult, keep func(*domain.CandidateResult) bool) []*domain.CandidateResult {
	var out []*domain.CandidateResult
	for _, candidate := range group {
		if keep(candidate) {
			out = append(out, candidate)
		}
	}

	return out
}

func combineMessages(messages ...string) string {
	var parts []string
	for _, message := range messages {
		trimmed := strings.TrimSpace(message)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}

	return strings.Join(parts, " | ")
}

func buildRows(ssp []domain.SspRow, byKey map[string][]*domain.CandidateResult, outcomes map[string]selectionOutcome) []domain.CalcRow {
	rows := make([]domain.CalcRow, 0, len(ssp))
	for _, demand := range ssp {
		key := demand.DemandKey()
		row := domain.CalcRow{
			DemandKey:    key,
			RowID:        demand.RowID,
			Period:       demand.Period,
			ClientID:     demand.ClientID,
			ClientName:   demand.ClientName,
			MaterialID:   demand.MaterialID,
			MaterialName: demand.MaterialName,
			Contract:     demand.Contract,
			Currency:     demand.Currency,
			Forecast:     demand.Forecast,
		}

		switch {
		case strings.EqualFold(demand.Contract, "SPOT"):
			row.Status = domain.StatusSpotNotCalculated
		case len(byKey[key]) == 0:
			row.Status = domain.StatusFormulaNotFound
		default:
			outcome := outcomes[key]
			row.Status = outcome.finalStatus
			row.Price = outcome.top.Price
			row.CandidateCount = len(byKey[key])
			row.EqualPriorityCount = outcome.equalCount
			row.RequiresReview = outcome.requiresReview
			row.Warning = outcome.warning
			row.Error = outcome.errText
		}

		rows = append(rows, row)
	}

	return rows
}
