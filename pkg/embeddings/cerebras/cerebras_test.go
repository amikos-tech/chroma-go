//go:build ef

package cerebras

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCerebrasAPI is a helper to create a mock HTTP server for Cerebras API calls.
func mockCerebrasAPI(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestCerebrasEmbeddingFunction_EmbedDocuments_Success(t *testing.T) {
	var turn1Request ChatCompletionRequest
	var turn2Request ChatCompletionRequest
	requestCount := 0

	doc1 := "First test document"
	doc2 := "Second test document"
	expectedDocs := []string{doc1, doc2}
	expectedTaskType := "test_retrieval_doc"
	expectedInstruction := "Generate test embeddings for semantic search."
	expectedNormalize := true

	expectedEmbedding1 := []float32{0.11, 0.22, 0.33}
	expectedEmbedding2 := []float32{0.44, 0.55, 0.66}

	server := mockCerebrasAPI(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		bodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		if requestCount == 1 { // Turn 1: Tool call request
			t.Logf("Mock Server: Received Turn 1 request (Tool Call)")
			err = json.Unmarshal(bodyBytes, &turn1Request)
			require.NoError(t, err)

			assert.Len(t, turn1Request.Messages, 2)
			assert.Equal(t, "system", turn1Request.Messages[0].Role)
			assert.Equal(t, "user", turn1Request.Messages[1].Role)
			assert.Contains(t, turn1Request.Messages[1].Content, fmt.Sprintf("TaskType: '%s'", expectedTaskType))
			assert.Contains(t, turn1Request.Messages[1].Content, fmt.Sprintf("Instruction: '%s'", expectedInstruction))
			assert.Contains(t, turn1Request.Messages[1].Content, fmt.Sprintf("NormalizeOutput: %v", expectedNormalize))
			assert.Contains(t, turn1Request.Messages[1].Content, doc1)
			assert.Contains(t, turn1Request.Messages[1].Content, doc2)

			require.NotNil(t, turn1Request.Tools)
			require.Len(t, turn1Request.Tools, 1)
			assert.Equal(t, EmbeddingToolName, turn1Request.Tools[0].Function.Name)

			toolCallArgs := EmbeddingToolParameters{
				InputTexts:      expectedDocs,
				TaskType:        expectedTaskType,
				Instruction:     expectedInstruction,
				NormalizeOutput: expectedNormalize,
			}
			argsBytes, _ := json.Marshal(toolCallArgs)

			resp := ChatCompletionResponse{
				ID: "chatcmpl-turn1-success",
				Choices: []ChatCompletionChoice{{
					Message: ChatMessage{Role: "assistant", ToolCalls: []ToolCall{{
						ID: "call_success123", Type: "function", Function: ToolCallFunction{Name: EmbeddingToolName, Arguments: string(argsBytes)},
					}}},
					FinishReason: "tool_calls",
				}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		} else if requestCount == 2 { // Turn 2: Embedding generation request
			t.Logf("Mock Server: Received Turn 2 request (Embedding Generation)")
			err = json.Unmarshal(bodyBytes, &turn2Request)
			require.NoError(t, err)

			assert.Len(t, turn2Request.Messages, 4) // system, user(orig), assistant(tool_call), user(new_instr)
			assert.Equal(t, "user", turn2Request.Messages[3].Role)
			assert.Contains(t, turn2Request.Messages[3].Content, "You have decided to call the 'advanced_embedding_generator' tool")
			assert.Contains(t, turn2Request.Messages[3].Content, fmt.Sprintf("TaskType='%s'", expectedTaskType))

			require.NotNil(t, turn2Request.ResponseFormat)
			assert.Equal(t, "json_schema", turn2Request.ResponseFormat.Type)
			assert.Equal(t, "embedding_tool_output_schema", turn2Request.ResponseFormat.JSONSchema.Name)

			embeddingOutput := EmbeddingToolOutput{
				Results: []EmbeddingResultItem{
					{SourceText: doc1, EmbeddingVector: expectedEmbedding1, Normalized: expectedNormalize},
					{SourceText: doc2, EmbeddingVector: expectedEmbedding2, Normalized: expectedNormalize},
				},
				ModelUsed: string(DefaultChatModel),
				UsageInfo: map[string]int{"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30},
			}
			outputBytes, _ := json.Marshal(embeddingOutput)

			resp := ChatCompletionResponse{
				ID: "chatcmpl-turn2-success",
				Choices: []ChatCompletionChoice{{
					Message:      ChatMessage{Role: "assistant", Content: string(outputBytes)},
					FinishReason: "stop",
				}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		} else {
			t.Fatalf("Unexpected request count: %d", requestCount)
		}
	})
	defer server.Close()

	// Temporarily set the environment variable for this test
	// The actual value "env-api-key-for-test" doesn't matter for the mock server,
	// but it allows us to verify the client would pick it up.
	t.Setenv(DefaultAPIKeyEnvVarName, "env-api-key-for-test")

	t.Logf("Test: Initializing EmbeddingFunction with BaseURL: %s", server.URL)
	ef, err := NewEmbeddingFunction(
		"", // Direct apiKey is now less relevant as WithAPIKeyFromEnv will override
		[]Option{WithBaseURL(server.URL), WithDefaultModel(DefaultChatModel), WithAPIKeyFromEnv(DefaultAPIKeyEnvVarName)},
		WithTaskType(expectedTaskType),
		WithInstruction(expectedInstruction),
		WithNormalizeOutput(expectedNormalize),
	)
	require.NoError(t, err)

	t.Logf("Test: Calling EmbedDocuments with %d documents", len(expectedDocs))
	embeddingsResult, err := ef.EmbedDocuments(context.Background(), expectedDocs)
	t.Logf("Test: EmbedDocuments call completed. Error: %v", err)
	require.NoError(t, err)
	// You can optionally assert that the client's APIKey was indeed set from the env var
	assert.Equal(t, "env-api-key-for-test", ef.client.APIKey, "Client APIKey should be set from environment variable")

	require.Equal(t, 2, requestCount, "Expected two API calls")

	require.Len(t, embeddingsResult, 2)
	t.Logf("Test: Received %d embeddings. First embedding length: %d, Second embedding length: %d", len(embeddingsResult), embeddingsResult[0].Len(), embeddingsResult[1].Len())
	assert.Equal(t, expectedEmbedding1, embeddingsResult[0].ContentAsFloat32())
	assert.Equal(t, expectedEmbedding2, embeddingsResult[1].ContentAsFloat32())
}

func TestCerebrasEmbeddingFunction_EmbedQuery_Success(t *testing.T) {
	requestCount := 0
	queryDoc := "Single query document for testing"
	expectedEmbedding := []float32{0.77, 0.88, 0.99}

	server := mockCerebrasAPI(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		bodyBytes, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		if requestCount == 1 { // Turn 1
			var req ChatCompletionRequest
			json.Unmarshal(bodyBytes, &req)
			assert.Contains(t, req.Messages[1].Content, queryDoc)

			toolCallArgs := EmbeddingToolParameters{InputTexts: []string{queryDoc}} // Simplified for test
			argsBytes, _ := json.Marshal(toolCallArgs)
			resp := ChatCompletionResponse{
				Choices: []ChatCompletionChoice{{
					Message:      ChatMessage{Role: "assistant", ToolCalls: []ToolCall{{ID: "call_q1", Type: "function", Function: ToolCallFunction{Name: EmbeddingToolName, Arguments: string(argsBytes)}}}},
					FinishReason: "tool_calls",
				}},
			}
			json.NewEncoder(w).Encode(resp)
		} else if requestCount == 2 { // Turn 2
			embeddingOutput := EmbeddingToolOutput{
				Results:   []EmbeddingResultItem{{SourceText: queryDoc, EmbeddingVector: expectedEmbedding, Normalized: false}},
				ModelUsed: string(DefaultChatModel),
			}
			outputBytes, _ := json.Marshal(embeddingOutput)
			resp := ChatCompletionResponse{
				Choices: []ChatCompletionChoice{{
					Message:      ChatMessage{Role: "assistant", Content: string(outputBytes)},
					FinishReason: "stop",
				}},
			}
			json.NewEncoder(w).Encode(resp)
		}
	})
	defer server.Close()

	ef, err := NewEmbeddingFunction("dummy-api-key", []Option{WithBaseURL(server.URL)})
	require.NoError(t, err)

	embeddingResult, err := ef.EmbedQuery(context.Background(), queryDoc)
	require.NoError(t, err)
	require.Equal(t, 2, requestCount, "Expected two API calls for EmbedQuery")
	require.NotNil(t, embeddingResult)
	assert.Equal(t, expectedEmbedding, embeddingResult.ContentAsFloat32())
}

func TestCerebrasEmbeddingFunction_EmbedDocuments_EmptyInput(t *testing.T) {
	ef, err := NewEmbeddingFunction("dummy-key", nil)
	require.NoError(t, err)

	results, err := ef.EmbedDocuments(context.Background(), []string{})
	require.NoError(t, err)
	require.NotNil(t, results)
	assert.Empty(t, results)
	assert.Len(t, results, 0)
}

func TestCerebrasEmbeddingFunction_EmbedDocuments_Turn1_NoToolCall(t *testing.T) {
	server := mockCerebrasAPI(t, func(w http.ResponseWriter, r *http.Request) {
		resp := ChatCompletionResponse{
			Choices: []ChatCompletionChoice{{
				Message:      ChatMessage{Role: "assistant", Content: "I cannot use tools right now."},
				FinishReason: "stop",
			}},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	ef, _ := NewEmbeddingFunction("dummy", []Option{WithBaseURL(server.URL)})
	_, err := ef.EmbedDocuments(context.Background(), []string{"test doc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LLM did not call the embedding tool as expected")
}

func TestCerebrasEmbeddingFunction_EmbedDocuments_Turn2_BadJSONOutput(t *testing.T) {
	requestCount := 0
	server := mockCerebrasAPI(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 { // Turn 1 - successful tool call
			toolCallArgs := EmbeddingToolParameters{InputTexts: []string{"test doc"}}
			argsBytes, _ := json.Marshal(toolCallArgs)
			resp := ChatCompletionResponse{
				Choices: []ChatCompletionChoice{{
					Message:      ChatMessage{Role: "assistant", ToolCalls: []ToolCall{{ID: "call_badjson", Type: "function", Function: ToolCallFunction{Name: EmbeddingToolName, Arguments: string(argsBytes)}}}},
					FinishReason: "tool_calls",
				}},
			}
			json.NewEncoder(w).Encode(resp)
		} else { // Turn 2 - LLM returns malformed JSON
			resp := ChatCompletionResponse{
				Choices: []ChatCompletionChoice{{
					Message:      ChatMessage{Role: "assistant", Content: "this is not valid json { definitely not"},
					FinishReason: "stop",
				}},
			}
			json.NewEncoder(w).Encode(resp)
		}
	})
	defer server.Close()

	ef, _ := NewEmbeddingFunction("dummy", []Option{WithBaseURL(server.URL)})
	_, err := ef.EmbedDocuments(context.Background(), []string{"test doc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "turn 2: failed to unmarshal LLM response into EmbeddingToolOutput")
}
