package haskell

import (
	"goanna/haskell/parser"
	"testing"
)

// Helper functions for testing
func hasGlobalTermIdents(t *testing.T, code string, names []string) {
	t.Helper()
	env := &RenameEnv{}
	codeByte := []byte(code)
	moduleAST := parser.Parse(codeByte, "Test")
	result := env.Rename(*moduleAST)
	for _, name := range names {
		found := false
		for _, term := range result.Terms {
			if term.name == name && term.module == "Test" && term.effectiveRange.global {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected global term identifier '%s' in module 'Test', not found", name)
		}
	}
}

func hasLocalTermIdents(t *testing.T, code string, names []string) {
	t.Helper()
	env := &RenameEnv{}
	codeByte := []byte(code)
	moduleAST := parser.Parse(codeByte, "Test")
	result := env.Rename(*moduleAST)
	for _, name := range names {
		found := false
		for _, term := range result.Terms {
			if term.name == name && term.module == "Test" && !term.effectiveRange.global {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected local term identifier '%s' in module 'Test', not found", name)
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

func TestRenameVisitNodes(t *testing.T) {
	// TypeSig tests
	t.Run("TypeSig_Single", func(t *testing.T) {
		hasGlobalTermIdents(t, "f :: Int -> Int", []string{"f"})
	})

	t.Run("TypeSig_Multiple", func(t *testing.T) {
		hasGlobalTermIdents(t, "f, g, h :: Int -> Int", []string{"f", "g", "h"})
	})

	t.Run("TypeSig_WithConstraints", func(t *testing.T) {
		hasGlobalTermIdents(t, "sort :: Ord a => [a] -> [a]", []string{"sort"})
	})

	// PatBind tests
	t.Run("PatBind_Simple", func(t *testing.T) {
		hasGlobalTermIdents(t, "f = a", []string{"f"})
	})

	t.Run("PatBind_WithParameter", func(t *testing.T) {
		hasGlobalTermIdents(t, "f a = a", []string{"f"})
		hasLocalTermIdents(t, "f a = a", []string{"a"})
	})

	t.Run("PatBind_MultipleParameters", func(t *testing.T) {
		hasGlobalTermIdents(t, "add x y = x + y", []string{"add"})
		hasLocalTermIdents(t, "add x y = x + y", []string{"x", "y"})
	})

	t.Run("PatBind_TuplePattern", func(t *testing.T) {
		hasGlobalTermIdents(t, "swap (x, y) = (y, x)", []string{"swap"})
		hasLocalTermIdents(t, "swap (x, y) = (y, x)", []string{"x", "y"})
	})

	t.Run("PatBind_ListPattern", func(t *testing.T) {
		hasGlobalTermIdents(t, "head (x:xs) = x", []string{"head"})
		hasLocalTermIdents(t, "head (x:xs) = x", []string{"x", "xs"})
	})

	// ExpLambda tests
	t.Run("ExpLambda_SingleParam", func(t *testing.T) {
		code := "f = \\x -> x + 1"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"x"})
	})

	t.Run("ExpLambda_MultipleParams", func(t *testing.T) {
		code := "f = \\x y -> x + y"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"x", "y"})
	})

	t.Run("ExpLambda_PatternParam", func(t *testing.T) {
		code := "f = \\(x, y) -> x + y"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"x", "y"})
	})

	// ExpComprehension tests
	t.Run("ExpComprehension_SingleGenerator", func(t *testing.T) {
		code := "f = [x | x <- [1, 2, 3]]"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"x"})
	})

	t.Run("ExpComprehension_MultipleGenerators", func(t *testing.T) {
		code := "f = [(x, y) | x <- [1, 2], y <- [3, 4]]"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"x", "y"})
	})

	t.Run("ExpComprehension_WithPattern", func(t *testing.T) {
		code := "f = [a + b | (a, b) <- [(1, 2), (3, 4)]]"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"a", "b"})
	})

	// ExpDo tests
	t.Run("ExpDo_SingleGenerator", func(t *testing.T) {
		code := "f = do\n  x <- getLine\n  return x"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"x"})
	})

	t.Run("ExpDo_MultipleGenerators", func(t *testing.T) {
		code := "f = do\n  x <- getLine\n  y <- getLine\n  return (x, y)"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"x", "y"})
	})

	// Alt tests (case expressions)
	t.Run("Alt_SimplePattern", func(t *testing.T) {
		code := "f x = case x of\n  y -> y + 1"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"x", "y"})
	})

	t.Run("Alt_TuplePattern", func(t *testing.T) {
		code := "f p = case p of\n  (a, b) -> a + b"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"p", "a", "b"})
	})

	t.Run("Alt_ListPattern", func(t *testing.T) {
		code := "f xs = case xs of\n  (y:ys) -> y"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"xs", "y", "ys"})
	})

	t.Run("Alt_MultipleAlternatives", func(t *testing.T) {
		code := "f xs = case xs of\n  [] -> 0\n  (x:xs) -> x"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"xs", "x"})
	})

}
