# ears-lint-go

Deterministic standalone Go library for linting EARS requirement sentences.

## Scope

This library only does:
- EARS shell parsing and pattern classification
- boolean clause parsing inside `While` / `When` / `Where` / `If`
- deterministic catalog matching
- machine-readable diagnostics

## Installation

```bash
go get github.com/labeth/ears-lint-go
```

## Public API

```go
func LintEars(text string, catalog Catalog, options *Options) LintResult
func LintEarsBatch(items [][2]string, catalog Catalog, options *Options) []LintResult
```

`LintEarsBatch` input item format:
- `[2]string{ id, text }`

## Input Types

### Options

```go
type Options struct {
    Mode       Mode
    CommaAsAnd bool
    VagueTerms []string
}
```

Defaults:
- `Mode`: `strict`
- `CommaAsAnd`: `false`
- `VagueTerms`: `appropriate`, `sufficient`, `as needed`

### Mode

```go
type Mode string

const (
    ModeStrict Mode = "strict"
    ModeGuided Mode = "guided"
)
```

Mode behavior:
- `strict`: structural parse failures and certain validation failures are errors
- `guided`: structural parse failures are downgraded to warnings where applicable

### Catalog

```go
type Catalog struct {
    Systems    []CatalogEntry
    Actors     []CatalogEntry
    Events     []CatalogEntry
    States     []CatalogEntry
    Features   []CatalogEntry
    Modes      []CatalogEntry
    Conditions []CatalogEntry
    DataTerms  []CatalogEntry
}

type CatalogEntry struct {
    ID      string
    Name    string
    Aliases []string
}
```

Matching policy (deterministic only):
1. exact canonical name
2. exact alias
3. ambiguous (multiple)
4. unresolved (none)

No fuzzy or semantic matching is used.

## Output Types

### LintResult

```go
type LintResult struct {
    ID          string
    Valid       bool
    Pattern     Pattern
    AST         *EarsAST
    References  []ReferenceMatch
    Diagnostics []Diagnostic
}
```

`Valid` is computed only from diagnostics severity:
- `false` if at least one `error`
- `true` otherwise

### Pattern

```go
type Pattern string

const (
    PatternUbiquitous       Pattern = "ubiquitous"
    PatternStateDriven      Pattern = "state-driven"
    PatternEventDriven      Pattern = "event-driven"
    PatternOptionalFeature  Pattern = "optional-feature"
    PatternUnwantedBehavior Pattern = "unwanted-behaviour"
    PatternComplex          Pattern = "complex"
)
```

### EarsAST

```go
type EarsAST struct {
    Pattern       Pattern
    Preconditions *ClauseExpr
    Trigger       *ClauseExpr
    Feature       *ClauseExpr
    System        TermMatch
    Responses     []string
    Raw           string
}
```

### ClauseExpr

```go
type ClauseExpr struct {
    Kind  string // term | and | or | not | group | free-text
    Span  *Span
    Term  *TermMatch
    Text  string
    Items []ClauseExpr
    Item  *ClauseExpr
}
```

### TermMatch

```go
type TermMatch struct {
    Raw        string
    Role       TermRole
    Matched    *CatalogRef
    Ambiguous  []CatalogRef
    Unresolved bool
    ViaAlias   bool
}
```

### CatalogRef

```go
type CatalogRef struct {
    Group string
    ID    string
    Name  string
}
```

### TermRole

```go
type TermRole string

const (
    RoleSystem    TermRole = "system"
    RoleActor     TermRole = "actor"
    RoleEvent     TermRole = "event"
    RoleState     TermRole = "state"
    RoleFeature   TermRole = "feature"
    RoleMode      TermRole = "mode"
    RoleCondition TermRole = "condition"
    RoleDataTerm  TermRole = "data-term"
)
```

### ReferenceMatch

```go
type ReferenceMatch struct {
    Clause     string // system | preconditions | trigger | feature
    Text       string
    Role       TermRole
    Matched    *CatalogRef
    Ambiguous  []CatalogRef
    Unresolved bool
    ViaAlias   bool
    Span       *Span
}
```

### Diagnostic

```go
type Diagnostic struct {
    Code     string
    Severity Severity
    Message  string
    Span     *Span
}

type Severity string

const (
    SeverityError   Severity = "error"
    SeverityWarning Severity = "warning"
    SeverityInfo    Severity = "info"
)
```

### Span

```go
type Span struct {
    Start int // 0-based start offset in input text
    End   int // 0-based exclusive end offset
}
```

## Parsing and Validation Behavior

### Supported EARS shell patterns
- `The <system> shall <response>`
- `While <expr>, the <system> shall <response>`
- `Where <expr>, the <system> shall <response>`
- `When <expr>, the <system> shall <response>`
- `If <expr>, then the <system> shall <response>`
- complex combinations from these shell clauses

### Shell clause order
Current accepted order:
- `While* -> Where* -> When* -> If* -> the <system> shall ...`

Violations can produce `ears.invalid_clause_order`.

### Clause expression grammar
Inside clause bodies, parser supports:
- `and`
- `or`
- `not`
- parentheses
- comma as `and` only when `CommaAsAnd=true`

Operator precedence:
- `not` > `and` > `or`

### Keywords and casing
Shell keywords are matched case-insensitively (`while`, `WHEN`, `If`, `THEN` all parse).

### Response extraction
The response text after `shall` is split by semicolon (`;`) into `Responses`.

## Complete Diagnostic Codes

This is the full set of codes currently emitted by implementation.

### EARS shell diagnostics
- `ears.no_match`
- `ears.invalid_clause_order`
- `ears.missing_system`
- `ears.missing_shall`
- `ears.multiple_shall`
- `ears.invalid_if_then_form`

### Expression diagnostics
- `expr.unbalanced_parentheses`
- `expr.invalid_operator_sequence`
- `expr.empty_subexpression`
- `expr.operator_precedence_warning`
- `expr.mixed_unresolved_terms`
- `expr.ambiguous_term`
- `expr.unknown_term`

### Catalog diagnostics
Generated as:
- `catalog.<role>_unresolved`
- `catalog.<role>_ambiguous`

Roles currently emitted from this parser pipeline:
- `system`
- `state`
- `event`
- `feature`

Concrete examples:
- `catalog.system_unresolved`
- `catalog.system_ambiguous`
- `catalog.state_unresolved`
- `catalog.state_ambiguous`
- `catalog.event_unresolved`
- `catalog.event_ambiguous`
- `catalog.feature_unresolved`
- `catalog.feature_ambiguous`

### Lint diagnostics
- `lint.multiple_responses`
- `lint.vague_response`
- `lint.unparsed_tail`
- `lint.alias_used`
- `lint.suspicious_text_shape`

## Severity Rules

- `severityByMode(strict) = error`
- `severityByMode(guided) = warning`

Applied to structural errors and selected expression diagnostics.

Catalog severity notes:
- unresolved/ambiguous `system` uses mode severity
- unresolved/ambiguous non-system clause terms are warnings

## Determinism Guarantees

- No external calls
- No random behavior
- Diagnostics are stably sorted by span, code, message, severity
- Batch output order matches input order

## Usage Example (single item)

```go
package main

import (
    "fmt"

    earslint "github.com/labeth/ears-lint-go"
)

func main() {
    catalog := earslint.Catalog{
        Systems: []earslint.CatalogEntry{{ID: "SYS-ENGINE", Name: "engine control system"}},
        States: []earslint.CatalogEntry{{ID: "STATE-GROUND", Name: "aircraft is on ground"}},
        Events: []earslint.CatalogEntry{{ID: "EVT-REV", Name: "reverse thrust is commanded"}},
    }

    res := earslint.LintEars(
        "While aircraft is on ground, when reverse thrust is commanded, the engine control system shall enable reverse thrust.",
        catalog,
        &earslint.Options{Mode: earslint.ModeStrict},
    )

    fmt.Println(res.Valid, res.Pattern, len(res.Diagnostics))
}
```

## Batch JSON Example

Runnable example that prints raw API output as JSON:
- program: `examples/dump_batch_json/main.go`
- run:

```bash
go run ./examples/dump_batch_json > output.json
```

## License

MIT. See [LICENSE](./LICENSE).

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).
