package http

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// MaxResponseBodySize is the maximum allowed response body size (200 MB).
const MaxResponseBodySize = 200 * 1024 * 1024

const (
	maxSanitizedErrorBodyRunes = 512
	truncatedErrorBodySuffix   = "[truncated]"
	panicErrorBodyFallback     = truncatedErrorBodySuffix
)

// ReadLimitedBody reads up to MaxResponseBodySize bytes from r.
// Returns an error if the response exceeds the limit.
func ReadLimitedBody(r io.Reader) ([]byte, error) {
	limitedReader := io.LimitReader(r, int64(MaxResponseBodySize)+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}
	if len(data) > MaxResponseBodySize {
		return nil, fmt.Errorf("response body exceeds maximum size of %d bytes", MaxResponseBodySize)
	}
	return data, nil
}

func sanitizeErrorBody(body []byte) string {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(len(truncatedErrorBodySuffix) + min(len(trimmed), maxSanitizedErrorBodyRunes*utf8.UTFMax))
	runes := 0
	for len(trimmed) > 0 && runes < maxSanitizedErrorBodyRunes {
		r, size := utf8.DecodeRune(trimmed)
		b.WriteRune(r)
		trimmed = trimmed[size:]
		runes++
	}

	if len(trimmed) > 0 {
		b.WriteString(truncatedErrorBodySuffix)
	}

	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SanitizeErrorBody normalizes provider body text for display without affecting
// transport-level read limits. It never panics; recovery returns the best
// sanitized value available instead of surfacing raw body contents.
func SanitizeErrorBody(body []byte) (result string) {
	defer func() {
		if recover() != nil {
			if result == "" {
				result = panicErrorBodyFallback
			}
		}
	}()

	result = sanitizeErrorBody(body)
	return result
}

func ReadRespBody(resp io.Reader) string {
	if resp == nil {
		return ""
	}
	body, err := io.ReadAll(resp)
	if err != nil {
		return ""
	}
	return string(body)
}
