// Copyright (c) Cartesi Rollups GraphQL Contributors
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package commons

import (
	"context"
	"log/slog"
)

// contextKey is a private type for context keys used in this package.
// This prevents collisions with keys from other packages.
type contextKey int

// loggerKey is the key for logger values in contexts.
const loggerKey contextKey = iota

// WithLogger returns a new context with the given logger attached.
// This is similar to AsyncLocalStorage from Node.js, allowing a logger
// to be propagated through the context chain.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// GetLogger retrieves the logger from the context.
// If no logger is found in the context, it returns the default logger.
func GetLogger(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// LoggerWith returns a new logger with additional context appended,
// and stores it in a new context that is returned.
// This is a convenience function that combines WithLogger and slog.With.
func LoggerWith(ctx context.Context, args ...any) (context.Context, *slog.Logger) {
	logger := GetLogger(ctx).With(args...)
	return WithLogger(ctx, logger), logger
}
