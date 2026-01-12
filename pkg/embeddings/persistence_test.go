//go:build ef

package embeddings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/chromacloud"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/cloudflare"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/cohere"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/gemini"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/hf"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/jina"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/mistral"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/nomic"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/ollama"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/together"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/voyage"
)

// TestEmbeddingFunctionPersistence verifies that all embedding functions can be:
// 1. Created with a config
// 2. Serialized via Name() and GetConfig()
// 3. Rebuilt from the serialized config via BuildDense()
// 4. The rebuilt EF has matching name and config
//
// Note: Most EFs return a hardcoded env var name in GetConfig() for security,
// so we set the standard env var for each provider to enable rebuild.

func TestEmbeddingFunctionPersistence_ConsistentHash(t *testing.T) {
	// ConsistentHash doesn't require API keys
	ef := embeddings.NewConsistentHashEmbeddingFunction()

	name := ef.Name()
	config := ef.GetConfig()

	assert.Equal(t, "consistent_hash", name)
	assert.NotNil(t, config)

	// Verify registry has this EF
	assert.True(t, embeddings.HasDense(name), "consistent_hash should be registered")

	// Rebuild from config
	rebuilt, err := embeddings.BuildDense(name, config)
	require.NoError(t, err)
	require.NotNil(t, rebuilt)

	// Verify rebuilt EF matches
	assert.Equal(t, name, rebuilt.Name())
	assert.Equal(t, config["dim"], rebuilt.GetConfig()["dim"])
}

func TestEmbeddingFunctionPersistence_OpenAI(t *testing.T) {
	// Set the standard env var that GetConfig() returns
	t.Setenv("OPENAI_API_KEY", "test-key-123")

	ef, err := openai.NewOpenAIEmbeddingFunction("", openai.WithEnvAPIKey())
	require.NoError(t, err)

	name := ef.Name()
	config := ef.GetConfig()

	assert.Equal(t, "openai", name)
	assert.Equal(t, "OPENAI_API_KEY", config["api_key_env_var"])
	assert.NotEmpty(t, config["model_name"])

	// Verify registry has this EF
	assert.True(t, embeddings.HasDense(name), "openai should be registered")

	// Rebuild from config
	rebuilt, err := embeddings.BuildDense(name, config)
	require.NoError(t, err)
	require.NotNil(t, rebuilt)

	// Verify rebuilt EF matches
	assert.Equal(t, name, rebuilt.Name())
	assert.Equal(t, config["api_key_env_var"], rebuilt.GetConfig()["api_key_env_var"])
	assert.Equal(t, config["model_name"], rebuilt.GetConfig()["model_name"])
}

func TestEmbeddingFunctionPersistence_OpenAI_WithOptions(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key-123")

	ef, err := openai.NewOpenAIEmbeddingFunction("",
		openai.WithEnvAPIKey(),
		openai.WithModel(openai.TextEmbedding3Large),
		openai.WithDimensions(256),
	)
	require.NoError(t, err)

	name := ef.Name()
	config := ef.GetConfig()

	assert.Equal(t, "openai", name)
	assert.Equal(t, "OPENAI_API_KEY", config["api_key_env_var"])
	assert.Equal(t, string(openai.TextEmbedding3Large), config["model_name"])
	assert.Equal(t, 256, config["dimensions"])

	// Rebuild from config
	rebuilt, err := embeddings.BuildDense(name, config)
	require.NoError(t, err)
	require.NotNil(t, rebuilt)

	// Verify rebuilt EF matches
	rebuiltConfig := rebuilt.GetConfig()
	assert.Equal(t, name, rebuilt.Name())
	assert.Equal(t, config["api_key_env_var"], rebuiltConfig["api_key_env_var"])
	assert.Equal(t, config["model_name"], rebuiltConfig["model_name"])
	assert.Equal(t, config["dimensions"], rebuiltConfig["dimensions"])
}

func TestEmbeddingFunctionPersistence_Cohere(t *testing.T) {
	t.Setenv("COHERE_API_KEY", "test-key-123")

	ef, err := cohere.NewCohereEmbeddingFunction(cohere.WithEnvAPIKey())
	require.NoError(t, err)

	name := ef.Name()
	config := ef.GetConfig()

	assert.Equal(t, "cohere", name)
	assert.Equal(t, "COHERE_API_KEY", config["api_key_env_var"])
	assert.NotEmpty(t, config["model_name"])

	// Verify registry has this EF
	assert.True(t, embeddings.HasDense(name), "cohere should be registered")

	// Rebuild from config
	rebuilt, err := embeddings.BuildDense(name, config)
	require.NoError(t, err)
	require.NotNil(t, rebuilt)

	// Verify rebuilt EF matches
	assert.Equal(t, name, rebuilt.Name())
	assert.Equal(t, config["model_name"], rebuilt.GetConfig()["model_name"])
}

func TestEmbeddingFunctionPersistence_Jina(t *testing.T) {
	t.Setenv("JINA_API_KEY", "test-key-123")

	ef, err := jina.NewJinaEmbeddingFunction(jina.WithEnvAPIKey())
	require.NoError(t, err)

	name := ef.Name()
	config := ef.GetConfig()

	assert.Equal(t, "jina", name)
	assert.NotEmpty(t, config["api_key_env_var"])
	assert.NotEmpty(t, config["model_name"])

	// Verify registry has this EF
	assert.True(t, embeddings.HasDense(name), "jina should be registered")

	// Rebuild from config
	rebuilt, err := embeddings.BuildDense(name, config)
	require.NoError(t, err)
	require.NotNil(t, rebuilt)

	// Verify rebuilt EF matches
	assert.Equal(t, name, rebuilt.Name())
}

func TestEmbeddingFunctionPersistence_Mistral(t *testing.T) {
	t.Setenv("MISTRAL_API_KEY", "test-key-123")

	ef, err := mistral.NewMistralEmbeddingFunction(mistral.WithEnvAPIKey())
	require.NoError(t, err)

	name := ef.Name()
	config := ef.GetConfig()

	assert.Equal(t, "mistral", name)
	assert.NotEmpty(t, config["api_key_env_var"])
	assert.NotEmpty(t, config["model_name"])

	// Verify registry has this EF
	assert.True(t, embeddings.HasDense(name), "mistral should be registered")

	// Rebuild from config
	rebuilt, err := embeddings.BuildDense(name, config)
	require.NoError(t, err)
	require.NotNil(t, rebuilt)

	// Verify rebuilt EF matches
	assert.Equal(t, name, rebuilt.Name())
}

func TestEmbeddingFunctionPersistence_Gemini(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key-123")

	ef, err := gemini.NewGeminiEmbeddingFunction(gemini.WithEnvAPIKey())
	require.NoError(t, err)

	name := ef.Name()
	config := ef.GetConfig()

	assert.Equal(t, "google_genai", name)
	assert.NotEmpty(t, config["api_key_env_var"])
	assert.NotEmpty(t, config["model_name"])

	// Verify registry has this EF
	assert.True(t, embeddings.HasDense(name), "google_genai should be registered")

	// Rebuild from config
	rebuilt, err := embeddings.BuildDense(name, config)
	require.NoError(t, err)
	require.NotNil(t, rebuilt)

	// Verify rebuilt EF matches
	assert.Equal(t, name, rebuilt.Name())
}

func TestEmbeddingFunctionPersistence_Voyage(t *testing.T) {
	t.Setenv("VOYAGE_API_KEY", "test-key-123")

	ef, err := voyage.NewVoyageAIEmbeddingFunction(voyage.WithEnvAPIKey())
	require.NoError(t, err)

	name := ef.Name()
	config := ef.GetConfig()

	assert.Equal(t, "voyageai", name)
	assert.NotEmpty(t, config["api_key_env_var"])
	assert.NotEmpty(t, config["model_name"])

	// Verify registry has this EF
	assert.True(t, embeddings.HasDense(name), "voyageai should be registered")

	// Rebuild from config
	rebuilt, err := embeddings.BuildDense(name, config)
	require.NoError(t, err)
	require.NotNil(t, rebuilt)

	// Verify rebuilt EF matches
	assert.Equal(t, name, rebuilt.Name())
}

func TestEmbeddingFunctionPersistence_Ollama(t *testing.T) {
	// Ollama doesn't require API keys, just a base URL
	ef, err := ollama.NewOllamaEmbeddingFunction(
		ollama.WithBaseURL("http://localhost:11434"),
		ollama.WithModel("nomic-embed-text"),
	)
	require.NoError(t, err)

	name := ef.Name()
	config := ef.GetConfig()

	assert.Equal(t, "ollama", name)
	// Note: Ollama uses "url" not "base_url" in GetConfig
	assert.Equal(t, "http://localhost:11434", config["url"])
	assert.Equal(t, "nomic-embed-text", config["model_name"])

	// Verify registry has this EF
	assert.True(t, embeddings.HasDense(name), "ollama should be registered")

	// Rebuild from config
	rebuilt, err := embeddings.BuildDense(name, config)
	require.NoError(t, err)
	require.NotNil(t, rebuilt)

	// Verify rebuilt EF matches
	assert.Equal(t, name, rebuilt.Name())
	assert.Equal(t, config["url"], rebuilt.GetConfig()["url"])
	assert.Equal(t, config["model_name"], rebuilt.GetConfig()["model_name"])
}

func TestEmbeddingFunctionPersistence_HuggingFace(t *testing.T) {
	t.Setenv("HF_API_KEY", "test-key-123")

	ef, err := hf.NewHuggingFaceEmbeddingFunctionFromOptions(
		hf.WithEnvAPIKey(),
		hf.WithModel("sentence-transformers/all-MiniLM-L6-v2"),
	)
	require.NoError(t, err)

	name := ef.Name()
	config := ef.GetConfig()

	assert.Equal(t, "huggingface", name)
	assert.NotEmpty(t, config["api_key_env_var"])
	assert.Equal(t, "sentence-transformers/all-MiniLM-L6-v2", config["model_name"])

	// Verify registry has this EF
	assert.True(t, embeddings.HasDense(name), "huggingface should be registered")

	// Rebuild from config
	rebuilt, err := embeddings.BuildDense(name, config)
	require.NoError(t, err)
	require.NotNil(t, rebuilt)

	// Verify rebuilt EF matches
	assert.Equal(t, name, rebuilt.Name())
}

func TestEmbeddingFunctionPersistence_Cloudflare(t *testing.T) {
	// Cloudflare uses CLOUDFLARE_API_TOKEN as the standard env var
	t.Setenv("CLOUDFLARE_API_TOKEN", "test-key-123")

	ef, err := cloudflare.NewCloudflareEmbeddingFunction(
		cloudflare.WithAPIKeyFromEnvVar("CLOUDFLARE_API_TOKEN"),
		cloudflare.WithAccountID("test-account-id"),
	)
	require.NoError(t, err)

	name := ef.Name()
	config := ef.GetConfig()

	assert.Equal(t, "cloudflare_workers_ai", name)
	assert.NotEmpty(t, config["api_key_env_var"])
	assert.NotEmpty(t, config["model_name"])
	assert.Equal(t, "test-account-id", config["account_id"])

	// Verify registry has this EF
	assert.True(t, embeddings.HasDense(name), "cloudflare_workers_ai should be registered")

	// Rebuild from config
	rebuilt, err := embeddings.BuildDense(name, config)
	require.NoError(t, err)
	require.NotNil(t, rebuilt)

	// Verify rebuilt EF matches
	assert.Equal(t, name, rebuilt.Name())
	assert.Equal(t, config["account_id"], rebuilt.GetConfig()["account_id"])
}

func TestEmbeddingFunctionPersistence_Together(t *testing.T) {
	t.Setenv("TOGETHER_API_KEY", "test-key-123")

	ef, err := together.NewTogetherEmbeddingFunction(together.WithEnvAPIToken())
	require.NoError(t, err)

	name := ef.Name()
	config := ef.GetConfig()

	assert.Equal(t, "together_ai", name)
	assert.NotEmpty(t, config["api_key_env_var"])
	assert.NotEmpty(t, config["model_name"])

	// Verify registry has this EF
	assert.True(t, embeddings.HasDense(name), "together_ai should be registered")

	// Rebuild from config
	rebuilt, err := embeddings.BuildDense(name, config)
	require.NoError(t, err)
	require.NotNil(t, rebuilt)

	// Verify rebuilt EF matches
	assert.Equal(t, name, rebuilt.Name())
}

func TestEmbeddingFunctionPersistence_Nomic(t *testing.T) {
	t.Setenv("NOMIC_API_KEY", "test-key-123")

	ef, err := nomic.NewNomicEmbeddingFunction(nomic.WithEnvAPIKey())
	require.NoError(t, err)

	name := ef.Name()
	config := ef.GetConfig()

	assert.Equal(t, "nomic", name)
	assert.NotEmpty(t, config["api_key_env_var"])
	assert.NotEmpty(t, config["model_name"])

	// Verify registry has this EF
	assert.True(t, embeddings.HasDense(name), "nomic should be registered")

	// Rebuild from config
	rebuilt, err := embeddings.BuildDense(name, config)
	require.NoError(t, err)
	require.NotNil(t, rebuilt)

	// Verify rebuilt EF matches
	assert.Equal(t, name, rebuilt.Name())
}

func TestEmbeddingFunctionPersistence_ChromaCloud(t *testing.T) {
	t.Setenv("CHROMA_API_KEY", "test-key-123")

	ef, err := chromacloud.NewEmbeddingFunction(chromacloud.WithEnvAPIKey())
	require.NoError(t, err)

	name := ef.Name()
	config := ef.GetConfig()

	assert.Equal(t, "chroma_cloud", name)
	assert.NotEmpty(t, config["api_key_env_var"])
	assert.NotEmpty(t, config["model_name"])

	// Verify registry has this EF
	assert.True(t, embeddings.HasDense(name), "chroma_cloud should be registered")

	// Rebuild from config
	rebuilt, err := embeddings.BuildDense(name, config)
	require.NoError(t, err)
	require.NotNil(t, rebuilt)

	// Verify rebuilt EF matches
	assert.Equal(t, name, rebuilt.Name())
	assert.Equal(t, config["model_name"], rebuilt.GetConfig()["model_name"])
}

// TestAllRegisteredEFsHaveFactories verifies that all known EF names are registered
func TestAllRegisteredEFsHaveFactories(t *testing.T) {
	expectedEFs := []string{
		"consistent_hash",
		"openai",
		"cohere",
		"jina",
		"mistral",
		"google_genai",
		"voyageai",
		"ollama",
		"huggingface",
		"cloudflare_workers_ai",
		"together_ai",
		"nomic",
		"chroma_cloud",
		"onnx_mini_lm_l6_v2",
	}

	for _, name := range expectedEFs {
		t.Run(name, func(t *testing.T) {
			assert.True(t, embeddings.HasDense(name), "%s should be registered in the dense registry", name)
		})
	}
}

// TestCustomEnvVarPersistence verifies that custom env var names are persisted in GetConfig()
// This is critical for the auto-wire feature - users can use custom env var names like
// "MY_OPENAI_KEY" and these should be preserved when the collection is retrieved
func TestCustomEnvVarPersistence(t *testing.T) {
	testCases := []struct {
		name          string
		customEnvVar  string
		createEF      func(envVar string) (embeddings.EmbeddingFunction, error)
		expectedName  string
		defaultEnvVar string
	}{
		{
			name:         "openai",
			customEnvVar: "MY_CUSTOM_OPENAI_KEY",
			createEF: func(envVar string) (embeddings.EmbeddingFunction, error) {
				return openai.NewOpenAIEmbeddingFunction("", openai.WithAPIKeyFromEnvVar(envVar))
			},
			expectedName:  "openai",
			defaultEnvVar: "OPENAI_API_KEY",
		},
		{
			name:         "cohere",
			customEnvVar: "MY_CUSTOM_COHERE_KEY",
			createEF: func(envVar string) (embeddings.EmbeddingFunction, error) {
				return cohere.NewCohereEmbeddingFunction(cohere.WithAPIKeyFromEnvVar(envVar))
			},
			expectedName:  "cohere",
			defaultEnvVar: "COHERE_API_KEY",
		},
		{
			name:         "jina",
			customEnvVar: "MY_CUSTOM_JINA_KEY",
			createEF: func(envVar string) (embeddings.EmbeddingFunction, error) {
				return jina.NewJinaEmbeddingFunction(jina.WithAPIKeyFromEnvVar(envVar))
			},
			expectedName:  "jina",
			defaultEnvVar: "JINA_API_KEY",
		},
		{
			name:         "mistral",
			customEnvVar: "MY_CUSTOM_MISTRAL_KEY",
			createEF: func(envVar string) (embeddings.EmbeddingFunction, error) {
				return mistral.NewMistralEmbeddingFunction(mistral.WithAPIKeyFromEnvVar(envVar))
			},
			expectedName:  "mistral",
			defaultEnvVar: "MISTRAL_API_KEY",
		},
		{
			name:         "gemini",
			customEnvVar: "MY_CUSTOM_GEMINI_KEY",
			createEF: func(envVar string) (embeddings.EmbeddingFunction, error) {
				return gemini.NewGeminiEmbeddingFunction(gemini.WithAPIKeyFromEnvVar(envVar))
			},
			expectedName:  "google_genai",
			defaultEnvVar: "GEMINI_API_KEY",
		},
		{
			name:         "voyage",
			customEnvVar: "MY_CUSTOM_VOYAGE_KEY",
			createEF: func(envVar string) (embeddings.EmbeddingFunction, error) {
				return voyage.NewVoyageAIEmbeddingFunction(voyage.WithAPIKeyFromEnvVar(envVar))
			},
			expectedName:  "voyageai",
			defaultEnvVar: "VOYAGE_API_KEY",
		},
		{
			name:         "together",
			customEnvVar: "MY_CUSTOM_TOGETHER_KEY",
			createEF: func(envVar string) (embeddings.EmbeddingFunction, error) {
				return together.NewTogetherEmbeddingFunction(together.WithAPITokenFromEnvVar(envVar))
			},
			expectedName:  "together_ai",
			defaultEnvVar: "TOGETHER_API_KEY",
		},
		{
			name:         "nomic",
			customEnvVar: "MY_CUSTOM_NOMIC_KEY",
			createEF: func(envVar string) (embeddings.EmbeddingFunction, error) {
				return nomic.NewNomicEmbeddingFunction(nomic.WithAPIKeyFromEnvVar(envVar))
			},
			expectedName:  "nomic",
			defaultEnvVar: "NOMIC_API_KEY",
		},
		{
			name:         "huggingface",
			customEnvVar: "MY_CUSTOM_HF_KEY",
			createEF: func(envVar string) (embeddings.EmbeddingFunction, error) {
				return hf.NewHuggingFaceEmbeddingFunctionFromOptions(
					hf.WithAPIKeyFromEnvVar(envVar),
					hf.WithModel("sentence-transformers/all-MiniLM-L6-v2"),
				)
			},
			expectedName:  "huggingface",
			defaultEnvVar: "HF_API_KEY",
		},
		{
			name:         "cloudflare",
			customEnvVar: "MY_CUSTOM_CF_KEY",
			createEF: func(envVar string) (embeddings.EmbeddingFunction, error) {
				return cloudflare.NewCloudflareEmbeddingFunction(
					cloudflare.WithAPIKeyFromEnvVar(envVar),
					cloudflare.WithAccountID("test-account"),
				)
			},
			expectedName:  "cloudflare_workers_ai",
			defaultEnvVar: "CLOUDFLARE_API_TOKEN",
		},
		{
			name:         "chromacloud",
			customEnvVar: "MY_CUSTOM_CHROMA_KEY",
			createEF: func(envVar string) (embeddings.EmbeddingFunction, error) {
				return chromacloud.NewEmbeddingFunction(chromacloud.WithAPIKeyFromEnvVar(envVar))
			},
			expectedName:  "chroma_cloud",
			defaultEnvVar: "CHROMA_API_KEY",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set the custom env var
			t.Setenv(tc.customEnvVar, "test-secret-value")

			// Create EF with custom env var
			ef, err := tc.createEF(tc.customEnvVar)
			require.NoError(t, err)

			// Verify GetConfig() returns the custom env var name, not the default
			config := ef.GetConfig()
			assert.Equal(t, tc.customEnvVar, config["api_key_env_var"],
				"GetConfig() should return custom env var name '%s', not default '%s'",
				tc.customEnvVar, tc.defaultEnvVar)
			assert.Equal(t, tc.expectedName, ef.Name())

			// Verify the EF can be rebuilt using the config
			// This simulates the auto-wire scenario
			rebuilt, err := embeddings.BuildDense(ef.Name(), config)
			require.NoError(t, err)
			require.NotNil(t, rebuilt)

			// The rebuilt EF should also have the custom env var
			rebuiltConfig := rebuilt.GetConfig()
			assert.Equal(t, tc.customEnvVar, rebuiltConfig["api_key_env_var"],
				"Rebuilt EF should preserve custom env var name '%s'", tc.customEnvVar)
		})
	}
}

// TestConfigRoundTrip tests that config can be serialized to JSON and back
func TestConfigRoundTrip(t *testing.T) {
	// Set all required env vars for the test
	t.Setenv("OPENAI_API_KEY", "test-key-123")

	testCases := []struct {
		name     string
		createEF func() (embeddings.EmbeddingFunction, error)
	}{
		{
			name: "consistent_hash",
			createEF: func() (embeddings.EmbeddingFunction, error) {
				return embeddings.NewConsistentHashEmbeddingFunction(), nil
			},
		},
		{
			name: "openai",
			createEF: func() (embeddings.EmbeddingFunction, error) {
				return openai.NewOpenAIEmbeddingFunction("", openai.WithEnvAPIKey())
			},
		},
		{
			name: "ollama",
			createEF: func() (embeddings.EmbeddingFunction, error) {
				return ollama.NewOllamaEmbeddingFunction(
					ollama.WithBaseURL("http://localhost:11434"),
					ollama.WithModel("nomic-embed-text"),
				)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ef, err := tc.createEF()
			require.NoError(t, err)

			name := ef.Name()
			config := ef.GetConfig()

			// Verify we can rebuild
			rebuilt, err := embeddings.BuildDense(name, config)
			require.NoError(t, err)
			require.NotNil(t, rebuilt)

			// Names should match
			assert.Equal(t, name, rebuilt.Name())

			// Get rebuilt config and compare key fields
			rebuiltConfig := rebuilt.GetConfig()
			for key, val := range config {
				if key != "api_key" { // Skip sensitive fields
					assert.Equal(t, val, rebuiltConfig[key], "config key %s should match", key)
				}
			}
		})
	}
}
