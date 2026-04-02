package main

import (
	"encoding/json"
	"os"

	earslint "github.com/labeth/ears-lint-go"
)

func main() {
	catalog := earslint.Catalog{
		Systems: []earslint.CatalogEntry{
			{ID: "SYS-ENG-CTRL", Name: "engine control system", Aliases: []string{"FADEC", "ECS"}},
			{ID: "SYS-ENV-CTRL", Name: "environmental control system", Aliases: []string{"ECS"}},
			{ID: "SYS-BRAKE-CTRL", Name: "brake control system", Aliases: []string{"brake controller", "BCS"}},
		},
		Events: []earslint.CatalogEntry{
			{ID: "EVT-REV-CMD", Name: "reverse thrust is commanded", Aliases: []string{"reverse thrust command"}},
			{ID: "EVT-REV-CMD-PILOT", Name: "reverse thrust is commanded by pilot", Aliases: []string{"reverse thrust command"}},
			{ID: "EVT-BRAKE-REQ", Name: "brake is requested", Aliases: []string{"brake request"}},
		},
		States: []earslint.CatalogEntry{
			{ID: "STATE-ONGROUND", Name: "aircraft is on ground", Aliases: []string{"on ground"}},
		},
	}

	items := [][2]string{
		{"V001", "The brake control system shall apply brake torque."},
		{"V002", "While aircraft is on ground, when reverse thrust is commanded, the engine control system shall enable reverse thrust."},
		{"A301", "When reverse thrust command, the ECS shall enable reverse thrust."},
	}

	results := earslint.LintEarsBatch(items, catalog, &earslint.Options{
		Mode:       earslint.ModeStrict,
		CommaAsAnd: true,
	})

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(results); err != nil {
		panic(err)
	}
}
