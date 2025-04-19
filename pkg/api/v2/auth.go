package v2

import (
	"encoding/base64"

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
