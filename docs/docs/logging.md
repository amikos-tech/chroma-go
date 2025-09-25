# Logging

!!! note "V2 API Only"
    The logging feature is only available for the V2 API. V1 API uses standard library logging and is maintained for backward compatibility.

## Overview

The chroma-go V2 client provides a flexible logging interface that allows you to inject custom loggers instead of using stdio for debug output. This feature enables better integration with your application's logging infrastructure and provides structured logging capabilities.

## Features

- **Pluggable Logger Interface**: Define your own logger implementation or use the provided ones
- **Structured Logging**: Support for structured fields and context-aware logging
- **Multiple Log Levels**: Debug, Info, Warn, and Error levels
- **Context Support**: Pass context for distributed tracing and request correlation
- **Zero Allocation**: NoopLogger for production scenarios where logging should be disabled

## Logger Interface

The Logger interface defines the contract for all logger implementations:

```go
type Logger interface {
    // Standard logging methods
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)

    // Context-aware logging methods
    DebugWithContext(ctx context.Context, msg string, fields ...Field)
    InfoWithContext(ctx context.Context, msg string, fields ...Field)
    WarnWithContext(ctx context.Context, msg string, fields ...Field)
    ErrorWithContext(ctx context.Context, msg string, fields ...Field)

    // With returns a new logger with the given fields
    With(fields ...Field) Logger

    // Enabled returns true if the given level is enabled
    IsDebugEnabled() bool
}
```

## Built-in Implementations

### ZapLogger

The ZapLogger wraps [uber-go/zap](https://github.com/uber-go/zap) for high-performance structured logging.

```go
import (
    "go.uber.org/zap"
    chromalogger "github.com/amikos-tech/chroma-go/pkg/logger"
    v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

// Create a zap logger
zapLogger, _ := zap.NewProduction()

// Wrap it in ChromaLogger
logger := chromalogger.NewZapLogger(zapLogger)

// Use it with the client
client, err := v2.NewHTTPClient(
    v2.WithBaseURL("http://localhost:8000"),
    v2.WithLogger(logger),
)
```

#### Development Logger

For development, you can use a pre-configured development logger:

```go
logger, _ := chromalogger.NewDevelopmentZapLogger()

client, err := v2.NewHTTPClient(
    v2.WithBaseURL("http://localhost:8000"),
    v2.WithLogger(logger),
)
```

### NoopLogger

The NoopLogger discards all log messages and is useful for production scenarios where you want to disable logging completely:

```go
logger := chromalogger.NewNoopLogger()

client, err := v2.NewHTTPClient(
    v2.WithBaseURL("http://localhost:8000"),
    v2.WithLogger(logger),
)
```

## Using WithDebug()

When you use `WithDebug()` without providing a custom logger, the client automatically creates a development logger:

```go
// This will automatically create a development logger with debug level enabled
client, err := v2.NewHTTPClient(
    v2.WithBaseURL("http://localhost:8000"),
    v2.WithDebug(),
)
```

## Structured Logging with Fields

The logger supports structured logging with fields for better log analysis:

```go
logger := chromalogger.NewDevelopmentZapLogger()

// Create a logger with persistent fields
requestLogger := logger.With(
    chromalogger.String("request_id", "123"),
    chromalogger.String("user_id", "user-456"),
)

client, err := v2.NewHTTPClient(
    v2.WithBaseURL("http://localhost:8000"),
    v2.WithLogger(requestLogger),
)
```

### Field Helpers

The logger package provides convenient field constructors:

```go
chromalogger.String("key", "value")
chromalogger.Int("count", 42)
chromalogger.Bool("enabled", true)
chromalogger.ErrorField("error", err)
chromalogger.Any("data", complexObject)
```

## Context-Aware Logging

For distributed tracing and request correlation, use the context-aware methods:

```go
ctx := context.WithValue(context.Background(), "trace-id", "abc123")

// The logger implementation can extract values from context
logger.InfoWithContext(ctx, "Processing request",
    chromalogger.String("operation", "query"),
)
```

## Custom Logger Implementation

You can implement your own logger by implementing the Logger interface:

```go
type MyCustomLogger struct {
    // your logger implementation
}

func (l *MyCustomLogger) Debug(msg string, fields ...chromalogger.Field) {
    // implement debug logging
}

func (l *MyCustomLogger) Info(msg string, fields ...chromalogger.Field) {
    // implement info logging
}

// ... implement other required methods

// Use your custom logger
client, err := v2.NewHTTPClient(
    v2.WithBaseURL("http://localhost:8000"),
    v2.WithLogger(&MyCustomLogger{}),
)
```

## Cloud Client Support

The logging feature is also available for the Cloud client:

```go
logger := chromalogger.NewDevelopmentZapLogger()

client, err := v2.NewCloudClient(
    v2.WithLogger(logger),
    v2.WithCloudAPIKey("your-api-key"),
    v2.WithDatabaseAndTenant("database", "tenant"),
)
```

## Performance Considerations

1. **Use NoopLogger in Production**: If you don't need logging, use NoopLogger to avoid any performance overhead
2. **Check IsDebugEnabled()**: Before expensive debug operations, check if debug logging is enabled
3. **Use Structured Fields**: Instead of string formatting, use structured fields for better performance
4. **Reuse Loggers**: Create loggers with common fields using `With()` and reuse them

## Example: Complete Logging Setup

```go
package main

import (
    "context"
    "log"

    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"

    chromalogger "github.com/amikos-tech/chroma-go/pkg/logger"
    v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
    // Configure zap logger
    config := zap.NewProductionConfig()
    config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
    config.OutputPaths = []string{"stdout", "/var/log/chroma-client.log"}

    zapLogger, err := config.Build()
    if err != nil {
        log.Fatal(err)
    }
    defer zapLogger.Sync()

    // Create chroma logger
    logger := chromalogger.NewZapLogger(zapLogger)

    // Add request-specific fields
    requestLogger := logger.With(
        chromalogger.String("service", "my-app"),
        chromalogger.String("version", "1.0.0"),
    )

    // Create client with logger
    client, err := v2.NewHTTPClient(
        v2.WithBaseURL("http://localhost:8000"),
        v2.WithLogger(requestLogger),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Use the client - all operations will be logged
    ctx := context.Background()
    collections, err := client.ListCollections(ctx)
    if err != nil {
        requestLogger.Error("Failed to list collections",
            chromalogger.ErrorField("error", err),
        )
        return
    }

    requestLogger.Info("Listed collections successfully",
        chromalogger.Int("count", len(collections)),
    )
}
```

## Migration from Debug Flag

If you were previously using the debug flag with direct logging:

**Before:**
```go
client, err := v2.NewHTTPClient(
    v2.WithBaseURL("http://localhost:8000"),
    v2.WithDebug(), // This would print to stdout
)
```

**After:**
```go
// Option 1: Keep using WithDebug() - it now uses a proper logger
client, err := v2.NewHTTPClient(
    v2.WithBaseURL("http://localhost:8000"),
    v2.WithDebug(), // Now uses structured logging
)

// Option 2: Use a custom logger for more control
logger, _ := chromalogger.NewDevelopmentZapLogger()
client, err := v2.NewHTTPClient(
    v2.WithBaseURL("http://localhost:8000"),
    v2.WithLogger(logger),
)
```

## Troubleshooting

### No logs appearing

1. Check if your logger is properly configured and the log level allows the messages
2. Ensure you're not using NoopLogger unintentionally
3. For custom loggers, verify your implementation outputs to the expected destination

### Too many logs

1. Adjust the log level in your logger configuration
2. Use NoopLogger for specific operations that don't need logging
3. Consider using a logger with filtering capabilities

### Performance impact

1. Use NoopLogger in production if logging is not needed
2. Avoid expensive operations in log message formatting
3. Use structured fields instead of string concatenation