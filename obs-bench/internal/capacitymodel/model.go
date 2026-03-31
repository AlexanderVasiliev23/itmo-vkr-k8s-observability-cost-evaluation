package capacitymodel

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
)

var ResourceKeys = []string{"cpu_cores", "mem_peak_bytes", "disk_bytes"}

const (
	// Целевой порог относительной ошибки (MAPE) для CPU.
	QualityTargetMAPECPU = 0.2
	// Целевой порог относительной ошибки (MAPE) для пика RAM.
	QualityTargetMAPEMemPeak = 0.2
	// Целевой порог относительной ошибки (MAPE) для диска.
	QualityTargetMAPEDisk = 0.2

	// Окно агрегации CPU при анализе устойчивого режима.
	AggregationWindowCPU = "avg over 15m steady-state window"
	// Окно агрегации средней памяти при анализе устойчивого режима.
	AggregationWindowMemAvg = "avg over 15m steady-state window"
	// Окно агрегации пика памяти: максимум за весь прогон и p95 на устойчивом окне.
	AggregationWindowMemPeak = "max over full run and p95 over 15m steady-state window"
	// Окно агрегации диска: конечное значение прогона и наклон за последний час.
	AggregationWindowDisk = "end-of-run value and slope over last 60m"
)

type Row struct {
	Instrument      string
	WorkloadType    string
	LoadValue       float64
	RetentionDays   float64
	DurationSeconds float64
	CPUCores        float64
	MemPeakBytes    float64
	DiskBytes       float64
}

type EstimateInput struct {
	Instrument          string
	WorkloadType        string
	TargetLoad          float64
	TargetRetentionDays float64
	ErrorBudget         float64
}

func InferWorkloadType(instrument string) string {
	if instrument == "loki" || instrument == "opensearch" {
		return "logs"
	}
	return "metrics"
}

func toFloat(v any, def float64) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return def
		}
		return f
	default:
		return def
	}
}

func LoadRows(path string) ([]Row, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw []map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		// Поддерживаем расширенный формат: { "measurements": [...], "model_params": ... }.
		var wrapped struct {
			Measurements []map[string]any `json:"measurements"`
		}
		if err2 := json.Unmarshal(b, &wrapped); err2 != nil {
			return nil, err
		}
		raw = wrapped.Measurements
	}

	rows := make([]Row, 0, len(raw))
	for _, item := range raw {
		instrument, _ := item["instrument"].(string)
		if instrument == "" {
			continue
		}
		workloadType, _ := item["workload_type"].(string)
		if workloadType == "" {
			workloadType = InferWorkloadType(instrument)
		}

		loadValue := toFloat(item["load_value"], toFloat(item["series"], 0))
		retentionDays := toFloat(item["retention_days"], 7)
		durationSeconds := toFloat(item["duration_seconds"], 0)
		if loadValue <= 0 || retentionDays <= 0 {
			continue
		}

		memPeak := toFloat(item["mem_peak_bytes"], toFloat(item["mem_avg_bytes"], 0))
		rows = append(rows, Row{
			Instrument:      instrument,
			WorkloadType:    workloadType,
			LoadValue:       loadValue,
			RetentionDays:   retentionDays,
			DurationSeconds: durationSeconds,
			CPUCores:        toFloat(item["cpu_cores"], 0),
			MemPeakBytes:    memPeak,
			DiskBytes:       toFloat(item["disk_bytes"], 0),
		})
	}
	return rows, nil
}

func keyVal(r Row, key string) float64 {
	switch key {
	case "cpu_cores":
		return r.CPUCores
	case "mem_peak_bytes":
		return r.MemPeakBytes
	case "disk_bytes":
		return r.DiskBytes
	default:
		return 0
	}
}

func interpolate(points [][2]float64, x float64) float64 {
	if len(points) == 0 {
		return 0
	}
	sort.Slice(points, func(i, j int) bool { return points[i][0] < points[j][0] })
	if len(points) == 1 {
		return points[0][1]
	}
	if x <= points[0][0] {
		return segment(points[0], points[1], x)
	}
	if x >= points[len(points)-1][0] {
		return segment(points[len(points)-2], points[len(points)-1], x)
	}
	i := 0
	for i < len(points)-1 && points[i+1][0] < x {
		i++
	}
	return segment(points[i], points[i+1], x)
}

func segment(p0, p1 [2]float64, x float64) float64 {
	dx := p1[0] - p0[0]
	if dx == 0 {
		return p0[1]
	}
	return p0[1] + ((x-p0[0])/dx)*(p1[1]-p0[1])
}

func median(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sort.Float64s(vals)
	return vals[len(vals)/2]
}

func estimateAlphaByRetention(rows []Row, resourceKey string) float64 {
	byRet := map[float64][]float64{}
	for _, r := range rows {
		val := keyVal(r, resourceKey)
		if val <= 0 {
			continue
		}
		byRet[r.RetentionDays] = append(byRet[r.RetentionDays], val)
	}
	if len(byRet) < 2 {
		switch resourceKey {
		case "disk_bytes":
			return 1.0
		case "mem_peak_bytes":
			return 0.25
		default:
			return 0.15
		}
	}

	type pt struct{ x, y float64 }
	pts := make([]pt, 0, len(byRet))
	for ret, vals := range byRet {
		m := median(vals)
		if ret > 0 && m > 0 {
			pts = append(pts, pt{x: math.Log(ret), y: math.Log(m)})
		}
	}
	if len(pts) < 2 {
		if resourceKey == "disk_bytes" {
			return 1.0
		}
		return 0.2
	}

	var xMean, yMean float64
	for _, p := range pts {
		xMean += p.x
		yMean += p.y
	}
	xMean /= float64(len(pts))
	yMean /= float64(len(pts))

	var cov, vr float64
	for _, p := range pts {
		cov += (p.x - xMean) * (p.y - yMean)
		vr += (p.x - xMean) * (p.x - xMean)
	}
	if vr == 0 {
		if resourceKey == "disk_bytes" {
			return 1.0
		}
		return 0.2
	}
	alpha := cov / vr

	switch resourceKey {
	case "disk_bytes":
		return math.Max(0.6, math.Min(1.4, alpha))
	case "mem_peak_bytes":
		return math.Max(0.0, math.Min(0.8, alpha))
	default:
		return math.Max(0.0, math.Min(0.6, alpha))
	}
}

func estimateResource(rows []Row, targetLoad, targetRetention float64, resourceKey string) float64 {
	alpha := estimateAlphaByRetention(rows, resourceKey)
	points := make([][2]float64, 0, len(rows))
	for _, r := range rows {
		base := keyVal(r, resourceKey)
		if base <= 0 {
			continue
		}
		scaled := base * math.Pow(targetRetention/r.RetentionDays, alpha)
		points = append(points, [2]float64{r.LoadValue, scaled})
	}
	return math.Max(0.0, interpolate(points, targetLoad))
}

func estimateAll(rows []Row, targetLoad, targetRetention float64) map[string]float64 {
	return map[string]float64{
		"cpu_cores":      estimateResource(rows, targetLoad, targetRetention, "cpu_cores"),
		"mem_peak_bytes": estimateResource(rows, targetLoad, targetRetention, "mem_peak_bytes"),
		"disk_bytes":     estimateResource(rows, targetLoad, targetRetention, "disk_bytes"),
	}
}

func mape(yTrue, yPred float64) float64 {
	if yTrue <= 0 {
		return 0
	}
	return math.Abs(yTrue-yPred) / yTrue
}

func validateHoldout(rows []Row) map[string]float64 {
	out := map[string]float64{
		"cpu_cores":      math.NaN(),
		"mem_peak_bytes": math.NaN(),
		"disk_bytes":     math.NaN(),
	}
	if len(rows) < 3 {
		return out
	}

	errs := map[string][]float64{
		"cpu_cores":      {},
		"mem_peak_bytes": {},
		"disk_bytes":     {},
	}
	for i, test := range rows {
		train := make([]Row, 0, len(rows)-1)
		train = append(train, rows[:i]...)
		train = append(train, rows[i+1:]...)
		if len(train) < 2 {
			continue
		}
		pred := estimateAll(train, test.LoadValue, test.RetentionDays)
		errs["cpu_cores"] = append(errs["cpu_cores"], mape(test.CPUCores, pred["cpu_cores"]))
		errs["mem_peak_bytes"] = append(errs["mem_peak_bytes"], mape(test.MemPeakBytes, pred["mem_peak_bytes"]))
		errs["disk_bytes"] = append(errs["disk_bytes"], mape(test.DiskBytes, pred["disk_bytes"]))
	}
	for _, k := range ResourceKeys {
		if len(errs[k]) == 0 {
			continue
		}
		var s float64
		for _, v := range errs[k] {
			s += v
		}
		out[k] = s / float64(len(errs[k]))
	}
	return out
}

func sanitizeNaNMap(in map[string]float64) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			out[k] = nil
			continue
		}
		out[k] = v
	}
	return out
}

func formatBytes(n float64) string {
	gib := n / math.Pow(1024, 3)
	return fmt.Sprintf("%.0f bytes (%.2f GiB)", n, gib)
}

func BuildReport(rows []Row, in EstimateInput) (map[string]any, error) {
	subset := make([]Row, 0)
	for _, r := range rows {
		if r.Instrument == in.Instrument && r.WorkloadType == in.WorkloadType {
			subset = append(subset, r)
		}
	}
	if len(subset) == 0 {
		return nil, fmt.Errorf("no rows for instrument=%s, workload_type=%s", in.Instrument, in.WorkloadType)
	}

	estimate := estimateAll(subset, in.TargetLoad, in.TargetRetentionDays)
	holdout := validateHoldout(subset)

	rangeOut := map[string]map[string]float64{}
	for _, k := range ResourceKeys {
		rangeOut[k] = map[string]float64{
			"min": estimate[k] * (1.0 - in.ErrorBudget),
			"max": estimate[k] * (1.0 + in.ErrorBudget),
		}
	}

	out := map[string]any{
		"target": map[string]any{
			"instrument":     in.Instrument,
			"workload_type":  in.WorkloadType,
			"load_value":     in.TargetLoad,
			"retention_days": in.TargetRetentionDays,
		},
		"estimate": estimate,
		"estimate_human": map[string]any{
			"cpu_cores":      math.Round(estimate["cpu_cores"]*10000) / 10000,
			"mem_peak_bytes": formatBytes(estimate["mem_peak_bytes"]),
			"disk_bytes":     formatBytes(estimate["disk_bytes"]),
		},
		"validation_mape": sanitizeNaNMap(holdout),
		"quality_targets": map[string]float64{
			"cpu_cores_mape_max":      QualityTargetMAPECPU,
			"mem_peak_bytes_mape_max": QualityTargetMAPEMemPeak,
			"disk_bytes_mape_max":     QualityTargetMAPEDisk,
		},
		"aggregation_windows": map[string]string{
			"cpu_cores":      AggregationWindowCPU,
			"mem_avg_bytes":  AggregationWindowMemAvg,
			"mem_peak_bytes": AggregationWindowMemPeak,
			"disk_bytes":     AggregationWindowDisk,
		},
		"range_with_error_budget": rangeOut,
	}
	return out, nil
}
