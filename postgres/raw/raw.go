package raw

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/compose"
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

func RunDockerCompose(stdCtx context.Context) error {
	slog.Debug("running docker compose")
	if stdCtx == nil {
		stdCtx = context.Background()
	}
	ctx, cancel := context.WithCancel(stdCtx)
	defer cancel()

	filePath, err := GetFilePath(DOCKER_COMPOSE_FILE)
	if err != nil {
		return err
	}

	dockerCompose, err := compose.NewDockerCompose(filePath)
	if err != nil {
		return fmt.Errorf("failed to create docker compose: %s", err)
	}

	err = dockerCompose.Up(ctx, compose.RunServices(DOCKER_COMPOSE_SERVICE), compose.Wait(true))

	return nil
}

func StopDockerCompose(stdCtx context.Context) error {
	slog.Debug("stopping docker compose")
	ctx, cancel := context.WithCancel(stdCtx)
	defer cancel()

	filePath, err := GetFilePath(DOCKER_COMPOSE_FILE)
	if err != nil {
		return err
	}

	dockerCompose, err := compose.NewDockerComposeWith(compose.WithStackFiles(filePath))

	container, err := dockerCompose.ServiceContainer(ctx, DOCKER_COMPOSE_SERVICE)
	if err != nil {
		return fmt.Errorf("failed to get service container: %s", err)
	}
	ShowDockerComposeLog(ctx, container)

	err = dockerCompose.Down(ctx, compose.RemoveOrphans(true), compose.RemoveVolumes(true))

	return err
}

func ShowDockerComposeLog(ctx context.Context, container *testcontainers.DockerContainer) error {
	if container == nil {
		return fmt.Errorf("container is nil")
	}
	logs, err := container.Logs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get logs: %s", err)
	}
	defer logs.Close()
	val, err := io.ReadAll(logs)
	if err != nil {
		return fmt.Errorf("failed to read logs: %s", err)
	}
	slog.Debug("docker compose logs", "logs", string(val))

	return nil
}

func CleanupDockerCompose(stdCtx context.Context) error {
	slog.Debug("stopping docker compose")
	ctx, cancel := context.WithCancel(stdCtx)
	defer cancel()

	filePath, err := GetFilePath(DOCKER_COMPOSE_FILE)
	if err != nil {
		return err
	}

	dockerCompose, err := compose.NewDockerComposeWith(compose.WithStackFiles(filePath))
	if err != nil {
		return fmt.Errorf("failed to create docker compose: %s", err)
	}

	container, err := dockerCompose.ServiceContainer(ctx, DOCKER_COMPOSE_SERVICE)
	if err != nil {
		return fmt.Errorf("failed to get service container: %s", err)
	}
	ShowDockerComposeLog(ctx, container)

	err = dockerCompose.Down(ctx, compose.RemoveOrphans(true), compose.RemoveVolumes(true), compose.RemoveImages(compose.RemoveImagesLocal))

	return err
}
