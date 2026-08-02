//go:build basicv2 && !cloud

package v2

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
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
		{
			name:     "regex",
			filter:   Regex("^[a-zA-Z0-9._%+-]+$"),
			expected: `{"$regex":"^[a-zA-Z0-9._%+-]+$"}`,
		},
		{
			name:     "not regex",
			filter:   NotRegex("^[a-zA-Z0-9._%+-]+$"),
			expected: `{"$not_regex":"^[a-zA-Z0-9._%+-]+$"}`,
		},
		{
			name:     "and with regex/contains",
			filter:   AndDocument(Contains("test"), Regex("^[a-zA-Z0-9._%+-]+$")),
			expected: `{"$and":[{"$contains":"test"},{"$regex":"^[a-zA-Z0-9._%+-]+$"}]}`,
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

// TestNestedNilWhereDocumentClause mirrors the WhereClause And/Or nil-clause guard
// for WhereDocumentFilter compound clauses.
func TestNestedNilWhereDocumentClause(t *testing.T) {
	var nilDoc *WhereDocumentClauseContainsOrNotContains

	t.Run("nil clause in AndDocument returns an error", func(t *testing.T) {
		clause := AndDocument(Contains("draft"), nilDoc)
		var err error
		require.NotPanics(t, func() { err = clause.Validate() })
		require.EqualError(t, err, "nil clause in $and expression")
	})

	t.Run("nil clause in OrDocument returns an error", func(t *testing.T) {
		clause := OrDocument(Contains("draft"), nilDoc)
		var err error
		require.NotPanics(t, func() { err = clause.Validate() })
		require.EqualError(t, err, "nil clause in $or expression")
	})

	t.Run("valid compound document clauses still validate", func(t *testing.T) {
		require.NoError(t, AndDocument(Contains("a"), Contains("b")).Validate())
		require.NoError(t, OrDocument(Contains("a"), Contains("b")).Validate())
	})
}

func TestWhereDocumentExpressionDepthGuard(t *testing.T) {
	buildExpression := func(recursiveChildCalls int) WhereDocumentFilter {
		var filter WhereDocumentFilter = Contains("leaf")
		for i := 0; i < recursiveChildCalls; i++ {
			if i%2 == 0 {
				filter = AndDocument(filter)
			} else {
				filter = OrDocument(filter)
			}
		}
		return filter
	}

	t.Run("permits shared expression depth", func(t *testing.T) {
		filter := buildExpression(MaxExpressionDepth)
		require.NoError(t, filter.Validate())

		data, err := json.Marshal(filter)
		require.NoError(t, err)
		require.NotEmpty(t, data)
	})

	t.Run("validates each compound sibling from the parent depth", func(t *testing.T) {
		filter := AndDocument(
			buildExpression(MaxExpressionDepth-1),
			Contains("shallow"),
		)

		require.NoError(t, filter.Validate())

		data, err := json.Marshal(filter)
		require.NoError(t, err)
		require.NotEmpty(t, data)
	})

	t.Run("rejects one child beyond shared expression depth", func(t *testing.T) {
		filter := buildExpression(MaxExpressionDepth + 1)
		expectedErr := fmt.Sprintf("where document expression exceeds maximum depth of %d", MaxExpressionDepth)

		var validateErr error
		require.NotPanics(t, func() { validateErr = filter.Validate() })
		require.ErrorContains(t, validateErr, expectedErr)

		var data []byte
		var marshalErr error
		require.NotPanics(t, func() { data, marshalErr = json.Marshal(filter) })
		require.Nil(t, data)
		require.ErrorContains(t, marshalErr, expectedErr)
	})
}
