package parser

import (
	"testing"
)

func TestLocIsInside(t *testing.T) {
	tests := []struct {
		name     string
		loc      Loc
		other    Loc
		expected bool
	}{
		{
			name: "completely inside",
			loc: Loc{
				fromLine: 2,
				toLine:   3,
				fromCol:  5,
				toCol:    10,
			},
			other: Loc{
				fromLine: 1,
				toLine:   5,
				fromCol:  0,
				toCol:    20,
			},
			expected: true,
		},
		{
			name: "same location",
			loc: Loc{
				fromLine: 2,
				toLine:   3,
				fromCol:  5,
				toCol:    10,
			},
			other: Loc{
				fromLine: 2,
				toLine:   3,
				fromCol:  5,
				toCol:    10,
			},
			expected: true,
		},
		{
			name: "starts before other",
			loc: Loc{
				fromLine: 1,
				toLine:   3,
				fromCol:  0,
				toCol:    10,
			},
			other: Loc{
				fromLine: 2,
				toLine:   5,
				fromCol:  0,
				toCol:    20,
			},
			expected: false,
		},
		{
			name: "ends after other",
			loc: Loc{
				fromLine: 2,
				toLine:   6,
				fromCol:  5,
				toCol:    20,
			},
			other: Loc{
				fromLine: 1,
				toLine:   5,
				fromCol:  0,
				toCol:    20,
			},
			expected: false,
		},
		{
			name: "on same line, starts inside",
			loc: Loc{
				fromLine: 1,
				toLine:   1,
				fromCol:  5,
				toCol:    10,
			},
			other: Loc{
				fromLine: 1,
				toLine:   1,
				fromCol:  0,
				toCol:    20,
			},
			expected: true,
		},
		{
			name: "on same line, starts before",
			loc: Loc{
				fromLine: 1,
				toLine:   1,
				fromCol:  0,
				toCol:    10,
			},
			other: Loc{
				fromLine: 1,
				toLine:   1,
				fromCol:  5,
				toCol:    20,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.loc.IsInside(tt.other)
			if result != tt.expected {
				t.Errorf("IsInside() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLocEnvelopes(t *testing.T) {
	tests := []struct {
		name     string
		loc      Loc
		other    Loc
		expected bool
	}{
		{
			name: "completely envelopes",
			loc: Loc{
				fromLine: 1,
				toLine:   5,
				fromCol:  0,
				toCol:    20,
			},
			other: Loc{
				fromLine: 2,
				toLine:   3,
				fromCol:  5,
				toCol:    10,
			},
			expected: true,
		},
		{
			name: "same location",
			loc: Loc{
				fromLine: 2,
				toLine:   3,
				fromCol:  5,
				toCol:    10,
			},
			other: Loc{
				fromLine: 2,
				toLine:   3,
				fromCol:  5,
				toCol:    10,
			},
			expected: true,
		},
		{
			name: "starts after other",
			loc: Loc{
				fromLine: 2,
				toLine:   5,
				fromCol:  0,
				toCol:    20,
			},
			other: Loc{
				fromLine: 1,
				toLine:   3,
				fromCol:  0,
				toCol:    10,
			},
			expected: false,
		},
		{
			name: "ends before other",
			loc: Loc{
				fromLine: 1,
				toLine:   4,
				fromCol:  0,
				toCol:    20,
			},
			other: Loc{
				fromLine: 2,
				toLine:   5,
				fromCol:  0,
				toCol:    20,
			},
			expected: false,
		},
		{
			name: "on same line, envelopes",
			loc: Loc{
				fromLine: 1,
				toLine:   1,
				fromCol:  0,
				toCol:    20,
			},
			other: Loc{
				fromLine: 1,
				toLine:   1,
				fromCol:  5,
				toCol:    10,
			},
			expected: true,
		},
		{
			name: "on same line, doesn't envelop",
			loc: Loc{
				fromLine: 1,
				toLine:   1,
				fromCol:  5,
				toCol:    20,
			},
			other: Loc{
				fromLine: 1,
				toLine:   1,
				fromCol:  0,
				toCol:    10,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.loc.Envelopes(tt.other)
			if result != tt.expected {
				t.Errorf("Envelopes() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLocEqual(t *testing.T) {
	tests := []struct {
		name     string
		loc      Loc
		other    Loc
		expected bool
	}{
		{
			name: "exactly equal",
			loc: Loc{
				fromLine: 2,
				toLine:   3,
				fromCol:  5,
				toCol:    10,
			},
			other: Loc{
				fromLine: 2,
				toLine:   3,
				fromCol:  5,
				toCol:    10,
			},
			expected: true,
		},
		{
			name: "different fromLine",
			loc: Loc{
				fromLine: 1,
				toLine:   3,
				fromCol:  5,
				toCol:    10,
			},
			other: Loc{
				fromLine: 2,
				toLine:   3,
				fromCol:  5,
				toCol:    10,
			},
			expected: false,
		},
		{
			name: "different toLine",
			loc: Loc{
				fromLine: 2,
				toLine:   4,
				fromCol:  5,
				toCol:    10,
			},
			other: Loc{
				fromLine: 2,
				toLine:   3,
				fromCol:  5,
				toCol:    10,
			},
			expected: false,
		},
		{
			name: "different fromCol",
			loc: Loc{
				fromLine: 2,
				toLine:   3,
				fromCol:  4,
				toCol:    10,
			},
			other: Loc{
				fromLine: 2,
				toLine:   3,
				fromCol:  5,
				toCol:    10,
			},
			expected: false,
		},
		{
			name: "different toCol",
			loc: Loc{
				fromLine: 2,
				toLine:   3,
				fromCol:  5,
				toCol:    11,
			},
			other: Loc{
				fromLine: 2,
				toLine:   3,
				fromCol:  5,
				toCol:    10,
			},
			expected: false,
		},
		{
			name: "all different",
			loc: Loc{
				fromLine: 1,
				toLine:   2,
				fromCol:  3,
				toCol:    4,
			},
			other: Loc{
				fromLine: 5,
				toLine:   6,
				fromCol:  7,
				toCol:    8,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.loc.Equal(tt.other)
			if result != tt.expected {
				t.Errorf("Equal() = %v, want %v", result, tt.expected)
			}
		})
	}
}
