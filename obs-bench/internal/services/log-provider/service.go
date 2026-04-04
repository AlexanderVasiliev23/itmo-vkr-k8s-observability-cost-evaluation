package log_provider

import (
	"context"
	"encoding/hex"
	"fmt"

	"obs-bench/internal/config"
	"obs-bench/internal/enum"
	"obs-bench/internal/providers/docker"
	docker_registry "obs-bench/internal/providers/docker-registry"
	"obs-bench/internal/providers/kubernetes"

	"golang.org/x/mod/sumdb/dirhash"
)

// ILogProviderService поднимает деплой log-load-generator в кластере.
type ILogProviderService interface {
	UpLogProvider(ctx context.Context, instrument enum.Instrument, logsPerSec int) error
}

type service struct {
	dockerProvider         docker.IDockerProvider
	dockerRegistryProvider docker_registry.IDockerRegistryProvider
	kubernetesProvider     kubernetes.IKubernetesProvider
	cfg                    *config.Config
}

func NewLogProviderService(
	dockerProvider docker.IDockerProvider,
	dockerRegistryProvider docker_registry.IDockerRegistryProvider,
	kubernetesProvider kubernetes.IKubernetesProvider,
	cfg *config.Config,
) ILogProviderService {
	return &service{
		dockerProvider:         dockerProvider,
		dockerRegistryProvider: dockerRegistryProvider,
		kubernetesProvider:     kubernetesProvider,
		cfg:                    cfg,
	}
}

func (s *service) UpLogProvider(ctx context.Context, instrument enum.Instrument, logsPerSec int) error {
	const contextPath = "./images/log-load-generator"

	hash, err := dirhash.HashDir(contextPath, "", dirhash.Hash1)
	if err != nil {
		return err
	}
	hexHash := hex.EncodeToString([]byte(hash))
	tag := fmt.Sprintf("log-load-generator:dev-%s", hexHash[:12])

	if err := s.dockerProvider.RecreateImageWithNewTag(ctx, tag, contextPath); err != nil {
		return err
	}
	if err := s.dockerRegistryProvider.PushImage(ctx, tag); err != nil {
		return err
	}

	target, err := s.cfg.Topology.InstrumentTarget(instrument)
	if err != nil {
		return err
	}

	ns := s.cfg.Topology.LogProducerNamespace
	if err := s.kubernetesProvider.RecreateNamespace(ctx, ns); err != nil {
		return err
	}

	lokiPush := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d/loki/api/v1/push",
		target.QueryServiceName, target.DeployNamespace, target.QueryRemotePort)
	osBase := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d",
		target.QueryServiceName, target.DeployNamespace, target.QueryRemotePort)

	backend := "loki"
	if instrument == enum.InstrumentOpenSearch {
		backend = "opensearch"
	}

	return s.kubernetesProvider.CreateLogProducerDeployment(ctx, ns, tag, backend, logsPerSec, lokiPush, osBase)
}
