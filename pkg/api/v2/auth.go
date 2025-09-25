package v2

import (
	"encoding/base64"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

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
	return "BasicAuthCredentialsProvider {" + _obfuscateBasicAuth(b.Username, b.Password) + "}"
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
	return "TokenAuthCredentialsProvider {" + string(t.Header) + ": " + _obfuscateToken(t.Token) + "}"
}

func _obfuscateBasicAuth(username, password string) string {
	// This is a placeholder for any obfuscation logic you might want to implement.
	// For now, it just returns the username and password as is.
	return username + ":****"
}

func _obfuscateRequestDump(reqDump string) string {
	// X-Chroma-Token obfuscation - handle tokens of any length
	re := regexp.MustCompile(`(?m)^X-Chroma-Token:\s*(.+)$`)
	result := re.ReplaceAllStringFunc(reqDump, func(match string) string {
		parts := strings.SplitN(match, ":", 2)
		if len(parts) != 2 {
			return match
		}
		token := strings.TrimSpace(parts[1])
		return "X-Chroma-Token: " + _obfuscateToken(token)
	})

	// bearer token obfuscation - handle tokens of any length
	re = regexp.MustCompile(`(?m)^Authorization:\s*Bearer\s+(.+)$`)
	result = re.ReplaceAllStringFunc(result, func(match string) string {
		parts := strings.SplitN(match, "Bearer ", 2)
		if len(parts) != 2 {
			return match
		}
		token := strings.TrimSpace(parts[1])
		return "Authorization: Bearer " + _obfuscateToken(token)
	})

	// Basic auth obfuscation - handle tokens of any length
	re = regexp.MustCompile(`(?m)^Authorization:\s*Basic\s+(.+)$`)
	result = re.ReplaceAllStringFunc(result, func(match string) string {
		parts := strings.SplitN(match, "Basic ", 2)
		if len(parts) != 2 {
			return match
		}
		token := strings.TrimSpace(parts[1])
		return "Authorization: Basic " + _obfuscateToken(token)
	})

	return result
}

// _obfuscateToken safely obfuscates tokens of any length
func _obfuscateToken(token string) string {
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
	// First obfuscate any tokens that might be in headers
	result := _obfuscateRequestDump(respDump)

	// Sanitize potential sensitive data in JSON responses
	// Look for common sensitive field patterns in JSON
	patterns := []struct {
		regex   *regexp.Regexp
		replace string
	}{
		{regexp.MustCompile(`"(api_key|apiKey|api_token|apiToken|secret|password|token|auth|credential)":\s*"[^"]+"`), `"$1": "***REDACTED***"`},
		{regexp.MustCompile(`"(access_token|accessToken|refresh_token|refreshToken|id_token|idToken)":\s*"[^"]+"`), `"$1": "***REDACTED***"`},
		{regexp.MustCompile(`"(private_key|privateKey|secret_key|secretKey)":\s*"[^"]+"`), `"$1": "***REDACTED***"`},
		{regexp.MustCompile(`"(authorization|Authorization)":\s*"[^"]+"`), `"$1": "***REDACTED***"`},
	}

	for _, p := range patterns {
		result = p.regex.ReplaceAllString(result, p.replace)
	}

	return result
}
