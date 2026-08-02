package services

import (
	"reflect"
	"testing"
)

func TestExtractJSONObjects(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "single object",
			raw:  `{"a":1}`,
			want: []string{`{"a":1}`},
		},
		{
			name: "two adjacent objects",
			raw:  `{"a":1}{"b":2}`,
			want: []string{`{"a":1}`, `{"b":2}`},
		},
		{
			name: "braces inside string",
			raw:  `{"a":"{not an object}","b":2}`,
			want: []string{`{"a":"{not an object}","b":2}`},
		},
		{
			name: "nested objects",
			raw:  `{"a":{"b":1},"c":[{"d":2}]}`,
			want: []string{`{"a":{"b":1},"c":[{"d":2}]}`},
		},
		{
			name: "prose around object",
			raw:  `Here is the JSON: {"a":1} Hope that helps`,
			want: []string{`{"a":1}`},
		},
		{
			name: "escaped quotes in string",
			raw:  `{"a":"say \"hi\" now","b":1}`,
			want: []string{`{"a":"say \"hi\" now","b":1}`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSONObjects(tt.raw)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractJSONObjects(%q) = %#v, want %#v", tt.raw, got, tt.want)
			}
		})
	}
}
