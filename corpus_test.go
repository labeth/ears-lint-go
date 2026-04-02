package earslint

import "testing"

func fullCatalog() Catalog {
	return Catalog{
		Systems: []CatalogEntry{
			{ID: "SYS-ENG-CTRL", Name: "engine control system", Aliases: []string{"FADEC", "ECS"}},
			{ID: "SYS-ENV-CTRL", Name: "environmental control system", Aliases: []string{"ECS"}},
			{ID: "SYS-BRAKE-CTRL", Name: "brake control system", Aliases: []string{"brake controller", "BCS"}},
			{ID: "SYS-DIAG", Name: "diagnostic system", Aliases: []string{"diagnostics"}},
			{ID: "SYS-POWER-MGMT", Name: "power management system", Aliases: []string{"PMS"}},
			{ID: "SYS-NAV", Name: "navigation system", Aliases: []string{"nav system"}},
			{ID: "SYS-DOOR-CTRL", Name: "door control system", Aliases: []string{"door controller"}},
			{ID: "SYS-ATM", Name: "ATM", Aliases: []string{"cash machine"}},
		},
		Actors: []CatalogEntry{
			{ID: "ACT-PILOT", Name: "pilot", Aliases: []string{"flight crew member"}},
			{ID: "ACT-FLIGHT-CREW", Name: "flight crew", Aliases: []string{"crew"}},
			{ID: "ACT-DRIVER", Name: "driver"},
			{ID: "ACT-MAINT", Name: "maintenance technician", Aliases: []string{"technician", "maintainer"}},
			{ID: "ACT-USER", Name: "user", Aliases: []string{"operator"}},
		},
		Events: []CatalogEntry{
			{ID: "EVT-REV-CMD", Name: "reverse thrust is commanded", Aliases: []string{"reverse thrust command", "reverse thrust commanded"}},
			{ID: "EVT-REV-CMD-PILOT", Name: "reverse thrust is commanded by pilot", Aliases: []string{"reverse thrust command"}},
			{ID: "EVT-BRAKE-REQ", Name: "brake is requested", Aliases: []string{"brake request", "brake requested"}},
			{ID: "EVT-EMERG-BRAKE", Name: "emergency braking is active", Aliases: []string{"emergency braking"}},
			{ID: "EVT-SENSOR-FAULT", Name: "sensor fault is detected", Aliases: []string{"sensor fault"}},
			{ID: "EVT-LINK-LOSS", Name: "data link is lost", Aliases: []string{"link loss"}},
			{ID: "EVT-DOOR-OPEN-CMD", Name: "door open is commanded", Aliases: []string{"door open command"}},
			{ID: "EVT-DOOR-CLOSE-CMD", Name: "door close is commanded", Aliases: []string{"door close command"}},
			{ID: "EVT-INVALID-CARD", Name: "an invalid credit card number is entered", Aliases: []string{"invalid credit card number"}},
			{ID: "EVT-LOW-VOLTAGE", Name: "bus voltage falls below threshold", Aliases: []string{"low voltage event"}},
		},
		States: []CatalogEntry{
			{ID: "STATE-ONGROUND", Name: "aircraft is on ground", Aliases: []string{"on ground", "aircraft on ground"}},
			{ID: "STATE-INFLIGHT", Name: "aircraft is in flight", Aliases: []string{"in flight"}},
			{ID: "STATE-HYD-PRESS", Name: "hydraulic pressure is available", Aliases: []string{"hydraulic pressure available"}},
			{ID: "STATE-SERVICE-MODE", Name: "service mode is active", Aliases: []string{"service mode"}},
			{ID: "STATE-DEGRADED", Name: "degraded mode is active", Aliases: []string{"degraded mode"}},
			{ID: "STATE-RUNWAY-MODE", Name: "runway mode is active", Aliases: []string{"runway mode"}},
			{ID: "STATE-AUTOPILOT", Name: "autopilot is engaged", Aliases: []string{"autopilot engaged"}},
			{ID: "STATE-CARD-ABSENT", Name: "there is no card in the ATM", Aliases: []string{"no card in the ATM"}},
			{ID: "STATE-DOOR-OPEN", Name: "the door is open", Aliases: []string{"door is open"}},
			{ID: "STATE-DOOR-CLOSED", Name: "the door is closed", Aliases: []string{"door is closed"}},
		},
		Features: []CatalogEntry{
			{ID: "FEAT-REV", Name: "reverse thrust is installed", Aliases: []string{"reverse thrust installed"}},
			{ID: "FEAT-REMOTE-START", Name: "remote start is enabled", Aliases: []string{"remote start"}},
			{ID: "FEAT-PREMIUM-NAV", Name: "premium navigation is enabled", Aliases: []string{"premium navigation", "premium nav"}},
			{ID: "FEAT-SUNROOF", Name: "sunroof is installed", Aliases: []string{"sunroof"}},
			{ID: "FEAT-DATALINK", Name: "data link is installed", Aliases: []string{"data link installed"}},
		},
		Modes: []CatalogEntry{
			{ID: "MODE-NORMAL", Name: "normal mode is active", Aliases: []string{"normal mode"}},
			{ID: "MODE-DEGRADED", Name: "degraded mode is active", Aliases: []string{"degraded mode"}},
			{ID: "MODE-MAINT", Name: "maintenance mode is active", Aliases: []string{"maintenance mode"}},
		},
		Conditions: []CatalogEntry{
			{ID: "COND-OVERTEMP", Name: "engine overtemperature exists", Aliases: []string{"engine overtemperature"}},
			{ID: "COND-POWER-UNSTABLE", Name: "power is unstable", Aliases: []string{"unstable power"}},
			{ID: "COND-SAFE-TO-OPEN", Name: "it is safe to open the door", Aliases: []string{"safe to open"}},
			{ID: "COND-SAFE-TO-CLOSE", Name: "it is safe to close the door", Aliases: []string{"safe to close"}},
		},
		DataTerms: []CatalogEntry{
			{ID: "DATA-BUS-VOLTAGE", Name: "bus voltage"},
			{ID: "DATA-BRAKE-TORQUE", Name: "brake torque"},
			{ID: "DATA-CREDIT-CARD-NUMBER", Name: "credit card number", Aliases: []string{"card number"}},
		},
	}
}

func hasDiagCode(diags []Diagnostic, code string) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

type validity string

const (
	mustBeValid   validity = "valid"
	mustBeInvalid validity = "invalid"
	eitherValid   validity = "either"
)

func TestProvidedCorpus(t *testing.T) {
	catalog := fullCatalog()
	cases := []struct {
		id      string
		text    string
		pattern Pattern
		valid   validity
		diags   []string
	}{
		// Canonical valid
		{"V001", "The brake control system shall apply brake torque.", PatternUbiquitous, mustBeValid, nil},
		{"V002", "While aircraft is on ground, the engine control system shall enable reverse thrust.", PatternStateDriven, mustBeValid, nil},
		{"V003", "When brake is requested, the brake control system shall apply brake torque.", PatternEventDriven, mustBeValid, nil},
		{"V004", "Where sunroof is installed, the door control system shall display the sunroof control.", PatternOptionalFeature, mustBeValid, nil},
		{"V005", "If an invalid credit card number is entered, then the ATM shall display please re-enter credit card details.", PatternUnwantedBehavior, mustBeValid, nil},
		{"V006", "While aircraft is on ground, when reverse thrust is commanded, the engine control system shall enable reverse thrust.", PatternComplex, mustBeValid, nil},

		// Valid hard
		{"V102", "While aircraft is on ground and (hydraulic pressure is available or service mode is active), when reverse thrust is commanded, the engine control system shall enable reverse thrust.", PatternComplex, mustBeValid, nil},
		{"V105", "Where remote start is enabled and premium navigation is enabled, the power management system shall precondition the cabin.", PatternOptionalFeature, mustBeValid, nil},
		{"V108", "While aircraft is on ground, where reverse thrust is installed, when reverse thrust is commanded, the engine control system shall enable reverse thrust.", PatternComplex, mustBeValid, nil},
		{"V109", "While aircraft is on ground and not service mode is active, when reverse thrust is commanded or reverse thrust is commanded by pilot, the engine control system shall enable reverse thrust.", PatternComplex, mustBeValid, nil},
		{"V110", "When low voltage event and (sensor fault is detected or data link is lost), the diagnostic system shall record a degraded-power fault.", PatternEventDriven, mustBeValid, nil},

		// Alias-heavy valid
		{"V201", "When brake request, the brake controller shall apply brake torque.", PatternEventDriven, mustBeValid, []string{"lint.alias_used"}},
		{"V202", "While on ground, when reverse thrust commanded, the FADEC shall enable reverse thrust.", PatternComplex, mustBeValid, []string{"lint.alias_used"}},
		{"V204", "Where premium nav, the nav system shall display turn-by-turn guidance.", PatternOptionalFeature, mustBeValid, []string{"lint.alias_used"}},

		// Ambiguity traps
		{"A301", "When reverse thrust command, the ECS shall enable reverse thrust.", PatternEventDriven, mustBeInvalid, []string{"catalog.system_ambiguous", "catalog.event_ambiguous"}},
		{"A303", "While degraded mode, when sensor fault, the diagnostics shall raise an alert.", PatternComplex, eitherValid, []string{"catalog.state_ambiguous"}},

		// Invalid shell
		{"I402", "If sensor fault is detected, the diagnostic system shall store a fault record.", PatternUnwantedBehavior, mustBeInvalid, []string{"ears.invalid_if_then_form"}},
		{"I404", "While aircraft is on ground, the engine control system shall shall enable reverse thrust.", PatternStateDriven, mustBeInvalid, []string{"ears.multiple_shall"}},
		{"I405", "The engine control system enable reverse thrust.", "", mustBeInvalid, []string{"ears.missing_shall"}},
		{"I406", "When reverse thrust is commanded, the engine control system.", "", mustBeInvalid, []string{"ears.missing_shall"}},
		{"I410", "When reverse thrust is commanded, if emergency braking is active, then the engine control system shall enable reverse thrust.", PatternComplex, mustBeInvalid, []string{"ears.invalid_clause_order"}},

		// Invalid boolean expressions
		{"I501", "While aircraft is on ground and, when reverse thrust is commanded, the engine control system shall enable reverse thrust.", PatternComplex, mustBeInvalid, []string{"expr.invalid_operator_sequence"}},
		{"I502", "While (aircraft is on ground and hydraulic pressure is available, when reverse thrust is commanded, the engine control system shall enable reverse thrust.", "", mustBeInvalid, []string{"expr.unbalanced_parentheses"}},
		{"I506", "While aircraft is on ground or or hydraulic pressure is available, the engine control system shall enable reverse thrust.", PatternStateDriven, mustBeInvalid, []string{"expr.invalid_operator_sequence"}},

		// Near misses / stress
		{"N701", "The engine control system should enable reverse thrust when commanded.", "", mustBeInvalid, []string{"ears.missing_shall"}},
		{"S801", "  While   aircraft is on ground ,   when reverse thrust is commanded , the engine control system shall enable reverse thrust . ", PatternComplex, mustBeValid, nil},
		{"S803", "IF sensor fault is detected, THEN the diagnostic system shall store a fault record.", PatternUnwantedBehavior, mustBeValid, nil},
		{"K901", "While (aircraft is on ground and (hydraulic pressure is available or (service mode is active and not degraded mode is active))), when ((reverse thrust is commanded) or (emergency braking is active and not data link is lost)), the engine control system shall enable reverse thrust.", PatternComplex, mustBeValid, nil},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			res := LintEars(tc.text, catalog, &Options{Mode: ModeStrict, CommaAsAnd: true})
			if tc.pattern != "" && res.Pattern != tc.pattern {
				t.Fatalf("pattern mismatch: want=%q got=%q", tc.pattern, res.Pattern)
			}
			switch tc.valid {
			case mustBeValid:
				if !res.Valid {
					t.Fatalf("expected valid, got invalid with diagnostics: %+v", res.Diagnostics)
				}
			case mustBeInvalid:
				if res.Valid {
					t.Fatalf("expected invalid, got valid")
				}
			}
			for _, code := range tc.diags {
				if !hasDiagCode(res.Diagnostics, code) {
					t.Fatalf("expected diagnostic code %q, got: %+v", code, res.Diagnostics)
				}
			}
		})
	}
}
