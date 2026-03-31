package meta

import (
	"goanna/haskell/parser"
)

// GetClassSuperclasses traverses all modules and returns a map from each
// typeclass canonical name to all of its superclasses (direct and transitive).
//
// For example, given:
//
//	class Eq a
//	class (Eq a) => Ord a
//	class (Ord a) => Enum a
//
// The result will be:
//
//	"Eq"   -> []
//	"Ord"  -> ["Eq"]
//	"Enum" -> ["Ord", "Eq"]
//
// Note: only classes declared in the parsed modules are present as keys.
// Superclasses referenced but not declared (e.g. from external libraries)
// will appear in the value lists but not as keys.
func GetClassSuperclasses(modules []*parser.Module) map[string][]string {
	// Step 1: collect direct (subclass, superclass) pairs from ClassDecl nodes.
	type pair struct{ sub, super string }
	var pairs []pair

	for _, m := range modules {
		for _, decl := range m.Decls {
			cd, ok := decl.(*parser.ClassDecl)
			if !ok {
				continue
			}
			className := cd.DHead.Canonical
			if className == "" {
				className = cd.DHead.Name
			}
			// Ensure the class appears as a key even if it has no superclasses.
			pairs = append(pairs, pair{className, ""})
			for _, assertion := range cd.Assertions {
				superName := assertion.Canonical
				if superName == "" {
					superName = assertion.Name
				}
				if superName != "" {
					pairs = append(pairs, pair{className, superName})
				}
			}
		}
	}

	// Step 2: build direct superclass map (sub -> []direct supers).
	direct := make(map[string][]string)
	for _, p := range pairs {
		if p.super == "" {
			if _, ok := direct[p.sub]; !ok {
				direct[p.sub] = []string{}
			}
		} else {
			direct[p.sub] = append(direct[p.sub], p.super)
		}
	}

	// Step 3: transitively close — for each class, collect all ancestors via DFS.
	memo := make(map[string][]string)
	visiting := make(map[string]bool)

	var allAncestors func(name string) []string
	allAncestors = func(name string) []string {
		if result, ok := memo[name]; ok {
			return result
		}
		if visiting[name] {
			return nil
		}
		visiting[name] = true
		defer func() { visiting[name] = false }()

		seen := make(map[string]bool)
		var result []string
		for _, super := range direct[name] {
			if !seen[super] {
				seen[super] = true
				result = append(result, super)
			}
			for _, ancestor := range allAncestors(super) {
				if !seen[ancestor] {
					seen[ancestor] = true
					result = append(result, ancestor)
				}
			}
		}
		memo[name] = result
		return result
	}

	out := make(map[string][]string, len(direct))
	for class := range direct {
		out[class] = allAncestors(class)
	}
	return out
}
