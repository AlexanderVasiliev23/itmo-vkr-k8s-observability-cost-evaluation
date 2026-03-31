package docker_registry

import (
	"context"
	"fmt"
	"os/exec"
)

type DockerRegistryType string

const (
	DockerRegistryTypeMinikube DockerRegistryType = "minikube"
)

type IDockerRegistryProvider interface {
	PushImage(ctx context.Context, tag string) error
}

type minikubeProvider struct {
}

func NewDockerRegistryProvider(t DockerRegistryType) (IDockerRegistryProvider, error) {
	switch t {
	case DockerRegistryTypeMinikube:
		return &minikubeProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown Docker registry type: %s", t)
	}
}

func (p *minikubeProvider) PushImage(ctx context.Context, tag string) error {
	const (
		executable = "/home/linuxbrew/.linuxbrew/bin/minikube"
	)

	if _, err := exec.LookPath(executable); err != nil {
		return fmt.Errorf("minikube executable not found in PATH: %w", err)
	}

	cmd := exec.Command(executable, "image", "load", tag)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("minikube image load failed: %w, output: %s", err, string(output))
	}

	return nil
}
