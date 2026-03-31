//go:build ef

package openrouter

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defaultOpenRouterTestModel = "openai/text-embedding-3-small"
	openrouterTestModelEnvVar  = "OPENROUTER_TEST_MODEL"
)

func openrouterTestModel() string {
	if model := os.Getenv(openrouterTestModelEnvVar); model != "" {
		return model
	}
	return defaultOpenRouterTestModel
}

func requireOpenRouterSuccessOrSkip(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "service_unavailable") ||
		strings.Contains(msg, "Service unavailable") ||
		strings.Contains(msg, "503 Service Unavailable") ||
		strings.Contains(msg, "rate_limit") ||
		strings.Contains(msg, "429") {
		t.Skipf("Skipping test due to OpenRouter service availability: %v", err)
	}
	require.NoError(t, err)
}

func loadOpenRouterAPIKey(t *testing.T) {
	t.Helper()
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}
}

func TestIntegration_CreateEmbedding(t *testing.T) {
	loadOpenRouterAPIKey(t)
	client, err := NewOpenRouterClient(
		WithEnvAPIKey(),
		WithModel(openrouterTestModel()),
	)
	require.NoError(t, err)

	resp, err := client.CreateEmbedding(context.Background(), &CreateEmbeddingRequest{
		Model: openrouterTestModel(),
		Input: &Input{Text: "Test document"},
	})
	requireOpenRouterSuccessOrSkip(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.Data)
	require.Greater(t, len(resp.Data[0].Embedding), 0)
}

func TestIntegration_EmbedDocuments(t *testing.T) {
	loadOpenRouterAPIKey(t)

	ef, err := NewOpenRouterEmbeddingFunction(
		WithEnvAPIKey(),
		WithModel(openrouterTestModel()),
	)
	require.NoError(t, err)

	resp, err := ef.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})
	requireOpenRouterSuccessOrSkip(t, err)
	require.Len(t, resp, 2)
	require.Greater(t, resp[0].Len(), 0)
	require.Greater(t, resp[1].Len(), 0)
}

func TestIntegration_EmbedQuery(t *testing.T) {
	loadOpenRouterAPIKey(t)

	ef, err := NewOpenRouterEmbeddingFunction(
		WithEnvAPIKey(),
		WithModel(openrouterTestModel()),
	)
	require.NoError(t, err)

	resp, err := ef.EmbedQuery(context.Background(), "Test query")
	requireOpenRouterSuccessOrSkip(t, err)
	require.NotNil(t, resp)
	require.Greater(t, resp.Len(), 0)
}

func TestIntegration_EmbedDocumentsEmpty(t *testing.T) {
	loadOpenRouterAPIKey(t)

	ef, err := NewOpenRouterEmbeddingFunction(
		WithEnvAPIKey(),
		WithModel(openrouterTestModel()),
	)
	require.NoError(t, err)

	resp, err := ef.EmbedDocuments(context.Background(), []string{})
	require.NoError(t, err)
	require.Empty(t, resp)
}

func TestIntegration_ProviderPreferences(t *testing.T) {
	loadOpenRouterAPIKey(t)

	ef, err := NewOpenRouterEmbeddingFunction(
		WithEnvAPIKey(),
		WithModel(openrouterTestModel()),
		WithProviderPreferences(&ProviderPreferences{
			Order:          []string{"OpenAI"},
			AllowFallbacks: boolPtr(true),
		}),
	)
	require.NoError(t, err)

	resp, err := ef.EmbedQuery(context.Background(), "Test with provider preferences")
	requireOpenRouterSuccessOrSkip(t, err)
	require.NotNil(t, resp)
	require.Greater(t, resp.Len(), 0)
}

func TestIntegration_WithExplicitAPIKey(t *testing.T) {
	loadOpenRouterAPIKey(t)

	ef, err := NewOpenRouterEmbeddingFunction(
		WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		WithModel(openrouterTestModel()),
	)
	require.NoError(t, err)

	resp, err := ef.EmbedDocuments(context.Background(), []string{"Test document"})
	requireOpenRouterSuccessOrSkip(t, err)
	require.Len(t, resp, 1)
	require.Greater(t, resp[0].Len(), 0)
}
