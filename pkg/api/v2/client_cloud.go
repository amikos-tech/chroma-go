package v2

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
)

type CloudClientOption func(client *CloudAPIClient) error
type CloudAPIClient struct {
	*APIClientV2
}

func NewCloudAPIClient(options ...ClientOption) (*CloudAPIClient, error) {
	bc, err := newBaseAPIClient()
	if err != nil {
		return nil, err
	}
	c := &CloudAPIClient{
		&APIClientV2{
			BaseAPIClient:      *bc,
			preflightLimits:    map[string]interface{}{},
			preflightCompleted: false,
			collectionCache:    map[string]Collection{},
		},
	}
	updatedOpts := make([]ClientOption, 0)
	for _, option := range options {
		if option != nil {
			updatedOpts = append(updatedOpts, option)
		}
	}
	// we override the base URL for the cloud client
	updatedOpts = append(updatedOpts, WithBaseURL("https://api.trychroma.com:8000/api/v2"))

	for _, option := range updatedOpts {
		if err := option(&c.BaseAPIClient); err != nil {
			return nil, err
		}
	}

	if c.authProvider == nil && os.Getenv("CHROMA_API_KEY") == "" {
		return nil, errors.New("api key not provided. Use WithCloudAPIKey option or set CHROMA_API_KEY environment variable")
	} else if c.authProvider == nil {
		c.authProvider = NewTokenAuthCredentialsProvider(os.Getenv("CHROMA_API_KEY"), XChromaTokenHeader)
	}
	fmt.Println(c.authProvider)
	return c, nil
}

func WithCloudAPIKey(apiKey string) ClientOption {
	return func(c *BaseAPIClient) error {
		if apiKey == "" {
			return errors.New("api key is empty")
		}
		c.authProvider = NewTokenAuthCredentialsProvider(apiKey, XChromaTokenHeader)
		return nil
	}
}
