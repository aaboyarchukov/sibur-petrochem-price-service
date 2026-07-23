// Package calculations — жизненный цикл расчёта: запуск, прогресс, строки,
// расшифровка, ручные правки, KPI, сводный документ. Состояние расчётов
// живёт в памяти процесса (MVP): рестарт — новый расчёт.
package calculations

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"sibur-petrochem-price-service/internal/domain"
	"sibur-petrochem-price-service/internal/service/pricing"
)

// AnalystName — аналитик MVP (многопользовательность вне скоупа).
const AnalystName = "А. Смирнов"

const partName = "Ваш участок"

// AllPeriods — расчёт по всему горизонту спроса (без фильтра по месяцу).
const AllPeriods = "all"

// SourcesLoader — источники расчёта (реализация — repository/postgres).
type SourcesLoader interface {
	LoadSources(ctx context.Context) (domain.Sources, error)
}

type Service struct {
	loader SourcesLoader
	engine *pricing.Engine

	mu    sync.RWMutex
	calcs map[string]*calculation
	order []string
}

// calculation — расчёт с мутационным состоянием поверх результата движка.
type calculation struct {
	mu          sync.RWMutex
	id          string
	period      string
	createdAt   time.Time
	finishedAt  time.Time
	result      pricing.RunResult
	manual      map[string]float64 // demand_key → ручная цена
	selected    map[string]string  // demand_key → выбранная formula_id
	submitted   bool
	submittedAt time.Time
}

func New(loader SourcesLoader, engine *pricing.Engine) *Service {
	return &Service{
		loader: loader,
		engine: engine,
		calcs:  map[string]*calculation{},
	}
}

// Reset — удаление всех расчётов сессии (правки и участки пропадают вместе с ними).
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calcs = map[string]*calculation{}
	s.order = nil
}

// Create — загрузка источников, прогон движка, регистрация расчёта.
// period == AllPeriods — весь горизонт спроса; иначе фильтр по месяцу (YYYY-MM).
// Расчёт синхронный (сотни-тысячи строк за доли секунды); контрактный статус — done.
func (s *Service) Create(ctx context.Context, period string) (Info, error) {
	src, err := s.loader.LoadSources(ctx)
	if err != nil {
		return Info{}, fmt.Errorf("load sources: %w", err)
	}

	if period != AllPeriods {
		src.Ssp = filterByPeriod(src.Ssp, period)
	}
	if len(src.Ssp) == 0 {
		return Info{}, fmt.Errorf("%w: нет строк спроса за период %s", domain.ErrSourcesNotLoaded, period)
	}

	calc := &calculation{
		id:        uuid.NewString(),
		period:    period,
		createdAt: time.Now().UTC(),
		manual:    map[string]float64{},
		selected:  map[string]string{},
	}
	calc.result = s.engine.Run(src, nil)
	calc.finishedAt = time.Now().UTC()

	s.mu.Lock()
	s.calcs[calc.id] = calc
	s.order = append(s.order, calc.id)
	s.mu.Unlock()

	return calc.info(), nil
}

// filterByPeriod — строки спроса за месяц period (формат YYYY-MM).
func filterByPeriod(rows []domain.SspRow, period string) []domain.SspRow {
	filtered := make([]domain.SspRow, 0, len(rows))
	for _, row := range rows {
		if row.Period.Format("2006-01") == period {
			filtered = append(filtered, row)
		}
	}

	return filtered
}

func (s *Service) Get(id string) (Info, error) {
	calc, err := s.find(id)
	if err != nil {
		return Info{}, err
	}

	return calc.info(), nil
}

// Subscribe — SSE-поток прогресса. Расчёт синхронный, поэтому подписчик
// сразу получает снапшот-событие done, после чего канал закрывается.
func (s *Service) Subscribe(id string) (events <-chan ProgressEvent, unsubscribe func(), err error) {
	calc, err := s.find(id)
	if err != nil {
		return nil, nil, err
	}

	stream := make(chan ProgressEvent, 1)
	info := calc.info()
	stream <- ProgressEvent{
		Status:    info.Status,
		Processed: info.Processed,
		Total:     info.Total,
		Percent:   percentOf(info.Processed, info.Total),
	}
	close(stream)

	return stream, func() {}, nil
}

func (s *Service) find(id string) (*calculation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	calc, ok := s.calcs[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", domain.ErrCalculationNotFound, id)
	}

	return calc, nil
}

func (s *Service) byPeriod(period string) []*calculation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []*calculation
	for _, id := range s.order {
		if calc := s.calcs[id]; calc.period == period {
			out = append(out, calc)
		}
	}

	return out
}

func (c *calculation) info() Info {
	c.mu.RLock()
	defer c.mu.RUnlock()

	finished := c.finishedAt

	return Info{
		ID:         c.id,
		Period:     c.period,
		Status:     "done",
		Processed:  len(c.result.Rows),
		Total:      len(c.result.Rows),
		CreatedAt:  c.createdAt,
		FinishedAt: &finished,
	}
}

func percentOf(part, total int) int {
	const full = 100
	if total == 0 {
		return 0
	}

	return part * full / total
}
