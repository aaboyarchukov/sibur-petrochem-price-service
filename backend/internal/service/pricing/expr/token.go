package expr

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type tokenKind int

const (
	tokNumber tokenKind = iota
	tokIdent
	tokOp
	tokLParen
	tokRParen
	tokComma
	tokEOF
)

type token struct {
	kind tokenKind
	text string
	num  float64
	pos  int
}

// syntaxError — ошибка разбора с позицией в строке выражения.
type syntaxError struct {
	message  string
	position int
}

func (e *syntaxError) Error() string {
	return e.message
}

// Символы имён переменных — как в эталоне: A-Za-z0-9_$€¥₱.
func isTokenChar(r rune) bool {
	if r == '_' || r == '$' || r == '€' || r == '¥' || r == '₱' {
		return true
	}

	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || unicode.IsDigit(r)
}

func isNumberLiteral(text string) bool {
	dots := 0
	for _, r := range text {
		if r == '.' {
			dots++
			continue
		}
		if !unicode.IsDigit(r) {
			return false
		}
	}

	return dots <= 1 && text != "." && text != ""
}

func tokenize(expression string) ([]token, error) {
	runes := []rune(expression)
	tokens := make([]token, 0, len(runes)/2)

	i := 0
	for i < len(runes) {
		r := runes[i]

		if unicode.IsSpace(r) {
			i++
			continue
		}

		switch r {
		case '(':
			tokens = append(tokens, token{kind: tokLParen, text: "(", pos: i})
			i++
			continue
		case ')':
			tokens = append(tokens, token{kind: tokRParen, text: ")", pos: i})
			i++
			continue
		case ',':
			tokens = append(tokens, token{kind: tokComma, text: ",", pos: i})
			i++
			continue
		}

		if op, width := scanOperator(runes, i); width > 0 {
			tokens = append(tokens, token{kind: tokOp, text: op, pos: i})
			i += width
			continue
		}

		if isTokenChar(r) || r == '.' {
			tok, width, err := scanWordToken(runes, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
			i += width

			continue
		}

		return nil, &syntaxError{message: fmt.Sprintf("неожиданный символ: %q", string(r)), position: i}
	}

	tokens = append(tokens, token{kind: tokEOF, text: "", pos: len(runes)})

	return tokens, nil
}

func scanWordToken(runes []rune, start int) (token, int, error) {
	text, width := scanWord(runes, start)
	if !isNumberLiteral(text) {
		return token{kind: tokIdent, text: text, pos: start}, width, nil
	}

	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return token{}, 0, &syntaxError{message: fmt.Sprintf("некорректное число: %s", text), position: start}
	}

	return token{kind: tokNumber, text: text, num: parsed, pos: start}, width, nil
}

func scanOperator(runes []rune, start int) (op string, width int) {
	const opWidth = 2
	rest := string(runes[start:min(start+opWidth, len(runes))])
	twoChar := []string{"**", "<=", ">=", "<>", "==", "!="}
	for _, candidate := range twoChar {
		if strings.HasPrefix(rest, candidate) {
			return candidate, opWidth
		}
	}

	switch runes[start] {
	case '+', '-', '*', '/', '%', '<', '>', '=':
		return string(runes[start]), 1
	}

	return "", 0
}

// scanWord — максимальная последовательность символов имени; точка входит
// только внутри числового литерала (1.05), но не в имя (1_13 — переменная).
func scanWord(runes []rune, start int) (text string, width int) {
	i := start
	sawNonDigit := false
	for i < len(runes) {
		r := runes[i]
		if isTokenChar(r) {
			if !unicode.IsDigit(r) {
				sawNonDigit = true
			}
			i++

			continue
		}
		if r == '.' && !sawNonDigit {
			i++

			continue
		}

		break
	}

	return string(runes[start:i]), i - start
}
