package raw

import (
	"bytes"
	"context"
	"embed"
	"io/fs"
	"testing"

	"github.com/joho/godotenv"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	// "github.com/testcontainers/testcontainers-go/modules/compose"
)

type DBManager struct {
	opts []testcontainers.ContainerCustomizer
}

//go:embed *.sql migrate.sh
var initScripts embed.FS

//go:embed .env
var envfs []byte

const POSTGRES_VERSION = "postgres:16-alpine"

func (d *DBManager) getEnv() (map[string]string, error) {
	buffer := bytes.NewBuffer(envfs)
	return godotenv.Parse(buffer)
}

func (d *DBManager) getInitFiles() ([]string, error) {
	scripts := []string{}

	err := fs.WalkDir(initScripts, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			scripts = append(scripts, path)
		}

		return nil
	})

	return scripts, err
}

func (d *DBManager) Run(ctx context.Context, t *testing.T) error {
	container, err := postgres.Run(ctx, POSTGRES_VERSION, d.opts...)
	if err != nil {
		return err
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate pgContainer: %s", err)
		}
	})

	return nil
}

func (d *DBManager) Create() error {
	var opts []testcontainers.ContainerCustomizer

	env, err := d.getEnv()
	if err != nil {
		return err
	}
	opts = append(opts, testcontainers.WithEnv(env))

	scripts, err := d.getInitFiles()
	if err != nil {
		return err
	}
	opts = append(opts, postgres.WithInitScripts(scripts...))

	opts = append(opts, postgres.WithUsername("postgres"))

	// var timeout time.Duration = 5 * time.Second
	// retry := 2
	// waiter := testcontainers.WithWaitStrategy(
	// 	wait.ForLog("database system is ready to accept connections").
	// 		WithOccurrence(retry).WithStartupTimeout(timeout))
	// opts = append(opts, waiter)

	d.opts = opts

	return nil
}
