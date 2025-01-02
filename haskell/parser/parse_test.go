package parser

import (
	"github.com/stretchr/testify/assert"
	"testing"
)
	func withModule(text string) string {
		return "module Main where\n" + text
	}

func TestParser(t *testing.T) {
	type testcase struct {
		input  string
		expect string
	}


	cases := []testcase{
		{"module X", "module X where"},
		{"module Y where", "module Y where"},
		{"", "module Main where"},
		{"x = 1", "module Main where\nx = 1"},
		{"x = y z", "module Main where\nx = (y z)"},
		{"x = 1 / 3", "module Main where\nx = (1 / 3)"},
		{"x = 1 * 2 + 3", "module Main where\nx = ((1 * 2) + 3)"},
		{"x = 1 + 2 * 3", "module Main where\nx = (1 + (2 * 3))"},
		{"x = 1 + 2 + 3", "module Main where\nx = ((1 + 2) + 3)"},
		{"x = a . b . c", "module Main where\nx = (a . (b . c))"},
		{"x = a . b $ c", "module Main where\nx = ((a . b) $ c)"},
		{"x = a . b . c $ d", "module Main where\nx = ((a . (b . c)) $ d)"},
		{"x = ()", withModule("x = ()")},
  	{"x = \\a b -> a", withModule("x = (\\a b -> a)")},
		{"x = if True then 1 else 2", withModule("x = if True then 1 else 2")},
		{"x = case a of \n  1 -> 1\n  2 -> 2", withModule("x = case a of 1 -> 1; 2 -> 2;")},
	}

	for _, tc := range cases {
		output := parse([]byte(tc.input), "Main").pretty()
		assert.Equal(t, tc.expect, output, "Output should equal expected")
	}
}
