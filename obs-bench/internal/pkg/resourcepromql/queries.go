package resourcepromql

import (
	"fmt"
	"regexp"

	"obs-bench/internal/config"
)

// cadvisorPodSelector возвращает regex для pod StatefulSet (например ^loki-[0-9]+$), если его можно вывести из топологии.
// У kubelet /metrics/cadvisor часто нет image/container; cpu="total" — агрегат по CPU. Дубли одной цели с разными service: max по id.
func cadvisorPodSelector(t config.InstrumentTarget) (podRE string, ok bool) {
	if t.HelmReleaseName != "" {
		return "^" + regexp.QuoteMeta(t.HelmReleaseName) + "-[0-9]+$", true
	}
	if t.QueryServiceName != "" {
		return "^" + regexp.QuoteMeta(t.QueryServiceName) + "-[0-9]+$", true
	}
	return "", false
}

// ResourceQueries возвращает PromQL для CPU (среднее за окно), памяти avg/peak и диска.
//
// Диск: pvc_used_bytes со всех инструментов — через disk-usage-metrics-exporter; лейбл namespace
// добавляется при scrape пода (kube-prometheus-stack).
func ResourceQueries(target config.InstrumentTarget, durSeconds int) (cpu, memAvg, memPeak, disk string, err error) {
	ns := target.PVCQueryNamespace
	// max_over_time: берём пик за окно сбора — защита от просадок при сжатии/compaction.
	disk = fmt.Sprintf(`max_over_time(sum(pvc_used_bytes{namespace="%s"})[%ds:30s])`, ns, durSeconds)

	// CadvisorPodSelector: явный pod-regex для Deployment-подов (не StatefulSet).
	// Приоритет выше CadvisorContainerName и ProcessMetricsJob.
	if target.CadvisorPodSelector != "" {
		podRE := target.CadvisorPodSelector
		const selCPU = `namespace="%s",pod=~"%s",container!="POD",cpu="total"`
		const selMem = `namespace="%s",pod=~"%s",container!="POD"`
		cpu = fmt.Sprintf(
			`avg_over_time((sum by (namespace, pod) (max by (namespace, pod, id) (rate(container_cpu_usage_seconds_total{`+selCPU+`}[1m]))))[%ds:15s])`,
			ns, podRE, durSeconds,
		)
		memAvg = fmt.Sprintf(
			`avg_over_time((sum by (namespace, pod) (max by (namespace, pod, id) (container_memory_working_set_bytes{`+selMem+`})))[%ds:15s])`,
			ns, podRE, durSeconds,
		)
		memPeak = fmt.Sprintf(
			`max_over_time((sum by (namespace, pod) (max by (namespace, pod, id) (container_memory_working_set_bytes{`+selMem+`})))[%ds:15s])`,
			ns, podRE, durSeconds,
		)
		return cpu, memAvg, memPeak, disk, nil
	}

	if target.CadvisorContainerName != "" {
		if podRE, ok := cadvisorPodSelector(target); ok {
			const selCPU = `namespace="%s",pod=~"%s",container!="POD",cpu="total"`
			const selMem = `namespace="%s",pod=~"%s",container!="POD"`
			cpu = fmt.Sprintf(
				`avg_over_time((sum by (namespace, pod) (max by (namespace, pod, id) (rate(container_cpu_usage_seconds_total{`+selCPU+`}[1m]))))[%ds:15s])`,
				ns, podRE, durSeconds,
			)
			memAvg = fmt.Sprintf(
				`avg_over_time((sum by (namespace, pod) (max by (namespace, pod, id) (container_memory_working_set_bytes{`+selMem+`})))[%ds:15s])`,
				ns, podRE, durSeconds,
			)
			memPeak = fmt.Sprintf(
				`max_over_time((sum by (namespace, pod) (max by (namespace, pod, id) (container_memory_working_set_bytes{`+selMem+`})))[%ds:15s])`,
				ns, podRE, durSeconds,
			)
			return cpu, memAvg, memPeak, disk, nil
		}
		cn := target.CadvisorContainerName
		const selCPU = `namespace="%s",container="%s",cpu="total"`
		const selMem = `namespace="%s",container="%s"`
		cpu = fmt.Sprintf(
			`avg_over_time((sum by (namespace, pod) (max by (namespace, pod, id) (rate(container_cpu_usage_seconds_total{`+selCPU+`}[1m]))))[%ds:15s])`,
			ns, cn, durSeconds,
		)
		memAvg = fmt.Sprintf(
			`avg_over_time((sum by (namespace, pod) (max by (namespace, pod, id) (container_memory_working_set_bytes{`+selMem+`})))[%ds:15s])`,
			ns, cn, durSeconds,
		)
		memPeak = fmt.Sprintf(
			`max_over_time((sum by (namespace, pod) (max by (namespace, pod, id) (container_memory_working_set_bytes{`+selMem+`})))[%ds:15s])`,
			ns, cn, durSeconds,
		)
		return cpu, memAvg, memPeak, disk, nil
	}

	if target.ProcessMetricsJob == "" {
		return "", "", "", "", fmt.Errorf("instrument target: need ProcessMetricsJob or CadvisorContainerName")
	}

	job := target.ProcessMetricsJob
	cpu = fmt.Sprintf(`avg_over_time(rate(process_cpu_seconds_total{job="%s"}[1m])[%ds:15s])`, job, durSeconds)
	memAvg = fmt.Sprintf(`avg_over_time(process_resident_memory_bytes{job="%s"}[%ds])`, job, durSeconds)
	memPeak = fmt.Sprintf(`max_over_time(process_resident_memory_bytes{job="%s"}[%ds])`, job, durSeconds)
	return cpu, memAvg, memPeak, disk, nil
}
