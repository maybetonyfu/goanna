package parser

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func withModule(text string) string {
	return "module Main where\n" + text
}

func TestModule(t *testing.T) {
	type testcase struct {
		input  string
		expect string
	}

	cases := []testcase{
		{"module X", "module X where"},
		{"module Y where", "module Y where"},
		{"", "module Main where"},
		{"x = 1", "module Main where\nx = 1"},
	}

	for _, tc := range cases {
		output := parse([]byte(tc.input), "Main").pretty()
		assert.Equal(t, tc.expect, output, "Output should equal expected")
	}
}

func TestExp(t *testing.T) {
	type testcase struct {
		input  string
		expect string
	}

	cases := []testcase{
		{"x = y z", "x = (y z)"}, // Application
		{"x = (* z)", "x = (* z)"}, // Sectioning
		{"x = (z *)", "x = (z *)"}, // Sectioning
		{"x = (1, 2, 3)", "x = (1, 2, 3)"}, // Tuple
		{"x = []", "x = []"}, // List Empty
		{"x = [1, 2, 3]", "x = [1, 2, 3]"}, // List
		{"x = 1 / 3", "x = (1 / 3)"}, // Infix
		{"x = 1 * 2 + 3", "x = ((1 * 2) + 3)"},
		{"x = 1 + 2 * 3", "x = (1 + (2 * 3))"},
		{"x = 1 + 2 + 3", "x = ((1 + 2) + 3)"},
		{"x = a . b . c", "x = (a . (b . c))"},
		{"x = a . b $ c", "x = ((a . b) $ c)"},
		{"x = a . b . c $ d", "x = ((a . (b . c)) $ d)"},
		{"x = ()", "x = ()"}, // Unit
		{"x = \\a b -> a", "x = (\\a b -> a)"}, // lambda
		{"x = if True then 1 else 2", "x = if True then 1 else 2"}, // if
		{"x = case a of \n  1 -> 1\n  2 -> 2", "x = case a of 1 -> 1; 2 -> 2"}, //case
		{"x = let y = 1; z = y in z", "x = let {y = 1; z = y} in z"}, // let
		{"x = [1..3]", "x = [1..3]"}, // Enum From..To
   	{"x = [1..]", "x = [1..]"}, // Enum From..
		{"x = do {exp}", "x = do {exp}"}, // Do notation
  	{"x = do {x <- exp z}", "x = do {x <- (exp z)}"},
	 	{
			`x = do
  let x = 3
  return x`, "x = do {let x = 3; (return x)}"},
		{"x = [(x, y) | x <- [1..3], y <- [1..4], x < y]", "x = [(x, y) | x <- [1..3], y <- [1..4], (x < y)]"},
	}

	for _, tc := range cases {
		output := parse([]byte(tc.input), "Main").pretty()
		assert.Equal(t, withModule(tc.expect),output, "Output should equal expected")
	}
}

func TestType(t *testing.T) {
	type testcase struct {
		input  string
		expect string
	}

	cases := []testcase{
		{"x :: Int", "x :: Int"}, // TCon
		{"x :: a", "x :: a"}, // TVar
		{"x, y :: a", "x, y :: a"}, // Multi Decl
		{"x :: a -> b", "x :: a -> (b)"}, // Func
		{"x :: a -> b -> c", "x :: a -> (b -> (c))"}, // Func
	}

	for _, tc := range cases {
		output := parse([]byte(tc.input), "Main").pretty()
		assert.Equal(t, withModule(tc.expect),output, "Output should equal expected")
	}
}


