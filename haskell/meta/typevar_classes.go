package meta

import (
	"goanna/haskell/parser"
)

// GetTypeVarClasses returns a map from each type variable canonical (e.g. P0)
// to the deduplicated list of typeclass canonicals that constrain it, including
// transitive superclasses.
//
// superclasses is the output of GetClassSuperclasses: a map from class canonical
// to all its ancestor class canonicals.
func GetTypeVarClasses(modules []*parser.Module, superclasses map[string][]string) map[string][]string {
	// Collect direct (tyvar canonical -> class canonical) pairs from all TypeSigs.
	// We use a map of sets to deduplicate as we go.
	direct := make(map[string]map[string]bool)

	for _, m := range modules {
		collectConstraints(m.Decls, direct)
	}

	// Expand each tyvar's class set with transitive superclasses.
	result := make(map[string][]string, len(direct))
	for tyvar, classSet := range direct {
		expanded := make(map[string]bool)
		for cls := range classSet {
			expanded[cls] = true
			for _, super := range superclasses[cls] {
				expanded[super] = true
			}
		}
		list := make([]string, 0, len(expanded))
		for cls := range expanded {
			list = append(list, cls)
		}
		result[tyvar] = list
	}

	return result
}

func collectConstraints(decls []parser.Decl, direct map[string]map[string]bool) {
	for _, decl := range decls {
		switch d := decl.(type) {
		case *parser.TypeSig:
			collectFromTypeSig(d.Ty, direct)
		case *parser.ClassDecl:
			collectConstraints(d.Decls, direct)
		case *parser.InstDecl:
			collectConstraints(d.Body, direct)
		case *parser.PatBind:
			collectConstraintsFromRhs(d.Rhs, direct)
		}
	}
}

func collectConstraintsFromRhs(rhs parser.Rhs, direct map[string]map[string]bool) {
	switch r := rhs.(type) {
	case *parser.UnguardedRhs:
		collectConstraints(r.Wheres, direct)
	case *parser.GuardedRhs:
		collectConstraints(r.Wheres, direct)
	}
}

// collectFromTypeSig walks a Type looking for TyForall nodes and records the
// (tyvar -> class) constraints found in assertions.
func collectFromTypeSig(ty parser.Type, direct map[string]map[string]bool) {
	if ty == nil {
		return
	}
	switch t := ty.(type) {
	case *parser.TyForall:
		for _, assertion := range t.Assertions {
			className := assertion.Canonical
			if className == "" {
				className = assertion.Name
			}
			if className == "" {
				continue
			}
			for _, aty := range assertion.Types {
				tv, ok := aty.(*parser.TyVar)
				if !ok {
					continue
				}
				tvCanon := tv.Canonical
				if tvCanon == "" {
					tvCanon = tv.Name
				}
				if tvCanon == "" {
					continue
				}
				if direct[tvCanon] == nil {
					direct[tvCanon] = make(map[string]bool)
				}
				direct[tvCanon][className] = true
			}
		}
		// Recurse into the body type (there may be nested foralls).
		collectFromTypeSig(t.Ty, direct)
	case *parser.TyApp:
		collectFromTypeSig(t.Ty1, direct)
		collectFromTypeSig(t.Ty2, direct)
	case *parser.TyFunction:
		collectFromTypeSig(t.Ty1, direct)
		collectFromTypeSig(t.Ty2, direct)
	case *parser.TyTuple:
		for _, sub := range t.Tys {
			collectFromTypeSig(sub, direct)
		}
	case *parser.TyList:
		collectFromTypeSig(t.Ty, direct)
	}
}
