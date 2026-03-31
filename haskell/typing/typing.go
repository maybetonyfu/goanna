package typing

import (
	prolog "goanna/prolog-tool"
)

// RuleKind distinguishes whether a rule describes a type or an instance.
type RuleKind string

const (
	RuleKindType     RuleKind = "type"
	RuleKindInstance RuleKind = "instance"
)

// RuleHead corresponds to state.py's RuleHead: the head of a Prolog clause
// that types a declaration or instance.
type RuleHead struct {
	Kind   RuleKind
	Name   string
	Module string
	ID     *int
}

func (rh RuleHead) String() string {
	return rh.Name
}

// Rule corresponds to state.py's Rule: a Prolog clause with a head and a body.
type Rule struct {
	Head   RuleHead
	Body   prolog.LTerm
	NodeID *int
	Axiom  bool
	ID     *int
}

func (r Rule) String() string {
	return r.Head.String() + " :- " + r.Body.String()
}

// TypingEnv holds the accumulated typing rules and the decl map.
// The decl map mirrors state.py's declarations list as a set (map to struct{})
// for O(1) membership, and also carries the canonical → declaration name mapping.
type TypingEnv struct {
	Rules        []*Rule
	DeclMap      map[string][]string // canonical decl name → related names (mirrors meta decl maps)
	Declarations []string
	Collectors   map[string][]string // head name → list of class-var names (mirrors state.py collectors)
}

func NewTypingEnv() *TypingEnv {
	return &TypingEnv{
		Rules:        make([]*Rule, 0),
		DeclMap:      make(map[string][]string),
		Declarations: make([]string, 0),
		Collectors:   make(map[string][]string),
	}
}

// AddClassVar records a class-variable name for the given head, mirroring
// state.py's add_class_var / collectors defaultdict.
func (te *TypingEnv) AddClassVar(headName string, classVar string) {
	te.Collectors[headName] = append(te.Collectors[headName], classVar)
}

// IsParentOf reports whether parent is an ancestor of child in the decl map
// (where DeclMap[child] lists the child's ancestors/parents).
func (te *TypingEnv) IsParentOf(parent string, child string) bool {
	ancestors, ok := te.DeclMap[child]
	if !ok {
		return false
	}
	for _, a := range ancestors {
		if a == parent {
			return true
		}
	}
	return false
}

// AddRule appends a rule, mirroring state.py's add_rule which also sets rule.id.
func (te *TypingEnv) AddRule(rule *Rule) {
	rule.ID = rule.NodeID
	te.Rules = append(te.Rules, rule)
}
