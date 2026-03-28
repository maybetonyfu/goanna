package rename

import (
	"goanna/haskell/parser"
	"slices"
)

// Resolve mutates the module in place, resolving canonical names for all name nodes.
func Resolve(module *parser.Module, result RenameResult, importMap map[string][]parser.Import) {
	visitor := parser.NewTraverser(
		func(_ int, ast parser.AST, parent parser.AST) int {
			resolveNode(ast, module.Name, result, importMap)
			return 0
		},
		0,
	)
	visitor.Visit(module, nil)
}

// resolveNode processes individual nodes during resolution
// Sets canonical names for ExpVar nodes by finding the most specific matching identifier
func resolveNode(ast parser.AST, moduleName string, result RenameResult, importMap map[string][]parser.Import) {
	switch node := ast.(type) {
	case *parser.ExpVar:
		// Find all term identifiers that match the ExpVar name
		candidates := []TermIdentifier{}

		for _, term := range result.Terms {
			// Check if name matches
			if term.name != node.Name {
				continue
			}

			// Check if the effective range envelopes the ExpVar location
			if !envelopesLocation(term.effectiveRange, node.Loc()) {
				continue
			}

			// If ExpVar has a module qualifier, only consider that module
			if node.Module != "" {
				if term.module == node.Module {
					candidates = append(candidates, term)
				}
				continue
			}

			// For unqualified names, consider identifiers from current module
			if term.module == moduleName {
				candidates = append(candidates, term)
				continue
			}

			// Also consider identifiers from imported modules (must be global)
			if term.effectiveRange.global && isImported(term.module, moduleName, importMap, node.Name) {
				candidates = append(candidates, term)
			}
		}

		// Choose the most specific identifier
		if len(candidates) > 0 {
			mostSpecific := chooseMostSpecific(candidates, moduleName)
			node.Canonical = mostSpecific.getIdentifier().internalName
		} else {
			// No match found, keep original name
			node.Canonical = node.Name
		}

	case *parser.TyCon:
		// Find all type identifiers that match the TyCon name
		candidates := []TypeIdentifier{}

		for _, ty := range result.Types {
			// Check if name matches
			if ty.name != node.Name {
				continue
			}

			// Check if the effective range envelopes the TyCon location
			if !envelopesLocation(ty.effectiveRange, node.Loc()) {
				continue
			}

			// If TyCon has a module qualifier, only consider that module
			if node.Module != "" {
				if ty.module == node.Module {
					candidates = append(candidates, ty)
				}
				continue
			}

			// For unqualified names, consider identifiers from current module
			if ty.module == moduleName {
				candidates = append(candidates, ty)
				continue
			}

			// Also consider identifiers from imported modules (must be global)
			if ty.effectiveRange.global && isImported(ty.module, moduleName, importMap, node.Name) {
				candidates = append(candidates, ty)
			}
		}

		// Choose the most specific identifier
		if len(candidates) > 0 {
			mostSpecific := chooseMostSpecific(candidates, moduleName)
			node.Canonical = mostSpecific.getIdentifier().internalName
		} else {
			// No match found, keep original name
			node.Canonical = node.Name
		}

	case *parser.InstDecl, *parser.Assertion:
		// Both InstDecl and Assertion resolve against class identifiers
		var name, module string
		switch n := node.(type) {
		case *parser.InstDecl:
			name, module = n.Name, n.Module
		case *parser.Assertion:
			name, module = n.Name, n.Module
		}

		candidates := []ClassIdentifier{}
		for _, cls := range result.Classes {
			if cls.name != name {
				continue
			}
			if !envelopesLocation(cls.effectiveRange, ast.Loc()) {
				continue
			}
			if module != "" {
				if cls.module == module {
					candidates = append(candidates, cls)
				}
				continue
			}
			if cls.module == moduleName {
				candidates = append(candidates, cls)
				continue
			}
			if cls.effectiveRange.global && isImported(cls.module, moduleName, importMap, name) {
				candidates = append(candidates, cls)
			}
		}

		canonical := name
		if len(candidates) > 0 {
			canonical = chooseMostSpecific(candidates, moduleName).getIdentifier().internalName
		}
		switch n := node.(type) {
		case *parser.InstDecl:
			n.Canonical = canonical
		case *parser.Assertion:
			n.Canonical = canonical
		}
	}
}

// envelopesLocation checks if an effective range envelopes a location
func envelopesLocation(effectiveRange EffectiveRange, loc parser.Loc) bool {
	// Global ranges envelop everything
	if effectiveRange.global {
		return true
	}

	// Check if any of the ranges envelop the location
	for _, r := range effectiveRange.ranges {
		if r.Envelopes(loc) {
			return true
		}
	}

	return false
}

// isImported checks if a module is imported in the current module
// and the name is not in the exclude list (hiding clause)
func isImported(targetModule string, currentModule string, importMap map[string][]parser.Import, name string) bool {
	imports, ok := importMap[currentModule]
	if !ok {
		return false
	}

	for _, imp := range imports {
		// Check if this import matches the target module
		if imp.Module == targetModule || (imp.Alias != "" && imp.Alias == targetModule) {
			// If hiding is true and name is in Items, it's hidden
			if imp.Hiding {
				return !slices.Contains(imp.Items, name)
			}

			// If not hiding and Items is specified, check if name is in Items
			if len(imp.Items) > 0 {
				return slices.Contains(imp.Items, name)
			}

			// No Items specified and not hiding - everything is imported
			return true
		}
	}

	return false
}

// chooseMostSpecific selects the most specific identifier from candidates.
// Priority: local module > foreign module, smallest effective range > larger range.
func chooseMostSpecific[T HasIdentifier](candidates []T, currentModule string) T {
	if len(candidates) == 0 {
		panic("chooseMostSpecific called with empty candidates")
	}

	var local []T
	var foreign []T

	for _, c := range candidates {
		if c.getIdentifier().module == currentModule {
			local = append(local, c)
		} else {
			foreign = append(foreign, c)
		}
	}

	var toChooseFrom []T
	if len(local) > 0 {
		toChooseFrom = local
	} else {
		toChooseFrom = foreign
	}

	mostSpecific := toChooseFrom[0]
	for _, c := range toChooseFrom[1:] {
		if isMoreSpecific(c.getIdentifier().effectiveRange, mostSpecific.getIdentifier().effectiveRange) {
			mostSpecific = c
		}
	}

	return mostSpecific
}

// isMoreSpecific returns true if range1 is more specific (smaller) than range2
func isMoreSpecific(range1, range2 EffectiveRange) bool {
	// Local is more specific than global
	if !range1.global && range2.global {
		return true
	}
	if range1.global && !range2.global {
		return false
	}
	if range1.global && range2.global {
		return false // Both global, equal specificity
	}

	// Both are local - compare range sizes
	// For simplicity, compare the first range (if both have ranges)
	if len(range1.ranges) > 0 && len(range2.ranges) > 0 {
		r1 := range1.ranges[0]
		r2 := range2.ranges[0]

		// Calculate range size (number of lines)
		size1 := r1.ToLine() - r1.FromLine()
		size2 := r2.ToLine() - r2.FromLine()

		return size1 < size2
	}

	return false
}

// ResolveAll mutates all modules in place, resolving canonical names.
func ResolveAll(modules []*parser.Module, result RenameResult) {
	importMap := BuildImportMap(modules)
	for _, module := range modules {
		Resolve(module, result, importMap)
	}
}

// BuildImportMap creates a map from module names to their imports.
func BuildImportMap(modules []*parser.Module) map[string][]parser.Import {
	importMap := make(map[string][]parser.Import)
	for _, module := range modules {
		importMap[module.Name] = module.Imports
	}
	return importMap
}
