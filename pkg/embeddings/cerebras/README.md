# Cerebras LLM Embedding Generation

## Overview

This document outlines an innovative approach to using Cerebras LLM's "tool calling" functionality for embedding generation. Rather than delegating to an external service, we leverage the LLM's inherent capabilities to generate embeddings directly.

Our findings suggest that Cerebras LLM models can generate embeddings effectively, making the concept of an `EmbeddingToolExecutor` unnecessary for this specific task. Our vision is to allow Cerebras (and its underlying LLM models) to handle embeddings end-to-end.

## Technical Approach

The Cerebras API supports a chat/completions style endpoint with a `tools` parameter. This allows us to define functions that the LLM can choose to "call," outputting a structured JSON object with the function name and necessary arguments. Our application then processes this output and continues the conversation.

Our experimental `cerebras.go` file, with `EmbeddingToolSchema`, `EmbeddingParameters`, and `EmbeddingResults`, provides a foundation for defining such a tool. The key is structuring these definitions to align with the Cerebras API's expectations.

## Core Implementation

### Architecture Components

- **Client for Chat/Completions**: Designed to interact with the Cerebras chat/completions endpoint
- **Tool Definition**: Structures and helpers to define the embedding tool schema
- **Prompt Engineering**: Crafting specific prompts for embedding generation
- **Structured JSON Output**: Using `response_format` with `json_schema` for predictable output
- **Parsing**: Extracting embedding vectors from JSON responses
- **Result Formatting**: Properly formatting output for the Cerebras LLM

This approach is more experimental than calling a dedicated embedding API, as it depends on the LLM's ability to follow complex instructions and output structured data.

## Multi-Turn Conversation Flow

### Turn 1: Request with Tool Definition

**Client to LLM:**
```
"Please generate embeddings for these texts: ['text A', 'text B'], considering task_type='retrieval', 
instruction='Focus on semantic meaning for search', and normalize_output=true. 
Use the advanced_embedding_generator tool."
```

The API call includes the definition of the `advanced_embedding_generator` tool, specifying its parameters:
- `input_texts`
- `task_type`
- `instruction`
- `normalize_output`

### Turn 1 Response: Tool Invocation

The LLM, understanding the request and seeing the tool, decides to "call" it.

**Response:**
```json
{
  "id": "call_abc123",
  "type": "function",
  "function": {
    "name": "advanced_embedding_generator",
    "arguments": "{\"input_texts\": [\"text A\", \"text B\"], \"task_type\": \"retrieval\", \"instruction\": \"Focus on semantic meaning for search\", \"normalize_output\": true}"
  }
}
```

The LLM has essentially confirmed the parameters it will use.

### Turn 2: Execute Tool & Request Structured Output

Our client receives this response. Instead of calling an external API, we instruct the same LLM to perform the actual embedding generation based on the arguments it provided, and to format the output according to a specific JSON schema.

**Client to LLM:**
```
"Okay, proceed with the advanced_embedding_generator call (ID: call_abc123). For each input text, 
generate the embedding. Return the results as a JSON object strictly adhering to this schema: 
{'results': [{'source_text': '...', 'embedding_vector': [...], 'normalized': true/false}], 
'model_used': '...', 'usage_info': {'prompt_tokens': ..., 'completion_tokens': ..., 'total_tokens': ...}}."
```

This API call uses the `response_format` parameter with the detailed JSON schema for the embeddings and usage info.

### Turn 2 Response: Structured Embedding Output

The LLM processes this, generates the embeddings, and formats the output according to the `response_format` schema.

**Response:** A chat message containing the JSON string with embeddings, model info, and usage.

This makes the "tool" a mechanism for the LLM to acknowledge and structure the parameters of a complex internal task, followed by a highly specific prompt that elicits the structured output.

## Technical Implementation Details

### EmbeddingToolParameters

- **InputTexts**: Changed from a single `InputText` to `InputTexts []string` to handle batching
- **TaskType, Instruction, NormalizeOutput**: Added with defaults to guide the LLM

### EmbeddingToolOutput & EmbeddingResultItem

- Structure demanded from the LLM in Turn 2 using `response_format`
- **EmbeddingResultItem** includes:
  - `SourceText`
  - `EmbeddingVector`
  - `Normalized` (confirmation of normalization attempt)
- **EmbeddingToolOutput** wraps results and adds:
  - `ModelUsed`
  - `UsageInfo`

### Key Functions

- **getEmbeddingToolDefinition()**: Creates the JSON schema for the tool
- **getEmbeddingOutputResponseFormat()**: Defines the JSON schema for expected output (with `strict: true`)

### EmbeddingFunction

- Configurable defaults via `EmbeddingFunctionOptions`
- **EmbedDocuments Orchestration**:
  - **Turn 1**: Sends initial prompt with texts, parameters, and tool definition
  - **Turn 2**:
    - Parses arguments from the LLM's tool call
    - Constructs a specific prompt for embedding generation
    - Uses `ResponseFormat` to enforce the output schema
    - Uses low `Temperature` (e.g., 0.1) for deterministic JSON output
    - Parses JSON to extract embeddings

### Error Handling and Robustness

- Checks for tool calling issues
- Success depends on the LLM's ability to follow instructions for both tool use and structured JSON output

## Innovation and Vision Alignment

- **LLM-Powered Structuring**: Using tool-calling to establish parameters for an internal task
- **Structured Output Enforcement**: Using `response_format` for reliable JSON output
- **Flexibility**: Nuanced control via natural language instructions
- **Single LLM Endpoint**: All processing through the chat/completions endpoint

## Challenges and Considerations

### LLM Reliability Requirements

- Understanding when to use the defined tool
- Correctly populating tool call arguments
- Following complex instructions for embedding generation
- Strictly adhering to the JSON schema

### Other Considerations

- **Prompt Engineering**: Critical prompts requiring iteration and testing
- **Latency**: Two LLM roundtrips introduce more latency than direct API calls
- **Cost**: Two LLM calls likely more expensive
- **Normalization**: May require client-side handling if LLM struggles
- **Token Limits**: JSON output for embeddings can be large

## Conclusion

This sophisticated approach treats the LLM as a flexible processing unit. While experimental, it offers a promising path forward. Thorough testing with the actual Cerebras API and models will be essential to validate this approach.