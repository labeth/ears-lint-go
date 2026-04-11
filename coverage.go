package earslint

import (
	"fmt"
	"strings"
)

// LintCatalogCoverage lints catalog-term coverage across a batch of requirements.
// It emits warning diagnostics for catalog entries that are never referenced by
// any requirement text. Coverage diagnostics are emitted only in strict mode.
func LintCatalogCoverage(items [][2]string, catalog Catalog, options *Options) []Diagnostic {
	opts := withDefaults(options)
	if len(items) == 0 || opts.Mode != ModeStrict {
		return nil
	}
	texts := make([]string, 0, len(items))
	for _, item := range items {
		texts = append(texts, item[1])
	}
	return lintCatalogCoverageTexts(texts, catalog)
}

// LintCatalogCoverageFromResults lints catalog-term coverage using already
// computed lint results. Coverage diagnostics are emitted only in strict mode.
func LintCatalogCoverageFromResults(results []LintResult, catalog Catalog, options *Options) []Diagnostic {
	opts := withDefaults(options)
	if len(results) == 0 || opts.Mode != ModeStrict {
		return nil
	}

	texts := make([]string, 0, len(results))
	for _, res := range results {
		if res.AST == nil {
			continue
		}
		texts = append(texts, res.AST.Raw)
	}
	return lintCatalogCoverageTexts(texts, catalog)
}

func lintCatalogCoverageTexts(texts []string, catalog Catalog) []Diagnostic {
	if len(texts) == 0 {
		return nil
	}
	lowered := make([]string, 0, len(texts))
	for _, t := range texts {
		lowered = append(lowered, strings.ToLower(strings.TrimSpace(t)))
	}
	out := make([]Diagnostic, 0)
	add := func(group string, entries []CatalogEntry) {
		for _, e := range entries {
			id := strings.TrimSpace(e.ID)
			name := strings.TrimSpace(e.Name)
			if id == "" || name == "" {
				continue
			}
			if catalogEntryCovered(lowered, e) {
				continue
			}
			out = append(out, Diagnostic{
				Code:     "catalog.term_unreferenced",
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("catalog %s term %q (%s) is not referenced by any requirement text", group, name, id),
			})
		}
	}

	add("systems", catalog.Systems)
	add("actors", catalog.Actors)
	add("events", catalog.Events)
	add("states", catalog.States)
	add("features", catalog.Features)
	add("modes", catalog.Modes)
	add("conditions", catalog.Conditions)
	add("dataTerms", catalog.DataTerms)

	return sortDiagnostics(out)
}

func catalogEntryCovered(texts []string, entry CatalogEntry) bool {
	candidates := []string{strings.TrimSpace(entry.Name)}
	candidates = append(candidates, entry.Aliases...)
	for _, c := range candidates {
		term := strings.ToLower(strings.TrimSpace(c))
		if term == "" {
			continue
		}
		for _, text := range texts {
			if containsPhrase(text, term) {
				return true
			}
		}
	}
	return false
}

func containsPhrase(text, phrase string) bool {
	if text == "" || phrase == "" {
		return false
	}
	from := 0
	for {
		i := strings.Index(text[from:], phrase)
		if i < 0 {
			return false
		}
		start := from + i
		end := start + len(phrase)
		if phraseBoundary(text, start-1) && phraseBoundary(text, end) {
			return true
		}
		from = end
		if from >= len(text) {
			return false
		}
	}
}

func phraseBoundary(s string, idx int) bool {
	if idx < 0 || idx >= len(s) {
		return true
	}
	b := s[idx]
	return !((b >= 'a' && b <= 'z') || (b >= '0' && b <= '9'))
}
