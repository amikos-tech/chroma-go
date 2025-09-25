# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Build and Test
```bash
# Build all packages
make build

# Run tests by category (use build tags)
make test          # V1 API tests
make test-v2       # V2 API tests
make test-cloud    # Cloud integration tests (requires CHROMA_CLOUD_* env vars)
make test-ef       # Embedding function tests
make test-rf       # Reranking function tests

# Run specific test
go test -tags=basicv2 -run TestCollectionAdd ./test/client_v2/...

# Linting
make lint          # Check for linting issues
make lint-fix      # Auto-fix linting issues

# Local development server
make server        # Start Docker Chroma server on port 8000
```

### Environment Variables
The codebase heavily relies on environment variables for configuration:
- `CHROMA_URL` - Chroma server URL (default: http://localhost:8000)
- `CHROMA_CLOUD_API_KEY` - Cloud API key
- `CHROMA_CLOUD_HOST` - Cloud host
- `CHROMA_CLOUD_TENANT` - Cloud tenant ID
- `CHROMA_CLOUD_DATABASE` - Cloud database ID

## Architecture

### API Structure
The codebase maintains two API versions:
- **V2 API** (`/pkg/api/v2/`) - Current primary API, all new features go here
- **V1 API** (`/pkg/api/v1/`, root files) - Legacy, maintained for backward compatibility

### Core Components
- **Client**: Main entry point in `/pkg/api/v2/client.go` for V2, `/chroma.go` for V1
- **Collections**: Vector collection management with embedding/query operations
- **Embeddings**: Modular embedding functions in `/pkg/embeddings/` supporting 12+ providers
- **Metadata**: Rich filtering capabilities with type-safe metadata handling
- **Authentication**: Multiple auth methods (Basic, Bearer, X-Chroma-Token)

### Testing Strategy
Tests are segregated by build tags to run specific test suites:
- `basic` - V1 tests
- `basicv2` - V2 tests
- `cloud` - Cloud integration
- `ef` - Embedding functions
- `rf` - Reranking functions

Integration tests use `testcontainers-go` for Docker-based testing against real Chroma instances.

### Key Patterns
- **Functional Options**: Client initialization uses option functions pattern
- **Context Propagation**: All API methods accept context for cancellation/timeout
- **Interface-based Design**: Clean interfaces for testability and extensibility
- **Build Tags**: Feature segregation to avoid unnecessary dependencies

## Development Guidelines

### Adding New Features
1. New features should target V2 API (`/pkg/api/v2/`)
2. Add corresponding tests with appropriate build tags
3. Update examples in `/examples/v2/` if applicable
4. Ensure backward compatibility for V1 if modifying shared components

### Testing Requirements
- Write tests with appropriate build tags
- Use `testify` for assertions
- Integration tests should use testcontainers
- Run `make lint` before committing

### Common Tasks
- **Adding Embedding Provider**: Implement in `/pkg/embeddings/`, follow existing provider patterns
- **Modifying Client**: V2 changes in `/pkg/api/v2/client.go`, ensure collection caching logic is maintained
- **Updating Authentication**: Modify `/pkg/api/v2/openapi/configuration.go` and auth middleware
- **Working with Metadata**: Use `/pkg/api/v2/metadata/` utilities for type conversions

### Version Compatibility
The client is tested against Chroma versions 0.4.8 to 1.0.20. Ensure changes maintain compatibility across this range.
- Always lint before commiting or pushing code