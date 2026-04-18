package docker_registry

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type IDockerRegistryProvider interface {
	PushImage(ctx context.Context, tag string) error
}

type dockerHubProvider struct{}

func New() IDockerRegistryProvider {
	return &dockerHubProvider{}
}

// PushImage выполняет docker push. Тег должен содержать registry-префикс
// (например "myuser/disk-usage-metrics-exporter:dev-xxx") — задаётся через OBS_BENCH_DOCKERHUB_NAMESPACE.
func (p *dockerHubProvider) PushImage(ctx context.Context, tag string) error {
	cmd := exec.CommandContext(ctx, "docker", "push", tag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker push %s failed: %w", tag, err)
	}
	return nil
}
