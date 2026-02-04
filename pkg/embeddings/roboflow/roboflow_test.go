//go:build ef

package roboflow

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// validPNGBase64 is a valid 8x8 red PNG image encoded as base64.
// This is a pre-generated valid PNG that Roboflow's API can process.
const validPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAgAAAAICAIAAABLbSncAAAADklEQVR4nGP4z8DAwMAAAj4C/ZfyKMkAAAAASUVORK5CYII="

func TestRoboflowEmbeddingFunction(t *testing.T) {
	apiKey := os.Getenv("ROBOFLOW_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv("ROBOFLOW_API_KEY")
	}

	t.Run("Test text embedding with defaults", func(t *testing.T) {
		if apiKey == "" {
			t.Skip("ROBOFLOW_API_KEY not set")
		}
		ef, err := NewRoboflowEmbeddingFunction(WithAPIKey(apiKey))
		require.NoError(t, err)
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
		}
		resp, err := ef.EmbedDocuments(context.Background(), documents)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Greater(t, resp[0].Len(), 0)
	})

	t.Run("Test text embedding with env API key", func(t *testing.T) {
		if apiKey == "" {
			t.Skip("ROBOFLOW_API_KEY not set")
		}
		ef, err := NewRoboflowEmbeddingFunction(WithEnvAPIKey())
		require.NoError(t, err)
		documents := []string{
			"Document 1 content here",
		}
		resp, err := ef.EmbedDocuments(context.Background(), documents)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp, 1)
		require.Greater(t, resp[0].Len(), 0)
	})

	t.Run("Test EmbedQuery", func(t *testing.T) {
		if apiKey == "" {
			t.Skip("ROBOFLOW_API_KEY not set")
		}
		ef, err := NewRoboflowEmbeddingFunction(WithAPIKey(apiKey))
		require.NoError(t, err)
		resp, err := ef.EmbedQuery(context.Background(), "What is the meaning of life?")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Greater(t, resp.Len(), 0)
	})

	t.Run("Test image embedding from base64", func(t *testing.T) {
		if apiKey == "" {
			t.Skip("ROBOFLOW_API_KEY not set")
		}
		ef, err := NewRoboflowEmbeddingFunction(WithAPIKey(apiKey))
		require.NoError(t, err)

		image := embeddings.NewImageInputFromBase64(validPNGBase64)
		resp, err := ef.EmbedImage(context.Background(), image)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Greater(t, resp.Len(), 0)
		t.Logf("Image embedding length: %d", resp.Len())
	})

	t.Run("Test EmbedImages batch", func(t *testing.T) {
		if apiKey == "" {
			t.Skip("ROBOFLOW_API_KEY not set")
		}
		ef, err := NewRoboflowEmbeddingFunction(WithAPIKey(apiKey))
		require.NoError(t, err)

		images := []embeddings.ImageInput{
			embeddings.NewImageInputFromBase64(validPNGBase64),
			embeddings.NewImageInputFromBase64(validPNGBase64),
		}
		resp, err := ef.EmbedImages(context.Background(), images)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Greater(t, resp[0].Len(), 0)
		require.Greater(t, resp[1].Len(), 0)
	})

	t.Run("Test missing API key", func(t *testing.T) {
		_, err := NewRoboflowEmbeddingFunction()
		require.Error(t, err)
		require.Contains(t, err.Error(), "'APIKey' failed on the 'required'")
	})

	t.Run("Test HTTP endpoint rejected without WithInsecure", func(t *testing.T) {
		_, err := NewRoboflowEmbeddingFunction(WithAPIKey("test-key"), WithBaseURL("http://example.com"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "base URL must use HTTPS")
	})

	t.Run("Test HTTP endpoint accepted with WithInsecure", func(t *testing.T) {
		_, err := NewRoboflowEmbeddingFunction(WithAPIKey("test-key"), WithBaseURL("http://example.com"), WithInsecure())
		require.NoError(t, err)
	})

	t.Run("Test HTTPS endpoint accepted", func(t *testing.T) {
		_, err := NewRoboflowEmbeddingFunction(WithAPIKey("test-key"), WithBaseURL("https://example.com"))
		require.NoError(t, err)
	})

	t.Run("Test GetConfig default", func(t *testing.T) {
		if apiKey == "" {
			t.Skip("ROBOFLOW_API_KEY not set")
		}
		ef, err := NewRoboflowEmbeddingFunction(WithEnvAPIKey())
		require.NoError(t, err)
		cfg := ef.GetConfig()
		require.Equal(t, "ROBOFLOW_API_KEY", cfg["api_key_env_var"])
		_, hasBaseURL := cfg["base_url"]
		require.False(t, hasBaseURL)
	})

	t.Run("Test GetConfig with custom base URL", func(t *testing.T) {
		ef, err := NewRoboflowEmbeddingFunction(WithAPIKey("test-key"), WithBaseURL("https://custom.api.com"), WithInsecure())
		require.NoError(t, err)
		cfg := ef.GetConfig()
		require.Equal(t, "https://custom.api.com", cfg["base_url"])
		require.Equal(t, true, cfg["insecure"])
	})

	t.Run("Test Name returns roboflow", func(t *testing.T) {
		ef, err := NewRoboflowEmbeddingFunction(WithAPIKey("test-key"))
		require.NoError(t, err)
		require.Equal(t, "roboflow", ef.Name())
	})

	t.Run("Test DefaultSpace returns COSINE", func(t *testing.T) {
		ef, err := NewRoboflowEmbeddingFunction(WithAPIKey("test-key"))
		require.NoError(t, err)
		require.Equal(t, embeddings.COSINE, ef.DefaultSpace())
	})

	t.Run("Test SupportedSpaces", func(t *testing.T) {
		ef, err := NewRoboflowEmbeddingFunction(WithAPIKey("test-key"))
		require.NoError(t, err)
		spaces := ef.SupportedSpaces()
		require.Contains(t, spaces, embeddings.COSINE)
		require.Contains(t, spaces, embeddings.L2)
		require.Contains(t, spaces, embeddings.IP)
	})

	t.Run("Test empty documents returns nil", func(t *testing.T) {
		ef, err := NewRoboflowEmbeddingFunction(WithAPIKey("test-key"))
		require.NoError(t, err)
		resp, err := ef.EmbedDocuments(context.Background(), []string{})
		require.NoError(t, err)
		require.Nil(t, resp)
	})

	t.Run("Test empty images returns nil", func(t *testing.T) {
		ef, err := NewRoboflowEmbeddingFunction(WithAPIKey("test-key"))
		require.NoError(t, err)
		resp, err := ef.EmbedImages(context.Background(), []embeddings.ImageInput{})
		require.NoError(t, err)
		require.Nil(t, resp)
	})
}

func TestImageInput(t *testing.T) {
	t.Run("Test Validate with base64", func(t *testing.T) {
		img := embeddings.NewImageInputFromBase64("abc123")
		err := img.Validate()
		require.NoError(t, err)
		require.Equal(t, embeddings.ImageInputTypeBase64, img.Type())
	})

	t.Run("Test Validate with URL", func(t *testing.T) {
		img := embeddings.NewImageInputFromURL("https://example.com/image.png")
		err := img.Validate()
		require.NoError(t, err)
		require.Equal(t, embeddings.ImageInputTypeURL, img.Type())
	})

	t.Run("Test Validate with file path", func(t *testing.T) {
		img := embeddings.NewImageInputFromFile("/path/to/image.png")
		err := img.Validate()
		require.NoError(t, err)
		require.Equal(t, embeddings.ImageInputTypeFilePath, img.Type())
	})

	t.Run("Test Validate with no input", func(t *testing.T) {
		img := embeddings.ImageInput{}
		err := img.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "must have exactly one")
	})

	t.Run("Test Validate with multiple inputs", func(t *testing.T) {
		img := embeddings.ImageInput{
			Base64: "abc123",
			URL:    "https://example.com/image.png",
		}
		err := img.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "got multiple")
	})

	t.Run("Test ToBase64 returns base64 directly", func(t *testing.T) {
		img := embeddings.NewImageInputFromBase64("abc123")
		result, err := img.ToBase64(context.Background())
		require.NoError(t, err)
		require.Equal(t, "abc123", result)
	})

	t.Run("Test ToBase64 with invalid input", func(t *testing.T) {
		img := embeddings.ImageInput{}
		_, err := img.ToBase64(context.Background())
		require.Error(t, err)
	})
}

func TestRoboflowFromConfig(t *testing.T) {
	apiKey := os.Getenv("ROBOFLOW_API_KEY")

	t.Run("Test config missing api_key_env_var", func(t *testing.T) {
		cfg := embeddings.EmbeddingFunctionConfig{}
		_, err := NewRoboflowEmbeddingFunctionFromConfig(cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "api_key_env_var is required")
	})

	t.Run("Test config with all options", func(t *testing.T) {
		if apiKey == "" {
			t.Skip("ROBOFLOW_API_KEY not set")
		}
		cfg := embeddings.EmbeddingFunctionConfig{
			"api_key_env_var": "ROBOFLOW_API_KEY",
			"base_url":        "https://custom.api.com",
			"insecure":        true,
		}
		ef, err := NewRoboflowEmbeddingFunctionFromConfig(cfg)
		require.NoError(t, err)
		require.Equal(t, "https://custom.api.com", ef.baseURL)
		require.True(t, ef.insecure)
	})
}

func TestMultimodalInterface(t *testing.T) {
	t.Run("Test implements MultimodalEmbeddingFunction", func(t *testing.T) {
		ef, err := NewRoboflowEmbeddingFunction(WithAPIKey("test-key"))
		require.NoError(t, err)

		var _ embeddings.MultimodalEmbeddingFunction = ef
		var _ embeddings.EmbeddingFunction = ef
	})
}
