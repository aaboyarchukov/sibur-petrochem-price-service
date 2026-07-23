package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/godepo/groat"
	"github.com/godepo/groat/integration"
	"github.com/godepo/pgrx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaswdr/faker/v2"

	"sibur-petrochem-price-service/internal/domain"

	sqlc_gen "sibur-petrochem-price-service/internal/generated/sqlc"
)

type (
	Givens struct {
		SspRows       []sqlc_gen.ListSspRow
		FormulaRows   []sqlc_gen.ListFormulasRow
		ComponentRows []sqlc_gen.ListFormulaComponentsRow
		QuoteRows     []sqlc_gen.ListQuotesRow
		MappingRows   []sqlc_gen.ListQuoteMappingRow
		RateRows      []sqlc_gen.ListCurrencyRatesRow
		GroupRows     []sqlc_gen.ListMaterialGroupsRow

		// Интеграционные сценарии: доменные строки для Replace* и счётчики seed-набора.
		DomainSsp      []domain.SspRow
		DomainFormulas []domain.Formula
		SeedCounts     map[string]int64
	}

	Responses struct {
		Ssp        []domain.SspRow
		Formulas   []domain.Formula
		Components []domain.FormulaComponent
		Quotes     []domain.Quote
		Mapping    []domain.QuoteMapping
		Rates      []domain.CurrencyRate
		Groups     []domain.MaterialGroup

		// Интеграционные сценарии: результат чтения из реальной БД.
		Sources    domain.Sources
		Counts     map[string]int64
		ReplaceErr error
	}

	Deps struct {
		// Пул подключений к изолированной БД теста; инжектится pgrx по groat-тегу.
		DB *pgxpool.Pool `groat:"pgxpool"`
	}

	State struct {
		Given    Givens
		Faker    faker.Faker
		Response Responses
	}
)

// suite — контейнер PostgreSQL на весь пакет; каждому тесту — своя БД с миграциями.
var suite *integration.Container[Deps, State, *Repository]

func TestMain(m *testing.M) {
	const migrationsDir = "../../../db/migrations"

	migrator, err := upMigrator(migrationsDir)
	if err != nil {
		fmt.Printf("can't load migrations: %v\n", err)
		os.Exit(1)
	}

	suite = integration.New[Deps, State, *Repository](m,
		func(t *testing.T) *groat.Case[Deps, State, *Repository] {
			return groat.New[Deps, State, *Repository](t,
				func(t *testing.T, deps Deps) *Repository {
					t.Helper()

					return New(deps.DB)
				},
			)
		},
		pgrx.New[Deps](
			pgrx.WithContainerImage("postgres:17-alpine"),
			pgrx.WithMigrator(migrator),
		),
	)
	os.Exit(suite.Go())
}

// upMigrator — прогон только *.up.sql в порядке номеров: pgrx.PlainMigrator
// выполняет все файлы каталога подряд, включая down-миграции, что ломает схему.
func upMigrator(dir string) (pgrx.Migrator, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	migrations := make([]string, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", name, err)
		}
		migrations = append(migrations, string(data))
	}

	return func(ctx context.Context, cfg pgrx.MigratorConfig) error {
		for _, migration := range migrations {
			if _, err := cfg.Pool.Exec(ctx, migration); err != nil {
				return fmt.Errorf("apply migration: %w", err)
			}
		}

		return nil
	}, nil
}

// newTestContext — лёгкий контекст для unit-тестов маппинга: без БД и контейнера.
func newTestContext(t *testing.T) *groat.Case[Deps, State, *Repository] {
	t.Helper()

	tc := groat.New[Deps, State, *Repository](t, func(t *testing.T, deps Deps) *Repository {
		t.Helper()

		return New(nil)
	})

	tc.Go()

	tc.State.Faker = faker.New()

	return tc
}

// newIntegrationContext — контекст с реальной БД из suite; faker готов к Given-шагам.
func newIntegrationContext(t *testing.T) *groat.Case[Deps, State, *Repository] {
	t.Helper()

	tc := suite.Case(t)
	tc.State.Faker = faker.New()

	return tc
}
