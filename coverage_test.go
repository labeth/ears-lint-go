package earslint

import "testing"

func TestLintCatalogCoverage_WarnsOnUnreferencedTermsInStrict(t *testing.T) {
	catalog := Catalog{
		Systems: []CatalogEntry{
			{ID: "SYS-A", Name: "aircraft system"},
		},
		Events: []CatalogEntry{
			{ID: "EVT-REV", Name: "reverse thrust is commanded"},
		},
		Actors: []CatalogEntry{
			{ID: "ACT-PILOT", Name: "pilot"},
		},
	}
	items := [][2]string{
		{"REQ-1", "When reverse thrust is commanded, the aircraft system shall enable reverse thrust."},
	}

	diags := LintCatalogCoverage(items, catalog, &Options{Mode: ModeStrict})
	if len(diags) == 0 {
		t.Fatalf("expected coverage diagnostics")
	}

	found := false
	for _, d := range diags {
		if d.Code == "catalog.term_unreferenced" && d.Severity == SeverityWarning {
			if d.Message == `catalog actors term "pilot" (ACT-PILOT) is not referenced by any requirement text` {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("expected unreferenced actor warning, got: %+v", diags)
	}
}

func TestLintCatalogCoverage_NoWarningsInGuidedMode(t *testing.T) {
	catalog := Catalog{
		Systems: []CatalogEntry{
			{ID: "SYS-A", Name: "aircraft system"},
		},
		Actors: []CatalogEntry{
			{ID: "ACT-PILOT", Name: "pilot"},
		},
	}
	items := [][2]string{
		{"REQ-1", "The aircraft system shall enable reverse thrust."},
	}

	diags := LintCatalogCoverage(items, catalog, &Options{Mode: ModeGuided})
	if len(diags) != 0 {
		t.Fatalf("expected no coverage diagnostics in guided mode, got: %+v", diags)
	}
}
