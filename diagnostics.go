package earslint

import "sort"

func sortDiagnostics(diags []Diagnostic) []Diagnostic {
	out := append([]Diagnostic(nil), diags...)
	sort.SliceStable(out, func(i, j int) bool {
		a := out[i]
		b := out[j]
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
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Message != b.Message {
			return a.Message < b.Message
		}
		return string(a.Severity) < string(b.Severity)
	})
	return out
}

func severityByMode(mode Mode) Severity {
	if mode == ModeGuided {
		return SeverityWarning
	}
	return SeverityError
}
