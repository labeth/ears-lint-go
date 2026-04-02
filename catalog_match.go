package earslint

import (
	"sort"
	"strings"
)

type matchCandidate struct {
	ref      CatalogRef
	role     TermRole
	viaAlias bool
}

func resolveAndCollect(ast *EarsAST, catalog Catalog, options Options) ([]ReferenceMatch, []Diagnostic) {
	refs := []ReferenceMatch{}
	diags := []Diagnostic{}

	resolveTerm(&ast.System, &diags, RoleSystem, catalog, options, nil)
	refs = append(refs, ReferenceMatch{
		Clause:     "system",
		Text:       ast.System.Raw,
		Role:       ast.System.Role,
		Matched:    ast.System.Matched,
		Ambiguous:  ast.System.Ambiguous,
		Unresolved: ast.System.Unresolved,
		ViaAlias:   ast.System.ViaAlias,
	})

	if ast.Preconditions != nil {
		r, d := resolveExpr("preconditions", ast.Preconditions, catalog, options)
		refs = append(refs, r...)
		diags = append(diags, d...)
	}
	if ast.Trigger != nil {
		r, d := resolveExpr("trigger", ast.Trigger, catalog, options)
		refs = append(refs, r...)
		diags = append(diags, d...)
	}
	if ast.Feature != nil {
		r, d := resolveExpr("feature", ast.Feature, catalog, options)
		refs = append(refs, r...)
		diags = append(diags, d...)
	}

	sort.SliceStable(refs, func(i, j int) bool {
		a := refs[i]
		b := refs[j]
		aStart := 1<<30 - 1
		bStart := 1<<30 - 1
		if a.Span != nil {
			aStart = a.Span.Start
		}
		if b.Span != nil {
			bStart = b.Span.Start
		}
		if aStart != bStart {
			return aStart < bStart
		}
		if a.Clause != b.Clause {
			return a.Clause < b.Clause
		}
		return a.Text < b.Text
	})

	return refs, sortDiagnostics(diags)
}

func resolveExpr(clauseName string, expr *ClauseExpr, catalog Catalog, options Options) ([]ReferenceMatch, []Diagnostic) {
	refs := []ReferenceMatch{}
	diags := []Diagnostic{}
	resolvedCount := 0
	unresolvedCount := 0

	var walk func(*ClauseExpr)
	walk = func(node *ClauseExpr) {
		if node == nil {
			return
		}
		if node.Kind == "term" && node.Term != nil {
			resolveTerm(node.Term, &diags, node.Term.Role, catalog, options, node.Span)
			if node.Term.Matched != nil {
				resolvedCount++
			}
			if node.Term.Unresolved {
				unresolvedCount++
			}
			refs = append(refs, ReferenceMatch{
				Clause:     clauseName,
				Text:       node.Term.Raw,
				Role:       node.Term.Role,
				Matched:    node.Term.Matched,
				Ambiguous:  node.Term.Ambiguous,
				Unresolved: node.Term.Unresolved,
				ViaAlias:   node.Term.ViaAlias,
				Span:       node.Span,
			})
		}
		if node.Item != nil {
			walk(node.Item)
		}
		for i := range node.Items {
			walk(&node.Items[i])
		}
	}

	walk(expr)
	if resolvedCount > 0 && unresolvedCount > 0 {
		diags = append(diags, Diagnostic{
			Code:     "expr.mixed_unresolved_terms",
			Severity: SeverityWarning,
			Message:  "expression mixes resolved and unresolved terms",
			Span:     expr.Span,
		})
	}
	return refs, diags
}

func resolveTerm(term *TermMatch, diags *[]Diagnostic, requestedRole TermRole, catalog Catalog, options Options, span *Span) {
	matches := findMatches(term.Raw, requestedRole, catalog)
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].ref.Group != matches[j].ref.Group {
			return matches[i].ref.Group < matches[j].ref.Group
		}
		return matches[i].ref.ID < matches[j].ref.ID
	})

	term.Role = requestedRole
	if len(matches) == 1 {
		m := matches[0]
		term.Role = m.role
		term.Matched = &m.ref
		term.ViaAlias = m.viaAlias
		if m.viaAlias {
			*diags = append(*diags, Diagnostic{
				Code:     "lint.alias_used",
				Severity: SeverityWarning,
				Message:  "alias used instead of canonical term",
				Span:     span,
			})
		}
		return
	}

	if len(matches) > 1 {
		term.Ambiguous = make([]CatalogRef, 0, len(matches))
		for _, m := range matches {
			term.Ambiguous = append(term.Ambiguous, m.ref)
		}
		*diags = append(*diags,
			Diagnostic{Code: "expr.ambiguous_term", Severity: SeverityWarning, Message: "ambiguous catalog term in expression", Span: span},
			Diagnostic{Code: catalogCode(requestedRole, "ambiguous"), Severity: ambiguousSeverity(requestedRole, options.Mode), Message: "ambiguous catalog term match", Span: span},
		)
		return
	}

	term.Unresolved = true
	*diags = append(*diags,
		Diagnostic{Code: "expr.unknown_term", Severity: SeverityWarning, Message: "unknown term in expression", Span: span},
		Diagnostic{Code: catalogCode(requestedRole, "unresolved"), Severity: unresolvedSeverity(requestedRole, options.Mode), Message: "unresolved catalog term", Span: span},
	)
}

func unresolvedSeverity(role TermRole, mode Mode) Severity {
	if role == RoleSystem {
		return severityByMode(mode)
	}
	return SeverityWarning
}

func ambiguousSeverity(role TermRole, mode Mode) Severity {
	if role == RoleSystem {
		return severityByMode(mode)
	}
	return SeverityWarning
}

func catalogCode(role TermRole, suffix string) string {
	r := string(role)
	r = strings.ReplaceAll(r, "-", "_")
	return "catalog." + r + "_" + suffix
}

func findMatches(raw string, requestedRole TermRole, catalog Catalog) []matchCandidate {
	key := normalizeKey(raw)
	if key == "" {
		return nil
	}
	candidates := allowedGroups(requestedRole, catalog)
	matches := []matchCandidate{}
	for _, g := range candidates {
		for _, e := range g.entries {
			if normalizeKey(e.Name) == key {
				matches = append(matches, matchCandidate{ref: CatalogRef{Group: g.group, ID: e.ID, Name: e.Name}, role: g.role, viaAlias: false})
				continue
			}
			for _, a := range e.Aliases {
				if normalizeKey(a) == key {
					matches = append(matches, matchCandidate{ref: CatalogRef{Group: g.group, ID: e.ID, Name: e.Name}, role: g.role, viaAlias: true})
					break
				}
			}
		}
	}
	return dedupeMatches(matches)
}

type groupEntries struct {
	group   string
	role    TermRole
	entries []CatalogEntry
}

func allowedGroups(role TermRole, catalog Catalog) []groupEntries {
	all := []groupEntries{
		{group: "systems", role: RoleSystem, entries: catalog.Systems},
		{group: "actors", role: RoleActor, entries: catalog.Actors},
		{group: "events", role: RoleEvent, entries: catalog.Events},
		{group: "states", role: RoleState, entries: catalog.States},
		{group: "features", role: RoleFeature, entries: catalog.Features},
		{group: "modes", role: RoleMode, entries: catalog.Modes},
		{group: "conditions", role: RoleCondition, entries: catalog.Conditions},
		{group: "dataTerms", role: RoleDataTerm, entries: catalog.DataTerms},
	}

	if role == RoleSystem {
		return []groupEntries{all[0]}
	}
	if role == RoleFeature {
		return []groupEntries{all[4], all[5], all[6], all[7]}
	}
	if role == RoleEvent {
		return []groupEntries{all[2], all[6], all[3], all[5], all[1], all[4], all[7]}
	}
	if role == RoleState {
		return []groupEntries{all[3], all[6], all[5], all[7], all[1], all[2], all[4]}
	}
	return all
}

func dedupeMatches(in []matchCandidate) []matchCandidate {
	seen := map[string]bool{}
	out := make([]matchCandidate, 0, len(in))
	for _, c := range in {
		key := c.ref.Group + ":" + c.ref.ID
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
	}
	return out
}
