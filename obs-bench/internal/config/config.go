package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	docker_registry "obs-bench/internal/providers/docker-registry"
)

type Config struct {
	DockerRegistryType docker_registry.DockerRegistryType
	StorageDSN         string
	SQLDebug           bool
	Topology           ClusterLayout
}

func NewConfig() (*Config, error) {
	regType, err := parseDockerRegistryType(os.Getenv("OBS_BENCH_DOCKER_REGISTRY"))
	if err != nil {
		return nil, err
	}

	dsn := os.Getenv("OBS_BENCH_STORAGE_DSN")
	if dsn == "" {
		dsn = "file:test.db?cache=shared&mode=rwc"
	}

	promRelease := os.Getenv("OBS_BENCH_PROMETHEUS_RELEASE")
	if promRelease == "" {
		promRelease = "prometheus"
	}

	vmSingle := os.Getenv("OBS_BENCH_VM_SINGLE_SERVICE")
	if vmSingle == "" {
		vmSingle = "vmsingle-victoria-metrics-k8s-stack"
	}

	lokiSvc := os.Getenv("OBS_BENCH_LOKI_QUERY_SERVICE")
	if lokiSvc == "" {
		lokiSvc = "loki"
	}
	osSvc := os.Getenv("OBS_BENCH_OPENSEARCH_QUERY_SERVICE")
	if osSvc == "" {
		osSvc = "opensearch-cluster-master"
	}

	topo := defaultTopology(promRelease, vmSingle, lokiSvc, osSvc)
	applyTopologyEnvOverrides(&topo)
	if err := topo.ValidateInstrumentCoverage(); err != nil {
		return nil, err
	}

	sqlDebug := strings.EqualFold(strings.TrimSpace(os.Getenv("OBS_BENCH_SQL_DEBUG")), "1") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("OBS_BENCH_SQL_DEBUG")), "true")

	return &Config{
		DockerRegistryType: regType,
		StorageDSN:         dsn,
		SQLDebug:           sqlDebug,
		Topology:           topo,
	}, nil
}

// applyTopologyEnvOverrides подставляет значения из env поверх defaultTopology (частичный override).
func applyTopologyEnvOverrides(t *ClusterLayout) {
	if v := strings.TrimSpace(os.Getenv("OBS_BENCH_MONITORING_NAMESPACE")); v != "" {
		t.CentralMonitoring.Namespace = v
	}
	if v := strings.TrimSpace(os.Getenv("OBS_BENCH_CENTRAL_PROMETHEUS_SERVICE")); v != "" {
		t.CentralMonitoring.PrometheusServiceName = v
	}
	if v := strings.TrimSpace(os.Getenv("OBS_BENCH_METRICS_PROVIDER_NAMESPACE")); v != "" {
		t.MetricsProviderNamespace = v
	}
	if v := strings.TrimSpace(os.Getenv("OBS_BENCH_LOG_PRODUCER_NAMESPACE")); v != "" {
		t.LogProducerNamespace = v
	}
	if v := strings.TrimSpace(os.Getenv("OBS_BENCH_PROMETHEUS_TARGET_NAMESPACE")); v != "" {
		t.Prometheus.DeployNamespace = v
		t.Prometheus.PVCQueryNamespace = v
	}
	if v := strings.TrimSpace(os.Getenv("OBS_BENCH_VICTORIA_TARGET_NAMESPACE")); v != "" {
		t.VictoriaMetrics.DeployNamespace = v
		t.VictoriaMetrics.PVCQueryNamespace = v
	}
	if v := strings.TrimSpace(os.Getenv("OBS_BENCH_PROMETHEUS_QUERY_LOCAL_PORT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			t.Prometheus.QueryLocalPort = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("OBS_BENCH_VICTORIA_QUERY_LOCAL_PORT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			t.VictoriaMetrics.QueryLocalPort = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("OBS_BENCH_LOKI_TARGET_NAMESPACE")); v != "" {
		t.Loki.DeployNamespace = v
		t.Loki.PVCQueryNamespace = v
	}
	if v := strings.TrimSpace(os.Getenv("OBS_BENCH_OPENSEARCH_TARGET_NAMESPACE")); v != "" {
		t.OpenSearch.DeployNamespace = v
		t.OpenSearch.PVCQueryNamespace = v
	}
	if v := strings.TrimSpace(os.Getenv("OBS_BENCH_LOKI_QUERY_LOCAL_PORT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			t.Loki.QueryLocalPort = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("OBS_BENCH_OPENSEARCH_QUERY_LOCAL_PORT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			t.OpenSearch.QueryLocalPort = n
		}
	}
}

func parseDockerRegistryType(s string) (docker_registry.DockerRegistryType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", string(docker_registry.DockerRegistryTypeMinikube):
		return docker_registry.DockerRegistryTypeMinikube, nil
	default:
		return "", fmt.Errorf("unknown OBS_BENCH_DOCKER_REGISTRY %q (supported: minikube)", s)
	}
}
