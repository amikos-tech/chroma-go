//go:build basicv2 && !cloud

package v2

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type unsupportedOperand struct{}

func (unsupportedOperand) IsOperand() {}

// mustNewKnnRank is a test helper that panics if NewKnnRank returns an error
func mustNewKnnRank(t *testing.T, query KnnQueryOption, knnOptions ...KnnOption) *KnnRank {
	t.Helper()
	knn, err := NewKnnRank(query, knnOptions...)
	require.NoError(t, err)
	return knn
}

func mustNewRrfRank(t *testing.T, opts ...RrfOption) *RrfRank {
	t.Helper()
	rrf, err := NewRrfRank(opts...)
	require.NoError(t, err)
	return rrf
}

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
		makeRank func(t *testing.T) *KnnRank
		expected string
	}{
		{
			name: "text query with defaults",
			makeRank: func(t *testing.T) *KnnRank {
				return mustNewKnnRank(t, KnnQueryText("machine learning"))
			},
			expected: `{"$knn":{"query":"machine learning","key":"#embedding","limit":16}}`,
		},
		{
			name: "text query with custom limit",
			makeRank: func(t *testing.T) *KnnRank {
				return mustNewKnnRank(t, KnnQueryText("deep learning"), WithKnnLimit(100))
			},
			expected: `{"$knn":{"query":"deep learning","key":"#embedding","limit":100}}`,
		},
		{
			name: "text query with custom key",
			makeRank: func(t *testing.T) *KnnRank {
				return mustNewKnnRank(t, KnnQueryText("neural networks"), WithKnnKey(K("sparse_embedding")))
			},
			expected: `{"$knn":{"query":"neural networks","key":"sparse_embedding","limit":16}}`,
		},
		{
			name: "text query with default score",
			makeRank: func(t *testing.T) *KnnRank {
				return mustNewKnnRank(t, KnnQueryText("AI research"), WithKnnDefault(10.0))
			},
			expected: `{"$knn":{"query":"AI research","key":"#embedding","limit":16,"default":10}}`,
		},
		{
			name: "text query with return_rank",
			makeRank: func(t *testing.T) *KnnRank {
				return mustNewKnnRank(t, KnnQueryText("papers"), WithKnnReturnRank())
			},
			expected: `{"$knn":{"query":"papers","key":"#embedding","limit":16,"return_rank":true}}`,
		},
		{
			name: "all options",
			makeRank: func(t *testing.T) *KnnRank {
				return mustNewKnnRank(t,
					KnnQueryText("complete example"),
					WithKnnLimit(50),
					WithKnnKey(K("custom_field")),
					WithKnnDefault(100.0),
					WithKnnReturnRank(),
				)
			},
			expected: `{"$knn":{"query":"complete example","key":"custom_field","limit":50,"default":100,"return_rank":true}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rank := tt.makeRank(t)
			data, err := rank.MarshalJSON()
			require.NoError(t, err)
			require.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestKnnRankWithVectors(t *testing.T) {
	t.Run("dense vector", func(t *testing.T) {
		// Create a KnnRank with a float32 slice directly
		knn := mustNewKnnRank(t, nil)
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
		sparseVector, err := embeddings.NewSparseVector(
			[]int{1, 5, 10},
			[]float32{0.5, 0.3, 0.8},
		)
		require.NoError(t, err)
		rank := mustNewKnnRank(t,
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
		makeRank func(t *testing.T) Rank
		expected string
	}{
		{
			name:     "addition with val",
			makeRank: func(_ *testing.T) Rank { return Val(1.0).Add(FloatOperand(2.0)) },
			expected: `{"$sum":[{"$val":1},{"$val":2}]}`,
		},
		{
			name:     "subtraction with val",
			makeRank: func(_ *testing.T) Rank { return Val(5.0).Sub(FloatOperand(3.0)) },
			expected: `{"$sub":{"left":{"$val":5},"right":{"$val":3}}}`,
		},
		{
			name:     "multiplication with val",
			makeRank: func(_ *testing.T) Rank { return Val(2.0).Multiply(FloatOperand(3.0)) },
			expected: `{"$mul":[{"$val":2},{"$val":3}]}`,
		},
		{
			name:     "division with val",
			makeRank: func(_ *testing.T) Rank { return Val(10.0).Div(FloatOperand(2.0)) },
			expected: `{"$div":{"left":{"$val":10},"right":{"$val":2}}}`,
		},
		{
			name:     "negation",
			makeRank: func(_ *testing.T) Rank { return Val(5.0).Negate() },
			expected: `{"$mul":[{"$val":-1},{"$val":5}]}`,
		},
		{
			name: "knn multiply by scalar",
			makeRank: func(t *testing.T) Rank {
				return mustNewKnnRank(t, KnnQueryText("test")).Multiply(FloatOperand(0.5))
			},
			expected: `{"$mul":[{"$knn":{"query":"test","key":"#embedding","limit":16}},{"$val":0.5}]}`,
		},
		{
			name: "knn add knn",
			makeRank: func(t *testing.T) Rank {
				return mustNewKnnRank(t, KnnQueryText("a")).Add(mustNewKnnRank(t, KnnQueryText("b")))
			},
			expected: `{"$sum":[{"$knn":{"query":"a","key":"#embedding","limit":16}},{"$knn":{"query":"b","key":"#embedding","limit":16}}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rank := tt.makeRank(t)
			data, err := rank.MarshalJSON()
			require.NoError(t, err)
			require.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestMathFunctions(t *testing.T) {
	tests := []struct {
		name     string
		makeRank func(t *testing.T) Rank
		expected string
	}{
		{
			name:     "abs",
			makeRank: func(_ *testing.T) Rank { return Val(-5.0).Abs() },
			expected: `{"$abs":{"$val":-5}}`,
		},
		{
			name:     "exp",
			makeRank: func(_ *testing.T) Rank { return Val(1.0).Exp() },
			expected: `{"$exp":{"$val":1}}`,
		},
		{
			name:     "log",
			makeRank: func(_ *testing.T) Rank { return Val(10.0).Log() },
			expected: `{"$log":{"$val":10}}`,
		},
		{
			name:     "max",
			makeRank: func(_ *testing.T) Rank { return Val(1.0).Max(FloatOperand(5.0)) },
			expected: `{"$max":[{"$val":1},{"$val":5}]}`,
		},
		{
			name:     "min",
			makeRank: func(_ *testing.T) Rank { return Val(10.0).Min(FloatOperand(5.0)) },
			expected: `{"$min":[{"$val":10},{"$val":5}]}`,
		},
		{
			name: "knn with exp",
			makeRank: func(t *testing.T) Rank {
				return mustNewKnnRank(t, KnnQueryText("test")).Exp()
			},
			expected: `{"$exp":{"$knn":{"query":"test","key":"#embedding","limit":16}}}`,
		},
		{
			name: "knn with min and max (clamping)",
			makeRank: func(t *testing.T) Rank {
				return mustNewKnnRank(t, KnnQueryText("test")).Min(FloatOperand(0.0)).Max(FloatOperand(1.0))
			},
			expected: `{"$max":[{"$min":[{"$knn":{"query":"test","key":"#embedding","limit":16}},{"$val":0}]},{"$val":1}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rank := tt.makeRank(t)
			data, err := rank.MarshalJSON()
			require.NoError(t, err)
			require.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestDivisionByZero(t *testing.T) {
	t.Run("literal zero denominator", func(t *testing.T) {
		rank := Val(10.0).Div(Val(0.0))
		_, err := rank.MarshalJSON()
		require.Error(t, err)
		require.Contains(t, err.Error(), "division by zero")
	})

	t.Run("float operand zero denominator", func(t *testing.T) {
		rank := Val(10.0).Div(FloatOperand(0.0))
		_, err := rank.MarshalJSON()
		require.Error(t, err)
		require.Contains(t, err.Error(), "division by zero")
	})

	t.Run("int operand zero denominator", func(t *testing.T) {
		rank := Val(10.0).Div(IntOperand(0))
		_, err := rank.MarshalJSON()
		require.Error(t, err)
		require.Contains(t, err.Error(), "division by zero")
	})

	t.Run("non-zero denominator succeeds", func(t *testing.T) {
		rank := Val(10.0).Div(Val(2.0))
		data, err := rank.MarshalJSON()
		require.NoError(t, err)
		require.JSONEq(t, `{"$div":{"left":{"$val":10},"right":{"$val":2}}}`, string(data))
	})
}

func TestUnknownRankError(t *testing.T) {
	t.Run("unknown rank errors on marshal", func(t *testing.T) {
		unknown := &UnknownRank{}
		_, err := unknown.MarshalJSON()
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown operand type")
	})

	t.Run("unknown rank in expression errors on marshal", func(t *testing.T) {
		// UnknownRank embedded in an expression should still error
		rank := Val(10.0).Add(&UnknownRank{})
		_, err := rank.MarshalJSON()
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown operand type")
	})
}

func TestRankArithmeticTypedNilOperandMarshal(t *testing.T) {
	var operand *KnnRank

	tests := []struct {
		name  string
		build func() Rank
	}{
		{name: "Add", build: func() Rank { return Val(1).Add(operand) }},
		{name: "Multiply", build: func() Rank { return Val(1).Multiply(operand) }},
		{name: "Sub", build: func() Rank { return Val(1).Sub(operand) }},
		{name: "Div", build: func() Rank { return Val(1).Div(operand) }},
		{name: "Max", build: func() Rank { return Val(1).Max(operand) }},
		{name: "Min", build: func() Rank { return Val(1).Min(operand) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rank := tt.build()
			var err error
			require.NotPanics(t, func() {
				_, err = rank.MarshalJSON()
			})
			require.ErrorIs(t, err, ErrNilRank)
		})
	}
}

func TestRankTypedNilReceiverMarshal(t *testing.T) {
	var knn *KnnRank
	var rankReceiver Rank = knn

	tests := []struct {
		name  string
		build func() Rank
	}{
		{name: "Add through Rank interface", build: func() Rank { return rankReceiver.Add(Val(1)) }},
		{name: "Multiply", build: func() Rank { return knn.Multiply(Val(1)) }},
		{name: "Sub", build: func() Rank { return knn.Sub(Val(1)) }},
		{name: "Div", build: func() Rank { return knn.Div(Val(1)) }},
		{name: "Max", build: func() Rank { return knn.Max(Val(1)) }},
		{name: "Min", build: func() Rank { return knn.Min(Val(1)) }},
		{name: "Negate", build: func() Rank { return knn.Negate() }},
		{name: "Abs", build: func() Rank { return knn.Abs() }},
		{name: "Exp", build: func() Rank { return knn.Exp() }},
		{name: "Log", build: func() Rank { return knn.Log() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rank Rank
			require.NotPanics(t, func() {
				rank = tt.build()
			})

			var err error
			require.NotPanics(t, func() {
				_, err = rank.MarshalJSON()
			})
			require.ErrorIs(t, err, ErrNilRank)
		})
	}
}

func TestSelfFlatteningRankTypedNilReceiverMarshal(t *testing.T) {
	var sum *SumRank
	var mul *MulRank
	var max *MaxRank
	var min *MinRank
	var abs *AbsRank

	tests := []struct {
		name  string
		build func() Rank
	}{
		{name: "SumRank.Add", build: func() Rank { return sum.Add(Val(1)) }},
		{name: "MulRank.Multiply", build: func() Rank { return mul.Multiply(Val(1)) }},
		{name: "MaxRank.Max", build: func() Rank { return max.Max(Val(1)) }},
		{name: "MinRank.Min", build: func() Rank { return min.Min(Val(1)) }},
		{name: "AbsRank.Abs", build: func() Rank { return abs.Abs() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rank Rank
			require.NotPanics(t, func() {
				rank = tt.build()
			})

			var err error
			require.NotPanics(t, func() {
				_, err = rank.MarshalJSON()
			})
			require.ErrorIs(t, err, ErrNilRank)
		})
	}
}

func TestCompositeRankNilChildMarshal(t *testing.T) {
	var child *KnnRank

	tests := []struct {
		name string
		rank Rank
	}{
		{name: "SumRank first of two", rank: &SumRank{ranks: []Rank{child, Val(1)}}},
		{name: "SumRank second of two", rank: &SumRank{ranks: []Rank{Val(1), child}}},
		{name: "SumRank middle of three", rank: &SumRank{ranks: []Rank{Val(1), child, Val(2)}}},
		{name: "SubRank left", rank: &SubRank{left: child, right: Val(1)}},
		{name: "SubRank right", rank: &SubRank{left: Val(1), right: child}},
		{name: "MulRank first of two", rank: &MulRank{ranks: []Rank{child, Val(1)}}},
		{name: "MulRank second of two", rank: &MulRank{ranks: []Rank{Val(1), child}}},
		{name: "MulRank middle of three", rank: &MulRank{ranks: []Rank{Val(1), child, Val(2)}}},
		{name: "DivRank left", rank: &DivRank{left: child, right: Val(1)}},
		{name: "DivRank right", rank: &DivRank{left: Val(1), right: child}},
		{name: "AbsRank", rank: &AbsRank{rank: child}},
		{name: "ExpRank", rank: &ExpRank{rank: child}},
		{name: "LogRank", rank: &LogRank{rank: child}},
		{name: "MaxRank first of two", rank: &MaxRank{ranks: []Rank{child, Val(1)}}},
		{name: "MaxRank second of two", rank: &MaxRank{ranks: []Rank{Val(1), child}}},
		{name: "MaxRank middle of three", rank: &MaxRank{ranks: []Rank{Val(1), child, Val(2)}}},
		{name: "MinRank first of two", rank: &MinRank{ranks: []Rank{child, Val(1)}}},
		{name: "MinRank second of two", rank: &MinRank{ranks: []Rank{Val(1), child}}},
		{name: "MinRank middle of three", rank: &MinRank{ranks: []Rank{Val(1), child, Val(2)}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			require.NotPanics(t, func() {
				_, err = tt.rank.MarshalJSON()
			})
			require.ErrorIs(t, err, ErrNilRank)
		})
	}
}

func TestComplexExpressions(t *testing.T) {
	t.Run("weighted combination", func(t *testing.T) {
		// weighted_combo = knn1 * 0.7 + knn2 * 0.3
		knn1 := mustNewKnnRank(t, KnnQueryText("machine learning"))
		knn2 := mustNewKnnRank(t, KnnQueryText("machine learning"), WithKnnKey(K("sparse_embedding")))
		rank := knn1.Multiply(FloatOperand(0.7)).Add(knn2.Multiply(FloatOperand(0.3)))

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
		knn := mustNewKnnRank(t, KnnQueryText("deep learning"))
		rank := knn.Add(FloatOperand(1)).Log()

		data, err := rank.MarshalJSON()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Contains(t, result, "$log")
	})

	t.Run("exponential with clamping", func(t *testing.T) {
		// knn.exp().min(0.0)
		knn := mustNewKnnRank(t, KnnQueryText("AI"))
		rank := knn.Exp().Min(FloatOperand(0.0))

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
		knn1 := mustNewKnnRank(t, KnnQueryText("query1"), WithKnnReturnRank())
		knn2 := mustNewKnnRank(t, KnnQueryText("query2"), WithKnnReturnRank())
		rrf, err := NewRrfRank(
			WithRrfRanks(
				knn1.WithWeight(1.0),
				knn2.WithWeight(1.0),
			),
			WithRrfK(60),
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
		knn := mustNewKnnRank(t, KnnQueryText("test"))
		rrf, err := NewRrfRank(
			WithRrfRanks(
				knn.WithWeight(1.0),
			),
			WithRrfK(100),
		)
		require.NoError(t, err)
		require.Equal(t, 100, rrf.K)
	})

	t.Run("rrf with normalization", func(t *testing.T) {
		knnA := mustNewKnnRank(t, KnnQueryText("a"))
		knnB := mustNewKnnRank(t, KnnQueryText("b"))
		rrf, err := NewRrfRank(
			WithRrfRanks(
				knnA.WithWeight(3.0),
				knnB.WithWeight(1.0),
			),
			WithRrfNormalize(),
		)
		require.NoError(t, err)
		require.True(t, rrf.Normalize)

		// Should serialize without error
		_, err = rrf.MarshalJSON()
		require.NoError(t, err)
	})

	t.Run("rrf requires at least one rank", func(t *testing.T) {
		_, err := NewRrfRank()
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one rank")
	})

	t.Run("rrf k must be positive", func(t *testing.T) {
		_, err := NewRrfRank(WithRrfK(0))
		require.Error(t, err)
		require.Contains(t, err.Error(), "must be >= 1")
	})

	t.Run("rrf rejects negative weights", func(t *testing.T) {
		knn := mustNewKnnRank(t, KnnQueryText("test"))
		_, err := NewRrfRank(
			WithRrfRanks(knn.WithWeight(-0.5)),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "negative weight")
	})

	t.Run("rrf rejects NaN weights", func(t *testing.T) {
		knn := mustNewKnnRank(t, KnnQueryText("test"))
		_, err := NewRrfRank(
			WithRrfRanks(knn.WithWeight(math.NaN())),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid weight")
	})

	t.Run("rrf rejects Inf weights", func(t *testing.T) {
		knn := mustNewKnnRank(t, KnnQueryText("test"))
		_, err := NewRrfRank(
			WithRrfRanks(knn.WithWeight(math.Inf(1))),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid weight")
	})

	t.Run("rrf detects weight sum overflow on normalize", func(t *testing.T) {
		knn1 := mustNewKnnRank(t, KnnQueryText("a"))
		knn2 := mustNewKnnRank(t, KnnQueryText("b"))
		rrf, err := NewRrfRank(
			WithRrfRanks(
				knn1.WithWeight(math.MaxFloat64),
				knn2.WithWeight(math.MaxFloat64),
			),
			WithRrfNormalize(),
		)
		require.NoError(t, err)
		_, err = rrf.MarshalJSON()
		require.Error(t, err)
		require.Contains(t, err.Error(), "overflowed")
	})

	for _, tt := range []struct {
		name string
		rank Rank
	}{
		{name: "nil rank", rank: nil},
		{name: "typed nil rank", rank: (*KnnRank)(nil)},
	} {
		t.Run("rrf rejects "+tt.name, func(t *testing.T) {
			_, err := NewRrfRank(
				WithRrfRanks(RankWithWeight{Rank: tt.rank, Weight: 1}),
			)
			require.ErrorIs(t, err, ErrNilRank)
			require.Contains(t, err.Error(), "rank 0")
		})
	}

	t.Run("direct invalid rrf is revalidated on marshal", func(t *testing.T) {
		var rank *KnnRank
		rrf := &RrfRank{
			K: 60,
			Ranks: []RankWithWeight{
				{Rank: rank, Weight: 1},
			},
		}

		_, err := rrf.MarshalJSON()
		require.ErrorIs(t, err, ErrNilRank)
		require.Contains(t, err.Error(), "rank 0")
	})
}

func TestRrfRankArithmetic(t *testing.T) {
	knn := mustNewKnnRank(t, KnnQueryText("test"), WithKnnReturnRank())
	rrf := mustNewRrfRank(t, WithRrfRanks(knn.WithWeight(1.0)), WithRrfK(60))

	rrfJSON, err := rrf.MarshalJSON()
	require.NoError(t, err)
	rrfStr := string(rrfJSON)

	tests := []struct {
		name     string
		apply    func(r *RrfRank) Rank
		expected string
	}{
		{
			name:     "Multiply",
			apply:    func(r *RrfRank) Rank { return r.Multiply(FloatOperand(2.0)) },
			expected: `{"$mul":[` + rrfStr + `,{"$val":2}]}`,
		},
		{
			name:     "Sub",
			apply:    func(r *RrfRank) Rank { return r.Sub(FloatOperand(1.0)) },
			expected: `{"$sub":{"left":` + rrfStr + `,"right":{"$val":1}}}`,
		},
		{
			name:     "Add",
			apply:    func(r *RrfRank) Rank { return r.Add(FloatOperand(3.0)) },
			expected: `{"$sum":[` + rrfStr + `,{"$val":3}]}`,
		},
		{
			name:     "Div",
			apply:    func(r *RrfRank) Rank { return r.Div(FloatOperand(2.0)) },
			expected: `{"$div":{"left":` + rrfStr + `,"right":{"$val":2}}}`,
		},
		{
			name:     "Negate",
			apply:    func(r *RrfRank) Rank { return r.Negate() },
			expected: `{"$mul":[{"$val":-1},` + rrfStr + `]}`,
		},
		{
			name:     "Abs",
			apply:    func(r *RrfRank) Rank { return r.Abs() },
			expected: `{"$abs":` + rrfStr + `}`,
		},
		{
			name:     "Exp",
			apply:    func(r *RrfRank) Rank { return r.Exp() },
			expected: `{"$exp":` + rrfStr + `}`,
		},
		{
			name:     "Log",
			apply:    func(r *RrfRank) Rank { return r.Log() },
			expected: `{"$log":` + rrfStr + `}`,
		},
		{
			name:     "Max",
			apply:    func(r *RrfRank) Rank { return r.Max(FloatOperand(0.0)) },
			expected: `{"$max":[` + rrfStr + `,{"$val":0}]}`,
		},
		{
			name:     "Min",
			apply:    func(r *RrfRank) Rank { return r.Min(FloatOperand(1.0)) },
			expected: `{"$min":[` + rrfStr + `,{"$val":1}]}`,
		},
		{
			name:     "Add_IntOperand",
			apply:    func(r *RrfRank) Rank { return r.Add(IntOperand(7)) },
			expected: `{"$sum":[` + rrfStr + `,{"$val":7}]}`,
		},
		{
			name:     "Multiply_RankOperand",
			apply:    func(r *RrfRank) Rank { return r.Multiply(Val(5).Abs()) },
			expected: `{"$mul":[` + rrfStr + `,{"$abs":{"$val":5}}]}`,
		},
		{
			name: "chained Add then Log",
			apply: func(r *RrfRank) Rank {
				return r.Add(FloatOperand(1)).Log()
			},
			expected: `{"$log":{"$sum":[` + rrfStr + `,{"$val":1}]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalJSON, err := rrf.MarshalJSON()
			require.NoError(t, err)

			result := tt.apply(rrf)

			resultJSON, err := result.MarshalJSON()
			require.NoError(t, err)
			require.JSONEq(t, tt.expected, string(resultJSON))

			// Receiver must be unchanged
			afterJSON, err := rrf.MarshalJSON()
			require.NoError(t, err)
			require.Equal(t, string(originalJSON), string(afterJSON), "receiver was mutated")
		})
	}

	t.Run("wrappers remain independent across sequential calls", func(t *testing.T) {
		a := rrf.Add(FloatOperand(1))
		b := rrf.Multiply(FloatOperand(2))
		c := rrf.Log()

		bJSON, err := b.MarshalJSON()
		require.NoError(t, err)
		cJSON, err := c.MarshalJSON()
		require.NoError(t, err)
		aJSON, err := a.MarshalJSON()
		require.NoError(t, err)

		require.JSONEq(t, `{"$sum":[`+rrfStr+`,{"$val":1}]}`, string(aJSON))
		require.JSONEq(t, `{"$mul":[`+rrfStr+`,{"$val":2}]}`, string(bJSON))
		require.JSONEq(t, `{"$log":`+rrfStr+`}`, string(cJSON))
	})
}

// Regression test: LogRank.Log() must build a nested log expression.
// log(log(x)) != log(x), so returning the receiver silently drops the outer Log.
func TestLogRankLogComposition(t *testing.T) {
	inner := Val(10.0).Log()
	nested := inner.Log()

	require.False(t, nested == inner, "LogRank.Log() must return a new Rank, not the receiver")

	nestedJSON, err := nested.MarshalJSON()
	require.NoError(t, err)
	require.JSONEq(t, `{"$log":{"$log":{"$val":10}}}`, string(nestedJSON))
}

func TestRankWithWeight(t *testing.T) {
	t.Run("knn with weight", func(t *testing.T) {
		knn := mustNewKnnRank(t, KnnQueryText("test"))
		rw := knn.WithWeight(0.5)

		require.Equal(t, knn, rw.Rank)
		require.Equal(t, 0.5, rw.Weight)
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

	t.Run("untyped nil operand becomes zero", func(t *testing.T) {
		var operand Operand
		rank := Val(1.0).Add(operand)
		data, err := rank.MarshalJSON()
		require.NoError(t, err)
		require.JSONEq(t, `{"$sum":[{"$val":1},{"$val":0}]}`, string(data))
	})

	t.Run("unsupported operand remains unknown", func(t *testing.T) {
		rank := Val(1.0).Add(unsupportedOperand{})
		_, err := rank.MarshalJSON()
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrNilRank)
		require.Contains(t, err.Error(), "unknown operand type")
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

func TestMaxExpressionDepthConstant(t *testing.T) {
	// Verify the constant is defined and has a reasonable value
	require.Greater(t, MaxExpressionDepth, 0)
	require.LessOrEqual(t, MaxExpressionDepth, 1000)
}

func TestRankMarshalExpressionDepthGuard(t *testing.T) {
	tests := []struct {
		name          string
		build         func(Rank) Rank
		acceptedDepth int
		rejectedDepth int
		wantContext   string
	}{
		{
			name:          "sum",
			build:         func(rank Rank) Rank { return rank.Add(Val(1)) },
			acceptedDepth: 99,
			rejectedDepth: 100,
		},
		{
			name:          "sub",
			build:         func(rank Rank) Rank { return rank.Sub(Val(1)) },
			acceptedDepth: 99,
			rejectedDepth: 100,
		},
		{
			name:          "sub-right",
			build:         func(rank Rank) Rank { return Val(1).Sub(rank) },
			acceptedDepth: 99,
			rejectedDepth: 100,
		},
		{
			name:          "mul",
			build:         func(rank Rank) Rank { return rank.Multiply(Val(1)) },
			acceptedDepth: 99,
			rejectedDepth: 100,
		},
		{
			name:          "div",
			build:         func(rank Rank) Rank { return rank.Div(Val(1)) },
			acceptedDepth: 99,
			rejectedDepth: 100,
		},
		{
			name:          "div-right",
			build:         func(rank Rank) Rank { return Val(1).Div(rank) },
			acceptedDepth: 99,
			rejectedDepth: 100,
		},
		{
			name:          "abs",
			build:         func(rank Rank) Rank { return rank.Abs() },
			acceptedDepth: 99,
			rejectedDepth: 100,
		},
		{
			name:          "exp",
			build:         func(rank Rank) Rank { return rank.Exp() },
			acceptedDepth: 99,
			rejectedDepth: 100,
		},
		{
			name:          "log",
			build:         func(rank Rank) Rank { return rank.Log() },
			acceptedDepth: 99,
			rejectedDepth: 100,
		},
		{
			name:          "max",
			build:         func(rank Rank) Rank { return rank.Max(Val(1)) },
			acceptedDepth: 99,
			rejectedDepth: 100,
		},
		{
			name:          "min",
			build:         func(rank Rank) Rank { return rank.Min(Val(1)) },
			acceptedDepth: 99,
			rejectedDepth: 100,
		},
		{
			name: "rrf",
			build: func(rank Rank) Rank {
				return mustNewRrfRank(t, WithRrfRanks(RankWithWeight{Rank: rank, Weight: 1}))
			},
			acceptedDepth: 96,
			rejectedDepth: 97,
			wantContext:   "cannot marshal RrfRank expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("accepts deepest valid child", func(t *testing.T) {
				var inner Rank = Val(0)
				for i := 0; i < tt.acceptedDepth; i++ {
					inner = inner.Log()
				}
				rank := tt.build(inner)

				var data []byte
				var err error
				require.NotPanics(t, func() {
					data, err = rank.MarshalJSON()
				})
				require.NoError(t, err)
				require.NotEmpty(t, data)
			})

			t.Run("rejects next child depth", func(t *testing.T) {
				var inner Rank = Val(0)
				for i := 0; i < tt.rejectedDepth; i++ {
					inner = inner.Log()
				}
				rank := tt.build(inner)

				var err error
				require.NotPanics(t, func() {
					_, err = rank.MarshalJSON()
				})
				require.Error(t, err)
				require.Contains(t, err.Error(), fmt.Sprintf("rank expression exceeds maximum depth of %d", MaxExpressionDepth))
				if tt.wantContext != "" {
					require.Contains(t, err.Error(), tt.wantContext)
				}
			})
		})
	}
}

// TestMarshalRankFallbackForNestedNonDepthAwareChild exercises the
// depthAwareRank fallback path (rank.go marshalRank) with a caller-supplied
// Rank nested inside a built-in composite, not just used standalone. This is
// what the depthAwareRank doc comment describes: a type that doesn't
// implement depthAwareRank still marshals correctly via its own
// MarshalJSON() when it's a child of a depth-tracked expression.
func TestMarshalRankFallbackForNestedNonDepthAwareChild(t *testing.T) {
	mr := mapBackedRank{"k": "v"}
	rank := Val(0).Add(mr)

	data, err := rank.MarshalJSON()
	require.NoError(t, err)

	var result map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &result))
	require.Contains(t, result, "$sum")

	var terms []json.RawMessage
	require.NoError(t, json.Unmarshal(result["$sum"], &terms))
	require.Len(t, terms, 2)

	mrData, err := mr.MarshalJSON()
	require.NoError(t, err)
	require.JSONEq(t, string(mrData), string(terms[1]))
}
