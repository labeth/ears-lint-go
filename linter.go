package earslint

import "strings"

func lintAST(ast *EarsAST, options Options) []Diagnostic {
	diags := []Diagnostic{}

	if len(ast.Responses) == 0 {
		diags = append(diags, Diagnostic{
			Code:     "ears.no_match",
			Severity: severityByMode(options.Mode),
			Message:  "at least one response is required",
		})
		return diags
	}

	if len(ast.Responses) == 1 {
		r := strings.ToLower(ast.Responses[0])
		if strings.Contains(r, " and ") || strings.Contains(r, ",") {
			diags = append(diags, Diagnostic{
				Code:     "lint.multiple_responses",
				Severity: SeverityWarning,
				Message:  "response may contain multiple responses joined together",
			})
		}
	}
	if len(ast.Responses) > 1 {
		diags = append(diags, Diagnostic{
			Code:     "lint.multiple_responses",
			Severity: SeverityWarning,
			Message:  "multiple responses detected",
		})
	}

	for _, resp := range ast.Responses {
		lower := strings.ToLower(resp)
		for _, w := range options.VagueTerms {
			needle := strings.ToLower(strings.TrimSpace(w))
			if needle != "" && strings.Contains(lower, needle) {
				diags = append(diags, Diagnostic{
					Code:     "lint.vague_response",
					Severity: SeverityWarning,
					Message:  "response contains vague wording: " + needle,
				})
			}
		}
	}

	if ast.System.Raw == "" {
		diags = append(diags, Diagnostic{
			Code:     "ears.missing_system",
			Severity: severityByMode(options.Mode),
			Message:  "system term is missing",
		})
	}

	return diags
}
