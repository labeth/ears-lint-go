package earslint

import (
	"sort"
	"strings"
)

type tokenKind int

const (
	tokWord tokenKind = iota
	tokAnd
	tokOr
	tokNot
	tokLParen
	tokRParen
	tokComma
)

type exprToken struct {
	kind       tokenKind
	text       string
	start, end int
}

type exprParser struct {
	tokens      []exprToken
	pos         int
	role        TermRole
	options     Options
	diagnostics []Diagnostic
	hasAnd      bool
	hasOr       bool
	hasGroup    bool
}

func parseClauseExpression(raw string, baseOffset int, role TermRole, options Options) (*ClauseExpr, []Diagnostic) {
	tokens, tokDiags := tokenizeExpression(raw, baseOffset, options.Mode)
	p := &exprParser{tokens: tokens, role: role, options: options}

	expr := p.parseExpr()
	if expr == nil {
		span := &Span{Start: baseOffset, End: baseOffset + len(raw)}
		expr = &ClauseExpr{Kind: "free-text", Text: strings.TrimSpace(raw), Span: span}
		p.diagnostics = append(p.diagnostics, Diagnostic{
			Code:     "expr.empty_subexpression",
			Severity: severityByMode(options.Mode),
			Message:  "clause expression is empty",
			Span:     span,
		})
	}

	if p.pos < len(p.tokens) {
		start := p.tokens[p.pos].start
		end := p.tokens[len(p.tokens)-1].end
		p.diagnostics = append(p.diagnostics, Diagnostic{
			Code:     "lint.unparsed_tail",
			Severity: SeverityWarning,
			Message:  "unparsed tail remains in clause expression",
			Span:     &Span{Start: start, End: end},
		})
	}

	if p.hasAnd && p.hasOr && !p.hasGroup {
		p.diagnostics = append(p.diagnostics, Diagnostic{
			Code:     "expr.operator_precedence_warning",
			Severity: SeverityWarning,
			Message:  "expression mixes 'and' and 'or' without grouping",
			Span:     expr.Span,
		})
	}

	allDiags := append(tokDiags, p.diagnostics...)
	allDiags = sortDiagnostics(allDiags)
	return expr, allDiags
}

func tokenizeExpression(raw string, baseOffset int, mode Mode) ([]exprToken, []Diagnostic) {
	tokens := []exprToken{}
	diags := []Diagnostic{}
	depth := 0

	i := 0
	for i < len(raw) {
		c := raw[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}
		if c == '(' {
			depth++
			tokens = append(tokens, exprToken{kind: tokLParen, text: "(", start: baseOffset + i, end: baseOffset + i + 1})
			i++
			continue
		}
		if c == ')' {
			if depth == 0 {
				diags = append(diags, Diagnostic{
					Code:     "expr.unbalanced_parentheses",
					Severity: severityByMode(mode),
					Message:  "unbalanced closing parenthesis",
					Span:     &Span{Start: baseOffset + i, End: baseOffset + i + 1},
				})
			} else {
				depth--
			}
			tokens = append(tokens, exprToken{kind: tokRParen, text: ")", start: baseOffset + i, end: baseOffset + i + 1})
			i++
			continue
		}
		if c == ',' {
			tokens = append(tokens, exprToken{kind: tokComma, text: ",", start: baseOffset + i, end: baseOffset + i + 1})
			i++
			continue
		}

		start := i
		for i < len(raw) {
			x := raw[i]
			if x == ' ' || x == '\t' || x == '\n' || x == '\r' || x == '(' || x == ')' || x == ',' {
				break
			}
			i++
		}
		chunk := raw[start:i]
		kind := tokWord
		switch strings.ToLower(chunk) {
		case "and":
			kind = tokAnd
		case "or":
			kind = tokOr
		case "not":
			kind = tokNot
		}
		tokens = append(tokens, exprToken{kind: kind, text: chunk, start: baseOffset + start, end: baseOffset + i})
	}

	if depth > 0 {
		diags = append(diags, Diagnostic{
			Code:     "expr.unbalanced_parentheses",
			Severity: severityByMode(mode),
			Message:  "unbalanced opening parenthesis",
			Span:     &Span{Start: baseOffset, End: baseOffset + len(raw)},
		})
	}

	return tokens, sortDiagnostics(diags)
}

func (p *exprParser) parseExpr() *ClauseExpr {
	return p.parseOr()
}

func (p *exprParser) parseOr() *ClauseExpr {
	left := p.parseAnd()
	if left == nil {
		return nil
	}
	items := []ClauseExpr{*left}
	for p.match(tokOr) {
		p.hasOr = true
		right := p.parseAnd()
		if right == nil {
			p.addInvalidOperatorDiag("operator 'or' must be followed by an expression")
			break
		}
		items = append(items, *right)
	}
	if len(items) == 1 {
		return &items[0]
	}
	span := mergeSpan(items)
	return &ClauseExpr{Kind: "or", Items: items, Span: span}
}

func (p *exprParser) parseAnd() *ClauseExpr {
	left := p.parseUnary()
	if left == nil {
		return nil
	}
	items := []ClauseExpr{*left}
	for {
		if p.match(tokAnd) {
			p.hasAnd = true
		} else if p.options.CommaAsAnd && p.match(tokComma) {
			p.hasAnd = true
		} else {
			break
		}
		right := p.parseUnary()
		if right == nil {
			p.addInvalidOperatorDiag("operator 'and' must be followed by an expression")
			break
		}
		items = append(items, *right)
	}
	if len(items) == 1 {
		return &items[0]
	}
	span := mergeSpan(items)
	return &ClauseExpr{Kind: "and", Items: items, Span: span}
}

func (p *exprParser) parseUnary() *ClauseExpr {
	if p.match(tokNot) {
		n := p.parseUnary()
		if n == nil {
			p.addInvalidOperatorDiag("operator 'not' must be followed by an expression")
			return &ClauseExpr{Kind: "free-text", Text: "not"}
		}
		span := n.Span
		return &ClauseExpr{Kind: "not", Item: n, Span: span}
	}
	return p.parsePrimary()
}

func (p *exprParser) parsePrimary() *ClauseExpr {
	if p.match(tokLParen) {
		p.hasGroup = true
		inner := p.parseExpr()
		if !p.match(tokRParen) {
			span := p.currentSpan()
			p.diagnostics = append(p.diagnostics, Diagnostic{
				Code:     "expr.unbalanced_parentheses",
				Severity: severityByMode(p.options.Mode),
				Message:  "missing closing parenthesis",
				Span:     span,
			})
		}
		if inner == nil {
			span := p.currentSpan()
			p.diagnostics = append(p.diagnostics, Diagnostic{
				Code:     "expr.empty_subexpression",
				Severity: severityByMode(p.options.Mode),
				Message:  "grouped expression is empty",
				Span:     span,
			})
			inner = &ClauseExpr{Kind: "free-text", Text: "", Span: span}
		}
		return &ClauseExpr{Kind: "group", Item: inner, Span: inner.Span}
	}

	if p.peek(tokRParen) || p.peek(tokAnd) || p.peek(tokOr) || p.peek(tokComma) {
		p.addInvalidOperatorDiag("invalid operator sequence")
		p.pos++
		return &ClauseExpr{Kind: "free-text", Text: ""}
	}

	return p.parseTerm()
}

func (p *exprParser) parseTerm() *ClauseExpr {
	if p.pos >= len(p.tokens) || p.tokens[p.pos].kind != tokWord {
		return nil
	}
	start := p.tokens[p.pos].start
	end := p.tokens[p.pos].end
	parts := []string{p.tokens[p.pos].text}
	p.pos++
	for p.pos < len(p.tokens) && p.tokens[p.pos].kind == tokWord {
		parts = append(parts, p.tokens[p.pos].text)
		end = p.tokens[p.pos].end
		p.pos++
	}
	text := strings.TrimSpace(strings.Join(parts, " "))
	term := TermMatch{Raw: text, Role: p.role}
	return &ClauseExpr{
		Kind: "term",
		Term: &term,
		Span: &Span{Start: start, End: end},
	}
}

func (p *exprParser) match(kind tokenKind) bool {
	if p.pos >= len(p.tokens) || p.tokens[p.pos].kind != kind {
		return false
	}
	p.pos++
	return true
}

func (p *exprParser) peek(kind tokenKind) bool {
	return p.pos < len(p.tokens) && p.tokens[p.pos].kind == kind
}

func (p *exprParser) addInvalidOperatorDiag(msg string) {
	span := p.currentSpan()
	p.diagnostics = append(p.diagnostics, Diagnostic{
		Code:     "expr.invalid_operator_sequence",
		Severity: severityByMode(p.options.Mode),
		Message:  msg,
		Span:     span,
	})
}

func (p *exprParser) currentSpan() *Span {
	if p.pos < len(p.tokens) {
		return &Span{Start: p.tokens[p.pos].start, End: p.tokens[p.pos].end}
	}
	if len(p.tokens) == 0 {
		return nil
	}
	last := p.tokens[len(p.tokens)-1]
	return &Span{Start: last.start, End: last.end}
}

func mergeSpan(items []ClauseExpr) *Span {
	starts := []int{}
	ends := []int{}
	for _, it := range items {
		if it.Span != nil {
			starts = append(starts, it.Span.Start)
			ends = append(ends, it.Span.End)
		}
	}
	if len(starts) == 0 {
		return nil
	}
	sort.Ints(starts)
	sort.Ints(ends)
	return &Span{Start: starts[0], End: ends[len(ends)-1]}
}
