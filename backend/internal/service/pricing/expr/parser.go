package expr

import "fmt"

// Узлы AST выражения формулы.
type (
	node interface{ isNode() }

	// numNode хранит исходный текст: имя переменной может выглядеть как число
	// («2», «40000») — тогда, как в эталоне, значение переменной важнее литерала.
	numNode struct {
		value float64
		text  string
	}

	varNode struct {
		name string
		pos  int
	}

	unaryNode struct {
		op      string
		operand node
	}

	binNode struct {
		op    string
		left  node
		right node
	}

	// cmpNode — цепочка сравнений python-семантики: a < b < c → (a<b) and (b<c).
	cmpNode struct {
		first node
		ops   []string
		rest  []node
	}

	callNode struct {
		name string
		pos  int
		args []node
	}
)

func (numNode) isNode()   {}
func (varNode) isNode()   {}
func (unaryNode) isNode() {}
func (binNode) isNode()   {}
func (cmpNode) isNode()   {}
func (callNode) isNode()  {}

type parser struct {
	tokens []token
	index  int

	variables []string
	functions []string
	seenVars  map[string]bool
	seenFuncs map[string]bool
}

func parseExpression(expression string) (node, *parser, error) {
	tokens, err := tokenize(expression)
	if err != nil {
		return nil, nil, err
	}

	p := &parser{
		tokens:    tokens,
		seenVars:  map[string]bool{},
		seenFuncs: map[string]bool{},
	}

	root, err := p.parseComparison()
	if err != nil {
		return nil, nil, err
	}

	if p.current().kind != tokEOF {
		return nil, nil, &syntaxError{
			message:  fmt.Sprintf("неожиданный токен: %s", p.current().text),
			position: p.current().pos,
		}
	}

	return root, p, nil
}

func (p *parser) current() token {
	return p.tokens[p.index]
}

func (p *parser) advance() {
	p.index++
}

func (p *parser) matchOp(ops ...string) (string, bool) {
	tok := p.current()
	if tok.kind != tokOp {
		return "", false
	}
	for _, op := range ops {
		if tok.text == op {
			p.index++

			return op, true
		}
	}

	return "", false
}

func (p *parser) parseComparison() (node, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}

	var ops []string
	var rest []node
	for {
		op, ok := p.matchOp("<", "<=", ">", ">=", "=", "==", "<>", "!=")
		if !ok {
			break
		}
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
		rest = append(rest, right)
	}

	if len(ops) == 0 {
		return left, nil
	}

	return cmpNode{first: left, ops: ops, rest: rest}, nil
}

func (p *parser) parseAdditive() (node, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}

	for {
		op, ok := p.matchOp("+", "-")
		if !ok {
			return left, nil
		}
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = binNode{op: op, left: left, right: right}
	}
}

func (p *parser) parseMultiplicative() (node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		op, ok := p.matchOp("*", "/", "%")
		if !ok {
			return left, nil
		}
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = binNode{op: op, left: left, right: right}
	}
}

// parseUnary — как в python: -A**2 = -(A**2).
func (p *parser) parseUnary() (node, error) {
	if op, ok := p.matchOp("+", "-"); ok {
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}

		return unaryNode{op: op, operand: operand}, nil
	}

	return p.parsePower()
}

func (p *parser) parsePower() (node, error) {
	base, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	if _, ok := p.matchOp("**"); ok {
		exponent, err := p.parseUnary()
		if err != nil {
			return nil, err
		}

		return binNode{op: "**", left: base, right: exponent}, nil
	}

	return base, nil
}

func (p *parser) parsePrimary() (node, error) {
	tok := p.current()

	//nolint:exhaustive // остальные виды токенов в этой позиции — ошибка разбора (default)
	switch tok.kind {
	case tokNumber:
		p.advance()

		return numNode{value: tok.num, text: tok.text}, nil

	case tokIdent:
		p.advance()
		if p.current().kind == tokLParen {
			return p.parseCall(tok)
		}
		if !p.seenVars[tok.text] {
			p.seenVars[tok.text] = true
			p.variables = append(p.variables, tok.text)
		}

		return varNode{name: tok.text, pos: tok.pos}, nil

	case tokLParen:
		p.advance()
		inner, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		if p.current().kind != tokRParen {
			return nil, &syntaxError{message: "ожидалась закрывающая скобка", position: p.current().pos}
		}
		p.advance()

		return inner, nil

	default:
		return nil, &syntaxError{
			message:  fmt.Sprintf("неожиданный токен: %s", tokenText(tok)),
			position: tok.pos,
		}
	}
}

func (p *parser) parseCall(name token) (node, error) {
	p.advance() // (

	var args []node
	if p.current().kind != tokRParen {
		for {
			arg, err := p.parseComparison()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			if p.current().kind == tokComma {
				p.advance()

				continue
			}

			break
		}
	}

	if p.current().kind != tokRParen {
		return nil, &syntaxError{message: "ожидалась закрывающая скобка вызова", position: p.current().pos}
	}
	p.advance()

	if !p.seenFuncs[name.text] {
		p.seenFuncs[name.text] = true
		p.functions = append(p.functions, name.text)
	}

	return callNode{name: name.text, pos: name.pos, args: args}, nil
}

func tokenText(tok token) string {
	if tok.kind == tokEOF {
		return "конец выражения"
	}

	return tok.text
}
