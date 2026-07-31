//go:build basicv2 && !cloud

package v2_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

type customEmbeddedSumRank struct {
	*v2.SumRank
}

func (*customEmbeddedSumRank) MarshalJSON() ([]byte, error) {
	return []byte(`{"$custom":true}`), nil
}

func TestCustomRankEmbeddingCompositePreservesMarshalJSON(t *testing.T) {
	sum := v2.Val(1).Add(v2.Val(2)).(*v2.SumRank)
	custom := &customEmbeddedSumRank{SumRank: sum}

	data, err := json.Marshal(&v2.SearchRequest{Rank: custom})
	require.NoError(t, err)
	require.JSONEq(t, `{"rank":{"$custom":true}}`, string(data))
}
