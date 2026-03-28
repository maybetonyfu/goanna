package rename

import (
	"goanna/haskell/parser"
	"testing"
)

// Helper functions for testing
func hasGlobalTermIdents(t *testing.T, code string, names []string) []TermIdentifier {
	t.Helper()
	env := &RenameEnv{}
	codeByte := []byte(code)
	moduleAST := parser.Parse(codeByte, "Test")
	result := env.GenIdentifiers(*moduleAST)
	
	var matched []TermIdentifier
	for _, name := range names {
		found := false
		for _, term := range result.Terms {
			if term.name == name && term.module == "Test" && term.effectiveRange.global {
				matched = append(matched, term)
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected global term identifier '%s' in module 'Test', not found", name)
		}
	}
	return matched
}

func hasLocalTermIdents(t *testing.T, code string, names []string) []TermIdentifier {
	t.Helper()
	env := &RenameEnv{}
	codeByte := []byte(code)
	moduleAST := parser.Parse(codeByte, "Test")
	result := env.GenIdentifiers(*moduleAST)
	
	var matched []TermIdentifier
	for _, name := range names {
		found := false
		for _, term := range result.Terms {
			if term.name == name && term.module == "Test" && !term.effectiveRange.global {
				matched = append(matched, term)
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected local term identifier '%s' in module 'Test', not found", name)
		}
	}
	return matched
}

func hasTypeIdents(t *testing.T, result RenameResult, module string, names []string) []TypeIdentifier {
	t.Helper()
	var matched []TypeIdentifier
	for _, name := range names {
		found := false
		for _, typ := range result.Types {
			if typ.name == name && typ.module == module {
				matched = append(matched, typ)
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected type identifier '%s' in module '%s', not found", name, module)
		}
	}
	return matched
}

func hasClassIdents(t *testing.T, result RenameResult, module string, names []string) []ClassIdentifier {
	t.Helper()
	var matched []ClassIdentifier
	for _, name := range names {
		found := false
		for _, cls := range result.Classes {
			if cls.name == name && cls.module == module {
				matched = append(matched, cls)
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected class identifier '%s' in module '%s', not found", name, module)
		}
	}
	return matched
}

func TestInternTerm(t *testing.T) {
	global := EffectiveRange{global: true}
	env := &RenameEnv{}

	r1 := env.InternTerm("foo", "Main", global)
	if r1 != "V0" {
		t.Errorf("InternTerm('foo', 'Main', global) = %s, want V0", r1)
	}
	r2 := env.InternTerm("foo", "Main", global)
	if r2 != "V0" {
		t.Errorf("InternTerm('foo', 'Main', global) second call = %s, want V0", r2)
	}
	r3 := env.InternTerm("bar", "Main", global)
	if r3 != "V1" {
		t.Errorf("InternTerm('bar', 'Main', global) = %s, want V1", r3)
	}
	r4 := env.InternTerm("foo", "Other", global)
	if r4 != "V2" {
		t.Errorf("InternTerm('foo', 'Other', global) = %s, want V2", r4)
	}
	r5 := env.InternTerm("foo", "Main", global)
	if r5 != "V0" {
		t.Errorf("InternTerm('foo', 'Main', global) third call = %s, want V0", r5)
	}

	local := EffectiveRange{global: false, ranges: []parser.Loc{parser.NewLoc(1, 1, 5, 10)}}
	r6 := env.InternTerm("foo", "Main", local)
	if r6 == "V0" {
		t.Errorf("InternTerm('foo', 'Main', local) should differ from global, got V0")
	}
	r7 := env.InternTerm("foo", "Main", local)
	if r7 != r6 {
		t.Errorf("InternTerm('foo', 'Main', local) second call = %s, want %s", r7, r6)
	}
}

func TestInternTypeSeparateCounters(t *testing.T) {
	env := &RenameEnv{}
	global := EffectiveRange{global: true}

	term := env.InternTerm("Foo", "Main", global)
	ty := env.InternType("Foo", "Main", global)
	cls := env.InternClass("Foo", "Main", global)

	if term != "V0" {
		t.Errorf("InternTerm = %s, want V0", term)
	}
	if ty != "t0" {
		t.Errorf("InternType = %s, want t0", ty)
	}
	if cls != "c0" {
		t.Errorf("InternClass = %s, want c0", cls)
	}
}

func TestInternWhereClauseShadowing(t *testing.T) {
	// 'x = x where x = 1' should produce a global x and a distinct local x
	code := "x = x where x = 1"
	env := &RenameEnv{}
	moduleAST := parser.Parse([]byte(code), "Test")
	result := env.GenIdentifiers(*moduleAST)

	var globalX, localX *TermIdentifier
	for i := range result.Terms {
		term := &result.Terms[i]
		if term.name == "x" && term.module == "Test" {
			if term.effectiveRange.global {
				globalX = term
			} else {
				localX = term
			}
		}
	}

	if globalX == nil {
		t.Fatal("Expected a global term identifier 'x', not found")
	}
	if localX == nil {
		t.Fatal("Expected a local term identifier 'x' (from where clause), not found")
	}
	if globalX.internalName == localX.internalName {
		t.Errorf("Global x and local x should have different internal names, both got %s", globalX.internalName)
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

	// Where clause tests
	t.Run("WhereClause_Simple", func(t *testing.T) {
		code := "f = y where y = 1"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"y"})
	})

	t.Run("WhereClause_Multiple", func(t *testing.T) {
		code := "f x = x + y where y = 10"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"x", "y"})
	})

	t.Run("WhereClause_MultipleBindings", func(t *testing.T) {
		code := "f x = a + b where\n  a = x * 2\n  b = x * 3"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"x", "a", "b"})
	})

	t.Run("WhereClause_WithGuards", func(t *testing.T) {
		code := "f x\n  | x > 0 = y\n  | otherwise = z\n  where\n    y = 1\n    z = 2"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"x", "y", "z"})
	})

	// Let expression tests
	t.Run("ExpLet_Simple", func(t *testing.T) {
		code := "f = let x = 1 in x + 2"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"x"})
	})

	t.Run("ExpLet_MultipleBindings", func(t *testing.T) {
		code := "f = let x = 1; y = 2 in x + y"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"x", "y"})
	})

	t.Run("ExpLet_NestedLet", func(t *testing.T) {
		code := "f = let x = 1 in let y = x + 1 in y"
		hasGlobalTermIdents(t, code, []string{"f"})
		hasLocalTermIdents(t, code, []string{"x", "y"})
	})

	// Multiple pattern bindings for same function
	t.Run("MultiplePatternBindings", func(t *testing.T) {
		code := "f 1 = 1\nf 2 = 2"
		idents := hasGlobalTermIdents(t, code, []string{"f"})
		if len(idents) != 1 {
			t.Errorf("Expected 1 identifier for 'f', got %d", len(idents))
		}
		if len(idents) > 0 && len(idents[0].declaredAt) != 2 {
			t.Errorf("Expected 2 declared locations for 'f', got %d", len(idents[0].declaredAt))
		}
	})

	// Note: Data declarations are not currently tested here as the parser
	// does not fully support parsing data declarations in the test environment.
	// However, the rename logic for DataDecl and DataCon is implemented and
	// would work correctly once the parser supports them.
	// The implementation generates:
	// - A TypeIdentifier for the data type name (e.g., "Bool", "Maybe", "Point")
	// - A TermIdentifier for each data constructor (e.g., "True", "False", "Nothing", "Just")

	// Type synonym tests
	t.Run("TypeDecl_Simple", func(t *testing.T) {
		code := "type String = [Char]"
		env := &RenameEnv{}
		moduleAST := parser.Parse([]byte(code), "Test")
		result := env.GenIdentifiers(*moduleAST)

		// Check type identifier
		typeIdents := hasTypeIdents(t, result, "Test", []string{"String"})
		if len(typeIdents) != 1 {
			t.Errorf("Expected 1 type identifier for 'String', got %d", len(typeIdents))
		}
		if len(typeIdents) > 0 && !typeIdents[0].effectiveRange.global {
			t.Errorf("Expected 'String' to be global")
		}
	})

	t.Run("TypeDecl_WithParameters", func(t *testing.T) {
		code := "type Pair a b = (a, b)"
		env := &RenameEnv{}
		moduleAST := parser.Parse([]byte(code), "Test")
		result := env.GenIdentifiers(*moduleAST)

		// Check type identifier
		typeIdents := hasTypeIdents(t, result, "Test", []string{"Pair"})
		if len(typeIdents) != 1 {
			t.Errorf("Expected 1 type identifier for 'Pair', got %d", len(typeIdents))
		}
		if len(typeIdents) > 0 && !typeIdents[0].effectiveRange.global {
			t.Errorf("Expected 'Pair' to be global")
		}
	})

	// Class declaration tests
	t.Run("ClassDecl_Simple", func(t *testing.T) {
		code := "class Eq a where\n  eq :: a -> a -> Bool"
		env := &RenameEnv{}
		moduleAST := parser.Parse([]byte(code), "Test")
		result := env.GenIdentifiers(*moduleAST)

		// Check class identifier
		classIdents := hasClassIdents(t, result, "Test", []string{"Eq"})
		if len(classIdents) != 1 {
			t.Errorf("Expected 1 class identifier for 'Eq', got %d", len(classIdents))
		}
		if len(classIdents) > 0 && !classIdents[0].effectiveRange.global {
			t.Errorf("Expected 'Eq' to be global")
		}
	})

	t.Run("ClassDecl_WithContext", func(t *testing.T) {
		code := "class Eq a => Ord a where\n  compare :: a -> a -> Ordering"
		env := &RenameEnv{}
		moduleAST := parser.Parse([]byte(code), "Test")
		result := env.GenIdentifiers(*moduleAST)

		// Check class identifier
		classIdents := hasClassIdents(t, result, "Test", []string{"Ord"})
		if len(classIdents) != 1 {
			t.Errorf("Expected 1 class identifier for 'Ord', got %d", len(classIdents))
		}
		if len(classIdents) > 0 && !classIdents[0].effectiveRange.global {
			t.Errorf("Expected 'Ord' to be global")
		}
	})

	t.Run("ClassDecl_WithMultipleMethods", func(t *testing.T) {
		code := "class Monad m where\n  bind :: m a -> (a -> m b) -> m b\n  return :: a -> m a"
		env := &RenameEnv{}
		moduleAST := parser.Parse([]byte(code), "Test")
		result := env.GenIdentifiers(*moduleAST)

		// Check class identifier
		classIdents := hasClassIdents(t, result, "Test", []string{"Monad"})
		if len(classIdents) != 1 {
			t.Errorf("Expected 1 class identifier for 'Monad', got %d", len(classIdents))
		}
		if len(classIdents) > 0 && !classIdents[0].effectiveRange.global {
			t.Errorf("Expected 'Monad' to be global")
		}
		// Note: Method type signatures within class declarations are currently
		// treated as local to the class, not global identifiers
	})

}
