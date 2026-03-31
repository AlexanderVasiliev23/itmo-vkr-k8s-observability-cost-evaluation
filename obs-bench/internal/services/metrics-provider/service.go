package metrics_provider

import (
	"context"
	"encoding/hex"
	"fmt"

	"obs-bench/internal/providers/docker"
	docker_registry "obs-bench/internal/providers/docker-registry"
	"obs-bench/internal/providers/kubernetes"

	"golang.org/x/mod/sumdb/dirhash"
)

type IMetricsProviderService interface {
	UpMetricsProvider(ctx context.Context, namespace string, series int) error
}

type service struct {
	dockerProvider         docker.IDockerProvider
	dockerRegistryProvider docker_registry.IDockerRegistryProvider
	kubernetesProvider     kubernetes.IKubernetesProvider
}

func NewMetricsProviderService(
	dockerProvider docker.IDockerProvider,
	dockerRegistryProvider docker_registry.IDockerRegistryProvider,
	kubernetesProvider kubernetes.IKubernetesProvider,
) IMetricsProviderService {
	return &service{
		dockerProvider:         dockerProvider,
		dockerRegistryProvider: dockerRegistryProvider,
		kubernetesProvider:     kubernetesProvider,
	}
}

func (s *service) UpMetricsProvider(ctx context.Context, namespace string, series int) error {
	const (
		providerDockerfileContextPath = "./metrics-provider"
	)

	hash, err := dirhash.HashDir(providerDockerfileContextPath, "", dirhash.Hash1)
	if err != nil {
		return err
	}
	hexHash := hex.EncodeToString([]byte(hash))

	tag := fmt.Sprintf("metrics-provider:dev-%s", hexHash[:12])

	if err := s.dockerProvider.RecreateImageWithNewTag(ctx, tag, providerDockerfileContextPath); err != nil {
		return err
	}

	if err := s.dockerRegistryProvider.PushImage(ctx, tag); err != nil {
		return err
	}

	if err := s.kubernetesProvider.RecreateNamespace(ctx, namespace); err != nil {
		return err
	}

	if err := s.kubernetesProvider.CreateMetricsExporterDeployment(ctx, namespace, tag, series); err != nil {
		return err
	}

	if err := s.kubernetesProvider.CreateService(ctx, namespace); err != nil {
		return err
	}

	// ServiceMonitor для metrics-exporter не создаём: с лейблом release=kube-prometheus-stack
	// центральный Prometheus в monitoring начинал забирать весь поток экспериментальных серий.
	// Снимают только тестируемые стеки: Prometheus (extraScrapeConfigs), VictoriaMetrics (VMServiceScrape/vmagent).

	return nil
}
