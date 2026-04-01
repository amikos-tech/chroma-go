package twelvelabs

import (
	"net/http"
	"os"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// Option configures a TwelveLabsClient.
type Option func(p *TwelveLabsClient) error

func WithModel(model embeddings.EmbeddingModel) Option {
	return func(p *TwelveLabsClient) error {
		if model == "" {
			return errors.New("model cannot be empty")
		}
		p.DefaultModel = model
		return nil
	}
}

func WithAPIKey(apiKey string) Option {
	return func(p *TwelveLabsClient) error {
		if apiKey == "" {
			return errors.New("API key cannot be empty")
		}
		p.APIKey = embeddings.NewSecret(apiKey)
		return nil
	}
}

func WithEnvAPIKey() Option {
	return func(p *TwelveLabsClient) error {
		if apiKey := os.Getenv(APIKeyEnvVar); apiKey != "" {
			p.APIKey = embeddings.NewSecret(apiKey)
			p.APIKeyEnvVar = APIKeyEnvVar
			return nil
		}
		return errors.Errorf("%s not set", APIKeyEnvVar)
	}
}

// WithAPIKeyFromEnvVar sets the API key from a specified environment variable.
func WithAPIKeyFromEnvVar(envVar string) Option {
	return func(p *TwelveLabsClient) error {
		if apiKey := os.Getenv(envVar); apiKey != "" {
			p.APIKey = embeddings.NewSecret(apiKey)
			p.APIKeyEnvVar = envVar
			return nil
		}
		return errors.Errorf("%s not set", envVar)
	}
}

func WithBaseURL(baseURL string) Option {
	return func(p *TwelveLabsClient) error {
		if baseURL == "" {
			return errors.New("base URL cannot be empty")
		}
		p.BaseAPI = baseURL
		return nil
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(p *TwelveLabsClient) error {
		if client == nil {
			return errors.New("HTTP client cannot be nil")
		}
		p.Client = client
		return nil
	}
}

// WithInsecure allows the client to connect to HTTP endpoints without TLS.
func WithInsecure() Option {
	return func(p *TwelveLabsClient) error {
		p.Insecure = true
		return nil
	}
}

// WithAudioEmbeddingOption sets the audio embedding option.
// Valid values: "audio", "transcription", "fused".
func WithAudioEmbeddingOption(opt string) Option {
	return func(p *TwelveLabsClient) error {
		switch opt {
		case "audio", "transcription", "fused":
			p.AudioEmbeddingOption = opt
			return nil
		default:
			return errors.Errorf("invalid audio embedding option %q: must be one of audio, transcription, fused", opt)
		}
	}
}
