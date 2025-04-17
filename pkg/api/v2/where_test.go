//go:build basicv2

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
