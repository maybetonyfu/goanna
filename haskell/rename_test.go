package haskell

import (
	"goanna/haskell/parser"
	"testing"
)

// Helper functions for testing
func hasGlobalTermIdents(t *testing.T, result RenameResult, module string, names []string) {
	t.Helper()
	for _, name := range names {
		found := false
		for _, term := range result.Terms {
			if term.name == name && term.module == module && term.effectiveRange.global {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected global term identifier '%s' in module '%s', not found", name, module)
		}
	}
}

func hasLocalTermIdents(t *testing.T, result RenameResult, module string, names []string) {
	t.Helper()
	for _, name := range names {
		found := false
		for _, term := range result.Terms {
			if term.name == name && term.module == module && !term.effectiveRange.global {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected local term identifier '%s' in module '%s', not found", name, module)
		}
	}
}

func hasTypeIdents(t *testing.T, result RenameResult, module string, names []string) {
	t.Helper()
	for _, name := range names {
		found := false
		for _, typ := range result.Types {
			if typ.name == name && typ.module == module {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected type identifier '%s' in module '%s', not found", name, module)
		}
	}
}

func hasClassIdents(t *testing.T, result RenameResult, module string, names []string) {
	t.Helper()
	for _, name := range names {
		found := false
		for _, cls := range result.Classes {
			if cls.name == name && cls.module == module {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected class identifier '%s' in module '%s', not found", name, module)
		}
	}
}

func TestGenUniqName(t *testing.T) {
	env := &RenameEnv{}

	tests := []struct {
		expected string
	}{
		{"V0"},
		{"V1"},
		{"V2"},
		{"V3"},
		{"V4"},
	}

	for _, tt := range tests {
		result := env.GenUniqName()
		if result != tt.expected {
			t.Errorf("GenUniqName() = %s, want %s", result, tt.expected)
		}
	}
}

func TestIntern(t *testing.T) {
	env := &RenameEnv{}

	// First call should generate V0
	result1 := env.Intern("foo", "Main")
	if result1 != "V0" {
		t.Errorf("First Intern('foo', 'Main') = %s, want V0", result1)
	}

	// Second call with same symbol and module should return same name
	result2 := env.Intern("foo", "Main")
	if result2 != "V0" {
		t.Errorf("Second Intern('foo', 'Main') = %s, want V0", result2)
	}

	// Different symbol in same module should get new name
	result3 := env.Intern("bar", "Main")
	if result3 != "V1" {
		t.Errorf("Intern('bar', 'Main') = %s, want V1", result3)
	}

	// Same symbol in different module should get new name
	result4 := env.Intern("foo", "Other")
	if result4 != "V2" {
		t.Errorf("Intern('foo', 'Other') = %s, want V2", result4)
	}

	// Verify the first one still returns the same name
	result5 := env.Intern("foo", "Main")
	if result5 != "V0" {
		t.Errorf("Third Intern('foo', 'Main') = %s, want V0", result5)
	}
}

func TestRenameTypeSig(t *testing.T) {
	env := &RenameEnv{}

	// Create a simple module with a type signature
	code := []byte("module Test where\nf :: Int -> Int")
	module := parser.Parse(code, "Test")
	result := env.Rename(*module)

	// Check we have the expected identifiers
	hasGlobalTermIdents(t, result, "Test", []string{"f"})
	hasTypeIdents(t, result, "Test", []string{})
	hasClassIdents(t, result, "Test", []string{})
}

func TestRenameMultipleNames(t *testing.T) {
	env := &RenameEnv{}

	// Create a module with multiple names in a single type signature
	code := []byte("module Test where\nf, g, h :: Int -> Int")
	module := parser.Parse(code, "Test")
	result := env.Rename(*module)

	// Check we have the expected global term identifiers
	hasGlobalTermIdents(t, result, "Test", []string{"f", "g", "h"})
	hasTypeIdents(t, result, "Test", []string{})
	hasClassIdents(t, result, "Test", []string{})
}
