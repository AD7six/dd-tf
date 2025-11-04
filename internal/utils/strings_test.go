package utils

import (
	"reflect"
	"testing"
)

func TestParseCommaSeparatedIDs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  []string
	}{
		{"empty string", "", []string{}},
		{"single value", "abc", []string{"abc"}},
		{"simple list", "a,b,c", []string{"a", "b", "c"}},
		{"trims spaces", " a ,  b , c ", []string{"a", "b", "c"}},
		{"deduplicates", "a,b,a,c,b", []string{"a", "b", "c"}},
		{"ignores empties", ",,a,,b, ,c,", []string{"a", "b", "c"}},
		{"keeps case", "A,a", []string{"A", "a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCommaSeparatedIDs(tt.in)
			if !reflect.DeepEqual(got, tt.out) {
				t.Fatalf("ParseCommaSeparatedIDs(%q) = %#v, want %#v", tt.in, got, tt.out)
			}
		})
	}
}
