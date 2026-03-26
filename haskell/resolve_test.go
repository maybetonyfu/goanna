package haskell

import (
	"goanna/haskell/parser"
	"testing"
)

func TestResolve(t *testing.T) {
	code := "f x = x + 1"
	env := &RenameEnv{}
	moduleAST := parser.Parse([]byte(code), "Test")
	result := env.Rename(*moduleAST)

	// Resolve should return a module (currently unchanged)
	resolved := Resolve(*moduleAST, result)

	// Verify the module is returned
	if resolved.Name != moduleAST.Name {
		t.Errorf("Expected module name '%s', got '%s'", moduleAST.Name, resolved.Name)
	}
}

func TestResolveAll(t *testing.T) {
	code1 := "f x = x + 1"
	code2 := "g y = y * 2"

	env := &RenameEnv{}
	module1 := parser.Parse([]byte(code1), "Test1")
	module2 := parser.Parse([]byte(code2), "Test2")

	modules := []parser.Module{*module1, *module2}
	result := env.RenameAll(modules)

	// ResolveAll should return a list of modules (currently unchanged)
	resolved := ResolveAll(modules, result)

	// Verify we get the same number of modules back
	if len(resolved) != len(modules) {
		t.Errorf("Expected %d modules, got %d", len(modules), len(resolved))
	}

	// Verify module names are preserved
	if len(resolved) >= 1 && resolved[0].Name != "Test1" {
		t.Errorf("Expected first module name 'Test1', got '%s'", resolved[0].Name)
	}
	if len(resolved) >= 2 && resolved[1].Name != "Test2" {
		t.Errorf("Expected second module name 'Test2', got '%s'", resolved[1].Name)
	}
}
