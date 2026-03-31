package meta

import (
	"goanna/haskell/parser"
)

// GetDeclTypeVars returns a map from each declaration's canonical name to the
// list of type variable canonicals (P0, P1, ...) that appear in its type
// signature, in order of first appearance.
//
// A TypeSig may name multiple declarations (e.g. `f, g :: a -> b`); both
// receive the same type variable list.
//
// DataDecl and TypeDecl contribute their DeclHead type variables.
// ClassDecl contributes its DeclHead type variables.
func GetDeclTypeVars(modules []*parser.Module) map[string][]string {
	result := make(map[string][]string)

	for _, m := range modules {
		collectFromDecls(m.Decls, result)
	}

	return result
}

func collectFromDecls(decls []parser.Decl, result map[string][]string) {
	for _, decl := range decls {
		switch d := decl.(type) {
		case *parser.TypeSig:
			// Collect unique TyVar canonicals from the signature type, in order.
			vars := tyVarsFromType(d.Ty)
			// Map every named decl in this sig to the same var list.
			for _, canonical := range d.Canonicals {
				if canonical != "" {
					result[canonical] = vars
				}
			}

		case *parser.DataDecl:
			// data Foo a b = ... → type vars are the DeclHead's TypeVars
			name := d.DHead.Canonical
			if name == "" {
				name = d.DHead.Name
			}
			result[name] = tyVarsFromDeclHead(d.DHead)

		case *parser.TypeDecl:
			// type Foo a b = ... → type vars from DeclHead
			name := d.DHead.Canonical
			if name == "" {
				name = d.DHead.Name
			}
			result[name] = tyVarsFromDeclHead(d.DHead)

		case *parser.ClassDecl:
			// class Foo a where ... → type vars from DeclHead
			name := d.DHead.Canonical
			if name == "" {
				name = d.DHead.Name
			}
			result[name] = tyVarsFromDeclHead(d.DHead)
			// Recurse into class body (method type sigs).
			collectFromDecls(d.Decls, result)

		case *parser.InstDecl:
			// Recurse into instance body.
			collectFromDecls(d.Body, result)

		case *parser.PatBind:
			// PatBinds can have where-clauses with nested sigs.
			collectFromRhs(d.Rhs, result)
		}
	}
}

func collectFromRhs(rhs parser.Rhs, result map[string][]string) {
	switch r := rhs.(type) {
	case *parser.UnguardedRhs:
		collectFromDecls(r.Wheres, result)
	case *parser.GuardedRhs:
		collectFromDecls(r.Wheres, result)
	}
}

// tyVarsFromType collects unique TyVar canonicals from a type, in order of
// first appearance.
func tyVarsFromType(ty parser.Type) []string {
	seen := make(map[string]bool)
	var vars []string
	collectTyVars(ty, seen, &vars)
	return vars
}

func collectTyVars(ty parser.Type, seen map[string]bool, vars *[]string) {
	if ty == nil {
		return
	}
	switch t := ty.(type) {
	case *parser.TyVar:
		canon := t.Canonical
		if canon == "" {
			canon = t.Name
		}
		if canon != "" && !seen[canon] {
			seen[canon] = true
			*vars = append(*vars, canon)
		}
	case *parser.TyApp:
		collectTyVars(t.Ty1, seen, vars)
		collectTyVars(t.Ty2, seen, vars)
	case *parser.TyFunction:
		collectTyVars(t.Ty1, seen, vars)
		collectTyVars(t.Ty2, seen, vars)
	case *parser.TyTuple:
		for _, sub := range t.Tys {
			collectTyVars(sub, seen, vars)
		}
	case *parser.TyList:
		collectTyVars(t.Ty, seen, vars)
	case *parser.TyForall:
		for i := range t.Assertions {
			for _, aty := range t.Assertions[i].Types {
				collectTyVars(aty, seen, vars)
			}
		}
		collectTyVars(t.Ty, seen, vars)
	case *parser.TyCon:
		// no type variables
	}
}

// tyVarsFromDeclHead extracts canonicals from a DeclHead's TypeVars slice.
func tyVarsFromDeclHead(dh parser.DeclHead) []string {
	vars := make([]string, 0, len(dh.TypeVars))
	for _, tv := range dh.TypeVars {
		canon := tv.Canonical
		if canon == "" {
			canon = tv.Name
		}
		if canon != "" {
			vars = append(vars, canon)
		}
	}
	return vars
}
