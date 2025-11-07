// go:build basicv2
//go:build basicv2

package v2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func TestKnnRank_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rank    *KnnRank
		wantErr bool
	}{
		{
			name: "valid with query texts",
			rank: &KnnRank{
				QueryTexts: []string{"test query"},
				K:          5,
			},
			wantErr: false,
		},
		{
			name: "valid with query embeddings",
			rank: &KnnRank{
				QueryEmbeddings: []embeddings.Embedding{embeddings.NewFloat32Embedding([]float32{1.0, 2.0})},
				K:               5,
			},
			wantErr: false,
		},
		{
			name: "invalid - no query",
			rank: &KnnRank{
				K: 5,
			},
			wantErr: true,
		},
		{
			name: "invalid - k is zero",
			rank: &KnnRank{
				QueryTexts: []string{"test"},
				K:          0,
			},
			wantErr: true,
		},
		{
			name: "invalid - k is negative",
			rank: &KnnRank{
				QueryTexts: []string{"test"},
				K:          -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rank.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestKnnRank_ToJSON(t *testing.T) {
	rank := &KnnRank{
		QueryTexts: []string{"test query"},
		K:          5,
	}

	result := rank.ToJSON()
	require.NotNil(t, result)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "knn", resultMap["type"])
	assert.Equal(t, 5, resultMap["k"])
	assert.Equal(t, []string{"test query"}, resultMap["query_texts"])
}

func TestRrfRank_Validate(t *testing.T) {
	validKnn1 := &KnnRank{QueryTexts: []string{"query1"}, K: 5}
	validKnn2 := &KnnRank{QueryTexts: []string{"query2"}, K: 5}
	invalidKnn := &KnnRank{K: 5} // missing query

	tests := []struct {
		name    string
		rank    *RrfRank
		wantErr bool
	}{
		{
			name: "valid with 2 ranks",
			rank: &RrfRank{
				Ranks:     []RankExpression{validKnn1, validKnn2},
				K:         60,
				Normalize: true,
			},
			wantErr: false,
		},
		{
			name: "invalid - only 1 rank",
			rank: &RrfRank{
				Ranks: []RankExpression{validKnn1},
				K:     60,
			},
			wantErr: true,
		},
		{
			name: "invalid - contains invalid rank",
			rank: &RrfRank{
				Ranks: []RankExpression{validKnn1, invalidKnn},
				K:     60,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rank.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestArithmeticRank_Validate(t *testing.T) {
	validKnn := &KnnRank{QueryTexts: []string{"query"}, K: 5}
	invalidKnn := &KnnRank{K: 5}

	tests := []struct {
		name    string
		rank    *ArithmeticRank
		wantErr bool
	}{
		{
			name: "valid add",
			rank: &ArithmeticRank{
				Operator: "add",
				Left:     validKnn,
				Right:    validKnn,
			},
			wantErr: false,
		},
		{
			name: "valid sub",
			rank: &ArithmeticRank{
				Operator: "sub",
				Left:     validKnn,
				Right:    validKnn,
			},
			wantErr: false,
		},
		{
			name: "valid mul",
			rank: &ArithmeticRank{
				Operator: "mul",
				Left:     validKnn,
				Right:    validKnn,
			},
			wantErr: false,
		},
		{
			name: "valid div",
			rank: &ArithmeticRank{
				Operator: "div",
				Left:     validKnn,
				Right:    validKnn,
			},
			wantErr: false,
		},
		{
			name: "invalid operator",
			rank: &ArithmeticRank{
				Operator: "mod",
				Left:     validKnn,
				Right:    validKnn,
			},
			wantErr: true,
		},
		{
			name: "invalid left operand",
			rank: &ArithmeticRank{
				Operator: "add",
				Left:     invalidKnn,
				Right:    validKnn,
			},
			wantErr: true,
		},
		{
			name: "invalid right operand",
			rank: &ArithmeticRank{
				Operator: "add",
				Left:     validKnn,
				Right:    invalidKnn,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rank.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFunctionRank_Validate(t *testing.T) {
	validKnn := &KnnRank{QueryTexts: []string{"query"}, K: 5}
	invalidKnn := &KnnRank{K: 5}

	tests := []struct {
		name    string
		rank    *FunctionRank
		wantErr bool
	}{
		{
			name: "valid exp",
			rank: &FunctionRank{
				Function: "exp",
				Operand:  validKnn,
			},
			wantErr: false,
		},
		{
			name: "valid log",
			rank: &FunctionRank{
				Function: "log",
				Operand:  validKnn,
			},
			wantErr: false,
		},
		{
			name: "valid abs",
			rank: &FunctionRank{
				Function: "abs",
				Operand:  validKnn,
			},
			wantErr: false,
		},
		{
			name: "invalid function",
			rank: &FunctionRank{
				Function: "sqrt",
				Operand:  validKnn,
			},
			wantErr: true,
		},
		{
			name: "invalid operand",
			rank: &FunctionRank{
				Function: "exp",
				Operand:  invalidKnn,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rank.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCollectionSearchOp_PrepareAndValidate(t *testing.T) {
	tests := []struct {
		name    string
		op      *CollectionSearchOp
		wantErr bool
	}{
		{
			name: "valid search with knn",
			op: &CollectionSearchOp{
				Rank: &KnnRank{
					QueryTexts: []string{"test"},
					K:          5,
				},
			},
			wantErr: false,
		},
		{
			name: "valid search with where filter",
			op: &CollectionSearchOp{
				Rank: &KnnRank{
					QueryTexts: []string{"test"},
					K:          5,
				},
				Where: EqString("category", "tech"),
			},
			wantErr: false,
		},
		{
			name: "valid search with limit",
			op: &CollectionSearchOp{
				Rank: &KnnRank{
					QueryTexts: []string{"test"},
					K:          5,
				},
				Limit: &SearchLimit{
					Limit:  10,
					Offset: 5,
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid - missing rank",
			op:      &CollectionSearchOp{},
			wantErr: true,
		},
		{
			name: "invalid - invalid rank",
			op: &CollectionSearchOp{
				Rank: &KnnRank{K: 5}, // missing query
			},
			wantErr: true,
		},
		{
			name: "invalid - negative limit",
			op: &CollectionSearchOp{
				Rank: &KnnRank{
					QueryTexts: []string{"test"},
					K:          5,
				},
				Limit: &SearchLimit{
					Limit:  -1,
					Offset: 0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid - negative offset",
			op: &CollectionSearchOp{
				Rank: &KnnRank{
					QueryTexts: []string{"test"},
					K:          5,
				},
				Limit: &SearchLimit{
					Limit:  10,
					Offset: -1,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op.PrepareAndValidate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCollectionSearchOp_MarshalJSON(t *testing.T) {
	op := &CollectionSearchOp{
		Rank: &KnnRank{
			QueryTexts: []string{"test query"},
			K:          5,
		},
		Where: EqString("category", "tech"),
		Limit: &SearchLimit{
			Limit:  10,
			Offset: 5,
		},
		Select: []SelectKey{SelectID, SelectDocument, SelectScore},
	}

	data, err := op.MarshalJSON()
	require.NoError(t, err)
	require.NotNil(t, data)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Verify rank
	assert.NotNil(t, result["rank"])
	rankMap, ok := result["rank"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "knn", rankMap["type"])

	// Verify where
	assert.NotNil(t, result["where"])

	// Verify limit
	assert.NotNil(t, result["limit"])
	limitMap, ok := result["limit"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(10), limitMap["limit"])
	assert.Equal(t, float64(5), limitMap["offset"])

	// Verify select
	assert.NotNil(t, result["select"])
	selectList, ok := result["select"].([]interface{})
	require.True(t, ok)
	assert.Len(t, selectList, 3)
}

func TestArithmeticRankConstructors(t *testing.T) {
	knn1 := &KnnRank{QueryTexts: []string{"query1"}, K: 5}
	knn2 := &KnnRank{QueryTexts: []string{"query2"}, K: 5}

	tests := []struct {
		name     string
		rank     RankExpression
		operator string
	}{
		{
			name:     "AddRanks",
			rank:     AddRanks(knn1, knn2),
			operator: "add",
		},
		{
			name:     "SubRanks",
			rank:     SubRanks(knn1, knn2),
			operator: "sub",
		},
		{
			name:     "MulRanks",
			rank:     MulRanks(knn1, knn2),
			operator: "mul",
		},
		{
			name:     "DivRanks",
			rank:     DivRanks(knn1, knn2),
			operator: "div",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arithRank, ok := tt.rank.(*ArithmeticRank)
			require.True(t, ok)
			assert.Equal(t, tt.operator, arithRank.Operator)
			assert.NotNil(t, arithRank.Left)
			assert.NotNil(t, arithRank.Right)
			assert.NoError(t, arithRank.Validate())
		})
	}
}

func TestFunctionRankConstructors(t *testing.T) {
	knn := &KnnRank{QueryTexts: []string{"query"}, K: 5}

	tests := []struct {
		name     string
		rank     RankExpression
		function string
	}{
		{
			name:     "ExpRank",
			rank:     ExpRank(knn),
			function: "exp",
		},
		{
			name:     "LogRank",
			rank:     LogRank(knn),
			function: "log",
		},
		{
			name:     "AbsRank",
			rank:     AbsRank(knn),
			function: "abs",
		},
		{
			name:     "MaxRank",
			rank:     MaxRank(knn),
			function: "max",
		},
		{
			name:     "MinRank",
			rank:     MinRank(knn),
			function: "min",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			funcRank, ok := tt.rank.(*FunctionRank)
			require.True(t, ok)
			assert.Equal(t, tt.function, funcRank.Function)
			assert.NotNil(t, funcRank.Operand)
			assert.NoError(t, funcRank.Validate())
		})
	}
}

func TestSearchOptions(t *testing.T) {
	t.Run("WithSearchWhere", func(t *testing.T) {
		search := &CollectionSearchOp{}
		where := EqString("key", "value")
		opt := WithSearchWhere(where)
		err := opt(search)
		assert.NoError(t, err)
		assert.Equal(t, where, search.Where)
	})

	t.Run("WithSearchRankKnn", func(t *testing.T) {
		search := &CollectionSearchOp{}
		emb := embeddings.NewFloat32Embedding([]float32{1.0, 2.0})
		opt := WithSearchRankKnn([]embeddings.Embedding{emb}, 5)
		err := opt(search)
		assert.NoError(t, err)
		require.NotNil(t, search.Rank)
		knn, ok := search.Rank.(*KnnRank)
		require.True(t, ok)
		assert.Equal(t, 5, knn.K)
		assert.Len(t, knn.QueryEmbeddings, 1)
	})

	t.Run("WithSearchRankKnnTexts", func(t *testing.T) {
		search := &CollectionSearchOp{}
		opt := WithSearchRankKnnTexts([]string{"query"}, 5)
		err := opt(search)
		assert.NoError(t, err)
		require.NotNil(t, search.Rank)
		knn, ok := search.Rank.(*KnnRank)
		require.True(t, ok)
		assert.Equal(t, 5, knn.K)
		assert.Equal(t, []string{"query"}, knn.QueryTexts)
	})

	t.Run("WithSearchRankRrf", func(t *testing.T) {
		search := &CollectionSearchOp{}
		knn1 := &KnnRank{QueryTexts: []string{"q1"}, K: 5}
		knn2 := &KnnRank{QueryTexts: []string{"q2"}, K: 5}
		opt := WithSearchRankRrf([]RankExpression{knn1, knn2}, 60, true)
		err := opt(search)
		assert.NoError(t, err)
		require.NotNil(t, search.Rank)
		rrf, ok := search.Rank.(*RrfRank)
		require.True(t, ok)
		assert.Equal(t, 60, rrf.K)
		assert.True(t, rrf.Normalize)
		assert.Len(t, rrf.Ranks, 2)
	})

	t.Run("WithSearchLimit", func(t *testing.T) {
		search := &CollectionSearchOp{}
		opt := WithSearchLimit(10, 5)
		err := opt(search)
		assert.NoError(t, err)
		require.NotNil(t, search.Limit)
		assert.Equal(t, 10, search.Limit.Limit)
		assert.Equal(t, 5, search.Limit.Offset)
	})

	t.Run("WithSearchSelect", func(t *testing.T) {
		search := &CollectionSearchOp{}
		opt := WithSearchSelect(SelectID, SelectDocument, SelectScore)
		err := opt(search)
		assert.NoError(t, err)
		assert.Len(t, search.Select, 3)
		assert.Contains(t, search.Select, SelectID)
		assert.Contains(t, search.Select, SelectDocument)
		assert.Contains(t, search.Select, SelectScore)
	})
}

func TestSearchResult_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"ids": [["id1", "id2"], ["id3"]],
		"documents": [["doc1", "doc2"], ["doc3"]],
		"metadatas": [[{"key": "value1"}, {"key": "value2"}], [{"key": "value3"}]],
		"scores": [[0.95, 0.85], [0.75]]
	}`

	result := &SearchResultImpl{}
	err := json.Unmarshal([]byte(jsonData), result)
	require.NoError(t, err)

	assert.Len(t, result.IDLists, 2)
	assert.Equal(t, DocumentID("id1"), result.IDLists[0][0])
	assert.Equal(t, DocumentID("id2"), result.IDLists[0][1])
	assert.Equal(t, DocumentID("id3"), result.IDLists[1][0])

	assert.Len(t, result.DocumentsLists, 2)
	assert.Equal(t, "doc1", result.DocumentsLists[0][0].ContentString())

	assert.Len(t, result.MetadatasLists, 2)
	val, _ := result.MetadatasLists[0][0].Get("key")
	assert.Equal(t, "value1", val)

	assert.Len(t, result.ScoresLists, 2)
	assert.Equal(t, embeddings.Distance(0.95), result.ScoresLists[0][0])
	assert.Equal(t, embeddings.Distance(0.85), result.ScoresLists[0][1])
}
