package commons

import (
	"context"
	"io"
	"log/slog"
)

type SlogWriter struct {
	logger *slog.Logger
	level  slog.Level
}

func (w *SlogWriter) Write(p []byte) (n int, err error) {
	ctx := context.Background()
	w.logger.Log(ctx, w.level, string(p))
	return len(p), nil
}

func NewSlogWriter(logger *slog.Logger, level slog.Level) io.Writer {
	return &SlogWriter{
		logger: logger,
		level:  level,
	}
}
