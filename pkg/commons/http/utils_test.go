package http

import (
	"bytes"
	"errors"
	"os"
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

type errorReader struct{}

func (errorReader) Read(_ []byte) (int, error) {
	return 0, errors.New("boom")
}

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
			name: "exactly 512 runes does not append suffix",
			body: []byte(strings.Repeat("x", testSanitizeErrorBodyLimit)),
			want: strings.Repeat("x", testSanitizeErrorBodyLimit),
		},
		{
			name: "513 runes appends suffix",
			body: []byte(strings.Repeat("x", testSanitizeErrorBodyLimit+1)),
			want: strings.Repeat("x", testSanitizeErrorBodyLimit) + testSanitizeErrorBodySuffix,
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
		{
			name: "invalid utf8 decodes as replacement runes",
			body: append([]byte(strings.Repeat("a", testSanitizeErrorBodyLimit-1)), 0xff),
			want: strings.Repeat("a", testSanitizeErrorBodyLimit-1) + string(utf8.RuneError),
			verify: func(t *testing.T, got string) {
				t.Helper()
				require.True(t, utf8.ValidString(got))
				assert.Len(t, []rune(got), testSanitizeErrorBodyLimit)
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

func TestSanitizeErrorBodyBoundsAllocationForLargeBodies(t *testing.T) {
	body := []byte(strings.Repeat("x", 100_000))
	var got string
	allocs := testing.AllocsPerRun(10, func() {
		got = SanitizeErrorBody(body)
	})

	assert.Equal(t, strings.Repeat("x", testSanitizeErrorBodyLimit)+testSanitizeErrorBodySuffix, got)
	assert.LessOrEqual(t, allocs, float64(2))
}

func TestSanitizeErrorBodyAvoidsWholeBodyMaterialization(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile("utils.go")
	require.NoError(t, err)

	assert.NotContains(t, string(source), "[]rune(trimmed)")
	assert.NotContains(t, string(source), "fallback = string(body)")
	assert.NotContains(t, string(source), "result = string(body)")
}

func TestSanitizeErrorBodyRecoversFromPanic(t *testing.T) {
	original := sanitizeErrorBodyFunc
	sanitizeErrorBodyFunc = func([]byte) string {
		panic("boom")
	}
	t.Cleanup(func() {
		sanitizeErrorBodyFunc = original
	})

	assert.Equal(t, panicErrorBodyFallback, SanitizeErrorBody([]byte("body")))
}

func TestReadRespBodyReportsReadErrors(t *testing.T) {
	assert.Contains(t, ReadRespBody(errorReader{}), "failed to read response body")
	assert.Contains(t, ReadRespBody(errorReader{}), "boom")
}
