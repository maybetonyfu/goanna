package haskell

import (
	"fmt"
	"goanna/haskell/parser"
)

// EffectiveRange represents the scope information for an identifier
type EffectiveRange struct {
	ranges []parser.Loc
	global bool
}

// Identifier represents a named entity in Haskell code with its scope information
type Identifier struct {
	name           string
	module         string
	effectiveRange EffectiveRange
	internalName   string
	isParameter    bool
	declaredAt     []parser.Loc
}

// TermIdentifier represents a value-level identifier (functions, variables)
type TermIdentifier struct {
	Identifier
}

// TypeIdentifier represents a type-level identifier (type constructors, type variables)
type TypeIdentifier struct {
	Identifier
}

// ClassIdentifier represents a type class identifier
type ClassIdentifier struct {
	Identifier
}

// RenameResult holds the results of identifier extraction from a module
type RenameResult struct {
	Terms   []TermIdentifier
	Types   []TypeIdentifier
	Classes []ClassIdentifier
}

// RenameEnv holds the environment for identifier renaming and analysis
type RenameEnv struct {
	counter       int
	internedNames map[string]map[string]string // module -> (symbol -> internalName)
}

// GenUniqName generates a unique internal name by combining "V" with the current counter value,
// then increments the counter for the next call
func (env *RenameEnv) GenUniqName() string {
	name := fmt.Sprintf("V%d", env.counter)
	env.counter++
	return name
}

// Intern returns the internal name for a symbol in a given module.
// If the symbol has already been interned, it returns the existing internal name.
// Otherwise, it generates a new unique name, stores it, and returns it.
func (env *RenameEnv) Intern(symbolName string, moduleName string) string {
	// Initialize the map if needed
	if env.internedNames == nil {
		env.internedNames = make(map[string]map[string]string)
	}

	// Initialize the module map if needed
	if env.internedNames[moduleName] == nil {
		env.internedNames[moduleName] = make(map[string]string)
	}

	// Check if already interned
	if internalName, exists := env.internedNames[moduleName][symbolName]; exists {
		return internalName
	}

	// Generate new unique name
	internalName := env.GenUniqName()
	env.internedNames[moduleName][symbolName] = internalName
	return internalName
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

// Rename analyzes an AST and returns identifiers of all three kinds with their scope information
func (env *RenameEnv) Rename(ast parser.Module) RenameResult {
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
				ranges: []parser.Loc{node.Loc()},
				global: true,
			}
		} else {
			// Parent is not a module - local scope
			if parent != nil {
				effectiveRange = EffectiveRange{
					ranges: []parser.Loc{parent.Loc()},
					global: false,
				}
			}
		}

		for _, name := range node.Names {
			internalName := env.Intern(name, moduleName)
			termId := TermIdentifier{
				Identifier: Identifier{
					name:           name,
					module:         moduleName,
					effectiveRange: effectiveRange,
					internalName:   internalName,
					isParameter:    false,
					declaredAt:     []parser.Loc{node.Loc()},
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
				// First name gets parent scope or global scope
				if _, isModule := parent.(*parser.Module); isModule || parent == nil {
					effectiveRange = EffectiveRange{
						ranges: []parser.Loc{node.Loc()},
						global: true,
					}
				} else {
					if parent != nil {
						effectiveRange = EffectiveRange{
							ranges: []parser.Loc{parent.Loc()},
							global: false,
						}
					}
				}
				isParam = false
			} else {
				// Other names get RHS scope and are parameters
				effectiveRange = EffectiveRange{
					ranges: []parser.Loc{node.Rhs.Loc()},
					global: false,
				}
				isParam = true
			}

			internalName := env.Intern(nameInfo.name, moduleName)
			termId := TermIdentifier{
				Identifier: Identifier{
					name:           nameInfo.name,
					module:         moduleName,
					effectiveRange: effectiveRange,
					internalName:   internalName,
					isParameter:    isParam,
					declaredAt:     []parser.Loc{node.Loc()},
				},
			}
			result.Terms = append(result.Terms, termId)
		}

	case *parser.DataCon:
		// Data constructors are always global terms
		internalName := env.Intern(node.Name, moduleName)
		termId := TermIdentifier{
			Identifier: Identifier{
				name:   node.Name,
				module: moduleName,
				effectiveRange: EffectiveRange{
					ranges: []parser.Loc{node.Loc()},
					global: true,
				},
				internalName: internalName,
				isParameter:  false,
				declaredAt:   []parser.Loc{node.Loc()},
			},
		}
		result.Terms = append(result.Terms, termId)

	case *parser.DataDecl:
		// Data type declarations - extract type name from DeclHead
		dHead := node.DHead
		internalName := env.Intern(dHead.Name, moduleName)
		typeId := TypeIdentifier{
			Identifier: Identifier{
				name:   dHead.Name,
				module: moduleName,
				effectiveRange: EffectiveRange{
					ranges: []parser.Loc{dHead.Loc()},
					global: true,
				},
				internalName: internalName,
				isParameter:  false,
				declaredAt:   []parser.Loc{dHead.Loc()},
			},
		}
		result.Types = append(result.Types, typeId)

	case *parser.ClassDecl:
		// Class declarations - extract class name from DeclHead
		dHead := node.DHead
		internalName := env.Intern(dHead.Name, moduleName)
		classId := ClassIdentifier{
			Identifier: Identifier{
				name:   dHead.Name,
				module: moduleName,
				effectiveRange: EffectiveRange{
					ranges: []parser.Loc{dHead.Loc()},
					global: true,
				},
				internalName: internalName,
				isParameter:  false,
				declaredAt:   []parser.Loc{dHead.Loc()},
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
				internalName := env.Intern(nameInfo.name, moduleName)
				termId := TermIdentifier{
					Identifier: Identifier{
						name:           nameInfo.name,
						module:         moduleName,
						effectiveRange: effectiveRange,
						internalName:   internalName,
						isParameter:    false,
						declaredAt:     []parser.Loc{node.Loc()},
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
					internalName := env.Intern(nameInfo.name, moduleName)
					termId := TermIdentifier{
						Identifier: Identifier{
							name:           nameInfo.name,
							module:         moduleName,
							effectiveRange: effectiveRange,
							internalName:   internalName,
							isParameter:    false,
							declaredAt:     []parser.Loc{node.Loc()},
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
				internalName := env.Intern(nameInfo.name, moduleName)
				termId := TermIdentifier{
					Identifier: Identifier{
						name:           nameInfo.name,
						module:         moduleName,
						effectiveRange: effectiveRange,
						internalName:   internalName,
						isParameter:    true,
						declaredAt:     []parser.Loc{node.Loc()},
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
			internalName := env.Intern(nameInfo.name, moduleName)
			termId := TermIdentifier{
				Identifier: Identifier{
					name:           nameInfo.name,
					module:         moduleName,
					effectiveRange: effectiveRange,
					internalName:   internalName,
					isParameter:    true,
					declaredAt:     []parser.Loc{node.Loc()},
				},
			}
			result.Terms = append(result.Terms, termId)
		}
	}
}

// RenameAll analyzes multiple modules and returns all identifiers of all three kinds from all modules
func (env *RenameEnv) RenameAll(modules []parser.Module) RenameResult {
	allResult := RenameResult{
		Terms:   []TermIdentifier{},
		Types:   []TypeIdentifier{},
		Classes: []ClassIdentifier{},
	}

	for _, module := range modules {
		result := env.Rename(module)
		allResult.Terms = append(allResult.Terms, result.Terms...)
		allResult.Types = append(allResult.Types, result.Types...)
		allResult.Classes = append(allResult.Classes, result.Classes...)
	}

	return allResult
}
