package earslint

import (
	"regexp"
	"strings"
)

var shallWordRe = regexp.MustCompile(`(?i)\bshall\b`)

type shellParseResult struct {
	AST         *EarsAST
	Diagnostics []Diagnostic
}

type clauseSlot struct {
	kind ClauseType
	text string
	span Span
	expr *ClauseExpr
}

type shellState struct {
	text        string
	pos         int
	options     Options
	diagnostics []Diagnostic
	clauses     []clauseSlot
}

func parseShell(text string, options Options) shellParseResult {
	normalized := normalizeForParsing(text)
	state := shellState{text: normalized, options: options}

	if normalized == "" {
		state.addDiag("ears.no_match", severityByMode(options.Mode), "requirement text is empty", nil)
		return shellParseResult{Diagnostics: sortDiagnostics(state.diagnostics)}
	}
	state.diagnostics = append(state.diagnostics, balanceDiagnostics(normalized, options.Mode)...)

	shallCount := countShall(normalized)
	if shallCount == 0 {
		state.addDiag("ears.missing_shall", severityByMode(options.Mode), "requirement must contain exactly one 'shall'", nil)
	}
	if shallCount > 1 {
		state.addDiag("ears.multiple_shall", severityByMode(options.Mode), "requirement contains more than one 'shall'", nil)
	}

	ast, ok := state.parse()
	if !ok {
		if options.Mode == ModeGuided {
			state.addDiag("lint.suspicious_text_shape", SeverityWarning, "text shape does not appear to follow EARS shell form", nil)
		}
		state.addDiag("ears.no_match", severityByMode(options.Mode), "text does not match a supported EARS pattern", nil)
		return shellParseResult{Diagnostics: sortDiagnostics(state.diagnostics)}
	}

	return shellParseResult{AST: ast, Diagnostics: sortDiagnostics(state.diagnostics)}
}

func balanceDiagnostics(text string, mode Mode) []Diagnostic {
	diags := []Diagnostic{}
	depth := 0
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				diags = append(diags, Diagnostic{
					Code:     "expr.unbalanced_parentheses",
					Severity: severityByMode(mode),
					Message:  "unbalanced closing parenthesis",
					Span:     &Span{Start: i, End: i + 1},
				})
			} else {
				depth--
			}
		}
	}
	if depth > 0 {
		diags = append(diags, Diagnostic{
			Code:     "expr.unbalanced_parentheses",
			Severity: severityByMode(mode),
			Message:  "unbalanced opening parenthesis",
			Span:     &Span{Start: 0, End: len(text)},
		})
	}
	return diags
}

func countShall(text string) int {
	return len(shallWordRe.FindAllStringIndex(text, -1))
}

func (s *shellState) parse() (*EarsAST, bool) {
	s.pos = skipWS(s.text, 0)
	if s.consumeKeyword("if") {
		ifClause, ok := s.parseIfClause()
		if !ok {
			return nil, false
		}
		s.clauses = append(s.clauses, ifClause)
		s.pos = skipWS(s.text, s.pos)
	} else {
		for {
			s.pos = skipWS(s.text, s.pos)
			switch {
			case s.peekKeyword("while"):
				s.consumeKeyword("while")
				cl := s.parseCommaClause(ClauseWhile)
				s.clauses = append(s.clauses, cl)
			case s.peekKeyword("when"):
				s.consumeKeyword("when")
				cl := s.parseCommaClause(ClauseWhen)
				s.clauses = append(s.clauses, cl)
			case s.peekKeyword("where"):
				s.consumeKeyword("where")
				cl := s.parseCommaClause(ClauseWhere)
				s.clauses = append(s.clauses, cl)
			case s.peekKeyword("if"):
				s.consumeKeyword("if")
				ifClause, ok := s.parseIfClause()
				if !ok {
					return nil, false
				}
				s.clauses = append(s.clauses, ifClause)
			default:
				goto parseThe
			}
		}
	}

parseThe:
	if !s.consumeKeyword("the") {
		s.addDiag("ears.missing_system", severityByMode(s.options.Mode), "missing 'the <system> shall ...' clause", nil)
		return nil, false
	}

	systemStart := s.pos
	shallIdx := findWordFrom(s.text, s.pos, "shall")
	if shallIdx < 0 {
		s.addDiag("ears.missing_shall", severityByMode(s.options.Mode), "missing 'shall' in system clause", nil)
		return nil, false
	}

	systemRaw := strings.TrimSpace(s.text[systemStart:shallIdx])
	if systemRaw == "" {
		span := Span{Start: systemStart, End: shallIdx}
		s.addDiag("ears.missing_system", severityByMode(s.options.Mode), "system name is empty", &span)
	}

	s.pos = shallIdx + len("shall")
	responseRaw := strings.TrimSpace(s.text[s.pos:])
	if responseRaw == "" {
		s.addDiag("ears.no_match", severityByMode(s.options.Mode), "missing system response after 'shall'", nil)
		return nil, false
	}

	responses := parseResponses(responseRaw)
	if len(responses) == 0 {
		s.addDiag("ears.no_match", severityByMode(s.options.Mode), "missing system response after 'shall'", nil)
		return nil, false
	}

	if !validClauseOrder(s.clauses) {
		s.addDiag("ears.invalid_clause_order", severityByMode(s.options.Mode), "invalid EARS clause order", nil)
	}
	s.validateClauseCardinality()

	ast := &EarsAST{
		Pattern: classifyPattern(s.clauses),
		System: TermMatch{
			Raw:  systemRaw,
			Role: RoleSystem,
		},
		Responses: responses,
		Raw:       s.text,
	}

	for i := range s.clauses {
		cl := &s.clauses[i]
		role := roleForClause(cl.kind)
		expr, exprDiags := parseClauseExpression(cl.text, cl.span.Start, role, s.options)
		cl.expr = expr
		s.diagnostics = append(s.diagnostics, exprDiags...)
		s.assignClause(ast, cl)
	}

	return ast, true
}

func (s *shellState) parseCommaClause(kind ClauseType) clauseSlot {
	start := s.pos
	end, next := scanUntilClauseBoundary(s.text, s.pos)
	body := strings.TrimSpace(s.text[start:end])
	span := Span{Start: start, End: end}
	s.pos = next
	return clauseSlot{kind: kind, text: body, span: span}
}

func (s *shellState) parseIfClause() (clauseSlot, bool) {
	start := s.pos
	thenIdx, thenWordLen := findThenBoundary(s.text, s.pos)
	if thenIdx < 0 {
		span := Span{Start: start, End: len(s.text)}
		s.addDiag("ears.invalid_if_then_form", severityByMode(s.options.Mode), "'If' clause must be paired with 'then'", &span)
		end, next := scanUntilClauseBoundary(s.text, s.pos)
		body := strings.TrimSpace(s.text[start:end])
		s.pos = next
		return clauseSlot{kind: ClauseIf, text: body, span: Span{Start: start, End: end}}, true
	}

	body := strings.TrimSpace(s.text[start:thenIdx])
	body = strings.TrimSuffix(body, ",")
	span := Span{Start: start, End: thenIdx}
	s.pos = skipWS(s.text, thenIdx+thenWordLen)
	if !s.peekKeyword("the") {
		s.addDiag("ears.invalid_if_then_form", severityByMode(s.options.Mode), "'then' must be followed by 'the <system> shall ...'", &span)
		return clauseSlot{}, false
	}
	// parse() will consume "the" in the common path.
	return clauseSlot{kind: ClauseIf, text: body, span: span}, true
}

func (s *shellState) assignClause(ast *EarsAST, cl *clauseSlot) {
	switch cl.kind {
	case ClauseWhile:
		if ast.Preconditions == nil {
			ast.Preconditions = cl.expr
		} else {
			ast.Preconditions = &ClauseExpr{
				Kind:  "and",
				Items: []ClauseExpr{*ast.Preconditions, *cl.expr},
			}
		}
	case ClauseWhen:
		if ast.Trigger == nil {
			ast.Trigger = cl.expr
		} else {
			ast.Trigger = &ClauseExpr{
				Kind:  "and",
				Items: []ClauseExpr{*ast.Trigger, *cl.expr},
			}
		}
	case ClauseWhere:
		if ast.Feature == nil {
			ast.Feature = cl.expr
		} else {
			ast.Feature = &ClauseExpr{
				Kind:  "and",
				Items: []ClauseExpr{*ast.Feature, *cl.expr},
			}
		}
	case ClauseIf:
		ast.Trigger = cl.expr
	}
}

func classifyPattern(clauses []clauseSlot) Pattern {
	if len(clauses) == 0 {
		return PatternUbiquitous
	}
	counts := map[ClauseType]int{}
	for _, c := range clauses {
		counts[c.kind]++
	}
	if len(clauses) == 1 {
		switch clauses[0].kind {
		case ClauseWhile:
			return PatternStateDriven
		case ClauseWhen:
			return PatternEventDriven
		case ClauseWhere:
			return PatternOptionalFeature
		case ClauseIf:
			return PatternUnwantedBehavior
		}
	}
	if counts[ClauseIf] > 0 && len(clauses) > 1 {
		return PatternComplex
	}
	return PatternComplex
}

func validClauseOrder(clauses []clauseSlot) bool {
	// Accept the clause ordering used in the supplied EARS corpus:
	// While* -> Where* -> When* -> If*
	// This keeps shell parsing deterministic while allowing complex combinations
	// such as "While ..., where ..., when ..., the ... shall ...".
	order := map[ClauseType]int{
		ClauseWhile: 1,
		ClauseWhere: 2,
		ClauseWhen:  3,
		ClauseIf:    4,
	}
	max := 0
	for _, cl := range clauses {
		v := order[cl.kind]
		if v < max {
			return false
		}
		max = v
	}
	return true
}

func (s *shellState) validateClauseCardinality() {
	counts := map[ClauseType]int{}
	for _, cl := range s.clauses {
		counts[cl.kind]++
	}

	// EARS shell allows zero or one trigger slot at top level.
	triggerCount := counts[ClauseWhen] + counts[ClauseIf]
	if triggerCount > 1 {
		s.addDiag("ears.invalid_clause_order", severityByMode(s.options.Mode), "multiple trigger clauses found; expected zero or one", nil)
	}
	if counts[ClauseWhere] > 1 {
		s.addDiag("ears.invalid_clause_order", severityByMode(s.options.Mode), "multiple feature clauses found; expected zero or one", nil)
	}
	if counts[ClauseIf] > 1 {
		s.addDiag("ears.invalid_if_then_form", severityByMode(s.options.Mode), "multiple 'If ... then' clauses are not supported", nil)
	}
}

func parseResponses(raw string) []string {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "."))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{raw}
	}
	return out
}

func findThenBoundary(text string, start int) (int, int) {
	depth := 0
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		}
		if depth == 0 && hasWordAt(text, i, "then") {
			return i, len("then")
		}
	}
	return -1, 0
}

func scanUntilClauseBoundary(text string, start int) (end int, next int) {
	depth := 0
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		}
		if depth == 0 && text[i] == ',' {
			j := skipWS(text, i+1)
			if hasAnyKeywordAt(text, j, []string{"while", "when", "where", "if", "the"}) {
				return i, j
			}
		}
	}
	return len(text), len(text)
}

func (s *shellState) addDiag(code string, severity Severity, msg string, span *Span) {
	s.diagnostics = append(s.diagnostics, Diagnostic{Code: code, Severity: severity, Message: msg, Span: span})
}

func (s *shellState) peekKeyword(keyword string) bool {
	return hasWordAt(s.text, s.pos, keyword)
}

func (s *shellState) consumeKeyword(keyword string) bool {
	if !s.peekKeyword(keyword) {
		return false
	}
	s.pos += len(keyword)
	s.pos = skipWS(s.text, s.pos)
	return true
}

func skipWS(text string, pos int) int {
	for pos < len(text) {
		if text[pos] != ' ' && text[pos] != '\t' && text[pos] != '\n' && text[pos] != '\r' {
			return pos
		}
		pos++
	}
	return pos
}

func hasAnyKeywordAt(text string, pos int, kws []string) bool {
	for _, kw := range kws {
		if hasWordAt(text, pos, kw) {
			return true
		}
	}
	return false
}

func hasWordAt(text string, pos int, word string) bool {
	if pos < 0 || pos+len(word) > len(text) {
		return false
	}
	if !strings.EqualFold(text[pos:pos+len(word)], word) {
		return false
	}
	beforeOK := pos == 0 || !isWordByte(text[pos-1])
	afterPos := pos + len(word)
	afterOK := afterPos >= len(text) || !isWordByte(text[afterPos])
	return beforeOK && afterOK
}

func isWordByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func findWordFrom(text string, pos int, word string) int {
	for i := pos; i < len(text); i++ {
		if hasWordAt(text, i, word) {
			return i
		}
	}
	return -1
}

func roleForClause(kind ClauseType) TermRole {
	switch kind {
	case ClauseWhen, ClauseIf:
		return RoleEvent
	case ClauseWhere:
		return RoleFeature
	default:
		return RoleState
	}
}
