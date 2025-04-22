package commons

import (
	"context"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

// contextKey is a private type for context keys used in this package.
// This prevents collisions with keys from other packages.
type contextKey int

// workerKey is the key for logger values in contexts.
const workerKey contextKey = iota

type LoggerWithContext struct {
	slog.Handler
}

func (g *LoggerWithContext) Handle(ctx context.Context, r slog.Record) error {
	if workerName, ok := ctx.Value(workerKey).(string); ok {
		r.AddAttrs(slog.String("worker", workerName))
	}

	// Call the parent handler
	return g.Handler.Handle(ctx, r)
}

func ConfigureLog(level slog.Leveler) {
	logOpts := new(tint.Options)
	logOpts.Level = level
	logOpts.AddSource = true
	logOpts.NoColor = false
	logOpts.TimeFormat = "[15:04:05.000]"
	handler := tint.NewHandler(os.Stdout, logOpts)
	wrappedHandler := &LoggerWithContext{Handler: handler}
	logger := slog.New(wrappedHandler)
	slog.SetDefault(logger)
}

// Remove the timeformat by completely removing the time attribute
// Inspired by https://pkg.go.dev/log/slog/internal/slogtest@go1.24.2#RemoveTime
func removeTimestampFromLog(groups []string, a slog.Attr) slog.Attr {
	// If the attribute is the time key and is at the root level (no groups), return an empty attribute
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{}
	}
	return a
}

func ConfigureLogForProduction(level slog.Leveler, hasColor bool) {
	logOpts := &tint.Options{
		Level:       level,
		AddSource:   level == slog.LevelDebug,
		NoColor:     !hasColor || !isatty.IsTerminal(os.Stdout.Fd()),
		ReplaceAttr: removeTimestampFromLog,
	}

	handler := tint.NewHandler(os.Stdout, logOpts)
	wrappedHandler := &LoggerWithContext{Handler: handler}

	logger := slog.New(wrappedHandler)
	slog.SetDefault(logger)
}

func AddWorkerNameToContext(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, workerKey, name)
}
