package logger

import (
	"context"
	"log/slog"
	"os"
)

// SlogLogger is a Logger implementation using the standard library's slog package
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger creates a new SlogLogger with the provided slog.Logger
func NewSlogLogger(logger *slog.Logger) *SlogLogger {
	return &SlogLogger{
		logger: logger,
	}
}

// NewSlogLoggerWithHandler creates a new SlogLogger with the provided handler
func NewSlogLoggerWithHandler(handler slog.Handler) *SlogLogger {
	return &SlogLogger{
		logger: slog.New(handler),
	}
}

// NewDefaultSlogLogger creates a new SlogLogger with JSON handler and production configuration
func NewDefaultSlogLogger() (*SlogLogger, error) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	return &SlogLogger{
		logger: slog.New(handler),
	}, nil
}

// NewTextSlogLogger creates a new SlogLogger with text handler for human-readable output
func NewTextSlogLogger() (*SlogLogger, error) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	return &SlogLogger{
		logger: slog.New(handler),
	}, nil
}

// Debug logs a message at debug level
func (s *SlogLogger) Debug(msg string, fields ...Field) {
	s.logger.Debug(msg, convertFieldsToAttrs(fields)...)
}

// Info logs a message at info level
func (s *SlogLogger) Info(msg string, fields ...Field) {
	s.logger.Info(msg, convertFieldsToAttrs(fields)...)
}

// Warn logs a message at warn level
func (s *SlogLogger) Warn(msg string, fields ...Field) {
	s.logger.Warn(msg, convertFieldsToAttrs(fields)...)
}

// Error logs a message at error level
func (s *SlogLogger) Error(msg string, fields ...Field) {
	s.logger.Error(msg, convertFieldsToAttrs(fields)...)
}

// DebugWithContext logs a message at debug level with context
func (s *SlogLogger) DebugWithContext(ctx context.Context, msg string, fields ...Field) {
	ctxAttrs := extractContextAttrs(ctx)
	allAttrs := append(convertFieldsToAttrs(fields), ctxAttrs...)
	s.logger.DebugContext(ctx, msg, allAttrs...)
}

// InfoWithContext logs a message at info level with context
func (s *SlogLogger) InfoWithContext(ctx context.Context, msg string, fields ...Field) {
	ctxAttrs := extractContextAttrs(ctx)
	allAttrs := append(convertFieldsToAttrs(fields), ctxAttrs...)
	s.logger.InfoContext(ctx, msg, allAttrs...)
}

// WarnWithContext logs a message at warn level with context
func (s *SlogLogger) WarnWithContext(ctx context.Context, msg string, fields ...Field) {
	ctxAttrs := extractContextAttrs(ctx)
	allAttrs := append(convertFieldsToAttrs(fields), ctxAttrs...)
	s.logger.WarnContext(ctx, msg, allAttrs...)
}

// ErrorWithContext logs a message at error level with context
func (s *SlogLogger) ErrorWithContext(ctx context.Context, msg string, fields ...Field) {
	ctxAttrs := extractContextAttrs(ctx)
	allAttrs := append(convertFieldsToAttrs(fields), ctxAttrs...)
	s.logger.ErrorContext(ctx, msg, allAttrs...)
}

// With returns a new logger with the given fields
func (s *SlogLogger) With(fields ...Field) Logger {
	return &SlogLogger{
		logger: s.logger.With(convertFieldsToAttrs(fields)...),
	}
}

// IsDebugEnabled returns true if debug level is enabled
func (s *SlogLogger) IsDebugEnabled() bool {
	return s.logger.Enabled(context.Background(), slog.LevelDebug)
}

// Sync flushes any buffered log entries
// slog doesn't require explicit sync, but we implement it for interface compatibility
func (s *SlogLogger) Sync() error {
	// slog handlers typically don't buffer, so this is a no-op
	// If using a custom handler that does buffer, it should handle syncing internally
	return nil
}

// convertFieldsToAttrs converts our Field type to slog.Attr
func convertFieldsToAttrs(fields []Field) []any {
	attrs := make([]any, 0, len(fields))
	for _, f := range fields {
		switch v := f.Value.(type) {
		case string:
			attrs = append(attrs, slog.String(f.Key, v))
		case int:
			attrs = append(attrs, slog.Int(f.Key, v))
		case int32:
			attrs = append(attrs, slog.Int(f.Key, int(v)))
		case int64:
			attrs = append(attrs, slog.Int64(f.Key, v))
		case uint:
			attrs = append(attrs, slog.Uint64(f.Key, uint64(v)))
		case uint32:
			attrs = append(attrs, slog.Uint64(f.Key, uint64(v)))
		case uint64:
			attrs = append(attrs, slog.Uint64(f.Key, v))
		case bool:
			attrs = append(attrs, slog.Bool(f.Key, v))
		case float32:
			attrs = append(attrs, slog.Float64(f.Key, float64(v)))
		case float64:
			attrs = append(attrs, slog.Float64(f.Key, v))
		case error:
			// For error fields, use "error" as the key if the field key is empty
			key := f.Key
			if key == "" {
				key = "error"
			}
			attrs = append(attrs, slog.String(key, v.Error()))
		default:
			attrs = append(attrs, slog.Any(f.Key, v))
		}
	}
	return attrs
}

// extractContextAttrs extracts attributes from context
// This can be extended to extract trace IDs, request IDs, etc.
func extractContextAttrs(ctx context.Context) []any {
	if ctx == nil {
		return []any{}
	}

	attrs := []any{}

	// Example: Safely extract trace ID if present
	// if traceID := ctx.Value("trace-id"); traceID != nil {
	//     switch v := traceID.(type) {
	//     case string:
	//         attrs = append(attrs, slog.String("trace_id", v))
	//     case fmt.Stringer:
	//         attrs = append(attrs, slog.String("trace_id", v.String()))
	//     default:
	//         attrs = append(attrs, slog.Any("trace_id", v))
	//     }
	// }

	// Example: Safely extract request ID
	// if reqID := ctx.Value("request-id"); reqID != nil {
	//     switch v := reqID.(type) {
	//     case string:
	//         attrs = append(attrs, slog.String("request_id", v))
	//     case fmt.Stringer:
	//         attrs = append(attrs, slog.String("request_id", v.String()))
	//     default:
	//         attrs = append(attrs, slog.Any("request_id", v))
	//     }
	// }

	return attrs
}
