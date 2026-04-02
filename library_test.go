package earslint

import (
	"reflect"
	"testing"
)

func testCatalog() Catalog {
	return Catalog{
		Systems: []CatalogEntry{{ID: "SYS-ENGINE", Name: "engine control system", Aliases: []string{"engine ctrl system"}}},
		States: []CatalogEntry{
			{ID: "STATE-GROUND", Name: "aircraft is on ground"},
			{ID: "STATE-HYD", Name: "hydraulic pressure is available"},
		},
		Events: []CatalogEntry{
			{ID: "EVT-REV", Name: "reverse thrust is commanded"},
			{ID: "EVT-EMB", Name: "emergency braking is active"},
		},
		Features: []CatalogEntry{{ID: "FEAT-PREMIUM", Name: "premium mode is enabled"}},
	}
}

func TestUbiquitousPattern(t *testing.T) {
	res := LintEars("The engine control system shall enable reverse thrust.", testCatalog(), nil)
	if res.Pattern != PatternUbiquitous {
		t.Fatalf("expected ubiquitous, got %q", res.Pattern)
	}
	if res.AST == nil || len(res.AST.Responses) != 1 {
		t.Fatalf("expected one response in AST")
	}
}

func TestStateDrivenPattern(t *testing.T) {
	res := LintEars(
		"While aircraft is on ground, the engine control system shall enable reverse thrust.",
		testCatalog(),
		nil,
	)
	if res.Pattern != PatternStateDriven {
		t.Fatalf("expected state-driven pattern, got %q", res.Pattern)
	}
}

func TestEventDrivenPattern(t *testing.T) {
	res := LintEars(
		"When reverse thrust is commanded, the engine control system shall enable reverse thrust.",
		testCatalog(),
		nil,
	)
	if res.Pattern != PatternEventDriven {
		t.Fatalf("expected event-driven pattern, got %q", res.Pattern)
	}
}

func TestOptionalFeaturePattern(t *testing.T) {
	res := LintEars(
		"Where premium mode is enabled, the engine control system shall enable reverse thrust.",
		testCatalog(),
		nil,
	)
	if res.Pattern != PatternOptionalFeature {
		t.Fatalf("expected optional-feature pattern, got %q", res.Pattern)
	}
}

func TestComplexPatternWithExpressions(t *testing.T) {
	res := LintEars(
		"While aircraft is on ground and hydraulic pressure is available, when reverse thrust is commanded or emergency braking is active, the engine control system shall enable reverse thrust.",
		testCatalog(),
		&Options{Mode: ModeStrict},
	)
	if res.Pattern != PatternComplex {
		t.Fatalf("expected complex pattern, got %q", res.Pattern)
	}
	if res.AST == nil || res.AST.Preconditions == nil || res.AST.Trigger == nil {
		t.Fatalf("expected preconditions and trigger AST")
	}
	if !res.Valid {
		t.Fatalf("expected valid result, got diagnostics: %+v", res.Diagnostics)
	}
}

func TestIfThenValidation(t *testing.T) {
	res := LintEars("If reverse thrust is commanded, the engine control system shall enable reverse thrust.", testCatalog(), nil)
	if res.Valid {
		t.Fatalf("expected invalid result due to missing then")
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "ears.invalid_if_then_form" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected ears.invalid_if_then_form diagnostic")
	}
}

func TestValidIfThenPattern(t *testing.T) {
	res := LintEars("If reverse thrust is commanded, then the engine control system shall enable reverse thrust.", testCatalog(), nil)
	if res.Pattern != PatternUnwantedBehavior {
		t.Fatalf("expected unwanted-behaviour pattern, got %q", res.Pattern)
	}
	if !res.Valid {
		t.Fatalf("expected valid if/then result, got diagnostics: %+v", res.Diagnostics)
	}
}

func TestBatchMode(t *testing.T) {
	results := LintEarsBatch([][2]string{
		{"REQ-1", "The engine control system shall enable reverse thrust."},
		{"REQ-2", "When reverse thrust is commanded, the engine control system shall enable reverse thrust."},
	}, testCatalog(), nil)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != "REQ-1" || results[1].ID != "REQ-2" {
		t.Fatalf("batch IDs not preserved")
	}
}

func TestSystemUnresolvedInStrictMode(t *testing.T) {
	res := LintEars("The unknown system shall do something.", testCatalog(), &Options{Mode: ModeStrict})
	if res.Valid {
		t.Fatalf("expected invalid result for unresolved system")
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "catalog.system_unresolved" && d.Severity == SeverityError {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected catalog.system_unresolved error")
	}
}

func TestMultipleTriggerClausesInvalid(t *testing.T) {
	res := LintEars("When reverse thrust is commanded, when emergency braking is active, the engine control system shall enable reverse thrust.", testCatalog(), nil)
	if res.Valid {
		t.Fatalf("expected invalid result for multiple trigger clauses")
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "ears.invalid_clause_order" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected ears.invalid_clause_order for multiple triggers")
	}
}

func TestGuidedSuspiciousShape(t *testing.T) {
	res := LintEars("maybe enable reverse thrust someday", testCatalog(), &Options{Mode: ModeGuided})
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "lint.suspicious_text_shape" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected lint.suspicious_text_shape in guided mode")
	}
}

func TestCommaAsAndExpression(t *testing.T) {
	res := LintEars(
		"While aircraft is on ground, hydraulic pressure is available, the engine control system shall enable reverse thrust.",
		testCatalog(),
		&Options{Mode: ModeStrict, CommaAsAnd: true},
	)
	if res.AST == nil || res.AST.Preconditions == nil {
		t.Fatalf("expected preconditions expression")
	}
	if res.AST.Preconditions.Kind != "and" {
		t.Fatalf("expected comma-as-and to build 'and' expression, got %q", res.AST.Preconditions.Kind)
	}
}

func TestAliasUseWarning(t *testing.T) {
	res := LintEars("The engine ctrl system shall enable reverse thrust.", testCatalog(), nil)
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "lint.alias_used" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected lint.alias_used warning for alias match")
	}
}

func TestAmbiguousEventDiagnosticCode(t *testing.T) {
	c := testCatalog()
	c.Conditions = []CatalogEntry{
		{ID: "COND-REV", Name: "reverse thrust is commanded"},
	}
	res := LintEars(
		"When reverse thrust is commanded, the engine control system shall enable reverse thrust.",
		c,
		nil,
	)
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "catalog.event_ambiguous" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected catalog.event_ambiguous diagnostic")
	}
}

func TestDiagnosticsDeterministic(t *testing.T) {
	text := "When reverse thrust is commanded and, the unknown system shall do something appropriate."
	opts := &Options{Mode: ModeStrict}
	a := LintEars(text, testCatalog(), opts)
	b := LintEars(text, testCatalog(), opts)
	if !reflect.DeepEqual(a.Diagnostics, b.Diagnostics) {
		t.Fatalf("diagnostics order/content must be deterministic")
	}
}
