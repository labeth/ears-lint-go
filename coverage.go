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
	results := LintEarsBatch(items, catalog, &opts)
	return LintCatalogCoverageFromResults(results, catalog, &opts)
}

// LintCatalogCoverageFromResults lints catalog-term coverage using already
// computed lint results. Coverage diagnostics are emitted only in strict mode.
func LintCatalogCoverageFromResults(results []LintResult, catalog Catalog, options *Options) []Diagnostic {
	opts := withDefaults(options)
	if len(results) == 0 || opts.Mode != ModeStrict {
		return nil
	}

	referenced := map[string]bool{}
	for _, res := range results {
		for _, ref := range res.References {
			if ref.Matched == nil {
				continue
			}
			group := strings.TrimSpace(ref.Matched.Group)
			id := strings.TrimSpace(ref.Matched.ID)
			if group == "" || id == "" {
				continue
			}
			referenced[group+":"+id] = true
		}
	}

	out := make([]Diagnostic, 0)
	add := func(group string, entries []CatalogEntry) {
		for _, e := range entries {
			id := strings.TrimSpace(e.ID)
			name := strings.TrimSpace(e.Name)
			if id == "" || name == "" {
				continue
			}
			if referenced[group+":"+id] {
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
