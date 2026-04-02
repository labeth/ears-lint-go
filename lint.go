package earslint

func lintOne(id, text string, catalog Catalog, options Options) LintResult {
	result := LintResult{
		ID:         id,
		Valid:      false,
		References: []ReferenceMatch{},
	}

	parsed := parseShell(text, options)
	result.Diagnostics = append(result.Diagnostics, parsed.Diagnostics...)
	if parsed.AST == nil {
		result.Diagnostics = sortDiagnostics(result.Diagnostics)
		result.Valid = isValid(result.Diagnostics)
		return result
	}

	result.AST = parsed.AST
	result.Pattern = parsed.AST.Pattern

	refs, resolverDiags := resolveAndCollect(parsed.AST, catalog, options)
	result.References = refs
	result.Diagnostics = append(result.Diagnostics, resolverDiags...)
	result.Diagnostics = append(result.Diagnostics, lintAST(parsed.AST, options)...)
	result.Diagnostics = sortDiagnostics(result.Diagnostics)
	result.Valid = isValid(result.Diagnostics)
	return result
}

func isValid(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == SeverityError {
			return false
		}
	}
	return true
}
