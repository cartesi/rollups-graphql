package raw

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

const (
	DOCKER_COMPOSE_FILE     = "compose.yml"
	DOCKER_COMPOSE_SERVICE  = "postgres-raw"
	DOCKER_COMPOSE_LOGS_MAX = uint8(10)
)

func GetFilePath(name string) (string, error) {
	_, xdir, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current directory")
	}

	dir := filepath.Dir(xdir)
	filePath := path.Join(dir, name)

	return filePath, nil
}

func LoadMapEnvFile() (map[string]string, error) {
	path, err := GetFilePath(".env")
	if err != nil {
		return nil, err
	}
	return godotenv.Read(path)
}

// check if docker compose command is available
func CheckDockerCompose(ctx context.Context) error {
	slog.DebugContext(ctx, "checking docker compose")
	cmd := exec.CommandContext(ctx, "docker", "compose", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose not found: %s", err)
	}
	slog.DebugContext(ctx, "docker compose version", "output", string(output))

	return nil
}

func RunDockerCompose(stdCtx context.Context) error {
	ctx, cancel := context.WithCancel(stdCtx)
	defer cancel()

	slog.DebugContext(ctx, "running docker compose")

	err := CheckDockerCompose(ctx)
	if err != nil {
		return err
	}

	filePath, err := GetFilePath(DOCKER_COMPOSE_FILE)
	if err != nil {
		return err
	}
	slog.DebugContext(ctx, "docker compose file path", "path", filePath)

	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", filePath, "up", DOCKER_COMPOSE_SERVICE, "--wait")
	output, err := cmd.CombinedOutput()

	if err != nil {
		slog.DebugContext(ctx, "docker compose up failed", "output", string(output))
		_ = ShowDockerComposeLog(ctx, filePath)
		return fmt.Errorf("docker compose up failed: %s", err)
	}

	slog.DebugContext(ctx, "docker compose up", "output", string(output))

	return nil
}

func StopDockerCompose(stdCtx context.Context) error {
	ctx, cancel := context.WithCancel(stdCtx)
	defer cancel()

	slog.DebugContext(ctx, "stopping docker compose")

	filePath, err := GetFilePath(DOCKER_COMPOSE_FILE)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", filePath, "down", "--remove-orphans")

	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.DebugContext(ctx, "docker compose down failed", "output", string(output))
		_ = ShowDockerComposeLog(ctx, filePath)
		return fmt.Errorf("docker compose down failed: %s", err)
	}

	return nil
}

func ShowDockerComposeLog(ctx context.Context, filePath string) error {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", filePath, "logs", DOCKER_COMPOSE_SERVICE, "--tail", string(DOCKER_COMPOSE_LOGS_MAX))
	output, err := cmd.CombinedOutput()

	if err != nil {
		slog.DebugContext(ctx, "docker compose logs failed", "output", string(output))
		return fmt.Errorf("docker compose logs failed: %s", err)
	}

	slog.DebugContext(ctx, "docker compose logs", "output", string(output))

	return nil
}

func CleanupDockerCompose(stdCtx context.Context) error {
	ctx, cancel := context.WithCancel(stdCtx)
	defer cancel()

	filePath, err := GetFilePath(DOCKER_COMPOSE_FILE)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", filePath, "down", "--remove-orphans", "--volumes", "--rmi", "local")
	output, err := cmd.CombinedOutput()

	if err != nil {
		slog.DebugContext(ctx, "docker compose cleanup failed", "output", string(output))
		_ = ShowDockerComposeLog(ctx, filePath)
		return fmt.Errorf("docker compose cleanup failed: %s", err)
	}

	slog.DebugContext(ctx, "docker compose cleanup", "output", string(output))

	return nil
}
