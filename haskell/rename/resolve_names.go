package rename

import (
	"goanna/haskell/parser"
	"slices"
)

// Resolve takes a module, rename result, and import map and returns a resolved module
// Uses a traverser to visit ExpVar nodes and set their canonical names
func Resolve(module parser.Module, result RenameResult, importMap map[string][]parser.Import) parser.Module {
	// Create a copy of the module to modify
	resolvedModule := module

	// Use visitor pattern to traverse the AST and resolve names
	visitor := parser.NewTraverser(
		func(_ int, ast parser.AST, parent parser.AST) int {
			resolveNode(ast, module.Name, result, importMap)
			return 0
		},
		0,
	)
	visitor.Visit(&resolvedModule, nil)

	return resolvedModule
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
			node.Canonical = mostSpecific.internalName
		} else {
			// No match found, keep original name
			node.Canonical = node.Name
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

// chooseMostSpecific selects the most specific identifier from candidates
// Priority: local module > foreign module, smallest effective range > larger range
func chooseMostSpecific(candidates []TermIdentifier, currentModule string) TermIdentifier {
	if len(candidates) == 0 {
		panic("chooseMostSpecific called with empty candidates")
	}

	// Separate local and foreign identifiers
	var local []TermIdentifier
	var foreign []TermIdentifier

	for _, c := range candidates {
		if c.module == currentModule {
			local = append(local, c)
		} else {
			foreign = append(foreign, c)
		}
	}

	// Prefer local module over foreign
	var toChooseFrom []TermIdentifier
	if len(local) > 0 {
		toChooseFrom = local
	} else {
		toChooseFrom = foreign
	}

	// Among the chosen set, find the one with smallest effective range
	mostSpecific := toChooseFrom[0]
	for _, c := range toChooseFrom[1:] {
		if isMoreSpecific(c.effectiveRange, mostSpecific.effectiveRange) {
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

// ResolveAll takes a list of modules and rename result and returns resolved modules
// Builds an import map and passes it to each Resolve call
func ResolveAll(modules []parser.Module, result RenameResult) []parser.Module {
	// Build import map once for all modules
	importMap := BuildImportMap(modules)

	resolved := make([]parser.Module, len(modules))
	for i, module := range modules {
		resolved[i] = Resolve(module, result, importMap)
	}
	return resolved
}

// BuildImportMap creates a map from module names to their imports
// Key: module name (string)
// Value: list of Import statements from that module
func BuildImportMap(modules []parser.Module) map[string][]parser.Import {
	importMap := make(map[string][]parser.Import)

	for _, module := range modules {
		importMap[module.Name] = module.Imports
	}

	return importMap
}
