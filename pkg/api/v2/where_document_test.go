package v2

import (
	"testing"
)

func TestWhereDocument(t *testing.T) {
	tests := []struct {
		name     string
		filter   WhereDocumentFilter
		expected string
	}{
		{
			name:     "contain",
			filter:   Contains("test"),
			expected: `{"$contains":"test"}`,
		},
		{
			name:     "not contain",
			filter:   NotContains("test"),
			expected: `{"$not_contains":"test"}`,
		},
		{
			name:     "or",
			filter:   OrDocument(Contains("test"), NotContains("test")),
			expected: `{"$or":[{"$contains":"test"},{"$not_contains":"test"}]}`,
		},
		{
			name:     "and",
			filter:   AndDocument(Contains("test"), NotContains("test")),
			expected: `{"$and":[{"$contains":"test"},{"$not_contains":"test"}]}`,
		},
		{
			name:     "or and",
			filter:   OrDocument(AndDocument(Contains("test"), NotContains("test")), Contains("test")),
			expected: `{"$or":[{"$and":[{"$contains":"test"},{"$not_contains":"test"}]},{"$contains":"test"}]}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := test.filter.MarshalJSON()
			if err != nil {
				t.Errorf("error marshalling filter: %v", err)
			}
			if string(actual) != test.expected {
				t.Errorf("expected %s, got %s", test.expected, string(actual))
			}
		})
	}
}
