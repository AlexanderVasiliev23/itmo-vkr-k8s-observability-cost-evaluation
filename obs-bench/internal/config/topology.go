package config

import (
	"obs-bench/internal/enum"
	instr "obs-bench/internal/instrument"
)

// CentralMonitoring — неймспейс и сервис «центрального» Prometheus (kube-prometheus-stack),
// через который снимаются метрики целевых стеков и выполняется port-forward в коллекторах.
type CentralMonitoring struct {
	Namespace               string
	PrometheusServiceName   string
	PrometheusLocalPort     int
	PrometheusRemotePort    int
	StackHelmReleaseName    string
}

// InstrumentTarget — всё, что нужно для одного инструмента:
// деплой, query API (port-forward), метрики ресурсов (process_* или cAdvisor) и pvc_used_bytes.
type InstrumentTarget struct {
	DeployNamespace       string
	HelmReleaseName       string // для Prometheus / Loki (лейблы ServiceMonitor)
	QueryServiceName      string
	QueryLocalPort        int
	QueryRemotePort       int
	ProcessMetricsJob     string // kube-prometheus job для process_*; пусто если только CadvisorContainerName
	PVCQueryNamespace     string
	// CadvisorContainerName — использовать ветку container_* (kubelet/cAdvisor). Имя pod берётся из HelmReleaseName или QueryServiceName: pod=~^{name}-[0-9]+$.
	CadvisorContainerName string
}

// Topology — единая топология эксперимента (неймспейсы, сервисы, порты).
type Topology struct {
	CentralMonitoring        CentralMonitoring
	MetricsProviderNamespace string
	LogProducerNamespace     string // деплой log-load-generator (отдельно от metrics-provider)
	Prometheus               InstrumentTarget
	VictoriaMetrics          InstrumentTarget
	Loki                     InstrumentTarget
	OpenSearch               InstrumentTarget
}

func (t *Topology) instrumentTargetsByEnum() map[enum.Instrument]InstrumentTarget {
	return map[enum.Instrument]InstrumentTarget{
		enum.InstrumentPrometheus:      t.Prometheus,
		enum.InstrumentVictoriaMetrics: t.VictoriaMetrics,
		enum.InstrumentLoki:            t.Loki,
		enum.InstrumentOpenSearch:      t.OpenSearch,
	}
}

// InstrumentTarget возвращает профиль для enum.Instrument.
func (t *Topology) InstrumentTarget(i enum.Instrument) (InstrumentTarget, error) {
	return instr.Lookup(t.instrumentTargetsByEnum(), i)
}

// ValidateInstrumentCoverage проверяет, что топология задаёт цели для всех enum.AllInstruments.
func (t *Topology) ValidateInstrumentCoverage() error {
	return enum.EnsureAllInstrumentsInMap(t.instrumentTargetsByEnum())
}

func defaultTopology(prometheusHelmRelease, victoriaSingleService, lokiService, openSearchService string) Topology {
	return Topology{
		CentralMonitoring: CentralMonitoring{
			Namespace:              "monitoring",
			PrometheusServiceName:  "kube-prometheus-stack-prometheus",
			PrometheusLocalPort:    9099,
			PrometheusRemotePort:   9090,
			StackHelmReleaseName:   "kube-prometheus-stack",
		},
		MetricsProviderNamespace: "metrics-provider",
		LogProducerNamespace:     "log-producer",
		Prometheus: InstrumentTarget{
			DeployNamespace:   "prometheus",
			HelmReleaseName:   prometheusHelmRelease,
			QueryServiceName:  prometheusHelmRelease + "-server",
			QueryLocalPort:    9091,
			QueryRemotePort:   9090,
			ProcessMetricsJob: prometheusHelmRelease + "-server",
			PVCQueryNamespace: "prometheus",
		},
		VictoriaMetrics: InstrumentTarget{
			DeployNamespace:   "victoria-metrics",
			HelmReleaseName:   "", // релиз VM chart задаётся внутри victoria-metrics/service
			QueryServiceName:  victoriaSingleService,
			QueryLocalPort:    9092,
			QueryRemotePort:   8428,
			ProcessMetricsJob: victoriaSingleService,
			PVCQueryNamespace: "victoria-metrics",
		},
		Loki: InstrumentTarget{
			DeployNamespace:       "loki",
			HelmReleaseName:       "loki",
			QueryServiceName:      lokiService,
			QueryLocalPort:        3105,
			QueryRemotePort:       3100,
			PVCQueryNamespace:     "loki",
			CadvisorContainerName: "loki",
		},
		OpenSearch: InstrumentTarget{
			DeployNamespace:       "opensearch",
			HelmReleaseName:       "",
			QueryServiceName:      openSearchService,
			QueryLocalPort:        9201,
			QueryRemotePort:       9200,
			PVCQueryNamespace:     "opensearch",
			CadvisorContainerName: "opensearch",
		},
	}
}
