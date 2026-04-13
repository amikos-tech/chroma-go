package http

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSanitizeErrorBodyLimit  = 512
	testSanitizeErrorBodySuffix = "[truncated]"
)

func TestReadLimitedBody(t *testing.T) {
	t.Run("under limit", func(t *testing.T) {
		input := []byte("hello world")
		data, err := ReadLimitedBody(bytes.NewReader(input))
		require.NoError(t, err)
		assert.Equal(t, input, data)
	})

	t.Run("empty reader", func(t *testing.T) {
		data, err := ReadLimitedBody(bytes.NewReader(nil))
		require.NoError(t, err)
		assert.Empty(t, data)
	})

	t.Run("exactly at limit", func(t *testing.T) {
		input := strings.Repeat("a", MaxResponseBodySize)
		data, err := ReadLimitedBody(strings.NewReader(input))
		require.NoError(t, err)
		assert.Len(t, data, MaxResponseBodySize)
	})

	t.Run("over limit", func(t *testing.T) {
		input := strings.Repeat("a", MaxResponseBodySize+1)
		_, err := ReadLimitedBody(strings.NewReader(input))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "response body exceeds maximum size")
	})
}

func TestSanitizeErrorBody(t *testing.T) {
	testCases := []struct {
		name   string
		body   []byte
		want   string
		verify func(t *testing.T, got string)
	}{
		{
			name: "nil body returns empty string",
			body: nil,
			want: "",
		},
		{
			name: "empty body returns empty string",
			body: []byte(""),
			want: "",
		},
		{
			name: "trim whitespace before truncation",
			body: []byte(" \n\ttrim me\t\n "),
			want: "trim me",
		},
		{
			name: "short body passes through",
			body: []byte("short body"),
			want: "short body",
		},
		{
			name: "long ascii body truncates with exact suffix",
			body: []byte(strings.Repeat("x", testSanitizeErrorBodyLimit+32)),
			want: strings.Repeat("x", testSanitizeErrorBodyLimit) + testSanitizeErrorBodySuffix,
		},
		{
			name: "long utf8 body truncates by rune count",
			body: []byte(strings.Repeat("☺", testSanitizeErrorBodyLimit+10)),
			want: strings.Repeat("☺", testSanitizeErrorBodyLimit) + testSanitizeErrorBodySuffix,
			verify: func(t *testing.T, got string) {
				t.Helper()
				prefix := strings.TrimSuffix(got, testSanitizeErrorBodySuffix)
				require.True(t, utf8.ValidString(prefix))
				assert.Len(t, []rune(prefix), testSanitizeErrorBodyLimit)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeErrorBody(tc.body)
			assert.Equal(t, tc.want, got)
			if tc.verify != nil {
				tc.verify(t, got)
			}
		})
	}
}
