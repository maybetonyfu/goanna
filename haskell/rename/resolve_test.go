package rename

import (
	"goanna/haskell/parser"
	"testing"
)

func TestResolve(t *testing.T) {
	code := "f x = x + 1"
	env := &RenameEnv{}
	moduleAST := parser.Parse([]byte(code), "Test")
	result := env.GenIdentifiers(*moduleAST)

	modules := []*parser.Module{moduleAST}
	importMap := BuildImportMap(modules)

	name := moduleAST.Name
	Resolve(moduleAST, result, importMap)

	if moduleAST.Name != name {
		t.Errorf("Expected module name '%s', got '%s'", name, moduleAST.Name)
	}
}

func TestResolveAll(t *testing.T) {
	code1 := "f x = x + 1"
	code2 := "g y = y * 2"

	env := &RenameEnv{}
	module1 := parser.Parse([]byte(code1), "Test1")
	module2 := parser.Parse([]byte(code2), "Test2")

	moduleValues := []parser.Module{*module1, *module2}
	result := env.GenIdentifiersAll(moduleValues)

	modules := []*parser.Module{module1, module2}
	ResolveAll(modules, result)

	if module1.Name != "Test1" {
		t.Errorf("Expected first module name 'Test1', got '%s'", module1.Name)
	}
	if module2.Name != "Test2" {
		t.Errorf("Expected second module name 'Test2', got '%s'", module2.Name)
	}
}

func TestResolveExpVar(t *testing.T) {
	code := "f x = x + 1"
	env := &RenameEnv{}
	moduleAST := parser.Parse([]byte(code), "Test")
	result := env.GenIdentifiers(*moduleAST)

	importMap := BuildImportMap([]*parser.Module{moduleAST})
	Resolve(moduleAST, result, importMap)

	foundExpVar := false
	visitor := parser.NewTraverser(
		func(_ int, ast parser.AST, parent parser.AST) int {
			if expVar, ok := ast.(*parser.ExpVar); ok {
				foundExpVar = true
				if expVar.Canonical == "" {
					t.Errorf("Expected canonical name to be set, got empty string for '%s'", expVar.Name)
				}
				t.Logf("Variable '%s' resolved to canonical name '%s'", expVar.Name, expVar.Canonical)
			}
			return 0
		},
		0,
	)
	visitor.Visit(moduleAST, nil)

	if !foundExpVar {
		t.Errorf("Expected to find at least one ExpVar node in the resolved module")
	}
}

func TestBuildImportMap(t *testing.T) {
	tests := []struct {
		name           string
		modules        []*parser.Module
		expectedKeys   []string
		expectedCounts map[string]int
	}{
		{
			name:         "empty modules",
			modules:      []*parser.Module{},
			expectedKeys: []string{},
		},
		{
			name: "single module with no imports",
			modules: []*parser.Module{
				{
					Name:    "Test",
					Imports: []parser.Import{},
				},
			},
			expectedKeys:   []string{"Test"},
			expectedCounts: map[string]int{"Test": 0},
		},
		{
			name: "multiple modules with different imports",
			modules: []*parser.Module{
				{
					Name: "ModuleA",
					Imports: []parser.Import{
						{Module: "Data.List"},
						{Module: "Data.Maybe"},
					},
				},
				{
					Name: "ModuleB",
					Imports: []parser.Import{
						{Module: "Control.Monad"},
					},
				},
			},
			expectedKeys:   []string{"ModuleA", "ModuleB"},
			expectedCounts: map[string]int{"ModuleA": 2, "ModuleB": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importMap := BuildImportMap(tt.modules)

			// Check the number of keys
			if len(importMap) != len(tt.expectedKeys) {
				t.Errorf("Expected %d keys, got %d", len(tt.expectedKeys), len(importMap))
			}

			// Check that all expected keys are present
			for _, key := range tt.expectedKeys {
				if _, ok := importMap[key]; !ok {
					t.Errorf("Expected key '%s' not found in import map", key)
				}

				// Check import counts if specified
				if expectedCount, ok := tt.expectedCounts[key]; ok {
					if len(importMap[key]) != expectedCount {
						t.Errorf("Module '%s': expected %d imports, got %d", key, expectedCount, len(importMap[key]))
					}
				}
			}
		})
	}
}

func TestResolveExpVarToInternalName(t *testing.T) {
	code := "x = 1\ny = x"
	env := &RenameEnv{}
	moduleAST := parser.Parse([]byte(code), "Test")
	result := env.GenIdentifiers(*moduleAST)

	importMap := BuildImportMap([]*parser.Module{moduleAST})
	Resolve(moduleAST, result, importMap)

	// Find the internal name of identifier 'x'
	var xInternalName string
	for _, term := range result.Terms {
		if term.name == "x" {
			xInternalName = term.internalName
			break
		}
	}

	if xInternalName == "" {
		t.Errorf("Expected to find identifier 'x' in rename result")
		return
	}

	// Find the ExpVar 'x' in the resolved module and verify its canonical name
	foundX := false
	visitor := parser.NewTraverser(
		func(_ int, ast parser.AST, parent parser.AST) int {
			if expVar, ok := ast.(*parser.ExpVar); ok {
				if expVar.Name == "x" {
					foundX = true
					if expVar.Canonical != xInternalName {
						t.Errorf("Expected ExpVar 'x' to resolve to '%s', got '%s'", xInternalName, expVar.Canonical)
					}
					t.Logf("Successfully resolved 'x' to canonical name '%s'", expVar.Canonical)
				}
			}
			return 0
		},
		0,
	)
	visitor.Visit(moduleAST, nil)

	if !foundX {
		t.Errorf("Expected to find ExpVar 'x' in the resolved module")
	}
}
