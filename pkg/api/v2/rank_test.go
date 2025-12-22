//go:build basicv2 && !cloud

package v2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func TestValRank(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected string
	}{
		{
			name:     "positive value",
			value:    0.5,
			expected: `{"$val":0.5}`,
		},
		{
			name:     "negative value",
			value:    -1.0,
			expected: `{"$val":-1}`,
		},
		{
			name:     "zero",
			value:    0,
			expected: `{"$val":0}`,
		},
		{
			name:     "large value",
			value:    1000.0,
			expected: `{"$val":1000}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := Val(tt.value)
			data, err := val.MarshalJSON()
			require.NoError(t, err)
			require.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestKnnRank(t *testing.T) {
	tests := []struct {
		name     string
		rank     *KnnRank
		expected string
	}{
		{
			name:     "text query with defaults",
			rank:     NewKnnRank(KnnQueryText("machine learning")),
			expected: `{"$knn":{"query":"machine learning","key":"#embedding","limit":16}}`,
		},
		{
			name: "text query with custom limit",
			rank: NewKnnRank(
				KnnQueryText("deep learning"),
				WithKnnLimit(100),
			),
			expected: `{"$knn":{"query":"deep learning","key":"#embedding","limit":100}}`,
		},
		{
			name: "text query with custom key",
			rank: NewKnnRank(
				KnnQueryText("neural networks"),
				WithKnnKey(K("sparse_embedding")),
			),
			expected: `{"$knn":{"query":"neural networks","key":"sparse_embedding","limit":16}}`,
		},
		{
			name: "text query with default score",
			rank: NewKnnRank(
				KnnQueryText("AI research"),
				WithKnnDefault(10.0),
			),
			expected: `{"$knn":{"query":"AI research","key":"#embedding","limit":16,"default":10}}`,
		},
		{
			name: "text query with return_rank",
			rank: NewKnnRank(
				KnnQueryText("papers"),
				WithKnnReturnRank(),
			),
			expected: `{"$knn":{"query":"papers","key":"#embedding","limit":16,"return_rank":true}}`,
		},
		{
			name: "all options",
			rank: NewKnnRank(
				KnnQueryText("complete example"),
				WithKnnLimit(50),
				WithKnnKey(K("custom_field")),
				WithKnnDefault(100.0),
				WithKnnReturnRank(),
			),
			expected: `{"$knn":{"query":"complete example","key":"custom_field","limit":50,"default":100,"return_rank":true}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.rank.MarshalJSON()
			require.NoError(t, err)
			require.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestKnnRankWithVectors(t *testing.T) {
	t.Run("dense vector", func(t *testing.T) {
		// Create a KnnRank with a float32 slice directly
		knn := NewKnnRank(nil)
		knn.Query = []float32{0.1, 0.2, 0.3}

		data, err := knn.MarshalJSON()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		knnData := result["$knn"].(map[string]interface{})
		query := knnData["query"].([]interface{})
		require.Len(t, query, 3)
	})

	t.Run("sparse vector", func(t *testing.T) {
		sparseVector := embeddings.NewSparseVector(
			[]int{1, 5, 10},
			[]float32{0.5, 0.3, 0.8},
		)
		rank := NewKnnRank(
			KnnQuerySparseVector(sparseVector),
			WithKnnKey(K("sparse_embedding")),
		)
		data, err := rank.MarshalJSON()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		knn := result["$knn"].(map[string]interface{})
		query := knn["query"].(map[string]interface{})
		require.Contains(t, query, "indices")
		require.Contains(t, query, "values")
	})
}

func TestArithmeticOperations(t *testing.T) {
	tests := []struct {
		name     string
		rank     Rank
		expected string
	}{
		{
			name:     "addition with val",
			rank:     Val(1.0).Add(FloatOperand(2.0)),
			expected: `{"$sum":[{"$val":1},{"$val":2}]}`,
		},
		{
			name:     "subtraction with val",
			rank:     Val(5.0).Sub(FloatOperand(3.0)),
			expected: `{"$sub":{"left":{"$val":5},"right":{"$val":3}}}`,
		},
		{
			name:     "multiplication with val",
			rank:     Val(2.0).Multiply(FloatOperand(3.0)),
			expected: `{"$mul":[{"$val":2},{"$val":3}]}`,
		},
		{
			name:     "division with val",
			rank:     Val(10.0).Div(FloatOperand(2.0)),
			expected: `{"$div":{"left":{"$val":10},"right":{"$val":2}}}`,
		},
		{
			name:     "negation",
			rank:     Val(5.0).Negate(),
			expected: `{"$mul":[{"$val":-1},{"$val":5}]}`,
		},
		{
			name: "knn multiply by scalar",
			rank: NewKnnRank(KnnQueryText("test")).Multiply(FloatOperand(0.5)),
			expected: `{"$mul":[{"$knn":{"query":"test","key":"#embedding","limit":16}},{"$val":0.5}]}`,
		},
		{
			name: "knn add knn",
			rank: NewKnnRank(KnnQueryText("a")).Add(NewKnnRank(KnnQueryText("b"))),
			expected: `{"$sum":[{"$knn":{"query":"a","key":"#embedding","limit":16}},{"$knn":{"query":"b","key":"#embedding","limit":16}}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.rank.MarshalJSON()
			require.NoError(t, err)
			require.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestMathFunctions(t *testing.T) {
	tests := []struct {
		name     string
		rank     Rank
		expected string
	}{
		{
			name:     "abs",
			rank:     Val(-5.0).Abs(),
			expected: `{"$abs":{"$val":-5}}`,
		},
		{
			name:     "exp",
			rank:     Val(1.0).Exp(),
			expected: `{"$exp":{"$val":1}}`,
		},
		{
			name:     "log",
			rank:     Val(10.0).Log(),
			expected: `{"$log":{"$val":10}}`,
		},
		{
			name:     "max",
			rank:     Val(1.0).Max(FloatOperand(5.0)),
			expected: `{"$max":[{"$val":1},{"$val":5}]}`,
		},
		{
			name:     "min",
			rank:     Val(10.0).Min(FloatOperand(5.0)),
			expected: `{"$min":[{"$val":10},{"$val":5}]}`,
		},
		{
			name: "knn with exp",
			rank: NewKnnRank(KnnQueryText("test")).Exp(),
			expected: `{"$exp":{"$knn":{"query":"test","key":"#embedding","limit":16}}}`,
		},
		{
			name: "knn with min and max (clamping)",
			rank: NewKnnRank(KnnQueryText("test")).Min(FloatOperand(0.0)).Max(FloatOperand(1.0)),
			expected: `{"$max":[{"$min":[{"$knn":{"query":"test","key":"#embedding","limit":16}},{"$val":0}]},{"$val":1}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.rank.MarshalJSON()
			require.NoError(t, err)
			require.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestComplexExpressions(t *testing.T) {
	t.Run("weighted combination", func(t *testing.T) {
		// weighted_combo = knn1 * 0.7 + knn2 * 0.3
		rank := NewKnnRank(KnnQueryText("machine learning")).
			Multiply(FloatOperand(0.7)).
			Add(
				NewKnnRank(
					KnnQueryText("machine learning"),
					WithKnnKey(K("sparse_embedding")),
				).Multiply(FloatOperand(0.3)),
			)

		data, err := rank.MarshalJSON()
		require.NoError(t, err)

		// Verify structure
		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Contains(t, result, "$sum")
	})

	t.Run("log compression", func(t *testing.T) {
		// (knn + 1).log()
		rank := NewKnnRank(KnnQueryText("deep learning")).
			Add(FloatOperand(1)).
			Log()

		data, err := rank.MarshalJSON()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Contains(t, result, "$log")
	})

	t.Run("exponential with clamping", func(t *testing.T) {
		// knn.exp().min(0.0)
		rank := NewKnnRank(KnnQueryText("AI")).
			Exp().
			Min(FloatOperand(0.0))

		data, err := rank.MarshalJSON()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Contains(t, result, "$min")
	})
}

func TestRrfRank(t *testing.T) {
	t.Run("basic rrf", func(t *testing.T) {
		rrf, err := NewRrfRank(
			WithRffRanks(
				NewKnnRank(KnnQueryText("query1"), WithKnnReturnRank()).WithWeight(1.0),
				NewKnnRank(KnnQueryText("query2"), WithKnnReturnRank()).WithWeight(1.0),
			),
			WithRffK(60),
		)
		require.NoError(t, err)

		data, err := rrf.MarshalJSON()
		require.NoError(t, err)

		// RRF produces: -sum(w/(k+rank))
		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Contains(t, result, "$mul") // negation creates $mul with -1
	})

	t.Run("rrf with custom k", func(t *testing.T) {
		rrf, err := NewRrfRank(
			WithRffRanks(
				NewKnnRank(KnnQueryText("test")).WithWeight(1.0),
			),
			WithRffK(100),
		)
		require.NoError(t, err)
		require.Equal(t, 100, rrf.K)
	})

	t.Run("rrf with normalization", func(t *testing.T) {
		rrf, err := NewRrfRank(
			WithRffRanks(
				NewKnnRank(KnnQueryText("a")).WithWeight(3.0),
				NewKnnRank(KnnQueryText("b")).WithWeight(1.0),
			),
			WithRffNormalize(),
		)
		require.NoError(t, err)
		require.True(t, rrf.Normalize)

		// Should serialize without error
		_, err = rrf.MarshalJSON()
		require.NoError(t, err)
	})

	t.Run("rrf requires at least one rank", func(t *testing.T) {
		rrf, err := NewRrfRank()
		require.NoError(t, err)

		_, err = rrf.MarshalJSON()
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one rank")
	})

	t.Run("rrf k must be positive", func(t *testing.T) {
		_, err := NewRrfRank(WithRffK(0))
		require.Error(t, err)
		require.Contains(t, err.Error(), "must be > 0")
	})
}

func TestRankWithWeight(t *testing.T) {
	t.Run("knn with weight", func(t *testing.T) {
		knn := NewKnnRank(KnnQueryText("test"))
		rw := knn.WithWeight(0.5)

		require.Equal(t, knn, rw.Rank)
		require.Equal(t, 0.5, rw.Weight)
	})

	t.Run("expression with weight", func(t *testing.T) {
		expr := NewKnnRank(KnnQueryText("test")).Multiply(FloatOperand(0.8))
		rw := expr.WithWeight(0.3)

		require.Equal(t, 0.3, rw.Weight)
	})
}

func TestOperandConversion(t *testing.T) {
	t.Run("int operand", func(t *testing.T) {
		rank := Val(1.0).Add(IntOperand(5))
		data, err := rank.MarshalJSON()
		require.NoError(t, err)
		require.JSONEq(t, `{"$sum":[{"$val":1},{"$val":5}]}`, string(data))
	})

	t.Run("float operand", func(t *testing.T) {
		rank := Val(1.0).Multiply(FloatOperand(2.5))
		data, err := rank.MarshalJSON()
		require.NoError(t, err)
		require.JSONEq(t, `{"$mul":[{"$val":1},{"$val":2.5}]}`, string(data))
	})
}

func TestKnnOptionValidation(t *testing.T) {
	t.Run("limit must be >= 1", func(t *testing.T) {
		knn := &KnnRank{}
		err := WithKnnLimit(0)(knn)
		require.Error(t, err)
		require.Contains(t, err.Error(), "must be >= 1")
	})

	t.Run("valid limit", func(t *testing.T) {
		knn := &KnnRank{}
		err := WithKnnLimit(100)(knn)
		require.NoError(t, err)
		require.Equal(t, 100, knn.Limit)
	})
}
