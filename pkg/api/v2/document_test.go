//go:build basicv2

package v2

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTextDocument(t *testing.T) {

	doc := "Hello, world!\n"

	tdoc := NewTextDocument(doc)

	marshal, err := json.Marshal(tdoc)
	require.NoError(t, err)
	require.Equal(t, `"Hello, world!\n"`, string(marshal))
}
