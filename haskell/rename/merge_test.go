package rename

import (
	"goanna/haskell/parser"
	"testing"
)

func TestMergeTermIdentifiers(t *testing.T) {
	tests := []struct {
		name     string
		input    []TermIdentifier
		expected int // expected number of merged identifiers
	}{
		{
			name:     "no duplicates",
			input:    []TermIdentifier{},
			expected: 0,
		},
		{
			name: "single identifier",
			input: []TermIdentifier{
				{
					Identifier: Identifier{
						name:   "foo",
						module: "Test",
						effectiveRange: EffectiveRange{
							ranges: []parser.Loc{},
							global: true,
						},
						declaredAt: []parser.Loc{parser.NewLoc(0, 1, 0, 3)},
					},
				},
			},
			expected: 1,
		},
		{
			name: "merge global identifiers with same module and name",
			input: []TermIdentifier{
				{
					Identifier: Identifier{
						name:   "foo",
						module: "Test",
						effectiveRange: EffectiveRange{
							ranges: []parser.Loc{},
							global: true,
						},
						declaredAt: []parser.Loc{parser.NewLoc(0, 1, 0, 3)},
					},
				},
				{
					Identifier: Identifier{
						name:   "foo",
						module: "Test",
						effectiveRange: EffectiveRange{
							ranges: []parser.Loc{},
							global: true,
						},
						declaredAt: []parser.Loc{parser.NewLoc(2, 3, 0, 3)},
					},
				},
			},
			expected: 1,
		},
		{
			name: "don't merge different names",
			input: []TermIdentifier{
				{
					Identifier: Identifier{
						name:   "foo",
						module: "Test",
						effectiveRange: EffectiveRange{
							ranges: []parser.Loc{},
							global: true,
						},
						declaredAt: []parser.Loc{parser.NewLoc(0, 1, 0, 3)},
					},
				},
				{
					Identifier: Identifier{
						name:   "bar",
						module: "Test",
						effectiveRange: EffectiveRange{
							ranges: []parser.Loc{},
							global: true,
						},
						declaredAt: []parser.Loc{parser.NewLoc(2, 3, 0, 3)},
					},
				},
			},
			expected: 2,
		},
		{
			name: "don't merge different modules",
			input: []TermIdentifier{
				{
					Identifier: Identifier{
						name:   "foo",
						module: "Test1",
						effectiveRange: EffectiveRange{
							ranges: []parser.Loc{},
							global: true,
						},
						declaredAt: []parser.Loc{parser.NewLoc(0, 1, 0, 3)},
					},
				},
				{
					Identifier: Identifier{
						name:   "foo",
						module: "Test2",
						effectiveRange: EffectiveRange{
							ranges: []parser.Loc{},
							global: true,
						},
						declaredAt: []parser.Loc{parser.NewLoc(2, 3, 0, 3)},
					},
				},
			},
			expected: 2,
		},
		{
			name: "don't merge different effective ranges",
			input: []TermIdentifier{
				{
					Identifier: Identifier{
						name:   "foo",
						module: "Test",
						effectiveRange: EffectiveRange{
							ranges: []parser.Loc{},
							global: true,
						},
						declaredAt: []parser.Loc{parser.NewLoc(0, 1, 0, 3)},
					},
				},
				{
					Identifier: Identifier{
						name:   "foo",
						module: "Test",
						effectiveRange: EffectiveRange{
							ranges: []parser.Loc{parser.NewLoc(0, 10, 0, 0)},
							global: false,
						},
						declaredAt: []parser.Loc{parser.NewLoc(2, 3, 0, 3)},
					},
				},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeTermIdentifiers(tt.input)
			if len(result) != tt.expected {
				t.Errorf("Expected %d merged identifiers, got %d", tt.expected, len(result))
			}

			// For the merge test, verify declaredAt was merged correctly
			if tt.name == "merge global identifiers with same module and name" {
				if len(result) > 0 && len(result[0].declaredAt) != 2 {
					t.Errorf("Expected 2 declared locations, got %d", len(result[0].declaredAt))
				}
			}
		})
	}
}

func TestEffectiveRangeEqual(t *testing.T) {
	tests := []struct {
		name     string
		er1      EffectiveRange
		er2      EffectiveRange
		expected bool
	}{
		{
			name: "both global",
			er1: EffectiveRange{
				ranges: []parser.Loc{},
				global: true,
			},
			er2: EffectiveRange{
				ranges: []parser.Loc{},
				global: true,
			},
			expected: true,
		},
		{
			name: "one global, one local",
			er1: EffectiveRange{
				ranges: []parser.Loc{},
				global: true,
			},
			er2: EffectiveRange{
				ranges: []parser.Loc{parser.NewLoc(0, 10, 0, 0)},
				global: false,
			},
			expected: false,
		},
		{
			name: "both local, same ranges",
			er1: EffectiveRange{
				ranges: []parser.Loc{parser.NewLoc(0, 10, 0, 0)},
				global: false,
			},
			er2: EffectiveRange{
				ranges: []parser.Loc{parser.NewLoc(0, 10, 0, 0)},
				global: false,
			},
			expected: true,
		},
		{
			name: "both local, different ranges",
			er1: EffectiveRange{
				ranges: []parser.Loc{parser.NewLoc(0, 10, 0, 0)},
				global: false,
			},
			er2: EffectiveRange{
				ranges: []parser.Loc{parser.NewLoc(0, 20, 0, 0)},
				global: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.er1.Equal(tt.er2)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
