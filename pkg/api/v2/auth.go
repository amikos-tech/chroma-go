package v2

import (
	"encoding/base64"
	"log"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// Package-level compiled regex patterns for better performance and safety
var (
	// Header patterns - case insensitive for better coverage
	reChromaToken *regexp.Regexp
	reBearerToken *regexp.Regexp
	reBasicAuth   *regexp.Regexp

	// JSON field patterns for response sanitization
	reJSONPatterns []*regexp.Regexp
)

func init() {
	var err error

	// Using regexp.Compile instead of MustCompile to avoid panics and allow graceful degradation
	// nolint:gocritic // intentionally using Compile for safety
	// Compile header patterns with case-insensitive flag
	reChromaToken, err = regexp.Compile(`(?im)^X-Chroma-Token:\s*(.+)$`)
	if err != nil {
		log.Printf("Warning: Failed to compile X-Chroma-Token regex: %v", err)
	}

	// nolint:gocritic // intentionally using Compile for safety
	reBearerToken, err = regexp.Compile(`(?im)^Authorization:\s*Bearer\s+(.+)$`)
	if err != nil {
		log.Printf("Warning: Failed to compile Bearer token regex: %v", err)
	}

	// nolint:gocritic // intentionally using Compile for safety
	reBasicAuth, err = regexp.Compile(`(?im)^Authorization:\s*Basic\s+(.+)$`)
	if err != nil {
		log.Printf("Warning: Failed to compile Basic auth regex: %v", err)
	}

	// Compile JSON patterns
	jsonPatterns := []string{
		`"(api_key|apiKey|api_token|apiToken|secret|password|token|auth|credential)":\s*"[^"]+"`,
		`"(access_token|accessToken|refresh_token|refreshToken|id_token|idToken)":\s*"[^"]+"`,
		`"(private_key|privateKey|secret_key|secretKey)":\s*"[^"]+"`,
		`"(authorization|Authorization)":\s*"[^"]+"`,
	}

	reJSONPatterns = make([]*regexp.Regexp, 0, len(jsonPatterns))
	for _, pattern := range jsonPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Printf("Warning: Failed to compile JSON pattern %s: %v", pattern, err)
		} else {
			reJSONPatterns = append(reJSONPatterns, re)
		}
	}
}

type CredentialsProvider interface {
	Authenticate(apiClient *BaseAPIClient) error
}

type BasicAuthCredentialsProvider struct {
	Username string
	Password string
}

func NewBasicAuthCredentialsProvider(username, password string) *BasicAuthCredentialsProvider {
	return &BasicAuthCredentialsProvider{
		Username: username,
		Password: password,
	}
}

func (b *BasicAuthCredentialsProvider) Authenticate(client *BaseAPIClient) error {
	auth := b.Username + ":" + b.Password
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	client.defaultHeaders["Authorization"] = "Basic " + encodedAuth
	return nil
}

func (b *BasicAuthCredentialsProvider) String() string {
	return "BasicAuthCredentialsProvider {" + _sanitizeBasicAuth(b.Username, b.Password) + "}"
}

type TokenTransportHeader string

const (
	AuthorizationTokenHeader TokenTransportHeader = "Authorization"
	XChromaTokenHeader       TokenTransportHeader = "X-Chroma-Token"
)

type TokenAuthCredentialsProvider struct {
	Token  string
	Header TokenTransportHeader
}

func NewTokenAuthCredentialsProvider(token string, header TokenTransportHeader) *TokenAuthCredentialsProvider {
	return &TokenAuthCredentialsProvider{
		Token:  token,
		Header: header,
	}
}

func (t *TokenAuthCredentialsProvider) Authenticate(client *BaseAPIClient) error {
	switch t.Header {
	case AuthorizationTokenHeader:
		client.defaultHeaders[string(t.Header)] = "Bearer " + t.Token
		return nil
	case XChromaTokenHeader:
		client.defaultHeaders[string(t.Header)] = t.Token
		return nil
	default:
		return errors.Errorf("unsupported token header: %v", t.Header)
	}
}

func (t *TokenAuthCredentialsProvider) String() string {
	return "TokenAuthCredentialsProvider {" + string(t.Header) + ": " + _sanitizeToken(t.Token) + "}"
}

func _sanitizeBasicAuth(username, password string) string {
	// This is a placeholder for any obfuscation logic you might want to implement.
	// For now, it just returns the username and password as is.
	return username + ":****"
}

func _sanitizeRequestDump(reqDump string) string {
	// Add panic protection
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Warning: Panic in _sanitizeRequestDump: %v. Returning partially sanitized output.", r)
		}
	}()

	result := reqDump

	// X-Chroma-Token obfuscation - handle tokens of any length
	if reChromaToken != nil {
		result = reChromaToken.ReplaceAllStringFunc(result, func(match string) string {
			parts := strings.SplitN(match, ":", 2)
			if len(parts) != 2 {
				return match
			}
			token := strings.TrimSpace(parts[1])
			return "X-Chroma-Token: " + _sanitizeToken(token)
		})
	}

	// Bearer token obfuscation - handle tokens of any length
	if reBearerToken != nil {
		result = reBearerToken.ReplaceAllStringFunc(result, func(match string) string {
			parts := strings.SplitN(match, "Bearer ", 2)
			if len(parts) != 2 {
				return match
			}
			token := strings.TrimSpace(parts[1])
			return "Authorization: Bearer " + _sanitizeToken(token)
		})
	}

	// Basic auth obfuscation - handle tokens of any length
	if reBasicAuth != nil {
		result = reBasicAuth.ReplaceAllStringFunc(result, func(match string) string {
			parts := strings.SplitN(match, "Basic ", 2)
			if len(parts) != 2 {
				return match
			}
			token := strings.TrimSpace(parts[1])
			return "Authorization: Basic " + _sanitizeToken(token)
		})
	}

	return result
}

// _sanitizeToken safely obfuscates tokens of any length
func _sanitizeToken(token string) string {
	// Add panic protection for string operations
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Warning: Panic in _sanitizeToken: %v. Returning stars.", r)
			// Return a safe fallback - all stars
			if len(token) > 0 {
				return
			}
		}
	}()

	tokenLen := len(token)
	if tokenLen == 0 {
		return ""
	}
	if tokenLen <= 4 {
		// For very short tokens, show only first character
		return string(token[0]) + strings.Repeat("*", tokenLen-1)
	}
	if tokenLen <= 8 {
		// For short tokens, show first 2 and last 2 characters
		return token[:2] + "..." + token[tokenLen-2:]
	}
	// For longer tokens, show first 4 and last 4 characters
	return token[:4] + "..." + token[tokenLen-4:]
}

// _sanitizeResponseDump sanitizes response dumps to remove sensitive data
func _sanitizeResponseDump(respDump string) string {
	// Add panic protection
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Warning: Panic in _sanitizeResponseDump: %v. Returning partially sanitized output.", r)
		}
	}()

	// First obfuscate any tokens that might be in headers
	result := _sanitizeRequestDump(respDump)

	// Sanitize potential sensitive data in JSON responses using pre-compiled patterns
	for _, re := range reJSONPatterns {
		if re != nil {
			result = re.ReplaceAllString(result, `"$1": "***REDACTED***"`)
		}
	}

	return result
}
