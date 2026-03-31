package prolog_tool

import "strings"

// LTerm is the interface for all constructable Prolog terms.
type LTerm interface {
	String() string
}

// LVar is a Prolog variable (starts with uppercase or _).
type LVar struct {
	Value string
}

func (v LVar) String() string { return v.Value }

// LAtom is a Prolog atom (lowercase identifier or quoted symbol).
type LAtom struct {
	Value string
}

func (a LAtom) String() string { return a.Value }

// LStruct is a Prolog compound term: functor(arg1, arg2, ...).
// The special functor "=" is printed infix.
type LStruct struct {
	Functor string
	Args    []LTerm
}

func (s LStruct) String() string {
	if s.Functor == "=" && len(s.Args) == 2 {
		return s.Args[0].String() + " = " + s.Args[1].String()
	}
	parts := make([]string, len(s.Args))
	for i, a := range s.Args {
		parts[i] = a.String()
	}
	return s.Functor + "(" + strings.Join(parts, ", ") + ")"
}

// LList is a Prolog list: [elem1, elem2, ...].
type LList struct {
	Elements []LTerm
}

func (l LList) String() string {
	parts := make([]string, len(l.Elements))
	for i, e := range l.Elements {
		parts[i] = e.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// ---------------------------------------------------------------------------
// Pre-defined special variables
// ---------------------------------------------------------------------------

var (
	T        = LVar{Value: "T"}
	Call     = LVar{Value: "Calls"}
	Call_    = LVar{Value: "Calls_"}
	GammaVar = LVar{Value: "Gamma"}
	GammaVar_ = LVar{Value: "Gamma_"}
	ZetaVar  = LVar{Value: "Zeta"}
	ZetaVar_ = LVar{Value: "Zeta_"}
	Classes  = LVar{Value: "Classes"}
	Wildcard = LVar{Value: "_"}
)

// ---------------------------------------------------------------------------
// Pre-defined special atoms
// ---------------------------------------------------------------------------

var (
	Succeed = LAtom{Value: "true"}
	Fail    = LAtom{Value: "false"}
	Nil     = LAtom{Value: "nil"}
	Cut     = LAtom{Value: "!"}
)

// ---------------------------------------------------------------------------
// Special functors / predicates
// ---------------------------------------------------------------------------

// Cons constructs a Prolog list cell: [X | XS].
func Cons(x LTerm, xs LTerm) LTerm {
	return LStruct{Functor: "[|]", Args: []LTerm{x, xs}}
}

// Unify constructs an eq(A, B) term. String arguments are interpreted as
// atoms (lowercase) or variables (uppercase / _).
func Unify(a, b any) LTerm {
	return LStruct{Functor: "eq", Args: []LTerm{toTerm(a), toTerm(b)}}
}

// UnifyAll constructs an all_equal([...]) term.
func UnifyAll(terms []LTerm) LTerm {
	return LStruct{Functor: "all_equal", Args: []LTerm{LList{Elements: terms}}}
}

// Once wraps a term in once(...).
func Once(term LTerm) LTerm {
	return LStruct{Functor: "once", Args: []LTerm{term}}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// toTerm converts a string to LAtom or LVar by Prolog convention, or returns
// an LTerm unchanged.
func toTerm(v any) LTerm {
	switch s := v.(type) {
	case string:
		if len(s) > 0 && (s[0] == '_' || (s[0] >= 'A' && s[0] <= 'Z')) {
			return LVar{Value: s}
		}
		return LAtom{Value: s}
	case LTerm:
		return s
	default:
		panic("toTerm: unsupported type")
	}
}
