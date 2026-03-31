package prometheus

import (
	"context"
	"fmt"

	"obs-bench/internal/config"
	"obs-bench/internal/pkg/diskexporter"
	"obs-bench/internal/providers/docker"
	docker_registry "obs-bench/internal/providers/docker-registry"
	"obs-bench/internal/providers/helm"
	"obs-bench/internal/providers/kubernetes"
)

type IPrometheusService interface {
	UpPrometheusStack(ctx context.Context, namespace string) error
}

type service struct {
	kubernetesProvider       kubernetes.IKubernetesProvider
	helmProvider             helm.IHelmProvider
	dockerProvider           docker.IDockerProvider
	dockerRegistryProvider   docker_registry.IDockerRegistryProvider
	releaseName              string
	metricsProviderNamespace string
}

func NewPrometheusService(
	kubernetesProvider kubernetes.IKubernetesProvider,
	helmProvider helm.IHelmProvider,
	dockerProvider docker.IDockerProvider,
	dockerRegistryProvider docker_registry.IDockerRegistryProvider,
	cfg *config.Config,
) IPrometheusService {
	return &service{
		kubernetesProvider:       kubernetesProvider,
		helmProvider:             helmProvider,
		dockerProvider:           dockerProvider,
		dockerRegistryProvider:   dockerRegistryProvider,
		releaseName:              cfg.Topology.Prometheus.HelmReleaseName,
		metricsProviderNamespace: cfg.Topology.MetricsProviderNamespace,
	}
}

func (s *service) UpPrometheusStack(ctx context.Context, namespace string) error {
	tag, err := diskexporter.BuildDevImageTag()
	if err != nil {
		return err
	}

	if err := s.dockerProvider.RecreateImageWithNewTag(ctx, tag, diskexporter.ContextPath); err != nil {
		return err
	}

	if err := s.dockerRegistryProvider.PushImage(ctx, tag); err != nil {
		return err
	}

	if err := s.kubernetesProvider.DeleteNamespace(ctx, namespace); err != nil {
		return err
	}

	const (
		repoURL   = "https://prometheus-community.github.io/helm-charts"
		chartName = "prometheus"
	)
	releaseName := s.releaseName

	extraScrapeConfigs := fmt.Sprintf(
		`- job_name: 'metrics-exporter'
  scrape_interval: 30s
  scrape_timeout: 20s
  static_configs:
    - targets: ['metrics-exporter.%s.svc.cluster.local:8080']
- job_name: 'disk-metrics-exporter'
  scrape_interval: 30s
  scrape_timeout: 20s
  static_configs:
    - targets: ['disk-metrics-exporter.%s.svc.cluster.local:8080']`,
		s.metricsProviderNamespace,
		namespace,
	)

	helmValues := map[string]interface{}{
		"server": map[string]interface{}{
			"retention": "1h",
			"persistentVolume": map[string]interface{}{
				"enabled":   true,
				"size":      "10Gi",
				"resources": map[string]interface{}{},
			},
			// Без лимитов Prometheus часто уходит в OOMKilled при scrape + TSDB (особенно на minikube).
			"resources": map[string]interface{}{
				"limits": map[string]interface{}{
					"memory": "4Gi",
					"cpu":    "2",
				},
				"requests": map[string]interface{}{
					"memory": "1Gi",
					"cpu":    "500m",
				},
			},
		},
		"kube-state-metrics": map[string]interface{}{
			"enabled": false,
		},
		"prometheus-node-exporter": map[string]interface{}{
			"enabled": false,
		},
		"alertmanager": map[string]interface{}{
			"enabled": false,
		},
		"prometheus-pushgateway": map[string]interface{}{
			"enabled": false,
		},
		"scrapeConfigs": map[string]interface{}{
			"prometheus": map[string]interface{}{
				"enabled": true,
			},
			"kubernetes-api-servers": map[string]interface{}{
				"enabled": false,
			},
			"kubernetes-nodes": map[string]interface{}{
				"enabled": false,
			},
			"kubernetes-nodes-cadvisor": map[string]interface{}{
				"enabled": false,
			},
			"kubernetes-service-endpoints": map[string]interface{}{
				"enabled": false,
			},
			"kubernetes-service-endpoints-slow": map[string]interface{}{
				"enabled": false,
			},
			"prometheus-pushgateway": map[string]interface{}{
				"enabled": false,
			},
			"kubernetes-services": map[string]interface{}{
				"enabled": false,
			},
			"kubernetes-pods": map[string]interface{}{
				"enabled": false,
			},
			"kubernetes-pods-slow": map[string]interface{}{
				"enabled": false,
			},
		},
		"extraScrapeConfigs": extraScrapeConfigs,
	}

	if err := s.helmProvider.Up(ctx, namespace, helmValues, repoURL, chartName, releaseName); err != nil {
		return err
	}

	if err := s.kubernetesProvider.CreateDiskMetricsExporter(
		ctx,
		namespace,
		tag,
		releaseName+"-server",
		namespace,
	); err != nil {
		return err
	}

	if err := s.kubernetesProvider.CreateDiskMetricsService(ctx, namespace); err != nil {
		return err
	}
	if err := s.kubernetesProvider.CreateServiceMonitor(ctx, namespace, "disk-metrics-exporter", "metrics", map[string]string{"app": "disk-metrics-exporter"}); err != nil {
		return err
	}
	if err := s.kubernetesProvider.CreateServiceMonitor(ctx, namespace, releaseName+"-server", "http", map[string]string{
		"app.kubernetes.io/name":     "prometheus",
		"app.kubernetes.io/instance": releaseName,
	}); err != nil {
		return err
	}

	return nil
}
