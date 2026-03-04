package v2

import (
	"os"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/logger"
)

const ChromaCloudEndpoint = "https://api.trychroma.com:8000/api/v2"

type CloudClientOption func(client *CloudAPIClient) error
type CloudAPIClient struct {
	*APIClientV2
}

func NewCloudClient(options ...ClientOption) (*CloudAPIClient, error) {
	updatedOpts := make([]ClientOption, 0, len(options)+3)
	updatedOpts = append(updatedOpts, WithDatabaseAndTenantFromEnv())
	for _, option := range options {
		if option != nil {
			updatedOpts = append(updatedOpts, option)
		}
	}
	// we override the base URL for the cloud client
	updatedOpts = append(updatedOpts, WithBaseURL(ChromaCloudEndpoint))
	updatedOpts = append(updatedOpts, withCloudAPIKeyFromEnvIfUnset())

	bc, err := newBaseAPIClient(updatedOpts...)
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

	tenant, database := c.TenantAndDatabase()
	if tenant == nil || tenant.Name() == DefaultTenant || database == nil || database.Name() == DefaultDatabase {
		return nil, errors.New("tenant and database must be set for cloud client. Use WithDatabaseAndTenantFromEnv option or set CHROMA_TENANT and CHROMA_DATABASE environment variables")
	}

	if c.authProvider == nil {
		return nil, errors.New("api key not provided. Use WithCloudAPIKey option or set CHROMA_API_KEY environment variable")
	}

	// Ensure logger is never nil - but don't override if already set by options like WithDebug()
	if c.logger == nil {
		c.logger = logger.NewNoopLogger()
	}

	return c, nil
}

// Deprecated: use NewCloudClient instead
func NewCloudAPIClient(options ...ClientOption) (*CloudAPIClient, error) {
	return NewCloudClient(options...)
}

func withCloudAPIKeyFromEnvIfUnset() ClientOption {
	return func(c *BaseAPIClient) error {
		if c.authProvider != nil {
			return nil
		}
		apiKey := os.Getenv("CHROMA_API_KEY")
		if apiKey == "" {
			return nil
		}
		c.authProvider = NewTokenAuthCredentialsProvider(apiKey, XChromaTokenHeader)
		return nil
	}
}

// WithCloudAPIKey sets the API key for the cloud client. It will automatically set a new TokenAuthCredentialsProvider.
func WithCloudAPIKey(apiKey string) ClientOption {
	return func(c *BaseAPIClient) error {
		if apiKey == "" {
			return errors.New("api key is empty")
		}
		c.authProvider = NewTokenAuthCredentialsProvider(apiKey, XChromaTokenHeader)
		return nil
	}
}
