package commons

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
)

type DbFactory struct {
	TempDir string
	Timeout time.Duration
}

const TimeoutInSeconds = 10

func NewDbFactory() (*DbFactory, error) {
	tempDir, err := os.MkdirTemp("", "nonodo-test-*")
	if err != nil {
		return nil, fmt.Errorf("error creating temp dir: %w", err)
	}

	return &DbFactory{
		TempDir: tempDir,
		Timeout: TimeoutInSeconds * time.Second,
	}, nil
}

func (d *DbFactory) CreateDbCtx(ctx context.Context, sqliteFileName string) (*sqlx.DB, error) {
	sqlitePath := filepath.Join(d.TempDir, sqliteFileName)
	slog.InfoContext(ctx, "Creating db (with ctx) attempting", "sqlitePath", sqlitePath)
	return sqlx.ConnectContext(ctx, "sqlite3", sqlitePath)
}

func (d *DbFactory) CreateDb(ctx context.Context, sqliteFileName string) *sqlx.DB {
	// db := sqlx.MustConnect("sqlite3", ":memory:")
	sqlitePath := filepath.Join(d.TempDir, sqliteFileName)
	slog.InfoContext(ctx, "Creating db attempting", "sqlitePath", sqlitePath)
	return sqlx.MustConnect("sqlite3", sqlitePath)
}

func (d *DbFactory) Cleanup(ctx context.Context) {
	if d.TempDir != "" {
		slog.InfoContext(ctx, "Cleaning up temp dir", "tempDir", d.TempDir)
		err := os.RemoveAll(d.TempDir)
		if err != nil {
			slog.ErrorContext(ctx, "Error removing temp dir", "err", err)
		}
	}
}
