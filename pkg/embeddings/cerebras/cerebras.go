package cerebras

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	chttp "github.com/guiperry/chroma-go_cerebras/pkg/commons/http"
	"github.com/guiperry/chroma-go_cerebras/pkg/embeddings"
)

// CerebrasModel represents the available LLM models from Cerebras.
type CerebrasModel string

const (
	DefaultBaseURL = "https://api.cerebras.ai/v1/chat/completions" // Example, adjust. This should point to the chat/completions endpoint base.

	Llama4Scout17B16EInstruct CerebrasModel = "llama-4-scout-17b-16e-instruct"
	Llama3_1_8B               CerebrasModel = "llama3.1-8b"
	Llama3_3_70B              CerebrasModel = "llama-3.3-70b"
	Qwen3_32B                 CerebrasModel = "qwen-3-32b"

	DefaultChatModel             = Llama3_1_8B // Choose a default model
	EmbeddingToolName            = "advanced_embedding_generator"
	DefaultEmbeddingTaskType     = "retrieval_document"
	DefaultEmbeddingInstruction  = "Generate a dense vector representation of the text suitable for semantic search."
	DefaultEmbeddingMaxTokens    = 512 // Max tokens for the LLM's response containing the embedding JSON
)

// Client is the Cerebras API client for chat completions with tool use.
type Client struct {
	BaseURL        string
	APIKey         string
	DefaultModel   embeddings.EmbeddingModel
	HTTPClient     *http.Client
	DefaultHeaders map[string]string
}

// Option is a function that configures a Client.
type Option func(*Client) error

// NewClient creates a new Cerebras client.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	c := &Client{
		APIKey:       apiKey,
		BaseURL:      DefaultBaseURL,
		DefaultModel: embeddings.EmbeddingModel(DefaultChatModel),
		HTTPClient:   &http.Client{},
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, errors.Wrap(err, "failed to apply Cerebras option")
		}
	}
	if err := c.validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate Cerebras client options")
	}
	return c, nil
}

func (c *Client) validate() error { /* ... same as before ... */
	if c.APIKey == "" {
		return errors.New("API key is required")
	}
	if c.BaseURL == "" {
		return errors.New("base URL is required")
	}
	if _, err := url.ParseRequestURI(c.BaseURL); err != nil {
		return errors.Wrap(err, "invalid base URL")
	}
	return nil
}

// --- Cerebras API Request/Response Structs (Chat Completions & Tools) ---
type ChatMessage struct {
	Role       string      `json:"role"` // "system", "user", "assistant", "tool"
	Content    string      `json:"content"`
	ToolCallID string      `json:"tool_call_id,omitempty"` // For role: "tool"
	Name       string      `json:"name,omitempty"`         // For role: "tool", the name of the tool
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`   // For role: "assistant" when it calls tools
}

type ToolFunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"` // JSON Schema object
}

type ToolDefinition struct {
	Type     string                 `json:"type"` // "function"
	Function ToolFunctionDefinition `json:"function"`
}

type ToolCall struct {
	ID       string              `json:"id"`
	Type     string              `json:"type"` // "function"
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

type JSONSchemaDefinition struct { /* ... same as before ... */
	Type        string                              `json:"type"`
	Properties  map[string]JSONSchemaDefinition `json:"properties,omitempty"`
	Items       *JSONSchemaDefinition               `json:"items,omitempty"`
	Description string                              `json:"description,omitempty"`
	Required    []string                            `json:"required,omitempty"`
}
type JSONSchemaEnvelope struct { /* ... same as before ... */
	Name   string               `json:"name"`
	Strict bool                 `json:"strict"`
	Schema JSONSchemaDefinition `json:"schema"`
}
type ResponseFormat struct { /* ... same as before ... */
	Type       string             `json:"type"`
	JSONSchema JSONSchemaEnvelope `json:"json_schema"`
}

type ChatCompletionRequest struct {
	Model           string           `json:"model"`
	Messages        []ChatMessage    `json:"messages"`
	Tools           []ToolDefinition `json:"tools,omitempty"`
	ToolChoice      any              `json:"tool_choice,omitempty"` // string ("none", "auto", "required") or object
	ResponseFormat  *ResponseFormat  `json:"response_format,omitempty"`
	Temperature     *float32         `json:"temperature,omitempty"`
	MaxTokens       *int             `json:"max_completion_tokens,omitempty"`
	// ... other parameters
}

type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"` // e.g., "stop", "tool_calls"
}

type ChatCompletionResponse struct { /* ... same as before ... */
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
}

// --- Embedding Tool Specific Structs ---
type EmbeddingToolParameters struct {
	InputTexts      []string `json:"input_texts"`
	TaskType        string   `json:"task_type,omitempty"`
	Instruction     string   `json:"instruction,omitempty"`
	NormalizeOutput bool     `json:"normalize_output,omitempty"`
}

type EmbeddingResultItem struct {
	SourceText      string    `json:"source_text"`
	EmbeddingVector []float32 `json:"embedding_vector"`
	Normalized      bool      `json:"normalized"` // Indicates if the LLM claims to have normalized it
}

type EmbeddingToolOutput struct {
	Results    []EmbeddingResultItem `json:"results"`
	ModelUsed  string                `json:"model_used"` // Model reported by LLM for the embedding task
	UsageInfo  map[string]int        `json:"usage_info,omitempty"` // e.g., {"prompt_tokens": N, "completion_tokens": M, "total_tokens": P}
}

func getEmbeddingToolDefinition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunctionDefinition{
			Name:        EmbeddingToolName,
			Description: "Generates dense vector embeddings for a list of input texts based on specified parameters.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"input_texts": map[string]any{
						"type":        "array",
						"description": "A list of text strings to embed.",
						"items":       map[string]any{"type": "string"},
					},
					"task_type": map[string]any{
						"type":        "string",
						"description": "Type of task for which the embedding is generated (e.g., 'retrieval_document', 'similarity', 'classification'). Defaults to 'retrieval_document'.",
						"default":     DefaultEmbeddingTaskType,
					},
					"instruction": map[string]any{
						"type":        "string",
						"description": "Specific instruction to guide the embedding generation process. Defaults to a general semantic search instruction.",
						"default":     DefaultEmbeddingInstruction,
					},
					"normalize_output": map[string]any{
						"type":        "boolean",
						"description": "Whether the output embedding vectors should be normalized (L2 norm). Defaults to false.",
						"default":     false,
					},
				},
				"required": []string{"input_texts"},
			},
		},
	}
}

func getEmbeddingOutputResponseFormat() *ResponseFormat {
	return &ResponseFormat{
		Type: "json_schema",
		JSONSchema: JSONSchemaEnvelope{
			Name:   "embedding_tool_output_schema",
			Strict: true, // Crucial for reliable parsing
			Schema: JSONSchemaDefinition{
				Type: "object",
				Properties: map[string]JSONSchemaDefinition{
					"results": {
						Type: "array",
						Items: &JSONSchemaDefinition{
							Type: "object",
							Properties: map[string]JSONSchemaDefinition{
								"source_text":      {Type: "string"},
								"embedding_vector": {Type: "array", Items: &JSONSchemaDefinition{Type: "number"}},
								"normalized":       {Type: "boolean"},
							},
							Required: []string{"source_text", "embedding_vector", "normalized"},
						},
					},
					"model_used": {Type: "string"},
					"usage_info": {
						Type: "object",
						Properties: map[string]JSONSchemaDefinition{
							"prompt_tokens":     {Type: "integer"},
							"completion_tokens": {Type: "integer"},
							"total_tokens":      {Type: "integer"},
						},
						// Not strictly required as LLM might not always provide it
					},
				},
				Required: []string{"results", "model_used"},
			},
		},
	}
}

func (c *Client) CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) { /* ... same as before ... */
	if req.Model == "" {
		req.Model = string(c.DefaultModel)
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal chat completion request")
	}
	endpoint, err := url.JoinPath(c.BaseURL, "chat/completions")
	if err != nil {
		return nil, errors.Wrap(err, "failed to join URL path for chat/completions")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", chttp.ChromaGoClientUserAgent)
	for k, v := range c.DefaultHeaders {
		httpReq.Header.Set(k, v)
	}
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request to Cerebras chat API")
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Cerebras chat API request failed with status %s: %s", resp.Status, string(bodyBytes))
	}
	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal Cerebras chat response")
	}
	return &chatResp, nil
}

// --- EmbeddingFunction Implementation ---
var _ embeddings.EmbeddingFunction = (*EmbeddingFunction)(nil)

type EmbeddingFunction struct {
	client          *Client
	taskType        string // Default task type for this function instance
	instruction     string // Default instruction
	normalizeOutput bool   // Default normalization
}

type EmbeddingFunctionOption func(*EmbeddingFunction)

func WithTaskType(taskType string) EmbeddingFunctionOption {
	return func(ef *EmbeddingFunction) { ef.taskType = taskType }
}
func WithInstruction(instruction string) EmbeddingFunctionOption {
	return func(ef *EmbeddingFunction) { ef.instruction = instruction }
}
func WithNormalizeOutput(normalize bool) EmbeddingFunctionOption {
	return func(ef *EmbeddingFunction) { ef.normalizeOutput = normalize }
}

func NewEmbeddingFunction(apiKey string, clientOpts []Option, efOpts ...EmbeddingFunctionOption) (*EmbeddingFunction, error) {
	client, err := NewClient(apiKey, clientOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Cerebras client for embedding function")
	}
	ef := &EmbeddingFunction{
		client:          client,
		taskType:        DefaultEmbeddingTaskType,
		instruction:     DefaultEmbeddingInstruction,
		normalizeOutput: false,
	}
	for _, opt := range efOpts {
		opt(ef)
	}
	return ef, nil
}

func (ef *EmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
	if len(documents) == 0 {
		return embeddings.NewEmptyEmbeddings(), nil
	}

	// Turn 1: Ask LLM to use the tool
	initialPrompt := fmt.Sprintf("Please generate embeddings for the following %d texts using the '%s' tool. Texts: %s. TaskType: '%s', Instruction: '%s', NormalizeOutput: %v.",
		len(documents), EmbeddingToolName, "[\""+strings.Join(documents, "\", \"")+"\"]", ef.taskType, ef.instruction, ef.normalizeOutput)

	initialMessages := []ChatMessage{
		{Role: "system", Content: "You are an assistant that can use tools to perform tasks, including generating text embeddings."},
		{Role: "user", Content: initialPrompt},
	}

	toolDef := getEmbeddingToolDefinition()
	maxTokensForToolCall := 100 // Usually small for tool call confirmation
	reqTurn1 := &ChatCompletionRequest{
		Model:      string(ef.client.DefaultModel),
		Messages:   initialMessages,
		Tools:      []ToolDefinition{toolDef},
		ToolChoice: "auto", // or `map[string]any{"type": "function", "function": map[string]string{"name": EmbeddingToolName}}` to force
		MaxTokens:  &maxTokensForToolCall,
	}

	respTurn1, err := ef.client.CreateChatCompletion(ctx, reqTurn1)
	if err != nil {
		return nil, errors.Wrap(err, "turn 1: failed to request tool use from LLM")
	}

	if len(respTurn1.Choices) == 0 || respTurn1.Choices[0].FinishReason != "tool_calls" || len(respTurn1.Choices[0].Message.ToolCalls) == 0 {
		return nil, errors.Errorf("turn 1: LLM did not call the embedding tool as expected. Finish reason: %s, Response: %+v", respTurn1.Choices[0].FinishReason, respTurn1.Choices[0].Message)
	}

	toolCall := respTurn1.Choices[0].Message.ToolCalls[0]
	if toolCall.Function.Name != EmbeddingToolName {
		return nil, errors.Errorf("turn 1: LLM called an unexpected tool: %s", toolCall.Function.Name)
	}

	var toolParams EmbeddingToolParameters
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &toolParams); err != nil {
		return nil, errors.Wrapf(err, "turn 1: failed to unmarshal tool arguments: %s", toolCall.Function.Arguments)
	}

	// Turn 2: Instruct LLM to "execute" the tool call and provide structured JSON output
	// We use the parameters confirmed by the LLM in toolParams.
	// The prompt for turn 2 needs to be carefully crafted.
	textsForEmbeddingJson, _ := json.Marshal(toolParams.InputTexts)
	promptTurn2 := fmt.Sprintf(
		"You have decided to call the '%s' tool with ID '%s' and the following arguments: TaskType='%s', Instruction='%s', NormalizeOutput=%v, InputTexts=%s. "+
			"Now, please execute this and provide the results. "+
			"Generate the embeddings for each input text. "+
			"Your response MUST be a single JSON object strictly adhering to the schema provided in the 'response_format'. "+
			"The JSON object must contain 'results' (an array of objects, each with 'source_text', 'embedding_vector', and 'normalized' boolean), "+
			"'model_used' (string, the name of the model you used for this embedding generation), and optionally 'usage_info' (object with token counts).",
		EmbeddingToolName, toolCall.ID, toolParams.TaskType, toolParams.Instruction, toolParams.NormalizeOutput, string(textsForEmbeddingJson))

	messagesTurn2 := []ChatMessage{
		// It's good to include the conversation history that led to the tool call
		{Role: "system", Content: "You are an assistant that can use tools to perform tasks, including generating text embeddings."},
		{Role: "user", Content: initialPrompt}, // Original user request
		respTurn1.Choices[0].Message,          // Assistant's response calling the tool
		// Now, the message that "simulates" the tool execution result, but is actually a new instruction
		// This is a bit of a conceptual leap. The Cerebras API expects a `role: "tool"` message here.
		// However, the *content* of that tool message is what we're debating.
		// Instead of providing the *actual* embedding (which we don't have yet),
		// we're using this turn to *request* the embedding in a structured way.
		// A more direct approach for Turn 2 might be a new User message.
		// Let's try a new User message for Turn 2 for clarity of instruction to the LLM.
		{Role: "user", Content: promptTurn2},
	}

	maxTokensForResult := DefaultEmbeddingMaxTokens * len(documents) // Estimate, adjust based on embedding size
	if maxTokensForResult == 0 { maxTokensForResult = 2048 } // Fallback

	reqTurn2 := &ChatCompletionRequest{
		Model:          string(ef.client.DefaultModel),
		Messages:       messagesTurn2,
		ResponseFormat: getEmbeddingOutputResponseFormat(), // CRITICAL for getting structured JSON
		MaxTokens:      &maxTokensForResult,
		Temperature:    ptrFloat32(0.1), // Low temperature for deterministic JSON output
	}

	respTurn2, err := ef.client.CreateChatCompletion(ctx, reqTurn2)
	if err != nil {
		return nil, errors.Wrap(err, "turn 2: failed to get structured embedding output from LLM")
	}

	if len(respTurn2.Choices) == 0 {
		return nil, errors.New("turn 2: no response choices from LLM for embedding generation")
	}

	var embeddingOutput EmbeddingToolOutput
	llmResponseContent := respTurn2.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(llmResponseContent), &embeddingOutput); err != nil {
		return nil, errors.Wrapf(err, "turn 2: failed to unmarshal LLM response into EmbeddingToolOutput. Content: %s", llmResponseContent)
	}

	// Convert to standard embeddings.Embedding type
	finalEmbeddings := make([]embeddings.Embedding, len(embeddingOutput.Results))
	for i, item := range embeddingOutput.Results {
		// Sanity check
		// if i < len(documents) && item.SourceText != documents[i] {
		// log.Printf("Warning: Source text mismatch in LLM output. Expected '%s', got '%s'", documents[i], item.SourceText)
		// }
		finalEmbeddings[i] = embeddings.NewEmbeddingFromFloat32(item.EmbeddingVector)
	}

	return finalEmbeddings, nil
}

func (ef *EmbeddingFunction) EmbedQuery(ctx context.Context, document string) (embeddings.Embedding, error) {
	results, err := ef.EmbedDocuments(ctx, []string{document})
	if err != nil {
		return nil, errors.Wrap(err, "EmbedQuery failed")
	}
	if len(results) == 0 {
		return nil, errors.New("EmbedQuery returned no results")
	}
	return results[0], nil
}

func ptrFloat32(f float32) *float32 { return &f }

