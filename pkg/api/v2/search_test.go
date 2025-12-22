//go:build basicv2 && !cloud

package v2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// mustKnnRank is a test helper that creates a KnnRank or fails the test
func mustKnnRank(t *testing.T, query KnnQueryOption, knnOptions ...KnnOption) *KnnRank {
	t.Helper()
	knn, err := NewKnnRank(query, knnOptions...)
	if err != nil {
		t.Fatalf("mustKnnRank: %v", err)
	}
	return knn
}

func TestSearchPage(t *testing.T) {
	tests := []struct {
		name        string
		opts        []PageOpts
		expected    *SearchPage
		shouldError bool
	}{
		{
			name:     "limit only",
			opts:     []PageOpts{WithLimit(10)},
			expected: &SearchPage{Limit: 10},
		},
		{
			name:     "offset only",
			opts:     []PageOpts{WithOffset(5)},
			expected: &SearchPage{Offset: 5},
		},
		{
			name:     "limit and offset",
			opts:     []PageOpts{WithLimit(20), WithOffset(10)},
			expected: &SearchPage{Limit: 20, Offset: 10},
		},
		{
			name:        "invalid limit",
			opts:        []PageOpts{WithLimit(0)},
			shouldError: true,
		},
		{
			name:        "negative offset",
			opts:        []PageOpts{WithOffset(-1)},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := &SearchPage{}
			var err error
			for _, opt := range tt.opts {
				if e := opt(page); e != nil {
					err = e
					break
				}
			}

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected.Limit, page.Limit)
				require.Equal(t, tt.expected.Offset, page.Offset)
			}
		})
	}
}

func TestSearchSelect(t *testing.T) {
	t.Run("select standard keys", func(t *testing.T) {
		req := &SearchRequest{}
		err := WithSelect(KDocument, KScore, KEmbedding)(req)
		require.NoError(t, err)
		require.Len(t, req.Select.Keys, 3)
		require.Contains(t, req.Select.Keys, KDocument)
		require.Contains(t, req.Select.Keys, KScore)
		require.Contains(t, req.Select.Keys, KEmbedding)
	})

	t.Run("select custom keys", func(t *testing.T) {
		req := &SearchRequest{}
		err := WithSelect(K("title"), K("author"))(req)
		require.NoError(t, err)
		require.Len(t, req.Select.Keys, 2)
	})

	t.Run("select all", func(t *testing.T) {
		req := &SearchRequest{}
		err := WithSelectAll()(req)
		require.NoError(t, err)
		require.Len(t, req.Select.Keys, 5)
		require.Contains(t, req.Select.Keys, KID)
		require.Contains(t, req.Select.Keys, KDocument)
		require.Contains(t, req.Select.Keys, KEmbedding)
		require.Contains(t, req.Select.Keys, KMetadata)
		require.Contains(t, req.Select.Keys, KScore)
	})

	t.Run("append to existing select", func(t *testing.T) {
		req := &SearchRequest{}
		_ = WithSelect(KDocument)(req)
		_ = WithSelect(K("custom"))(req)
		require.Len(t, req.Select.Keys, 2)
	})
}

func TestSearchFilter(t *testing.T) {
	t.Run("with where clause", func(t *testing.T) {
		req := &SearchRequest{}
		err := WithFilter(EqString("status", "active"))(req)
		require.NoError(t, err)
		require.NotNil(t, req.Filter)
		require.NotNil(t, req.Filter.Where)
	})

	t.Run("with filter ids", func(t *testing.T) {
		req := &SearchRequest{}
		err := WithFilterIDs("id1", "id2", "id3")(req)
		require.NoError(t, err)
		require.NotNil(t, req.Filter)
		require.Len(t, req.Filter.IDs, 3)
	})

	t.Run("combine filter and ids", func(t *testing.T) {
		req := &SearchRequest{}
		_ = WithFilter(EqString("type", "document"))(req)
		_ = WithFilterIDs("doc1", "doc2")(req)
		require.NotNil(t, req.Filter.Where)
		require.Len(t, req.Filter.IDs, 2)
	})
}

func TestSearchRequestJSON(t *testing.T) {
	t.Run("basic request with knn rank", func(t *testing.T) {
		req := &SearchRequest{
			Rank: mustKnnRank(t, KnnQueryText("test query")),
			Limit: &SearchPage{
				Limit:  10,
				Offset: 0,
			},
		}

		data, err := req.MarshalJSON()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		require.Contains(t, result, "rank")
		require.Contains(t, result, "limit")
	})

	t.Run("request with filter", func(t *testing.T) {
		req := &SearchRequest{}
		_ = WithFilter(EqString("category", "tech"))(req)
		_ = WithPage(WithLimit(20))(req)

		data, err := req.MarshalJSON()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		require.Contains(t, result, "filter")
		require.Contains(t, result, "limit")
	})

	t.Run("request with select", func(t *testing.T) {
		req := &SearchRequest{}
		_ = WithSelect(KDocument, KScore, K("title"))(req)

		data, err := req.MarshalJSON()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		require.Contains(t, result, "select")
		selectObj := result["select"].(map[string]interface{})
		keys := selectObj["keys"].([]interface{})
		require.Len(t, keys, 3)
	})

	t.Run("empty request produces empty json", func(t *testing.T) {
		req := &SearchRequest{}
		data, err := req.MarshalJSON()
		require.NoError(t, err)
		require.JSONEq(t, `{}`, string(data))
	})
}

func TestSearchQuery(t *testing.T) {
	t.Run("single search request", func(t *testing.T) {
		sq := &SearchQuery{}
		opt := NewSearchRequest(
			WithKnnRank(KnnQueryText("test"), WithKnnLimit(50)),
			WithPage(WithLimit(10)),
		)
		err := opt(sq)
		require.NoError(t, err)
		require.Len(t, sq.Searches, 1)
	})

	t.Run("multiple search requests", func(t *testing.T) {
		sq := &SearchQuery{}

		opt1 := NewSearchRequest(
			WithKnnRank(KnnQueryText("query1")),
		)
		opt2 := NewSearchRequest(
			WithKnnRank(KnnQueryText("query2")),
		)

		_ = opt1(sq)
		_ = opt2(sq)

		require.Len(t, sq.Searches, 2)
	})
}

func TestWithKnnRank(t *testing.T) {
	t.Run("basic knn rank", func(t *testing.T) {
		req := &SearchRequest{}
		err := WithKnnRank(KnnQueryText("machine learning"))(req)
		require.NoError(t, err)
		require.NotNil(t, req.Rank)

		knn, ok := req.Rank.(*KnnRank)
		require.True(t, ok)
		require.Equal(t, "machine learning", knn.Query)
	})

	t.Run("knn rank with options", func(t *testing.T) {
		req := &SearchRequest{}
		err := WithKnnRank(
			KnnQueryText("AI research"),
			WithKnnLimit(100),
			WithKnnDefault(10.0),
			WithKnnKey(K("custom_field")),
		)(req)
		require.NoError(t, err)

		knn, ok := req.Rank.(*KnnRank)
		require.True(t, ok)
		require.Equal(t, 100, knn.Limit)
		require.NotNil(t, knn.DefaultScore)
		require.Equal(t, 10.0, *knn.DefaultScore)
		require.Equal(t, ProjectionKey("custom_field"), knn.Key)
	})
}

func TestWithRffRank(t *testing.T) {
	t.Run("basic rff rank", func(t *testing.T) {
		req := &SearchRequest{}
		knn1 := mustKnnRank(t, KnnQueryText("query1"))
		knn2 := mustKnnRank(t, KnnQueryText("query2"))
		err := WithRffRank(
			WithRffRanks(
				knn1.WithWeight(0.5),
				knn2.WithWeight(0.5),
			),
		)(req)
		require.NoError(t, err)
		require.NotNil(t, req.Rank)

		rrf, ok := req.Rank.(*RrfRank)
		require.True(t, ok)
		require.Len(t, rrf.Ranks, 2)
	})

	t.Run("rff with custom k", func(t *testing.T) {
		req := &SearchRequest{}
		knn := mustKnnRank(t, KnnQueryText("test"))
		err := WithRffRank(
			WithRffRanks(knn.WithWeight(1.0)),
			WithRffK(100),
		)(req)
		require.NoError(t, err)

		rrf := req.Rank.(*RrfRank)
		require.Equal(t, 100, rrf.K)
	})

	t.Run("rff with invalid k returns error", func(t *testing.T) {
		req := &SearchRequest{}
		knn := mustKnnRank(t, KnnQueryText("test"))
		err := WithRffRank(
			WithRffRanks(knn.WithWeight(1.0)),
			WithRffK(-1),
		)(req)
		require.Error(t, err)
	})
}

func TestProjectionKey(t *testing.T) {
	t.Run("standard keys", func(t *testing.T) {
		require.Equal(t, ProjectionKey("#document"), KDocument)
		require.Equal(t, ProjectionKey("#embedding"), KEmbedding)
		require.Equal(t, ProjectionKey("#score"), KScore)
		require.Equal(t, ProjectionKey("#metadata"), KMetadata)
		require.Equal(t, ProjectionKey("#id"), KID)
	})

	t.Run("custom key", func(t *testing.T) {
		key := K("my_custom_field")
		require.Equal(t, ProjectionKey("my_custom_field"), key)
	})
}

func TestSearchFilterJSON(t *testing.T) {
	t.Run("filter with where", func(t *testing.T) {
		filter := &SearchFilter{
			Where: EqString("status", "active"),
		}

		data, err := filter.MarshalJSON()
		require.NoError(t, err)
		require.NotNil(t, data)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Contains(t, result, "where")
	})

	t.Run("filter with ids", func(t *testing.T) {
		filter := &SearchFilter{
			IDs: []DocumentID{"id1", "id2"},
		}

		data, err := filter.MarshalJSON()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Contains(t, result, "ids")
	})

	t.Run("empty filter returns nil", func(t *testing.T) {
		filter := &SearchFilter{}
		data, err := filter.MarshalJSON()
		require.NoError(t, err)
		require.Nil(t, data)
	})
}

func TestCompleteSearchScenario(t *testing.T) {
	t.Run("full search with all options", func(t *testing.T) {
		sq := &SearchQuery{}

		opt := NewSearchRequest(
			WithFilter(
				And(
					EqString("status", "published"),
					GtInt("views", 100),
				),
			),
			WithKnnRank(
				KnnQueryText("machine learning tutorials"),
				WithKnnLimit(50),
				WithKnnDefault(1000.0),
			),
			WithPage(WithLimit(20), WithOffset(0)),
			WithSelect(KDocument, KScore, K("title"), K("author")),
		)

		err := opt(sq)
		require.NoError(t, err)
		require.Len(t, sq.Searches, 1)

		search := sq.Searches[0]
		require.NotNil(t, search.Filter)
		require.NotNil(t, search.Rank)
		require.NotNil(t, search.Limit)
		require.NotNil(t, search.Select)

		// Verify JSON serialization
		data, err := json.Marshal(sq)
		require.NoError(t, err)
		require.NotEmpty(t, data)
	})
}
