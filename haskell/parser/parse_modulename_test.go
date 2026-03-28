package parser

import "testing"

func TestGuessModuleName(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "simple file",
			filePath: "list.hs",
			expected: "List",
		},
		{
			name:     "file with path",
			filePath: "./data/list.hs",
			expected: "Data.List",
		},
		{
			name:     "nested path",
			filePath: "./data/map/internal.hs",
			expected: "Data.Map.Internal",
		},
		{
			name:     "path without leading ./",
			filePath: "data/list.hs",
			expected: "Data.List",
		},
		{
			name:     "single letter parts",
			filePath: "a/b/c.hs",
			expected: "A.B.C",
		},
		{
			name:     "already capitalized",
			filePath: "Data/List.hs",
			expected: "Data.List",
		},
		{
			name:     "mixed case",
			filePath: "myModule/myFile.hs",
			expected: "MyModule.MyFile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GuessModuleName(tt.filePath, ".")
			if result != tt.expected {
				t.Errorf("GuessModuleName(%q, \".\") = %q, want %q", tt.filePath, result, tt.expected)
			}
		})
	}
}
