package earslint

type Mode string

const (
	ModeStrict Mode = "strict"
	ModeGuided Mode = "guided"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type Pattern string

const (
	PatternUbiquitous       Pattern = "ubiquitous"
	PatternStateDriven      Pattern = "state-driven"
	PatternEventDriven      Pattern = "event-driven"
	PatternOptionalFeature  Pattern = "optional-feature"
	PatternUnwantedBehavior Pattern = "unwanted-behaviour"
	PatternComplex          Pattern = "complex"
)

type ClauseType string

const (
	ClauseWhile ClauseType = "while"
	ClauseWhen  ClauseType = "when"
	ClauseWhere ClauseType = "where"
	ClauseIf    ClauseType = "if"
)

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

type Span struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type CatalogEntry struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Aliases []string `json:"aliases,omitempty"`
}

type Catalog struct {
	Systems    []CatalogEntry `json:"systems,omitempty"`
	Actors     []CatalogEntry `json:"actors,omitempty"`
	Events     []CatalogEntry `json:"events,omitempty"`
	States     []CatalogEntry `json:"states,omitempty"`
	Features   []CatalogEntry `json:"features,omitempty"`
	Modes      []CatalogEntry `json:"modes,omitempty"`
	Conditions []CatalogEntry `json:"conditions,omitempty"`
	DataTerms  []CatalogEntry `json:"dataTerms,omitempty"`
}

type CatalogRef struct {
	Group string `json:"group"`
	ID    string `json:"id"`
	Name  string `json:"name"`
}

type TermMatch struct {
	Raw        string       `json:"raw"`
	Role       TermRole     `json:"role"`
	Matched    *CatalogRef  `json:"matched,omitempty"`
	Ambiguous  []CatalogRef `json:"ambiguous,omitempty"`
	Unresolved bool         `json:"unresolved,omitempty"`
	ViaAlias   bool         `json:"viaAlias,omitempty"`
}

type ClauseExpr struct {
	Kind string `json:"kind"`
	Span *Span  `json:"span,omitempty"`

	Term *TermMatch `json:"term,omitempty"`
	Text string     `json:"text,omitempty"`

	Items []ClauseExpr `json:"items,omitempty"`
	Item  *ClauseExpr  `json:"item,omitempty"`
}

type EarsAST struct {
	Pattern Pattern `json:"pattern"`

	Preconditions *ClauseExpr `json:"preconditions,omitempty"`
	Trigger       *ClauseExpr `json:"trigger,omitempty"`
	Feature       *ClauseExpr `json:"feature,omitempty"`

	System    TermMatch `json:"system"`
	Responses []string  `json:"responses"`
	Raw       string    `json:"raw"`
}

type ReferenceMatch struct {
	Clause     string       `json:"clause"`
	Text       string       `json:"text"`
	Role       TermRole     `json:"role"`
	Matched    *CatalogRef  `json:"matched,omitempty"`
	Ambiguous  []CatalogRef `json:"ambiguous,omitempty"`
	Unresolved bool         `json:"unresolved,omitempty"`
	ViaAlias   bool         `json:"viaAlias,omitempty"`
	Span       *Span        `json:"span,omitempty"`
}

type Diagnostic struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Span     *Span    `json:"span,omitempty"`
}

type LintResult struct {
	ID          string           `json:"id,omitempty"`
	Valid       bool             `json:"valid"`
	Pattern     Pattern          `json:"pattern,omitempty"`
	AST         *EarsAST         `json:"ast,omitempty"`
	References  []ReferenceMatch `json:"references"`
	Diagnostics []Diagnostic     `json:"diagnostics"`
}

type Options struct {
	Mode       Mode     `json:"mode,omitempty"`
	CommaAsAnd bool     `json:"commaAsAnd,omitempty"`
	VagueTerms []string `json:"vagueTerms,omitempty"`
}
