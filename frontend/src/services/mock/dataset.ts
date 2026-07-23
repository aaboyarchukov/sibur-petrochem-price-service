// Детерминированный датасет для MockPricingApi. Без Math.random — seeded LCG,
// одинаковый набор между запусками. Формы данных повторяют результат эталонного
// прогона pricing_pipeline_fixed.py и колонки прототипа.
import type {
  RowStatus,
  DealType,
  MatchScope,
  FormulaComponent,
  Source,
  ConsolidatedPart,
} from '@/types'

export const CALCULATION_PERIOD = '2026-06'
export const MY_ANALYST = 'А. Смирнов'

// Кандидат-формула строки (цена уже посчитана в валюте строки).
export interface FormulaCandidate {
  formula_id: string
  formula_text: string
  variables: string[]
  formula_currency: string
  match_scope: MatchScope
  valid_from: string
  valid_to: string
  created_on: string
  is_actual: boolean
  is_extended: boolean
  price: number | null
  calc_error: string | null
  components: FormulaComponent[]
}

// Полная запись строки — источник для CalculationRow и RowDetails.
export interface RowRecord {
  row_id: string
  period: string
  client_id: string
  client_name: string
  material_id: string
  material_name: string
  material_group_m: string | null
  deal_type: DealType
  currency: string
  volume: number | null
  base_status: RowStatus
  reference_price: number | null
  candidates: FormulaCandidate[]
  default_formula_id: string | null
}

const MATERIALS: [string, string, number][] = [
  ['226814', 'Полипропилен, марка PP H030 GP/3', 108600],
  ['1005972', 'Полиэтилен, марка HD 03580 SB', 96200],
  ['1182578', 'Полистирол, марка 825ES', 121500],
  ['1233003', 'Полипропилен, марка PP H253 FF/7', 103400],
  ['1353711', 'Полиэтилентерефталат, ПЭТФ Т-96', 112800],
  ['211770', 'Полипропилен, Полиом PP H253 FF', 98700],
  ['1197228', 'Полистирол, марка 585', 89600],
  ['1380972', 'Полиэтилен, марка LD 15803-020', 92400],
]
const CLIENTS: [string, string][] = [
  ['CL-10328', 'Клиент 328'],
  ['CL-10126', 'Клиент 126'],
  ['CL-10101', 'Клиент 101'],
  ['CL-10364', 'Клиент 364'],
  ['CL-10152', 'Клиент 152'],
  ['CL-10342', 'Клиент 342'],
  ['CL-10009', 'Клиент 009'],
  ['CL-10145', 'Клиент 145'],
]
const CURRENCIES = ['RUB', 'RUB', 'RUB', 'USD', 'RUB']
const QUOTES = ['CPT_MOSCOW_109', 'CFR_TUR_CURRENT', 'CFR_CHINA', 'SPOT_FOB_NEASIA', 'ICIS_STYRENE']

// Имена котировок из quote_mapping (техимя → человекочитаемое имя).
const QUOTE_NAMES: Record<string, string> = {
  CPT_MOSCOW_109: 'PP raffia CPT Moscow',
  CFR_TUR_CURRENT: 'PE film CFR Turkey current',
  CFR_CHINA: 'PP homo CFR China',
  SPOT_FOB_NEASIA: 'Styrene spot FOB NE Asia',
  ICIS_STYRENE: 'ICIS Styrene Europe',
}

// План статусов: гарантирует присутствие каждого RowStatus в наборе.
const STATUS_PLAN: Record<number, RowStatus> = {
  3: 'no_formula',
  9: 'no_formula',
  4: 'component_error',
  14: 'component_error',
  22: 'component_error',
  6: 'formula_conflict',
  17: 'formula_conflict',
  28: 'formula_conflict',
  11: 'calculated_expired',
  24: 'calculated_expired',
  19: 'invalid_formula',
  8: 'manual',
  26: 'manual',
  5: 'spot_not_calculated',
  20: 'spot_not_calculated',
}

function makeLcg(seed: number): () => number {
  let s = seed
  return () => {
    s = (s * 1103515245 + 12345) & 0x7fffffff
    return s / 0x7fffffff
  }
}

function makeComponents(quote: string, currency: string, missing: boolean): FormulaComponent[] {
  const comps: FormulaComponent[] = [
    {
      var_name: quote,
      type: 'quote',
      type_label: 'Котировка',
      status: missing ? 'error' : 'ok',
      value: missing ? null : 1408,
      source: 'quotes.csv',
      quote_name: QUOTE_NAMES[quote] ?? quote,
      quote_code: 938751651,
      value_date: missing ? null : '2026-05-01',
      version_type: missing ? null : 'Факт',
      date_gap_days: missing ? null : 31,
      warning: null,
      error: missing ? 'нет значений котировки на дату периода' : null,
    },
    {
      var_name: 'L',
      type: 'logistics',
      type_label: 'Логистика',
      status: 'ok',
      value: 3650,
      source: 'formula_components.csv',
      quote_name: null,
      quote_code: null,
      value_date: null,
      version_type: null,
      date_gap_days: null,
      warning: null,
      error: null,
    },
    {
      var_name: 'H1',
      type: 'constant',
      type_label: 'Константа (H)',
      status: 'ok',
      value: 0.8906,
      source: 'formula_components.csv',
      quote_name: null,
      quote_code: null,
      value_date: null,
      version_type: null,
      date_gap_days: null,
      warning: null,
      error: null,
    },
    {
      var_name: 'D',
      type: 'markup',
      type_label: 'Надбавка',
      status: 'ok',
      value: 0.7154,
      source: 'formula_components.csv',
      quote_name: null,
      quote_code: null,
      value_date: null,
      version_type: null,
      date_gap_days: null,
      warning: null,
      error: null,
    },
    {
      var_name: 'SPOT',
      type: 'constant',
      type_label: 'Константа (H)',
      status: 'ok',
      value: 98728.12,
      source: 'formula_components.csv',
      quote_name: null,
      quote_code: null,
      value_date: null,
      version_type: null,
      date_gap_days: null,
      warning: null,
      error: null,
    },
  ]
  if (currency !== 'RUB') {
    comps.push({
      var_name: 'USD_RUB',
      type: 'currency_rate',
      type_label: 'Курс валют',
      status: 'ok',
      value: 89.42,
      source: 'currency_rates.csv',
      quote_name: null,
      quote_code: null,
      value_date: '2026-06-15',
      version_type: 'Факт',
      date_gap_days: 0,
      warning: null,
      error: null,
    })
  }
  return comps
}

function makeCandidate(
  quote: string,
  currency: string,
  code: string,
  createdOn: string,
  validFrom: string,
  validTo: string,
  price: number | null,
  opts: { actual?: boolean; extended?: boolean; scope?: MatchScope; error?: string | null } = {},
): FormulaCandidate {
  const missing = price == null && !opts.error
  return {
    formula_id: code,
    formula_text: `IF ( ( ( ${quote} - L ) / H1 ) * D < SPOT , ( ( ${quote} - L ) / H1 ) * D , SPOT )`,
    variables: [quote, 'L', 'H1', 'D', 'SPOT'],
    formula_currency: currency,
    match_scope: opts.scope ?? 'group_m',
    valid_from: validFrom,
    valid_to: validTo,
    created_on: createdOn,
    is_actual: opts.actual ?? true,
    is_extended: opts.extended ?? false,
    price,
    calc_error: opts.error ?? null,
    components: makeComponents(quote, currency, missing),
  }
}

// Строится один раз; MockPricingApi держит поверх мутационное состояние.
export function buildRows(): RowRecord[] {
  const rnd = makeLcg(20260601)
  const pick = <T>(arr: T[]): T => arr[Math.floor(rnd() * arr.length)] as T
  const rows: RowRecord[] = []

  for (let i = 0; i < 42; i++) {
    const [materialId, materialName, base] = MATERIALS[i % MATERIALS.length]!
    const [clientId, clientName] = pick(CLIENTS)
    const currency = pick(CURRENCIES)
    const quote = pick(QUOTES)
    const volume = Math.round((15 + rnd() * 600) * 5)
    const reference = Math.round(base * (0.92 + rnd() * 0.16))
    const status: RowStatus = STATUS_PLAN[i] ?? 'calculated'
    const dealType: DealType = status === 'spot_not_calculated' ? 'SPOT' : 'Formula'
    const rowId = String(1023422 + i * 71)
    const price = Math.round(reference * (0.97 + rnd() * 0.06))

    let candidates: FormulaCandidate[] = []
    let defaultFormulaId: string | null = null

    if (status === 'no_formula' || status === 'spot_not_calculated') {
      candidates = []
    } else if (status === 'formula_conflict') {
      candidates = [
        makeCandidate(quote, currency, `Z9000${28500 + i}`, '2026-05-20', '2026-06-01', '2026-12-31', price),
        makeCandidate(quote, currency, `Z9000${11000 + i}`, '2026-05-20', '2026-03-01', '2026-12-31', Math.round(price * 1.04)),
      ]
      defaultFormulaId = candidates[0]!.formula_id
    } else if (status === 'component_error') {
      candidates = [
        makeCandidate(quote, currency, `Z9000${26000 + i}`, '2026-04-01', '2026-06-01', '2026-12-31', null),
      ]
      defaultFormulaId = candidates[0]!.formula_id
    } else if (status === 'invalid_formula') {
      const bad = makeCandidate(quote, currency, `Z9000${25000 + i}`, '2026-04-01', '2026-06-01', '2026-12-31', null, {
        error: 'синтаксическая ошибка формулы: неожиданный токен',
      })
      bad.formula_text = `( ( ${quote} - L ) / / H1 ) * D`
      candidates = [bad]
      defaultFormulaId = bad.formula_id
    } else if (status === 'calculated_expired') {
      candidates = [
        makeCandidate(quote, currency, `Z9000${23000 + i}`, '2026-01-10', '2026-02-01', '2026-05-31', price, {
          actual: false,
          extended: true,
        }),
      ]
      defaultFormulaId = candidates[0]!.formula_id
    } else {
      // calculated и manual (manual — предзаданная ручная правка поверх calculated).
      candidates = [
        makeCandidate(quote, currency, `Z9000${20000 + i}`, '2026-04-15', '2026-02-01', '2027-12-31', price, {
          scope: i % 3 === 0 ? 'material' : 'group_m',
        }),
      ]
      defaultFormulaId = candidates[0]!.formula_id
    }

    rows.push({
      row_id: rowId,
      period: CALCULATION_PERIOD,
      client_id: clientId!,
      client_name: clientName!,
      material_id: materialId,
      material_name: materialName,
      material_group_m: `MT0000${100 + (i % 8)}`,
      deal_type: dealType,
      currency,
      volume,
      base_status: status,
      reference_price: status === 'spot_not_calculated' ? null : reference,
      candidates,
      default_formula_id: defaultFormulaId,
    })
  }
  return rows
}

// Восемь источников: 2 пользовательских + 6 справочников.
export function buildSources(): Source[] {
  return [
    { key: 'ssp', name: 'Прогноз спроса', file_name: 'ssp.csv', kind: 'uploaded', status: 'loaded', row_count: 3076, issues: [] },
    { key: 'formulas', name: 'Каталог формул', file_name: 'formulas.csv', kind: 'uploaded', status: 'loaded', row_count: 576, issues: [] },
    { key: 'formula_components', name: 'Компоненты формул', file_name: 'formula_components.csv', kind: 'reference', status: 'loaded', row_count: 4101, issues: [] },
    { key: 'term_types', name: 'Типы термов', file_name: 'term_types.csv', kind: 'reference', status: 'loaded', row_count: 11, issues: [] },
    { key: 'quotes', name: 'Котировки', file_name: 'quotes.csv', kind: 'reference', status: 'loaded', row_count: 298, issues: [] },
    { key: 'quote_mapping', name: 'Маппинг котировок', file_name: 'quote_mapping.csv', kind: 'reference', status: 'loaded', row_count: 70, issues: [] },
    { key: 'currency_rates', name: 'Курсы валют', file_name: 'currency_rates.csv', kind: 'reference', status: 'loaded', row_count: 1581, issues: [] },
    { key: 'material_groups', name: 'Группы материалов', file_name: 'material_groups.csv', kind: 'reference', status: 'loaded', row_count: 614, issues: [] },
  ]
}

// Участки других аналитиков в сводном документе (статические mock-данные).
export const OTHER_PARTS: ConsolidatedPart[] = [
  {
    calculation_id: 'calc-petrov',
    analyst_name: 'И. Петров',
    part_name: 'Ароматика и гликоли',
    status: 'joined',
    row_count: 16,
    priced_pct: 81,
    submitted_at: '2026-06-20T14:12:00Z',
  },
  {
    calculation_id: 'calc-kozlova',
    analyst_name: 'М. Козлова',
    part_name: 'Каучуки и латексы',
    status: 'review',
    row_count: 13,
    priced_pct: 74,
    submitted_at: null,
  },
]
