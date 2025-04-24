package raw

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"log/slog"

	"github.com/joho/godotenv"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/log"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	DOCKER_COMPOSE_IDENTIFIER = "database-rollups"
	DOCKER_COMPOSE_SERVICE    = "postgres-raw"
	DOCKER_COMPOSE_LOGS_MAX   = uint8(10)
)

type DockerComposeContainer struct {
	stack compose.ComposeStack
}

//go:embed compose.yml .env
var composeFiles embed.FS

type stdcomposeLogger struct{}

// Printf implements log.Logger.
func (c *stdcomposeLogger) Printf(format string, v ...any) {
	slog.Debug(fmt.Sprintf(format, v...))
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

func NewDockerComposeContainer() *DockerComposeContainer {
	return &DockerComposeContainer{}
}

func (d *DockerComposeContainer) RunDockerCompose(ctx context.Context) error {
	var err error

	if d.stack != nil {
		var container testcontainers.Container
		if container, err = d.stack.ServiceContainer(ctx, DOCKER_COMPOSE_SERVICE); err != nil {
			return fmt.Errorf("docker compose get container failed: %s", err)
		}

		if container.IsRunning() {
			slog.DebugContext(ctx, "docker compose already running, stopping")
			if err := d.StopDockerCompose(ctx); err != nil {
				return fmt.Errorf("docker compose down failed: %s", err)
			}
			slog.DebugContext(ctx, "docker compose down successful")
		}

	}

	slog.DebugContext(ctx, "running docker compose")

	d.stack, err = createStack()
	if err != nil {
		return fmt.Errorf("docker compose create stack failed: %s", err)
	}

	if err := d.stack.Up(ctx, compose.Wait(true)); err != nil {
		return fmt.Errorf("docker compose up failed: %s", err)
	}

	slog.DebugContext(ctx, "docker compose up successful")

	return nil
}

func (d *DockerComposeContainer) StopDockerCompose(ctx context.Context) error {
	slog.DebugContext(ctx, "stopping docker compose")

	if d.stack == nil {
		slog.DebugContext(ctx, "docker compose stack is nil, nothing to stop")
		return nil
	}

	if err := d.stack.Down(ctx, compose.RemoveOrphans(true)); err != nil {
		return fmt.Errorf("docker compose down failed: %s", err)
	}
	slog.DebugContext(ctx, "docker compose down successful")

	return nil
}

func (d *DockerComposeContainer) CleanupDockerCompose(ctx context.Context) error {
	if d.stack == nil {
		slog.DebugContext(ctx, "docker compose stack is nil, nothing to cleanup")
		return nil
	}

	downOpts := []compose.StackDownOption{
		compose.RemoveOrphans(true),
		compose.RemoveVolumes(true),
		compose.RemoveImages(compose.RemoveImagesLocal),
	}
	if err := d.stack.Down(ctx, downOpts...); err != nil {
		return fmt.Errorf("docker compose cleanup failed: %s", err)
	}

	slog.DebugContext(ctx, "docker compose cleanup successful")

	return nil
}

func (d *DockerComposeContainer) GetPostgresURI(ctx context.Context) (string, error) {
	if d.stack == nil {
		return "", fmt.Errorf("docker compose stack is nil")
	}

	env, err := loadMapEnvFile()
	if env == nil {
		return "", fmt.Errorf("docker compose env is nil")
	}

	container, err := d.stack.ServiceContainer(ctx, DOCKER_COMPOSE_SERVICE)
	if err != nil {
		return "", fmt.Errorf("docker compose get container failed: %s", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return "", fmt.Errorf("docker compose get host failed: %s", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return "", fmt.Errorf("docker compose get port failed: %s", err)
	}

	dbUser := env["POSTGRES_USER"]
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPass := env["POSTGRES_PASSWORD"]
	if dbPass == "" {
		dbPass = "password"
	}
	dbName := env["POSTGRES_DB"]
	if dbName == "" {
		dbName = "postgres"
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", dbUser, dbPass, host, port.Port(), dbName), nil
}
