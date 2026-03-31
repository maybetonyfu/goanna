package rename

import (
	"fmt"
	"goanna/haskell/parser"
)

// RenameTypeVars assigns canonical names P0, P1, P2, ... to every TyVar in
// every TypeSig across all modules.
//
// The rules are:
//   - Within a single TypeSig, TyVars with the same source name share the
//     same canonical (e.g. both `a` in `f :: a -> a` become the same Pn).
//   - Across different TypeSigs, TyVars always get distinct canonicals even
//     if their source names are identical.
//
// The function mutates TyVar nodes in place by setting their Canonical field.
func RenameTypeVars(modules []*parser.Module) {
	counter := 0

	for _, m := range modules {
		for _, decl := range m.Decls {
			renameTyVarsInDecl(decl, &counter)
		}
	}
}

// renameTyVarsInDecl recurses into declarations. TypeSigs are handled directly;
// ClassDecl and InstDecl bodies contain nested decls that we recurse into.
func renameTyVarsInDecl(decl parser.Decl, counter *int) {
	switch d := decl.(type) {
	case *parser.TypeSig:
		renameTyVarsInTypeSig(d, counter)
	case *parser.ClassDecl:
		for _, inner := range d.Decls {
			renameTyVarsInDecl(inner, counter)
		}
	case *parser.InstDecl:
		for _, inner := range d.Body {
			renameTyVarsInDecl(inner, counter)
		}
	case *parser.PatBind:
		renameTyVarsInRhs(d.Rhs, counter)
	}
}

// renameTyVarsInTypeSig assigns fresh canonicals to all TyVars within one
// TypeSig. Same source name → same canonical; the global counter advances
// once per distinct name in this sig.
func renameTyVarsInTypeSig(sig *parser.TypeSig, counter *int) {
	// local map: source name → canonical for this sig
	local := make(map[string]string)

	assignCanonical := func(tv *parser.TyVar) {
		if tv == nil {
			return
		}
		canon, ok := local[tv.Name]
		if !ok {
			canon = fmt.Sprintf("P%d", *counter)
			*counter++
			local[tv.Name] = canon
		}
		tv.Canonical = canon
	}

	walkType(sig.Ty, assignCanonical)
}

// renameTyVarsInRhs handles RHS nodes that can contain type annotations
// (e.g. expression type signatures inside where/let).
func renameTyVarsInRhs(rhs parser.Rhs, counter *int) {
	switch r := rhs.(type) {
	case *parser.UnguardedRhs:
		for _, w := range r.Wheres {
			renameTyVarsInDecl(w, counter)
		}
	case *parser.GuardedRhs:
		for _, w := range r.Wheres {
			renameTyVarsInDecl(w, counter)
		}
	}
}

// walkType recursively visits every Type node and calls f on each TyVar.
func walkType(ty parser.Type, f func(*parser.TyVar)) {
	if ty == nil {
		return
	}
	switch t := ty.(type) {
	case *parser.TyVar:
		f(t)
	case *parser.TyCon:
		// no children
	case *parser.TyApp:
		walkType(t.Ty1, f)
		walkType(t.Ty2, f)
	case *parser.TyFunction:
		walkType(t.Ty1, f)
		walkType(t.Ty2, f)
	case *parser.TyTuple:
		for _, ty := range t.Tys {
			walkType(ty, f)
		}
	case *parser.TyList:
		walkType(t.Ty, f)
	case *parser.TyForall:
		for i := range t.Assertions {
			for _, aty := range t.Assertions[i].Types {
				walkType(aty, f)
			}
		}
		walkType(t.Ty, f)
	}
}
