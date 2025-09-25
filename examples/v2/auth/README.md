# Authentication Examples

This directory contains practical examples of different authentication methods for the Chroma Go v2 API.

## Examples

- **basic_auth.go** - Basic authentication with username and password
- **bearer_token.go** - Bearer token authentication using Authorization header
- **x_chroma_token.go** - Token authentication using X-Chroma-Token header
- **custom_headers.go** - Custom headers for advanced authentication scenarios
- **chroma_cloud.go** - Chroma Cloud authentication with API key

## Running the Examples

### Prerequisites

1. For self-hosted examples, ensure Chroma is running locally:

```bash
docker run -p 8000:8000 chromadb/chroma
```

2. Set up authentication on your Chroma server as needed.

### Basic Authentication

```bash
go run basic_auth.go
```

### Bearer Token

```bash
# Set token via environment variable
export CHROMA_AUTH_TOKEN="your-token-here"
go run bearer_token.go
```

### X-Chroma-Token

```bash
export CHROMA_AUTH_TOKEN="your-token-here"
go run x_chroma_token.go
```

### Custom Headers

```bash
export AUTH_TOKEN="your-bearer-token"
export API_KEY="your-api-key"
go run custom_headers.go
```

### Chroma Cloud

```bash
# Required environment variables
export CHROMA_CLOUD_API_KEY="your-api-key"
export CHROMA_CLOUD_TENANT="your-tenant"     
export CHROMA_CLOUD_DATABASE="your-database"

go run chroma_cloud.go
```

## Environment Variables

| Variable              | Description                     | Used By                            |
|-----------------------|---------------------------------|------------------------------------|
| CHROMA_AUTH_TOKEN     | Authentication token            | bearer_token.go, x_chroma_token.go |
| AUTH_TOKEN            | Bearer token for custom headers | custom_headers.go                  |
| API_KEY               | API key for custom headers      | custom_headers.go                  |
| CHROMA_CLOUD_API_KEY  | Chroma Cloud API key            | chroma_cloud.go                    |
| CHROMA_CLOUD_HOST     | Chroma Cloud host               | chroma_cloud.go                    |
| CHROMA_CLOUD_TENANT   | Chroma Cloud tenant             | chroma_cloud.go                    |
| CHROMA_CLOUD_DATABASE | Chroma Cloud database           | chroma_cloud.go                    |