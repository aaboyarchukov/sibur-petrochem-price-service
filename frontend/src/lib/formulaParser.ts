// Лёгкий парсер выражения формулы: извлекает переменные и функции, проверяет базовый синтаксис.
// Грамматика соответствует эталону: операторы + - * / % **, сравнения, функции IF/RND_X/MIN/MAX,
// переменные с символами $ € ¥ ₱, цифрами и _, имя может начинаться с цифры.
import type { ParsedFormula } from '@/types'

const ALLOWED_FUNCTIONS = new Set(['IF', 'RND_X', 'MIN', 'MAX'])
const TOKEN = /[A-Za-z0-9_$€¥₱]+/g

export function parseFormulaExpression(text: string): ParsedFormula {
  const errors: ParsedFormula['errors'] = []

  // Баланс скобок.
  let depth = 0
  for (let i = 0; i < text.length; i++) {
    if (text[i] === '(') depth++
    else if (text[i] === ')') {
      depth--
      if (depth < 0) {
        errors.push({ message: 'лишняя закрывающая скобка', position: i })
        break
      }
    }
  }
  if (depth > 0) errors.push({ message: 'незакрытая скобка', position: null })

  type FnName = 'IF' | 'RND_X' | 'MIN' | 'MAX'
  const variables: string[] = []
  const functions: FnName[] = []
  const seenVar = new Set<string>()
  const seenFn = new Set<string>()

  const tokens = text.match(TOKEN) ?? []
  for (const raw of tokens) {
    // Чистое число — не переменная.
    if (/^[0-9]+([.,][0-9]+)?$/.test(raw)) continue
    if (ALLOWED_FUNCTIONS.has(raw)) {
      if (!seenFn.has(raw)) {
        seenFn.add(raw)
        functions.push(raw as FnName)
      }
      continue
    }
    // Латинская функция-подобная запись, которой нет в списке разрешённых.
    if (/^[A-Za-z_]+$/.test(raw) && raw === raw.toUpperCase() && raw.length <= 4 && /[A-Z]/.test(raw)) {
      // не блокируем — это может быть короткая переменная; функции ловим по вхождению в ALLOWED.
    }
    if (!seenVar.has(raw)) {
      seenVar.add(raw)
      variables.push(raw)
    }
  }

  return { valid: errors.length === 0, variables, functions, errors }
}
