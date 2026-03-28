package rename

import (
	"fmt"
	"goanna/haskell/parser"
)

// EffectiveRange represents the scope information for an identifier
type EffectiveRange struct {
	ranges []parser.Loc
	global bool
}

// Equal checks if two EffectiveRanges are equal
// All global ranges are considered equal to each other
func (er EffectiveRange) Equal(other EffectiveRange) bool {
	// If both are global, they are equal
	if er.global && other.global {
		return true
	}

	// If one is global and the other is not, they are not equal
	if er.global != other.global {
		return false
	}

	// Both are local - compare the ranges
	if len(er.ranges) != len(other.ranges) {
		return false
	}

	// Check if all ranges are equal
	for i := range er.ranges {
		if !er.ranges[i].Equal(other.ranges[i]) {
			return false
		}
	}

	return true
}

// Identifier represents a named entity in Haskell code with its scope information
type Identifier struct {
	name           string
	module         string
	effectiveRange EffectiveRange
	internalName   string
	isParameter    bool
	declaredAt     []int
}

// TermIdentifier represents a value-level identifier (functions, variables)
type TermIdentifier struct {
	Identifier
}

func (t TermIdentifier) getIdentifier() Identifier { return t.Identifier }

// TypeIdentifier represents a type-level identifier (type constructors, type variables)
type TypeIdentifier struct {
	Identifier
}

func (t TypeIdentifier) getIdentifier() Identifier { return t.Identifier }

// ClassIdentifier represents a type class identifier
type ClassIdentifier struct {
	Identifier
}

func (c ClassIdentifier) getIdentifier() Identifier { return c.Identifier }

// HasIdentifier is a type constraint for types that embed Identifier
type HasIdentifier interface {
	TermIdentifier | TypeIdentifier | ClassIdentifier
	getIdentifier() Identifier
}

// RenameResult holds the results of identifier extraction from a module
type RenameResult struct {
	Terms   []TermIdentifier
	Types   []TypeIdentifier
	Classes []ClassIdentifier
}

// internEntry holds a symbol+effectiveRange pair mapped to an internal name
type internEntry struct {
	symbol string
	er     EffectiveRange
	name   string
}

// internTable is a per-namespace intern table with its own counter and prefix
type internTable struct {
	counter int
	prefix  string
	entries map[string][]internEntry // module -> list of interned entries
}

func newInternTable(prefix string) internTable {
	return internTable{
		prefix:  prefix,
		entries: make(map[string][]internEntry),
	}
}

func (t *internTable) intern(symbolName string, moduleName string, er EffectiveRange) string {
	for _, entry := range t.entries[moduleName] {
		if entry.symbol == symbolName && entry.er.Equal(er) {
			return entry.name
		}
	}
	name := fmt.Sprintf("%s%d", t.prefix, t.counter)
	t.counter++
	t.entries[moduleName] = append(t.entries[moduleName], internEntry{
		symbol: symbolName,
		er:     er,
		name:   name,
	})
	return name
}

// RenameEnv holds the environment for identifier renaming and analysis
type RenameEnv struct {
	terms   internTable
	types   internTable
	classes internTable
}

// InternTerm interns a term-level name, producing identifiers like V0, V1, V2...
func (env *RenameEnv) InternTerm(symbolName string, moduleName string, er EffectiveRange) string {
	if env.terms.entries == nil {
		env.terms = newInternTable("V")
	}
	return env.terms.intern(symbolName, moduleName, er)
}

// InternType interns a type-level name, producing identifiers like t0, t1, t2...
func (env *RenameEnv) InternType(symbolName string, moduleName string, er EffectiveRange) string {
	if env.types.entries == nil {
		env.types = newInternTable("t")
	}
	return env.types.intern(symbolName, moduleName, er)
}

// InternClass interns a class name, producing identifiers like c0, c1, c2...
func (env *RenameEnv) InternClass(symbolName string, moduleName string, er EffectiveRange) string {
	if env.classes.entries == nil {
		env.classes = newInternTable("c")
	}
	return env.classes.intern(symbolName, moduleName, er)
}

// namesFromPat extracts all names and their node IDs from a pattern
func namesFromPat(pat parser.Pat) []struct {
	name string
	id   int
} {
	var names []struct {
		name string
		id   int
	}

	switch p := pat.(type) {
	case *parser.PVar:
		names = append(names, struct {
			name string
			id   int
		}{p.Name, p.Id()})
	case *parser.PApp:
		names = append(names, struct {
			name string
			id   int
		}{p.Constructor.Name, p.Constructor.Id()})
		for _, subpat := range p.Pats {
			names = append(names, namesFromPat(subpat)...)
		}
	case *parser.PList:
		for _, subpat := range p.Pats {
			names = append(names, namesFromPat(subpat)...)
		}
	case *parser.PTuple:
		for _, subpat := range p.Pats {
			names = append(names, namesFromPat(subpat)...)
		}
	case *parser.PInfix:
		names = append(names, namesFromPat(p.Pat1)...)
		names = append(names, namesFromPat(p.Pat2)...)
	}

	return names
}

// GenIdentifiers analyzes an AST and returns identifiers of all three kinds with their scope information
func (env *RenameEnv) GenIdentifiers(ast parser.Module) RenameResult {
	result := &RenameResult{
		Terms:   []TermIdentifier{},
		Types:   []TypeIdentifier{},
		Classes: []ClassIdentifier{},
	}

	moduleName := ast.Name

	// Use visitor pattern to traverse the AST
	visitor := parser.NewTraverser(
		func(_ int, ast parser.AST, parent parser.AST) int {
			env.visitNode(ast, moduleName, result, parent)
			return 0
		},
		0,
	)
	visitor.Visit(&ast, nil)

	// Merge duplicate TermIdentifiers
	result.Terms = mergeTermIdentifiers(result.Terms)

	return *result
}

// visitNode processes a single AST node and extracts identifiers
func (env *RenameEnv) visitNode(ast parser.AST, moduleName string, result *RenameResult, parent parser.AST) {
	switch node := ast.(type) {
	case *parser.TypeSig:
		// Process type signature - intern all names as term identifiers
		var effectiveRange EffectiveRange

		// Check if parent is a module (global scope)
		if _, isModule := parent.(*parser.Module); isModule || parent == nil {
			effectiveRange = EffectiveRange{
				ranges: []parser.Loc{},
				global: true,
			}
		} else {
			// Parent is not a module - local scope
			effectiveRange = EffectiveRange{
				ranges: []parser.Loc{parent.Loc()},
				global: false,
			}

		}

		for _, name := range node.Names {
			internalName := env.InternTerm(name, moduleName, effectiveRange)
			termId := TermIdentifier{
				Identifier: Identifier{
					name:           name,
					module:         moduleName,
					effectiveRange: effectiveRange,
					internalName:   internalName,
					isParameter:    false,
					declaredAt:     []int{node.Id()},
				},
			}
			result.Terms = append(result.Terms, termId)
		}

	case *parser.PatBind:
		// Process pattern bindings - extract names from patterns
		names := namesFromPat(node.Pat)
		for i, nameInfo := range names {
			var effectiveRange EffectiveRange
			var isParam bool

			if i == 0 {
				// First name determines scope based on context
				// Check if this is a local declaration (where clause or let expression)
				isLocalDecl := false
				if parent != nil {
					switch parent.(type) {
					case *parser.UnguardedRhs, *parser.GuardedRhs, *parser.Alt, *parser.ExpLet:
						isLocalDecl = true
					}
				}

				if isLocalDecl {
					// Local declarations (where clauses and let expressions) are local to the parent scope
					effectiveRange = EffectiveRange{
						ranges: []parser.Loc{parent.Loc()},
						global: false,
					}
					isParam = false
				} else if _, isModule := parent.(*parser.Module); isModule || parent == nil {
					// Module-level declarations are global
					effectiveRange = EffectiveRange{
						ranges: []parser.Loc{},
						global: true,
					}
					isParam = false
				} else {
					// Other contexts are local
					effectiveRange = EffectiveRange{
						ranges: []parser.Loc{parent.Loc()},
						global: false,
					}
					isParam = false
				}
			} else {
				// Other names get RHS scope and are parameters
				effectiveRange = EffectiveRange{
					ranges: []parser.Loc{node.Rhs.Loc()},
					global: false,
				}
				isParam = true
			}

			internalName := env.InternTerm(nameInfo.name, moduleName, effectiveRange)
			termId := TermIdentifier{
				Identifier: Identifier{
					name:           nameInfo.name,
					module:         moduleName,
					effectiveRange: effectiveRange,
					internalName:   internalName,
					isParameter:    isParam,
					declaredAt:     []int{nameInfo.id},
				},
			}
			result.Terms = append(result.Terms, termId)
		}

	case *parser.DataCon:
		// Data constructors are always global terms
		internalName := env.InternTerm(node.Name, moduleName, EffectiveRange{global: true})
		termId := TermIdentifier{
			Identifier: Identifier{
				name:   node.Name,
				module: moduleName,
				effectiveRange: EffectiveRange{
					ranges: []parser.Loc{},
					global: true,
				},
				internalName: internalName,
				isParameter:  false,
				declaredAt:   []int{node.Id()},
			},
		}
		result.Terms = append(result.Terms, termId)

	case *parser.DataDecl:
		// Data type declarations - extract type name from DeclHead
		dHead := node.DHead
		internalName := env.InternType(dHead.Name, moduleName, EffectiveRange{global: true})
		typeId := TypeIdentifier{
			Identifier: Identifier{
				name:   dHead.Name,
				module: moduleName,
				effectiveRange: EffectiveRange{
					ranges: []parser.Loc{},
					global: true,
				},
				internalName: internalName,
				isParameter:  false,
				declaredAt:   []int{dHead.Id()},
			},
		}
		result.Types = append(result.Types, typeId)

	case *parser.TypeDecl:
		// Type declarations - extract type name from DeclHead
		dHead := node.DHead
		internalName := env.InternType(dHead.Name, moduleName, EffectiveRange{global: true})
		typeId := TypeIdentifier{
			Identifier: Identifier{
				name:   dHead.Name,
				module: moduleName,
				effectiveRange: EffectiveRange{
					ranges: []parser.Loc{},
					global: true,
				},
				internalName: internalName,
				isParameter:  false,
				declaredAt:   []int{dHead.Id()},
			},
		}
		result.Types = append(result.Types, typeId)

	case *parser.ClassDecl:
		// Class declarations - extract class name from DeclHead
		dHead := node.DHead
		internalName := env.InternClass(dHead.Name, moduleName, EffectiveRange{global: true})
		classId := ClassIdentifier{
			Identifier: Identifier{
				name:   dHead.Name,
				module: moduleName,
				effectiveRange: EffectiveRange{
					ranges: []parser.Loc{},
					global: true,
				},
				internalName: internalName,
				isParameter:  false,
				declaredAt:   []int{dHead.Id()},
			},
		}
		result.Classes = append(result.Classes, classId)

	case *parser.ExpComprehension:
		// List comprehension - extract names from generators
		for _, gen := range node.Generators {
			names := namesFromPat(gen.Pat)
			effectiveRange := EffectiveRange{
				ranges: []parser.Loc{node.Loc()},
				global: false,
			}

			for _, nameInfo := range names {
				internalName := env.InternTerm(nameInfo.name, moduleName, effectiveRange)
				termId := TermIdentifier{
					Identifier: Identifier{
						name:           nameInfo.name,
						module:         moduleName,
						effectiveRange: effectiveRange,
						internalName:   internalName,
						isParameter:    false,
						declaredAt:     []int{nameInfo.id},
					},
				}
				result.Terms = append(result.Terms, termId)
			}
		}

	case *parser.ExpDo:
		// Do block - extract names from generators and let bindings
		effectiveRange := EffectiveRange{
			ranges: []parser.Loc{node.Loc()},
			global: false,
		}

		for _, stmt := range node.Stmts {
			if gen, ok := stmt.(*parser.Generator); ok {
				names := namesFromPat(gen.Pat)
				for _, nameInfo := range names {
					internalName := env.InternTerm(nameInfo.name, moduleName, effectiveRange)
					termId := TermIdentifier{
						Identifier: Identifier{
							name:           nameInfo.name,
							module:         moduleName,
							effectiveRange: effectiveRange,
							internalName:   internalName,
							isParameter:    false,
							declaredAt:     []int{nameInfo.id},
						},
					}
					result.Terms = append(result.Terms, termId)
				}
			}
		}

	case *parser.ExpLambda:
		// Lambda expression - extract names from patterns
		effectiveRange := EffectiveRange{
			ranges: []parser.Loc{node.Exp.Loc()},
			global: false,
		}

		for _, pat := range node.Pats {
			names := namesFromPat(pat)
			for _, nameInfo := range names {
				internalName := env.InternTerm(nameInfo.name, moduleName, effectiveRange)
				termId := TermIdentifier{
					Identifier: Identifier{
						name:           nameInfo.name,
						module:         moduleName,
						effectiveRange: effectiveRange,
						internalName:   internalName,
						isParameter:    true,
						declaredAt:     []int{nameInfo.id},
					},
				}
				result.Terms = append(result.Terms, termId)
			}
		}

	case *parser.Alt:
		// Case alternative - extract names from pattern
		names := namesFromPat(node.Pat)
		effectiveRange := EffectiveRange{
			ranges: []parser.Loc{node.Loc()},
			global: false,
		}

		for _, nameInfo := range names {
			internalName := env.InternTerm(nameInfo.name, moduleName, effectiveRange)
			termId := TermIdentifier{
				Identifier: Identifier{
					name:           nameInfo.name,
					module:         moduleName,
					effectiveRange: effectiveRange,
					internalName:   internalName,
					isParameter:    true,
					declaredAt:     []int{nameInfo.id},
				},
			}
			result.Terms = append(result.Terms, termId)
		}
	}
}

// RenameDecl traverses all nodes in module and sets Canonical on any node
// whose ID appears in the declaredAt list of an identifier in result.
func RenameDecl(module *parser.Module, result RenameResult) {
	// Build a map from node ID to internalName
	declaredAtMap := make(map[int]string)

	for _, term := range result.Terms {
		id := term.getIdentifier()
		for _, nodeID := range id.declaredAt {
			declaredAtMap[nodeID] = id.internalName
		}
	}
	for _, ty := range result.Types {
		id := ty.getIdentifier()
		for _, nodeID := range id.declaredAt {
			declaredAtMap[nodeID] = id.internalName
		}
	}
	for _, cls := range result.Classes {
		id := cls.getIdentifier()
		for _, nodeID := range id.declaredAt {
			declaredAtMap[nodeID] = id.internalName
		}
	}

	// Traverse the module and set Canonical on matching nodes
	traverser := parser.NewTraverser(
		func(_ int, ast parser.AST, _ parser.AST) int {
			if internalName, ok := declaredAtMap[ast.Id()]; ok {
				if named, ok := ast.(parser.Name); ok {
					named.SetCanonical(internalName)
				}
			}
			return 0
		},
		0,
	)
	traverser.Visit(module, nil)
}

// RenameTypeDecl traverses all nodes in module and for every TypeSig sets
// Canonicals to the internal names of the TermIdentifiers whose declaredAt
// includes the TypeSig's node ID, ordered to match TypeSig.Names.
func RenameTypeDecl(module *parser.Module, result RenameResult) {
	// Build a map from TypeSig node ID -> (name -> internalName)
	// A TermIdentifier's declaredAt may include a TypeSig node ID.
	type nameMap = map[string]string
	typeSigMap := make(map[int]nameMap)

	for _, term := range result.Terms {
		id := term.getIdentifier()
		for _, nodeID := range id.declaredAt {
			if typeSigMap[nodeID] == nil {
				typeSigMap[nodeID] = make(nameMap)
			}
			typeSigMap[nodeID][id.name] = id.internalName
		}
	}

	traverser := parser.NewTraverser(
		func(_ int, ast parser.AST, _ parser.AST) int {
			typeSig, ok := ast.(*parser.TypeSig)
			if !ok {
				return 0
			}
			names, ok := typeSigMap[typeSig.Id()]
			if !ok {
				return 0
			}
			canonicals := make([]string, len(typeSig.Names))
			for i, name := range typeSig.Names {
				if internalName, found := names[name]; found {
					canonicals[i] = internalName
				} else {
					canonicals[i] = name
				}
			}
			typeSig.Canonicals = canonicals
			return 0
		},
		0,
	)
	traverser.Visit(module, nil)
}

func mergeTermIdentifiers(terms []TermIdentifier) []TermIdentifier {
	if len(terms) == 0 {
		return terms
	}

	// Map key: module + name + effectiveRange representation
	// We'll use a custom key structure
	type key struct {
		module string
		name   string
		// We can't use EffectiveRange directly as a map key, so we'll search linearly
	}

	merged := make(map[int]*TermIdentifier) // index -> merged identifier
	keyToIndex := make(map[key][]int)       // key -> list of potential match indices

	for _, term := range terms {
		k := key{module: term.module, name: term.name}

		// Look for existing term with same module, name, and effectiveRange
		found := false
		if indices, exists := keyToIndex[k]; exists {
			for _, idx := range indices {
				if merged[idx].effectiveRange.Equal(term.effectiveRange) {
					// Found a match - merge declaredAt
					merged[idx].declaredAt = append(merged[idx].declaredAt, term.declaredAt...)
					found = true
					break
				}
			}
		}

		if !found {
			// Create new entry
			idx := len(merged)
			termCopy := term
			merged[idx] = &termCopy
			keyToIndex[k] = append(keyToIndex[k], idx)
		}
	}

	// Convert map back to slice
	result := make([]TermIdentifier, 0, len(merged))
	for i := 0; i < len(merged); i++ {
		result = append(result, *merged[i])
	}

	return result
}

// GenIdentifiersAll analyzes multiple modules and returns all identifiers of all three kinds from all modules
func (env *RenameEnv) GenIdentifiersAll(modules []parser.Module) RenameResult {
	allResult := RenameResult{
		Terms:   []TermIdentifier{},
		Types:   []TypeIdentifier{},
		Classes: []ClassIdentifier{},
	}

	for _, module := range modules {
		result := env.GenIdentifiers(module)
		allResult.Terms = append(allResult.Terms, result.Terms...)
		allResult.Types = append(allResult.Types, result.Types...)
		allResult.Classes = append(allResult.Classes, result.Classes...)
	}

	// Merge duplicate TermIdentifiers
	allResult.Terms = mergeTermIdentifiers(allResult.Terms)

	return allResult
}
