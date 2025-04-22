package raw

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"log/slog"

	"github.com/joho/godotenv"
	"github.com/testcontainers/testcontainers-go/log"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	_ "github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	DOCKER_COMPOSE_IDENTIFIER = "database-rollups"
	DOCKER_COMPOSE_SERVICE    = "postgres-raw"
	DOCKER_COMPOSE_LOGS_MAX   = uint8(10)
)

//go:embed compose.yml .env
var composeFiles embed.FS

type stdcomposeLogger struct{}

// Printf implements log.Logger.
func (c *stdcomposeLogger) Printf(format string, v ...any) {
	fmt.Printf(format, v...)
}

func newComposeLogger() log.Logger {
	return &stdcomposeLogger{}
}

func loadMapEnvFile() (map[string]string, error) {
	content, err := composeFiles.ReadFile(".env")
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(content)
	return godotenv.Parse(reader)
}

func loadCompose() (io.Reader, error) {
	content, err := composeFiles.ReadFile("compose.yml")
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(content)
	return reader, nil
}

func createStack() (compose.ComposeStack, error) {
	env, err := loadMapEnvFile()
	if err != nil {
		return nil, err
	}

	composeContent, err := loadCompose()
	if err != nil {
		return nil, err
	}

	req, err := compose.NewDockerComposeWith(
		compose.StackIdentifier(DOCKER_COMPOSE_IDENTIFIER),
		compose.WithStackReaders(composeContent),
		compose.WithLogger(newComposeLogger()),
	)

	if err != nil {
		return nil, fmt.Errorf("docker compose request failed: %s", err)
	}

	stack := req.WithEnv(env).WaitForService(DOCKER_COMPOSE_SERVICE, wait.ForHealthCheck())

	return stack, nil
}

func RunDockerCompose(ctx context.Context) error {
	slog.DebugContext(ctx, "running docker compose")

	stack, err := createStack()
	if err != nil {
		return fmt.Errorf("docker compose create stack failed: %s", err)
	}

	if err := stack.Up(ctx, compose.Wait(true)); err != nil {
		return fmt.Errorf("docker compose up failed: %s", err)
	}

	slog.DebugContext(ctx, "docker compose up successful")

	return nil
}

func StopDockerCompose(ctx context.Context) error {
	slog.DebugContext(ctx, "stopping docker compose")

	stack, err := createStack()
	if err != nil {
		return fmt.Errorf("docker compose create stack failed: %s", err)
	}

	if err = stack.Down(ctx, compose.RemoveOrphans(true)); err != nil {
		return fmt.Errorf("docker compose down failed: %s", err)
	}
	slog.DebugContext(ctx, "docker compose down successful")

	return nil
}

func CleanupDockerCompose(ctx context.Context) error {
	stack, err := createStack()
	if err != nil {
		return fmt.Errorf("docker compose create stack failed: %s", err)
	}

	downOpts := []compose.StackDownOption{
		compose.RemoveOrphans(true),
		compose.RemoveVolumes(true),
		compose.RemoveImages(compose.RemoveImagesLocal),
	}
	if err = stack.Down(ctx, downOpts...); err != nil {
		return fmt.Errorf("docker compose cleanup failed: %s", err)
	}

	slog.DebugContext(ctx, "docker compose cleanup successful")

	return nil
}
