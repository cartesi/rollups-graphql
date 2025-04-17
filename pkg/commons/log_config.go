package commons

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

func ConfigureLog(level slog.Leveler) {
	logOpts := new(tint.Options)
	logOpts.Level = level
	logOpts.AddSource = true
	logOpts.NoColor = false
	logOpts.TimeFormat = "[15:04:05.000]"
	handler := tint.NewHandler(os.Stdout, logOpts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func ConfigureLogForProduction(level slog.Leveler, hasColor bool) {
	logOpts := new(tint.Options)
	logOpts.Level = level
	logOpts.AddSource = level == slog.LevelDebug
	logOpts.NoColor = !hasColor || !isatty.IsTerminal(os.Stdout.Fd())
	logOpts.TimeFormat = "[15:04:05.000]"
	handler := tint.NewHandler(os.Stdout, logOpts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
