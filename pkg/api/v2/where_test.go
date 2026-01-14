//go:build basicv2 && !cloud

package v2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWhere(t *testing.T) {
	var tests = []struct {
		name       string
		clause     WhereClause
		expected   string
		shouldFail bool
	}{
		{
			name: "eq string",
			clause: func() WhereClause {
				return EqString("name", "value")
			}(),
			expected: `{"name":{"$eq":"value"}}`,
		},
		{
			name: "eq int",
			clause: func() WhereClause {
				return EqInt("name", 42)
			}(),
			expected: `{"name":{"$eq":42}}`,
		},
		{
			name: "eq float",
			clause: func() WhereClause {
				return EqFloat("name", 42.42)
			}(),
			expected: `{"name":{"$eq":42.42}}`,
		},
		{
			name: "eq bool",
			clause: func() WhereClause {
				return EqBool("name", true)
			}(),
			expected: `{"name":{"$eq":true}}`,
		},

		{
			name: "Ne string",
			clause: func() WhereClause {
				return NotEqString("name", "value")
			}(),
			expected: `{"name":{"$ne":"value"}}`,
		},
		{
			name: "Ne int",
			clause: func() WhereClause {
				return NotEqInt("name", 42)
			}(),
			expected: `{"name":{"$ne":42}}`,
		},
		{
			name: "Ne float",
			clause: func() WhereClause {
				return NotEqFloat("name", 42.42)
			}(),
			expected: `{"name":{"$ne":42.42}}`,
		},
		{
			name: "Ne bool",
			clause: func() WhereClause {
				return NotEqBool("name", false)
			}(),
			expected: `{"name":{"$ne":false}}`,
		},
		{
			name: "Gt int",
			clause: func() WhereClause {
				return GtInt("name", 42)
			}(),
			expected: `{"name":{"$gt":42}}`,
		},
		{
			name: "Gte int",
			clause: func() WhereClause {
				return GteInt("name", 42)
			}(),
			expected: `{"name":{"$gte":42}}`,
		},
		{
			name: "Gt float",
			clause: func() WhereClause {
				return GtFloat("name", 42.42)
			}(),
			expected: `{"name":{"$gt":42.42}}`,
		},
		{
			name: "Gte float",
			clause: func() WhereClause {
				return GteFloat("name", 42.42)
			}(),
			expected: `{"name":{"$gte":42.42}}`,
		},

		//-----
		{
			name: "Lt int",
			clause: func() WhereClause {
				return LtInt("name", 42)
			}(),
			expected: `{"name":{"$lt":42}}`,
		},
		{
			name: "Lte int",
			clause: func() WhereClause {
				return LteInt("name", 42)
			}(),
			expected: `{"name":{"$lte":42}}`,
		},
		{
			name: "Lt float",
			clause: func() WhereClause {
				return LtFloat("name", 42.42)
			}(),
			expected: `{"name":{"$lt":42.42}}`,
		},
		{
			name: "Lte float",
			clause: func() WhereClause {
				return LteFloat("name", 42.42)
			}(),
			expected: `{"name":{"$lte":42.42}}`,
		},
		//-----
		{
			name: "In int",
			clause: func() WhereClause {
				return InInt("name", 42, 43)
			}(),
			expected: `{"name":{"$in":[42,43]}}`,
		},
		{
			name: "In float",
			clause: func() WhereClause {
				return InFloat("name", 42.42, 43.43)
			}(),
			expected: `{"name":{"$in":[42.42, 43.43]}}`,
		},
		{
			name: "In string",
			clause: func() WhereClause {
				return InString("name", "ok", "ko")
			}(),
			expected: `{"name":{"$in":["ok","ko"]}}`,
		},
		{
			name: "In bool",
			clause: func() WhereClause {
				return InBool("name", true, false)
			}(),
			expected: `{"name":{"$in":[true,false]}}`,
		},
		//----
		{
			name: "Nin int",
			clause: func() WhereClause {
				return NinInt("name", 42, 43)
			}(),
			expected: `{"name":{"$nin":[42,43]}}`,
		},
		{
			name: "Nin float",
			clause: func() WhereClause {
				return NinFloat("name", 42.42, 43.43)
			}(),
			expected: `{"name":{"$nin":[42.42, 43.43]}}`,
		},
		{
			name: "Nin string",
			clause: func() WhereClause {
				return NinString("name", "ok", "ko")
			}(),
			expected: `{"name":{"$nin":["ok","ko"]}}`,
		},
		{
			name: "Nin bool",
			clause: func() WhereClause {
				return NinBool("name", true, false)
			}(),
			expected: `{"name":{"$nin":[true,false]}}`,
		},
		//--- ID filters
		{
			name: "IDIn",
			clause: func() WhereClause {
				return IDIn("doc1", "doc2", "doc3")
			}(),
			expected: `{"#id":{"$in":["doc1","doc2","doc3"]}}`,
		},
		{
			name: "IDNotIn",
			clause: func() WhereClause {
				return IDNotIn("seen1", "seen2")
			}(),
			expected: `{"#id":{"$nin":["seen1","seen2"]}}`,
		},
		{
			name: "IDNotIn combined with And",
			clause: func() WhereClause {
				return And(EqString("category", "tech"), IDNotIn("seen1", "seen2"))
			}(),
			expected: `{"$and":[{"category":{"$eq":"tech"}},{"#id":{"$nin":["seen1","seen2"]}}]}`,
		},
		//--- Document content filters
		{
			name: "DocumentContains",
			clause: func() WhereClause {
				return DocumentContains("search text")
			}(),
			expected: `{"#document":{"$contains":"search text"}}`,
		},
		{
			name: "DocumentNotContains",
			clause: func() WhereClause {
				return DocumentNotContains("excluded text")
			}(),
			expected: `{"#document":{"$not_contains":"excluded text"}}`,
		},
		{
			name: "DocumentContains combined with metadata filter",
			clause: func() WhereClause {
				return And(EqString("category", "tech"), DocumentContains("AI"))
			}(),
			expected: `{"$and":[{"category":{"$eq":"tech"}},{"#document":{"$contains":"AI"}}]}`,
		},
		//---
		{
			name: "And",
			clause: func() WhereClause {
				return And(EqString("name", "value"), EqInt("age", 42))
			}(),
			expected: `{"$and":[{"name":{"$eq":"value"}},{"age":{"$eq":42}}]}`,
		},
		{
			name: "Or",
			clause: func() WhereClause {
				return Or(EqString("name", "value"), EqInt("age", 42))
			}(),
			expected: `{"$or":[{"name":{"$eq":"value"}},{"age":{"$eq":42}}]}`,
		},

		{
			name: "And Or",
			clause: func() WhereClause {
				return Or(EqString("name", "value"), EqInt("age", 42), Or(EqString("name", "value"), EqInt("age", 42)))
			}(),
			expected: `{"$or":[{"name":{"$eq":"value"}},{"age":{"$eq":42}},{"$or":[{"name":{"$eq":"value"}},{"age":{"$eq":42}}]}]}`,
		},
		{
			name: "And Or And",
			clause: func() WhereClause {
				return Or(EqString("name", "value"), EqInt("age", 42), And(EqString("name", "value"), EqInt("age", 42)))
			}(),
			expected: `{"$or":[{"name":{"$eq":"value"}},{"age":{"$eq":42}},{"$and":[{"name":{"$eq":"value"}},{"age":{"$eq":42}}]}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			json, err := tt.clause.MarshalJSON()
			if tt.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.JSONEq(t, tt.expected, string(json))
		})
	}
}

func TestWhereClauseEmptyOperandValidation(t *testing.T) {
	tests := []struct {
		name        string
		clause      WhereClause
		expectedErr string
	}{
		{
			name:        "IDIn with no arguments",
			clause:      IDIn(),
			expectedErr: "invalid operand for $in on key \"#id\", expected at least one value",
		},
		{
			name:        "IDNotIn with no arguments",
			clause:      IDNotIn(),
			expectedErr: "invalid operand for $nin on key \"#id\", expected at least one value",
		},
		{
			name:        "InString with no values",
			clause:      InString("field"),
			expectedErr: "invalid operand for $in on key \"field\", expected at least one value",
		},
		{
			name:        "NinString with no values",
			clause:      NinString("field"),
			expectedErr: "invalid operand for $nin on key \"field\", expected at least one value",
		},
		{
			name:        "Empty IDIn nested in And",
			clause:      And(EqString("status", "active"), IDIn()),
			expectedErr: "invalid operand for $in on key \"#id\", expected at least one value",
		},
		{
			name:        "DocumentContains with empty string",
			clause:      DocumentContains(""),
			expectedErr: "invalid operand for $contains on key \"#document\", expected non-empty string",
		},
		{
			name:        "DocumentNotContains with empty string",
			clause:      DocumentNotContains(""),
			expectedErr: "invalid operand for $not_contains on key \"#document\", expected non-empty string",
		},
		{
			name:        "Empty DocumentContains nested in And",
			clause:      And(EqString("category", "tech"), DocumentContains("")),
			expectedErr: "invalid operand for $contains on key \"#document\", expected non-empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Construction should succeed (lazy validation)
			require.NotNil(t, tt.clause)

			// Validation should fail
			err := tt.clause.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}
