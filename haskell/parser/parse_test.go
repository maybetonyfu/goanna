package parser

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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
		{"x = 1 * 2 + 3", "module Main where\nx = ((1 * 2) + 3)"},
		{"x = 1 + 2 * 3", "module Main where\nx = (1 + (2 * 3))"},
	}

	for _, tc := range cases {
		output := parse([]byte(tc.input), "Main").pretty()
		assert.Equal(t, output, tc.expect, "Output should equal expected")
	}
}
