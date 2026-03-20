package embeddings

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubIntentMapper struct {
	mappings map[Intent]string
	errs     map[Intent]error
}

func (m *stubIntentMapper) MapIntent(intent Intent) (string, error) {
	if err, ok := m.errs[intent]; ok {
		return "", err
	}
	if native, ok := m.mappings[intent]; ok {
		return native, nil
	}
	return string(intent), nil
}

var _ IntentMapper = (*stubIntentMapper)(nil)

func TestIsNeutralIntent(t *testing.T) {
	cases := []struct {
		intent   Intent
		expected bool
	}{
		{IntentRetrievalQuery, true},
		{IntentRetrievalDocument, true},
		{IntentClassification, true},
		{IntentClustering, true},
		{IntentSemanticSimilarity, true},
		{Intent(""), false},
		{Intent("RETRIEVAL_QUERY_V2"), false},
		{Intent("my_custom_task"), false},
	}

	for _, tc := range cases {
		t.Run(string(tc.intent), func(t *testing.T) {
			require.Equal(t, tc.expected, IsNeutralIntent(tc.intent))
		})
	}
}

func TestIntentMapperContract(t *testing.T) {
	mapper := &stubIntentMapper{
		mappings: map[Intent]string{
			IntentRetrievalQuery:     "RETRIEVAL_QUERY",
			IntentSemanticSimilarity: "SEMANTIC_SIMILARITY",
		},
	}

	native, err := mapper.MapIntent(IntentRetrievalQuery)
	require.NoError(t, err)
	require.Equal(t, "RETRIEVAL_QUERY", native)

	native, err = mapper.MapIntent(IntentSemanticSimilarity)
	require.NoError(t, err)
	require.Equal(t, "SEMANTIC_SIMILARITY", native)

	native, err = mapper.MapIntent(IntentClustering)
	require.NoError(t, err)
	require.Equal(t, "clustering", native)
}

func TestIntentMapperEscapeHatch(t *testing.T) {
	mapper := &stubIntentMapper{}

	native, err := mapper.MapIntent(Intent("CUSTOM_TASK"))
	require.NoError(t, err)
	require.Equal(t, "CUSTOM_TASK", native)

	errMapper := &stubIntentMapper{
		errs: map[Intent]error{
			Intent("bad_task"): fmt.Errorf("unknown intent"),
		},
	}

	native, err = errMapper.MapIntent(Intent("bad_task"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown intent")
	require.Empty(t, native)
}

func TestIntentMapperNilCheck(t *testing.T) {
	ef := &recordingEmbeddingFunction{}
	_, ok := any(ef).(IntentMapper)
	require.False(t, ok)
}
