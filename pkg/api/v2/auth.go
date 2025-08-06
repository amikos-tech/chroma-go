package v2

import (
	"encoding/base64"
	"regexp"

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

func _obfuscateToken(token string) string {
	// This is a placeholder for any obfuscation logic you might want to implement.
	// For now, it just returns the token as is.
	if len(token) < 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
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
	// X-Chroma-Token obfuscation
	re := regexp.MustCompile(`(?m)^X-Chroma-Token:\s*(.{1,4}).*(.{4})$`)
	result := re.ReplaceAllString(reqDump, `X-Chroma-Token: $1...$2`)
	// bearer token obfuscation
	re = regexp.MustCompile(`(?m)^Authorization:\s*Bearer\s+(.{1,4}).*(.{4})$`)
	result = re.ReplaceAllString(result, `Authorization: Bearer $1...$2`)
	// Basic auth obfuscation
	re = regexp.MustCompile(`(?m)^Authorization:\s*Basic\s+(.{1,4}).*(.{4})$`)
	result = re.ReplaceAllString(result, `Authorization: Basic $1...$2`)
	return result
}
