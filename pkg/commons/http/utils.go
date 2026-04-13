package http

import (
	"fmt"
	"io"
	"strings"
)

// MaxResponseBodySize is the maximum allowed response body size (200 MB).
const MaxResponseBodySize = 200 * 1024 * 1024

const (
	maxSanitizedErrorBodyRunes = 512
	truncatedErrorBodySuffix   = "[truncated]"
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

func sanitizeErrorBodyString(body string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= maxSanitizedErrorBodyRunes {
		return trimmed
	}
	return string(runes[:maxSanitizedErrorBodyRunes]) + truncatedErrorBodySuffix
}

// SanitizeErrorBody normalizes provider body text for display without affecting
// transport-level read limits. It never panics; recovery returns the best
// sanitized value available instead of surfacing raw body contents.
func SanitizeErrorBody(body []byte) (result string) {
	defer func() {
		if recover() != nil {
			fallback := result
			if fallback == "" {
				fallback = string(body)
			}
			result = sanitizeErrorBodyString(fallback)
		}
	}()

	result = string(body)
	result = sanitizeErrorBodyString(result)
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
